package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/inventory-core/internal/domain"
	"github.com/DaDevFox/task-systems/inventory-core/internal/repository"
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
