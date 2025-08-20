using CommunityToolkit.Mvvm.ComponentModel;
using InventoryClient.Models;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using ScottPlot;
using ScottPlot.Avalonia;

namespace InventoryClient.ViewModels;

/// <summary>
/// ViewModel for mini chart previews in item cards
/// </summary>
public partial class MiniChartViewModel : ObservableObject
{
    private readonly ILogger<MiniChartViewModel> _logger;
    private AvaPlot? _chartControl;

    [ObservableProperty]
    private InventoryItemViewModel? _item;

    public MiniChartViewModel(IInventoryService inventoryService, ILogger<MiniChartViewModel> logger)
    {
        _logger = logger;
    }

    public void SetChartControl(AvaPlot chartControl)
    {
        _chartControl = chartControl;
        if (Item != null)
        {
            _ = Task.Run(RefreshChart);
        }
    }

    public void SetItem(InventoryItemViewModel item)
    {
        Item = item;
        if (_chartControl != null)
        {
            _ = Task.Run(RefreshChart);
        }
    }

    private async Task RefreshChart()
    {
        if (_chartControl == null || Item == null)
            return;

        try
        {
            await Task.Run(() =>
            {
                // Create a mini trend chart with mock data
                _chartControl.Plot.Clear();

                // Generate some mock historical data
                var random = new Random(Item.Id.GetHashCode());
                var days = 14;
                var dataX = new double[days];
                var dataY = new double[days];

                var baseLevel = Item.CurrentLevel;
                for (int i = 0; i < days; i++)
                {
                    dataX[i] = i - days + 1; // Days ago (negative values)
                    // Simulate gradual consumption with some variation
                    dataY[i] = Math.Max(0, baseLevel + (random.NextDouble() - 0.3) * 2 + (i * 0.3));
                }

                // Add historical data line
                var historyPlot = _chartControl.Plot.Add.Scatter(dataX, dataY);
                historyPlot.Color = Colors.Blue;
                historyPlot.LineWidth = 1.5f;
                historyPlot.MarkerSize = 0;

                // Add current level point
                var currentX = new double[] { 0 };
                var currentY = new double[] { baseLevel };
                var currentPlot = _chartControl.Plot.Add.Scatter(currentX, currentY);
                currentPlot.Color = Colors.Red;
                currentPlot.MarkerSize = 4;
                currentPlot.LineWidth = 0;

                // Add low stock threshold line
                var thresholdPlot = _chartControl.Plot.Add.HorizontalLine(Item.LowStockThreshold);
                thresholdPlot.Color = Colors.Orange;
                thresholdPlot.LineWidth = 1;
                thresholdPlot.LinePattern = LinePattern.Dashed;

                // Configure mini chart appearance - hide axes and make it compact
                _chartControl.Plot.Axes.Bottom.IsVisible = false;
                _chartControl.Plot.Axes.Top.IsVisible = false;
                _chartControl.Plot.Axes.Left.IsVisible = false;
                _chartControl.Plot.Axes.Right.IsVisible = false;

                // Remove margins for compact view
                _chartControl.Plot.Axes.Margins(0, 0, 0.1, 0.1);

                _chartControl.Refresh();
            });
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to refresh mini chart for item {ItemId}", Item?.Id);
        }
    }
}
