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
            CreateMiniChart(item);
        }
    }

    private void CreateMiniChart(InventoryItemViewModel item)
    {
        try
        {
            // Create a mini AvaPlot chart
            var miniChart = new AvaPlot
            {
                Width = 110,
                Height = 60,
                Margin = new Avalonia.Thickness(2)
            };

            // Generate simple mock data for mini chart
            var plot = miniChart.Plot;
            plot.Clear();

            // Simple trend line with a few points
            var dates = new[]
            {
                DateTime.Now.AddDays(-7).ToOADate(),
                DateTime.Now.AddDays(-5).ToOADate(),
                DateTime.Now.AddDays(-3).ToOADate(),
                DateTime.Now.AddDays(-1).ToOADate(),
                DateTime.Now.ToOADate()
            };

            // Generate some realistic-looking data based on current level
            var currentLevel = item.CurrentLevel;
            if (currentLevel <= 0 && item.LowStockThreshold <= 0)
            {
                // Show a simple placeholder instead of an empty line
                var presenter2 = this.FindControl<ContentPresenter>("MiniChartPresenter");
                if (presenter2 != null)
                {
                    presenter2.Content = new TextBlock
                    {
                        Text = "No history",
                        HorizontalAlignment = Avalonia.Layout.HorizontalAlignment.Center,
                        VerticalAlignment = Avalonia.Layout.VerticalAlignment.Center,
                        FontSize = 12,
                        Foreground = Avalonia.Media.Brushes.Gray
                    };
                }
                return;
            }

            var levels = new[]
            {
                Math.Max(0, currentLevel - 5 + (new Random().NextDouble() * 4 - 2)),
                Math.Max(0, currentLevel - 3 + (new Random().NextDouble() * 4 - 2)),
                Math.Max(0, currentLevel - 1 + (new Random().NextDouble() * 4 - 2)),
                Math.Max(0, currentLevel + 1 + (new Random().NextDouble() * 4 - 2)),
                currentLevel
            };

            try
            {
                var line = plot.Add.Scatter(dates, levels);
                line.Color = new ScottPlot.Color(59, 130, 246); // Blue
                line.LineWidth = 1.5f;
                line.MarkerSize = 0;

                // Add threshold line if applicable
                if (item.LowStockThreshold > 0)
                {
                    var thresholdLine = plot.Add.HorizontalLine(item.LowStockThreshold);
                    thresholdLine.Color = new ScottPlot.Color(220, 38, 38); // Red
                    thresholdLine.LineWidth = 1;
                    thresholdLine.LinePattern = ScottPlot.LinePattern.Dashed;
                }

                // Configure mini chart - very minimal
                plot.Axes.DateTimeTicksBottom();
                plot.Axes.Left.TickGenerator = new ScottPlot.TickGenerators.NumericAutomatic() { MaxTickCount = 3 };
                plot.Axes.Bottom.TickGenerator = new ScottPlot.TickGenerators.DateTimeAutomatic() { MaxTickCount = 3 };

                // Set range
                plot.Axes.Left.Range.Min = 0;
                plot.Axes.Left.Range.Max = Math.Max(item.MaxCapacity * 1.1, levels.Max() * 1.2);

                // Hide labels for mini chart
                plot.Axes.Left.Label.Text = "";
                plot.Axes.Bottom.Label.Text = "";

                // Make ticks smaller
                plot.Axes.Left.MajorTickStyle.Length = 2;
                plot.Axes.Bottom.MajorTickStyle.Length = 2;
                plot.Axes.Left.MinorTickStyle.Length = 1;
                plot.Axes.Bottom.MinorTickStyle.Length = 1;

                plot.Layout.Fixed(padding: new ScottPlot.PixelPadding(5, 5, 5, 5));

                DebugService.LogDebug("Created mini chart for item: {0} with {1} data points", item.Name, levels.Length);
            }
            catch (Exception ex)
            {
                DebugService.LogDebug("Failed to populate mini chart, using fallback: {0}", ex.Message);
                plot.Clear();
                plot.Add.Text("📊", 0.5, 0.5);
            }

            var presenter = this.FindControl<ContentPresenter>("MiniChartPresenter");
            if (presenter != null)
            {
                presenter.Content = miniChart;
                DebugService.LogDebug("Mini chart assigned to presenter for item: {0}", item.Name);
            }
            else
            {
                DebugService.LogDebug("MiniChartPresenter not found for item: {0}", item.Name);
            }
        }
        catch (Exception ex)
        {
            DebugService.LogDebug("Failed to create mini chart: {0}", ex.Message);

            // Fallback to text placeholder
            var placeholder = new TextBlock
            {
                Text = "📊",
                FontSize = 10,
                Foreground = Avalonia.Media.Brushes.Gray,
                HorizontalAlignment = Avalonia.Layout.HorizontalAlignment.Center,
                VerticalAlignment = Avalonia.Layout.VerticalAlignment.Center
            };

            var presenter = this.FindControl<ContentPresenter>("MiniChartPresenter");
            if (presenter != null)
            {
                presenter.Content = placeholder;
            }
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
