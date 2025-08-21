using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using FluentAssertions;
using Grpc.Net.Client;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using InventoryClient.IntegrationTests.Infrastructure;
using InventoryClient.Models;
using InventoryClient.Services;
using InventoryClient.ViewModels;
using TaskSystems.Shared.Services;
using Inventory.V1;
using Xunit;

namespace InventoryClient.IntegrationTests;

/// <summary>
/// Integration tests for InventoryLevelChartViewModel that verify chart data matches direct backend responses
/// </summary>
public class ChartViewModelIntegrationTests : IntegrationTestBase
{
    private GrpcChannel? _grpcChannel;
    private InventoryService.InventoryServiceClient? _directClient;
    private InventoryLevelChartViewModel? _chartViewModel;

    public override async Task InitializeAsync()
    {
        await base.InitializeAsync();

        // Create direct gRPC client for comparison
        _grpcChannel = GrpcChannel.ForAddress($"http://{ServerAddress}");
        _directClient = new InventoryService.InventoryServiceClient(_grpcChannel);

        // Create chart view model
        var loggerFactory = Host.Services.GetRequiredService<ILoggerFactory>();
        var chartLogger = loggerFactory.CreateLogger<InventoryLevelChartViewModel>();
        var settingsService = Host.Services.GetRequiredService<ISettingsService>();

        _chartViewModel = new InventoryLevelChartViewModel(
            InventoryService,
            settingsService,
            chartLogger);
    }

    [Fact]
    public async Task ChartViewModel_ShouldReturnSameDataAsDirectGrpcCall_GranularityMode()
    {
        // Arrange - Create test item with some history
        var addResponse = await _directClient!.AddInventoryItemAsync(new AddInventoryItemRequest
        {
            Name = "Chart Test Item Granularity",
            Description = "Item for testing chart data consistency - Granularity mode",
            InitialLevel = 10.0,
            MaxCapacity = 20.0,
            LowStockThreshold = 3.0,
            UnitId = "kg"
        });

        // Add some historical data by updating levels
        await Task.Delay(100);
        await _directClient.UpdateInventoryLevelAsync(new UpdateInventoryLevelRequest
        {
            ItemId = addResponse.Item.Id,
            NewLevel = 8.5,
            Reason = "Test update 1"
        });

        await Task.Delay(100);
        await _directClient.UpdateInventoryLevelAsync(new UpdateInventoryLevelRequest
        {
            ItemId = addResponse.Item.Id,
            NewLevel = 7.0,
            Reason = "Test update 2"
        });

        await Task.Delay(100);
        await _directClient.UpdateInventoryLevelAsync(new UpdateInventoryLevelRequest
        {
            ItemId = addResponse.Item.Id,
            NewLevel = 5.5,
            Reason = "Test update 3"
        });

        // Set up chart settings for Granularity mode
        var settingsService = Host.Services.GetRequiredService<ISettingsService>();
        settingsService.SetSetting(ChartSettings.ModeKey, ChartDataMode.Granularity);
        settingsService.SetSetting(ChartSettings.GranularityKey, HistoryGranularity.Minute);
        settingsService.SetSetting(ChartSettings.MaxPointsKey, 100);

        // Connect the UI service
        await InventoryService.ConnectAsync(ServerAddress);

        // Act - Get data through both paths
        
        // Direct gRPC call
        var directRequest = new GetItemHistoryRequest
        {
            ItemId = addResponse.Item.Id,
            TimeRange = new TimeRangeQuery
            {
                Granularity = HistoryGranularity.Minute,
                MaxPoints = 100
            }
        };
        var directResponse = await _directClient.GetItemHistoryAsync(directRequest);

        // Chart view model
        var item = new InventoryItemViewModel
        {
            Id = addResponse.Item.Id,
            Name = addResponse.Item.Name,
            CurrentLevel = addResponse.Item.CurrentLevel,
            MaxCapacity = addResponse.Item.MaxCapacity,
            LowStockThreshold = addResponse.Item.LowStockThreshold,
            UnitId = addResponse.Item.UnitId
        };

        _chartViewModel!.Item = item;
        
        // Wait for the chart to load data
        await Task.Delay(1000);

        // Assert
        Logger.LogInformation("Direct gRPC returned {DirectCount} history points", directResponse.History.Count);
        Logger.LogInformation("Chart ViewModel has {ChartCount} historical data points", _chartViewModel.HistoricalData.Count);

        // Verify counts match
        _chartViewModel.HistoricalData.Count.Should().Be(directResponse.History.Count,
            "Chart view model should have same number of historical points as direct gRPC response");

        // Verify data points match (allowing for small time differences due to processing)
        var directHistoryList = directResponse.History.OrderBy(h => h.Timestamp?.ToDateTime()).ToList();
        var chartHistoryList = _chartViewModel.HistoricalData.OrderBy(h => h.Date).ToList();

        for (int i = 0; i < Math.Min(directHistoryList.Count, chartHistoryList.Count); i++)
        {
            var directPoint = directHistoryList[i];
            var chartPoint = chartHistoryList[i];

            Logger.LogInformation("Point {Index}: Direct={DirectLevel}@{DirectTime}, Chart={ChartLevel}@{ChartTime}",
                i, directPoint.Level, directPoint.Timestamp?.ToDateTime(), chartPoint.Level, chartPoint.Date);

            chartPoint.Level.Should().Be(directPoint.Level, $"Level should match for history point {i}");
            chartPoint.Date.Should().BeCloseTo(directPoint.Timestamp?.ToDateTime() ?? DateTime.MinValue,
                TimeSpan.FromSeconds(5), $"Timestamp should be close for history point {i}");
        }

        // Verify chart reports having data when it should
        var expectedHasData = directResponse.History.Count > 1;
        _chartViewModel.HasAnyData.Should().Be(expectedHasData,
            "Chart should report having data correctly based on history count");
    }

