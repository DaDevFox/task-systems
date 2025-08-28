package client

import (
	"context"
	"fmt"
	"time"

	pb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventClient provides a client for interacting with the events service
type EventClient struct {
	conn        *grpc.ClientConn
	client      pb.EventServiceClient
	serviceName string
}

// NewEventClient creates a new event client
func NewEventClient(target string, serviceName string) (*EventClient, error) {
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to events service: %w", err)
	}

	client := pb.NewEventServiceClient(conn)

	return &EventClient{
		conn:        conn,
		client:      client,
		serviceName: serviceName,
	}, nil
}

// Close closes the client connection
func (c *EventClient) Close() error {
	return c.conn.Close()
}

// PublishEvent publishes an event to the events service
func (c *EventClient) PublishEvent(ctx context.Context, event *pb.Event) error {
	req := &pb.PublishEventRequest{
		Event: event,
	}

	resp, err := c.client.PublishEvent(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to publish event: %s", resp.Message)
	}

	return nil
}

// PublishTaskEvent publishes a task-related event
func (c *EventClient) PublishTaskEvent(ctx context.Context, taskID, taskName, userID, locationPath string, eventType pb.EventType) error {
	event := &pb.Event{
		Id:            generateEventID(),
		SourceService: c.serviceName,
		Timestamp:     timestamppb.Now(),
		Metadata: map[string]string{
			"task_id":       taskID,
			"task_name":     taskName,
			"user_id":       userID,
			"location_path": locationPath,
		},
	}

	// Set the appropriate event type based on the oneof
	switch eventType {
	case pb.EventType_TASK_CREATED:
		event.EventData = &pb.Event_TaskCreated{
			TaskCreated: &pb.TaskCreatedEvent{
				TaskId:    taskID,
				TaskName:  taskName,
				CreatedBy: userID,
				CreatedAt: timestamppb.Now(),
			},
		}
	case pb.EventType_TASK_COMPLETED:
		event.EventData = &pb.Event_TaskCompleted{
			TaskCompleted: &pb.TaskCompletedEvent{
				TaskId:         taskID,
				TaskName:       taskName,
				UserId:         userID,
				LocationPath:   locationPath,
				CompletionTime: timestamppb.Now(),
			},
		}
	case pb.EventType_TASK_ASSIGNED:
		event.EventData = &pb.Event_TaskAssigned{
			TaskAssigned: &pb.TaskAssignedEvent{
				TaskId:     taskID,
				TaskName:   taskName,
				UserId:     userID,
				AssignedAt: timestamppb.Now(),
			},
		}
	}

	return c.PublishEvent(ctx, event)
}

// SubscribeToEvents subscribes to events with the given filters
func (c *EventClient) SubscribeToEvents(ctx context.Context, eventTypes []pb.EventType, filters map[string]string, handler func(*pb.Event)) error {
	req := &pb.SubscribeToEventsRequest{
		EventTypes: eventTypes,
		Filters:    filters,
	}

	stream, err := c.client.SubscribeToEvents(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Handle incoming events
	for {
		resp, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("error receiving event: %w", err)
		}

		// Call the handler with the event
		handler(resp.Event)
	}
}

// SubscribeToTaskEvents subscribes to task-related events
func (c *EventClient) SubscribeToTaskEvents(ctx context.Context, handler func(*pb.Event)) error {
	eventTypes := []pb.EventType{
		pb.EventType_TASK_CREATED,
		pb.EventType_TASK_COMPLETED,
		pb.EventType_TASK_ASSIGNED,
		pb.EventType_TASK_STAGE_CHANGED,
	}

	return c.SubscribeToEvents(ctx, eventTypes, nil, handler)
}

// SubscribeToUserEvents subscribes to user-related events
func (c *EventClient) SubscribeToUserEvents(ctx context.Context, handler func(*pb.Event)) error {
	eventTypes := []pb.EventType{
		pb.EventType_USER_CREATED,
		pb.EventType_USER_UPDATED,
		pb.EventType_USER_DELETED,
	}

	return c.SubscribeToEvents(ctx, eventTypes, nil, handler)
}

