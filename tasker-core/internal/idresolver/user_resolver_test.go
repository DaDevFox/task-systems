package idresolver

import (
	"testing"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
)

func TestUserResolverResolveUser(t *testing.T) {
	resolver := NewUserResolver()

	users := []*domain.User{
		{ID: "user123", Name: "Alice Smith", Email: "alice@example.com"},
		{ID: "user456", Name: "Bob Johnson", Email: "bob@example.com"},
		{ID: "user789", Name: "Carol Davis", Email: "carol@example.com"},
		{ID: "user999", Name: "David Wilson", Email: "david@example.com"},
	}

	err := resolver.UpdateUsers(users)
	if err != nil {
		t.Fatalf("Failed to update users: %v", err)
	}

	tests := []struct {
		name        string
		identifier  string
		expectedID  string
		shouldError bool
	}{
		{
			name:        "Resolve by exact ID",
			identifier:  "user123",
			expectedID:  "user123",
			shouldError: false,
		},
		{
			name:        "Resolve by exact name",
			identifier:  "Alice Smith",
			expectedID:  "user123",
			shouldError: false,
		},
		{
			name:        "Resolve by name case insensitive",
			identifier:  "alice smith",
			expectedID:  "user123",
			shouldError: false,
		},
		{
			name:        "Resolve by partial name match",
			identifier:  "Bob",
			expectedID:  "user456",
			shouldError: false,
		},
		{
			name:        "Resolve by partial name case insensitive",
			identifier:  "carol",
			expectedID:  "user789",
			shouldError: false,
		},
		{
			name:        "Ambiguous partial name",
			identifier:  "Dav", // Could match "David" but only one user, so should work
			expectedID:  "user999",
			shouldError: false,
		},
		{
			name:        "Non-existent user",
			identifier:  "NonExistent",
			expectedID:  "",
			shouldError: true,
		},
		{
			name:        "Empty identifier",
			identifier:  "",
			expectedID:  "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := resolver.ResolveUser(tt.identifier)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for identifier '%s', but got none", tt.identifier)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for identifier '%s': %v", tt.identifier, err)
				return
			}

			if user.ID != tt.expectedID {
				t.Errorf("For identifier '%s', expected user ID '%s', got '%s'", tt.identifier, tt.expectedID, user.ID)
			}
		})
	}
}

func TestUserResolverResolveUserID(t *testing.T) {
	resolver := NewUserResolver()

	users := []*domain.User{
		{ID: "user123", Name: "Alice Smith", Email: "alice@example.com"},
		{ID: "user456", Name: "Bob Johnson", Email: "bob@example.com"},
	}

	err := resolver.UpdateUsers(users)
	if err != nil {
		t.Fatalf("Failed to update users: %v", err)
	}

	tests := []struct {
		identifier  string
		expectedID  string
		shouldError bool
	}{
		{
			identifier:  "Alice Smith",
			expectedID:  "user123",
			shouldError: false,
		},
		{
			identifier:  "user456",
			expectedID:  "user456",
			shouldError: false,
		},
		{
			identifier:  "NonExistent",
			expectedID:  "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.identifier, func(t *testing.T) {
			userID, err := resolver.ResolveUserID(tt.identifier)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for identifier '%s', but got none", tt.identifier)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for identifier '%s': %v", tt.identifier, err)
				return
			}

			if userID != tt.expectedID {
				t.Errorf("For identifier '%s', expected user ID '%s', got '%s'", tt.identifier, tt.expectedID, userID)
			}
		})
	}
}

func TestUserResolverDuplicateNames(t *testing.T) {
	resolver := NewUserResolver()

	// Users with duplicate names should cause an error
	users := []*domain.User{
		{ID: "user123", Name: "Alice Smith", Email: "alice1@example.com"},
		{ID: "user456", Name: "Alice Smith", Email: "alice2@example.com"}, // Duplicate name
		{ID: "user789", Name: "Bob Johnson", Email: "bob@example.com"},
	}

	err := resolver.UpdateUsers(users)
	if err == nil {
		t.Error("Expected error due to duplicate user names, but got none")
	}

	// Should contain information about the duplicate name
	if err != nil && !contains(err.Error(), "Alice Smith") {
		t.Errorf("Error message should mention the duplicate name 'Alice Smith': %v", err)
	}
}

