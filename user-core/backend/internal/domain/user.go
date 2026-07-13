package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// User represents a user account in the system
type User struct {
	ID            string     // Unique user identifier
	Email         string     // Primary email address (unique)
	PhoneNumber   string     // Phone number
	Name          string     // Display name
	FirstName     string     // First name
	MiddleName    string     // Middle name
	LastName      string     // Last name
	CreatedAt     time.Time  // When account was created
	LastUpdatedAt time.Time  // When account was last modified
	LastLogin     *time.Time // When user last logged in (future)
}

// NewUser creates a new user with default settings
func NewUser(userId, email, name string) *User {
	if strings.Trim(userId, " ") == "" {
		userId = GenerateUserID()
	}

	now := time.Now()
	return &User{
		ID:            GenerateUserID(),
		Email:         email,
		Name:          name,
		CreatedAt:     now,
		LastUpdatedAt: now,
	}
}

// GenerateUserID generates a unique 8-character user ID
func GenerateUserID() string {
	id := uuid.New().String()
	// Take first 8 characters and remove dashes
	cleanID := strings.ReplaceAll(id, "-", "")
	if len(cleanID) >= 8 {
		return cleanID[:8]
	}
	return cleanID
}

// ValidateUser performs basic validation on user data
func (u *User) Validate() error {
	if u.Email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if u.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if u.ID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// TODO: regex match email
	// Basic email format validation
	if !strings.Contains(u.Email, "@") {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// FullName returns the user's full name if first/last names are available
func (u *User) FullName() string {
	if u.FirstName != "" && u.LastName != "" && u.MiddleName != "" {
		return fmt.Sprintf("%s %s %s", u.FirstName, u.MiddleName, u.LastName)
	}
	if u.FirstName != "" && u.LastName != "" {
		return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
	}
	return u.Name
}
