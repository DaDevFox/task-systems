package prediction

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	errNoPredictorsFound = "no predictors found for item %s"
)

// PredictionService manages multiple predictors for inventory items
type PredictionService struct {
	predictors map[string]map[PredictionModel]Predictor // itemName -> model -> predictor
	logger     *logrus.Logger
}

// NewPredictionService creates a new prediction service
func NewPredictionService(logger *logrus.Logger) *PredictionService {
	return &PredictionService{
		predictors: make(map[string]map[PredictionModel]Predictor),
		logger:     logger,
	}
}

// CreatePredictor creates a new predictor for an item with the specified model
func (s *PredictionService) CreatePredictor(itemName string, model PredictionModel) (Predictor, error) {
	if s.predictors[itemName] == nil {
		s.predictors[itemName] = make(map[PredictionModel]Predictor)
	}

	var predictor Predictor

	switch model {
	case ModelMarkov:
		predictor = NewMarkovPredictor(itemName)
	case ModelCroston:
		predictor = NewCrostonPredictor(itemName)
	case ModelDriftImpulse:
		predictor = NewDriftImpulsePredictor(itemName)
	case ModelBayesian:
		predictor = NewBayesianPredictor(itemName)
	case ModelMemoryWindow:
		predictor = NewMemoryWindowPredictor(itemName)
	default:
		return nil, fmt.Errorf("unsupported prediction model: %s", model)
	}

	s.predictors[itemName][model] = predictor

	s.logger.WithFields(logrus.Fields{
		"item_name": itemName,
		"model":     model,
	}).Info("Created new predictor")

	return predictor, nil
}

// GetPredictor retrieves a predictor for an item and model
func (s *PredictionService) GetPredictor(itemName string, model PredictionModel) (Predictor, error) {
	if itemPredictors, exists := s.predictors[itemName]; exists {
		if predictor, exists := itemPredictors[model]; exists {
			return predictor, nil
		}
	}
	return nil, fmt.Errorf("predictor not found for item %s with model %s", itemName, model)
}

// GetBestPredictor returns the best performing predictor for an item
func (s *PredictionService) GetBestPredictor(itemName string) (Predictor, error) {
	itemPredictors, exists := s.predictors[itemName]
	if !exists {
		return nil, fmt.Errorf(errNoPredictorsFound, itemName)
	}

	var bestPredictor Predictor
	bestAccuracy := -1.0

	for _, predictor := range itemPredictors {
		if predictor.IsTrainingComplete() {
			status := predictor.GetTrainingStatus()
			if status.Accuracy > bestAccuracy {
				bestAccuracy = status.Accuracy
				bestPredictor = predictor
			}
		}
	}

	if bestPredictor == nil {
		// Return any available predictor if none are trained
		for _, predictor := range itemPredictors {
			return predictor, nil
		}
		return nil, fmt.Errorf("no predictors available for item %s", itemName)
	}

	return bestPredictor, nil
}

// UpdateAllPredictors updates all predictors for an item with a new report
func (s *PredictionService) UpdateAllPredictors(itemName string, report InventoryReport) error {
	itemPredictors, exists := s.predictors[itemName]
	if !exists {
		return fmt.Errorf(errNoPredictorsFound, itemName)
	}

	for model, predictor := range itemPredictors {
		predictor.Update(report)

		s.logger.WithFields(logrus.Fields{
			"item_name": itemName,
			"model":     model,
			"level":     report.Level,
			"timestamp": report.Timestamp,
		}).Debug("Updated predictor with new report")
	}

	return nil
}

