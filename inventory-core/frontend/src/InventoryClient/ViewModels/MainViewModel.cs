using System.Collections.ObjectModel;
using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using InventoryClient.Models;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using TaskSystems.Shared.ViewModels;
using TaskSystems.Shared.Services;
using Inventory.V1;
using Avalonia.Threading;

namespace InventoryClient.ViewModels;

/// <summary>
/// Main view model for the inventory application
/// </summary>
public partial class MainViewModel : ServiceViewModelBase
{
    private readonly IInventoryService _inventoryService;
    private readonly ISettingsService _settingsService;
    private readonly SemaphoreSlim _refreshSemaphore = new(1, 1);

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

    partial void OnSelectedItemChanged(InventoryItemViewModel? value)
    {
        DebugService.LogDebug($"🔄 MAIN: OnSelectedItemChanged called - SelectedItem: {value?.Name ?? "null"}");

        // Safely dispose/clear the previous chart first
        try
        {
            if (SelectedItemChart != null)
            {
                DebugService.LogDebug("🧹 MAIN: Clearing previous chart");
                SelectedItemChart = null;
            }
        }
        catch (Exception ex)
        {
            DebugService.LogDebug($"⚠️ MAIN: Error clearing previous chart: {ex.Message}");
        }

        // Create chart for prediction tab when item is selected
        if (value != null)
        {
            try
            {
                DebugService.LogDebug($"📊 MAIN: Creating chart for selected item: {value.Name} (ID: {value.Id})");

                // Validate required services before creating chart
                if (_inventoryService == null)
                {
                    DebugService.LogDebug("❌ MAIN: _inventoryService is null, cannot create chart");
                    SelectedItemChart = null;
                    return;
                }

                if (_settingsService == null)
                {
                    DebugService.LogDebug("❌ MAIN: _settingsService is null, cannot create chart");
                    SelectedItemChart = null;
                    return;
                }

                var loggerFactory = Microsoft.Extensions.Logging.Abstractions.NullLoggerFactory.Instance;
                var chartLogger = loggerFactory.CreateLogger<InventoryLevelChartViewModel>();
                var chartViewModel = new InventoryLevelChartViewModel(_inventoryService, _settingsService, chartLogger)
                {
                    Item = value
                };
                SelectedItemChart = chartViewModel;

                DebugService.LogDebug("✅ MAIN: Chart ViewModel created and assigned to SelectedItemChart");

                // Trigger chart data refresh with proper UI thread marshalling
                _ = Task.Run(async () =>
                {
                    try
                    {
                        // Add a longer delay and recheck if the chart is still valid
                        await Task.Delay(200);

                        // Verify the chart is still the current one before refreshing
                        if (SelectedItemChart != chartViewModel)
                        {
                            DebugService.LogDebug("⚠️ MAIN: Chart was replaced during async operation, skipping refresh");
                            return;
                        }

                        if (chartViewModel.Item?.Id != value.Id)
                        {
                            DebugService.LogDebug("⚠️ MAIN: Item changed during async operation, skipping refresh");
                            return;
                        }

                        DebugService.LogDebug($"🔄 MAIN: Starting chart data refresh for item: {value.Name}");

                        // Marshall back to UI thread for command execution
                        await Dispatcher.UIThread.InvokeAsync(async () =>
                        {
                            // Check if RefreshDataCommand is available (on UI thread)
                            if (chartViewModel.RefreshDataCommand == null)
                            {
                                DebugService.LogDebug("❌ MAIN: RefreshDataCommand is null");
                                return;
                            }

                            await chartViewModel.RefreshDataCommand.ExecuteAsync(null);
                            DebugService.LogDebug($"✅ MAIN: Chart data refresh completed for item: {value.Name}");
                        });
                    }
                    catch (Exception ex)
                    {
                        DebugService.LogDebug($"❌ MAIN: Failed to refresh chart data for selected item {value.Id}: {ex.Message}\n{ex.StackTrace}");
                    }
                });

                DebugService.LogDebug($"✅ MAIN: Updated chart for selected item: {value.Name}");
            }
            catch (Exception ex)
            {
                DebugService.LogDebug($"❌ MAIN: Failed to create chart for selected item {value.Id}: {ex.Message}\n{ex.StackTrace}");
                SelectedItemChart = null;
            }
        }
        else
        {
            DebugService.LogDebug("ℹ️ MAIN: No item selected, chart cleared");
            SelectedItemChart = null;
        }
    }

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

