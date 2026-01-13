package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
)

func TestAddInventorySnapshotAndGetHistory(t *testing.T) {
	// Setup temporary database
	tmpDir, err := os.MkdirTemp("", "test_inventory_history")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := NewBadgerInventoryRepository(dbPath)
	require.NoError(t, err)
	defer repo.Close()

	ctx := context.Background()
	itemID := "test-item-1"

	// Create snapshots with different timestamps
	now := time.Now()
	snapshot1 := &domain.InventoryLevelSnapshot{
		Timestamp: now.Add(-2 * time.Hour),
		Level:     100.0,
		UnitID:    "unit-1",
		Source:    "initial_creation",
		Context:   "Item created",
		Metadata:  map[string]string{"created_by": "system"},
	}

	snapshot2 := &domain.InventoryLevelSnapshot{
		Timestamp: now.Add(-1 * time.Hour),
		Level:     75.0,
		UnitID:    "unit-1",
		Source:    "inventory_update",
		Context:   "Manual adjustment",
		Metadata:  map[string]string{"previous_level": "100.00"},
	}

	snapshot3 := &domain.InventoryLevelSnapshot{
		Timestamp: now,
		Level:     50.0,
		UnitID:    "unit-1",
		Source:    "inventory_update",
		Context:   "Used in production",
		Metadata:  map[string]string{"previous_level": "75.00"},
	}

	// Add snapshots
	err = repo.AddInventorySnapshot(ctx, itemID, snapshot1)
	require.NoError(t, err)

	err = repo.AddInventorySnapshot(ctx, itemID, snapshot2)
	require.NoError(t, err)

	err = repo.AddInventorySnapshot(ctx, itemID, snapshot3)
	require.NoError(t, err)

	// Test GetInventoryHistory
	filters := HistoryFilters{
		StartTime: now.Add(-3 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
		Limit:     10,
	}

	snapshots, totalCount, err := repo.GetInventoryHistory(ctx, itemID, filters)
	require.NoError(t, err)

	t.Logf("Retrieved %d snapshots, total count: %d", len(snapshots), totalCount)
	for i, s := range snapshots {
		t.Logf("Snapshot %d: Level=%.1f, Time=%s", i, s.Level, s.Timestamp.Format(time.RFC3339))
	}

	// Should have all 3 snapshots
	assert.Len(t, snapshots, 3)
	assert.Equal(t, 3, totalCount)

	// Verify they're in chronological order
	assert.Equal(t, 100.0, snapshots[0].Level)
	assert.Equal(t, 75.0, snapshots[1].Level)
	assert.Equal(t, 50.0, snapshots[2].Level)

	// Test with limited results
	limitedFilters := HistoryFilters{
		Limit: 2,
	}

	limitedSnapshots, limitedTotal, err := repo.GetInventoryHistory(ctx, itemID, limitedFilters)
	require.NoError(t, err)

	t.Logf("Limited query: Retrieved %d snapshots, total count: %d", len(limitedSnapshots), limitedTotal)

	// Should have only 2 snapshots but know there are 3 total
	assert.Len(t, limitedSnapshots, 2)
	assert.Equal(t, 3, limitedTotal)

	// Test with time range filtering
	recentFilters := HistoryFilters{
		StartTime: now.Add(-30 * time.Minute),
		EndTime:   now.Add(30 * time.Minute),
	}

	recentSnapshots, recentTotal, err := repo.GetInventoryHistory(ctx, itemID, recentFilters)
	require.NoError(t, err)

	// Should have only the latest snapshot
	assert.Len(t, recentSnapshots, 1)
	assert.Equal(t, 1, recentTotal)
	assert.Equal(t, 50.0, recentSnapshots[0].Level)

	// Test GetEarliestSnapshot
	earliest, err := repo.GetEarliestSnapshot(ctx, itemID)
	require.NoError(t, err)
	require.NotNil(t, earliest)
	assert.Equal(t, 100.0, earliest.Level)

	// Test GetLatestSnapshot
	latest, err := repo.GetLatestSnapshot(ctx, itemID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, 50.0, latest.Level)
}

func TestGetInventoryHistoryNoResults(t *testing.T) {
	// Setup temporary database
	tmpDir, err := os.MkdirTemp("", "test_inventory_history")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := NewBadgerInventoryRepository(dbPath)
	require.NoError(t, err)
	defer repo.Close()

	ctx := context.Background()
	itemID := "nonexistent-item"

	// Test GetInventoryHistory for non-existent item
	filters := HistoryFilters{
		Limit: 10,
	}

	snapshots, totalCount, err := repo.GetInventoryHistory(ctx, itemID, filters)
	require.NoError(t, err)

	// Should have no snapshots
	assert.Len(t, snapshots, 0)
	assert.Equal(t, 0, totalCount)

	// Test GetEarliestSnapshot for non-existent item
	earliest, err := repo.GetEarliestSnapshot(ctx, itemID)
	require.NoError(t, err)
	assert.Nil(t, earliest)

	// Test GetLatestSnapshot for non-existent item
	latest, err := repo.GetLatestSnapshot(ctx, itemID)
	require.NoError(t, err)
	assert.Nil(t, latest)
}
