package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/prediction"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

const (
	errDomainToPbConversion          = "domain to protobuf conversion failed"
	errResponseFormatting            = "response formatting failed"
	errItemIdRequired                = "item_id is required"
	errInventoryItemNotFound         = "inventory item not found"
	errFailedToUpdateInventoryItem   = "failed to update inventory item"
	errUnitIdRequired                = "unit_id is required"
	errUnitNotFound                  = "unit not found"
	errFailedToConvertUnitToProtobuf = "failed to convert unit to protobuf"
	errFailedToFormatUnitResponse    = "failed to format unit response"
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

	// Set default parametric prediction model
	_ = item.GetActivePredictionModel() // This will create and set the default if none exists

	err = s.repo.AddItem(ctx, item)
	if err != nil {
		s.logger.WithError(err).WithField("item_name", req.Name).Error("failed to add inventory item")
		return nil, status.Errorf(codes.Internal, "failed to create inventory item")
	}

	// Store initial snapshot
	initialSnapshot := &domain.InventoryLevelSnapshot{
		Timestamp: item.CreatedAt,
		Level:     item.CurrentLevel,
		UnitID:    item.UnitID,
		Source:    "initial_creation",
		Context:   "Item created with initial level",
		Metadata: map[string]string{
			"created_by": "system",
		},
	}

	err = s.repo.AddInventorySnapshot(ctx, item.ID, initialSnapshot)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", item.ID).Warn("failed to store initial inventory snapshot")
		// Don't fail the request if snapshot storage fails
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
		return nil, status.Errorf(codes.NotFound, errInventoryItemNotFound)
	}

	pbItem, err := s.domainToPbItem(item)
	if err != nil {
		s.logger.WithError(err).Error(errDomainToPbConversion)
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
	}

	return &pb.GetInventoryItemResponse{Item: pbItem}, nil
}

// ListInventoryItems retrieves filtered list of items
func (s *InventoryService) ListInventoryItems(ctx context.Context, req *pb.ListInventoryItemsRequest) (*pb.ListInventoryItemsResponse, error) {
	// Set default limit if not provided
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50 // Default limit
	}

	filters := repository.ListFilters{
		LowStockOnly:   req.LowStockOnly,
		UnitTypeFilter: req.UnitTypeFilter,
		Limit:          limit,
		Offset:         int(req.Offset),
	}

	items, totalCount, err := s.repo.ListItems(ctx, filters)
	if err != nil {
		s.logger.WithError(err).Error("failed to list inventory items")
		return nil, status.Errorf(codes.Internal, "failed to list inventory items")
	}

	pbItems, err := s.domainToPbItems(items)
	if err != nil {
		s.logger.WithError(err).Error("failed to convert domain items to protobuf")
		return nil, status.Errorf(codes.Internal, "failed to format response")
	}

	return &pb.ListInventoryItemsResponse{
		Items:      pbItems,
		TotalCount: int32(totalCount),
	}, nil
}

// UpdateInventoryItem updates metadata and configuration of an inventory item
func (s *InventoryService) UpdateInventoryItem(ctx context.Context, req *pb.UpdateInventoryItemRequest) (*pb.UpdateInventoryItemResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	}

	// Get the existing item
	item, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item for update")
		return nil, status.Errorf(codes.NotFound, errInventoryItemNotFound)
	}

	// Update the item and track changes
	itemChanged, err := s.updateItemFields(ctx, item, req)
	if err != nil {
		return nil, err
	}

	// Save changes if any were made
	if itemChanged {
		err = s.saveItemChanges(ctx, item, req)
		if err != nil {
			return nil, err
		}
	}

	// Convert to protobuf response
	pbItem, err := s.domainToPbItem(item)
	if err != nil {
		s.logger.WithError(err).Error(errDomainToPbConversion)
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
	}

	return &pb.UpdateInventoryItemResponse{
		Item:        pbItem,
		ItemChanged: itemChanged,
	}, nil
}

// RemoveInventoryItem removes an inventory item from the system
func (s *InventoryService) RemoveInventoryItem(ctx context.Context, req *pb.RemoveInventoryItemRequest) (*pb.RemoveInventoryItemResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	}

	// Get the item first to retrieve its details for logging and response
	item, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item for removal")
		return nil, status.Errorf(codes.NotFound, errInventoryItemNotFound)
	}

	// Store item details before deletion
	itemName := item.Name
	itemId := item.ID

	// Delete the item from repository
	err = s.repo.DeleteItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to delete inventory item")
		return nil, status.Errorf(codes.Internal, "failed to remove inventory item")
	}

	// Publish inventory item removed event
	err = s.eventBus.PublishInventoryItemRemoved(ctx, itemId, itemName)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", itemId).Error("failed to publish inventory item removed event")
		// Don't fail the operation if event publishing fails
	}

	s.logger.WithFields(logrus.Fields{
		"item_id":   itemId,
		"item_name": itemName,
	}).Info("inventory item removed")

	return &pb.RemoveInventoryItemResponse{
		ItemRemoved:     true,
		RemovedItemId:   itemId,
		RemovedItemName: itemName,
	}, nil
}

