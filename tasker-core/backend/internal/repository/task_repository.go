package repository

import (
	"context"
	"fmt"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
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

	// ListByStageAndUser returns all tasks in a given stage for a specific user
	ListByStageAndUser(ctx context.Context, stage domain.TaskStage, userID string) ([]*domain.Task, error)

	// ListByUser returns all tasks for a specific user
	ListByUser(ctx context.Context, userID string) ([]*domain.Task, error)

	// ListAll returns all tasks
	ListAll(ctx context.Context) ([]*domain.Task, error)

	// CountByStage returns the number of tasks in a given stage
	CountByStage(ctx context.Context, stage domain.TaskStage) (int, error)

	// GetTasksByIDs retrieves multiple tasks by their IDs
	GetTasksByIDs(ctx context.Context, ids []string) ([]*domain.Task, error)
}

// UserRepository defines the interface for user persistence
type UserRepository interface {
	// Create stores a new user
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by their ID
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by their email
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user
	Delete(ctx context.Context, id string) error

	// ListAll returns all users
	ListAll(ctx context.Context) ([]*domain.User, error)
}

// ErrTaskNotFound is returned when a task is not found
var ErrTaskNotFound = fmt.Errorf("task not found")

// ErrUserNotFound is returned when a user is not found
var ErrUserNotFound = fmt.Errorf("user not found")
