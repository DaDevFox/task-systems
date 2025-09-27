package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

func setupTestService(repo *MockRepository) *InventoryService {
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests
	service := NewInventoryService(repo, mockEventBus, logger)
	service.DisableAuthForTesting()
	return service
}

// Unit tests with mocks - testing business logic and edge cases
func TestGetItemHistory_UnitTests(t *testing.T) {
	t.Run("Success_TimeRangeQuery", func(t *testing.T) {
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
			},
			{
				Timestamp: now.Add(-1 * time.Hour),
				Level:     75.0,
				UnitID:    "kg",
				Source:    "inventory_update",
				Context:   "Manual adjustment",
			},
		}

		repo.On("GetInventoryHistory", ctx, testItemID, mock.Anything).Return(snapshots, 2, nil)
		repo.On("GetEarliestSnapshot", ctx, testItemID).Return(snapshots[0], nil)
		repo.On("GetLatestSnapshot", ctx, testItemID).Return(snapshots[1], nil)

		// Test request
		req := &pb.GetItemHistoryRequest{
			ItemId: testItemID,
			QueryParams: &pb.GetItemHistoryRequest_TimeRange{
				TimeRange: &pb.TimeRangeQuery{
					StartTime:   timestamppb.New(now.Add(-3 * time.Hour)),
					EndTime:     timestamppb.New(now.Add(1 * time.Hour)),
					Granularity: pb.HistoryGranularity_HISTORY_GRANULARITY_HOUR,
					MaxPoints:   10,
				},
			},
		}

		// Execute
		resp, err := service.GetItemHistory(ctx, req)

		// Verify
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.History, 2)
		assert.Equal(t, int32(2), resp.TotalPoints)
		assert.False(t, resp.MoreDataAvailable)

		repo.AssertExpectations(t)
	})

	t.Run("Success_CountBasedQuery", func(t *testing.T) {
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

		// Mock historical snapshots
		snapshots := []*domain.InventoryLevelSnapshot{
			{
				Timestamp: time.Now().Add(-1 * time.Hour),
				Level:     75.0,
				UnitID:    "kg",
				Source:    "inventory_update",
			},
		}

		repo.On("GetInventoryHistory", ctx, testItemID, mock.Anything).Return(snapshots, 1, nil)
		repo.On("GetEarliestSnapshot", ctx, testItemID).Return(snapshots[0], nil)
		repo.On("GetLatestSnapshot", ctx, testItemID).Return(snapshots[0], nil)

		// Test request
		req := &pb.GetItemHistoryRequest{
			ItemId: testItemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{
					Count: 5,
				},
			},
		}

		// Execute
		resp, err := service.GetItemHistory(ctx, req)

		// Verify
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.History, 1)

		repo.AssertExpectations(t)
	})

	t.Run("Success_TimePointQuery", func(t *testing.T) {
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

		// Mock historical snapshots
		snapshots := []*domain.InventoryLevelSnapshot{
			{
				Timestamp: time.Now().Add(-30 * time.Minute),
				Level:     60.0,
				UnitID:    "kg",
				Source:    "inventory_update",
			},
		}

		repo.On("GetInventoryHistory", ctx, testItemID, mock.Anything).Return(snapshots, 1, nil)
		repo.On("GetEarliestSnapshot", ctx, testItemID).Return(snapshots[0], nil)
		repo.On("GetLatestSnapshot", ctx, testItemID).Return(snapshots[0], nil)

		// Test request
		req := &pb.GetItemHistoryRequest{
			ItemId: testItemID,
			QueryParams: &pb.GetItemHistoryRequest_TimePoint{
				TimePoint: &pb.TimePointQuery{
					FromTime:  timestamppb.New(time.Now().Add(-1 * time.Hour)),
					MaxPoints: 10,
				},
			},
		}

		// Execute
		resp, err := service.GetItemHistory(ctx, req)

		// Verify
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		repo.AssertExpectations(t)
	})

	t.Run("Error_ItemNotFound", func(t *testing.T) {
		repo := &MockRepository{}
		service := setupTestService(repo)
		ctx := context.Background()

		itemID := "non-existent-item"

		// Mock item not found
		repo.On("GetItem", ctx, itemID).Return((*domain.InventoryItem)(nil), &domain.InventoryItemNotFoundError{ID: itemID})

		// Test request
		req := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_TimeRange{
				TimeRange: &pb.TimeRangeQuery{
					StartTime: timestamppb.New(time.Now().Add(-24 * time.Hour)),
					EndTime:   timestamppb.New(time.Now()),
				},
			},
		}

		// Execute
		resp, err := service.GetItemHistory(ctx, req)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "item not found")

		repo.AssertExpectations(t)
	})

	t.Run("Error_EmptyItemId", func(t *testing.T) {
		repo := &MockRepository{}
		service := setupTestService(repo)
		ctx := context.Background()

		// Test request with empty item ID
		req := &pb.GetItemHistoryRequest{
			ItemId: "",
			QueryParams: &pb.GetItemHistoryRequest_TimeRange{
				TimeRange: &pb.TimeRangeQuery{
					StartTime: timestamppb.New(time.Now().Add(-24 * time.Hour)),
					EndTime:   timestamppb.New(time.Now()),
				},
			},
		}

		// Execute
		resp, err := service.GetItemHistory(ctx, req)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "item_id is required")

		// No repository calls should be made
		repo.AssertExpectations(t)
	})

	t.Run("Success_NoHistory", func(t *testing.T) {
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
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{
					Count: 5,
				},
			},
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
	})
}

