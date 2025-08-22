# EmphasisButton Component

## Overview
The `EmphasisButton` is a custom Avalonia UserControl that provides a modern, parallelogram-shaped button with a smooth sliding hover effect. It replaces traditional rounded buttons with angular, geometric shapes while providing engaging visual feedback.

## Features

### Visual Design
- **Parallelogram Shape**: Uses CSS skew transforms to create clean, angular button shapes
- **Sliding Hover Effect**: Color slides in from left-to-right with smooth animation
- **Hand Cursor**: Changes pointer to "Hand" cursor on hover for better UX
- **No Rounded Corners**: Clean rectangular/parallelogram edges for precise geometric design

### Interaction
- **Click Events**: Supports both routed events and ICommand binding
- **Smooth Animations**: Optimized transition timings with easing functions
- **Visual Feedback**: Immediate response to user interaction

### Customization
- **Colors**: Fully customizable base color, hover color, and text color
- **Text**: Configurable button text with emoji support
- **Command Binding**: Full MVVM support with Command and CommandParameter properties

## Usage

### Basic Usage
```xml
<views:EmphasisButton Text="Click Me" 
                     Click="OnButtonClick"/>
```

### With Custom Colors
```xml
<views:EmphasisButton Text="Custom Button"
                     BaseColor="#3b82f6"
                     HoverColor="#2563eb"
                     TextColor="White"
                     Click="OnButtonClick"/>
```

### With Command Binding
```xml
<views:EmphasisButton Text="Save" 
                     Command="{Binding SaveCommand}"
                     CommandParameter="{Binding CurrentItem}"/>
```

### Predefined Styles
The component includes factory methods for common button types:

```csharp
// Update button (blue theme)
var updateButton = EmphasisButton.CreateUpdateButton();

// View button (green theme) 
var viewButton = EmphasisButton.CreateViewButton();

// Delete button (red theme)
var deleteButton = EmphasisButton.CreateDeleteButton();
```

## Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `Text` | `string` | "Button" | Button text content |
| `BaseColor` | `IBrush` | Blue (#3b82f6) | Base background color |
| `HoverColor` | `IBrush` | Dark Blue (#2563eb) | Hover overlay color |
| `TextColor` | `IBrush` | White | Text color |
| `Command` | `ICommand?` | null | Command to execute on click |
| `CommandParameter` | `object?` | null | Parameter for command |

## Events

| Event | Type | Description |
|-------|------|-------------|
| `Click` | `RoutedEvent<RoutedEventArgs>` | Fired when button is clicked |

## Implementation Details

### Animation System
- **Hover Overlay**: Starts at -120% translateX, slides to 0% on hover
- **Opacity Transition**: Fades from 0 to 1 over 0.25 seconds
- **Transform Transition**: Slides over 0.35 seconds with QuadraticEaseOut easing
- **Counter-Skew Text**: Text is reverse-skewed to appear normal within skewed container

### Color Scheme Examples
```xml
<!-- Primary Action (Blue) -->
BaseColor="#3b82f6" HoverColor="#2563eb"

<!-- Success Action (Green) -->  
BaseColor="#10b981" HoverColor="#059669"

<!-- Destructive Action (Red) -->
BaseColor="#dc2626" HoverColor="#b91c1c"

<!-- Secondary Action (Gray) -->
BaseColor="#6b7280" HoverColor="#4b5563"
```

## Design Philosophy

The `EmphasisButton` follows these design principles:

1. **No Rounded Corners**: Clean geometric shapes avoid the stress of imperfect curve rendering
2. **Directional Animation**: Left-to-right motion suggests forward progress/action
3. **Parallelogram Geometry**: Creates visual interest while maintaining button recognition
4. **Consistent Interaction**: Hand cursor and smooth feedback provide clear affordances
5. **Color Hierarchy**: Darker hover states suggest depth and interaction

## Integration with InventoryItemCard

The buttons have been integrated into the inventory item cards as follows:

- **Update Button**: Default blue styling for primary actions
- **View/History Button**: Green styling with chart emoji for data viewing
- **Delete Button**: Red styling with trash emoji for destructive actions

This creates a clear visual hierarchy and consistent interaction patterns across the application.

## Performance Considerations

- Animations use GPU-accelerated transforms (translateX, skewX)
- Minimal DOM impact with single UserControl per button
- CSS transitions handled by Avalonia's optimized animation system
- No bitmap operations or complex path rendering

## Browser/Platform Compatibility

Designed for Avalonia UI framework with cross-platform support:
- Windows Desktop ✓
- macOS Desktop ✓  
- Linux Desktop ✓
- Future web support when Avalonia adds WebAssembly target

## Future Enhancements

1. **Ripple Effect**: Add material design-style ripple on click
2. **Loading State**: Built-in spinner/loading animation support
3. **Icon Support**: Dedicated icon property with proper sizing
4. **Size Variants**: Small, medium, large preset sizes
5. **Themes**: Light/dark theme support with automatic color adaptation
