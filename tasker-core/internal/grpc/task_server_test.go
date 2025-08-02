package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	"github.com/DaDevFox/task-systems/task-core/internal/repository"
	"github.com/DaDevFox/task-systems/task-core/internal/service"
	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

func TestTaskServer(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T, *TaskServer)
	}{
		{"AddTask", testGRPCAddTask},
		{"AddTaskInvalidRequest", testGRPCAddTaskInvalidRequest},
		{"MoveToStaging", testGRPCMoveToStaging},
		{"StartTask", testGRPCStartTask},
		{"StopTask", testGRPCStopTask},
		{"CompleteTask", testGRPCCompleteTask},
		{"ListTasks", testGRPCListTasks},
		{"GetTask", testGRPCGetTask},
		{"UpdateTaskTags", testGRPCUpdateTaskTags},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewInMemoryTaskRepository()
			userRepo := repository.NewInMemoryUserRepository()
			taskService := service.NewTaskService(repo, 5, userRepo, nil, nil)

			// Create default user for tests
			ctx := context.Background()
			defaultUser := &domain.User{
				ID:    "default-user",
				Email: "default@example.com",
				Name:  "Default User",
			}
			userRepo.Create(ctx, defaultUser)

			server := NewTaskServer(taskService)
			tt.test(t, server)
		})
	}
}

func testGRPCAddTask(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	req := &pb.AddTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
	}

	resp, err := server.AddTask(ctx, req)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if resp.Task.Name != "Test Task" {
		t.Errorf("Expected name 'Test Task', got %s", resp.Task.Name)
	}
	if resp.Task.Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got %s", resp.Task.Description)
	}
	if resp.Task.Stage != pb.TaskStage_STAGE_INBOX {
		t.Errorf("Expected stage INBOX, got %s", resp.Task.Stage)
	}
}

func testGRPCAddTaskInvalidRequest(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	req := &pb.AddTaskRequest{
		Name:        "", // Empty name should fail
		Description: "Test Description",
	}

	_, err := server.AddTask(ctx, req)
	if err == nil {
		t.Error("Expected error for empty task name, got nil")
	}
}

func testGRPCMoveToStaging(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	// First create a task
	addReq := &pb.AddTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
	}

	addResp, err := server.AddTask(ctx, addReq)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Move to staging
	moveReq := &pb.MoveToStagingRequest{
		SourceId: addResp.Task.Id,
		Destination: &pb.MoveToStagingRequest_NewLocation{
			NewLocation: &pb.MoveToStagingRequest_NewLocationList{
				NewLocation: []string{"project", "backend"},
			},
		},
		Points: []*pb.Point{
			{Title: "work", Value: 5},
		},
	}

	moveResp, err := server.MoveToStaging(ctx, moveReq)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	if moveResp.Task.Stage != pb.TaskStage_STAGE_STAGING {
		t.Errorf("Expected stage STAGING, got %s", moveResp.Task.Stage)
	}
	if len(moveResp.Task.Location) != 2 {
		t.Errorf("Expected 2 location elements, got %d", len(moveResp.Task.Location))
	}
	if len(moveResp.Task.Points) != 1 {
		t.Errorf("Expected 1 point, got %d", len(moveResp.Task.Points))
	}
}

func testGRPCStartTask(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	// Create and move task to staging
	task := createAndMoveToStaging(t, server, ctx)

	// Start the task
	startReq := &pb.StartTaskRequest{
		Id: task.Id,
	}

	startResp, err := server.StartTask(ctx, startReq)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	if startResp.Task.Stage != pb.TaskStage_STAGE_ACTIVE {
		t.Errorf("Expected stage ACTIVE, got %s", startResp.Task.Stage)
	}
}

func testGRPCStopTask(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	// Create, move to staging, and start task
	task := createAndMoveToStaging(t, server, ctx)

	startReq := &pb.StartTaskRequest{Id: task.Id}
	startResp, err := server.StartTask(ctx, startReq)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Stop the task
	stopReq := &pb.StopTaskRequest{
		Id: startResp.Task.Id,
		PointsCompleted: []*pb.Point{
			{Title: "work", Value: 2}, // Partial completion
		},
	}

	stopResp, err := server.StopTask(ctx, stopReq)
	if err != nil {
		t.Fatalf("StopTask failed: %v", err)
	}

	if stopResp.Completed {
		t.Error("Task should not be completed with partial points")
	}
	if stopResp.Task.Stage != pb.TaskStage_STAGE_STAGING {
		t.Errorf("Expected stage STAGING after partial stop, got %s", stopResp.Task.Stage)
	}
}