// updateItemFields updates the item fields based on the request and returns whether changes were made
func (s *InventoryService) updateItemFields(ctx context.Context, item *domain.InventoryItem, req *pb.UpdateInventoryItemRequest) (bool, error) {
	itemChanged := false

	// Update basic fields
	if s.updateBasicFields(item, req) {
		itemChanged = true
	}

	// Update unit fields
	unitChanged, err := s.updateUnitFields(ctx, item, req)
	if err != nil {
		return false, err
	}
	if unitChanged {
		itemChanged = true
	}

	// Update consumption behavior
	if s.updateConsumptionBehavior(item, req) {
		itemChanged = true
	}

	// Update metadata
	if s.updateMetadata(item, req) {
		itemChanged = true
	}

	return itemChanged, nil
}

// updateBasicFields updates name, description, capacity, and threshold
func (s *InventoryService) updateBasicFields(item *domain.InventoryItem, req *pb.UpdateInventoryItemRequest) bool {
	changed := false

	if req.Name != "" && req.Name != item.Name {
		item.Name = req.Name
		changed = true
	}

	if req.Description != item.Description {
		item.Description = req.Description
		changed = true
	}

	if req.MaxCapacity != item.MaxCapacity {
		item.MaxCapacity = req.MaxCapacity
		changed = true
	}

	if req.LowStockThreshold != item.LowStockThreshold {
		item.LowStockThreshold = req.LowStockThreshold
		changed = true
	}

	return changed
}

// updateUnitFields updates unit ID and alternate unit IDs
func (s *InventoryService) updateUnitFields(ctx context.Context, item *domain.InventoryItem, req *pb.UpdateInventoryItemRequest) (bool, error) {
	changed := false

	// Update primary unit ID
	if req.UnitId != "" && req.UnitId != item.UnitID {
		_, err := s.repo.GetUnit(ctx, req.UnitId)
		if err != nil {
			s.logger.WithError(err).WithField("unit_id", req.UnitId).Error("unit validation failed")
			return false, status.Errorf(codes.InvalidArgument, "invalid unit_id: %s", req.UnitId)
		}
		item.UnitID = req.UnitId
		changed = true
	}

	// Update alternate unit IDs
	if s.alternateUnitsChanged(item, req) {
		for _, altUnitID := range req.AlternateUnitIds {
			_, err := s.repo.GetUnit(ctx, altUnitID)
			if err != nil {
				s.logger.WithError(err).WithField("unit_id", altUnitID).Error("alternate unit validation failed")
				return false, status.Errorf(codes.InvalidArgument, "invalid alternate unit_id: %s", altUnitID)
			}
		}
		item.AlternateUnitIDs = req.AlternateUnitIds
		changed = true
	}

	return changed, nil
}

// alternateUnitsChanged checks if the alternate units have changed
func (s *InventoryService) alternateUnitsChanged(item *domain.InventoryItem, req *pb.UpdateInventoryItemRequest) bool {
	if len(req.AlternateUnitIds) != len(item.AlternateUnitIDs) {
		return true
	}

	for i, altUnitID := range req.AlternateUnitIds {
		if i >= len(item.AlternateUnitIDs) || altUnitID != item.AlternateUnitIDs[i] {
			return true
		}
	}

	return false
}

// updateConsumptionBehavior updates consumption behavior if provided
func (s *InventoryService) updateConsumptionBehavior(item *domain.InventoryItem, req *pb.UpdateInventoryItemRequest) bool {
	if req.ConsumptionBehavior == nil {
		return false
	}

	if item.ConsumptionBehavior == nil {
		item.ConsumptionBehavior = &domain.ConsumptionBehavior{}
	}

	changed := false
	pbBehavior := req.ConsumptionBehavior

	if pbBehavior.Pattern != pb.ConsumptionPattern_CONSUMPTION_PATTERN_UNSPECIFIED {
		newPattern := domain.ConsumptionPattern(pbBehavior.Pattern)
		if item.ConsumptionBehavior.Pattern != newPattern {
			item.ConsumptionBehavior.Pattern = newPattern
			changed = true
		}
	}

	if pbBehavior.AverageRatePerDay != item.ConsumptionBehavior.AverageRatePerDay {
		item.ConsumptionBehavior.AverageRatePerDay = pbBehavior.AverageRatePerDay
		changed = true
	}

	if pbBehavior.Variance != item.ConsumptionBehavior.Variance {
		item.ConsumptionBehavior.Variance = pbBehavior.Variance
		changed = true
	}

	if len(pbBehavior.SeasonalFactors) > 0 && s.seasonalFactorsChanged(item, pbBehavior.SeasonalFactors) {
		item.ConsumptionBehavior.SeasonalFactors = pbBehavior.SeasonalFactors
		changed = true
	}

	if changed {
		item.ConsumptionBehavior.LastUpdated = time.Now()
	}

	return changed
}

// seasonalFactorsChanged checks if seasonal factors have changed
func (s *InventoryService) seasonalFactorsChanged(item *domain.InventoryItem, newFactors []float64) bool {
	if len(item.ConsumptionBehavior.SeasonalFactors) != len(newFactors) {
		return true
	}

	for i, factor := range newFactors {
		if i < len(item.ConsumptionBehavior.SeasonalFactors) &&
			item.ConsumptionBehavior.SeasonalFactors[i] != factor {
			return true
		}
	}

	return false
}

