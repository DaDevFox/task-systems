using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Net.NetworkInformation;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.Extensions.Logging;

namespace InventoryClient.IntegrationTests.Infrastructure;

/// <summary>
/// Manages backend server instances for integration tests
/// </summary>
public class BackendServerManager : IDisposable
{
    private readonly ILogger<BackendServerManager> _logger;
    private readonly Dictionary<int, Process> _runningServers = new();
    private readonly HashSet<string> _temporaryExecutables = new();
    private readonly object _lock = new();
    private bool _disposed = false;

    public BackendServerManager(ILogger<BackendServerManager> logger)
    {
        _logger = logger;
    }

    /// <summary>
    /// Starts a backend server on an available port
    /// </summary>
    /// <param name="persistent">If true, uses a persistent database. If false, uses in-memory storage.</param>
    /// <returns>The port number the server is running on</returns>
    public async Task<int> StartServerAsync(bool persistent = false)
    {
        if (_disposed)
            throw new ObjectDisposedException(nameof(BackendServerManager));

        var port = FindAvailablePort();
        var serverExecutable = GetServerExecutablePath();

        // Track the temporary executable for cleanup
        lock (_lock)
        {
            _temporaryExecutables.Add(serverExecutable);
        }

        _logger.LogInformation("Starting backend server on port {Port} using executable: {Executable}", port, serverExecutable);

        if (!File.Exists(serverExecutable))
        {
            throw new FileNotFoundException($"Backend server executable not found at: {serverExecutable}");
        }

        var startInfo = new ProcessStartInfo
        {
            FileName = serverExecutable,
            UseShellExecute = false,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            CreateNoWindow = true,
            WorkingDirectory = Path.GetDirectoryName(serverExecutable)
        };

        // Clear and set environment variables (don't inherit everything)
        startInfo.Environment.Clear();

        // Copy essential environment variables
        foreach (var key in new[] { "PATH", "SystemRoot", "TEMP", "TMP" })
        {
            var value = Environment.GetEnvironmentVariable(key);
            if (!string.IsNullOrEmpty(value))
            {
                startInfo.Environment[key] = value;
            }
        }

        // Set our specific environment variables for the backend server
        startInfo.Environment["INVENTORY_PORT"] = port.ToString();
        if (persistent)
        {
            // Use a unique persistent database path for this server instance
            var dbPath = Path.Combine(Path.GetTempPath(), $"inventory_test_db_{port}");
            startInfo.Environment["INVENTORY_DB_PATH"] = dbPath;
            _logger.LogDebug("Using persistent database path: {DbPath}", dbPath);
        }
        else
        {
            // For non-persistent, we'll still create a temp path but it will be isolated
            var dbPath = Path.Combine(Path.GetTempPath(), $"inventory_test_memory_{port}");
            startInfo.Environment["INVENTORY_DB_PATH"] = dbPath;
            _logger.LogDebug("Using temporary database path: {DbPath}", dbPath);
        }

        var process = new Process { StartInfo = startInfo };

        // Set up logging for server output
        process.OutputDataReceived += (sender, e) =>
        {
            if (!string.IsNullOrEmpty(e.Data))
                _logger.LogDebug("Backend[{Port}] OUT: {Data}", port, e.Data);
        };

        process.ErrorDataReceived += (sender, e) =>
        {
            if (!string.IsNullOrEmpty(e.Data))
                _logger.LogWarning("Backend[{Port}] ERR: {Data}", port, e.Data);
        };

        process.Exited += (sender, e) =>
        {
            _logger.LogInformation("Backend server on port {Port} has exited", port);
            lock (_lock)
            {
                _runningServers.Remove(port);
            }
        };

        process.EnableRaisingEvents = true;

        try
        {
            var dbPath = startInfo.Environment.TryGetValue("INVENTORY_DB_PATH", out var path) ? path : "not set";
            _logger.LogDebug("Starting process: {FileName} with PORT={Port}, DB_PATH={DbPath}",
                serverExecutable, port, dbPath);

            // Log all environment variables for debugging
            _logger.LogDebug("Environment variables: {EnvVars}",
                string.Join(", ", startInfo.Environment.Select(kv => $"{kv.Key}={kv.Value}")));

            process.Start();
            process.BeginOutputReadLine();
            process.BeginErrorReadLine();

            _logger.LogDebug("Process started with PID: {ProcessId}", process.Id);

            lock (_lock)
            {
                _runningServers[port] = process;
            }

            // Give the process a moment to start and log any immediate output
            await Task.Delay(2000);

            // Check if process is still running
            if (process.HasExited)
            {
                var exitCode = process.ExitCode;
                _logger.LogError("Backend server process exited immediately with code {ExitCode}", exitCode);
                throw new InvalidOperationException($"Backend server process exited immediately with code {exitCode}");
            }

            _logger.LogDebug("Process is still running after 2s, checking server readiness...");

            // Wait for server to be ready with very short timeout
            await WaitForServerToBeReady($"localhost:{port}");

            _logger.LogInformation("Started backend server on port {Port} (PID: {ProcessId}, Persistent: {Persistent})",
                port, process.Id, persistent);

            return port;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to start backend server on port {Port}", port);

            if (!process.HasExited)
            {
                process.Kill();
            }
            process.Dispose();
            throw;
        }
    }