func TestUserResolverAmbiguousPartialMatch(t *testing.T) {
	resolver := NewUserResolver()

	users := []*domain.User{
		{ID: "user123", Name: "Alice Anderson", Email: "alice.a@example.com"},
		{ID: "user456", Name: "Alice Adams", Email: "alice.adams@example.com"},
		{ID: "user789", Name: "Bob Johnson", Email: "bob@example.com"},
	}

	err := resolver.UpdateUsers(users)
	if err != nil {
		t.Fatalf("Failed to update users: %v", err)
	}

	// "Alice" should be ambiguous since it matches both "Alice Anderson" and "Alice Adams"
	_, err = resolver.ResolveUser("Alice")
	if err == nil {
		t.Error("Expected error due to ambiguous partial match, but got none")
	}

	// Should mention both matching names
	if err != nil {
		errorMsg := err.Error()
		if !contains(errorMsg, "Alice Anderson") || !contains(errorMsg, "Alice Adams") {
			t.Errorf("Error message should mention both matching names: %v", err)
		}
	}
}

func TestUserResolverValidation(t *testing.T) {
	resolver := NewUserResolver()

	users := []*domain.User{
		{ID: "user123", Name: "Alice Smith", Email: "alice@example.com"},
		{ID: "user456", Name: "Bob Johnson", Email: "bob@example.com"},
	}

	err := resolver.UpdateUsers(users)
	if err != nil {
		t.Fatalf("Failed to update users: %v", err)
	}

	// Test ValidateUserExists
	t.Run("ValidateUserExists", func(t *testing.T) {
		// Existing user should not error
		err := resolver.ValidateUserExists("user123")
		if err != nil {
			t.Errorf("Unexpected error validating existing user: %v", err)
		}

		// Non-existing user should error
		err = resolver.ValidateUserExists("nonexistent")
		if err == nil {
			t.Error("Expected error validating non-existent user, but got none")
		}
	})

	// Test ValidateUserNameUnique
	t.Run("ValidateUserNameUnique", func(t *testing.T) {
		// New unique name should not error
		err := resolver.ValidateUserNameUnique("Charlie Brown", "")
		if err != nil {
			t.Errorf("Unexpected error validating unique name: %v", err)
		}

		// Existing name should error when not excluding the same user
		err = resolver.ValidateUserNameUnique("Alice Smith", "")
		if err == nil {
			t.Error("Expected error validating existing name, but got none")
		}

		// Existing name should not error when excluding the same user
		err = resolver.ValidateUserNameUnique("Alice Smith", "user123")
		if err != nil {
			t.Errorf("Unexpected error validating existing name with exclusion: %v", err)
		}

		// Case insensitive check
		err = resolver.ValidateUserNameUnique("alice smith", "")
		if err == nil {
			t.Error("Expected error validating existing name (case insensitive), but got none")
		}
	})
}

func TestUserResolverSuggestions(t *testing.T) {
	resolver := NewUserResolver()

	users := []*domain.User{
		{ID: "user123", Name: "Alice Smith", Email: "alice@example.com"},
		{ID: "user456", Name: "Alice Adams", Email: "alice.adams@example.com"},
		{ID: "user789", Name: "Bob Johnson", Email: "bob@example.com"},
		{ID: "user999", Name: "Charlie Brown", Email: "charlie@example.com"},
	}

	err := resolver.UpdateUsers(users)
	if err != nil {
		t.Fatalf("Failed to update users: %v", err)
	}

	tests := []struct {
		identifier       string
		maxSuggestions   int
		expectedCount    int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			identifier:     "Ali",
			maxSuggestions: 5,
			expectedCount:  2,
			shouldContain:  []string{"Alice Smith", "Alice Adams"},
		},
		{
			identifier:     "A",
			maxSuggestions: 1,
			expectedCount:  1, // Should be limited to 1
		},
		{
			identifier:     "Bob",
			maxSuggestions: 5,
			expectedCount:  1,
			shouldContain:  []string{"Bob Johnson"},
		},
		{
			identifier:     "user123",
			maxSuggestions: 5,
			expectedCount:  1,
			shouldContain:  []string{"user123"},
		},
		{
			identifier:     "nonexistent",
			maxSuggestions: 5,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.identifier, func(t *testing.T) {
			suggestions := resolver.SuggestUsers(tt.identifier, tt.maxSuggestions)

			if len(suggestions) != tt.expectedCount {
				t.Errorf("Expected %d suggestions, got %d: %v", tt.expectedCount, len(suggestions), suggestions)
			}

			for _, expected := range tt.shouldContain {
				found := false
				for _, suggestion := range suggestions {
					if suggestion == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion '%s' not found in results: %v", expected, suggestions)
				}
			}

			for _, notExpected := range tt.shouldNotContain {
				for _, suggestion := range suggestions {
					if suggestion == notExpected {
						t.Errorf("Unexpected suggestion '%s' found in results: %v", notExpected, suggestions)
					}
				}
			}
		})
	}
}