// updateMetadata updates the metadata if provided
func (s *InventoryService) updateMetadata(item *domain.InventoryItem, req *pb.UpdateInventoryItemRequest) bool {
	if req.Metadata == nil {
		return false
	}

	if item.Metadata == nil {
		item.Metadata = make(map[string]string)
	}

	// Check if metadata has changed
	if len(req.Metadata) != len(item.Metadata) {
		// Replace metadata completely with new values
		item.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			item.Metadata[k] = v
		}
		return true
	}

	changed := false
	for k, v := range req.Metadata {
		if item.Metadata[k] != v {
			changed = true
			break
		}
	}

	if changed {
		// Replace metadata completely with new values
		item.Metadata = make(map[string]string)
		for k, v := range req.Metadata {
			item.Metadata[k] = v
		}
	}

	return changed
}

// saveItemChanges saves the updated item and logs the changes
func (s *InventoryService) saveItemChanges(ctx context.Context, item *domain.InventoryItem, req *pb.UpdateInventoryItemRequest) error {
	item.UpdatedAt = time.Now()

	// Save the updated item
	err := s.repo.UpdateItem(ctx, item)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error(errFailedToUpdateInventoryItem)
		return status.Errorf(codes.Internal, errFailedToUpdateInventoryItem)
	}

	// Log the changes
	s.logItemUpdates(item, req)

	return nil
}

// logItemUpdates logs the changes made to the inventory item
func (s *InventoryService) logItemUpdates(item *domain.InventoryItem, req *pb.UpdateInventoryItemRequest) {
	logFields := logrus.Fields{
		"item_id":   item.ID,
		"item_name": item.Name,
		"changed":   true,
	}

	if req.Name != "" {
		logFields["name_updated"] = true
	}
	if req.Description != "" {
		logFields["description_updated"] = true
	}
	if req.MaxCapacity != 0 {
		logFields["max_capacity_updated"] = true
	}
	if req.LowStockThreshold != 0 {
		logFields["low_stock_threshold_updated"] = true
	}
	if req.UnitId != "" {
		logFields["unit_id_updated"] = true
	}
	if req.Metadata != nil {
		logFields["metadata_updated"] = true
	}

	s.logger.WithFields(logFields).Info("inventory item updated")
}

// UpdateInventoryLevel updates the quantity of an inventory item
func (s *InventoryService) UpdateInventoryLevel(ctx context.Context, req *pb.UpdateInventoryLevelRequest) (*pb.UpdateInventoryLevelResponse, error) {
	var pbItem *pb.InventoryItem
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	}

	item, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item for update")
		return nil, status.Errorf(codes.NotFound, errInventoryItemNotFound)
	}

	previousLevel := item.CurrentLevel
	wasLowStock := item.IsLowStock()

	item.CurrentLevel = req.NewLevel
	item.UpdatedAt = time.Now()

	err = s.repo.UpdateItem(ctx, item)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to update inventory item")
		return nil, status.Errorf(codes.Internal, "failed to update inventory level")
	}

	levelChanged := previousLevel != req.NewLevel
	belowThreshold := item.IsLowStock()

	// Store historical snapshot if level changed
	if levelChanged {
		snapshot := &domain.InventoryLevelSnapshot{
			Timestamp: item.UpdatedAt,
			Level:     item.CurrentLevel,
			UnitID:    item.UnitID,
			Source:    "inventory_update",
			Context:   req.Reason,
			Metadata: map[string]string{
				"previous_level": fmt.Sprintf("%.2f", previousLevel),
				"change_amount":  fmt.Sprintf("%.2f", item.CurrentLevel-previousLevel),
			},
		}

		err = s.repo.AddInventorySnapshot(ctx, item.ID, snapshot)
		if err != nil {
			s.logger.WithError(err).WithField("item_id", req.ItemId).Warn("failed to store inventory snapshot")
			// Don't fail the request if snapshot storage fails
		}
	}

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

	pbItem, err = s.domainToPbItem(item)
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

// GetInventoryStatus retrieves the current inventory status
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
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
	}

	pbLowStockItems, err := s.domainToPbItems(lowStockItems)
	if err != nil {
		s.logger.WithError(err).Error("failed to convert low stock items to protobuf")
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
	}

	pbEmptyItems, err := s.domainToPbItems(emptyItems)
	if err != nil {
		s.logger.WithError(err).Error("failed to convert empty items to protobuf")
		return nil, status.Errorf(codes.Internal, errResponseFormatting)
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

// PredictConsumption generates consumption predictions using the active prediction model
func (s *InventoryService) PredictConsumption(ctx context.Context, req *pb.PredictConsumptionRequest) (*pb.PredictConsumptionResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	}

	if req.DaysAhead <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "days_ahead must be positive")
	}

	// Get the inventory item
	item, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item for prediction")
		return nil, status.Errorf(codes.NotFound, errInventoryItemNotFound)
	}

	// Get the active prediction model (creates default if none exists)
	activeModelConfig := item.GetActivePredictionModel()

	// Create parametric predictor with the active model configuration
	parametricPredictor := prediction.NewParametricPredictor(item.ID, activeModelConfig, s.logger)

	// No historical consumption data available - will need to collect data through reports
	// If no history, create a recent report for the current state
	report := prediction.InventoryReport{
		ItemName:  item.ID,
		Timestamp: time.Now(),
		Level:     item.CurrentLevel,
		Context:   "current_state",
		Metadata:  make(map[string]string),
	}
	parametricPredictor.Update(report)

	// Calculate target time
	daysAhead := int32(30) // Default to 30 days
	if req.DaysAhead > 0 {
		daysAhead = req.DaysAhead
	}
	targetTime := time.Now().AddDate(0, 0, int(daysAhead))

	// Generate prediction
	estimate := parametricPredictor.Predict(targetTime)

	// Convert to protobuf
	prediction := &pb.ConsumptionPrediction{
		ItemId:                  req.ItemId,
		PredictedDaysRemaining:  s.calculateDaysRemaining(item.CurrentLevel, estimate.Estimate, daysAhead),
		ConfidenceScore:         estimate.Confidence,
		PredictedEmptyDate:      s.calculateEmptyDate(item.CurrentLevel, estimate.Estimate, daysAhead),
		RecommendedRestockLevel: s.calculateRestockLevel(item),
		PredictionModel:         string(estimate.ModelUsed),
		Estimate:                estimate.Estimate,
		LowerBound:              estimate.LowerBound,
		UpperBound:              estimate.UpperBound,
		Recommendation:          estimate.Recommendation,
	}

	s.logger.WithFields(logrus.Fields{
		"item_id":         req.ItemId,
		"days_ahead":      daysAhead,
		"current_level":   item.CurrentLevel,
		"predicted_level": estimate.Estimate,
		"confidence":      estimate.Confidence,
		"model_type":      parametricPredictor.Name(),
	}).Info("Generated consumption prediction")

	return &pb.PredictConsumptionResponse{Prediction: prediction}, nil
}

