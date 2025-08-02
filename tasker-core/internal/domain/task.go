package domain

import (
	"strings"
	"github.com/google/uuid"
)

// ShortID generates a short unique identifier from a UUID
func ShortID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
}
