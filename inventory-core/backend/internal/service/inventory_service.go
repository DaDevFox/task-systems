package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/prediction"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/proto/proto"
	"github.com/DaDevFox/task-systems/shared/events"
)

const (
	errDomainToPbConversion = "domain to protobuf conversion failed"
	errResponseFormatting   = "response formatting failed"
	errItemIdRequired       = "item_id is required"
)

// InventoryService implements the gRPC InventoryService interface
type InventoryService struct {
	pb.UnimplementedInventoryServiceServer

	repo          repository.InventoryRepository
	eventBus      *events.EventBus
	logger        *logrus.Logger
	predictionSvc *prediction.PredictionService
}

// NewInventoryService creates a new inventory service instance
func NewInventoryService(
	repo repository.InventoryRepository,
	eventBus *events.EventBus,
	logger *logrus.Logger,
) *InventoryService {
	return &InventoryService{
		repo:          repo,
		eventBus:      eventBus,
		logger:        logger,
		predictionSvc: prediction.NewPredictionService(logger),
	}
}

// AddInventoryItem creates a new inventory item
func (s *InventoryService) AddInventoryItem(ctx context.Context, req *pb.AddInventoryItemRequest) (*pb.AddInventoryItemResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "item name is required")
	}

	if req.UnitId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "unit_id is required")
	}

	// Validate unit exists
	_, err := s.repo.GetUnit(ctx, req.UnitId)
	if err != nil {
		s.logger.WithError(err).WithField("unit_id", req.UnitId).Error("unit validation failed")
		return nil, status.Errorf(codes.InvalidArgument, "invalid unit_id: %s", req.UnitId)
	}

	item := &domain.InventoryItem{
		Name:              req.Name,
		Description:       req.Description,
		CurrentLevel:      req.InitialLevel,
		MaxCapacity:       req.MaxCapacity,
		LowStockThreshold: req.LowStockThreshold,
		UnitID:            req.UnitId,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Metadata:          req.Metadata,
	}

	if item.Metadata == nil {
		item.Metadata = make(map[string]string)
	}

	err = s.repo.AddItem(ctx, item)
	if err != nil {
		s.logger.WithError(err).WithField("item_name", req.Name).Error("failed to add inventory item")
		return nil, status.Errorf(codes.Internal, "failed to create inventory item")
	}

	s.logger.WithFields(logrus.Fields{
		"item_id":   item.ID,
		"item_name": item.Name,
		"level":     item.CurrentLevel,
		"unit_id":   item.UnitID,
	}).Info("inventory item created")

	pbItem, err := s.domainToPbItem(item)
	if err != nil {
		s.logger.WithError(err).Error(errDomainToPbConversion)
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
	}

	return &pb.AddInventoryItemResponse{Item: pbItem}, nil
}

// GetInventoryItem retrieves a single inventory item by ID
func (s *InventoryService) GetInventoryItem(ctx context.Context, req *pb.GetInventoryItemRequest) (*pb.GetInventoryItemResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	}

	item, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item")
		return nil, status.Errorf(codes.NotFound, "inventory item not found")
	}

	pbItem, err := s.domainToPbItem(item)
	if err != nil {
		s.logger.WithError(err).Error(errDomainToPbConversion)
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
	}

	return &pb.GetInventoryItemResponse{Item: pbItem}, nil
}

