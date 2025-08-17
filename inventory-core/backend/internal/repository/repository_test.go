package repository

import (
	"context"
	"iter"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
)

const (
	testItemName     = "Test Item"
	errFailedAddItem = "Failed to add item: %v"
)

func createBadgerRepository(t *testing.T) (*BadgerInventoryRepository, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := NewBadgerInventoryRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create Badger repository: %v", err)
	}

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tempDir)
	}

	return repo, cleanup
}

func createTestRepositories(t *testing.T) iter.Seq2[InventoryRepository, func()] {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	repo, err := NewBadgerInventoryRepository(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tempDir)
	}

	var sequence iter.Seq2[InventoryRepository, func()]
	sequence = func(yield func(InventoryRepository, func()) bool) {
		if !yield(repo, cleanup) {
			return
		}
	}
	return sequence
}

func TestInventoryRepositoryAddItem(t *testing.T) {
	for repo, cleanup := range createTestRepositories(t) {
		defer cleanup()

		ctx := context.Background()
		item := &domain.InventoryItem{
			Name:              "Test Item",
			Description:       "A test inventory item",
			CurrentLevel:      50.0,
			MaxCapacity:       100.0,
			LowStockThreshold: 10.0,
			UnitID:            "kg",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
			Metadata:          map[string]string{"category": "test"},
		}

		err := repo.AddItem(ctx, item)
		if err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}

		if item.ID == "" {
			t.Error("Expected item ID to be generated")
		}
	}
}

// Relies on correct Add implementation
func TestInventoryRepositoryGetItem(t *testing.T) {
	for repo, cleanup := range createTestRepositories(t) {
		defer cleanup()

		ctx := context.Background()
		originalItem := &domain.InventoryItem{
			Name:              "Test Item",
			Description:       "A test inventory item",
			CurrentLevel:      50.0,
			MaxCapacity:       100.0,
			LowStockThreshold: 10.0,
			UnitID:            "kg",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
			Metadata:          map[string]string{"category": "test"},
		}

		err := repo.AddItem(ctx, originalItem)
		if err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}

		retrievedItem, err := repo.GetItem(ctx, originalItem.ID)
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}

		if retrievedItem.Name != originalItem.Name {
			t.Errorf("Expected name %s, got %s", originalItem.Name, retrievedItem.Name)
		}

		if retrievedItem.CurrentLevel != originalItem.CurrentLevel {
			t.Errorf("Expected level %v, got %v", originalItem.CurrentLevel, retrievedItem.CurrentLevel)
		}
	}
}

func TestInventoryRepositoryUpdateItem(t *testing.T) {
	for repo, cleanup := range createTestRepositories(t) {
		defer cleanup()

		ctx := context.Background()
		item := &domain.InventoryItem{
			Name:              "Test Item",
			CurrentLevel:      50.0,
			MaxCapacity:       100.0,
			LowStockThreshold: 10.0,
			UnitID:            "kg",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.AddItem(ctx, item)
		if err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}

		// Update the item
		item.CurrentLevel = 75.0
		item.Description = "Updated description"

		err = repo.UpdateItem(ctx, item)
		if err != nil {
			t.Fatalf("Failed to update item: %v", err)
		}

		// Verify update
		retrievedItem, err := repo.GetItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("Failed to get updated item: %v", err)
		}

		if retrievedItem.CurrentLevel != 75.0 {
			t.Errorf("Expected updated level 75.0, got %v", retrievedItem.CurrentLevel)
		}

		if retrievedItem.Description != "Updated description" {
			t.Errorf("Expected updated description, got %s", retrievedItem.Description)
		}
	}
}

func TestInventoryRepositoryGetAllItems(t *testing.T) {
	for repo, cleanup := range createTestRepositories(t) {
		defer cleanup()

		ctx := context.Background()

		// Add multiple items
		items := []*domain.InventoryItem{
			{
				Name:         "Item 1",
				CurrentLevel: 50.0,
				UnitID:       "kg",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			{
				Name:         "Item 2",
				CurrentLevel: 25.0,
				UnitID:       "liters",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
		}

		for _, item := range items {
			err := repo.AddItem(ctx, item)
			if err != nil {
				t.Fatalf("Failed to add item: %v", err)
			}
		}

		allItems, err := repo.GetAllItems(ctx)
		if err != nil {
			t.Fatalf("Failed to get all items: %v", err)
		}

		if len(allItems) < 2 {
			t.Errorf("Expected at least 2 items, got %d", len(allItems))
		}
	}
}

func TestInventoryRepositoryGetLowStockItems(t *testing.T) {
	for repo, cleanup := range createTestRepositories(t) {
		defer cleanup()

		ctx := context.Background()

		// Add items with different stock levels
		normalStockItem := &domain.InventoryItem{
			Name:              "Normal Stock",
			CurrentLevel:      50.0,
			LowStockThreshold: 10.0,
			UnitID:            "kg",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		lowStockItem := &domain.InventoryItem{
			Name:              "Low Stock",
			CurrentLevel:      5.0,
			LowStockThreshold: 10.0,
			UnitID:            "kg",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.AddItem(ctx, normalStockItem)
		if err != nil {
			t.Fatalf("Failed to add normal stock item: %v", err)
		}

		err = repo.AddItem(ctx, lowStockItem)
		if err != nil {
			t.Fatalf("Failed to add low stock item: %v", err)
		}

		lowStockItems, err := repo.GetLowStockItems(ctx)
		if err != nil {
			t.Fatalf("Failed to get low stock items: %v", err)
		}

		found := false
		for _, item := range lowStockItems {
			if item.ID == lowStockItem.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find low stock item in results")
		}
	}
}
