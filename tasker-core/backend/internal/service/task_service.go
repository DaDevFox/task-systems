package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/calendar"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/email"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/events"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/repository"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// TaskService provides comprehensive business logic for task management
type TaskService struct {
	repo            repository.TaskRepository
	userRepo        repository.UserRepository
	calendarService *calendar.CalendarService
	emailService    *email.EmailService
	logger          *logrus.Logger
	eventBus        *events.PubSub
	maxInboxSize    int
	syncEnabled     bool
}

// NewTaskService creates a new task service with optional integrations
func NewTaskService(
	repo repository.TaskRepository,
	maxInboxSize int,
	userRepo repository.UserRepository,
	calendarService *calendar.CalendarService,
	emailService *email.EmailService,
	logger *logrus.Logger,
	eventBus *events.PubSub,
) *TaskService {
	if maxInboxSize <= 0 {
		maxInboxSize = 5 // default
	}
	if logger == nil {
		logger = logrus.New()
	}
	if eventBus == nil {
		eventBus = events.NewPubSub(logger)
	}

	return &TaskService{
		repo:            repo,
		userRepo:        userRepo,
		calendarService: calendarService,
		emailService:    emailService,
		logger:          logger,
		eventBus:        eventBus,
		maxInboxSize:    maxInboxSize,
		syncEnabled:     true,
	}
}

// SetSyncEnabled enables or disables automatic calendar sync
func (s *TaskService) SetSyncEnabled(enabled bool) {
	s.syncEnabled = enabled
}

// AddTask creates a new task in pending stage
func (s *TaskService) AddTask(ctx context.Context, name, description string) (*domain.Task, error) {
	return s.AddTaskForUser(ctx, name, description, "default-user")
}

// AddTaskForUser creates a new task in pending stage for a specific user
func (s *TaskService) AddTaskForUser(ctx context.Context, name, description, userID string) (*domain.Task, error) {
	if name == "" {
		return nil, errors.New("task name cannot be empty")
	}
	if userID == "" {
		userID = "default-user" // fallback for backward compatibility
	}

	// Verify user exists if userRepo is available
	if s.userRepo != nil {
		if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
			return nil, fmt.Errorf("user not found: %w", err)
		}
	}

	task := domain.NewTask(name, description, userID)
	task.Stage = domain.StageInbox // New tasks go to inbox first

	if err := s.repo.Create(ctx, task); err != nil {
		s.logger.WithError(err).Error("task creation failed")
		return nil, errors.Wrap(err, "task creation failed")
	}

	// Send assignment notification if email service is available
	if s.emailService != nil && s.userRepo != nil {
		if user, userErr := s.userRepo.GetByID(ctx, userID); userErr == nil {
			if err := s.emailService.SendTaskAssignedNotification(user, task); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Failed to send assignment notification: %v\n", err)
			}
		}
	}

	// Publish task created event
	if s.eventBus != nil {
		s.eventBus.Publish(ctx, events.Event{
			Type: events.EventTaskCreated,
			Data: map[string]interface{}{
				"task_id": task.ID,
				"user_id": task.UserID,
			},
			UserID: task.UserID,
		})
	}

	return task, nil
}

// MoveToStaging moves a task from pending/inbox to staging
func (s *TaskService) MoveToStaging(ctx context.Context, sourceID string, destinationID *string, newLocation []string, points []domain.Point) (*domain.Task, error) {
	// Check inbox constraint before any operations
	if err := s.checkInboxConstraint(ctx); err != nil {
		return nil, err
	}

	task, err := s.repo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("source task not found: %w", err)
	}

	if err := task.CanMoveToStaging(); err != nil {
		return nil, err
	}

	// Handle destination logic
	if destinationID != nil {
		destTask, err := s.repo.GetByID(ctx, *destinationID)
		if err != nil {
			return nil, fmt.Errorf("destination task not found: %w", err)
		}

		// Inherit location from destination
		task.Location = destTask.Location

		// Set up dependency: source depends on destination
		task.Inflows = append(task.Inflows, *destinationID)
		destTask.Outflows = append(destTask.Outflows, sourceID)

		// Update destination task
		if err := s.repo.Update(ctx, destTask); err != nil {
			return nil, fmt.Errorf("failed to update destination task: %w", err)
		}
	} else if len(newLocation) > 0 {
		task.Location = newLocation
	} else {
		return nil, fmt.Errorf("either destination_id or new_location must be provided")
	}

	// Set points if provided
	if len(points) > 0 {
		task.Points = points
	}

	// Move to staging
	task.Stage = domain.StageStaging
	task.AddStatusUpdate("Moved to staging")

	err = s.repo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// StartTask moves a task to active stage and sets it as in progress
