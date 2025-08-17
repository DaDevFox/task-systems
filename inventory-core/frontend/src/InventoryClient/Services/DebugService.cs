using System;
using System.Diagnostics;
using System.IO;

namespace InventoryClient.Services;

/// <summary>
/// Debug service that provides multiple debugging outputs to ensure messages are seen
/// </summary>
public static class DebugService
{
    private static readonly object _lockObject = new();
    private static readonly string _logFilePath = InitializeLogFile();

    private static string InitializeLogFile()
    {
        var tempDir = Path.GetTempPath();
        var logFile = Path.Combine(tempDir, $"InventoryClient_Debug_{DateTime.Now:yyyyMMdd_HHmmss}.log");

        // Write initial log entries
        var initialMessage = $"[{DateTime.Now:HH:mm:ss.fff}] DebugService initialized{Environment.NewLine}";
        initialMessage += $"[{DateTime.Now:HH:mm:ss.fff}] Debug log file: {logFile}{Environment.NewLine}";
        initialMessage += $"[{DateTime.Now:HH:mm:ss.fff}] Debugger attached: {Debugger.IsAttached}{Environment.NewLine}";

        try
        {
            File.WriteAllText(logFile, initialMessage);
        }
        catch (Exception ex)
        {
            Debug.WriteLine($"Failed to initialize log file: {ex.Message}");
        }

        return logFile;
    }

    public static void LogDebug(string message, params object[] args)
    {
        var formattedMessage = args.Length > 0 ? string.Format(message, args) : message;
        var timestampedMessage = $"[{DateTime.Now:HH:mm:ss.fff}] {formattedMessage}";

        lock (_lockObject)
        {
            // Console output
            try
            {
                Console.WriteLine(timestampedMessage);
            }
            catch (Exception ex)
            {
                // Ignore console errors to prevent crashes
                Debug.WriteLine($"Console output failed: {ex.Message}");
            }

            // Debug output (visible in VS Output window when debugging)
            Debug.WriteLine(timestampedMessage);

            // Trace output
            try
            {
                Trace.TraceInformation(timestampedMessage);
            }
            catch (Exception ex)
            {
                // Ignore trace errors to prevent crashes  
                Debug.WriteLine($"Trace output failed: {ex.Message}");
            }

            // File output for persistent debugging
            try
            {
                File.AppendAllText(_logFilePath, timestampedMessage + Environment.NewLine);
            }
            catch (Exception ex)
            {
                // Ignore file write errors to prevent crashes
                Debug.WriteLine($"File output failed: {ex.Message}");
            }
        }
    }

    public static void LogError(string message, Exception? exception = null, params object[] args)
    {
        var formattedMessage = args.Length > 0 ? string.Format(message, args) : message;

        if (exception != null)
        {
            formattedMessage += $" Exception: {exception}";
        }

        LogDebug($"ERROR: {formattedMessage}");
    }

    public static string GetLogFilePath()
    {
        return _logFilePath;
    }

    /// <summary>
    /// Opens the debug log file in the default text editor
    /// </summary>
    public static void OpenLogFile()
    {
        try
        {
            if (File.Exists(_logFilePath))
            {
                Process.Start(new ProcessStartInfo
                {
                    FileName = _logFilePath,
                    UseShellExecute = true
                });
                LogDebug("Opened log file: {0}", _logFilePath);
            }
            else
            {
                LogDebug("Log file not found: {0}", _logFilePath);
            }
        }
        catch (Exception ex)
        {
            LogDebug("Failed to open log file: {0}", ex.Message);
        }
    }
}