    /// <summary>
    /// Stops a specific server instance
    /// </summary>
    public async Task StopServerAsync(int port)
    {
        Process? process = null;

        lock (_lock)
        {
            _runningServers.TryGetValue(port, out process);
        }

        if (process != null && !process.HasExited)
        {
            try
            {
                _logger.LogInformation("Stopping backend server on port {Port} (PID: {ProcessId})", port, process.Id);

                // For Go servers, CloseMainWindow doesn't work well, so kill directly
                process.Kill();

                // Wait briefly for process to exit
                using var cts = new CancellationTokenSource(1000); // 1 second timeout
                try
                {
                    await Task.Run(() => process.WaitForExit(), cts.Token);
                    _logger.LogInformation("Backend server on port {Port} stopped successfully", port);
                }
                catch (OperationCanceledException)
                {
                    _logger.LogWarning("Backend server on port {Port} did not exit within 1 second after kill", port);
                    // Process may be zombie, continue cleanup
                }
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Error stopping backend server on port {Port}", port);
            }
            finally
            {
                process.Dispose();
                lock (_lock)
                {
                    _runningServers.Remove(port);
                }
            }
        }
    }

    /// <summary>
    /// Stops all running servers
    /// </summary>
    public async Task StopAllServersAsync()
    {
        var ports = new List<int>();

        lock (_lock)
        {
            ports.AddRange(_runningServers.Keys);
        }

        var tasks = ports.Select(port => StopServerAsync(port));
        await Task.WhenAll(tasks);
    }

    /// <summary>
    /// Gets the address for connecting to a server
    /// </summary>
    public static string GetServerAddress(int port) => $"localhost:{port}";

    private static int FindAvailablePort()
    {
        // Start from a high port to avoid conflicts
        const int startPort = 55000;
        const int endPort = 60000;

        var ipGlobalProperties = IPGlobalProperties.GetIPGlobalProperties();
        var usedPorts = new HashSet<int>();

        // Get all used TCP ports
        foreach (var endpoint in ipGlobalProperties.GetActiveTcpListeners())
        {
            usedPorts.Add(endpoint.Port);
        }

        // Find first available port
        for (int port = startPort; port < endPort; port++)
        {
            if (!usedPorts.Contains(port))
            {
                return port;
            }
        }

        throw new InvalidOperationException($"No available ports found between {startPort} and {endPort}");
    }

    private static string GetServerExecutablePath()
    {
        // Build the backend server on-demand with a temporary executable
        var currentDir = Directory.GetCurrentDirectory();
        var solutionRoot = FindSolutionRoot(currentDir);

        if (solutionRoot == null)
        {
            throw new DirectoryNotFoundException("Could not find solution root directory with backend and frontend folders.");
        }

        var backendDir = Path.Combine(solutionRoot, "backend");
        if (!Directory.Exists(backendDir))
        {
            throw new DirectoryNotFoundException($"Backend directory not found at: {backendDir}");
        }

        var serverMainPath = Path.Combine(backendDir, "cmd", "server");
        if (!Directory.Exists(serverMainPath))
        {
            throw new DirectoryNotFoundException($"Server main directory not found at: {serverMainPath}");
        }

        // Create a temporary executable name unique to this test run
        var tempExecutableName = $"test-inventory-server-{Guid.NewGuid():N}.exe";
        var tempExecutablePath = Path.Combine(backendDir, tempExecutableName);

        // Build the server executable
        var buildProcess = new ProcessStartInfo
        {
            FileName = "go",
            Arguments = $"build -o \"{tempExecutablePath}\" ./cmd/server",
            WorkingDirectory = backendDir,
            UseShellExecute = false,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            CreateNoWindow = true
        };

        using (var process = Process.Start(buildProcess))
        {
            if (process == null)
            {
                throw new InvalidOperationException("Failed to start go build process");
            }

            process.WaitForExit();

            if (process.ExitCode != 0)
            {
                var stdout = process.StandardOutput.ReadToEnd();
                var stderr = process.StandardError.ReadToEnd();
                throw new InvalidOperationException($"Go build failed with exit code {process.ExitCode}. Stdout: {stdout}. Stderr: {stderr}");
            }
        }

        if (!File.Exists(tempExecutablePath))
        {
            throw new FileNotFoundException($"Built executable not found at expected path: {tempExecutablePath}");
        }

        return tempExecutablePath;
    }

