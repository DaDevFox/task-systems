package security

import (
	"context"
	"github.com/dgraph-io/badger/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

// BadgerRefreshTokenStore persists refresh tokens in BadgerDB.
type BadgerRefreshTokenStore struct {
	db     *badger.DB
	logger *logrus.Logger
}

// NewBadgerRefreshTokenStore initializes a Badger-backed token store.
func NewBadgerRefreshTokenStore(badgerDir string, logger *logrus.Logger) (*BadgerRefreshTokenStore, error) {
	if logger == nil {
		logger = logrus.New()
	}
	opts := badger.DefaultOptions(badgerDir).WithLogger(nil) // Suppress Badger's internal logs.
	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open BadgerDB")
	}

	return &BadgerRefreshTokenStore{db: db, logger: logger}, nil
}

// Save persists the provided refresh token metadata.
func (s *BadgerRefreshTokenStore) Save(ctx context.Context, token string, metadata RefreshTokenMetadata) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		data, marshalErr := json.Marshal(metadata)
		if marshalErr != nil {
			return errors.Wrap(marshalErr, "marshal metadata")
		}
		e := badger.NewEntry([]byte(token), data).WithTTL(time.Until(metadata.ExpiresAt))
		return txn.SetEntry(e)
	})
	return err
}

// Get retrieves a refresh token's metadata when present and not expired.
func (s *BadgerRefreshTokenStore) Get(ctx context.Context, token string) (RefreshTokenMetadata, error) {
	var metadata RefreshTokenMetadata
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(token))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrRefreshTokenNotFound
			}
			return errors.Wrap(err, "get token")
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &metadata)
		})
	})

	if err != nil {
		return RefreshTokenMetadata{}, err
	}

	if time.Now().After(metadata.ExpiresAt) {
		_ = s.Delete(ctx, token)
		return RefreshTokenMetadata{}, ErrRefreshTokenExpired
	}

	return metadata, nil
}

// Delete removes a refresh token from the store.
func (s *BadgerRefreshTokenStore) Delete(ctx context.Context, token string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(token))
	})
}

// Close releases all resources held by the store.
func (s *BadgerRefreshTokenStore) Close() error {
	return s.db.Close()
}
