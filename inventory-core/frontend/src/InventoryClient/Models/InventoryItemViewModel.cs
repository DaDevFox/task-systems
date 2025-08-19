using Avalonia.Media;
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
    private double _currentProposedLevel;
    private double _currentLevel;
    private double _maxCapacity;
    private double _lowStockThreshold;
    private string _unitId = string.Empty;
    private DateTime _lastUpdated;
    private double _predictedDaysRemaining;
    private double _confidenceScore;
    private DateTime _predictedEmptyDate;
    // private PredictionTrainingStatusViewModel? _trainingStatus;
    // private ConsumptionBehaviorViewModel? _consumptionBehavior;
    private IList<string> _alternateUnitIds = new List<string>();
    private Dictionary<string, string> _metadata = new();

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

    public double ProposedLevel
    {
        get => _currentProposedLevel;
        set => SetProperty(ref _currentProposedLevel, value);
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

    public double CurrentFraction => CurrentLevel / MaxCapacity;
    public double DeltaFraction => (ProposedLevel - CurrentLevel) / MaxCapacity;
    public bool IsIncrease => ProposedLevel >= CurrentLevel;

    public Brush DeltaBrush => (Brush)(ProposedLevel >= CurrentLevel 
      ? Brushes.Green 
      : Brushes.Red);

    // TODO: #42 - enable when adaptive prediction models + training
    //
    // public PredictionTrainingStatusViewModel? TrainingStatus
    // {
    //     get => _trainingStatus;
    //     set => SetProperty(ref _trainingStatus, value);
    // }
    //
    // public ConsumptionBehaviorViewModel? ConsumptionBehavior
    // {
    //     get => _consumptionBehavior;
    //     set => SetProperty(ref _consumptionBehavior, value);
    // }

    // PERF: minimize memory usage over CPU runtime -- deref into conversion table 
    public IList<string> AlternateUnitIds
    {
        get => _alternateUnitIds;
        set => SetProperty(ref _alternateUnitIds, value);
    }

    public Dictionary<string, string> Metadata
    {
        get => _metadata;
        set => SetProperty(ref _metadata, value);
    }

    // public bool HasPredictionModel => TrainingStatus?.ActiveModel != PredictionModel.Unspecified;
    //
    // public string ActivePredictionModel => TrainingStatus?.ModelDescription ?? "No model active";
    //
    // public bool IsTrainingComplete => TrainingStatus?.IsTrainingComplete ?? false;

    public double CurrentLevelPercentage => MaxCapacity > 0 ? (CurrentLevel / MaxCapacity) * 100 : 0;

    public bool IsLowStock => CurrentLevel > 0 && CurrentLevel <= LowStockThreshold;

    public bool IsEmpty => CurrentLevel <= 0;

    public bool WouldBeOverCapacity => ProposedLevel > MaxCapacity;

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

    public string CurrentLevelDisplay => $"{CurrentLevel:F2} {UnitId}";

    public string MaxCapacityDisplay => $"Max: {MaxCapacity:F2} {UnitId}";

    public bool HasPrediction => PredictedDaysRemaining > 0;

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
