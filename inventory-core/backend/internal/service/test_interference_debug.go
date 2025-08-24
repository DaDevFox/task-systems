package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
)

// TestDebugInterference - Run the problematic sequence in isolation to debug
func TestDebugInterference(t *testing.T) {
	t.Log("=== DEBUG: Testing interference patterns ===")

	// Run the same sequence as the failing tests
	t.Run("Step1_CompleteWorkflow", func(t *testing.T) {
		service, cleanup := setupTestServiceWithRealDB(t)
		defer cleanup()

		itemID, _ := createTestItemViaService(t, service)
		ctx := context.Background()

		// Make updates like TestServiceAPICompleteWorkflow
		updates := []struct {
			level  float64
			reason string
		}{
			{75.0, "consumption"},
			{90.0, "restock"},
			{50.0, "consumption"},
		}

		for _, update := range updates {
			updateReq := &pb.UpdateInventoryLevelRequest{
				ItemId:            itemID,
				NewLevel:          update.level,
				Reason:            update.reason,
				RecordConsumption: update.reason == "consumption",
			}
			_, err := service.UpdateInventoryLevel(ctx, updateReq)
			require.NoError(t, err)
		}

		// Check final history
		historyReq := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 10},
			},
		}
		historyResp, err := service.GetItemHistory(ctx, historyReq)
		require.NoError(t, err)

		t.Logf("Step1 final history (%d entries):", len(historyResp.History))
		for i, snapshot := range historyResp.History {
			t.Logf("  [%d] Level=%.1f, Source=%s, Context=%s", i, snapshot.Level, snapshot.Source, snapshot.Context)
		}
	})

	t.Run("Step2_InitialHistory", func(t *testing.T) {
		service, cleanup := setupTestServiceWithRealDB(t)
		defer cleanup()

		itemID, unitID := createTestItemViaService(t, service)
		ctx := context.Background()

		historyReq := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 10},
			},
		}

		historyResp, err := service.GetItemHistory(ctx, historyReq)
		require.NoError(t, err)

		t.Logf("Step2 initial history (%d entries):", len(historyResp.History))
		for i, snapshot := range historyResp.History {
			t.Logf("  [%d] Level=%.1f, Source=%s, Context=%s", i, snapshot.Level, snapshot.Source, snapshot.Context)
		}

		assert.GreaterOrEqual(t, len(historyResp.History), 1, "Should have initial creation snapshot")
		if len(historyResp.History) > 0 {
			initialSnapshot := historyResp.History[0]
			assert.Equal(t, 100.0, initialSnapshot.Level, "Initial level should be 100.0")
			assert.Equal(t, "initial_creation", initialSnapshot.Source, "Source should be initial_creation")
			assert.Equal(t, unitID, initialSnapshot.UnitId, "Unit ID should match")
		}
	})

	t.Run("Step3_UpdateLevel", func(t *testing.T) {
		service, cleanup := setupTestServiceWithRealDB(t)
		defer cleanup()

		itemID, _ := createTestItemViaService(t, service)
		ctx := context.Background()

		// Check initial history
		initialHistoryReq := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 10},
			},
		}
		initialHistoryResp, err := service.GetItemHistory(ctx, initialHistoryReq)
		require.NoError(t, err)
		t.Logf("Step3 initial history (%d entries):", len(initialHistoryResp.History))

		// Update level
		updateReq := &pb.UpdateInventoryLevelRequest{
			ItemId:            itemID,
			NewLevel:          75.0,
			Reason:            "service_api_test",
			RecordConsumption: false,
		}

		updateResp, err := service.UpdateInventoryLevel(ctx, updateReq)
		require.NoError(t, err)
		assert.True(t, updateResp.LevelChanged)

		// Check history after update
		historyReq := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 10},
			},
		}

		historyResp, err := service.GetItemHistory(ctx, historyReq)
		require.NoError(t, err)

		t.Logf("Step3 history after update (%d entries):", len(historyResp.History))
		for i, snapshot := range historyResp.History {
			t.Logf("  [%d] Level=%.1f, Source=%s, Context=%s", i, snapshot.Level, snapshot.Source, snapshot.Context)
		}

		assert.GreaterOrEqual(t, len(historyResp.History), 2, "Should have at least 2 history entries after update")
	})

	t.Run("Step4_MultipleUpdates", func(t *testing.T) {
		service, cleanup := setupTestServiceWithRealDB(t)
		defer cleanup()

		itemID, _ := createTestItemViaService(t, service)
		ctx := context.Background()

		// Check initial history
		initialHistoryReq := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 10},
			},
		}
		initialHistoryResp, err := service.GetItemHistory(ctx, initialHistoryReq)
		require.NoError(t, err)
		t.Logf("Step4 initial history (%d entries):", len(initialHistoryResp.History))

		// Make multiple updates
		updates := []struct {
			level  float64
			reason string
		}{
			{60.0, "consumption"},
			{80.0, "restock"},
			{45.0, "consumption"},
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
			t.Logf("Step4 Update %d: Level changed to %.1f, LevelChanged=%t", i+1, updateResp.Item.CurrentLevel, updateResp.LevelChanged)
			time.Sleep(2 * time.Millisecond)
		}

		// Check complete history
		historyReq := &pb.GetItemHistoryRequest{
			ItemId: itemID,
			QueryParams: &pb.GetItemHistoryRequest_CountBased{
				CountBased: &pb.CountBasedQuery{Count: 20},
			},
		}

		historyResp, err := service.GetItemHistory(ctx, historyReq)
		require.NoError(t, err)

		t.Logf("Step4 complete history (%d entries):", len(historyResp.History))
		for i, snapshot := range historyResp.History {
			t.Logf("  [%d] Level=%.1f, Source=%s, Context=%s, Time=%s", i, snapshot.Level, snapshot.Source, snapshot.Context, snapshot.Timestamp.AsTime().Format("15:04:05.000"))
		}

		// Verify we have the expected number of entries
		assert.GreaterOrEqual(t, len(historyResp.History), 4, "Should have complete timeline (initial + 3 updates)")

		// Check if initial level exists
		levels := make([]float64, 0)
		for _, snapshot := range historyResp.History {
			levels = append(levels, snapshot.Level)
		}

		t.Logf("Step4 levels found: %v", levels)

		levelMap := make(map[float64]bool)
		for _, level := range levels {
			levelMap[level] = true
		}

		t.Logf("Step4 level map: %v", levelMap)
		assert.True(t, levelMap[100.0], "Should have initial level 100.0")
		assert.True(t, levelMap[60.0], "Should have updated level 60.0")
		assert.True(t, levelMap[80.0], "Should have updated level 80.0")
		assert.True(t, levelMap[45.0], "Should have updated level 45.0")
	})
}

