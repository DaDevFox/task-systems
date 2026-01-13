using System.ComponentModel;

namespace InventoryClient.Services;

/// <summary>
/// Interface for application settings service
/// </summary>
public interface ISettingsService : INotifyPropertyChanged
{
    /// <summary>
    /// Gets a setting value
    /// </summary>
    T? GetSetting<T>(string key, T? defaultValue = default);

    /// <summary>
    /// Sets a setting value
    /// </summary>
    void SetSetting<T>(string key, T value);

    /// <summary>
    /// Checks if a setting exists
    /// </summary>
    bool HasSetting(string key);

    /// <summary>
    /// Removes a setting
    /// </summary>
    void RemoveSetting(string key);

    /// <summary>
    /// Saves settings to persistent storage
    /// </summary>
    Task SaveAsync();

    /// <summary>
    /// Loads settings from persistent storage
    /// </summary>
    Task LoadAsync();

    /// <summary>
    /// Clears all settings
    /// </summary>
    void Clear();

    /// <summary>
    /// Gets all setting keys
    /// </summary>
    IEnumerable<string> GetAllKeys();

    /// <summary>
    /// Export settings to a dictionary
    /// </summary>
    Dictionary<string, object> ExportSettings();

    /// <summary>
    /// Import settings from a dictionary
    /// </summary>
    void ImportSettings(Dictionary<string, object> settings);
}
