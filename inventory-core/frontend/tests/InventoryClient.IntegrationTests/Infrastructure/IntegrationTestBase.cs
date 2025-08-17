using System;
using System.Threading.Tasks;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using InventoryClient.Services;
using TaskSystems.Shared.Services;
using Xunit;

namespace InventoryClient.IntegrationTests.Infrastructure;

/// <summary>
/// Base class for integration tests that need a managed backend server
/// </summary>
public abstract class IntegrationTestBase : IAsyncLifetime, IDisposable
{
    protected IHost Host { get; private set; } = null!;
    protected BackendServerManager ServerManager { get; private set; } = null!;
    protected IInventoryService InventoryService { get; private set; } = null!;
    protected ILogger Logger { get; private set; } = null!;
    protected int ServerPort { get; private set; }
    protected string ServerAddress => BackendServerManager.GetServerAddress(ServerPort);

    /// <summary>
    /// Override to specify whether the test needs persistent storage
    /// Default is false (in-memory storage)
    /// </summary>
    protected virtual bool UsePersistentStorage => false;

    /// <summary>
    /// Override to configure additional services for the test
    /// </summary>
    protected virtual void ConfigureServices(IServiceCollection services)
    {
        // Default implementation - can be overridden by derived classes
    }

    public virtual async Task InitializeAsync()
    {
        // Build host with DI container
        Host = CreateHost();

        // Get server manager and start backend
        ServerManager = Host.Services.GetRequiredService<BackendServerManager>();
        ServerPort = await ServerManager.StartServerAsync(UsePersistentStorage);

        // Get services
        InventoryService = Host.Services.GetRequiredService<IInventoryService>();
        Logger = Host.Services.GetRequiredService<ILogger<IntegrationTestBase>>();

        Logger.LogInformation("Test initialized with backend server on port {Port}", ServerPort);
    }

    public virtual async Task DisposeAsync()
    {
        if (ServerManager != null)
        {
            await ServerManager.StopServerAsync(ServerPort);
        }

        Host?.Dispose();
        ServerManager?.Dispose();
    }

    public void Dispose()
    {
        Dispose(true);
        GC.SuppressFinalize(this);
    }

    protected virtual void Dispose(bool disposing)
    {
        if (disposing)
        {
            try
            {
                DisposeAsync().GetAwaiter().GetResult();
            }
            catch
            {
                // Ignore errors during disposal
            }
        }
    }

    private IHost CreateHost()
    {
        return Microsoft.Extensions.Hosting.Host.CreateDefaultBuilder()
            .ConfigureServices((context, services) =>
            {
                // Add logging
                services.AddLogging(builder =>
                {
                    builder.AddConsole();
                    builder.SetMinimumLevel(LogLevel.Information);
                });

                // Add backend server manager
                services.AddSingleton<BackendServerManager>();

                // Add settings service (isolated per test)
                services.AddSingleton<ISettingsService>(provider =>
                {
                    var settingsService = new JsonSettingsService();
                    // Don't load existing settings - start fresh for each test
                    return settingsService;
                });

                // Add inventory services
                services.AddSingleton<InventoryGrpcService>();
                services.AddSingleton<IServiceClient>(provider =>
                    provider.GetRequiredService<InventoryGrpcService>());

                // Add cached inventory service
                services.AddSingleton<IInventoryService>(provider =>
                {
                    var grpcService = provider.GetRequiredService<InventoryGrpcService>();
                    var settingsService = provider.GetRequiredService<ISettingsService>();
                    var logger = provider.GetRequiredService<ILogger<CachedInventoryService>>();
                    return new CachedInventoryService(grpcService, settingsService, logger);
                });

                // Allow derived classes to configure additional services
                ConfigureServices(services);
            })
            .Build();
    }
}
