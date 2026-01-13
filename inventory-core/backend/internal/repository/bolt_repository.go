package repository

import (
	   "context"
	   "encoding/json"
	   "fmt"
	   "os"
	   "path/filepath"
	   "sort"
	   "strings"
	   "time"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
)

// BoltInventoryRepository implements InventoryRepository using BoltDB (bbolt)
// BoltDB is much more compact than BadgerDB and doesn't create large .vlog files
type BoltInventoryRepository struct {
	db *bbolt.DB
}

// NewBoltInventoryRepository creates a new BoltDB-backed repository
func NewBoltInventoryRepository(dbPath string) (*BoltInventoryRepository, error) {
	   // Diagnostics for debugging file creation issues
	   fmt.Printf("[BoltDB DIAG] dbPath: %s\n", dbPath)
	   dir := filepath.Dir(dbPath)
	   dirInfo, dirErr := os.Stat(dir)
	   fmt.Printf("[BoltDB DIAG] Parent dir: %s, Exists: %v, IsDir: %v, Err: %v\n", dir, dirErr == nil, dirErr == nil && dirInfo.IsDir(), dirErr)
	   _, fileErr := os.Stat(dbPath)
	   fmt.Printf("[BoltDB DIAG] DB file exists before open: %v, Err: %v\n", fileErr == nil, fileErr)
	   // Try to create a test file in the directory
	   testFilePath := filepath.Join(dir, "__bolt_test_write.tmp")
	   f, testFileErr := os.Create(testFilePath)
	   if testFileErr == nil {
			   f.Close()
			   os.Remove(testFilePath)
			   fmt.Printf("[BoltDB DIAG] Successfully created and removed a test file in parent dir.\n")
	   } else {
			   fmt.Printf("[BoltDB DIAG] FAILED to create a test file in parent dir: %v\n", testFileErr)
	   }
	   // Ensure parent directory exists (important for Windows)
	   if err := os.MkdirAll(dir, 0755); err != nil {
			   return nil, fmt.Errorf("failed to create parent directory for bolt db: %w", err)
	   }

	   // Open BoltDB with optimized settings for smaller size
	   db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{
			   Timeout:      1 * time.Second,
			   NoGrowSync:   false,                 // Ensure data durability
			   FreelistType: bbolt.FreelistMapType, // More efficient freelist
	   })
	   if err != nil {
			   return nil, fmt.Errorf("failed to open bolt db: %w", err)
	   }

	repo := &BoltInventoryRepository{db: db}

	// Create buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{"items", "units", "history"}
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	// Initialize with default units
	if err := repo.initializeDefaultUnits(); err != nil {
		return nil, fmt.Errorf("failed to initialize default units: %w", err)
	}

	return repo, nil
}

// Close closes the database connection
func (r *BoltInventoryRepository) Close() error {
	return r.db.Close()
}

// AddItem adds a new inventory item
func (r *BoltInventoryRepository) AddItem(ctx context.Context, item *domain.InventoryItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("items"))
		if bucket == nil {
			return fmt.Errorf("items bucket not found")
		}

		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}

		return bucket.Put([]byte(item.ID), data)
	})
}

// GetItem retrieves an inventory item by ID
func (r *BoltInventoryRepository) GetItem(ctx context.Context, id string) (*domain.InventoryItem, error) {
	var item *domain.InventoryItem

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("items"))
		if bucket == nil {
			return fmt.Errorf("items bucket not found")
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return &domain.InventoryItemNotFoundError{ID: id}
		}

		var foundItem domain.InventoryItem
		if err := json.Unmarshal(data, &foundItem); err != nil {
			return fmt.Errorf("failed to unmarshal item: %w", err)
		}

		item = &foundItem
		return nil
	})

	return item, err
}

// UpdateItem updates an existing inventory item
func (r *BoltInventoryRepository) UpdateItem(ctx context.Context, item *domain.InventoryItem) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("items"))
		if bucket == nil {
			return fmt.Errorf("items bucket not found")
		}

		// Check if item exists
		if bucket.Get([]byte(item.ID)) == nil {
			return &domain.InventoryItemNotFoundError{ID: item.ID}
		}

		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal item: %w", err)
		}

		return bucket.Put([]byte(item.ID), data)
	})
}

// DeleteItem removes an inventory item
func (r *BoltInventoryRepository) DeleteItem(ctx context.Context, id string) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("items"))
		if bucket == nil {
			return fmt.Errorf("items bucket not found")
		}

		if bucket.Get([]byte(id)) == nil {
			return &domain.InventoryItemNotFoundError{ID: id}
		}

		return bucket.Delete([]byte(id))
	})
}

