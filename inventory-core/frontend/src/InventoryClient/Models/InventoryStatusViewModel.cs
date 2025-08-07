using System.ComponentModel;
using System.Runtime.CompilerServices;

namespace InventoryClient.Models;

/// <summary>
/// Client-side representation of inventory status overview
/// </summary>
public class InventoryStatusViewModel : INotifyPropertyChanged
{
    private IList<InventoryItemViewModel> _items = new List<InventoryItemViewModel>();
    private IList<InventoryItemViewModel> _lowStockItems = new List<InventoryItemViewModel>();
    private IList<InventoryItemViewModel> _emptyItems = new List<InventoryItemViewModel>();
    private int _totalItems;
    private DateTime _lastUpdated;

    public IList<InventoryItemViewModel> Items
    {
        get => _items;
        set => SetProperty(ref _items, value);
    }

    public IList<InventoryItemViewModel> LowStockItems
    {
        get => _lowStockItems;
        set => SetProperty(ref _lowStockItems, value);
    }

    public IList<InventoryItemViewModel> EmptyItems
    {
        get => _emptyItems;
        set => SetProperty(ref _emptyItems, value);
    }

    public int TotalItems
    {
        get => _totalItems;
        set => SetProperty(ref _totalItems, value);
    }

    public DateTime LastUpdated
    {
        get => _lastUpdated;
        set => SetProperty(ref _lastUpdated, value);
    }

    public int LowStockCount => LowStockItems.Count;
    public int EmptyCount => EmptyItems.Count;
    public int NormalStockCount => TotalItems - LowStockCount - EmptyCount;

    public string StatusSummary => $"{TotalItems} items total - {NormalStockCount} normal, {LowStockCount} low stock, {EmptyCount} empty";

    public event PropertyChangedEventHandler? PropertyChanged;

    protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
    {
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
    }

    protected bool SetProperty<T>(ref T field, T value, [CallerMemberName] string? propertyName = null)
    {
        if (EqualityComparer<T>.Default.Equals(field, value)) return false;
        field = value;
        OnPropertyChanged(propertyName);
        
        // Update derived properties when collections change
        if (propertyName == nameof(LowStockItems) || propertyName == nameof(EmptyItems) || propertyName == nameof(TotalItems))
        {
            OnPropertyChanged(nameof(LowStockCount));
            OnPropertyChanged(nameof(EmptyCount));
            OnPropertyChanged(nameof(NormalStockCount));
            OnPropertyChanged(nameof(StatusSummary));
        }
        
        return true;
    }
}
