using System;
using System.Linq;
using System.Threading.Tasks;
using FluentAssertions;
using Grpc.Net.Client;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using InventoryClient.IntegrationTests.Infrastructure;
using Inventory.V1;
using Xunit;

namespace InventoryClient.IntegrationTests;

/// <summary>
/// Tests to verify backend/frontend data consistency with managed server instances
/// </summary>
public class BackendConsistencyTests : IntegrationTestBase
{
    private GrpcChannel? _grpcChannel;
    private InventoryService.InventoryServiceClient? _directClient;

    public override async Task InitializeAsync()
    {
        await base.InitializeAsync();

        // Create direct gRPC client for comparison
        _grpcChannel = GrpcChannel.ForAddress($"http://{ServerAddress}");
        _directClient = new InventoryService.InventoryServiceClient(_grpcChannel);
    }

    [Fact]
    public async Task ItemCount_ShouldMatchBetweenDirectGrpcAndUIService()
    {
        // Arrange - Add some test data directly via backend
        var addRequests = new[]
        {
            new AddInventoryItemRequest { Name = "Test Item 1", Description = "First test", InitialLevel = 5, MaxCapacity = 10, LowStockThreshold = 2, UnitId = "kg" },
            new AddInventoryItemRequest { Name = "Test Item 2", Description = "Second test", InitialLevel = 1, MaxCapacity = 8, LowStockThreshold = 2, UnitId = "kg" },
            new AddInventoryItemRequest { Name = "Test Item 3", Description = "Third test", InitialLevel = 0, MaxCapacity = 6, LowStockThreshold = 1, UnitId = "liters" }
        };

        foreach (var request in addRequests)
        {
            await _directClient!.AddInventoryItemAsync(request);
        }

        // Act - Get data through both paths
        var directResponse = await _directClient!.ListInventoryItemsAsync(new ListInventoryItemsRequest
        {
            LowStockOnly = false,
            Limit = 100,
            Offset = 0
        });

        await InventoryService.ConnectAsync(ServerAddress);
        var (uiItems, uiTotalCount) = await InventoryService.ListInventoryItemsAsync(
            lowStockOnly: false,
            unitTypeFilter: null,
            limit: 100,
            offset: 0);

        // Assert - Both should return the same data
        var directItemCount = directResponse.Items.Count;
        var uiItemCount = uiItems.Count();

        Logger.LogInformation("Direct gRPC: {DirectCount} items, UI Service: {UICount} items, UI Total: {UITotal}",
            directItemCount, uiItemCount, uiTotalCount);

        uiItemCount.Should().Be(directItemCount, "UI service should return same count as direct gRPC");
        uiTotalCount.Should().Be(directResponse.TotalCount, "UI total count should match gRPC total");

        // Verify item details match
        var uiItemsList = uiItems.OrderBy(i => i.Id).ToList();
        var directItemsList = directResponse.Items.OrderBy(i => i.Id).ToList();

        for (int i = 0; i < Math.Min(directItemCount, uiItemCount); i++)
        {
            var directItem = directItemsList[i];
            var uiItem = uiItemsList[i];

            uiItem.Id.Should().Be(directItem.Id, $"Item {i} ID should match");
            uiItem.Name.Should().Be(directItem.Name, $"Item {i} name should match");
            uiItem.CurrentLevel.Should().Be(directItem.CurrentLevel, $"Item {i} level should match");
            uiItem.MaxCapacity.Should().Be(directItem.MaxCapacity, $"Item {i} capacity should match");
            uiItem.LowStockThreshold.Should().Be(directItem.LowStockThreshold, $"Item {i} threshold should match");
        }
    }

    [Fact]
    public async Task LowStockFilter_ShouldMatchDirectGrpcResults()
    {
        // Arrange - Add items with known stock levels
        var items = new[]
        {
            new { Name = "High Stock Item", Level = 8.0, Threshold = 2.0 },
            new { Name = "Low Stock Item", Level = 1.5, Threshold = 2.0 },
            new { Name = "Empty Item", Level = 0.0, Threshold = 1.0 },
            new { Name = "Medium Stock Item", Level = 5.0, Threshold = 2.0 }
        };

        foreach (var item in items)
        {
            await _directClient!.AddInventoryItemAsync(new AddInventoryItemRequest
            {
                Name = item.Name,
                Description = $"Test item: {item.Name}",
                InitialLevel = item.Level,
                MaxCapacity = 10.0,
                LowStockThreshold = item.Threshold,
                UnitId = "kg"
            });
        }

        // Act - Get low stock items through both paths
        var directLowStock = await _directClient!.ListInventoryItemsAsync(new ListInventoryItemsRequest
        {
            LowStockOnly = true,
            Limit = 100,
            Offset = 0
        });

        await InventoryService.ConnectAsync(ServerAddress);
        var (uiLowStockItems, uiLowStockTotal) = await InventoryService.ListInventoryItemsAsync(
            lowStockOnly: true,
            unitTypeFilter: null,
            limit: 100,
            offset: 0);

        // Assert
        var directLowStockCount = directLowStock.Items.Count;
        var uiLowStockCount = uiLowStockItems.Count();

        Logger.LogInformation("Low stock - Direct: {DirectCount}, UI: {UICount}, Expected: 2",
            directLowStockCount, uiLowStockCount);

        uiLowStockCount.Should().Be(directLowStockCount,
            "UI service should return same low stock count as direct gRPC");
        uiLowStockTotal.Should().Be(directLowStock.TotalCount,
            "UI low stock total should match gRPC total");

        // We expect 2 items: "Low Stock Item" (1.5 < 2.0) and "Empty Item" (0.0 < 1.0)
        directLowStockCount.Should().Be(2, "Should have exactly 2 low stock items");
    }

    [Fact]
    public async Task CacheConsistency_ShouldMaintainDataIntegrity()
    {
        // Arrange
        await InventoryService.ConnectAsync(ServerAddress);

        // Add an item
        await InventoryService.AddInventoryItemAsync("Cache Test Item", "Testing cache", 3.0, 10.0, 2.0, "kg");

        // Act - Make multiple calls to test cache behavior
        var firstCall = await InventoryService.ListInventoryItemsAsync();
        await Task.Delay(100); // Brief pause
        var secondCall = await InventoryService.ListInventoryItemsAsync();

        // Add another item
        await InventoryService.AddInventoryItemAsync("Cache Test Item 2", "Testing cache 2", 1.0, 5.0, 2.0, "kg");

        // Get fresh data (should bypass/invalidate cache)
        var thirdCall = await InventoryService.ListInventoryItemsAsync();

        // Assert
        var firstCount = firstCall.Items.Count();
        var secondCount = secondCall.Items.Count();
        var thirdCount = thirdCall.Items.Count();

        Logger.LogInformation("Cache test - First: {First}, Second: {Second}, Third: {Third}",
            firstCount, secondCount, thirdCount);

        firstCount.Should().Be(secondCount, "Cache should return consistent data");
        thirdCount.Should().Be(firstCount + 1, "Fresh data should include the new item");
    }

    public override async Task DisposeAsync()
    {
        _grpcChannel?.Dispose();
        await base.DisposeAsync();
    }
}
