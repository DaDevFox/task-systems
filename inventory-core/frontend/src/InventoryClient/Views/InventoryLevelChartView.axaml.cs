using Avalonia.Controls;
using InventoryClient.ViewModels;
using ScottPlot.Avalonia;

namespace InventoryClient.Views;

/// <summary>
/// Code-behind for the InventoryLevelChartView
/// </summary>
public partial class InventoryLevelChartView : UserControl
{
    public InventoryLevelChartView()
    {
        InitializeComponent();
        DataContextChanged += OnDataContextChanged;
    }

    private void OnDataContextChanged(object? sender, EventArgs e)
    {
        if (DataContext is InventoryLevelChartViewModel viewModel)
        {
            viewModel.SetChartControl(ChartControl);
        }
    }
}
