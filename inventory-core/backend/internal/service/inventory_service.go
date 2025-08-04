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
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

const (
	errDomainToPbConversion = "domain to protobuf conversion failed"
	errResponseFormatting   = "response formatting failed"
)

// InventoryService implements the gRPC InventoryService interface
type InventoryService struct {
	pb.UnimplementedInventoryServiceServer

	repo     repository.InventoryRepository
	eventBus *events.EventBus
	logger   *logrus.Logger
}

// NewInventoryService creates a new inventory service instance
func NewInventoryService(
	repo repository.InventoryRepository,
	eventBus *events.EventBus,
	logger *logrus.Logger,
) *InventoryService {
	return &InventoryService{
		repo:     repo,
		eventBus: eventBus,
		logger:   logger,
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
		return nil, status.Errorf(codes.InvalidArgument, "item_id is required")
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
		return nil, status.Errorf(codes.InvalidArgument, "item_id is required")
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
		"item_id":         req.Report.ItemId,
		"level":          req.Report.Level,
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

func (s *InventoryService) domainToPbTrainingStatus(itemId string, status *prediction.TrainingStatus, model *prediction.Model) *pb.PredictionTrainingStatus {
	if status == nil {
		return nil
	}

	return &pb.PredictionTrainingStatus{
		ItemId:         itemId,
		ModelId:        model.ID,
		Status:         status.State,
		Error:          status.Error,
		ProgressPercent: int32(status.Progress * 100),
		TrainedUntil:   timestamppb.New(status.TrainedUntil),
		CreatedAt:      timestamppb.New(status.CreatedAt),
		UpdatedAt:      timestamppb.New(status.UpdatedAt),
	}
}

func (s *InventoryService) pbToDomainModel(pbModel *pb.PredictionModel) *prediction.Model {
	if pbModel == nil {
		return nil
	}

	return &prediction.Model{
		ID:          pbModel.Id,
		Name:        pbModel.Name,
		Description: pbModel.Description,
		Type:        prediction.ModelType(pbModel.Type),
		Parameters:  pbModel.Parameters,
	}
}