// Integration tests with real database - testing full functionality
func TestGetItemHistory_Integration(t *testing.T) {
	// Setup temporary database
	tmpDir, err := os.MkdirTemp("", "test_history_integration")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewInventoryRepository(dbPath, repository.DatabaseTypeBadger)
	require.NoError(t, err)
	defer repo.Close()

	eventBus := events.GetGlobalBus("integration_test")
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	service := NewInventoryService(repo, eventBus, logger)

	ctx := context.Background()

	// Create test unit
	testUnit := &domain.Unit{
		ID:                   "kg",
		Name:                 "Kilograms",
		Symbol:               "kg",
		BaseConversionFactor: 1.0,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	err = repo.AddUnit(ctx, testUnit)
	require.NoError(t, err)

	// Create test item
	testItem := &domain.InventoryItem{
		Name:              "Integration Test Item",
		Description:       "Test item for history integration tests",
		CurrentLevel:      100.0,
		LowStockThreshold: 10.0,
		UnitID:            "kg",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	createResp, err := service.AddInventoryItem(ctx, &pb.AddInventoryItemRequest{
		Name:              testItem.Name,
		Description:       testItem.Description,
		InitialLevel:      testItem.CurrentLevel,
		LowStockThreshold: testItem.LowStockThreshold,
		UnitId:            testItem.UnitID,
	})
	require.NoError(t, err)

	// Use the generated item ID
	itemID := createResp.Item.Id

	// Add some history by updating inventory levels
	baseTime := time.Now()
	updates := []struct {
		level float64
		time  time.Time
	}{
		{85.0, baseTime.Add(-3 * time.Hour)},
		{70.0, baseTime.Add(-2 * time.Hour)},
		{50.0, baseTime.Add(-1 * time.Hour)},
	}

	for _, update := range updates {
		// Add snapshots directly to repository to simulate historical data
		snapshot := &domain.InventoryLevelSnapshot{
			Timestamp: update.time,
			Level:     update.level,
			UnitID:    "kg",
			Source:    "integration_test",
			Context:   "Test data",
		}
		err = repo.AddInventorySnapshot(ctx, itemID, snapshot)
		require.NoError(t, err)
		t.Logf("Added snapshot: Level=%.1f, Time=%v", update.level, update.time)
	}

	t.Run("TimeRangeQuery_Integration", func(t *testing.T) {
		req := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_TimeRange{
				TimeRange: &pb.TimeRangeQuery{
					StartTime:   timestamppb.New(baseTime.Add(-5 * time.Hour)),
					EndTime:     timestamppb.New(baseTime.Add(1 * time.Hour)),
					Granularity: pb.HistoryGranularity_HISTORY_GRANULARITY_MINUTE,
					MaxPoints:   20,
				},
			},
		}

		resp, err := service.GetItemHistory(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Debug logging
		t.Logf("Response has %d history entries and %d total points", len(resp.History), resp.TotalPoints)
		for i, h := range resp.History {
			t.Logf("  [%d] Level: %.1f, Time: %v, Source: %s", i, h.Level, h.Timestamp.AsTime(), h.Source)
		}

		// Should have at least the snapshots we added (plus potentially the initial creation snapshot)
		assert.GreaterOrEqual(t, len(resp.History), 1, "Should have at least one history entry")
		assert.GreaterOrEqual(t, resp.TotalPoints, int32(1), "Should have at least one data point")

		// Verify timestamps are within range
		for _, snapshot := range resp.History {
			assert.True(t, snapshot.Timestamp.AsTime().After(baseTime.Add(-5*time.Hour).Add(-time.Minute)))
			assert.True(t, snapshot.Timestamp.AsTime().Before(baseTime.Add(1*time.Hour).Add(time.Minute)))
		}
	})

	t.Run("CountBasedQuery_Integration", func(t *testing.T) {
		req := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{
					Count: 2,
				},
			},
		}

		resp, err := service.GetItemHistory(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Should return the requested number of points or all available
		assert.LessOrEqual(t, len(resp.History), 2)

		// Verify consistency - same query should return same results
		resp2, err := service.GetItemHistory(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, len(resp.History), len(resp2.History))
		assert.Equal(t, resp.TotalPoints, resp2.TotalPoints)
	})

	t.Run("TimePointQuery_Integration", func(t *testing.T) {
		req := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_TimePoint{
				TimePoint: &pb.TimePointQuery{
					FromTime:  timestamppb.New(baseTime.Add(-2*time.Hour - 30*time.Minute)),
					MaxPoints: 5,
				},
			},
		}

		resp, err := service.GetItemHistory(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Should get data points from the specified time onwards
		for _, snapshot := range resp.History {
			assert.True(t, snapshot.Timestamp.AsTime().After(baseTime.Add(-2*time.Hour-30*time.Minute).Add(-time.Minute)))
		}
	})

	t.Run("ValidationErrors_Integration", func(t *testing.T) {
		// Test empty item ID
		req := &pb.GetItemHistoryRequest{
			ItemId: "",
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 5},
			},
		}

		resp, err := service.GetItemHistory(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "item_id is required")

		// Test non-existent item
		req.ItemId = "non-existent-item"
		resp, err = service.GetItemHistory(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "item not found")
	})
}
