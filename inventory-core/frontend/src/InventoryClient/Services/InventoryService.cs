using Grpc.Net.Client;
using Microsoft.Extensions.Logging;
using TaskSystems.Shared.Services;
using InventoryClient.Models;
using Inventory.V1;
// using Google.Protobuf.WellKnownTypes;

namespace InventoryClient.Services;

/// <summary>
/// Real gRPC client service for inventory operations
/// </summary>
public class InventoryGrpcService : ServiceClientBase, IInventoryService
{
    private const string NotConnectedMessage = "Not connected to inventory service";
    private GrpcChannel? _channel;
    private InventoryService.InventoryServiceClient? _client;

    public InventoryGrpcService(ILogger<InventoryGrpcService> logger) : base(logger)
    {
    }

    public override string ServiceName => "Inventory";

    public override async Task<bool> ConnectAsync(string address, CancellationToken cancellationToken = default)
    {
        try
        {
            IsConnected = false;
            Logger.LogInformation("Attempting to connect to inventory service at {Address}", address);

            // Ensure the address has a proper scheme
            string formattedAddress = address;
            if (!address.StartsWith("http://") && !address.StartsWith("https://"))
                formattedAddress = $"http://{address}";

            // Create gRPC channel
            _channel = GrpcChannel.ForAddress(formattedAddress);
            _client = new InventoryService.InventoryServiceClient(_channel);

            // Test the connection with a simple ping call
            await _client.GetInventoryStatusAsync(new GetInventoryStatusRequest(),
                cancellationToken: cancellationToken);

            IsConnected = true;
            Logger.LogInformation("Successfully connected to inventory service at {Address}", formattedAddress);
            return true;
        }
        catch (OperationCanceledException ex)
        {
            Logger.LogWarning(ex, "Connection to inventory service at {Address} was cancelled", address);
            await CleanupConnection();
            return false;
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to connect to inventory service at {Address}", address);
            await CleanupConnection();
            return false;
        }
    }

    public override async Task DisconnectAsync()
    {
        await CleanupConnection();
        Logger.LogInformation("Disconnected from inventory service");
    }

    private async Task CleanupConnection()
    {
        IsConnected = false;
        _client = null;

        if (_channel != null)
        {
            await _channel.ShutdownAsync();
            _channel?.Dispose();
            _channel = null;
        }
    }

    public async Task<bool> PingAsync()
    {
        if (!IsConnected || _client == null) return false;

        try
        {
            await _client.GetInventoryStatusAsync(new GetInventoryStatusRequest
            {
                IncludeLowStockOnly = false
            });
            return true;
        }
        catch
        {
            IsConnected = false;
            return false;
        }
    }

    public async Task<InventoryStatusViewModel> GetInventoryStatusAsync(bool lowStockOnly = false, IEnumerable<string>? itemIds = null)
    {
        if (!IsConnected || _client == null)
            throw new InvalidOperationException(NotConnectedMessage);

        try
        {
            var request = new GetInventoryStatusRequest
            {
                IncludeLowStockOnly = lowStockOnly
            };

            if (itemIds != null)
                request.ItemIds.AddRange(itemIds);

            var response = await _client.GetInventoryStatusAsync(request);
            return MapToInventoryStatusViewModel(response.Status);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to get inventory status");
            throw new InvalidOperationException("Failed to retrieve inventory status", ex);
        }
    }

    public async Task<InventoryItemViewModel?> GetInventoryItemAsync(string itemId)
    {
        if (!IsConnected || _client == null)
            throw new InvalidOperationException(NotConnectedMessage);

        try
        {
            var request = new GetInventoryItemRequest { ItemId = itemId };
            var response = await _client.GetInventoryItemAsync(request);
            return MapToInventoryItemViewModel(response.Item);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to get inventory item {ItemId}", itemId);
            throw new InvalidOperationException($"Failed to retrieve inventory item {itemId}", ex);
        }
    }

    public async Task<(IEnumerable<InventoryItemViewModel> Items, int TotalCount)> ListInventoryItemsAsync(
        bool lowStockOnly = false, string? unitTypeFilter = null, int limit = 100, int offset = 0)
    {
        if (!IsConnected || _client == null)
            throw new InvalidOperationException(NotConnectedMessage);

        try
        {
            var request = new ListInventoryItemsRequest
            {
                LowStockOnly = lowStockOnly,
                UnitTypeFilter = unitTypeFilter ?? string.Empty,
                Limit = limit,
                Offset = offset
            };

            var response = await _client.ListInventoryItemsAsync(request);
            var items = response.Items.Select(MapToInventoryItemViewModel);
            return (items, response.TotalCount);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to list inventory items");
            throw;
        }
    }

