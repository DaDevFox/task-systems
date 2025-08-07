using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using InventoryClient.Models;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using ScottPlot;
using ScottPlot.Avalonia;
using System.Collections.ObjectModel;

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
    private PredictionModel _selectedPredictionModel = PredictionModel.Linear;

    [ObservableProperty]
    private List<HistoricalDataPoint> _historicalData = new();

    [ObservableProperty]
    private List<PredictionDataPoint> _predictionData = new();

    public string ItemName => Item?.Name ?? "No Item Selected";
    public string CurrentLevelDisplay => Item != null ? 
        $"Current: {Item.CurrentLevel:F2} {Item.UnitId}" : string.Empty;
    public string PredictionSummary => PredictionData.Count > 0 ? 
        $"Predicted empty in {PredictionData.LastOrDefault()?.DaysRemaining:F1} days" : string.Empty;

    public InventoryLevelChartViewModel(IInventoryService inventoryService, ILogger<InventoryLevelChartViewModel> logger)
    {
        _inventoryService = inventoryService;
        _logger = logger;
        
        // Initialize available models
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
        }
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

    private async Task LoadHistoricalDataAsync()
    {
        if (Item == null)
            return;

        // Generate mock historical data - in a real application this would come from the backend
        var random = new Random();
        var historicalData = new List<HistoricalDataPoint>();
        var currentDate = DateTime.Now.Date.AddDays(-30);
        var level = Item.MaxCapacity * 0.8; // Start at 80% capacity

        for (int i = 0; i < 30; i++)
        {
            // Simulate some consumption pattern with randomness
            var consumption = random.NextDouble() * 5 + 1; // 1-6 units consumed per day
            if (random.NextDouble() < 0.1) // 10% chance of restocking
            {
                level = Item.MaxCapacity * (0.8 + random.NextDouble() * 0.2); // Restock to 80-100%
            }
            else
            {
                level = Math.Max(0, level - consumption);
            }

            historicalData.Add(new HistoricalDataPoint
            {
                Date = currentDate.AddDays(i),
                Level = level
            });
        }

        HistoricalData = historicalData;
    }

    private async Task LoadPredictionDataAsync()
    {
        if (Item == null)
            return;

        try
        {
            var predictions = new List<PredictionDataPoint>();
            
            // Make prediction requests for the next 5 days
            for (int day = 1; day <= 5; day++)
            {
                var prediction = await _inventoryService.PredictConsumptionAsync(
                    Item.Id, 
                    daysAhead: day, 
                    updateBehavior: false);

                if (prediction != null)
                {
                    predictions.Add(new PredictionDataPoint
                    {
                        Date = DateTime.Now.Date.AddDays(day),
                        PredictedLevel = Math.Max(0, Item.CurrentLevel - (prediction.Estimate * day)),
                        ConfidenceHigh = Math.Max(0, Item.CurrentLevel - (prediction.LowerBound * day)),
                        ConfidenceLow = Math.Max(0, Item.CurrentLevel - (prediction.UpperBound * day)),
                        DaysRemaining = prediction.PredictedDaysRemaining
                    });
                }
            }

            PredictionData = predictions;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error loading prediction data for item {ItemId}", Item.Id);
            
            // Fallback to mock prediction data
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
    }

    private async Task RefreshChart()
    {
        if (_chartControl == null || Item == null)
            return;

        await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
        {
            var plot = _chartControl.Plot;
            plot.Clear();

            // Plot historical data
            if (HistoricalData.Count > 0)
            {
                var historicalDates = HistoricalData.Select(d => d.Date.ToOADate()).ToArray();
                var historicalLevels = HistoricalData.Select(d => d.Level).ToArray();

                var historicalLine = plot.Add.Scatter(historicalDates, historicalLevels);
                historicalLine.Color = Color.FromHex("#3b82f6");
                historicalLine.LineWidth = 2;
                historicalLine.MarkerSize = 4;
                historicalLine.Label = "Historical Levels";
            }

            // Plot prediction data
            if (PredictionData.Count > 0)
            {
                var predictionDates = PredictionData.Select(d => d.Date.ToOADate()).ToArray();
                var predictionLevels = PredictionData.Select(d => d.PredictedLevel).ToArray();

                var predictionLine = plot.Add.Scatter(predictionDates, predictionLevels);
                predictionLine.Color = Color.FromHex("#7c3aed");
                predictionLine.LineWidth = 2;
                predictionLine.LineStyle = LineStyle.Dash;
                predictionLine.MarkerSize = 4;
                predictionLine.Label = "Predictions";

                // Add confidence band
                var confidenceHigh = PredictionData.Select(d => d.ConfidenceHigh).ToArray();
                var confidenceLow = PredictionData.Select(d => d.ConfidenceLow).ToArray();
                
                var confidenceBand = plot.Add.FillY(predictionDates, confidenceHigh, confidenceLow);
                confidenceBand.Color = Color.FromHex("#7c3aed").WithAlpha(50);
                confidenceBand.Label = "Confidence Range";
            }

            // Add low stock threshold line
            var lowStockLine = plot.Add.HorizontalLine(Item.LowStockThreshold);
            lowStockLine.Color = Color.FromHex("#ef4444");
            lowStockLine.LineWidth = 2;
            lowStockLine.LineStyle = LineStyle.Dot;
            lowStockLine.Label = "Low Stock Threshold";

            // Configure axes
            plot.Axes.DateTimeTicksBottom();
            plot.Axes.Left.Label.Text = $"Level ({Item.UnitId})";
            plot.Axes.Bottom.Label.Text = "Date";

            // Set Y-axis range
            plot.Axes.Left.Range.Min = 0;
            plot.Axes.Left.Range.Max = Item.MaxCapacity * 1.1;

            // Style the plot
            plot.FigureBackground.Color = Color.White;
            plot.DataBackground.Color = Color.White;
            plot.ShowLegend();

            _chartControl.Refresh();
        });
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
