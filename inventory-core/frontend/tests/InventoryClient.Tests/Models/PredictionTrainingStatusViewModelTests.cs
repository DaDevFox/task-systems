using InventoryClient.Models;
using System.ComponentModel;
using Xunit;

namespace InventoryClient.Tests.Models;

public class PredictionTrainingStatusViewModelTests
{
    [Fact]
    public void StageDescription_ReturnsCorrectDescription()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel();

        // Act & Assert
        viewModel.Stage = TrainingStage.Collecting;
        Assert.Equal("Collecting training data", viewModel.StageDescription);

        viewModel.Stage = TrainingStage.Learning;
        Assert.Equal("Processing collected data", viewModel.StageDescription);

        viewModel.Stage = TrainingStage.Trained;
        Assert.Equal("Model is trained and active", viewModel.StageDescription);

        viewModel.Stage = TrainingStage.Retraining;
        Assert.Equal("Model being retrained with new data", viewModel.StageDescription);

        viewModel.Stage = TrainingStage.Unspecified;
        Assert.Equal("Training status unknown", viewModel.StageDescription);
    }

    [Fact]
    public void ModelDescription_ReturnsCorrectDescription()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel();

        // Act & Assert
        viewModel.ActiveModel = PredictionModel.Markov;
        Assert.Equal("Markov Chain Model", viewModel.ModelDescription);

        viewModel.ActiveModel = PredictionModel.Croston;
        Assert.Equal("Croston's Method", viewModel.ModelDescription);

        viewModel.ActiveModel = PredictionModel.DriftImpulse;
        Assert.Equal("Drift & Impulse Model", viewModel.ModelDescription);

        viewModel.ActiveModel = PredictionModel.Bayesian;
        Assert.Equal("Bayesian Inference", viewModel.ModelDescription);

        viewModel.ActiveModel = PredictionModel.MemoryWindow;
        Assert.Equal("Memory Window Model", viewModel.ModelDescription);

        viewModel.ActiveModel = PredictionModel.EventTrigger;
        Assert.Equal("Event Trigger Model", viewModel.ModelDescription);

        viewModel.ActiveModel = PredictionModel.Unspecified;
        Assert.Equal("No model selected", viewModel.ModelDescription);
    }

    [Fact]
    public void TrainingProgress_CalculatedCorrectly()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel
        {
            MinSamplesRequired = 100,
            TrainingSamples = 75
        };

        // Act & Assert
        Assert.Equal(0.75, viewModel.TrainingProgress);

        // Test edge cases
        viewModel.TrainingSamples = 0;
        Assert.Equal(0.0, viewModel.TrainingProgress);

        viewModel.TrainingSamples = 150;
        Assert.Equal(1.0, viewModel.TrainingProgress); // Should cap at 1.0

        viewModel.MinSamplesRequired = 0;
        Assert.Equal(0.0, viewModel.TrainingProgress); // Should handle division by zero
    }

    [Fact]
    public void IsTrainingComplete_ReturnsTrueWhenTrained()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel();

        // Act & Assert
        viewModel.Stage = TrainingStage.Trained;
        Assert.True(viewModel.IsTrainingComplete);

        viewModel.Stage = TrainingStage.Learning;
        Assert.False(viewModel.IsTrainingComplete);

        viewModel.Stage = TrainingStage.Collecting;
        Assert.False(viewModel.IsTrainingComplete);
    }

    [Fact]
    public void CanStartTraining_ReturnsTrueWhenConditionsMet()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel
        {
            MinSamplesRequired = 100,
            TrainingSamples = 100,
            Stage = TrainingStage.Collecting
        };

        // Act & Assert
        Assert.True(viewModel.CanStartTraining);

        viewModel.TrainingSamples = 50;
        Assert.False(viewModel.CanStartTraining); // Not enough samples

        viewModel.TrainingSamples = 150;
        viewModel.Stage = TrainingStage.Learning;
        Assert.False(viewModel.CanStartTraining); // Already learning
    }

    [Fact]
    public void TrainingProgressText_FormattedCorrectly()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel
        {
            MinSamplesRequired = 100,
            TrainingSamples = 75
        };

        // Act & Assert
        Assert.Equal("75/100 samples (75.0%)", viewModel.TrainingProgressText);

        viewModel.TrainingSamples = 0;
        Assert.Equal("0/100 samples (0.0%)", viewModel.TrainingProgressText);
    }

    [Fact]
    public void AccuracyDisplayText_FormattedCorrectly()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel();

        // Act & Assert
        viewModel.TrainingAccuracy = 0.875;
        Assert.Equal("87.5%", viewModel.AccuracyDisplayText);

        viewModel.TrainingAccuracy = 0.0;
        Assert.Equal("Not available", viewModel.AccuracyDisplayText);

        viewModel.TrainingAccuracy = -0.1;
        Assert.Equal("Not available", viewModel.AccuracyDisplayText);
    }

    [Fact]
    public void PropertyChanged_FiresWhenPropertiesChange()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel();
        var propertyChangedEvents = new List<string>();
        viewModel.PropertyChanged += (sender, e) => propertyChangedEvents.Add(e.PropertyName!);

        // Act
        viewModel.Stage = TrainingStage.Learning;

        // Assert
        Assert.Contains(nameof(PredictionTrainingStatusViewModel.Stage), propertyChangedEvents);
        Assert.Contains(nameof(PredictionTrainingStatusViewModel.StageDescription), propertyChangedEvents);
        Assert.Contains(nameof(PredictionTrainingStatusViewModel.IsTrainingComplete), propertyChangedEvents);
        Assert.Contains(nameof(PredictionTrainingStatusViewModel.CanStartTraining), propertyChangedEvents);
    }

    [Fact]
    public void DerivedProperties_UpdateWhenBasePropertiesChange()
    {
        // Arrange
        var viewModel = new PredictionTrainingStatusViewModel();
        var propertyChangedEvents = new List<string>();
        viewModel.PropertyChanged += (sender, e) => propertyChangedEvents.Add(e.PropertyName!);

        // Act - Change samples
        viewModel.TrainingSamples = 50;

        // Assert
        Assert.Contains(nameof(PredictionTrainingStatusViewModel.TrainingProgress), propertyChangedEvents);
        Assert.Contains(nameof(PredictionTrainingStatusViewModel.CanStartTraining), propertyChangedEvents);
        Assert.Contains(nameof(PredictionTrainingStatusViewModel.TrainingProgressText), propertyChangedEvents);

        propertyChangedEvents.Clear();

        // Act - Change model
        viewModel.ActiveModel = PredictionModel.Markov;

        // Assert
        Assert.Contains(nameof(PredictionTrainingStatusViewModel.ModelDescription), propertyChangedEvents);
    }
}