    // // Prediction model management properties
    // [ObservableProperty]
    // private PredictionTrainingStatusViewModel? _selectedItemPredictionStatus;

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
    private InventoryLevelChartViewModel? _staticPredictionChart;

    [ObservableProperty]
    private InventoryItemViewModel? _selectedPredictionItem;

    // Chart Settings Properties
    public ChartDataMode ChartDataMode
    {
        get => _settingsService.GetSetting(ChartSettings.ModeKey, ChartDataMode.Granularity);
        set
        {
            _settingsService.SetSetting(ChartSettings.ModeKey, value);
            OnPropertyChanged();
            Logger.LogDebug("Chart data mode changed to: {Mode}", value);
        }
    }

    public HistoryGranularity ChartGranularity
    {
        get => _settingsService.GetSetting(ChartSettings.GranularityKey, HistoryGranularity.Day);
        set
        {
            _settingsService.SetSetting(ChartSettings.GranularityKey, value);
            OnPropertyChanged();
            Logger.LogDebug("Chart granularity changed to: {Granularity}", value);
        }
    }

    public int ChartMaxPoints
    {
        get => _settingsService.GetSetting(ChartSettings.MaxPointsKey, 100);
        set
        {
            _settingsService.SetSetting(ChartSettings.MaxPointsKey, value);
            OnPropertyChanged();
            Logger.LogDebug("Chart max points changed to: {MaxPoints}", value);
        }
    }

    public int ChartTimeRangeDays
    {
        get => _settingsService.GetSetting(ChartSettings.TimeRangeDaysKey, 30);
        set
        {
            _settingsService.SetSetting(ChartSettings.TimeRangeDaysKey, value);
            OnPropertyChanged();
            Logger.LogDebug("Chart time range days changed to: {Days}", value);
        }
    }

    public bool ShowChartPredictions
    {
        get => _settingsService.GetSetting(ChartSettings.ShowPredictionsKey, true);
        set
        {
            _settingsService.SetSetting(ChartSettings.ShowPredictionsKey, value);
            OnPropertyChanged();
            Logger.LogDebug("Show chart predictions changed to: {Show}", value);
        }
    }

    public int ChartPredictionDaysAhead
    {
        get => _settingsService.GetSetting(ChartSettings.PredictionDaysAheadKey, 7);
        set
        {
            _settingsService.SetSetting(ChartSettings.PredictionDaysAheadKey, value);
            OnPropertyChanged();
            Logger.LogDebug("Chart prediction days ahead changed to: {Days}", value);
        }
    }

    // Enum options for binding (made static to satisfy lint)
    public static Array ChartDataModeOptions => Enum.GetValues(typeof(ChartDataMode));
    public static Array ChartGranularityOptions => Enum.GetValues(typeof(HistoryGranularity));

