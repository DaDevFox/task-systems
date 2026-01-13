package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
)

// InMemoryUserRepository is a simple in-memory implementation of UserRepository
// Used for testing and development
type InMemoryUserRepository struct {
	users      map[string]*domain.User
	emailIndex map[string]string // email -> userID mapping
	mutex      sync.RWMutex
}

// NewInMemoryUserRepository creates a new in-memory user repository
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:      make(map[string]*domain.User),
		emailIndex: make(map[string]string),
	}
}

// Create stores a new user
func (r *InMemoryUserRepository) Create(ctx context.Context, user *domain.User) error {
	if user == nil {
		return ErrInvalidUserData
	}

	if err := user.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidUserData, err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if user with ID already exists
	if _, exists := r.users[user.ID]; exists {
		return fmt.Errorf("%w: user with ID %s already exists", ErrUserAlreadyExists, user.ID)
	}

	// Check if user with email already exists
	if _, exists := r.emailIndex[user.Email]; exists {
		return fmt.Errorf("%w: user with email %s already exists", ErrUserAlreadyExists, user.Email)
	}

	// Create a copy to avoid reference issues
	userCopy := *user
	r.users[user.ID] = &userCopy
	r.emailIndex[user.Email] = user.ID

	return nil
}

// GetByID retrieves a user by their ID
func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: user ID cannot be empty", ErrInvalidUserData)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Return a copy to avoid reference issues
	userCopy := *user
	return &userCopy, nil
}

// GetByEmail retrieves a user by their email address
func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, fmt.Errorf("%w: email cannot be empty", ErrInvalidUserData)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	userID, exists := r.emailIndex[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	user := r.users[userID]
	// Return a copy to avoid reference issues
	userCopy := *user
	return &userCopy, nil
}

// GetByName retrieves a user by their exact name
func (r *InMemoryUserRepository) GetByName(ctx context.Context, name string) (*domain.User, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name cannot be empty", ErrInvalidUserData)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, user := range r.users {
		if user.Name == name {
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, ErrUserNotFound
}

// Update updates an existing user
func (r *InMemoryUserRepository) Update(ctx context.Context, user *domain.User) error {
	if user == nil {
		return ErrInvalidUserData
	}

	if err := user.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidUserData, err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existingUser, exists := r.users[user.ID]
	if !exists {
		return ErrUserNotFound
	}

	// Check if email changed and if new email is available
	if existingUser.Email != user.Email {
		if existingUserID, exists := r.emailIndex[user.Email]; exists && existingUserID != user.ID {
			return fmt.Errorf("%w: user with email %s already exists", ErrUserAlreadyExists, user.Email)
		}

		// Update email index
		delete(r.emailIndex, existingUser.Email)
		r.emailIndex[user.Email] = user.ID
	}

	// Create a copy to avoid reference issues
	userCopy := *user
	r.users[user.ID] = &userCopy

	return nil
}

// Delete removes a user (soft delete sets status to inactive)
func (r *InMemoryUserRepository) Delete(ctx context.Context, id string, hardDelete bool) error {
	if id == "" {
		return fmt.Errorf("%w: user ID cannot be empty", ErrInvalidUserData)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	user, exists := r.users[id]
	if !exists {
		return ErrUserNotFound
	}

	if hardDelete {
		// Remove from both maps
		delete(r.emailIndex, user.Email)
		delete(r.users, id)
	} else {
		// Soft delete - set status to inactive
		userCopy := *user
		userCopy.Status = domain.UserStatusInactive
		r.users[id] = &userCopy
	}

	return nil
}

// List returns users with optional filtering and pagination
func (r *InMemoryUserRepository) List(ctx context.Context, filter ListUsersFilter) ([]*domain.User, string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var users []*domain.User

	// Apply filters
	for _, user := range r.users {
		// Role filter
		if filter.Role != nil && user.Role != *filter.Role {
			continue
		}

		// Status filter
		if filter.Status != nil && user.Status != *filter.Status {
			continue
		}

		// Name prefix filter
		if filter.NamePrefix != "" {
			if !strings.HasPrefix(strings.ToLower(user.Name), strings.ToLower(filter.NamePrefix)) {
				continue
			}
		}

		// Create copy and add to results
		userCopy := *user
		users = append(users, &userCopy)
	}

	// Simple pagination (in production, use proper cursor-based pagination)
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50 // Default page size
	}

	if len(users) <= pageSize {
		return users, "", nil
	}

	// Return first page and indicate there are more results
	return users[:pageSize], "has_more", nil
}

// Search performs text search across user profiles
func (r *InMemoryUserRepository) Search(ctx context.Context, query string, limit int) ([]*domain.User, error) {
	if query == "" {
		return []*domain.User{}, nil
	}

	if limit <= 0 {
		limit = 10
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var matches []*domain.User
	queryLower := strings.ToLower(query)

	for _, user := range r.users {
		// Search in name, email, first name, last name
		if strings.Contains(strings.ToLower(user.Name), queryLower) ||
			strings.Contains(strings.ToLower(user.Email), queryLower) ||
			strings.Contains(strings.ToLower(user.FirstName), queryLower) ||
			strings.Contains(strings.ToLower(user.LastName), queryLower) {

			userCopy := *user
			matches = append(matches, &userCopy)

			if len(matches) >= limit {
				break
			}
		}
	}

	return matches, nil
}

// BulkGet retrieves multiple users by their IDs
func (r *InMemoryUserRepository) BulkGet(ctx context.Context, ids []string) ([]*domain.User, []string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var foundUsers []*domain.User
	var notFoundIDs []string

	for _, id := range ids {
		if user, exists := r.users[id]; exists {
			userCopy := *user
			foundUsers = append(foundUsers, &userCopy)
		} else {
			notFoundIDs = append(notFoundIDs, id)
		}
	}

	return foundUsers, notFoundIDs, nil
}

// Exists checks if a user exists and returns their status
func (r *InMemoryUserRepository) Exists(ctx context.Context, id string) (bool, domain.UserStatus, error) {
	if id == "" {
		return false, domain.UserStatusUnspecified, fmt.Errorf("%w: user ID cannot be empty", ErrInvalidUserData)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return false, domain.UserStatusUnspecified, nil
	}

	return true, user.Status, nil
}

// Count returns the total number of users matching the filter
func (r *InMemoryUserRepository) Count(ctx context.Context, filter ListUsersFilter) (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, user := range r.users {
		// Apply same filters as List method
		if filter.Role != nil && user.Role != *filter.Role {
			continue
		}

		if filter.Status != nil && user.Status != *filter.Status {
			continue
		}

		if filter.NamePrefix != "" {
			if !strings.HasPrefix(strings.ToLower(user.Name), strings.ToLower(filter.NamePrefix)) {
				continue
			}
		}

		count++
	}

	return count, nil
}
