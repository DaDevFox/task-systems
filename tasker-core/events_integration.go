package main

import (
	"context"
	"log"

	"github.com/DaDevFox/task-systems/shared/events/client"
	pb "github.com/DaDevFox/task-systems/tasker-core/backend/pkg/proto/taskcore/v1"
)

// TaskServiceWithEvents extends the task service with event publishing
type TaskServiceWithEvents struct {
	taskService *TaskService
	eventClient *client.EventClient
}

// NewTaskServiceWithEvents creates a new task service with event integration
func NewTaskServiceWithEvents(taskService *TaskService, eventsAddr string) (*TaskServiceWithEvents, error) {
	eventClient, err := client.NewEventClient(eventsAddr, "tasker-core")
	if err != nil {
		return nil, err
	}

	return &TaskServiceWithEvents{
		taskService: taskService,
		eventClient: eventClient,
	}, nil
}

// CreateTask creates a task and publishes an event
func (ts *TaskServiceWithEvents) CreateTask(ctx context.Context, req *pb.AddTaskRequest) (*pb.AddTaskResponse, error) {
	// Create the task using the existing service
	resp, err := ts.taskService.AddTask(ctx, req)
	if err != nil {
		return nil, err
	}

	// Publish task created event
	if resp.Task != nil {
		err = ts.eventClient.PublishTaskEvent(ctx, resp.Task.Id, resp.Task.Name, resp.Task.UserId, "", pb.EventType_TASK_CREATED)
		if err != nil {
			log.Printf("Failed to publish task created event: %v", err)
			// Don't fail the request if event publishing fails
		}
	}

	return resp, nil
}

// CompleteTask completes a task and publishes an event
func (ts *TaskServiceWithEvents) CompleteTask(ctx context.Context, req *pb.CompleteTaskRequest) (*pb.CompleteTaskResponse, error) {
	// Complete the task using the existing service
	resp, err := ts.taskService.CompleteTask(ctx, req)
	if err != nil {
		return nil, err
	}

	// Publish task completed event
	if resp.Task != nil {
		err = ts.eventClient.PublishTaskEvent(ctx, resp.Task.Id, resp.Task.Name, resp.Task.UserId, resp.Task.LocationPath, pb.EventType_TASK_COMPLETED)
		if err != nil {
			log.Printf("Failed to publish task completed event: %v", err)
			// Don't fail the request if event publishing fails
		}
	}

	return resp, nil
}

// StartEventSubscription starts listening for events from other services
func (ts *TaskServiceWithEvents) StartEventSubscription(ctx context.Context) error {
	// Subscribe to user events to update task assignments when users are created/updated
	go func() {
		handler := func(event *pb.Event) {
			log.Printf("Received user event: %s from %s", GetEventType(event), event.SourceService)

			// Handle user events (e.g., update task assignments, send notifications)
			switch GetEventType(event) {
			case pb.EventType_USER_CREATED:
				if userEvent := event.GetUserCreated(); userEvent != nil {
					log.Printf("New user created: %s (%s)", userEvent.FirstName, userEvent.Email)
					// Could assign default tasks to new users
				}
			case pb.EventType_USER_UPDATED:
				if userEvent := event.GetUserUpdated(); userEvent != nil {
					log.Printf("User updated: %s", userEvent.FirstName)
					// Could update task assignments for the user
				}
			}
		}

		err := ts.eventClient.SubscribeToUserEvents(ctx, handler)
		if err != nil {
			log.Printf("Failed to subscribe to user events: %v", err)
		}
	}()

	// Subscribe to inventory events to create tasks when stock is low
	go func() {
		handler := func(event *pb.Event) {
			log.Printf("Received inventory event: %s from %s", GetEventType(event), event.SourceService)

			// Handle inventory events (e.g., create restocking tasks)
			switch GetEventType(event) {
			case pb.EventType_INVENTORY_LOW_STOCK_ALERT:
				if inventoryEvent := event.GetInventoryLowStockAlert(); inventoryEvent != nil {
					log.Printf("Low stock alert for %s: %.2f %s remaining",
						inventoryEvent.ItemName, inventoryEvent.CurrentLevel, inventoryEvent.Unit)

					// Could create a task to restock the item
					// ts.createRestockTask(inventoryEvent.ItemId, inventoryEvent.ItemName)
				}
			}
		}

		err := ts.eventClient.SubscribeToInventoryEvents(ctx, handler)
		if err != nil {
			log.Printf("Failed to subscribe to inventory events: %v", err)
		}
	}()

	return nil
}

// Close closes the event client
func (ts *TaskServiceWithEvents) Close() error {
	return ts.eventClient.Close()
}

// getEventType determines the EventType from the oneof field
func GetEventType(event *pb.Event) pb.EventType {
	switch event.EventType.(type) {
	case *pb.Event_InventoryLevelChanged:
		return pb.EventType_INVENTORY_LEVEL_CHANGED
	case *pb.Event_InventoryLowStockAlert:
		return pb.EventType_INVENTORY_LOW_STOCK_ALERT
	case *pb.Event_InventoryConsumptionPredicted:
		return pb.EventType_INVENTORY_CONSUMPTION_PREDICTED
	case *pb.Event_InventoryItemRemoved:
		return pb.EventType_INVENTORY_ITEM_REMOVED
	case *pb.Event_TaskCreated:
		return pb.EventType_TASK_CREATED
	case *pb.Event_TaskStageChanged:
		return pb.EventType_TASK_STAGE_CHANGED
	case *pb.Event_TaskCompleted:
		return pb.EventType_TASK_COMPLETED
	case *pb.Event_TaskAssigned:
		return pb.EventType_TASK_ASSIGNED
	case *pb.Event_GroupTaskAssigned:
		return pb.EventType_GROUP_TASK_ASSIGNED
	case *pb.Event_ScheduleTrigger:
		return pb.EventType_SCHEDULE_TRIGGER
	case *pb.Event_PipelineWorkStarted:
		return pb.EventType_PIPELINE_WORK_STARTED
	case *pb.Event_PipelineWorkCompleted:
		return pb.EventType_PIPELINE_WORK_COMPLETED
	case *pb.Event_UserCreated:
		return pb.EventType_USER_CREATED
	case *pb.Event_UserUpdated:
		return pb.EventType_USER_UPDATED
	case *pb.Event_UserDeleted:
		return pb.EventType_USER_DELETED
	default:
		return pb.EventType_EVENT_TYPE_UNSPECIFIED
	}
}
