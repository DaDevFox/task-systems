package integration_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

func TestInventoryServiceGRPCIntegration(t *testing.T) {
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

	eventBus := events.GetGlobalBus("test-inventory")

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

	if addResp.Item.Name != "Test Item" {
		t.Errorf("Expected item name 'Test Item', got %s", addResp.Item.Name)
	}

	itemID := addResp.Item.Id

	// Test getting the item
	getReq := &pb.GetInventoryItemRequest{ItemId: itemID}
	getResp, err := svc.GetInventoryItem(ctx, getReq)
	if err != nil {
		t.Fatalf("Failed to get inventory item: %v", err)
	}

	if getResp.Item.CurrentLevel != 100.0 {
		t.Errorf("Expected current level 100.0, got %v", getResp.Item.CurrentLevel)
	}

	// Test updating inventory level
	updateReq := &pb.UpdateInventoryLevelRequest{
		ItemId:            itemID,
		NewLevel:          50.0,
		Reason:            "Test consumption",
		RecordConsumption: true,
	}

	updateResp, err := svc.UpdateInventoryLevel(ctx, updateReq)
	if err != nil {
		t.Fatalf("Failed to update inventory level: %v", err)
	}

	if updateResp.Item.CurrentLevel != 50.0 {
		t.Errorf("Expected updated level 50.0, got %v", updateResp.Item.CurrentLevel)
	}

	if !updateResp.LevelChanged {
		t.Error("Expected level changed to be true")
	}

	// Test getting inventory status
	statusReq := &pb.GetInventoryStatusRequest{}
	statusResp, err := svc.GetInventoryStatus(ctx, statusReq)
	if err != nil {
		t.Fatalf("Failed to get inventory status: %v", err)
	}

	if len(statusResp.Status.Items) == 0 {
		t.Error("Expected at least one item in status")
	}

	// Verify level was updated correctly
	if updateResp.Item.CurrentLevel != 50.0 {
		t.Errorf("Expected updated level 50.0, got %v", updateResp.Item.CurrentLevel)
	}
}
