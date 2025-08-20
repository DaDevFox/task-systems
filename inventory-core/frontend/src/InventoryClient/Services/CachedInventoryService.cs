using System.Collections.Concurrent;
using InventoryClient.Models;
using Microsoft.Extensions.Logging;

namespace InventoryClient.Services;

/// <summary>
/// Cache entry for inventory data with temperature-based expiration
/// </summary>
public class CacheEntry<T>
{
    public T Data { get; }
    public DateTime LastAccessed { get; set; }
    public DateTime CreatedAt { get; }
    public int AccessCount { get; set; }
    public TimeSpan MaxAge { get; }

    public CacheEntry(T data, TimeSpan maxAge)
    {
        Data = data;
        CreatedAt = DateTime.UtcNow;
        LastAccessed = DateTime.UtcNow;
        AccessCount = 0;
        MaxAge = maxAge;
    }

    /// <summary>
    /// Gets the "heat" of this cache entry (0.0 = cold, 1.0 = hot)
    /// </summary>
    public double Heat
    {
        get
        {
            var age = DateTime.UtcNow - CreatedAt;
            var timeSinceLastAccess = DateTime.UtcNow - LastAccessed;

            // Heat based on age (newer = hotter)
            var ageHeat = Math.Max(0, 1.0 - (age.TotalSeconds / MaxAge.TotalSeconds));

            // Heat based on recent access (more recent = hotter)
            var accessHeat = Math.Max(0, 1.0 - (timeSinceLastAccess.TotalSeconds / MaxAge.TotalSeconds));

            // Heat based on access frequency (more accessed = hotter)
            var frequencyHeat = Math.Min(1.0, AccessCount / 10.0); // Cap at 10 accesses for max heat

            // Combined heat score
            return (ageHeat * 0.4 + accessHeat * 0.4 + frequencyHeat * 0.2);
        }
    }

    /// <summary>
    /// Gets whether this cache entry should be refreshed based on heat threshold
    /// </summary>
    public bool ShouldRefresh(double heatThreshold)
    {
        return Heat < heatThreshold;
    }

    /// <summary>
    /// Marks this entry as accessed
    /// </summary>
    public void Touch()
    {
        LastAccessed = DateTime.UtcNow;
        AccessCount++;
    }
}

/// <summary>
/// Cached inventory service wrapper that provides temperature-based caching
/// </summary>
public class CachedInventoryService : IInventoryService, IDisposable
{
    private readonly IInventoryService _innerService;
    private readonly ISettingsService _settingsService;
    private readonly ILogger<CachedInventoryService> _logger;
    private readonly ConcurrentDictionary<string, CacheEntry<object>> _cache = new();
    private readonly Timer _cleanupTimer;

    // Settings keys for cache configuration
    private const string HeatThresholdKey = "Cache.HeatThreshold";
    private const string DefaultCacheTimeKey = "Cache.DefaultCacheTime";
    private const string ItemsCacheTimeKey = "Cache.ItemsCacheTime";
    private const string StatusCacheTimeKey = "Cache.StatusCacheTime";
    private const string CleanupIntervalKey = "Cache.CleanupInterval";

    // Default values
    private const double DefaultHeatThreshold = 0.3; // Refresh when heat drops below 30%
    private const int DefaultCacheTimeMinutes = 5;
    private const int ItemsCacheTimeMinutes = 2;
    private const int StatusCacheTimeMinutes = 1;
    private const int CleanupIntervalMinutes = 10;

    public CachedInventoryService(
        IInventoryService innerService,
        ISettingsService settingsService,
        ILogger<CachedInventoryService> logger)
    {
        _innerService = innerService;
        _settingsService = settingsService;
        _logger = logger;

        // Initialize default settings if they don't exist
        InitializeDefaultSettings();

        // Setup cleanup timer
        var cleanupInterval = _settingsService.GetSetting(CleanupIntervalKey, CleanupIntervalMinutes);
        _cleanupTimer = new Timer(CleanupExpiredEntries, null,
            TimeSpan.FromMinutes(cleanupInterval),
            TimeSpan.FromMinutes(cleanupInterval));

        DebugService.LogDebug("CachedInventoryService initialized with heat threshold: {0}",
            _settingsService.GetSetting(HeatThresholdKey, DefaultHeatThreshold));
    }

