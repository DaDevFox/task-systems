package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
)

// TestSequentialUpdatesConsistent tests that sequential updates create consistent history
// This tests the core race condition fix without concurrent access complexity
func TestSequentialUpdatesConsistent(t *testing.T) {
	service, cleanup := setupTestServiceWithRealDB(t)
	defer cleanup()

	itemID, _ := createTestItemViaService(t, service)
	ctx := context.Background()

	// Test that each update immediately reflects in history
	levels := []float64{90.0, 80.0, 70.0, 60.0, 50.0}

	for i, level := range levels {
		// Update inventory level
		updateReq := &pb.UpdateInventoryLevelRequest{
			ItemId:   itemID,
			NewLevel: level,
			Reason:   "sequential_test",
		}

		_, err := service.UpdateInventoryLevel(ctx, updateReq)
		require.NoError(t, err, "Update %d should succeed", i)

		// Immediately check history - this is where the race condition would occur
		historyReq := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 10},
			},
		}

		historyResp, err := service.GetItemHistory(ctx, historyReq)
		require.NoError(t, err, "History query %d should succeed", i)

		// Verify that the history contains the level we just updated
		found := false
		for _, snapshot := range historyResp.History {
			if snapshot.Level == level {
				found = true
				break
			}
		}

		assert.True(t, found, "Level %.1f should be found in history after update %d", level, i)

		// Verify we have the expected number of history entries (initial + updates so far)
		expectedCount := 1 + i + 1 // initial + (i+1) updates
		assert.Equal(t, expectedCount, len(historyResp.History),
			"Should have %d history entries after update %d", expectedCount, i)

		// Small delay between updates to ensure different timestamps
		time.Sleep(2 * time.Millisecond)
	}

	t.Logf("✓ Sequential updates test passed: All %d updates immediately visible in history", len(levels))
}

// TestInitialHistoryAlwaysPresent tests that initial history is always created atomically with item
func TestInitialHistoryAlwaysPresent(t *testing.T) {
	// Run this test multiple times to catch any timing issues
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			service, cleanup := setupTestServiceWithRealDB(t)
			defer cleanup()

			itemID, unitID := createTestItemViaService(t, service)
			ctx := context.Background()

			// Immediately check for initial history - this is where race condition would occur
			historyReq := &pb.GetItemHistoryRequest{
				ItemId: itemID,
				QueryParams: &pb.GetItemHistoryRequest_CountBased{
					CountBased: &pb.CountBasedQuery{Count: 10},
				},
			}

			historyResp, err := service.GetItemHistory(ctx, historyReq)
			require.NoError(t, err)

			// Should always have exactly 1 initial history entry
			assert.Equal(t, 1, len(historyResp.History), "Should have exactly 1 initial history entry")
			assert.Equal(t, int32(1), historyResp.TotalPoints, "Should have exactly 1 total point")

			if len(historyResp.History) > 0 {
				initialSnapshot := historyResp.History[0]
				assert.Equal(t, 100.0, initialSnapshot.Level, "Initial level should be 100.0")
				assert.Equal(t, "initial_creation", initialSnapshot.Source, "Source should be initial_creation")
				assert.Equal(t, unitID, initialSnapshot.UnitId, "Unit ID should match")
			}
		})
	}

	t.Logf("✓ Initial history always present: Passed 5 iterations")
}
