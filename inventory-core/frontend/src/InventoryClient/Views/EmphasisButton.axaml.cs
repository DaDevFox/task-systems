using Avalonia;
using Avalonia.Controls;
using Avalonia.Input;
using Avalonia.Interactivity;
using Avalonia.Media;
using Avalonia.Animation;
using Avalonia.Animation.Easings;
using Avalonia.Styling;
using System.Windows.Input;
using Avalonia.Media.Transformation;

namespace InventoryClient.Views;

/// <summary>
/// Custom emphasis button with parallelogram shape and sliding hover effect
/// </summary>
public partial class EmphasisButton : UserControl
{
    // Dependency Properties
    public static readonly StyledProperty<string> TextProperty =
        AvaloniaProperty.Register<EmphasisButton, string>(nameof(Text), "Button");

    public static readonly StyledProperty<IBrush> BaseColorProperty =
        AvaloniaProperty.Register<EmphasisButton, IBrush>(nameof(BaseColor), new SolidColorBrush(Color.FromRgb(59, 130, 246))); // Blue

    public static readonly StyledProperty<IBrush> HoverColorProperty =
        AvaloniaProperty.Register<EmphasisButton, IBrush>(nameof(HoverColor), new SolidColorBrush(Color.FromRgb(37, 99, 235))); // Darker blue

    public static readonly StyledProperty<IBrush> TextColorProperty =
        AvaloniaProperty.Register<EmphasisButton, IBrush>(nameof(TextColor), Brushes.White);

    public static readonly StyledProperty<double> ButtonFontSizeProperty =
        AvaloniaProperty.Register<EmphasisButton, double>(nameof(ButtonFontSize), 12.0);

    public static readonly StyledProperty<FontWeight> ButtonFontWeightProperty =
        AvaloniaProperty.Register<EmphasisButton, FontWeight>(nameof(ButtonFontWeight), FontWeight.Medium);

    public static readonly StyledProperty<double> ButtonHeightProperty =
        AvaloniaProperty.Register<EmphasisButton, double>(nameof(ButtonHeight), 32.0);

    public static readonly StyledProperty<double> ButtonMinWidthProperty =
        AvaloniaProperty.Register<EmphasisButton, double>(nameof(ButtonMinWidth), 80.0);

    public static readonly StyledProperty<string> IconProperty =
        AvaloniaProperty.Register<EmphasisButton, string>(nameof(Icon), "");

    public static readonly StyledProperty<double> SkewAngleProperty =
        AvaloniaProperty.Register<EmphasisButton, double>(nameof(SkewAngle), -12.0);

    public static readonly StyledProperty<TimeSpan> AnimationDurationProperty =
        AvaloniaProperty.Register<EmphasisButton, TimeSpan>(nameof(AnimationDuration), TimeSpan.FromMilliseconds(300));

    public static readonly StyledProperty<ICommand?> CommandProperty =
        AvaloniaProperty.Register<EmphasisButton, ICommand?>(nameof(Command));

    public static readonly StyledProperty<object?> CommandParameterProperty =
        AvaloniaProperty.Register<EmphasisButton, object?>(nameof(CommandParameter));

    // Events
    public static readonly RoutedEvent<RoutedEventArgs> ClickEvent =
        RoutedEvent.Register<EmphasisButton, RoutedEventArgs>(nameof(Click), RoutingStrategies.Bubble);

    // Properties
    public string Text
    {
        get => GetValue(TextProperty);
        set => SetValue(TextProperty, value);
    }

    public IBrush BaseColor
    {
        get => GetValue(BaseColorProperty);
        set => SetValue(BaseColorProperty, value);
    }

    public IBrush HoverColor
    {
        get => GetValue(HoverColorProperty);
        set => SetValue(HoverColorProperty, value);
    }

    public IBrush TextColor
    {
        get => GetValue(TextColorProperty);
        set => SetValue(TextColorProperty, value);
    }

    public double ButtonFontSize
    {
        get => GetValue(ButtonFontSizeProperty);
        set => SetValue(ButtonFontSizeProperty, value);
    }

    public FontWeight ButtonFontWeight
    {
        get => GetValue(ButtonFontWeightProperty);
        set => SetValue(ButtonFontWeightProperty, value);
    }

    public double ButtonHeight
    {
        get => GetValue(ButtonHeightProperty);
        set => SetValue(ButtonHeightProperty, value);
    }

    public double ButtonMinWidth
    {
        get => GetValue(ButtonMinWidthProperty);
        set => SetValue(ButtonMinWidthProperty, value);
    }

    public string Icon
    {
        get => GetValue(IconProperty);
        set => SetValue(IconProperty, value);
    }