    public bool IsConnected => _innerService.IsConnected;

    private void InitializeDefaultSettings()
    {
        if (!_settingsService.HasSetting(HeatThresholdKey))
            _settingsService.SetSetting(HeatThresholdKey, DefaultHeatThreshold);

        if (!_settingsService.HasSetting(DefaultCacheTimeKey))
            _settingsService.SetSetting(DefaultCacheTimeKey, DefaultCacheTimeMinutes);

        if (!_settingsService.HasSetting(ItemsCacheTimeKey))
            _settingsService.SetSetting(ItemsCacheTimeKey, ItemsCacheTimeMinutes);

        if (!_settingsService.HasSetting(StatusCacheTimeKey))
            _settingsService.SetSetting(StatusCacheTimeKey, StatusCacheTimeMinutes);

        if (!_settingsService.HasSetting(CleanupIntervalKey))
            _settingsService.SetSetting(CleanupIntervalKey, CleanupIntervalMinutes);
    }

    public async Task<bool> ConnectAsync(string address, CancellationToken cancellationToken = default)
    {
        var result = await _innerService.ConnectAsync(address, cancellationToken);
        if (result)
        {
            // Clear cache on new connection
            _cache.Clear();
            DebugService.LogDebug("Cache cleared on new connection to: {0}", address);
        }
        return result;
    }

    public async Task DisconnectAsync()
    {
        await _innerService.DisconnectAsync();
        _cache.Clear();
        DebugService.LogDebug("Cache cleared on disconnect");
    }

    public async Task<bool> PingAsync()
    {
        return await _innerService.PingAsync();
    }

    public async Task<InventoryStatusViewModel> GetInventoryStatusAsync(bool lowStockOnly = false, IEnumerable<string>? itemIds = null)
    {
        var cacheKey = $"status_{lowStockOnly}_{string.Join(",", itemIds?.OrderBy(x => x) ?? Enumerable.Empty<string>())}";
        var cacheTime = TimeSpan.FromMinutes(_settingsService.GetSetting(StatusCacheTimeKey, StatusCacheTimeMinutes));
        var heatThreshold = _settingsService.GetSetting(HeatThresholdKey, DefaultHeatThreshold);

        return await GetOrSetCacheAsync(cacheKey, cacheTime, heatThreshold,
            () => _innerService.GetInventoryStatusAsync(lowStockOnly, itemIds));
    }

    public async Task<InventoryItemViewModel?> GetInventoryItemAsync(string itemId)
    {
        var cacheKey = $"item_{itemId}";
        var cacheTime = TimeSpan.FromMinutes(_settingsService.GetSetting(DefaultCacheTimeKey, DefaultCacheTimeMinutes));
        var heatThreshold = _settingsService.GetSetting(HeatThresholdKey, DefaultHeatThreshold);

        return await GetOrSetCacheAsync(cacheKey, cacheTime, heatThreshold,
            () => _innerService.GetInventoryItemAsync(itemId));
    }

    public async Task<(IEnumerable<InventoryItemViewModel> Items, int TotalCount)> ListInventoryItemsAsync(
        bool lowStockOnly = false, string? unitTypeFilter = null, int limit = 100, int offset = 0)
    {
        var cacheKey = $"list_{lowStockOnly}_{unitTypeFilter ?? "all"}_{limit}_{offset}";
        var cacheTime = TimeSpan.FromMinutes(_settingsService.GetSetting(ItemsCacheTimeKey, ItemsCacheTimeMinutes));
        var heatThreshold = _settingsService.GetSetting(HeatThresholdKey, DefaultHeatThreshold);

        return await GetOrSetCacheAsync(cacheKey, cacheTime, heatThreshold,
            () => _innerService.ListInventoryItemsAsync(lowStockOnly, unitTypeFilter, limit, offset));
    }

