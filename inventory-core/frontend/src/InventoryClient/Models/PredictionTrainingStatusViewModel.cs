// using System.ComponentModel;
// using System.Runtime.CompilerServices;
//
// namespace InventoryClient.Models;
//
// /// <summary>
// /// Training stage enumeration for prediction models
// /// </summary>
// public enum TrainingStage
// {
//     Unspecified = 0,
//     Collecting = 1,       // Actively collecting training data
//     Learning = 2,         // Processing collected data
//     Trained = 3,          // Model is trained and active
//     Retraining = 4        // Model being retrained with new data
// }
//
// /// <summary>
// /// Available prediction model types
// /// </summary>
// public enum PredictionModel
// {
//     Unspecified = 0,
//     Markov = 1,           // Finite state consumption model
//     Croston = 2,          // Intermittent demand forecasting
//     DriftImpulse = 3,     // Physical system with drift and impulses
//     Bayesian = 4,         // Bayesian inference with confidence intervals
//     MemoryWindow = 5,     // Memory-augmented rolling windows
//     EventTrigger = 6      // Temporal event trigger modeling
// }
//
// /// <summary>
// /// Client-side representation of prediction training status
// /// </summary>
// public class PredictionTrainingStatusViewModel : INotifyPropertyChanged
// {
//     private string _itemId = string.Empty;
//     private TrainingStage _stage;
//     private PredictionModel _activeModel;
//     private IList<PredictionModel> _availableModels = new List<PredictionModel>();
//     private int _trainingSamples;
//     private int _minSamplesRequired;
//     private double _trainingAccuracy;
//     private DateTime _trainingStarted;
//     private DateTime _lastUpdated;
//     private Dictionary<string, double> _modelParameters = new();
//
//     public string ItemId
//     {
//         get => _itemId;
//         set => SetProperty(ref _itemId, value);
//     }
//
//     public TrainingStage Stage
//     {
//         get => _stage;
//         set => SetProperty(ref _stage, value);
//     }
//
//     public PredictionModel ActiveModel
//     {
//         get => _activeModel;
//         set => SetProperty(ref _activeModel, value);
//     }
//
//     public IList<PredictionModel> AvailableModels
//     {
//         get => _availableModels;
//         set => SetProperty(ref _availableModels, value);
//     }
//
//     public int TrainingSamples
//     {
//         get => _trainingSamples;
//         set => SetProperty(ref _trainingSamples, value);
//     }
//
//     public int MinSamplesRequired
//     {
//         get => _minSamplesRequired;
//         set => SetProperty(ref _minSamplesRequired, value);
//     }
//
//     public double TrainingAccuracy
//     {
//         get => _trainingAccuracy;
//         set => SetProperty(ref _trainingAccuracy, value);
//     }
//
//     public DateTime TrainingStarted
//     {
//         get => _trainingStarted;
//         set => SetProperty(ref _trainingStarted, value);
//     }
//
//     public DateTime LastUpdated
//     {
//         get => _lastUpdated;
//         set => SetProperty(ref _lastUpdated, value);
//     }
//
//     public Dictionary<string, double> ModelParameters
//     {
//         get => _modelParameters;
//         set => SetProperty(ref _modelParameters, value);
//     }
//
//     public string StageDescription
//     {
//         get
//         {
//             return Stage switch
//             {
//                 TrainingStage.Collecting => "Collecting training data",
//                 TrainingStage.Learning => "Processing collected data",
//                 TrainingStage.Trained => "Model is trained and active",
//                 TrainingStage.Retraining => "Model being retrained with new data",
//                 _ => "Training status unknown"
//             };
//         }
//     }
//
//     public string ModelDescription
//     {
//         get
//         {
//             return ActiveModel switch
//             {
//                 PredictionModel.Markov => "Markov Chain Model",
//                 PredictionModel.Croston => "Croston's Method",
//                 PredictionModel.DriftImpulse => "Drift & Impulse Model",
//                 PredictionModel.Bayesian => "Bayesian Inference",
//                 PredictionModel.MemoryWindow => "Memory Window Model",
//                 PredictionModel.EventTrigger => "Event Trigger Model",
//                 _ => "No model selected"
//             };
//         }
//     }
//
//     public double TrainingProgress => MinSamplesRequired > 0 ? Math.Min(1.0, (double)TrainingSamples / MinSamplesRequired) : 0;
//
//     public bool IsTrainingComplete => Stage == TrainingStage.Trained;
//
//     public bool CanStartTraining => TrainingSamples >= MinSamplesRequired && Stage != TrainingStage.Learning;
//
//     public string TrainingProgressText => $"{TrainingSamples}/{MinSamplesRequired} samples ({TrainingProgress * 100:F1}%)";
//
//     public string AccuracyDisplayText => TrainingAccuracy > 0 ? $"{TrainingAccuracy * 100:F1}%" : "Not available";
//
//     public event PropertyChangedEventHandler? PropertyChanged;
//
//     protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
//     {
//         PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
//
//         // Update derived properties when base properties change
//         if (propertyName == nameof(Stage))
//         {
//             OnPropertyChanged(nameof(StageDescription));
//             OnPropertyChanged(nameof(IsTrainingComplete));
//             OnPropertyChanged(nameof(CanStartTraining));
//         }
//         else if (propertyName == nameof(ActiveModel))
//         {
//             OnPropertyChanged(nameof(ModelDescription));
//         }
//         else if (propertyName == nameof(TrainingSamples) || propertyName == nameof(MinSamplesRequired))
//         {
//             OnPropertyChanged(nameof(TrainingProgress));
//             OnPropertyChanged(nameof(CanStartTraining));
//             OnPropertyChanged(nameof(TrainingProgressText));
//         }
//     }
//
//     protected bool SetProperty<T>(ref T field, T value, [CallerMemberName] string? propertyName = null)
//     {
//         if (EqualityComparer<T>.Default.Equals(field, value)) return false;
//         field = value;
//         OnPropertyChanged(propertyName);
//         return true;
//     }
// }
