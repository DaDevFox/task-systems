using Avalonia.Data.Converters;
using System.Globalization;

namespace TaskSystems.Shared.Converters;

/// <summary>
/// Converts numeric values to percentage strings with optional decimal places
/// </summary>
public class PercentageConverter : IValueConverter
{
    public static readonly PercentageConverter Instance = new();

    public object? Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        if (value is double d)
        {
            var decimalPlaces = parameter is string s && int.TryParse(s, out var places) ? places : 1;
            return d.ToString($"F{decimalPlaces}", culture) + "%";
        }
        return value?.ToString() ?? "0%";
    }

    public object? ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        throw new NotImplementedException();
    }
}
