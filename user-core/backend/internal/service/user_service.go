package service

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/DaDevFox/hof"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/security"
	pb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	errMsgEmailEmpty    = "email cannot be empty"
	errMsgNameEmpty     = "name cannot be empty"
	errMsgPasswordEmpty = "password cannot be empty"
	errMsgUserIDEmpty   = "user ID cannot be empty"
)

// CreateUserParams holds inputs required to create a new user
type CreateUserParams struct {
	Email     string
	Name      string
	FirstName string
	LastName  string
}

// UserService provides business logic for user management operations
type UserService struct {
	userRepo    repository.UserRepository
	groupRepo   repository.GroupRepository
	baggageRepo repository.BaggageRepository
	logger      *logrus.Logger
}

// NewUserServiceWithRepos creates a new user service with optional repos
func NewUserServiceWithRepos(userRepo repository.UserRepository, groupRepo repository.GroupRepository, baggageRepo repository.BaggageRepository, logger *logrus.Logger) *UserService {
	if logger == nil {
		logger = logrus.New()
	}

	if groupRepo == nil {
		groupRepo = repository.NewInMemoryGroupRepository()
	}

	if baggageRepo == nil {
		baggageRepo = repository.NewInMemoryBaggageRepository()
	}

	return &UserService{
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		baggageRepo: baggageRepo,
		logger:      logger,
	}
}

// NewUserService preserves the original constructor signature for callers
func NewUserService(userRepo repository.UserRepository, logger *logrus.Logger) *UserService {
	return NewUserServiceWithRepos(userRepo, nil, nil, logger)
}

// CreateUser creates a new user account
func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*domain.User, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "create_user",
		"email":     req.User.Email,
		"name":      req.User.Name,
	})

	if req.User.Email == "" {
		logger.Error(errMsgEmailEmpty)
		return nil, fmt.Errorf(errMsgEmailEmpty)
	}

	if req.User.Name == "" {
		logger.Error(errMsgNameEmpty)
		return nil, fmt.Errorf(errMsgNameEmpty)
	}

	// Check if user with id already exists
	existingUser, err := s.userRepo.GetByID(ctx, req.User.Id)
	if err == nil {
		logger.WithField("existing_user_email", existingUser.Email).Error("user with id already exists")
		return nil, fmt.Errorf("user with id %s already exists", req.User.Id)
	}

	// Check if user with email already exists
	existingUser, err = s.userRepo.GetByEmail(ctx, req.User.Email)
	if err == nil {
		logger.WithField("existing_user_id", existingUser.ID).Error("user with email already exists")
		return nil, fmt.Errorf("user with email %s already exists", req.User.Email)
	}

	if err != repository.ErrUserNotFound {
		logger.WithError(err).Error("failed to check existing user by email")
		return nil, fmt.Errorf("failed to verify user uniqueness: %w", err)
	}

	// Create user with default or provided configuration
	user := domain.NewUser(req.User.Id, req.User.Email, req.User.Name)
	user.FirstName = req.User.FirstName
	user.LastName = req.User.LastName

	// Store in repository
	if err := s.userRepo.Create(ctx, user); err != nil {
		logger.WithError(err).Error("failed to create user in repository")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	logger.WithField("user_id", user.ID).Info("user created successfully")
	return user, nil
}

// GetUser retrieves a user by ID, email, or name
func (s *UserService) GetUser(ctx context.Context, identifier, lookupType string) (*domain.User, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation":   "get_user",
		"identifier":  identifier,
		"lookup_type": lookupType,
	})

	if identifier == "" {
		logger.Error("GetUser: identifier cannot be empty")
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	var user *domain.User
	var err error

	switch lookupType {
	case "id":
		user, err = s.userRepo.GetByID(ctx, identifier)
	case "email":
		user, err = s.userRepo.GetByEmail(ctx, identifier)
	case "name":
		user, err = s.userRepo.GetByName(ctx, identifier)
	default:
		logger.WithField("invalid_lookup_type", lookupType).Error("invalid lookup type")
		return nil, fmt.Errorf("invalid lookup type: %s", lookupType)
	}

	if err != nil {
		if err == repository.ErrUserNotFound {
			logger.WithField("not_found", identifier).Warn("user not found")
		} else {
			logger.WithError(err).Error("failed to get user")
		}
		return nil, err
	}

	logger.WithField("user_id", user.ID).Debug("user retrieved successfully")
	return user, nil
}