func TestUserResolverGetAllUsers(t *testing.T) {
	resolver := NewUserResolver()

	users := []*domain.User{
		{ID: "user123", Name: "Alice Smith", Email: "alice@example.com"},
		{ID: "user456", Name: "Bob Johnson", Email: "bob@example.com"},
	}

	err := resolver.UpdateUsers(users)
	if err != nil {
		t.Fatalf("Failed to update users: %v", err)
	}

	allUsers := resolver.GetAllUsers()

	if len(allUsers) != 2 {
		t.Errorf("Expected 2 users, got %d", len(allUsers))
	}

	// Check that all users are returned
	foundAlice := false
	foundBob := false
	for _, user := range allUsers {
		if user.Name == "Alice Smith" {
			foundAlice = true
		}
		if user.Name == "Bob Johnson" {
			foundBob = true
		}
	}

	if !foundAlice {
		t.Error("Alice Smith not found in all users")
	}
	if !foundBob {
		t.Error("Bob Johnson not found in all users")
	}
}

func TestUserResolverEdgeCases(t *testing.T) {
	resolver := NewUserResolver()

	// Test with empty user list
	t.Run("EmptyUserList", func(t *testing.T) {
		err := resolver.UpdateUsers([]*domain.User{})
		if err != nil {
			t.Errorf("Unexpected error with empty user list: %v", err)
		}

		_, err = resolver.ResolveUser("any")
		if err == nil {
			t.Error("Expected error when resolving user in empty list")
		}

		allUsers := resolver.GetAllUsers()
		if len(allUsers) != 0 {
			t.Errorf("Expected empty user list, got %d users", len(allUsers))
		}
	})

	// Test with single user
	t.Run("SingleUser", func(t *testing.T) {
		users := []*domain.User{{ID: "single123", Name: "Only User", Email: "only@example.com"}}
		err := resolver.UpdateUsers(users)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should resolve by partial name
		user, err := resolver.ResolveUser("Only")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if user.ID != "single123" {
			t.Errorf("Expected 'single123', got '%s'", user.ID)
		}
	})

	// Test with nil users in list (should be handled gracefully)
	t.Run("UsersWithNil", func(t *testing.T) {
		users := []*domain.User{
			{ID: "valid123", Name: "Valid User", Email: "valid@example.com"},
			nil, // nil user should be skipped or handled gracefully
		}

		// Should not panic
		err := resolver.UpdateUsers(users)
		if err != nil {
			t.Errorf("Unexpected error with nil user in list: %v", err)
		}
	})

	// Test with users having empty names
	t.Run("EmptyUserNames", func(t *testing.T) {
		users := []*domain.User{
			{ID: "user123", Name: "", Email: "empty@example.com"},
			{ID: "user456", Name: "Valid User", Email: "valid@example.com"},
		}

		err := resolver.UpdateUsers(users)
		if err != nil {
			t.Errorf("Unexpected error with empty user name: %v", err)
		}

		// Should still be able to resolve by ID
		user, err := resolver.ResolveUser("user123")
		if err != nil {
			t.Errorf("Unexpected error resolving by ID: %v", err)
		}
		if user.ID != "user123" {
			t.Errorf("Expected 'user123', got '%s'", user.ID)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
