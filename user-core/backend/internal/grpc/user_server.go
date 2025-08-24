package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserServer implements the UserService gRPC interface
type UserServer struct {
	pb.UnimplementedUserServiceServer
	userService *service.UserService
	logger      *logrus.Logger
}

// NewUserServer creates a new UserServer
func NewUserServer(userService *service.UserService, logger *logrus.Logger) *UserServer {
	if logger == nil {
		logger = logrus.New()
	}

	return &UserServer{
		userService: userService,
		logger:      logger,
	}
}

// CreateUser creates a new user account
func (s *UserServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "CreateUser",
		"request_id": fmt.Sprintf("create_user_%d", startTime.UnixNano()),
		"email":      req.Email,
		"name":       req.Name,
	})

	logger.Info("rpc_start")

	// Validation
	if req.Email == "" {
		logger.WithField("validation_error", "empty_email").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if req.Name == "" {
		logger.WithField("validation_error", "empty_name").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	// Convert proto to domain
	role := s.protoToDomainUserRole(req.Role)
	var config *domain.UserConfiguration
	if req.Config != nil {
		domainConfig := s.protoToDomainUserConfig(req.Config)
		config = &domainConfig
	}

	// Create user via service
	user, err := s.userService.CreateUser(ctx, req.Email, req.Name, req.FirstName, req.LastName, role, config)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		// Convert service errors to appropriate gRPC status codes
		if err.Error() == fmt.Sprintf("user with email %s already exists", req.Email) {
			return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		}

		return nil, status.Error(codes.Internal, "failed to create user")
	}

	// Convert to proto response
	response := &pb.CreateUserResponse{
		User: s.domainToProtoUser(user),
	}

	logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// GetUser retrieves a user by ID, email, or name
func (s *UserServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "GetUser",
		"request_id": fmt.Sprintf("get_user_%d", startTime.UnixNano()),
	})

	// Determine lookup type and identifier
	var identifier, lookupType string
	switch req.Identifier.(type) {
	case *pb.GetUserRequest_UserId:
		identifier = req.GetUserId()
		lookupType = "id"
	case *pb.GetUserRequest_Email:
		identifier = req.GetEmail()
		lookupType = "email"
	case *pb.GetUserRequest_Name:
		identifier = req.GetName()
		lookupType = "name"
	default:
		logger.Error("no identifier provided")
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}

	if identifier == "" {
		logger.WithField("lookup_type", lookupType).Error("empty identifier")
		return nil, status.Error(codes.InvalidArgument, "identifier cannot be empty")
	}

	logger = logger.WithFields(logrus.Fields{
		"identifier":  identifier,
		"lookup_type": lookupType,
	})
	logger.Info("rpc_start")

	// Get user via service
	user, err := s.userService.GetUser(ctx, identifier, lookupType)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		if err == repository.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		return nil, status.Error(codes.Internal, "failed to get user")
	}

	response := &pb.GetUserResponse{
		User: s.domainToProtoUser(user),
	}

	logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// UpdateUser modifies user information
