package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

// setupTestServiceWithRealDB creates a test service with real database (for integration tests)
func setupTestServiceWithRealDB(t *testing.T) (*InventoryService, func()) {
	tmpDir, err := os.MkdirTemp("", "test_service")
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := repository.NewInventoryRepository(dbPath, repository.DatabaseTypeBadger)
	require.NoError(t, err)

	// Create unique event bus for each test to avoid interference
	eventBus := events.NewEventBus(fmt.Sprintf("test_%s_%d", t.Name(), time.Now().UnixNano()))
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

	service := NewInventoryService(repo, eventBus, logger)

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tmpDir)
	}

	return service, cleanup
}

// createTestDomainItem creates a test domain inventory item (for unit tests with mocks)
func createTestDomainItem() *domain.InventoryItem {
	return &domain.InventoryItem{
		ID:                testItemID,
		Name:              testItemName,
		Description:       testItemDescription,
		CurrentLevel:      10.0,
		MaxCapacity:       100.0,
		LowStockThreshold: 5.0,
		UnitID:            "kg",
		AlternateUnitIDs:  []string{"g"},
		ConsumptionBehavior: &domain.ConsumptionBehavior{
			Pattern:           domain.ConsumptionPatternLinear,
			AverageRatePerDay: 1.0,
			Variance:          0.1,
			SeasonalFactors:   []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
			LastUpdated:       time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  map[string]string{"category": "test", "location": "warehouse"},
	}
}

// createTestDomainUnit creates a test domain unit (for unit tests with mocks)
func createTestDomainUnit(id, name string) *domain.Unit {
	return &domain.Unit{
		ID:                   id,
		Name:                 name,
		Symbol:               id,
		Description:          "Test unit for " + name,
		BaseConversionFactor: 1.0,
		Category:             "test",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

// createTestItemViaService creates a test inventory item through the service API (for integration tests)
func createTestItemViaService(t *testing.T, service *InventoryService) (string, string) {
	ctx := context.Background()

	// Create unit first
	unitReq := &pb.AddUnitRequest{
		Name:                 "Kilograms",
		Symbol:               "kg",
		Description:          "Unit of mass",
		BaseConversionFactor: 1.0,
		Category:             "weight",
	}
	unitResp, err := service.AddUnit(ctx, unitReq)
	require.NoError(t, err)

	// Create item with initial level - this should create initial history
	addReq := &pb.AddInventoryItemRequest{
		Name:              "Test Item",
		Description:       "Item created via service API",
		InitialLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitId:            unitResp.Unit.Id,
	}
	addResp, err := service.AddInventoryItem(ctx, addReq)
	require.NoError(t, err)

	return addResp.Item.Id, unitResp.Unit.Id
}