// ListItems returns a list of inventory items with filtering
func (r *BoltInventoryRepository) ListItems(ctx context.Context, filters ListFilters) ([]*domain.InventoryItem, int, error) {
	var items []*domain.InventoryItem
	var totalCount int

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("items"))
		if bucket == nil {
			return fmt.Errorf("items bucket not found")
		}

		cursor := bucket.Cursor()
		collected := 0

		for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
			var item domain.InventoryItem
			if err := json.Unmarshal(value, &item); err != nil {
				continue // Skip malformed items
			}

			// Apply filters
			if filters.LowStockOnly && !item.IsLowStock() {
				continue
			}

			totalCount++

			// Apply offset
			if filters.Offset > 0 && collected < filters.Offset {
				collected++
				continue
			}

			// Apply limit
			if filters.Limit > 0 && len(items) >= filters.Limit {
				continue
			}

			items = append(items, &item)
			collected++
		}

		return nil
	})

	return items, totalCount, err
}

// GetAllItems returns all inventory items
func (r *BoltInventoryRepository) GetAllItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	items, _, err := r.ListItems(ctx, ListFilters{})
	return items, err
}

// GetLowStockItems returns items that are below their low stock threshold
func (r *BoltInventoryRepository) GetLowStockItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	items, _, err := r.ListItems(ctx, ListFilters{LowStockOnly: true})
	return items, err
}

// GetEmptyItems returns items that have zero current level
func (r *BoltInventoryRepository) GetEmptyItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	var emptyItems []*domain.InventoryItem

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("items"))
		if bucket == nil {
			return fmt.Errorf("items bucket not found")
		}

		cursor := bucket.Cursor()
		for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
			var item domain.InventoryItem
			if err := json.Unmarshal(value, &item); err != nil {
				continue
			}

			if item.IsEmpty() {
				emptyItems = append(emptyItems, &item)
			}
		}
		return nil
	})

	return emptyItems, err
}

// AddUnit adds a new unit
func (r *BoltInventoryRepository) AddUnit(ctx context.Context, unit *domain.Unit) error {
	if unit.ID == "" {
		unit.ID = uuid.New().String()
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("units"))
		if bucket == nil {
			return fmt.Errorf("units bucket not found")
		}

		data, err := json.Marshal(unit)
		if err != nil {
			return fmt.Errorf("failed to marshal unit: %w", err)
		}

		return bucket.Put([]byte(unit.ID), data)
	})
}

// GetUnit retrieves a unit by ID
func (r *BoltInventoryRepository) GetUnit(ctx context.Context, id string) (*domain.Unit, error) {
	var unit *domain.Unit

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("units"))
		if bucket == nil {
			return fmt.Errorf("units bucket not found")
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return &domain.UnitNotFoundError{ID: id}
		}

		var foundUnit domain.Unit
		if err := json.Unmarshal(data, &foundUnit); err != nil {
			return fmt.Errorf("failed to unmarshal unit: %w", err)
		}

		unit = &foundUnit
		return nil
	})

	return unit, err
}

// UpdateUnit updates an existing unit
func (r *BoltInventoryRepository) UpdateUnit(ctx context.Context, unit *domain.Unit) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("units"))
		if bucket == nil {
			return fmt.Errorf("units bucket not found")
		}

		if bucket.Get([]byte(unit.ID)) == nil {
			return &domain.UnitNotFoundError{ID: unit.ID}
		}

		data, err := json.Marshal(unit)
		if err != nil {
			return fmt.Errorf("failed to marshal unit: %w", err)
		}

		return bucket.Put([]byte(unit.ID), data)
	})
}

// DeleteUnit removes a unit
func (r *BoltInventoryRepository) DeleteUnit(ctx context.Context, id string) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("units"))
		if bucket == nil {
			return fmt.Errorf("units bucket not found")
		}

		if bucket.Get([]byte(id)) == nil {
			return &domain.UnitNotFoundError{ID: id}
		}

		return bucket.Delete([]byte(id))
	})
}

// ListUnits returns all available units
func (r *BoltInventoryRepository) ListUnits(ctx context.Context) ([]*domain.Unit, error) {
	var units []*domain.Unit

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("units"))
		if bucket == nil {
			return fmt.Errorf("units bucket not found")
		}

		cursor := bucket.Cursor()
		for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
			var unit domain.Unit
			if err := json.Unmarshal(value, &unit); err != nil {
				continue
			}
			units = append(units, &unit)
		}
		return nil
	})

	return units, err
}

// AddInventorySnapshot stores a historical inventory level snapshot
func (r *BoltInventoryRepository) AddInventorySnapshot(ctx context.Context, itemID string, snapshot *domain.InventoryLevelSnapshot) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if bucket == nil {
			return fmt.Errorf("history bucket not found")
		}

		// Create a time-ordered key: itemID:timestamp
		key := fmt.Sprintf("%s:%s", itemID, snapshot.Timestamp.Format(time.RFC3339Nano))

		data, err := json.Marshal(snapshot)
		if err != nil {
			return fmt.Errorf("failed to marshal snapshot: %w", err)
		}

		return bucket.Put([]byte(key), data)
	})
}

