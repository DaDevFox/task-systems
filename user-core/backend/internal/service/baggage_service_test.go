package service

import (
	"context"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestBaggageCRUDPermissions(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	repo := repository.NewInMemoryUserRepository()
	bRepo := repository.NewInMemoryBaggageRepository()
	svc := NewBaggageService(bRepo, repo, logger)

	ctx := context.Background()
	// create two users
	u1 := domain.NewUser("owner@example.com", "Owner")
	u1.PasswordHash = "x"
	_ = repo.Create(ctx, u1)

	u2 := domain.NewUser("other@example.com", "Other")
	u2.PasswordHash = "x"
	_ = repo.Create(ctx, u2)

	// owner puts baggage
	entry := domain.BaggageEntry{Key: "theme", Value: "dark"}
	reqErr := svc.Put(ctx, u1.ID, u1.ID, entry)
	require.NoError(t, reqErr)

	// owner can get
	got, err := svc.Get(ctx, u1.ID, u1.ID, "theme")
	require.NoError(t, err)
	require.Equal(t, "dark", got.Value)

	// other cannot get
	_, err = svc.Get(ctx, u2.ID, u1.ID, "theme")
	require.Error(t, err)

	// promote u2 to admin via user repo (global admin)
	u2.Role = domain.UserRoleAdmin
	_ = repo.Update(ctx, u2)

	// now u2 (admin) can get
	got2, err := svc.Get(ctx, u2.ID, u1.ID, "theme")
	require.NoError(t, err)
	require.Equal(t, "dark", got2.Value)

	// delete by owner
	err = svc.Delete(ctx, u1.ID, u1.ID, "theme")
	require.NoError(t, err)

	// ensure deleted
	_, err = svc.Get(ctx, u1.ID, u1.ID, "theme")
	require.Error(t, err)
}
