package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
)

const (
	itemPrefix    = "item:"
	unitPrefix    = "unit:"
	historyPrefix = "history:" // history:item_id:timestamp

	// Error message templates
	itemNotFoundMsg = "item not found: %s"
	unitNotFoundMsg = "unit not found: %s"
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
				return fmt.Errorf(itemNotFoundMsg, id)
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
				return fmt.Errorf(itemNotFoundMsg, item.ID)
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
				return fmt.Errorf(unitNotFoundMsg, id)
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

// UpdateUnit updates an existing unit definition
func (r *BadgerInventoryRepository) UpdateUnit(ctx context.Context, unit *domain.Unit) error {
	return r.db.Update(func(txn *badger.Txn) error {
		key := unitPrefix + unit.ID

		// Check if unit exists
		_, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf(unitNotFoundMsg, unit.ID)
			}
			return err
		}

		data, err := json.Marshal(unit)
		if err != nil {
			return fmt.Errorf("failed to marshal unit: %w", err)
		}

		return txn.Set([]byte(key), data)
	})
}

// DeleteUnit removes a unit definition
func (r *BadgerInventoryRepository) DeleteUnit(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badger.Txn) error {
		key := unitPrefix + id

		// Check if unit exists before deletion
		_, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf(unitNotFoundMsg, id)
			}
			return err
		}

		return txn.Delete([]byte(key))
	})
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
	now := time.Now()
	defaultUnits := []*domain.Unit{
		// Weight units
		{ID: "kg", Name: "Kilograms", Symbol: "kg", Category: "weight", BaseConversionFactor: 1.0, Description: "Base weight unit", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},
		{ID: "g", Name: "Grams", Symbol: "g", Category: "weight", BaseConversionFactor: 0.001, Description: "Metric weight unit", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},
		{ID: "lbs", Name: "Pounds", Symbol: "lbs", Category: "weight", BaseConversionFactor: 0.453592, Description: "Imperial weight unit", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},
		{ID: "oz", Name: "Ounces", Symbol: "oz", Category: "weight", BaseConversionFactor: 0.0283495, Description: "Imperial weight unit", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},

		// Volume units
		{ID: "l", Name: "Liters", Symbol: "L", Category: "volume", BaseConversionFactor: 1.0, Description: "Base volume unit", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},
		{ID: "ml", Name: "Milliliters", Symbol: "mL", Category: "volume", BaseConversionFactor: 0.001, Description: "Metric volume unit", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},
		{ID: "cups", Name: "Cups", Symbol: "cups", Category: "volume", BaseConversionFactor: 0.236588, Description: "Imperial volume unit", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},
		{ID: "gal", Name: "Gallons", Symbol: "gal", Category: "volume", BaseConversionFactor: 3.78541, Description: "Imperial volume unit", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},

		// Count units
		{ID: "pcs", Name: "Pieces", Symbol: "pcs", Category: "count", BaseConversionFactor: 1.0, Description: "Individual pieces", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},
		{ID: "items", Name: "Items", Symbol: "items", Category: "count", BaseConversionFactor: 1.0, Description: "Individual items", CreatedAt: now, UpdatedAt: now, Metadata: make(map[string]string)},
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

// ====================================
// HISTORY OPERATIONS
// ====================================

// AddInventorySnapshot stores a historical inventory level snapshot
func (r *BadgerInventoryRepository) AddInventorySnapshot(ctx context.Context, itemID string, snapshot *domain.InventoryLevelSnapshot) error {
	// Create timestamp-based key for efficient range queries
	// Key format: history:item_id:timestamp_rfc3339nano
	key := fmt.Sprintf("%s%s:%s", historyPrefix, itemID, snapshot.Timestamp.Format(time.RFC3339Nano))

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	return r.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// GetInventoryHistory retrieves historical inventory snapshots with filtering
func (r *BadgerInventoryRepository) GetInventoryHistory(ctx context.Context, itemID string, filters HistoryFilters) ([]*domain.InventoryLevelSnapshot, int, error) {
	var snapshots []*domain.InventoryLevelSnapshot
	
	// Construct key prefix for this item's history
	prefix := fmt.Sprintf("%s%s:", historyPrefix, itemID)
	
	// Convert time filters to RFC3339Nano format for key comparison
	startKey := prefix
	endKey := prefix + "z" // Default to end of range
	
	if !filters.StartTime.IsZero() {
		startKey = prefix + filters.StartTime.Format(time.RFC3339Nano)
	}
	if !filters.EndTime.IsZero() {
		endKey = prefix + filters.EndTime.Format(time.RFC3339Nano)
	}

	var totalCount int

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		
		it := txn.NewIterator(opts)
		defer it.Close()

		collectedCount := 0
		
		for it.Seek([]byte(startKey)); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			
			// Stop if we've passed the end key
			if key > endKey {
				break
			}

			// Skip if not part of this item's history
			if !strings.HasPrefix(key, prefix) {
				continue
			}

			totalCount++ // Count all matching records

			// Apply offset - skip records until we reach the desired offset
			if filters.Offset > 0 && collectedCount < filters.Offset {
				collectedCount++
				continue
			}
			
			// Apply limit - stop collecting if we've reached the limit
			if filters.Limit > 0 && len(snapshots) >= filters.Limit {
				continue // Keep counting but don't collect more snapshots
			}

			var snapshot domain.InventoryLevelSnapshot
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &snapshot)
			})
			if err != nil {
				return fmt.Errorf("failed to unmarshal snapshot: %w", err)
			}

			snapshots = append(snapshots, &snapshot)
			collectedCount++
		}
		
		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	// Apply granularity filtering if needed
	if filters.Granularity != "" && filters.Granularity != "all" {
		snapshots = r.applyGranularityFilter(snapshots, filters.Granularity)
	}

	return snapshots, totalCount, nil
}

