package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/constants"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"

	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
)

// SettingsService handles user settings operations with ACL enforcement
type SettingsService struct {
	pb.UnimplementedUserSettingsServiceServer
	settingsRepo repository.SettingsRepository
	userRepo     repository.UserRepository
	logger       *logrus.Logger
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
func (s *SettingsService) GetUserSettings(ctx context.Context, req *pb.GetUserSettingsRequest) (*pb.GetUserSettingsResponse, error) {
	if req == nil || req.UserId == nil || strings.Trim(*req.UserId, constants.STRING_TRIMSET) == "" {
		return nil, fmt.Errorf("invalid request")
	}

	// allow if requester is the user
	settings, err := s.settingsRepo.Get(ctx, *req.UserId)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving settings: %w", err)
	}

	return &pb.GetUserSettingsResponse{
		Settings: settings,
	}, nil
}

// Put creates or updates a settings entry; only owner can modify their settings
func (s *SettingsService) UpdateUserSettings(ctx context.Context, req *pb.UpdateUserSettingsRequest) (*pb.UpdateUserSettingsResponse, error) {
	err := s.settingsRepo.Update(ctx, req.Settings)
	if err != nil {
		return nil, fmt.Errorf("Error updating settings: %w", err)
	}

	return &pb.UpdateUserSettingsResponse{}, nil
}