// GetEnsemblePrediction combines predictions from multiple models
func (s *PredictionService) GetEnsemblePrediction(itemName string, targetTime time.Time) (InventoryEstimate, error) {
	itemPredictors, exists := s.predictors[itemName]
	if !exists {
		return InventoryEstimate{}, fmt.Errorf(errNoPredictorsFound, itemName)
	}

	var predictions []InventoryEstimate
	totalWeight := 0.0

	// Collect predictions from all trained models
	for _, predictor := range itemPredictors {
		if predictor.IsTrainingComplete() {
			prediction := predictor.Predict(targetTime)
			status := predictor.GetTrainingStatus()

			// Weight by accuracy
			weight := status.Accuracy
			if weight > 0 {
				predictions = append(predictions, prediction)
				totalWeight += weight
			}
		}
	}

	if len(predictions) == 0 {
		return InventoryEstimate{}, fmt.Errorf("no trained predictors available for item %s", itemName)
	}

	// Weighted ensemble prediction
	weightedEstimate := 0.0
	weightedLowerBound := 0.0
	weightedUpperBound := 0.0
	weightedConfidence := 0.0

	for i, prediction := range predictions {
		predictor := s.getPredictorByIndex(itemName, i)
		if predictor == nil {
			continue
		}

		status := predictor.GetTrainingStatus()
		weight := status.Accuracy / totalWeight

		weightedEstimate += prediction.Estimate * weight
		weightedLowerBound += prediction.LowerBound * weight
		weightedUpperBound += prediction.UpperBound * weight
		weightedConfidence += prediction.Confidence * weight
	}

	return InventoryEstimate{
		ItemName:       itemName,
		Estimate:       weightedEstimate,
		LowerBound:     weightedLowerBound,
		UpperBound:     weightedUpperBound,
		NextCheck:      targetTime,
		Confidence:     weightedConfidence,
		Recommendation: s.generateEnsembleRecommendation(weightedEstimate, weightedConfidence),
		ModelUsed:      "Ensemble",
	}, nil
}

func (s *PredictionService) getPredictorByIndex(itemName string, index int) Predictor {
	itemPredictors, exists := s.predictors[itemName]
	if !exists {
		return nil
	}

	i := 0
	for _, predictor := range itemPredictors {
		if predictor.IsTrainingComplete() {
			if i == index {
				return predictor
			}
			i++
		}
	}
	return nil
}

func (s *PredictionService) generateEnsembleRecommendation(estimate, confidence float64) string {
	if confidence < 0.5 {
		return "Ensemble prediction has low confidence - collect more data"
	}

	if estimate <= 1.0 {
		return "Ensemble models predict low stock - consider restocking"
	}

	if estimate <= 3.0 {
		return "Ensemble prediction shows moderate stock levels"
	}

	return "Ensemble models predict adequate inventory levels"
}

// StartTraining begins training for a specific predictor
func (s *PredictionService) StartTraining(itemName string, model PredictionModel, minSamples int, parameters map[string]float64) error {
	predictor, err := s.GetPredictor(itemName, model)
	if err != nil {
		// Create predictor if it doesn't exist
		predictor, err = s.CreatePredictor(itemName, model)
		if err != nil {
			return fmt.Errorf("failed to create predictor: %w", err)
		}
	}

	err = predictor.StartTraining(minSamples, parameters)
	if err != nil {
		return fmt.Errorf("failed to start training: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"item_name":   itemName,
		"model":       model,
		"min_samples": minSamples,
		"parameters":  parameters,
	}).Info("Started predictor training")

	return nil
}

// GetTrainingStatus retrieves training status for a specific predictor
func (s *PredictionService) GetTrainingStatus(itemName string, model PredictionModel) (TrainingStatus, error) {
	predictor, err := s.GetPredictor(itemName, model)
	if err != nil {
		return TrainingStatus{}, err
	}

	return predictor.GetTrainingStatus(), nil
}

// ListAvailableModels returns all available prediction models for an item
func (s *PredictionService) ListAvailableModels(itemName string) []PredictionModel {
	itemPredictors, exists := s.predictors[itemName]
	if !exists {
		return []PredictionModel{}
	}

	models := make([]PredictionModel, 0, len(itemPredictors))
	for model := range itemPredictors {
		models = append(models, model)
	}

	return models
}

// GetAllTrainingStatuses returns training status for all predictors of an item
func (s *PredictionService) GetAllTrainingStatuses(itemName string) map[PredictionModel]TrainingStatus {
	itemPredictors, exists := s.predictors[itemName]
	if !exists {
		return make(map[PredictionModel]TrainingStatus)
	}

	statuses := make(map[PredictionModel]TrainingStatus)
	for model, predictor := range itemPredictors {
		statuses[model] = predictor.GetTrainingStatus()
	}

	return statuses
}
