package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
)

func (s *TaskService) RegisterSystem(system TaskSystem) error {
	err := s.systemRegistry.Register(system)
	if err != nil {
		return err
	}

	return s.taskFrame.RegisterSystemName(system.Name())
}

func (s *TaskService) RegisteredSystems() []string {
	return s.systemRegistry.ListNames()
}

func (s *TaskService) DefineTaskDomain(domain TaskDomain) error {
	return s.taskFrame.DefineDomain(domain)
}

func (s *TaskService) DefineTaskFactory(factory TaskFactoryFrame) error {
	return s.taskFrame.DefineFactory(factory)
}

func (s *TaskService) DefineTaskResult(result TaskResultRequirement) error {
	return s.taskFrame.DefineResult(result)
}

func (s *TaskService) DefineTaskResource(resource TaskResourceDependency) error {
	return s.taskFrame.DefineResource(resource)
}

func (s *TaskService) DefineTrait(trait TraitDefinition) error {
	return s.taskFrame.DefineTrait(trait)
}

func (s *TaskService) beforeAction(
	ctx context.Context,
	action TaskAction,
	task *domain.Task,
	actorUserID string,
	metadata map[string]string,
) error {
	denied, err := s.systemRegistry.Before(ctx, TaskActionContext{
		Action:      action,
		Task:        task,
		ActorUserID: actorUserID,
		OccurredAt:  time.Now().UTC(),
		Metadata:    metadata,
	})
	if err != nil {
		return err
	}
	if denied == nil {
		return nil
	}
	return denied
}

func (s *TaskService) afterAction(
	ctx context.Context,
	action TaskAction,
	task *domain.Task,
	actorUserID string,
	metadata map[string]string,
) {
	s.systemRegistry.ReindexTask(task)
	err := s.systemRegistry.After(ctx, TaskActionContext{
		Action:      action,
		Task:        task,
		ActorUserID: actorUserID,
		OccurredAt:  time.Now().UTC(),
		Metadata:    metadata,
	})
	if err == nil {
		return
	}
	s.logger.WithError(err).WithFields(map[string]interface{}{
		"action": string(action),
		"task_id": func() string {
			if task == nil {
				return ""
			}
			return task.ID
		}(),
	}).Warn("post-action system hook failed")
}

func (s *TaskService) runSimpleActionHooks(
	ctx context.Context,
	action TaskAction,
	task *domain.Task,
	actorUserID string,
	metadata map[string]string,
) error {
	err := s.beforeAction(ctx, action, task, actorUserID, metadata)
	if err == nil {
		return nil
	}

	var deniedErr *ActionDeniedError
	if !AsActionDeniedError(err, &deniedErr) {
		return err
	}

	return fmt.Errorf("task action denied: system=%s message=%s", deniedErr.SystemName, deniedErr.Message)
}

func AsActionDeniedError(err error, target **ActionDeniedError) bool {
	if err == nil {
		return false
	}
	denied, ok := err.(*ActionDeniedError)
	if !ok {
		return false
	}
	*target = denied
	return true
}
