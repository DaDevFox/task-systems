package domain

import (
	"strings"
)

// ShortID generates a short unique identifier from a UUID
func ShortID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
}
