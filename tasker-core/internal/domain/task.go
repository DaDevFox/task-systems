package domain

import (
	"strings"
	"github.com/google/uuid"
)

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
