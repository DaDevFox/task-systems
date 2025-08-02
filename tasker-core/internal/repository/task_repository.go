package repository

import (
	"context"
	"fmt"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
)

// TaskRepository defines the interface for task persistence
type TaskRepository interface {
	// Create stores a new task
	Create(ctx context.Context, task *domain.Task) error

	// GetByID retrieves a task by its ID
	GetByID(ctx context.Context, id string) (*domain.Task, error)

	// Update updates an existing task
	Update(ctx context.Context, task *domain.Task) error

	// Delete removes a task
	Delete(ctx context.Context, id string) error

	// ListByStage returns all tasks in a given stage
	ListByStage(ctx context.Context, stage domain.TaskStage) ([]*domain.Task, error)

	// ListAll returns all tasks
	ListAll(ctx context.Context) ([]*domain.Task, error)

	// CountByStage returns the number of tasks in a given stage
	CountByStage(ctx context.Context, stage domain.TaskStage) (int, error)

	// GetTasksByIDs retrieves multiple tasks by their IDs
	GetTasksByIDs(ctx context.Context, ids []string) ([]*domain.Task, error)
}

// ErrTaskNotFound is returned when a task is not found
var ErrTaskNotFound = fmt.Errorf("task not found")
