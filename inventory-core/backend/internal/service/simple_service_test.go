package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/repository"
	pb "github.com/DaDevFox/task-systems/inventory-core/proto/proto"
	"github.com/DaDevFox/task-systems/shared/events"
)

// MockRepository implements repository.InventoryRepository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) AddItem(ctx context.Context, item *domain.InventoryItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockRepository) GetItem(ctx context.Context, id string) (*domain.InventoryItem, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.InventoryItem), args.Error(1)
}

func (m *MockRepository) UpdateItem(ctx context.Context, item *domain.InventoryItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockRepository) DeleteItem(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListItems(ctx context.Context, filters repository.ListFilters) ([]*domain.InventoryItem, int, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]*domain.InventoryItem), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetAllItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.InventoryItem), args.Error(1)
}

func (m *MockRepository) GetLowStockItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.InventoryItem), args.Error(1)
}

func (m *MockRepository) GetEmptyItems(ctx context.Context) ([]*domain.InventoryItem, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.InventoryItem), args.Error(1)
}

func (m *MockRepository) AddUnit(ctx context.Context, unit *domain.Unit) error {
	args := m.Called(ctx, unit)
	return args.Error(0)
}

func (m *MockRepository) GetUnit(ctx context.Context, id string) (*domain.Unit, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Unit), args.Error(1)
}

func (m *MockRepository) ListUnits(ctx context.Context) ([]*domain.Unit, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.Unit), args.Error(1)
}

func TestSimpleInventoryService_ListInventoryItems(t *testing.T) {
	// Setup
	mockRepo := &MockRepository{}
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	service := NewInventoryService(mockRepo, mockEventBus, logger)

	// Create test items
	testItems := []*domain.InventoryItem{
		{
			ID:                "item1",
			Name:              "Test Item 1",
			Description:       "Test Description 1",
			CurrentLevel:      100.0,
			MaxCapacity:       200.0,
			LowStockThreshold: 20.0,
			UnitID:            "kg",
		},
		{
			ID:                "item2",
			Name:              "Test Item 2",
			Description:       "Test Description 2",
			CurrentLevel:      50.0,
			MaxCapacity:       100.0,
			LowStockThreshold: 10.0,
			UnitID:            "pieces",
		},
	}

	// Setup mock expectations
	filters := repository.ListFilters{
		Limit:  50, // Default limit
		Offset: 0,
	}
	mockRepo.On("ListItems", mock.Anything, filters).Return(testItems, 2, nil)

	// Execute
	ctx := context.Background()
	req := &pb.ListInventoryItemsRequest{}

	resp, err := service.ListInventoryItems(ctx, req)

	// Assert
	switch {
	case err != nil:
		t.Errorf("Expected no error, got %v", err)
	case resp == nil:
		t.Fatal("Expected response, got nil")
	case resp.TotalCount != 2:
		t.Errorf("Expected total count 2, got %d", resp.TotalCount)
	case len(resp.Items) != 2:
		t.Errorf("Expected 2 items, got %d (despite resp.TotalCount == 2)", len(resp.Items))
	}

	// Check first item
	switch {
	case resp.Items[0].Id != "item1":
		t.Errorf("Expected first item ID 'item1', got %s", resp.Items[0].Id)
	case resp.Items[0].Name != "Test Item 1":
		t.Errorf("Expected first item name 'Test Item 1', got %s", resp.Items[0].Name)
	case resp.Items[0].CurrentLevel != 100.0:
		t.Errorf("Expected first item current level 100.0, got %f", resp.Items[0].CurrentLevel)
	}
	// Verify mock was called
	mockRepo.AssertExpectations(t)
}

func TestSimpleInventoryService_ListInventoryItems_WithFilters(t *testing.T) {
	// Setup
	mockRepo := &MockRepository{}
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	service := NewInventoryService(mockRepo, mockEventBus, logger)

	// Setup mock expectations with filters
	filters := repository.ListFilters{
		LowStockOnly:   true,
		UnitTypeFilter: "kg",
		Limit:          10,
		Offset:         5,
	}
	mockRepo.On("ListItems", mock.Anything, filters).Return([]*domain.InventoryItem{}, 0, nil)

	// Execute
	ctx := context.Background()
	req := &pb.ListInventoryItemsRequest{
		LowStockOnly:   true,
		UnitTypeFilter: "kg",
		Limit:          10,
		Offset:         5,
	}

	resp, err := service.ListInventoryItems(ctx, req)

	// Assert
	switch {
	case err != nil:
		t.Errorf("Expected no error, got %v", err)
	case resp == nil:
		t.Fatal("Expected response, got nil")
	case resp.TotalCount != 0:
		t.Errorf("Expected total count 0, got %d", resp.TotalCount)
	case len(resp.Items) != 0:
		t.Errorf("Expected 0 items, got %d", len(resp.Items))
	}

	mockRepo.AssertExpectations(t)
}