func (s *TaskService) StartTask(ctx context.Context, id string) (*domain.Task, error) {
	// Check inbox constraint
	if err := s.checkInboxConstraint(ctx); err != nil {
		return nil, err
	}

	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	if err := task.CanStart(); err != nil {
		return nil, err
	}

	// Check dependencies
	if err := s.checkDependencies(ctx, task); err != nil {
		return nil, err
	}

	task.Stage = domain.StageActive
	task.Status = domain.StatusInProgress
	task.AddStatusUpdate("Task started")

	// Add work interval
	now := time.Now()
	task.Schedule.WorkIntervals = append(task.Schedule.WorkIntervals, domain.WorkInterval{
		Start: now,
		Stop:  time.Time{}, // Will be set when stopped
	})

	err = s.repo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// Sync to calendar if enabled
	if s.syncEnabled && s.calendarService != nil {
		go s.syncTaskToCalendar(task)
	}

	// Send start notification
	if s.emailService != nil && s.userRepo != nil {
		if user, userErr := s.userRepo.GetByID(ctx, task.UserID); userErr == nil {
			if err := s.emailService.SendTaskStartedNotification(user, task); err != nil {
				fmt.Printf("Failed to send start notification: %v\n", err)
			}
		}
	}

	return task, nil

	err = s.repo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// StopTask stops an active task and marks completed points
func (s *TaskService) StopTask(ctx context.Context, id string, pointsCompleted []domain.Point) (*domain.Task, bool, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, false, fmt.Errorf("task not found: %w", err)
	}

	if err := task.CanStop(); err != nil {
		return nil, false, err
	}

	// Update the last work interval
	if len(task.Schedule.WorkIntervals) > 0 {
		lastInterval := &task.Schedule.WorkIntervals[len(task.Schedule.WorkIntervals)-1]
		lastInterval.Stop = time.Now()
		lastInterval.PointsCompleted = pointsCompleted
	}

	// Check if task is complete
	isComplete := task.IsComplete()
	if isComplete {
		task.Status = domain.StatusCompleted
		task.Stage = domain.StageArchived
		task.AddStatusUpdate("Task completed")
	} else {
		task.Status = domain.StatusTodo
		task.Stage = domain.StageStaging
		task.AddStatusUpdate("Task stopped")
	}

	err = s.repo.Update(ctx, task)
	if err != nil {
		return nil, false, fmt.Errorf("failed to update task: %w", err)
	}

	return task, isComplete, nil
}

