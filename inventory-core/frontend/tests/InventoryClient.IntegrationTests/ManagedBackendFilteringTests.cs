using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using FluentAssertions;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using InventoryClient.IntegrationTests.Infrastructure;
using Xunit;

namespace InventoryClient.IntegrationTests;

/// <summary>
/// Integration tests that verify backend filtering behavior against isolated backend instances
/// </summary>
public class ManagedBackendFilteringTests : IntegrationTestBase
{
    /// <summary>
    /// Test basic inventory CRUD operations work with managed backend
    /// </summary>
    [Fact(Timeout = 15000)]
    public async Task BackendOperations_WithManagedBackend_ShouldWorkCorrectly()
    {
        // Arrange - Connect to backend
        await InventoryService.ConnectAsync(ServerAddress);
        Logger.LogInformation("Connected to backend at {Address}", ServerAddress);

        // Act - Add some test inventory items
        var testItems = new[]
        {
            ("Test Widget A", "Electronics widget type A", 10.0, 20.0, 2.0, "kg"),
            ("Test Widget B", "Electronics widget type B", 5.0, 15.0, 2.0, "kg"),
            ("Office Supplies", "Basic office supplies", 20.0, 50.0, 5.0, "kg")
        };

        var addedItems = new List<string>();

        Logger.LogInformation("Adding {Count} test items to backend", testItems.Length);

        foreach (var (name, description, level, maxCapacity, threshold, unit) in testItems)
        {
            var item = await InventoryService.AddInventoryItemAsync(
                name, description, level, maxCapacity, threshold, unit);

            item.Should().NotBeNull($"Should successfully add item {name}");
            addedItems.Add(item!.Id);
            Logger.LogDebug("Added item: {Name} with ID {Id}", name, item.Id);
        }

        // Assert - List all items (no filtering)
        var (allItems, totalCount) = await InventoryService.ListInventoryItemsAsync(limit: 100);

        allItems.Should().HaveCountGreaterOrEqualTo(3, "Should have at least the 3 items we added");
        totalCount.Should().BeGreaterOrEqualTo(3, "Total count should include our items");

        var itemNames = allItems.Select(i => i.Name).ToList();
        itemNames.Should().Contain("Test Widget A");
        itemNames.Should().Contain("Test Widget B");
        itemNames.Should().Contain("Office Supplies");

        Logger.LogInformation("=== Backend CRUD operations test passed ===");
    }

    /// <summary>
    /// Test that low stock filtering works correctly
    /// </summary>
    [Fact(Timeout = 15000)]
    public async Task LowStockFiltering_WithManagedBackend_ShouldWorkCorrectly()
    {
        // Arrange - Connect to backend
        await InventoryService.ConnectAsync(ServerAddress);

        // Add items with different stock levels
        var normalStockItem = await InventoryService.AddInventoryItemAsync(
            "Normal Stock Item", "Item with normal stock", 10.0, 20.0, 2.0, "kg");

        var lowStockItem = await InventoryService.AddInventoryItemAsync(
            "Low Stock Item", "Item with low stock", 1.0, 20.0, 2.0, "kg");

        var emptyItem = await InventoryService.AddInventoryItemAsync(
            "Empty Item", "Item that is empty", 0.0, 20.0, 2.0, "kg");

        normalStockItem.Should().NotBeNull();
        lowStockItem.Should().NotBeNull();
        emptyItem.Should().NotBeNull();

        Logger.LogInformation("Added test items - Normal: {Normal}, Low: {Low}, Empty: {Empty}",
            normalStockItem!.IsLowStock, lowStockItem!.IsLowStock, emptyItem!.IsEmpty);

        // Act - Filter for low stock items only
        var (lowStockItems, _) = await InventoryService.ListInventoryItemsAsync(
            lowStockOnly: true, limit: 100);

        // Assert - Should only return low stock and empty items
        var lowStockIds = lowStockItems.Select(i => i.Id).ToList();

        // The exact count depends on whether low stock includes empty items
        // but we should definitely see our low stock and empty items
        lowStockIds.Should().Contain(lowStockItem.Id, "Low stock item should be in results");
        lowStockIds.Should().Contain(emptyItem.Id, "Empty item should be in results");
        lowStockIds.Should().NotContain(normalStockItem.Id, "Normal stock item should not be in results");

        Logger.LogInformation("Low stock filtering found {Count} items", lowStockItems.Count());
        Logger.LogInformation("=== Low stock filtering test passed ===");
    }

