package security

import (
	"fmt"
	"regexp"
	"unicode"
)

const (
	// MaxKeyLength defines maximum allowed key length for BadgerDB
	MaxKeyLength = 1024
	// MaxValueLength defines maximum allowed value length
	MaxValueLength = 10 * 1024 * 1024 // 10MB
	// MaxIDLength defines maximum ID length
	MaxIDLength = 128
	// MaxEmailLength defines maximum email length
	MaxEmailLength = 254 // RFC 5321 limit
	// MaxNameLength defines maximum name length
	MaxNameLength = 256
	// MaxDescriptionLength defines maximum description length
	MaxDescriptionLength = 4096
)

// Regex patterns for validation
var (
	// alphanumericPattern allows only alphanumeric characters
	alphanumericPattern = regexp.MustCompile(`^[a-zA-Z0-9\-_.]+$`)
	// safeIDPattern allows UUID-style IDs and safe alphanumeric
	safeIDPattern = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	// emailPattern validates email format (basic)
	emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	// pathTraversalPattern detects path traversal attempts
	pathTraversalPattern = regexp.MustCompile(`(\.\.|\.\.|/\\)`)
	// controlCharPattern detects control characters
	controlCharPattern = regexp.MustCompile(`[\x00-\x1F\x7F]`)
	// injectionPattern detects common injection patterns
	injectionPattern = regexp.MustCompile(`(;|--|/\*|\*/|'|"|\\|\$\{|\$\(|\x00)`)
)

// ValidationErrors collects multiple validation errors
type ValidationErrors []error

// Add adds an error to the collection
func (ve *ValidationErrors) Add(err error) {
	*ve = append(*ve, err)
}

// Error returns the combined error message
func (ve ValidationErrors) Error() string {
	switch len(ve) {
	case 0:
		return "no validation errors"
	case 1:
		return ve[0].Error()
	default:
		return fmt.Sprintf("multiple validation errors (%d): %v", len(ve), ve)
	}
}

// ValidateID validates an identifier (user ID, task ID, etc.)
func ValidateID(id string) error {
	switch {
	case id == "":
		return fmt.Errorf("id cannot be empty")
	case len(id) > MaxIDLength:
		return fmt.Errorf("id length %d exceeds maximum %d", len(id), MaxIDLength)
	case !safeIDPattern.MatchString(id):
		return fmt.Errorf("id contains invalid characters: %s", id)
	case pathTraversalPattern.MatchString(id):
		return fmt.Errorf("id contains path traversal pattern: %s", id)
	case controlCharPattern.MatchString(id):
		return fmt.Errorf("id contains control characters")
	case injectionPattern.MatchString(id):
		return fmt.Errorf("id contains potential injection pattern: %s", id)
	}
	return nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	switch {
	case email == "":
		return fmt.Errorf("email cannot be empty")
	case len(email) > MaxEmailLength:
		return fmt.Errorf("email length %d exceeds maximum %d", len(email), MaxEmailLength)
	case !emailPattern.MatchString(email):
		return fmt.Errorf("invalid email format: %s", email)
	case injectionPattern.MatchString(email):
		return fmt.Errorf("email contains potential injection pattern: %s", email)
	}
	return nil
}

// ValidateName validates a name string
func ValidateName(name string, fieldName string) error {
	switch {
	case name == "":
		return fmt.Errorf("%s cannot be empty", fieldName)
	case len(name) > MaxNameLength:
		return fmt.Errorf("%s length %d exceeds maximum %d", fieldName, len(name), MaxNameLength)
	case controlCharPattern.MatchString(name):
		return fmt.Errorf("%s contains control characters", fieldName)
	case injectionPattern.MatchString(name):
		return fmt.Errorf("%s contains potential injection pattern: %s", fieldName, name)
	}
	return nil
}

// ValidateDescription validates a description string
func ValidateDescription(description string, fieldName string) error {
	switch {
	case len(description) > MaxDescriptionLength:
		return fmt.Errorf("%s length %d exceeds maximum %d", fieldName, len(description), MaxDescriptionLength)
	case controlCharPattern.MatchString(description):
		return fmt.Errorf("%s contains control characters", fieldName)
	}
	return nil
}

// ValidateKey validates a BadgerDB key
func ValidateKey(key []byte) error {
	keyStr := string(key)
	switch {
	case len(key) == 0:
		return fmt.Errorf("key cannot be empty")
	case len(key) > MaxKeyLength:
		return fmt.Errorf("key length %d exceeds maximum %d", len(key), MaxKeyLength)
	case controlCharPattern.MatchString(keyStr):
		return fmt.Errorf("key contains control characters")
	case pathTraversalPattern.MatchString(keyStr):
		return fmt.Errorf("key contains path traversal pattern")
	}
	return nil
}

// ValidateValue validates a BadgerDB value
func ValidateValue(value []byte) error {
	switch {
	case len(value) > MaxValueLength:
		return fmt.Errorf("value length %d exceeds maximum %d", len(value), MaxValueLength)
	}
	return nil
}

// ValidateStringLength validates a string field with length constraint
func ValidateStringLength(value string, fieldName string, minLength, maxLength int) error {
	switch {
	case len(value) < minLength:
		return fmt.Errorf("%s length %d is less than minimum %d", fieldName, len(value), minLength)
	case len(value) > maxLength:
		return fmt.Errorf("%s length %d exceeds maximum %d", fieldName, len(value), maxLength)
	}
	return nil
}

// SanitizeForKey sanitizes a string for use in BadgerDB keys
func SanitizeForKey(input string) string {
	// Remove control characters
	result := controlCharPattern.ReplaceAllString(input, "")
	// Remove path traversal patterns
	result = pathTraversalPattern.ReplaceAllString(result, "")
	// Trim whitespace
	result = stripWhitespace(result)
	return result
}

// SanitizeForValue sanitizes a string for use in BadgerDB values
func SanitizeForValue(input string) string {
	// Remove control characters
	result := controlCharPattern.ReplaceAllString(input, "")
	return result
}

// ContainsControlChars checks if string contains control characters
func ContainsControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// stripWhitespace removes leading/trailing and internal control whitespace
func stripWhitespace(s string) string {
	// Simple whitespace trimming
	start := 0
	end := len(s)

	for start < end && (s[start] <= ' ' || s[start] == 0x7F) {
		start++
	}

	for end > start && (s[end-1] <= ' ' || s[end-1] == 0x7F) {
		end--
	}

	return s[start:end]
}
