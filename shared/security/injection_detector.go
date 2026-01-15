package security

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// Injection attempt detection patterns
	sqlCommentPattern     = `(--|/\*|\*/|;)`
	nosqlOperatorPattern  = `(\$ne|\$gt|\$lt|\$gte|\$lte|\$regex|\$where|\$and|\$or)`
	pathTraversalPattern  = `(\.\.\/|\.\.\\|\/\.\.\/|\\\.\\.\\)`
	crlfPattern           = `(\r\n|\n|\r)`
	unicodeControlPattern = `[\x00-\x1F\x7F]`

	// Suspicious patterns in keys
	directoryTraversalPattern = `(^\.\.|^\.\.\/|^\.\.\\)`
	prefixBypassPattern       = `^.*:.*\.\./.*:.*$`
)

// InjectionAttempt represents a detected injection attempt
type InjectionAttempt struct {
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`       // service, user_id, ip, etc.
	Operation   string    `json:"operation"`    // get, set, delete, scan
	AttemptType string    `json:"attempt_type"` // path_traversal, sql_injection, control_chars, etc.
	Input       string    `json:"input"`        // sanitized input value
	Key         string    `json:"key"`          // the key being constructed
}

// Detector detects and logs injection attempts
type Detector struct {
	mu          sync.RWMutex
	logger      *logrus.Logger
	attempts    []InjectionAttempt
	maxAttempts int // max attempts to keep in memory
}

// NewDetector creates a new injection detector
func NewDetector(logger *logrus.Logger, maxAttempts int) *Detector {
	switch {
	case logger == nil:
		logger = logrus.New()
	case maxAttempts <= 0:
		maxAttempts = 100 // default
	}

	return &Detector{
		logger:      logger,
		attempts:    make([]InjectionAttempt, 0, maxAttempts),
		maxAttempts: maxAttempts,
	}
}

// CheckInput checks input for injection patterns
func (d *Detector) CheckInput(ctx context.Context, source, operation, key, input string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check for various injection patterns
	switch {
	case input == "":
		return nil // Empty input is not an injection attempt
	case d.containsSQLComment(input):
		return d.logAttempt(ctx, source, operation, key, "sql_injection", input)
	case d.containsNoSQLOperators(input):
		return d.logAttempt(ctx, source, operation, key, "nosql_operator_injection", input)
	case d.containsPathTraversal(input):
		return d.logAttempt(ctx, source, operation, key, "path_traversal", input)
	case d.containsCRLF(input):
		return d.logAttempt(ctx, source, operation, key, "crlf_injection", input)
	case d.containsControlChars(input):
		return d.logAttempt(ctx, source, operation, key, "control_characters", input)
	case d.containsPrefixBypass(input):
		return d.logAttempt(ctx, source, operation, key, "prefix_bypass", input)
	}

	return nil
}

// CheckKey checks a constructed BadgerDB key for injection patterns
func (d *Detector) CheckKey(ctx context.Context, source, operation, key []byte) error {
	keyStr := string(key)

	// Check key for suspicious patterns
	switch {
	case len(keyStr) == 0:
		return nil
	case d.containsSQLComment(keyStr):
		return d.logAttempt(ctx, source, operation, keyStr, "sql_injection", keyStr)
	case d.containsPathTraversal(keyStr):
		return d.logAttempt(ctx, source, operation, keyStr, "path_traversal", keyStr)
	case d.containsCRLF(keyStr):
		return d.logAttempt(ctx, source, operation, keyStr, "crlf_injection", keyStr)
	case d.containsControlChars(keyStr):
		return d.logAttempt(ctx, source, operation, keyStr, "control_characters", keyStr)
	case d.containsPrefixBypass(keyStr):
		return d.logAttempt(ctx, source, operation, keyStr, "prefix_bypass", keyStr)
	}

	return nil
}

// logAttempt records an injection attempt
func (d *Detector) logAttempt(ctx context.Context, source, operation, key, attemptType, input string) error {
	attempt := InjectionAttempt{
		Timestamp:   time.Now(),
		Source:      source,
		Operation:   operation,
		AttemptType: attemptType,
		Input:       input,
		Key:         key,
	}

	// Add to attempts (circular buffer)
	if len(d.attempts) >= d.maxAttempts {
		// Remove oldest attempt
		d.attempts = d.attempts[1:]
	}
	d.attempts = append(d.attempts, attempt)

	// Log the attempt with structured logging
	d.logger.WithFields(logrus.Fields{
		"event":        "injection_attempt",
		"source":       source,
		"operation":    operation,
		"attempt_type": attemptType,
		"input":        input,
		"key":          key,
		"timestamp":    attempt.Timestamp,
	}).Warn("NoSQL injection attempt detected")

	return fmt.Errorf("potential injection attempt detected: %s", attemptType)
}

// GetRecentAttempts returns recent injection attempts
func (d *Detector) GetRecentAttempts(count int) []InjectionAttempt {
	d.mu.RLock()
	defer d.mu.RUnlock()

	switch {
	case count <= 0:
		return []InjectionAttempt{}
	case count > len(d.attempts):
		return d.attempts
	}

	return d.attempts[len(d.attempts)-count:]
}

// containsSQLComment checks for SQL comment patterns
func (d *Detector) containsSQLComment(input string) bool {
	return regexp.MustCompile(sqlCommentPattern).MatchString(input)
}

// containsNoSQLOperators checks for NoSQL operator patterns
func (d *Detector) containsNoSQLOperators(input string) bool {
	return regexp.MustCompile(nosqlOperatorPattern).MatchString(input)
}

// containsPathTraversal checks for path traversal patterns
func (d *Detector) containsPathTraversal(input string) bool {
	return regexp.MustCompile(pathTraversalPattern).MatchString(input)
}

// containsCRLF checks for CRLF injection patterns
func (d *Detector) containsCRLF(input string) bool {
	return regexp.MustCompile(crlfPattern).MatchString(input)
}

// containsControlChars checks for control characters
func (d *Detector) containsControlChars(input string) bool {
	return regexp.MustCompile(unicodeControlPattern).MatchString(input)
}

// containsPrefixBypass checks for prefix bypass patterns
func (d *Detector) containsPrefixBypass(input string) bool {
	return regexp.MustCompile(prefixBypassPattern).MatchString(input)
}

// IsSuspiciousKey checks if a key contains suspicious patterns
func IsSuspiciousKey(key string) bool {
	suspiciousPatterns := []string{
		"..",
		"\x00",
		"\r\n",
		"\n",
		"\r",
		"../",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}

	return false
}

// SanitizeLogInput sanitizes input for logging (removes control chars)
func SanitizeLogInput(input string) string {
	// Remove control characters
	result := regexp.MustCompile(unicodeControlPattern).ReplaceAllString(input, "")
	// Limit length for logging
	if len(result) > 200 {
		result = result[:200] + "..."
	}
	return result
}