// CompleteTask marks a task as completed
func (s *TaskService) CompleteTask(ctx context.Context, id string) (*domain.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	if err := task.CanStop(); err != nil {
		return nil, err
	}

	// Complete all remaining points
	remainingPoints := task.TotalPoints() - task.CompletedPoints()
	if remainingPoints > 0 && len(task.Points) > 0 {
		// Create completion points based on original points structure
		completionPoints := make([]domain.Point, len(task.Points))
		for i, point := range task.Points {
			completionPoints[i] = domain.Point{
				Title: point.Title,
				Value: point.Value,
			}
		}

		// Update the last work interval
		if len(task.Schedule.WorkIntervals) > 0 {
			lastInterval := &task.Schedule.WorkIntervals[len(task.Schedule.WorkIntervals)-1]
			lastInterval.Stop = time.Now()
			lastInterval.PointsCompleted = completionPoints
		}
	}

	task.Status = domain.StatusCompleted
	task.Stage = domain.StageArchived
	task.AddStatusUpdate("Task completed")

	err = s.repo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// MergeTasks combines two tasks in the same dependency chain
func (s *TaskService) MergeTasks(ctx context.Context, fromID, toID string) (*domain.Task, error) {
	fromTask, err := s.repo.GetByID(ctx, fromID)
	if err != nil {
		return nil, fmt.Errorf("from task not found: %w", err)
	}

	toTask, err := s.repo.GetByID(ctx, toID)
	if err != nil {
		return nil, fmt.Errorf("to task not found: %w", err)
	}

	// Validate they're in the same chain
	if err := s.validateSameChain(ctx, fromTask, toTask); err != nil {
		return nil, err
	}

	// Merge data
	toTask.Description = toTask.Description + "\n" + fromTask.Description
	toTask.Points = append(toTask.Points, fromTask.Points...)

	// Merge status history
	toTask.StatusHist.Updates = append(toTask.StatusHist.Updates, fromTask.StatusHist.Updates...)

	// Merge tags
	for k, v := range fromTask.Tags {
		toTask.Tags[k] = v
	}

	toTask.AddStatusUpdate(fmt.Sprintf("Merged with task %s", fromID))

	// Update dependencies
	if err := s.updateDependenciesForMerge(ctx, fromTask, toTask); err != nil {
		return nil, err
	}

	// Delete the from task
	if err := s.repo.Delete(ctx, fromID); err != nil {
		return nil, fmt.Errorf("failed to delete from task: %w", err)
	}

	// Update the to task
	if err := s.repo.Update(ctx, toTask); err != nil {
		return nil, fmt.Errorf("failed to update to task: %w", err)
	}

	return toTask, nil
}

// SplitTask creates new tasks based on the original
func (s *TaskService) SplitTask(ctx context.Context, id string, newNames, newDescriptions []string) ([]*domain.Task, error) {
	if len(newNames) != len(newDescriptions) {
		return nil, fmt.Errorf("new names and descriptions must have the same length")
	}

	if len(newNames) == 0 {
		return nil, fmt.Errorf("at least one new task must be specified")
	}

	originalTask, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("original task not found: %w", err)
	}

	var newTasks []*domain.Task

	for i, name := range newNames {
		newTask := domain.NewTask(name, newDescriptions[i], originalTask.UserID)
		newTask.Stage = originalTask.Stage
		newTask.Location = originalTask.Location
		newTask.Tags = make(map[string]domain.TagValue)

		// Copy tags
		for k, v := range originalTask.Tags {
			newTask.Tags[k] = v
		}

		newTask.AddStatusUpdate(fmt.Sprintf("Split from task %s", id))

		if err := s.repo.Create(ctx, newTask); err != nil {
			return nil, fmt.Errorf("failed to create new task %s: %w", name, err)
		}

		newTasks = append(newTasks, newTask)
	}

	// Update dependencies
	if err := s.updateDependenciesForSplit(ctx, originalTask, newTasks); err != nil {
		return nil, err
	}

	// Delete original task
	if err := s.repo.Delete(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to delete original task: %w", err)
	}

	return newTasks, nil
}

// AdvertiseTask makes one task outflow into many
func (s *TaskService) AdvertiseTask(ctx context.Context, id string, targetIDs []string) (*domain.Task, error) {
	sourceTask, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("source task not found: %w", err)
	}

	// Get target tasks and validate they exist
	targetTasks, err := s.repo.GetTasksByIDs(ctx, targetIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get target tasks: %w", err)
	}

	// Check for chain hopping
	for _, target := range targetTasks {
		if err := s.checkChainHopping(ctx, sourceTask, target); err != nil {
			return nil, err
		}
	}

	// Update outflows
	for _, targetID := range targetIDs {
		if !contains(sourceTask.Outflows, targetID) {
			sourceTask.Outflows = append(sourceTask.Outflows, targetID)
		}
	}

	// Update target tasks' inflows
	for _, target := range targetTasks {
		if !contains(target.Inflows, id) {
			target.Inflows = append(target.Inflows, id)
		}
		if err := s.repo.Update(ctx, target); err != nil {
			return nil, fmt.Errorf("failed to update target task %s: %w", target.ID, err)
		}
	}

	sourceTask.AddStatusUpdate("Task advertised to multiple targets")

	err = s.repo.Update(ctx, sourceTask)
	if err != nil {
		return nil, fmt.Errorf("failed to update source task: %w", err)
	}

	return sourceTask, nil
}