// SetActivePredictionModel configures the active prediction model for an item
func (s *InventoryService) SetActivePredictionModel(ctx context.Context, req *pb.SetActivePredictionModelRequest) (*pb.SetActivePredictionModelResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	}

	if req.ModelConfig == nil {
		return nil, status.Errorf(codes.InvalidArgument, "model_config is required")
	}

	// Get the inventory item
	item, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item for model configuration")
		return nil, status.Errorf(codes.NotFound, errInventoryItemNotFound)
	}

	// Update the active prediction model
	oldConfig := item.ActivePredictionModel
	item.SetActivePredictionModel(req.ModelConfig)

	// Save the updated item
	err = s.repo.UpdateItem(ctx, item)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to update item with new prediction model")
		return nil, status.Errorf(codes.Internal, "failed to update prediction model")
	}

	// Log the change
	s.logger.WithFields(logrus.Fields{
		"item_id":   item.ID,
		"item_name": item.Name,
		"old_model": s.getModelTypeName(oldConfig),
		"new_model": s.getModelTypeName(req.ModelConfig),
	}).Info("Updated active prediction model")

	// Convert to protobuf
	pbItem, err := s.domainToPbItem(item)
	if err != nil {
		return nil, status.Errorf(codes.Internal, errDomainToPbConversion)
	}

	return &pb.SetActivePredictionModelResponse{
		Item:         pbItem,
		ModelChanged: !s.modelsEqual(oldConfig, req.ModelConfig),
	}, nil
}

// GetActivePredictionModel retrieves the active prediction model for an item
func (s *InventoryService) GetActivePredictionModel(ctx context.Context, req *pb.GetActivePredictionModelRequest) (*pb.GetActivePredictionModelResponse, error) {
	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errItemIdRequired)
	}

	// Get the inventory item
	item, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).WithField("item_id", req.ItemId).Error("failed to get inventory item for model retrieval")
		return nil, status.Errorf(codes.NotFound, errInventoryItemNotFound)
	}

	// Get the active prediction model (creates default if none exists)
	activeModel := item.GetActivePredictionModel()

	return &pb.GetActivePredictionModelResponse{
		ModelConfig:    activeModel,
		HasActiveModel: activeModel != nil,
	}, nil
}

