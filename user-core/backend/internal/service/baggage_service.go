package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

// SettingsService handles user settings operations with ACL enforcement
type SettingsService struct {
	settingsRepo repository.SettingsRepository
	userRepo    repository.UserRepository
	logger      *logrus.Logger
}

func NewSettingsService(settingsRepo repository.SettingsRepository, userRepo repository.UserRepository, logger *logrus.Logger) *SettingsService {
	if logger == nil {
		logger = logrus.New()
	}
	if settingsRepo == nil {
		settingsRepo = repository.NewInMemorySettingsRepository()
	}
	return &SettingsService{settingsRepo: settingsRepo, userRepo: userRepo, logger: logger}
}

// Get retrieves a settings entry; only the owner or admins can retrieve
func (s *SettingsService) Get(ctx context.Context, requesterID, targetUserID, key string) (*domain.SettingsEntry, error) {
	if requesterID == "" || targetUserID == "" || key == "" {
		return nil, fmt.Errorf("invalid request")
	}
	// allow if requester is the user
	if requesterID == targetUserID {
		entry, err := s.settingsRepo.Get(ctx, targetUserID, key)
		if err != nil {
			return nil, err
		}
		return &entry, nil
	}
	// otherwise check if requester is admin in any group? For simplicity, allow admins (global) - require userRepo.GetByID and check role
	requester, err := s.userRepo.GetByID(ctx, requesterID)
	if err != nil {
		return nil, fmt.Errorf("requester not found")
	}
	if requester.Role == domain.UserRoleAdmin {
		entry, err := s.settingsRepo.Get(ctx, targetUserID, key)
		if err != nil {
			return nil, err
		}
		return &entry, nil
	}
	return nil, fmt.Errorf("permission denied")
}

// Put creates or updates a settings entry; only owner can modify their settings
func (s *SettingsService) Put(ctx context.Context, requesterID, targetUserID string, entry domain.SettingsEntry) error {
	if requesterID == "" || targetUserID == "" || entry.Key == "" {
		return fmt.Errorf("invalid request")
	}
	if requesterID != targetUserID {
		return fmt.Errorf("permission denied")
	}
	entry.UpdatedAt = time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	return s.settingsRepo.Put(ctx, targetUserID, entry)
}

// Delete removes a settings entry; only owner can delete
func (s *SettingsService) Delete(ctx context.Context, requesterID, targetUserID, key string) error {
	if requesterID == "" || targetUserID == "" || key == "" {
		return fmt.Errorf("invalid request")
	}
	if requesterID != targetUserID {
		return fmt.Errorf("permission denied")
	}
	return s.settingsRepo.Delete(ctx, targetUserID, key)
}
