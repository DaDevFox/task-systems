package prediction

import (
	"fmt"
	"math"
	"time"

	"github.com/sirupsen/logrus"
)

// FitnessDataPoint represents a single fitness measurement
type FitnessDataPoint struct {
	Timestamp      time.Time
	ActualValue    float64
	PredictedValue float64
	Error          float64
	FitnessScore   float64
}

// ModelFitness tracks historical performance of a prediction model
type ModelFitness struct {
	ItemID          string
	Model           PredictionModel
	CurrentFitness  float64
	PredictionCount int
	AverageError    float64
	ErrorVariance   float64
	LastUpdated     time.Time
	FitnessHistory  []FitnessDataPoint
}

// PredictionConfig defines which models are enabled for an item
type PredictionConfig struct {
	ItemID         string
	EnabledModels  []PredictionModel
	PreferredModel PredictionModel
	AutoSelectBest bool
	GlobalSettings map[string]string
}

// FitnessTracker manages fitness tracking for prediction models
type FitnessTracker struct {
	modelFitness map[string]map[PredictionModel]*ModelFitness // itemID -> model -> fitness
	configs      map[string]*PredictionConfig                 // itemID -> config
	logger       *logrus.Logger
}

// NewFitnessTracker creates a new fitness tracker
func NewFitnessTracker(logger *logrus.Logger) *FitnessTracker {
	return &FitnessTracker{
		modelFitness: make(map[string]map[PredictionModel]*ModelFitness),
		configs:      make(map[string]*PredictionConfig),
		logger:       logger,
	}
}

// UpdateFitness updates the fitness score for a model based on a new prediction vs actual
func (ft *FitnessTracker) UpdateFitness(itemID string, model PredictionModel, actualValue, predictedValue float64, observationTime time.Time) error {
	if ft.modelFitness[itemID] == nil {
		ft.modelFitness[itemID] = make(map[PredictionModel]*ModelFitness)
	}

	fitness := ft.modelFitness[itemID][model]
	if fitness == nil {
		fitness = &ModelFitness{
			ItemID:         itemID,
			Model:          model,
			CurrentFitness: 0.5, // Start with neutral fitness
			FitnessHistory: make([]FitnessDataPoint, 0),
			LastUpdated:    observationTime,
		}
		ft.modelFitness[itemID][model] = fitness
	}

	// Calculate error and fitness score
	error := math.Abs(actualValue - predictedValue)
	relativeError := error / math.Max(actualValue, 0.1) // Avoid division by zero

	// Fitness score: 1.0 = perfect prediction, 0.0 = worst prediction
	// Use exponential decay to penalize large errors more severely
	fitnessScore := math.Exp(-relativeError)

	// Create fitness data point
	dataPoint := FitnessDataPoint{
		Timestamp:      observationTime,
		ActualValue:    actualValue,
		PredictedValue: predictedValue,
		Error:          error,
		FitnessScore:   fitnessScore,
	}

	// Add to history (keep last 100 points)
	fitness.FitnessHistory = append(fitness.FitnessHistory, dataPoint)
	if len(fitness.FitnessHistory) > 100 {
		fitness.FitnessHistory = fitness.FitnessHistory[1:]
	}

	// Update running statistics
	fitness.PredictionCount++

	// Update average error with exponential moving average
	alpha := 0.1 // Smoothing factor
	if fitness.PredictionCount == 1 {
		fitness.AverageError = error
		fitness.ErrorVariance = 0
	} else {
		fitness.AverageError = alpha*error + (1-alpha)*fitness.AverageError

		// Update variance (simplified calculation)
		variance := math.Pow(error-fitness.AverageError, 2)
		fitness.ErrorVariance = alpha*variance + (1-alpha)*fitness.ErrorVariance
	}

	// Calculate current fitness based on recent performance (last 20 predictions)
	fitness.CurrentFitness = ft.calculateCurrentFitness(fitness)
	fitness.LastUpdated = observationTime

	ft.logger.WithFields(logrus.Fields{
		"item_id":         itemID,
		"model":           model,
		"actual":          actualValue,
		"predicted":       predictedValue,
		"error":           error,
		"fitness_score":   fitnessScore,
		"current_fitness": fitness.CurrentFitness,
	}).Debug("Updated model fitness")

	return nil
}

// calculateCurrentFitness calculates current fitness based on recent performance
func (ft *FitnessTracker) calculateCurrentFitness(fitness *ModelFitness) float64 {
	if len(fitness.FitnessHistory) == 0 {
		return 0.5
	}

	// Use last 20 predictions for current fitness, or all if less than 20
	recentCount := int(math.Min(20, float64(len(fitness.FitnessHistory))))
	recentStart := len(fitness.FitnessHistory) - recentCount

	totalFitness := 0.0
	totalWeight := 0.0
	now := time.Now()

	for i := recentStart; i < len(fitness.FitnessHistory); i++ {
		point := fitness.FitnessHistory[i]

		// Apply time decay - more recent predictions have higher weight
		daysSince := now.Sub(point.Timestamp).Hours() / 24.0
		weight := math.Exp(-0.1 * daysSince) // 10% decay per day

		totalFitness += point.FitnessScore * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.5
	}

	return totalFitness / totalWeight
}

// GetModelFitness retrieves fitness data for a specific model
func (ft *FitnessTracker) GetModelFitness(itemID string, model PredictionModel) (*ModelFitness, error) {
	if itemFitness, exists := ft.modelFitness[itemID]; exists {
		if fitness, exists := itemFitness[model]; exists {
			return fitness, nil
		}
	}
	return nil, fmt.Errorf("fitness data not found for item %s model %s", itemID, model)
}