// GetItemHistory retrieves historical inventory levels for an item
func (s *InventoryService) GetItemHistory(ctx context.Context, req *pb.GetItemHistoryRequest) (*pb.GetItemHistoryResponse, error) {
	s.logger.WithField("item_id", req.ItemId).Info("Getting item history")

	if req.ItemId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "item_id is required")
	}

	// Verify the item exists
	_, err := s.repo.GetItem(ctx, req.ItemId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "item not found: %s", req.ItemId)
		}
		return nil, status.Errorf(codes.Internal, "failed to get item: %v", err)
	}

	// Handle different query patterns using the oneof field
	var snapshots []*domain.InventoryLevelSnapshot
	var totalCount int
	var queryType string

	switch queryParams := req.QueryParams.(type) {
	case *pb.GetItemHistoryRequest_TimeRange:
		queryType = "time_range"
		snapshots, totalCount, err = s.handleTimeRangeQuery(ctx, req.ItemId, queryParams.TimeRange)
	case *pb.GetItemHistoryRequest_CountBased:
		queryType = "count_based"
		snapshots, totalCount, err = s.handleCountBasedQuery(ctx, req.ItemId, queryParams.CountBased)
	case *pb.GetItemHistoryRequest_TimePoint:
		queryType = "time_point"
		snapshots, totalCount, err = s.handleTimePointQuery(ctx, req.ItemId, queryParams.TimePoint)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "one of query_params must be specified")
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get inventory history: %v", err)
	}

	// Convert to protobuf
	pbSnapshots := make([]*pb.InventoryLevelSnapshot, len(snapshots))
	for i, snapshot := range snapshots {
		pbSnapshots[i] = &pb.InventoryLevelSnapshot{
			Timestamp: timestamppb.New(snapshot.Timestamp),
			Level:     snapshot.Level,
			UnitId:    snapshot.UnitID,
			Source:    snapshot.Source,
			Context:   snapshot.Context,
			Metadata:  snapshot.Metadata,
		}
	}

	// Get earliest and latest timestamps
	earliestSnapshot, err := s.repo.GetEarliestSnapshot(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get earliest snapshot")
	}

	latestSnapshot, err := s.repo.GetLatestSnapshot(ctx, req.ItemId)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get latest snapshot")
	}

	// Determine if more data is available based on query type
	var moreDataAvailable bool
	switch queryParams := req.QueryParams.(type) {
	case *pb.GetItemHistoryRequest_TimeRange:
		moreDataAvailable = queryParams.TimeRange.MaxPoints > 0 && totalCount > int(queryParams.TimeRange.MaxPoints)
	case *pb.GetItemHistoryRequest_CountBased:
		moreDataAvailable = totalCount > int(queryParams.CountBased.Count)
	case *pb.GetItemHistoryRequest_TimePoint:
		moreDataAvailable = queryParams.TimePoint.MaxPoints > 0 && totalCount > int(queryParams.TimePoint.MaxPoints)
	}

	response := &pb.GetItemHistoryResponse{
		History:           pbSnapshots,
		TotalPoints:       int32(totalCount),
		MoreDataAvailable: moreDataAvailable,
	}

	if earliestSnapshot != nil {
		response.EarliestTimestamp = timestamppb.New(earliestSnapshot.Timestamp)
	}
	if latestSnapshot != nil {
		response.LatestTimestamp = timestamppb.New(latestSnapshot.Timestamp)
	}

	s.logger.WithFields(logrus.Fields{
		"item_id":      req.ItemId,
		"query_type":   queryType,
		"total_points": totalCount,
		"returned":     len(pbSnapshots),
	}).Info("Successfully retrieved item history")

	return response, nil
}

// handleTimeRangeQuery handles time-range based history queries
// This is the original query method, good for general-purpose queries with time filtering
func (s *InventoryService) handleTimeRangeQuery(ctx context.Context, itemID string, timeRange *pb.TimeRangeQuery) ([]*domain.InventoryLevelSnapshot, int, error) {
	filters := repository.HistoryFilters{}

	if timeRange.StartTime != nil {
		filters.StartTime = timeRange.StartTime.AsTime()
	}
	if timeRange.EndTime != nil {
		filters.EndTime = timeRange.EndTime.AsTime()
	}

	// Convert granularity enum to string
	switch timeRange.Granularity {
	case pb.HistoryGranularity_HISTORY_GRANULARITY_MINUTE:
		filters.Granularity = "minute"
	case pb.HistoryGranularity_HISTORY_GRANULARITY_HOUR:
		filters.Granularity = "hour"
	case pb.HistoryGranularity_HISTORY_GRANULARITY_DAY:
		filters.Granularity = "day"
	case pb.HistoryGranularity_HISTORY_GRANULARITY_WEEK:
		filters.Granularity = "week"
	case pb.HistoryGranularity_HISTORY_GRANULARITY_MONTH:
		filters.Granularity = "month"
	default:
		filters.Granularity = "all"
	}

	if timeRange.MaxPoints > 0 {
		filters.Limit = int(timeRange.MaxPoints)
	}

	return s.repo.GetInventoryHistory(ctx, itemID, filters)
}

// handleCountBasedQuery handles count-based history queries
// PERFORMANCE NOTE: This is highly efficient as it uses database LIMIT directly
func (s *InventoryService) handleCountBasedQuery(ctx context.Context, itemID string, countBased *pb.CountBasedQuery) ([]*domain.InventoryLevelSnapshot, int, error) {
	filters := repository.HistoryFilters{
		Limit: int(countBased.Count),
	}

	// CountBasedQuery returns the most recent N data points (newest-first)
	// This is the natural BadgerDB order (most recent timestamps first)
	return s.repo.GetInventoryHistory(ctx, itemID, filters)
}

// handleTimePointQuery handles time-point based history queries
// PERFORMANCE NOTE: Less efficient for large datasets as it needs to scan from a specific time point
func (s *InventoryService) handleTimePointQuery(ctx context.Context, itemID string, timePoint *pb.TimePointQuery) ([]*domain.InventoryLevelSnapshot, int, error) {
	if timePoint.FromTime == nil {
		return nil, 0, fmt.Errorf("from_time is required for time point queries")
	}

	fromTime := timePoint.FromTime.AsTime()

	// TimePointQuery gets all data from the specified time backwards to present (newest-first)
	filters := repository.HistoryFilters{
		StartTime: fromTime, // Everything from this time onwards
	}

	if timePoint.MaxPoints > 0 {
		filters.Limit = int(timePoint.MaxPoints)
	}

	return s.repo.GetInventoryHistory(ctx, itemID, filters)
}

// Unit Management Methods

// ListUnits retrieves all unit definitions in the system
func (s *InventoryService) ListUnits(ctx context.Context, req *pb.ListUnitsRequest) (*pb.ListUnitsResponse, error) {
	units, err := s.repo.ListUnits(ctx)
	if err != nil {
		s.logger.WithError(err).Error("failed to list units")
		return nil, status.Errorf(codes.Internal, "failed to retrieve units")
	}

	pbUnits, err := s.domainToPbUnits(units)
	if err != nil {
		s.logger.WithError(err).Error(errFailedToConvertUnitToProtobuf)
		return nil, status.Errorf(codes.Internal, errFailedToFormatUnitResponse)
	}

	s.logger.WithField("unit_count", len(units)).Debug("listed all units")

	return &pb.ListUnitsResponse{
		Units: pbUnits,
	}, nil
}

