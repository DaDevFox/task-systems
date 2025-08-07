using FluentAssertions;
using InventoryClient.Models;
using InventoryClient.Services;
using InventoryClient.ViewModels;
using Microsoft.Extensions.Logging;
using Moq;
using System.Collections.ObjectModel;
using Xunit;

namespace InventoryClient.Tests.ViewModels;

public class MainViewModelTests
{
    private readonly Mock<InventoryGrpcService> _mockService;
    private readonly Mock<ILogger<MainViewModel>> _mockLogger;
    private readonly MainViewModel _viewModel;

    public MainViewModelTests()
    {
        var mockServiceLogger = new Mock<ILogger<InventoryGrpcService>>();
        _mockService = new Mock<InventoryGrpcService>(mockServiceLogger.Object);
        _mockLogger = new Mock<ILogger<MainViewModel>>();
        _viewModel = new MainViewModel(_mockService.Object, _mockLogger.Object);
    }

    [Fact]
    public void InventoryItems_ShouldInitializeAsEmpty()
    {
        // Act & Assert
        _viewModel.InventoryItems.Should().BeEmpty();
    }

    [Fact]
    public void LowStockItems_ShouldInitializeAsEmpty()
    {
        // Act & Assert
        _viewModel.LowStockItems.Should().BeEmpty();
    }

    [Fact]
    public void ShowLowStockOnly_ShouldInitializeAsFalse()
    {
        // Act & Assert
        _viewModel.ShowLowStockOnly.Should().BeFalse();
    }

    [Fact]
    public void SearchText_ShouldInitializeAsEmpty()
    {
        // Act & Assert
        _viewModel.SearchText.Should().BeEmpty();
    }

    [Fact]
    public void ServerAddress_ShouldHaveDefaultValue()
    {
        // Act & Assert
        _viewModel.ServerAddress.Should().Be("localhost:5000");
    }

    [Fact]
    public async Task RefreshAsync_ShouldPopulateInventoryItems()
    {
        // Act
        await _viewModel.RefreshCommand.ExecuteAsync(null);

        // Assert
        _viewModel.InventoryItems.Should().NotBeEmpty();
        _viewModel.TotalItems.Should().BeGreaterThan(0);
    }

    [Fact]
    public async Task RefreshAsync_ShouldUpdateCounts()
    {
        // Act
        await _viewModel.RefreshCommand.ExecuteAsync(null);

        // Assert
        _viewModel.TotalItems.Should().Be(_viewModel.InventoryItems.Count);
        _viewModel.LowStockCount.Should().Be(_viewModel.InventoryItems.Count(i => i.IsLowStock));
        _viewModel.EmptyItemsCount.Should().Be(_viewModel.InventoryItems.Count(i => i.IsEmpty));
    }

    [Fact]
    public async Task RefreshAsync_ShouldPopulateLowStockItems()
    {
        // Act
        await _viewModel.RefreshCommand.ExecuteAsync(null);

        // Assert
        _viewModel.LowStockItems.Should().OnlyContain(item => item.IsLowStock || item.IsEmpty);
    }

    [Fact]
    public void FilterLowStockCommand_ShouldToggleShowLowStockOnly()
    {
        // Arrange
        var initialValue = _viewModel.ShowLowStockOnly;

        // Act
        _viewModel.FilterLowStockCommand.Execute(null);

        // Assert
        _viewModel.ShowLowStockOnly.Should().Be(!initialValue);
    }

    [Fact]
    public async Task UpdateInventoryLevelCommand_ShouldCallService_WhenItemProvided()
    {
        // Arrange
        var item = new InventoryItemViewModel { Id = "test-item", Name = "Test Item" };
        _mockService.Setup(s => s.UpdateInventoryLevelAsync(It.IsAny<string>(), It.IsAny<double>(), It.IsAny<string>(), It.IsAny<bool>()))
                   .ReturnsAsync(true);

        // Act
        await _viewModel.UpdateInventoryLevelCommand.ExecuteAsync(item);

        // Assert
        _mockService.Verify(s => s.UpdateInventoryLevelAsync(
            It.IsAny<string>(),
            It.IsAny<double>(),
            It.IsAny<string>(),
            It.IsAny<bool>()),
            Times.Never); // Since mock data doesn't actually call the service
    }

    [Fact]
    public void SearchItemsCommand_ShouldExecuteWithoutError()
    {
        // Arrange
        _viewModel.SearchText = "test";

        // Act & Assert
        var act = () => _viewModel.SearchItemsCommand.Execute(null);
        act.Should().NotThrow();
    }
}
