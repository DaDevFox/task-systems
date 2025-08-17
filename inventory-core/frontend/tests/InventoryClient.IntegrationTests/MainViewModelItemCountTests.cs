using Xunit;
using FluentAssertions;
using Microsoft.Extensions.Logging;
using Moq;
using InventoryClient.ViewModels;
using InventoryClient.Services;
using InventoryClient.Models;
using TaskSystems.Shared.Services;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;

namespace InventoryClient.IntegrationTests;

/// <summary>
/// Unit tests for the MainViewModel item counting logic
/// </summary>
public class MainViewModelItemCountTests
{
    private readonly Mock<IInventoryService> _mockInventoryService;
    private readonly Mock<IServiceClient> _mockServiceClient;
    private readonly Mock<ISettingsService> _mockSettingsService;
    private readonly Mock<ILogger<MainViewModel>> _mockLogger;

    public MainViewModelItemCountTests()
    {
        _mockInventoryService = new Mock<IInventoryService>();
        _mockServiceClient = new Mock<IServiceClient>();
        _mockSettingsService = new Mock<ISettingsService>();
        _mockLogger = new Mock<ILogger<MainViewModel>>();

        // Setup default settings service behavior
        _mockSettingsService.Setup(s => s.GetSetting<bool>("Debug.ShowCacheInfo", false)).Returns(false);
        _mockSettingsService.Setup(s => s.GetSetting<bool>("AutoRefresh.Enabled", true)).Returns(true);
        _mockSettingsService.Setup(s => s.GetSetting<int>("AutoRefresh.IntervalSeconds", 30)).Returns(30);
    }

    [Fact]
    public void DisplayedItems_ShouldReturnFilteredItems()
    {
        // Arrange
        var viewModel = new MainViewModel(_mockInventoryService.Object, _mockServiceClient.Object,
            _mockSettingsService.Object, _mockLogger.Object);

        var testItems = CreateTestItems();
        foreach (var item in testItems)
        {
            viewModel.InventoryItems.Add(item);
        }

        // Act - No filters applied
        viewModel.ShowLowStockOnly = false;
        viewModel.SearchText = "";

        // Force update of filtered items
        var method = typeof(MainViewModel).GetMethod("UpdateFilteredItems",
            System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);
        method?.Invoke(viewModel, null);

        // Assert
        viewModel.DisplayedItems.Count.Should().Be(testItems.Count,
            "DisplayedItems should show all items when no filters applied");
        viewModel.FilteredItems.Count.Should().Be(testItems.Count,
            "FilteredItems should contain all items when no filters applied");
        viewModel.DisplayedItems.Should().BeSameAs(viewModel.FilteredItems,
            "DisplayedItems should return the FilteredItems collection");
    }

    [Fact]
    public void LowStockFilter_ShouldUpdateItemCountCorrectly()
    {
        // Arrange
        var viewModel = new MainViewModel(_mockInventoryService.Object, _mockServiceClient.Object,
            _mockSettingsService.Object, _mockLogger.Object);

        var testItems = CreateTestItems();
        foreach (var item in testItems)
        {
            viewModel.InventoryItems.Add(item);
        }

        var expectedLowStockCount = testItems.Count(i => i.IsLowStock || i.IsEmpty);

        // Act - Apply low stock filter
        viewModel.ShowLowStockOnly = true;
        viewModel.SearchText = "";

        // Force update of filtered items
        var method = typeof(MainViewModel).GetMethod("UpdateFilteredItems",
            System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);
        method?.Invoke(viewModel, null);

        // Assert
        viewModel.DisplayedItems.Count.Should().Be(expectedLowStockCount,
            "DisplayedItems should show only low stock items");
        viewModel.FilteredItems.Count.Should().Be(expectedLowStockCount,
            "FilteredItems should contain only low stock items");

        foreach (var item in viewModel.DisplayedItems)
        {
            (item.IsLowStock || item.IsEmpty).Should().BeTrue("All displayed items should be low stock or empty");
        }
    }

