package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TagType represents the type of a tag value
type TagType int

const (
	TagTypeUnspecified TagType = iota
	TagTypeText
	TagTypeLocation
	TagTypeTime
)

func (t TagType) String() string {
	switch t {
	case TagTypeText:
		return "text"
	case TagTypeLocation:
		return "location"
	case TagTypeTime:
		return "time"
	default:
		return "unspecified"
	}
}

// GeographicLocation represents a geographic location
type GeographicLocation struct {
	Latitude  float64
	Longitude float64
	Address   string
}

// TagValue represents a typed tag value
type TagValue struct {
	Type          TagType
	TextValue     string
	LocationValue *GeographicLocation
	TimeValue     *time.Time
}

// NotificationType represents types of notifications
type NotificationType int

const (
	NotificationTypeUnspecified NotificationType = iota
	NotificationOnAssign
	NotificationOnStart
	NotificationNDaysBeforeDue
)

func (n NotificationType) String() string {
	switch n {
	case NotificationOnAssign:
		return "on_assign"
	case NotificationOnStart:
		return "on_start"
	case NotificationNDaysBeforeDue:
		return "n_days_before_due"
	default:
		return "unspecified"
	}
}

// NotificationSetting represents user notification preferences
type NotificationSetting struct {
	Type       NotificationType
	Enabled    bool
	DaysBefore int32 // For N_DAYS_BEFORE_DUE type
}

// User represents a user in the system
type User struct {
	ID                   string
	Email                string
	Name                 string
	GoogleCalendarToken  string
	NotificationSettings []NotificationSetting
}

// ShortID generates a short unique identifier from a UUID
func ShortID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
}

// TaskStage represents the lifecycle stage of a task
type TaskStage int

const (
	StagePending TaskStage = iota
	StageInbox
	StageStaging
	StageActive
	StageArchived
)

func (s TaskStage) String() string {
	switch s {
	case StagePending:
		return "pending"
	case StageInbox:
		return "inbox"
	case StageStaging:
		return "staging"
	case StageActive:
		return "active"
	case StageArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// TaskStatus represents the detailed status within a stage
type TaskStatus int

const (
	StatusUnspecified TaskStatus = iota
	StatusTodo
	StatusInProgress
	StatusPaused
	StatusBlocked
	StatusCompleted
	StatusCancelled
)

func (s TaskStatus) String() string {
	switch s {
	case StatusTodo:
		return "todo"
	case StatusInProgress:
		return "in_progress"
	case StatusPaused:
		return "paused"
	case StatusBlocked:
		return "blocked"
	case StatusCompleted:
		return "completed"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unspecified"
	}
}

// Point represents a work tracking unit
type Point struct {
	Title string
	Value uint32
}

// WorkInterval represents a scheduled work period
type WorkInterval struct {
	Start           time.Time
	Stop            time.Time
	PointsCompleted []Point
}

// Schedule contains scheduling information for a task
type Schedule struct {
	WorkIntervals []WorkInterval
	Due           time.Time
}

// StatusUpdate represents a status change event
type StatusUpdate struct {
	Time   time.Time
	Update string
}

// Status tracks the history of status updates
type Status struct {
	Updates []StatusUpdate
}

// Task represents a unit of work in the system
type Task struct {
	ID                    string
	Name                  string
	Description           string
	Stage                 TaskStage
	Status                TaskStatus
	Location              []string            // hierarchical path
	Points                []Point             // work units to complete
	Schedule              Schedule            // scheduling information
	StatusHist            Status              // status update history
	Tags                  map[string]TagValue // user configurable metadata with types
	Inflows               []string            // task IDs this depends on
	Outflows              []string            // task IDs that depend on this
	UserID                string              // owner of the task
	GoogleCalendarEventID string              // for calendar sync
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// NewTask creates a new task in pending stage
func NewTask(name, description, userID string) *Task {
	now := time.Now()
	return &Task{
		ID:          ShortID(),
		Name:        name,
		Description: description,
		Stage:       StagePending,
		Status:      StatusTodo,
		Location:    []string{},
		Points:      []Point{},
		Schedule:    Schedule{},
		StatusHist:  Status{Updates: []StatusUpdate{}},
		Tags:        make(map[string]TagValue),
		Inflows:     []string{},
		Outflows:    []string{},
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// TotalPoints calculates the total point value for the task
func (t *Task) TotalPoints() uint32 {
	var total uint32
	for _, point := range t.Points {
		total += point.Value
	}
	return total
}

// CompletedPoints calculates the total completed points across all work intervals
func (t *Task) CompletedPoints() uint32 {
	var total uint32
	for _, interval := range t.Schedule.WorkIntervals {
		for _, point := range interval.PointsCompleted {
			total += point.Value
		}
	}
	return total
}

// IsComplete returns true if all points have been completed
func (t *Task) IsComplete() bool {
	return t.CompletedPoints() >= t.TotalPoints() && t.TotalPoints() > 0
}

// CanMoveToStaging validates if a task can be moved to staging
func (t *Task) CanMoveToStaging() error {
	if t.Stage != StagePending && t.Stage != StageInbox {
		return fmt.Errorf("task %s is in stage %s, can only move to staging from pending or inbox", t.ID, t.Stage)
	}
	return nil
}

// CanStart validates if a task can be started
func (t *Task) CanStart() error {
	if t.Stage != StageStaging {
		return fmt.Errorf("task %s is in stage %s, can only start tasks in staging", t.ID, t.Stage)
	}
	if t.Status == StatusInProgress {
		return fmt.Errorf("task %s is already in progress", t.ID)
	}
	if t.Status == StatusCompleted {
		return fmt.Errorf("task %s is already completed", t.ID)
	}
	return nil
}

// CanStop validates if a task can be stopped
func (t *Task) CanStop() error {
	if t.Status != StatusInProgress {
		return fmt.Errorf("task %s is not in progress, current status: %s", t.ID, t.Status)
	}
	return nil
}

// AddStatusUpdate adds a new status update
func (t *Task) AddStatusUpdate(update string) {
	t.StatusHist.Updates = append(t.StatusHist.Updates, StatusUpdate{
		Time:   time.Now(),
		Update: update,
	})
	t.UpdatedAt = time.Now()
}

// LocationPath returns the location as a path string
func (t *Task) LocationPath() string {
	return strings.Join(t.Location, "/")
}