func (s *UserServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "UpdateUser",
		"request_id": fmt.Sprintf("update_user_%d", startTime.UnixNano()),
	})

	logger.Info("rpc_start")

	// Validation
	if req.User == nil {
		logger.Error("user is required")
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}

	if req.User.Id == "" {
		logger.Error("user ID is required")
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	// Convert proto to domain
	user := s.protoToDomainUser(req.User)

	logger = logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"user_email": user.Email,
	})

	// Update user via service
	updatedUser, err := s.userService.UpdateUser(ctx, user)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		if err == repository.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		return nil, status.Error(codes.Internal, "failed to update user")
	}

	response := &pb.UpdateUserResponse{
		User: s.domainToProtoUser(updatedUser),
	}

	logger.WithFields(logrus.Fields{
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// ListUsers retrieves multiple users with filtering and pagination
func (s *UserServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "ListUsers",
		"request_id": fmt.Sprintf("list_users_%d", startTime.UnixNano()),
		"page_size":  req.PageSize,
	})

	logger.Info("rpc_start")

	// Build filter
	filter := repository.ListUsersFilter{
		NamePrefix: req.NamePrefix,
		PageSize:   int(req.PageSize),
		PageToken:  req.PageToken,
	}

	if req.RoleFilter != pb.UserRole_USER_ROLE_UNSPECIFIED {
		role := s.protoToDomainUserRole(req.RoleFilter)
		filter.Role = &role
	}

	if req.StatusFilter != pb.UserStatus_USER_STATUS_UNSPECIFIED {
		status := s.protoToDomainUserStatus(req.StatusFilter)
		filter.Status = &status
	}

	// Get users via service
	users, nextToken, totalCount, err := s.userService.ListUsers(ctx, filter)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")
		return nil, status.Error(codes.Internal, "failed to list users")
	}

	// Convert to proto
	var protoUsers []*pb.User
	for _, user := range users {
		protoUsers = append(protoUsers, s.domainToProtoUser(user))
	}

	response := &pb.ListUsersResponse{
		Users:         protoUsers,
		NextPageToken: nextToken,
		TotalCount:    int32(totalCount),
	}

	logger.WithFields(logrus.Fields{
		"users_returned": len(protoUsers),
		"total_count":    totalCount,
		"duration":       time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// DeleteUser removes a user account
func (s *UserServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":         "DeleteUser",
		"request_id":  fmt.Sprintf("delete_user_%d", startTime.UnixNano()),
		"user_id":     req.UserId,
		"hard_delete": req.HardDelete,
	})

	logger.Info("rpc_start")

	// Validation
	if req.UserId == "" {
		logger.Error("user ID is required")
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	// Delete user via service
	err := s.userService.DeleteUser(ctx, req.UserId, req.HardDelete)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		if err == repository.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	response := &pb.DeleteUserResponse{
		Success: true,
	}

	logger.WithFields(logrus.Fields{
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// ValidateUser checks if a user exists and is active
func (s *UserServer) ValidateUser(ctx context.Context, req *pb.ValidateUserRequest) (*pb.ValidateUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "ValidateUser",
		"request_id": fmt.Sprintf("validate_user_%d", startTime.UnixNano()),
		"user_id":    req.UserId,
	})

	logger.Info("rpc_start")

	// Validation
	if req.UserId == "" {
		logger.Error("user ID is required")
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	// Validate user via service
	exists, active, user, err := s.userService.ValidateUser(ctx, req.UserId)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")
		return nil, status.Error(codes.Internal, "failed to validate user")
	}

	response := &pb.ValidateUserResponse{
		Exists: exists,
		Active: active,
	}

	if user != nil {
		response.User = s.domainToProtoUser(user)
	}

	logger.WithFields(logrus.Fields{
		"exists":   exists,
		"active":   active,
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// SearchUsers performs text search across user profiles
func (s *UserServer) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "SearchUsers",
		"request_id": fmt.Sprintf("search_users_%d", startTime.UnixNano()),
		"query":      req.Query,
		"limit":      req.Limit,
	})

	logger.Info("rpc_start")

	// Validation
	if req.Query == "" {
		logger.Error("search query is required")
		return nil, status.Error(codes.InvalidArgument, "search query is required")
	}

	// Search users via service
	users, totalMatches, err := s.userService.SearchUsers(ctx, req.Query, int(req.Limit))
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")
		return nil, status.Error(codes.Internal, "failed to search users")
	}

	// Convert to proto
	var protoUsers []*pb.User
	for _, user := range users {
		protoUsers = append(protoUsers, s.domainToProtoUser(user))
	}

	response := &pb.SearchUsersResponse{
		Users:        protoUsers,
		TotalMatches: int32(totalMatches),
	}

	logger.WithFields(logrus.Fields{
		"results_found": len(protoUsers),
		"total_matches": totalMatches,
		"duration":      time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// BulkGetUsers retrieves multiple users by ID in a single request
func (s *UserServer) BulkGetUsers(ctx context.Context, req *pb.BulkGetUsersRequest) (*pb.BulkGetUsersResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":           "BulkGetUsers",
		"request_id":    fmt.Sprintf("bulk_get_users_%d", startTime.UnixNano()),
		"requested_ids": len(req.UserIds),
	})

	logger.Info("rpc_start")

	// Validation
	if len(req.UserIds) == 0 {
		logger.Error("user IDs are required")
		return nil, status.Error(codes.InvalidArgument, "user IDs are required")
	}

	// Get users via service
	users, notFoundIDs, err := s.userService.BulkGetUsers(ctx, req.UserIds)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")
		return nil, status.Error(codes.Internal, "failed to bulk get users")
	}

	// Convert to proto
	var protoUsers []*pb.User
	for _, user := range users {
		protoUsers = append(protoUsers, s.domainToProtoUser(user))
	}

	response := &pb.BulkGetUsersResponse{
		Users:       protoUsers,
		NotFoundIds: notFoundIDs,
	}

	logger.WithFields(logrus.Fields{
		"found_users":     len(protoUsers),
		"not_found_users": len(notFoundIDs),
		"duration":        time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}
