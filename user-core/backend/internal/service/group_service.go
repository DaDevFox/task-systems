package service

import (
	"context"
	"fmt"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

// GroupService coordinates group operations using repositories
type GroupService struct {
	groupRepo repository.GroupRepository
	userRepo  repository.UserRepository
	logger    *logrus.Logger
}

// NewGroupService constructs a GroupService
func NewGroupService(groupRepo repository.GroupRepository, userRepo repository.UserRepository, logger *logrus.Logger) *GroupService {
	if logger == nil {
		logger = logrus.New()
	}
	if groupRepo == nil {
		groupRepo = repository.NewInMemoryGroupRepository()
	}
	return &GroupService{groupRepo: groupRepo, userRepo: userRepo, logger: logger}
}

// CreateGroup creates a new group with an owner
func (s *GroupService) CreateGroup(ctx context.Context, id, name, ownerUserID string) (*domain.Group, error) {
	if id == "" || name == "" {
		return nil, fmt.Errorf("id and name required")
	}
	g, err := domain.NewGroup(id, name, ownerUserID)
	if err != nil {
		return nil, err
	}
	if err := s.groupRepo.Create(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// AddMember adds a user to a group with role enforcement
func (s *GroupService) AddMember(ctx context.Context, groupID, requesterID, targetUserID string, role domain.GroupRole) error {
	// requester must be owner or admin for group; owner can assign admin
	g, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to load group: %w", err)
	}
	reqRole := g.GetRole(requesterID)
	if reqRole != domain.GroupRoleOwner && reqRole != domain.GroupRoleAdmin {
		return fmt.Errorf("permission denied")
	}
	// Admins cannot promote to owner
	if reqRole == domain.GroupRoleAdmin && role == domain.GroupRoleOwner {
		return fmt.Errorf("admins cannot assign owner role")
	}
	// Admins cannot set other admins? per OBJECTIVE admins may not set others to admin; adjust: admins may add/remove members excepting owners and may not set others to admin.
	if reqRole == domain.GroupRoleAdmin && role == domain.GroupRoleAdmin {
		return fmt.Errorf("admins cannot assign admin role")
	}
	// Owners can set admin or transfer ownership
	if err := g.AddMember(targetUserID, role); err != nil {
		return err
	}
	if err := s.groupRepo.Update(ctx, g); err != nil {
		return err
	}
	return nil
}

// RemoveMember removes a member; owner can't be removed by admin
func (s *GroupService) RemoveMember(ctx context.Context, groupID, requesterID, targetUserID string) error {
	g, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to load group: %w", err)
	}
	reqRole := g.GetRole(requesterID)
	targetRole := g.GetRole(targetUserID)
	if reqRole != domain.GroupRoleOwner && reqRole != domain.GroupRoleAdmin {
		return fmt.Errorf("permission denied")
	}
	if reqRole == domain.GroupRoleAdmin && targetRole == domain.GroupRoleOwner {
		return fmt.Errorf("admins cannot remove owners")
	}
	g.RemoveMember(targetUserID)
	return s.groupRepo.Update(ctx, g)
}

// Subsumes establishes a subsumption (parent subsumes child)
func (s *GroupService) Subsumes(ctx context.Context, parentID, requesterID, childID string) error {
	parent, err := s.groupRepo.GetByID(ctx, parentID)
	if err != nil {
		return fmt.Errorf("failed to load parent group: %w", err)
	}
	reqRole := parent.GetRole(requesterID)
	if reqRole != domain.GroupRoleOwner && reqRole != domain.GroupRoleAdmin {
		return fmt.Errorf("permission denied")
	}
	// ensure child exists
	if _, err := s.groupRepo.GetByID(ctx, childID); err != nil {
		return fmt.Errorf("failed to load child group: %w", err)
	}
	if err := parent.AddSubsumed(childID); err != nil {
		return err
	}
	return s.groupRepo.Update(ctx, parent)
}

// IsMember checks subsumption chain to determine membership
func (s *GroupService) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	visited := make(map[string]bool)
	var dfs func(id string) (bool, error)
	dfs = func(id string) (bool, error) {
		if visited[id] {
			return false, nil
		}
		visited[id] = true
		g, err := s.groupRepo.GetByID(ctx, id)
		if err != nil {
			return false, fmt.Errorf("failed to load group: %w", err)
		}
		if g.IsMember(userID) {
			return true, nil
		}
		for child := range g.SubsumedGroups {
			in, err := dfs(child)
			if err != nil {
				return false, err
			}
			if in {
				return true, nil
			}
		}
		return false, nil
	}
	return dfs(groupID)
}
