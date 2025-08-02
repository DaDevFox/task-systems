package repository

import (
	"context"
	"testing"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
)

func TestInMemoryTaskRepository(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T, TaskRepository)
	}{
		{"CreateAndGetTask", testCreateAndGetTask},
		{"CreateDuplicateTask", testCreateDuplicateTask},
		{"GetNonExistentTask", testGetNonExistentTask},
		{"UpdateTask", testUpdateTask},
		{"UpdateNonExistentTask", testUpdateNonExistentTask},
		{"DeleteTask", testDeleteTask},
		{"DeleteNonExistentTask", testDeleteNonExistentTask},
		{"ListByStage", testListByStage},
		{"ListAll", testListAll},
		{"CountByStage", testCountByStage},
		{"GetTasksByIDs", testGetTasksByIDs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryTaskRepository()
			tt.test(t, repo)
		})
	}
}

func testCreateAndGetTask(t *testing.T, repo TaskRepository) {
	ctx := context.Background()
	task := domain.NewTask("Test Task", "Test Description")

	// Create task
	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get task
	retrieved, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
	}
	if retrieved.Name != task.Name {
		t.Errorf("Expected Name %s, got %s", task.Name, retrieved.Name)
	}
	if retrieved.Description != task.Description {
		t.Errorf("Expected Description %s, got %s", task.Description, retrieved.Description)
	}
}

func testCreateDuplicateTask(t *testing.T, repo TaskRepository) {
	ctx := context.Background()
	task := domain.NewTask("Test Task", "Test Description")

	// Create task first time
	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Try to create same task again
	err = repo.Create(ctx, task)
	if err == nil {
		t.Fatal("Expected error when creating duplicate task, got nil")
	}
}

