package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/service"
	pb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	rpcErrUserNotFound             = "user not found"
	rpcErrUserIDRequired           = "user ID is required"
	rpcErrPasswordRequired         = "password is required"
	rpcErrIdentifierRequired       = "identifier is required"
	rpcErrAccessTokenRequired      = "access token is required"
	rpcErrRefreshTokenRequired     = "refresh token is required"
	rpcErrCurrentPasswordRequired  = "current password is required"
	rpcErrNewPasswordRequired      = "new password is required"
)

// UserServer implements the UserService gRPC interface
type UserServer struct {
	pb.UnimplementedUserServiceServer
	userService *service.UserService
	authService *service.AuthService
	logger      *logrus.Logger
}

// NewUserServer creates a new UserServer
func NewUserServer(userService *service.UserService, authService *service.AuthService, logger *logrus.Logger) *UserServer {
	if logger == nil {
		logger = logrus.New()
	}

	return &UserServer{
		userService: userService,
		authService: authService,
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

	if req.Password == "" {
		logger.WithField("validation_error", "empty_password").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, rpcErrPasswordRequired)
	}

	// Convert proto to domain
	role := s.protoToDomainUserRole(req.Role)
	var config *domain.UserConfiguration
	if req.Config != nil {
		domainConfig := s.protoToDomainUserConfig(req.Config)
		config = &domainConfig
	}

	createParams := service.CreateUserParams{
		Email:     req.Email,
		Name:      req.Name,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
		Role:      role,
		Config:    config,
	}

	// Create user via service
	user, err := s.userService.CreateUser(ctx, createParams)
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

// Authenticate validates credentials and issues tokens
func (s *UserServer) Authenticate(ctx context.Context, req *pb.AuthenticateUserRequest) (*pb.AuthenticateUserResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "Authenticate",
		"request_id": fmt.Sprintf("authenticate_user_%d", startTime.UnixNano()),
	})

	logger.Info("rpc_start")

	if req.Identifier == "" {
		logger.WithField("validation_error", "empty_identifier").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, rpcErrIdentifierRequired)
	}

	if req.Password == "" {
		logger.WithField("validation_error", "empty_password").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, rpcErrPasswordRequired)
	}

	result, err := s.authService.Authenticate(ctx, req.Identifier, req.Password)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Warn("rpc_service_call_failed")

		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}

		if errors.Is(err, service.ErrRefreshTokenInvalid) {
			return nil, status.Error(codes.Internal, "refresh token bootstrap failed")
		}

		return nil, status.Error(codes.Internal, "failed to authenticate user")
	}

	response := &pb.AuthenticateUserResponse{
		AccessToken: result.AccessToken,
		RefreshToken: result.RefreshToken,
		AccessTokenExpiresAt: timestamppb.New(result.AccessTokenExpiresAt),
		User: s.domainToProtoUser(result.User),
	}

	logger.WithFields(logrus.Fields{
		"user_id":  result.User.ID,
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// RefreshToken rotates refresh tokens and issues a new access token
func (s *UserServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "RefreshToken",
		"request_id": fmt.Sprintf("refresh_token_%d", startTime.UnixNano()),
	})

	logger.Info("rpc_start")

	if req.RefreshToken == "" {
		logger.WithField("validation_error", "empty_refresh_token").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, rpcErrRefreshTokenRequired)
	}

	result, err := s.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Warn("rpc_service_call_failed")

		if errors.Is(err, service.ErrRefreshTokenInvalid) {
			return nil, status.Error(codes.PermissionDenied, "refresh token invalid")
		}

		if errors.Is(err, service.ErrRefreshTokenExpired) {
			return nil, status.Error(codes.Unauthenticated, "refresh token expired")
		}

		return nil, status.Error(codes.Internal, "failed to refresh token")
	}

	response := &pb.RefreshTokenResponse{
		AccessToken: result.AccessToken,
		AccessTokenExpiresAt: timestamppb.New(result.AccessTokenExpiresAt),
		RefreshToken: result.RefreshToken,
	}

	logger.WithFields(logrus.Fields{
		"user_id":  result.User.ID,
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// ValidateToken verifies an access token and returns its claims
func (s *UserServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "ValidateToken",
		"request_id": fmt.Sprintf("validate_token_%d", startTime.UnixNano()),
	})

	logger.Info("rpc_start")

	if req.AccessToken == "" {
		logger.WithField("validation_error", "empty_access_token").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, "access token is required")
	}

	result, err := s.authService.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Warn("rpc_service_call_failed")
		return nil, status.Error(codes.Unauthenticated, "token is invalid")
	}

	response := &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: result.Claims.UserID,
		Email:  result.Claims.Email,
		Role:   s.stringToProtoUserRole(result.Claims.Role),
	}

	if result.Claims.RegisteredClaims.ExpiresAt != nil {
		expiresAt := result.Claims.RegisteredClaims.ExpiresAt.Time
		response.ExpiresAt = timestamppb.New(expiresAt)
	}

	logger.WithFields(logrus.Fields{
		"user_id":  result.Claims.UserID,
		"duration": time.Since(startTime),
	}).Info("rpc_success")

	return response, nil
}

// UpdatePassword allows authenticated users to rotate their password
func (s *UserServer) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error) {
	startTime := time.Now()
	logger := s.logger.WithFields(logrus.Fields{
		"rpc":        "UpdatePassword",
		"request_id": fmt.Sprintf("update_password_%d", startTime.UnixNano()),
		"user_id":    req.UserId,
	})

	logger.Info("rpc_start")

	if req.UserId == "" {
		logger.WithField("validation_error", "empty_user_id").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, rpcErrUserIDRequired)
	}

	if req.CurrentPassword == "" {
		logger.WithField("validation_error", "empty_current_password").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, rpcErrCurrentPasswordRequired)
	}

	if req.NewPassword == "" {
		logger.WithField("validation_error", "empty_new_password").Error("rpc_validation_failed")
		return nil, status.Error(codes.InvalidArgument, rpcErrNewPasswordRequired)
	}

	err := s.authService.UpdatePassword(ctx, req.UserId, req.CurrentPassword, req.NewPassword)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Warn("rpc_service_call_failed")

		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}

		return nil, status.Error(codes.Internal, "failed to update password")
	}

	logger.WithField("duration", time.Since(startTime)).Info("rpc_success")

	return &pb.UpdatePasswordResponse{Success: true}, nil
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
	logger.Info("rpc_start")

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
		logger.Error(rpcErrUserIDRequired)
		return nil, status.Error(codes.InvalidArgument, rpcErrUserIDRequired)
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
			return nil, status.Error(codes.NotFound, rpcErrUserNotFound)
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
		logger.Error(rpcErrUserIDRequired)
		return nil, status.Error(codes.InvalidArgument, rpcErrUserIDRequired)
	}

	// Delete user via service
	err := s.userService.DeleteUser(ctx, req.UserId, req.HardDelete)
	if err != nil {
		logger.WithError(err).WithField("duration", time.Since(startTime)).Error("rpc_service_call_failed")

		if err == repository.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, rpcErrUserNotFound)
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
		logger.Error(rpcErrUserIDRequired)
		return nil, status.Error(codes.InvalidArgument, rpcErrUserIDRequired)
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
