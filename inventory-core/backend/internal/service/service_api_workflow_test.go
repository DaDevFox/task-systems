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
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/backend/pkg/proto/events/v1"
)

// TestServiceAPICompleteWorkflow demonstrates that history management works
// completely through the service API without needing direct repository access
func TestServiceAPICompleteWorkflow(t *testing.T) {
	// Setup clean test environment
	tmpDir, err := os.MkdirTemp("", "test_complete_workflow")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "workflow_test.db")
	repo, err := repository.NewInventoryRepository(dbPath, repository.DatabaseTypeBadger)
	require.NoError(t, err)
	defer repo.Close()

	eventBus := events.NewEventBus("workflow_test")
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	service := NewInventoryService(repo, eventBus, logger)
	ctx := context.Background()

	t.Log("=== WORKFLOW: Complete inventory management through service API ===")

	// Step 1: Create unit through service
	unitReq := &pb.AddUnitRequest{
		Name:                 "Kilograms",
		Symbol:               "kg",
		Description:          "Unit of mass",
		BaseConversionFactor: 1.0,
		Category:             "weight",
	}
	unitResp, err := service.AddUnit(ctx, unitReq)
	require.NoError(t, err)
	t.Logf("✓ Step 1: Unit created via service API (ID: %s)", unitResp.Unit.Id)

	// Step 2: Create inventory item through service (creates initial history)
	addReq := &pb.AddInventoryItemRequest{
		Name:              "Complete Workflow Item",
		Description:       "Item managed entirely through service API",
		InitialLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitId:            unitResp.Unit.Id,
	}
	addResp, err := service.AddInventoryItem(ctx, addReq)
	require.NoError(t, err)
	itemID := addResp.Item.Id
	t.Logf("✓ Step 2: Item created via service API (ID: %s, Level: %.1f)", itemID, addResp.Item.CurrentLevel)

	// Step 3: Check initial history through service API
	historyReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
		QueryParams: &pb.GetItemHistoryRequest_CountBased{
			CountBased: &pb.CountBasedQuery{Count: 10},
		},
	}
	historyResp, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(historyResp.History), 1, "Should have initial history")
	t.Logf("✓ Step 3: Initial history retrieved via service API (%d entries)", len(historyResp.History))

	// Step 4: Update inventory levels through service API (creates more history)
	updates := []struct {
		level  float64
		reason string
	}{
		{75.0, "consumption"},
		{90.0, "restock"},
		{50.0, "consumption"},
	}

	for i, update := range updates {
		updateReq := &pb.UpdateInventoryLevelRequest{
			ItemId:            itemID,
			NewLevel:          update.level,
			Reason:            update.reason,
			RecordConsumption: update.reason == "consumption",
		}
		updateResp, err := service.UpdateInventoryLevel(ctx, updateReq)
		require.NoError(t, err)
		t.Logf("✓ Step 4.%d: Level updated via service API to %.1f (reason: %s)", i+1, updateResp.Item.CurrentLevel, update.reason)

		// Small delay to ensure distinct timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// Step 5: Retrieve complete history through service API
	finalHistoryResp, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)

	t.Logf("✓ Step 5: Complete history retrieved via service API (%d entries):", len(finalHistoryResp.History))
	for i, snapshot := range finalHistoryResp.History {
		t.Logf("    [%d] Level: %.1f, Source: %s, Context: %s", i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// Verify complete workflow succeeded
	assert.GreaterOrEqual(t, len(finalHistoryResp.History), 4, "Should have complete history (initial + 3 updates)")

	// Verify we can see different types of operations
	sources := make(map[string]bool)
	levels := make(map[float64]bool)
	for _, snapshot := range finalHistoryResp.History {
		sources[snapshot.Source] = true
		levels[snapshot.Level] = true
	}

	assert.True(t, sources["initial_creation"], "Should have initial creation")
	assert.True(t, sources["inventory_update"], "Should have inventory updates")
	assert.True(t, levels[100.0], "Should have initial level")
	assert.True(t, levels[75.0], "Should have first update level")

	t.Log("✅ WORKFLOW COMPLETE: All inventory operations performed through service API")
	t.Log("✅ VERIFICATION: History created and accessed without direct repository access")
	t.Logf("✅ SUMMARY: Created item with %d history snapshots using only service methods", len(finalHistoryResp.History))
}
