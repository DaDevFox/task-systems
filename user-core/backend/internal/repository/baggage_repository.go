package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/constants"
	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// SettingsRepository stores key/value metadata per user
type SettingsRepository interface {
	Get(ctx context.Context, userID string) (*pb.UserSettings, error)
	Update(ctx context.Context, settings *pb.UserSettings) error
}

// InMemorySettingsRepository is a testable in-memory implementation
type InMemorySettingsRepository struct {
	store map[string]*pb.UserSettings // userID -> settings map
	mutex sync.RWMutex
}

// NewInMemorySettingsRepository creates a new in-memory settings repo
func NewInMemorySettingsRepository() *InMemorySettingsRepository {
	return &InMemorySettingsRepository{store: make(map[string]*pb.UserSettings)}
}

func (r *InMemorySettingsRepository) Get(ctx context.Context, userID string) (*pb.UserSettings, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()
	b, ok := r.store[userID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return proto.CloneOf(b), nil
}

// TODO: setting diff preview (dry_run setting on update?) endpoint

func (r *InMemorySettingsRepository) Update(ctx context.Context, settings *pb.UserSettings) error {
	if settings == nil {
		return errors.New("nonempty settings to upsert required")
	}

	if settings.UserId == nil || strings.Trim(*settings.UserId, constants.STRING_TRIMSET) == "" {
		return fmt.Errorf("user id required in settings")
	}

	userID := *settings.UserId

	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.store[userID]
	if !ok {
		r.store[userID] = proto.CloneOf(settings)
	}

	return nil
}
