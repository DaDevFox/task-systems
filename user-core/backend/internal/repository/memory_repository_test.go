package repository

import (
	"context"
	"os"
	"testing"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryUserRepository(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T, UserRepository)
	}{
		{"CreateAndGetUser", testCreateAndGetUser},
		{"CreateDuplicateUser", testCreateDuplicateUser},
		{"GetNonExistentUser", testGetNonExistentUser},
		{"GetByEmail", testGetByEmail},
		{"UpdateUser", testUpdateUser},
		{"SoftDeleteUser", testSoftDeleteUser},
		{"HardDeleteUser", testHardDeleteUser},
		{"ListUsers", testListUsers},
		{"SearchUsers", testSearchUsers},
		{"BulkGetUsers", testBulkGetUsers},
		{"ValidateUser", testValidateUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryUserRepository()
			tt.test(t, repo)
		})
	}
}

func TestBadgerUserRepository(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T, UserRepository)
	}{
		{"CreateAndGetUser", testCreateAndGetUser},
		{"CreateDuplicateUser", testCreateDuplicateUser},
		{"GetNonExistentUser", testGetNonExistentUser},
		{"GetByEmail", testGetByEmail},
		{"UpdateUser", testUpdateUser},
		{"SoftDeleteUser", testSoftDeleteUser},
		{"HardDeleteUser", testHardDeleteUser},
		{"ListUsers", testListUsers},
		{"SearchUsers", testSearchUsers},
		{"BulkGetUsers", testBulkGetUsers},
		{"ValidateUser", testValidateUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for BadgerDB
			tempDir, err := os.MkdirTemp("", "badger_test_*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir) // Clean up after test

			repo, err := NewBadgerUserRepository(tempDir, logrus.New())
			require.NoError(t, err)
			defer repo.Close() // Close the database after test

			tt.test(t, repo)
		})
	}
}

func testCreateAndGetUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := domain.NewUser("test@example.com", "Test User")
	user.FirstName = "Test"
	user.LastName = "User"

	// Create user
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Get user by ID
	retrievedUser, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Email, retrievedUser.Email)
	assert.Equal(t, user.Name, retrievedUser.Name)
	assert.Equal(t, user.FirstName, retrievedUser.FirstName)
	assert.Equal(t, user.LastName, retrievedUser.LastName)
}

func testCreateDuplicateUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := domain.NewUser("test@example.com", "Test User")

	// Create user
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Try to create duplicate by ID
	duplicateByID := domain.NewUser("different@example.com", "Different User")
	duplicateByID.ID = user.ID
	err = repo.Create(ctx, duplicateByID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Try to create duplicate by email
	duplicateByEmail := domain.NewUser("test@example.com", "Different User")
	err = repo.Create(ctx, duplicateByEmail)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func testGetNonExistentUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "non-existent")
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)

	_, err = repo.GetByEmail(ctx, "non-existent@example.com")
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func testGetByEmail(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := domain.NewUser("test@example.com", "Test User")

	// Create user
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Get user by email
	retrievedUser, err := repo.GetByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Email, retrievedUser.Email)
}

func testUpdateUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := domain.NewUser("test@example.com", "Test User")

	// Create user
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Update user
	user.Name = "Updated Name"
	user.Email = "updated@example.com"
	err = repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify update
	retrievedUser, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrievedUser.Name)
	assert.Equal(t, "updated@example.com", retrievedUser.Email)
}

func testSoftDeleteUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := domain.NewUser("test@example.com", "Test User")

	// Create user
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Soft delete user
	err = repo.Delete(ctx, user.ID, false)
	require.NoError(t, err)

	// Verify user still exists but is inactive
	retrievedUser, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.UserStatusInactive, retrievedUser.Status)

	// Verify exists check shows inactive
	exists, active, err := repo.Exists(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.True(t, active != domain.UserStatusActive)
}

func testHardDeleteUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := domain.NewUser("test@example.com", "Test User")

	// Create user
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Hard delete user
	err = repo.Delete(ctx, user.ID, true)
	require.NoError(t, err)

	// Verify user no longer exists
	_, err = repo.GetByID(ctx, user.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)

	// Verify exists check shows not found
	exists, active, err := repo.Exists(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.False(t, active == domain.UserStatusActive)
}

func testListUsers(t *testing.T, repo UserRepository) {
	ctx := context.Background()

	// Create multiple users
	users := []*domain.User{
		domain.NewUser("user1@example.com", "User 1"),
		domain.NewUser("user2@example.com", "User 2"),
		domain.NewUser("admin@example.com", "Admin User"),
	}
	users[2].Role = domain.UserRoleAdmin

	for _, user := range users {
		err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	// List all users
	filter := ListUsersFilter{PageSize: 10}
	allUsers, _, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, allUsers, 3)

	// Filter by role
	adminRole := domain.UserRoleAdmin
	filter.Role = &adminRole
	adminUsers, _, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, adminUsers, 1)
	assert.Equal(t, domain.UserRoleAdmin, adminUsers[0].Role)

	// Filter by name prefix
	filter = ListUsersFilter{NamePrefix: "User", PageSize: 10}
	filteredUsers, _, err := repo.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, filteredUsers, 2)
}

func testSearchUsers(t *testing.T, repo UserRepository) {
	ctx := context.Background()

	// Create test users
	users := []*domain.User{
		domain.NewUser("john.doe@example.com", "John Doe"),
		domain.NewUser("jane.smith@example.com", "Jane Smith"),
		domain.NewUser("bob.johnson@example.com", "Bob Johnson"),
	}
	users[0].FirstName = "John"
	users[0].LastName = "Doe"

	for _, user := range users {
		err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	// Search by name
	results, err := repo.Search(ctx, "john", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2) // Should find "John Doe" and "Bob Johnson"

	// Search by email
	results, err = repo.Search(ctx, "jane", 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Jane Smith", results[0].Name)

	// Search with limit
	results, err = repo.Search(ctx, "example.com", 1)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func testBulkGetUsers(t *testing.T, repo UserRepository) {
	ctx := context.Background()

	// Create test users
	users := []*domain.User{
		domain.NewUser("user1@example.com", "User 1"),
		domain.NewUser("user2@example.com", "User 2"),
		domain.NewUser("user3@example.com", "User 3"),
	}

	for _, user := range users {
		err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	// Bulk get existing users
	userIDs := []string{users[0].ID, users[1].ID}
	foundUsers, notFoundIDs, err := repo.BulkGet(ctx, userIDs)
	require.NoError(t, err)
	assert.Len(t, foundUsers, 2)
	assert.Len(t, notFoundIDs, 0)

	// Bulk get with some non-existent users
	userIDs = []string{users[0].ID, "non-existent", users[2].ID}
	foundUsers, notFoundIDs, err = repo.BulkGet(ctx, userIDs)
	require.NoError(t, err)
	assert.Len(t, foundUsers, 2)
	assert.Len(t, notFoundIDs, 1)
	assert.Equal(t, "non-existent", notFoundIDs[0])
}

func testValidateUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := domain.NewUser("test@example.com", "Test User")

	// Create user
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Validate existing active user
	exists, active, err := repo.Exists(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.True(t, active == domain.UserStatusActive)

	// Validate non-existent user
	exists, active, err = repo.Exists(ctx, "non-existent")
	require.NoError(t, err)
	assert.False(t, exists)
	assert.False(t, active == domain.UserStatusActive)
}
