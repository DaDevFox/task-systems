package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

func TestRemoveInventoryItemSuccess(t *testing.T) {
	// Setup temporary database
	tmpDir, err := os.MkdirTemp("", "test_inventory_db")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	eventBus := events.NewEventBus("test-service")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during testing
	service := NewInventoryService(repo, eventBus, logger)

	// Create a test item first
	testItem := &domain.InventoryItem{
		Name:              "Test Item",
		Description:       "A test item",
		CurrentLevel:      10.0,
		MaxCapacity:       100.0,
		LowStockThreshold: 5.0,
		UnitID:            "kg", // Use default unit
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = repo.AddItem(context.Background(), testItem)
	if err != nil {
		t.Fatalf("Failed to add test item: %v", err)
	}

	// Execute
	req := &pb.RemoveInventoryItemRequest{
		ItemId: testItem.ID,
	}

	resp, err := service.RemoveInventoryItem(context.Background(), req)

	// Verify
	switch {
	case err != nil:
		t.Errorf("Expected no error, got %v", err)
	case resp == nil:
		t.Fatal("Expected response, got nil")
	case !resp.ItemRemoved:
		t.Errorf("Expected item_removed to be true, got %v", resp.ItemRemoved)
	case resp.RemovedItemId != testItem.ID:
		t.Errorf("Expected removed_item_id to be %s, got %s", testItem.ID, resp.RemovedItemId)
	case resp.RemovedItemName != testItem.Name:
		t.Errorf("Expected removed_item_name to be %s, got %s", testItem.Name, resp.RemovedItemName)
	}

	// Verify item is actually deleted
	_, err = repo.GetItem(context.Background(), testItem.ID)
	if err == nil {
		t.Error("Expected item to be deleted, but it still exists")
	}
}

func TestRemoveInventoryItemMissingItemId(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "test_inventory_db")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	eventBus := events.NewEventBus("test-service")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	service := NewInventoryService(repo, eventBus, logger)

	// Execute
	req := &pb.RemoveInventoryItemRequest{
		ItemId: "", // Missing item_id
	}

	resp, err := service.RemoveInventoryItem(context.Background(), req)

	// Verify
	switch {
	case err == nil:
		t.Fatal("Expected error, got nil")
	case resp != nil:
		t.Errorf("Expected nil response, got %v", resp)
	case status.Code(err) != codes.InvalidArgument:
		t.Errorf("Expected InvalidArgument error code, got %v", status.Code(err))
	}
}

func TestRemoveInventoryItemNotFound(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "test_inventory_db")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewBadgerInventoryRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	eventBus := events.NewEventBus("test-service")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	service := NewInventoryService(repo, eventBus, logger)

	// Execute
	req := &pb.RemoveInventoryItemRequest{
		ItemId: "nonexistent-item",
	}

	resp, err := service.RemoveInventoryItem(context.Background(), req)

	// Verify
	switch {
	case err == nil:
		t.Fatal("Expected error, got nil")
	case resp != nil:
		t.Errorf("Expected nil response, got %v", resp)
	case status.Code(err) != codes.NotFound:
		t.Errorf("Expected NotFound error code, got %v", status.Code(err))
	}
}
