package service

import (
	"context"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
)

type CalendarSyncSystem struct {
	userRepo repositoryUserLookup
}

type repositoryUserLookup interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

func NewCalendarSyncSystem(userRepo repositoryUserLookup) *CalendarSyncSystem {
	return &CalendarSyncSystem{userRepo: userRepo}
}

func (s *CalendarSyncSystem) Name() string {
	return "calendar_sync"
}

func (s *CalendarSyncSystem) TrackedTraits() []string {
	return []string{"tag.calendar.sync_required", "user_id"}
}

func (s *CalendarSyncSystem) BeforeTaskAction(ctx context.Context, actionCtx TaskActionContext) (*ActionDeniedError, error) {
	if actionCtx.Action != TaskActionStartTask {
		return nil, nil
	}
	if actionCtx.Task == nil {
		return nil, nil
	}
	if actionCtx.Task.Tags["calendar.sync_required"].TextValue != "true" {
		return nil, nil
	}
	if s.userRepo == nil {
		return &ActionDeniedError{SystemName: s.Name(), Message: "calendar sync requires configured user repository"}, nil
	}

	user, err := s.userRepo.GetByID(ctx, actionCtx.Task.UserID)
	if err != nil {
		return &ActionDeniedError{SystemName: s.Name(), Message: "calendar sync requires a resolvable user"}, nil
	}
	if user.GoogleCalendarToken != "" {
		return nil, nil
	}
	if user.SystemSettings != nil && user.SystemSettings["calendar.api_key"] != "" {
		return nil, nil
	}

	return &ActionDeniedError{SystemName: s.Name(), Message: "missing calendar credentials: set google_calendar_token or system_settings[calendar.api_key]"}, nil
}

func (s *CalendarSyncSystem) AfterTaskAction(ctx context.Context, actionCtx TaskActionContext) error {
	if actionCtx.Task == nil {
		return nil
	}
	if actionCtx.Action != TaskActionStartTask && actionCtx.Action != TaskActionCompleteTask {
		return nil
	}
	if actionCtx.Task.Tags["calendar.sync_required"].TextValue != "true" {
		return nil
	}

	actionCtx.Task.AddStatusUpdate("Calendar sync system observed task transition")
	return nil
}

var _ TaskSystem = (*CalendarSyncSystem)(nil)
