using Xunit;
using FluentAssertions;
using Microsoft.Extensions.Logging;
using InventoryClient.Services;
using InventoryClient.ViewModels;
using Microsoft.Extensions.DependencyInjection;
using TaskSystems.Shared.Services;
using System;
using System.Linq;
using System.Threading.Tasks;

namespace InventoryClient.IntegrationTests;

/// <summary>
/// Simple debug test to understand the filtering issues
/// </summary>
public class FilteringDebugTests : IDisposable
{
    private readonly IServiceProvider _serviceProvider;
    private readonly ILogger<FilteringDebugTests> _logger;

    public FilteringDebugTests()
    {
        var services = new ServiceCollection();

        // Add logging
        services.AddLogging(builder =>
        {
            builder.AddConsole();
            builder.SetMinimumLevel(LogLevel.Information);
        });

        // Add settings service
        services.AddSingleton<ISettingsService>(provider =>
        {
            var settingsService = new JsonSettingsService();
            settingsService.LoadAsync().Wait();
            return settingsService;
        });

        // Add inventory services
        services.AddSingleton<InventoryGrpcService>();
        services.AddSingleton<IServiceClient>(provider => provider.GetRequiredService<InventoryGrpcService>());
        services.AddSingleton<IInventoryService>(provider =>
        {
            var grpcService = provider.GetRequiredService<InventoryGrpcService>();
            var settingsService = provider.GetRequiredService<ISettingsService>();
            var logger = provider.GetRequiredService<ILogger<CachedInventoryService>>();
            return new CachedInventoryService(grpcService, settingsService, logger);
        });

        // Add view models
        services.AddTransient<MainViewModel>(provider =>
        {
            var inventoryService = provider.GetRequiredService<IInventoryService>();
            var serviceClient = provider.GetRequiredService<IServiceClient>();
            var settingsService = provider.GetRequiredService<ISettingsService>();
            var logger = provider.GetRequiredService<ILogger<MainViewModel>>();
            return new MainViewModel(inventoryService, serviceClient, settingsService, logger);
        });

        _serviceProvider = services.BuildServiceProvider();
        _logger = _serviceProvider.GetRequiredService<ILogger<FilteringDebugTests>>();
    }

    [Fact]
    public async Task DebugItemProperties_ShouldShowCurrentItemStates()
    {
        // Arrange
        var mainViewModel = _serviceProvider.GetRequiredService<MainViewModel>();
        var serviceClient = _serviceProvider.GetRequiredService<IServiceClient>();

        // Connect to the backend
        var connected = await serviceClient.ConnectAsync("localhost:50052");
        connected.Should().BeTrue("we should be able to connect to the backend");

        // Act - Wait for refresh to complete
        await Task.Delay(2000); // Give it time to load data

        // Debug output
        _logger.LogInformation("=== INVENTORY ITEMS DEBUG ===");
        _logger.LogInformation("Total items loaded: {Count}", mainViewModel.InventoryItems.Count);

        foreach (var item in mainViewModel.InventoryItems.Take(5)) // Just show first 5
        {
            _logger.LogInformation(
                "Item: '{Name}' - Current: {Current}, Threshold: {Threshold}, IsLowStock: {IsLowStock}, IsEmpty: {IsEmpty}",
                item.Name, item.CurrentLevel, item.LowStockThreshold, item.IsLowStock, item.IsEmpty);
        }

        // Test filtering logic manually
        mainViewModel.ShowLowStockOnly = true;
        mainViewModel.SearchText = "";

        // Force update filtered items (simulate what the UI would do)
        var filterLowStockCommand = typeof(MainViewModel)
            .GetMethod("FilterLowStock", System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);
        filterLowStockCommand?.Invoke(mainViewModel, null);

        await Task.Delay(500); // Give time for filtering

        _logger.LogInformation("=== AFTER LOW STOCK FILTER ===");
        _logger.LogInformation("ShowLowStockOnly: {ShowLowStockOnly}", mainViewModel.ShowLowStockOnly);
        _logger.LogInformation("FilteredItems count: {Count}", mainViewModel.FilteredItems.Count);
        _logger.LogInformation("DisplayedItems count: {Count}", mainViewModel.DisplayedItems.Count);
        _logger.LogInformation("Actual low stock items: {Count}", mainViewModel.InventoryItems.Count(i => i.IsLowStock));

        // Test search filter
        mainViewModel.ShowLowStockOnly = false;
        mainViewModel.SearchText = "test";

        var searchCommand = typeof(MainViewModel)
            .GetMethod("SearchItems", System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);
        searchCommand?.Invoke(mainViewModel, null);

        await Task.Delay(500);

        _logger.LogInformation("=== AFTER SEARCH FILTER ===");
        _logger.LogInformation("SearchText: '{SearchText}'", mainViewModel.SearchText);
        _logger.LogInformation("FilteredItems count: {Count}", mainViewModel.FilteredItems.Count);
        _logger.LogInformation("DisplayedItems count: {Count}", mainViewModel.DisplayedItems.Count);
        _logger.LogInformation("Items matching 'test': {Count}",
            mainViewModel.InventoryItems.Count(i =>
                i.Name.Contains("test", StringComparison.OrdinalIgnoreCase) ||
                i.Description.Contains("test", StringComparison.OrdinalIgnoreCase)));

        // Cleanup
        await serviceClient.DisconnectAsync();
    }

    public void Dispose()
    {
        _serviceProvider?.GetService<IServiceClient>()?.DisconnectAsync().Wait();
        if (_serviceProvider is IDisposable disposable)
            disposable.Dispose();
    }
}
