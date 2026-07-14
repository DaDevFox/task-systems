package repository

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	queryutils "github.com/DaDevFox/task-systems/user-core/backend/internal/query"
	pb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto"
	"github.com/dgraph-io/badger/v3"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// BadgerUserRepository is a BadgerDB implementation of UserRepository
type BadgerUserRepository struct {
	db     *badger.DB
	logger *logrus.Logger
}

// NewBadgerUserRepository creates a new BadgerDB user repository
func NewBadgerUserRepository(dbPath string, logger *logrus.Logger) (*BadgerUserRepository, error) {
	if logger == nil {
		logger = logrus.New()
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = &badgerLogger{logger: logger}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open BadgerDB")
	}

	return &BadgerUserRepository{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (r *BadgerUserRepository) Close() error {
	return r.db.Close()
}

// Create stores a new user
func (r *BadgerUserRepository) Create(ctx context.Context, user *pb.User) error {
	if user == nil {
		return ErrInvalidUserData
	}

	if err := user.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidUserData, err)
	}

	// Check if user already exists
	exists, _, err := r.Exists(ctx, user.ID)
	if err != nil {
		return errors.Wrap(err, "failed to check if user exists")
	}
	if exists {
		return fmt.Errorf("%w: user with ID %s already exists", ErrUserAlreadyExists, user.ID)
	}

	// Check if email is already in use
	_, err = r.GetByEmail(ctx, user.Email)
	if err == nil {
		return fmt.Errorf("%w: user with email %s already exists", ErrUserAlreadyExists, user.Email)
	}
	if !errors.Is(err, ErrUserNotFound) {
		return errors.Wrap(err, "failed to check email uniqueness")
	}

	// Set creation timestamp
	now := time.Now()
	// user.CreatedAt = now
	// user.LastUpdatedAt = now

	// Serialize user data
	userData, err := proto.Marshal(user)
	if err != nil {
		return errors.Wrap(err, "failed to marshal user data")
	}

	err = r.db.Update(func(txn *badger.Txn) error {
		// Store user data
		userKey := []byte(fmt.Sprintf("user:%s", user.ID))
		if err := txn.Set(userKey, userData); err != nil {
			return err
		}

		// Store email index
		emailKey := []byte(fmt.Sprintf("email:%s", user.Email))
		if err := txn.Set(emailKey, []byte(user.ID)); err != nil {
			return err
		}

		// Store name index for search
		nameKey := []byte(fmt.Sprintf("name:%s", strings.ToLower(user.Name)))
		return txn.Set(nameKey, []byte(user.ID))
	})

	if err != nil {
		return errors.Wrap(err, "failed to store user data")
	}

	r.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
	}).Info("user created")

	return nil
}

// GetByID retrieves a user by their ID
func (r *BadgerUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: user ID cannot be empty", ErrInvalidUserData)
	}

	var user *pb.User
	err := r.db.View(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("user:%s", id))
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrUserNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			user = &pb.User{}
			return proto.Unmarshal(val, user)
		})
	})

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
		return nil, errors.Wrap(err, "failed to get user by ID")
	}

	return user, nil
}

// GetByEmail retrieves a user by their email address
func (r *BadgerUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, fmt.Errorf("%w: email cannot be empty", ErrInvalidUserData)
	}

	// First get the user ID from email index
	var userID string
	err := r.db.View(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("email:%s", email))
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrUserNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			userID = string(val)
			return nil
		})
	})

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
		return nil, errors.Wrap(err, "failed to get user ID by email")
	}

	// Then get the user by ID
	return r.GetByID(ctx, userID)
}

// TODO: review -- looks reallly buggy and unnecessary (just check FL, FML, display)
// GetByName retrieves a user by their exact name
func (r *BadgerUserRepository) GetByName(ctx context.Context, name string) (*domain.User, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name cannot be empty", ErrInvalidUserData)
	}

	// First get the user ID from name index
	var userID string
	err := r.db.View(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("name:%s", strings.ToLower(name)))
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrUserNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			userID = string(val)
			return nil
		})
	})

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, err
		}
		return nil, errors.Wrap(err, "failed to get user ID by name")
	}

	// Then get the user by ID
	return r.GetByID(ctx, userID)
}

