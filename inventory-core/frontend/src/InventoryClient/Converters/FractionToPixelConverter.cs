using System;
using System.Globalization;
using Avalonia.Data.Converters;

namespace InventoryClient.Converters
{
    // Converts a fraction (0–1) * total width into an actual Width
    public class FractionToWidthConverter : IMultiValueConverter
    {
        public object Convert(IList<object?> values, Type targetType, object? parameter, CultureInfo culture)
        {
            try
            {
                if (values?.Count >= 2 && 
                    values[0] is double fraction &&
                    values[1] is double totalWidth)
                {
                    // Ensure fraction is in valid range and totalWidth is positive
                    if (double.IsNaN(fraction) || double.IsInfinity(fraction)) return 0.0;
                    if (double.IsNaN(totalWidth) || double.IsInfinity(totalWidth) || totalWidth <= 0) return 0.0;
                    
                    // Clamp fraction to valid range
                    fraction = Math.Max(0, Math.Min(1, Math.Abs(fraction)));
                    return fraction * totalWidth;
                }
            }
            catch (Exception)
            {
                // Return safe default on any conversion error
            }
            return 0.0;
        }
    }

    // Converts a fraction (0–1) * total width into a Margin.Left
    public class FractionToMarginConverter : IMultiValueConverter
    {
        public object Convert(IList<object?> values, Type targetType, object? parameter, CultureInfo culture)
        {
            try
            {
                if (values?.Count >= 2 && 
                    values[0] is double fraction &&
                    values[1] is double totalWidth)
                {
                    // Ensure fraction is in valid range and totalWidth is positive
                    if (double.IsNaN(fraction) || double.IsInfinity(fraction)) return new Avalonia.Thickness(0);
                    if (double.IsNaN(totalWidth) || double.IsInfinity(totalWidth) || totalWidth <= 0) return new Avalonia.Thickness(0);
                    
                    // Clamp fraction to valid range
                    fraction = Math.Max(0, Math.Min(1, Math.Abs(fraction)));
                    var left = fraction * totalWidth;
                    return new Avalonia.Thickness(left, 0, 0, 0);
                }
            }
            catch (Exception)
            {
                // Return safe default on any conversion error
            }
            return new Avalonia.Thickness(0);
        }
    }
}

