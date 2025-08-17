using Avalonia;
using Avalonia.Controls.ApplicationLifetimes;
using Avalonia.Markup.Xaml;
using InventoryClient.Services;
using InventoryClient.ViewModels;
using InventoryClient.Views;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using TaskSystems.Shared.Services;

namespace InventoryClient;

public partial class App : Application
{
    private IHost? _host;

    public override void Initialize()
    {
        AvaloniaXamlLoader.Load(this);
    }

    public override void OnFrameworkInitializationCompleted()
    {
        // Configure dependency injection
        _host = Host.CreateDefaultBuilder()
            .ConfigureServices((context, services) =>
            {
                // Register logging
                services.AddLogging(builder =>
                {
                    builder.AddConsole();
                    builder.SetMinimumLevel(LogLevel.Information);
                });

                // Register settings service
                services.AddSingleton<ISettingsService>(provider =>
                {
                    var settingsService = new JsonSettingsService();
                    // Load settings synchronously during startup
                    settingsService.LoadAsync().Wait();
                    return settingsService;
                });

                // Register inventory services
                services.AddSingleton<InventoryGrpcService>();
                services.AddSingleton<IServiceClient>(provider => provider.GetRequiredService<InventoryGrpcService>());

                // Register cached inventory service
                services.AddSingleton<IInventoryService>(provider =>
                {
                    var grpcService = provider.GetRequiredService<InventoryGrpcService>();
                    var settingsService = provider.GetRequiredService<ISettingsService>();
                    var logger = provider.GetRequiredService<ILogger<CachedInventoryService>>();
                    return new CachedInventoryService(grpcService, settingsService, logger);
                });

                // Register view models
                services.AddTransient<MainViewModel>(provider =>
                {
                    var inventoryService = provider.GetRequiredService<IInventoryService>();
                    var serviceClient = provider.GetRequiredService<IServiceClient>();
                    var settingsService = provider.GetRequiredService<ISettingsService>();
                    var logger = provider.GetRequiredService<ILogger<MainViewModel>>();
                    return new MainViewModel(inventoryService, serviceClient, settingsService, logger);
                });
            })
            .Build();

        // Get the main view model from DI container
        var mainViewModel = _host.Services.GetRequiredService<MainViewModel>();

        if (ApplicationLifetime is IClassicDesktopStyleApplicationLifetime desktop)
        {
            desktop.MainWindow = new MainWindow
            {
                DataContext = mainViewModel
            };

            // Save settings on application shutdown
            desktop.ShutdownRequested += async (sender, e) =>
            {
                var settingsService = _host.Services.GetRequiredService<ISettingsService>();
                try
                {
                    await settingsService.SaveAsync();
                    DebugService.LogDebug("Settings saved on application shutdown");
                }
                catch (Exception ex)
                {
                    DebugService.LogError("Failed to save settings on shutdown", ex);
                }
            };
        }

        base.OnFrameworkInitializationCompleted();
    }
}