// StitchTasks makes multiple tasks outflow into one
func (s *TaskService) StitchTasks(ctx context.Context, sourceIDs []string, targetID string) ([]*domain.Task, error) {
	sourceTasks, err := s.repo.GetTasksByIDs(ctx, sourceIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get source tasks: %w", err)
	}

	targetTask, err := s.repo.GetByID(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("target task not found: %w", err)
	}

	// Check for chain hopping
	for _, source := range sourceTasks {
		if err := s.checkChainHopping(ctx, source, targetTask); err != nil {
			return nil, err
		}
	}

	// Update source tasks' outflows
	var updatedTasks []*domain.Task
	for _, source := range sourceTasks {
		if !contains(source.Outflows, targetID) {
			source.Outflows = append(source.Outflows, targetID)
		}
		source.AddStatusUpdate("Task stitched to common target")

		if err := s.repo.Update(ctx, source); err != nil {
			return nil, fmt.Errorf("failed to update source task %s: %w", source.ID, err)
		}

		updatedTasks = append(updatedTasks, source)
	}

	// Update target task's inflows
	for _, sourceID := range sourceIDs {
		if !contains(targetTask.Inflows, sourceID) {
			targetTask.Inflows = append(targetTask.Inflows, sourceID)
		}
	}

	targetTask.AddStatusUpdate("Multiple tasks stitched to this target")

	if err := s.repo.Update(ctx, targetTask); err != nil {
		return nil, fmt.Errorf("failed to update target task: %w", err)
	}

	updatedTasks = append(updatedTasks, targetTask)
	return updatedTasks, nil
}

// ListTasks returns tasks by stage
func (s *TaskService) ListTasks(ctx context.Context, stage domain.TaskStage) ([]*domain.Task, error) {
	return s.repo.ListByStage(ctx, stage)
}

// ListTasksByUser returns tasks for a specific user, optionally filtered by stage
func (s *TaskService) ListTasksByUser(ctx context.Context, userID string, stage *domain.TaskStage) ([]*domain.Task, error) {
	var allTasks []*domain.Task
	var err error

	if stage != nil {
		allTasks, err = s.repo.ListByStage(ctx, *stage)
	} else {
		// Get tasks from all stages
		stages := []domain.TaskStage{
			domain.StagePending,
			domain.StageInbox,
			domain.StageStaging,
			domain.StageActive,
			domain.StageArchived,
		}

		for _, st := range stages {
			tasks, err := s.repo.ListByStage(ctx, st)
			if err != nil {
				return nil, err
			}
			allTasks = append(allTasks, tasks...)
		}
	}

	if err != nil {
		return nil, err
	}

	// Filter by user ID
	var userTasks []*domain.Task
	for _, task := range allTasks {
		if task.UserID == userID {
			userTasks = append(userTasks, task)
		}
	}

	return userTasks, nil
}

