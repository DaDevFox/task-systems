package service

import (
	"context"
	"fmt"
	"strconv"
)

type PomodoroSystem struct {
	userRepo repositoryUserLookup
}

func NewPomodoroSystem(userRepo repositoryUserLookup) *PomodoroSystem {
	return &PomodoroSystem{userRepo: userRepo}
}

func (s *PomodoroSystem) Name() string {
	return "pomodoro"
}

func (s *PomodoroSystem) TrackedTraits() []string {
	return []string{"tag.pomodoro.enabled", "tag.pomodoro.session_minutes"}
}

func (s *PomodoroSystem) BeforeTaskAction(ctx context.Context, actionCtx TaskActionContext) (*ActionDeniedError, error) {
	if actionCtx.Action != TaskActionStartTask {
		return nil, nil
	}
	if actionCtx.Task == nil {
		return nil, nil
	}

	enabled := actionCtx.Task.Tags["pomodoro.enabled"].TextValue
	if enabled != "true" {
		return nil, nil
	}

	rawMinutes := actionCtx.Task.Tags["pomodoro.session_minutes"].TextValue
	if rawMinutes == "" && s.userRepo != nil {
		user, userErr := s.userRepo.GetByID(ctx, actionCtx.Task.UserID)
		if userErr == nil && user.SystemSettings != nil {
			rawMinutes = user.SystemSettings["pomodoro.session_minutes"]
		}
	}
	if rawMinutes == "" {
		return nil, nil
	}

	minutes, err := strconv.Atoi(rawMinutes)
	if err != nil {
		return &ActionDeniedError{SystemName: s.Name(), Message: fmt.Sprintf("invalid pomodoro.session_minutes value: %v", err)}, nil
	}
	if minutes > 0 {
		return nil, nil
	}

	return &ActionDeniedError{SystemName: s.Name(), Message: "pomodoro.session_minutes must be > 0"}, nil
}

func (s *PomodoroSystem) AfterTaskAction(ctx context.Context, actionCtx TaskActionContext) error {
	if actionCtx.Task == nil {
		return nil
	}
	if actionCtx.Action != TaskActionStartTask {
		return nil
	}
	if actionCtx.Task.Tags["pomodoro.enabled"].TextValue != "true" {
		return nil
	}

	actionCtx.Task.AddStatusUpdate("Pomodoro session started")
	return nil
}

var _ TaskSystem = (*PomodoroSystem)(nil)
