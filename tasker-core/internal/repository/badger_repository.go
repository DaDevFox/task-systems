package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	"github.com/dgraph-io/badger/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Error constants
const (
	ErrEmptyTaskID = "task ID cannot be empty"
)

// BadgerTaskRepository implements TaskRepository using BadgerDB for persistence
type BadgerTaskRepository struct {
	db      *badger.DB
	metrics *TaskMetrics
	logger  *logrus.Logger
}

// TaskMetrics stores task statistics
type TaskMetrics struct {
	ActiveTasks    int       `json:"active_tasks"`
	BlockedTasks   int       `json:"blocked_tasks"`
	DueSoonTasks   int       `json:"due_soon_tasks"`
	CompletedTasks int       `json:"completed_tasks"`
	LastUpdated    time.Time `json:"last_updated"`
}

// NewBadgerTaskRepository creates a new BadgerDB-backed task repository
func NewBadgerTaskRepository(dbPath string, logger *logrus.Logger) (*BadgerTaskRepository, error) {
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = &badgerLogger{logger: logger}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open badger database")
	}

	repo := &BadgerTaskRepository{
		db:      db,
		metrics: &TaskMetrics{},
		logger:  logger,
	}

	// Load existing metrics
	if err := repo.loadMetrics(); err != nil {
		logger.WithError(err).Warn("failed to load existing metrics, starting fresh")
	}

	return repo, nil
}

// Close closes the database connection
func (r *BadgerTaskRepository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

// Create stores a new task
func (r *BadgerTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	if task == nil {
		return errors.New("task cannot be nil")
	}

	if task.ID == "" {
		return errors.New(ErrEmptyTaskID)
	}

	data, err := json.Marshal(task)
	if err != nil {
		return errors.Wrap(err, "failed to marshal task to JSON")
	}

	err = r.db.Update(func(txn *badger.Txn) error {
		key := []byte("task:" + task.ID)
		return txn.Set(key, data)
	})

	if err != nil {
		return errors.Wrap(err, "failed to store task in database")
	}

	r.logger.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"stage":     task.Stage.String(),
	}).Info("task created")

	r.updateMetrics()
	return nil
}

// GetByID retrieves a task by ID
func (r *BadgerTaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	if id == "" {
		return nil, errors.New(ErrEmptyTaskID)
	}

	var task domain.Task
	err := r.db.View(func(txn *badger.Txn) error {
		key := []byte("task:" + id)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &task)
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, ErrTaskNotFound
		}
		return nil, errors.Wrap(err, "failed to retrieve task from database")
	}

	return &task, nil
}

// Update modifies an existing task
func (r *BadgerTaskRepository) Update(ctx context.Context, task *domain.Task) error {
	if task == nil {
		return errors.New("task cannot be nil")
	}

	if task.ID == "" {
		return errors.New(ErrEmptyTaskID)
	}

	// Check if task exists
	_, err := r.GetByID(ctx, task.ID)
	if err != nil {
		return errors.Wrap(err, "failed to verify task existence")
	}

	data, err := json.Marshal(task)
	if err != nil {
		return errors.Wrap(err, "failed to marshal task to JSON")
	}

	err = r.db.Update(func(txn *badger.Txn) error {
		key := []byte("task:" + task.ID)
		return txn.Set(key, data)
	})

	if err != nil {
		return errors.Wrap(err, "failed to update task in database")
	}

	r.logger.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_name": task.Name,
		"stage":     task.Stage.String(),
	}).Info("task updated")

	r.updateMetrics()
	return nil
}

// Delete removes a task
func (r *BadgerTaskRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New(ErrEmptyTaskID)
	}

	err := r.db.Update(func(txn *badger.Txn) error {
		key := []byte("task:" + id)
		return txn.Delete(key)
	})

	if err != nil {
		return errors.Wrap(err, "failed to delete task from database")
	}

	r.logger.WithField("task_id", id).Info("task deleted")

	r.updateMetrics()
	return nil
}

// List retrieves all tasks
func (r *BadgerTaskRepository) List() ([]*domain.Task, error) {
	var tasks []*domain.Task

	err := r.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte("task:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				var task domain.Task
				if err := json.Unmarshal(val, &task); err != nil {
					return errors.Wrap(err, "failed to unmarshal task from JSON")
				}
				tasks = append(tasks, &task)
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to list tasks from database")
	}

	return tasks, nil
}

// GetMetrics returns current task metrics
func (r *BadgerTaskRepository) GetMetrics() *TaskMetrics {
	if r.metrics == nil {
		return &TaskMetrics{}
	}
	return r.metrics
}