    private static string? FindSolutionRoot(string currentPath)
    {
        var directory = new DirectoryInfo(currentPath);

        while (directory != null)
        {
            // Look for characteristic files that indicate the solution root
            if (Directory.Exists(Path.Combine(directory.FullName, "backend")) &&
                Directory.Exists(Path.Combine(directory.FullName, "frontend")))
            {
                return directory.FullName;
            }

            directory = directory.Parent;
        }

        return null;
    }

    private async Task WaitForServerToBeReady(string address, int timeoutMs = 2000)
    {
        _logger.LogInformation("Waiting for backend server at {Address} to be ready (timeout: {TimeoutMs}ms)", address, timeoutMs);

        using var timeoutCts = new CancellationTokenSource(timeoutMs);
        var attempts = 0;
        var startTime = DateTime.UtcNow;

        while (!timeoutCts.Token.IsCancellationRequested)
        {
            attempts++;
            var elapsed = (DateTime.UtcNow - startTime).TotalMilliseconds;

            try
            {
                _logger.LogDebug("Attempt {Attempt} at {Elapsed}ms: Testing connection to {Address}", attempts, elapsed, address);

                // Try to create a gRPC channel and make a simple call
                using var channel = Grpc.Net.Client.GrpcChannel.ForAddress($"http://{address}");
                var client = new Inventory.V1.InventoryService.InventoryServiceClient(channel);

                // Try a simple ping-like operation with a very short timeout
                using var callCts = new CancellationTokenSource(500);  // 500ms max per call

                using var combinedCts = CancellationTokenSource.CreateLinkedTokenSource(
                    timeoutCts.Token, callCts.Token);

                await client.ListInventoryItemsAsync(new Inventory.V1.ListInventoryItemsRequest
                {
                    Limit = 1
                }, cancellationToken: combinedCts.Token);

                _logger.LogInformation("Backend server at {Address} is ready after {Attempts} attempts ({Elapsed}ms)",
                    address, attempts, elapsed);
                return;
            }
            catch (OperationCanceledException) when (timeoutCts.Token.IsCancellationRequested)
            {
                // Overall timeout reached
                _logger.LogError("Timeout reached after {Elapsed}ms and {Attempts} attempts", elapsed, attempts);
                break;
            }
            catch (Exception ex)
            {
                _logger.LogDebug("Attempt {Attempt} at {Elapsed}ms failed: {Error}", attempts, elapsed, ex.Message);
            }

            // Check if we're close to timeout before waiting
            var remainingTime = timeoutMs - elapsed;
            if (remainingTime <= 100)
            {
                _logger.LogError("Timeout approaching, remaining time: {RemainingTime}ms", remainingTime);
                break;
            }

            // Wait before next attempt, but not longer than remaining timeout
            var delayMs = Math.Min(250, (int)remainingTime - 50);
            if (delayMs > 0)
            {
                try
                {
                    await Task.Delay(delayMs, timeoutCts.Token);
                }
                catch (OperationCanceledException)
                {
                    break;
                }
            }
        }

        var totalElapsed = (DateTime.UtcNow - startTime).TotalMilliseconds;
        throw new TimeoutException($"Backend server at {address} did not become ready within {timeoutMs}ms. " +
            $"Made {attempts} attempts over {totalElapsed:F0}ms");
    }

    public void Dispose()
    {
        Dispose(true);
        GC.SuppressFinalize(this);
    }

    protected virtual void Dispose(bool disposing)
    {
        if (_disposed || !disposing)
            return;

        _disposed = true;

        try
        {
            StopAllServersAsync().Wait(10000); // Give 10 seconds for graceful shutdown
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error during BackendServerManager disposal");
        }

        lock (_lock)
        {
            foreach (var process in _runningServers.Values)
            {
                try
                {
                    if (!process.HasExited)
                    {
                        process.Kill();
                    }
                    process.Dispose();
                }
                catch
                {
                    // Ignore errors during cleanup
                }
            }
            _runningServers.Clear();

            // Clean up temporary executables
            foreach (var executable in _temporaryExecutables)
            {
                try
                {
                    if (File.Exists(executable))
                    {
                        _logger.LogInformation("Cleaning up temporary executable: {Executable}", executable);
                        File.Delete(executable);
                    }
                }
                catch (Exception ex)
                {
                    _logger.LogWarning(ex, "Failed to delete temporary executable: {Executable}", executable);
                }
            }
            _temporaryExecutables.Clear();
        }
    }
}
