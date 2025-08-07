using System.Collections.Generic;
using System.Globalization;
using Avalonia.Data;
using Avalonia.Data.Converters;

namespace InventoryClient.Converters;

public class DictionaryToKeyValuePairConverter : IValueConverter
{
    public object? Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        if (value is Dictionary<string, double> dictionary)
        {
            return dictionary.Select(kvp => new KeyValuePair<string, double>(kvp.Key, kvp.Value)).ToList();
        }
        return new List<KeyValuePair<string, double>>();
    }

    public object? ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        return BindingOperations.DoNothing;
    }
}