    public async Task<bool> UpdateInventoryLevelAsync(string itemId, double newLevel, string reason = "", bool recordConsumption = true)
    {
        var result = await _innerService.UpdateInventoryLevelAsync(itemId, newLevel, reason, recordConsumption);

        if (result)
        {
            // Invalidate related cache entries
            InvalidateCacheForItem(itemId);
            DebugService.LogDebug("Cache invalidated for item: {0}", itemId);
        }

        return result;
    }

    public async Task<InventoryItemViewModel?> AddInventoryItemAsync(
        string name, string description, double initialLevel, double maxCapacity,
        double lowStockThreshold, string unitId, Dictionary<string, string>? metadata = null)
    {
        var result = await _innerService.AddInventoryItemAsync(
            name, description, initialLevel, maxCapacity, lowStockThreshold, unitId, metadata);

        if (result != null)
        {
            // Clear status and list caches since there's a new item
            InvalidateStatusAndListCaches();
            DebugService.LogDebug("Status and list caches invalidated due to new item: {0}", result.Id);
        }

        return result;
    }

    // public async Task<ConsumptionPredictionViewModel?> PredictConsumptionAsync(string itemId, int daysAhead = 30, bool updateBehavior = false)
    // {
    //     // Predictions are not cached as they may change frequently and are computationally intensive
    //     return await _innerService.PredictConsumptionAsync(itemId, daysAhead, updateBehavior);
    // }
    //
    // public async Task<bool> SetConsumptionBehaviorAsync(string itemId, ConsumptionBehaviorViewModel behavior)
    // {
    //     var result = await _innerService.SetConsumptionBehaviorAsync(itemId, behavior);
    //
    //     if (result)
    //     {
    //         InvalidateCacheForItem(itemId);
    //         DebugService.LogDebug("Cache invalidated for item behavior update: {0}", itemId);
    //     }
    //
    //     return result;
    // }
    //
    public async Task<(double ConvertedAmount, bool Success, string? ErrorMessage)> ConvertUnitsAsync(double amount, string fromUnitId, string toUnitId)
    {
        // Unit conversions are cached as they're static
        var cacheKey = $"convert_{amount}_{fromUnitId}_{toUnitId}";
        var cacheTime = TimeSpan.FromHours(24); // Cache conversions for 24 hours
        var heatThreshold = _settingsService.GetSetting(HeatThresholdKey, DefaultHeatThreshold);

        return await GetOrSetCacheAsync(cacheKey, cacheTime, heatThreshold,
            () => _innerService.ConvertUnitsAsync(amount, fromUnitId, toUnitId));
    }

    public async Task<bool> RemoveInventoryItemAsync(string itemId)
    {
        var result = await _innerService.RemoveInventoryItemAsync(itemId);

        if (result)
        {
            // Invalidate all caches related to this item and refresh lists/status
            InvalidateCacheForItem(itemId);
            InvalidateStatusAndListCaches();
            DebugService.LogDebug("Cache invalidated due to item removal: {0}", itemId);
        }

        return result;
    }

