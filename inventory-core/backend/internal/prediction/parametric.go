package prediction

import (
	"fmt"
	"math"
	"time"

	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
	"github.com/sirupsen/logrus"
)

// ParametricPredictor implements consumption prediction using parametric models
// It supports linear equations and logistic differential equations
type ParametricPredictor struct {
	ItemName    string
	ModelConfig *pb.PredictionModelConfig
	LastReport  InventoryReport
	DataHistory []ConsumptionDataPoint
	logger      *logrus.Logger

	// Training fields
	trainingStage TrainingStage
	minSamples    int
	lastUpdated   time.Time
}

// ConsumptionDataPoint represents a data point for parametric model fitting
type ConsumptionDataPoint struct {
	Timestamp time.Time
	Level     float64
	TimeDelta float64 // Days since start of observation
}

// NewParametricPredictor creates a new parametric predictor
func NewParametricPredictor(itemName string, modelConfig *pb.PredictionModelConfig, logger *logrus.Logger) *ParametricPredictor {
	if modelConfig == nil {
		// Default to linear model with basic parameters
		modelConfig = &pb.PredictionModelConfig{
			ModelConfig: &pb.PredictionModelConfig_Parametric{
				Parametric: &pb.ParametricModel{
					ModelType: &pb.ParametricModel_Linear{
						Linear: &pb.LinearEquationModel{
							Slope:         -1.0, // Default consumption rate: 1 unit per day
							BaseLevel:     10.0, // Default starting level
							NoiseVariance: 0.5,  // Default uncertainty
						},
					},
				},
			},
		}
	}

	return &ParametricPredictor{
		ItemName:      itemName,
		ModelConfig:   modelConfig,
		DataHistory:   make([]ConsumptionDataPoint, 0),
		logger:        logger,
		trainingStage: TrainingStageCollecting,
		lastUpdated:   time.Now(),
	}
}

// StartTraining begins training for the parametric model
func (p *ParametricPredictor) StartTraining(minSamples int, parameters map[string]float64) error {
	p.minSamples = minSamples
	p.trainingStage = TrainingStageCollecting
	p.lastUpdated = time.Now()

	p.logger.WithFields(logrus.Fields{
		"item_name":   p.ItemName,
		"model_type":  p.getModelTypeName(),
		"min_samples": minSamples,
	}).Info("Started parametric model training")

	return nil
}

// GetTrainingStatus returns the current training status
func (p *ParametricPredictor) GetTrainingStatus() TrainingStatus {
	return TrainingStatus{
		Stage:            p.trainingStage,
		SamplesCollected: len(p.DataHistory),
		MinSamples:       p.minSamples,
		Accuracy:         p.calculateAccuracy(),
		LastUpdated:      p.lastUpdated,
		Parameters:       p.getParametersAsMap(),
	}
}

// IsTrainingComplete checks if training is complete
func (p *ParametricPredictor) IsTrainingComplete() bool {
	return p.trainingStage == TrainingStageTrained && len(p.DataHistory) >= p.minSamples
}

// GetModel returns the prediction model type
func (p *ParametricPredictor) GetModel() PredictionModel {
	return "Parametric"
}

// SetParameters updates the model parameters
func (p *ParametricPredictor) SetParameters(params map[string]float64) error {
	if p.ModelConfig == nil || p.ModelConfig.GetParametric() == nil {
		return fmt.Errorf("no parametric model configuration")
	}

	parametric := p.ModelConfig.GetParametric()

	switch model := parametric.GetModelType().(type) {
	case *pb.ParametricModel_Linear:
		return p.setLinearParameters(model.Linear, params)
	case *pb.ParametricModel_Logistic:
		return p.setLogisticParameters(model.Logistic, params)
	default:
		return fmt.Errorf("unsupported parametric model type")
	}
}

