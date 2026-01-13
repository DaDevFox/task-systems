using Avalonia.Controls;
using Avalonia.LogicalTree;
using InventoryClient.ViewModels;
using InventoryClient.Services;
using ScottPlot.Avalonia;
using System;
using System.Linq;

namespace InventoryClient.Views;

/// <summary>
/// Code-behind for the InventoryLevelChartView
/// </summary>
public partial class InventoryLevelChartView : UserControl
{
    public InventoryLevelChartView()
    {
        DebugService.LogDebug("🏗️ InventoryLevelChartView constructor START");
        try
        {
            InitializeComponent();
            DebugService.LogDebug("✅ InventoryLevelChartView InitializeComponent completed");

            DataContextChanged += OnDataContextChanged;
            DebugService.LogDebug("🔗 InventoryLevelChartView DataContextChanged event wired");

            this.AttachedToVisualTree += (_, __) =>
            {
                DebugService.LogDebug("🌳 InventoryLevelChartView AttachedToVisualTree event fired");
                // Ensure the visual tree is ready before wiring the chart
                Avalonia.Threading.Dispatcher.UIThread.Post(() =>
                {
                    DebugService.LogDebug("📮 Posted TryWireChartAndRefresh to UI thread");
                    TryWireChartAndRefresh();
                });
            };

            DebugService.LogDebug("✅ InventoryLevelChartView fully constructed and events wired");
        }
        catch (Exception ex)
        {
            DebugService.LogDebug($"❌ ERROR in InventoryLevelChartView constructor: {ex.Message}\n{ex.StackTrace}");
            throw;
        }
    }

    private void OnDataContextChanged(object? sender, EventArgs e)
    {
        try
        {
            var contextType = DataContext?.GetType().Name ?? "null";
            DebugService.LogDebug($"🔄 InventoryLevelChartView DataContextChanged - New DataContext type: {contextType}");
            
            if (DataContext is InventoryLevelChartViewModel vm)
            {
                DebugService.LogDebug($"📊 DataContext is InventoryLevelChartViewModel - Item: {vm.ItemName}");
            }
            
            TryWireChartAndRefresh();
        }
        catch (Exception ex)
        {
            DebugService.LogDebug($"❌ ERROR in OnDataContextChanged: {ex.Message}");
        }
    }

    private void TryWireChartAndRefresh()
    {
        try
        {
            DebugService.LogDebug("🔧 TryWireChartAndRefresh called");

            if (DataContext is InventoryLevelChartViewModel viewModel)
            {
                DebugService.LogDebug($"✅ Found InventoryLevelChartViewModel in DataContext - Item: {viewModel.ItemName}");

                var chartControl = this.FindControl<AvaPlot>("ChartControl");
                if (chartControl != null)
                {
                    DebugService.LogDebug("✅ Found AvaPlot control 'ChartControl', wiring to viewModel");
                    viewModel.SetChartControl(chartControl);
                    DebugService.LogDebug("🔗 Chart control set on viewModel, calling RefreshChart");
                    _ = viewModel.RefreshChart();
                    DebugService.LogDebug("✅ Chart control wired and refresh called successfully");
                }
                else
                {
                    DebugService.LogDebug("❌ Could not find AvaPlot control with name 'ChartControl'");
                    
                    // Debug: List all controls to see what's available
                    var allControls = this.GetLogicalDescendants().OfType<Control>().ToList();
                    DebugService.LogDebug($"🔍 All controls in view: {string.Join(", ", allControls.Select(c => $"{c.GetType().Name}({c.Name ?? "unnamed"})"))}");
                }
            }
            else
            {
                var contextType = DataContext?.GetType().Name ?? "null";
                DebugService.LogDebug($"❌ DataContext is not InventoryLevelChartViewModel: {contextType}");
            }
        }
        catch (Exception ex)
        {
            DebugService.LogDebug($"❌ ERROR in TryWireChartAndRefresh: {ex.Message}\n{ex.StackTrace}");
        }
    }
}
