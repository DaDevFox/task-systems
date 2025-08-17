using Xunit;
using FluentAssertions;
using Grpc.Net.Client;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using InventoryClient.Services;
using InventoryClient.ViewModels;
using Inventory.V1;
using TaskSystems.Shared.Services;
using System;
using System.Threading.Tasks;

namespace InventoryClient.IntegrationTests;

/// <summary>
/// Integration tests that verify the UI state matches the backend gRPC service
/// </summary>
public class InventoryUIIntegrationTests : IDisposable
{
    private readonly IHost _host;
    private readonly IInventoryService _inventoryService;
    private readonly InventoryService.InventoryServiceClient _directGrpcClient;
    private readonly GrpcChannel _grpcChannel;
    private readonly ILogger<InventoryUIIntegrationTests> _logger;

    public InventoryUIIntegrationTests()
    {
        // Set up the dependency injection container like the real app
        _host = Host.CreateDefaultBuilder()
            .ConfigureServices((context, services) =>
            {
                services.AddLogging(builder =>
                {
                    builder.AddConsole();
                    builder.SetMinimumLevel(LogLevel.Information);
                });

                // Register settings service
                services.AddSingleton<ISettingsService>(provider =>
                {
                    var settingsService = new JsonSettingsService();
                    settingsService.LoadAsync().Wait();
                    return settingsService;
                });

                // Register inventory services
                services.AddSingleton<InventoryGrpcService>();
                services.AddSingleton<IServiceClient>(provider => provider.GetRequiredService<InventoryGrpcService>());

                // Register cached inventory service
                services.AddSingleton<IInventoryService>(provider =>
                {
                    var grpcService = provider.GetRequiredService<InventoryGrpcService>();
                    var settingsService = provider.GetRequiredService<ISettingsService>();
                    var logger = provider.GetRequiredService<ILogger<CachedInventoryService>>();
                    return new CachedInventoryService(grpcService, settingsService, logger);
                });
            })
            .Build();

        _inventoryService = _host.Services.GetRequiredService<IInventoryService>();
        _logger = _host.Services.GetRequiredService<ILogger<InventoryUIIntegrationTests>>();

        // Create a direct gRPC client for comparison
        _grpcChannel = GrpcChannel.ForAddress("http://localhost:50052");
        _directGrpcClient = new InventoryService.InventoryServiceClient(_grpcChannel);
    }

    [Fact]
    public async Task ItemCount_InUI_ShouldMatchDirectGrpcCall()
    {
        // Arrange
        const string serverAddress = "localhost:50052";

        // Connect both the UI service and direct gRPC client
        var uiConnected = await _inventoryService.ConnectAsync(serverAddress);
        uiConnected.Should().BeTrue("UI service should connect to backend");

        // Test direct gRPC connection
        var directGrpcResponse = await _directGrpcClient.ListInventoryItemsAsync(new ListInventoryItemsRequest
        {
            LowStockOnly = false,
            Limit = 1000,
            Offset = 0
        });

        // Act - Get data through UI service (with caching)
        var (uiItems, uiTotalCount) = await _inventoryService.ListInventoryItemsAsync(
            lowStockOnly: false,
            unitTypeFilter: null,
            limit: 1000,
            offset: 0);

        // Assert - Both should return the same number of items
        var directItemCount = directGrpcResponse.Items.Count;
        var uiItemCount = uiItems.Count();

        _logger.LogInformation("Direct gRPC call returned {DirectCount} items", directItemCount);
        _logger.LogInformation("UI service returned {UICount} items", uiItemCount);
        _logger.LogInformation("UI total count: {UITotalCount}", uiTotalCount);

        uiItemCount.Should().Be(directItemCount,
            "UI service should return the same number of items as direct gRPC call");
        uiTotalCount.Should().Be(directGrpcResponse.TotalCount,
            "UI total count should match gRPC total count");

        // Verify item details match
        var uiItemsList = uiItems.ToList();
        for (int i = 0; i < Math.Min(directItemCount, uiItemCount); i++)
        {
            var directItem = directGrpcResponse.Items[i];
            var uiItem = uiItemsList[i];

            uiItem.Id.Should().Be(directItem.Id, $"Item {i} ID should match");
            uiItem.Name.Should().Be(directItem.Name, $"Item {i} name should match");
            uiItem.CurrentLevel.Should().Be(directItem.CurrentLevel, $"Item {i} level should match");
        }
    }

