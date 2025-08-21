using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using FluentAssertions;
using Microsoft.Extensions.Logging;
using Moq;
using InventoryClient.Models;
using InventoryClient.Services;
using InventoryClient.ViewModels;
using TaskSystems.Shared.Services;
using Inventory.V1;
using Xunit;

namespace InventoryClient.Tests.ViewModels;

/// <summary>
/// Unit tests for InventoryLevelChartViewModel focusing on data processing and chart mode behavior
/// </summary>
public class InventoryLevelChartViewModelTests
{
    private readonly Mock<IInventoryService> _mockInventoryService;
    private readonly Mock<ISettingsService> _mockSettingsService;
    private readonly Mock<ILogger<InventoryLevelChartViewModel>> _mockLogger;
    private readonly InventoryLevelChartViewModel _viewModel;

    public InventoryLevelChartViewModelTests()
    {
        _mockInventoryService = new Mock<IInventoryService>();
        _mockSettingsService = new Mock<ISettingsService>();
        _mockLogger = new Mock<ILogger<InventoryLevelChartViewModel>>();

        _viewModel = new InventoryLevelChartViewModel(
            _mockInventoryService.Object,
            _mockSettingsService.Object,
            _mockLogger.Object);
    }

    [Fact]
    public void Constructor_ShouldInitializeWithDefaultValues()
    {
        // Assert
        _viewModel.Item.Should().BeNull();
        _viewModel.HistoricalData.Should().NotBeNull().And.BeEmpty();
        _viewModel.PredictionData.Should().NotBeNull().And.BeEmpty();
        _viewModel.HasAnyData.Should().BeFalse();
        _viewModel.ItemName.Should().Be("No Item Selected");
        _viewModel.CurrentLevelDisplay.Should().BeEmpty();
        _viewModel.PredictionSummary.Should().BeEmpty();
        _viewModel.AvailablePredictionModels.Should().NotBeEmpty();
        _viewModel.SelectedPredictionModel.Should().Be(PredictionModel.Parametric);
    }

    [Fact]
    public void HasAnyData_ShouldReturnTrue_WhenHistoricalDataHasMultiplePoints()
    {
        // Arrange
        var historicalData = new List<HistoricalDataPoint>
        {
            new() { Date = DateTime.Today.AddDays(-2), Level = 5.0 },
            new() { Date = DateTime.Today.AddDays(-1), Level = 4.0 },
            new() { Date = DateTime.Today, Level = 3.0 }
        };

        // Act
        _viewModel.HistoricalData = historicalData;

        // Assert
        _viewModel.HasAnyData.Should().BeTrue();
    }

    [Fact]
    public void HasAnyData_ShouldReturnTrue_WhenPredictionDataExists()
    {
        // Arrange
        var predictionData = new List<PredictionDataPoint>
        {
            new() { Date = DateTime.Today.AddDays(1), PredictedLevel = 2.0, DaysRemaining = 3.0 }
        };

        // Act
        _viewModel.PredictionData = predictionData;

        // Assert
        _viewModel.HasAnyData.Should().BeTrue();
    }

    [Fact]
    public void HasAnyData_ShouldReturnFalse_WhenOnlyOneHistoricalPoint()
    {
        // Arrange
        var historicalData = new List<HistoricalDataPoint>
        {
            new() { Date = DateTime.Today, Level = 3.0 }
        };

        // Act
        _viewModel.HistoricalData = historicalData;

        // Assert
        _viewModel.HasAnyData.Should().BeFalse();
    }

