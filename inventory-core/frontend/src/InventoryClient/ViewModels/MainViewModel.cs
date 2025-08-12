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

    // Dialog and chart properties
    [ObservableProperty]
    private ObservableCollection<InventoryItemViewModel> _displayedItems = new();

    [ObservableProperty]
    private InventoryLevelChartViewModel? _selectedItemChart;

    [ObservableProperty]
    private bool _isChartVisible;

    public MainViewModel(InventoryGrpcService inventoryService, ILogger<MainViewModel> logger)
        : base(inventoryService, logger)
    {
        _inventoryService = inventoryService;

        DebugService.LogDebug("MainViewModel constructor called");
        DebugService.LogDebug("Debug log file available at: {0}", DebugService.GetLogFilePath());

        // Subscribe to SelectedItem changes to update prediction status
        PropertyChanged += OnPropertyChanged;
        
        DebugService.LogDebug("MainViewModel initialization completed");
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

    [RelayCommand]
    private async Task AddItem()
    {
        try
        {
            if (!IsConnected)
            {
                SetConnectionError("Not connected to server. Please connect first.");
                return;
            }

            // Create and show the Add Item dialog
            var dialogViewModel = new AddItemDialogViewModel(_inventoryService,
                Logger as ILogger<AddItemDialogViewModel> ??
                Microsoft.Extensions.Logging.Abstractions.NullLogger<AddItemDialogViewModel>.Instance);
            var dialogResult = false;

            dialogViewModel.OnItemAdded += (s, e) =>
            {
                dialogResult = true;
                // Close dialog logic would go here
            };

            // Show dialog in UI - this will be handled by the view
            Logger.LogInformation("Add Item dialog requested");

            // After successful add, refresh the data
            if (dialogResult)
            {
                await RefreshDataAsync();
            }
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Error opening Add Item dialog");
            SetConnectionError($"Failed to open Add Item dialog: {ex.Message}");
        }
    }

    [RelayCommand]
    private async Task ReportInventoryLevel()
    {
        try
        {
            if (!IsConnected)
            {
                SetConnectionError("Not connected to server. Please connect first.");
                return;
            }

            // Create and show the Report Inventory Level dialog
            var dialogViewModel = new ReportInventoryDialogViewModel(_inventoryService,
                Microsoft.Extensions.Logging.Abstractions.NullLogger<ReportInventoryDialogViewModel>.Instance);
            await dialogViewModel.LoadAvailableItemsAsync();

            var dialogResult = false;

            dialogViewModel.OnLevelUpdated += (s, e) =>
            {
                dialogResult = true;
                // Close dialog logic would go here
            };

            // Show dialog in UI - this will be handled by the view
            Logger.LogInformation("Report Inventory Level dialog requested");

            // After successful update, refresh the data
            if (dialogResult)
            {
                await RefreshDataAsync();
            }
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Error opening Report Inventory Level dialog");
            SetConnectionError($"Failed to open Report Inventory Level dialog: {ex.Message}");
        }
    }

    [RelayCommand]
    private async Task UpdateItemLevel(InventoryItemViewModel item)
    {
        if (item == null) return;

        try
        {
            if (!IsConnected)
            {
                SetConnectionError("Not connected to server. Please connect first.");
                return;
            }

            // Create and show the Report Inventory Level dialog with the specific item pre-selected
            var dialogViewModel = new ReportInventoryDialogViewModel(_inventoryService,
                Microsoft.Extensions.Logging.Abstractions.NullLogger<ReportInventoryDialogViewModel>.Instance);
            await dialogViewModel.LoadAvailableItemsAsync();
            dialogViewModel.SelectedItem = item;

            var dialogResult = false;

            dialogViewModel.OnLevelUpdated += (s, e) =>
            {
                dialogResult = true;
                // Close dialog logic would go here
            };

            // Show dialog in UI - this will be handled by the view
            Logger.LogInformation("Update Item Level dialog requested for {ItemName}", item.Name);

            // After successful update, refresh the data
            if (dialogResult)
            {
                await RefreshDataAsync();
            }
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Error opening Update Item Level dialog for {ItemName}", item.Name);
            SetConnectionError($"Failed to open Update Item Level dialog: {ex.Message}");
        }
    }

    [RelayCommand]
    private Task ShowItemChart(InventoryItemViewModel item)
    {
        if (item == null) return Task.CompletedTask;

        try
        {
            // Create chart view model for the selected item
            SelectedItemChart = new InventoryLevelChartViewModel(_inventoryService,
                Microsoft.Extensions.Logging.Abstractions.NullLogger<InventoryLevelChartViewModel>.Instance);
            SelectedItemChart.Item = item;
            IsChartVisible = true;

            Logger.LogInformation("Showing chart for {ItemName}", item.Name);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Error showing chart for {ItemName}", item.Name);
            SetConnectionError($"Failed to show chart: {ex.Message}");
        }

        return Task.CompletedTask;
    }

    [RelayCommand]
    private void CloseChart()
    {
        IsChartVisible = false;
        SelectedItemChart = null;
    }

    private void UpdateDisplayedItems()
    {
        // Take up to 10 items from the filtered list for display in cards
        DisplayedItems.Clear();
        foreach (var item in FilteredItems.Take(10))
        {
            DisplayedItems.Add(item);
        }
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

        // Update displayed items for cards
        UpdateDisplayedItems();

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
    private async Task StartTraining()
    {
        if (SelectedItem == null || SelectedItemPredictionStatus == null) return;

        try
        {
            ClearConnectionError();
            IsLoading = true;

            // Simulate training start
            SelectedItemPredictionStatus.Stage = TrainingStage.Learning;
            SelectedItemPredictionStatus.LastUpdated = DateTime.Now;

            Logger.LogInformation("Started training for {ItemName}", SelectedItem.Name);

            // In a real app, this would call the gRPC service
            await Task.Delay(100);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Error starting training");
            SetConnectionError($"Failed to start training: {ex.Message}");
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private async Task RefreshPredictionStatus()
    {
        if (SelectedItem == null) return;

        try
        {
            ClearConnectionError();
            IsLoading = true;

            // Refresh the prediction status
            SelectedItemPredictionStatus = CreatePredictionStatusForItem(SelectedItem);

            Logger.LogInformation("Refreshed prediction status for {ItemName}", SelectedItem.Name);

            await Task.Delay(100);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Error refreshing prediction status");
            SetConnectionError($"Failed to refresh prediction status: {ex.Message}");
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private void OpenDebugLog()
    {
        try
        {
            DebugService.LogDebug("OpenDebugLog command called");
            var logFilePath = DebugService.GetLogFilePath();
            DebugService.LogDebug("Debug log file path: {0}", logFilePath);
            
            DebugService.OpenLogFile();
            
            Logger.LogInformation("Opened debug log file: {LogFilePath}", logFilePath);
        }
        catch (Exception ex)
        {
            DebugService.LogError("Failed to open debug log file", ex);
            Logger.LogError(ex, "Failed to open debug log file");
            SetConnectionError($"Failed to open debug log: {ex.Message}");
        }
    }
}