    private async Task<T> GetOrSetCacheAsync<T>(string cacheKey, TimeSpan maxAge, double heatThreshold, Func<Task<T>> factory)
    {
        // Check if we have a cached entry
        if (_cache.TryGetValue(cacheKey, out var cacheEntry) && cacheEntry.Data is T)
        {
            var typedEntry = new CacheEntry<T>((T)cacheEntry.Data, maxAge)
            {
                LastAccessed = cacheEntry.LastAccessed,
                AccessCount = cacheEntry.AccessCount
            };

            typedEntry.Touch();

            // Check if the cache is still "hot" enough
            if (!typedEntry.ShouldRefresh(heatThreshold))
            {
                DebugService.LogDebug("Cache HIT for '{0}' (heat: {1:F2})", cacheKey, typedEntry.Heat);
                return (T)cacheEntry.Data;
            }
            
            DebugService.LogDebug("Cache COLD for '{0}' (heat: {1:F2}, threshold: {2:F2})",
                cacheKey, typedEntry.Heat, heatThreshold);
        }
        
        if (!_cache.TryGetValue(cacheKey, out _))
        {
            DebugService.LogDebug("Cache MISS for '{0}'", cacheKey);
        }// Fetch fresh data
        try
        {
            var data = await factory();
            var newEntry = new CacheEntry<object>(data!, maxAge);
            _cache[cacheKey] = newEntry;

            DebugService.LogDebug("Cache SET for '{0}' (expires in {1} minutes)", cacheKey, maxAge.TotalMinutes);
            return data;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to fetch data for cache key: {CacheKey}", cacheKey);

            // If we have stale data, return it instead of failing
            if (_cache.TryGetValue(cacheKey, out var staleEntry) && staleEntry.Data is T staleData)
            {
                DebugService.LogDebug("Returning STALE cache data for '{0}' due to fetch error", cacheKey);
                return staleData;
            }

            throw;
        }
    }

    private void InvalidateCacheForItem(string itemId)
    {
        var keysToRemove = _cache.Keys.Where(key =>
            key.StartsWith($"item_{itemId}") ||
            key.StartsWith("status_") ||
            key.StartsWith("list_")).ToList();

        foreach (var key in keysToRemove)
        {
            _cache.TryRemove(key, out _);
        }
    }

    private void InvalidateStatusAndListCaches()
    {
        var keysToRemove = _cache.Keys.Where(key =>
            key.StartsWith("status_") ||
            key.StartsWith("list_")).ToList();

        foreach (var key in keysToRemove)
        {
            _cache.TryRemove(key, out _);
        }
    }

    private void CleanupExpiredEntries(object? state)
    {
        var heatThreshold = _settingsService.GetSetting(HeatThresholdKey, DefaultHeatThreshold);
        var expiredKeys = new List<string>();

        foreach (var kvp in _cache)
        {
            var entry = kvp.Value;
            if (entry.Heat < heatThreshold * 0.5) // Remove entries that are very cold
            {
                expiredKeys.Add(kvp.Key);
            }
        }

        foreach (var key in expiredKeys)
        {
            _cache.TryRemove(key, out _);
        }

        if (expiredKeys.Count > 0)
        {
            DebugService.LogDebug("Cache cleanup: removed {0} expired entries", expiredKeys.Count);
        }
    }

    /// <summary>
    /// Gets cache statistics for debugging
    /// </summary>
    public CacheStatistics GetCacheStatistics()
    {
        var stats = new CacheStatistics
        {
            TotalEntries = _cache.Count,
            HotEntries = _cache.Values.Count(e => e.Heat > 0.7),
            WarmEntries = _cache.Values.Count(e => e.Heat > 0.3 && e.Heat <= 0.7),
            ColdEntries = _cache.Values.Count(e => e.Heat <= 0.3),
            AverageHeat = _cache.Values.Any() ? _cache.Values.Average(e => e.Heat) : 0,
            HeatThreshold = _settingsService.GetSetting(HeatThresholdKey, DefaultHeatThreshold)
        };

        return stats;
    }

    /// <summary>
    /// Clears all cached data
    /// </summary>
    public void ClearCache()
    {
        _cache.Clear();
        DebugService.LogDebug("All cache entries cleared manually");
    }

    public void Dispose()
    {
        _cleanupTimer?.Dispose();
        _cache.Clear();
    }
}

/// <summary>
/// Statistics about the cache performance
/// </summary>
public class CacheStatistics
{
    public int TotalEntries { get; set; }
    public int HotEntries { get; set; }
    public int WarmEntries { get; set; }
    public int ColdEntries { get; set; }
    public double AverageHeat { get; set; }
    public double HeatThreshold { get; set; }
}