// TestConcurrentHistoryAccess - Test potential race conditions
func TestConcurrentHistoryAccess(t *testing.T) {
	service, cleanup := setupTestServiceWithRealDB(t)
	defer cleanup()

	itemID, _ := createTestItemViaService(t, service)
	ctx := context.Background()

	t.Log("=== DEBUG: Testing concurrent access patterns ===")

	// Simulate concurrent history queries during updates
	done := make(chan bool)

	// Goroutine 1: Continuous updates
	go func() {
		for i := 0; i < 5; i++ {
			level := 50.0 + float64(i*10)
			updateReq := &pb.UpdateInventoryLevelRequest{
				ItemId:            itemID,
				NewLevel:          level,
				Reason:            "concurrent_test",
				RecordConsumption: false,
			}
			_, err := service.UpdateInventoryLevel(ctx, updateReq)
			if err != nil {
				t.Errorf("Concurrent update failed: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Continuous history queries
	go func() {
		for i := 0; i < 10; i++ {
			historyReq := &pb.GetItemHistoryRequest{
				ItemId: itemID,
				QueryParams: &pb.GetItemHistoryRequest_CountBased{
					CountBased: &pb.CountBasedQuery{Count: 20},
				},
			}
			historyResp, err := service.GetItemHistory(ctx, historyReq)
			if err != nil {
				t.Errorf("Concurrent history query failed: %v", err)
			} else {
				t.Logf("Concurrent query %d: %d history entries", i, len(historyResp.History))
			}
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Final history check
	historyReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
		QueryParams: &pb.GetItemHistoryRequest_CountBased{
			CountBased: &pb.CountBasedQuery{Count: 20},
		},
	}
	historyResp, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)

	t.Logf("Final concurrent test history (%d entries):", len(historyResp.History))
	for i, snapshot := range historyResp.History {
		t.Logf("  [%d] Level=%.1f, Source=%s, Context=%s", i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// Should still have initial creation entry
	hasInitial := false
	for _, snapshot := range historyResp.History {
		if snapshot.Source == "initial_creation" && snapshot.Level == 100.0 {
			hasInitial = true
			break
		}
	}
	assert.True(t, hasInitial, "Should still have initial creation entry after concurrent access")
}