// AddUnit creates a new unit definition
func (s *InventoryService) AddUnit(ctx context.Context, req *pb.AddUnitRequest) (*pb.AddUnitResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "unit name is required")
	}

	if req.Symbol == "" {
		return nil, status.Errorf(codes.InvalidArgument, "unit symbol is required")
	}

	if req.BaseConversionFactor <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "base conversion factor must be positive")
	}

	// Create domain unit
	unit := &domain.Unit{
		ID:                   uuid.New().String(),
		Name:                 req.Name,
		Symbol:               req.Symbol,
		Description:          req.Description,
		BaseConversionFactor: req.BaseConversionFactor,
		Category:             req.Category,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		Metadata:             req.Metadata,
	}

	if unit.Metadata == nil {
		unit.Metadata = make(map[string]string)
	}

	// Add the unit to repository
	err := s.repo.AddUnit(ctx, unit)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"unit_name":   req.Name,
			"unit_symbol": req.Symbol,
		}).Error("failed to create unit")
		return nil, status.Errorf(codes.Internal, "failed to create unit")
	}

	s.logger.WithFields(logrus.Fields{
		"unit_id":     unit.ID,
		"unit_name":   unit.Name,
		"unit_symbol": unit.Symbol,
		"category":    unit.Category,
	}).Info("unit created")

	// Convert to protobuf response
	pbUnit, err := s.domainToPbUnit(unit)
	if err != nil {
		s.logger.WithError(err).Error(errFailedToConvertUnitToProtobuf)
		return nil, status.Errorf(codes.Internal, errFailedToFormatUnitResponse)
	}

	return &pb.AddUnitResponse{Unit: pbUnit}, nil
}

// GetUnit retrieves a specific unit by ID
func (s *InventoryService) GetUnit(ctx context.Context, req *pb.GetUnitRequest) (*pb.GetUnitResponse, error) {
	if req.UnitId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errUnitIdRequired)
	}

	unit, err := s.repo.GetUnit(ctx, req.UnitId)
	if err != nil {
		s.logger.WithError(err).WithField("unit_id", req.UnitId).Error("failed to get unit")
		return nil, status.Errorf(codes.NotFound, errUnitNotFound)
	}

	pbUnit, err := s.domainToPbUnit(unit)
	if err != nil {
		s.logger.WithError(err).Error(errFailedToConvertUnitToProtobuf)
		return nil, status.Errorf(codes.Internal, errFailedToFormatUnitResponse)
	}

	return &pb.GetUnitResponse{Unit: pbUnit}, nil
}

// UpdateUnit updates an existing unit definition
func (s *InventoryService) UpdateUnit(ctx context.Context, req *pb.UpdateUnitRequest) (*pb.UpdateUnitResponse, error) {
	if req.UnitId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errUnitIdRequired)
	}

	// Get the existing unit
	unit, err := s.repo.GetUnit(ctx, req.UnitId)
	if err != nil {
		s.logger.WithError(err).WithField("unit_id", req.UnitId).Error("failed to get unit for update")
		return nil, status.Errorf(codes.NotFound, errUnitNotFound)
	}

	// Track changes
	unitChanged := false

	// Update basic fields
	if req.Name != "" && req.Name != unit.Name {
		unit.Name = req.Name
		unitChanged = true
	}

	if req.Symbol != "" && req.Symbol != unit.Symbol {
		unit.Symbol = req.Symbol
		unitChanged = true
	}

	if req.Description != unit.Description {
		unit.Description = req.Description
		unitChanged = true
	}

	if req.BaseConversionFactor > 0 && req.BaseConversionFactor != unit.BaseConversionFactor {
		unit.BaseConversionFactor = req.BaseConversionFactor
		unitChanged = true
	}

	if req.Category != "" && req.Category != unit.Category {
		unit.Category = req.Category
		unitChanged = true
	}

	// Update metadata if provided
	if req.Metadata != nil {
		if unit.Metadata == nil {
			unit.Metadata = make(map[string]string)
		}

		// Check if metadata has changed
		metadataChanged := len(req.Metadata) != len(unit.Metadata)
		if !metadataChanged {
			for k, v := range req.Metadata {
				if unit.Metadata[k] != v {
					metadataChanged = true
					break
				}
			}
		}

		if metadataChanged {
			// Replace metadata completely with new values
			unit.Metadata = make(map[string]string)
			for k, v := range req.Metadata {
				unit.Metadata[k] = v
			}
			unitChanged = true
		}
	}

	// Save changes if any were made
	if unitChanged {
		unit.UpdatedAt = time.Now()

		err = s.repo.UpdateUnit(ctx, unit)
		if err != nil {
			s.logger.WithError(err).WithField("unit_id", req.UnitId).Error("failed to update unit")
			return nil, status.Errorf(codes.Internal, "failed to update unit")
		}

		s.logger.WithFields(logrus.Fields{
			"unit_id":     unit.ID,
			"unit_name":   unit.Name,
			"unit_symbol": unit.Symbol,
		}).Info("unit updated")
	}

	// Convert to protobuf response
	pbUnit, err := s.domainToPbUnit(unit)
	if err != nil {
		s.logger.WithError(err).Error(errFailedToConvertUnitToProtobuf)
		return nil, status.Errorf(codes.Internal, errFailedToFormatUnitResponse)
	}

	return &pb.UpdateUnitResponse{
		Unit:        pbUnit,
		UnitChanged: unitChanged,
	}, nil
}

