using Avalonia.Controls;
using InventoryClient.ViewModels;
using ScottPlot.Avalonia;

namespace InventoryClient.Views;

/// <summary>
/// Code-behind for the MiniChartView
/// </summary>
public partial class MiniChartView : UserControl
{
    public MiniChartView()
    {
        InitializeComponent();
        DataContextChanged += OnDataContextChanged;
    }

    private void OnDataContextChanged(object? sender, EventArgs e)
    {
        if (DataContext is MiniChartViewModel viewModel)
        {
            var chartControl = this.FindControl<AvaPlot>("MiniChartControl");
            if (chartControl != null)
            {
                viewModel.SetChartControl(chartControl);
            }
        }
    }
}
