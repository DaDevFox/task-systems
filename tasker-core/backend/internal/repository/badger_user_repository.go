package repository

import (
	"context"
	"encoding/json"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
	"github.com/dgraph-io/badger/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Constants for error messages and prefixes
const (
	ErrEmptyUserID = "user ID cannot be empty"
	ErrEmptyEmail  = "email cannot be empty"
	UserKeyPrefix  = "user:"
	EmailKeyPrefix = "email:"
)

// BadgerUserRepository implements UserRepository using BadgerDB for persistence
type BadgerUserRepository struct {
	db     *badger.DB
	logger *logrus.Logger
}

// NewBadgerUserRepository creates a new BadgerDB-backed user repository
func NewBadgerUserRepository(dbPath string, logger *logrus.Logger) (*BadgerUserRepository, error) {
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = &badgerLogger{logger: logger}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open badger database")
	}

	repo := &BadgerUserRepository{
		db:     db,
		logger: logger,
	}

	return repo, nil
}

// Close closes the database connection
func (r *BadgerUserRepository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

// Create stores a new user
func (r *BadgerUserRepository) Create(ctx context.Context, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.ID == "" {
		return errors.New("user ID cannot be empty")
	}

	// Check if user already exists
	exists, err := r.userExists(user.ID)
	if err != nil {
		return errors.Wrap(err, "failed to check if user exists")
	}
	if exists {
		return errors.Errorf("user with ID %s already exists", user.ID)
	}

	// Check if email already exists
	existingUserID, err := r.getUserIDByEmail(user.Email)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return errors.Wrap(err, "failed to check email uniqueness")
	}
	if existingUserID != "" {
		return errors.Errorf("user with email %s already exists", user.Email)
	}

	data, err := json.Marshal(user)
	if err != nil {
		return errors.Wrap(err, "failed to marshal user to JSON")
	}

	err = r.db.Update(func(txn *badger.Txn) error {
		// Store user data
		userKey := []byte("user:" + user.ID)
		if err := txn.Set(userKey, data); err != nil {
			return err
		}

		// Store email index
		emailKey := []byte("email:" + user.Email)
		return txn.Set(emailKey, []byte(user.ID))
	})

	if err != nil {
		return errors.Wrap(err, "failed to store user in database")
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
		return nil, errors.New("user ID cannot be empty")
	}

	var user *domain.User
	err := r.db.View(func(txn *badger.Txn) error {
		key := []byte("user:" + id)
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrUserNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			user = &domain.User{}
			return json.Unmarshal(val, user)
		})
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByEmail retrieves a user by their email
func (r *BadgerUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	// First get the user ID from the email index
	userID, err := r.getUserIDByEmail(email)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, errors.Wrap(err, "failed to get user ID by email")
	}

	// Then get the user by ID
	return r.GetByID(ctx, userID)
}

// Update updates an existing user
func (r *BadgerUserRepository) Update(ctx context.Context, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.ID == "" {
		return errors.New("user ID cannot be empty")
	}

	// Check if user exists
	exists, err := r.userExists(user.ID)
	if err != nil {
		return errors.Wrap(err, "failed to check if user exists")
	}
	if !exists {
		return ErrUserNotFound
	}

	// Get current user to check email changes
	currentUser, err := r.GetByID(ctx, user.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get current user")
	}

	// If email changed, check uniqueness
	if currentUser.Email != user.Email {
		existingUserID, err := r.getUserIDByEmail(user.Email)
		if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
			return errors.Wrap(err, "failed to check email uniqueness")
		}
		if existingUserID != "" && existingUserID != user.ID {
			return errors.Errorf("user with email %s already exists", user.Email)
		}
	}

	data, err := json.Marshal(user)
	if err != nil {
		return errors.Wrap(err, "failed to marshal user to JSON")
	}

	err = r.db.Update(func(txn *badger.Txn) error {
		// Update user data
		userKey := []byte("user:" + user.ID)
		if err := txn.Set(userKey, data); err != nil {
			return err
		}

		// Update email index if email changed
		if currentUser.Email != user.Email {
			// Remove old email index
			oldEmailKey := []byte("email:" + currentUser.Email)
			if err := txn.Delete(oldEmailKey); err != nil {
				return err
			}

			// Add new email index
			newEmailKey := []byte("email:" + user.Email)
			if err := txn.Set(newEmailKey, []byte(user.ID)); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "failed to update user in database")
	}

	r.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
	}).Info("user updated")

	return nil
}

// Delete removes a user
func (r *BadgerUserRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("user ID cannot be empty")
	}

	// Get user to get email for index cleanup
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = r.db.Update(func(txn *badger.Txn) error {
		// Delete user data
		userKey := []byte("user:" + id)
		if err := txn.Delete(userKey); err != nil {
			return err
		}

		// Delete email index
		emailKey := []byte("email:" + user.Email)
		return txn.Delete(emailKey)
	})

	if err != nil {
		return errors.Wrap(err, "failed to delete user from database")
	}

	r.logger.WithField("user_id", id).Info("user deleted")
	return nil
}

// ListAll returns all users
func (r *BadgerUserRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	var users []*domain.User

	err := r.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("user:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var user domain.User
				if err := json.Unmarshal(val, &user); err != nil {
					return err
				}
				users = append(users, &user)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to list users")
	}

	return users, nil
}

// Helper methods

// userExists checks if a user with the given ID exists
func (r *BadgerUserRepository) userExists(userID string) (bool, error) {
	err := r.db.View(func(txn *badger.Txn) error {
		key := []byte("user:" + userID)
		_, err := txn.Get(key)
		return err
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// getUserIDByEmail gets the user ID associated with an email
func (r *BadgerUserRepository) getUserIDByEmail(email string) (string, error) {
	var userID string
	err := r.db.View(func(txn *badger.Txn) error {
		key := []byte("email:" + email)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			userID = string(val)
			return nil
		})
	})

	return userID, err
}
