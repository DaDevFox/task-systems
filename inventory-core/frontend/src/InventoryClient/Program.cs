using Avalonia;
using System;

namespace InventoryClient;

internal static class Program
{
    [STAThread]
    public static void Main(string[] args)
    {
        // Ensure console is available for debugging
        if (System.Diagnostics.Debugger.IsAttached)
        {
            // Allocate console for debug output if we're debugging
            AllocConsole();
        }

        Console.WriteLine($"[{DateTime.Now:HH:mm:ss.fff}] InventoryClient starting...");

        BuildAvaloniaApp()
            .StartWithClassicDesktopLifetime(args);
    }

    public static AppBuilder BuildAvaloniaApp()
        => AppBuilder.Configure<App>()
            .UsePlatformDetect()
            .WithInterFont();

    [System.Runtime.InteropServices.DllImport("kernel32.dll", SetLastError = true)]
    [return: System.Runtime.InteropServices.MarshalAs(System.Runtime.InteropServices.UnmanagedType.Bool)]
    private static extern bool AllocConsole();
}
