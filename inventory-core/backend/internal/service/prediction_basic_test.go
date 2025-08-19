package service

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

func TestPredictConsumptionBasic(t *testing.T) {
	mockRepo := &MockRepository{}
	mockEventBus := events.NewEventBus("test-service")
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewInventoryService(mockRepo, mockEventBus, logger)

	// Test item without active prediction model
	testItem := &domain.InventoryItem{
		ID:           "test-item",
		Name:         "Test Item",
		CurrentLevel: 100,
		UnitID:       "units",
	}

	mockRepo.On("GetItem", mock.Anything, "test-item").Return(testItem, nil)
	mockRepo.On("UpdateItem", mock.Anything, mock.Anything).Return(nil)

	req := &pb.PredictConsumptionRequest{
		ItemId:    "test-item",
		DaysAhead: 7,
	}

	resp, err := svc.PredictConsumption(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Prediction)
}