// UpdateInventoryLevel updates the quantity of an inventory item
func (s *InventoryService) UpdateInventoryLevel(ctx context.Context, req *pb.UpdateInventoryLevelRequest) (*pb.UpdateInventoryLevelResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	}

	item, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item for update")
		return nil, status.Errorf(codes.NotFound, "inventory item not found")
	}

	previousLevel := item.CurrentLevel
	wasLowStock := item.IsLowStock()

	item.CurrentLevel = req.NewLevel
	item.UpdatedAt = time.Now()

	// Record consumption if requested and level decreased
	if req.RecordConsumption && req.NewLevel < previousLevel {
		consumptionAmount := previousLevel - req.NewLevel
		item.AddConsumptionRecord(consumptionAmount, item.UnitID, req.Reason)
	}

	err = s.repo.UpdateItem(ctx, item)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to update inventory item")
		return nil, status.Errorf(codes.Internal, "failed to update inventory level")
	}

	levelChanged := previousLevel != req.NewLevel
	belowThreshold := item.IsLowStock()

	// Publish inventory level changed event
	if levelChanged {
		err = s.eventBus.PublishInventoryLevelChanged(
			ctx,
			item.ID,
			item.Name,
			previousLevel,
			item.CurrentLevel,
			item.UnitID,
			item.LowStockThreshold,
		)
		if err != nil {
			s.logger.WithError(err).Error("failed to publish inventory level changed event")
		}

		s.logger.WithFields(logrus.Fields{
			"item_id":         item.ID,
			"item_name":       item.Name,
			"previous_level":  previousLevel,
			"new_level":       item.CurrentLevel,
			"below_threshold": belowThreshold,
			"was_low_stock":   wasLowStock,
		}).Info("inventory level updated")
	}

	pbItem, err := s.domainToPbItem(item)
	if err != nil {
		s.logger.WithError(err).Error(errDomainToPbConversion)
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
	}

	return &pb.UpdateInventoryLevelResponse{
		Item:           pbItem,
		LevelChanged:   levelChanged,
		BelowThreshold: belowThreshold,
	}, nil
}

// ConfigureInventoryItem updates inventory item configuration (excluding level)
func (s *InventoryService) ConfigureInventoryItem(ctx context.Context, req *pb.ConfigureInventoryItemRequest) (*pb.ConfigureInventoryItemResponse, error) {
	switch {
	case req.ItemId == "":
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	case req.Name == "":
		return nil, status.Errorf(codes.InvalidArgument, "item name is required")
	case req.UnitId == "":
		return nil, status.Errorf(codes.InvalidArgument, "unit_id is required")
	}

	// Get the existing item
	item, err := s.repo.GetItem(ctx, req.ItemId)
	switch {
	case err != nil:
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item for configuration")
		return nil, status.Errorf(codes.NotFound, "inventory item not found")
	}

	// Validate unit exists if it's being changed
	if req.UnitId != item.UnitID {
		_, err := s.repo.GetUnit(ctx, req.UnitId)
		switch {
		case err != nil:
			s.logger.WithError(err).WithField("unit_id", req.UnitId).Error("unit validation failed")
			return nil, status.Errorf(codes.InvalidArgument, "invalid unit_id: %s", req.UnitId)
		}
	}

	// Update the configurable fields
	previousName := item.Name
	previousDescription := item.Description
	previousMaxCapacity := item.MaxCapacity
	previousLowStockThreshold := item.LowStockThreshold
	previousUnitID := item.UnitID
	
	item.Name = req.Name
	item.Description = req.Description
	item.MaxCapacity = req.MaxCapacity
	item.LowStockThreshold = req.LowStockThreshold
	item.UnitID = req.UnitId
	item.UpdatedAt = time.Now()

	// Update metadata (replace completely)
	if req.Metadata != nil {
		item.Metadata = req.Metadata
	}
	switch {
	case item.Metadata == nil:
		item.Metadata = make(map[string]string)
	}

	err = s.repo.UpdateItem(ctx, item)
	switch {
	case err != nil:
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to update inventory item configuration")
		return nil, status.Errorf(codes.Internal, "failed to configure inventory item")
	}

	s.logger.WithFields(logrus.Fields{
		"item_id":                     item.ID,
		"item_name":                   item.Name,
		"previous_name":               previousName,
		"previous_description":        previousDescription,
		"previous_max_capacity":       previousMaxCapacity,
		"previous_low_stock_threshold": previousLowStockThreshold,
		"previous_unit_id":            previousUnitID,
		"new_name":                    item.Name,
		"new_description":             item.Description,
		"new_max_capacity":            item.MaxCapacity,
		"new_low_stock_threshold":     item.LowStockThreshold,
		"new_unit_id":                 item.UnitID,
	}).Info("inventory item configured")

	pbItem, err := s.domainToPbItem(item)
	switch {
	case err != nil:
		s.logger.WithError(err).Error(errDomainToPbConversion)
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
	}

	return &pb.ConfigureInventoryItemResponse{Item: pbItem}, nil
}

