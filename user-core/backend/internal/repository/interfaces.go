package repository

import (
	"context"
	"fmt"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
)

// UserRepository defines the interface for user persistence operations
type UserRepository interface {
	// Create stores a new user
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by their ID
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by their email address
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetByName retrieves a user by their exact name
	GetByName(ctx context.Context, name string) (*domain.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user (soft delete sets status to inactive)
	Delete(ctx context.Context, id string, hardDelete bool) error

	// List returns users with optional filtering and pagination
	List(ctx context.Context, filter ListUsersFilter) ([]*domain.User, string, error)

	// Search performs text search across user profiles
	Search(ctx context.Context, query string, limit int) ([]*domain.User, error)

	// BulkGet retrieves multiple users by their IDs
	BulkGet(ctx context.Context, ids []string) ([]*domain.User, []string, error)

	// Exists checks if a user exists and returns their status
	Exists(ctx context.Context, id string) (bool, domain.UserStatus, error)

	// Count returns the total number of users matching the filter
	Count(ctx context.Context, filter ListUsersFilter) (int, error)
}

// ListUsersFilter defines filtering options for listing users
type ListUsersFilter struct {
	Role       *domain.UserRole   // Filter by role
	Status     *domain.UserStatus // Filter by status
	NamePrefix string             // Filter by name prefix
	PageSize   int                // Maximum users to return
	PageToken  string             // Token for pagination
}

// Common errors
var (
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserAlreadyExists = fmt.Errorf("user already exists")
	ErrInvalidUserData   = fmt.Errorf("invalid user data")
)
