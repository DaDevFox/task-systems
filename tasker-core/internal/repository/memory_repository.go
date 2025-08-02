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

// ListByStageAndUser returns all tasks in a given stage for a specific user
func (r *InMemoryTaskRepository) ListByStageAndUser(ctx context.Context, stage domain.TaskStage, userID string) ([]*domain.Task, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var tasks []*domain.Task
	for _, task := range r.tasks {
		if task.Stage == stage && task.UserID == userID {
			taskCopy := *task
			tasks = append(tasks, &taskCopy)
		}
	}

	return tasks, nil
}

// ListByUser returns all tasks for a specific user
func (r *InMemoryTaskRepository) ListByUser(ctx context.Context, userID string) ([]*domain.Task, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var tasks []*domain.Task
	for _, task := range r.tasks {
		if task.UserID == userID {
			taskCopy := *task
			tasks = append(tasks, &taskCopy)
		}
	}

	return tasks, nil
}

// InMemoryUserRepository is a simple in-memory implementation of UserRepository
type InMemoryUserRepository struct {
	users map[string]*domain.User
	mutex sync.RWMutex
}

// NewInMemoryUserRepository creates a new in-memory user repository
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users: make(map[string]*domain.User),
	}
}

// Create stores a new user
func (r *InMemoryUserRepository) Create(ctx context.Context, user *domain.User) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if user already exists
	if _, exists := r.users[user.ID]; exists {
		return fmt.Errorf("user with ID %s already exists", user.ID)
	}

	// Create a copy to avoid external modifications
	userCopy := *user
	r.users[user.ID] = &userCopy
	return nil
}

// GetByID retrieves a user by their ID
func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Return a copy to avoid external modifications
	userCopy := *user
	return &userCopy, nil
}

// GetByEmail retrieves a user by their email
func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, user := range r.users {
		if user.Email == email {
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, ErrUserNotFound
}

// Update updates an existing user
func (r *InMemoryUserRepository) Update(ctx context.Context, user *domain.User) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return ErrUserNotFound
	}

	// Create a copy to avoid external modifications
	userCopy := *user
	r.users[user.ID] = &userCopy
	return nil
}

// Delete removes a user
func (r *InMemoryUserRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.users[id]; !exists {
		return ErrUserNotFound
	}

	delete(r.users, id)
	return nil
}

// ListAll returns all users
func (r *InMemoryUserRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var users []*domain.User
	for _, user := range r.users {
		userCopy := *user
		users = append(users, &userCopy)
	}

	return users, nil
}
