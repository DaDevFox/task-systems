package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
)

func (s *TaskService) CreateTaskFrame(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if task == nil {
		return nil, fmt.Errorf("task is required")
	}
	if task.Name == "" {
		return nil, fmt.Errorf("task name is required")
	}

	owner := task.UserID
	if owner == "" && len(task.Assignees) > 0 {
		owner = task.Assignees[0]
	}
	if owner == "" {
		owner = "default-user"
	}

	created, err := s.AddTaskForUser(ctx, task.Name, task.Description, owner)
	if err != nil {
		return nil, err
	}

	created.DomainID = task.DomainID
	created.Status = task.Status
	if created.Status == domain.StatusUnspecified {
		created.Status = domain.StatusTodo
	}
	created.Assignees = task.Assignees
	if len(created.Assignees) == 0 {
		created.Assignees = []string{owner}
	}
	created.Results = task.Results
	created.Resources = task.Resources

	if task.Tags != nil {
		created.Tags = task.Tags
	}

	err = s.repo.Update(ctx, created)
	if err != nil {
		return nil, fmt.Errorf("failed to persist objective task fields: %w", err)
	}

	return created, nil
}

func (s *TaskService) UpdateTaskFrame(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if task == nil {
		return nil, fmt.Errorf("task is required")
	}
	if task.ID == "" {
		return nil, fmt.Errorf("task id is required")
	}

	existing, err := s.repo.GetByID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	existing.Name = task.Name
	existing.Description = task.Description
	existing.DomainID = task.DomainID
	if task.Status != domain.StatusUnspecified {
		existing.Status = task.Status
	}
	existing.Assignees = task.Assignees
	existing.Results = task.Results
	existing.Resources = task.Resources
	if task.Tags != nil {
		existing.Tags = task.Tags
	}

	err = s.repo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return existing, nil
}

func (s *TaskService) DeleteTaskFrame(ctx context.Context, taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	return s.repo.Delete(ctx, taskID)
}

func (s *TaskService) ListTasksByDomain(ctx context.Context, domainID string, pageSize int32, pageToken string) ([]*domain.Task, string, error) {
	tasks, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, "", err
	}

	filtered := make([]*domain.Task, 0, len(tasks))
	for _, task := range tasks {
		if domainID == "" || task.DomainID == domainID {
			filtered = append(filtered, task)
		}
	}

	start := 0
	if pageToken != "" {
		parsed, parseErr := strconv.Atoi(pageToken)
		if parseErr == nil && parsed >= 0 {
			start = parsed
		}
	}
	if start >= len(filtered) {
		return []*domain.Task{}, "", nil
	}

	if pageSize <= 0 {
		return filtered[start:], "", nil
	}

	end := start + int(pageSize)
	if end >= len(filtered) {
		return filtered[start:], "", nil
	}

	nextPage := strconv.Itoa(end)
	return filtered[start:end], nextPage, nil
}

func (s *TaskService) CompleteTaskFrame(ctx context.Context, taskID string) (*domain.Task, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	task.Status = domain.StatusCompleted
	task.AddStatusUpdate("Task completed")
	err = s.repo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	return task, nil
}

func (s *TaskService) ResolveTaskIDByInput(ctx context.Context, input string) (string, string, []string, error) {
	if input == "" {
		return "", "", nil, fmt.Errorf("task input is required")
	}

	tasks, err := s.repo.ListAll(ctx)
	if err != nil {
		return "", "", nil, err
	}

	exact := ""
	suggestions := []string{}
	for _, task := range tasks {
		if task.ID == input {
			exact = task.ID
			break
		}
		if strings.HasPrefix(task.ID, input) {
			suggestions = append(suggestions, task.ID)
		}
	}

	if exact != "" {
		return exact, input, []string{}, nil
	}
	if len(suggestions) == 1 {
		return suggestions[0], input, []string{}, nil
	}
	if len(suggestions) > 1 {
		return "", "", suggestions, fmt.Errorf("ambiguous task id prefix")
	}
	return "", "", []string{}, fmt.Errorf("task not found")
}
