using FluentAssertions;
using InventoryClient.Models;
using Xunit;

namespace InventoryClient.Tests.Models;

public class InventoryItemViewModelTests
{
    [Fact]
    public void CurrentLevelPercentage_ShouldCalculateCorrectly()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            CurrentLevel = 25,
            MaxCapacity = 100
        };

        // Act
        var percentage = item.CurrentLevelPercentage;

        // Assert
        percentage.Should().Be(25.0);
    }

    [Fact]
    public void CurrentLevelPercentage_ShouldReturnZero_WhenMaxCapacityIsZero()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            CurrentLevel = 25,
            MaxCapacity = 0
        };

        // Act
        var percentage = item.CurrentLevelPercentage;

        // Assert
        percentage.Should().Be(0.0);
    }

    [Theory]
    [InlineData(0, true)]
    [InlineData(-1, true)]
    [InlineData(1, false)]
    [InlineData(10, false)]
    public void IsEmpty_ShouldReturnCorrectValue(double currentLevel, bool expected)
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            CurrentLevel = currentLevel
        };

        // Act & Assert
        item.IsEmpty.Should().Be(expected);
    }

    [Theory]
    [InlineData(5, 10, true)]   // Below threshold
    [InlineData(10, 10, true)]  // At threshold
    [InlineData(15, 10, false)] // Above threshold
    [InlineData(0, 10, false)]  // Empty (not low stock)
    public void IsLowStock_ShouldReturnCorrectValue(double currentLevel, double threshold, bool expected)
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            CurrentLevel = currentLevel,
            LowStockThreshold = threshold
        };

        // Act & Assert
        item.IsLowStock.Should().Be(expected);
    }

    [Theory]
    [InlineData(0, "Empty")]
    [InlineData(5, "Low")]    // Assuming threshold is 10
    [InlineData(15, "Normal")] // Above threshold
    public void StockStatus_ShouldReturnCorrectStatus(double currentLevel, string expected)
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            CurrentLevel = currentLevel,
            LowStockThreshold = 10
        };

        // Act & Assert
        item.StockStatus.Should().Be(expected);
    }

    [Fact]
    public void StockStatusDescription_ShouldReturnCorrectDescription_WhenEmpty()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            CurrentLevel = 0
        };

        // Act & Assert
        item.StockStatusDescription.Should().Be("Out of stock");
    }

    [Fact]
    public void StockStatusDescription_ShouldReturnCorrectDescription_WhenLowStock()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            CurrentLevel = 2.5,
            LowStockThreshold = 5.0,
            UnitId = "kg"
        };

        // Act & Assert
        item.StockStatusDescription.Should().Be("Low stock - 2.5 kg remaining");
    }

    [Fact]
    public void StockStatusDescription_ShouldReturnCorrectDescription_WhenNormalStock()
    {
        // Arrange
        var item = new InventoryItemViewModel
        {
            CurrentLevel = 15.0,
            LowStockThreshold = 5.0,
            UnitId = "kg"
        };

        // Act & Assert
        item.StockStatusDescription.Should().Be("15.0 kg available");
    }

    [Fact]
    public void PropertyChanged_ShouldRaise_WhenCurrentLevelChanges()
    {
        // Arrange
        var item = new InventoryItemViewModel();
        var eventRaised = false;
        item.PropertyChanged += (_, e) =>
        {
            if (e.PropertyName == nameof(InventoryItemViewModel.CurrentLevel))
                eventRaised = true;
        };

        // Act
        item.CurrentLevel = 10.0;

        // Assert
        eventRaised.Should().BeTrue();
    }
}
