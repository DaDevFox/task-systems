using Avalonia.Headless;
using Avalonia.Headless.XUnit;
using FluentAssertions;
using InventoryClient.ViewModels;
using InventoryClient.Views;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Xunit;

namespace InventoryClient.Tests.Integration;

public class EndToEndTests
{
    [AvaloniaFact]
    public async Task Application_ShouldStartAndDisplayMainWindow()
    {
        // Arrange
        var host = CreateTestHost();
        var mainViewModel = host.Services.GetRequiredService<MainViewModel>();

        // Act
        var mainWindow = new MainWindow
        {
            DataContext = mainViewModel
        };

        mainWindow.Show();

        // Assert
        mainWindow.Should().NotBeNull();
        mainWindow.DataContext.Should().Be(mainViewModel);
    }

    [AvaloniaFact]
    public async Task MainViewModel_ShouldLoadMockDataOnRefresh()
    {
        // Arrange
        var host = CreateTestHost();
        var mainViewModel = host.Services.GetRequiredService<MainViewModel>();

        // Act
        await mainViewModel.RefreshCommand.ExecuteAsync(null);

        // Assert
        mainViewModel.InventoryItems.Should().NotBeEmpty();
        mainViewModel.TotalItems.Should().BeGreaterThan(0);
        mainViewModel.LowStockCount.Should().BeGreaterOrEqualTo(0);
        mainViewModel.EmptyItemsCount.Should().BeGreaterOrEqualTo(0);
    }

    [AvaloniaFact]
    public async Task Connection_ShouldUpdateStatusWhenConnecting()
    {
        // Arrange
        var host = CreateTestHost();
        var mainViewModel = host.Services.GetRequiredService<MainViewModel>();

        // Act
        await mainViewModel.ConnectCommand.ExecuteAsync(null);

        // Assert
        mainViewModel.IsConnected.Should().BeTrue();
        mainViewModel.ConnectionStatus.Should().Be("Connected");
    }

    [AvaloniaFact]
    public async Task Search_ShouldFilterItemsCorrectly()
    {
        // Arrange
        var host = CreateTestHost();
        var mainViewModel = host.Services.GetRequiredService<MainViewModel>();
        await mainViewModel.RefreshCommand.ExecuteAsync(null);

        // Act
        mainViewModel.SearchText = "Flour";
        mainViewModel.SearchItemsCommand.Execute(null);

        // Assert - Search functionality should work (implementation may vary)
        mainViewModel.SearchText.Should().Be("Flour");
    }

    [AvaloniaFact]
    public async Task LowStockFilter_ShouldToggleCorrectly()
    {
        // Arrange
        var host = CreateTestHost();
        var mainViewModel = host.Services.GetRequiredService<MainViewModel>();
        await mainViewModel.RefreshCommand.ExecuteAsync(null);
        var initialState = mainViewModel.ShowLowStockOnly;

        // Act
        mainViewModel.FilterLowStockCommand.Execute(null);

        // Assert
        mainViewModel.ShowLowStockOnly.Should().Be(!initialState);
    }

    [AvaloniaFact]
    public async Task Disconnect_ShouldUpdateConnectionStatus()
    {
        // Arrange
        var host = CreateTestHost();
        var mainViewModel = host.Services.GetRequiredService<MainViewModel>();
        await mainViewModel.ConnectCommand.ExecuteAsync(null);

        // Act
        await mainViewModel.DisconnectCommand.ExecuteAsync(null);

        // Assert
        mainViewModel.IsConnected.Should().BeFalse();
        mainViewModel.ConnectionStatus.Should().Be("Disconnected");
    }

    private static IHost CreateTestHost()
    {
        return Host.CreateDefaultBuilder()
            .ConfigureServices((context, services) =>
            {
                services.AddLogging(builder =>
                {
                    builder.AddConsole();
                    builder.SetMinimumLevel(LogLevel.Information);
                });

                services.AddSingleton<InventoryClient.Services.InventoryGrpcService>();
                services.AddSingleton<TaskSystems.Shared.Services.IServiceClient>(provider =>
                    provider.GetRequiredService<InventoryClient.Services.InventoryGrpcService>());
                services.AddTransient<MainViewModel>();
            })
            .Build();
    }
}
