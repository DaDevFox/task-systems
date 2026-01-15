package security

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	// Safe key prefix pattern (alphanumeric with underscore and dash)
	safeKeyPrefixPattern = `^[a-zA-Z0-9\-_]+$`
)

// SanitizeKey sanitizes a string for safe use in BadgerDB keys
// Removes dangerous characters and prevents injection attacks
func SanitizeKey(input string) string {
	switch {
	case input == "":
		return input
	}

	// Convert to bytes for processing
	result := make([]rune, 0, len(input))

	for _, r := range input {
		// Skip control characters
		if unicode.IsControl(r) {
			continue
		}

		// Skip CRLF characters
		if r == '\r' || r == '\n' {
			continue
		}

		// Append safe character
		result = append(result, r)
	}

	// Convert back to string
	sanitized := string(result)

	// Trim whitespace
	sanitized = strings.TrimSpace(sanitized)

	// Limit length
	if len(sanitized) > MaxKeyLength {
		sanitized = sanitized[:MaxKeyLength]
	}

	return sanitized
}

// SanitizeValue sanitizes a string for safe use in BadgerDB values
func SanitizeValue(input string) string {
	switch {
	case input == "":
		return input
	}

	// Remove control characters (but keep newlines for text fields)
	result := make([]rune, 0, len(input))

	for _, r := range input {
		// Skip dangerous control characters but keep newline/tab for formatting
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}

		result = append(result, r)
	}

	sanitized := string(result)

	// Limit length
	if len(sanitized) > MaxValueLength {
		sanitized = sanitized[:MaxValueLength]
	}

	return sanitized
}

// SanitizeID sanitizes an ID (UUID, numeric IDs, etc.)
func SanitizeID(id string) string {
	switch {
	case id == "":
		return id
	}

	// Remove whitespace
	sanitized := strings.TrimSpace(id)

	// Remove control characters
	sanitized = RemoveControlChars(sanitized)

	// Limit length
	if len(sanitized) > MaxIDLength {
		sanitized = sanitized[:MaxIDLength]
	}

	return sanitized
}

// SanitizeEmail sanitizes an email address
func SanitizeEmail(email string) string {
	switch {
	case email == "":
		return email
	}

	// Trim whitespace
	sanitized := strings.TrimSpace(strings.ToLower(email))

	// Remove control characters
	sanitized = RemoveControlChars(sanitized)

	// Limit length
	if len(sanitized) > MaxEmailLength {
		sanitized = sanitized[:MaxEmailLength]
	}

	return sanitized
}

// SanitizeName sanitizes a name field
func SanitizeName(name string) string {
	switch {
	case name == "":
		return name
	}

	// Trim whitespace
	sanitized := strings.TrimSpace(name)

	// Remove control characters
	sanitized = RemoveControlChars(sanitized)

	// Limit length
	if len(sanitized) > MaxNameLength {
		sanitized = sanitized[:MaxNameLength]
	}

	return sanitized
}

// RemoveControlChars removes control characters from a string
func RemoveControlChars(s string) string {
	result := make([]rune, 0, len(s))

	for _, r := range s {
		// Keep printable characters (including unicode printable)
		if !unicode.IsControl(r) {
			result = append(result, r)
		}
	}

	return string(result)
}

// BuildSafeKey constructs a safe BadgerDB key with validation
// Format: prefix:value where both parts are validated and sanitized
func BuildSafeKey(prefix, value string) (string, error) {
	// Validate and sanitize value
	sanitizedValue := SanitizeKey(value)

	// Validate prefix
	if len(prefix) == 0 {
		return "", fmt.Errorf("key prefix cannot be empty")
	}

	// Check for suspicious patterns in prefix
	if ContainsControlChars(prefix) {
		return "", fmt.Errorf("key prefix contains control characters")
	}

	// Construct key
	key := prefix + ":" + sanitizedValue

	// Validate final key
	err := ValidateKey([]byte(key))
	if err != nil {
		return "", fmt.Errorf("failed to build safe key: %w", err)
	}

	return key, nil
}

// BuildSafeUserKey constructs a safe user-specific key
func BuildSafeUserKey(id string) (string, error) {
	sanitizedID := SanitizeID(id)

	switch {
	case sanitizedID == "":
		return "", fmt.Errorf("user ID cannot be empty")
	case len(sanitizedID) > MaxIDLength:
		return "", fmt.Errorf("user ID exceeds maximum length")
	}

	return BuildSafeKey("user", sanitizedID)
}

// BuildSafeEmailKey constructs a safe email index key
func BuildSafeEmailKey(email string) (string, error) {
	sanitizedEmail := SanitizeEmail(email)

	switch {
	case sanitizedEmail == "":
		return "", fmt.Errorf("email cannot be empty")
	case len(sanitizedEmail) > MaxEmailLength:
		return "", fmt.Errorf("email exceeds maximum length")
	}

	return BuildSafeKey("email", sanitizedEmail)
}

// BuildSafeNameKey constructs a safe name index key
func BuildSafeNameKey(name string) (string, error) {
	sanitizedName := SanitizeName(name)

	switch {
	case sanitizedName == "":
		return "", fmt.Errorf("name cannot be empty")
	case len(sanitizedName) > MaxNameLength:
		return "", fmt.Errorf("name exceeds maximum length")
	}

	return BuildSafeKey("name", strings.ToLower(sanitizedName))
}

// BuildSafeTaskKey constructs a safe task-specific key
func BuildSafeTaskKey(id string) (string, error) {
	sanitizedID := SanitizeID(id)

	switch {
	case sanitizedID == "":
		return "", fmt.Errorf("task ID cannot be empty")
	case len(sanitizedID) > MaxIDLength:
		return "", fmt.Errorf("task ID exceeds maximum length")
	}

	return BuildSafeKey("task", sanitizedID)
}

// BuildSafeItemKey constructs a safe inventory item key
func BuildSafeItemKey(id string) (string, error) {
	sanitizedID := SanitizeID(id)

	switch {
	case sanitizedID == "":
		return "", fmt.Errorf("item ID cannot be empty")
	case len(sanitizedID) > MaxIDLength:
		return "", fmt.Errorf("item ID exceeds maximum length")
	}

	return BuildSafeKey("item", sanitizedID)
}

// BuildSafeUnitKey constructs a safe unit definition key
func BuildSafeUnitKey(id string) (string, error) {
	sanitizedID := SanitizeID(id)

	switch {
	case sanitizedID == "":
		return "", fmt.Errorf("unit ID cannot be empty")
	case len(sanitizedID) > MaxIDLength:
		return "", fmt.Errorf("unit ID exceeds maximum length")
	}

	return BuildSafeKey("unit", sanitizedID)
}

// BuildSafeHistoryKey constructs a safe history key with timestamp
func BuildSafeHistoryKey(itemID string, timestamp string) (string, error) {
	sanitizedID := SanitizeID(itemID)
	sanitizedTimestamp := SanitizeKey(timestamp)

	switch {
	case sanitizedID == "":
		return "", fmt.Errorf("item ID cannot be empty")
	case sanitizedTimestamp == "":
		return "", fmt.Errorf("timestamp cannot be empty")
	}

	key := fmt.Sprintf("history:%s:%s", sanitizedID, sanitizedTimestamp)

	// Validate final key
	err := ValidateKey([]byte(key))
	if err != nil {
		return "", fmt.Errorf("failed to build safe history key: %w", err)
	}

	return key, nil
}
