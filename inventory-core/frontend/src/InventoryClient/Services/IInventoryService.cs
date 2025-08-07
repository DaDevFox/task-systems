using InventoryClient.Models;

namespace InventoryClient.Services;

/// <summary>
/// Interface for inventory service operations
/// </summary>
public interface IInventoryService
{
    /// <summary>
    /// Gets whether the service is currently connected
    /// </summary>
    bool IsConnected { get; }

    /// <summary>
    /// Connects to the inventory service
    /// </summary>
    Task<bool> ConnectAsync(string address, CancellationToken cancellationToken = default);

    /// <summary>
    /// Disconnects from the inventory service
    /// </summary>
    Task DisconnectAsync();

    /// <summary>
    /// Pings the service to check connectivity
    /// </summary>
    Task<bool> PingAsync();

    /// <summary>
    /// Gets the overall inventory status
    /// </summary>
    Task<InventoryStatusViewModel> GetInventoryStatusAsync(bool lowStockOnly = false, IEnumerable<string>? itemIds = null);

    /// <summary>
    /// Gets a specific inventory item
    /// </summary>
    Task<InventoryItemViewModel?> GetInventoryItemAsync(string itemId);

    /// <summary>
    /// Lists inventory items with optional filtering
    /// </summary>
    Task<(IEnumerable<InventoryItemViewModel> Items, int TotalCount)> ListInventoryItemsAsync(
        bool lowStockOnly = false, 
        string? unitTypeFilter = null, 
        int limit = 100, 
        int offset = 0);

    /// <summary>
    /// Updates the inventory level for an item
    /// </summary>
    Task<bool> UpdateInventoryLevelAsync(string itemId, double newLevel, string reason = "", bool recordConsumption = true);

    /// <summary>
    /// Adds a new inventory item
    /// </summary>
    Task<InventoryItemViewModel?> AddInventoryItemAsync(
        string name, 
        string description, 
        double initialLevel, 
        double maxCapacity, 
        double lowStockThreshold, 
        string unitId,
        Dictionary<string, string>? metadata = null);

    /// <summary>
    /// Predicts consumption for an item
    /// </summary>
    Task<ConsumptionPredictionViewModel?> PredictConsumptionAsync(string itemId, int daysAhead = 30, bool updateBehavior = false);

    /// <summary>
    /// Sets consumption behavior for an item
    /// </summary>
    Task<bool> SetConsumptionBehaviorAsync(string itemId, ConsumptionBehaviorViewModel behavior);

    /// <summary>
    /// Converts units between different measurement types
    /// </summary>
    Task<(double ConvertedAmount, bool Success, string? ErrorMessage)> ConvertUnitsAsync(double amount, string fromUnitId, string toUnitId);
}
