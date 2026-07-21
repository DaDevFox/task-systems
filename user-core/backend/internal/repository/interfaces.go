package repository

import (
	"context"
	"fmt"

	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
)

// UserRepository defines the interface for user persistence operations
type UserRepository interface {
	// Create stores a new user
	Create(ctx context.Context, user *pb.User) error

	// GetByID retrieves a user by their ID
	GetByID(ctx context.Context, id string) (*pb.User, error)

	// GetByEmail retrieves a user by their email address
	GetByEmail(ctx context.Context, email string) (*pb.User, error)

	// GetByName retrieves a user by their exact name
	GetByName(ctx context.Context, name string) (*pb.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *pb.User) error

	// Delete removes a user (soft delete sets status to inactive)
	Delete(ctx context.Context, id string) error

	ListIDs(ctx context.Context, query *pb.UserQuery) ([]string, error)
	List(ctx context.Context, query *pb.UserQuery) ([]*pb.User, error)

	// Search performs text search across user profiles
	SearchIDs(ctx context.Context, query *pb.ApproximateUserQuery, limit int) ([]string, error)
	Search(ctx context.Context, query *pb.ApproximateUserQuery, limit int) ([]*pb.User, error)

	// BulkGet retrieves multiple users by their IDs
	BulkGet(ctx context.Context, ids []string) ([]*pb.User, []string, error)

	// Exists checks if a user exists
	Exists(ctx context.Context, id string) (bool, error)

	// Count returns the total number of users matching the filter
	Count(ctx context.Context, filter ListUsersFilter) (int, error)
}

// ListUsersFilter defines filtering options for listing users
type ListUsersFilter struct {
	RegEx     string // Filter by names matching RegEx
	PageSize  uint32 // Maximum users to return
	PageToken string // Token for pagination
}

// Common errors
var (
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserAlreadyExists = fmt.Errorf("user already exists")
	ErrInvalidUserData   = fmt.Errorf("invalid user data")
)
