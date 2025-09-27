package auth

import "strings"

// Role represents a normalized user role string extracted from access token claims.
type Role string

const (
	RoleUnknown Role = ""
	RoleGuest   Role = "guest"
	RoleUser    Role = "user"
	RoleAdmin   Role = "admin"
)

// NormalizeRole converts arbitrary role strings to the closest supported Role constant.
func NormalizeRole(role string) Role {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case string(RoleAdmin):
		return RoleAdmin
	case string(RoleUser):
		return RoleUser
	case string(RoleGuest):
		return RoleGuest
	default:
		return RoleUnknown
	}
}

// Claims captures the minimum information propagated through request contexts after authentication.
type Claims struct {
	UserID string
	Email  string
	Role   Role
}

// HasRole checks whether the claims role matches one of the expected roles.
func (c *Claims) HasRole(expected Role) bool {
	if c == nil {
		return false
	}

	return c.Role == expected
}

// HasAnyRole reports true if the claims role matches any of the provided roles.
func (c *Claims) HasAnyRole(expected ...Role) bool {
	if c == nil {
		return false
	}

	role := c.Role
	for _, r := range expected {
		if role == r {
			return true
		}
	}

	return false
}