    public async Task<bool> UpdateInventoryLevelAsync(string itemId, double newLevel, string reason = "", bool recordConsumption = true)
    {
        if (!IsConnected || _client == null)
            throw new InvalidOperationException(NotConnectedMessage);

        try
        {
            var request = new UpdateInventoryLevelRequest
            {
                ItemId = itemId,
                NewLevel = newLevel,
                Reason = reason,
                RecordConsumption = recordConsumption
            };

            var response = await _client.UpdateInventoryLevelAsync(request);
            return response.LevelChanged;
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to update inventory level for item {ItemId}", itemId);
            return false;
        }
    }

    public async Task<InventoryItemViewModel?> AddInventoryItemAsync(string name, string description, double initialLevel,
        double maxCapacity, double lowStockThreshold, string unitId, Dictionary<string, string>? metadata = null)
    {
        if (!IsConnected || _client == null)
            throw new InvalidOperationException(NotConnectedMessage);

        try
        {
            var request = new AddInventoryItemRequest
            {
                Name = name,
                Description = description,
                InitialLevel = initialLevel,
                MaxCapacity = maxCapacity,
                LowStockThreshold = lowStockThreshold,
                UnitId = unitId
            };

            if (metadata != null)
                request.Metadata.Add(metadata);

            var response = await _client.AddInventoryItemAsync(request);
            return MapToInventoryItemViewModel(response.Item);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to add inventory item {0}", name);
            return null;
        }
    }

    // public async Task<ConsumptionPredictionViewModel?> PredictConsumptionAsync(string itemId, int daysAhead = 30, bool updateBehavior = false)
    // {
    //     if (!IsConnected || _client == null)
    //         throw new InvalidOperationException(NotConnectedMessage);
    //
    //     try
    //     {
    //         var request = new PredictConsumptionRequest
    //         {
    //             ItemId = itemId,
    //             DaysAhead = daysAhead,
    //             UpdateBehavior = updateBehavior
    //         };
    //
    //         var response = await _client.PredictConsumptionAsync(request);
    //         return MapToConsumptionPredictionViewModel(response.Prediction);
    //     }
    //     catch (Exception ex)
    //     {
    //         Logger.LogError(ex, "Failed to predict consumption for item {ItemId}", itemId);
    //         return null;
    //     }
    // }

    // public async Task<bool> SetConsumptionBehaviorAsync(string itemId, ConsumptionBehaviorViewModel behavior)
    // {
    //     if (!IsConnected || _client == null)
    //         throw new InvalidOperationException(NotConnectedMessage);
    //
    //     try
    //     {
    //         var request = new SetConsumptionBehaviorRequest
    //         {
    //             ItemId = itemId,
    //             Behavior = MapFromConsumptionBehaviorViewModel(behavior)
    //         };
    //
    //         await _client.SetConsumptionBehaviorAsync(request);
    //         return true;
    //     }
    //     catch (Exception ex)
    //     {
    //         Logger.LogError(ex, "Failed to set consumption behavior for item {ItemId}", itemId);
    //         return false;
    //     }
    // }

    public async Task<(double ConvertedAmount, bool Success, string? ErrorMessage)> ConvertUnitsAsync(double amount, string fromUnitId, string toUnitId)
    {
        if (!IsConnected || _client == null)
            throw new InvalidOperationException(NotConnectedMessage);

        try
        {
            var request = new ConvertUnitsRequest
            {
                Amount = amount,
                FromUnitId = fromUnitId,
                ToUnitId = toUnitId
            };

            var response = await _client.ConvertUnitsAsync(request);
            return (response.ConvertedAmount, response.ConversionPossible, response.ErrorMessage);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to convert units from {FromUnit} to {ToUnit}", fromUnitId, toUnitId);
            return (0, false, ex.Message);
        }
    }

    public async Task<bool> RemoveInventoryItemAsync(string itemId)
    {
        if (!IsConnected || _client == null)
            throw new InvalidOperationException(NotConnectedMessage);

        try
        {
            var request = new RemoveInventoryItemRequest { ItemId = itemId };
            var response = await _client.RemoveInventoryItemAsync(request);

            Logger.LogInformation("Successfully removed inventory item {ItemId}: {ItemName}",
                response.RemovedItemId, response.RemovedItemName);

            return response.ItemRemoved;
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to remove inventory item {ItemId}", itemId);
            return false;
        }
    }

    // Mapping methods
    private static InventoryStatusViewModel MapToInventoryStatusViewModel(InventoryStatus status)
    {
        return new InventoryStatusViewModel
        {
            Items = status.Items.Select(MapToInventoryItemViewModel).ToList(),
            LowStockItems = status.LowStockItems.Select(MapToInventoryItemViewModel).ToList(),
            EmptyItems = status.EmptyItems.Select(MapToInventoryItemViewModel).ToList(),
            TotalItems = status.TotalItems,
            LastUpdated = status.LastUpdated?.ToDateTime() ?? DateTime.MinValue
        };
    }

