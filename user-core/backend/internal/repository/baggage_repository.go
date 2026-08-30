package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
)

// SettingsRepository stores key/value metadata per user
type SettingsRepository interface {
	Get(ctx context.Context, userID string, key string) (domain.SettingsEntry, error)
	List(ctx context.Context, userID string) (domain.Settings, error)
	Put(ctx context.Context, userID string, entry domain.SettingsEntry) error
	Delete(ctx context.Context, userID string, key string) error
}

// InMemorySettingsRepository is a testable in-memory implementation
type InMemorySettingsRepository struct {
	store map[string]domain.Settings // userID -> settings map
	mutex sync.RWMutex
}

// NewInMemorySettingsRepository creates a new in-memory settings repo
func NewInMemorySettingsRepository() *InMemorySettingsRepository {
	return &InMemorySettingsRepository{store: make(map[string]domain.Settings)}
}

func (r *InMemorySettingsRepository) Get(ctx context.Context, userID string, key string) (domain.SettingsEntry, error) {
	if userID == "" || key == "" {
		return domain.SettingsEntry{}, fmt.Errorf("user id and key required")
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	b, ok := r.store[userID]
	if !ok {
		return domain.SettingsEntry{}, fmt.Errorf("not found")
	}
	entry, ok := b[key]
	if !ok {
		return domain.SettingsEntry{}, fmt.Errorf("not found")
	}
	return entry, nil
}

func (r *InMemorySettingsRepository) List(ctx context.Context, userID string) (domain.Settings, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id required")
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	b, ok := r.store[userID]
	if !ok {
		return domain.Settings{}, nil
	}
	copyB := make(domain.Settings)
	for k, v := range b {
		copyB[k] = v
	}
	return copyB, nil
}

func (r *InMemorySettingsRepository) Put(ctx context.Context, userID string, entry domain.SettingsEntry) error {
	if userID == "" || entry.Key == "" {
		return fmt.Errorf("user id and key required")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	b, ok := r.store[userID]
	if !ok {
		b = make(domain.Settings)
		r.store[userID] = b
	}
	entry.UpdatedAt = time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	b[entry.Key] = entry
	return nil
}

func (r *InMemorySettingsRepository) Delete(ctx context.Context, userID string, key string) error {
	if userID == "" || key == "" {
		return fmt.Errorf("user id and key required")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	b, ok := r.store[userID]
	if !ok {
		return fmt.Errorf("not found")
	}
	delete(b, key)
	return nil
}
