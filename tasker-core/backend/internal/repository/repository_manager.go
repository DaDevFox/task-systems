package repository

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// DatabaseConfig represents database configuration options
type DatabaseConfig struct {
	Type    DatabaseType
	Path    string // For file-based databases
	Options map[string]interface{}
}

// DatabaseType represents the type of database to use
type DatabaseType int

const (
	DatabaseMemory DatabaseType = iota
	DatabaseBadger
)

func (dt DatabaseType) String() string {
	switch dt {
	case DatabaseMemory:
		return "memory"
	case DatabaseBadger:
		return "badger"
	default:
		return "unknown"
	}
}

// RepositoryManager manages task and user repositories
type RepositoryManager struct {
	taskRepo TaskRepository
	userRepo UserRepository
	config   DatabaseConfig
	logger   *logrus.Logger
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(config DatabaseConfig, logger *logrus.Logger) (*RepositoryManager, error) {
	if logger == nil {
		logger = logrus.New()
	}

	manager := &RepositoryManager{
		config: config,
		logger: logger,
	}

	if err := manager.initializeRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	return manager, nil
}

// initializeRepositories initializes the repositories based on configuration
func (rm *RepositoryManager) initializeRepositories() error {
	switch rm.config.Type {
	case DatabaseMemory:
		rm.taskRepo = NewInMemoryTaskRepository()
		rm.userRepo = NewInMemoryUserRepository()
		log.Println("✓ Initialized in-memory repositories")

	case DatabaseBadger:
		if rm.config.Path == "" {
			// Default to a data directory in the current working directory
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			rm.config.Path = filepath.Join(wd, "data")
		}

		// Ensure data directory exists
		if err := os.MkdirAll(rm.config.Path, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
		taskRepo, err := NewBadgerTaskRepository(filepath.Join(rm.config.Path, "tasks"), rm.logger)
		if err != nil {
			return fmt.Errorf("failed to create task repository: %w", err)
		}
		rm.taskRepo = taskRepo

		userRepo, err := NewBadgerUserRepository(filepath.Join(rm.config.Path, "users"), rm.logger)
		if err != nil {
			return fmt.Errorf("failed to create user repository: %w", err)
		}
		rm.userRepo = userRepo

		log.Printf("✓ Initialized Badger repositories at %s", rm.config.Path)

	default:
		return fmt.Errorf("unsupported database type: %s", rm.config.Type)
	}

	return nil
}

// TaskRepo returns the task repository
func (rm *RepositoryManager) TaskRepo() TaskRepository {
	return rm.taskRepo
}

// UserRepo returns the user repository
func (rm *RepositoryManager) UserRepo() UserRepository {
	return rm.userRepo
}

// Close closes the repositories and cleans up resources
func (rm *RepositoryManager) Close() error {
	var errors []error

	// Close task repository if it implements io.Closer
	if closer, ok := rm.taskRepo.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close task repository: %w", err))
		}
	}

	// Close user repository if it implements io.Closer
	if closer, ok := rm.userRepo.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close user repository: %w", err))
		}
	}

	if len(errors) > 0 {
		// Return the first error
		return errors[0]
	}

	return nil
}

// DefaultConfig returns the default database configuration
func DefaultConfig() DatabaseConfig {
	// Default to persistent Badger database
	return DatabaseConfig{
		Type: DatabaseBadger,
		Path: "", // Will be set to default in initializeRepositories
	}
}

// MemoryConfig returns configuration for in-memory databases
func MemoryConfig() DatabaseConfig {
	return DatabaseConfig{
		Type: DatabaseMemory,
	}
}
