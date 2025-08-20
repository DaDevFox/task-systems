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

func TestSnapshotCreationDebugging(t *testing.T) {
	// Setup temporary database
	tmpDir, err := os.MkdirTemp("", "test_snapshot_debug")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "debug.db")
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	require.NoError(t, err)
	defer repo.Close()

	eventBus := events.GetGlobalBus("debug")
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	service := NewInventoryService(repo, eventBus, logger)
	ctx := context.Background()

	// Create a unit first
	unitReq := &pb.AddUnitRequest{
		Name:                   "Kilograms",
		Symbol:                 "kg",
		Description:            "Unit of mass",
		BaseConversionFactor:   1.0,
		Category:              "weight",
	}
	unitResp, err := service.AddUnit(ctx, unitReq)
	require.NoError(t, err)
	
	t.Logf("Unit created with ID: %s", unitResp.Unit.Id)

	// Create an inventory item
	addReq := &pb.AddInventoryItemRequest{
		Name:              "Debug Test Item",
		Description:       "Item for debugging snapshot creation",
		InitialLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitId:            unitResp.Unit.Id,
	}

	addResp, err := service.AddInventoryItem(ctx, addReq)
	require.NoError(t, err)
	require.NotNil(t, addResp)
	
	itemID := addResp.Item.Id
	t.Logf("Item created with ID: %s", itemID)

	// Now check if snapshots were created by calling the repository directly
	filters := repository.HistoryFilters{}
	snapshots, totalCount, err := repo.GetInventoryHistory(ctx, itemID, filters)
	require.NoError(t, err)

	t.Logf("Direct repository call: %d snapshots, total count: %d", len(snapshots), totalCount)
	for i, s := range snapshots {
		t.Logf("Snapshot %d: Level=%.1f, Source=%s, Time=%s", i, s.Level, s.Source, s.Timestamp.Format(time.RFC3339))
	}

	// Should have 1 snapshot from item creation
	assert.Len(t, snapshots, 1)
	assert.Equal(t, 1, totalCount)

	if len(snapshots) > 0 {
		assert.Equal(t, 100.0, snapshots[0].Level)
		assert.Equal(t, "initial_creation", snapshots[0].Source)
	}

	// Now test the service endpoint
	historyReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
	}

	historyResp, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)
	require.NotNil(t, historyResp)

	t.Logf("Service endpoint response: %d snapshots, total: %d", len(historyResp.History), historyResp.TotalPoints)

	// Should match the direct repository call
	assert.Len(t, historyResp.History, 1)
	assert.Equal(t, int32(1), historyResp.TotalPoints)
}
