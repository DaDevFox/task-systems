using System.Collections.ObjectModel;
using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using InventoryClient.Models;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using TaskSystems.Shared.ViewModels;
using TaskSystems.Shared.Services;

namespace InventoryClient.ViewModels;

/// <summary>
/// Main view model for the inventory application
/// </summary>
public partial class MainViewModel : ServiceViewModelBase
{
    private readonly IInventoryService _inventoryService;
    private readonly ISettingsService _settingsService;

    // Constants for common error messages and default values
    private const string NotConnectedErrorMessage = "Not connected to server. Please connect first.";
    private const string AllCategoriesFilter = "All Categories";

    [ObservableProperty]
    private ObservableCollection<InventoryItemViewModel> _inventoryItems = new();

    [ObservableProperty]
    private ObservableCollection<InventoryItemViewModel> _filteredItems = new();

    [ObservableProperty]
    private ObservableCollection<InventoryItemViewModel> _lowStockItems = new();

    [ObservableProperty]
    private InventoryItemViewModel? _selectedItem;

    [ObservableProperty]
    [NotifyPropertyChangedFor(nameof(DisplayedItems))]
    private bool _showLowStockOnly;

    [ObservableProperty]
    [NotifyPropertyChangedFor(nameof(DisplayedItems))]
    private string _searchText = string.Empty;

    [ObservableProperty]
    [NotifyPropertyChangedFor(nameof(DisplayedItems))]
    private string _sortOption = "Stock Level (Low to High)";

    [ObservableProperty]
    [NotifyPropertyChangedFor(nameof(DisplayedItems))]
    private string _filterCategory = AllCategoriesFilter;

    [ObservableProperty]
    private ObservableCollection<string> _availableSortOptions = new()
    {
        "Stock Level (Low to High)",
        "Stock Level (High to Low)", 
        "Name (A-Z)",
        "Name (Z-A)",
        "Last Updated (Recent First)",
        "Last Updated (Oldest First)"
    };

    [ObservableProperty]
    private ObservableCollection<string> _availableCategories = new() { AllCategoriesFilter };

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

    // Cache-related properties for debugging
    [ObservableProperty]
    private string _cacheInfo = string.Empty;

    [ObservableProperty]
    private bool _showCacheInfo;

    // Chart-related properties
    [ObservableProperty]
    private bool _isChartVisible;

    [ObservableProperty]
    private InventoryLevelChartViewModel? _selectedItemChart;

    [ObservableProperty]
    private bool _isAddItemDialogVisible;

    [ObservableProperty]
    private AddItemDialogViewModel? _addItemDialog;

    // Property for XAML binding - returns filtered items for display
    public ObservableCollection<InventoryItemViewModel> DisplayedItems => FilteredItems;

    public MainViewModel(IInventoryService inventoryService, IServiceClient serviceClient, ISettingsService settingsService, ILogger<MainViewModel> logger)
        : base(serviceClient, logger)
    {
        _inventoryService = inventoryService;
        _settingsService = settingsService;

        // Subscribe to SelectedItem changes to update prediction status
        PropertyChanged += OnPropertyChanged;

        // Initialize cache settings visibility
        ShowCacheInfo = _settingsService.GetSetting("Debug.ShowCacheInfo", false);

        // Set up auto-refresh timer based on settings
        InitializeAutoRefresh();
    }