// setLinearParameters updates linear model parameters
func (p *ParametricPredictor) setLinearParameters(model *pb.LinearEquationModel, params map[string]float64) error {
	if slope, exists := params["slope"]; exists {
		model.Slope = slope
	}
	if baseLevel, exists := params["base_level"]; exists {
		model.BaseLevel = baseLevel
	}
	if noiseVariance, exists := params["noise_variance"]; exists {
		model.NoiseVariance = noiseVariance
	}
	return nil
}

// setLogisticParameters updates logistic model parameters
func (p *ParametricPredictor) setLogisticParameters(model *pb.LogisticEquationModel, params map[string]float64) error {
	if growthRate, exists := params["growth_rate"]; exists {
		model.GrowthRate = growthRate
	}
	if carryingCapacity, exists := params["carrying_capacity"]; exists {
		model.CarryingCapacity = carryingCapacity
	}
	if initialPopulation, exists := params["initial_population"]; exists {
		model.InitialPopulation = initialPopulation
	}
	if noiseVariance, exists := params["noise_variance"]; exists {
		model.NoiseVariance = noiseVariance
	}
	return nil
}

// GetParameters returns the current model parameters as a map
func (p *ParametricPredictor) GetParameters() map[string]float64 {
	return p.getParametersAsMap()
}

// getParametersAsMap converts model parameters to map format
func (p *ParametricPredictor) getParametersAsMap() map[string]float64 {
	params := make(map[string]float64)

	if p.ModelConfig == nil || p.ModelConfig.GetParametric() == nil {
		return params
	}

	parametric := p.ModelConfig.GetParametric()

	switch model := parametric.GetModelType().(type) {
	case *pb.ParametricModel_Linear:
		params["slope"] = model.Linear.Slope
		params["base_level"] = model.Linear.BaseLevel
		params["noise_variance"] = model.Linear.NoiseVariance
	case *pb.ParametricModel_Logistic:
		params["growth_rate"] = model.Logistic.GrowthRate
		params["carrying_capacity"] = model.Logistic.CarryingCapacity
		params["initial_population"] = model.Logistic.InitialPopulation
		params["noise_variance"] = model.Logistic.NoiseVariance
	}

	return params
}

// Predict generates a consumption prediction using the parametric model
func (p *ParametricPredictor) Predict(targetTime time.Time) InventoryEstimate {
	if !p.IsTrainingComplete() {
		return InventoryEstimate{
			ItemName:       p.ItemName,
			Estimate:       p.LastReport.Level,
			LowerBound:     p.LastReport.Level * 0.8,
			UpperBound:     p.LastReport.Level * 1.2,
			NextCheck:      targetTime,
			Confidence:     0.4,
			Recommendation: "Collecting data for parametric model training",
			ModelUsed:      "Parametric",
		}
	}

	daysAhead := targetTime.Sub(p.LastReport.Timestamp).Hours() / 24.0
	estimate, lowerBound, upperBound := p.calculatePrediction(daysAhead)
	confidence := p.calculateConfidence()

	return InventoryEstimate{
		ItemName:       p.ItemName,
		Estimate:       estimate,
		LowerBound:     lowerBound,
		UpperBound:     upperBound,
		NextCheck:      targetTime,
		Confidence:     confidence,
		Recommendation: p.generateRecommendation(estimate, daysAhead),
		ModelUsed:      "Parametric",
	}
}

// calculatePrediction computes the prediction based on the parametric model
func (p *ParametricPredictor) calculatePrediction(daysAhead float64) (estimate, lowerBound, upperBound float64) {
	if p.ModelConfig == nil || p.ModelConfig.GetParametric() == nil {
		return p.LastReport.Level, p.LastReport.Level * 0.8, p.LastReport.Level * 1.2
	}

	parametric := p.ModelConfig.GetParametric()

	switch model := parametric.GetModelType().(type) {
	case *pb.ParametricModel_Linear:
		return p.calculateLinearPrediction(model.Linear, daysAhead)
	case *pb.ParametricModel_Logistic:
		return p.calculateLogisticPrediction(model.Logistic, daysAhead)
	default:
		return p.LastReport.Level, p.LastReport.Level * 0.8, p.LastReport.Level * 1.2
	}
}

