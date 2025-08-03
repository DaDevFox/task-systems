package repository

import (
	"context"
	"testing"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
)

const userTestID = "test-user-1"
const testEmail = "test@example.com"
const testName = "Test User"

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
		{"DeleteUser", testDeleteUser},
		{"ListAllUsers", testListAllUsers},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryUserRepository()
			tt.test(t, repo)
		})
	}
}

func testCreateAndGetUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := &domain.User{
		ID:    userTestID,
		Email: testEmail,
		Name:  testName,
	}

	// Create user
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get user by ID
	retrievedUser, err := repo.GetByID(ctx, userTestID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected ID %s, got %s", user.ID, retrievedUser.ID)
	}
	if retrievedUser.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrievedUser.Email)
	}
	if retrievedUser.Name != user.Name {
		t.Errorf("Expected name %s, got %s", user.Name, retrievedUser.Name)
	}
}

func testCreateDuplicateUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := &domain.User{
		ID:    userTestID,
		Email: testEmail,
		Name:  testName,
	}

	// Create user
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Try to create duplicate
	err = repo.Create(ctx, user)
	if err == nil {
		t.Error("Expected error for duplicate user, got none")
	}
}

func testGetNonExistentUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent user, got none")
	}
}

func testGetByEmail(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := &domain.User{
		ID:    userTestID,
		Email: testEmail,
		Name:  testName,
	}

	// Create user
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get user by email
	retrievedUser, err := repo.GetByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected ID %s, got %s", user.ID, retrievedUser.ID)
	}

	// Test non-existent email
	_, err = repo.GetByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for non-existent email, got none")
	}
}

func testUpdateUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := &domain.User{
		ID:    userTestID,
		Email: testEmail,
		Name:  testName,
	}

	// Create user
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update user
	user.Name = "Updated Name"
	user.Email = "updated@example.com"
	err = repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrievedUser, err := repo.GetByID(ctx, userTestID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrievedUser.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", retrievedUser.Name)
	}
	if retrievedUser.Email != "updated@example.com" {
		t.Errorf("Expected email 'updated@example.com', got %s", retrievedUser.Email)
	}
}

func testDeleteUser(t *testing.T, repo UserRepository) {
	ctx := context.Background()
	user := &domain.User{
		ID:    userTestID,
		Email: testEmail,
		Name:  testName,
	}

	// Create user
	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete user
	err = repo.Delete(ctx, userTestID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(ctx, userTestID)
	if err == nil {
		t.Error("Expected error for deleted user, got none")
	}
}

func testListAllUsers(t *testing.T, repo UserRepository) {
	ctx := context.Background()

	// Create multiple users
	users := []*domain.User{
		{ID: "user1", Email: "user1@example.com", Name: "User 1"},
		{ID: "user2", Email: "user2@example.com", Name: "User 2"},
		{ID: "user3", Email: "user3@example.com", Name: "User 3"},
	}

	for _, user := range users {
		err := repo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create user %s failed: %v", user.ID, err)
		}
	}

	// List all users
	allUsers, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(allUsers) != 3 {
		t.Errorf("Expected 3 users, got %d", len(allUsers))
	}

	// Verify all users are present
	userMap := make(map[string]*domain.User)
	for _, user := range allUsers {
		userMap[user.ID] = user
	}

	for _, expectedUser := range users {
		if _, exists := userMap[expectedUser.ID]; !exists {
			t.Errorf("Expected user %s not found in list", expectedUser.ID)
		}
	}
}
