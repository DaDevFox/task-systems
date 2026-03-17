package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
)

// TaskAction identifies the lifecycle action being processed.
type TaskAction string

const (
	TaskActionAddTask        TaskAction = "add_task"
	TaskActionMoveToStaging  TaskAction = "move_to_staging"
	TaskActionStartTask      TaskAction = "start_task"
	TaskActionStopTask       TaskAction = "stop_task"
	TaskActionCompleteTask   TaskAction = "complete_task"
	TaskActionMergeTasks     TaskAction = "merge_tasks"
	TaskActionSplitTask      TaskAction = "split_task"
	TaskActionAdvertiseTask  TaskAction = "advertise_task"
	TaskActionStitchTasks    TaskAction = "stitch_tasks"
	TaskActionUpdateTaskTags TaskAction = "update_task_tags"
)

// TaskActionContext is passed to systems before and after task actions.
type TaskActionContext struct {
	Action      TaskAction
	Task        *domain.Task
	ActorUserID string
	OccurredAt  time.Time
	Metadata    map[string]string
}

// ActionDeniedError represents a veto from a registered system.
type ActionDeniedError struct {
	SystemName string
	Message    string
}

func (e *ActionDeniedError) Error() string {
	return fmt.Sprintf("action denied by system=%s message=%s", e.SystemName, e.Message)
}

// TaskSystem is the extension contract for pluggable systems.
type TaskSystem interface {
	Name() string
	TrackedTraits() []string
	BeforeTaskAction(ctx context.Context, actionCtx TaskActionContext) (*ActionDeniedError, error)
	AfterTaskAction(ctx context.Context, actionCtx TaskActionContext) error
}

// SystemRegistry stores systems and a trait index for quick system-side lookups.
type SystemRegistry struct {
	mu         sync.RWMutex
	systems    []TaskSystem
	systemsMap map[string]TaskSystem
	traitIndex map[string]map[string]map[string]struct{}
}

func NewSystemRegistry() *SystemRegistry {
	return &SystemRegistry{
		systems:    []TaskSystem{},
		systemsMap: map[string]TaskSystem{},
		traitIndex: map[string]map[string]map[string]struct{}{},
	}
}

func (r *SystemRegistry) Register(system TaskSystem) error {
	if system == nil {
		return fmt.Errorf("system cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.TrimSpace(system.Name())
	if name == "" {
		return fmt.Errorf("system name cannot be empty")
	}

	if _, exists := r.systemsMap[name]; exists {
		return fmt.Errorf("system already registered: %s", name)
	}

	r.systems = append(r.systems, system)
	r.systemsMap[name] = system
	return nil
}

func (r *SystemRegistry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.systemsMap))
	for name := range r.systemsMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *SystemRegistry) Before(ctx context.Context, actionCtx TaskActionContext) (*ActionDeniedError, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, system := range r.systems {
		denied, err := system.BeforeTaskAction(ctx, actionCtx)
		if err != nil {
			return nil, fmt.Errorf("before hook failed for system %s: %w", system.Name(), err)
		}
		if denied == nil {
			continue
		}
		if denied.SystemName == "" {
			denied.SystemName = system.Name()
		}
		return denied, nil
	}

	return nil, nil
}

func (r *SystemRegistry) After(ctx context.Context, actionCtx TaskActionContext) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, system := range r.systems {
		err := system.AfterTaskAction(ctx, actionCtx)
		if err != nil {
			return fmt.Errorf("after hook failed for system %s: %w", system.Name(), err)
		}
	}

	return nil
}

func (r *SystemRegistry) ReindexTask(task *domain.Task) {
	if task == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for trait, values := range r.traitIndex {
		for value, ids := range values {
			if _, exists := ids[task.ID]; !exists {
				continue
			}
			delete(ids, task.ID)
			if len(ids) == 0 {
				delete(values, value)
			}
		}
		if len(values) == 0 {
			delete(r.traitIndex, trait)
		}
	}

	traits := extractTaskTraits(task)
	for trait, value := range traits {
		if _, exists := r.traitIndex[trait]; !exists {
			r.traitIndex[trait] = map[string]map[string]struct{}{}
		}
		if _, exists := r.traitIndex[trait][value]; !exists {
			r.traitIndex[trait][value] = map[string]struct{}{}
		}
		r.traitIndex[trait][value][task.ID] = struct{}{}
	}
}

func (r *SystemRegistry) TasksByTraitValue(trait, value string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	values, exists := r.traitIndex[trait]
	if !exists {
		return []string{}
	}

	ids, exists := values[value]
	if !exists {
		return []string{}
	}

	result := make([]string, 0, len(ids))
	for taskID := range ids {
		result = append(result, taskID)
	}
	sort.Strings(result)
	return result
}

func extractTaskTraits(task *domain.Task) map[string]string {
	traits := map[string]string{}
	traits["stage"] = task.Stage.String()
	traits["status"] = task.Status.String()
	traits["user_id"] = task.UserID

	if len(task.Location) > 0 {
		traits["location"] = strings.Join(task.Location, "/")
	}

	for key, value := range task.Tags {
		traits["tag."+key] = value.String()
	}

	return traits
}
