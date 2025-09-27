package events

import (
	"context"

	pb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
)

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	// PublishInventoryLevelChanged publishes an inventory level change event
	PublishInventoryLevelChanged(ctx context.Context, itemID, itemName string, previousLevel, newLevel float64, unit string, threshold float64) error

	// PublishInventoryItemRemoved publishes an inventory item removal event
	PublishInventoryItemRemoved(ctx context.Context, itemID, itemName string) error

	// PublishTaskCompleted publishes a task completion event
	PublishTaskCompleted(ctx context.Context, taskID, taskName, userID, locationPath string, completedPoints []string) error

	// PublishTaskAssigned publishes a task assignment event
	PublishTaskAssigned(ctx context.Context, taskID, taskName, userID, assignedBy, groupID string) error

	// PublishScheduleTrigger publishes a schedule trigger event
	PublishScheduleTrigger(ctx context.Context, triggerID, triggerName, cronExpr string, context map[string]string) error

	// PublishEvent publishes a generic event (for compatibility)
	PublishEvent(ctx context.Context, event *pb.Event) error
}

// EventSubscriber defines the interface for subscribing to events
type EventSubscriber interface {
	// Subscribe registers a handler for a specific event type (for in-memory eventbus compatibility)
	Subscribe(eventType pb.EventType, handler EventHandler)

	// SubscribeToEvents subscribes to events with filters (for client compatibility)
	SubscribeToEvents(ctx context.Context, eventTypes []pb.EventType, filters map[string]string, handler func(*pb.Event)) error
}

// EventService combines publisher and subscriber interfaces
type EventService interface {
	EventPublisher
	EventSubscriber
}