    private void OnPropertyChanged(object? sender, System.ComponentModel.PropertyChangedEventArgs e)
    {
        if (e.PropertyName == nameof(SelectedItem))
        {
            OnSelectedItemChanged();
        }
        // Automatically update filtered items when filter/sort properties change
        else if (e.PropertyName == nameof(ShowLowStockOnly) || 
                 e.PropertyName == nameof(SearchText) || 
                 e.PropertyName == nameof(SortOption) || 
                 e.PropertyName == nameof(FilterCategory))
        {
            UpdateFilteredItems();
            Logger.LogDebug("Filter updated due to {PropertyName} change", e.PropertyName);
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

                // Clear the collection before adding items to prevent duplicates
                InventoryItems.Clear();
                foreach (var item in newItems)
                {
                    InventoryItems.Add(item);
                }
            }

            UpdateCounts();
            UpdateFilteredItems(); // Update filtered items after loading
            UpdateCacheInfo(); // Update cache information
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
            UpdateCacheInfo(); // Update cache info even with mock data
        }
    }

    [RelayCommand]
    private void FilterLowStock()
    {
        ShowLowStockOnly = !ShowLowStockOnly;
        Logger.LogDebug("Low stock filter toggled to {FilterState}", ShowLowStockOnly);
        // UpdateFilteredItems() is called automatically by PropertyChanged event
    }

    [RelayCommand]
    private void SearchItems()
    {
        Logger.LogDebug("Searching items with text: {SearchText}", SearchText);
        // UpdateFilteredItems() is called automatically by PropertyChanged event
    }

    [RelayCommand]
    private void ChangeSortOption()
    {
        Logger.LogDebug("Sort option changed to: {SortOption}", SortOption);
        // UpdateFilteredItems() is called automatically by PropertyChanged event
    }

    [RelayCommand]
    private void ChangeFilterCategory()
    {
        Logger.LogDebug("Filter category changed to: {FilterCategory}", FilterCategory);
        // UpdateFilteredItems() is called automatically by PropertyChanged event
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
        // Filter items based on search text, category, and low stock filter
        var filteredItems = InventoryItems.AsEnumerable();

        // Filter by low stock if enabled
        if (ShowLowStockOnly)
        {
            filteredItems = filteredItems.Where(i => i.IsLowStock || i.IsEmpty);
        }

        // Filter by search text
        if (!string.IsNullOrWhiteSpace(SearchText))
        {
            filteredItems = filteredItems.Where(i =>
                i.Name.Contains(SearchText, StringComparison.OrdinalIgnoreCase) ||
                i.Description.Contains(SearchText, StringComparison.OrdinalIgnoreCase));
        }

        // Filter by category
        if (FilterCategory != AllCategoriesFilter && !string.IsNullOrEmpty(FilterCategory))
        {
            filteredItems = filteredItems.Where(i =>
                GetItemCategory(i).Equals(FilterCategory, StringComparison.OrdinalIgnoreCase));
        }

        // Apply sorting
        filteredItems = SortOption switch
        {
            "Stock Level (Low to High)" => filteredItems.OrderBy(i => i.CurrentLevelPercentage),
            "Stock Level (High to Low)" => filteredItems.OrderByDescending(i => i.CurrentLevelPercentage),
            "Name (A-Z)" => filteredItems.OrderBy(i => i.Name),
            "Name (Z-A)" => filteredItems.OrderByDescending(i => i.Name),
            "Last Updated (Recent First)" => filteredItems.OrderByDescending(i => i.LastUpdated),
            "Last Updated (Oldest First)" => filteredItems.OrderBy(i => i.LastUpdated),
            _ => filteredItems.OrderBy(i => i.CurrentLevelPercentage) // Default to stock level ascending
        };

        // Update the filtered collection for UI binding
        FilteredItems.Clear();
        foreach (var item in filteredItems)
        {
            FilteredItems.Add(item);
        }

        // Notify that DisplayedItems has changed (since it returns FilteredItems)
        OnPropertyChanged(nameof(DisplayedItems));

        Logger.LogDebug("Filtered and sorted items: {Count} (Sort: {Sort}, Category: {Category})", 
            FilteredItems.Count, SortOption, FilterCategory);
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

        // Update available categories
        UpdateAvailableCategories();

        Logger.LogDebug("Updated counts - Total: {Total}, Low Stock: {LowStock}, Empty: {Empty}",
            TotalItems, LowStockCount, EmptyItemsCount);
    }

    private void UpdateAvailableCategories()
    {
        var categories = new HashSet<string> { AllCategoriesFilter };
        
        foreach (var item in InventoryItems)
        {
            var category = GetItemCategory(item);
            if (!string.IsNullOrEmpty(category))
            {
                categories.Add(category);
            }
        }

        AvailableCategories.Clear();
        foreach (var category in categories.OrderBy(c => c == AllCategoriesFilter ? "" : c))
        {
            AvailableCategories.Add(category);
        }

        // Reset filter if current category no longer exists
        if (!AvailableCategories.Contains(FilterCategory))
        {
            FilterCategory = AllCategoriesFilter;
        }
    }

    private static string GetItemCategory(InventoryItemViewModel item)
    {
        // Try to get category from metadata first, then fall back to simple categorization
        if (item.Metadata?.ContainsKey("category") == true)
        {
            var category = item.Metadata["category"];
            if (!string.IsNullOrWhiteSpace(category))
            {
                return category;
            }
        }

        // Simple categorization based on unit type as fallback
        return item.UnitId.ToLowerInvariant() switch
        {
            "kg" or "lbs" or "g" => "Food & Ingredients",
            "liters" or "l" or "gallons" or "ml" => "Liquids",
            "pieces" or "pcs" or "units" => "Items & Parts",
            "meters" or "m" or "feet" or "ft" => "Materials",
            "boxes" or "packs" => "Packaging",
            _ => "Other"
        };
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
    private Task AddItem()
    {
        try
        {
            ClearConnectionError();

            if (!IsConnected)
            {
                SetConnectionError(NotConnectedErrorMessage);
                return Task.CompletedTask;
            }

            // Show the Add Item dialog
            return ShowAddItemDialog();
        }
        catch (Exception ex)
        {
            var errorMessage = $"Failed to show add item dialog: {ex.Message}";
            SetConnectionError(errorMessage);
            Logger.LogError(ex, "Failed to show add item dialog");
            return Task.CompletedTask;
        }
    }

    private Task ShowAddItemDialog()
    {
        try
        {
            // Create and configure the dialog ViewModel
            var dialogLogger = Microsoft.Extensions.Logging.Abstractions.NullLogger<AddItemDialogViewModel>.Instance;
            var dialogViewModel = new AddItemDialogViewModel(_inventoryService, dialogLogger);

            // Subscribe to dialog events
            dialogViewModel.OnItemAdded += async (sender, args) =>
            {
                // Close dialog and refresh data
                IsAddItemDialogVisible = false;
                AddItemDialog = null;

                // Refresh the inventory list
                await RefreshDataAsync();
                Logger.LogInformation("Add item dialog closed - item added successfully");
            };

            dialogViewModel.OnCanceled += (sender, args) =>
            {
                // Close dialog without refreshing
                IsAddItemDialogVisible = false;
                AddItemDialog = null;
                Logger.LogInformation("Add item dialog closed - cancelled");
            };

            // Show the dialog
            AddItemDialog = dialogViewModel;
            IsAddItemDialogVisible = true;

            Logger.LogInformation("Add item dialog should now be visible with ViewModel: {Type}", dialogViewModel.GetType().Name);

            // The focus should be handled by the AddItemDialog.axaml.cs code-behind
            return Task.CompletedTask;
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to create add item dialog");
            SetConnectionError($"Failed to open add item dialog: {ex.Message}");
            return Task.CompletedTask;
        }
    }

    [RelayCommand]
    private void CloseAddItemDialog()
    {
        IsAddItemDialogVisible = false;
        AddItemDialog = null;
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
                SetConnectionError(NotConnectedErrorMessage);
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

            // Note: Real gRPC service call would be implemented here
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
                SetConnectionError(NotConnectedErrorMessage);
                return;
            }

            IsLoading = true;

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
                SetConnectionError(NotConnectedErrorMessage);
                return;
            }

            IsLoading = true;

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

    private void InitializeAutoRefresh()
    {
        var autoRefreshEnabled = _settingsService.GetSetting("AutoRefresh.Enabled", true);
        var autoRefreshInterval = _settingsService.GetSetting("AutoRefresh.IntervalSeconds", 30);

        if (autoRefreshEnabled)
        {
            // Set up auto-refresh timer
            var timer = new System.Timers.Timer(autoRefreshInterval * 1000);
            timer.Elapsed += async (sender, e) =>
            {
                if (IsConnected && !IsLoading)
                {
                    try
                    {
                        await RefreshDataAsync();
                        UpdateCacheInfo();
                    }
                    catch (Exception ex)
                    {
                        Logger.LogWarning(ex, "Auto-refresh failed");
                    }
                }
            };
            timer.Start();

            Logger.LogInformation("Auto-refresh enabled with {Interval}s interval", autoRefreshInterval);
        }
    }

    private void UpdateCacheInfo()
    {
        if (_inventoryService is CachedInventoryService cachedService)
        {
            var stats = cachedService.GetCacheStatistics();
            CacheInfo = $"Cache: {stats.TotalEntries} entries " +
                       $"(🔥{stats.HotEntries} ⚡{stats.WarmEntries} ❄️{stats.ColdEntries}) " +
                       $"Avg Heat: {stats.AverageHeat:F2} Threshold: {stats.HeatThreshold:F2}";
        }
        else
        {
            CacheInfo = "Cache: Not using cached service";
        }
    }

    [RelayCommand]
    private void ToggleCacheInfo()
    {
        ShowCacheInfo = !ShowCacheInfo;
        _settingsService.SetSetting("Debug.ShowCacheInfo", ShowCacheInfo);
        Logger.LogDebug("Cache info display toggled to: {Show}", ShowCacheInfo);
    }

    [RelayCommand]
    private void ClearCache()
    {
        if (_inventoryService is CachedInventoryService cachedService)
        {
            cachedService.ClearCache();
            UpdateCacheInfo();
            Logger.LogInformation("Cache cleared manually");
        }
    }

    [RelayCommand]
    private async Task ConfigureSettings()
    {
        // For now, just toggle some common settings
        var currentHeatThreshold = _settingsService.GetSetting("Cache.HeatThreshold", 0.3);
        var newThreshold = Math.Abs(currentHeatThreshold - 0.3) < 0.01 ? 0.1 : 0.3;
        _settingsService.SetSetting("Cache.HeatThreshold", newThreshold);

        var autoRefreshEnabled = _settingsService.GetSetting("AutoRefresh.Enabled", true);
        _settingsService.SetSetting("AutoRefresh.Enabled", !autoRefreshEnabled);

        await _settingsService.SaveAsync();

        Logger.LogInformation("Settings updated - Heat threshold: {Threshold}, Auto-refresh: {Enabled}",
            newThreshold, !autoRefreshEnabled);
    }

    [RelayCommand]
    private async Task GetPredictionForSelectedItem()
    {
        if (SelectedItem == null) return;

        try
        {
            ClearConnectionError();
            IsLoading = true;

            var prediction = await _inventoryService.PredictConsumptionAsync(SelectedItem.Id, 30, false);
            if (prediction != null)
            {
                SelectedItem.PredictedDaysRemaining = prediction.PredictedDaysRemaining;
                SelectedItem.ConfidenceScore = prediction.ConfidenceScore;
                Logger.LogInformation("Updated prediction for {ItemName}: {Days} days remaining (confidence: {Confidence:P})",
                    SelectedItem.Name, prediction.PredictedDaysRemaining, prediction.ConfidenceScore);
            }
        }
        catch (Exception ex)
        {
            SetConnectionError($"Failed to get prediction: {ex.Message}");
            Logger.LogError(ex, "Failed to get prediction for item {ItemId}", SelectedItem.Id);
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private async Task UpdateItemLevel(InventoryItemViewModel item)
    {
        if (item == null) return;

        try
        {
            ClearConnectionError();

            if (!IsConnected)
            {
                SetConnectionError(NotConnectedErrorMessage);
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
            UpdateCacheInfo();
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
    private async Task ShowItemChart(InventoryItemViewModel item)
    {
        if (item == null) return;

        try
        {
            // For now, just select the item and get its prediction
            SelectedItem = item;
            await GetPredictionForSelectedItem();

            Logger.LogInformation("Displaying chart for item: {ItemName}", item.Name);
        }
        catch (Exception ex)
        {
            SetConnectionError($"Failed to show item chart: {ex.Message}");
            Logger.LogError(ex, "Failed to show chart for item {ItemId}", item.Id);
        }
    }

    [RelayCommand]
    private void OpenDebugLog()
    {
        try
        {
            DebugService.OpenLogFile();
            Logger.LogInformation("Opened debug log file");
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to open debug log");
        }
    }

    [RelayCommand]
    private Task ReportInventoryLevel()
    {
        // For now, just refresh the data
        return RefreshDataAsync();
    }

    [RelayCommand]
    private void CloseChart()
    {
        IsChartVisible = false;
        SelectedItemChart = null;
        Logger.LogDebug("Chart closed");
    }

    [RelayCommand]
    private async Task UpdateInventoryLevel(InventoryItemViewModel item)
    {
        // This command is called from the DataGrid action buttons
        await UpdateItemLevel(item);
    }
}
