package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/calendar"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/email"
	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/repository"
)

// EnhancedTaskService provides comprehensive business logic for task management
type EnhancedTaskService struct {
	taskRepo        repository.TaskRepository
	userRepo        repository.UserRepository
	calendarService *calendar.CalendarService
	emailService    *email.EmailService
	maxInboxSize    int
	syncEnabled     bool
}

// NewEnhancedTaskService creates a new enhanced task service
func NewEnhancedTaskService(
	taskRepo repository.TaskRepository,
	userRepo repository.UserRepository,
	calendarService *calendar.CalendarService,
	emailService *email.EmailService,
	maxInboxSize int,
) *EnhancedTaskService {
	if maxInboxSize <= 0 {
		maxInboxSize = 5 // default
	}
	return &EnhancedTaskService{
		taskRepo:        taskRepo,
		userRepo:        userRepo,
		calendarService: calendarService,
		emailService:    emailService,
		maxInboxSize:    maxInboxSize,
		syncEnabled:     true,
	}
}

// SetSyncEnabled enables or disables automatic calendar sync
func (s *EnhancedTaskService) SetSyncEnabled(enabled bool) {
	s.syncEnabled = enabled
}

// AddTask creates a new task in inbox stage for a specific user
func (s *EnhancedTaskService) AddTask(ctx context.Context, name, description, userID string) (*domain.Task, error) {
	if name == "" {
		return nil, fmt.Errorf("task name cannot be empty")
	}
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Check inbox constraint before adding task
	if err := s.checkInboxConstraint(ctx); err != nil {
		return nil, err
	}

	// Verify user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	task := domain.NewTask(name, description, userID)
	task.Stage = domain.StageInbox // New tasks go to inbox first
	task.AddStatusUpdate("Task created")

	err = s.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Send assignment notification
	if s.emailService != nil {
		if err := s.emailService.SendTaskAssignedNotification(user, task); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to send assignment notification: %v\n", err)
		}
	}

	return task, nil
}