    [Fact]
    public async Task ChartViewModel_ShouldReturnSameDataAsDirectGrpcCall_TimeRangeMode()
    {
        // Arrange - Create test item with some history
        var addResponse = await _directClient!.AddInventoryItemAsync(new AddInventoryItemRequest
        {
            Name = "Chart Test Item TimeRange",
            Description = "Item for testing chart data consistency - TimeRange mode",
            InitialLevel = 15.0,
            MaxCapacity = 25.0,
            LowStockThreshold = 4.0,
            UnitId = "kg"
        });

        // Add some historical data points
        var updates = new[]
        {
            (Level: 13.0, Reason: "Update 1"),
            (Level: 11.5, Reason: "Update 2"),
            (Level: 9.0, Reason: "Update 3"),
            (Level: 7.5, Reason: "Update 4")
        };

        foreach (var update in updates)
        {
            await Task.Delay(100);
            await _directClient.UpdateInventoryLevelAsync(new UpdateInventoryLevelRequest
            {
                ItemId = addResponse.Item.Id,
                NewLevel = update.Level,
                Reason = update.Reason
            });
        }

        // Set up chart settings for TimeRange mode
        var settingsService = Host.Services.GetRequiredService<ISettingsService>();
        settingsService.SetSetting(ChartSettings.ModeKey, ChartDataMode.TimeRange);
        settingsService.SetSetting(ChartSettings.TimeRangeDaysKey, 1); // Last 1 day

        // Connect the UI service
        await InventoryService.ConnectAsync(ServerAddress);

        // Act - Get data through both paths
        
        // Direct gRPC call with same time range
        var endTime = DateTime.UtcNow;
        var startTime = endTime.AddDays(-1);
        
        var directRequest = new GetItemHistoryRequest
        {
            ItemId = addResponse.Item.Id,
            TimeRange = new TimeRangeQuery
            {
                StartTime = Google.Protobuf.WellKnownTypes.Timestamp.FromDateTime(startTime),
                EndTime = Google.Protobuf.WellKnownTypes.Timestamp.FromDateTime(endTime)
            }
        };
        var directResponse = await _directClient.GetItemHistoryAsync(directRequest);

        // Chart view model
        var item = new InventoryItemViewModel
        {
            Id = addResponse.Item.Id,
            Name = addResponse.Item.Name,
            CurrentLevel = addResponse.Item.CurrentLevel,
            MaxCapacity = addResponse.Item.MaxCapacity,
            LowStockThreshold = addResponse.Item.LowStockThreshold,
            UnitId = addResponse.Item.UnitId
        };

        _chartViewModel!.Item = item;
        
        // Wait for the chart to load data
        await Task.Delay(1000);

        // Assert
        Logger.LogInformation("TimeRange mode - Direct gRPC returned {DirectCount} history points", directResponse.History.Count);
        Logger.LogInformation("TimeRange mode - Chart ViewModel has {ChartCount} historical data points", _chartViewModel.HistoricalData.Count);

        // Verify counts match
        _chartViewModel.HistoricalData.Count.Should().Be(directResponse.History.Count,
            "Chart view model should have same number of historical points as direct gRPC response in TimeRange mode");

        // Verify data points match
        var directHistoryList = directResponse.History.OrderBy(h => h.Timestamp?.ToDateTime()).ToList();
        var chartHistoryList = _chartViewModel.HistoricalData.OrderBy(h => h.Date).ToList();

        for (int i = 0; i < Math.Min(directHistoryList.Count, chartHistoryList.Count); i++)
        {
            var directPoint = directHistoryList[i];
            var chartPoint = chartHistoryList[i];

            Logger.LogInformation("TimeRange Point {Index}: Direct={DirectLevel}@{DirectTime}, Chart={ChartLevel}@{ChartTime}",
                i, directPoint.Level, directPoint.Timestamp?.ToDateTime(), chartPoint.Level, chartPoint.Date);

            chartPoint.Level.Should().Be(directPoint.Level, $"Level should match for history point {i} in TimeRange mode");
            chartPoint.Date.Should().BeCloseTo(directPoint.Timestamp?.ToDateTime() ?? DateTime.MinValue,
                TimeSpan.FromSeconds(5), $"Timestamp should be close for history point {i} in TimeRange mode");
        }
    }