// GetEarliestSnapshot retrieves the earliest snapshot for an item
func (r *BadgerInventoryRepository) GetEarliestSnapshot(ctx context.Context, itemID string) (*domain.InventoryLevelSnapshot, error) {
	prefix := fmt.Sprintf("%s%s:", historyPrefix, itemID)

	var earliest *domain.InventoryLevelSnapshot

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 1
		
		it := txn.NewIterator(opts)
		defer it.Close()

		it.Seek([]byte(prefix))
		if !it.Valid() {
			return nil // No history found
		}

		item := it.Item()
		key := string(item.Key())
		
		if !strings.HasPrefix(key, prefix) {
			return nil // No history found
		}

		var snapshot domain.InventoryLevelSnapshot
		err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &snapshot)
		})
		if err != nil {
			return fmt.Errorf("failed to unmarshal snapshot: %w", err)
		}

		earliest = &snapshot
		return nil
	})

	return earliest, err
}

// GetLatestSnapshot retrieves the latest snapshot for an item
func (r *BadgerInventoryRepository) GetLatestSnapshot(ctx context.Context, itemID string) (*domain.InventoryLevelSnapshot, error) {
	prefix := fmt.Sprintf("%s%s:", historyPrefix, itemID)

	var latest *domain.InventoryLevelSnapshot

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 1
		opts.Reverse = true // Start from the end
		
		it := txn.NewIterator(opts)
		defer it.Close()

		// Start from the highest possible key for this prefix
		maxKey := prefix + "z"
		it.Seek([]byte(maxKey))
		
		// Find the last entry with our prefix
		for it.Valid() {
			item := it.Item()
			key := string(item.Key())
			
			if strings.HasPrefix(key, prefix) {
				var snapshot domain.InventoryLevelSnapshot
				err := item.Value(func(val []byte) error {
					return json.Unmarshal(val, &snapshot)
				})
				if err != nil {
					return fmt.Errorf("failed to unmarshal snapshot: %w", err)
				}

				latest = &snapshot
				return nil
			}
			
			it.Next()
		}

		return nil // No history found
	})

	return latest, err
}

// applyGranularityFilter filters snapshots based on requested granularity
func (r *BadgerInventoryRepository) applyGranularityFilter(snapshots []*domain.InventoryLevelSnapshot, granularity string) []*domain.InventoryLevelSnapshot {
	if len(snapshots) == 0 {
		return snapshots
	}

	var filtered []*domain.InventoryLevelSnapshot
	var lastTimestamp time.Time
	var interval time.Duration

	// Determine the time interval based on granularity
	switch granularity {
	case "minute":
		interval = time.Minute
	case "hour":
		interval = time.Hour
	case "day":
		interval = 24 * time.Hour
	case "week":
		interval = 7 * 24 * time.Hour
	case "month":
		interval = 30 * 24 * time.Hour // Approximate
	default:
		return snapshots // Return all if granularity not recognized
	}

	for _, snapshot := range snapshots {
		// Include first snapshot or if enough time has passed
		if lastTimestamp.IsZero() || snapshot.Timestamp.Sub(lastTimestamp) >= interval {
			filtered = append(filtered, snapshot)
			lastTimestamp = snapshot.Timestamp
		}
	}

	return filtered
}