// UpdateUser modifies user information
func (s *UserService) UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "update_user",
		"user_id":   user.ID,
	})

	if user == nil {
		logger.Error("user cannot be nil")
		return nil, fmt.Errorf("user cannot be nil")
	}

	// Update timestamp
	user.LastUpdatedAt = time.Now()

	existingUser, err := s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve existing user for update")
		return nil, fmt.Errorf("failed to fetch user for update: %w", err)
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = existingUser.CreatedAt
	}

	if user.LastLogin == nil && existingUser.LastLogin != nil {
		copyLastLogin := *existingUser.LastLogin
		user.LastLogin = &copyLastLogin
	}

	// Update in repository
	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.WithError(err).Error("failed to update user in repository")
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	logger.Info("user updated successfully")
	return user, nil
}

// ListUsers retrieves multiple users with filtering and pagination
func (s *UserService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "list_users",
		"page_size": req.PageSize,
	})

	filter := repository.ListUsersFilter{
		RegEx:     req.RegexMatch,
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	}

	users, nextToken, err := s.userRepo.List(ctx, filter)

	if err != nil {
		logger.WithError(err).Error("failed to list users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Get total count
	totalCount, err := s.userRepo.Count(ctx, filter)
	if err != nil {
		logger.WithError(err).Warn("failed to get user count")
		totalCount = len(users) // Fallback to current page count
	}

	logger.WithFields(logrus.Fields{
		"users_found": len(users),
		"total_count": totalCount,
	}).Debug("users listed successfully")
	userIDs := hof.Map(func(user *domain.User) string { return user.ID }, users)

	return &pb.ListUsersResponse{UserIds: userIDs, NextPageToken: nextToken, TotalCount: uint32(totalCount)}, nil
}

// DeleteUser removes a user account
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "delete_user",
		"user_id":   userID,
	})

	if userID == "" {
		logger.Error(errMsgUserIDEmpty)
		return fmt.Errorf(errMsgUserIDEmpty)
	}

	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("user not found for deletion")
		return fmt.Errorf("user not found: %w", err)
	}

	// Delete user
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		logger.WithError(err).Error("failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"user_email": user.Email,
	}).Info("user deleted successfully")

	return nil
}

// ValidateUser quickly checks if a user exists and is active
func (s *UserService) ValidateUser(ctx context.Context, userID string) (bool, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "validate_user",
		"user_id":   userID,
	})

	if userID == "" {
		logger.Error(errMsgUserIDEmpty)
		return false, false, nil, fmt.Errorf(errMsgUserIDEmpty)
	}

	exists, userStatus, err := s.userRepo.Exists(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("failed to check user existence")
		return false, fmt.Errorf("failed to validate user: %w", err)
	}
	return exists, nil
}

// TODO: reconcile with new proto
// SearchUsers performs text search across user profiles
func (s *UserService) SearchUsers(ctx context.Context, query *pb.PotentiallyInexactUserQuery, limit int) ([]*domain.User, int, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "search_users",
		"query":     query,
		"limit":     limit,
	})

	if terminal := query.GetTerminal(); terminal != nil {
		s.userRepo.Search(ctx, terminal.IdTextQuery, limit)
	}

	if query == "" {
		logger.Error("search query cannot be empty")
		return nil, 0, fmt.Errorf("search query cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	users, err := s.userRepo.Search(ctx, query, limit)
	if err != nil {
		logger.WithError(err).Error("failed to search users")
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"results_found": len(users),
	}).Debug("user search completed")

	return users, len(users), nil
}

// BulkGetUsers retrieves multiple users by their IDs
func (s *UserService) BulkGetUsers(ctx context.Context, userIDs []string) ([]*domain.User, []string, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation":     "bulk_get_users",
		"requested_ids": len(userIDs),
	})

	if len(userIDs) == 0 {
		logger.Error("user IDs list cannot be empty")
		return nil, nil, fmt.Errorf("user IDs list cannot be empty")
	}

	foundUsers, notFoundIDs, err := s.userRepo.BulkGet(ctx, userIDs)
	if err != nil {
		logger.WithError(err).Error("failed to bulk get users")
		return nil, nil, fmt.Errorf("failed to bulk get users: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"found_users":     len(foundUsers),
		"not_found_users": len(notFoundIDs),
	}).Debug("bulk get users completed")

	return foundUsers, notFoundIDs, nil
}
