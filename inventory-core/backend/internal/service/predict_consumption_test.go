package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/DaDevFox/task-systems/inventory-core/backend/internal/domain"
	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
	"github.com/DaDevFox/task-systems/shared/events"
)

func TestPredictConsumptionSuccess(t *testing.T) {
	mockRepo := &MockRepository{}
	mockEventBus := events.NewEventBus("test-service")
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewInventoryService(mockRepo, mockEventBus, logger)

	// Create test item with active prediction model
	testItem := &domain.InventoryItem{
		ID:           testItemID,
		Name:         testItemName,
		CurrentLevel: 100,
		UnitID:       "units",
		ActivePredictionModel: &pb.PredictionModelConfig{
			ModelConfig: &pb.PredictionModelConfig_Parametric{
				Parametric: &pb.ParametricModel{
					ModelType: &pb.ParametricModel_Linear{
						Linear: &pb.LinearEquationModel{
							BaseLevel: 100,
							Slope:     -5, // 5 units per day consumption
						},
					},
				},
			},
		},
	}

	mockRepo.On("GetItem", mock.Anything, testItemID).Return(testItem, nil)

	req := &pb.PredictConsumptionRequest{
		ItemId:    testItemID,
		DaysAhead: 7,
	}

	resp, err := svc.PredictConsumption(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Prediction)
	require.Equal(t, float64(65), resp.Prediction.Estimate) // 100 - (5 * 7)
	require.NotEmpty(t, resp.Prediction.PredictionModel)
}

func TestPredictConsumptionItemNotFound(t *testing.T) {
	mockRepo := &MockRepository{}
	mockEventBus := events.NewEventBus(testServiceName)
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewInventoryService(mockRepo, mockEventBus, logger)

	notFoundErr := errors.New("item not found")
	mockRepo.On("GetItem", mock.Anything, "nonexistent-item").Return((*domain.InventoryItem)(nil), notFoundErr)

	req := &pb.PredictConsumptionRequest{
		ItemId:    "nonexistent-item",
		DaysAhead: 7,
	}

	resp, err := svc.PredictConsumption(context.Background(), req)

	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "not found")
}

func TestPredictConsumptionNoActiveModel(t *testing.T) {
	mockRepo := &MockRepository{}
	mockEventBus := events.NewEventBus(testServiceName)
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewInventoryService(mockRepo, mockEventBus, logger)

	// Create test item without active prediction model
	testItem := &domain.InventoryItem{
		ID:           testItemID,
		Name:         testItemName,
		CurrentLevel: 100,
		UnitID:       "units",
	}

	mockRepo.On("GetItem", mock.Anything, testItemID).Return(testItem, nil)
	mockRepo.On("UpdateItem", mock.Anything, mock.Anything).Return(nil)

	req := &pb.PredictConsumptionRequest{
		ItemId:    testItemID,
		DaysAhead: 7,
	}

	resp, err := svc.PredictConsumption(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Prediction)
	require.Greater(t, resp.Prediction.Estimate, float64(0)) // Should have some prediction from default model
}

func TestPredictConsumptionInvalidRequest(t *testing.T) {
	mockRepo := &MockRepository{}
	mockEventBus := events.NewEventBus(testServiceName)
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewInventoryService(mockRepo, mockEventBus, logger)

	// Test empty item_id
	req := &pb.PredictConsumptionRequest{
		ItemId:    "",
		DaysAhead: 7,
	}

	resp, err := svc.PredictConsumption(context.Background(), req)

	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "item_id is required")

	// Test negative days_ahead
	req = &pb.PredictConsumptionRequest{
		ItemId:    testItemID,
		DaysAhead: -1,
	}

	resp, err = svc.PredictConsumption(context.Background(), req)

	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "days_ahead must be positive")
}

func TestSetActivePredictionModelSuccess(t *testing.T) {
	mockRepo := &MockRepository{}
	mockEventBus := events.NewEventBus(testServiceName)
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewInventoryService(mockRepo, mockEventBus, logger)

	testItem := &domain.InventoryItem{
		ID:           testItemID,
		Name:         testItemName,
		CurrentLevel: 100,
		UnitID:       "units",
	}

	mockRepo.On("GetItem", mock.Anything, testItemID).Return(testItem, nil)
	mockRepo.On("UpdateItem", mock.Anything, mock.Anything).Return(nil)

	req := &pb.SetActivePredictionModelRequest{
		ItemId: testItemID,
		ModelConfig: &pb.PredictionModelConfig{
			ModelConfig: &pb.PredictionModelConfig_Parametric{
				Parametric: &pb.ParametricModel{
					ModelType: &pb.ParametricModel_Linear{
						Linear: &pb.LinearEquationModel{
							BaseLevel: 100,
							Slope:     -2,
						},
					},
				},
			},
		},
	}

	resp, err := svc.SetActivePredictionModel(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Item)
	require.True(t, resp.ModelChanged)
}

func TestGetActivePredictionModelSuccess(t *testing.T) {
	mockRepo := &MockRepository{}
	mockEventBus := events.NewEventBus(testServiceName)
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewInventoryService(mockRepo, mockEventBus, logger)

	testItem := &domain.InventoryItem{
		ID:           testItemID,
		Name:         testItemName,
		CurrentLevel: 100,
		UnitID:       "units",
		ActivePredictionModel: &pb.PredictionModelConfig{
			ModelConfig: &pb.PredictionModelConfig_Parametric{
				Parametric: &pb.ParametricModel{
					ModelType: &pb.ParametricModel_Linear{
						Linear: &pb.LinearEquationModel{
							BaseLevel: 100,
							Slope:     -2,
						},
					},
				},
			},
		},
	}

	mockRepo.On("GetItem", mock.Anything, testItemID).Return(testItem, nil)

	req := &pb.GetActivePredictionModelRequest{
		ItemId: testItemID,
	}

	resp, err := svc.GetActivePredictionModel(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.ModelConfig)
}

func TestPredictConsumptionLogisticModel(t *testing.T) {
	mockRepo := &MockRepository{}
	mockEventBus := events.NewEventBus(testServiceName)
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewInventoryService(mockRepo, mockEventBus, logger)

	// Create test item with logistic prediction model
	testItem := &domain.InventoryItem{
		ID:           testItemID,
		Name:         testItemName,
		CurrentLevel: 50,
		UnitID:       "units",
		ActivePredictionModel: &pb.PredictionModelConfig{
			ModelConfig: &pb.PredictionModelConfig_Parametric{
				Parametric: &pb.ParametricModel{
					ModelType: &pb.ParametricModel_Logistic{
						Logistic: &pb.LogisticEquationModel{
							GrowthRate:        0.1,
							CarryingCapacity:  100,
							InitialPopulation: 50,
						},
					},
				},
			},
		},
	}

	mockRepo.On("GetItem", mock.Anything, testItemID).Return(testItem, nil)

	req := &pb.PredictConsumptionRequest{
		ItemId:    testItemID,
		DaysAhead: 10,
	}

	resp, err := svc.PredictConsumption(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Prediction)
	require.Greater(t, resp.Prediction.Estimate, float64(50))      // Should grow from initial value
	require.LessOrEqual(t, resp.Prediction.Estimate, float64(100)) // Should not exceed carrying capacity
}
