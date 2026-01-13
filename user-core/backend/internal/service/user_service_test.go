package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/security"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newTestUserService(t *testing.T) (*UserService, repository.UserRepository, context.Context) {
	t.Helper()

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	repo := repository.NewInMemoryUserRepository()
	svc := NewUserService(repo, logger)

	return svc, repo, context.Background()
}

func TestUserServiceCreateUserHashesPassword(t *testing.T) {
	svc, repo, ctx := newTestUserService(t)

	params := CreateUserParams{
		Email:    "hash@test.dev",
		Name:     "Hash Tester",
		Password: "Str0ngPassw0rd!",
	}

	createdUser, err := svc.CreateUser(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, createdUser)

	assert.NotEqual(t, params.Password, createdUser.PasswordHash)
	assert.NotEmpty(t, createdUser.PasswordHash)

	compareErr := bcrypt.CompareHashAndPassword([]byte(createdUser.PasswordHash), []byte(params.Password))
	assert.NoError(t, compareErr)

	storedUser, err := repo.GetByID(ctx, createdUser.ID)
	require.NoError(t, err)
	assert.Equal(t, createdUser.PasswordHash, storedUser.PasswordHash)
}

func TestUserServiceCreateUserRejectsShortPassword(t *testing.T) {
	svc, _, ctx := newTestUserService(t)

	shortPassword := strings.Repeat("a", security.MinPasswordLength-1)

	params := CreateUserParams{
		Email:    "short@test.dev",
		Name:     "Short Password",
		Password: shortPassword,
	}

	createdUser, err := svc.CreateUser(ctx, params)
	require.Error(t, err)
	assert.Nil(t, createdUser)

	expectedMessage := fmt.Sprintf("password must be at least %d characters", security.MinPasswordLength)
	assert.Equal(t, expectedMessage, err.Error())
}

func TestUserServiceUpdateUserPreservesPasswordHash(t *testing.T) {
	svc, repo, ctx := newTestUserService(t)

	params := CreateUserParams{
		Email:    "update@test.dev",
		Name:     "Updater",
		Password: "Sup3rPassw0rd!",
	}

	createdUser, err := svc.CreateUser(ctx, params)
	require.NoError(t, err)

	originalHash := createdUser.PasswordHash
	require.NotEmpty(t, originalHash)

	createdUser.PasswordHash = ""
	createdUser.Name = "Updated Name"

	updatedUser, err := svc.UpdateUser(ctx, createdUser)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)

	assert.Equal(t, "Updated Name", updatedUser.Name)
	assert.Equal(t, originalHash, updatedUser.PasswordHash)

	storedUser, err := repo.GetByID(ctx, createdUser.ID)
	require.NoError(t, err)
	assert.Equal(t, originalHash, storedUser.PasswordHash)
}
