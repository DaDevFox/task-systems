using Microsoft.Extensions.Logging;
using TaskSystems.Shared.Services;

namespace InventoryClient.Services;

/// <summary>
/// Mock gRPC client service for inventory operations (until protobuf is set up)
/// </summary>
public class InventoryGrpcService : ServiceClientBase
{
    public InventoryGrpcService(ILogger<InventoryGrpcService> logger) : base(logger)
    {
    }

    public override string ServiceName => "Inventory";

    public override async Task<bool> ConnectAsync(string address, CancellationToken cancellationToken = default)
    {
        try
        {
            // Simulate connection delay
            await Task.Delay(1000, cancellationToken);
            
            IsConnected = true;
            Logger.LogInformation("Connected to inventory service at {Address}", address);
            return true;
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to connect to inventory service at {Address}", address);
            IsConnected = false;
            return false;
        }
    }

    public override async Task DisconnectAsync()
    {
        IsConnected = false;
        await Task.Delay(100); // Simulate disconnection delay
        Logger.LogInformation("Disconnected from inventory service");
    }

    // Mock methods that will be replaced with real gRPC calls once protobuf is working
    public async Task<bool> UpdateInventoryLevelAsync(string itemId, double newLevel, string reason = "", bool recordConsumption = true)
    {
        await Task.Delay(500); // Simulate network call
        Logger.LogInformation("Updated inventory level for item {ItemId} to {NewLevel}", itemId, newLevel);
        return true;
    }
}