    /// <summary>
    /// Test that empty filters return all results
    /// </summary>
    [Fact(Timeout = 10000)]
    public async Task ListItems_WithVariousParameters_ShouldReturnCorrectResults()
    {
        // Arrange - Connect and add a test item
        await InventoryService.ConnectAsync(ServerAddress);

        var testItem = await InventoryService.AddInventoryItemAsync(
            "Test Item", "A test item", 5.0, 10.0, 1.0, "kg");
        testItem.Should().NotBeNull();

        // Act & Assert - Test different parameter combinations

        // Test basic list
        var (basicItems, basicCount) = await InventoryService.ListInventoryItemsAsync();
        basicItems.Should().NotBeEmpty("Should return items with default parameters");
        basicCount.Should().BeGreaterThan(0, "Total count should be positive");

        // Test with limit
        var (limitedItems, limitedCount) = await InventoryService.ListInventoryItemsAsync(limit: 1);
        limitedItems.Should().HaveCount(1, "Should respect limit parameter");
        limitedCount.Should().BeGreaterOrEqualTo(1, "Total count should still reflect all items");

        // Test with offset
        if (basicItems.Count() > 1)
        {
            var (offsetItems, _) = await InventoryService.ListInventoryItemsAsync(
                offset: 1, limit: 100);
            offsetItems.Count().Should().Be(basicItems.Count() - 1, "Should skip first item with offset");
        }

        Logger.LogInformation("List items parameter testing passed");
    }

    /// <summary>
    /// Test inventory status retrieval
    /// </summary>
    [Fact(Timeout = 10000)]
    public async Task GetInventoryStatus_WithManagedBackend_ShouldReturnValidStatus()
    {
        // Arrange
        await InventoryService.ConnectAsync(ServerAddress);

        // Add some test items to ensure we have data
        await InventoryService.AddInventoryItemAsync(
            "Status Test Item", "Item for status testing", 5.0, 10.0, 2.0, "kg");

        // Act - Get overall inventory status
        var status = await InventoryService.GetInventoryStatusAsync();

        // Assert
        status.Should().NotBeNull("Should return inventory status");
        status.TotalItems.Should().BeGreaterThan(0, "Should have at least one item");
        status.Items.Should().NotBeEmpty("Should have item details");

        Logger.LogInformation("Inventory status: {TotalItems} items, {Summary}",
            status.TotalItems, status.StatusSummary);

        // Test low stock only status
        var lowStockStatus = await InventoryService.GetInventoryStatusAsync(lowStockOnly: true);
        lowStockStatus.Should().NotBeNull("Should return low stock status");

        Logger.LogInformation("=== Inventory status test passed ===");
    }

    /// <summary>
    /// Test individual item retrieval and updates
    /// </summary>
    [Fact(Timeout = 10000)]
    public async Task ItemOperations_WithManagedBackend_ShouldWorkCorrectly()
    {
        // Arrange
        await InventoryService.ConnectAsync(ServerAddress);

        // Add a test item
        var originalItem = await InventoryService.AddInventoryItemAsync(
            "Update Test Item", "Item for update testing", 10.0, 20.0, 2.0, "kg");
        originalItem.Should().NotBeNull();

        // Act & Assert - Get individual item
        var retrievedItem = await InventoryService.GetInventoryItemAsync(originalItem!.Id);
        retrievedItem.Should().NotBeNull("Should retrieve item by ID");
        retrievedItem!.Name.Should().Be("Update Test Item");
        retrievedItem.CurrentLevel.Should().Be(10.0);

        // Update inventory level
        var updateResult = await InventoryService.UpdateInventoryLevelAsync(
            originalItem.Id, 5.0, "Test update");
        updateResult.Should().BeTrue("Should successfully update inventory level");

        // Verify update
        var updatedItem = await InventoryService.GetInventoryItemAsync(originalItem.Id);
        updatedItem.Should().NotBeNull("Should still retrieve item after update");
        updatedItem!.CurrentLevel.Should().Be(5.0, "Should reflect updated level");

        Logger.LogInformation("=== Item operations test passed ===");
    }

    /// <summary>
    /// Test ping functionality to verify connectivity
    /// </summary>
    [Fact(Timeout = 5000)]
    public async Task Ping_WithManagedBackend_ShouldReturnTrue()
    {
        // Arrange
        await InventoryService.ConnectAsync(ServerAddress);
        InventoryService.IsConnected.Should().BeTrue("Should be connected after ConnectAsync");

        // Act
        var pingResult = await InventoryService.PingAsync();

        // Assert
        pingResult.Should().BeTrue("Ping should succeed when connected to managed backend");

        Logger.LogInformation("=== Ping test passed ===");
    }
}
