using Avalonia;
using Avalonia.Controls;
using Avalonia.Controls.Shapes;
using Avalonia.Input;
using Avalonia.Interactivity;
using Avalonia.Media;
using Avalonia.Animation;
using Avalonia.Animation.Easings;
using Avalonia.Media.Transformation;
using System.Windows.Input;

namespace InventoryClient.Views;

/// <summary>
/// Simple button with underline animation on hover
/// </summary>
public partial class UnderlineButton : UserControl
{
    // Dependency Properties
    public static readonly StyledProperty<string> TextProperty =
        AvaloniaProperty.Register<UnderlineButton, string>(nameof(Text), "Button");

    public static readonly StyledProperty<IBrush> TextColorProperty =
        AvaloniaProperty.Register<UnderlineButton, IBrush>(nameof(TextColor), new SolidColorBrush(Color.FromRgb(55, 65, 81))); // Gray-700

    public static readonly StyledProperty<IBrush> HoverTextColorProperty =
        AvaloniaProperty.Register<UnderlineButton, IBrush>(nameof(HoverTextColor), new SolidColorBrush(Color.FromRgb(37, 99, 235))); // Blue-600

    public static readonly StyledProperty<IBrush> UnderlineColorProperty =
        AvaloniaProperty.Register<UnderlineButton, IBrush>(nameof(UnderlineColor), new SolidColorBrush(Color.FromRgb(59, 130, 246))); // Blue-500

    public static readonly StyledProperty<double> ButtonFontSizeProperty =
        AvaloniaProperty.Register<UnderlineButton, double>(nameof(ButtonFontSize), 12.0);

    public static readonly StyledProperty<FontWeight> ButtonFontWeightProperty =
        AvaloniaProperty.Register<UnderlineButton, FontWeight>(nameof(ButtonFontWeight), FontWeight.Medium);

    public static readonly StyledProperty<double> ButtonHeightProperty =
        AvaloniaProperty.Register<UnderlineButton, double>(nameof(ButtonHeight), 32.0);

    public static readonly StyledProperty<double> ButtonMinWidthProperty =
        AvaloniaProperty.Register<UnderlineButton, double>(nameof(ButtonMinWidth), 80.0);

    public static readonly StyledProperty<string> IconProperty =
        AvaloniaProperty.Register<UnderlineButton, string>(nameof(Icon), "");

    public static readonly StyledProperty<double> UnderlineThicknessProperty =
        AvaloniaProperty.Register<UnderlineButton, double>(nameof(UnderlineThickness), 2.0);

    public static readonly StyledProperty<TimeSpan> AnimationDurationProperty =
        AvaloniaProperty.Register<UnderlineButton, TimeSpan>(nameof(AnimationDuration), TimeSpan.FromMilliseconds(250));

    public static readonly StyledProperty<ICommand?> CommandProperty =
        AvaloniaProperty.Register<UnderlineButton, ICommand?>(nameof(Command));

    public static readonly StyledProperty<object?> CommandParameterProperty =
        AvaloniaProperty.Register<UnderlineButton, object?>(nameof(CommandParameter));

    // Events
    public static readonly RoutedEvent<RoutedEventArgs> ClickEvent =
        RoutedEvent.Register<UnderlineButton, RoutedEventArgs>(nameof(Click), RoutingStrategies.Bubble);

    // Properties
    public string Text
    {
        get => GetValue(TextProperty);
        set => SetValue(TextProperty, value);
    }

    public IBrush TextColor
    {
        get => GetValue(TextColorProperty);
        set => SetValue(TextColorProperty, value);
    }

    public IBrush HoverTextColor
    {
        get => GetValue(HoverTextColorProperty);
        set => SetValue(HoverTextColorProperty, value);
    }

