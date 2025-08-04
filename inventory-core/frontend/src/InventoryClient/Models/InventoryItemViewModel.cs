using System.ComponentModel;
using System.Runtime.CompilerServices;

namespace InventoryClient.Models;

/// <summary>
/// Client-side representation of an inventory item with additional UI properties
/// </summary>
public class InventoryItemViewModel : INotifyPropertyChanged
{
    private string _id = string.Empty;
    private string _name = string.Empty;
    private string _description = string.Empty;
    private double _currentLevel;
    private double _maxCapacity;
    private double _lowStockThreshold;
    private string _unitId = string.Empty;
    private DateTime _lastUpdated;
    private double _predictedDaysRemaining;
    private double _confidenceScore;
    private DateTime _predictedEmptyDate;

    public string Id
    {
        get => _id;
        set => SetProperty(ref _id, value);
    }

    public string Name
    {
        get => _name;
        set => SetProperty(ref _name, value);
    }

    public string Description
    {
        get => _description;
        set => SetProperty(ref _description, value);
    }

    public double CurrentLevel
    {
        get => _currentLevel;
        set
        {
            if (SetProperty(ref _currentLevel, value))
            {
                OnPropertyChanged(nameof(CurrentLevelPercentage));
                OnPropertyChanged(nameof(IsLowStock));
                OnPropertyChanged(nameof(IsEmpty));
                OnPropertyChanged(nameof(StockStatus));
                OnPropertyChanged(nameof(StockStatusDescription));
            }
        }
    }

    public double MaxCapacity
    {
        get => _maxCapacity;
        set
        {
            if (SetProperty(ref _maxCapacity, value))
            {
                OnPropertyChanged(nameof(CurrentLevelPercentage));
            }
        }
    }

    public double LowStockThreshold
    {
        get => _lowStockThreshold;
        set
        {
            if (SetProperty(ref _lowStockThreshold, value))
            {
                OnPropertyChanged(nameof(IsLowStock));
                OnPropertyChanged(nameof(StockStatus));
                OnPropertyChanged(nameof(StockStatusDescription));
            }
        }
    }

    public string UnitId
    {
        get => _unitId;
        set => SetProperty(ref _unitId, value);
    }

    public DateTime LastUpdated
    {
        get => _lastUpdated;
        set => SetProperty(ref _lastUpdated, value);
    }

    public double PredictedDaysRemaining
    {
        get => _predictedDaysRemaining;
        set => SetProperty(ref _predictedDaysRemaining, value);
    }

    public double ConfidenceScore
    {
        get => _confidenceScore;
        set => SetProperty(ref _confidenceScore, value);
    }

    public DateTime PredictedEmptyDate
    {
        get => _predictedEmptyDate;
        set => SetProperty(ref _predictedEmptyDate, value);
    }

    public double CurrentLevelPercentage => MaxCapacity > 0 ? (CurrentLevel / MaxCapacity) * 100 : 0;

    public bool IsLowStock => CurrentLevel > 0 && CurrentLevel <= LowStockThreshold;

    public bool IsEmpty => CurrentLevel <= 0;

    public string StockStatus
    {
        get
        {
            if (IsEmpty) return "Empty";
            if (IsLowStock) return "Low";
            return "Normal";
        }
    }

    public string StockStatusDescription
    {
        get
        {
            if (IsEmpty) return "Out of stock";
            if (IsLowStock) return $"Low stock - {CurrentLevel:F1} {UnitId} remaining";
            return $"{CurrentLevel:F1} {UnitId} available";
        }
    }

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
        return true;
    }
}
