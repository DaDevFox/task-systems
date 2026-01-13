using Avalonia.Data.Converters;
using Avalonia.Media;
using System.Globalization;

namespace TaskSystems.Shared.Controls;

public class StatusColorConverter : IValueConverter
{
    public static readonly StatusColorConverter Instance = new();

    public object? Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        return value?.ToString()?.ToLowerInvariant() switch
        {
            "normal" or "good" or "ok" => Brushes.Green,
            "warning" or "low" => Brushes.Orange,
            "critical" or "empty" or "error" => Brushes.Red,
            "unknown" or null => Brushes.Gray,
            _ => Brushes.LightBlue
        };
    }

    public object? ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        throw new NotImplementedException();
    }
}