// GetTask retrieves a task by ID
func (s *TaskService) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateTaskTags updates the tags for a task
func (s *TaskService) UpdateTaskTags(ctx context.Context, taskID string, tags map[string]domain.TagValue) (*domain.Task, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Update tags
	for key, value := range tags {
		task.Tags[key] = value
	}

	task.AddStatusUpdate("Tags updated")

	err = s.repo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// SyncCalendar syncs tasks to Google Calendar for a user
func (s *TaskService) SyncCalendar(ctx context.Context, userID string) (int, []string, error) {
	if s.calendarService == nil {
		return 0, nil, fmt.Errorf("calendar service not configured")
	}
	if s.userRepo == nil {
		return 0, nil, fmt.Errorf("user repository not configured")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, nil, fmt.Errorf("user not found: %w", err)
	}

	if user.GoogleCalendarToken == "" {
		return 0, nil, fmt.Errorf("user has no Google Calendar token")
	}

	token, err := s.calendarService.TokenFromJSON(user.GoogleCalendarToken)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid Google Calendar token: %w", err)
	}

	// Get user's active tasks
	var tasks []*domain.Task
	if userAwareRepo, ok := s.repo.(interface {
		ListByStageAndUser(ctx context.Context, stage domain.TaskStage, userID string) ([]*domain.Task, error)
	}); ok {
		tasks, err = userAwareRepo.ListByStageAndUser(ctx, domain.StageActive, userID)
	} else {
		// Fallback: get all active tasks and filter by user
		allTasks, err := s.repo.ListByStage(ctx, domain.StageActive)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to get tasks: %w", err)
		}
		for _, task := range allTasks {
			if task.UserID == userID {
				tasks = append(tasks, task)
			}
		}
	}

	if err != nil {
		return 0, nil, fmt.Errorf("failed to get user tasks: %w", err)
	}

	// Sync to calendar
	synced, errors := s.calendarService.SyncTasksToCalendar(ctx, token, tasks, user.Email)

	// Update tasks with calendar event IDs
	for _, task := range tasks {
		if task.GoogleCalendarEventID != "" {
			if updateErr := s.repo.Update(ctx, task); updateErr != nil {
				errors = append(errors, fmt.Sprintf("Failed to update task %s: %v", task.ID, updateErr))
			}
		}
	}

	// Sync calendar changes back to tasks
	if s.syncEnabled {
		updatedTasks, syncErrors := s.calendarService.SyncCalendarToTasks(ctx, token, tasks)
		errors = append(errors, syncErrors...)

		for _, task := range updatedTasks {
			if updateErr := s.repo.Update(ctx, task); updateErr != nil {
				errors = append(errors, fmt.Sprintf("Failed to update task %s from calendar: %v", task.ID, updateErr))
			}
		}
	}

	return synced, errors, nil
}

// User management methods (if userRepo is available)

// CreateUser creates a new user
func (s *TaskService) CreateUser(ctx context.Context, id, email, name string, notificationSettings []domain.NotificationSetting) (*domain.User, error) {
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repository not configured")
	}

	if id == "" {
		id = domain.ShortID()
	}
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	user := &domain.User{
		ID:                   id,
		Email:                email,
		Name:                 name,
		NotificationSettings: notificationSettings,
	}

	err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (s *TaskService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repository not configured")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email address
func (s *TaskService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repository not configured")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// UpdateUser updates user information
func (s *TaskService) UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repository not configured")
	}

	err := s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return user, nil
}

// GetAllUsers retrieves all users
func (s *TaskService) GetAllUsers(ctx context.Context) ([]*domain.User, error) {
	if s.userRepo == nil {
		return []*domain.User{}, nil
	}
	return s.userRepo.ListAll(ctx)
}

// GetTaskDAG returns tasks in topological order for DAG visualization
func (s *TaskService) GetTaskDAG(ctx context.Context, userID string) ([]*domain.Task, error) {
	var tasks []*domain.Task
	var err error

	if userAwareRepo, ok := s.repo.(interface {
		ListByUser(ctx context.Context, userID string) ([]*domain.Task, error)
	}); ok && userID != "" {
		tasks, err = userAwareRepo.ListByUser(ctx, userID)
	} else {
		// Fallback: get all tasks and filter by user
		tasks, err = s.repo.ListAll(ctx)
		if err == nil && userID != "" {
			var filteredTasks []*domain.Task
			for _, task := range tasks {
				if task.UserID == userID {
					filteredTasks = append(filteredTasks, task)
				}
			}
			tasks = filteredTasks
		}
	}

	if err != nil {
		return nil, err
	}

	// Perform topological sort
	return s.topologicalSort(tasks), nil
}