// GetInventoryStatus provides overview of inventory state
func (s *InventoryService) GetInventoryStatus(ctx context.Context, req *pb.GetInventoryStatusRequest) (*pb.GetInventoryStatusResponse, error) {
	var items []*domain.InventoryItem
	var err error

	if len(req.ItemIds) > 0 {
		// Get specific items
		for _, itemID := range req.ItemIds {
			item, getErr := s.repo.GetItem(ctx, itemID)
			if getErr != nil {
				s.logger.WithError(getErr).WithField("item_id", itemID).Warn("failed to get specific item for status")
				continue
			}
			items = append(items, item)
		}
	} else if req.IncludeLowStockOnly {
		// Get only low stock items
		items, err = s.repo.GetLowStockItems(ctx)
	} else {
		// Get all items
		items, err = s.repo.GetAllItems(ctx)
	}

	if err != nil {
		s.logger.WithError(err).Error("failed to get inventory items for status")
		return nil, status.Errorf(codes.Internal, "failed to retrieve inventory status")
	}

	// Categorize items
	var lowStockItems []*domain.InventoryItem
	var emptyItems []*domain.InventoryItem

	for _, item := range items {
		if item.IsEmpty() {
			emptyItems = append(emptyItems, item)
		} else if item.IsLowStock() {
			lowStockItems = append(lowStockItems, item)
		}
	}

	// Convert to protobuf
	pbItems, err := s.domainToPbItems(items)
	if err != nil {
		s.logger.WithError(err).Error("failed to convert items to protobuf")
		return nil, status.Errorf(codes.Internal, "response formatting failed")
	}

	pbLowStockItems, err := s.domainToPbItems(lowStockItems)
	if err != nil {
		s.logger.WithError(err).Error("failed to convert low stock items to protobuf")
		return nil, status.Errorf(codes.Internal, "response formatting failed")
	}

	pbEmptyItems, err := s.domainToPbItems(emptyItems)
	if err != nil {
		s.logger.WithError(err).Error("failed to convert empty items to protobuf")
		return nil, status.Errorf(codes.Internal, "response formatting failed")
	}

	status := &pb.InventoryStatus{
		Items:         pbItems,
		LowStockItems: pbLowStockItems,
		EmptyItems:    pbEmptyItems,
		TotalItems:    int32(len(items)),
		LastUpdated:   timestamppb.Now(),
	}

	return &pb.GetInventoryStatusResponse{Status: status}, nil
}

// SubmitInventoryReport submits a user report for training or updates
func (s *InventoryService) SubmitInventoryReport(ctx context.Context, req *pb.SubmitInventoryReportRequest) (*pb.SubmitInventoryReportResponse, error) {
	if req.Report == nil {
		return nil, status.Errorf(codes.InvalidArgument, "report is required")
	}

	if req.Report.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "item_id is required")
	}

	// Verify item exists
	_, err := s.repo.GetItem(ctx, req.Report.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.Report.ItemId).Error("item not found")
		return nil, status.Errorf(codes.NotFound, "item not found: %s", req.Report.ItemId)
	}

	// Convert to domain report
	report := prediction.InventoryReport{
		ItemName:  req.Report.ItemId,
		Timestamp: req.Report.Timestamp.AsTime(),
		Level:     req.Report.Level,
		Context:   req.Report.Context,
		Metadata:  req.Report.Metadata,
	}

	// Update all predictors for this item
	err = s.predictionSvc.UpdateAllPredictors(req.Report.ItemId, report)
	trainingUpdated := err == nil

	// Get training status for the best predictor
	bestPredictor, err := s.predictionSvc.GetBestPredictor(req.Report.ItemId)
	var trainingStatus *pb.PredictionTrainingStatus

	if err == nil {
		status := bestPredictor.GetTrainingStatus()
		trainingStatus = s.domainToPbTrainingStatus(req.Report.ItemId, status, bestPredictor.GetModel())
	}

	s.logger.WithFields(logrus.Fields{
		"item_id":          req.Report.ItemId,
		"level":            req.Report.Level,
		"training_updated": trainingUpdated,
	}).Info("processed inventory report")

	return &pb.SubmitInventoryReportResponse{
		TrainingUpdated: trainingUpdated,
		TrainingStatus:  trainingStatus,
	}, nil
}

