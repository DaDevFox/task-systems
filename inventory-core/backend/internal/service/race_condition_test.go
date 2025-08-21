package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
)

// TestRaceConditionFixed demonstrates that the race condition is fixed
// by running concurrent operations and ensuring consistent results
func TestRaceConditionFixed(t *testing.T) {
	service, cleanup := setupTestServiceWithRealDB(t)
	defer cleanup()

	itemID, _ := createTestItemViaService(t, service)
	ctx := context.Background()

	// Test concurrent updates and history queries
	const numGoroutines = 10
	const numUpdates = 5

	var wg sync.WaitGroup
	results := make([]int, numGoroutines)

	// Start multiple goroutines that update inventory levels and immediately check history
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numUpdates; j++ {
				level := float64(50 + goroutineID*10 + j)
				
				// Update inventory level
				updateReq := &pb.UpdateInventoryLevelRequest{
					ItemId:   itemID,
					NewLevel: level,
					Reason:   "race_condition_test",
				}

				updateResp, err := service.UpdateInventoryLevel(ctx, updateReq)
				if err != nil {
					t.Errorf("Goroutine %d: failed to update level: %v", goroutineID, err)
					return
				}

				// Immediately check history - this is where race condition would manifest
				historyReq := &pb.GetItemHistoryRequest{
					ItemId: itemID,
					QueryParams: &pb.GetItemHistoryRequest_CountBased{
						CountBased: &pb.CountBasedQuery{Count: 100},
					},
				}

				historyResp, err := service.GetItemHistory(ctx, historyReq)
				if err != nil {
					t.Errorf("Goroutine %d: failed to get history: %v", goroutineID, err)
					return
				}

				// Verify that the history contains the level we just updated
				found := false
				for _, snapshot := range historyResp.History {
					if snapshot.Level == level {
						found = true
						break
					}
				}

				if found {
					results[goroutineID]++
				} else {
					t.Errorf("Goroutine %d: level %.1f not found in history after update", goroutineID, level)
				}

				// Small delay to allow other goroutines to interleave
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify that all updates were properly recorded in history
	for i, successCount := range results {
		assert.Equal(t, numUpdates, successCount, "Goroutine %d should have %d successful history checks", i, numUpdates)
	}

	// Final verification: check total history count
	finalHistoryReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
		QueryParams: &pb.GetItemHistoryRequest_CountBased{
			CountBased: &pb.CountBasedQuery{Count: 1000},
		},
	}

	finalHistoryResp, err := service.GetItemHistory(ctx, finalHistoryReq)
	require.NoError(t, err)

	// Should have initial history + all updates
	expectedMinCount := 1 + (numGoroutines * numUpdates)
	assert.GreaterOrEqual(t, len(finalHistoryResp.History), expectedMinCount,
		"Should have at least %d history entries (initial + %d updates)", expectedMinCount, numGoroutines*numUpdates)

	t.Logf("✓ Race condition test passed: %d total history entries found", len(finalHistoryResp.History))
}

// TestDeterministicHistoryOrdering verifies that history is returned in deterministic order
func TestDeterministicHistoryOrdering(t *testing.T) {
	service, cleanup := setupTestServiceWithRealDB(t)
	defer cleanup()

	itemID, _ := createTestItemViaService(t, service)
	ctx := context.Background()

	// Make several updates with specific timing
	levels := []float64{90.0, 80.0, 70.0, 60.0}
	for i, level := range levels {
		updateReq := &pb.UpdateInventoryLevelRequest{
			ItemId:   itemID,
			NewLevel: level,
			Reason:   "deterministic_test",
		}

		_, err := service.UpdateInventoryLevel(ctx, updateReq)
		require.NoError(t, err)

		// Ensure different timestamps
		if i < len(levels)-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Query history multiple times to ensure consistent ordering
	const queryCount = 5
	var allResults [][]float64

	for i := 0; i < queryCount; i++ {
		historyReq := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 10},
			},
		}

		historyResp, err := service.GetItemHistory(ctx, historyReq)
		require.NoError(t, err)

		levelSequence := make([]float64, len(historyResp.History))
		for j, snapshot := range historyResp.History {
			levelSequence[j] = snapshot.Level
		}

		allResults = append(allResults, levelSequence)

		// Small delay between queries
		time.Sleep(time.Millisecond)
	}

	// Verify all results are identical
	for i := 1; i < len(allResults); i++ {
		assert.Equal(t, allResults[0], allResults[i], "Query %d should return same order as query 0", i)
	}

	t.Logf("✓ Deterministic ordering verified: %d queries returned identical sequences", queryCount)
}
