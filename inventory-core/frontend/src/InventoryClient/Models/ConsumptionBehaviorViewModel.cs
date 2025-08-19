// using System.ComponentModel;
// using System.Runtime.CompilerServices;
//
// namespace InventoryClient.Models;
//
// /// <summary>
// /// Client-side representation of consumption behavior
// /// </summary>
// public class ConsumptionBehaviorViewModel : INotifyPropertyChanged
// {
//     private ConsumptionPattern _pattern;
//     private double _averageRatePerDay;
//     private double _variance;
//     private IList<double> _seasonalFactors = new List<double>();
//     private DateTime _lastUpdated;
//
//     public ConsumptionPattern Pattern
//     {
//         get => _pattern;
//         set => SetProperty(ref _pattern, value);
//     }
//
//     public double AverageRatePerDay
//     {
//         get => _averageRatePerDay;
//         set => SetProperty(ref _averageRatePerDay, value);
//     }
//
//     public double Variance
//     {
//         get => _variance;
//         set => SetProperty(ref _variance, value);
//     }
//
//     public IList<double> SeasonalFactors
//     {
//         get => _seasonalFactors;
//         set => SetProperty(ref _seasonalFactors, value);
//     }
//
//     public DateTime LastUpdated
//     {
//         get => _lastUpdated;
//         set => SetProperty(ref _lastUpdated, value);
//     }
//
//     public string PatternDescription
//     {
//         get
//         {
//             return Pattern switch
//             {
//                 ConsumptionPattern.Linear => "Steady, consistent usage",
//                 ConsumptionPattern.Seasonal => "Varies by season/time of year",
//                 ConsumptionPattern.Batch => "Used in large amounts at once",
//                 ConsumptionPattern.Random => "Unpredictable usage",
//                 _ => "Unspecified pattern"
//             };
//         }
//     }
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
//
// /// <summary>
// /// Enumeration for consumption patterns
// /// </summary>
// public enum ConsumptionPattern
// {
//     Unspecified = 0,
//     Linear = 1,
//     Seasonal = 2,
//     Batch = 3,
//     Random = 4
// }
