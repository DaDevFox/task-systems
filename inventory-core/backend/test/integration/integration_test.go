package integration_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
	eventspb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
)

const (
	testServiceName = "test-inventory"
)

func TestInventoryServiceIntegration(t *testing.T) {
	svc, itemId := setupIntegrationTest(t)
	ctx := context.Background()

	t.Run("GetInventoryItem", func(t *testing.T) {
		testGetInventoryItem(t, svc, ctx, itemId)
	})

	t.Run("UpdateInventoryItemMetadata", func(t *testing.T) {
		testUpdateInventoryItemMetadata(t, svc, ctx, itemId)
	})

	t.Run("UpdateInventoryLevel", func(t *testing.T) {
		testUpdateInventoryLevel(t, svc, ctx, itemId)
	})

	t.Run("GetInventoryStatus", func(t *testing.T) {
		testGetInventoryStatus(t, svc, ctx)
	})
}

func setupIntegrationTest(t *testing.T) (*service.InventoryService, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	eventBus := events.NewEventBus(testServiceName)
	svc := service.NewInventoryService(repo, eventBus, logger)

	// Add initial test item
	addReq := &pb.AddInventoryItemRequest{
		Name:              "Test Item",
		Description:       "Integration test item",
		InitialLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitId:            "kg",
		Metadata:          map[string]string{"category": "test"},
	}

	addResp, err := svc.AddInventoryItem(context.Background(), addReq)
	if err != nil {
		t.Fatalf("Failed to add inventory item: %v", err)
	}

	if addResp.Item == nil {
		t.Fatal("Expected item in response, got nil")
	}

	return svc, addResp.Item.Id
}

func testGetInventoryItem(t *testing.T, svc *service.InventoryService, ctx context.Context, itemId string) {
	getReq := &pb.GetInventoryItemRequest{ItemId: itemId}
	getResp, err := svc.GetInventoryItem(ctx, getReq)

	switch {
	case err != nil:
		t.Fatalf("Failed to get inventory item: %v", err)
	case getResp.Item == nil:
		t.Fatal("Expected item in response, got nil")
	case getResp.Item.Name != "Test Item":
		t.Errorf("Expected name 'Test Item', got %s", getResp.Item.Name)
	case getResp.Item.CurrentLevel != 100.0:
		t.Errorf("Expected current level 100.0, got %f", getResp.Item.CurrentLevel)
	}
}

func testUpdateInventoryItemMetadata(t *testing.T, svc *service.InventoryService, ctx context.Context, itemId string) {
	updateReq := &pb.UpdateInventoryItemRequest{
		ItemId:            itemId,
		Name:              "Updated Test Item",
		Description:       "Updated description for integration test",
		MaxCapacity:       250.0,
		LowStockThreshold: 15.0,
		UnitId:            "lbs",
		Metadata:          map[string]string{"category": "updated", "location": "warehouse"},
	}

	updateResp, err := svc.UpdateInventoryItem(ctx, updateReq)
	if err != nil {
		t.Fatalf("Failed to update inventory item: %v", err)
	}

	item := updateResp.Item
	switch {
	case item.Name != "Updated Test Item":
		t.Errorf("Expected updated name 'Updated Test Item', got %s", item.Name)
	case item.Description != "Updated description for integration test":
		t.Errorf("Expected updated description, got %s", item.Description)
	case item.MaxCapacity != 250.0:
		t.Errorf("Expected updated max capacity 250.0, got %f", item.MaxCapacity)
	case item.LowStockThreshold != 15.0:
		t.Errorf("Expected updated low stock threshold 15.0, got %f", item.LowStockThreshold)
	case item.UnitId != "lbs":
		t.Errorf("Expected updated unit ID 'lbs', got %s", item.UnitId)
	case item.Metadata["category"] != "updated":
		t.Errorf("Expected updated metadata category 'updated', got %s", item.Metadata["category"])
	case item.Metadata["location"] != "warehouse":
		t.Errorf("Expected updated metadata location 'warehouse', got %s", item.Metadata["location"])
	case item.CurrentLevel != 100.0:
		t.Errorf("Expected current level to remain 100.0 after metadata update, got %f", item.CurrentLevel)
	}
}

func testUpdateInventoryLevel(t *testing.T, svc *service.InventoryService, ctx context.Context, itemId string) {
	updateReq := &pb.UpdateInventoryLevelRequest{
		ItemId:   itemId,
		NewLevel: 70.0,
		Reason:   "Integration test consumption",
	}

	updateResp, err := svc.UpdateInventoryLevel(ctx, updateReq)
	switch {
	case err != nil:
		t.Fatalf("Failed to update inventory level: %v", err)
	case updateResp.Item.CurrentLevel != 70.0:
		t.Errorf("Expected current level 70.0 after update, got %f", updateResp.Item.CurrentLevel)
	}
}

func testGetInventoryStatus(t *testing.T, svc *service.InventoryService, ctx context.Context) {
	statusReq := &pb.GetInventoryStatusRequest{}
	statusResp, err := svc.GetInventoryStatus(ctx, statusReq)

	switch {
	case err != nil:
		t.Fatalf("Failed to get inventory status: %v", err)
	case len(statusResp.Status.Items) != 1:
		t.Errorf("Expected 1 item in status, got %d", len(statusResp.Status.Items))
	}
}

func TestInventoryServiceEventPublishing(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	eventBus := events.NewEventBus(testServiceName)

	svc := service.NewInventoryService(repo, eventBus, logger)

	ctx := context.Background()

	// Subscribe to events using channel-based handler
	eventsChan := make(chan *eventspb.Event, 10)
	eventHandler := func(ctx context.Context, event *eventspb.Event) error {
		eventsChan <- event
		return nil
	}

	eventBus.Subscribe(eventspb.EventType_INVENTORY_LEVEL_CHANGED, eventHandler)

	// Add an item that will trigger event
	addReq := &pb.AddInventoryItemRequest{
		Name:              "Event Test Item",
		Description:       "Item for testing events",
		InitialLevel:      50.0,
		MaxCapacity:       100.0,
		LowStockThreshold: 30.0,
		UnitId:            "kg",
	}

	addResp, err := svc.AddInventoryItem(ctx, addReq)
	if err != nil {
		t.Fatalf("Failed to add inventory item: %v", err)
	}

	// Update inventory level to trigger event
	updateReq := &pb.UpdateInventoryLevelRequest{
		ItemId:   addResp.Item.Id,
		NewLevel: 25.0,
		Reason:   "Testing event publishing",
	}

	_, err = svc.UpdateInventoryLevel(ctx, updateReq)
	if err != nil {
		t.Fatalf("Failed to update inventory level: %v", err)
	}

	// Check if event was published
	select {
	case event := <-eventsChan:
		if event.Type != eventspb.EventType_INVENTORY_LEVEL_CHANGED {
			t.Errorf("Expected INVENTORY_LEVEL_CHANGED event, got %s", event.Type.String())
		}
		if event.SourceService != testServiceName {
			t.Errorf("Expected source service '%s', got %s", testServiceName, event.SourceService)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive inventory level changed event")
	}
}