// SubscribeToInventoryEvents subscribes to inventory-related events
func (c *EventClient) SubscribeToInventoryEvents(ctx context.Context, handler func(*pb.Event)) error {
	eventTypes := []pb.EventType{
		pb.EventType_INVENTORY_LEVEL_CHANGED,
		pb.EventType_INVENTORY_LOW_STOCK_ALERT,
		pb.EventType_INVENTORY_CONSUMPTION_PREDICTED,
		pb.EventType_INVENTORY_ITEM_REMOVED,
	}

	return c.SubscribeToEvents(ctx, eventTypes, nil, handler)
}

// PublishUserEvent publishes a user-related event
func (c *EventClient) PublishUserEvent(ctx context.Context, userID, firstName, lastName, email string, eventType pb.EventType) error {
	event := &pb.Event{
		Id:            generateEventID(),
		SourceService: c.serviceName,
		Timestamp:     timestamppb.Now(),
		Metadata: map[string]string{
			"user_id":    userID,
			"first_name": firstName,
			"last_name":  lastName,
			"email":      email,
		},
	}

	// Set the appropriate event type based on the oneof
	switch eventType {
	case pb.EventType_USER_CREATED:
		event.EventData = &pb.Event_UserCreated{
			UserCreated: &pb.UserCreatedEvent{
				UserId:    userID,
				FirstName: firstName,
				LastName:  lastName,
				Email:     email,
				CreatedAt: timestamppb.Now(),
			},
		}
	case pb.EventType_USER_UPDATED:
		event.EventData = &pb.Event_UserUpdated{
			UserUpdated: &pb.UserUpdatedEvent{
				UserId:    userID,
				FirstName: firstName,
				LastName:  lastName,
				Email:     email,
				UpdatedAt: timestamppb.Now(),
			},
		}
	case pb.EventType_USER_DELETED:
		event.EventData = &pb.Event_UserDeleted{
			UserDeleted: &pb.UserDeletedEvent{
				UserId:    userID,
				DeletedAt: timestamppb.Now(),
			},
		}
	}

	return c.PublishEvent(ctx, event)
}

// PublishInventoryEvent publishes an inventory-related event
func (c *EventClient) PublishInventoryEvent(ctx context.Context, itemID, itemName string, currentLevel float64, unit string, eventType pb.EventType) error {
	event := &pb.Event{
		Id:            generateEventID(),
		SourceService: c.serviceName,
		Timestamp:     timestamppb.Now(),
		Metadata: map[string]string{
			"item_id":   itemID,
			"item_name": itemName,
			"unit":      unit,
		},
	}

	// Set the appropriate event type based on the oneof
	switch eventType {
	case pb.EventType_INVENTORY_LEVEL_CHANGED:
		event.EventData = &pb.Event_InventoryLevelChanged{
			InventoryLevelChanged: &pb.InventoryLevelChangedEvent{
				ItemId:   itemID,
				ItemName: itemName,
				NewLevel: currentLevel,
				Unit:     unit,
			},
		}
	case pb.EventType_INVENTORY_LOW_STOCK_ALERT:
		event.EventData = &pb.Event_InventoryLowStockAlert{
			InventoryLowStockAlert: &pb.InventoryLowStockAlertEvent{
				ItemId:       itemID,
				ItemName:     itemName,
				CurrentLevel: currentLevel,
				Unit:         unit,
			},
		}
	}

	return c.PublishEvent(ctx, event)
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
}

// EventHandler is a function type for handling events
type EventHandler func(event *pb.Event)

// EventSubscriber provides a convenient way to subscribe to events
type EventSubscriber struct {
	client   *EventClient
	ctx      context.Context
	cancel   context.CancelFunc
	handlers []EventHandler
}

// NewEventSubscriber creates a new event subscriber
func NewEventSubscriber(client *EventClient) *EventSubscriber {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventSubscriber{
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddHandler adds an event handler
func (es *EventSubscriber) AddHandler(handler EventHandler) {
	es.handlers = append(es.handlers, handler)
}

// Start starts the event subscription with the given filters
func (es *EventSubscriber) Start(eventTypes []pb.EventType, filters map[string]string) error {
	handler := func(event *pb.Event) {
		for _, h := range es.handlers {
			h(event)
		}
	}

	return es.client.SubscribeToEvents(es.ctx, eventTypes, filters, handler)
}

// Stop stops the event subscription
func (es *EventSubscriber) Stop() {
	es.cancel()
}
