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

    public InventoryLevelChartViewModel(IInventoryService inventoryService, ILogger<InventoryLevelChartViewModel> logger)
    {
        _inventoryService = inventoryService;
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
            return;

        try
        {
            await LoadHistoricalDataAsync();
            await LoadPredictionDataAsync();
            await RefreshChart();
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error refreshing chart data for item {ItemId}", Item.Id);
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
        try
        {
            // Fetch live history from the backend
            var snapshots = await _inventoryService.GetItemHistoryAsync(Item!.Id);
            var historicalData = snapshots
                .OrderBy(s => s.Timestamp)
                .Select(s => new HistoricalDataPoint
                {
                    Date = s.Timestamp,
                    Level = s.Level
                })
                .ToList();
            HistoricalData = historicalData;
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
            // Only create mock predictions if we have historical data to base on
            if (HistoricalData.Count == 0)
            {
                PredictionData = new List<PredictionDataPoint>();
                return Task.CompletedTask;
            }

            var mockPredictions = new List<PredictionDataPoint>();
            var currentLevel = Item.CurrentLevel;
            var dailyConsumption = 2.5; // Mock consumption rate

            for (int day = 1; day <= 5; day++)
            {
                var predictedLevel = Math.Max(0, currentLevel - (dailyConsumption * day));
                mockPredictions.Add(new PredictionDataPoint
                {
                    Date = DateTime.Now.Date.AddDays(day),
                    PredictedLevel = predictedLevel,
                    ConfidenceHigh = Math.Max(0, predictedLevel + 1),
                    ConfidenceLow = Math.Max(0, predictedLevel - 1),
                    DaysRemaining = predictedLevel > 0 ? currentLevel / dailyConsumption - day : 0
                });
            }

            PredictionData = mockPredictions;
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

                // Prepare default X range (last 30 days) in case no data is present
                double defaultXMin = DateTime.Now.Date.AddDays(-30).ToOADate();
                double defaultXMax = DateTime.Now.Date.ToOADate();
                DebugService.LogDebug($"📅 CHART: Default X range: {defaultXMin} to {defaultXMax}");

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

                // Configure axes - use simple configuration
                try
                {
                    plot.Axes.DateTimeTicksBottom();
                    plot.Axes.Left.Label.Text = $"Level ({Item.UnitId})";
                    plot.Axes.Bottom.Label.Text = "Date";

                    // Determine Y max to show full capacity
                    double yMax = Math.Max(Item.MaxCapacity * 1.1, 10);

                    // Apply limits explicitly (prevents empty or invalid ranges)
                    plot.Axes.SetLimits(defaultXMin, defaultXMax, 0, yMax);

                    _logger.LogDebug("Configured chart axes with SetLimits - X: [{XMin},{XMax}] Y: [0,{YMax}]", defaultXMin, defaultXMax, yMax);
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
