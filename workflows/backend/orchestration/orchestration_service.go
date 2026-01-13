package orchestration

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/DaDevFox/task-systems/workflows/backend/clients"
)

// OrchestrationService coordinates between inventory-core and tasker-core services
type OrchestrationService struct {
	inventoryClient *clients.InventoryClient
	taskClient      *clients.TaskClient
	logger          *logrus.Logger
}

// NewOrchestrationService creates a new orchestration service
func NewOrchestrationService(inventoryAddr, taskAddr string, logger *logrus.Logger) (*OrchestrationService, error) {
	inventoryClient, err := clients.NewInventoryClient(inventoryAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory client: %w", err)
	}

	taskClient, err := clients.NewTaskClient(taskAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create task client: %w", err)
	}

	return &OrchestrationService{
		inventoryClient: inventoryClient,
		taskClient:      taskClient,
		logger:          logger,
	}, nil
}

// Close closes all client connections
func (o *OrchestrationService) Close() error {
	var errors []error

	if err := o.inventoryClient.Close(); err != nil {
		errors = append(errors, fmt.Errorf("inventory client close error: %w", err))
	}

	if err := o.taskClient.Close(); err != nil {
		errors = append(errors, fmt.Errorf("task client close error: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("close errors: %v", errors)
	}

	return nil
}

// ProcessTaskCompletion handles task completion and potential inventory updates
func (o *OrchestrationService) ProcessTaskCompletion(ctx context.Context, taskID, userID string) error {
	o.logger.WithFields(logrus.Fields{
		"task_id": taskID,
		"user_id": userID,
	}).Info("processing task completion")

	// Get task details to understand if it affects inventory
	task, err := o.taskClient.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task details: %w", err)
	}

	// Complete the task
	completedTask, err := o.taskClient.CompleteTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	o.logger.WithFields(logrus.Fields{
		"task_id":   completedTask.Id,
		"task_name": completedTask.Name,
		"status":    completedTask.Status,
	}).Info("task completed successfully")

	// Check if this task has inventory implications
	if err := o.processInventoryImplications(ctx, task); err != nil {
		o.logger.WithError(err).Warn("failed to process inventory implications")
		// Don't fail the whole operation for inventory issues
	}

	return nil
}

// ProcessInventoryLevelChange handles inventory level changes and creates restocking tasks if needed
func (o *OrchestrationService) ProcessInventoryLevelChange(ctx context.Context, itemID string, previousLevel, newLevel float64) error {
	o.logger.WithFields(logrus.Fields{
		"item_id":        itemID,
		"previous_level": previousLevel,
		"new_level":      newLevel,
	}).Info("processing inventory level change")

	// Get current inventory status to check if restocking is needed
	status, err := o.inventoryClient.GetInventoryStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get inventory status: %w", err)
	}

	// Find items that are below threshold in the low stock list
	for _, item := range status.LowStockItems {
		if item.Id == itemID {
			o.logger.WithFields(logrus.Fields{
				"item_id":   item.Id,
				"item_name": item.Name,
				"level":     item.CurrentLevel,
				"threshold": item.LowStockThreshold,
			}).Info("item below threshold, creating restock task")

			// Create a restocking task
			taskName := fmt.Sprintf("Restock %s", item.Name)
			taskDesc := fmt.Sprintf("Current level: %.2f%s, threshold: %.2f%s",
				item.CurrentLevel, item.UnitId,
				item.LowStockThreshold, item.UnitId)

			_, err := o.taskClient.AddTask(ctx, taskName, taskDesc, "system")
			if err != nil {
				return fmt.Errorf("failed to create restock task: %w", err)
			}

			o.logger.WithField("task_name", taskName).Info("restock task created")
			break
		}
	}

	return nil
}

// processInventoryImplications checks if a completed task affects inventory levels
func (o *OrchestrationService) processInventoryImplications(ctx context.Context, task interface{}) error {
	// This is a placeholder for task-to-inventory mapping logic
	// In a real implementation, you might:
	// 1. Check task metadata for inventory item IDs
	// 2. Parse task description for consumption patterns
	// 3. Use predefined mappings between task types and inventory consumption

	o.logger.WithField("task_id", "unknown").Debug("checking inventory implications")

	// Example: If task involves cooking, it might consume ingredients
	// This would be implemented based on your specific business logic

	return nil
}
