using Microsoft.Extensions.Logging;

namespace TaskSystems.Shared.Services;

/// <summary>
/// Base interface for all gRPC service clients across the task systems
/// </summary>
public interface IServiceClient
{
    string ServiceName { get; }
    bool IsConnected { get; }
    Task<bool> ConnectAsync(string address, CancellationToken cancellationToken = default);
    Task DisconnectAsync();
    event EventHandler<bool> ConnectionStatusChanged;
}

/// <summary>
/// Base implementation for gRPC service clients with common functionality
/// </summary>
public abstract class ServiceClientBase : IServiceClient, IDisposable
{
    protected readonly ILogger Logger;
    private bool _isConnected;
    private bool _disposed;

    protected ServiceClientBase(ILogger logger)
    {
        Logger = logger;
    }

    public abstract string ServiceName { get; }

    public bool IsConnected
    {
        get => _isConnected;
        protected set
        {
            if (_isConnected != value)
            {
                _isConnected = value;
                ConnectionStatusChanged?.Invoke(this, value);
            }
        }
    }

    public event EventHandler<bool>? ConnectionStatusChanged;

    public abstract Task<bool> ConnectAsync(string address, CancellationToken cancellationToken = default);
    
    public virtual Task DisconnectAsync()
    {
        IsConnected = false;
        return Task.CompletedTask;
    }

    protected virtual void Dispose(bool disposing)
    {
        if (!_disposed)
        {
            if (disposing)
            {
                _ = DisconnectAsync();
            }
            _disposed = true;
        }
    }

    public void Dispose()
    {
        Dispose(disposing: true);
        GC.SuppressFinalize(this);
    }
}
