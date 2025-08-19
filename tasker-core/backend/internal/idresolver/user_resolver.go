package idresolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
)

// UserResolver provides user ID and name resolution functionality
type UserResolver struct {
	users    []*domain.User
	nameMap  map[string]*domain.User // name -> user
	idMap    map[string]*domain.User // id -> user
	emailMap map[string]*domain.User // email -> user
}

// NewUserResolver creates a new user resolver
func NewUserResolver() *UserResolver {
	return &UserResolver{
		users:    []*domain.User{},
		nameMap:  make(map[string]*domain.User),
		idMap:    make(map[string]*domain.User),
		emailMap: make(map[string]*domain.User),
	}
}

// UpdateUsers updates the internal maps with the current set of users
func (r *UserResolver) UpdateUsers(users []*domain.User) error {
	// Reset maps
	r.users = users
	r.nameMap = make(map[string]*domain.User)
	r.idMap = make(map[string]*domain.User)
	r.emailMap = make(map[string]*domain.User)

	// Check for duplicate names
	nameCount := make(map[string]int)
	originalNames := make(map[string]string) // lowercase -> original case
	emailCount := make(map[string]int)
	for _, user := range users {
		if user == nil {
			continue // Skip nil users
		}
		lowerName := strings.ToLower(user.Name)
		nameCount[lowerName]++
		if _, exists := originalNames[lowerName]; !exists {
			originalNames[lowerName] = user.Name
		}

		// Check for duplicate emails
		if user.Email != "" {
			lowerEmail := strings.ToLower(user.Email)
			emailCount[lowerEmail]++
		}
	}

	for name, count := range nameCount {
		if count > 1 {
			originalName := originalNames[name]
			return fmt.Errorf("duplicate user name found: '%s' (names must be unique)", originalName)
		}
	}

	for email, count := range emailCount {
		if count > 1 {
			return fmt.Errorf("duplicate user email found: '%s' (emails must be unique)", email)
		}
	}

	// Build maps
	for _, user := range users {
		if user == nil {
			continue // Skip nil users
		}
		r.nameMap[strings.ToLower(user.Name)] = user
		r.idMap[user.ID] = user
		if user.Email != "" {
			r.emailMap[strings.ToLower(user.Email)] = user
		}
	}

	return nil
}

// ResolveUser resolves a user by ID, name, or email
func (r *UserResolver) ResolveUser(identifier string, resolveName, resolveEmail bool) (*domain.User, error) {
	if identifier == "" {
		return nil, fmt.Errorf("empty user identifier provided")
	}

	// Try by ID first
	if user, exists := r.idMap[identifier]; exists {
		return user, nil
	}

	if resolveEmail {
		// Try by email (case-insensitive)
		if user, exists := r.emailMap[strings.ToLower(identifier)]; exists {
			return user, nil
		}
	}

	if resolveName {
		// Try by name (case-insensitive)
		if user, exists := r.nameMap[strings.ToLower(identifier)]; exists {
			return user, nil
		}

		// Try partial name match
		matches := r.findPartialNameMatches(identifier)
		if len(matches) == 1 {
			return matches[0], nil
		}

		if len(matches) > 1 {
			names := make([]string, len(matches))
			for i, user := range matches {
				names[i] = user.Name
			}
			sort.Strings(names)
			return nil, fmt.Errorf("ambiguous user identifier '%s', matches: %s", identifier, strings.Join(names, ", "))
		}
	}

	// No matches found
	return nil, fmt.Errorf("user not found: '%s'", identifier)
}

// ResolveUserID resolves a user identifier to a user ID
// This function performs more work than ResolveUser -- if you need the full object use that directly
func (r *UserResolver) ResolveUserID(identifier string) (string, error) {
	user, err := r.ResolveUser(identifier, false, false)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

// ResolveUserIDByName resolves a user name to a user ID
// This function performs more work than ResolveUser -- if you need the full object use that directly
func (r *UserResolver) ResolveUserIDByName(identifier string) (string, error) {
	user, err := r.ResolveUser(identifier, true, false)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

// ResolveUserIDByEmail resolves a user email to a user ID
// This function performs more work than ResolveUser -- if you need the full object use that directly
func (r *UserResolver) ResolveUserIDByEmail(identifier string) (string, error) {
	user, err := r.ResolveUser(identifier, false, true)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

// findPartialNameMatches finds users whose names start with the given prefix
func (r *UserResolver) findPartialNameMatches(prefix string) []*domain.User {
	var matches []*domain.User
	lowerPrefix := strings.ToLower(prefix)

	for _, user := range r.users {
		if strings.HasPrefix(strings.ToLower(user.Name), lowerPrefix) {
			matches = append(matches, user)
		}
	}

	return matches
}

// GetAllUsers returns all users
func (r *UserResolver) GetAllUsers() []*domain.User {
	return r.users
}

// ValidateUserExists checks if a user exists by ID
func (r *UserResolver) ValidateUserExists(userID string) error {
	if _, exists := r.idMap[userID]; !exists {
		return fmt.Errorf("user with ID '%s' does not exist", userID)
	}
	return nil
}

// ValidateUserNameUnique checks if a user name is unique
func (r *UserResolver) ValidateUserNameUnique(name string, excludeUserID string) error {
	lowerName := strings.ToLower(name)
	if user, exists := r.nameMap[lowerName]; exists && user.ID != excludeUserID {
		return fmt.Errorf("user name '%s' is already taken", name)
	}
	return nil
}

// SuggestUsers suggests similar users for a given identifier
func (r *UserResolver) SuggestUsers(identifier string, maxSuggestions int) []string {
	suggestions := []string{}
	lowerIdentifier := strings.ToLower(identifier)

	// Find users by name prefix
	for _, user := range r.users {
		if strings.HasPrefix(strings.ToLower(user.Name), lowerIdentifier) {
			suggestions = append(suggestions, user.Name)
		}
	}

	// Find users by email prefix
	for _, user := range r.users {
		if user.Email != "" && strings.HasPrefix(strings.ToLower(user.Email), lowerIdentifier) {
			suggestions = append(suggestions, user.Email)
		}
	}

	// Find users by ID prefix
	for _, user := range r.users {
		if strings.HasPrefix(strings.ToLower(user.ID), lowerIdentifier) {
			suggestions = append(suggestions, user.ID)
		}
	}

	// Remove duplicates and sort
	suggestionMap := make(map[string]bool)
	uniqueSuggestions := []string{}
	for _, s := range suggestions {
		if !suggestionMap[s] {
			suggestionMap[s] = true
			uniqueSuggestions = append(uniqueSuggestions, s)
		}
	}

	sort.Strings(uniqueSuggestions)

	// Limit suggestions
	if len(uniqueSuggestions) > maxSuggestions {
		uniqueSuggestions = uniqueSuggestions[:maxSuggestions]
	}

	return uniqueSuggestions
}