    public double SkewAngle
    {
        get => GetValue(SkewAngleProperty);
        set => SetValue(SkewAngleProperty, value);
    }

    public TimeSpan AnimationDuration
    {
        get => GetValue(AnimationDurationProperty);
        set => SetValue(AnimationDurationProperty, value);
    }

    public ICommand? Command
    {
        get => GetValue(CommandProperty);
        set => SetValue(CommandProperty, value);
    }

    public object? CommandParameter
    {
        get => GetValue(CommandParameterProperty);
        set => SetValue(CommandParameterProperty, value);
    }

    public event EventHandler<RoutedEventArgs> Click
    {
        add => AddHandler(ClickEvent, value);
        remove => RemoveHandler(ClickEvent, value);
    }

    public EmphasisButton()
    {
        InitializeComponent();

        // Update UI elements when this control is loaded
        this.Loaded += OnLoaded;

        // Set up property change notifications to update UI elements
        this.PropertyChanged += OnPropertyChanged;

        // Set up pointer events for click functionality
        this.PointerPressed += OnPointerPressed;
        this.PointerReleased += OnPointerReleased;

        // Set up hover events for animation
        this.PointerEntered += OnPointerEntered;
        this.PointerExited += OnPointerExited;
    }

    private void OnLoaded(object? sender, RoutedEventArgs e)
    {
        UpdateUIElements();
    }

    private void OnPropertyChanged(object? sender, AvaloniaPropertyChangedEventArgs e)
    {
        if (e.Property == TextProperty || 
            e.Property == BaseColorProperty || 
            e.Property == HoverColorProperty || 
            e.Property == TextColorProperty ||
            e.Property == ButtonFontSizeProperty ||
            e.Property == ButtonFontWeightProperty ||
            e.Property == ButtonHeightProperty ||
            e.Property == ButtonMinWidthProperty ||
            e.Property == IconProperty ||
            e.Property == SkewAngleProperty)
        {
            UpdateUIElements();
        }
    }

    private void UpdateUIElements()
    {
        // Find the main grid by name
        var grid = this.FindControl<Grid>("MainGrid");
        var baseElement = this.FindControl<Border>("BaseElement");
        var hoverOverlay = this.FindControl<Border>("HoverOverlay");
        var buttonText = this.FindControl<TextBlock>("ButtonText");

        // Update grid dimensions
        if (grid != null)
        {
            grid.Height = ButtonHeight;
            grid.MinWidth = ButtonMinWidth;
        }

        // Update base element
        if (baseElement != null)
        {
            baseElement.Background = BaseColor;
            baseElement.RenderTransform = TransformOperations.Parse($"skew({SkewAngle}deg, 0deg)");
        }

        // Update hover overlay
        if (hoverOverlay != null)
        {
            hoverOverlay.Background = HoverColor;
        }

        // Update text with icon support
        if (buttonText != null)
        {
            var displayText = string.IsNullOrEmpty(Icon) ? Text : $"{Icon} {Text}";
            buttonText.Text = displayText;
            buttonText.Foreground = TextColor;
            buttonText.FontSize = ButtonFontSize;
            buttonText.FontWeight = ButtonFontWeight;
        }

        // Update hover overlay transitions with dynamic duration
        if (hoverOverlay != null)
        {
            var transitions = new Transitions();
            transitions.Add(new DoubleTransition
            {
                Property = Border.OpacityProperty,
                Duration = AnimationDuration
            });
            transitions.Add(new TransformOperationsTransition
            {
                Property = Visual.RenderTransformProperty,
                Duration = TimeSpan.FromMilliseconds(AnimationDuration.TotalMilliseconds * 0.8) // Slightly faster transform
            });
            hoverOverlay.Transitions = transitions;
        }
    }

    private void OnPointerEntered(object? sender, PointerEventArgs e)
    {
        // Find the hover overlay
        var hoverOverlay = this.FindControl<Border>("HoverOverlay");

        if (hoverOverlay != null)
        {
            // Animate to full scale while maintaining skew using TransformOperations.Parse
            // Set opacity and transform using TransformOperations.Parse with scaleX
            hoverOverlay.Opacity = 1.0;
            hoverOverlay.RenderTransform = TransformOperations.Parse("skew(-12deg, 0deg) scaleX(1)");
        }
    }

    private void OnPointerExited(object? sender, PointerEventArgs e)
    {
        // Find the hover overlay
        var hoverOverlay = this.FindControl<Border>("HoverOverlay");

        if (hoverOverlay != null)
        {
            // Animate back to zero scale while maintaining skew using TransformOperations.Parse
            // Set opacity and transform using TransformOperations.Parse with scaleX
            hoverOverlay.Opacity = 0.0;
            hoverOverlay.RenderTransform = TransformOperations.Parse("skew(-12deg, 0deg) scaleX(0)");
        }
    }