// calculateLinearPrediction computes prediction using linear equation: y = base_level + slope * time
func (p *ParametricPredictor) calculateLinearPrediction(model *pb.LinearEquationModel, daysAhead float64) (estimate, lowerBound, upperBound float64) {
	// Linear prediction: level(t) = current_level + slope * t
	estimate = math.Max(0, p.LastReport.Level+model.Slope*daysAhead)

	// Calculate uncertainty bounds using noise variance
	stdDev := math.Sqrt(model.NoiseVariance * daysAhead)
	lowerBound = math.Max(0, estimate-1.96*stdDev) // 95% confidence interval
	upperBound = estimate + 1.96*stdDev

	return estimate, lowerBound, upperBound
}

// calculateLogisticPrediction computes prediction using logistic equation: dP/dt = r*P*(1 - P/K)
func (p *ParametricPredictor) calculateLogisticPrediction(model *pb.LogisticEquationModel, daysAhead float64) (estimate, lowerBound, upperBound float64) {
	// Logistic equation solution: P(t) = K * P0 / (P0 + (K - P0) * exp(-r*t))
	// Where P0 = current_level, K = carrying_capacity, r = growth_rate

	P0 := p.LastReport.Level
	K := model.CarryingCapacity
	r := model.GrowthRate
	t := daysAhead

	if K <= 0 || P0 <= 0 {
		return p.calculateLinearPrediction(&pb.LinearEquationModel{
			Slope:         -1.0,
			BaseLevel:     P0,
			NoiseVariance: model.NoiseVariance,
		}, daysAhead)
	}

	// Logistic growth/decay solution
	denominator := P0 + (K-P0)*math.Exp(-r*t)
	if denominator <= 0 {
		estimate = K // Approach carrying capacity
	} else {
		estimate = math.Max(0, (K*P0)/denominator)
	}

	// Calculate uncertainty bounds
	stdDev := math.Sqrt(model.NoiseVariance * daysAhead)
	lowerBound = math.Max(0, estimate-1.96*stdDev)
	upperBound = estimate + 1.96*stdDev

	return estimate, lowerBound, upperBound
}

// Update adds a new data point and potentially refits the model
func (p *ParametricPredictor) Update(report InventoryReport) {
	// Add data point to history
	var timeDelta float64
	if len(p.DataHistory) == 0 {
		timeDelta = 0
	} else {
		timeDelta = report.Timestamp.Sub(p.DataHistory[0].Timestamp).Hours() / 24.0
	}

	dataPoint := ConsumptionDataPoint{
		Timestamp: report.Timestamp,
		Level:     report.Level,
		TimeDelta: timeDelta,
	}

	p.DataHistory = append(p.DataHistory, dataPoint)
	p.LastReport = report
	p.lastUpdated = time.Now()

	// Log the update
	p.logger.WithFields(logrus.Fields{
		"item_name":   p.ItemName,
		"level":       report.Level,
		"data_points": len(p.DataHistory),
		"model_type":  p.getModelTypeName(),
	}).Debug("Updated parametric model with new data point")

	// Check if training should complete
	if p.trainingStage == TrainingStageCollecting && len(p.DataHistory) >= p.minSamples {
		p.trainingStage = TrainingStageLearning
		p.fitModelToData()
		p.trainingStage = TrainingStageTrained

		p.logger.WithFields(logrus.Fields{
			"item_name":   p.ItemName,
			"model_type":  p.getModelTypeName(),
			"data_points": len(p.DataHistory),
			"accuracy":    p.calculateAccuracy(),
		}).Info("Completed parametric model training")
	}
}

