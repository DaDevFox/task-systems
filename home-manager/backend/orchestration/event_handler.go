package orchestration

import (
	"context"

	"github.com/sirupsen/logrus"

	eventspb "github.com/DaDevFox/task-systems/shared/proto/events/v1"
)

// EventHandler handles events from external services and triggers orchestration
type EventHandler struct {
	orchestrationService *OrchestrationService
	logger               *logrus.Logger
}

// NewEventHandler creates a new event handler
func NewEventHandler(orchestrationService *OrchestrationService, logger *logrus.Logger) *EventHandler {
	return &EventHandler{
		orchestrationService: orchestrationService,
		logger:               logger,
	}
}

// HandleEvent processes incoming events and triggers appropriate orchestration actions
func (h *EventHandler) HandleEvent(ctx context.Context, event *eventspb.Event) error {
	h.logger.WithFields(logrus.Fields{
		"event_id":       event.Id,
		"event_type":     event.Type.String(),
		"source_service": event.SourceService,
	}).Info("handling orchestration event")

	switch event.Type {
	case eventspb.EventType_INVENTORY_LEVEL_CHANGED:
		return h.handleInventoryLevelChanged(ctx, event)
	case eventspb.EventType_TASK_COMPLETED:
		return h.handleTaskCompleted(ctx, event)
	case eventspb.EventType_TASK_ASSIGNED:
		return h.handleTaskAssigned(ctx, event)
	case eventspb.EventType_SCHEDULE_TRIGGER:
		return h.handleScheduleTrigger(ctx, event)
	default:
		h.logger.WithField("event_type", event.Type.String()).Debug("ignoring unhandled event type")
		return nil
	}
}

// handleInventoryLevelChanged processes inventory level change events
func (h *EventHandler) handleInventoryLevelChanged(ctx context.Context, event *eventspb.Event) error {
	var inventoryEvent eventspb.InventoryLevelChangedEvent
	if err := event.Payload.UnmarshalTo(&inventoryEvent); err != nil {
		return err
	}

	h.logger.WithFields(logrus.Fields{
		"item_id":         inventoryEvent.ItemId,
		"item_name":       inventoryEvent.ItemName,
		"previous_level":  inventoryEvent.PreviousLevel,
		"new_level":       inventoryEvent.NewLevel,
		"below_threshold": inventoryEvent.BelowThreshold,
	}).Info("processing inventory level changed event")

	return h.orchestrationService.ProcessInventoryLevelChange(
		ctx,
		inventoryEvent.ItemId,
		inventoryEvent.PreviousLevel,
		inventoryEvent.NewLevel,
	)
}

// handleTaskCompleted processes task completion events
func (h *EventHandler) handleTaskCompleted(ctx context.Context, event *eventspb.Event) error {
	var taskEvent eventspb.TaskCompletedEvent
	if err := event.Payload.UnmarshalTo(&taskEvent); err != nil {
		return err
	}

	h.logger.WithFields(logrus.Fields{
		"task_id":       taskEvent.TaskId,
		"task_name":     taskEvent.TaskName,
		"user_id":       taskEvent.UserId,
		"location_path": taskEvent.LocationPath,
	}).Info("processing task completed event")

	return h.orchestrationService.ProcessTaskCompletion(
		ctx,
		taskEvent.TaskId,
		taskEvent.UserId,
	)
}

// handleTaskAssigned processes task assignment events
func (h *EventHandler) handleTaskAssigned(ctx context.Context, event *eventspb.Event) error {
	var taskEvent eventspb.TaskAssignedEvent
	if err := event.Payload.UnmarshalTo(&taskEvent); err != nil {
		return err
	}

	h.logger.WithFields(logrus.Fields{
		"task_id":     taskEvent.TaskId,
		"task_name":   taskEvent.TaskName,
		"user_id":     taskEvent.UserId,
		"assigned_by": taskEvent.AssignedBy,
	}).Info("processing task assigned event")

	// Add any orchestration logic for task assignments
	// For example, checking if required inventory items are available
	return nil
}

// handleScheduleTrigger processes schedule trigger events
func (h *EventHandler) handleScheduleTrigger(ctx context.Context, event *eventspb.Event) error {
	var scheduleEvent eventspb.ScheduleTriggerEvent
	if err := event.Payload.UnmarshalTo(&scheduleEvent); err != nil {
		return err
	}

	h.logger.WithFields(logrus.Fields{
		"trigger_id":   scheduleEvent.TriggerId,
		"trigger_name": scheduleEvent.TriggerName,
		"cron_expr":    scheduleEvent.CronExpression,
	}).Info("processing schedule trigger event")

	// Parse context for any orchestration instructions
	if contextData := scheduleEvent.Context; contextData != nil {
		// Handle different types of scheduled orchestration
		if action, exists := contextData["action"]; exists {
			switch action {
			case "inventory_check":
				return h.handleScheduledInventoryCheck(ctx, contextData)
			case "task_reminder":
				return h.handleScheduledTaskReminder(ctx, contextData)
			default:
				h.logger.WithField("action", action).Debug("unknown scheduled action")
			}
		}
	}

	return nil
}

// handleScheduledInventoryCheck performs scheduled inventory level checks
func (h *EventHandler) handleScheduledInventoryCheck(ctx context.Context, contextData map[string]string) error {
	h.logger.Info("performing scheduled inventory check")

	status, err := h.orchestrationService.inventoryClient.GetInventoryStatus(ctx)
	if err != nil {
		return err
	}

	// Log low stock items
	lowStockCount := len(status.LowStockItems)
	for _, item := range status.LowStockItems {
		h.logger.WithFields(logrus.Fields{
			"item_name": item.Name,
			"level":     item.CurrentLevel,
			"threshold": item.LowStockThreshold,
		}).Warn("item below threshold detected during scheduled check")
	}

	// Log empty items
	emptyCount := len(status.EmptyItems)
	for _, item := range status.EmptyItems {
		h.logger.WithFields(logrus.Fields{
			"item_name": item.Name,
			"level":     item.CurrentLevel,
		}).Error("empty item detected during scheduled check")
	}

	h.logger.WithFields(logrus.Fields{
		"low_stock_count": lowStockCount,
		"empty_count":     emptyCount,
		"total_items":     status.TotalItems,
	}).Info("scheduled inventory check completed")

	return nil
}

// handleScheduledTaskReminder handles scheduled task reminders
func (h *EventHandler) handleScheduledTaskReminder(ctx context.Context, contextData map[string]string) error {
	h.logger.Info("processing scheduled task reminder")

	// This could trigger notifications or create follow-up tasks
	// Implementation depends on your specific reminder logic

	return nil
}
