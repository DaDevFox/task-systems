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
    private ObservableCollection<InventoryItemViewModel> _filteredItems = new();

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

    [ObservableProperty]
    private string _connectionError = string.Empty;

    [ObservableProperty]
    private bool _hasConnectionError;

    // Prediction model management properties
    [ObservableProperty]
    private PredictionTrainingStatusViewModel? _selectedItemPredictionStatus;

    [ObservableProperty]
    private bool _isPredictionModelSelected;

    public MainViewModel(InventoryGrpcService inventoryService, ILogger<MainViewModel> logger) 
        : base(inventoryService, logger)
    {
        _inventoryService = inventoryService;
        
        // Subscribe to SelectedItem changes to update prediction status
        PropertyChanged += OnPropertyChanged;
    }

    private void OnPropertyChanged(object? sender, System.ComponentModel.PropertyChangedEventArgs e)
    {
        if (e.PropertyName == nameof(SelectedItem))
        {
            OnSelectedItemChanged();
        }
    }

    private void OnSelectedItemChanged()
    {
        if (SelectedItem != null)
        {
            SelectedItemPredictionStatus = CreatePredictionStatusForItem(SelectedItem);
            IsPredictionModelSelected = true;
        }
        else
        {
            SelectedItemPredictionStatus = null;
            IsPredictionModelSelected = false;
        }
    }

    private PredictionTrainingStatusViewModel CreatePredictionStatusForItem(InventoryItemViewModel item)
    {
        // Create mock prediction status - in a real app this would come from the server
        var status = new PredictionTrainingStatusViewModel
        {
            ItemId = item.Id,
            Stage = TrainingStage.Trained,
            ActiveModel = PredictionModel.Bayesian,
            AvailableModels = Enum.GetValues<PredictionModel>().Where(m => m != PredictionModel.Unspecified).ToList(),
            TrainingSamples = 150,
            MinSamplesRequired = 100,
            TrainingAccuracy = 0.87,
            TrainingStarted = DateTime.Now.AddDays(-2),
            LastUpdated = DateTime.Now.AddHours(-1),
            ModelParameters = new Dictionary<string, double>
            {
                { "Confidence Threshold", 0.85 },
                { "Window Size", 7.0 },
                { "Learning Rate", 0.1 },
                { "Regularization", 0.01 }
            }
        };

        return status;
    }

    protected override async Task RefreshDataAsync()
    {
        try
        {
            ClearConnectionError();
            
            // Check if service is actually connected and responsive
            if (!await _inventoryService.PingAsync())
            {
                SetConnectionError("Lost connection to server. Please reconnect.");
                return;
            }

            // Get real data from the backend
            var (items, _) = await _inventoryService.ListInventoryItemsAsync(
                lowStockOnly: false, 
                unitTypeFilter: null, 
                limit: 1000, 
                offset: 0);

            InventoryItems.Clear();
            foreach (var item in items)
            {
                InventoryItems.Add(item);
            }

            // If no items from backend, add some sample data for demonstration
            if (InventoryItems.Count == 0)
            {
                Logger.LogInformation("No items found in backend, adding sample data for demonstration");
                await AddSampleDataToBackend();
                
                // Refresh again to get the newly added items
                var (newItems, _) = await _inventoryService.ListInventoryItemsAsync(
                    lowStockOnly: false, 
                    unitTypeFilter: null, 
                    limit: 1000, 
                    offset: 0);

                foreach (var item in newItems)
                {
                    InventoryItems.Add(item);
                }
            }

            UpdateCounts();
            UpdateFilteredItems(); // Update filtered items after loading
            Logger.LogInformation("Successfully refreshed inventory data with {Count} items", InventoryItems.Count);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to refresh inventory data");
            SetConnectionError($"Failed to refresh data: {ex.Message}");
            
            // Fall back to mock data if gRPC fails
            Logger.LogInformation("Falling back to mock data due to backend error");
            var mockItems = CreateMockData();
            InventoryItems.Clear();
            foreach (var item in mockItems)
            {
                InventoryItems.Add(item);
            }
            UpdateCounts();
            UpdateFilteredItems(); // Update filtered items for mock data too
        }
    }

    [RelayCommand]
    private async Task UpdateInventoryLevel(InventoryItemViewModel item)
    {
        if (item == null) return;

        try
        {
            ClearConnectionError();
            
            if (!IsConnected)
            {
                SetConnectionError("Not connected to server. Please connect first.");
                return;
            }

            IsLoading = true;
            var success = await _inventoryService.UpdateInventoryLevelAsync(
                item.Id, 
                item.CurrentLevel, 
                "Manual update from UI", 
                true);
            
            if (!success)
            {
                SetConnectionError("Failed to update inventory level. Check server connection.");
                return;
            }

            Logger.LogInformation("Successfully updated inventory level for {ItemName}", item.Name);
            UpdateCounts();
        }
        catch (Exception ex)
        {
            var errorMessage = $"Failed to update inventory level: {ex.Message}";
            SetConnectionError(errorMessage);
            Logger.LogError(ex, "Failed to update inventory level for item {ItemId}", item.Id);
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private void FilterLowStock()
    {
        ShowLowStockOnly = !ShowLowStockOnly;
        Logger.LogDebug("Low stock filter toggled to {FilterState}", ShowLowStockOnly);
        UpdateFilteredItems();
    }

    [RelayCommand]
    private void SearchItems()
    {
        Logger.LogDebug("Searching items with text: {SearchText}", SearchText);
        UpdateFilteredItems();
    }

    [RelayCommand]
    private void ClearConnectionError()
    {
        ConnectionError = string.Empty;
        HasConnectionError = false;
    }

    private void SetConnectionError(string error)
    {
        ConnectionError = error;
        HasConnectionError = true;
        Logger.LogWarning("Connection error: {Error}", error);
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

        // Update the filtered collection for UI binding
        FilteredItems.Clear();
        foreach (var item in filteredItems)
        {
            FilteredItems.Add(item);
        }

        Logger.LogDebug("Filtered items: {Count}", FilteredItems.Count);
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

        Logger.LogDebug("Updated counts - Total: {Total}, Low Stock: {LowStock}, Empty: {Empty}", 
            TotalItems, LowStockCount, EmptyItemsCount);
    }

    private static List<InventoryItemViewModel> CreateMockData()
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

    private async Task AddSampleDataToBackend()
    {
        try
        {
            Logger.LogInformation("Adding sample data to backend for demonstration");
            
            var sampleItems = new[]
            {
                new { Name = "Flour", Description = "All-purpose flour for baking", InitialLevel = 2.5, MaxCapacity = 10.0, LowStockThreshold = 2.0, UnitId = "kg" },
                new { Name = "Sugar", Description = "White granulated sugar", InitialLevel = 0.8, MaxCapacity = 5.0, LowStockThreshold = 1.0, UnitId = "kg" },
                new { Name = "Salt", Description = "Table salt for cooking", InitialLevel = 0.0, MaxCapacity = 2.0, LowStockThreshold = 0.5, UnitId = "kg" },
                new { Name = "Olive Oil", Description = "Extra virgin olive oil", InitialLevel = 4.2, MaxCapacity = 5.0, LowStockThreshold = 1.0, UnitId = "L" }
            };

            foreach (var sample in sampleItems)
            {
                try
                {
                    await _inventoryService.AddInventoryItemAsync(
                        sample.Name,
                        sample.Description,
                        sample.InitialLevel,
                        sample.MaxCapacity,
                        sample.LowStockThreshold,
                        sample.UnitId);
                    
                    Logger.LogDebug("Added sample item: {ItemName}", sample.Name);
                }
                catch (Exception ex)
                {
                    Logger.LogWarning(ex, "Failed to add sample item {ItemName}, it may already exist", sample.Name);
                }
            }
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to add sample data to backend");
        }
    }

    [RelayCommand]
    private async Task AddItem()
    {
        try
        {
            ClearConnectionError();
            
            if (!IsConnected)
            {
                SetConnectionError("Not connected to server. Please connect first.");
                return;
            }

            // For now, add a sample item - in a real app this would open a dialog
            IsLoading = true;
            var newItem = await _inventoryService.AddInventoryItemAsync(
                "New Item",
                "Sample item description",
                5.0,  // initial level
                10.0, // max capacity
                2.0,  // low stock threshold
                "kg"); // unit ID - use a simple unit that should exist

            if (newItem != null)
            {
                InventoryItems.Add(newItem);
                UpdateCounts();
                UpdateFilteredItems(); // Update filtered items when adding new item
                Logger.LogInformation("Successfully added new item: {ItemName}", newItem.Name);
            }
            else
            {
                SetConnectionError("Failed to add new item. Check server connection.");
            }
        }
        catch (Exception ex)
        {
            var errorMessage = $"Failed to add item: {ex.Message}";
            SetConnectionError(errorMessage);
            Logger.LogError(ex, "Failed to add new item");
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private async Task StartTraining()
    {
        if (SelectedItemPredictionStatus == null || SelectedItem == null) return;

        try
        {
            ClearConnectionError();
            
            if (!IsConnected)
            {
                SetConnectionError("Not connected to server. Please connect first.");
                return;
            }

            IsLoading = true;
            
            // For now, simulate starting training
            SelectedItemPredictionStatus.Stage = TrainingStage.Learning;
            SelectedItemPredictionStatus.TrainingStarted = DateTime.Now;
            SelectedItemPredictionStatus.LastUpdated = DateTime.Now;
            
            // Simulate async operation
            await Task.Delay(100);
            
            Logger.LogInformation("Started training for {ItemName} using {Model}", 
                SelectedItem.Name, SelectedItemPredictionStatus.ActiveModel);
                
            // TODO: Call actual gRPC service to start training
            // await _inventoryService.StartTrainingAsync(SelectedItem.Id, SelectedItemPredictionStatus.ActiveModel);
        }
        catch (Exception ex)
        {
            var errorMessage = $"Failed to start training: {ex.Message}";
            SetConnectionError(errorMessage);
            Logger.LogError(ex, "Failed to start training for item {ItemId}", SelectedItem?.Id);
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private async Task RefreshPredictionStatus()
    {
        if (SelectedItemPredictionStatus == null || SelectedItem == null) return;

        try
        {
            ClearConnectionError();
            
            if (!IsConnected)
            {
                SetConnectionError("Not connected to server. Please connect first.");
                return;
            }

            IsLoading = true;
            
            // TODO: Call actual gRPC service to get prediction status
            // var status = await _inventoryService.GetPredictionStatusAsync(SelectedItem.Id);
            
            // For now, simulate refreshing status
            await Task.Delay(100);
            SelectedItemPredictionStatus.LastUpdated = DateTime.Now;
            
            Logger.LogInformation("Refreshed prediction status for {ItemName}", SelectedItem.Name);
        }
        catch (Exception ex)
        {
            var errorMessage = $"Failed to refresh prediction status: {ex.Message}";
            SetConnectionError(errorMessage);
            Logger.LogError(ex, "Failed to refresh prediction status for item {ItemId}", SelectedItem?.Id);
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private async Task ApplyModelConfiguration()
    {
        if (SelectedItemPredictionStatus == null || SelectedItem == null) return;

        try
        {
            ClearConnectionError();
            
            if (!IsConnected)
            {
                SetConnectionError("Not connected to server. Please connect first.");
                return;
            }

            IsLoading = true;
            
            // TODO: Call actual gRPC service to apply configuration
            // await _inventoryService.ApplyModelConfigurationAsync(SelectedItem.Id, SelectedItemPredictionStatus);
            
            // Simulate async operation
            await Task.Delay(100);
            SelectedItemPredictionStatus.LastUpdated = DateTime.Now;
            
            Logger.LogInformation("Applied model configuration for {ItemName}", SelectedItem.Name);
        }
        catch (Exception ex)
        {
            var errorMessage = $"Failed to apply model configuration: {ex.Message}";
            SetConnectionError(errorMessage);
            Logger.LogError(ex, "Failed to apply model configuration for item {ItemId}", SelectedItem?.Id);
        }
        finally
        {
            IsLoading = false;
        }
    }
}
