// TODO: #15 - backs the chart view

using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using InventoryClient.Models;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using ScottPlot;
using ScottPlot.Avalonia;
using System.Collections.ObjectModel;
using Inventory.V1;

namespace InventoryClient.ViewModels;

/// <summary>
/// ViewModel for the inventory level chart component
/// </summary>
public partial class InventoryLevelChartViewModel : ObservableObject
{
    private readonly IInventoryService _inventoryService;
    private readonly ISettingsService _settingsService;
    private readonly ILogger<InventoryLevelChartViewModel> _logger;
    private AvaPlot? _chartControl;

    [ObservableProperty]
    private InventoryItemViewModel? _item;

    [ObservableProperty]
    private ObservableCollection<PredictionModel> _availablePredictionModels = new();

    [ObservableProperty]
    private PredictionModel _selectedPredictionModel = PredictionModel.Parametric;

    [ObservableProperty]
    private List<HistoricalDataPoint> _historicalData = new();

    [ObservableProperty]
    private List<PredictionDataPoint> _predictionData = new();

    // Consider at least 2 historical points as having plottable data; predictions alone also count
    public bool HasAnyData => (HistoricalData?.Count ?? 0) > 1 || (PredictionData?.Count ?? 0) > 0;

    public string ItemName => Item?.Name ?? "No Item Selected";
    public string CurrentLevelDisplay => Item != null ?
        $"Current: {Item.CurrentLevel:F2} {Item.UnitId}" : string.Empty;
    public string PredictionSummary => PredictionData.Count > 0 ?
        $"Predicted empty in {PredictionData.LastOrDefault()?.DaysRemaining:F1} days" : string.Empty;

    public InventoryLevelChartViewModel(
        IInventoryService inventoryService,
        ISettingsService settingsService,
        ILogger<InventoryLevelChartViewModel> logger)
    {
        _inventoryService = inventoryService;
        _settingsService = settingsService;
        _logger = logger;

        // Initialize available models - Use generated protobuf enum values
        AvailablePredictionModels = new ObservableCollection<PredictionModel>(
            Enum.GetValues<PredictionModel>().Where(m => m != PredictionModel.Unspecified));

        PropertyChanged += OnPropertyChanged;
    }

    public void SetChartControl(AvaPlot chartControl)
    {
        _chartControl = chartControl;
        if (Item != null)
        {
            _ = Task.Run(RefreshChart);
        }
    }

    private void OnPropertyChanged(object? sender, System.ComponentModel.PropertyChangedEventArgs e)
    {
        if (e.PropertyName == nameof(Item))
        {
            OnPropertyChanged(nameof(ItemName));
            OnPropertyChanged(nameof(CurrentLevelDisplay));

            if (Item != null)
            {
                _ = Task.Run(() => RefreshDataAsync());
            }
        }
        else if (e.PropertyName == nameof(SelectedPredictionModel))
        {
            if (Item != null)
            {
                _ = Task.Run(() => LoadPredictionDataAsync());
            }
        }
        else if (e.PropertyName == nameof(PredictionData))
        {
            OnPropertyChanged(nameof(PredictionSummary));
            _ = Task.Run(RefreshChart);
        }
        else if (e.PropertyName == nameof(HistoricalData))
        {
            _ = Task.Run(RefreshChart);
        }
    }

    partial void OnHistoricalDataChanged(List<HistoricalDataPoint> value)
    {
        OnPropertyChanged(nameof(HasAnyData));
    }

    partial void OnPredictionDataChanged(List<PredictionDataPoint> value)
    {
        OnPropertyChanged(nameof(HasAnyData));
    }

    [RelayCommand]
    private async Task RefreshData()
    {
        await RefreshDataAsync();
    }

    private async Task RefreshDataAsync()
    {
        if (Item == null)
        {
            _logger.LogDebug("Item is null, skipping chart data refresh");
            return;
        }

        try
        {
            _logger.LogDebug("Starting chart data refresh for item: {ItemId}", Item.Id);
            await LoadHistoricalDataAsync();
            await LoadPredictionDataAsync();
            await RefreshChart();
            _logger.LogDebug("Completed chart data refresh for item: {ItemId}", Item.Id);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error refreshing chart data for item {ItemId}", Item?.Id);
        }
    }

    private Task LoadHistoricalDataAsync()
    {
        if (Item == null)
            return Task.CompletedTask;

        return LoadHistoricalDataFromServiceAsync();
    }

