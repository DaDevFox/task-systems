using System;
using System.Threading.Tasks;
using FluentAssertions;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using InventoryClient.IntegrationTests.Infrastructure;
using Xunit;
using Xunit.Abstractions;

namespace InventoryClient.IntegrationTests;

/// <summary>
/// Simple test to verify the backend server manager can start and stop servers
/// </summary>
public class BackendServerStartupTests : IDisposable
{
    private readonly ServiceProvider _serviceProvider;
    private readonly ILogger<BackendServerStartupTests> _logger;

    public BackendServerStartupTests(ITestOutputHelper testOutput)
    {
        var services = new ServiceCollection();
        services.AddLogging(builder =>
        {
            builder.AddConsole();
            builder.SetMinimumLevel(LogLevel.Debug);
        });
        services.AddSingleton<BackendServerManager>();

        _serviceProvider = services.BuildServiceProvider();
        _logger = _serviceProvider.GetRequiredService<ILogger<BackendServerStartupTests>>();
    }

    [Fact(Timeout = 30000)] // 30 second hard timeout
    public async Task ServerManager_ShouldStartAndStopServerQuickly()
    {
        // Arrange
        var serverManager = _serviceProvider.GetRequiredService<BackendServerManager>();
        var startTime = DateTime.UtcNow;

        _logger.LogInformation("=== Starting backend server startup test ===");

        try
        {
            // Act - Start server (should timeout quickly if it fails)
            _logger.LogInformation("Starting server...");
            var port = await serverManager.StartServerAsync(persistent: false);
            var startDuration = (DateTime.UtcNow - startTime).TotalMilliseconds;

            _logger.LogInformation("Server started on port {Port} in {Duration}ms", port, startDuration);

            // Verify server is accessible
            port.Should().BeGreaterThan(50000, "Port should be in expected range");

            // Stop server
            _logger.LogInformation("Stopping server...");
            var stopTime = DateTime.UtcNow;
            await serverManager.StopServerAsync(port);
            var stopDuration = (DateTime.UtcNow - stopTime).TotalMilliseconds;

            _logger.LogInformation("Server stopped in {Duration}ms", stopDuration);

            var totalDuration = (DateTime.UtcNow - startTime).TotalMilliseconds;
            _logger.LogInformation("=== Test completed successfully in {TotalDuration}ms ===", totalDuration);

            // Ensure the test doesn't take too long
            totalDuration.Should().BeLessThan(25000, "Test should complete within 25 seconds");
        }
        catch (Exception ex)
        {
            var failDuration = (DateTime.UtcNow - startTime).TotalMilliseconds;
            _logger.LogError(ex, "=== Test failed after {Duration}ms ===", failDuration);

            // Make sure we don't wait forever
            failDuration.Should().BeLessThan(30000, "Test should fail quickly, not hang");
            throw;
        }
    }

    public void Dispose()
    {
        _serviceProvider?.Dispose();
    }
}
