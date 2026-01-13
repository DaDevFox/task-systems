package security

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BadgerRefreshTokenStore struct {
	db     *badger.DB
	logger *logrus.Logger
}

func NewBadgerRefreshTokenStore(db *badger.DB, logger *logrus.Logger) *BadgerRefreshTokenStore {
	if logger == nil {
		logger = logrus.New()
	}
	return &BadgerRefreshTokenStore{db: db, logger: logger}
}

func (s *BadgerRefreshTokenStore) Save(ctx context.Context, token string, metadata RefreshTokenMetadata) error {
	if token == "" {
		return errors.New(msgRefreshTokenEmpty)
	}
	if metadata.UserID == "" {
		return errors.New(msgRefreshTokenMissingUser)
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return errors.Wrap(err, "failed to serialize refresh token metadata")
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("refresh:%s", token)), metadataBytes)
	})
	if err != nil {
		s.logger.WithError(err).WithField("token", token).Error("failed to save refresh token")
		return errors.Wrap(err, "failed to save refresh token")
	}

	s.logger.WithField("token", token).Debug("refresh token saved successfully")
	return nil
}

func (s *BadgerRefreshTokenStore) Get(ctx context.Context, token string) (RefreshTokenMetadata, error) {
	if token == "" {
		return RefreshTokenMetadata{}, errors.New(msgRefreshTokenEmpty)
	}

	var metadata RefreshTokenMetadata
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(fmt.Sprintf("refresh:%s", token)))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrRefreshTokenNotFound
			}
			return errors.Wrap(err, "failed to retrieve refresh token metadata")
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &metadata)
		})
	})
	if err != nil {
		return RefreshTokenMetadata{}, err
	}

	if time.Now().After(metadata.ExpiresAt) {
		s.Delete(ctx, token)
		return RefreshTokenMetadata{}, ErrRefreshTokenExpired
	}

	return metadata, nil
}

func (s *BadgerRefreshTokenStore) Delete(ctx context.Context, token string) error {
	if token == "" {
		return errors.New(msgRefreshTokenEmpty)
	}

	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(fmt.Sprintf("refresh:%s", token)))
	})
	if err != nil {
		s.logger.WithError(err).WithField("token", token).Error("failed to delete refresh token")
		return errors.Wrap(err, "failed to delete refresh token")
	}

	s.logger.WithField("token", token).Debug("refresh token deleted successfully")
	return nil
}