// Update updates an existing user
func (r *BadgerUserRepository) Update(ctx context.Context, user *pb.User) error {
	if user == nil {
		return ErrInvalidUserData
	}

	// TODO: reimplement for proto
	// if err := user.Validate(); err != nil {
	// 	return fmt.Errorf("%w: %v", ErrInvalidUserData, err)
	// }

	// Get current user to check for changes
	currentUser, err := r.GetByID(ctx, user.ID)
	if err != nil {
		return err
	}

	// Check if email changed and if new email is available
	if currentUser.Email != user.Email {
		_, err := r.GetByEmail(ctx, user.Email)
		if err == nil {
			return fmt.Errorf("%w: user with email %s already exists", ErrUserAlreadyExists, user.Email)
		}
		if !errors.Is(err, ErrUserNotFound) {
			return errors.Wrap(err, "failed to check email uniqueness")
		}
	}

	// Update timestamp
	// user.UpdatedAt = time.Now()

	// Serialize user data
	userData, err := proto.Marshal(user)
	if err != nil {
		return errors.Wrap(err, "failed to marshal user data")
	}

	err = r.db.Update(func(txn *badger.Txn) error {
		// Update user data
		userKey := []byte(fmt.Sprintf("user:%s", user.Id))
		if err := txn.Set(userKey, userData); err != nil {
			return err
		}

		// Update email index if changed
		if currentUser.Email != user.Email {
			// Remove old email index
			oldEmailKey := []byte(fmt.Sprintf("email:%s", currentUser.Email))
			if err := txn.Delete(oldEmailKey); err != nil {
				return err
			}

			// Add new email index
			newEmailKey := []byte(fmt.Sprintf("email:%s", user.Email))
			if err := txn.Set(newEmailKey, []byte(user.Id)); err != nil {
				return err
			}
		}

		// TODO: eval need for name index
		// Update name index if changed
		if currentUser.Name != user.Name {
			// Remove old name index
			oldNameKey := []byte(fmt.Sprintf("name:%s", strings.ToLower(currentUser.Name)))
			if err := txn.Delete(oldNameKey); err != nil {
				return err
			}

			// Add new name index
			newNameKey := []byte(fmt.Sprintf("name:%s", strings.ToLower(user.Name)))
			if err := txn.Set(newNameKey, []byte(user.ID)); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "failed to update user data")
	}

	r.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
	}).Info("user updated")

	return nil
}

// Delete removes a user (soft delete sets status to inactive)
func (r *BadgerUserRepository) Delete(ctx context.Context, id string, hardDelete bool) error {
	if id == "" {
		return fmt.Errorf("%w: user ID cannot be empty", ErrInvalidUserData)
	}

	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Hard delete - remove all data
	err = r.db.Update(func(txn *badger.Txn) error {
		// Remove user data
		userKey := []byte(fmt.Sprintf("user:%s", id))
		if err := txn.Delete(userKey); err != nil {
			return err
		}

		// Remove email index
		emailKey := []byte(fmt.Sprintf("email:%s", user.Email))
		if err := txn.Delete(emailKey); err != nil {
			return err
		}

		// Remove name index
		nameKey := []byte(fmt.Sprintf("name:%s", strings.ToLower(user.Name)))
		return txn.Delete(nameKey)
	})

	r.logger.WithField("user_id", id).Info("user hard deleted")
	if err != nil {
		return errors.Wrap(err, "failed to delete user")
	}

	return nil
}

func (r *BadgerUserRepository) ListIDs(ctx context.Context, query *pb.UserQuery) ([]string, error) {
	var userIDs []string

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("user:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				user := &pb.User{}
				if err := proto.Unmarshal(val, user); err != nil {
					return err
				}

				if !queryutils.TestUserQuery(query, user) {
					return nil
				}

				userIDs = append(userIDs, user.Id)
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read database: %w", err)
	}

	// TODO: cache results + page in wrapper for requests
	return userIDs, nil
}

