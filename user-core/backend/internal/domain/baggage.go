package domain

import (
	"time"
)

// SettingsEntry represents a key/value metadata item stored for a user
type SettingsEntry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Settings is a map of string keys to entries
type Settings map[string]SettingsEntry
