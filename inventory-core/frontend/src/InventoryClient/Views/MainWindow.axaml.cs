using Avalonia.Controls;
using InventoryClient.ViewModels;

namespace InventoryClient.Views;

public partial class MainWindow : Window
{
    public MainWindow()
    {
        InitializeComponent();

        // Subscribe to dialog events when DataContext is set
        DataContextChanged += OnDataContextChanged;
    }

    private void OnDataContextChanged(object? sender, EventArgs e)
    {
        // Wire up dialog events when DataContext is available
        // In a production app, you would use a proper dialog service
    }
}
