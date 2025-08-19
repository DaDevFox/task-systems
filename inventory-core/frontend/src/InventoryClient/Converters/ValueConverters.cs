using System;
using System.Globalization;
using Avalonia.Data.Converters;

namespace InventoryClient.Converters
{
    /// <summary>
    /// Converter that returns true if the value is greater than zero
    /// </summary>
    public class IsPositiveConverter : IValueConverter
    {
        public static readonly IsPositiveConverter Instance = new();

        public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
        {
            if (value is double doubleValue)
            {
                return !double.IsNaN(doubleValue) && !double.IsInfinity(doubleValue) && doubleValue > 0;
            }
            if (value is int intValue)
            {
                return intValue > 0;
            }
            if (value is decimal decimalValue)
            {
                return decimalValue > 0;
            }
            return false;
        }

        public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }

    /// <summary>
    /// Converter that returns true if the value is not zero (positive or negative)
    /// </summary>
    public class IsNonZeroConverter : IValueConverter
    {
        public static readonly IsNonZeroConverter Instance = new();

        public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
        {
            if (value is double doubleValue)
            {
                return !double.IsNaN(doubleValue) && !double.IsInfinity(doubleValue) && Math.Abs(doubleValue) > 0.001; // Small epsilon for floating point comparison
            }
            if (value is int intValue)
            {
                return intValue != 0;
            }
            if (value is decimal decimalValue)
            {
                return decimalValue != 0;
            }
            return false;
        }

        public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }
}
