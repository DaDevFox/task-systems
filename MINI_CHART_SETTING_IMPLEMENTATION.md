# Mini Chart History Days Setting Implementation

## Summary
Added a user-configurable setting to control the number of days of historical data shown in mini charts. The setting allows users to customize the time range from the default 14 days to any value they prefer.

## Changes Made

### 1. InventoryItemCard.axaml.cs
- **Added setting constants** (Lines 17-18):
  - `MiniChartHistoryDaysKey = "MiniChart.HistoryDays"`
  - `DefaultMiniChartHistoryDays = 14`

- **Updated data fetching logic** (Lines 99-103):
  - Reads setting value using `mainViewModel.SettingsService?.GetSetting()`
  - Falls back to default value if setting not available
  - Uses setting value to calculate `startTime` for historical data query
  - Added debug logging to show which setting value is being used

### 2. MainViewModel.cs
- **Exposed SettingsService property** (Lines 410-413):
  - Added public `SettingsService` property to allow child components access
  - Enables InventoryItemCard to read user settings

- **Initialize default setting** (Lines 1047-1049):
  - Added initialization of "MiniChart.HistoryDays" to 14 days in `InitializeChartSettings()`
  - Ensures setting exists with sensible default value

## Technical Details

### Setting Key Pattern
- Follows existing convention: `"Category.SettingName"`
- Key: `"MiniChart.HistoryDays"`
- Default: `14` days
- Type: `int`

### Data Flow
1. User can modify the setting via settings file or future UI
2. `MainViewModel.InitializeChartSettings()` ensures default exists
3. `InventoryItemCard` reads setting when creating charts
4. Historical data query uses setting value for time range
5. Charts display the configured number of days

### Setting Storage
- Stored in JSON settings file via `JsonSettingsService`
- Location: `%AppData%\InventoryClient\settings.json`
- Persisted between application sessions
- Can be manually edited or changed programmatically

### Error Handling
- Graceful fallback to default value if setting read fails
- Null-safe access using `?.` operator
- Default constant ensures consistent fallback behavior

## Usage Examples

### Current Implementation
```csharp
// Default behavior (14 days)
var historyDays = mainViewModel.SettingsService?.GetSetting("MiniChart.HistoryDays", 14) ?? 14;
var startTime = endTime.AddDays(-historyDays);
```

### Manual Setting
Users can edit the settings file:
```json
{
  "MiniChart.HistoryDays": 30
}
```

### Programmatic Setting
```csharp
settingsService.SetSetting("MiniChart.HistoryDays", 7); // Show 1 week
```

## Future Enhancements

1. **Settings UI**: Add user interface to modify this setting
2. **Validation**: Add range validation (e.g., 1-90 days)
3. **Performance Optimization**: Adjust max data points based on days
4. **Per-Item Settings**: Allow different history ranges per item type
5. **Smart Defaults**: Auto-adjust based on data availability

## Testing Scenarios

### Manual Testing
1. **Default Value**: Verify 14 days used when setting doesn't exist
2. **Custom Value**: Set to 30 days, verify charts show 30 days
3. **Edge Cases**: Test with 1 day, 365 days
4. **Settings File**: Test manual JSON editing
5. **Service Unavailable**: Verify fallback when settings service is null

### Integration Testing
- Test setting persistence across app restarts
- Verify concurrent chart updates when setting changes
- Test with various data densities

## Commit Structure Recommendation

1. **feat: Add MiniChart.HistoryDays setting constants and default** (~4 lines)
   - Setting key constant and default value in InventoryItemCard

2. **feat: Expose SettingsService property in MainViewModel** (~4 lines)
   - Public property for child component access

3. **feat: Initialize MiniChart.HistoryDays default setting** (~3 lines)
   - Add to InitializeChartSettings method

4. **feat: Use MiniChart.HistoryDays setting in chart data fetching** (~5 lines)
   - Read setting and use for time range calculation
   - Add debug logging

Each commit maintains single responsibility and keeps the build functional.