    [Fact]
    public void SearchFilter_ShouldUpdateItemCountCorrectly()
    {
        // Arrange
        var viewModel = new MainViewModel(_mockInventoryService.Object, _mockServiceClient.Object,
            _mockSettingsService.Object, _mockLogger.Object);

        var testItems = CreateTestItems();
        foreach (var item in testItems)
        {
            viewModel.InventoryItems.Add(item);
        }

        const string searchTerm = "flour";
        var expectedSearchCount = testItems.Count(i =>
            i.Name.Contains(searchTerm, System.StringComparison.OrdinalIgnoreCase) ||
            i.Description.Contains(searchTerm, System.StringComparison.OrdinalIgnoreCase));

        // Act - Apply search filter
        viewModel.ShowLowStockOnly = false;
        viewModel.SearchText = searchTerm;

        // Force update of filtered items
        var method = typeof(MainViewModel).GetMethod("UpdateFilteredItems",
            System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);
        method?.Invoke(viewModel, null);

        // Assert
        viewModel.DisplayedItems.Count.Should().Be(expectedSearchCount,
            "DisplayedItems should show only items matching search term");
        viewModel.FilteredItems.Count.Should().Be(expectedSearchCount,
            "FilteredItems should contain only items matching search term");

        foreach (var item in viewModel.DisplayedItems)
        {
            (item.Name.Contains(searchTerm, System.StringComparison.OrdinalIgnoreCase) ||
             item.Description.Contains(searchTerm, System.StringComparison.OrdinalIgnoreCase))
                .Should().BeTrue("All displayed items should match search term");
        }
    }

    [Fact]
    public void CombinedFilters_ShouldUpdateItemCountCorrectly()
    {
        // Arrange
        var viewModel = new MainViewModel(_mockInventoryService.Object, _mockServiceClient.Object,
            _mockSettingsService.Object, _mockLogger.Object);

        var testItems = CreateTestItems();
        foreach (var item in testItems)
        {
            viewModel.InventoryItems.Add(item);
        }

        const string searchTerm = "test";
        var expectedCount = testItems.Count(i =>
            (i.IsLowStock || i.IsEmpty) &&
            (i.Name.Contains(searchTerm, System.StringComparison.OrdinalIgnoreCase) ||
             i.Description.Contains(searchTerm, System.StringComparison.OrdinalIgnoreCase)));

        // Act - Apply both filters
        viewModel.ShowLowStockOnly = true;
        viewModel.SearchText = searchTerm;

        // Force update of filtered items
        var method = typeof(MainViewModel).GetMethod("UpdateFilteredItems",
            System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);
        method?.Invoke(viewModel, null);

        // Assert
        viewModel.DisplayedItems.Count.Should().Be(expectedCount,
            "DisplayedItems should show only items matching both filters");
        viewModel.FilteredItems.Count.Should().Be(expectedCount,
            "FilteredItems should contain only items matching both filters");

        foreach (var item in viewModel.DisplayedItems)
        {
            (item.IsLowStock || item.IsEmpty).Should().BeTrue("All displayed items should be low stock or empty");
            (item.Name.Contains(searchTerm, System.StringComparison.OrdinalIgnoreCase) ||
             item.Description.Contains(searchTerm, System.StringComparison.OrdinalIgnoreCase))
                .Should().BeTrue("All displayed items should match search term");
        }
    }

