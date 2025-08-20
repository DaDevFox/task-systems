package repository

import (
	"context"
	"time"

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
	UpdateUnit(ctx context.Context, unit *domain.Unit) error
	DeleteUnit(ctx context.Context, id string) error
	ListUnits(ctx context.Context) ([]*domain.Unit, error)

	// History operations
	AddInventorySnapshot(ctx context.Context, itemID string, snapshot *domain.InventoryLevelSnapshot) error
	GetInventoryHistory(ctx context.Context, itemID string, filters HistoryFilters) ([]*domain.InventoryLevelSnapshot, int, error)
	GetEarliestSnapshot(ctx context.Context, itemID string) (*domain.InventoryLevelSnapshot, error)
	GetLatestSnapshot(ctx context.Context, itemID string) (*domain.InventoryLevelSnapshot, error)
}

// ListFilters provides filtering options for listing items
type ListFilters struct {
	LowStockOnly   bool
	UnitTypeFilter string
	Limit          int
	Offset         int
}

// HistoryFilters provides filtering options for inventory history queries
type HistoryFilters struct {
	StartTime    time.Time
	EndTime      time.Time
	Granularity  string // "minute", "hour", "day", "week", "month"
	Limit        int
	Offset       int
}