// MoveToStaging moves a task from pending/inbox to staging with optional tag updates
func (s *EnhancedTaskService) MoveToStaging(ctx context.Context, sourceID string, destinationID *string, newLocation []string, points []domain.Point, tags map[string]domain.TagValue) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("source task not found: %w", err)
	}

	if err := task.CanMoveToStaging(); err != nil {
		return nil, err
	}

	// Handle destination logic
	if destinationID != nil {
		destTask, err := s.taskRepo.GetByID(ctx, *destinationID)
		if err != nil {
			return nil, fmt.Errorf("destination task not found: %w", err)
		}

		// Inherit location from destination
		task.Location = destTask.Location

		// Set up dependency: source depends on destination
		task.Inflows = append(task.Inflows, *destinationID)
		destTask.Outflows = append(destTask.Outflows, sourceID)

		// Update destination task
		if err := s.taskRepo.Update(ctx, destTask); err != nil {
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

	// Update tags if provided
	if len(tags) > 0 {
		for key, value := range tags {
			task.Tags[key] = value
		}
	}

	// Move to staging
	task.Stage = domain.StageStaging
	task.AddStatusUpdate("Moved to staging")

	err = s.taskRepo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// UpdateTaskTags updates the tags for a task
func (s *EnhancedTaskService) UpdateTaskTags(ctx context.Context, taskID string, tags map[string]domain.TagValue) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Update tags
	for key, value := range tags {
		task.Tags[key] = value
	}

	task.AddStatusUpdate("Tags updated")

	err = s.taskRepo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// StartTask begins work on a task
func (s *EnhancedTaskService) StartTask(ctx context.Context, id string) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	if err := task.CanStart(); err != nil {
		return nil, err
	}

	// Create work interval
	now := time.Now()
	interval := domain.WorkInterval{
		Start:           now,
		PointsCompleted: []domain.Point{},
	}

	task.Schedule.WorkIntervals = append(task.Schedule.WorkIntervals, interval)
	task.Stage = domain.StageActive
	task.Status = domain.StatusInProgress
	task.AddStatusUpdate("Task started")

	err = s.taskRepo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// Sync to calendar if enabled
	if s.syncEnabled && s.calendarService != nil {
		go s.syncTaskToCalendar(task)
	}

	// Send start notification
	if s.emailService != nil {
		user, userErr := s.userRepo.GetByID(ctx, task.UserID)
		if userErr == nil {
			if err := s.emailService.SendTaskStartedNotification(user, task); err != nil {
				fmt.Printf("Failed to send start notification: %v\n", err)
			}
		}
	}

	return task, nil
}

// SyncCalendar syncs tasks to Google Calendar for a user
func (s *EnhancedTaskService) SyncCalendar(ctx context.Context, userID string) (int, []string, error) {
	if s.calendarService == nil {
		return 0, nil, fmt.Errorf("calendar service not configured")
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
	tasks, err := s.taskRepo.ListByStageAndUser(ctx, domain.StageActive, userID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get user tasks: %w", err)
	}

	// Sync to calendar
	synced, errors := s.calendarService.SyncTasksToCalendar(ctx, token, tasks, user.Email)

	// Update tasks with calendar event IDs
	for _, task := range tasks {
		if task.GoogleCalendarEventID != "" {
			if updateErr := s.taskRepo.Update(ctx, task); updateErr != nil {
				errors = append(errors, fmt.Sprintf("Failed to update task %s: %v", task.ID, updateErr))
			}
		}
	}

	// Sync calendar changes back to tasks
	if s.syncEnabled {
		updatedTasks, syncErrors := s.calendarService.SyncCalendarToTasks(ctx, token, tasks)
		errors = append(errors, syncErrors...)

		for _, task := range updatedTasks {
			if updateErr := s.taskRepo.Update(ctx, task); updateErr != nil {
				errors = append(errors, fmt.Sprintf("Failed to update task %s from calendar: %v", task.ID, updateErr))
			}
		}
	}

	return synced, errors, nil
}

// CreateUser creates a new user
func (s *EnhancedTaskService) CreateUser(ctx context.Context, id, email, name string, notificationSettings []domain.NotificationSetting) (*domain.User, error) {
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
func (s *EnhancedTaskService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// UpdateUser updates user information
func (s *EnhancedTaskService) UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	err := s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return user, nil
}

// ListTasksByUser returns all tasks for a specific user
func (s *EnhancedTaskService) ListTasksByUser(ctx context.Context, userID string, stage *domain.TaskStage) ([]*domain.Task, error) {
	if stage != nil {
		return s.taskRepo.ListByStageAndUser(ctx, *stage, userID)
	}
	return s.taskRepo.ListByUser(ctx, userID)
}

// GetTaskDAG returns tasks in topological order for DAG visualization
func (s *EnhancedTaskService) GetTaskDAG(ctx context.Context, userID string) ([]*domain.Task, error) {
	tasks, err := s.taskRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Perform topological sort
	return s.topologicalSort(tasks), nil
}

// CheckDueReminders checks for tasks with upcoming due dates and sends notifications
func (s *EnhancedTaskService) CheckDueReminders(ctx context.Context) error {
	if s.emailService == nil {
		return nil // No email service configured
	}

	users, err := s.userRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	tasks, err := s.taskRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	return s.emailService.CheckAndSendDueReminders(users, tasks)
}

// syncTaskToCalendar syncs a single task to calendar (async)
func (s *EnhancedTaskService) syncTaskToCalendar(task *domain.Task) {
	if s.calendarService == nil {
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
	if err := s.taskRepo.Update(ctx, task); err != nil {
		fmt.Printf("Failed to update task %s with calendar event ID: %v\n", task.ID, err)
	}
}

// topologicalSort performs topological sorting on tasks based on dependencies
func (s *EnhancedTaskService) topologicalSort(tasks []*domain.Task) []*domain.Task {
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

// checkInboxConstraint ensures inbox doesn't exceed maximum size
func (s *EnhancedTaskService) checkInboxConstraint(ctx context.Context) error {
	count, err := s.taskRepo.CountByStage(ctx, domain.StageInbox)
	if err != nil {
		return fmt.Errorf("failed to check inbox size: %w", err)
	}

	if count >= s.maxInboxSize {
		return fmt.Errorf("inbox is full (max %d tasks), cannot add more tasks", s.maxInboxSize)
	}

	return nil
}