// CheckDueReminders checks for tasks with upcoming due dates and sends notifications
func (s *TaskService) CheckDueReminders(ctx context.Context) error {
	if s.emailService == nil || s.userRepo == nil {
		return nil // Services not configured
	}

	users, err := s.userRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	tasks, err := s.repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	return s.emailService.CheckAndSendDueReminders(users, tasks)
}

// syncTaskToCalendar syncs a single task to calendar (async)
func (s *TaskService) syncTaskToCalendar(task *domain.Task) {
	if s.calendarService == nil || s.userRepo == nil {
		return
	}

	ctx := context.Background()
	user, err := s.userRepo.GetByID(ctx, task.UserID)
	if err != nil {
		fmt.Printf("Failed to get user for calendar sync: %v\n", err)
		return
	}

	if user.GoogleCalendarToken == "" {
		return // No token configured
	}

	token, err := s.calendarService.TokenFromJSON(user.GoogleCalendarToken)
	if err != nil {
		fmt.Printf("Invalid calendar token for user %s: %v\n", user.ID, err)
		return
	}

	eventID, err := s.calendarService.CreateOrUpdateEvent(ctx, token, task, user.Email)
	if err != nil {
		fmt.Printf("Failed to sync task %s to calendar: %v\n", task.ID, err)
		return
	}

	// Update task with event ID
	task.GoogleCalendarEventID = eventID
	if err := s.repo.Update(ctx, task); err != nil {
		fmt.Printf("Failed to update task %s with calendar event ID: %v\n", task.ID, err)
	}
}

// topologicalSort performs topological sorting on tasks based on dependencies
func (s *TaskService) topologicalSort(tasks []*domain.Task) []*domain.Task {
	// Create maps for easy lookup
	taskMap := make(map[string]*domain.Task)
	inDegree := make(map[string]int)

	// Initialize
	for _, task := range tasks {
		taskMap[task.ID] = task
		inDegree[task.ID] = len(task.Inflows)
	}

	// Kahn's algorithm
	var queue []*domain.Task
	var result []*domain.Task

	// Find all nodes with no incoming edges
	for _, task := range tasks {
		if inDegree[task.ID] == 0 {
			queue = append(queue, task)
		}
	}

	// Process queue
	for len(queue) > 0 {
		// Remove a node from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each outflow (dependent task)
		for _, outflowID := range current.Outflows {
			inDegree[outflowID]--
			if inDegree[outflowID] == 0 {
				if task, exists := taskMap[outflowID]; exists {
					queue = append(queue, task)
				}
			}
		}
	}

	return result
}

// Helper methods

// checkInboxConstraint verifies that moving a task to inbox doesn't exceed the limit
func (s *TaskService) checkInboxConstraint(ctx context.Context) error {
	inboxTasks, err := s.ListTasks(ctx, domain.StageInbox)
	if err != nil {
		return fmt.Errorf("failed to check inbox constraint: %w", err)
	}

	if len(inboxTasks) >= s.maxInboxSize {
		return fmt.Errorf("inbox is full (max %d tasks)", s.maxInboxSize)
	}

	return nil
}

// checkDependencies verifies that all dependencies are completed
func (s *TaskService) checkDependencies(ctx context.Context, task *domain.Task) error {
	if len(task.Inflows) == 0 {
		return nil // No dependencies
	}

	dependencies, err := s.repo.GetTasksByIDs(ctx, task.Inflows)
	if err != nil {
		return fmt.Errorf("failed to check dependencies: %w", err)
	}

	for _, dep := range dependencies {
		if dep.Status != domain.StatusCompleted {
			return fmt.Errorf("dependency %s is not completed", dep.ID)
		}
	}

	return nil
}

// validateSameChain checks if two tasks are in the same location hierarchy
func (s *TaskService) validateSameChain(ctx context.Context, task1, task2 *domain.Task) error {
	if len(task1.Location) != len(task2.Location) {
		return fmt.Errorf("tasks are not in the same chain")
	}

	for i, loc := range task1.Location {
		if task2.Location[i] != loc {
			return fmt.Errorf("tasks are not in the same chain")
		}
	}

	return nil
}

