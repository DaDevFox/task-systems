using Avalonia.Controls;
using Avalonia.Controls.Presenters;
using Avalonia.Interactivity;
using InventoryClient.Models;
using InventoryClient.ViewModels;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using ScottPlot.Avalonia;
using System;
using System.Linq;

namespace InventoryClient.Views;

/// <summary>
/// Code-behind for the InventoryItemCard
/// </summary>
public partial class InventoryItemCard : UserControl
{
    // Setting key for mini chart history days
    private const string MiniChartHistoryDaysKey = "MiniChart.HistoryDays";
    private const int DefaultMiniChartHistoryDays = 14;
    public InventoryItemCard()
    {
        InitializeComponent();
        DebugService.LogDebug("InventoryItemCard initialized");
        DataContextChanged += OnDataContextChanged;
    }

    private void OnDataContextChanged(object? sender, EventArgs e)
    {
        if (DataContext is InventoryItemViewModel item)
        {
            // Schedule the chart creation on a background thread to avoid blocking UI
            _ = Task.Run(async () =>
            {
                try
                {
                    await CreateMiniChartAsync(item);
                }
                catch (Exception ex)
                {
                    DebugService.LogDebug("Failed to create mini chart: {0}", ex.Message);

                    // Fallback to UI thread for showing error message
                    await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
                    {
                        ShowNoDataAvailable("Chart unavailable");
                    });
                }
            });
        }
    }

    private async Task CreateMiniChartAsync(InventoryItemViewModel item)
    {
        try
        {
            DebugService.LogDebug("🔍 MINI_CHART: Starting CreateMiniChartAsync for item: {0}", item.Name);

            // Get the inventory service from the main view model's data context
            MainViewModel? mainViewModel = null;

            // We need to get the MainViewModel from the UI thread
            await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
            {
                mainViewModel = FindMainViewModel();
                DebugService.LogDebug("🔍 MINI_CHART: FindMainViewModel result: {0}", mainViewModel != null ? "Found" : "Not Found");
            });

            if (mainViewModel?.InventoryService == null)
            {
                DebugService.LogDebug("❌ MINI_CHART: InventoryService is null - mainViewModel: {0}, service: {1}",
                    mainViewModel != null, mainViewModel?.InventoryService != null);
                await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
                {
                    ShowNoDataAvailable("Service unavailable");
                });
                return;
            }

            DebugService.LogDebug("🔍 MINI_CHART: InventoryService found, checking connection status...");
            var isConnected = mainViewModel.InventoryService.IsConnected;
            DebugService.LogDebug("🔍 MINI_CHART: Connection status: {0}", isConnected);

            if (!isConnected)
            {
                await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
                {
                    ShowNoDataAvailable("Not connected to server");
                });
                return;
            }

            try
            {
                DebugService.LogDebug("🔍 MINI_CHART: Fetching historical data for item: {0}", item.Id);

                // Get the number of days to show from settings
                var historyDays = mainViewModel.SettingsService?.GetSetting(MiniChartHistoryDaysKey, DefaultMiniChartHistoryDays) ?? DefaultMiniChartHistoryDays;
                DebugService.LogDebug("🔍 MINI_CHART: Using {0} days of history from settings", historyDays);

                // Fetch actual historical data from the server
                var endTime = DateTime.UtcNow;
                var startTime = endTime.AddDays(-historyDays);

                DebugService.LogDebug("🔍 MINI_CHART: Time range: {0} to {1}", startTime.ToString("yyyy-MM-dd HH:mm"), endTime.ToString("yyyy-MM-dd HH:mm"));

                var historyData = await mainViewModel.InventoryService.GetItemHistoryAsync(
                    item.Id,
                    startTime,
                    endTime,
                    "HOUR", // Hourly granularity for mini chart
                    20 // Max 20 points for performance
                );

                DebugService.LogDebug("🔍 MINI_CHART: History data result: {0} records", historyData?.Count ?? 0);

                if (historyData == null || !historyData.Any())
                {
                    DebugService.LogDebug("⚠️ MINI_CHART: No historical data available for item: {0}", item.Id);
                    await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
                    {
                        ShowNoDataAvailable("No history available");
                    });
                    return;
                }

                DebugService.LogDebug("✅ MINI_CHART: Creating chart from {0} data points", historyData.Count);

                // Create chart on UI thread
                await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
                {
                    try
                    {
                        CreateChartFromData(item, historyData);
                    }
                    catch (Exception ex)
                    {
                        DebugService.LogDebug("❌ MINI_CHART: Failed to create chart from data: {0}", ex.Message);
                        ShowNoDataAvailable("Chart creation failed");
                    }
                });
            }
            catch (Exception ex)
            {
                DebugService.LogDebug("❌ MINI_CHART: Failed to fetch historical data: {0}", ex.Message);
                DebugService.LogDebug("❌ MINI_CHART: Exception details: {0}", ex.ToString());
                await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
                {
                    ShowNoDataAvailable("Failed to load data");
                });
            }
        }
        catch (Exception ex)
        {
            DebugService.LogDebug("❌ MINI_CHART: Top-level exception: {0}", ex.Message);
            await Avalonia.Threading.Dispatcher.UIThread.InvokeAsync(() =>
            {
                ShowNoDataAvailable("Chart unavailable");
            });
        }
    }

    private void CreateChartFromData(InventoryItemViewModel item, IReadOnlyList<InventoryLevelSnapshotViewModel> historyData)
    {
        // Create a mini AvaPlot chart
        var miniChart = new AvaPlot
        {
            Width = 110,
            Height = 60,
            Margin = new Avalonia.Thickness(2)
        };

        var plot = miniChart.Plot;
        plot.Clear();

        // Convert to chart data
        var dates = historyData.Select(h => h.Timestamp.ToOADate()).ToArray();
        var levels = historyData.Select(h => h.Level).ToArray();

        // Add historical data line
        var line = plot.Add.Scatter(dates, levels);
        line.Color = new ScottPlot.Color(59, 130, 246); // Blue
        line.LineWidth = 1.5f;
        line.MarkerSize = 0;

        // Add current level point if different from last historical point
        var lastHistoricalLevel = levels.LastOrDefault();
        if (Math.Abs(item.CurrentLevel - lastHistoricalLevel) > 0.01)
        {
            var currentX = new[] { DateTime.UtcNow.ToOADate() };
            var currentY = new[] { item.CurrentLevel };
            var currentPlot = plot.Add.Scatter(currentX, currentY);
            currentPlot.Color = new ScottPlot.Color(220, 38, 38); // Red
            currentPlot.MarkerSize = 3;
            currentPlot.LineWidth = 0;
        }

        // Add threshold line if applicable
        if (item.LowStockThreshold > 0)
        {
            var thresholdLine = plot.Add.HorizontalLine(item.LowStockThreshold);
            thresholdLine.Color = new ScottPlot.Color(255, 165, 0); // Orange
            thresholdLine.LineWidth = 1;
            thresholdLine.LinePattern = ScottPlot.LinePattern.Dashed;
        }

        // Configure mini chart - very minimal
        plot.Axes.DateTimeTicksBottom();
        plot.Axes.Left.TickGenerator = new ScottPlot.TickGenerators.NumericAutomatic() { MaxTickCount = 3 };
        plot.Axes.Bottom.TickGenerator = new ScottPlot.TickGenerators.DateTimeAutomatic() { MaxTickCount = 3 };

        // Set range
        plot.Axes.Left.Range.Min = 0;
        var maxY = Math.Max(item.MaxCapacity, Math.Max(levels.Max(), item.CurrentLevel)) * 1.1;
        plot.Axes.Left.Range.Max = maxY;

        // Hide labels for mini chart
        plot.Axes.Left.Label.Text = "";
        plot.Axes.Bottom.Label.Text = "";

        // Make ticks smaller and less intrusive
        plot.Axes.Left.MajorTickStyle.Length = 2;
        plot.Axes.Bottom.MajorTickStyle.Length = 2;
        plot.Axes.Left.MinorTickStyle.Length = 1;
        plot.Axes.Bottom.MinorTickStyle.Length = 1;
        plot.Axes.Left.TickLabelStyle.FontSize = 6;
        plot.Axes.Bottom.TickLabelStyle.FontSize = 6;

        plot.Layout.Fixed(padding: new ScottPlot.PixelPadding(5, 5, 5, 5));

        // Assign chart to presenter
        var presenter = this.FindControl<ContentPresenter>("MiniChartPresenter");
        if (presenter != null)
        {
            presenter.Content = miniChart;
            DebugService.LogDebug("Mini chart created for item: {0} with {1} historical data points", item.Name, historyData.Count);
        }
        else
        {
            DebugService.LogDebug("MiniChartPresenter not found for item: {0}", item.Name);
        }
    }

    private void ShowNoDataAvailable(string message)
    {
        var placeholder = new TextBlock
        {
            Text = message,
            FontSize = 8,
            Foreground = Avalonia.Media.Brushes.Gray,
            HorizontalAlignment = Avalonia.Layout.HorizontalAlignment.Center,
            VerticalAlignment = Avalonia.Layout.VerticalAlignment.Center,
            TextWrapping = Avalonia.Media.TextWrapping.Wrap
        };

        var presenter = this.FindControl<ContentPresenter>("MiniChartPresenter");
        if (presenter != null)
        {
            presenter.Content = placeholder;
        }
    }

    private void UpdateButton_Click(object sender, RoutedEventArgs e)
    {
        DebugService.LogDebug("UpdateButton_Click called");

        if (DataContext is InventoryItemViewModel item)
        {
            DebugService.LogDebug("UpdateButton clicked for item: {0} (ID: {1})", item.Name, item.Id);

            // Find the MainViewModel in the visual tree
            var mainViewModel = FindMainViewModel();
            if (mainViewModel == null)
            {
                DebugService.LogDebug("ERROR: MainViewModel not found in visual tree");
                return;
            }

            DebugService.LogDebug("Found MainViewModel, checking if UpdateInventoryLevelCommand can execute...");
            if (mainViewModel.UpdateInventoryLevelCommand.CanExecute(item))
            {
                DebugService.LogDebug("Executing UpdateInventoryLevelCommand for item: {0}", item.Name);
                mainViewModel.UpdateInventoryLevelCommand.Execute(item);
                DebugService.LogDebug("UpdateInventoryLevelCommand executed successfully");
            }
            else
            {
                DebugService.LogDebug("UpdateInventoryLevelCommand cannot execute for item: {0}", item.Name);
            }
        }
        else
        {
            DebugService.LogDebug("ERROR: DataContext is not InventoryItemViewModel. DataContext type: {0}",
                DataContext?.GetType().Name ?? "null");
        }
    }

    private void ChartButton_Click(object sender, RoutedEventArgs e)
    {
        DebugService.LogDebug("ChartButton_Click called");

        if (DataContext is InventoryItemViewModel item)
        {
            DebugService.LogDebug("ChartButton clicked for item: {0} (ID: {1})", item.Name, item.Id);

            // Find the MainViewModel in the visual tree
            var mainViewModel = FindMainViewModel();
            if (mainViewModel == null)
            {
                DebugService.LogDebug("ERROR: MainViewModel not found in visual tree");
                return;
            }

            DebugService.LogDebug("Found MainViewModel, checking if ShowItemChartCommand can execute...");
            if (mainViewModel.ShowItemChartCommand.CanExecute(item))
            {
                DebugService.LogDebug("Executing ShowItemChartCommand for item: {0}", item.Name);
                mainViewModel.ShowItemChartCommand.Execute(item);
                DebugService.LogDebug("ShowItemChartCommand executed successfully");
            }
            else
            {
                DebugService.LogDebug("ShowItemChartCommand cannot execute for item: {0}", item.Name);
            }
        }
        else
        {
            DebugService.LogDebug("ERROR: DataContext is not InventoryItemViewModel. DataContext type: {0}",
                DataContext?.GetType().Name ?? "null");
        }
    }

    private void RemoveButton_Click(object sender, RoutedEventArgs e)
    {
        DebugService.LogDebug("RemoveButton_Click called");

        if (DataContext is InventoryItemViewModel item)
        {
            DebugService.LogDebug("Remove button clicked for item: {0} (ID: {1})", item.Name, item.Id);

            // Find the MainViewModel in the visual tree
            var mainViewModel = FindMainViewModel();
            if (mainViewModel == null)
            {
                DebugService.LogDebug("ERROR: MainViewModel not found in visual tree");
                return;
            }

            DebugService.LogDebug("Found MainViewModel, checking if RemoveItemCommand can execute...");
            if (mainViewModel.RemoveItemCommand.CanExecute(item))
            {
                DebugService.LogDebug("Executing RemoveItemCommand for item: {0}", item.Name);
                mainViewModel.RemoveItemCommand.Execute(item);
                DebugService.LogDebug("RemoveItemCommand executed successfully");
            }
            else
            {
                DebugService.LogDebug("RemoveItemCommand cannot execute for item: {0}", item.Name);
            }
        }
        else
        {
            DebugService.LogDebug("ERROR: DataContext is not InventoryItemViewModel. DataContext type: {0}",
                DataContext?.GetType().Name ?? "null");
        }
    }

    private MainViewModel? FindMainViewModel()
    {
        DebugService.LogDebug("Searching for MainViewModel in visual tree...");

        // Walk up the visual tree to find the MainViewModel
        var current = this.Parent;
        int depth = 0;

        while (current != null)
        {
            depth++;
            DebugService.LogDebug("Checking parent at depth {0}: {1}", depth, current.GetType().Name);

            if (current.DataContext is MainViewModel mainViewModel)
            {
                DebugService.LogDebug("Found MainViewModel at depth {0} in {1}", depth, current.GetType().Name);
                return mainViewModel;
            }

            if (current.DataContext != null)
            {
                DebugService.LogDebug("Parent at depth {0} has DataContext of type: {1}", depth, current.DataContext.GetType().Name);
            }
            else
            {
                DebugService.LogDebug("Parent at depth {0} has null DataContext", depth);
            }

            current = current.Parent;

            // Prevent infinite loops
            if (depth > 20)
            {
                DebugService.LogDebug("Reached maximum search depth (20), stopping search");
                break;
            }
        }

        DebugService.LogDebug("MainViewModel not found in visual tree after searching {0} levels", depth);
        return null;
    }
}
