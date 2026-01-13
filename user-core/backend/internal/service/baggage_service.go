package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

// BaggageService handles user baggage operations with ACL enforcement
type BaggageService struct {
	baggageRepo repository.BaggageRepository
	userRepo    repository.UserRepository
	logger      *logrus.Logger
}

func NewBaggageService(baggageRepo repository.BaggageRepository, userRepo repository.UserRepository, logger *logrus.Logger) *BaggageService {
	if logger == nil {
		logger = logrus.New()
	}
	if baggageRepo == nil {
		baggageRepo = repository.NewInMemoryBaggageRepository()
	}
	return &BaggageService{baggageRepo: baggageRepo, userRepo: userRepo, logger: logger}
}

// Get retrieves a baggage entry; only the owner or admins can retrieve
func (s *BaggageService) Get(ctx context.Context, requesterID, targetUserID, key string) (*domain.BaggageEntry, error) {
	if requesterID == "" || targetUserID == "" || key == "" {
		return nil, fmt.Errorf("invalid request")
	}
	// allow if requester is the user
	if requesterID == targetUserID {
		entry, err := s.baggageRepo.Get(ctx, targetUserID, key)
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
		entry, err := s.baggageRepo.Get(ctx, targetUserID, key)
		if err != nil {
			return nil, err
		}
		return &entry, nil
	}
	return nil, fmt.Errorf("permission denied")
}

// Put creates or updates a baggage entry; only owner can modify their baggage
func (s *BaggageService) Put(ctx context.Context, requesterID, targetUserID string, entry domain.BaggageEntry) error {
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
	return s.baggageRepo.Put(ctx, targetUserID, entry)
}

// Delete removes a baggage entry; only owner can delete
func (s *BaggageService) Delete(ctx context.Context, requesterID, targetUserID, key string) error {
	if requesterID == "" || targetUserID == "" || key == "" {
		return fmt.Errorf("invalid request")
	}
	if requesterID != targetUserID {
		return fmt.Errorf("permission denied")
	}
	return s.baggageRepo.Delete(ctx, targetUserID, key)
}