// GetAllModelFitness retrieves fitness data for all models of an item
func (ft *FitnessTracker) GetAllModelFitness(itemID string) map[PredictionModel]*ModelFitness {
	if itemFitness, exists := ft.modelFitness[itemID]; exists {
		return itemFitness
	}
	return make(map[PredictionModel]*ModelFitness)
}

// GetBestModel returns the model with the highest current fitness for an item
func (ft *FitnessTracker) GetBestModel(itemID string) (PredictionModel, float64, error) {
	itemFitness, exists := ft.modelFitness[itemID]
	if !exists {
		return "", 0, fmt.Errorf("no fitness data found for item %s", itemID)
	}

	var bestModel PredictionModel
	bestFitness := -1.0

	for model, fitness := range itemFitness {
		if fitness.CurrentFitness > bestFitness {
			bestFitness = fitness.CurrentFitness
			bestModel = model
		}
	}

	if bestFitness < 0 {
		return "", 0, fmt.Errorf("no valid fitness data found for item %s", itemID)
	}

	return bestModel, bestFitness, nil
}

// GetMultipleModelFitness retrieves fitness data for specified models (empty slice = all models)
func (ft *FitnessTracker) GetMultipleModelFitness(itemID string, models []PredictionModel) ([]*ModelFitness, error) {
	var result []*ModelFitness

	itemFitness := ft.modelFitness[itemID]
	if itemFitness == nil {
		return result, nil // Return empty slice if no fitness data exists
	}

	if len(models) == 0 {
		// Return all models for this item
		for _, fitness := range itemFitness {
			result = append(result, fitness)
		}
	} else {
		// Return specific models
		for _, model := range models {
			if fitness, exists := itemFitness[model]; exists {
				result = append(result, fitness)
			}
		}
	}

	return result, nil
}

// SetPredictionConfig sets the prediction configuration for an item
func (ft *FitnessTracker) SetPredictionConfig(config *PredictionConfig) error {
	if config.ItemID == "" {
		return fmt.Errorf("item_id is required")
	}

	// Validate enabled models
	for _, model := range config.EnabledModels {
		if !ft.isValidModel(model) {
			return fmt.Errorf("invalid model: %s", model)
		}
	}

	// Validate preferred model
	if config.PreferredModel != "" && !ft.isValidModel(config.PreferredModel) {
		return fmt.Errorf("invalid preferred model: %s", config.PreferredModel)
	}

	ft.configs[config.ItemID] = config
	ft.logger.WithFields(logrus.Fields{
		"item_id":         config.ItemID,
		"enabled_models":  config.EnabledModels,
		"preferred_model": config.PreferredModel,
		"auto_select":     config.AutoSelectBest,
	}).Debug("Updated prediction config")

	return nil
}

// GetPredictionConfig retrieves the prediction configuration for an item
func (ft *FitnessTracker) GetPredictionConfig(itemID string) (*PredictionConfig, error) {
	if config, exists := ft.configs[itemID]; exists {
		return config, nil
	}

	// Return default config if none exists
	defaultConfig := &PredictionConfig{
		ItemID: itemID,
		EnabledModels: []PredictionModel{
			ModelMarkov,
			ModelCroston,
			ModelDriftImpulse,
			ModelBayesian,
			ModelMemoryWindow,
		},
		PreferredModel: ModelMarkov,
		AutoSelectBest: true,
		GlobalSettings: make(map[string]string),
	}

	return defaultConfig, nil
}

// isValidModel checks if a model is valid
func (ft *FitnessTracker) isValidModel(model PredictionModel) bool {
	validModels := []PredictionModel{
		ModelMarkov,
		ModelCroston,
		ModelDriftImpulse,
		ModelBayesian,
		ModelMemoryWindow,
		ModelEventTrigger,
	}

	for _, validModel := range validModels {
		if model == validModel {
			return true
		}
	}

	return false
}

// GetRecommendedModel returns the best model to use for predictions based on fitness
func (ft *FitnessTracker) GetRecommendedModel(itemID string) (PredictionModel, error) {
	config, err := ft.GetPredictionConfig(itemID)
	if err != nil {
		return ModelMarkov, err // Default fallback
	}

	// If auto-select is disabled, use preferred model
	if !config.AutoSelectBest && config.PreferredModel != "" {
		// Check if preferred model is enabled
		if ft.ShouldUseModel(itemID, config.PreferredModel) {
			return config.PreferredModel, nil
		}
	}

	// Auto-select best model among enabled models
	itemFitness := ft.GetAllModelFitness(itemID)
	var bestModel PredictionModel
	bestFitness := -1.0

	for _, enabledModel := range config.EnabledModels {
		if fitness, exists := itemFitness[enabledModel]; exists {
			if fitness.CurrentFitness > bestFitness {
				bestFitness = fitness.CurrentFitness
				bestModel = enabledModel
			}
		}
	}

	if bestModel == "" {
		// No fitness data available, return first enabled model
		if len(config.EnabledModels) > 0 {
			return config.EnabledModels[0], nil
		}
		return ModelMarkov, nil // Ultimate fallback
	}

	return bestModel, nil
}

// ShouldUseModel checks if a model should be used for prediction based on configuration
func (ft *FitnessTracker) ShouldUseModel(itemID string, model PredictionModel) bool {
	config, err := ft.GetPredictionConfig(itemID)
	if err != nil {
		ft.logger.WithError(err).Warn("Failed to get prediction config, allowing all models")
		return true // Default to allowing all models if config is unavailable
	}

	// Check if model is in enabled models list
	for _, enabledModel := range config.EnabledModels {
		if enabledModel == model {
			return true
		}
	}

	return false
}
