using System.ComponentModel;
using System.Runtime.CompilerServices;

namespace InventoryClient.Models;

/// <summary>
/// Client-side representation of prediction model fitness tracking
/// </summary>
public class PredictionModelFitnessViewModel : INotifyPropertyChanged
{
    private string _itemId = string.Empty;
    private PredictionModel _model;
    private double _currentFitness;
    private int _predictionCount;
    private double _averageError;
    private double _errorVariance;
    private DateTime _lastUpdated;
    private IList<FitnessDataPointViewModel> _fitnessHistory = new List<FitnessDataPointViewModel>();

    public string ItemId
    {
        get => _itemId;
        set => SetProperty(ref _itemId, value);
    }

    public PredictionModel Model
    {
        get => _model;
        set => SetProperty(ref _model, value);
    }

    public double CurrentFitness
    {
        get => _currentFitness;
        set => SetProperty(ref _currentFitness, value);
    }

    public int PredictionCount
    {
        get => _predictionCount;
        set => SetProperty(ref _predictionCount, value);
    }

    public double AverageError
    {
        get => _averageError;
        set => SetProperty(ref _averageError, value);
    }

    public double ErrorVariance
    {
        get => _errorVariance;
        set => SetProperty(ref _errorVariance, value);
    }

    public DateTime LastUpdated
    {
        get => _lastUpdated;
        set => SetProperty(ref _lastUpdated, value);
    }

    public IList<FitnessDataPointViewModel> FitnessHistory
    {
        get => _fitnessHistory;
        set => SetProperty(ref _fitnessHistory, value);
    }

    public string ModelDescription
    {
        get
        {
            return Model switch
            {
                PredictionModel.Markov => "Markov Chain Model",
                PredictionModel.Croston => "Croston's Method",
                PredictionModel.DriftImpulse => "Drift & Impulse Model",
                PredictionModel.Bayesian => "Bayesian Inference",
                PredictionModel.MemoryWindow => "Memory Window Model",
                PredictionModel.EventTrigger => "Event Trigger Model",
                _ => "Unknown Model"
            };
        }
    }

    public string FitnessDescription => $"{CurrentFitness * 100:F1}% fitness";

    public string PerformanceSummary => $"{PredictionCount} predictions, {AverageError:F2} avg error";

    public event PropertyChangedEventHandler? PropertyChanged;

    protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
    {
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));

        if (propertyName == nameof(Model))
        {
            OnPropertyChanged(nameof(ModelDescription));
        }
        else if (propertyName == nameof(CurrentFitness))
        {
            OnPropertyChanged(nameof(FitnessDescription));
        }
        else if (propertyName == nameof(PredictionCount) || propertyName == nameof(AverageError))
        {
            OnPropertyChanged(nameof(PerformanceSummary));
        }
    }

    protected bool SetProperty<T>(ref T field, T value, [CallerMemberName] string? propertyName = null)
    {
        if (EqualityComparer<T>.Default.Equals(field, value)) return false;
        field = value;
        OnPropertyChanged(propertyName);
        return true;
    }
}

/// <summary>
/// Represents a single fitness measurement data point
/// </summary>
public class FitnessDataPointViewModel : INotifyPropertyChanged
{
    private DateTime _timestamp;
    private double _actualValue;
    private double _predictedValue;
    private double _error;
    private double _fitnessScore;

    public DateTime Timestamp
    {
        get => _timestamp;
        set => SetProperty(ref _timestamp, value);
    }

    public double ActualValue
    {
        get => _actualValue;
        set => SetProperty(ref _actualValue, value);
    }

    public double PredictedValue
    {
        get => _predictedValue;
        set => SetProperty(ref _predictedValue, value);
    }

    public double Error
    {
        get => _error;
        set => SetProperty(ref _error, value);
    }

    public double FitnessScore
    {
        get => _fitnessScore;
        set => SetProperty(ref _fitnessScore, value);
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