func testGetNonExistentTask(t *testing.T, repo TaskRepository) {
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func testUpdateTask(t *testing.T, repo TaskRepository) {
	ctx := context.Background()
	task := domain.NewTask("Test Task", "Test Description")

	// Create task
	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update task
	task.Name = "Updated Task"
	task.Description = "Updated Description"
	err = repo.Update(ctx, task)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Name != "Updated Task" {
		t.Errorf("Expected Name 'Updated Task', got %s", retrieved.Name)
	}
	if retrieved.Description != "Updated Description" {
		t.Errorf("Expected Description 'Updated Description', got %s", retrieved.Description)
	}
}

func testUpdateNonExistentTask(t *testing.T, repo TaskRepository) {
	ctx := context.Background()
	task := domain.NewTask("Test Task", "Test Description")

	err := repo.Update(ctx, task)
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func testDeleteTask(t *testing.T, repo TaskRepository) {
	ctx := context.Background()
	task := domain.NewTask("Test Task", "Test Description")

	// Create task
	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete task
	err = repo.Delete(ctx, task.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(ctx, task.ID)
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound after deletion, got %v", err)
	}
}

func testDeleteNonExistentTask(t *testing.T, repo TaskRepository) {
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func testListByStage(t *testing.T, repo TaskRepository) {
	ctx := context.Background()

	// Create tasks in different stages
	task1 := domain.NewTask("Task 1", "Description 1")
	task1.Stage = domain.StagePending

	task2 := domain.NewTask("Task 2", "Description 2")
	task2.Stage = domain.StageInbox

	task3 := domain.NewTask("Task 3", "Description 3")
	task3.Stage = domain.StagePending

	err := repo.Create(ctx, task1)
	if err != nil {
		t.Fatalf("Create task1 failed: %v", err)
	}
	err = repo.Create(ctx, task2)
	if err != nil {
		t.Fatalf("Create task2 failed: %v", err)
	}
	err = repo.Create(ctx, task3)
	if err != nil {
		t.Fatalf("Create task3 failed: %v", err)
	}

	// List tasks in pending stage
	pendingTasks, err := repo.ListByStage(ctx, domain.StagePending)
	if err != nil {
		t.Fatalf("ListByStage failed: %v", err)
	}

	if len(pendingTasks) != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", len(pendingTasks))
	}

	// List tasks in inbox stage
	inboxTasks, err := repo.ListByStage(ctx, domain.StageInbox)
	if err != nil {
		t.Fatalf("ListByStage failed: %v", err)
	}

	if len(inboxTasks) != 1 {
		t.Errorf("Expected 1 inbox task, got %d", len(inboxTasks))
	}
}

func testListAll(t *testing.T, repo TaskRepository) {
	ctx := context.Background()

	// Create multiple tasks
	task1 := domain.NewTask("Task 1", "Description 1")
	task2 := domain.NewTask("Task 2", "Description 2")
	task3 := domain.NewTask("Task 3", "Description 3")

	err := repo.Create(ctx, task1)
	if err != nil {
		t.Fatalf("Create task1 failed: %v", err)
	}
	err = repo.Create(ctx, task2)
	if err != nil {
		t.Fatalf("Create task2 failed: %v", err)
	}
	err = repo.Create(ctx, task3)
	if err != nil {
		t.Fatalf("Create task3 failed: %v", err)
	}

	// List all tasks
	allTasks, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(allTasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(allTasks))
	}
}

func testCountByStage(t *testing.T, repo TaskRepository) {
	ctx := context.Background()

	// Create tasks in different stages
	task1 := domain.NewTask("Task 1", "Description 1")
	task1.Stage = domain.StagePending

	task2 := domain.NewTask("Task 2", "Description 2")
	task2.Stage = domain.StagePending

	task3 := domain.NewTask("Task 3", "Description 3")
	task3.Stage = domain.StageInbox

	err := repo.Create(ctx, task1)
	if err != nil {
		t.Fatalf("Create task1 failed: %v", err)
	}
	err = repo.Create(ctx, task2)
	if err != nil {
		t.Fatalf("Create task2 failed: %v", err)
	}
	err = repo.Create(ctx, task3)
	if err != nil {
		t.Fatalf("Create task3 failed: %v", err)
	}

	// Count pending tasks
	pendingCount, err := repo.CountByStage(ctx, domain.StagePending)
	if err != nil {
		t.Fatalf("CountByStage failed: %v", err)
	}
	if pendingCount != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", pendingCount)
	}

	// Count inbox tasks
	inboxCount, err := repo.CountByStage(ctx, domain.StageInbox)
	if err != nil {
		t.Fatalf("CountByStage failed: %v", err)
	}
	if inboxCount != 1 {
		t.Errorf("Expected 1 inbox task, got %d", inboxCount)
	}

	// Count staging tasks (should be 0)
	stagingCount, err := repo.CountByStage(ctx, domain.StageStaging)
	if err != nil {
		t.Fatalf("CountByStage failed: %v", err)
	}
	if stagingCount != 0 {
		t.Errorf("Expected 0 staging tasks, got %d", stagingCount)
	}
}

func testGetTasksByIDs(t *testing.T, repo TaskRepository) {
	ctx := context.Background()

	// Create tasks
	task1 := domain.NewTask("Task 1", "Description 1")
	task2 := domain.NewTask("Task 2", "Description 2")
	task3 := domain.NewTask("Task 3", "Description 3")

	err := repo.Create(ctx, task1)
	if err != nil {
		t.Fatalf("Create task1 failed: %v", err)
	}
	err = repo.Create(ctx, task2)
	if err != nil {
		t.Fatalf("Create task2 failed: %v", err)
	}
	err = repo.Create(ctx, task3)
	if err != nil {
		t.Fatalf("Create task3 failed: %v", err)
	}

	// Get existing tasks
	ids := []string{task1.ID, task3.ID}
	tasks, err := repo.GetTasksByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetTasksByIDs failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Get mix of existing and non-existing tasks
	ids = []string{task1.ID, "nonexistent", task2.ID}
	_, err = repo.GetTasksByIDs(ctx, ids)
	if err == nil {
		t.Error("Expected error when getting non-existent tasks, got nil")
	}
}
