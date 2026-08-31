package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/DaDevFox/hof"
	queryutils "github.com/DaDevFox/task-systems/user-core/backend/internal/query"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/validity"
	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
	"google.golang.org/protobuf/proto"
)

// InMemoryUserRepository is a simple in-memory implementation of UserRepository
// Used for testing and development
type InMemoryUserRepository struct {
	users      map[string]*pb.User
	emailIndex map[string]string // email -> userID mapping
	mutex      sync.RWMutex
}

// NewInMemoryUserRepository creates a new in-memory user repository
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:      make(map[string]*pb.User),
		emailIndex: make(map[string]string),
	}
}

// Create stores a new user
func (r *InMemoryUserRepository) Create(ctx context.Context, user *pb.User) error {
	if user == nil {
		return ErrInvalidUserData
	}

	if err := validity.ValidateUser(user); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidUserData, err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if user with ID already exists
	if _, exists := r.users[*user.Id]; exists {
		return fmt.Errorf("%w: user with ID %s already exists", ErrUserAlreadyExists, user.Id)
	}

	hasEmail := user.Email != nil
	if hasEmail {
		// Check if user with email already exists
		if _, exists := r.emailIndex[*user.Email]; exists {
			return fmt.Errorf("%w: user with email %s already exists", ErrUserAlreadyExists, user.Email)
		}
	}

	// Create a copy to avoid reference issues
	r.users[*user.Id] = proto.CloneOf(user)
	if hasEmail {
		r.emailIndex[*user.Email] = *user.Id
	}
	return nil
}

// GetByID retrieves a user by their ID
func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*pb.User, error) {
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
	return proto.CloneOf(user), nil
}

// GetByEmail retrieves a user by their email address
func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*pb.User, error) {
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
	return proto.CloneOf(user), nil
}

// GetByName retrieves a user by their exact name
func (r *InMemoryUserRepository) GetByName(ctx context.Context, name string) (*pb.User, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name cannot be empty", ErrInvalidUserData)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, user := range r.users {
		userName, err := validity.UserName(user)
		if err != nil {
			// TODO: log here
			continue
		}
		if userName == name {
			return proto.CloneOf(user), nil
		}
	}

	return nil, ErrUserNotFound
}

// Update updates an existing user
func (r *InMemoryUserRepository) Update(ctx context.Context, user *pb.User) error {
	if user == nil {
		return ErrInvalidUserData
	}

	if err := validity.ValidateUser(user); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidUserData, err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	existingUser, exists := r.users[*user.Id]
	if !exists {
		return ErrUserNotFound
	}

	hasEmail := user.Email != nil

	// Check if email changed and if new email is available
	if hasEmail && (existingUser.Email == nil || *existingUser.Email != *user.Email) {
		if existingUserID, exists := r.emailIndex[*user.Email]; exists && existingUserID != *user.Id {
			return fmt.Errorf("%w: user with email %s already exists", ErrUserAlreadyExists, user.Email)
		}

		// Update email index
		delete(r.emailIndex, *existingUser.Email)
		r.emailIndex[*user.Email] = *user.Id
	}

	// Create a copy to avoid reference issues
	r.users[*user.Id] = proto.CloneOf(user)

	return nil
}

// Delete removes a user (soft delete sets status to inactive)
func (r *InMemoryUserRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: user ID cannot be empty", ErrInvalidUserData)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	user, exists := r.users[id]
	if !exists {
		return ErrUserNotFound
	}

	if user.Email != nil {
		delete(r.emailIndex, *user.Email)
	}

	delete(r.users, id)

	return nil
}

// List returns users with optional filtering and pagination
func (r *InMemoryUserRepository) List(ctx context.Context, query *pb.UserQuery) ([]*pb.User, error) {
	if query == nil {
		return []*pb.User{}, errors.New("empty input")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var users []*pb.User

	// Apply filters
	for _, user := range r.users {
		if !queryutils.TestUserQuery(query, user) {
			continue
		}

		// Create copy and add to results
		users = append(users, proto.CloneOf(user))
	}

	// Return first page and indicate there are more results
	return users, nil
}

func (r *InMemoryUserRepository) idsFor(action func() ([]*pb.User, error)) ([]string, error) {
	result, err := action()
	if err != nil {
		return nil, err
	}
	return slices.Collect(hof.Map(result, func(user *pb.User) string { return *user.Id })), nil
}

func (r *InMemoryUserRepository) ListIDs(ctx context.Context, query *pb.UserQuery) ([]string, error) {
	return r.idsFor(func() ([]*pb.User, error) {
		return r.List(ctx, query)
	})
}

// Search performs text search across user profiles
func (r *InMemoryUserRepository) Search(ctx context.Context, query *pb.ApproximateUserQuery, limit int) ([]*pb.User, error) {
	if query == nil {
		return []*pb.User{}, errors.New("empty input")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var matches []*pb.User

	for _, user := range r.users {
		// Search in name, email, first name, last name

		if !queryutils.TestApproximateUserQuery(query, user) {
			continue
		}

		matches = append(matches, proto.CloneOf(user))
	}

	return matches, nil
}

func (r *InMemoryUserRepository) SearchIDs(ctx context.Context, query *pb.ApproximateUserQuery, limit int) ([]string, error) {
	return r.idsFor(func() ([]*pb.User, error) {
		return r.Search(ctx, query, limit)
	})
}

// BulkGet retrieves multiple users by their IDs
func (r *InMemoryUserRepository) BulkGet(ctx context.Context, ids []string) ([]*pb.User, []string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var foundUsers []*pb.User
	var notFoundIDs []string

	for _, id := range ids {
		if user, exists := r.users[id]; exists {
			foundUsers = append(foundUsers, proto.CloneOf(user))
		} else {
			notFoundIDs = append(notFoundIDs, id)
		}
	}

	return foundUsers, notFoundIDs, nil
}

// Exists checks if a user exists and returns their status
func (r *InMemoryUserRepository) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("%w: user ID cannot be empty", ErrInvalidUserData)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.users[id]
	return exists, nil
}

// Count returns the total number of users matching the filter
func (r *InMemoryUserRepository) Count(ctx context.Context, query *pb.UserQuery) (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	count := 0
	for _, user := range r.users {
		// Apply same filters as List method
		if !queryutils.TestUserQuery(query, user) {
			continue
		}

		count++
	}

	return count, nil
}
