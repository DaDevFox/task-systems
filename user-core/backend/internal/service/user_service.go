package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/constants"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/validity"
	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/sirupsen/logrus"
)

const (
	errMsgEmailEmpty  = "email cannot be empty"
	errMsgNameEmpty   = "name cannot be empty"
	errMsgUserIDEmpty = "user ID cannot be empty"
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
	userRepo     repository.UserRepository
	groupRepo    repository.GroupRepository
	settingsRepo repository.SettingsRepository
	logger       *logrus.Logger
}

// NewUserServiceWithRepos creates a new user service with optional repos
func NewUserServiceWithRepos(userRepo repository.UserRepository, groupRepo repository.GroupRepository, settingsRepo repository.SettingsRepository, logger *logrus.Logger) *UserService {
	if logger == nil {
		logger = logrus.New()
	}

	if groupRepo == nil {
		groupRepo = repository.NewInMemoryGroupRepository()
	}

	if settingsRepo == nil {
		settingsRepo = repository.NewInMemorySettingsRepository()
	}

	return &UserService{
		userRepo:     userRepo,
		groupRepo:    groupRepo,
		settingsRepo: settingsRepo,
		logger:       logger,
	}
}

// NewUserService preserves the original constructor signature for callers
func NewUserService(userRepo repository.UserRepository, logger *logrus.Logger) *UserService {
	return NewUserServiceWithRepos(userRepo, nil, nil, logger)
}

// CreateUser creates a new user account
func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	err := validity.ValidateUser(req.User)
	if err != nil {
		return nil, fmt.Errorf("invalid user: %w", err)
	}

	userName, err := validity.UserName(req.User)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal user name: %w", err)
	}

	if req.User.Email == nil {
		return nil, fmt.Errorf(errMsgEmailEmpty)
	}

	logger := s.logger.WithFields(logrus.Fields{
		"operation": "create_user",
		"email":     *req.User.Email,
		"name":      userName,
	})

	// Check if user with id already exists
	existingUser, err := s.userRepo.GetByID(ctx, *req.User.Id)
	if err == nil {
		logger.WithField("existing_user_email", *existingUser.Email).Trace("user with id already exists")
		return nil, fmt.Errorf("user with id %s already exists", *req.User.Id)
	}

	// Check if user with email already exists
	existingUser, err = s.userRepo.GetByEmail(ctx, *req.User.Email)
	if err == nil {
		logger.WithField("existing_user_id", *existingUser.Id).Trace("user with email already exists")
		return nil, fmt.Errorf("user with email %s already exists", *req.User.Email)
	}

	if err != repository.ErrUserNotFound {
		return nil, fmt.Errorf("failed to verify user uniqueness: %w", err)
	}

	// Store in repository
	if err := s.userRepo.Create(ctx, req.User); err != nil {
		return nil, fmt.Errorf("failed to store user in repository: %w", err)
	}

	logger.WithField("user_id", *req.User.Id).Debug("user created successfully")
	return req.User, nil
}

// GetUser retrieves a user by ID, email, or name
func (s *UserService) GetUser(ctx context.Context, identifier, lookupType string) (*pb.User, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation":   "get_user",
		"identifier":  identifier,
		"lookup_type": lookupType,
	})

	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	var user *pb.User
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
			err = fmt.Errorf("user not found: %w", err)
		} else {
			err = fmt.Errorf("failed to get user: %w", err)
		}
		return nil, err
	}

	logger.WithField("user_id", *user.Id).Debug("user retrieved successfully")
	return user, nil
}

// UpdateUser modifies user information
func (s *UserService) UpdateUser(ctx context.Context, user *pb.User) (*pb.User, error) {
	err := validity.ValidateUser(user)
	if err != nil {
		return nil, fmt.Errorf("invalid user: %w", err)
	}

	logger := s.logger.WithFields(logrus.Fields{
		"operation": "update_user",
		"user_id":   *user.Id,
	})

	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}

	// Update in repository
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user in repository: %w", err)
	}

	logger.Debug("user updated successfully")
	return user, nil
}

// ListUsers retrieves multiple users with filtering and pagination
func (s *UserService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "list_users",
		"page_size": req.PageSize,
	})

	response := &pb.ListUsersResponse{}

	var userIDs []string
	var users []*pb.User
	var err error
	if req.WantFullUserObjects != nil && *req.WantFullUserObjects {
		users, err = s.userRepo.List(ctx, req.Query)
		response.UserObjects = users
	} else {
		userIDs, err = s.userRepo.ListIDs(ctx, req.Query)
		response.UserIds = userIDs
	}

	if err != nil {
		logger.WithError(err).Error("failed to list users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// // Get total count
	// totalCount, err := s.userRepo.Count(ctx, req.Query)
	// if err != nil {
	// 	logger.WithError(err).Warn("failed to get user count")
	// 	totalCount = len(users) // Fallback to current page count
	// }

	count := uint32(len(users))
	logger.WithFields(logrus.Fields{
		"users_found": count,
		// "total_count": totalCount,
	}).Debug("users listed successfully")

	response.TotalCount = &count
	return response, nil
}

// DeleteUser removes a user account
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "delete_user",
		"user_id":   userID,
	})

	if strings.Trim(userID, constants.STRING_TRIMSET) == "" {
		return fmt.Errorf(errMsgUserIDEmpty)
	}

	// Delete user
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	logger.Debug("user deleted successfully")
	return nil
}

// ValidateUser quickly checks if a user exists and is active
func (s *UserService) ValidateUser(ctx context.Context, userID string) (bool, error) {
	// logger := s.logger.WithFields(logrus.Fields{
	// 	"operation": "validate_user",
	// 	"user_id":   userID,
	// })

	if userID == "" {
		return false, fmt.Errorf(errMsgUserIDEmpty)
	}

	exists, err := s.userRepo.Exists(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to validate user existence: %w", err)
	}
	return exists, nil
}

// Implementation plan:
// OR => parallel resolve; simple result merge in O(parts)
// AND => parallel resolve; content-aware intersect in O(n)
// NOT => result becomes excl. list

// TODO: reconcile with new proto
// SearchUsers performs text search across user profiles
func (s *UserService) SearchUsers(ctx context.Context, query *pb.ApproximateUserQuery, limit int) ([]*pb.User, int, error) {
	queryText, err := prototext.Marshal(query)
	var logger *logrus.Entry
	if err != nil {
		logger = s.logger.WithFields(logrus.Fields{
			"operation": "search_users",
			"limit":     limit,
		})
		logger.WithError(err).Error("couldn't marshal query string")
	} else {
		logger = s.logger.WithFields(logrus.Fields{
			"operation": "search_users",
			"query":     queryText,
			"limit":     limit,
		})
	}

	if query == nil {
		logger.Trace("search query cannot be empty")
		return nil, 0, fmt.Errorf("search query cannot be empty")
	}

	users, err := s.userRepo.Search(ctx, query, limit)
	if err != nil {
		logger.WithError(fmt.Errorf("failed to search users: %w", err)).Trace("search query cannot be empty")
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"results_found": len(users),
	}).Debug("user search completed")
	return users, len(users), nil
}

// BulkGetUsers retrieves multiple users by their IDs
func (s *UserService) BulkGetUsers(ctx context.Context, userIDs []string) ([]*pb.User, []string, error) {
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
