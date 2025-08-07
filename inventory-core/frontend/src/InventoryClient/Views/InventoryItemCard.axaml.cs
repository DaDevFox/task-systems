using Avalonia.Controls;
using Avalonia.Interactivity;
using InventoryClient.Models;
using InventoryClient.ViewModels;

namespace InventoryClient.Views;

/// <summary>
/// Code-behind for the InventoryItemCard
/// </summary>
public partial class InventoryItemCard : UserControl
{
    public InventoryItemCard()
    {
        InitializeComponent();
    }

    private void UpdateButton_Click(object sender, RoutedEventArgs e)
    {
        if (DataContext is InventoryItemViewModel item)
        {
            // Find the MainViewModel in the visual tree
            var mainViewModel = FindMainViewModel();
            if (mainViewModel?.UpdateItemLevelCommand.CanExecute(item) == true)
            {
                mainViewModel.UpdateItemLevelCommand.Execute(item);
            }
        }
    }

    private void ChartButton_Click(object sender, RoutedEventArgs e)
    {
        if (DataContext is InventoryItemViewModel item)
        {
            // Find the MainViewModel in the visual tree
            var mainViewModel = FindMainViewModel();
            if (mainViewModel?.ShowItemChartCommand.CanExecute(item) == true)
            {
                mainViewModel.ShowItemChartCommand.Execute(item);
            }
        }
    }

    private MainViewModel? FindMainViewModel()
    {
        // Walk up the visual tree to find the MainViewModel
        var current = this.Parent;
        while (current != null)
        {
            if (current.DataContext is MainViewModel mainViewModel)
                return mainViewModel;
            current = current.Parent;
        }
        return null;
    }
}
