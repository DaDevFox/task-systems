package repository

import (
	"context"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
)

// InventoryRepository defines the interface for inventory data persistence
type InventoryRepository interface {
	// Item operations
	AddItem(ctx context.Context, item *domain.InventoryItem) error
	GetItem(ctx context.Context, id string) (*domain.InventoryItem, error)
	UpdateItem(ctx context.Context, item *domain.InventoryItem) error
	DeleteItem(ctx context.Context, id string) error
	ListItems(ctx context.Context, filters ListFilters) ([]*domain.InventoryItem, int, error)

	// Bulk operations
	GetAllItems(ctx context.Context) ([]*domain.InventoryItem, error)
	GetLowStockItems(ctx context.Context) ([]*domain.InventoryItem, error)
	GetEmptyItems(ctx context.Context) ([]*domain.InventoryItem, error)

	// Unit operations
	AddUnit(ctx context.Context, unit *domain.Unit) error
	GetUnit(ctx context.Context, id string) (*domain.Unit, error)
	ListUnits(ctx context.Context) ([]*domain.Unit, error)
}

// ListFilters provides filtering options for listing items
type ListFilters struct {
	LowStockOnly   bool
	UnitTypeFilter string
	Limit          int
	Offset         int
}
