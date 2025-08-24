package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
)

const logFormatHistoryEntry = "  [%d] Level=%.1f, Source=%s, Context=%s"

func TestServiceAPIInitialHistoryCreated(t *testing.T) {
	service, cleanup := setupTestServiceWithRealDB(t)
	defer cleanup()

	itemID, unitID := createTestItemViaService(t, service)
	ctx := context.Background()

	// Check initial history using service API with proper query params
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
	require.NotNil(t, historyResp)

	// Should have 1 snapshot from item creation
	assert.GreaterOrEqual(t, len(historyResp.History), 1, "Should have initial creation snapshot")
	assert.GreaterOrEqual(t, historyResp.TotalPoints, int32(1), "Should have at least one total point")

	initialSnapshot := historyResp.History[0]
	assert.Equal(t, 100.0, initialSnapshot.Level, "Initial level should be 100.0")
	assert.Equal(t, "initial_creation", initialSnapshot.Source, "Source should be initial_creation")
	assert.Equal(t, unitID, initialSnapshot.UnitId, "Unit ID should match")

	t.Logf("✓ Initial history verified: %d snapshots, level: %.1f", len(historyResp.History), initialSnapshot.Level)
}

func TestServiceAPIUpdateInventoryLevelCreatesHistory(t *testing.T) {
	service, cleanup := setupTestServiceWithRealDB(t)
	defer cleanup()

	itemID, _ := createTestItemViaService(t, service)
	ctx := context.Background()

	t.Log("--- Checking initial history before update ---")
	initialHistoryReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
		QueryParams: &pb.GetItemHistoryRequest_CountBased{
			CountBased: &pb.CountBasedQuery{Count: 10},
		},
	}
	initialHistoryResp, err := service.GetItemHistory(ctx, initialHistoryReq)
	require.NoError(t, err)
	t.Logf("Initial history entries: %d", len(initialHistoryResp.History))
	for i, snapshot := range initialHistoryResp.History {
		t.Logf(logFormatHistoryEntry, i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// Update inventory level through service API - using correct protobuf fields
	updateReq := &pb.UpdateInventoryLevelRequest{
		ItemId:            itemID,
		NewLevel:          75.0,
		Reason:            "service_api_test",
		RecordConsumption: false,
	}

	updateResp, err := service.UpdateInventoryLevel(ctx, updateReq)
	require.NoError(t, err)
	require.NotNil(t, updateResp)

	// Verify the update was applied - check the item itself
	assert.True(t, updateResp.LevelChanged)
	assert.Equal(t, 75.0, updateResp.Item.CurrentLevel, "Current level should be updated to 75.0")

	// Check that history now has 2 entries
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

	t.Logf("History after update (%d entries):", len(historyResp.History))
	for i, snapshot := range historyResp.History {
		t.Logf(logFormatHistoryEntry, i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// Should now have 2 snapshots: initial creation + level update
	assert.GreaterOrEqual(t, len(historyResp.History), 2, "Should have at least 2 history entries after update")

	// Find the level update snapshot - look for the latest one with the correct level
	var levelUpdateSnapshot *pb.InventoryLevelSnapshot
	for _, snapshot := range historyResp.History {
		if snapshot.Level == 75.0 {
			levelUpdateSnapshot = snapshot
			break
		}
	}

	require.NotNil(t, levelUpdateSnapshot, "Should find the level update snapshot with 75.0 level")
	assert.Equal(t, 75.0, levelUpdateSnapshot.Level, "Updated level should be 75.0")

	t.Logf("✓ Level update history verified: Level=%.1f, reason provided", levelUpdateSnapshot.Level)
}

func TestServiceAPIMultipleUpdatesCreateTimeline(t *testing.T) {
	service, cleanup := setupTestServiceWithRealDB(t)
	defer cleanup()

	itemID, _ := createTestItemViaService(t, service)
	ctx := context.Background()

	t.Log("--- Checking initial history before updates ---")
	initialHistoryReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
		QueryParams: &pb.GetItemHistoryRequest_CountBased{
			CountBased: &pb.CountBasedQuery{Count: 10},
		},
	}
	initialHistoryResp, err := service.GetItemHistory(ctx, initialHistoryReq)
	require.NoError(t, err)
	t.Logf("Initial history entries: %d", len(initialHistoryResp.History))
	for i, snapshot := range initialHistoryResp.History {
		t.Logf(logFormatHistoryEntry, i, snapshot.Level, snapshot.Source, snapshot.Context)
	}

	// Make several updates to create a timeline - using correct protobuf fields
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
		t.Logf("Update %d: Level changed to %.1f, LevelChanged=%t", i+1, updateResp.Item.CurrentLevel, updateResp.LevelChanged)

		// Add small delay to ensure timestamps are different
		time.Sleep(2 * time.Millisecond)
	}

	// Check complete history
	historyReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
		QueryParams: &pb.GetItemHistoryRequest_CountBased{
			CountBased: &pb.CountBasedQuery{
				Count: 20, // Get all history
			},
		},
	}

	historyResp, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)

	t.Logf("Complete history (%d entries):", len(historyResp.History))
	for i, snapshot := range historyResp.History {
		t.Logf("  [%d] Level=%.1f, Source=%s, Context=%s, Time=%s", i, snapshot.Level, snapshot.Source, snapshot.Context, snapshot.Timestamp.AsTime().Format("15:04:05.000"))
	}

	// Should have 4+ snapshots: initial + 3 updates
	assert.GreaterOrEqual(t, len(historyResp.History), 4, "Should have complete timeline (initial + 3 updates)")

	// Verify we can see the progression of levels
	levels := make([]float64, 0)
	for _, snapshot := range historyResp.History {
		levels = append(levels, snapshot.Level)
	}

	// Should include all our test levels
	levelMap := make(map[float64]bool)
	for _, level := range levels {
		levelMap[level] = true
	}

	assert.True(t, levelMap[100.0], "Should have initial level 100.0")
	assert.True(t, levelMap[60.0], "Should have updated level 60.0")
	assert.True(t, levelMap[80.0], "Should have updated level 80.0")
	assert.True(t, levelMap[45.0], "Should have updated level 45.0")

	t.Logf("✓ Timeline verified with %d entries and levels: %v", len(historyResp.History), levels)
}