    [Fact]
    public async Task FilteredItemCount_InMainViewModel_ShouldReflectActualFilteredData()
    {
        // Arrange
        const string serverAddress = "localhost:50052";

        var serviceClient = _host.Services.GetRequiredService<IServiceClient>();
        var settingsService = _host.Services.GetRequiredService<ISettingsService>();
        var logger = _host.Services.GetRequiredService<ILogger<MainViewModel>>();

        var viewModel = new MainViewModel(_inventoryService, serviceClient, settingsService, logger);

        // Connect and load data
        await _inventoryService.ConnectAsync(serverAddress);
        await viewModel.RefreshCommand.ExecuteAsync(null);

        // Wait a bit for async loading
        await Task.Delay(1000);

        // Act - Test without any filters
        viewModel.ShowLowStockOnly = false;
        viewModel.SearchText = "";

        await Task.Delay(100); // Allow UI updates

        var totalItemCount = viewModel.InventoryItems.Count;
        var displayedItemCount = viewModel.DisplayedItems.Count;
        var filteredItemCount = viewModel.FilteredItems.Count;

        // Assert - All counts should match when no filters applied
        _logger.LogInformation("Total items: {Total}, Displayed: {Displayed}, Filtered: {Filtered}",
            totalItemCount, displayedItemCount, filteredItemCount);

        displayedItemCount.Should().Be(totalItemCount,
            "DisplayedItems.Count should match total items when no filters applied");
        filteredItemCount.Should().Be(totalItemCount,
            "FilteredItems.Count should match total items when no filters applied");
        displayedItemCount.Should().Be(filteredItemCount,
            "DisplayedItems.Count should match FilteredItems.Count");

        // Act - Test with low stock filter
        viewModel.FilterLowStockCommand.Execute(null);
        await Task.Delay(100); // Allow UI updates

        var lowStockDisplayedCount = viewModel.DisplayedItems.Count;
        var lowStockFilteredCount = viewModel.FilteredItems.Count;
        var actualLowStockCount = viewModel.InventoryItems.Count(i => i.IsLowStock || i.IsEmpty);

        // Assert - Filtered counts should match actual low stock items
        _logger.LogInformation("Low stock - Displayed: {Displayed}, Filtered: {Filtered}, Actual: {Actual}",
            lowStockDisplayedCount, lowStockFilteredCount, actualLowStockCount);

        lowStockDisplayedCount.Should().Be(actualLowStockCount,
            "DisplayedItems.Count should match actual low stock items");
        lowStockFilteredCount.Should().Be(actualLowStockCount,
            "FilteredItems.Count should match actual low stock items");
        lowStockDisplayedCount.Should().Be(lowStockFilteredCount,
            "DisplayedItems.Count should match FilteredItems.Count with low stock filter");
    }

    [Fact]
    public async Task SearchFilter_InMainViewModel_ShouldUpdateItemCountCorrectly()
    {
        // Arrange
        const string serverAddress = "localhost:50052";

        var serviceClient = _host.Services.GetRequiredService<IServiceClient>();
        var settingsService = _host.Services.GetRequiredService<ISettingsService>();
        var logger = _host.Services.GetRequiredService<ILogger<MainViewModel>>();

        var viewModel = new MainViewModel(_inventoryService, serviceClient, settingsService, logger);

        // Connect and load data
        await _inventoryService.ConnectAsync(serverAddress);
        await viewModel.RefreshCommand.ExecuteAsync(null);

        // Wait for async loading
        await Task.Delay(1000);

        // Ensure we have some items to test with
        if (viewModel.InventoryItems.Count == 0)
        {
            // Add some test items if none exist
            await _inventoryService.AddInventoryItemAsync("Test Flour", "Test flour item", 5.0, 10.0, 2.0, "kg");
            await _inventoryService.AddInventoryItemAsync("Test Sugar", "Test sugar item", 3.0, 5.0, 1.0, "kg");
            await viewModel.RefreshCommand.ExecuteAsync(null);
            await Task.Delay(500);
        }

        // Act - Test search functionality
        viewModel.ShowLowStockOnly = false;
        viewModel.SearchText = "Test";
        viewModel.SearchItemsCommand.Execute(null);
        await Task.Delay(100); // Allow UI updates

        var searchedDisplayedCount = viewModel.DisplayedItems.Count;
        var searchedFilteredCount = viewModel.FilteredItems.Count;
        var actualSearchMatchCount = viewModel.InventoryItems.Count(i =>
            i.Name.Contains("Test", StringComparison.OrdinalIgnoreCase) ||
            i.Description.Contains("Test", StringComparison.OrdinalIgnoreCase));

        // Assert - Search results should match
        _logger.LogInformation("Search 'Test' - Displayed: {Displayed}, Filtered: {Filtered}, Actual: {Actual}",
            searchedDisplayedCount, searchedFilteredCount, actualSearchMatchCount);

        searchedDisplayedCount.Should().Be(actualSearchMatchCount,
            "DisplayedItems.Count should match actual search results");
        searchedFilteredCount.Should().Be(actualSearchMatchCount,
            "FilteredItems.Count should match actual search results");
        searchedDisplayedCount.Should().Be(searchedFilteredCount,
            "DisplayedItems.Count should match FilteredItems.Count with search filter");
    }

    [Fact]
    public async Task CacheHeatSystem_ShouldWorkCorrectly()
    {
        // Arrange
        const string serverAddress = "localhost:50052";
        await _inventoryService.ConnectAsync(serverAddress);

        if (_inventoryService is not CachedInventoryService cachedService)
        {
            throw new InvalidOperationException("Expected CachedInventoryService for this test");
        }

        // Act - Make multiple calls to warm up the cache
        var firstCall = await _inventoryService.ListInventoryItemsAsync();
        await Task.Delay(100);
        var secondCall = await _inventoryService.ListInventoryItemsAsync();

        var stats = cachedService.GetCacheStatistics();

        // Assert - Cache should have entries and some should be warm
        _logger.LogInformation("Cache stats - Total: {Total}, Hot: {Hot}, Warm: {Warm}, Cold: {Cold}, Avg Heat: {AvgHeat:F2}",
            stats.TotalEntries, stats.HotEntries, stats.WarmEntries, stats.ColdEntries, stats.AverageHeat);

        stats.TotalEntries.Should().BeGreaterThan(0, "Cache should have entries");
        firstCall.Items.Count().Should().Be(secondCall.Items.Count(), "Both calls should return same number of items");

        // The second call should likely come from cache, so it should be fast
        // We can't easily measure timing in this test, but we can verify cache behavior
        stats.AverageHeat.Should().BeGreaterThan(0, "Cache should have some heat");
    }

    public void Dispose()
    {
        _grpcChannel?.Dispose();
        _host?.Dispose();
    }
}
