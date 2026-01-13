package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
)

// GroupRepository defines persistence for groups
type GroupRepository interface {
	Create(ctx context.Context, g *domain.Group) error
	GetByID(ctx context.Context, id string) (*domain.Group, error)
	Update(ctx context.Context, g *domain.Group) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*domain.Group, error)
}

// InMemoryGroupRepository is an in-memory implementation for tests/dev
type InMemoryGroupRepository struct {
	groups map[string]*domain.Group
	mutex  sync.RWMutex
}

// NewInMemoryGroupRepository creates a new repository
func NewInMemoryGroupRepository() *InMemoryGroupRepository {
	return &InMemoryGroupRepository{groups: make(map[string]*domain.Group)}
}

func (r *InMemoryGroupRepository) Create(ctx context.Context, g *domain.Group) error {
	if g == nil {
		return fmt.Errorf("group cannot be nil")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, exists := r.groups[g.ID]; exists {
		return fmt.Errorf("group already exists")
	}
	copyG := *g
	r.groups[g.ID] = &copyG
	return nil
}

func (r *InMemoryGroupRepository) GetByID(ctx context.Context, id string) (*domain.Group, error) {
	if id == "" {
		return nil, fmt.Errorf("group id required")
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	g, ok := r.groups[id]
	if !ok {
		return nil, fmt.Errorf("group not found")
	}
	copyG := *g
	return &copyG, nil
}

func (r *InMemoryGroupRepository) Update(ctx context.Context, g *domain.Group) error {
	if g == nil {
		return fmt.Errorf("group cannot be nil")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.groups[g.ID]; !ok {
		return fmt.Errorf("group not found")
	}
	copyG := *g
	r.groups[g.ID] = &copyG
	return nil
}

func (r *InMemoryGroupRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("group id required")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.groups, id)
	return nil
}

func (r *InMemoryGroupRepository) List(ctx context.Context) ([]*domain.Group, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var out []*domain.Group
	for _, g := range r.groups {
		copyG := *g
		out = append(out, &copyG)
	}
	return out, nil
}
