package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NotificationType defines the types of notifications a user can receive
type NotificationType int

const (
	NotificationTypeUnspecified NotificationType = iota
	NotificationOnAssign
	NotificationOnStart
	NotificationOnComplete
	NotificationOnDue
	NotificationNDaysBeforeDue
)

func (nt NotificationType) String() string {
	switch nt {
	case NotificationOnAssign:
		return "on_assign"
	case NotificationOnStart:
		return "on_start"
	case NotificationOnComplete:
		return "on_complete"
	case NotificationOnDue:
		return "on_due"
	case NotificationNDaysBeforeDue:
		return "n_days_before_due"
	default:
		return "unspecified"
	}
}

// NotificationMethod defines how notifications are delivered
type NotificationMethod int

const (
	NotificationMethodUnspecified NotificationMethod = iota
	NotificationMethodEmail
	NotificationMethodInApp
	NotificationMethodSMS
	NotificationMethodPush
)

func (nm NotificationMethod) String() string {
	switch nm {
	case NotificationMethodEmail:
		return "email"
	case NotificationMethodInApp:
		return "in_app"
	case NotificationMethodSMS:
		return "sms"
	case NotificationMethodPush:
		return "push"
	default:
		return "unspecified"
	}
}

// UserRole defines the role/permission level of a user
type UserRole int

const (
	UserRoleUnspecified UserRole = iota
	UserRoleGuest
	UserRoleUser
	UserRoleAdmin
)

func (ur UserRole) String() string {
	switch ur {
	case UserRoleGuest:
		return "guest"
	case UserRoleUser:
		return "user"
	case UserRoleAdmin:
		return "admin"
	default:
		return "unspecified"
	}
}

// UserStatus defines the current status of a user account
type UserStatus int

const (
	UserStatusUnspecified UserStatus = iota
	UserStatusActive
	UserStatusInactive
	UserStatusSuspended
)

func (us UserStatus) String() string {
	switch us {
	case UserStatusActive:
		return "active"
	case UserStatusInactive:
		return "inactive"
	case UserStatusSuspended:
		return "suspended"
	default:
		return "unspecified"
	}
}

// NotificationSetting represents user preferences for a specific notification type
type NotificationSetting struct {
	Type       NotificationType
	Enabled    bool
	Methods    []NotificationMethod
	DaysBefore int32 // For NotificationNDaysBeforeDue type
}

// UserConfiguration stores user-specific settings and preferences
type UserConfiguration struct {
	// Notification preferences
	NotificationSettings []NotificationSetting

	// Task management preferences
	DefaultTimezone string // User's timezone (e.g., "America/New_York")
	DateFormat      string // Preferred date format
	TimeFormat      string // Preferred time format (12h/24h)

	// Integration settings
	GoogleCalendarToken string // Google Calendar OAuth token
	EmailAddress        string // Email for notifications (may differ from login email)
	PhoneNumber         string // Phone for SMS notifications

	// Privacy settings
	AllowPublicProfile bool // Whether profile is visible to other users
	AllowTaskSharing   bool // Whether user can share tasks with others
}

// User represents a user account in the system
type User struct {
	ID        string            // Unique user identifier
	Email     string            // Primary email address (unique)
	Name      string            // Display name
	FirstName string            // First name
	LastName  string            // Last name
	Role      UserRole          // User's role/permission level
	Status    UserStatus        // Account status
	Config    UserConfiguration // User preferences and settings
	CreatedAt time.Time         // When account was created
	UpdatedAt time.Time         // When account was last modified
	LastLogin *time.Time        // When user last logged in (future)
}

// NewUser creates a new user with default settings
func NewUser(email, name string) *User {
	now := time.Now()
	return &User{
		ID:        GenerateUserID(),
		Email:     email,
		Name:      name,
		Role:      UserRoleUser,
		Status:    UserStatusActive,
		Config:    DefaultUserConfiguration(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// DefaultUserConfiguration returns default user configuration
func DefaultUserConfiguration() UserConfiguration {
	return UserConfiguration{
		NotificationSettings: []NotificationSetting{
			{Type: NotificationOnAssign, Enabled: true, Methods: []NotificationMethod{NotificationMethodEmail, NotificationMethodInApp}},
			{Type: NotificationOnStart, Enabled: false, Methods: []NotificationMethod{NotificationMethodInApp}},
			{Type: NotificationOnComplete, Enabled: true, Methods: []NotificationMethod{NotificationMethodInApp}},
			{Type: NotificationOnDue, Enabled: true, Methods: []NotificationMethod{NotificationMethodEmail}},
		},
		DefaultTimezone:    "UTC",
		DateFormat:         "2006-01-02",
		TimeFormat:         "24h",
		AllowPublicProfile: true,
		AllowTaskSharing:   true,
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

	// Basic email format validation
	if !strings.Contains(u.Email, "@") {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// FullName returns the user's full name if first/last names are available
func (u *User) FullName() string {
	if u.FirstName != "" && u.LastName != "" {
		return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
	}
	return u.Name
}

// IsActive returns whether the user account is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// HasRole returns whether the user has at least the specified role level
func (u *User) HasRole(role UserRole) bool {
	return u.Role >= role
}

// GetNotificationSetting returns the notification setting for a specific type
func (u *User) GetNotificationSetting(notificationType NotificationType) *NotificationSetting {
	for i := range u.Config.NotificationSettings {
		if u.Config.NotificationSettings[i].Type == notificationType {
			return &u.Config.NotificationSettings[i]
		}
	}
	return nil
}

// UpdateNotificationSetting updates or adds a notification setting
func (u *User) UpdateNotificationSetting(setting NotificationSetting) {
	for i := range u.Config.NotificationSettings {
		if u.Config.NotificationSettings[i].Type == setting.Type {
			u.Config.NotificationSettings[i] = setting
			return
		}
	}
	// Add new setting if not found
	u.Config.NotificationSettings = append(u.Config.NotificationSettings, setting)
}
