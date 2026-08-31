package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/constants"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	rpcErrUserNotFound            = "user not found"
	rpcErrUserIDRequired          = "user ID is required"
	rpcErrPasswordRequired        = "password is required"
	rpcErrIdentifierRequired      = "identifier is required"
	rpcErrAccessTokenRequired     = "access token is required"
	rpcErrRefreshTokenRequired    = "refresh token is required"
	rpcErrCurrentPasswordRequired = "current password is required"
	rpcErrNewPasswordRequired     = "new password is required"
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

	if req.User.Id == nil || req.User.Email == nil {
		return nil, errors.New(rpcErrIdentifierRequired)
	}

	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "CreateUser",
		"request_id": fmt.Sprintf("create_user_%d", startTime.UnixNano()),
		"email":      req.User.Email,
		"id":         req.User.Id,
	})

	logger.Trace("rpc_start")

	// Create user via service
	user, err := s.userService.CreateUser(ctx, req)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		// Convert service errors to appropriate gRPC status codes
		// TODO: establish standardized error message for already exists
		// if err.Error() == fmt.Sprintf("user with email %s already exists", req.Email) {
		// 	return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		// }

		return nil, status.Error(codes.Internal, "failed to create user")
	}

	// Convert to proto response
	response := &pb.CreateUserResponse{}

	logger.WithFields(logrus.Fields{
		"user_id":  user.Id,
		"duration": time.Since(startTime),
	}).Trace("rpc_success")

	return response, nil
}

// TODO: refactor
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
	// case *pb.GetUserRequest_Name:
	// 	identifier = req.GetName()
	// 	lookupType = "name"
	default:
		logger.Error("no identifier provided")
		return nil, status.Error(codes.InvalidArgument, rpcErrIdentifierRequired)
	}

	if identifier == "" {
		logger.WithField("lookup_type", lookupType).Error("empty identifier")
		return nil, status.Error(codes.InvalidArgument, rpcErrIdentifierRequired)
	}

	logger = logger.WithFields(logrus.Fields{
		"identifier":  identifier,
		"lookup_type": lookupType,
	})
	logger.Trace("rpc_start")

	// Get user via service
	user, err := s.userService.GetUser(ctx, identifier, lookupType)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		if err == repository.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, rpcErrUserNotFound)
		}

		return nil, status.Error(codes.Internal, "failed to get user")
	}

	response := &pb.GetUserResponse{
		User: user,
	}

	logger.WithFields(logrus.Fields{
		"user_id":  user.Id,
		"duration": time.Since(startTime),
	}).Trace("rpc_success")

	return response, nil
}

// TODO: refactor
// UpdateUser modifies user information
func (s *UserServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "UpdateUser",
		"request_id": fmt.Sprintf("update_user_%d", startTime.UnixNano()),
	})

	logger.Trace("rpc_start")

	// Update user via service
	updatedUser, err := s.userService.UpdateUser(ctx, req.User)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		if err == repository.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, rpcErrUserNotFound)
		}

		return nil, status.Error(codes.Internal, "failed to update user")
	}

	response := &pb.UpdateUserResponse{
		User: updatedUser,
	}

	logger.WithFields(logrus.Fields{
		"duration": time.Since(startTime),
	}).Trace("rpc_success")

	return response, nil
}

// TODO: refactor
// ListUsers retrieves multiple users with filtering and pagination
func (s *UserServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "ListUsers",
		"request_id": fmt.Sprintf("list_users_%d", startTime.UnixNano()),
		"page_size":  req.PageSize,
	})

	logger.Trace("rpc_start")

	response, err := s.userService.ListUsers(ctx, req)

	logger.WithFields(logrus.Fields{
		"duration": time.Since(startTime),
	}).Trace("rpc_success")

	if err != nil {
		return nil, err
	}
	return response, nil
}

