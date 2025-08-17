using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using InventoryClient.Models;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using System.Collections.ObjectModel;
using System.ComponentModel.DataAnnotations;

namespace InventoryClient.ViewModels;

/// <summary>
/// ViewModel for the Add Item dialog
/// </summary>
public partial class AddItemDialogViewModel : ObservableValidator
{
    private readonly IInventoryService _inventoryService;
    private readonly ILogger<AddItemDialogViewModel> _logger;

    [ObservableProperty]
    [NotifyCanExecuteChangedFor(nameof(AddItemCommand))]
    [Required(ErrorMessage = "Item name is required")]
    private string _name = string.Empty;

    [ObservableProperty]
    private string _description = string.Empty;

    [ObservableProperty]
    [NotifyCanExecuteChangedFor(nameof(AddItemCommand))]
    [Range(0, double.MaxValue, ErrorMessage = "Initial level must be 0 or greater")]
    private double _initialLevel = 0;

    [ObservableProperty]
    [NotifyCanExecuteChangedFor(nameof(AddItemCommand))]
    [Range(0.01, double.MaxValue, ErrorMessage = "Max capacity must be greater than 0")]
    private double _maxCapacity = 100;

    [ObservableProperty]
    [NotifyCanExecuteChangedFor(nameof(AddItemCommand))]
    [Range(0, double.MaxValue, ErrorMessage = "Low stock threshold must be 0 or greater")]
    private double _lowStockThreshold = 10;

    [ObservableProperty]
    [NotifyCanExecuteChangedFor(nameof(AddItemCommand))]
    [Required(ErrorMessage = "Unit type is required")]
    private string _unitId = "kg";

    [ObservableProperty]
    private ObservableCollection<MetadataItem> _metadataItems;

    [ObservableProperty]
    private string _validationError = string.Empty;

    [ObservableProperty]
    private bool _hasValidationError;

    [ObservableProperty]
    private bool _isValid = false;

    [ObservableProperty]
    [NotifyPropertyChangedFor(nameof(AddItemButtonText))]
    [NotifyCanExecuteChangedFor(nameof(AddItemCommand))]
    private bool _isSubmitting = false;

    public string AddItemButtonText => IsSubmitting ? "Adding..." : "Add Item";

    public event EventHandler? OnItemAdded;
    public event EventHandler? OnCanceled;

    public AddItemDialogViewModel(IInventoryService inventoryService, ILogger<AddItemDialogViewModel> logger)
    {
        _inventoryService = inventoryService;
        _logger = logger;

        // Initialize metadata with default fields
        _metadataItems = new ObservableCollection<MetadataItem>
        {
            new MetadataItem("quality", "", true),
            new MetadataItem("damage", "", true),
            new MetadataItem("misc_notes", "", true)
        };

        // Validate on property changes
        PropertyChanged += (s, e) => ValidateAll();
        ValidateAll(); // Initial validation
    }

    [RelayCommand(CanExecute = nameof(CanAddItem))]
    private async Task AddItem()
    {
        if (!IsValid || IsSubmitting)
            return;

        try
        {
            IsSubmitting = true;
            ClearValidationError();

            var metadata = new Dictionary<string, string>();

            // Add metadata from the dynamic collection
            foreach (var item in MetadataItems.Where(m => m.IsValid && !string.IsNullOrWhiteSpace(m.Value)))
            {
                metadata[item.Key] = item.Value;
            }

            var result = await _inventoryService.AddInventoryItemAsync(
                Name,
                Description,
                InitialLevel,
                MaxCapacity,
                LowStockThreshold,
                UnitId,
                metadata.Count > 0 ? metadata : null);

            if (result != null)
            {
                _logger.LogInformation("Successfully added inventory item: {Name}", Name);
                OnItemAdded?.Invoke(this, EventArgs.Empty);
            }
            else
            {
                SetValidationError("Failed to add inventory item. Please try again.");
            }
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error adding inventory item");
            SetValidationError($"Error: {ex.Message}");
        }
        finally
        {
            IsSubmitting = false;
        }
    }

    [RelayCommand]
    private void Cancel()
    {
        OnCanceled?.Invoke(this, EventArgs.Empty);
    }

    [RelayCommand]
    private void AddMetadataItem()
    {
        MetadataItems.Add(new MetadataItem("", "", false));
    }

    [RelayCommand]
    private void RemoveMetadataItem(MetadataItem item)
    {
        if (item != null && !item.IsDefault)
        {
            MetadataItems.Remove(item);
        }
    }

    private bool CanAddItem()
    {
        return IsValid && !IsSubmitting;
    }

    private void ValidateAll()
    {
        var errors = new List<string>();

        // Name validation
        if (string.IsNullOrWhiteSpace(Name))
            errors.Add("Item name is required");

        // Unit ID validation
        if (string.IsNullOrWhiteSpace(UnitId))
            errors.Add("Unit type is required");

        // Numeric validations
        if (MaxCapacity <= 0)
            errors.Add("Max capacity must be greater than 0");

        if (InitialLevel < 0)
            errors.Add("Initial level cannot be negative");

        if (LowStockThreshold < 0)
            errors.Add("Low stock threshold cannot be negative");

        if (InitialLevel > MaxCapacity)
            errors.Add("Initial level cannot exceed max capacity");

        if (LowStockThreshold > MaxCapacity)
            errors.Add("Low stock threshold cannot exceed max capacity");

        // Update validation state
        IsValid = errors.Count == 0;

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