    public IBrush UnderlineColor
    {
        get => GetValue(UnderlineColorProperty);
        set => SetValue(UnderlineColorProperty, value);
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

    public double UnderlineThickness
    {
        get => GetValue(UnderlineThicknessProperty);
        set => SetValue(UnderlineThicknessProperty, value);
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

    public UnderlineButton()
    {
        InitializeComponent();

        // Update UI elements when this control is loaded
        this.Loaded += OnLoaded;

        // Set up property change notifications to update UI elements
        this.PropertyChanged += OnPropertyChanged;

        // Set up pointer events for click functionality on the main grid
        this.PointerPressed += OnPointerPressed;
        this.PointerReleased += OnPointerReleased;

        // Set up hover events for animation on the main grid
        this.PointerEntered += OnPointerEntered;
        this.PointerExited += OnPointerExited;
    }

    private void OnLoaded(object? sender, RoutedEventArgs e)
    {
        UpdateUIElements();
        
        // Also attach events to the main grid for better event handling
        var mainGrid = this.FindControl<Grid>("MainGrid");
        if (mainGrid != null)
        {
            mainGrid.PointerPressed += OnPointerPressed;
            mainGrid.PointerReleased += OnPointerReleased;
            mainGrid.PointerEntered += OnPointerEntered;
            mainGrid.PointerExited += OnPointerExited;
        }
    }

    private void OnPropertyChanged(object? sender, AvaloniaPropertyChangedEventArgs e)
    {
        if (e.Property == TextProperty ||
            e.Property == TextColorProperty ||
            e.Property == HoverTextColorProperty ||
            e.Property == UnderlineColorProperty ||
            e.Property == ButtonFontSizeProperty ||
            e.Property == ButtonFontWeightProperty ||
            e.Property == ButtonHeightProperty ||
            e.Property == ButtonMinWidthProperty ||
            e.Property == IconProperty ||
            e.Property == UnderlineThicknessProperty)
        {
            UpdateUIElements();
        }
    }

    private void UpdateUIElements()
    {
        // Find the main grid by name
        var grid = this.FindControl<Grid>("MainGrid");
        var buttonText = this.FindControl<TextBlock>("ButtonText");
        var underline = this.FindControl<Rectangle>("Underline");

        // Update grid dimensions
        if (grid != null)
        {
            grid.Height = ButtonHeight;
            grid.MinWidth = ButtonMinWidth;
        }

        // Update text with icon support
        if (buttonText != null)
        {
            var displayText = string.IsNullOrEmpty(Icon) ? Text : $"{Icon} {Text}";
            buttonText.Text = displayText;
            buttonText.Foreground = TextColor;
            buttonText.FontSize = ButtonFontSize;
            buttonText.FontWeight = ButtonFontWeight;

            // Update text transitions with dynamic duration
            var textTransitions = new Transitions();
            textTransitions.Add(new BrushTransition
            {
                Property = TextBlock.ForegroundProperty,
                Duration = TimeSpan.FromMilliseconds(AnimationDuration.TotalMilliseconds * 0.8) // Slightly faster than underline
            });
            buttonText.Transitions = textTransitions;
        }

        // Update underline
        if (underline != null)
        {
            underline.Fill = UnderlineColor;
            underline.Height = UnderlineThickness;
            
            // Ensure initial state is set properly
            underline.RenderTransform = TransformOperations.Parse("scaleX(0)");
            underline.RenderTransformOrigin = RelativePoint.Parse("0,0.5");

            // Update underline transitions with dynamic duration and easing
            var transitions = new Transitions();
            transitions.Add(new TransformOperationsTransition
            {
                Property = Visual.RenderTransformProperty,
                Duration = AnimationDuration,
                Easing = new QuadraticEaseOut()
            });
            underline.Transitions = transitions;
        }
    }

    private void OnPointerEntered(object? sender, PointerEventArgs e)
    {
        var buttonText = this.FindControl<TextBlock>("ButtonText");
        var underline = this.FindControl<Rectangle>("Underline");

        // Change text color and scale in underline
        if (buttonText != null)
        {
            buttonText.Foreground = HoverTextColor;
        }

        if (underline != null)
        {
            underline.RenderTransform = TransformOperations.Parse("scaleX(1)");
        }

        e.Handled = true;
    }

    private void OnPointerExited(object? sender, PointerEventArgs e)
    {
        var buttonText = this.FindControl<TextBlock>("ButtonText");
        var underline = this.FindControl<Rectangle>("Underline");

        // Revert text color and scale out underline
        if (buttonText != null)
        {
            buttonText.Foreground = TextColor;
        }

        if (underline != null)
        {
            underline.RenderTransform = TransformOperations.Parse("scaleX(0)");
        }

        e.Handled = true;
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
    /// Creates a standard navigation underline button
    /// </summary>
    public static UnderlineButton CreateNavigationButton(string text)
    {
        return new UnderlineButton
        {
            Text = text,
            TextColor = new SolidColorBrush(Color.FromRgb(55, 65, 81)), // Gray-700
            HoverTextColor = new SolidColorBrush(Color.FromRgb(37, 99, 235)), // Blue-600
            UnderlineColor = new SolidColorBrush(Color.FromRgb(59, 130, 246)), // Blue-500
            ButtonFontSize = 12,
            ButtonMinWidth = 60
        };
    }

    /// <summary>
    /// Creates a secondary action underline button
    /// </summary>
    public static UnderlineButton CreateSecondaryButton(string text)
    {
        return new UnderlineButton
        {
            Text = text,
            TextColor = new SolidColorBrush(Color.FromRgb(107, 114, 128)), // Gray-500
            HoverTextColor = new SolidColorBrush(Color.FromRgb(55, 65, 81)), // Gray-700
            UnderlineColor = new SolidColorBrush(Color.FromRgb(107, 114, 128)), // Gray-500
            ButtonFontSize = 11,
            UnderlineThickness = 1.5
        };
    }

    /// <summary>
    /// Creates a subtle action underline button
    /// </summary>
    public static UnderlineButton CreateSubtleButton(string text)
    {
        return new UnderlineButton
        {
            Text = text,
            TextColor = new SolidColorBrush(Color.FromRgb(156, 163, 175)), // Gray-400
            HoverTextColor = new SolidColorBrush(Color.FromRgb(107, 114, 128)), // Gray-500
            UnderlineColor = new SolidColorBrush(Color.FromRgb(156, 163, 175)), // Gray-400
            ButtonFontSize = 10,
            UnderlineThickness = 1
        };
    }
}