func TestInventoryService_ConfigureInventoryItem(t *testing.T) {
	// Setup
	mockRepo := &MockRepository{}
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	service := NewInventoryService(mockRepo, mockEventBus, logger)

	// Create test item
	existingItem := &domain.InventoryItem{
		ID:                "test-item-id",
		Name:              "Original Name",
		Description:       "Original Description",
		CurrentLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitID:            "kg",
		Metadata:          map[string]string{"category": "original"},
	}

	// Create test unit
	testUnit := &domain.Unit{
		ID:   "lbs",
		Name: "Pounds",
	}

	// Setup mock expectations
	mockRepo.On("GetItem", mock.Anything, "test-item-id").Return(existingItem, nil)
	mockRepo.On("GetUnit", mock.Anything, "lbs").Return(testUnit, nil)
	mockRepo.On("UpdateItem", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	// Execute
	ctx := context.Background()
	req := &pb.ConfigureInventoryItemRequest{
		ItemId:             "test-item-id",
		Name:               "Updated Name",
		Description:        "Updated Description",
		MaxCapacity:        300.0,
		LowStockThreshold:  30.0,
		UnitId:             "lbs",
		Metadata:           map[string]string{"category": "updated", "priority": "high"},
	}

	resp, err := service.ConfigureInventoryItem(ctx, req)

	// Assert
	switch {
	case err != nil:
		t.Errorf("Expected no error, got %v", err)
	case resp == nil:
		t.Fatal("Expected response, got nil")
	case resp.Item == nil:
		t.Fatal("Expected item in response, got nil")
	case resp.Item.Id != "test-item-id":
		t.Errorf("Expected item ID 'test-item-id', got %s", resp.Item.Id)
	case resp.Item.Name != "Updated Name":
		t.Errorf("Expected item name 'Updated Name', got %s", resp.Item.Name)
	case resp.Item.Description != "Updated Description":
		t.Errorf("Expected item description 'Updated Description', got %s", resp.Item.Description)
	case resp.Item.MaxCapacity != 300.0:
		t.Errorf("Expected max capacity 300.0, got %f", resp.Item.MaxCapacity)
	case resp.Item.LowStockThreshold != 30.0:
		t.Errorf("Expected low stock threshold 30.0, got %f", resp.Item.LowStockThreshold)
	case resp.Item.UnitId != "lbs":
		t.Errorf("Expected unit ID 'lbs', got %s", resp.Item.UnitId)
	case resp.Item.CurrentLevel != 100.0:
		t.Errorf("Expected current level unchanged at 100.0, got %f", resp.Item.CurrentLevel)
	case len(resp.Item.Metadata) != 2:
		t.Errorf("Expected 2 metadata items, got %d", len(resp.Item.Metadata))
	case resp.Item.Metadata["category"] != "updated":
		t.Errorf("Expected metadata category 'updated', got %s", resp.Item.Metadata["category"])
	case resp.Item.Metadata["priority"] != "high":
		t.Errorf("Expected metadata priority 'high', got %s", resp.Item.Metadata["priority"])
	}

	mockRepo.AssertExpectations(t)
}

func TestInventoryService_ConfigureInventoryItem_ValidationErrors(t *testing.T) {
	// Setup
	mockRepo := &MockRepository{}
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	service := NewInventoryService(mockRepo, mockEventBus, logger)

	ctx := context.Background()

	tests := []struct {
		name        string
		req         *pb.ConfigureInventoryItemRequest
		expectError string
	}{
		{
			name: "missing_item_id",
			req: &pb.ConfigureInventoryItemRequest{
				Name:               "Test Name",
				Description:        "Test Description",
				MaxCapacity:        200.0,
				LowStockThreshold:  20.0,
				UnitId:             "kg",
			},
			expectError: "item_id is required",
		},
		{
			name: "missing_name",
			req: &pb.ConfigureInventoryItemRequest{
				ItemId:             "test-item-id",
				Description:        "Test Description",
				MaxCapacity:        200.0,
				LowStockThreshold:  20.0,
				UnitId:             "kg",
			},
			expectError: "item name is required",
		},
		{
			name: "missing_unit_id",
			req: &pb.ConfigureInventoryItemRequest{
				ItemId:             "test-item-id",
				Name:               "Test Name",
				Description:        "Test Description",
				MaxCapacity:        200.0,
				LowStockThreshold:  20.0,
			},
			expectError: "unit_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.ConfigureInventoryItem(ctx, tt.req)
			switch {
			case err == nil:
				t.Errorf("Expected error containing '%s', got no error", tt.expectError)
			case resp != nil:
				t.Error("Expected nil response when error occurs, got non-nil")
			}
		})
	}
}

