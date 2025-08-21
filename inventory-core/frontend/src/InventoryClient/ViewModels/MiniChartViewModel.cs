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
    private readonly IInventoryService _inventoryService;
    private AvaPlot? _chartControl;

    [ObservableProperty]
    private InventoryItemViewModel? _item;

    public MiniChartViewModel(IInventoryService inventoryService, ILogger<MiniChartViewModel> logger)
    {
        _inventoryService = inventoryService;
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
            await Task.Run(async () =>
            {
                _chartControl.Plot.Clear();

                if (!_inventoryService.IsConnected)
                {
                    ShowNoDataMessage("Not connected");
                    return;
                }

                try
                {
                    // Fetch real historical data from the server
                    var endTime = DateTime.UtcNow;
                    var startTime = endTime.AddDays(-7); // Last week for mini chart
                    var historyData = await _inventoryService.GetItemHistoryAsync(
                        Item.Id,
                        startTime,
                        endTime,
                        "HOUR", // Hourly granularity
                        15 // Max 15 points for mini chart
                    );

                    if (historyData == null || !historyData.Any())
                    {
                        ShowNoDataMessage("No data");
                        return;
                    }

                    // Convert to chart data
                    var dataX = historyData.Select((h, i) => (double)i).ToArray(); // Use index for X axis
                    var dataY = historyData.Select(h => h.Level).ToArray();

                    // Add historical data line
                    var historyPlot = _chartControl.Plot.Add.Scatter(dataX, dataY);
                    historyPlot.Color = Colors.Blue;
                    historyPlot.LineWidth = 1.5f;
                    historyPlot.MarkerSize = 0;

                    // Add current level point if different
                    var lastHistoricalLevel = dataY.LastOrDefault();
                    if (Math.Abs(Item.CurrentLevel - lastHistoricalLevel) > 0.01)
                    {
                        var currentX = new[] { dataX.LastOrDefault() + 1 };
                        var currentY = new[] { Item.CurrentLevel };
                        var currentPlot = _chartControl.Plot.Add.Scatter(currentX, currentY);
                        currentPlot.Color = Colors.Red;
                        currentPlot.MarkerSize = 4;
                        currentPlot.LineWidth = 0;
                    }

                    // Add low stock threshold line
                    if (Item.LowStockThreshold > 0)
                    {
                        var thresholdPlot = _chartControl.Plot.Add.HorizontalLine(Item.LowStockThreshold);
                        thresholdPlot.Color = Colors.Orange;
                        thresholdPlot.LineWidth = 1;
                        thresholdPlot.LinePattern = LinePattern.Dashed;
                    }

                    // Configure mini chart appearance - hide axes and make it compact
                    _chartControl.Plot.Axes.Bottom.IsVisible = false;
                    _chartControl.Plot.Axes.Top.IsVisible = false;
                    _chartControl.Plot.Axes.Left.IsVisible = false;
                    _chartControl.Plot.Axes.Right.IsVisible = false;

                    // Remove margins for compact view
                    _chartControl.Plot.Axes.Margins(0, 0, 0.1, 0.1);
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Failed to fetch historical data for mini chart for item {ItemId}", Item?.Id);
                    ShowNoDataMessage("Data error");
                }

                _chartControl.Refresh();
            });
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to refresh mini chart for item {ItemId}", Item?.Id);
        }
    }

    private void ShowNoDataMessage(string message)
    {
        // Add a simple text message to the chart
        _chartControl?.Plot.Add.Text(message, 0.5, 0.5);
        _chartControl?.Refresh();
    }
}
