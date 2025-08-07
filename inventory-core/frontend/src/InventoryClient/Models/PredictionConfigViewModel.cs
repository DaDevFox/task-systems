using System.ComponentModel;
using System.Runtime.CompilerServices;

namespace InventoryClient.Models;

/// <summary>
/// Client-side representation of prediction configuration for an item
/// </summary>
public class PredictionConfigViewModel : INotifyPropertyChanged
{
    private string _itemId = string.Empty;
    private IList<PredictionModel> _enabledModels = new List<PredictionModel>();
    private PredictionModel _preferredModel;
    private bool _autoSelectBest;
    private Dictionary<string, string> _globalSettings = new();

    public string ItemId
    {
        get => _itemId;
        set => SetProperty(ref _itemId, value);
    }

    public IList<PredictionModel> EnabledModels
    {
        get => _enabledModels;
        set => SetProperty(ref _enabledModels, value);
    }

    public PredictionModel PreferredModel
    {
        get => _preferredModel;
        set => SetProperty(ref _preferredModel, value);
    }

    public bool AutoSelectBest
    {
        get => _autoSelectBest;
        set => SetProperty(ref _autoSelectBest, value);
    }

    public Dictionary<string, string> GlobalSettings
    {
        get => _globalSettings;
        set => SetProperty(ref _globalSettings, value);
    }

    public string PreferredModelDescription
    {
        get
        {
            return PreferredModel switch
            {
                PredictionModel.Markov => "Markov Chain Model",
                PredictionModel.Croston => "Croston's Method",
                PredictionModel.DriftImpulse => "Drift & Impulse Model",
                PredictionModel.Bayesian => "Bayesian Inference",
                PredictionModel.MemoryWindow => "Memory Window Model",
                PredictionModel.EventTrigger => "Event Trigger Model",
                _ => "No preferred model"
            };
        }
    }

    public string EnabledModelsText => string.Join(", ", EnabledModels.Select(GetModelName));

    public int EnabledModelCount => EnabledModels.Count;

    private static string GetModelName(PredictionModel model)
    {
        return model switch
        {
            PredictionModel.Markov => "Markov",
            PredictionModel.Croston => "Croston",
            PredictionModel.DriftImpulse => "Drift-Impulse",
            PredictionModel.Bayesian => "Bayesian",
            PredictionModel.MemoryWindow => "Memory-Window",
            PredictionModel.EventTrigger => "Event-Trigger",
            _ => "Unknown"
        };
    }

    public event PropertyChangedEventHandler? PropertyChanged;

    protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
    {
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));

        if (propertyName == nameof(PreferredModel))
        {
            OnPropertyChanged(nameof(PreferredModelDescription));
        }
        else if (propertyName == nameof(EnabledModels))
        {
            OnPropertyChanged(nameof(EnabledModelsText));
            OnPropertyChanged(nameof(EnabledModelCount));
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