    partial void OnSelectedPredictionItemChanged(InventoryItemViewModel? value)
    {
        DebugService.LogDebug($"🔄 MAIN: OnSelectedPredictionItemChanged called - Item: {value?.Name ?? "null"}");

        // Safely dispose/clear the previous chart first
        try
        {
            if (StaticPredictionChart != null)
            {
                DebugService.LogDebug("🧹 PREDICTION: Clearing previous chart");
                StaticPredictionChart = null;
            }
        }
        catch (Exception ex)
        {
            DebugService.LogDebug($"⚠️ PREDICTION: Error clearing previous chart: {ex.Message}");
        }

        if (value != null)
        {
            try
            {
                DebugService.LogDebug($"📊 PREDICTION: Creating chart for item: {value.Name} (ID: {value.Id})");

                // Validate required services before creating chart
                if (_inventoryService == null)
                {
                    DebugService.LogDebug("❌ PREDICTION: _inventoryService is null, cannot create chart");
                    StaticPredictionChart = null;
                    return;
                }

                if (_settingsService == null)
                {
                    DebugService.LogDebug("❌ PREDICTION: _settingsService is null, cannot create chart");
                    StaticPredictionChart = null;
                    return;
                }

                var loggerFactory = Microsoft.Extensions.Logging.Abstractions.NullLoggerFactory.Instance;
                var chartLogger = loggerFactory.CreateLogger<InventoryLevelChartViewModel>();
                var chartViewModel = new InventoryLevelChartViewModel(_inventoryService, _settingsService, chartLogger)
                {
                    Item = value
                };
                StaticPredictionChart = chartViewModel;

                DebugService.LogDebug("✅ PREDICTION: Chart ViewModel created and assigned to StaticPredictionChart");

                // Trigger chart data refresh with proper UI thread marshalling
                _ = Task.Run(async () =>
                {
                    try
                    {
                        // Add a longer delay and recheck if the chart is still valid
                        await Task.Delay(200);

                        // Verify the chart is still the current one before refreshing
                        if (StaticPredictionChart != chartViewModel)
                        {
                            DebugService.LogDebug("⚠️ PREDICTION: Chart was replaced during async operation, skipping refresh");
                            return;
                        }

                        if (chartViewModel.Item?.Id != value.Id)
                        {
                            DebugService.LogDebug("⚠️ PREDICTION: Item changed during async operation, skipping refresh");
                            return;
                        }

                        DebugService.LogDebug($"🔄 PREDICTION: Starting chart data refresh for item: {value.Name}");

                        // Marshall back to UI thread for command execution
                        await Dispatcher.UIThread.InvokeAsync(async () =>
                        {
                            // Check if RefreshDataCommand is available (on UI thread)
                            if (chartViewModel.RefreshDataCommand == null)
                            {
                                DebugService.LogDebug("❌ PREDICTION: RefreshDataCommand is null");
                                return;
                            }

                            await chartViewModel.RefreshDataCommand.ExecuteAsync(null);
                            DebugService.LogDebug($"✅ PREDICTION: Chart data refresh completed for item: {value.Name}");
                        });
                    }
                    catch (Exception ex)
                    {
                        DebugService.LogDebug($"❌ PREDICTION: Failed to refresh chart data for item {value.Id}: {ex.Message}\n{ex.StackTrace}");
                    }
                });
            }
            catch (Exception ex)
            {
                DebugService.LogDebug($"❌ PREDICTION: Failed to create chart for item {value.Id}: {ex.Message}\n{ex.StackTrace}");
                StaticPredictionChart = null;
            }
        }
        else
        {
            DebugService.LogDebug("ℹ️ PREDICTION: No item selected, chart cleared");
        }
    }

    [ObservableProperty]
    private bool _isAddItemDialogVisible;

    [ObservableProperty]
    private AddItemDialogViewModel? _addItemDialog;

    // Property for XAML binding - returns filtered items for display
    public ObservableCollection<InventoryItemViewModel> DisplayedItems => FilteredItems;

    /// <summary>
    /// Exposes the inventory service for child components like InventoryItemCard
    /// </summary>
    public IInventoryService InventoryService => _inventoryService;

