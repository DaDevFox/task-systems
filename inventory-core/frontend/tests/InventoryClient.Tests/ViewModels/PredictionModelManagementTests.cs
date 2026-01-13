using InventoryClient.Models;
using InventoryClient.ViewModels;
using InventoryClient.Services;
using Microsoft.Extensions.Logging;
using Moq;
using Xunit;

namespace InventoryClient.Tests.ViewModels;

public class PredictionModelManagementTests
{
    private readonly Mock<InventoryGrpcService> _mockService;
    private readonly Mock<ILogger<MainViewModel>> _mockLogger;
    private readonly MainViewModel _viewModel;

    public PredictionModelManagementTests()
    {
        _mockService = new Mock<InventoryGrpcService>(null!, null!);
        _mockLogger = new Mock<ILogger<MainViewModel>>();
        _viewModel = new MainViewModel(_mockService.Object, _mockLogger.Object);
    }

    [Fact]
    public void SelectedItem_WhenChanged_ShouldCreatePredictionStatus()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            Id = "test-item",
            Name = "Test Item",
            Description = "Test Description",
            CurrentLevel = 5.0,
            MaxCapacity = 10.0
        };

        // Act
        _viewModel.SelectedItem = item;

        // Assert
        Assert.NotNull(_viewModel.SelectedItemPredictionStatus);
        Assert.Equal(item.Id, _viewModel.SelectedItemPredictionStatus.ItemId);
        Assert.True(_viewModel.IsPredictionModelSelected);
    }

    [Fact]
    public void SelectedItem_WhenSetToNull_ShouldClearPredictionStatus()
    {
        // Arrange
        var item = new InventoryItemViewModel { Id = "test-item", Name = "Test Item" };
        _viewModel.SelectedItem = item;

        // Act
        _viewModel.SelectedItem = null;

        // Assert
        Assert.Null(_viewModel.SelectedItemPredictionStatus);
        Assert.False(_viewModel.IsPredictionModelSelected);
    }

    [Fact]
    public void PredictionStatus_WhenCreated_ShouldHaveValidDefaults()
    {
        // Arrange
        var item = new InventoryItemViewModel { Id = "test-item", Name = "Test Item" };

        // Act
        _viewModel.SelectedItem = item;
        var status = _viewModel.SelectedItemPredictionStatus;

        // Assert
        Assert.NotNull(status);
        Assert.Equal(TrainingStage.Trained, status.Stage);
        Assert.Equal(PredictionModel.Bayesian, status.ActiveModel);
        Assert.NotEmpty(status.AvailableModels);
        Assert.True(status.TrainingSamples >= status.MinSamplesRequired);
        Assert.NotEmpty(status.ModelParameters);
    }

    [Fact]
    public async Task StartTrainingCommand_WhenExecuted_ShouldUpdateTrainingStage()
    {
        // Arrange
        var item = new InventoryItemViewModel { Id = "test-item", Name = "Test Item" };
        _viewModel.SelectedItem = item;
        _viewModel.IsConnected = true;

        // Act
        await _viewModel.StartTrainingCommand.ExecuteAsync(null);

        // Assert
        Assert.Equal(TrainingStage.Learning, _viewModel.SelectedItemPredictionStatus?.Stage);
        Assert.True(_viewModel.SelectedItemPredictionStatus?.TrainingStarted <= DateTime.Now);
        Assert.True(_viewModel.SelectedItemPredictionStatus?.LastUpdated <= DateTime.Now);
    }

    [Fact]
    public async Task RefreshPredictionStatusCommand_WhenExecuted_ShouldUpdateLastUpdated()
    {
        // Arrange
        var item = new InventoryItemViewModel { Id = "test-item", Name = "Test Item" };
        _viewModel.SelectedItem = item;
        _viewModel.IsConnected = true;
        var originalLastUpdated = _viewModel.SelectedItemPredictionStatus?.LastUpdated;

        // Act
        await Task.Delay(100); // Ensure time difference
        await _viewModel.RefreshPredictionStatusCommand.ExecuteAsync(null);

        // Assert
        Assert.True(_viewModel.SelectedItemPredictionStatus?.LastUpdated > originalLastUpdated);
    }

    [Fact]
    public async Task ApplyModelConfigurationCommand_WhenExecuted_ShouldUpdateLastUpdated()
    {
        // Arrange
        var item = new InventoryItemViewModel { Id = "test-item", Name = "Test Item" };
        _viewModel.SelectedItem = item;
        _viewModel.IsConnected = true;
        var originalLastUpdated = _viewModel.SelectedItemPredictionStatus?.LastUpdated;

        // Act
        await Task.Delay(100); // Ensure time difference
        await _viewModel.ApplyModelConfigurationCommand.ExecuteAsync(null);

        // Assert
        Assert.True(_viewModel.SelectedItemPredictionStatus?.LastUpdated > originalLastUpdated);
    }

    [Fact]
    public async Task PredictionCommands_WhenNotConnected_ShouldSetConnectionError()
    {
        // Arrange
        var item = new InventoryItemViewModel { Id = "test-item", Name = "Test Item" };
        _viewModel.SelectedItem = item;
        _viewModel.IsConnected = false;

        // Act & Assert
        await _viewModel.StartTrainingCommand.ExecuteAsync(null);
        Assert.True(_viewModel.HasConnectionError);
        Assert.Contains("not connected", _viewModel.ConnectionError.ToLower());

        _viewModel.ClearConnectionErrorCommand.Execute(null);

        await _viewModel.RefreshPredictionStatusCommand.ExecuteAsync(null);
        Assert.True(_viewModel.HasConnectionError);

        _viewModel.ClearConnectionErrorCommand.Execute(null);

        await _viewModel.ApplyModelConfigurationCommand.ExecuteAsync(null);
        Assert.True(_viewModel.HasConnectionError);
    }

    [Fact]
    public async Task PredictionCommands_WhenNoSelectedItem_ShouldNotThrow()
    {
        // Arrange
        _viewModel.SelectedItem = null;
        _viewModel.IsConnected = true;

        // Act & Assert - Should not throw
        await _viewModel.StartTrainingCommand.ExecuteAsync(null);
        await _viewModel.RefreshPredictionStatusCommand.ExecuteAsync(null);
        await _viewModel.ApplyModelConfigurationCommand.ExecuteAsync(null);
    }
}
