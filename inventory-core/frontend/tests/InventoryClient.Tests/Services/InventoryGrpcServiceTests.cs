using FluentAssertions;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using Moq;
using Xunit;

namespace InventoryClient.Tests.Services;

public class InventoryGrpcServiceTests
{
    private readonly Mock<ILogger<InventoryGrpcService>> _mockLogger;
    private readonly InventoryGrpcService _service;

    public InventoryGrpcServiceTests()
    {
        _mockLogger = new Mock<ILogger<InventoryGrpcService>>();
        _service = new InventoryGrpcService(_mockLogger.Object);
    }

    [Fact]
    public void ServiceName_ShouldReturnInventory()
    {
        // Act & Assert
        _service.ServiceName.Should().Be("Inventory");
    }

    [Fact]
    public void IsConnected_ShouldInitiallyBeFalse()
    {
        // Act & Assert
        _service.IsConnected.Should().BeFalse();
    }

    [Fact]
    public async Task ConnectAsync_ShouldReturnTrue_AndSetConnectedState()
    {
        // Act
        var result = await _service.ConnectAsync("localhost:5000");

        // Assert
        result.Should().BeTrue();
        _service.IsConnected.Should().BeTrue();
    }

    [Fact]
    public async Task ConnectAsync_ShouldRaiseConnectionStatusChangedEvent()
    {
        // Arrange
        var eventRaised = false;
        _service.ConnectionStatusChanged += (_, connected) =>
        {
            eventRaised = connected;
        };

        // Act
        await _service.ConnectAsync("localhost:5000");

        // Assert
        eventRaised.Should().BeTrue();
    }

    [Fact]
    public async Task DisconnectAsync_ShouldSetConnectedStateToFalse()
    {
        // Arrange
        await _service.ConnectAsync("localhost:5000");

        // Act
        await _service.DisconnectAsync();

        // Assert
        _service.IsConnected.Should().BeFalse();
    }

    [Fact]
    public async Task UpdateInventoryLevelAsync_ShouldReturnTrue_WhenConnected()
    {
        // Arrange
        await _service.ConnectAsync("localhost:5000");

        // Act
        var result = await _service.UpdateInventoryLevelAsync("test-item", 25.0, "Test update");

        // Assert
        result.Should().BeTrue();
    }

    [Fact]
    public async Task UpdateInventoryLevelAsync_ShouldLogInformation()
    {
        // Arrange
        await _service.ConnectAsync("localhost:5000");

        // Act
        await _service.UpdateInventoryLevelAsync("test-item", 25.0, "Test update");

        // Assert
        _mockLogger.Verify(
            x => x.Log(
                LogLevel.Information,
                It.IsAny<EventId>(),
                It.Is<It.IsAnyType>((v, t) => v.ToString()!.Contains("Updated inventory level")),
                It.IsAny<Exception>(),
                It.IsAny<Func<It.IsAnyType, Exception?, string>>()),
            Times.Once);
    }

    [Fact]
    public void Dispose_ShouldNotThrow()
    {
        // Act & Assert
        var act = () => _service.Dispose();
        act.Should().NotThrow();
    }
}
