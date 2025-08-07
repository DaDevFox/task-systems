using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using Microsoft.Extensions.Logging;
using TaskSystems.Shared.Services;

namespace TaskSystems.Shared.ViewModels;

/// <summary>
/// Base view model providing common functionality for all service views
/// </summary>
public abstract partial class ServiceViewModelBase : ObservableObject
{
    protected readonly ILogger Logger;
    private readonly IServiceClient _serviceClient;

    [ObservableProperty]
    private bool _isLoading;

    [ObservableProperty]
    private bool _isConnected;

    [ObservableProperty]
    private string _connectionStatus = "Disconnected";

    [ObservableProperty]
    private string _lastError = string.Empty;

    [ObservableProperty]
    private string _serverAddress = "localhost:50052";

    protected ServiceViewModelBase(IServiceClient serviceClient, ILogger logger)
    {
        _serviceClient = serviceClient;
        Logger = logger;

        _serviceClient.ConnectionStatusChanged += OnConnectionStatusChanged;
    }

    [RelayCommand]
    private async Task ConnectAsync()
    {
        try
        {
            IsLoading = true;
            LastError = string.Empty;

            var success = await _serviceClient.ConnectAsync(ServerAddress);
            if (!success)
            {
                LastError = $"Failed to connect to {_serviceClient.ServiceName} service";
            }
        }
        catch (Exception ex)
        {
            LastError = $"Connection error: {ex.Message}";
            Logger.LogError(ex, "Failed to connect to service");
        }
        finally
        {
            IsLoading = false;
        }
    }

    [RelayCommand]
    private async Task DisconnectAsync()
    {
        try
        {
            await _serviceClient.DisconnectAsync();
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "Error disconnecting from service");
        }
    }

    [RelayCommand]
    protected virtual async Task RefreshAsync()
    {
        if (!IsConnected) return;

        try
        {
            IsLoading = true;
            await RefreshDataAsync();
        }
        catch (Exception ex)
        {
            LastError = $"Refresh error: {ex.Message}";
            Logger.LogError(ex, "Failed to refresh data");
        }
        finally
        {
            IsLoading = false;
        }
    }

    protected abstract Task RefreshDataAsync();

    protected void ClearError()
    {
        LastError = string.Empty;
    }

    private void OnConnectionStatusChanged(object? sender, bool isConnected)
    {
        IsConnected = isConnected;
        ConnectionStatus = isConnected ? "Connected" : "Disconnected";

        if (isConnected)
        {
            _ = RefreshAsync();
        }
    }
}
