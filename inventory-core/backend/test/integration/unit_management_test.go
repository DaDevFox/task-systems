package integration

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
	"github.com/sirupsen/logrus"
)

func TestUnitManagementIntegration(t *testing.T) {
	// Setup
	repo, cleanup := setupTestRepository()
	defer cleanup()

	eventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs during tests

	inventoryService := service.NewInventoryService(repo, eventBus, logger)
	ctx := context.Background()

	t.Run("ListUnits", func(t *testing.T) {
		// Test listing units (should return default units)
		req := &pb.ListUnitsRequest{}
		resp, err := inventoryService.ListUnits(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Greater(t, len(resp.Units), 0, "Should have default units")

		// Check that we have some expected default units
		unitIDs := make(map[string]bool)
		for _, unit := range resp.Units {
			unitIDs[unit.Id] = true
		}

		assert.True(t, unitIDs["kg"], "Should have kg unit")
		assert.True(t, unitIDs["g"], "Should have g unit")
		assert.True(t, unitIDs["l"], "Should have l unit")
	})

	t.Run("AddUnit", func(t *testing.T) {
		// Test adding a new unit
		req := &pb.AddUnitRequest{
			Name:                 "Tablespoons",
			Symbol:               "tbsp",
			Description:          "Imperial volume measurement",
			BaseConversionFactor: 0.0147868, // tbsp to liters
			Category:             "volume",
			Metadata:             map[string]string{"type": "cooking"},
		}

		resp, err := inventoryService.AddUnit(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Unit)

		assert.Equal(t, "Tablespoons", resp.Unit.Name)
		assert.Equal(t, "tbsp", resp.Unit.Symbol)
		assert.Equal(t, "Imperial volume measurement", resp.Unit.Description)
		assert.Equal(t, 0.0147868, resp.Unit.BaseConversionFactor)
		assert.Equal(t, "volume", resp.Unit.Category)
		assert.Equal(t, "cooking", resp.Unit.Metadata["type"])
		assert.NotEmpty(t, resp.Unit.Id)
		assert.NotNil(t, resp.Unit.CreatedAt)
		assert.NotNil(t, resp.Unit.UpdatedAt)
	})

	t.Run("GetUnit", func(t *testing.T) {
		// First, add a unit
		addReq := &pb.AddUnitRequest{
			Name:                 "Teaspoons",
			Symbol:               "tsp",
			Description:          "Imperial volume measurement",
			BaseConversionFactor: 0.00492892, // tsp to liters
			Category:             "volume",
		}

		addResp, err := inventoryService.AddUnit(ctx, addReq)
		require.NoError(t, err)
		unitId := addResp.Unit.Id

		// Now get the unit
		getReq := &pb.GetUnitRequest{UnitId: unitId}
		getResp, err := inventoryService.GetUnit(ctx, getReq)

		require.NoError(t, err)
		require.NotNil(t, getResp)
		require.NotNil(t, getResp.Unit)

		assert.Equal(t, unitId, getResp.Unit.Id)
		assert.Equal(t, "Teaspoons", getResp.Unit.Name)
		assert.Equal(t, "tsp", getResp.Unit.Symbol)
		assert.Equal(t, "volume", getResp.Unit.Category)
	})

	t.Run("UpdateUnit", func(t *testing.T) {
		// First, add a unit
		addReq := &pb.AddUnitRequest{
			Name:                 "Test Unit",
			Symbol:               "test",
			BaseConversionFactor: 1.0,
			Category:             "test",
		}

		addResp, err := inventoryService.AddUnit(ctx, addReq)
		require.NoError(t, err)
		unitId := addResp.Unit.Id

		// Now update the unit
		updateReq := &pb.UpdateUnitRequest{
			UnitId:               unitId,
			Name:                 "Updated Test Unit",
			Symbol:               "utest",
			Description:          "Updated description",
			BaseConversionFactor: 2.0,
			Category:             "updated",
			Metadata:             map[string]string{"updated": "true"},
		}

		updateResp, err := inventoryService.UpdateUnit(ctx, updateReq)

		require.NoError(t, err)
		require.NotNil(t, updateResp)
		require.NotNil(t, updateResp.Unit)
		assert.True(t, updateResp.UnitChanged)

		assert.Equal(t, unitId, updateResp.Unit.Id)
		assert.Equal(t, "Updated Test Unit", updateResp.Unit.Name)
		assert.Equal(t, "utest", updateResp.Unit.Symbol)
		assert.Equal(t, "Updated description", updateResp.Unit.Description)
		assert.Equal(t, 2.0, updateResp.Unit.BaseConversionFactor)
		assert.Equal(t, "updated", updateResp.Unit.Category)
		assert.Equal(t, "true", updateResp.Unit.Metadata["updated"])
	})

	t.Run("DeleteUnit", func(t *testing.T) {
		// First, add a unit
		addReq := &pb.AddUnitRequest{
			Name:                 "Deletable Unit",
			Symbol:               "del",
			BaseConversionFactor: 1.0,
			Category:             "test",
		}

		addResp, err := inventoryService.AddUnit(ctx, addReq)
		require.NoError(t, err)
		unitId := addResp.Unit.Id
		unitName := addResp.Unit.Name

		// Now delete the unit
		deleteReq := &pb.DeleteUnitRequest{
			UnitId: unitId,
			Force:  true, // Force deletion for test
		}

		deleteResp, err := inventoryService.DeleteUnit(ctx, deleteReq)

		require.NoError(t, err)
		require.NotNil(t, deleteResp)
		assert.True(t, deleteResp.UnitDeleted)
		assert.Equal(t, unitId, deleteResp.DeletedUnitId)
		assert.Equal(t, unitName, deleteResp.DeletedUnitName)

		// Verify unit is deleted by trying to get it
		getReq := &pb.GetUnitRequest{UnitId: unitId}
		_, err = inventoryService.GetUnit(ctx, getReq)

		assert.Error(t, err, "Should return error when trying to get deleted unit")
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		// Test validation errors
		t.Run("AddUnit with empty name", func(t *testing.T) {
			req := &pb.AddUnitRequest{
				Name:                 "", // Empty name should fail
				Symbol:               "test",
				BaseConversionFactor: 1.0,
			}

			_, err := inventoryService.AddUnit(ctx, req)
			assert.Error(t, err)
		})

		t.Run("AddUnit with empty symbol", func(t *testing.T) {
			req := &pb.AddUnitRequest{
				Name:                 "Test Unit",
				Symbol:               "", // Empty symbol should fail
				BaseConversionFactor: 1.0,
			}

			_, err := inventoryService.AddUnit(ctx, req)
			assert.Error(t, err)
		})

		t.Run("AddUnit with zero conversion factor", func(t *testing.T) {
			req := &pb.AddUnitRequest{
				Name:                 "Test Unit",
				Symbol:               "test",
				BaseConversionFactor: 0, // Zero should fail
			}

			_, err := inventoryService.AddUnit(ctx, req)
			assert.Error(t, err)
		})

		t.Run("GetUnit with empty ID", func(t *testing.T) {
			req := &pb.GetUnitRequest{UnitId: ""}

			_, err := inventoryService.GetUnit(ctx, req)
			assert.Error(t, err)
		})

		t.Run("GetUnit with non-existent ID", func(t *testing.T) {
			req := &pb.GetUnitRequest{UnitId: "non-existent"}

			_, err := inventoryService.GetUnit(ctx, req)
			assert.Error(t, err)
		})
	})
}

// setupTestRepository creates a test repository instance for integration testing
func setupTestRepository() (repository.InventoryRepository, func()) {
	// Create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "badger_test_*")
	if err != nil {
		panic(err)
	}

	repo, err := repository.NewBadgerInventoryRepository(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		panic(err)
	}

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tempDir)
	}
	return repo, cleanup
}
