using Avalonia.Controls;
using Avalonia.Interactivity;
using InventoryClient.Models;
using InventoryClient.ViewModels;

namespace InventoryClient.Views;

public partial class PredictionModelView : UserControl
{
    public PredictionModelView()
    {
        InitializeComponent();
    }

    private MainViewModel? GetMainViewModel()
    {
        // Walk up the DataContext chain to find the MainViewModel
        var current = this.Parent;
        while (current != null)
        {
            if (current.DataContext is MainViewModel mainViewModel)
                return mainViewModel;
            current = current.Parent;
        }
        return null;
    }

    private void StartTrainingButton_Click(object sender, RoutedEventArgs e)
    {
        var mainViewModel = GetMainViewModel();
        if (mainViewModel?.StartTrainingCommand.CanExecute(null) == true)
        {
            mainViewModel.StartTrainingCommand.Execute(null);
        }
    }

    private void RefreshButton_Click(object sender, RoutedEventArgs e)
    {
        var mainViewModel = GetMainViewModel();
        if (mainViewModel?.RefreshPredictionStatusCommand.CanExecute(null) == true)
        {
            mainViewModel.RefreshPredictionStatusCommand.Execute(null);
        }
    }

    private void ApplyConfigButton_Click(object sender, RoutedEventArgs e)
    {
        var mainViewModel = GetMainViewModel();
        if (mainViewModel?.ApplyModelConfigurationCommand.CanExecute(null) == true)
        {
            mainViewModel.ApplyModelConfigurationCommand.Execute(null);
        }
    }

    private void ViewAnalyticsButton_Click(object sender, RoutedEventArgs e)
    {
        // TODO: Open analytics window/dialog
        // For now, just show a simple message
        if (DataContext is PredictionTrainingStatusViewModel viewModel)
        {
            // In a real app, this would open a detailed analytics window
        }
    }
}