// DeleteUnit removes a unit definition from the system
func (s *InventoryService) DeleteUnit(ctx context.Context, req *pb.DeleteUnitRequest) (*pb.DeleteUnitResponse, error) {
	if req.UnitId == "" {
		return nil, status.Errorf(codes.InvalidArgument, errUnitIdRequired)
	}

	// Get the unit first to retrieve its details for logging and response
	unit, err := s.repo.GetUnit(ctx, req.UnitId)
	if err != nil {
		s.logger.WithError(err).WithField("unit_id", req.UnitId).Error("failed to get unit for deletion")
		return nil, status.Errorf(codes.NotFound, errUnitNotFound)
	}

	// Check if unit is being used by any inventory items (optional safety check)
	if !req.Force {
		items, err := s.repo.GetAllItems(ctx)
		if err != nil {
			s.logger.WithError(err).Error("failed to check unit usage")
			return nil, status.Errorf(codes.Internal, "failed to verify unit usage")
		}

		var usingItems []string
		for _, item := range items {
			if item.UnitID == req.UnitId {
				usingItems = append(usingItems, item.Name)
			}
			// Check alternate unit IDs too
			for _, altUnitID := range item.AlternateUnitIDs {
				if altUnitID == req.UnitId {
					usingItems = append(usingItems, item.Name)
					break
				}
			}
		}

		if len(usingItems) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition,
				"unit is being used by %d inventory item(s): %v. Use force=true to delete anyway",
				len(usingItems), usingItems)
		}
	}

	// Store unit details before deletion
	unitName := unit.Name
	unitId := unit.ID
	unitSymbol := unit.Symbol

	// Delete the unit from repository
	err = s.repo.DeleteUnit(ctx, req.UnitId)
	if err != nil {
		s.logger.WithError(err).WithField("unit_id", req.UnitId).Error("failed to delete unit")
		return nil, status.Errorf(codes.Internal, "failed to delete unit")
	}

	s.logger.WithFields(logrus.Fields{
		"unit_id":     unitId,
		"unit_name":   unitName,
		"unit_symbol": unitSymbol,
		"force":       req.Force,
	}).Info("unit deleted")

	return &pb.DeleteUnitResponse{
		UnitDeleted:     true,
		DeletedUnitId:   unitId,
		DeletedUnitName: unitName,
	}, nil
}

// Helper methods for domain/protobuf conversion

