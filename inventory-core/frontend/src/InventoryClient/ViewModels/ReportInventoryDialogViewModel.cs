using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using InventoryClient.Models;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using System.Collections.ObjectModel;

namespace InventoryClient.ViewModels;

/// <summary>
/// ViewModel for the Report Inventory Level dialog
/// </summary>
public partial class ReportInventoryDialogViewModel : ObservableObject
{
    private readonly IInventoryService _inventoryService;
    private readonly ILogger<ReportInventoryDialogViewModel> _logger;

    [ObservableProperty]
    private ObservableCollection<InventoryItemViewModel> _availableItems = new();

    [ObservableProperty]
    [NotifyPropertyChangedFor(nameof(IsItemSelected), nameof(CurrentLevelDisplay), nameof(MaxCapacityDisplay), nameof(MaxCapacity))]
    private InventoryItemViewModel? _selectedItem;

    [ObservableProperty]
    [NotifyPropertyChangedFor(nameof(LevelChange), nameof(LevelChangeColor), nameof(CanUpdateLevel))]
    private double _newLevel;

    [ObservableProperty]
    private string _reason = string.Empty;

    [ObservableProperty]
    private bool _recordConsumption = true;

    [ObservableProperty]
    private string _validationError = string.Empty;

    [ObservableProperty]
    private bool _hasValidationError;

    public bool IsItemSelected => SelectedItem != null;
    
    public string CurrentLevelDisplay => SelectedItem != null ? 
        $"{SelectedItem.CurrentLevel:F2} {SelectedItem.UnitId}" : string.Empty;
    
    public string MaxCapacityDisplay => SelectedItem != null ? 
        $"Max: {SelectedItem.MaxCapacity:F2} {SelectedItem.UnitId}" : string.Empty;
    
    public double MaxCapacity => SelectedItem?.MaxCapacity ?? 0;
    
    public double LevelChange => SelectedItem != null ? NewLevel - SelectedItem.CurrentLevel : 0;
    
    public string LevelChangeColor => LevelChange switch
    {
        > 0 => "#16a34a", // Green for increase
        < 0 => "#dc2626", // Red for decrease  
        _ => "#64748b"    // Gray for no change
    };

    public bool CanUpdateLevel => IsItemSelected && NewLevel >= 0 && NewLevel <= MaxCapacity && !HasValidationError;

    public event EventHandler? OnLevelUpdated;
    public event EventHandler? OnCanceled;

    public ReportInventoryDialogViewModel(IInventoryService inventoryService, ILogger<ReportInventoryDialogViewModel> logger)
    {
        _inventoryService = inventoryService;
        _logger = logger;
        
        PropertyChanged += OnPropertyChanged;
    }

    private void OnPropertyChanged(object? sender, System.ComponentModel.PropertyChangedEventArgs e)
    {
        if (e.PropertyName == nameof(SelectedItem))
        {
            if (SelectedItem != null)
            {
                NewLevel = SelectedItem.CurrentLevel;
                ClearValidationError();
            }
        }
        else if (e.PropertyName == nameof(NewLevel))
        {
            ValidateNewLevel();
        }
    }

    public async Task LoadAvailableItemsAsync()
    {
        try
        {
            var (items, _) = await _inventoryService.ListInventoryItemsAsync(
                lowStockOnly: false,
                unitTypeFilter: null,
                limit: 1000,
                offset: 0);

            AvailableItems.Clear();
            foreach (var item in items.OrderBy(i => i.Name))
            {
                AvailableItems.Add(item);
            }
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to load available items");
            SetValidationError($"Failed to load items: {ex.Message}");
        }
    }

    [RelayCommand(CanExecute = nameof(CanUpdateLevel))]
    private async Task UpdateLevel()
    {
        if (SelectedItem == null || !CanUpdateLevel)
            return;

        try
        {
            var success = await _inventoryService.UpdateInventoryLevelAsync(
                SelectedItem.Id,
                NewLevel,
                Reason,
                RecordConsumption);

            if (success)
            {
                _logger.LogInformation("Successfully updated inventory level for {ItemName} to {NewLevel}", 
                    SelectedItem.Name, NewLevel);
                OnLevelUpdated?.Invoke(this, EventArgs.Empty);
            }
            else
            {
                SetValidationError("Failed to update inventory level. Please try again.");
            }
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error updating inventory level");
            SetValidationError($"Error: {ex.Message}");
        }
    }

    [RelayCommand]
    private void Cancel()
    {
        OnCanceled?.Invoke(this, EventArgs.Empty);
    }

    private void ValidateNewLevel()
    {
        if (SelectedItem == null)
            return;

        var errors = new List<string>();

        if (NewLevel < 0)
            errors.Add("Level cannot be negative");

        if (NewLevel > SelectedItem.MaxCapacity)
            errors.Add($"Level cannot exceed maximum capacity ({SelectedItem.MaxCapacity:F2} {SelectedItem.UnitId})");

        if (errors.Count > 0)
        {
            SetValidationError(string.Join("; ", errors));
        }
        else
        {
            ClearValidationError();
        }
    }

    private void SetValidationError(string error)
    {
        ValidationError = error;
        HasValidationError = true;
    }

    private void ClearValidationError()
    {
        ValidationError = string.Empty;
        HasValidationError = false;
    }
}