// TODO: refactor
// DeleteUser removes a user account
func (s *UserServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "DeleteUser",
		"request_id": fmt.Sprintf("delete_user_%d", startTime.UnixNano()),
		"user_id":    req.UserId,
	})

	logger.Trace("rpc_start")

	// Validation
	if req.UserId == nil || strings.Trim(*req.UserId, constants.STRING_TRIMSET) == "" {
		return nil, status.Error(codes.InvalidArgument, rpcErrUserIDRequired)
	}

	// Delete user via service
	err := s.userService.DeleteUser(ctx, *req.UserId)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		if err == repository.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, rpcErrUserNotFound)
		}

		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	response := &pb.DeleteUserResponse{}

	logger.WithFields(logrus.Fields{
		"duration": time.Since(startTime),
	}).Trace("rpc_success")

	return response, nil
}

// TODO: refactor
// ValidateUser checks if a user exists and is active
func (s *UserServer) ValidateUser(ctx context.Context, req *pb.ValidateUserRequest) (*pb.ValidateUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "ValidateUser",
		"request_id": fmt.Sprintf("validate_user_%d", startTime.UnixNano()),
		"user_id":    req.UserId,
	})

	logger.Trace("rpc_start")

	// Validation
	if req.UserId == nil || strings.Trim(*req.UserId, constants.STRING_TRIMSET) == "" {
		logger.Error(rpcErrUserIDRequired)
		return nil, status.Error(codes.InvalidArgument, rpcErrUserIDRequired)
	}

	// Validate user via service
	exists, err := s.userService.ValidateUser(ctx, *req.UserId)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")
		return nil, status.Error(codes.Internal, "failed to validate user")
	}

	response := &pb.ValidateUserResponse{
		Exists: &exists,
	}

	logger.WithFields(logrus.Fields{
		"exists":   exists,
		"duration": time.Since(startTime),
	}).Trace("rpc_success")

	return response, nil
}

// TODO: refactor
// SearchUsers performs text search across user profiles
func (s *UserServer) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "SearchUsers",
		"request_id": fmt.Sprintf("search_users_%d", startTime.UnixNano()),
		"query":      req.Query,
		"limit":      req.Limit,
	})

	logger.Trace("rpc_start")

	// Validation
	if req.Query == nil || req.Limit == nil {
		logger.Error("search query + limit is required")
		return nil, status.Error(codes.InvalidArgument, "search query is required")
	}

	// Search users via service
	users, totalMatches, err := s.userService.SearchUsers(ctx, req.Query, int(*req.Limit))
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")
		return nil, status.Error(codes.Internal, "failed to search users")
	}

	uintMatches := uint32(totalMatches)
	response := &pb.SearchUsersResponse{
		Users:        users,
		TotalMatches: &uintMatches,
	}

	logger.WithFields(logrus.Fields{
		"results_found": len(users),
		"total_matches": totalMatches,
		"duration":      time.Since(startTime),
	}).Trace("rpc_success")

	return response, nil
}

// TODO: refactor
// BulkGetUsers retrieves multiple users by ID in a single request
func (s *UserServer) BulkGetUsers(ctx context.Context, req *pb.BulkGetUsersRequest) (*pb.BulkGetUsersResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":           "BulkGetUsers",
		"request_id":    fmt.Sprintf("bulk_get_users_%d", startTime.UnixNano()),
		"requested_ids": len(req.UserIds),
	})

	logger.Trace("rpc_start")

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
	response := &pb.BulkGetUsersResponse{
		Users:       users,
		NotFoundIds: notFoundIDs,
	}

	logger.WithFields(logrus.Fields{
		"found_users":     len(users),
		"not_found_users": len(notFoundIDs),
		"duration":        time.Since(startTime),
	}).Trace("rpc_success")

	return response, nil
}

func (s *UserServer) Resolve(ctx context.Context, req *pb.ResolveRequest) (*pb.ResolveResponse, error) {
	// query := req.Query

	return nil, nil
}

func (s *UserServer) Test(ctx context.Context, req *pb.TestRequest) (*pb.TestResponse, error) {
	return nil, nil
}

func (s *UserServer) ResolveStreaming(ctx context.Context, req *pb.ResolveRequest) (*pb.ResolveResponse, error) {
	return nil, nil
}

func (s *UserServer) TestStreaming(ctx context.Context, req *pb.TestRequest) (*pb.TestResponse, error) {
	return nil, nil
}