func TestInventoryService_ConfigureInventoryItem_ItemNotFound(t *testing.T) {
	// Setup
	mockRepo := &MockRepository{}
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	service := NewInventoryService(mockRepo, mockEventBus, logger)

	// Setup mock to return error for item not found
	mockRepo.On("GetItem", mock.Anything, "nonexistent-item").Return((*domain.InventoryItem)(nil), errors.New("item not found"))

	// Execute
	ctx := context.Background()
	req := &pb.ConfigureInventoryItemRequest{
		ItemId:             "nonexistent-item",
		Name:               "Test Name",
		Description:        "Test Description",
		MaxCapacity:        200.0,
		LowStockThreshold:  20.0,
		UnitId:             "kg",
	}

	resp, err := service.ConfigureInventoryItem(ctx, req)

	// Assert
	switch {
	case err == nil:
		t.Error("Expected error for nonexistent item, got no error")
	case resp != nil:
		t.Error("Expected nil response when item not found, got non-nil")
	}

	mockRepo.AssertExpectations(t)
}

func TestInventoryService_ConfigureInventoryItem_InvalidUnit(t *testing.T) {
	// Setup
	mockRepo := &MockRepository{}
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	service := NewInventoryService(mockRepo, mockEventBus, logger)

	// Create test item with existing unit
	existingItem := &domain.InventoryItem{
		ID:                "test-item-id",
		Name:              "Original Name",
		Description:       "Original Description",
		CurrentLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitID:            "kg",
	}

	// Setup mock expectations
	mockRepo.On("GetItem", mock.Anything, "test-item-id").Return(existingItem, nil)
	mockRepo.On("GetUnit", mock.Anything, "invalid-unit").Return((*domain.Unit)(nil), errors.New("unit not found"))

	// Execute
	ctx := context.Background()
	req := &pb.ConfigureInventoryItemRequest{
		ItemId:             "test-item-id",
		Name:               "Updated Name",
		Description:        "Updated Description",
		MaxCapacity:        300.0,
		LowStockThreshold:  30.0,
		UnitId:             "invalid-unit",
	}

	resp, err := service.ConfigureInventoryItem(ctx, req)

	// Assert
	switch {
	case err == nil:
		t.Error("Expected error for invalid unit, got no error")
	case resp != nil:
		t.Error("Expected nil response when invalid unit, got non-nil")
	}

	mockRepo.AssertExpectations(t)
}

func TestInventoryService_ConfigureInventoryItem_SameUnit(t *testing.T) {
	// Setup
	mockRepo := &MockRepository{}
	mockEventBus := events.GetGlobalBus("test")
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	service := NewInventoryService(mockRepo, mockEventBus, logger)

	// Create test item with existing unit
	existingItem := &domain.InventoryItem{
		ID:                "test-item-id",
		Name:              "Original Name",
		Description:       "Original Description",
		CurrentLevel:      100.0,
		MaxCapacity:       200.0,
		LowStockThreshold: 20.0,
		UnitID:            "kg",
		Metadata:          map[string]string{"category": "original"},
	}

	// Setup mock expectations - note we don't expect GetUnit to be called since unit hasn't changed
	mockRepo.On("GetItem", mock.Anything, "test-item-id").Return(existingItem, nil)
	mockRepo.On("UpdateItem", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	// Execute
	ctx := context.Background()
	req := &pb.ConfigureInventoryItemRequest{
		ItemId:             "test-item-id",
		Name:               "Updated Name",
		Description:        "Updated Description",
		MaxCapacity:        300.0,
		LowStockThreshold:  30.0,
		UnitId:             "kg", // Same unit as existing
		Metadata:           map[string]string{"category": "updated"},
	}

	resp, err := service.ConfigureInventoryItem(ctx, req)

	// Assert
	switch {
	case err != nil:
		t.Errorf("Expected no error, got %v", err)
	case resp == nil:
		t.Fatal("Expected response, got nil")
	case resp.Item == nil:
		t.Fatal("Expected item in response, got nil")
	case resp.Item.UnitId != "kg":
		t.Errorf("Expected unit ID 'kg', got %s", resp.Item.UnitId)
	}

	mockRepo.AssertExpectations(t)
}
