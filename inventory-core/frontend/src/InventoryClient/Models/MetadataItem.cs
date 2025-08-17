using CommunityToolkit.Mvvm.ComponentModel;

namespace InventoryClient.Models;

/// <summary>
/// Represents a key-value metadata item that can be added to inventory items
/// </summary>
public partial class MetadataItem : ObservableObject
{
    [ObservableProperty]
    private string _key = string.Empty;

    [ObservableProperty]
    private string _value = string.Empty;

    [ObservableProperty]
    private bool _isDefault = false;

    public MetadataItem()
    {
    }

    public MetadataItem(string key, string value = "", bool isDefault = false)
    {
        Key = key;
        Value = value;
        IsDefault = isDefault;
    }

    public bool IsValid => !string.IsNullOrWhiteSpace(Key);
}