// GetPredictionTrainingStatus retrieves training status for an item
func (s *InventoryService) GetPredictionTrainingStatus(ctx context.Context, req *pb.GetPredictionTrainingStatusRequest) (*pb.GetPredictionTrainingStatusResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "item_id is required")
	}

	// Get the best predictor for this item
	bestPredictor, err := s.predictionSvc.GetBestPredictor(req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get predictor")
		return nil, status.Errorf(codes.NotFound, "no predictors found for item: %s", req.ItemId)
	}

	status := bestPredictor.GetTrainingStatus()
	pbStatus := s.domainToPbTrainingStatus(req.ItemId, status, bestPredictor.GetModel())

	return &pb.GetPredictionTrainingStatusResponse{
		Status: pbStatus,
	}, nil
}

// StartTraining begins training for an item with a specific model
func (s *InventoryService) StartTraining(ctx context.Context, req *pb.StartTrainingRequest) (*pb.StartTrainingResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "item_id is required")
	}

	// Verify item exists
	_, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("item not found")
		return nil, status.Errorf(codes.NotFound, "item not found: %s", req.ItemId)
	}

	// Convert protobuf model to domain model
	model := s.pbToDomainModel(req.Model)

	// Start training
	minSamples := int(req.MinSamples)
	if minSamples <= 0 {
		minSamples = 10 // Default minimum samples
	}

	err = s.predictionSvc.StartTraining(req.ItemId, model, minSamples, req.Parameters)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"item_id": req.ItemId,
			"model":   req.Model,
		}).Error("failed to start training")
		return nil, status.Errorf(codes.Internal, "failed to start training: %v", err)
	}

	// Get updated training status
	predictor, _ := s.predictionSvc.GetPredictor(req.ItemId, model)
	var trainingStatus *pb.PredictionTrainingStatus
	if predictor != nil {
		status := predictor.GetTrainingStatus()
		trainingStatus = s.domainToPbTrainingStatus(req.ItemId, status, model)
	}

	s.logger.WithFields(logrus.Fields{
		"item_id":     req.ItemId,
		"model":       req.Model,
		"min_samples": minSamples,
	}).Info("started predictor training")

	return &pb.StartTrainingResponse{
		Status: trainingStatus,
	}, nil
}

func (s *InventoryService) ListInventoryItems(ctx context.Context, req *pb.ListInventoryItemsRequest) (*pb.ListInventoryItemsResponse, error) {
	filters := repository.ListFilters{
		LowStockOnly:   req.LowStockOnly,
		UnitTypeFilter: req.UnitTypeFilter,
		Limit:          int(req.Limit),
		Offset:         int(req.Offset),
	}

	if filters.Limit <= 0 {
		filters.Limit = 50 // Default limit
	}

	items, totalCount, err := s.repo.ListItems(ctx, filters)
	if err != nil {
		s.logger.WithError(err).Error("failed to list inventory items")
		return nil, status.Errorf(codes.Internal, "failed to list inventory items")
	}

	pbItems, err := s.domainToPbItems(items)
	if err != nil {
		s.logger.WithError(err).Error("failed to convert items to protobuf")
		return nil, status.Errorf(codes.Internal, "response formatting failed")
	}

	return &pb.ListInventoryItemsResponse{
		Items:      pbItems,
		TotalCount: int32(totalCount),
	}, nil
}

