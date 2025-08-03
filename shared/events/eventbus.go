package events

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/DaDevFox/task-systems/shared/proto/events/v1"
)

// EventHandler defines the interface for handling events
type EventHandler func(ctx context.Context, event *pb.Event) error

// EventBus provides in-memory pub/sub functionality
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[pb.EventType][]EventHandler
	serviceName string
}

// NewEventBus creates a new event bus for a service
func NewEventBus(serviceName string) *EventBus {
	return &EventBus{
		subscribers: make(map[pb.EventType][]EventHandler),
		serviceName: serviceName,
	}
}

// Subscribe registers a handler for a specific event type
func (eb *EventBus) Subscribe(eventType pb.EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Publish sends an event to all registered handlers
func (eb *EventBus) Publish(ctx context.Context, eventType pb.EventType, payload proto.Message) error {
	eb.mu.RLock()
	handlers := eb.subscribers[eventType]
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		return nil // No subscribers, which is fine
	}

	// Convert payload to Any
	payloadAny, err := anypb.New(payload)
	if err != nil {
		return fmt.Errorf("failed to convert payload to Any: %w", err)
	}

	// Create event
	event := &pb.Event{
		Id:            uuid.New().String(),
		Type:          eventType,
		SourceService: eb.serviceName,
		Timestamp:     timestamppb.Now(),
		Payload:       payloadAny,
	}

	// Send to all handlers (asynchronously to avoid blocking)
	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h(ctx, event); err != nil {
				log.Printf("Event handler error: %v", err)
			}
		}(handler)
	}

	return nil
}

// PublishEvent publishes a pre-constructed event
func (eb *EventBus) PublishEvent(ctx context.Context, event *pb.Event) error {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	// Send to all handlers
	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h(ctx, event); err != nil {
				log.Printf("Event handler error: %v", err)
			}
		}(handler)
	}

	return nil
}

// Convenience methods for publishing common events

// PublishInventoryLevelChanged publishes an inventory level change event
func (eb *EventBus) PublishInventoryLevelChanged(ctx context.Context, itemID, itemName string, previousLevel, newLevel float64, unit string, threshold float64) error {
	event := &pb.InventoryLevelChangedEvent{
		ItemId:         itemID,
		ItemName:       itemName,
		PreviousLevel:  previousLevel,
		NewLevel:       newLevel,
		Unit:           unit,
		Threshold:      threshold,
		BelowThreshold: newLevel < threshold,
	}

	return eb.Publish(ctx, pb.EventType_INVENTORY_LEVEL_CHANGED, event)
}

// PublishTaskCompleted publishes a task completion event
func (eb *EventBus) PublishTaskCompleted(ctx context.Context, taskID, taskName, userID, locationPath string, completedPoints []string) error {
	event := &pb.TaskCompletedEvent{
		TaskId:          taskID,
		TaskName:        taskName,
		UserId:          userID,
		LocationPath:    locationPath,
		CompletedPoints: completedPoints,
		CompletionTime:  timestamppb.Now(),
	}

	return eb.Publish(ctx, pb.EventType_TASK_COMPLETED, event)
}

// PublishTaskAssigned publishes a task assignment event
func (eb *EventBus) PublishTaskAssigned(ctx context.Context, taskID, taskName, userID, assignedBy, groupID string) error {
	event := &pb.TaskAssignedEvent{
		TaskId:     taskID,
		TaskName:   taskName,
		UserId:     userID,
		AssignedBy: assignedBy,
		AssignedAt: timestamppb.Now(),
		GroupId:    groupID,
	}

	return eb.Publish(ctx, pb.EventType_TASK_ASSIGNED, event)
}

// PublishScheduleTrigger publishes a schedule trigger event
func (eb *EventBus) PublishScheduleTrigger(ctx context.Context, triggerID, triggerName, cronExpr string, context map[string]string) error {
	event := &pb.ScheduleTriggerEvent{
		TriggerId:      triggerID,
		TriggerName:    triggerName,
		CronExpression: cronExpr,
		Context:        context,
	}

	return eb.Publish(ctx, pb.EventType_SCHEDULE_TRIGGER, event)
}

// EventBusManager manages multiple event buses for inter-service communication
type EventBusManager struct {
	buses map[string]*EventBus
	mu    sync.RWMutex
}

// NewEventBusManager creates a new event bus manager
func NewEventBusManager() *EventBusManager {
	return &EventBusManager{
		buses: make(map[string]*EventBus),
	}
}

// GetBus returns or creates an event bus for a service
func (ebm *EventBusManager) GetBus(serviceName string) *EventBus {
	ebm.mu.RLock()
	bus, exists := ebm.buses[serviceName]
	ebm.mu.RUnlock()

	if exists {
		return bus
	}

	ebm.mu.Lock()
	defer ebm.mu.Unlock()

	// Double-check in case another goroutine created it
	if bus, exists := ebm.buses[serviceName]; exists {
		return bus
	}

	bus = NewEventBus(serviceName)
	ebm.buses[serviceName] = bus
	return bus
}

// BroadcastEvent sends an event to all registered buses
func (ebm *EventBusManager) BroadcastEvent(ctx context.Context, event *pb.Event) error {
	ebm.mu.RLock()
	defer ebm.mu.RUnlock()

	for _, bus := range ebm.buses {
		if err := bus.PublishEvent(ctx, event); err != nil {
			return fmt.Errorf("failed to broadcast to bus: %w", err)
		}
	}

	return nil
}

// Global event bus manager instance
var globalManager = NewEventBusManager()

// GetGlobalBus returns the global event bus for a service
func GetGlobalBus(serviceName string) *EventBus {
	return globalManager.GetBus(serviceName)
}

// BroadcastGlobally broadcasts an event to all services
func BroadcastGlobally(ctx context.Context, event *pb.Event) error {
	return globalManager.BroadcastEvent(ctx, event)
}
