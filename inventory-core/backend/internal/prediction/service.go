package prediction

import (
	"fmt"
	"time"
	
	"github.com/sirupsen/logrus"
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
		return nil, fmt.Errorf("no predictors found for item %s", itemName)
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
		return fmt.Errorf("no predictors found for item %s", itemName)
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
