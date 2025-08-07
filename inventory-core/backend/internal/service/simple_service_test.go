package service

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(2), resp.TotalCount)
	assert.Len(t, resp.Items, 2)

	// Check first item
	assert.Equal(t, "item1", resp.Items[0].Id)
	assert.Equal(t, "Test Item 1", resp.Items[0].Name)
	assert.Equal(t, 100.0, resp.Items[0].CurrentLevel)

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
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(0), resp.TotalCount)
	assert.Len(t, resp.Items, 0)

	mockRepo.AssertExpectations(t)
}