// GetInventoryHistory retrieves historical inventory snapshots with filtering
func (r *BoltInventoryRepository) GetInventoryHistory(ctx context.Context, itemID string, filters HistoryFilters) ([]*domain.InventoryLevelSnapshot, int, error) {
	var snapshots []*domain.InventoryLevelSnapshot
	var totalCount int

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if bucket == nil {
			return fmt.Errorf("history bucket not found")
		}

		prefix := []byte(itemID + ":")
		cursor := bucket.Cursor()

		for key, value := cursor.Seek(prefix); key != nil && strings.HasPrefix(string(key), string(prefix)); key, value = cursor.Next() {
			var snapshot domain.InventoryLevelSnapshot
			if err := json.Unmarshal(value, &snapshot); err != nil {
				continue
			}

			// Apply time filters
			if !filters.StartTime.IsZero() && snapshot.Timestamp.Before(filters.StartTime) {
				continue
			}
			if !filters.EndTime.IsZero() && snapshot.Timestamp.After(filters.EndTime) {
				continue
			}

			totalCount++

			// Apply offset
			if filters.Offset > 0 && len(snapshots) < filters.Offset {
				continue
			}

			// Apply limit
			if filters.Limit > 0 && len(snapshots) >= filters.Limit {
				continue
			}

			snapshots = append(snapshots, &snapshot)
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
func (r *BoltInventoryRepository) GetEarliestSnapshot(ctx context.Context, itemID string) (*domain.InventoryLevelSnapshot, error) {
	var earliest *domain.InventoryLevelSnapshot

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if bucket == nil {
			return fmt.Errorf("history bucket not found")
		}

		prefix := []byte(itemID + ":")
		cursor := bucket.Cursor()

		// Get first item with the prefix (chronologically earliest)
		key, value := cursor.Seek(prefix)
		if key != nil && strings.HasPrefix(string(key), string(prefix)) {
			var snapshot domain.InventoryLevelSnapshot
			if err := json.Unmarshal(value, &snapshot); err != nil {
				return fmt.Errorf("failed to unmarshal snapshot: %w", err)
			}
			earliest = &snapshot
		}

		return nil
	})

	return earliest, err
}

// GetLatestSnapshot retrieves the latest snapshot for an item
func (r *BoltInventoryRepository) GetLatestSnapshot(ctx context.Context, itemID string) (*domain.InventoryLevelSnapshot, error) {
	var latest *domain.InventoryLevelSnapshot

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if bucket == nil {
			return fmt.Errorf("history bucket not found")
		}

		prefix := []byte(itemID + ":")
		cursor := bucket.Cursor()

		// Iterate through all snapshots for this item to find the latest
		for key, value := cursor.Seek(prefix); key != nil && strings.HasPrefix(string(key), string(prefix)); key, value = cursor.Next() {
			var snapshot domain.InventoryLevelSnapshot
			if err := json.Unmarshal(value, &snapshot); err != nil {
				continue
			}

			if latest == nil || snapshot.Timestamp.After(latest.Timestamp) {
				latest = &snapshot
			}
		}

		return nil
	})

	return latest, err
}

// applyGranularityFilter filters snapshots based on requested granularity
func (r *BoltInventoryRepository) applyGranularityFilter(snapshots []*domain.InventoryLevelSnapshot, granularity string) []*domain.InventoryLevelSnapshot {
	if len(snapshots) == 0 {
		return snapshots
	}

	// Sort snapshots by timestamp to ensure chronological order
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.Before(snapshots[j].Timestamp)
	})

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

// initializeDefaultUnits creates default units in the database
func (r *BoltInventoryRepository) initializeDefaultUnits() error {
	// Check if units already exist
	units, err := r.ListUnits(context.Background())
	if err == nil && len(units) > 0 {
		return nil // Units already initialized
	}

	defaultUnits := []*domain.Unit{
		{Name: "Grams", Symbol: "g", Description: "Unit of mass", BaseConversionFactor: 0.001, Category: "weight"},
		{Name: "Kilograms", Symbol: "kg", Description: "Unit of mass", BaseConversionFactor: 1.0, Category: "weight"},
		{Name: "Pounds", Symbol: "lbs", Description: "Unit of mass", BaseConversionFactor: 0.453592, Category: "weight"},
		{Name: "Milliliters", Symbol: "ml", Description: "Unit of volume", BaseConversionFactor: 0.001, Category: "volume"},
		{Name: "Liters", Symbol: "L", Description: "Unit of volume", BaseConversionFactor: 1.0, Category: "volume"},
		{Name: "Cups", Symbol: "cup", Description: "Unit of volume", BaseConversionFactor: 0.236588, Category: "volume"},
		{Name: "Pieces", Symbol: "pcs", Description: "Unit of count", BaseConversionFactor: 1.0, Category: "count"},
	}

	for _, unit := range defaultUnits {
		if err := r.AddUnit(context.Background(), unit); err != nil {
			return fmt.Errorf("failed to add default unit %s: %w", unit.Name, err)
		}
	}

	return nil
}
