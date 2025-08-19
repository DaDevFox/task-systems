using Avalonia.Controls;
using Avalonia.Interactivity;
using InventoryClient.Models;
using InventoryClient.ViewModels;
using InventoryClient.Services;
using System;

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

            // // Find the MainViewModel in the visual tree
            // var mainViewModel = FindMainViewModel();
            // if (mainViewModel == null)
            // {
            //     DebugService.LogDebug("ERROR: MainViewModel not found in visual tree");
            //     return;
            // }
            //
            // DebugService.LogDebug("Found MainViewModel, checking if ShowItemChartCommand can execute...");
            // if (mainViewModel.ShowItemChartCommand.CanExecute(item))
            // {
            //     DebugService.LogDebug("Executing ShowItemChartCommand for item: {0}", item.Name);
            //     mainViewModel.ShowItemChartCommand.Execute(item);
            //     DebugService.LogDebug("ShowItemChartCommand executed successfully");
            // }
            // else
            // {
            //     DebugService.LogDebug("ShowItemChartCommand cannot execute for item: {0}", item.Name);
            // }
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
