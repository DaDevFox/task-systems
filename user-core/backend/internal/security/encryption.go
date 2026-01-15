package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
)

const (
	// TOTPSecretKeyLength is the required key length for AES-256
	TOTPSecretKeyLength = 32
	// GCMNonceSize is the nonce size for AES-GCM
	GCMNonceSize = 12
)

// EncryptionResult contains the encrypted data and nonce
type EncryptionResult struct {
	Ciphertext string
	Nonce      string
}

// EncryptTOTPSecret encrypts a TOTP secret using AES-256-GCM
// Returns base64-encoded ciphertext and nonce
func EncryptTOTPSecret(secret string, key []byte, logger *logrus.Logger) (*EncryptionResult, error) {
	if secret == "" {
		logger.Error("totp secret cannot be empty")
		return nil, fmt.Errorf("totp secret cannot be empty")
	}

	if len(key) != TOTPSecretKeyLength {
		logger.WithField("key_length", len(key)).Error("encryption key must be 32 bytes for AES-256")
		return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	switch {
	case err != nil:
		logger.WithError(err).Error("failed to create cipher block")
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	switch {
	case err != nil:
		logger.WithError(err).Error("failed to create GCM mode")
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}

	nonce := make([]byte, GCMNonceSize)
	_, err = io.ReadFull(rand.Reader, nonce)
	switch {
	case err != nil:
		logger.WithError(err).Error("failed to generate nonce")
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(secret), nil)

	result := &EncryptionResult{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
	}

	logger.WithField("secret_length", len(secret)).Debug("totp secret encrypted successfully")
	return result, nil
}

// DecryptTOTPSecret decrypts a TOTP secret using AES-256-GCM
func DecryptTOTPSecret(ciphertextB64 string, nonceB64 string, key []byte, logger *logrus.Logger) (string, error) {
	if ciphertextB64 == "" {
		logger.Error("ciphertext cannot be empty")
		return "", fmt.Errorf("ciphertext cannot be empty")
	}

	if nonceB64 == "" {
		logger.Error("nonce cannot be empty")
		return "", fmt.Errorf("nonce cannot be empty")
	}

	if len(key) != TOTPSecretKeyLength {
		logger.WithField("key_length", len(key)).Error("encryption key must be 32 bytes for AES-256")
		return "", fmt.Errorf("encryption key must be 32 bytes for AES-256")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	switch {
	case err != nil:
		logger.WithError(err).Error("failed to decode ciphertext base64")
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	switch {
	case err != nil:
		logger.WithError(err).Error("failed to decode nonce base64")
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	block, err := aes.NewCipher(key)
	switch {
	case err != nil:
		logger.WithError(err).Error("failed to create cipher block")
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	switch {
	case err != nil:
		logger.WithError(err).Error("failed to create GCM mode")
		return "", fmt.Errorf("failed to create GCM mode: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	switch {
	case err != nil:
		logger.WithError(err).Warn("failed to decrypt totp secret")
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	logger.Debug("totp secret decrypted successfully")
	return string(plaintext), nil
}