    private async Task LoadHistoricalDataFromServiceAsync()
    {
        if (Item == null)
        {
            _logger.LogDebug("Item is null, skipping historical data load");
            HistoricalData = new List<HistoricalDataPoint>();
            return;
        }

        try
        {
            _logger.LogDebug("Loading historical data for item: {ItemId}", Item.Id);

            // Validate services before proceeding
            if (_settingsService == null)
            {
                _logger.LogError("Settings service is null, cannot load historical data");
                HistoricalData = new List<HistoricalDataPoint>();
                return;
            }

            if (_inventoryService == null)
            {
                _logger.LogError("Inventory service is null, cannot load historical data");
                HistoricalData = new List<HistoricalDataPoint>();
                return;
            }

            // Get chart settings from configuration
            var chartSettings = ChartSettings.FromSettings(_settingsService);

            DateTime? startTime = null;
            DateTime? endTime = null;
            string? granularity = null;
            int? maxPoints = null;

            // Configure request based on chart data mode
            if (chartSettings.Mode == ChartDataMode.Granularity)
            {
                // Use granularity-based sampling
                granularity = chartSettings.Granularity.ToString();
                maxPoints = chartSettings.MaxPoints;

                _logger.LogDebug("📊 CHART MODE: Granularity - Using granularity: {Granularity}, max points: {MaxPoints}",
                    granularity, maxPoints);
                DebugService.LogDebug($"📊 CHART MODE: Granularity - Granularity: {granularity}, MaxPoints: {maxPoints}");
            }
            else if (chartSettings.Mode == ChartDataMode.TimeRange)
            {
                // Fetch all data within time range
                endTime = DateTime.UtcNow;
                startTime = endTime.Value.AddDays(-chartSettings.TimeRangeDays);

                _logger.LogDebug("📊 CHART MODE: TimeRange - Loading data from {StartTime} to {EndTime} ({Days} days)",
                    startTime, endTime, chartSettings.TimeRangeDays);
                DebugService.LogDebug($"📊 CHART MODE: TimeRange - From: {startTime:yyyy-MM-dd HH:mm} to {endTime:yyyy-MM-dd HH:mm} ({chartSettings.TimeRangeDays} days)");
            }

            // Fetch live history from the backend with settings-based parameters
            var snapshots = await _inventoryService.GetItemHistoryAsync(
                Item.Id,
                startTime,
                endTime,
                granularity,
                maxPoints);

            if (snapshots == null)
            {
                _logger.LogWarning("GetItemHistoryAsync returned null for item {ItemId}", Item.Id);
                HistoricalData = new List<HistoricalDataPoint>();
                return;
            }

            var historicalData = snapshots
                .OrderBy(s => s.Timestamp)
                .Select(s => new HistoricalDataPoint
                {
                    Date = s.Timestamp,
                    Level = s.Level
                })
                .ToList();

            HistoricalData = historicalData;

            _logger.LogInformation("Loaded {Count} historical data points for item {ItemId} using {Mode} mode",
                historicalData.Count, Item.Id, chartSettings.Mode);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error loading historical data for item {ItemId}", Item?.Id);
            HistoricalData = new List<HistoricalDataPoint>();
        }
    }

    private Task LoadPredictionDataAsync()
    {
        if (Item == null)
            return Task.CompletedTask;

        try
        {
            // Get chart settings to determine if predictions should be shown
            var chartSettings = ChartSettings.FromSettings(_settingsService);

            if (!chartSettings.ShowPredictions)
            {
                PredictionData = new List<PredictionDataPoint>();
                _logger.LogDebug("Predictions disabled in settings, skipping prediction data generation");
                return Task.CompletedTask;
            }

            // NOTE: Prediction functionality is disabled until v1
            // In v0, we only show historical data without prediction models

            PredictionData = new List<PredictionDataPoint>();
            _logger.LogDebug("Predictions are disabled in v0 - only historical data is shown");
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error loading prediction data for item {ItemId}", Item.Id);

            // On error, leave predictions empty so UI can show fallback
            PredictionData = new List<PredictionDataPoint>();
        }

        return Task.CompletedTask;
    }