func (r *BadgerUserRepository) List(ctx context.Context, query *pb.UserQuery) ([]*pb.User, error) {
	var users []*pb.User

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("user:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				user := &pb.User{}
				if err := proto.Unmarshal(val, user); err != nil {
					return err
				}

				if !queryutils.TestUserQuery(query, user) {
					return nil
				}

				users = append(users, user)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read database: %w", err)
	}

	// TODO: cache results + page in wrapper for requests
	return users, nil
}

// TODO: follow parallel execution if search option permits non-determinism:
// + benchmark for improvement (expected NOT to have improvement, thus noting
// here as expansion potential):
// OR queries create fully parallel subexecutions to be merged together
// with final result -- run until atomic int hits limit (slightly costly?)
// AND queries subexecute in parallel, serial worker intersects to merge
// final results
// NOT queries save the list with lower length: original or total - original (of len len(total) - len(original)) + keep a flag to note how to invert at the end
// and execute in serial
//
// Search performs text search across user profiles
func (r *BadgerUserRepository) Search(ctx context.Context, query *pb.ApproximateUserQuery, limit int) ([]*domain.User, error) {
	if query == "" {
		return []*domain.User{}, nil
	}

	if limit <= 0 {
		limit = 10
	}

	var matches []*domain.User
	queryLower := strings.ToLower(query)

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("user:")
		for it.Seek(prefix); it.ValidForPrefix(prefix) && len(matches) < limit; it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				user := &pb.User{}
				if err := proto.Unmarshal(val, user); err != nil {
					return err
				}

				// Search in name, email, first name, last name
				if strings.Contains(strings.ToLower(user.Name), queryLower) ||
					strings.Contains(strings.ToLower(user.Email), queryLower) ||
					strings.Contains(strings.ToLower(user.FirstName), queryLower) ||
					strings.Contains(strings.ToLower(user.LastName), queryLower) {

					matches = append(matches, user)
				}
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to search users")
	}

	return matches, nil
}

// BulkGet retrieves multiple users by their IDs
func (r *BadgerUserRepository) BulkGet(ctx context.Context, ids []string) ([]*domain.User, []string, error) {
	var foundUsers []*domain.User
	var notFoundIDs []string

	for _, id := range ids {
		user, err := r.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				notFoundIDs = append(notFoundIDs, id)
				continue
			}
			return nil, nil, errors.Wrapf(err, "failed to get user %s", id)
		}
		foundUsers = append(foundUsers, user)
	}

	return foundUsers, notFoundIDs, nil
}

// Exists checks if a user exists and returns their status
func (r *BadgerUserRepository) Exists(ctx context.Context, id string) (bool, domain.UserStatus, error) {
	if id == "" {
		return false, domain.UserStatusUnspecified, fmt.Errorf("%w: user ID cannot be empty", ErrInvalidUserData)
	}

	user, err := r.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return false, domain.UserStatusUnspecified, nil
		}
		return false, domain.UserStatusUnspecified, err
	}

	return true, user.Status, nil
}

// Count returns the total number of users matching the filter
func (r *BadgerUserRepository) Count(ctx context.Context, query *pb.UserQuery) (int, error) {
	count := 0

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("user:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(val []byte) error {
				user := &pb.User{}
				if err := proto.Unmarshal(val, user); err != nil {
					return err
				}

				// Apply same filters as List method

				if !queryutils.TestUserQuery(query, user) {
					return nil
				}

				count++
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return 0, errors.Wrap(err, "failed to count users")
	}

	return count, nil
}

// badgerLogger adapts logrus logger to badger's logger interface
type badgerLogger struct {
	logger *logrus.Logger
}

func (l *badgerLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *badgerLogger) Warningf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *badgerLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *badgerLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}
