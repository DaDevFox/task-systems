using System.Collections.ObjectModel;
using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using InventoryClient.Models;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using TaskSystems.Shared.ViewModels;

namespace InventoryClient.ViewModels;

/// <summary>
/// Main view model for the inventory application
/// </summary>
public partial class MainViewModel : ServiceViewModelBase
{
    private readonly InventoryGrpcService _inventoryService;

    [ObservableProperty]
    private ObservableCollection<InventoryItemViewModel> _inventoryItems = new();

    [ObservableProperty]
    private ObservableCollection<InventoryItemViewModel> _lowStockItems = new();

    [ObservableProperty]
    private InventoryItemViewModel? _selectedItem;

    [ObservableProperty]
    private bool _showLowStockOnly;

    [ObservableProperty]
    private string _searchText = string.Empty;

    [ObservableProperty]
    private int _totalItems;

    [ObservableProperty]
    private int _lowStockCount;

    [ObservableProperty]
    private int _emptyItemsCount;

    public MainViewModel(InventoryGrpcService inventoryService, ILogger<MainViewModel> logger) 
        : base(inventoryService, logger)
    {
        _inventoryService = inventoryService;
    }

    protected override async Task RefreshDataAsync()
    {
        try
        {
            // For now, create some mock data until gRPC is working
            var mockItems = CreateMockData();
            InventoryItems.Clear();
            foreach (var item in mockItems)
            {
                InventoryItems.Add(item);
            }

            UpdateCounts();
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to refresh inventory data");
            throw;
        }
    }

    [RelayCommand]
    private async Task UpdateInventoryLevel(InventoryItemViewModel item)
    {
        if (item == null) return;

        try
        {
            // This would normally call the gRPC service
            // For now, just update the UI
            Logger.LogInformation("Updating inventory level for {ItemName}", item.Name);
            UpdateCounts();
        }
        catch (Exception ex)
        {
            LastError = $"Failed to update inventory level: {ex.Message}";
            Logger.LogError(ex, "Failed to update inventory level for item {ItemId}", item.Id);
        }
    }

    [RelayCommand]
    private void FilterLowStock()
    {
        ShowLowStockOnly = !ShowLowStockOnly;
        UpdateFilteredItems();
    }

    [RelayCommand]
    private void SearchItems()
    {
        UpdateFilteredItems();
    }

    private void UpdateFilteredItems()
    {
        // Filter items based on search text and low stock filter
        var filteredItems = InventoryItems.AsEnumerable();

        if (ShowLowStockOnly)
        {
            filteredItems = filteredItems.Where(i => i.IsLowStock || i.IsEmpty);
        }

        if (!string.IsNullOrWhiteSpace(SearchText))
        {
            filteredItems = filteredItems.Where(i => 
                i.Name.Contains(SearchText, StringComparison.OrdinalIgnoreCase) ||
                i.Description.Contains(SearchText, StringComparison.OrdinalIgnoreCase));
        }

        // In a real app, you would update a filtered collection view here
        Logger.LogDebug("Filtered items: {Count}", filteredItems.Count());
    }

    private void UpdateCounts()
    {
        TotalItems = InventoryItems.Count;
        LowStockCount = InventoryItems.Count(i => i.IsLowStock);
        EmptyItemsCount = InventoryItems.Count(i => i.IsEmpty);

        LowStockItems.Clear();
        foreach (var item in InventoryItems.Where(i => i.IsLowStock || i.IsEmpty))
        {
            LowStockItems.Add(item);
        }
    }

    private List<InventoryItemViewModel> CreateMockData()
    {
        return new List<InventoryItemViewModel>
        {
            new()
            {
                Id = "1",
                Name = "Flour",
                Description = "All-purpose flour for baking",
                CurrentLevel = 2.5,
                MaxCapacity = 10.0,
                LowStockThreshold = 2.0,
                UnitId = "kg",
                LastUpdated = DateTime.Now.AddHours(-2),
                PredictedDaysRemaining = 5.2,
                ConfidenceScore = 0.85
            },
            new()
            {
                Id = "2", 
                Name = "Sugar",
                Description = "White granulated sugar",
                CurrentLevel = 0.8,
                MaxCapacity = 5.0,
                LowStockThreshold = 1.0,
                UnitId = "kg",
                LastUpdated = DateTime.Now.AddHours(-1),
                PredictedDaysRemaining = 2.1,
                ConfidenceScore = 0.92
            },
            new()
            {
                Id = "3",
                Name = "Milk",
                Description = "Fresh whole milk",
                CurrentLevel = 0.0,
                MaxCapacity = 4.0,
                LowStockThreshold = 0.5,
                UnitId = "liters",
                LastUpdated = DateTime.Now.AddMinutes(-30),
                PredictedDaysRemaining = 0.0,
                ConfidenceScore = 1.0
            },
            new()
            {
                Id = "4",
                Name = "Eggs",
                Description = "Large eggs, grade A",
                CurrentLevel = 18,
                MaxCapacity = 24,
                LowStockThreshold = 6,
                UnitId = "pieces",
                LastUpdated = DateTime.Now.AddHours(-3),
                PredictedDaysRemaining = 8.5,
                ConfidenceScore = 0.78
            }
        };
    }
}