// fitModelToData performs parameter estimation based on observed data
func (p *ParametricPredictor) fitModelToData() {
	if len(p.DataHistory) < 2 || p.ModelConfig == nil || p.ModelConfig.GetParametric() == nil {
		return
	}

	parametric := p.ModelConfig.GetParametric()

	switch model := parametric.GetModelType().(type) {
	case *pb.ParametricModel_Linear:
		p.fitLinearModel(model.Linear)
	case *pb.ParametricModel_Logistic:
		p.fitLogisticModel(model.Logistic)
	}
}

// fitLinearModel estimates linear model parameters using least squares
func (p *ParametricPredictor) fitLinearModel(model *pb.LinearEquationModel) {
	n := len(p.DataHistory)
	if n < 2 {
		return
	}

	// Calculate means
	sumX := 0.0 // time
	sumY := 0.0 // level
	for _, point := range p.DataHistory {
		sumX += point.TimeDelta
		sumY += point.Level
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	// Calculate slope and intercept using least squares
	numerator := 0.0
	denominator := 0.0
	for _, point := range p.DataHistory {
		dx := point.TimeDelta - meanX
		dy := point.Level - meanY
		numerator += dx * dy
		denominator += dx * dx
	}

	if denominator != 0 {
		model.Slope = numerator / denominator
		model.BaseLevel = meanY - model.Slope*meanX

		// Update base level to match current level
		if len(p.DataHistory) > 0 {
			lastPoint := p.DataHistory[len(p.DataHistory)-1]
			model.BaseLevel = lastPoint.Level - model.Slope*lastPoint.TimeDelta
		}

		// Estimate noise variance from residuals
		sumSquaredResiduals := 0.0
		for _, point := range p.DataHistory {
			predicted := model.BaseLevel + model.Slope*point.TimeDelta
			residual := point.Level - predicted
			sumSquaredResiduals += residual * residual
		}
		model.NoiseVariance = sumSquaredResiduals / float64(n)
	}

	p.logger.WithFields(logrus.Fields{
		"item_name":      p.ItemName,
		"slope":          model.Slope,
		"base_level":     model.BaseLevel,
		"noise_variance": model.NoiseVariance,
	}).Info("Fitted linear parametric model")
}

// fitLogisticModel estimates logistic model parameters (simplified fitting)
func (p *ParametricPredictor) fitLogisticModel(model *pb.LogisticEquationModel) {
	n := len(p.DataHistory)
	if n < 3 {
		return
	}

	// For simplicity, use basic heuristics to estimate parameters
	// In production, would use more sophisticated curve fitting algorithms

	levels := make([]float64, n)
	for i, point := range p.DataHistory {
		levels[i] = point.Level
	}

	// Estimate carrying capacity as max level + buffer
	maxLevel := 0.0
	minLevel := math.Inf(1)
	for _, level := range levels {
		maxLevel = math.Max(maxLevel, level)
		minLevel = math.Min(minLevel, level)
	}

	model.CarryingCapacity = maxLevel * 1.2 // 20% buffer above max observed
	model.InitialPopulation = levels[0]

	// Estimate growth rate based on initial rate of change
	if len(p.DataHistory) >= 2 {
		deltaLevel := levels[1] - levels[0]
		deltaTime := p.DataHistory[1].TimeDelta - p.DataHistory[0].TimeDelta
		if deltaTime > 0 && levels[0] > 0 {
			// Estimate: dr/dt ≈ r*P*(1-P/K)
			model.GrowthRate = (deltaLevel / deltaTime) / (levels[0] * (1 - levels[0]/model.CarryingCapacity))
		}
	}

	// Ensure reasonable bounds
	model.GrowthRate = math.Max(-1.0, math.Min(1.0, model.GrowthRate))

	// Estimate noise variance from residuals (simplified)
	sumSquaredResiduals := 0.0
	for _, point := range p.DataHistory {
		// Simple prediction for residual calculation
		predicted := levels[0] // Simplified
		residual := point.Level - predicted
		sumSquaredResiduals += residual * residual
	}
	model.NoiseVariance = sumSquaredResiduals / float64(n)

	p.logger.WithFields(logrus.Fields{
		"item_name":         p.ItemName,
		"growth_rate":       model.GrowthRate,
		"carrying_capacity": model.CarryingCapacity,
		"initial_pop":       model.InitialPopulation,
		"noise_variance":    model.NoiseVariance,
	}).Info("Fitted logistic parametric model")
}

// calculateAccuracy computes model accuracy based on prediction vs actual
func (p *ParametricPredictor) calculateAccuracy() float64 {
	if len(p.DataHistory) < 3 {
		return 0.0
	}

	// Use cross-validation style accuracy: predict each point using previous points
	totalError := 0.0
	validPredictions := 0

	for i := 2; i < len(p.DataHistory); i++ {
		actual := p.DataHistory[i].Level
		if actual > 0 {
			// Make prediction using data up to point i-1
			baseTime := p.DataHistory[i-1].Timestamp
			targetTime := p.DataHistory[i].Timestamp
			daysAhead := targetTime.Sub(baseTime).Hours() / 24.0

			// Temporarily adjust for prediction from earlier point
			oldReport := p.LastReport
			p.LastReport = InventoryReport{
				ItemName:  p.ItemName,
				Timestamp: baseTime,
				Level:     p.DataHistory[i-1].Level,
			}

			predicted, _, _ := p.calculatePrediction(daysAhead)
			p.LastReport = oldReport // Restore

			relativeError := math.Abs(predicted-actual) / actual
			totalError += relativeError
			validPredictions++
		}
	}

	if validPredictions == 0 {
		return 0.0
	}

	// Convert error to accuracy (0 to 1 scale)
	averageError := totalError / float64(validPredictions)
	accuracy := math.Max(0.0, math.Min(1.0, 1.0-averageError))
	return accuracy
}

// calculateConfidence returns confidence level based on model fit and data quality
func (p *ParametricPredictor) calculateConfidence() float64 {
	if len(p.DataHistory) < p.minSamples {
		return 0.4
	}

	// Base confidence on training accuracy
	accuracy := p.calculateAccuracy()

	// Adjust for amount of training data
	dataFactor := math.Min(1.0, float64(len(p.DataHistory))/float64(p.minSamples*2))

	// Combine factors
	confidence := (accuracy * 0.7) + (dataFactor * 0.3)

	return math.Max(0.4, math.Min(0.95, confidence))
}

// generateRecommendation creates a human-readable recommendation
func (p *ParametricPredictor) generateRecommendation(estimate float64, daysAhead float64) string {
	modelType := p.getModelTypeName()

	if estimate <= 0 {
		return fmt.Sprintf("%s model predicts depletion in %.1f days", modelType, daysAhead)
	}
	if estimate <= 2.0 {
		return fmt.Sprintf("%s model indicates low stock (%.1f units remaining)", modelType, estimate)
	}
	if estimate <= 5.0 {
		return fmt.Sprintf("%s model shows moderate stock levels", modelType)
	}
	return fmt.Sprintf("%s model predicts adequate inventory", modelType)
}

// getModelTypeName returns a human-readable model type name
func (p *ParametricPredictor) getModelTypeName() string {
	if p.ModelConfig == nil || p.ModelConfig.GetParametric() == nil {
		return "Parametric"
	}

	parametric := p.ModelConfig.GetParametric()
	switch parametric.GetModelType().(type) {
	case *pb.ParametricModel_Linear:
		return "Linear"
	case *pb.ParametricModel_Logistic:
		return "Logistic"
	default:
		return "Parametric"
	}
}

// Name returns the predictor name
func (p *ParametricPredictor) Name() string {
	return fmt.Sprintf("Parametric-%s", p.getModelTypeName())
}