    private static InventoryItemViewModel MapToInventoryItemViewModel(InventoryItem item)
    {
        return new InventoryItemViewModel
        {
            Id = item.Id,
            Name = item.Name,
            Description = item.Description,
            CurrentLevel = item.CurrentLevel,
            MaxCapacity = item.MaxCapacity,
            LowStockThreshold = item.LowStockThreshold,
            UnitId = item.UnitId,
            AlternateUnitIds = item.AlternateUnitIds.ToList(),
            LastUpdated = item.UpdatedAt?.ToDateTime() ?? DateTime.MinValue,
            // ConsumptionBehavior = item.ConsumptionBehavior != null ? MapToConsumptionBehaviorViewModel(item.ConsumptionBehavior) : null,
            Metadata = item.Metadata.ToDictionary(kvp => kvp.Key, kvp => kvp.Value)
        };
    }

    // private static ConsumptionPredictionViewModel MapToConsumptionPredictionViewModel(ConsumptionPrediction prediction)
    // {
    //     return new ConsumptionPredictionViewModel
    //     {
    //         ItemId = prediction.ItemId,
    //         PredictedDaysRemaining = prediction.PredictedDaysRemaining,
    //         ConfidenceScore = prediction.ConfidenceScore,
    //         PredictedEmptyDate = prediction.PredictedEmptyDate?.ToDateTime() ?? DateTime.MinValue,
    //         RecommendedRestockLevel = prediction.RecommendedRestockLevel,
    //         PredictionModel = prediction.PredictionModel,
    //         Estimate = prediction.Estimate,
    //         LowerBound = prediction.LowerBound,
    //         UpperBound = prediction.UpperBound,
    //         Recommendation = prediction.Recommendation
    //     };
    // }
    //
    // private static ConsumptionBehaviorViewModel MapToConsumptionBehaviorViewModel(ConsumptionBehavior behavior)
    // {
    //     return new ConsumptionBehaviorViewModel
    //     {
    //         Pattern = (Models.ConsumptionPattern)(int)behavior.Pattern,
    //         AverageRatePerDay = behavior.AverageRatePerDay,
    //         Variance = behavior.Variance,
    //         SeasonalFactors = behavior.SeasonalFactors.ToList(),
    //         LastUpdated = behavior.LastUpdated?.ToDateTime() ?? DateTime.MinValue
    //     };
    // }
    //
    // private static ConsumptionBehavior MapFromConsumptionBehaviorViewModel(ConsumptionBehaviorViewModel behavior)
    // {
    //     var result = new ConsumptionBehavior
    //     {
    //         Pattern = (Inventory.V1.ConsumptionPattern)(int)behavior.Pattern,
    //         AverageRatePerDay = behavior.AverageRatePerDay,
    //         Variance = behavior.Variance,
    //         LastUpdated = Timestamp.FromDateTime(behavior.LastUpdated.ToUniversalTime())
    //     };
    //     result.SeasonalFactors.AddRange(behavior.SeasonalFactors);
    //     return result;
    // }
    public async Task<IReadOnlyList<InventoryLevelSnapshotViewModel>> GetItemHistoryAsync(
        string itemId,
        DateTime? startTime = null,
        DateTime? endTime = null,
        string? granularity = null,
        int? maxPoints = null,
        CancellationToken cancellationToken = default)
    {
        if (!IsConnected || _client == null)
            throw new InvalidOperationException(NotConnectedMessage);

        try
        {
            var request = new GetItemHistoryRequest
            {
                ItemId = itemId
            };
            if (startTime.HasValue)
                request.StartTime = Google.Protobuf.WellKnownTypes.Timestamp.FromDateTime(startTime.Value.ToUniversalTime());
            if (endTime.HasValue)
                request.EndTime = Google.Protobuf.WellKnownTypes.Timestamp.FromDateTime(endTime.Value.ToUniversalTime());
            if (!string.IsNullOrWhiteSpace(granularity) && Enum.TryParse<HistoryGranularity>(granularity, true, out var parsedGranularity))
                request.Granularity = parsedGranularity;
            if (maxPoints.HasValue)
                request.MaxPoints = maxPoints.Value;

            var response = await _client.GetItemHistoryAsync(request, cancellationToken: cancellationToken);
            return response.History.Select(s => new InventoryLevelSnapshotViewModel
            {
                Timestamp = s.Timestamp?.ToDateTime() ?? DateTime.MinValue,
                Level = s.Level,
                UnitId = s.UnitId,
                Source = s.Source,
                Context = s.Context,
                Metadata = s.Metadata.ToDictionary(kvp => kvp.Key, kvp => kvp.Value)
            }).ToList();
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to fetch item history for {ItemId}", itemId);
            throw new InvalidOperationException($"Failed to fetch item history for {itemId}", ex);
        }
    }
}