    [Fact]
    public void ItemName_ShouldReturnItemName_WhenItemIsSet()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            Id = "test-id",
            Name = "Test Item",
            CurrentLevel = 5.0,
            UnitId = "kg"
        };

        // Act
        _viewModel.Item = item;

        // Assert
        _viewModel.ItemName.Should().Be("Test Item");
    }

    [Fact]
    public void CurrentLevelDisplay_ShouldFormatCorrectly_WhenItemIsSet()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            Id = "test-id",
            Name = "Test Item",
            CurrentLevel = 5.25,
            UnitId = "kg"
        };

        // Act
        _viewModel.Item = item;

        // Assert
        _viewModel.CurrentLevelDisplay.Should().Be("Current: 5.25 kg");
    }

    [Fact]
    public void PredictionSummary_ShouldFormatCorrectly_WhenPredictionDataExists()
    {
        // Arrange
        var predictionData = new List<PredictionDataPoint>
        {
            new() { Date = DateTime.Today.AddDays(1), PredictedLevel = 2.0, DaysRemaining = 1.0 },
            new() { Date = DateTime.Today.AddDays(2), PredictedLevel = 1.0, DaysRemaining = 2.5 }
        };

        // Act
        _viewModel.PredictionData = predictionData;

        // Assert
        _viewModel.PredictionSummary.Should().Be("Predicted empty in 2.5 days");
    }

    [Theory]
    [InlineData(ChartDataMode.Granularity, "Day", 100, null, null)]
    [InlineData(ChartDataMode.TimeRange, null, null, 30, true)]
    public async Task LoadHistoricalDataFromServiceAsync_ShouldUseCorrectParameters_BasedOnChartMode(
        ChartDataMode mode, string? expectedGranularity, int? expectedMaxPoints, int? timeRangeDays, bool? expectsTimeRange)
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            Id = "test-item-id",
            Name = "Test Item",
            CurrentLevel = 5.0,
            UnitId = "kg"
        };

        var chartSettings = new ChartSettings
        {
            Mode = mode,
            Granularity = HistoryGranularity.Day,
            MaxPoints = 100,
            TimeRangeDays = timeRangeDays ?? 30
        };

        // Mock settings service to return our chart settings
        SetupMockSettingsService(chartSettings);

        // Mock service response
        var mockSnapshots = new List<InventoryLevelSnapshotViewModel>
        {
            new() { Timestamp = DateTime.Today.AddDays(-2), Level = 6.0, UnitId = "kg" },
            new() { Timestamp = DateTime.Today.AddDays(-1), Level = 5.5, UnitId = "kg" },
            new() { Timestamp = DateTime.Today, Level = 5.0, UnitId = "kg" }
        };

        DateTime? capturedStartTime = null;
        DateTime? capturedEndTime = null;
        string? capturedGranularity = null;
        int? capturedMaxPoints = null;

        _mockInventoryService.Setup(s => s.GetItemHistoryAsync(
                It.IsAny<string>(),
                It.IsAny<DateTime?>(),
                It.IsAny<DateTime?>(),
                It.IsAny<string?>(),
                It.IsAny<int?>(),
                It.IsAny<System.Threading.CancellationToken>()))
            .Callback<string, DateTime?, DateTime?, string?, int?, System.Threading.CancellationToken>(
                (itemId, startTime, endTime, granularity, maxPoints, ct) =>
                {
                    capturedStartTime = startTime;
                    capturedEndTime = endTime;
                    capturedGranularity = granularity;
                    capturedMaxPoints = maxPoints;
                })
            .ReturnsAsync(mockSnapshots);

        // Act
        _viewModel.Item = item;
        await Task.Delay(100); // Give time for async operations

        // Assert
        _mockInventoryService.Verify(s => s.GetItemHistoryAsync(
            "test-item-id",
            It.IsAny<DateTime?>(),
            It.IsAny<DateTime?>(),
            It.IsAny<string?>(),
            It.IsAny<int?>(),
            It.IsAny<System.Threading.CancellationToken>()), Times.Once);

        if (mode == ChartDataMode.Granularity)
        {
            capturedGranularity.Should().Be(expectedGranularity);
            capturedMaxPoints.Should().Be(expectedMaxPoints);
            capturedStartTime.Should().BeNull();
            capturedEndTime.Should().BeNull();
        }
        else if (mode == ChartDataMode.TimeRange)
        {
            capturedGranularity.Should().BeNull();
            capturedMaxPoints.Should().BeNull();
            capturedStartTime.Should().NotBeNull();
            capturedEndTime.Should().NotBeNull();
            
            var expectedStartTime = capturedEndTime!.Value.AddDays(-timeRangeDays!.Value);
            capturedStartTime.Should().BeCloseTo(expectedStartTime, TimeSpan.FromMinutes(1));
        }

        // Verify data was processed correctly
        _viewModel.HistoricalData.Should().HaveCount(3);
        _viewModel.HistoricalData[0].Level.Should().Be(6.0);
        _viewModel.HistoricalData[1].Level.Should().Be(5.5);
        _viewModel.HistoricalData[2].Level.Should().Be(5.0);
    }

    [Fact]
    public async Task LoadHistoricalDataFromServiceAsync_ShouldHandleNullResponse()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            Id = "test-item-id",
            Name = "Test Item",
            CurrentLevel = 5.0,
            UnitId = "kg"
        };

        SetupMockSettingsService(new ChartSettings());

        _mockInventoryService.Setup(s => s.GetItemHistoryAsync(
                It.IsAny<string>(),
                It.IsAny<DateTime?>(),
                It.IsAny<DateTime?>(),
                It.IsAny<string?>(),
                It.IsAny<int?>(),
                It.IsAny<System.Threading.CancellationToken>()))
            .ReturnsAsync((IReadOnlyList<InventoryLevelSnapshotViewModel>?)null);

        // Act
        _viewModel.Item = item;
        await Task.Delay(100); // Give time for async operations

        // Assert
        _viewModel.HistoricalData.Should().BeEmpty();
        _viewModel.HasAnyData.Should().BeFalse();
    }

    [Fact]
    public async Task LoadHistoricalDataFromServiceAsync_ShouldHandleEmptyResponse()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            Id = "test-item-id",
            Name = "Test Item",
            CurrentLevel = 5.0,
            UnitId = "kg"
        };

        SetupMockSettingsService(new ChartSettings());

        _mockInventoryService.Setup(s => s.GetItemHistoryAsync(
                It.IsAny<string>(),
                It.IsAny<DateTime?>(),
                It.IsAny<DateTime?>(),
                It.IsAny<string?>(),
                It.IsAny<int?>(),
                It.IsAny<System.Threading.CancellationToken>()))
            .ReturnsAsync(new List<InventoryLevelSnapshotViewModel>());

        // Act
        _viewModel.Item = item;
        await Task.Delay(100); // Give time for async operations

        // Assert
        _viewModel.HistoricalData.Should().BeEmpty();
        _viewModel.HasAnyData.Should().BeFalse();
    }

    [Fact]
    public async Task LoadHistoricalDataFromServiceAsync_ShouldHandleServiceException()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            Id = "test-item-id",
            Name = "Test Item",
            CurrentLevel = 5.0,
            UnitId = "kg"
        };

        SetupMockSettingsService(new ChartSettings());

        _mockInventoryService.Setup(s => s.GetItemHistoryAsync(
                It.IsAny<string>(),
                It.IsAny<DateTime?>(),
                It.IsAny<DateTime?>(),
                It.IsAny<string?>(),
                It.IsAny<int?>(),
                It.IsAny<System.Threading.CancellationToken>()))
            .ThrowsAsync(new InvalidOperationException("Service connection failed"));

        // Act
        _viewModel.Item = item;
        await Task.Delay(100); // Give time for async operations

        // Assert
        _viewModel.HistoricalData.Should().BeEmpty();
        _viewModel.HasAnyData.Should().BeFalse();

        // Verify error was logged
        _mockLogger.Verify(
            x => x.Log(
                LogLevel.Error,
                It.IsAny<EventId>(),
                It.Is<It.IsAnyType>((v, t) => v.ToString()!.Contains("Failed to load historical data")),
                It.IsAny<Exception>(),
                It.IsAny<Func<It.IsAnyType, Exception?, string>>()),
            Times.Once);
    }

    [Fact]
    public void LoadHistoricalDataFromServiceAsync_ShouldHandleNullItem()
    {
        // Act
        _viewModel.Item = null;

        // Assert
        _viewModel.HistoricalData.Should().BeEmpty();
        _viewModel.HasAnyData.Should().BeFalse();
        _viewModel.ItemName.Should().Be("No Item Selected");
        _viewModel.CurrentLevelDisplay.Should().BeEmpty();
    }

    private void SetupMockSettingsService(ChartSettings chartSettings)
    {
        _mockSettingsService.Setup(s => s.GetSetting(ChartSettings.ModeKey, It.IsAny<ChartDataMode>()))
            .Returns(chartSettings.Mode);
        _mockSettingsService.Setup(s => s.GetSetting(ChartSettings.GranularityKey, It.IsAny<HistoryGranularity>()))
            .Returns(chartSettings.Granularity);
        _mockSettingsService.Setup(s => s.GetSetting(ChartSettings.MaxPointsKey, It.IsAny<int>()))
            .Returns(chartSettings.MaxPoints);
        _mockSettingsService.Setup(s => s.GetSetting(ChartSettings.TimeRangeDaysKey, It.IsAny<int>()))
            .Returns(chartSettings.TimeRangeDays);
        _mockSettingsService.Setup(s => s.GetSetting(ChartSettings.ShowPredictionsKey, It.IsAny<bool>()))
            .Returns(chartSettings.ShowPredictions);
        _mockSettingsService.Setup(s => s.GetSetting(ChartSettings.PredictionDaysAheadKey, It.IsAny<int>()))
            .Returns(chartSettings.PredictionDaysAhead);
    }
}