// GetAdvancedPrediction generates detailed predictions with multiple models
func (s *InventoryService) GetAdvancedPrediction(ctx context.Context, req *pb.GetAdvancedPredictionRequest) (*pb.GetAdvancedPredictionResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "item_id is required")
	}

	targetTime := time.Now().Add(24 * time.Hour) // Default to 24 hours ahead
	if req.TargetTime != nil {
		targetTime = req.TargetTime.AsTime()
	}

	// Get predictions from all available models or specified models
	var predictions []*pb.ConsumptionPrediction

	if len(req.Models) > 0 {
		// Use specified models
		for _, pbModel := range req.Models {
			model := s.pbToDomainModel(pbModel)
			predictor, err := s.predictionSvc.GetPredictor(req.ItemId, model)
			if err == nil && predictor.IsTrainingComplete() {
				estimate := predictor.Predict(targetTime)
				pbPrediction := s.domainToPbPrediction(estimate)
				predictions = append(predictions, pbPrediction)
			}
		}
	} else {
		// Use all available models
		models := s.predictionSvc.ListAvailableModels(req.ItemId)
		for _, model := range models {
			predictor, err := s.predictionSvc.GetPredictor(req.ItemId, model)
			if err == nil && predictor.IsTrainingComplete() {
				estimate := predictor.Predict(targetTime)
				pbPrediction := s.domainToPbPrediction(estimate)
				predictions = append(predictions, pbPrediction)
			}
		}
	}

	// Generate ensemble prediction
	var consensusPrediction *pb.ConsumptionPrediction
	if len(predictions) > 0 {
		ensemble, err := s.predictionSvc.GetEnsemblePrediction(req.ItemId, targetTime)
		if err == nil {
			consensusPrediction = s.domainToPbPrediction(ensemble)
		}
	}

	return &pb.GetAdvancedPredictionResponse{
		Predictions:         predictions,
		ConsensusPrediction: consensusPrediction,
	}, nil
}

// Helper methods for domain/protobuf conversion

func (s *InventoryService) domainToPbItem(item *domain.InventoryItem) (*pb.InventoryItem, error) {
	if item == nil {
		return nil, fmt.Errorf("item cannot be nil")
	}

	pbItem := &pb.InventoryItem{
		Id:                item.ID,
		Name:              item.Name,
		Description:       item.Description,
		CurrentLevel:      item.CurrentLevel,
		MaxCapacity:       item.MaxCapacity,
		LowStockThreshold: item.LowStockThreshold,
		UnitId:            item.UnitID,
		AlternateUnitIds:  item.AlternateUnitIDs,
		CreatedAt:         timestamppb.New(item.CreatedAt),
		UpdatedAt:         timestamppb.New(item.UpdatedAt),
		Metadata:          item.Metadata,
	}

	// Convert consumption behavior
	if item.ConsumptionBehavior != nil {
		pbItem.ConsumptionBehavior = &pb.ConsumptionBehavior{
			Pattern:           pb.ConsumptionPattern(item.ConsumptionBehavior.Pattern),
			AverageRatePerDay: item.ConsumptionBehavior.AverageRatePerDay,
			Variance:          item.ConsumptionBehavior.Variance,
			SeasonalFactors:   item.ConsumptionBehavior.SeasonalFactors,
			LastUpdated:       timestamppb.New(item.ConsumptionBehavior.LastUpdated),
		}
	}

	// Convert consumption history
	for _, record := range item.ConsumptionHistory {
		pbRecord := &pb.ConsumptionRecord{
			Timestamp:      timestamppb.New(record.Timestamp),
			AmountConsumed: record.AmountConsumed,
			UnitId:         record.UnitID,
			Reason:         record.Reason,
		}
		pbItem.ConsumptionHistory = append(pbItem.ConsumptionHistory, pbRecord)
	}

	return pbItem, nil
}

func (s *InventoryService) domainToPbItems(items []*domain.InventoryItem) ([]*pb.InventoryItem, error) {
	if len(items) == 0 {
		return []*pb.InventoryItem{}, nil
	}

	pbItems := make([]*pb.InventoryItem, 0, len(items))
	for _, item := range items {
		pbItem, err := s.domainToPbItem(item)
		if err != nil {
			return nil, fmt.Errorf("failed to convert item %s: %w", item.ID, err)
		}
		pbItems = append(pbItems, pbItem)
	}

	return pbItems, nil
}

