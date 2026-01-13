package domain

import (
	"time"
)

// BaggageEntry represents a key/value metadata item stored for a user
type BaggageEntry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Baggage is a map of string keys to entries
type Baggage map[string]BaggageEntry
