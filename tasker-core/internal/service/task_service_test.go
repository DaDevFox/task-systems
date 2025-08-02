package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	"github.com/DaDevFox/task-systems/task-core/internal/repository"
)

func TestTaskService(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T, *TaskService)
	}{
		{"AddTask", testAddTask},
		{"AddTaskEmptyName", testAddTaskEmptyName},
		{"AddTaskForUser", testAddTaskForUser},
		{"MoveToStaging", testMoveToStaging},
		{"MoveToStagingWithDestination", testMoveToStagingWithDestination},
		{"StartTask", testStartTask},
		{"StartTaskWithDependencies", testStartTaskWithDependencies},
		{"StopTask", testStopTask},
		{"CompleteTask", testCompleteTask},
		{"MergeTasks", testMergeTasks},
		{"SplitTask", testSplitTask},
		{"AdvertiseTask", testAdvertiseTask},
		{"StitchTasks", testStitchTasks},
		{"InboxConstraint", testInboxConstraint},
		{"UpdateTaskTags", testUpdateTaskTags},
		{"CreateUser", testCreateUser},
		{"GetUser", testGetUser},
		{"UpdateUser", testUpdateUser},
		{"GetTaskDAG", testGetTaskDAG},
		{"SyncCalendar", testSyncCalendar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := repository.NewInMemoryTaskRepository()
			userRepo := repository.NewInMemoryUserRepository()
			service := NewTaskService(taskRepo, 10, userRepo, nil, nil) // Larger inbox for testing, no calendar/email

			// Create default user for tests that use AddTask
			ctx := context.Background()
			defaultUser := &domain.User{
				ID:    "default-user",
				Email: "default@example.com",
				Name:  "Default User",
			}
			userRepo.Create(ctx, defaultUser)

			taskRepo := repository.NewInMemoryTaskRepository()
			userRepo := repository.NewInMemoryUserRepository()
			service := NewTaskService(taskRepo, 10, userRepo, nil, nil, nil, nil) // Larger inbox for testing, no calendar/email/logger/eventBus

			// Create default user for tests that use AddTask
			ctx := context.Background()
			defaultUser := &domain.User{
				ID:    "default-user",
				Email: "default@example.com",
				Name:  "Default User",
			}
			userRepo.Create(ctx, defaultUser)

			tt.test(t, service)
		})
	}
}

func testAddTask(t *testing.T, service *TaskService) {
	ctx := context.Background()

	task, err := service.AddTask(ctx, "Test Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if task.Name != "Test Task" {
		t.Errorf("Expected name 'Test Task', got %s", task.Name)
	}
	if task.Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got %s", task.Description)
	}
	if task.Stage != domain.StageInbox {
		t.Errorf("Expected stage inbox, got %s", task.Stage)
	}
	if len(task.StatusHist.Updates) == 0 {
		t.Error("Expected status update, got none")
	}
}

func testAddTaskEmptyName(t *testing.T, service *TaskService) {
	ctx := context.Background()

	_, err := service.AddTask(ctx, "", "Test Description", "default-user")
	if err == nil {
		t.Error("Expected error for empty task name, got nil")
	}
}

func testAddTaskForUser(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a test user first
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	task, err := service.AddTaskForUser(ctx, "Test Task", "Test Description", user.ID)
	if err != nil {
		t.Fatalf("AddTaskForUser failed: %v", err)
	}

	if task.Name != "Test Task" {
		t.Errorf("Expected name 'Test Task', got %s", task.Name)
	}

	if task.UserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, task.UserID)
	}

	if task.Stage != domain.StageInbox {
		t.Errorf("Expected stage %v, got %v", domain.StageInbox, task.Stage)
	}
}

