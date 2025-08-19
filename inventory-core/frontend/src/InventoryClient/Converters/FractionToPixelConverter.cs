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
            if (values[0] is double fraction &&
                values[1] is double totalWidth)
            {
                return Math.Max(0, fraction * totalWidth);
            }
            return 0;
        }
    }

    // Converts a fraction (0–1) * total width into a Margin.Left
    public class FractionToMarginConverter : IMultiValueConverter
    {
        public object Convert(IList<object?> values, Type targetType, object? parameter, CultureInfo culture)
        {
            if (values[0] is double fraction &&
                values[1] is double totalWidth)
            {
                var left = Math.Max(0, fraction * totalWidth);
                return new Avalonia.Thickness(left, 0, 0, 0);
            }
            return new Avalonia.Thickness(0);
        }
    }
}

