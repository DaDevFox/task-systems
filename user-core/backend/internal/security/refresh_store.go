package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// RefreshTokenMetadata captures the owner and expiry of a refresh token.
type RefreshTokenMetadata struct {
	UserID    string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// RefreshTokenStore defines operations for persisting refresh tokens.
type RefreshTokenStore interface {
	Save(ctx context.Context, token string, metadata RefreshTokenMetadata) error
	Get(ctx context.Context, token string) (RefreshTokenMetadata, error)
	Delete(ctx context.Context, token string) error
}

// Errors returned by the refresh token store.
var (
	ErrRefreshTokenNotFound = fmt.Errorf("refresh token not found")
	ErrRefreshTokenExpired  = fmt.Errorf("refresh token expired")
)

const (
	msgRefreshTokenEmpty       = "refresh token cannot be empty"
	msgRefreshTokenMissingUser = "refresh token metadata missing user id"
)

// InMemoryRefreshTokenStore is a development-friendly implementation using a concurrent map.
type InMemoryRefreshTokenStore struct {
	tokens map[string]RefreshTokenMetadata
	mutex  sync.RWMutex
	logger *logrus.Logger
}

// NewInMemoryRefreshTokenStore builds an in-memory store for refresh tokens.
func NewInMemoryRefreshTokenStore(logger *logrus.Logger) *InMemoryRefreshTokenStore {
	if logger == nil {
		logger = logrus.New()
	}

	return &InMemoryRefreshTokenStore{
		tokens: make(map[string]RefreshTokenMetadata),
		logger: logger,
	}
}

// Save persists the provided refresh token metadata.
func (s *InMemoryRefreshTokenStore) Save(ctx context.Context, token string, metadata RefreshTokenMetadata) error {
	if token == "" {
		s.logger.Error(msgRefreshTokenEmpty)
		return fmt.Errorf(msgRefreshTokenEmpty)
	}

	if metadata.UserID == "" {
		s.logger.Error(msgRefreshTokenMissingUser)
		return fmt.Errorf(msgRefreshTokenMissingUser)
	}

	s.mutex.Lock()
	s.tokens[token] = metadata
	s.mutex.Unlock()
	return nil
}

// Get retrieves a refresh token's metadata when present and not expired.
func (s *InMemoryRefreshTokenStore) Get(ctx context.Context, token string) (RefreshTokenMetadata, error) {
	if token == "" {
		s.logger.Error(msgRefreshTokenEmpty)
		return RefreshTokenMetadata{}, fmt.Errorf(msgRefreshTokenEmpty)
	}

	s.mutex.RLock()
	metadata, exists := s.tokens[token]
	s.mutex.RUnlock()

	if !exists {
		s.logger.WithField("token", token).Debug("refresh token not found")
		return RefreshTokenMetadata{}, ErrRefreshTokenNotFound
	}

	if time.Now().After(metadata.ExpiresAt) {
		s.mutex.Lock()
		delete(s.tokens, token)
		s.mutex.Unlock()
		s.logger.WithField("token", token).Info("refresh token expired")
		return RefreshTokenMetadata{}, ErrRefreshTokenExpired
	}

	return metadata, nil
}

// Delete removes a refresh token from the store.
func (s *InMemoryRefreshTokenStore) Delete(ctx context.Context, token string) error {
	if token == "" {
		s.logger.Error(msgRefreshTokenEmpty)
		return fmt.Errorf(msgRefreshTokenEmpty)
	}

	s.mutex.Lock()
	delete(s.tokens, token)
	s.mutex.Unlock()
	return nil
}