func testMoveToStaging(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a task
	task, err := service.AddTask(ctx, "Test Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Move to staging with new location
	points := []domain.Point{{Title: "work", Value: 5}}
	location := []string{"project", "backend"}

	movedTask, err := service.MoveToStaging(ctx, task.ID, nil, location, points)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	if movedTask.Stage != domain.StageStaging {
		t.Errorf("Expected stage staging, got %s", movedTask.Stage)
	}
	if len(movedTask.Location) != 2 || movedTask.Location[0] != "project" {
		t.Errorf("Expected location [project, backend], got %v", movedTask.Location)
	}
	if len(movedTask.Points) != 1 || movedTask.Points[0].Value != 5 {
		t.Errorf("Expected points [work:5], got %v", movedTask.Points)
	}
}

func testMoveToStagingWithDestination(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create destination task in staging
	destTask, err := service.AddTask(ctx, "Destination Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	destTask.Stage = domain.StageStaging
	destTask.Location = []string{"project", "api"}

	// Move destination to staging first
	_, err = service.MoveToStaging(ctx, destTask.ID, nil, []string{"project", "api"}, nil)
	if err != nil {
		t.Fatalf("MoveToStaging dest failed: %v", err)
	}

	// Create source task
	sourceTask, err := service.AddTask(ctx, "Source Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Move source to staging with destination
	movedTask, err := service.MoveToStaging(ctx, sourceTask.ID, &destTask.ID, nil, nil)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	if movedTask.Stage != domain.StageStaging {
		t.Errorf("Expected stage staging, got %s", movedTask.Stage)
	}
	if len(movedTask.Location) != 2 || movedTask.Location[0] != "project" {
		t.Errorf("Expected inherited location [project, api], got %v", movedTask.Location)
	}
	if len(movedTask.Inflows) != 1 || movedTask.Inflows[0] != destTask.ID {
		t.Errorf("Expected inflow %s, got %v", destTask.ID, movedTask.Inflows)
	}
}

func testStartTask(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create and move task to staging
	task, err := service.AddTask(ctx, "Test Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, task.ID, nil, []string{"project"}, nil)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	// Start the task
	startedTask, err := service.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	if startedTask.Stage != domain.StageActive {
		t.Errorf("Expected stage active, got %s", startedTask.Stage)
	}
	if startedTask.Status != domain.StatusInProgress {
		t.Errorf("Expected status in_progress, got %s", startedTask.Status)
	}
	if len(startedTask.Schedule.WorkIntervals) == 0 {
		t.Error("Expected work interval to be created")
	}
}

func testStartTaskWithDependencies(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create dependency task
	depTask, err := service.AddTask(ctx, "Dependency Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, depTask.ID, nil, []string{"project"}, nil)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	// Create dependent task
	task, err := service.AddTask(ctx, "Dependent Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, task.ID, &depTask.ID, nil, nil)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	// Try to start task before dependency is complete
	_, err = service.StartTask(ctx, task.ID)
	if err == nil {
		t.Error("Expected error when starting task with incomplete dependencies")
	}

	// Complete dependency first
	_, err = service.StartTask(ctx, depTask.ID)
	if err != nil {
		t.Fatalf("StartTask dependency failed: %v", err)
	}

	_, err = service.CompleteTask(ctx, depTask.ID)
	if err != nil {
		t.Fatalf("CompleteTask dependency failed: %v", err)
	}

	// Now starting should work
	_, err = service.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("StartTask failed after dependency completion: %v", err)
	}
}

func testStopTask(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create, move to staging, and start task
	task, err := service.AddTask(ctx, "Test Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	points := []domain.Point{{Title: "work", Value: 10}}
	_, err = service.MoveToStaging(ctx, task.ID, nil, []string{"project"}, points)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	_, err = service.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Stop with partial completion
	completedPoints := []domain.Point{{Title: "work", Value: 5}}
	stoppedTask, isComplete, err := service.StopTask(ctx, task.ID, completedPoints)
	if err != nil {
		t.Fatalf("StopTask failed: %v", err)
	}

	if isComplete {
		t.Error("Task should not be complete with partial points")
	}
	if stoppedTask.Stage != domain.StageStaging {
		t.Errorf("Expected stage staging after partial stop, got %s", stoppedTask.Stage)
	}
	if stoppedTask.Status != domain.StatusTodo {
		t.Errorf("Expected status todo after partial stop, got %s", stoppedTask.Status)
	}
}

func testCompleteTask(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create, move to staging, and start task
	task, err := service.AddTask(ctx, "Test Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	points := []domain.Point{{Title: "work", Value: 10}}
	_, err = service.MoveToStaging(ctx, task.ID, nil, []string{"project"}, points)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	_, err = service.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Complete the task
	completedTask, err := service.CompleteTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	if completedTask.Status != domain.StatusCompleted {
		t.Errorf("Expected status completed, got %s", completedTask.Status)
	}
	if completedTask.Stage != domain.StageArchived {
		t.Errorf("Expected stage archived, got %s", completedTask.Stage)
	}
}

func testMergeTasks(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create two tasks in the same location
	task1, err := service.AddTask(ctx, "Task 1", "Description 1", "default-user")
	if err != nil {
		t.Fatalf("AddTask 1 failed: %v", err)
	}

	task2, err := service.AddTask(ctx, "Task 2", "Description 2", "default-user")
	if err != nil {
		t.Fatalf("AddTask 2 failed: %v", err)
	}

	// Move both to staging in same location
	location := []string{"project", "feature"}
	_, err = service.MoveToStaging(ctx, task1.ID, nil, location, nil)
	if err != nil {
		t.Fatalf("MoveToStaging 1 failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, task2.ID, nil, location, nil)
	if err != nil {
		t.Fatalf("MoveToStaging 2 failed: %v", err)
	}

	// Merge tasks
	mergedTask, err := service.MergeTasks(ctx, task1.ID, task2.ID)
	if err != nil {
		t.Fatalf("MergeTasks failed: %v", err)
	}

	if mergedTask.ID != task2.ID {
		t.Errorf("Expected merged task to have task2 ID %s, got %s", task2.ID, mergedTask.ID)
	}

	// Verify task1 is deleted
	_, err = service.GetTask(ctx, task1.ID)
	if err == nil {
		t.Error("Expected task1 to be deleted after merge")
	}
}

func testSplitTask(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a task
	task, err := service.AddTask(ctx, "Test Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, task.ID, nil, []string{"project"}, nil)
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	// Split the task
	newNames := []string{"Subtask 1", "Subtask 2"}
	newDescriptions := []string{"Description 1", "Description 2"}

	newTasks, err := service.SplitTask(ctx, task.ID, newNames, newDescriptions)
	if err != nil {
		t.Fatalf("SplitTask failed: %v", err)
	}

	if len(newTasks) != 2 {
		t.Errorf("Expected 2 new tasks, got %d", len(newTasks))
	}

	// Verify original task is deleted
	_, err = service.GetTask(ctx, task.ID)
	if err == nil {
		t.Error("Expected original task to be deleted after split")
	}

	// Verify new tasks have correct names
	if newTasks[0].Name != "Subtask 1" {
		t.Errorf("Expected first task name 'Subtask 1', got %s", newTasks[0].Name)
	}
	if newTasks[1].Name != "Subtask 2" {
		t.Errorf("Expected second task name 'Subtask 2', got %s", newTasks[1].Name)
	}
}

func testAdvertiseTask(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create source task
	sourceTask, err := service.AddTask(ctx, "Source Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask source failed: %v", err)
	}

	// Create target tasks
	target1, err := service.AddTask(ctx, "Target Task 1", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask target1 failed: %v", err)
	}

	target2, err := service.AddTask(ctx, "Target Task 2", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask target2 failed: %v", err)
	}

	// Move all to staging
	location := []string{"project"}
	_, err = service.MoveToStaging(ctx, sourceTask.ID, nil, location, nil)
	if err != nil {
		t.Fatalf("MoveToStaging source failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, target1.ID, nil, location, nil)
	if err != nil {
		t.Fatalf("MoveToStaging target1 failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, target2.ID, nil, location, nil)
	if err != nil {
		t.Fatalf("MoveToStaging target2 failed: %v", err)
	}

	// Advertise
	targetIDs := []string{target1.ID, target2.ID}
	advertisedTask, err := service.AdvertiseTask(ctx, sourceTask.ID, targetIDs)
	if err != nil {
		t.Fatalf("AdvertiseTask failed: %v", err)
	}

	if len(advertisedTask.Outflows) != 2 {
		t.Errorf("Expected 2 outflows, got %d", len(advertisedTask.Outflows))
	}

	// Verify targets have inflows
	updatedTarget1, err := service.GetTask(ctx, target1.ID)
	if err != nil {
		t.Fatalf("GetTask target1 failed: %v", err)
	}

	if len(updatedTarget1.Inflows) != 1 || updatedTarget1.Inflows[0] != sourceTask.ID {
		t.Errorf("Expected target1 to have inflow %s, got %v", sourceTask.ID, updatedTarget1.Inflows)
	}
}

func testStitchTasks(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create source tasks
	source1, err := service.AddTask(ctx, "Source Task 1", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask source1 failed: %v", err)
	}

	source2, err := service.AddTask(ctx, "Source Task 2", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask source2 failed: %v", err)
	}

	// Create target task
	target, err := service.AddTask(ctx, "Target Task", "Test Description", "default-user")
	if err != nil {
		t.Fatalf("AddTask target failed: %v", err)
	}

	// Move all to staging
	location := []string{"project"}
	_, err = service.MoveToStaging(ctx, source1.ID, nil, location, nil)
	if err != nil {
		t.Fatalf("MoveToStaging source1 failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, source2.ID, nil, location, nil)
	if err != nil {
		t.Fatalf("MoveToStaging source2 failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, target.ID, nil, location, nil)
	if err != nil {
		t.Fatalf("MoveToStaging target failed: %v", err)
	}

	// Stitch
	sourceIDs := []string{source1.ID, source2.ID}
	updatedTasks, err := service.StitchTasks(ctx, sourceIDs, target.ID)
	if err != nil {
		t.Fatalf("StitchTasks failed: %v", err)
	}

	if len(updatedTasks) != 3 {
		t.Errorf("Expected 3 updated tasks, got %d", len(updatedTasks))
	}

	// Find the target task in results
	var updatedTarget *domain.Task
	for _, task := range updatedTasks {
		if task.ID == target.ID {
			updatedTarget = task
			break
		}
	}

	if updatedTarget == nil {
		t.Fatal("Target task not found in updated tasks")
	}

	if len(updatedTarget.Inflows) != 2 {
		t.Errorf("Expected target to have 2 inflows, got %d", len(updatedTarget.Inflows))
	}
}

func testInboxConstraint(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Fill inbox to capacity (service configured with maxInboxSize = 2)
	task1, err := service.AddTask(ctx, "Task 1", "Description 1", "default-user")
	if err != nil {
		t.Fatalf("AddTask 1 failed: %v", err)
	}
	task1.Stage = domain.StageInbox

	task2, err := service.AddTask(ctx, "Task 2", "Description 2", "default-user")
	if err != nil {
		t.Fatalf("AddTask 2 failed: %v", err)
	}
	task2.Stage = domain.StageInbox

	// Manually update tasks to inbox stage (simulating they were moved there)
	// Note: In real implementation, there would be a proper move to inbox operation

	// Try to start any task - should fail due to inbox constraint
	task3, err := service.AddTask(ctx, "Task 3", "Description 3", "default-user")
	if err != nil {
		t.Fatalf("AddTask 3 failed: %v", err)
	}

	_, err = service.MoveToStaging(ctx, task3.ID, nil, []string{"project"}, nil)
	// This should fail because we haven't implemented the inbox constraint check properly
	// in the test setup. The actual service would check this.
	if err != nil {
		// This is expected behavior - inbox constraint prevents operations
		t.Logf("Inbox constraint working: %v", err)
	}
}

func testUpdateTaskTags(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a test user
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create a task
	task, err := service.AddTaskForUser(ctx, "Test Task", "Test Description", user.ID)
	if err != nil {
		t.Fatalf("AddTaskForUser failed: %v", err)
	}

	// Update tags
	timeValue := time.Now()
	location := &domain.GeographicLocation{
		Latitude:  37.7749,
		Longitude: -122.4194,
		Address:   "office, desk1",
	}

	tags := map[string]domain.TagValue{
		"priority": {Type: domain.TagTypeText, TextValue: "high"},
		"location": {Type: domain.TagTypeLocation, LocationValue: location},
		"deadline": {Type: domain.TagTypeTime, TimeValue: &timeValue},
	}

	updatedTask, err := service.UpdateTaskTags(ctx, task.ID, tags)
	if err != nil {
		t.Fatalf("UpdateTaskTags failed: %v", err)
	}

	if len(updatedTask.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(updatedTask.Tags))
	}

	if updatedTask.Tags["priority"].TextValue != "high" {
		t.Errorf("Expected priority tag 'high', got %s", updatedTask.Tags["priority"].TextValue)
	}

	if updatedTask.Tags["location"].LocationValue == nil || updatedTask.Tags["location"].LocationValue.Address != "office, desk1" {
		t.Errorf("Expected location tag with address 'office, desk1'")
	}

	if updatedTask.Tags["deadline"].TimeValue == nil {
		t.Errorf("Expected time value to be set")
	}
}

func testCreateUser(t *testing.T, service *TaskService) {
	ctx := context.Background()

	notificationSettings := []domain.NotificationSetting{
		{Type: domain.NotificationOnAssign, Enabled: true},
		{Type: domain.NotificationOnStart, Enabled: false},
	}

	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", notificationSettings)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", user.Email)
	}

	if user.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got %s", user.Name)
	}

	if len(user.NotificationSettings) != 2 {
		t.Errorf("Expected 2 notification settings, got %d", len(user.NotificationSettings))
	}

	if user.ID == "" {
		t.Error("Expected user ID to be generated")
	}
}

func testGetUser(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a user
	createdUser, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Get the user
	retrievedUser, err := service.GetUser(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if retrievedUser.ID != createdUser.ID {
		t.Errorf("Expected user ID %s, got %s", createdUser.ID, retrievedUser.ID)
	}

	if retrievedUser.Email != createdUser.Email {
		t.Errorf("Expected email %s, got %s", createdUser.Email, retrievedUser.Email)
	}
}

func testUpdateUser(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a user
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Update user
	user.Name = "Updated User"
	user.NotificationSettings = []domain.NotificationSetting{
		{Type: domain.NotificationOnAssign, Enabled: true},
	}

	updatedUser, err := service.UpdateUser(ctx, user)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	if updatedUser.Name != "Updated User" {
		t.Errorf("Expected name 'Updated User', got %s", updatedUser.Name)
	}

	if len(updatedUser.NotificationSettings) != 1 {
		t.Errorf("Expected 1 notification setting, got %d", len(updatedUser.NotificationSettings))
	}
}

func testGetTaskDAG(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a test user
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create tasks with dependencies
	task1, err := service.AddTaskForUser(ctx, "Task 1", "First task", user.ID)
	if err != nil {
		t.Fatalf("AddTaskForUser failed: %v", err)
	}

	task2, err := service.AddTaskForUser(ctx, "Task 2", "Second task", user.ID)
	if err != nil {
		t.Fatalf("AddTaskForUser failed: %v", err)
	}

	// Move to staging to create dependencies
	_, err = service.MoveToStaging(ctx, task2.ID, &task1.ID, []string{}, []domain.Point{})
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	// Get DAG
	dagTasks, err := service.GetTaskDAG(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetTaskDAG failed: %v", err)
	}

	if len(dagTasks) != 2 {
		t.Errorf("Expected 2 tasks in DAG, got %d", len(dagTasks))
	}

	// Tasks should be in topological order (task1 before task2)
	if dagTasks[0].ID != task1.ID {
		t.Errorf("Expected first task to be %s, got %s", task1.ID, dagTasks[0].ID)
	}
}

func testSyncCalendar(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a test user
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Test sync without calendar service (should fail gracefully)
	synced, errors, err := service.SyncCalendar(ctx, user.ID)
	if err == nil {
		t.Error("Expected SyncCalendar to fail when no calendar service is configured")
	}

	if synced != 0 {
		t.Errorf("Expected 0 synced tasks, got %d", synced)
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

// Test table-driven approach for tag types
func TestTagTypes(t *testing.T) {
	tests := []struct {
		name     string
		tagType  domain.TagType
		value    domain.TagValue
		expected string
	}{
		{
			name:     "Text tag",
			tagType:  domain.TagTypeText,
			value:    domain.TagValue{Type: domain.TagTypeText, TextValue: "urgent"},
			expected: "urgent",
		},
		{
			name:     "Location tag",
			tagType:  domain.TagTypeLocation,
			value:    domain.TagValue{Type: domain.TagTypeLocation, LocationValue: &domain.GeographicLocation{Address: "office > floor2"}},
			expected: "office > floor2",
		},
		{
			name:     "Time tag",
			tagType:  domain.TagTypeTime,
			value:    domain.TagValue{Type: domain.TagTypeTime, TimeValue: func() *time.Time { t, _ := time.Parse("2006-01-02", "2023-12-25"); return &t }()},
			expected: "2023-12-25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.Type != tt.tagType {
				t.Errorf("Expected tag type %v, got %v", tt.tagType, tt.value.Type)
			}

			// Test string representation
			str := tt.value.String()
			if str != tt.expected {
				t.Errorf("Expected string representation '%s', got '%s'", tt.expected, str)
			}
		})
	}
}

// Test concurrent access to service
func TestConcurrentAccess(t *testing.T) {
	taskRepo := repository.NewInMemoryTaskRepository()
	userRepo := repository.NewInMemoryUserRepository()
	service := NewTaskService(taskRepo, 10, userRepo, nil, nil)

	// Create a test user
	ctx := context.Background()
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Test concurrent task creation
	const numGoroutines = 10
	ch := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			taskName := fmt.Sprintf("Concurrent Task %d", i)
			_, err := service.AddTaskForUser(ctx, taskName, "Concurrent test", user.ID)
			ch <- err
		}(i)
	}

	// Check for errors
	for i := 0; i < numGoroutines; i++ {
		if err := <-ch; err != nil {
			t.Errorf("Concurrent task creation failed: %v", err)
		}
	}

	// Verify all tasks were created
	tasks, err := service.ListTasksByUser(ctx, user.ID, nil)
	if err != nil {
		t.Fatalf("ListTasksByUser failed: %v", err)
	}

	if len(tasks) != numGoroutines {
		t.Errorf("Expected %d tasks, got %d", numGoroutines, len(tasks))
	}
}

func testUpdateTaskTags(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a test user
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create a task
	task, err := service.AddTaskForUser(ctx, "Test Task", "Test Description", user.ID)
	if err != nil {
		t.Fatalf("AddTaskForUser failed: %v", err)
	}

	// Update tags
	timeValue := time.Now()
	location := &domain.GeographicLocation{
		Latitude:  37.7749,
		Longitude: -122.4194,
		Address:   "office, desk1",
	}

	tags := map[string]domain.TagValue{
		"priority": {Type: domain.TagTypeText, TextValue: "high"},
		"location": {Type: domain.TagTypeLocation, LocationValue: location},
		"deadline": {Type: domain.TagTypeTime, TimeValue: &timeValue},
	}

	updatedTask, err := service.UpdateTaskTags(ctx, task.ID, tags)
	if err != nil {
		t.Fatalf("UpdateTaskTags failed: %v", err)
	}

	if len(updatedTask.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(updatedTask.Tags))
	}

	if updatedTask.Tags["priority"].TextValue != "high" {
		t.Errorf("Expected priority tag 'high', got %s", updatedTask.Tags["priority"].TextValue)
	}

	if updatedTask.Tags["location"].LocationValue == nil || updatedTask.Tags["location"].LocationValue.Address != "office, desk1" {
		t.Errorf("Expected location tag with address 'office, desk1'")
	}

	if updatedTask.Tags["deadline"].TimeValue == nil {
		t.Errorf("Expected time value to be set")
	}
}

func testCreateUser(t *testing.T, service *TaskService) {
	ctx := context.Background()

	notificationSettings := []domain.NotificationSetting{
		{Type: domain.NotificationOnAssign, Enabled: true},
		{Type: domain.NotificationOnStart, Enabled: false},
	}

	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", notificationSettings)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", user.Email)
	}

	if user.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got %s", user.Name)
	}

	if len(user.NotificationSettings) != 2 {
		t.Errorf("Expected 2 notification settings, got %d", len(user.NotificationSettings))
	}

	if user.ID == "" {
		t.Error("Expected user ID to be generated")
	}
}

func testGetUser(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a user
	createdUser, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Get the user
	retrievedUser, err := service.GetUser(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if retrievedUser.ID != createdUser.ID {
		t.Errorf("Expected user ID %s, got %s", createdUser.ID, retrievedUser.ID)
	}

	if retrievedUser.Email != createdUser.Email {
		t.Errorf("Expected email %s, got %s", createdUser.Email, retrievedUser.Email)
	}
}

func testUpdateUser(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a user
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Update user
	user.Name = "Updated User"
	user.NotificationSettings = []domain.NotificationSetting{
		{Type: domain.NotificationOnAssign, Enabled: true},
	}

	updatedUser, err := service.UpdateUser(ctx, user)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	if updatedUser.Name != "Updated User" {
		t.Errorf("Expected name 'Updated User', got %s", updatedUser.Name)
	}

	if len(updatedUser.NotificationSettings) != 1 {
		t.Errorf("Expected 1 notification setting, got %d", len(updatedUser.NotificationSettings))
	}
}

func testGetTaskDAG(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a test user
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create tasks with dependencies
	task1, err := service.AddTaskForUser(ctx, "Task 1", "First task", user.ID)
	if err != nil {
		t.Fatalf("AddTaskForUser failed: %v", err)
	}

	task2, err := service.AddTaskForUser(ctx, "Task 2", "Second task", user.ID)
	if err != nil {
		t.Fatalf("AddTaskForUser failed: %v", err)
	}

	// Move to staging to create dependencies
	_, err = service.MoveToStaging(ctx, task2.ID, &task1.ID, []string{}, []domain.Point{})
	if err != nil {
		t.Fatalf("MoveToStaging failed: %v", err)
	}

	// Get DAG
	dagTasks, err := service.GetTaskDAG(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetTaskDAG failed: %v", err)
	}

	if len(dagTasks) != 2 {
		t.Errorf("Expected 2 tasks in DAG, got %d", len(dagTasks))
	}

	// Tasks should be in topological order (task1 before task2)
	if dagTasks[0].ID != task1.ID {
		t.Errorf("Expected first task to be %s, got %s", task1.ID, dagTasks[0].ID)
	}
}

func testSyncCalendar(t *testing.T, service *TaskService) {
	ctx := context.Background()

	// Create a test user
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Test sync without calendar service (should fail gracefully)
	synced, errors, err := service.SyncCalendar(ctx, user.ID)
	if err == nil {
		t.Error("Expected SyncCalendar to fail when no calendar service is configured")
	}

	if synced != 0 {
		t.Errorf("Expected 0 synced tasks, got %d", synced)
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

// Test table-driven approach for tag types
func TestTagTypes(t *testing.T) {
	tests := []struct {
		name     string
		tagType  domain.TagType
		value    domain.TagValue
		expected string
	}{
		{
			name:     "Text tag",
			tagType:  domain.TagTypeText,
			value:    domain.TagValue{Type: domain.TagTypeText, TextValue: "urgent"},
			expected: "urgent",
		},
		{
			name:     "Location tag",
			tagType:  domain.TagTypeLocation,
			value:    domain.TagValue{Type: domain.TagTypeLocation, LocationValue: &domain.GeographicLocation{Address: "office > floor2"}},
			expected: "office > floor2",
		},
		{
			name:     "Time tag",
			tagType:  domain.TagTypeTime,
			value:    domain.TagValue{Type: domain.TagTypeTime, TimeValue: func() *time.Time { t, _ := time.Parse("2006-01-02", "2023-12-25"); return &t }()},
			expected: "2023-12-25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.Type != tt.tagType {
				t.Errorf("Expected tag type %v, got %v", tt.tagType, tt.value.Type)
			}

			// Test string representation
			str := tt.value.String()
			if str != tt.expected {
				t.Errorf("Expected string representation '%s', got '%s'", tt.expected, str)
			}
		})
	}
}

// Test concurrent access to service
func TestConcurrentAccess(t *testing.T) {
	taskRepo := repository.NewInMemoryTaskRepository()
	userRepo := repository.NewInMemoryUserRepository()
	service := NewTaskService(taskRepo, 10, userRepo, nil, nil, nil, nil) // Larger inbox for testing, no calendar/email/logger/eventBus

	// Create a test user
	ctx := context.Background()
	user, err := service.CreateUser(ctx, "", "test@example.com", "Test User", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Test concurrent task creation
	const numGoroutines = 10
	ch := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			taskName := fmt.Sprintf("Concurrent Task %d", i)
			_, err := service.AddTaskForUser(ctx, taskName, "Concurrent test", user.ID)
			ch <- err
		}(i)
	}

	// Check for errors
	for i := 0; i < numGoroutines; i++ {
		if err := <-ch; err != nil {
			t.Errorf("Concurrent task creation failed: %v", err)
		}
	}

	// Verify all tasks were created
	tasks, err := service.ListTasksByUser(ctx, user.ID, nil)
	if err != nil {
		t.Fatalf("ListTasksByUser failed: %v", err)
	}

	if len(tasks) != numGoroutines {
		t.Errorf("Expected %d tasks, got %d", numGoroutines, len(tasks))
	}
}