    public MainViewModel(IInventoryService inventoryService, IServiceClient serviceClient, ISettingsService settingsService, ILogger<MainViewModel> logger)
        : base(serviceClient, logger)
    {
        _inventoryService = inventoryService;
        _settingsService = settingsService;

        // Initialize cache settings visibility
        ShowCacheInfo = _settingsService.GetSetting("Debug.ShowCacheInfo", false);

        // Initialize default chart settings if not set
        InitializeChartSettings();

        // Set up auto-refresh timer based on settings
        InitializeAutoRefresh();

        // TEST: Create a simple static chart immediately to test Prediction tab binding
        try
        {
            DebugService.LogDebug("🏗️ MAIN: Creating test static chart in constructor");
            var testItem = new InventoryItemViewModel
            {
                Id = "test-id",
                Name = "Test Item",
                Description = "Test item for chart",
                CurrentLevel = 5.0,
                MaxCapacity = 10.0,
                LowStockThreshold = 2.0,
                UnitId = "kg"
            };

            var loggerFactory = Microsoft.Extensions.Logging.Abstractions.NullLoggerFactory.Instance;
            var chartLogger = loggerFactory.CreateLogger<InventoryLevelChartViewModel>();
            var chartViewModel = new InventoryLevelChartViewModel(_inventoryService, _settingsService, chartLogger)
            {
                Item = testItem
            };
            StaticPredictionChart = chartViewModel;
            DebugService.LogDebug("✅ MAIN: Test StaticPredictionChart created and assigned in constructor");

            // Also set the regular SelectedItemChart for comparison
            SelectedItemChart = chartViewModel;
            DebugService.LogDebug("✅ MAIN: Also assigned to SelectedItemChart for comparison");
        }
        catch (Exception ex)
        {
            DebugService.LogDebug($"❌ MAIN: Failed to create test static chart in constructor: {ex.Message}\n{ex.StackTrace}");
        }
    }

    partial void OnShowLowStockOnlyChanged(bool value)
    {
        UpdateFilteredItems();
        Logger.LogDebug("Filter updated due to ShowLowStockOnly change");
    }

    partial void OnSearchTextChanged(string value)
    {
        UpdateFilteredItems();
        Logger.LogDebug("Filter updated due to SearchText change");
    }

    partial void OnSortOptionChanged(string value)
    {
        UpdateFilteredItems();
        Logger.LogDebug("Filter updated due to SortOption change");
    }

    partial void OnFilterCategoryChanged(string value)
    {
        UpdateFilteredItems();
        Logger.LogDebug("Filter updated due to FilterCategory change");
    }

    protected override async Task RefreshDataAsync()
    {
        if (!await _refreshSemaphore.WaitAsync(TimeSpan.FromMilliseconds(100)))
        {
            Logger.LogDebug("RefreshDataAsync already running, skipping duplicate call");
            return;
        }

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

            // Clear and reload to prevent duplicates
            InventoryItems.Clear();

            foreach (var item in items)
            {
                // Ensure ProposedLevel is initialized if not set
                if (item.ProposedLevel <= 0)
                {
                    item.ProposedLevel = item.CurrentLevel;
                }
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
                    // Ensure ProposedLevel is initialized if not set
                    if (item.ProposedLevel <= 0)
                    {
                        item.ProposedLevel = item.CurrentLevel;
                    }
                    InventoryItems.Add(item);
                }
            }

            UpdateCounts();
            UpdateFilteredItems(); // Update filtered items after loading
            UpdateCacheInfo(); // Update cache information
            Logger.LogInformation("Successfully refreshed inventory data with {Count} items", InventoryItems.Count);

            // --- BEGIN STEP 1: Create a static chart for the Prediction tab ---
            try
            {
                var staticItem = InventoryItems.FirstOrDefault(i => i.Name.Equals("Flour", StringComparison.OrdinalIgnoreCase)) ?? InventoryItems.FirstOrDefault();
                if (staticItem != null)
                {
                    Logger.LogInformation("Creating static chart for item: {ItemName}", staticItem.Name);
                    var loggerFactory = Microsoft.Extensions.Logging.Abstractions.NullLoggerFactory.Instance;
                    var chartLogger = loggerFactory.CreateLogger<InventoryLevelChartViewModel>();
                    var chartViewModel = new InventoryLevelChartViewModel(_inventoryService, _settingsService, chartLogger)
                    {
                        Item = staticItem
                    };
                    StaticPredictionChart = chartViewModel;
                    _ = chartViewModel.RefreshDataCommand.ExecuteAsync(null);
                    Logger.LogInformation("StaticPredictionChart created and assigned.");
                }
                else
                {
                    Logger.LogWarning("Could not find a suitable item to create a static chart.");
                    StaticPredictionChart = null;
                }
            }
            catch (Exception ex)
            {
                Logger.LogError(ex, "Failed to create static prediction chart.");
                StaticPredictionChart = null;
            }
            // --- END STEP 1 ---
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to refresh inventory data");
            SetConnectionError($"Failed to refresh data: {ex.Message}");

