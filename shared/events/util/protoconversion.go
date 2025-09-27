package util

import (
	"fmt"

	pb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
)

func ValidateEvent(event *pb.Event) error {
	if !ValidEventType(event) {
		return fmt.Errorf("event type %s does not match event data type", event.Type.String())
	}

	return nil
}

func ValidEventType(event *pb.Event) bool {
	return GetEventType(event) != event.Type
}

// GetEventType determines the EventType from the oneof field
func GetEventType(event *pb.Event) pb.EventType {
	switch event.EventData.(type) {
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

