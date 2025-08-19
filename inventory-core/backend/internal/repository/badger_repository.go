package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
)

const (
	itemPrefix = "item:"
	unitPrefix = "unit:"
)

// BadgerInventoryRepository implements InventoryRepository using BadgerDB
type BadgerInventoryRepository struct {
	db *badger.DB
}

// NewBadgerInventoryRepository creates a new BadgerDB-backed repository
func NewBadgerInventoryRepository(dbPath string) (*BadgerInventoryRepository, error) {
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil // Disable badger logging

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	repo := &BadgerInventoryRepository{db: db}

	// Initialize with default units
	if err := repo.initializeDefaultUnits(); err != nil {
		return nil, fmt.Errorf("failed to initialize default units: %w", err)
	}

	return repo, nil
}

// Close closes the database connection
func (r *BadgerInventoryRepository) Close() error {
	return r.db.Close()
}

// AddItem adds a new inventory item
func (r *BadgerInventoryRepository) AddItem(ctx context.Context, item *domain.InventoryItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}

	return r.db.Update(func(txn *badger.Txn) error {
		key := itemPrefix + item.ID
		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}

		return txn.Set([]byte(key), data)
	})
}

// GetItem retrieves an inventory item by ID
func (r *BadgerInventoryRepository) GetItem(ctx context.Context, id string) (*domain.InventoryItem, error) {
	var item *domain.InventoryItem

	err := r.db.View(func(txn *badger.Txn) error {
		key := itemPrefix + id
		dbItem, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("item not found: %s", id)
			}
			return err
		}

		return dbItem.Value(func(val []byte) error {
			item = &domain.InventoryItem{}
			return json.Unmarshal(val, item)
		})
	})

	return item, err
}

// UpdateItem updates an existing inventory item
func (r *BadgerInventoryRepository) UpdateItem(ctx context.Context, item *domain.InventoryItem) error {
	return r.db.Update(func(txn *badger.Txn) error {
		key := itemPrefix + item.ID

		// Check if item exists
		_, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("item not found: %s", item.ID)
			}
			return err
		}

		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}

		return txn.Set([]byte(key), data)
	})
}

// DeleteItem removes an inventory item
func (r *BadgerInventoryRepository) DeleteItem(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		key := itemPrefix + id
		return txn.Delete([]byte(key))
	})
}

// ListItems retrieves filtered list of inventory items
func (r *BadgerInventoryRepository) ListItems(ctx context.Context, filters ListFilters) ([]*domain.InventoryItem, int, error) {
	var items []*domain.InventoryItem

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(itemPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				var invItem domain.InventoryItem
				if err := json.Unmarshal(val, &invItem); err != nil {
					return err
				}

				// Apply filters
				if filters.LowStockOnly && !invItem.IsLowStock() {
					return nil
				}

				items = append(items, &invItem)
				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	total := len(items)

	// Apply pagination
	if filters.Offset > 0 {
		if filters.Offset >= len(items) {
			items = []*domain.InventoryItem{}
		} else {
			items = items[filters.Offset:]
		}
	}

	if filters.Limit > 0 && filters.Limit < len(items) {
		items = items[:filters.Limit]
	}

	return items, total, err
}

// GetAllItems retrieves all inventory items
func (r *BadgerInventoryRepository) GetAllItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	items, _, err := r.ListItems(ctx, ListFilters{})
	return items, err
}

// GetLowStockItems retrieves items below their threshold
func (r *BadgerInventoryRepository) GetLowStockItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	items, _, err := r.ListItems(ctx, ListFilters{LowStockOnly: true})
	return items, err
}

// GetEmptyItems retrieves items with zero stock
func (r *BadgerInventoryRepository) GetEmptyItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	allItems, err := r.GetAllItems(ctx)
	if err != nil {
		return nil, err
	}

	var emptyItems []*domain.InventoryItem
	for _, item := range allItems {
		if item.IsEmpty() {
			emptyItems = append(emptyItems, item)
		}
	}

	return emptyItems, nil
}

// Unit operations

// AddUnit adds a new unit definition
func (r *BadgerInventoryRepository) AddUnit(ctx context.Context, unit *domain.Unit) error {
	return r.db.Update(func(txn *badger.Txn) error {
		key := unitPrefix + unit.ID
		data, err := json.Marshal(unit)
		if err != nil {
			return fmt.Errorf("failed to marshal unit: %w", err)
		}

		return txn.Set([]byte(key), data)
	})
}

// GetUnit retrieves a unit by ID
func (r *BadgerInventoryRepository) GetUnit(ctx context.Context, id string) (*domain.Unit, error) {
	var unit *domain.Unit

	err := r.db.View(func(txn *badger.Txn) error {
		key := unitPrefix + id
		dbItem, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("unit not found: %s", id)
			}
			return err
		}

		return dbItem.Value(func(val []byte) error {
			unit = &domain.Unit{}
			return json.Unmarshal(val, unit)
		})
	})

	return unit, err
}

// ListUnits retrieves all unit definitions
func (r *BadgerInventoryRepository) ListUnits(ctx context.Context) ([]*domain.Unit, error) {
	var units []*domain.Unit

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(unitPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				var unit domain.Unit
				if err := json.Unmarshal(val, &unit); err != nil {
					return err
				}

				units = append(units, &unit)
				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return units, err
}

// initializeDefaultUnits adds common measurement units
func (r *BadgerInventoryRepository) initializeDefaultUnits() error {
	defaultUnits := []*domain.Unit{
		// Weight units
		{ID: "kg", Name: "Kilograms", Symbol: "kg", Type: domain.UnitTypeWeight, BaseConversionFactor: 1.0, BaseUnitID: "kg"},
		{ID: "g", Name: "Grams", Symbol: "g", Type: domain.UnitTypeWeight, BaseConversionFactor: 0.001, BaseUnitID: "kg"},
		{ID: "lbs", Name: "Pounds", Symbol: "lbs", Type: domain.UnitTypeWeight, BaseConversionFactor: 0.453592, BaseUnitID: "kg"},
		{ID: "oz", Name: "Ounces", Symbol: "oz", Type: domain.UnitTypeWeight, BaseConversionFactor: 0.0283495, BaseUnitID: "kg"},

		// Volume units
		{ID: "l", Name: "Liters", Symbol: "L", Type: domain.UnitTypeVolume, BaseConversionFactor: 1.0, BaseUnitID: "l"},
		{ID: "ml", Name: "Milliliters", Symbol: "mL", Type: domain.UnitTypeVolume, BaseConversionFactor: 0.001, BaseUnitID: "l"},
		{ID: "cups", Name: "Cups", Symbol: "cups", Type: domain.UnitTypeVolume, BaseConversionFactor: 0.236588, BaseUnitID: "l"},
		{ID: "gal", Name: "Gallons", Symbol: "gal", Type: domain.UnitTypeVolume, BaseConversionFactor: 3.78541, BaseUnitID: "l"},

		// Count units
		{ID: "pcs", Name: "Pieces", Symbol: "pcs", Type: domain.UnitTypeCount, BaseConversionFactor: 1.0, BaseUnitID: "pcs"},
		{ID: "items", Name: "Items", Symbol: "items", Type: domain.UnitTypeCount, BaseConversionFactor: 1.0, BaseUnitID: "pcs"},
	}

	for _, unit := range defaultUnits {
		// Check if unit already exists
		existing, err := r.GetUnit(context.Background(), unit.ID)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}

		if existing == nil {
			if err := r.AddUnit(context.Background(), unit); err != nil {
				return err
			}
		}
	}

	return nil
}
