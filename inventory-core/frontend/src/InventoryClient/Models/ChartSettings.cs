using Inventory.V1;

namespace InventoryClient.Models;

/// <summary>
/// Chart data mode determines how historical data is fetched
/// </summary>
public enum ChartDataMode
{
    /// <summary>
    /// Use granularity-based sampling for chart data
    /// </summary>
    Granularity = 0,

    /// <summary>
    /// Fetch all data points within a time duration
    /// </summary>
    TimeRange = 1
}

/// <summary>
/// Settings for chart data fetching and display
/// </summary>
public class ChartSettings
{
    /// <summary>
    /// Chart data mode (Granularity or TimeRange)
    /// </summary>
    public ChartDataMode Mode { get; set; } = ChartDataMode.Granularity;

    /// <summary>
    /// Granularity for data sampling (when Mode = Granularity)
    /// </summary>
    public HistoryGranularity Granularity { get; set; } = HistoryGranularity.Day;

    /// <summary>
    /// Maximum number of data points to fetch (when Mode = Granularity)
    /// </summary>
    public int MaxPoints { get; set; } = 100;

    /// <summary>
    /// Time duration in days to fetch all data points (when Mode = TimeRange)
    /// </summary>
    public int TimeRangeDays { get; set; } = 30;

    /// <summary>
    /// Whether to include prediction data in charts
    /// </summary>
    public bool ShowPredictions { get; set; } = true;

    /// <summary>
    /// Number of days ahead to predict (when showing predictions)
    /// </summary>
    public int PredictionDaysAhead { get; set; } = 7;

    // Settings keys
    public const string ModeKey = "Charts.DataMode";
    public const string GranularityKey = "Charts.Granularity";
    public const string MaxPointsKey = "Charts.MaxPoints";
    public const string TimeRangeDaysKey = "Charts.TimeRangeDays";
    public const string ShowPredictionsKey = "Charts.ShowPredictions";
    public const string PredictionDaysAheadKey = "Charts.PredictionDaysAhead";

    /// <summary>
    /// Creates ChartSettings from ISettingsService
    /// </summary>
    public static ChartSettings FromSettings(Services.ISettingsService settingsService)
    {
        return new ChartSettings
        {
            Mode = settingsService.GetSetting(ModeKey, ChartDataMode.Granularity),
            Granularity = settingsService.GetSetting(GranularityKey, HistoryGranularity.Day),
            MaxPoints = settingsService.GetSetting(MaxPointsKey, 100),
            TimeRangeDays = settingsService.GetSetting(TimeRangeDaysKey, 30),
            ShowPredictions = settingsService.GetSetting(ShowPredictionsKey, true),
            PredictionDaysAhead = settingsService.GetSetting(PredictionDaysAheadKey, 7)
        };
    }

    /// <summary>
    /// Saves ChartSettings to ISettingsService
    /// </summary>
    public void SaveToSettings(Services.ISettingsService settingsService)
    {
        settingsService.SetSetting(ModeKey, Mode);
        settingsService.SetSetting(GranularityKey, Granularity);
        settingsService.SetSetting(MaxPointsKey, MaxPoints);
        settingsService.SetSetting(TimeRangeDaysKey, TimeRangeDays);
        settingsService.SetSetting(ShowPredictionsKey, ShowPredictions);
        settingsService.SetSetting(PredictionDaysAheadKey, PredictionDaysAhead);
    }
}