// domainToPbItem converts a domain InventoryItem to protobuf InventoryItem
func (s *InventoryService) domainToPbItem(item *domain.InventoryItem) (*pb.InventoryItem, error) {
	if item == nil {
		return nil, fmt.Errorf("item cannot be nil")
	}

	pbItem := &pb.InventoryItem{
		Id:                    item.ID,
		Name:                  item.Name,
		Description:           item.Description,
		CurrentLevel:          item.CurrentLevel,
		MaxCapacity:           item.MaxCapacity,
		LowStockThreshold:     item.LowStockThreshold,
		UnitId:                item.UnitID,
		AlternateUnitIds:      item.AlternateUnitIDs,
		CreatedAt:             timestamppb.New(item.CreatedAt),
		UpdatedAt:             timestamppb.New(item.UpdatedAt),
		Metadata:              item.Metadata,
		ActivePredictionModel: item.ActivePredictionModel,
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

	return pbItem, nil
}

// domainToPbItems converts a slice of domain InventoryItems to protobuf InventoryItems
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

// calculateDaysRemaining estimates how many days until the item is empty
func (s *InventoryService) calculateDaysRemaining(currentLevel, predictedLevel float64, daysAhead int32) float64 {
	if currentLevel <= 0 {
		return 0
	}

	// Calculate consumption rate
	consumptionRate := (currentLevel - predictedLevel) / float64(daysAhead)

	if consumptionRate <= 0 {
		return float64(daysAhead) * 2 // If no consumption, return double the forecast period
	}

	daysRemaining := currentLevel / consumptionRate
	return math.Max(0, daysRemaining)
}

// calculateEmptyDate estimates when the item will be empty
func (s *InventoryService) calculateEmptyDate(currentLevel, predictedLevel float64, daysAhead int32) *timestamppb.Timestamp {
	daysRemaining := s.calculateDaysRemaining(currentLevel, predictedLevel, daysAhead)
	emptyDate := time.Now().AddDate(0, 0, int(daysRemaining))
	return timestamppb.New(emptyDate)
}

// calculateRestockLevel suggests an appropriate restock level
func (s *InventoryService) calculateRestockLevel(item *domain.InventoryItem) float64 {
	// Simple heuristic: restock to 80% of max capacity or double the low stock threshold
	restockLevel := item.MaxCapacity * 0.8

	if item.LowStockThreshold > 0 {
		alternativeLevel := item.LowStockThreshold * 2
		if alternativeLevel > restockLevel {
			restockLevel = alternativeLevel
		}
	}

	return math.Min(restockLevel, item.MaxCapacity)
}

// getModelTypeName returns a human-readable name for a prediction model configuration
func (s *InventoryService) getModelTypeName(config *pb.PredictionModelConfig) string {
	if config == nil {
		return "None"
	}

	switch config.ModelConfig.(type) {
	case *pb.PredictionModelConfig_Parametric:
		return "Parametric Model"
	case *pb.PredictionModelConfig_Markov:
		return "Markov Model"
	case *pb.PredictionModelConfig_Croston:
		return "Croston Model"
	case *pb.PredictionModelConfig_DriftImpulse:
		return "Drift Impulse Model"
	case *pb.PredictionModelConfig_Bayesian:
		return "Bayesian Model"
	case *pb.PredictionModelConfig_MemoryWindow:
		return "Memory Window Model"
	case *pb.PredictionModelConfig_EventTrigger:
		return "Event Trigger Model"
	default:
		return "Unknown Model"
	}
}

// modelsEqual compares two prediction model configurations for equality
func (s *InventoryService) modelsEqual(model1, model2 *pb.PredictionModelConfig) bool {
	if model1 == nil && model2 == nil {
		return true
	}
	if model1 == nil || model2 == nil {
		return false
	}

	// Compare the model types and their configurations
	switch config1 := model1.ModelConfig.(type) {
	case *pb.PredictionModelConfig_Parametric:
		config2, ok := model2.ModelConfig.(*pb.PredictionModelConfig_Parametric)
		if !ok {
			return false
		}
		return s.parametricModelsEqual(config1.Parametric, config2.Parametric)
	case *pb.PredictionModelConfig_Markov:
		config2, ok := model2.ModelConfig.(*pb.PredictionModelConfig_Markov)
		if !ok {
			return false
		}
		return s.markovModelsEqual(config1.Markov, config2.Markov)
	// Add other model type comparisons as needed
	default:
		// For deprecated/unsupported models, just check the type
		return false
	}
}

// parametricModelsEqual compares two parametric model configurations
func (s *InventoryService) parametricModelsEqual(model1, model2 *pb.ParametricModel) bool {
	if model1 == nil && model2 == nil {
		return true
	}
	if model1 == nil || model2 == nil {
		return false
	}

	// Compare model types first
	switch modelType1 := model1.ModelType.(type) {
	case *pb.ParametricModel_Linear:
		modelType2, ok := model2.ModelType.(*pb.ParametricModel_Linear)
		if !ok {
			return false
		}
		return s.linearModelsEqual(modelType1.Linear, modelType2.Linear)
	case *pb.ParametricModel_Logistic:
		modelType2, ok := model2.ModelType.(*pb.ParametricModel_Logistic)
		if !ok {
			return false
		}
		return s.logisticModelsEqual(modelType1.Logistic, modelType2.Logistic)
	default:
		return false
	}
}

// linearModelsEqual compares two linear equation model configurations
func (s *InventoryService) linearModelsEqual(model1, model2 *pb.LinearEquationModel) bool {
	if model1 == nil && model2 == nil {
		return true
	}
	if model1 == nil || model2 == nil {
		return false
	}

	return model1.Slope == model2.Slope &&
		model1.BaseLevel == model2.BaseLevel &&
		model1.NoiseVariance == model2.NoiseVariance
}

// logisticModelsEqual compares two logistic equation model configurations
func (s *InventoryService) logisticModelsEqual(model1, model2 *pb.LogisticEquationModel) bool {
	if model1 == nil && model2 == nil {
		return true
	}
	if model1 == nil || model2 == nil {
		return false
	}

	return model1.GrowthRate == model2.GrowthRate &&
		model1.CarryingCapacity == model2.CarryingCapacity &&
		model1.InitialPopulation == model2.InitialPopulation &&
		model1.NoiseVariance == model2.NoiseVariance
}

// markovModelsEqual compares two markov model configurations
func (s *InventoryService) markovModelsEqual(model1, model2 *pb.MarkovModelConfig) bool {
	if model1 == nil && model2 == nil {
		return true
	}
	if model1 == nil || model2 == nil {
		return false
	}

	return model1.LowThreshold == model2.LowThreshold &&
		model1.DepletedThreshold == model2.DepletedThreshold
}

// Unit conversion helper methods

// domainToPbUnit converts a domain Unit to protobuf Unit
func (s *InventoryService) domainToPbUnit(unit *domain.Unit) (*pb.Unit, error) {
	if unit == nil {
		return nil, fmt.Errorf("unit cannot be nil")
	}

	return &pb.Unit{
		Id:                   unit.ID,
		Name:                 unit.Name,
		Symbol:               unit.Symbol,
		Description:          unit.Description,
		BaseConversionFactor: unit.BaseConversionFactor,
		Category:             unit.Category,
		CreatedAt:            timestamppb.New(unit.CreatedAt),
		UpdatedAt:            timestamppb.New(unit.UpdatedAt),
		Metadata:             unit.Metadata,
	}, nil
}

// domainToPbUnits converts a slice of domain Units to protobuf Units
func (s *InventoryService) domainToPbUnits(units []*domain.Unit) ([]*pb.Unit, error) {
	if len(units) == 0 {
		return []*pb.Unit{}, nil
	}

	pbUnits := make([]*pb.Unit, 0, len(units))
	for _, unit := range units {
		pbUnit, err := s.domainToPbUnit(unit)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unit %s: %w", unit.ID, err)
		}
		pbUnits = append(pbUnits, pbUnit)
	}

	return pbUnits, nil
}