    [Fact]
    public async Task ChartViewModel_ShouldHandleEmptyHistory_Gracefully()
    {
        // Arrange - Create test item without any history updates
        var addResponse = await _directClient!.AddInventoryItemAsync(new AddInventoryItemRequest
        {
            Name = "Chart Test Item Empty",
            Description = "Item for testing empty history handling",
            InitialLevel = 5.0,
            MaxCapacity = 10.0,
            LowStockThreshold = 2.0,
            UnitId = "kg"
        });

        // Set up chart settings
        var settingsService = Host.Services.GetRequiredService<ISettingsService>();
        settingsService.SetSetting(ChartSettings.ModeKey, ChartDataMode.Granularity);
        settingsService.SetSetting(ChartSettings.GranularityKey, HistoryGranularity.Day);
        settingsService.SetSetting(ChartSettings.MaxPointsKey, 50);

        // Connect the UI service
        await InventoryService.ConnectAsync(ServerAddress);

        // Act - Get data through both paths
        
        // Direct gRPC call
        var directRequest = new GetItemHistoryRequest
        {
            ItemId = addResponse.Item.Id,
            TimeRange = new TimeRangeQuery
            {
                Granularity = HistoryGranularity.Day,
                MaxPoints = 50
            }
        };
        var directResponse = await _directClient!.GetItemHistoryAsync(directRequest);

        // Chart view model
        var item = new InventoryItemViewModel
        {
            Id = addResponse.Item.Id,
            Name = addResponse.Item.Name,
            CurrentLevel = addResponse.Item.CurrentLevel,
            MaxCapacity = addResponse.Item.MaxCapacity,
            LowStockThreshold = addResponse.Item.LowStockThreshold,
            UnitId = addResponse.Item.UnitId
        };

        _chartViewModel!.Item = item;
        
        // Wait for the chart to load data
        await Task.Delay(500);

        // Assert
        Logger.LogInformation("Empty history test - Direct gRPC returned {DirectCount} history points", directResponse.History.Count);
        Logger.LogInformation("Empty history test - Chart ViewModel has {ChartCount} historical data points", _chartViewModel.HistoricalData.Count);

        // Verify both return empty/minimal results consistently
        _chartViewModel.HistoricalData.Count.Should().Be(directResponse.History.Count,
            "Chart view model should match direct gRPC response even for empty history");

        // Chart should handle empty data gracefully
        _chartViewModel.ItemName.Should().Be("Chart Test Item Empty");
        _chartViewModel.CurrentLevelDisplay.Should().Be("Current: 5.00 units");
        _chartViewModel.PredictionSummary.Should().BeEmpty();
    }

