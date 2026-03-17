package service

import (
	"context"
	"fmt"
	"sync"
)

type ThreeCyclerSystem struct {
	mu      sync.Mutex
	history map[string][]string
	userRepo repositoryUserLookup
}

func NewThreeCyclerSystem(userRepo repositoryUserLookup) *ThreeCyclerSystem {
	return &ThreeCyclerSystem{history: map[string][]string{}, userRepo: userRepo}
}

func (s *ThreeCyclerSystem) Name() string {
	return "three_cycler"
}

func (s *ThreeCyclerSystem) TrackedTraits() []string {
	return []string{"tag.topic", "user_id"}
}

func (s *ThreeCyclerSystem) BeforeTaskAction(ctx context.Context, actionCtx TaskActionContext) (*ActionDeniedError, error) {
	if actionCtx.Action != TaskActionStartTask {
		return nil, nil
	}
	if actionCtx.Task == nil {
		return nil, nil
	}

	topic := actionCtx.Task.Tags["topic"].TextValue
	if topic == "" {
		return nil, nil
	}

	if s.userRepo != nil {
		user, userErr := s.userRepo.GetByID(ctx, actionCtx.Task.UserID)
		if userErr == nil && user.SystemSettings["three_cycler.enabled"] == "false" {
			return nil, nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	history := s.history[actionCtx.Task.UserID]
	if len(history) >= 2 {
		if history[len(history)-1] == topic || history[len(history)-2] == topic {
			msg := fmt.Sprintf("topic %q was used too recently; rotate across 3 distinct topics", topic)
			return &ActionDeniedError{SystemName: s.Name(), Message: msg}, nil
		}
	}

	return nil, nil
}

func (s *ThreeCyclerSystem) AfterTaskAction(ctx context.Context, actionCtx TaskActionContext) error {
	if actionCtx.Action != TaskActionStartTask {
		return nil
	}
	if actionCtx.Task == nil {
		return nil
	}

	topic := actionCtx.Task.Tags["topic"].TextValue
	if topic == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	history := append(s.history[actionCtx.Task.UserID], topic)
	if len(history) > 3 {
		history = history[len(history)-3:]
	}
	s.history[actionCtx.Task.UserID] = history
	actionCtx.Task.AddStatusUpdate("3-cycler rotation advanced")
	return nil
}

var _ TaskSystem = (*ThreeCyclerSystem)(nil)