// checkChainHopping prevents creating cycles in task dependencies
func (s *TaskService) checkChainHopping(ctx context.Context, source, target *domain.Task) error {
	if s.isTransitivelyDependent(ctx, target.ID, source.ID) {
		return fmt.Errorf("would create chain hopping: %s already depends on %s transitively", target.ID, source.ID)
	}
	return nil
}

// isTransitivelyDependent checks if taskID transitively depends on dependencyID
func (s *TaskService) isTransitivelyDependent(ctx context.Context, taskID, dependencyID string) bool {
	visited := make(map[string]bool)
	return s.hasTransitiveDependency(ctx, taskID, dependencyID, visited)
}

// hasTransitiveDependency recursively checks for transitive dependencies
func (s *TaskService) hasTransitiveDependency(ctx context.Context, taskID, dependencyID string, visited map[string]bool) bool {
	if visited[taskID] {
		return false // Avoid cycles
	}
	visited[taskID] = true

	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return false
	}

	for _, inflowID := range task.Inflows {
		if inflowID == dependencyID {
			return true
		}
		if s.hasTransitiveDependency(ctx, inflowID, dependencyID, visited) {
			return true
		}
	}

	return false
}

// updateDependenciesForMerge updates task dependencies when merging tasks
func (s *TaskService) updateDependenciesForMerge(ctx context.Context, fromTask, toTask *domain.Task) error {
	allTasks, err := s.repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all tasks: %w", err)
	}

	for _, task := range allTasks {
		updated := false

		// Update inflows: replace references to fromTask with toTask
		for i, inflow := range task.Inflows {
			if inflow == fromTask.ID {
				task.Inflows[i] = toTask.ID
				updated = true
			}
		}

		// Update outflows: replace references to fromTask with toTask
		for i, outflow := range task.Outflows {
			if outflow == fromTask.ID {
				task.Outflows[i] = toTask.ID
				updated = true
			}
		}

		if updated {
			if err := s.repo.Update(ctx, task); err != nil {
				return fmt.Errorf("failed to update task dependencies: %w", err)
			}
		}
	}

	return nil
}

// updateDependenciesForSplit updates task dependencies when splitting a task
func (s *TaskService) updateDependenciesForSplit(ctx context.Context, originalTask *domain.Task, newTasks []*domain.Task) error {
	allTasks, err := s.repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all tasks: %w", err)
	}

	// The first new task inherits the original task's dependencies
	if len(newTasks) > 0 {
		firstTask := newTasks[0]

		for _, task := range allTasks {
			updated := false

			// Update inflows: replace references to originalTask with firstTask
			for i, inflow := range task.Inflows {
				if inflow == originalTask.ID {
					task.Inflows[i] = firstTask.ID
					updated = true
				}
			}

			// Update outflows: replace references to originalTask with firstTask
			for i, outflow := range task.Outflows {
				if outflow == originalTask.ID {
					task.Outflows[i] = firstTask.ID
					updated = true
				}
			}

			if updated {
				if err := s.repo.Update(ctx, task); err != nil {
					return fmt.Errorf("failed to update task dependencies: %w", err)
				}
			}
		}

		// Set up dependencies between the new tasks (sequential)
		for i := 0; i < len(newTasks)-1; i++ {
			currentTask := newTasks[i]
			nextTask := newTasks[i+1]

			currentTask.Outflows = append(currentTask.Outflows, nextTask.ID)
			nextTask.Inflows = append(nextTask.Inflows, currentTask.ID)

			if err := s.repo.Update(ctx, currentTask); err != nil {
				return fmt.Errorf("failed to update task dependencies: %w", err)
			}
			if err := s.repo.Update(ctx, nextTask); err != nil {
				return fmt.Errorf("failed to update task dependencies: %w", err)
			}
		}
	}

	return nil
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetAllTasks retrieves all tasks regardless of stage or user
func (s *TaskService) GetAllTasks(ctx context.Context) ([]*domain.Task, error) {
	return s.repo.ListAll(ctx)
}
