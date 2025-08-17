using Avalonia.Controls;
using Avalonia.Input;
using Avalonia.Interactivity;
using InventoryClient.ViewModels;

namespace InventoryClient.Views;

/// <summary>
/// Code-behind for the AddItemDialog
/// </summary>
public partial class AddItemDialog : UserControl
{
    public AddItemDialog()
    {
        InitializeComponent();

        // Set focus to the first input when the dialog loads
        Loaded += OnLoaded;
    }

    private void OnLoaded(object? sender, RoutedEventArgs e)
    {
        // Focus the dialog itself first to ensure it can receive keyboard events
        this.Focus();

        // Then focus the first TextBox (Item Name) when the dialog opens
        var itemNameTextBox = this.FindControl<TextBox>("ItemNameTextBox");
        itemNameTextBox?.Focus();
    }

    protected override void OnKeyDown(KeyEventArgs e)
    {
        // Handle keyboard shortcuts at the UserControl level
        if (DataContext is AddItemDialogViewModel vm)
        {
            if (e.Key == Key.Escape)
            {
                vm.CancelCommand.Execute(null);
                e.Handled = true;
                return;
            }
            else if (e.KeyModifiers == KeyModifiers.Control && e.Key == Key.Enter && vm.AddItemCommand.CanExecute(null))
            {
                vm.AddItemCommand.Execute(null);
                e.Handled = true;
                return;
            }
        }

        base.OnKeyDown(e);
    }
}