func testGRPCCompleteTask(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	// Create, move to staging, and start task
	task := createAndMoveToStaging(t, server, ctx)

	startReq := &pb.StartTaskRequest{Id: task.Id}
	startResp, err := server.StartTask(ctx, startReq)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Complete the task
	completeReq := &pb.CompleteTaskRequest{
		Id: startResp.Task.Id,
	}

	completeResp, err := server.CompleteTask(ctx, completeReq)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	if completeResp.Task.Stage != pb.TaskStage_STAGE_ARCHIVED {
		t.Errorf("Expected stage ARCHIVED, got %s", completeResp.Task.Stage)
	}
}

func testGRPCListTasks(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	// Create multiple tasks
	for i := 0; i < 3; i++ {
		addReq := &pb.AddTaskRequest{
			Name:        fmt.Sprintf("Task %d", i+1),
			Description: fmt.Sprintf("Description %d", i+1),
		}
		_, err := server.AddTask(ctx, addReq)
		if err != nil {
			t.Fatalf("AddTask %d failed: %v", i+1, err)
		}
	}

	// List inbox tasks
	listReq := &pb.ListTasksRequest{
		Stage: pb.TaskStage_STAGE_INBOX,
	}

	listResp, err := server.ListTasks(ctx, listReq)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(listResp.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(listResp.Tasks))
	}
}

func testGRPCGetTask(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	// Create a task
	addReq := &pb.AddTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
	}

	addResp, err := server.AddTask(ctx, addReq)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Get the task
	getReq := &pb.GetTaskRequest{
		Id: addResp.Task.Id,
	}

	getResp, err := server.GetTask(ctx, getReq)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if getResp.Task.Id != addResp.Task.Id {
		t.Errorf("Expected ID %s, got %s", addResp.Task.Id, getResp.Task.Id)
	}
	if getResp.Task.Name != "Test Task" {
		t.Errorf("Expected name 'Test Task', got %s", getResp.Task.Name)
	}
}

func testGRPCUpdateTaskTags(t *testing.T, server *TaskServer) {
	ctx := context.Background()

	// Create a task first
	addReq := &pb.AddTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
	}

	addResp, err := server.AddTask(ctx, addReq)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Update tags
	updateReq := &pb.UpdateTaskTagsRequest{
		Id: addResp.Task.Id,
		Tags: map[string]*pb.TagValue{
			"priority": {
				Type:  pb.TagType_TAG_TYPE_TEXT,
				Value: &pb.TagValue_TextValue{TextValue: "high"},
			},
			"location": {
				Type: pb.TagType_TAG_TYPE_LOCATION,
				Value: &pb.TagValue_LocationValue{
					LocationValue: &pb.GeographicLocation{
						Latitude:  37.7749,
						Longitude: -122.4194,
						Address:   "San Francisco",
					},
				},
			},
		},
	}

	updateResp, err := server.UpdateTaskTags(ctx, updateReq)
	if err != nil {
		t.Fatalf("UpdateTaskTags failed: %v", err)
	}

	if len(updateResp.Task.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(updateResp.Task.Tags))
	}

	if updateResp.Task.Tags["priority"].GetTextValue() != "high" {
		t.Errorf("Expected priority tag 'high', got %s", updateResp.Task.Tags["priority"].GetTextValue())
	}
}

// Helper function to create and move a task to staging
func createAndMoveToStaging(t *testing.T, server *TaskServer, ctx context.Context) *pb.Task {
	// Create task
	addReq := &pb.AddTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
	}

	addResp, err := server.AddTask(ctx, addReq)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Move to staging
	moveReq := &pb.MoveToStagingRequest{
		SourceId: addResp.Task.Id,
		Destination: &pb.MoveToStagingRequest_NewLocation{
			NewLocation: &pb.MoveToStagingRequest_NewLocationList{
				NewLocation: []string{"project"},
			},
		},
		Points: []*pb.Point{
			{Title: "work", Value: 10},
		},
	}

	moveResp, err := server.MoveToStaging(ctx, moveReq)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	return moveResp.Task
}
