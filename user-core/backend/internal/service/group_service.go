package service

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/DaDevFox/hof"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
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
func (s *GroupService) CreateGroup(ctx context.Context, id, name, ownerUserID string) (*pb.Group, error) {
	if id == "" || name == "" {
		return nil, fmt.Errorf("id and name required")
	}
	g := &pb.Group{
		Id:   &id,
		Name: &name,
		Children: []*pb.Member{
			&pb.Member{
				Id:        &ownerUserID,
				Privilege: pb.Privilege_OWNER.Enum()},
		},
	}

	if err := s.groupRepo.Create(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// AddMember adds a user to a group with role enforcement
func (s *GroupService) SetMembership(ctx context.Context, groupID, userID string, privilege *pb.Privilege) error {
	// requester must be owner or admin for group; owner can assign admin
	g, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to load group: %w", err)
	}

	updateTo := proto.CloneOf(g)
	slices.DeleteFunc(updateTo.Children, func(member *pb.Member) bool {
		return *member.Id == userID
	})
	updateTo.Children = append(updateTo.Children, &pb.Member{
		Id:        &userID,
		Privilege: privilege,
	})

	s.groupRepo.Update(ctx, updateTo)

	return nil
}

// NOTE: removes group if last member is removed
// RemoveMember removes a member; owner can't be removed by admin
func (s *GroupService) RemoveMember(ctx context.Context, groupID, userID string) error {
	g, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to load group: %w", err)
	}

	updateTo := proto.CloneOf(g)
	slices.DeleteFunc(updateTo.Children, func(member *pb.Member) bool {
		return *member.Id == userID
	})

	if len(updateTo.Children) == 0 {
		s.groupRepo.Delete(ctx, *updateTo.Id)
	} else {
		s.groupRepo.Update(ctx, updateTo)
	}

	return s.groupRepo.Update(ctx, g)
}

// Subsumes establishes a subsumption (parent subsumes child) TODO: consider diff functionality (flat hoist?); curr just an alias for AddMember
func (s *GroupService) Subsumes(ctx context.Context, parentID, childID string, privilege *pb.Privilege) error {
	return s.SetMembership(ctx, parentID, childID, privilege)
}

// IsMember checks subsumption chain to determine membership
func (s *GroupService) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	visited := make(map[string]bool)
	queue := make(chan string)

	for id := range queue {
		if visited[id] {
			continue
		}
		visited[id] = true

		g, err := s.groupRepo.GetByID(ctx, id)
		if err != nil {
			s.logger.WithError(fmt.Errorf("failed to load group %s: %w", id, err)).Warn("skipping group")
			continue
		}

		found := hof.Some(g.Children, func(member *pb.Member) bool {
			if member.Id == nil {
				s.logger.WithError(errors.New("invalid user (no id)")).Error("skipping; potentially fatal to other systems")
				return false
			}
			return *member.Id == userID
		})

		if found {
			return true, nil
		}

		for _, child := range g.Children {
			queue <- *child.Id
		}
	}

	return false, nil

}
