// using System.ComponentModel;
// using System.Runtime.CompilerServices;
//
// namespace InventoryClient.Models;
//
// /// <summary>
// /// Client-side representation of consumption prediction with enhanced model information
// /// </summary>
// public class ConsumptionPredictionViewModel : INotifyPropertyChanged
// {
//     private string _itemId = string.Empty;
//     private double _predictedDaysRemaining;
//     private double _confidenceScore;
//     private DateTime _predictedEmptyDate;
//     private double _recommendedRestockLevel;
//     private string _predictionModel = string.Empty;
//     private double _estimate;
//     private double _lowerBound;
//     private double _upperBound;
//     private string _recommendation = string.Empty;
//
//     public string ItemId
//     {
//         get => _itemId;
//         set => SetProperty(ref _itemId, value);
//     }
//
//     public double PredictedDaysRemaining
//     {
//         get => _predictedDaysRemaining;
//         set => SetProperty(ref _predictedDaysRemaining, value);
//     }
//
//     public double ConfidenceScore
//     {
//         get => _confidenceScore;
//         set => SetProperty(ref _confidenceScore, value);
//     }
//
//     public DateTime PredictedEmptyDate
//     {
//         get => _predictedEmptyDate;
//         set => SetProperty(ref _predictedEmptyDate, value);
//     }
//
//     public double RecommendedRestockLevel
//     {
//         get => _recommendedRestockLevel;
//         set => SetProperty(ref _recommendedRestockLevel, value);
//     }
//
//     public string PredictionModel
//     {
//         get => _predictionModel;
//         set => SetProperty(ref _predictionModel, value);
//     }
//
//     public double Estimate
//     {
//         get => _estimate;
//         set => SetProperty(ref _estimate, value);
//     }
//
//     public double LowerBound
//     {
//         get => _lowerBound;
//         set => SetProperty(ref _lowerBound, value);
//     }
//
//     public double UpperBound
//     {
//         get => _upperBound;
//         set => SetProperty(ref _upperBound, value);
//     }
//
//     public string Recommendation
//     {
//         get => _recommendation;
//         set => SetProperty(ref _recommendation, value);
//     }
//
//     public string ConfidenceDescription => $"{ConfidenceScore * 100:F1}% confidence";
//
//     public string PredictionSummary => $"Estimated {PredictedDaysRemaining:F1} days remaining using {PredictionModel}";
//
//     public bool HasConfidenceInterval => Math.Abs(LowerBound - UpperBound) > 0.001;
//
//     public event PropertyChangedEventHandler? PropertyChanged;
//
//     protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
//     {
//         PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
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
