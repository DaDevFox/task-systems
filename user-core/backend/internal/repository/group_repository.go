package repository

import (
	"context"
	"fmt"
	"sync"

	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
	"google.golang.org/protobuf/proto"
)

// GroupRepository defines persistence for groups
type GroupRepository interface {
	Create(ctx context.Context, g *pb.Group) error
	GetByID(ctx context.Context, id string) (*pb.Group, error)
	Update(ctx context.Context, g *pb.Group) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*pb.Group, error)
}

// InMemoryGroupRepository is an in-memory implementation for tests/dev
type InMemoryGroupRepository struct {
	groups map[string]*pb.Group
	mutex  sync.RWMutex
}

// NewInMemoryGroupRepository creates a new repository
func NewInMemoryGroupRepository() *InMemoryGroupRepository {
	return &InMemoryGroupRepository{groups: make(map[string]*pb.Group)}
}

func (r *InMemoryGroupRepository) Create(ctx context.Context, g *pb.Group) error {
	if g == nil || g.Id == nil {
		return fmt.Errorf("group/group id cannot be nil")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, exists := r.groups[*g.Id]; exists {
		return fmt.Errorf("group already exists")
	}
	r.groups[*g.Id] = proto.CloneOf(g)
	return nil
}

func (r *InMemoryGroupRepository) GetByID(ctx context.Context, id string) (*pb.Group, error) {
	if id == "" {
		return nil, fmt.Errorf("group id required")
	}
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	g, ok := r.groups[id]
	if !ok {
		return nil, fmt.Errorf("group not found")
	}
	return proto.CloneOf(g), nil
}

func (r *InMemoryGroupRepository) Update(ctx context.Context, g *pb.Group) error {
	if g == nil || g.Id == nil {
		return fmt.Errorf("group/group id cannot be nil")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.groups[*g.Id]; !ok {
		return fmt.Errorf("group not found")
	}
	r.groups[*g.Id] = proto.CloneOf(g)
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

func (r *InMemoryGroupRepository) List(ctx context.Context) ([]*pb.Group, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	var out []*pb.Group
	for _, g := range r.groups {
		out = append(out, proto.CloneOf(g))
	}
	return out, nil
}