    [Theory]
    [InlineData(ChartDataMode.Granularity, HistoryGranularity.Hour, 50)]
    [InlineData(ChartDataMode.Granularity, HistoryGranularity.Day, 30)]
    [InlineData(ChartDataMode.TimeRange, null, null, 7)]
    [InlineData(ChartDataMode.TimeRange, null, null, 30)]
    public async Task ChartViewModel_ShouldRespectDifferentChartSettings(
        ChartDataMode mode, HistoryGranularity? granularity, int? maxPoints, int timeRangeDays = 30)
    {
        // Arrange - Create test item with history
        var addResponse = await _directClient!.AddInventoryItemAsync(new AddInventoryItemRequest
        {
            Name = $"Settings Test {mode}",
            Description = "Item for testing different chart settings",
            InitialLevel = 12.0,
            MaxCapacity = 20.0,
            LowStockThreshold = 3.0,
            UnitId = "kg"
        });

        // Add some history
        for (int i = 1; i <= 5; i++)
        {
            await Task.Delay(50);
            await _directClient.UpdateInventoryLevelAsync(new UpdateInventoryLevelRequest
            {
                ItemId = addResponse.Item.Id,
                NewLevel = 12.0 - i,
                Reason = $"Test update {i}"
            });
        }

        // Set up chart settings
        var settingsService = Host.Services.GetRequiredService<ISettingsService>();
        settingsService.SetSetting(ChartSettings.ModeKey, mode);
        if (granularity.HasValue)
            settingsService.SetSetting(ChartSettings.GranularityKey, granularity.Value);
        if (maxPoints.HasValue)
            settingsService.SetSetting(ChartSettings.MaxPointsKey, maxPoints.Value);
        settingsService.SetSetting(ChartSettings.TimeRangeDaysKey, timeRangeDays);

        // Connect the UI service
        await InventoryService.ConnectAsync(ServerAddress);

        // Act - Get data through chart view model
        var item = new InventoryItemViewModel
        {
            Id = addResponse.Item.Id,
            Name = addResponse.Item.Name,
            CurrentLevel = addResponse.Item.CurrentLevel,
            MaxCapacity = addResponse.Item.MaxCapacity,
            LowStockThreshold = addResponse.Item.LowStockThreshold,
            UnitId = addResponse.Item.UnitId
        };

        _chartViewModel!.Item = item;
        await Task.Delay(1000);

        // Create equivalent direct request
        var directRequest = new GetItemHistoryRequest { ItemId = addResponse.Item.Id };
        
        if (mode == ChartDataMode.Granularity)
        {
            directRequest.TimeRange = new TimeRangeQuery
            {
                Granularity = granularity!.Value,
                MaxPoints = maxPoints!.Value
            };
        }
        else
        {
            var endTime = DateTime.UtcNow;
            var startTime = endTime.AddDays(-timeRangeDays);
            directRequest.TimeRange = new TimeRangeQuery
            {
                StartTime = Google.Protobuf.WellKnownTypes.Timestamp.FromDateTime(startTime),
                EndTime = Google.Protobuf.WellKnownTypes.Timestamp.FromDateTime(endTime)
            };
        }

        var directResponse = await _directClient.GetItemHistoryAsync(directRequest);

        // Assert
        Logger.LogInformation("Settings test {Mode} - Direct: {DirectCount}, Chart: {ChartCount}",
            mode, directResponse.History.Count, _chartViewModel.HistoricalData.Count);

        _chartViewModel.HistoricalData.Count.Should().Be(directResponse.History.Count,
            $"Chart data count should match direct response for {mode} mode");

        // Verify the settings were actually applied by checking the data makes sense
        if (mode == ChartDataMode.Granularity && maxPoints.HasValue)
        {
            _chartViewModel.HistoricalData.Count.Should().BeLessOrEqualTo(maxPoints.Value,
                "Granularity mode should respect max points setting");
        }
    }

    public override async Task DisposeAsync()
    {
        _grpcChannel?.Dispose();
        await base.DisposeAsync();
    }
}
