using Avalonia.Controls;
using Avalonia.Input;
using Avalonia.Interactivity;
using InventoryClient.ViewModels;
using InventoryClient.Services;

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

    private void DebugValidation_Click(object sender, RoutedEventArgs e)
    {
        if (DataContext is AddItemDialogViewModel vm)
        {
            DebugService.LogDebug("=== VALIDATION DEBUG ===");
            DebugService.LogDebug("Name: '{0}'", vm.Name);
            DebugService.LogDebug("UnitId: '{0}'", vm.UnitId);
            DebugService.LogDebug("MaxCapacity: {0}", vm.MaxCapacity);
            DebugService.LogDebug("InitialLevel: {0}", vm.InitialLevel);
            DebugService.LogDebug("LowStockThreshold: {0}", vm.LowStockThreshold);
            DebugService.LogDebug("IsValid: {0}", vm.IsValid);
            DebugService.LogDebug("IsSubmitting: {0}", vm.IsSubmitting);
            DebugService.LogDebug("ValidationError: '{0}'", vm.ValidationError);
            DebugService.LogDebug("HasValidationError: {0}", vm.HasValidationError);
            DebugService.LogDebug("CanExecute AddItemCommand: {0}", vm.AddItemCommand.CanExecute(null));
            DebugService.LogDebug("=== END DEBUG ===");
        }
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
