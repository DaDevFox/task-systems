package service

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

// Test setup helper
func setupUpdateInventoryItemTest() (*InventoryService, *MockRepository) {
	repo := &MockRepository{}
	eventBus := events.NewEventBus("inventory-service")
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce log noise in tests

	service := NewInventoryService(repo, eventBus, logger)
	service.DisableAuthForTesting()

	return service, repo
}

func TestUpdateInventoryItemSuccess(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup test data
	testItem := createTestDomainItem()

	// Setup mock expectations (no unit validation needed since we're not changing units)
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)
	repo.On("UpdateItem", ctx, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	req := &pb.UpdateInventoryItemRequest{
		ItemId:            testItemID,
		Name:              "Updated Test Item",
		Description:       "Updated description",
		MaxCapacity:       150.0,
		LowStockThreshold: 8.0,
		// Don't change units, so no validation needed
		Metadata: map[string]string{"category": "updated", "location": "storage", "new_field": "value"},
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err != nil:
		t.Errorf(expectedNoError, err)
	case resp == nil:
		t.Fatal(expectedResponse)
	case !resp.ItemChanged:
		t.Error(expectedItemChanged)
	case resp.Item.Name != "Updated Test Item":
		t.Errorf("Expected name 'Updated Test Item', got %s", resp.Item.Name)
	case resp.Item.Description != "Updated description":
		t.Errorf("Expected description 'Updated description', got %s", resp.Item.Description)
	case resp.Item.MaxCapacity != 150.0:
		t.Errorf("Expected max capacity 150.0, got %f", resp.Item.MaxCapacity)
	case resp.Item.LowStockThreshold != 8.0:
		t.Errorf("Expected low stock threshold 8.0, got %f", resp.Item.LowStockThreshold)
	case len(resp.Item.Metadata) != 3:
		t.Errorf("Expected 3 metadata fields, got %d", len(resp.Item.Metadata))
	case resp.Item.Metadata["category"] != "updated":
		t.Errorf("Expected metadata category 'updated', got %s", resp.Item.Metadata["category"])
	case resp.Item.Metadata["new_field"] != "value":
		t.Errorf("Expected metadata new_field 'value', got %s", resp.Item.Metadata["new_field"])
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemNoChanges(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup test data
	testItem := createTestDomainItem()

	// Setup mock expectations
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)

	// Request with same values as existing item
	req := &pb.UpdateInventoryItemRequest{
		ItemId:            testItemID,
		Name:              testItemName,
		Description:       testItemDescription,
		MaxCapacity:       100.0,
		LowStockThreshold: 5.0,
		UnitId:            "kg",
		AlternateUnitIds:  []string{"g"},
		Metadata:          map[string]string{"category": "test", "location": "warehouse"},
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err != nil:
		t.Errorf(expectedNoError, err)
	case resp == nil:
		t.Fatal(expectedResponse)
	case resp.ItemChanged:
		t.Error("Expected no changes, but item was marked as changed")
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemConsumptionBehaviorUpdate(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup test data
	testItem := createTestDomainItem()

	// Setup mock expectations
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)
	repo.On("UpdateItem", ctx, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	req := &pb.UpdateInventoryItemRequest{
		ItemId: testItemID,
		ConsumptionBehavior: &pb.ConsumptionBehavior{
			Pattern:           pb.ConsumptionPattern_CONSUMPTION_PATTERN_SEASONAL,
			AverageRatePerDay: 2.0,
			Variance:          0.2,
			SeasonalFactors:   []float64{1.2, 1.1, 1.0, 0.9, 0.8, 0.7, 0.8, 0.9, 1.0, 1.1, 1.2, 1.3},
		},
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err != nil:
		t.Errorf(expectedNoError, err)
	case resp == nil:
		t.Fatal(expectedResponse)
	case !resp.ItemChanged:
		t.Error(expectedItemChanged)
	case resp.Item.ConsumptionBehavior.Pattern != pb.ConsumptionPattern_CONSUMPTION_PATTERN_SEASONAL:
		t.Errorf("Expected seasonal pattern, got %v", resp.Item.ConsumptionBehavior.Pattern)
	case resp.Item.ConsumptionBehavior.AverageRatePerDay != 2.0:
		t.Errorf("Expected average rate 2.0, got %f", resp.Item.ConsumptionBehavior.AverageRatePerDay)
	case len(resp.Item.ConsumptionBehavior.SeasonalFactors) != 12:
		t.Errorf("Expected 12 seasonal factors, got %d", len(resp.Item.ConsumptionBehavior.SeasonalFactors))
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemInvalidItemId(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	req := &pb.UpdateInventoryItemRequest{
		ItemId: "",
		Name:   updatedName,
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err == nil:
		t.Error("Expected error for empty item_id")
	case resp != nil:
		t.Error(expectedNilResponse)
	default:
		st, ok := status.FromError(err)
		if !ok {
			t.Error(expectedGRPCError)
		} else if st.Code() != codes.InvalidArgument {
			t.Errorf(expectedInvalidArg, st.Code())
		}
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemItemNotFound(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup mock expectations
	repo.On("GetItem", ctx, "non-existent-item").Return((*domain.InventoryItem)(nil), errors.New("item not found"))

	req := &pb.UpdateInventoryItemRequest{
		ItemId: "non-existent-item",
		Name:   updatedName,
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err == nil:
		t.Error("Expected error for non-existent item")
	case resp != nil:
		t.Error(expectedNilResponse)
	default:
		st, ok := status.FromError(err)
		if !ok {
			t.Error(expectedGRPCError)
		} else if st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound code, got %v", st.Code())
		}
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemInvalidUnitId(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup test data
	testItem := createTestDomainItem()

	// Setup mock expectations
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)
	repo.On("GetUnit", ctx, "invalid-unit").Return((*domain.Unit)(nil), errors.New("unit not found"))

	req := &pb.UpdateInventoryItemRequest{
		ItemId: testItemID,
		UnitId: "invalid-unit",
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err == nil:
		t.Error("Expected error for invalid unit_id")
	case resp != nil:
		t.Error(expectedNilResponse)
	default:
		st, ok := status.FromError(err)
		if !ok {
			t.Error(expectedGRPCError)
		} else if st.Code() != codes.InvalidArgument {
			t.Errorf(expectedInvalidArg, st.Code())
		}
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemInvalidAlternateUnitId(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup test data
	testItem := createTestDomainItem()
	kgUnit := createTestDomainUnit("kg", "Kilograms")

	// Setup mock expectations
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)
	repo.On("GetUnit", ctx, "kg").Return(kgUnit, nil)
	repo.On("GetUnit", ctx, "invalid-unit").Return((*domain.Unit)(nil), errors.New("unit not found"))

	req := &pb.UpdateInventoryItemRequest{
		ItemId:           testItemID,
		AlternateUnitIds: []string{"kg", "invalid-unit"},
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err == nil:
		t.Error("Expected error for invalid alternate unit_id")
	case resp != nil:
		t.Error(expectedNilResponse)
	default:
		st, ok := status.FromError(err)
		if !ok {
			t.Error(expectedGRPCError)
		} else if st.Code() != codes.InvalidArgument {
			t.Errorf(expectedInvalidArg, st.Code())
		}
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemRepositoryError(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup test data
	testItem := createTestDomainItem()

	// Setup mock expectations
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)
	repo.On("UpdateItem", ctx, mock.AnythingOfType("*domain.InventoryItem")).Return(errors.New(databaseError))

	req := &pb.UpdateInventoryItemRequest{
		ItemId: testItemID,
		Name:   updatedName,
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err == nil:
		t.Error("Expected error from repository UpdateItem")
	case resp != nil:
		t.Error(expectedNilResponse)
	default:
		st, ok := status.FromError(err)
		if !ok {
			t.Error(expectedGRPCError)
		} else if st.Code() != codes.Internal {
			t.Errorf("Expected Internal code, got %v", st.Code())
		}
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemMetadataClearing(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup test data
	testItem := createTestDomainItem()

	// Setup mock expectations
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)
	repo.On("UpdateItem", ctx, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	// Update with empty metadata to clear existing metadata
	req := &pb.UpdateInventoryItemRequest{
		ItemId:   testItemID,
		Metadata: map[string]string{}, // Empty map should clear metadata
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err != nil:
		t.Errorf(expectedNoError, err)
	case resp == nil:
		t.Fatal(expectedResponse)
	case !resp.ItemChanged:
		t.Error(expectedItemChanged)
	case len(resp.Item.Metadata) != 0:
		t.Errorf("Expected empty metadata, got %d fields", len(resp.Item.Metadata))
	}

	repo.AssertExpectations(t)
}

func TestUpdateInventoryItemAlternateUnitsUpdate(t *testing.T) {
	service, repo := setupUpdateInventoryItemTest()
	ctx := authenticatedTestContext()

	// Setup test data
	testItem := createTestDomainItem()
	kgUnit := createTestDomainUnit("kg", "Kilograms")

	// Setup mock expectations
	repo.On("GetItem", ctx, testItemID).Return(testItem, nil)
	repo.On("GetUnit", ctx, "kg").Return(kgUnit, nil)
	repo.On("UpdateItem", ctx, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	// Update with different alternate units
	req := &pb.UpdateInventoryItemRequest{
		ItemId:           testItemID,
		AlternateUnitIds: []string{"kg"}, // Changed from ["g"] to ["kg"]
	}

	resp, err := service.UpdateInventoryItem(ctx, req)

	switch {
	case err != nil:
		t.Errorf(expectedNoError, err)
	case resp == nil:
		t.Fatal(expectedResponse)
	case !resp.ItemChanged:
		t.Error(expectedItemChanged)
	case len(resp.Item.AlternateUnitIds) != 1:
		t.Errorf("Expected 1 alternate unit, got %d", len(resp.Item.AlternateUnitIds))
	case resp.Item.AlternateUnitIds[0] != "kg":
		t.Errorf("Expected alternate unit 'kg', got %s", resp.Item.AlternateUnitIds[0])
	}

	repo.AssertExpectations(t)
}
