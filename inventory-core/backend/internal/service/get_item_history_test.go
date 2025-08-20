package service

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

func setupTestService(repo *MockRepository) *InventoryService {
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests
	return NewInventoryService(repo, mockEventBus, logger)
}

func TestGetItemHistorySuccess(t *testing.T) {
	// Setup
	repo := &MockRepository{}
	service := setupTestService(repo)
	ctx := context.Background()

	now := time.Now()
	
	// Mock item exists
	testItem := &domain.InventoryItem{
		ID:           testItemID,
		Name:         testItemName,
		CurrentLevel: 50.0,
		UnitID:       "kg",
	}
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)

	// Mock historical snapshots
	snapshots := []*domain.InventoryLevelSnapshot{
		{
			Timestamp: now.Add(-2 * time.Hour),
			Level:     100.0,
			UnitID:    "kg",
			Source:    "initial_creation",
			Context:   "Item created",
			Metadata:  map[string]string{"created_by": "system"},
		},
		{
			Timestamp: now.Add(-1 * time.Hour),
			Level:     75.0,
			UnitID:    "kg",
			Source:    "inventory_update",
			Context:   "Manual adjustment",
			Metadata:  map[string]string{"previous_level": "100.00", "change_amount": "-25.00"},
		},
		{
			Timestamp: now,
			Level:     50.0,
			UnitID:    "kg",
			Source:    "inventory_update",
			Context:   "Used for cooking",
			Metadata:  map[string]string{"previous_level": "75.00", "change_amount": "-25.00"},
		},
	}
	
	repo.On("GetInventoryHistory", ctx, testItemID, mock.Anything).Return(snapshots, 3, nil)
	
	// Mock earliest and latest snapshots
	repo.On("GetEarliestSnapshot", ctx, testItemID).Return(snapshots[0], nil)
	repo.On("GetLatestSnapshot", ctx, testItemID).Return(snapshots[2], nil)

	// Test request
	req := &pb.GetItemHistoryRequest{
		ItemId:      testItemID,
		StartTime:   timestamppb.New(now.Add(-3 * time.Hour)),
		EndTime:     timestamppb.New(now.Add(1 * time.Hour)),
		Granularity: pb.HistoryGranularity_HISTORY_GRANULARITY_HOUR,
		MaxPoints:   10,
	}

	// Execute
	resp, err := service.GetItemHistory(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.History, 3)
	assert.Equal(t, int32(3), resp.TotalPoints)
	assert.False(t, resp.MoreDataAvailable)

	// Verify first snapshot
	firstSnapshot := resp.History[0]
	assert.Equal(t, 100.0, firstSnapshot.Level)
	assert.Equal(t, "kg", firstSnapshot.UnitId)
	assert.Equal(t, "initial_creation", firstSnapshot.Source)
	assert.Equal(t, "Item created", firstSnapshot.Context)

	// Verify timestamps
	assert.NotNil(t, resp.EarliestTimestamp)
	assert.NotNil(t, resp.LatestTimestamp)

	repo.AssertExpectations(t)
}

func TestGetItemHistoryItemNotFound(t *testing.T) {
	// Setup
	repo := &MockRepository{}
	service := setupTestService(repo)
	ctx := context.Background()

	itemID := "non-existent-item"
	
	// Mock item not found
	repo.On("GetItem", ctx, itemID).Return((*domain.InventoryItem)(nil), &domain.InventoryItemNotFoundError{ID: itemID})

	// Test request
	req := &pb.GetItemHistoryRequest{
		ItemId: itemID,
	}

	// Execute
	resp, err := service.GetItemHistory(ctx, req)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "item not found")

	repo.AssertExpectations(t)
}

func TestGetItemHistoryEmptyItemId(t *testing.T) {
	// Setup
	repo := &MockRepository{}
	service := setupTestService(repo)
	ctx := context.Background()

	// Test request with empty item ID
	req := &pb.GetItemHistoryRequest{
		ItemId: "",
	}

	// Execute
	resp, err := service.GetItemHistory(ctx, req)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "item_id is required")

	// No repository calls should be made
	repo.AssertExpectations(t)
}

func TestGetItemHistoryNoHistory(t *testing.T) {
	// Setup
	repo := &MockRepository{}
	service := setupTestService(repo)
	ctx := context.Background()
	
	// Mock item exists
	testItem := &domain.InventoryItem{
		ID:           testItemID,
		Name:         testItemName,
		CurrentLevel: 50.0,
		UnitID:       "kg",
	}
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)

	// Mock no history
	repo.On("GetInventoryHistory", ctx, testItemID, mock.Anything).Return([]*domain.InventoryLevelSnapshot{}, 0, nil)
	repo.On("GetEarliestSnapshot", ctx, testItemID).Return((*domain.InventoryLevelSnapshot)(nil), nil)
	repo.On("GetLatestSnapshot", ctx, testItemID).Return((*domain.InventoryLevelSnapshot)(nil), nil)

	// Test request
	req := &pb.GetItemHistoryRequest{
		ItemId: testItemID,
	}

	// Execute
	resp, err := service.GetItemHistory(ctx, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.History)
	assert.Equal(t, int32(0), resp.TotalPoints)
	assert.False(t, resp.MoreDataAvailable)
	assert.Nil(t, resp.EarliestTimestamp)
	assert.Nil(t, resp.LatestTimestamp)

	repo.AssertExpectations(t)
}
