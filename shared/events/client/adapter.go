package client

import (
	"context"

	pb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventClientAdapter adapts EventClient to implement EventPublisher interface
type EventClientAdapter struct {
	client *EventClient
}

// NewEventClientAdapter creates a new adapter for EventClient
func NewEventClientAdapter(client *EventClient) *EventClientAdapter {
	return &EventClientAdapter{client: client}
}

// PublishInventoryLevelChanged publishes an inventory level change event
func (a *EventClientAdapter) PublishInventoryLevelChanged(ctx context.Context, itemID, itemName string, previousLevel, newLevel float64, unit string, threshold float64) error {
	if a.client == nil {
		return nil // No-op if client is nil
	}

	event := &pb.Event{
		Id:            generateEventID(),
		SourceService: a.client.serviceName,
		Timestamp:     timestamppb.Now(),
		EventData: &pb.Event_InventoryLevelChanged{
			InventoryLevelChanged: &pb.InventoryLevelChangedEvent{
				ItemId:         itemID,
				ItemName:       itemName,
				PreviousLevel:  previousLevel,
				NewLevel:       newLevel,
				Unit:           unit,
				Threshold:      threshold,
				BelowThreshold: newLevel < threshold,
			},
		},
		Metadata: map[string]string{
			"item_id":   itemID,
			"item_name": itemName,
			"unit":      unit,
		},
	}

	return a.client.PublishEvent(ctx, event)
}

// PublishInventoryItemRemoved publishes an inventory item removal event
func (a *EventClientAdapter) PublishInventoryItemRemoved(ctx context.Context, itemID, itemName string) error {
	if a.client == nil {
		return nil // No-op if client is nil
	}

	event := &pb.Event{
		Id:            generateEventID(),
		SourceService: a.client.serviceName,
		Timestamp:     timestamppb.Now(),
		EventData: &pb.Event_InventoryItemRemoved{
			InventoryItemRemoved: &pb.InventoryItemRemovedEvent{
				ItemId:           itemID,
				ItemName:         itemName,
				RemovedByService: a.client.serviceName,
				RemovalTime:      timestamppb.Now(),
			},
		},
		Metadata: map[string]string{
			"item_id":   itemID,
			"item_name": itemName,
		},
	}

	return a.client.PublishEvent(ctx, event)
}

// PublishTaskCompleted publishes a task completion event
func (a *EventClientAdapter) PublishTaskCompleted(ctx context.Context, taskID, taskName, userID, locationPath string, completedPoints []string) error {
	if a.client == nil {
		return nil // No-op if client is nil
	}

	event := &pb.Event{
		Id:            generateEventID(),
		SourceService: a.client.serviceName,
		Timestamp:     timestamppb.Now(),
		EventData: &pb.Event_TaskCompleted{
			TaskCompleted: &pb.TaskCompletedEvent{
				TaskId:          taskID,
				TaskName:        taskName,
				UserId:          userID,
				LocationPath:    locationPath,
				CompletedPoints: completedPoints,
				CompletionTime:  timestamppb.Now(),
			},
		},
		Metadata: map[string]string{
			"task_id":       taskID,
			"task_name":     taskName,
			"user_id":       userID,
			"location_path": locationPath,
		},
	}

	return a.client.PublishEvent(ctx, event)
}

// PublishTaskAssigned publishes a task assignment event
func (a *EventClientAdapter) PublishTaskAssigned(ctx context.Context, taskID, taskName, userID, assignedBy, groupID string) error {
	if a.client == nil {
		return nil // No-op if client is nil
	}

	event := &pb.Event{
		Id:            generateEventID(),
		SourceService: a.client.serviceName,
		Timestamp:     timestamppb.Now(),
		EventData: &pb.Event_TaskAssigned{
			TaskAssigned: &pb.TaskAssignedEvent{
				TaskId:     taskID,
				TaskName:   taskName,
				UserId:     userID,
				AssignedBy: assignedBy,
				AssignedAt: timestamppb.Now(),
				GroupId:    groupID,
			},
		},
		Metadata: map[string]string{
			"task_id":     taskID,
			"task_name":   taskName,
			"user_id":     userID,
			"assigned_by": assignedBy,
			"group_id":    groupID,
		},
	}

	return a.client.PublishEvent(ctx, event)
}

// PublishScheduleTrigger publishes a schedule trigger event
func (a *EventClientAdapter) PublishScheduleTrigger(ctx context.Context, triggerID, triggerName, cronExpr string, contextMap map[string]string) error {
	if a.client == nil {
		return nil // No-op if client is nil
	}

	event := &pb.Event{
		Id:            generateEventID(),
		SourceService: a.client.serviceName,
		Timestamp:     timestamppb.Now(),
		EventData: &pb.Event_ScheduleTrigger{
			ScheduleTrigger: &pb.ScheduleTriggerEvent{
				TriggerId:      triggerID,
				TriggerName:    triggerName,
				CronExpression: cronExpr,
				Context:        contextMap,
			},
		},
		Metadata: map[string]string{
			"trigger_id":   triggerID,
			"trigger_name": triggerName,
		},
	}

	return a.client.PublishEvent(ctx, event)
}

// PublishEvent publishes a generic event
func (a *EventClientAdapter) PublishEvent(ctx context.Context, event *pb.Event) error {
	if a.client == nil {
		return nil // No-op if client is nil
	}
	return a.client.PublishEvent(ctx, event)
}

// Subscribe is a no-op for client adapter (not supported in client mode)
func (a *EventClientAdapter) Subscribe(eventType pb.EventType, handler func(context.Context, *pb.Event) error) {
	// No-op - client uses SubscribeToEvents instead
}

// SubscribeToEvents subscribes to events with filters
func (a *EventClientAdapter) SubscribeToEvents(ctx context.Context, eventTypes []pb.EventType, filters map[string]string, handler func(*pb.Event)) error {
	if a.client == nil {
		return nil // No-op if client is nil
	}
	return a.client.SubscribeToEvents(ctx, eventTypes, filters, handler)
}
