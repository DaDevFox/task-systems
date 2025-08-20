package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

func TestInventoryHistoryIntegration(t *testing.T) {
	// Setup temporary database
	tmpDir, err := os.MkdirTemp("", "test_inventory_history")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	require.NoError(t, err)
	defer repo.Close()

	eventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel) // Enable debug logging to see what's happening

	service := NewInventoryService(repo, eventBus, logger)
	ctx := context.Background()

	// First, add a unit
	unitReq := &pb.AddUnitRequest{
		Name:                   "Kilograms",
		Symbol:                 "kg",
		Description:            "Unit of mass",
		BaseConversionFactor:   1.0,
		Category:              "weight",
	}
	unitResp, err := service.AddUnit(ctx, unitReq)
	require.NoError(t, err)
	require.NotNil(t, unitResp)

	// Create an inventory item - should create initial snapshot
	addReq := &pb.AddInventoryItemRequest{
		Name:              "Test Item",
		Description:       "Test item for history tracking",
		InitialLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitId:            unitResp.Unit.Id,
	}

	addResp, err := service.AddInventoryItem(ctx, addReq)
	require.NoError(t, err)
	require.NotNil(t, addResp)
	
	itemID := addResp.Item.Id
	t.Logf("Created item with ID: %s, initial level: %.2f", itemID, addResp.Item.CurrentLevel)

	// Wait a small amount to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Update inventory level - should create another snapshot
	updateReq := &pb.UpdateInventoryLevelRequest{
		ItemId:   itemID,
		NewLevel: 75.0,
		Reason:   "Manual adjustment",
	}

	updateResp, err := service.UpdateInventoryLevel(ctx, updateReq)
	require.NoError(t, err)
	require.NotNil(t, updateResp)
	assert.Equal(t, 75.0, updateResp.Item.CurrentLevel)
	t.Logf("Updated item to level: %.2f", updateResp.Item.CurrentLevel)

	// Wait again
	time.Sleep(10 * time.Millisecond)

	// Another level update
	updateReq2 := &pb.UpdateInventoryLevelRequest{
		ItemId:   itemID,
		NewLevel: 50.0,
		Reason:   "Used for production",
	}

	updateResp2, err := service.UpdateInventoryLevel(ctx, updateReq2)
	require.NoError(t, err)
	require.NotNil(t, updateResp2)
	assert.Equal(t, 50.0, updateResp2.Item.CurrentLevel)
	t.Logf("Updated item again to level: %.2f", updateResp2.Item.CurrentLevel)

	// Now test the GetItemHistory endpoint
	historyReq := &pb.GetItemHistoryRequest{
		ItemId:      itemID,
		// Don't set time filters to get all history
		// Don't set granularity to get all snapshots without filtering
		MaxPoints:   10,
	}

	historyResp, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)
	require.NotNil(t, historyResp)

	// Debug: Print actual history response details
	t.Logf("History response: TotalPoints=%d, History length=%d", historyResp.TotalPoints, len(historyResp.History))
	for i, snapshot := range historyResp.History {
		t.Logf("Snapshot %d: Level=%.2f, Source=%s, Context=%s", i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// We should have 3 snapshots: initial creation, first update, second update
	assert.Len(t, historyResp.History, 3)
	assert.Equal(t, int32(3), historyResp.TotalPoints)
	assert.False(t, historyResp.MoreDataAvailable)

	// Verify the history is in correct order and has correct values
	snapshots := historyResp.History

	// First snapshot - initial creation
	assert.Equal(t, 100.0, snapshots[0].Level)
	assert.Equal(t, unitResp.Unit.Id, snapshots[0].UnitId)
	assert.Equal(t, "initial_creation", snapshots[0].Source)
	assert.Equal(t, "Item created with initial level", snapshots[0].Context)
	assert.Equal(t, "system", snapshots[0].Metadata["created_by"])

	// Second snapshot - first update
	assert.Equal(t, 75.0, snapshots[1].Level)
	assert.Equal(t, unitResp.Unit.Id, snapshots[1].UnitId)
	assert.Equal(t, "inventory_update", snapshots[1].Source)
	assert.Equal(t, "Manual adjustment", snapshots[1].Context)
	assert.Equal(t, "100.00", snapshots[1].Metadata["previous_level"])
	assert.Equal(t, "-25.00", snapshots[1].Metadata["change_amount"])

	// Third snapshot - second update
	assert.Equal(t, 50.0, snapshots[2].Level)
	assert.Equal(t, unitResp.Unit.Id, snapshots[2].UnitId)
	assert.Equal(t, "inventory_update", snapshots[2].Source)
	assert.Equal(t, "Used for production", snapshots[2].Context)
	assert.Equal(t, "75.00", snapshots[2].Metadata["previous_level"])
	assert.Equal(t, "-25.00", snapshots[2].Metadata["change_amount"])

	// Verify timestamps are in correct order (ascending)
	assert.True(t, snapshots[0].Timestamp.AsTime().Before(snapshots[1].Timestamp.AsTime()))
	assert.True(t, snapshots[1].Timestamp.AsTime().Before(snapshots[2].Timestamp.AsTime()))

	// Test earliest and latest timestamps
	assert.NotNil(t, historyResp.EarliestTimestamp)
	assert.NotNil(t, historyResp.LatestTimestamp)
	assert.True(t, historyResp.EarliestTimestamp.AsTime().Before(historyResp.LatestTimestamp.AsTime()))
}

func TestGetItemHistoryItemNotFoundError(t *testing.T) {
	// Setup temporary database
	tmpDir, err := os.MkdirTemp("", "test_inventory_history")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	require.NoError(t, err)
	defer repo.Close()

	eventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	service := NewInventoryService(repo, eventBus, logger)
	ctx := context.Background()

	// Request history for non-existent item
	historyReq := &pb.GetItemHistoryRequest{
		ItemId: "non-existent-item-id",
	}

	historyResp, err := service.GetItemHistory(ctx, historyReq)

	// Should get error
	assert.Error(t, err)
	assert.Nil(t, historyResp)
	assert.Contains(t, err.Error(), "item not found")
}

func TestGetItemHistoryValidation(t *testing.T) {
	// Setup temporary database
	tmpDir, err := os.MkdirTemp("", "test_inventory_history")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	require.NoError(t, err)
	defer repo.Close()

	eventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	service := NewInventoryService(repo, eventBus, logger)
	ctx := context.Background()

	// Test with empty item ID
	historyReq := &pb.GetItemHistoryRequest{
		ItemId: "",
	}

	historyResp, err := service.GetItemHistory(ctx, historyReq)

	// Should get validation error
	assert.Error(t, err)
	assert.Nil(t, historyResp)
	assert.Contains(t, err.Error(), "item_id is required")
}