// updateMetrics recalculates and stores task metrics
func (r *BadgerTaskRepository) updateMetrics() {
	tasks, err := r.List()
	if err != nil {
		r.logger.WithError(err).Error("failed to list tasks for metrics update")
		return
	}

	metrics := &TaskMetrics{
		LastUpdated: time.Now(),
	}

	dueSoonThreshold := time.Now().Add(24 * time.Hour) // Tasks due within 24 hours

	for _, task := range tasks {
		if task == nil {
			continue
		}

		switch task.Stage {
		case domain.StageActive:
			metrics.ActiveTasks++
		case domain.StageArchived:
			metrics.CompletedTasks++
		}

		// Check if task is blocked (has dependencies that aren't completed)
		if len(task.Inflows) > 0 {
			metrics.BlockedTasks++
		}

		// Check if task is due soon (this would need due date field in domain.Task)
		// For now, we'll use creation time + 7 days as a simple heuristic
		if task.CreatedAt.Add(7*24*time.Hour).Before(dueSoonThreshold) &&
			task.Stage != domain.StageArchived {
			metrics.DueSoonTasks++
		}
	}

	r.metrics = metrics

	// Store metrics persistently
	if err := r.storeMetrics(); err != nil {
		r.logger.WithError(err).Error("failed to store metrics")
	}

	r.logger.WithFields(logrus.Fields{
		"active_tasks":    metrics.ActiveTasks,
		"blocked_tasks":   metrics.BlockedTasks,
		"due_soon_tasks":  metrics.DueSoonTasks,
		"completed_tasks": metrics.CompletedTasks,
	}).Debug("metrics updated")
}

// storeMetrics saves metrics to the database
func (r *BadgerTaskRepository) storeMetrics() error {
	if r.metrics == nil {
		return errors.New("metrics cannot be nil")
	}

	data, err := json.Marshal(r.metrics)
	if err != nil {
		return errors.Wrap(err, "failed to marshal metrics to JSON")
	}

	return r.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("metrics"), data)
	})
}

// loadMetrics loads metrics from the database
func (r *BadgerTaskRepository) loadMetrics() error {
	return r.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("metrics"))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // No existing metrics, start fresh
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, r.metrics)
		})
	})
}

// ListByStage retrieves tasks by stage
func (r *BadgerTaskRepository) ListByStage(ctx context.Context, stage domain.TaskStage) ([]*domain.Task, error) {
	tasks, err := r.List()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list tasks for stage filtering")
	}

	var filtered []*domain.Task
	for _, task := range tasks {
		if task != nil && task.Stage == stage {
			filtered = append(filtered, task)
		}
	}

	return filtered, nil
}

// ListByStageAndUser retrieves tasks by stage and user
func (r *BadgerTaskRepository) ListByStageAndUser(ctx context.Context, stage domain.TaskStage, userID string) ([]*domain.Task, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	tasks, err := r.ListByStage(ctx, stage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list tasks by stage for user filtering")
	}

	var filtered []*domain.Task
	for _, task := range tasks {
		if task != nil && task.UserID == userID {
			filtered = append(filtered, task)
		}
	}

	return filtered, nil
}

// ListByUser retrieves all tasks for a user
func (r *BadgerTaskRepository) ListByUser(ctx context.Context, userID string) ([]*domain.Task, error) {
	if userID == "" {
		return nil, errors.New("user ID cannot be empty")
	}

	tasks, err := r.List()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list tasks for user filtering")
	}

	var filtered []*domain.Task
	for _, task := range tasks {
		if task != nil && task.UserID == userID {
			filtered = append(filtered, task)
		}
	}

	return filtered, nil
}

// ListAll retrieves all tasks (alias for List)
func (r *BadgerTaskRepository) ListAll(ctx context.Context) ([]*domain.Task, error) {
	return r.List()
}

// CountByStage counts tasks in a given stage
func (r *BadgerTaskRepository) CountByStage(ctx context.Context, stage domain.TaskStage) (int, error) {
	tasks, err := r.ListByStage(ctx, stage)
	if err != nil {
		return 0, errors.Wrap(err, "failed to count tasks by stage")
	}

	return len(tasks), nil
}

// GetTasksByIDs retrieves multiple tasks by their IDs
func (r *BadgerTaskRepository) GetTasksByIDs(ctx context.Context, ids []string) ([]*domain.Task, error) {
	if len(ids) == 0 {
		return []*domain.Task{}, nil
	}

	var tasks []*domain.Task
	var notFoundIDs []string

	for _, id := range ids {
		if id == "" {
			continue
		}

		task, err := r.GetByID(ctx, id)
		if err != nil {
			if err == ErrTaskNotFound {
				notFoundIDs = append(notFoundIDs, id)
				continue
			}
			return nil, errors.Wrapf(err, "failed to retrieve task with ID %s", id)
		}

		tasks = append(tasks, task)
	}

	if len(notFoundIDs) > 0 {
		r.logger.WithField("not_found_ids", notFoundIDs).Warn("some tasks not found during bulk retrieval")
	}

	return tasks, nil
}

// badgerLogger adapts logrus to BadgerDB's logger interface
type badgerLogger struct {
	logger *logrus.Logger
}

func (l *badgerLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *badgerLogger) Warningf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *badgerLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *badgerLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}
