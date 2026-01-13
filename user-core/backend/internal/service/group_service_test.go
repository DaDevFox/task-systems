package service

import (
	"context"
	"io"
	"testing"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGroupSubsumptionAndMembership(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	repo := repository.NewInMemoryUserRepository()
	gRepo := repository.NewInMemoryGroupRepository()
	svc := NewGroupService(gRepo, repo, logger)

	ctx := context.Background()

	// create users
	owner := domain.NewUser("owner@g.example", "Owner")
	owner.PasswordHash = "x"
	_ = repo.Create(ctx, owner)

	member := domain.NewUser("member@g.example", "Member")
	member.PasswordHash = "x"
	_ = repo.Create(ctx, member)

	// create child group
	child, err := svc.CreateGroup(ctx, "child", "Child Group", owner.ID)
	require.NoError(t, err)

	// add member to child by owner
	err = svc.AddMember(ctx, child.ID, owner.ID, member.ID, domain.GroupRoleMember)
	require.NoError(t, err)

	// create parent group and subsume child
	parent, err := svc.CreateGroup(ctx, "parent", "Parent Group", owner.ID)
	require.NoError(t, err)
	err = svc.Subsumes(ctx, parent.ID, owner.ID, child.ID)
	require.NoError(t, err)

	// ensure member of child is seen as member of parent
	isM, err := svc.IsMember(ctx, parent.ID, member.ID)
	require.NoError(t, err)
	require.True(t, isM)
}