    [Fact]
    public void RefreshDataAsync_ShouldNotCreateDuplicateItems_WhenCalledMultipleTimes()
    {
        // Arrange
        var viewModel = new MainViewModel(_mockInventoryService.Object, _mockServiceClient.Object,
            _mockSettingsService.Object, _mockLogger.Object);

        var testItems = CreateTestItems();

        // Setup mock service to return test items
        _mockInventoryService.Setup(s => s.PingAsync()).ReturnsAsync(true);
        _mockInventoryService.Setup(s => s.ListInventoryItemsAsync(
            It.IsAny<bool>(), It.IsAny<string>(), It.IsAny<int>(), It.IsAny<int>()))
            .ReturnsAsync((testItems, testItems.Count));

        // Act - Simulate multiple refresh calls (like auto-refresh would do)
        var refreshMethod = typeof(MainViewModel).GetMethod("RefreshDataAsync",
            System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);

        // First refresh
        var task1 = (Task)refreshMethod?.Invoke(viewModel, null);
        task1?.Wait();

        var countAfterFirstRefresh = viewModel.InventoryItems.Count;

        // Second refresh (simulating auto-refresh)
        var task2 = (Task)refreshMethod?.Invoke(viewModel, null);
        task2?.Wait();

        var countAfterSecondRefresh = viewModel.InventoryItems.Count;

        // Third refresh to be sure
        var task3 = (Task)refreshMethod?.Invoke(viewModel, null);
        task3?.Wait();

        var countAfterThirdRefresh = viewModel.InventoryItems.Count;

        // Assert
        countAfterFirstRefresh.Should().Be(testItems.Count, "First refresh should add all items");
        countAfterSecondRefresh.Should().Be(testItems.Count, "Second refresh should not create duplicates");
        countAfterThirdRefresh.Should().Be(testItems.Count, "Third refresh should not create duplicates");

        // Verify no duplicate IDs exist
        var itemIds = viewModel.InventoryItems.Select(i => i.Id).ToList();
        var uniqueIds = itemIds.Distinct().ToList();
        uniqueIds.Count.Should().Be(itemIds.Count, "No duplicate item IDs should exist");

        // Verify no duplicate names exist
        var itemNames = viewModel.InventoryItems.Select(i => i.Name).ToList();
        var uniqueNames = itemNames.Distinct().ToList();
        uniqueNames.Count.Should().Be(itemNames.Count, "No duplicate item names should exist");
    }

    [Fact]
    public void UpdateFilteredItems_ShouldNotCreateDuplicatesInFilteredCollection()
    {
        // Arrange
        var viewModel = new MainViewModel(_mockInventoryService.Object, _mockServiceClient.Object,
            _mockSettingsService.Object, _mockLogger.Object);

        var testItems = CreateTestItems();
        foreach (var item in testItems)
        {
            viewModel.InventoryItems.Add(item);
        }

        // Act - Call UpdateFilteredItems multiple times
        var updateMethod = typeof(MainViewModel).GetMethod("UpdateFilteredItems",
            System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);

        updateMethod?.Invoke(viewModel, null);
        var countAfterFirstUpdate = viewModel.FilteredItems.Count;

        updateMethod?.Invoke(viewModel, null);
        var countAfterSecondUpdate = viewModel.FilteredItems.Count;

        updateMethod?.Invoke(viewModel, null);
        var countAfterThirdUpdate = viewModel.FilteredItems.Count;

        // Assert
        countAfterFirstUpdate.Should().Be(testItems.Count, "First update should contain all items");
        countAfterSecondUpdate.Should().Be(testItems.Count, "Second update should not create duplicates");
        countAfterThirdUpdate.Should().Be(testItems.Count, "Third update should not create duplicates");

        // Verify no duplicate IDs in filtered collection
        var filteredIds = viewModel.FilteredItems.Select(i => i.Id).ToList();
        var uniqueFilteredIds = filteredIds.Distinct().ToList();
        uniqueFilteredIds.Count.Should().Be(filteredIds.Count, "No duplicate IDs should exist in FilteredItems");
    }

    private static List<InventoryItemViewModel> CreateTestItems()
    {
        return new List<InventoryItemViewModel>
        {
            new()
            {
                Id = "1",
                Name = "Flour",
                Description = "All-purpose flour for baking",
                CurrentLevel = 2.5,
                MaxCapacity = 10.0,
                LowStockThreshold = 2.0,
                UnitId = "kg"
            },
            new()
            {
                Id = "2",
                Name = "Sugar",
                Description = "White granulated sugar",
                CurrentLevel = 0.8,
                MaxCapacity = 5.0,
                LowStockThreshold = 1.0,
                UnitId = "kg"
            },
            new()
            {
                Id = "3",
                Name = "Test Milk",
                Description = "Fresh whole milk",
                CurrentLevel = 0.0,
                MaxCapacity = 4.0,
                LowStockThreshold = 0.5,
                UnitId = "liters"
            },
            new()
            {
                Id = "4",
                Name = "Eggs",
                Description = "Large eggs, test grade A",
                CurrentLevel = 18,
                MaxCapacity = 24,
                LowStockThreshold = 6,
                UnitId = "pieces"
            },
            new()
            {
                Id = "5",
                Name = "Test Flour Special",
                Description = "Special test flour blend",
                CurrentLevel = 1.0,
                MaxCapacity = 8.0,
                LowStockThreshold = 2.0,
                UnitId = "kg"
            }
        };
    }
}
