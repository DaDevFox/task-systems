using System;

namespace InventoryClient.Models;

/// <summary>
/// ViewModel for a single inventory level snapshot (for chart/history)
/// </summary>
public class InventoryLevelSnapshotViewModel
{
    public DateTime Timestamp { get; set; }
    public double Level { get; set; }
    public string UnitId { get; set; } = string.Empty;
    public string Source { get; set; } = string.Empty;
    public string Context { get; set; } = string.Empty;
    public IReadOnlyDictionary<string, string> Metadata { get; set; } = new Dictionary<string, string>();
}
