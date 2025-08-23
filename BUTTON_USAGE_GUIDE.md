# EmphasisButton and UnderlineButton Usage Examples

This document shows how to use the new custom button components in your XAML files.

## EmphasisButton Examples

### Basic Usage in XAML
```xml
<views:EmphasisButton Text="Add Item" 
                      Icon="➕"
                      BaseColor="#22C55E"
                      HoverColor="#16A34A"
                      TextColor="White"
                      ButtonHeight="36"
                      ButtonMinWidth="120"
                      Command="{Binding AddItemCommand}" />
```

### Using Factory Methods (Recommended)
```csharp
// In code-behind or view model
var addButton = EmphasisButton.CreateAddButton();
addButton.Command = AddItemCommand;

var refreshButton = EmphasisButton.CreateRefreshButton();
refreshButton.Command = RefreshChartsCommand;

var saveButton = EmphasisButton.CreateSaveButton();
saveButton.Command = SaveCommand;
```

### Available Factory Methods
- `CreateAddButton()` - Green "Add Item" button with ➕ icon
- `CreateRefreshButton()` - Blue "Refresh Charts" button with 🔄 icon  
- `CreateUpdateButton()` - Blue "Update" button
- `CreateViewButton()` - Green "View" button with 📊 icon
- `CreateDeleteButton()` - Red "Delete" button with 🗑️ icon
- `CreateSaveButton()` - Indigo "Save" button with 💾 icon
- `CreateCancelButton()` - Gray "Cancel" button

### Custom Configuration
```xml
<views:EmphasisButton Text="Custom Action"
                      Icon="⚡"
                      BaseColor="#8B5CF6"
                      HoverColor="#7C3AED"
                      TextColor="White"
                      ButtonHeight="40"
                      ButtonMinWidth="140"
                      ButtonFontSize="14"
                      ButtonFontWeight="SemiBold"
                      SkewAngle="-15"
                      AnimationDuration="0:0:0.4" />
```

## UnderlineButton Examples

### Basic Usage in XAML
```xml
<views:UnderlineButton Text="Settings"
                       TextColor="#374151"
                       HoverTextColor="#2563EB"
                       UnderlineColor="#3B82F6"
                       Command="{Binding ShowSettingsCommand}" />
```

### Using Factory Methods (Recommended)
```csharp
// Navigation buttons
var homeButton = UnderlineButton.CreateNavigationButton("Home");
var inventoryButton = UnderlineButton.CreateNavigationButton("Inventory");
var reportsButton = UnderlineButton.CreateNavigationButton("Reports");

// Secondary actions
var exportButton = UnderlineButton.CreateSecondaryButton("Export Data");
var helpButton = UnderlineButton.CreateSecondaryButton("Help");

// Subtle actions
var aboutButton = UnderlineButton.CreateSubtleButton("About");
var versionButton = UnderlineButton.CreateSubtleButton("v1.0.0");
```

### Available Factory Methods
- `CreateNavigationButton(string text)` - Standard navigation styling
- `CreateSecondaryButton(string text)` - Secondary action styling
- `CreateSubtleButton(string text)` - Subtle/minimal styling

### Custom Configuration
```xml
<views:UnderlineButton Text="Advanced"
                       Icon="🔧"
                       TextColor="#6B7280"
                       HoverTextColor="#111827"
                       UnderlineColor="#F59E0B"
                       ButtonFontSize="13"
                       ButtonFontWeight="Medium"
                       UnderlineThickness="3"
                       AnimationDuration="0:0:0.2" />
```

## Property Reference

### EmphasisButton Properties
- `Text` - Button text
- `Icon` - Optional emoji/symbol icon (displayed before text)
- `BaseColor` - Background color of parallelogram
- `HoverColor` - Background color on hover
- `TextColor` - Text color
- `ButtonFontSize` - Text font size (default: 12)
- `ButtonFontWeight` - Text font weight (default: Medium)
- `ButtonHeight` - Button height (default: 32)
- `ButtonMinWidth` - Minimum button width (default: 80)
- `SkewAngle` - Parallelogram skew angle in degrees (default: -12)
- `AnimationDuration` - Animation duration (default: 0:0:0.3)

### UnderlineButton Properties
- `Text` - Button text
- `Icon` - Optional emoji/symbol icon (displayed before text)
- `TextColor` - Text color in normal state
- `HoverTextColor` - Text color on hover
- `UnderlineColor` - Color of the underline
- `ButtonFontSize` - Text font size (default: 12)
- `ButtonFontWeight` - Text font weight (default: Medium)
- `ButtonHeight` - Button height (default: 32)
- `ButtonMinWidth` - Minimum button width (default: 80)
- `UnderlineThickness` - Thickness of underline in pixels (default: 2)
- `AnimationDuration` - Animation duration (default: 0:0:0.25)

## When to Use Which Button

### Use EmphasisButton for:
- Primary actions (Add, Save, Update, Delete)
- Call-to-action buttons
- Important operations that need visual emphasis
- Main navigation actions

### Use UnderlineButton for:
- Secondary navigation
- Less prominent actions
- Text-based links
- Subtle interactions
- Menu items
- Footer links

This design system ensures consistent styling across your application while providing flexibility for customization.