func TestServiceAPITimeRangeQuery(t *testing.T) {
	service, cleanup := setupTestServiceWithRealDB(t)
	defer cleanup()

	itemID, _ := createTestItemViaService(t, service)
	ctx := context.Background()

	// Add one more history point
	updateReq := &pb.UpdateInventoryLevelRequest{
		ItemId:            itemID,
		NewLevel:          90.0,
		Reason:            "time_range_test",
		RecordConsumption: false,
	}
	_, err := service.UpdateInventoryLevel(ctx, updateReq)
	require.NoError(t, err)

	// Test time range query using service API - use a much wider time range to ensure we capture data
	now := time.Now()
	historyReq := &pb.GetItemHistoryRequest{
		ItemId: itemID,
		QueryParams: &pb.GetItemHistoryRequest_TimeRange{
			TimeRange: &pb.TimeRangeQuery{
				StartTime:   timestamppb.New(now.Add(-24 * time.Hour)), // Go back 24 hours
				EndTime:     timestamppb.New(now.Add(1 * time.Hour)),   // Go forward 1 hour
				Granularity: pb.HistoryGranularity_HISTORY_GRANULARITY_MINUTE,
				MaxPoints:   50,
			},
		},
	}

	historyResp, err := service.GetItemHistory(ctx, historyReq)
	require.NoError(t, err)

	// Should get recent history within the time range
	assert.GreaterOrEqual(t, len(historyResp.History), 1, "Should have history within time range")

	// Verify all timestamps are within the requested range
	startTime := now.Add(-24 * time.Hour).Add(-time.Minute)
	endTime := now.Add(1 * time.Hour).Add(time.Minute)

	for _, snapshot := range historyResp.History {
		snapshotTime := snapshot.Timestamp.AsTime()
		assert.True(t, snapshotTime.After(startTime), "Snapshot should be after start time")
		assert.True(t, snapshotTime.Before(endTime), "Snapshot should be before end time")
	}

	t.Logf("✓ Time range query returned %d entries", len(historyResp.History))
}