    private void OnPointerPressed(object? sender, PointerPressedEventArgs e)
    {
        if (e.GetCurrentPoint(this).Properties.IsLeftButtonPressed)
        {
            e.Handled = true;
        }
    }

    private void OnPointerReleased(object? sender, PointerReleasedEventArgs e)
    {
        if (e.InitialPressMouseButton == MouseButton.Left)
        {
            // Fire click event
            var clickArgs = new RoutedEventArgs(ClickEvent);
            RaiseEvent(clickArgs);

            // Execute command if bound
            if (Command?.CanExecute(CommandParameter) == true)
            {
                Command.Execute(CommandParameter);
            }

            e.Handled = true;
        }
    }

    /// <summary>
    /// Creates a standard "Add Item" emphasis button
    /// </summary>
    public static EmphasisButton CreateAddButton()
    {
        return new EmphasisButton
        {
            Text = "Add Item",
            Icon = "➕",
            BaseColor = new SolidColorBrush(Color.FromRgb(34, 197, 94)), // Green
            HoverColor = new SolidColorBrush(Color.FromRgb(22, 163, 74)), // Darker green
            TextColor = Brushes.White,
            ButtonHeight = 36,
            ButtonMinWidth = 120,
            ButtonFontSize = 13,
            ButtonFontWeight = FontWeight.SemiBold
        };
    }

    /// <summary>
    /// Creates a standard "Refresh Charts" emphasis button
    /// </summary>
    public static EmphasisButton CreateRefreshButton()
    {
        return new EmphasisButton
        {
            Text = "Refresh Charts",
            Icon = "🔄",
            BaseColor = new SolidColorBrush(Color.FromRgb(59, 130, 246)), // Blue
            HoverColor = new SolidColorBrush(Color.FromRgb(37, 99, 235)), // Darker blue
            TextColor = Brushes.White,
            ButtonHeight = 32,
            ButtonMinWidth = 140,
            ButtonFontSize = 12
        };
    }

    /// <summary>
    /// Creates a standard "Update" emphasis button
    /// </summary>
    public static EmphasisButton CreateUpdateButton()
    {
        return new EmphasisButton
        {
            Text = "Update",
            BaseColor = new SolidColorBrush(Color.FromRgb(59, 130, 246)), // Blue
            HoverColor = new SolidColorBrush(Color.FromRgb(37, 99, 235)), // Darker blue
            TextColor = Brushes.White
        };
    }

    /// <summary>
    /// Creates a standard "View" emphasis button
    /// </summary>
    public static EmphasisButton CreateViewButton()
    {
        return new EmphasisButton
        {
            Text = "View",
            Icon = "📊",
            BaseColor = new SolidColorBrush(Color.FromRgb(16, 185, 129)), // Green
            HoverColor = new SolidColorBrush(Color.FromRgb(5, 150, 105)), // Darker green
            TextColor = Brushes.White
        };
    }

    /// <summary>
    /// Creates a standard "Delete" emphasis button
    /// </summary>
    public static EmphasisButton CreateDeleteButton()
    {
        return new EmphasisButton
        {
            Text = "Delete",
            Icon = "🗑️",
            BaseColor = new SolidColorBrush(Color.FromRgb(220, 38, 38)), // Red
            HoverColor = new SolidColorBrush(Color.FromRgb(185, 28, 28)), // Darker red
            TextColor = Brushes.White,
            ButtonMinWidth = 90
        };
    }

    /// <summary>
    /// Creates a standard "Save" emphasis button
    /// </summary>
    public static EmphasisButton CreateSaveButton()
    {
        return new EmphasisButton
        {
            Text = "Save",
            Icon = "💾",
            BaseColor = new SolidColorBrush(Color.FromRgb(99, 102, 241)), // Indigo
            HoverColor = new SolidColorBrush(Color.FromRgb(79, 70, 229)), // Darker indigo
            TextColor = Brushes.White,
            ButtonFontWeight = FontWeight.SemiBold
        };
    }

    /// <summary>
    /// Creates a standard "Cancel" emphasis button
    /// </summary>
    public static EmphasisButton CreateCancelButton()
    {
        return new EmphasisButton
        {
            Text = "Cancel",
            BaseColor = new SolidColorBrush(Color.FromRgb(107, 114, 128)), // Gray
            HoverColor = new SolidColorBrush(Color.FromRgb(75, 85, 99)), // Darker gray
            TextColor = Brushes.White
        };
    }
}
