package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
)

// InMemoryTaskRepository is a simple in-memory implementation of TaskRepository
type InMemoryTaskRepository struct {
	tasks map[string]*domain.Task
	mutex sync.RWMutex
}

// NewInMemoryTaskRepository creates a new in-memory task repository
func NewInMemoryTaskRepository() *InMemoryTaskRepository {
	return &InMemoryTaskRepository{
		tasks: make(map[string]*domain.Task),
	}
}

// Create stores a new task
func (r *InMemoryTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if task already exists
	if _, exists := r.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Create a copy to avoid external modifications
	taskCopy := *task
	r.tasks[task.ID] = &taskCopy
	return nil
}

// GetByID retrieves a task by its ID
func (r *InMemoryTaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}

	// Return a copy to avoid external modifications
	taskCopy := *task
	return &taskCopy, nil
}

// Update updates an existing task
func (r *InMemoryTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}

	// Create a copy to avoid external modifications
	taskCopy := *task
	r.tasks[task.ID] = &taskCopy
	return nil
}

// Delete removes a task
func (r *InMemoryTaskRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return ErrTaskNotFound
	}

	delete(r.tasks, id)
	return nil
}

// ListByStage returns all tasks in a given stage
func (r *InMemoryTaskRepository) ListByStage(ctx context.Context, stage domain.TaskStage) ([]*domain.Task, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var tasks []*domain.Task
	for _, task := range r.tasks {
		if task.Stage == stage {
			taskCopy := *task
			tasks = append(tasks, &taskCopy)
		}
	}

	return tasks, nil
}

// ListAll returns all tasks
func (r *InMemoryTaskRepository) ListAll(ctx context.Context) ([]*domain.Task, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var tasks []*domain.Task
	for _, task := range r.tasks {
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}

	return tasks, nil
}

// CountByStage returns the number of tasks in a given stage
func (r *InMemoryTaskRepository) CountByStage(ctx context.Context, stage domain.TaskStage) (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, task := range r.tasks {
		if task.Stage == stage {
			count++
		}
	}

	return count, nil
}

// GetTasksByIDs retrieves multiple tasks by their IDs
func (r *InMemoryTaskRepository) GetTasksByIDs(ctx context.Context, ids []string) ([]*domain.Task, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var tasks []*domain.Task
	var notFound []string

	for _, id := range ids {
		if task, exists := r.tasks[id]; exists {
			taskCopy := *task
			tasks = append(tasks, &taskCopy)
		} else {
			notFound = append(notFound, id)
		}
	}

	if len(notFound) > 0 {
		return tasks, fmt.Errorf("tasks not found: %v", notFound)
	}

	return tasks, nil
}
