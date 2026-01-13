package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
)

// BaggageRepository stores key/value metadata per user
type BaggageRepository interface {
	Get(ctx context.Context, userID string, key string) (domain.BaggageEntry, error)
	List(ctx context.Context, userID string) (domain.Baggage, error)
	Put(ctx context.Context, userID string, entry domain.BaggageEntry) error
	Delete(ctx context.Context, userID string, key string) error
}

// InMemoryBaggageRepository is a testable in-memory implementation
type InMemoryBaggageRepository struct {
	store map[string]domain.Baggage // userID -> baggage map
	mutex sync.RWMutex
}

// NewInMemoryBaggageRepository creates a new in-memory baggage repo
func NewInMemoryBaggageRepository() *InMemoryBaggageRepository {
	return &InMemoryBaggageRepository{store: make(map[string]domain.Baggage)}
}

func (r *InMemoryBaggageRepository) Get(ctx context.Context, userID string, key string) (domain.BaggageEntry, error) {
	if userID == "" || key == "" {
		return domain.BaggageEntry{}, fmt.Errorf("user id and key required")
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	b, ok := r.store[userID]
	if !ok {
		return domain.BaggageEntry{}, fmt.Errorf("not found")
	}
	entry, ok := b[key]
	if !ok {
		return domain.BaggageEntry{}, fmt.Errorf("not found")
	}
	return entry, nil
}

func (r *InMemoryBaggageRepository) List(ctx context.Context, userID string) (domain.Baggage, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id required")
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	b, ok := r.store[userID]
	if !ok {
		return domain.Baggage{}, nil
	}
	copyB := make(domain.Baggage)
	for k, v := range b {
		copyB[k] = v
	}
	return copyB, nil
}

func (r *InMemoryBaggageRepository) Put(ctx context.Context, userID string, entry domain.BaggageEntry) error {
	if userID == "" || entry.Key == "" {
		return fmt.Errorf("user id and key required")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	b, ok := r.store[userID]
	if !ok {
		b = make(domain.Baggage)
		r.store[userID] = b
	}
	entry.UpdatedAt = time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	b[entry.Key] = entry
	return nil
}

func (r *InMemoryBaggageRepository) Delete(ctx context.Context, userID string, key string) error {
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
