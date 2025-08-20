package repository

import (
	"context"
	"testing"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryUserRepository(t *testing.T) {
	repo := NewInMemoryUserRepository()
	ctx := context.Background()

	// Test user creation
	user := domain.NewUser("test@example.com", "Test User")
	user.FirstName = "Test"
	user.LastName = "User"

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Test get by ID
	retrievedUser, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, retrievedUser.Email)
	assert.Equal(t, user.Name, retrievedUser.Name)

	// Test get by email
	retrievedUser, err = repo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)

	// Test get by name
	retrievedUser, err = repo.GetByName(ctx, user.Name)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)

	// Test user exists
	exists, status, err := repo.Exists(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, domain.UserStatusActive, status)

	// Test user update
	user.Name = "Updated Name"
	err = repo.Update(ctx, user)
	require.NoError(t, err)

	retrievedUser, err = repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrievedUser.Name)

	// Test search
	users, err := repo.Search(ctx, "Updated", 10)
	require.NoError(t, err)
	assert.Len(t, users, 1)

	// Test listing
	users, nextToken, err := repo.List(ctx, ListUsersFilter{PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Empty(t, nextToken)

	// Test count
	count, err := repo.Count(ctx, ListUsersFilter{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Test bulk get
	foundUsers, notFound, err := repo.BulkGet(ctx, []string{user.ID, "nonexistent"})
	require.NoError(t, err)
	assert.Len(t, foundUsers, 1)
	assert.Len(t, notFound, 1)

	// Test soft delete
	err = repo.Delete(ctx, user.ID, false)
	require.NoError(t, err)

	exists, status, err = repo.Exists(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, domain.UserStatusInactive, status)

	// Test hard delete
	err = repo.Delete(ctx, user.ID, true)
	require.NoError(t, err)

	exists, _, err = repo.Exists(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestInMemoryUserRepository_Validation(t *testing.T) {
	repo := NewInMemoryUserRepository()
	ctx := context.Background()

	// Test duplicate email
	user1 := domain.NewUser("test@example.com", "User 1")
	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	user2 := domain.NewUser("test@example.com", "User 2")
	err = repo.Create(ctx, user2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Test empty user ID
	_, err = repo.GetByID(ctx, "")
	assert.Error(t, err)

	// Test nonexistent user
	_, err = repo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotFound)
}
