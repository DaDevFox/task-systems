package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/backend/pkg/proto/events/v1"
)

// TestHistoryCreationDiagnostics tests the basic flow of history creation step by step
func TestHistoryCreationDiagnostics(t *testing.T) {
	// Setup temporary database with more verbose logging
	tmpDir, err := os.MkdirTemp("", "test_history_diagnostics")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "diagnostics.db")
	repo, err := repository.NewInventoryRepository(dbPath, repository.DatabaseTypeBadger)
	require.NoError(t, err)
	defer repo.Close()

	eventBus := events.GetGlobalBus("history_diagnostics")
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel) // More verbose for debugging

	service := NewInventoryService(repo, eventBus, logger)
	ctx := context.Background()

	// Create unit
	unitReq := &pb.AddUnitRequest{
		Name:                 "Kilograms",
		Symbol:               "kg",
		Description:          "Unit of mass",
		BaseConversionFactor: 1.0,
		Category:             "weight",
	}
	unitResp, err := service.AddUnit(ctx, unitReq)
	require.NoError(t, err)
	t.Logf("✓ Unit created: ID=%s", unitResp.Unit.Id)

	// Create item
	addReq := &pb.AddInventoryItemRequest{
		Name:              "Diagnostic Test Item",
		Description:       "Item for step-by-step testing",
		InitialLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitId:            unitResp.Unit.Id,
	}
	addResp, err := service.AddInventoryItem(ctx, addReq)
	require.NoError(t, err)
	itemID := addResp.Item.Id
	t.Logf("✓ Item created: ID=%s, Initial Level=%.1f", itemID, addResp.Item.CurrentLevel)

	// Step 1: Check initial history via repository directly
	filters := repository.HistoryFilters{Limit: 10}
	snapshots, totalCount, err := repo.GetInventoryHistory(ctx, itemID, filters)
	require.NoError(t, err)
	t.Logf("Repository direct query - Snapshots: %d, Total: %d", len(snapshots), totalCount)
	for i, snapshot := range snapshots {
		t.Logf("  Snapshot[%d]: Level=%.1f, Source=%s, Time=%s", i, snapshot.Level, snapshot.Source, snapshot.Timestamp.Format("15:04:05"))
	}

	// Step 2: Check initial history via service API
	historyReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
		QueryParams: &pb.GetItemHistoryRequest_CountBased{
			CountBased: &pb.CountBasedQuery{
				Count: 10,
			},
		},
	}
	historyResp, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)
	t.Logf("Service API query - History entries: %d, Total points: %d", len(historyResp.History), historyResp.TotalPoints)
	for i, snapshot := range historyResp.History {
		t.Logf("  History[%d]: Level=%.1f, Source=%s, Context=%s", i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// Verify we have initial history
	assert.GreaterOrEqual(t, len(historyResp.History), 1, "Should have initial history")

	// Step 3: Update inventory level
	t.Logf("\n--- Making first inventory level update ---")
	updateReq := &pb.UpdateInventoryLevelRequest{
		ItemId:            itemID,
		NewLevel:          75.0,
		Reason:            "diagnostic_test_update",
		RecordConsumption: false,
	}
	updateResp, err := service.UpdateInventoryLevel(ctx, updateReq)
	require.NoError(t, err)
	t.Logf("✓ Level update completed: LevelChanged=%t, New Level=%.1f", updateResp.LevelChanged, updateResp.Item.CurrentLevel)

	// Step 4: Check history after update - repository direct
	snapshotsAfter, totalCountAfter, err := repo.GetInventoryHistory(ctx, itemID, filters)
	require.NoError(t, err)
	t.Logf("Repository after update - Snapshots: %d, Total: %d", len(snapshotsAfter), totalCountAfter)
	for i, snapshot := range snapshotsAfter {
		t.Logf("  Snapshot[%d]: Level=%.1f, Source=%s, Context=%s", i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// Step 5: Check history after update - service API
	historyRespAfter, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)
	t.Logf("Service API after update - History entries: %d, Total points: %d", len(historyRespAfter.History), historyRespAfter.TotalPoints)
	for i, snapshot := range historyRespAfter.History {
		t.Logf("  History[%d]: Level=%.1f, Source=%s, Context=%s", i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// Final verification
	if len(historyRespAfter.History) >= 2 {
		t.Logf("✓ SUCCESS: Found multiple history entries as expected")

		// Look for our specific update
		foundUpdate := false
		for _, snapshot := range historyRespAfter.History {
			if snapshot.Level == 75.0 {
				foundUpdate = true
				t.Logf("✓ Found our update: Level=%.1f, Source=%s", snapshot.Level, snapshot.Source)
				break
			}
		}
		assert.True(t, foundUpdate, "Should find the 75.0 level update")
	} else {
		t.Logf("❌ ISSUE: Only found %d history entries, expected at least 2", len(historyRespAfter.History))

		// Diagnostic: Let's see what's actually stored in the database
		t.Logf("\n--- Diagnostic: Checking database state directly ---")

		// Try to get the item again to see its current state
		item, err := repo.GetItem(ctx, itemID)
		require.NoError(t, err)
		t.Logf("Item current state: Level=%.1f, UpdatedAt=%s", item.CurrentLevel, item.UpdatedAt.Format("15:04:05"))

		// Check if there are any database errors or warnings in the logs
		t.Logf("Check the service logs above for any warnings about snapshot storage failures")
	}
}