func (s *InventoryService) domainToPbTrainingStatus(itemId string, status prediction.TrainingStatus, model prediction.PredictionModel) *pb.PredictionTrainingStatus {
	return &pb.PredictionTrainingStatus{
		ItemId:             itemId,
		Stage:              s.domainToPbTrainingStage(status.Stage),
		ActiveModel:        s.domainToPbModel(model),
		TrainingSamples:    int32(status.SamplesCollected),
		MinSamplesRequired: int32(status.MinSamples),
		TrainingAccuracy:   status.Accuracy,
		LastUpdated:        timestamppb.New(status.LastUpdated),
		ModelParameters:    status.Parameters,
	}
}

func (s *InventoryService) domainToPbModel(model prediction.PredictionModel) pb.PredictionModel {
	switch model {
	case prediction.ModelMarkov:
		return pb.PredictionModel_PREDICTION_MODEL_MARKOV
	case prediction.ModelCroston:
		return pb.PredictionModel_PREDICTION_MODEL_CROSTON
	case prediction.ModelDriftImpulse:
		return pb.PredictionModel_PREDICTION_MODEL_DRIFT_IMPULSE
	case prediction.ModelBayesian:
		return pb.PredictionModel_PREDICTION_MODEL_BAYESIAN
	case prediction.ModelMemoryWindow:
		return pb.PredictionModel_PREDICTION_MODEL_MEMORY_WINDOW
	case prediction.ModelEventTrigger:
		return pb.PredictionModel_PREDICTION_MODEL_EVENT_TRIGGER
	default:
		return pb.PredictionModel_PREDICTION_MODEL_UNSPECIFIED
	}
}

func (s *InventoryService) pbToDomainModel(pbModel pb.PredictionModel) prediction.PredictionModel {
	switch pbModel {
	case pb.PredictionModel_PREDICTION_MODEL_MARKOV:
		return prediction.ModelMarkov
	case pb.PredictionModel_PREDICTION_MODEL_CROSTON:
		return prediction.ModelCroston
	case pb.PredictionModel_PREDICTION_MODEL_DRIFT_IMPULSE:
		return prediction.ModelDriftImpulse
	case pb.PredictionModel_PREDICTION_MODEL_BAYESIAN:
		return prediction.ModelBayesian
	case pb.PredictionModel_PREDICTION_MODEL_MEMORY_WINDOW:
		return prediction.ModelMemoryWindow
	case pb.PredictionModel_PREDICTION_MODEL_EVENT_TRIGGER:
		return prediction.ModelEventTrigger
	default:
		return prediction.ModelMarkov // Default fallback
	}
}

func (s *InventoryService) domainToPbPrediction(estimate prediction.InventoryEstimate) *pb.ConsumptionPrediction {
	daysRemaining := 0.0
	if estimate.Estimate > 0 {
		// Simple calculation - could be more sophisticated
		daysRemaining = estimate.Estimate / 1.0 // Assume 1 unit per day consumption
	}

	return &pb.ConsumptionPrediction{
		ItemId:                  estimate.ItemName,
		PredictedDaysRemaining:  daysRemaining,
		ConfidenceScore:         estimate.Confidence,
		PredictedEmptyDate:      timestamppb.New(estimate.NextCheck),
		RecommendedRestockLevel: estimate.Estimate * 2, // Simple heuristic
		PredictionModel:         string(estimate.ModelUsed),
		Estimate:                estimate.Estimate,
		LowerBound:              estimate.LowerBound,
		UpperBound:              estimate.UpperBound,
		Recommendation:          estimate.Recommendation,
	}
}

func (s *InventoryService) domainToPbTrainingStage(stage prediction.TrainingStage) pb.TrainingStage {
	switch stage {
	case prediction.TrainingStageCollecting:
		return pb.TrainingStage_TRAINING_STAGE_COLLECTING
	case prediction.TrainingStageLearning:
		return pb.TrainingStage_TRAINING_STAGE_LEARNING
	case prediction.TrainingStageTrained:
		return pb.TrainingStage_TRAINING_STAGE_TRAINED
	case prediction.TrainingStageRetraining:
		return pb.TrainingStage_TRAINING_STAGE_RETRAINING
	default:
		return pb.TrainingStage_TRAINING_STAGE_UNSPECIFIED
	}
}