    public async Task RefreshChart()
    {
        DebugService.LogDebug($"📊 CHART: RefreshChart called - ChartControl: {_chartControl != null}, Item: {Item?.Name ?? "null"}");

        if (_chartControl == null || Item == null)
        {
            DebugService.LogDebug($"❌ CHART: Cannot refresh chart - chartControl: {_chartControl != null}, item: {Item != null}");
            _logger.LogWarning("Cannot refresh chart - chartControl: {HasChart}, item: {HasItem}",
                _chartControl != null, Item != null);
            return;
        }

        try
        {
            DebugService.LogDebug("🔄 CHART: Invoking refresh on UI thread");
            await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
            {
                DebugService.LogDebug($"🎨 CHART: Starting chart refresh for item: {Item.Name}");
                _logger.LogDebug("Refreshing chart for item: {ItemName}", Item.Name);

                var plot = _chartControl.Plot;
                DebugService.LogDebug("🧹 CHART: Clearing plot");
                plot.Clear();

                // Ensure visible styling regardless of theme
                try
                {
                    DebugService.LogDebug("🎨 CHART: Setting plot background colors");
                    plot.FigureBackground.Color = new ScottPlot.Color(255, 255, 255); // white
                    plot.DataBackground.Color = new ScottPlot.Color(255, 255, 255);   // white
                    DebugService.LogDebug("✅ CHART: Plot background colors set");
                }
                catch (Exception bgEx)
                {
                    DebugService.LogDebug($"⚠️ CHART: Failed to set background colors: {bgEx.Message}");
                }

                // Prepare default X range based on chart mode and data
                double defaultXMin, defaultXMax;

                // Get chart settings for proper range calculation
                var chartSettings = ChartSettings.FromSettings(_settingsService);

                if (chartSettings.Mode == ChartDataMode.TimeRange && HistoricalData.Count == 0)
                {
                    // For TimeRange mode with no data, show the requested time range
                    var endDate = DateTime.Now.Date;
                    var startDate = endDate.AddDays(-chartSettings.TimeRangeDays);
                    defaultXMin = startDate.ToOADate();
                    defaultXMax = endDate.ToOADate();
                    DebugService.LogDebug($"📅 CHART: TimeRange mode with no data - X range: {startDate:yyyy-MM-dd} to {endDate:yyyy-MM-dd}");
                }
                else if (HistoricalData.Count > 0)
                {
                    // Base range on actual data
                    var dataStartDate = HistoricalData.Min(d => d.Date);

                    if (chartSettings.Mode == ChartDataMode.Granularity)
                    {
                        // For Granularity mode: start 2 days before earliest data point
                        defaultXMin = dataStartDate.Date.AddDays(-2).ToOADate();
                        defaultXMax = DateTime.Now.Date.ToOADate(); // Present maximum
                        DebugService.LogDebug($"📅 CHART: Granularity mode - X range: {dataStartDate.Date.AddDays(-2):yyyy-MM-dd} to {DateTime.Now.Date:yyyy-MM-dd}");
                    }
                    else
                    {
                        // For TimeRange mode: use the requested time range minimum to present maximum
                        var requestStartDate = DateTime.Now.Date.AddDays(-chartSettings.TimeRangeDays);
                        defaultXMin = requestStartDate.ToOADate();
                        defaultXMax = DateTime.Now.Date.ToOADate();
                        DebugService.LogDebug($"📅 CHART: TimeRange mode with data - X range: {requestStartDate:yyyy-MM-dd} to {DateTime.Now.Date:yyyy-MM-dd}");
                    }
                }
                else
                {
                    // Fallback: last 30 days
                    defaultXMin = DateTime.Now.Date.AddDays(-30).ToOADate();
                    defaultXMax = DateTime.Now.Date.ToOADate();
                    DebugService.LogDebug($"📅 CHART: Fallback - X range: {DateTime.Now.Date.AddDays(-30):yyyy-MM-dd} to {DateTime.Now.Date:yyyy-MM-dd}");
                }

                // Plot historical data with simple line
                if (HistoricalData.Count > 0)
                {
                    var historicalDates = HistoricalData.Select(d => d.Date.ToOADate()).ToArray();
                    var historicalLevels = HistoricalData.Select(d => d.Level).ToArray();

                    try
                    {
                        var historicalLine = plot.Add.Scatter(historicalDates, historicalLevels);
                        historicalLine.Color = new ScottPlot.Color(0, 0, 255); // Blue
                        historicalLine.LineWidth = 2;
                        historicalLine.MarkerSize = 0; // No markers for cleaner look
                        _logger.LogDebug("Added historical data line with {Count} points", HistoricalData.Count);

                        // Update default X range to data extents
                        defaultXMin = historicalDates.Min();
                        defaultXMax = historicalDates.Max();
                    }
                    catch (Exception ex)
                    {
                        _logger.LogError(ex, "Failed to add historical line to chart");
                    }
                }
                else
                {
                    _logger.LogWarning("No historical data to plot");
                }

                // Plot prediction data
                if (PredictionData.Count > 0)
                {
                    var predictionDates = PredictionData.Select(d => d.Date.ToOADate()).ToArray();
                    var predictionLevels = PredictionData.Select(d => d.PredictedLevel).ToArray();

                    try
                    {
                        var predictionLine = plot.Add.Scatter(predictionDates, predictionLevels);
                        predictionLine.Color = new ScottPlot.Color(128, 0, 128); // Purple
                        predictionLine.LineWidth = 2;
                        predictionLine.MarkerSize = 0;
                        predictionLine.LinePattern = LinePattern.Dashed;
                        _logger.LogDebug("Added prediction data line with {Count} points", PredictionData.Count);

                        // Expand default X range to include predictions
                        defaultXMin = Math.Min(defaultXMin, predictionDates.Min());
                        defaultXMax = Math.Max(defaultXMax, predictionDates.Max());
                    }
                    catch (Exception ex)
                    {
                        _logger.LogError(ex, "Failed to add prediction line to chart");
                    }
                }
                else
                {
                    _logger.LogWarning("No prediction data to plot");
                }

                // Add low stock threshold line
                try
                {
                    var lowStockLine = plot.Add.HorizontalLine(Item.LowStockThreshold);
                    lowStockLine.Color = new ScottPlot.Color(255, 0, 0); // Red
                    lowStockLine.LineWidth = 1;
                    lowStockLine.LinePattern = LinePattern.Dashed;
                    _logger.LogDebug("Added low stock threshold line at level {Threshold}", Item.LowStockThreshold);
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Failed to add threshold line to chart");
                }

                // Configure axes - use smart Y-axis bounds
                try
                {
                    plot.Axes.DateTimeTicksBottom();
                    plot.Axes.Left.Label.Text = $"Level ({Item.UnitId})";
                    plot.Axes.Bottom.Label.Text = "Date";

                    // Determine Y-axis bounds based on data
                    double yMin = 0; // Default to 0
                    double yMax = Math.Max(Item.MaxCapacity * 1.1, 10); // Default based on capacity

                    // Check if data contains negative values
                    var allValues = new List<double>();
                    if (HistoricalData.Count > 0)
                        allValues.AddRange(HistoricalData.Select(d => d.Level));
                    if (PredictionData.Count > 0)
                        allValues.AddRange(PredictionData.Select(d => d.PredictedLevel));

                    // Include threshold in Y-axis calculation
                    allValues.Add(Item.LowStockThreshold);
                    allValues.Add(Item.MaxCapacity);

                    if (allValues.Count > 0)
                    {
                        var minValue = allValues.Min();
                        var maxValue = allValues.Max();

                        // Only use negative Y minimum if data actually contains negative values
                        if (minValue < 0)
                        {
                            yMin = minValue * 1.1; // Add 10% padding below minimum
                            DebugService.LogDebug($"📊 CHART: Data contains negative values, Y min: {yMin:F1}");
                        }
                        else
                        {
                            yMin = 0; // Keep at 0 for positive-only data
                            DebugService.LogDebug($"📊 CHART: Data is positive-only, Y min: 0");
                        }

                        // Set Y maximum with padding
                        yMax = Math.Max(maxValue * 1.1, yMax);
                        DebugService.LogDebug($"📊 CHART: Y max set to: {yMax:F1} (data max: {maxValue:F1})");
                    }

                    // Apply limits explicitly (prevents empty or invalid ranges)
                    plot.Axes.SetLimits(defaultXMin, defaultXMax, yMin, yMax);

                    _logger.LogDebug("Configured chart axes with SetLimits - X: [{XMin:F0},{XMax:F0}] Y: [{YMin:F1},{YMax:F1}]",
                        defaultXMin, defaultXMax, yMin, yMax);
                    DebugService.LogDebug($"📊 CHART: Final axis bounds - X: [{DateTime.FromOADate(defaultXMin):yyyy-MM-dd},{DateTime.FromOADate(defaultXMax):yyyy-MM-dd}] Y: [{yMin:F1},{yMax:F1}]");
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Failed to configure chart axes");
                }

                // If no data at all, draw a friendly placeholder
                if (HistoricalData.Count < 2 && PredictionData.Count == 0)
                {
                    try
                    {
                        var centerX = (defaultXMin + defaultXMax) / 2.0;
                        var centerY = (plot.Axes.Left.Range.Min + plot.Axes.Left.Range.Max) / 2.0;
                        var txt = plot.Add.Text("No history available", centerX, centerY);
                        txt.Color = new ScottPlot.Color(100, 116, 139); // slate-500
                        // alignment not supported in current API version
                    }
                    catch { /* placeholder not critical */ }
                }

                // Refresh the chart
                try
                {
                    _chartControl.Refresh();
                    _logger.LogDebug("Chart refreshed successfully for item: {ItemName}", Item.Name);
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Failed to refresh chart control");
                }
            });
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error in RefreshChart for item {ItemId}", Item.Id);
        }
    }
}

/// <summary>
/// Represents a historical data point
/// </summary>
public class HistoricalDataPoint
{
    public DateTime Date { get; set; }
    public double Level { get; set; }
}

/// <summary>
/// Represents a prediction data point
/// </summary>
public class PredictionDataPoint
{
    public DateTime Date { get; set; }
    public double PredictedLevel { get; set; }
    public double ConfidenceHigh { get; set; }
    public double ConfidenceLow { get; set; }
    public double DaysRemaining { get; set; }
}
