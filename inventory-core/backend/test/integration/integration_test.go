package integration_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/inventory-core/proto/proto"
	"github.com/DaDevFox/task-systems/shared/events"
	eventspb "github.com/DaDevFox/task-systems/shared/proto/events/v1"
)

const (
	testServiceName = "test-inventory"
)

func TestInventoryServiceIntegration(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

	eventBus := events.NewEventBus(testServiceName)

	svc := service.NewInventoryService(repo, eventBus, logger)

	ctx := context.Background()

	// Test adding an item
	addReq := &pb.AddInventoryItemRequest{
		Name:              "Test Item",
		Description:       "Integration test item",
		InitialLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitId:            "kg",
		Metadata:          map[string]string{"category": "test"},
	}

	addResp, err := svc.AddInventoryItem(ctx, addReq)
	if err != nil {
		t.Fatalf("Failed to add inventory item: %v", err)
	}

	if addResp.Item == nil {
		t.Fatal("Expected item in response, got nil")
	}

	itemId := addResp.Item.Id

	// Test getting the item
	getReq := &pb.GetInventoryItemRequest{
		ItemId: itemId,
	}

	getResp, err := svc.GetInventoryItem(ctx, getReq)
	if err != nil {
		t.Fatalf("Failed to get inventory item: %v", err)
	}

	if getResp.Item == nil {
		t.Fatal("Expected item in response, got nil")
	}

	item := getResp.Item
	if item.Name != "Test Item" {
		t.Errorf("Expected name 'Test Item', got %s", item.Name)
	}
	if item.CurrentLevel != 100.0 {
		t.Errorf("Expected current level 100.0, got %f", item.CurrentLevel)
	}

	// Test updating inventory level
	updateReq := &pb.UpdateInventoryLevelRequest{
		ItemId:   itemId,
		NewLevel: 70.0,
		Reason:   "Integration test consumption",
	}

	updateResp, err := svc.UpdateInventoryLevel(ctx, updateReq)
	if err != nil {
		t.Fatalf("Failed to update inventory level: %v", err)
	}

	if updateResp.Item.CurrentLevel != 70.0 {
		t.Errorf("Expected current level 70.0 after update, got %f", updateResp.Item.CurrentLevel)
	}

	// Test getting inventory status
	statusReq := &pb.GetInventoryStatusRequest{}

	statusResp, err := svc.GetInventoryStatus(ctx, statusReq)
	if err != nil {
		t.Fatalf("Failed to get inventory status: %v", err)
	}

	if len(statusResp.Status.Items) != 1 {
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
