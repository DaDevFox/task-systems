package service

import (
	"context"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

// UserService provides business logic for user management operations
type UserService struct {
	userRepo repository.UserRepository
	logger   *logrus.Logger
}

// NewUserService creates a new user service
func NewUserService(userRepo repository.UserRepository, logger *logrus.Logger) *UserService {
	if logger == nil {
		logger = logrus.New()
	}

	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// CreateUser creates a new user account
func (s *UserService) CreateUser(ctx context.Context, email, name, firstName, lastName string, role domain.UserRole, config *domain.UserConfiguration) (*domain.User, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "create_user",
		"email":     email,
		"name":      name,
	})

	if email == "" {
		logger.Error("email cannot be empty")
		return nil, fmt.Errorf("email cannot be empty")
	}

	if name == "" {
		logger.Error("name cannot be empty")
		return nil, fmt.Errorf("name cannot be empty")
	}

	// Check if user with email already exists
	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		logger.WithField("existing_email", email).Error("user with email already exists")
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Create user with default or provided configuration
	user := domain.NewUser(email, name)
	user.FirstName = firstName
	user.LastName = lastName

	if role != domain.UserRoleUnspecified {
		user.Role = role
	}

	if config != nil {
		user.Config = *config
	}

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
		logger.Error("identifier cannot be empty")
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
		// For name lookup, we'll use search functionality
		users, searchErr := s.userRepo.Search(ctx, identifier, 1)
		if searchErr != nil {
			err = searchErr
		} else if len(users) == 0 {
			err = repository.ErrUserNotFound
		} else {
			user = users[0]
		}
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
	user.UpdatedAt = time.Now()

	// Update in repository
	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.WithError(err).Error("failed to update user in repository")
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	logger.Info("user updated successfully")
	return user, nil
}

// ListUsers retrieves multiple users with filtering and pagination
func (s *UserService) ListUsers(ctx context.Context, filter repository.ListUsersFilter) ([]*domain.User, string, int, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation":   "list_users",
		"role_filter": filter.Role,
		"page_size":   filter.PageSize,
	})

	users, nextToken, err := s.userRepo.List(ctx, filter)
	if err != nil {
		logger.WithError(err).Error("failed to list users")
		return nil, "", 0, fmt.Errorf("failed to list users: %w", err)
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

	return users, nextToken, totalCount, nil
}

// DeleteUser removes a user account
func (s *UserService) DeleteUser(ctx context.Context, userID string, hardDelete bool) error {
	logger := s.logger.WithFields(logrus.Fields{
		"operation":   "delete_user",
		"user_id":     userID,
		"hard_delete": hardDelete,
	})

	if userID == "" {
		logger.Error("user ID cannot be empty")
		return fmt.Errorf("user ID cannot be empty")
	}

	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("user not found for deletion")
		return fmt.Errorf("user not found: %w", err)
	}

	// Delete user
	if err := s.userRepo.Delete(ctx, userID, hardDelete); err != nil {
		logger.WithError(err).Error("failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	deleteType := "soft"
	if hardDelete {
		deleteType = "hard"
	}

	logger.WithFields(logrus.Fields{
		"delete_type": deleteType,
		"user_email":  user.Email,
	}).Info("user deleted successfully")

	return nil
}

// ValidateUser checks if a user exists and is active
func (s *UserService) ValidateUser(ctx context.Context, userID string) (bool, bool, *domain.User, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "validate_user",
		"user_id":   userID,
	})

	if userID == "" {
		logger.Error("user ID cannot be empty")
		return false, false, nil, fmt.Errorf("user ID cannot be empty")
	}

	exists, userStatus, err := s.userRepo.Exists(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("failed to check user existence")
		return false, false, nil, fmt.Errorf("failed to validate user: %w", err)
	}

	// Check if user is active
	active := userStatus == domain.UserStatusActive

	// If caller wants full user details, fetch them
	var user *domain.User
	if exists {
		user, err = s.userRepo.GetByID(ctx, userID)
		if err != nil {
			logger.WithError(err).Warn("failed to get user details after validation")
			// Don't fail validation if user exists but we can't get details
		}
	}

	logger.WithFields(logrus.Fields{
		"exists": exists,
		"active": active,
	}).Debug("user validation completed")

	return exists, active, user, nil
}

// SearchUsers performs text search across user profiles
func (s *UserService) SearchUsers(ctx context.Context, query string, limit int) ([]*domain.User, int, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "search_users",
		"query":     query,
		"limit":     limit,
	})

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