            // Don't fall back to mock data - show the real error instead
            Logger.LogWarning("Not using mock data fallback to expose real connection issues");

            // Clear items to show empty state
            InventoryItems.Clear();
            UpdateCounts();
            UpdateFilteredItems();
            UpdateCacheInfo();
        }
        finally
        {
            _refreshSemaphore.Release();
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

    [RelayCommand]
    private async Task RefreshChartsAsync()
    {
        try
        {
            Logger.LogInformation("Refreshing charts with updated settings");

            // Refresh the current prediction chart if one exists
            if (StaticPredictionChart != null)
            {
                await StaticPredictionChart.RefreshDataCommand.ExecuteAsync(null);
            }

            // Refresh the selected item chart if one exists
            if (SelectedItemChart != null)
            {
                await SelectedItemChart.RefreshDataCommand.ExecuteAsync(null);
            }

            Logger.LogInformation("Chart refresh completed");
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Failed to refresh charts");
            SetConnectionError("Failed to refresh charts: " + ex.Message);
        }
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

    // [RelayCommand]
    // private async Task StartTraining()
    // {
    //     if (SelectedItemPredictionStatus == null || SelectedItem == null) return;
    //
    //     try
    //     {
    //         ClearConnectionError();
    //
    //         if (!IsConnected)
    //         {
    //             SetConnectionError(NotConnectedErrorMessage);
    //             return;
    //         }
    //
    //         IsLoading = true;
    //
    //         // For now, simulate starting training
    //         SelectedItemPredictionStatus.Stage = TrainingStage.Learning;
    //         SelectedItemPredictionStatus.TrainingStarted = DateTime.Now;
    //         SelectedItemPredictionStatus.LastUpdated = DateTime.Now;
    //
    //         // Simulate async operation
    //         await Task.Delay(100);
    //
    //         Logger.LogInformation("Started training for {ItemName} using {Model}",
    //             SelectedItem.Name, SelectedItemPredictionStatus.ActiveModel);
    //
    //         // Note: Real gRPC service call would be implemented here
    //     }
    //     catch (Exception ex)
    //     {
    //         var errorMessage = $"Failed to start training: {ex.Message}";
    //         SetConnectionError(errorMessage);
    //         Logger.LogError(ex, "Failed to start training for item {ItemId}", SelectedItem?.Id);
    //     }
    //     finally
    //     {
    //         IsLoading = false;
    //     }
    // }

    // [RelayCommand]
    // private async Task RefreshPredictionStatus()
    // {
    //     if (SelectedItemPredictionStatus == null || SelectedItem == null) return;
    //
    //     try
    //     {
    //         ClearConnectionError();
    //
    //         if (!IsConnected)
    //         {
    //             SetConnectionError(NotConnectedErrorMessage);
    //             return;
    //         }
    //
    //         IsLoading = true;
    //
    //         // For now, simulate refreshing status
    //         await Task.Delay(100);
    //         SelectedItemPredictionStatus.LastUpdated = DateTime.Now;
    //
    //         Logger.LogInformation("Refreshed prediction status for {ItemName}", SelectedItem.Name);
    //     }
    //     catch (Exception ex)
    //     {
    //         var errorMessage = $"Failed to refresh prediction status: {ex.Message}";
    //         SetConnectionError(errorMessage);
    //         Logger.LogError(ex, "Failed to refresh prediction status for item {ItemId}", SelectedItem?.Id);
    //     }
    //     finally
    //     {
    //         IsLoading = false;
    //     }
    // }
    //
    // [RelayCommand]
    // private async Task ApplyModelConfiguration()
    // {
    //     if (SelectedItemPredictionStatus == null || SelectedItem == null) return;
    //
    //     try
    //     {
    //         ClearConnectionError();
    //
    //         if (!IsConnected)
    //         {
    //             SetConnectionError(NotConnectedErrorMessage);
    //             return;
    //         }
    //
    //         IsLoading = true;
    //
    //         // Simulate async operation
    //         await Task.Delay(100);
    //         SelectedItemPredictionStatus.LastUpdated = DateTime.Now;
    //
    //         Logger.LogInformation("Applied model configuration for {ItemName}", SelectedItem.Name);
    //     }
    //     catch (Exception ex)
    //     {
    //         var errorMessage = $"Failed to apply model configuration: {ex.Message}";
    //         SetConnectionError(errorMessage);
    //         Logger.LogError(ex, "Failed to apply model configuration for item {ItemId}", SelectedItem?.Id);
    //     }
    //     finally
    //     {
    //         IsLoading = false;
    //     }
    // }

    private void InitializeChartSettings()
    {
        // Initialize default chart settings if they don't exist
        if (!_settingsService.HasSetting(ChartSettings.ModeKey))
            _settingsService.SetSetting(ChartSettings.ModeKey, ChartDataMode.Granularity);

        if (!_settingsService.HasSetting(ChartSettings.GranularityKey))
            _settingsService.SetSetting(ChartSettings.GranularityKey, HistoryGranularity.Day);

        if (!_settingsService.HasSetting(ChartSettings.MaxPointsKey))
            _settingsService.SetSetting(ChartSettings.MaxPointsKey, 100);

        if (!_settingsService.HasSetting(ChartSettings.TimeRangeDaysKey))
            _settingsService.SetSetting(ChartSettings.TimeRangeDaysKey, 30);

        if (!_settingsService.HasSetting(ChartSettings.ShowPredictionsKey))
            _settingsService.SetSetting(ChartSettings.ShowPredictionsKey, true);

        if (!_settingsService.HasSetting(ChartSettings.PredictionDaysAheadKey))
            _settingsService.SetSetting(ChartSettings.PredictionDaysAheadKey, 7);

        Logger.LogDebug("Chart settings initialized with defaults");
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

    // [RelayCommand]
    // private async Task GetPredictionForSelectedItem()
    // {
    //     if (SelectedItem == null) return;
    //
    //     try
    //     {
    //         ClearConnectionError();
    //         IsLoading = true;
    //
    //         var prediction = await _inventoryService.PredictConsumptionAsync(SelectedItem.Id, 30, false);
    //         if (prediction != null)
    //         {
    //             SelectedItem.PredictedDaysRemaining = prediction.PredictedDaysRemaining;
    //             SelectedItem.ConfidenceScore = prediction.ConfidenceScore;
    //             Logger.LogInformation("Updated prediction for {ItemName}: {Days} days remaining (confidence: {Confidence:P})",
    //                 SelectedItem.Name, prediction.PredictedDaysRemaining, prediction.ConfidenceScore);
    //         }
    //     }
    //     catch (Exception ex)
    //     {
    //         SetConnectionError($"Failed to get prediction: {ex.Message}");
    //         Logger.LogError(ex, "Failed to get prediction for item {ItemId}", SelectedItem.Id);
    //     }
    //     finally
    //     {
    //         IsLoading = false;
    //     }
    // }
    //
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
                item.ProposedLevel,
                "Manual update from UI",
                true);

            if (!success)
            {
                SetConnectionError("Failed to update inventory level. Check server connection.");
                return;
            }

            // Update the local item instead of full refresh
            item.CurrentLevel = item.ProposedLevel;
            item.LastUpdated = DateTime.Now;

            Logger.LogInformation("Successfully updated inventory level for {ItemName}", item.Name);
            UpdateCounts();
            UpdateFilteredItems();
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
    private async Task RemoveItem(InventoryItemViewModel item)
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
            var success = await _inventoryService.RemoveInventoryItemAsync(item.Id);

            if (!success)
            {
                SetConnectionError("Failed to remove inventory item. Check server connection.");
                return;
            }

            // Remove the item from the local collection
            InventoryItems.Remove(item);

            // If it was selected, clear the selection
            if (SelectedItem == item)
            {
                SelectedItem = null;
            }

            UpdateCounts();
            UpdateFilteredItems();
            UpdateCacheInfo();

            Logger.LogInformation("Successfully removed inventory item {ItemName} (ID: {ItemId})", item.Name, item.Id);
        }
        catch (Exception ex)
        {
            var errorMessage = $"Failed to remove inventory item: {ex.Message}";
            SetConnectionError(errorMessage);
            Logger.LogError(ex, "Failed to remove inventory item {ItemId}", item.Id);
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private async Task ShowItemChart(InventoryItemViewModel item)
    {
        if (item == null)
        {
            DebugService.LogDebug("❌ VIEW: ShowItemChart called with null item");
            return;
        }

        try
        {
            DebugService.LogDebug($"🚀 VIEW: Starting to show chart for item: {item.Name} (ID: {item.Id})");

            // Create chart viewmodel for the item
            DebugService.LogDebug("🏗️ VIEW: Creating chart viewmodel");
            var loggerFactory = Microsoft.Extensions.Logging.Abstractions.NullLoggerFactory.Instance;
            var chartLogger = loggerFactory.CreateLogger<InventoryLevelChartViewModel>();
            var chartViewModel = new InventoryLevelChartViewModel(_inventoryService, _settingsService, chartLogger);
            DebugService.LogDebug("✅ VIEW: Chart viewmodel created");

            // Set the item and initialize the chart
            DebugService.LogDebug($"🎯 VIEW: Setting item on chart viewmodel: {item.Name}");
            chartViewModel.Item = item;
            DebugService.LogDebug("✅ VIEW: Item set on chart viewmodel");

            // Show the chart overlay first
            DebugService.LogDebug("📊 VIEW: Assigning SelectedItemChart and setting IsChartVisible=true");
            SelectedItemChart = chartViewModel;
            IsChartVisible = true;
            DebugService.LogDebug($"✅ VIEW: Chart overlay should now be visible for item: {item.Name} - IsChartVisible={IsChartVisible}");

            // Trigger immediate data refresh
            DebugService.LogDebug("⏳ VIEW: Waiting 100ms for UI update");
            await Task.Delay(100); // Small delay to let UI update

            try
            {
                DebugService.LogDebug("🔄 VIEW: Calling RefreshDataCommand.ExecuteAsync");
                await chartViewModel.RefreshDataCommand.ExecuteAsync(null);
                DebugService.LogDebug($"✅ VIEW: Chart data refreshed successfully for item: {item.Name}");
            }
            catch (Exception refreshEx)
            {
                DebugService.LogDebug($"❌ VIEW: Failed to refresh chart data for item {item.Id}, but overlay is still shown: {refreshEx.Message}");
                // Don't fail the whole operation if refresh fails - the chart might still show something
            }

            DebugService.LogDebug($"🎉 VIEW: ShowItemChart completed successfully for item: {item.Name}");
        }
        catch (Exception ex)
        {
            DebugService.LogDebug($"❌ VIEW: Failed to show chart for item {item.Id}: {ex.Message}\n{ex.StackTrace}");
            SetConnectionError($"Failed to show item chart: {ex.Message}");

            // Ensure we clean up on error
            IsChartVisible = false;
            SelectedItemChart = null;
            DebugService.LogDebug("🧹 VIEW: Cleaned up chart state after error");
        }
    }

    [RelayCommand]
    private void CloseChart()
    {
        IsChartVisible = false;
        SelectedItemChart = null;
        Logger.LogDebug("Chart closed");
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
        // TODO: batch report dialog launches here, send multiple update requests or a new batch update request
        // For now, just refresh the data
        return RefreshDataAsync();
    }

    [RelayCommand]
    private async Task UpdateInventoryLevel(InventoryItemViewModel item)
    {
        // This command is called from the DataGrid action buttons
        await UpdateItemLevel(item);

        // Don't call RefreshDataAsync here - the UpdateItemLevel method already updates the UI
        // This prevents duplicate refresh calls and potential duplicate entries
    }
}
