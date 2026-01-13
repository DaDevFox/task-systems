package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
)

func TestBoltDBOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping BoltDB only test in short mode")
	}

	// Create temporary directory for BoltDB only
	tempDir, err := os.MkdirTemp("", "bolt_only_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	boltPath := filepath.Join(tempDir, "test.db")
	t.Logf("BoltDB path: %s", boltPath)

	ctx := context.Background()

	// Create BoltDB repository
	boltRepo, err := repository.NewCompactInventoryRepository(boltPath)
	require.NoError(t, err)
	defer boltRepo.Close()

	// Add test data
	testUnit := &domain.Unit{
		ID:   "unit1",
		Name: "pieces",
	}

	err = boltRepo.AddUnit(ctx, testUnit)
	require.NoError(t, err)

	testItem := &domain.InventoryItem{
		ID:                "item1",
		Name:              "Test Item",
		Description:       "Test Description",
		UnitID:            "unit1",
		CurrentLevel:      100,
		LowStockThreshold: 10,
	}

	err = boltRepo.AddItem(ctx, testItem)
	require.NoError(t, err)

	// Add some history
	snapshot := &domain.InventoryLevelSnapshot{
		Timestamp: time.Now(),
		Level:     100,
		UnitID:    "unit1",
		Source:    "test",
		Context:   "Test snapshot",
	}

	err = boltRepo.AddInventorySnapshot(ctx, "item1", snapshot)
	require.NoError(t, err)

	// Verify data was stored
	retrievedItem, err := boltRepo.GetItem(ctx, "item1")
	require.NoError(t, err)
	require.Equal(t, "Test Item", retrievedItem.Name)

	t.Log("BoltDB test completed successfully!")
}
