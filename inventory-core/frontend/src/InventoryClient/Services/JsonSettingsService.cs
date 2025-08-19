using System.ComponentModel;
using System.Text.Json;

namespace InventoryClient.Services;

/// <summary>
/// JSON-based settings service that stores settings in a local file
/// </summary>
public class JsonSettingsService : ISettingsService
{
    private readonly string _settingsFilePath;
    private readonly Dictionary<string, object> _settings = new();
    private readonly object _lock = new();
    private bool _loaded = false;

    public event PropertyChangedEventHandler? PropertyChanged;

    public JsonSettingsService(string? settingsDirectory = null)
    {
        var directory = settingsDirectory ?? Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData),
            "InventoryClient");

        Directory.CreateDirectory(directory);
        _settingsFilePath = Path.Combine(directory, "settings.json");

        DebugService.LogDebug("Settings service initialized with path: {0}", _settingsFilePath);

        // Try to load settings synchronously during construction
        TryLoadSynchronously();
    }

    private void TryLoadSynchronously()
    {
        try
        {
            if (File.Exists(_settingsFilePath))
            {
                var json = File.ReadAllText(_settingsFilePath);
                var loadedSettings = JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(json);

                if (loadedSettings != null)
                {
                    lock (_lock)
                    {
                        _settings.Clear();
                        foreach (var kvp in loadedSettings)
                        {
                            _settings[kvp.Key] = kvp.Value;
                        }
                    }

                    DebugService.LogDebug("Settings loaded synchronously from: {0} ({1} settings)", _settingsFilePath, loadedSettings.Count);
                    _loaded = true;
                }
            }
            else
            {
                DebugService.LogDebug("Settings file does not exist, starting with empty settings");
                _loaded = true;
            }
        }
        catch (Exception ex)
        {
            DebugService.LogError("Failed to load settings synchronously, will try async later", ex);
            // Settings will be loaded async when needed
        }
    }

    private void EnsureLoaded()
    {
        if (!_loaded)
        {
            TryLoadSynchronously();
        }
    }

    public T? GetSetting<T>(string key, T? defaultValue = default)
    {
        lock (_lock)
        {
            EnsureLoaded();

            if (!_settings.TryGetValue(key, out var value))
            {
                return defaultValue;
            }

            try
            {
                if (value is JsonElement jsonElement)
                {
                    // Handle JsonElement deserialization
                    return JsonSerializer.Deserialize<T>(jsonElement.GetRawText());
                }

                if (value is T directValue)
                {
                    return directValue;
                }

                // Try to convert the value
                return (T?)Convert.ChangeType(value, typeof(T));
            }
            catch (Exception ex)
            {
                DebugService.LogError("Failed to convert setting {0} to type {1}", ex, key, typeof(T).Name);
                return defaultValue;
            }
        }
    }

    public void SetSetting<T>(string key, T value)
    {
        lock (_lock)
        {
            EnsureLoaded();

            var oldValue = _settings.TryGetValue(key, out var existing) ? existing : null;
            _settings[key] = value!;

            if (!Equals(oldValue, value))
            {
                PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(key));
                DebugService.LogDebug("Setting '{0}' updated to: {1}", key, value?.ToString() ?? "null");
            }
        }
    }

    public bool HasSetting(string key)
    {
        lock (_lock)
        {
            EnsureLoaded();
            return _settings.ContainsKey(key);
        }
    }

    public void RemoveSetting(string key)
    {
        lock (_lock)
        {
            EnsureLoaded();
            if (_settings.Remove(key))
            {
                PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(key));
                DebugService.LogDebug("Setting '{0}' removed", key);
            }
        }
    }

    public async Task SaveAsync()
    {
        try
        {
            Dictionary<string, object> settingsCopy;
            lock (_lock)
            {
                settingsCopy = new Dictionary<string, object>(_settings);
            }

            var json = JsonSerializer.Serialize(settingsCopy, new JsonSerializerOptions
            {
                WriteIndented = true,
                PropertyNamingPolicy = JsonNamingPolicy.CamelCase
            });

            await File.WriteAllTextAsync(_settingsFilePath, json);
            DebugService.LogDebug("Settings saved to: {0}", _settingsFilePath);
        }
        catch (Exception ex)
        {
            DebugService.LogError("Failed to save settings", ex);
            throw;
        }
    }

    public async Task LoadAsync()
    {
        try
        {
            if (!File.Exists(_settingsFilePath))
            {
                DebugService.LogDebug("Settings file does not exist, starting with empty settings");
                return;
            }

            var json = await File.ReadAllTextAsync(_settingsFilePath);
            var loadedSettings = JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(json);

            if (loadedSettings != null)
            {
                lock (_lock)
                {
                    _settings.Clear();
                    foreach (var kvp in loadedSettings)
                    {
                        _settings[kvp.Key] = kvp.Value;
                    }
                }

                DebugService.LogDebug("Settings loaded from: {0} ({1} settings)", _settingsFilePath, loadedSettings.Count);
            }
        }
        catch (Exception ex)
        {
            DebugService.LogError("Failed to load settings", ex);
            // Don't rethrow - we can continue with default settings
        }
    }

    public void Clear()
    {
        lock (_lock)
        {
            _settings.Clear();
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(string.Empty));
        }

        DebugService.LogDebug("All settings cleared");
    }

    public IEnumerable<string> GetAllKeys()
    {
        lock (_lock)
        {
            return _settings.Keys.ToList();
        }
    }

    public Dictionary<string, object> ExportSettings()
    {
        lock (_lock)
        {
            return new Dictionary<string, object>(_settings);
        }
    }

    public void ImportSettings(Dictionary<string, object> settings)
    {
        lock (_lock)
        {
            foreach (var kvp in settings)
            {
                _settings[kvp.Key] = kvp.Value;
            }
        }

        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(string.Empty));
        DebugService.LogDebug("Imported {0} settings", settings.Count);
    }
}
