using Avalonia;
using Avalonia.Controls;
using Avalonia.Markup.Xaml;

namespace TaskSystems.Shared.Controls;

/// <summary>
/// Reusable status indicator control for showing item health/status across services
/// </summary>
public partial class StatusIndicator : UserControl
{
    public static readonly StyledProperty<string> StatusProperty =
        AvaloniaProperty.Register<StatusIndicator, string>(nameof(Status), "Normal");

    public static readonly StyledProperty<string> DescriptionProperty =
        AvaloniaProperty.Register<StatusIndicator, string>(nameof(Description), "");

    public string Status
    {
        get => GetValue(StatusProperty);
        set => SetValue(StatusProperty, value);
    }

    public string Description
    {
        get => GetValue(DescriptionProperty);
        set => SetValue(DescriptionProperty, value);
    }

    public StatusIndicator()
    {
        InitializeComponent();
    }

    private void InitializeComponent()
    {
        AvaloniaXamlLoader.Load(this);
    }
}
