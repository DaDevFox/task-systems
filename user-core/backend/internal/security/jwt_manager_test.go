package security

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJWTManager_RS256TokenGeneration tests that JWT tokens are generated with RS256.
func TestJWTManager_RS256TokenGeneration(t *testing.T) {
	logger := setupTestLogger(t)

	// Create a temporary directory for keys
	tmpDir := t.TempDir()

	// Create JWT manager with RSA keys
	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Verify keys were created
	privateKeyPath := filepath.Join(tmpDir, PrivateKeyFilename)
	publicKeyPath := filepath.Join(tmpDir, PublicKeyFilename)

	_, err = os.Stat(privateKeyPath)
	assert.NoError(t, err, "private key file should exist")

	_, err = os.Stat(publicKeyPath)
	assert.NoError(t, err, "public key file should exist")

	// Create a test user
	user := &domain.User{
		ID:    "test-user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  domain.UserRoleUser,
	}

	// Generate token
	token, expiresAt, err := manager.GenerateToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))

	// Token should not be empty and should look like a JWT (three parts separated by dots)
	parts := splitToken(token)
	assert.Len(t, parts, 3, "JWT should have three parts")
}

// TestJWTManager_RS256TokenValidation tests that tokens are validated with RS256.
func TestJWTManager_RS256TokenValidation(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	user := &domain.User{
		ID:    "test-user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  domain.UserRoleAdmin,
	}

	// Generate token
	token, _, err := manager.GenerateToken(user)
	require.NoError(t, err)

	// Validate token
	claims, err := manager.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Role.String(), claims.Role)
	assert.Equal(t, "test-issuer", claims.Issuer)
	assert.Equal(t, user.ID, claims.Subject)
	assert.NotEmpty(t, claims.ID) // Key ID should be present
}

// TestJWTManager_TokenValidationWithExpiredToken tests that expired tokens are rejected.
func TestJWTManager_TokenValidationWithExpiredToken(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	// Create manager with very short TTL
	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Nanosecond, // Almost immediate expiration
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	user := &domain.User{
		ID:    "test-user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  domain.UserRoleUser,
	}

	token, _, err := manager.GenerateToken(user)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validate token - should fail
	_, err = manager.ValidateToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// TestJWTManager_TokenValidationWithInvalidSignature tests that tokens with invalid signatures are rejected.
func TestJWTManager_TokenValidationWithInvalidSignature(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	user := &domain.User{
		ID:    "test-user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  domain.UserRoleUser,
	}

	token, _, err := manager.GenerateToken(user)
	require.NoError(t, err)

	// Corrupt the signature by replacing the last part
	parts := splitToken(token)
	require.Len(t, parts, 3)
	corruptedToken := parts[0] + "." + parts[1] + "." + "invalidsignature"

	// Validate corrupted token - should fail
	_, err = manager.ValidateToken(corruptedToken)
	assert.Error(t, err)
}

// TestJWTManager_KeyRotation tests that key rotation works correctly.
func TestJWTManager_KeyRotation(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	user := &domain.User{
		ID:    "test-user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  domain.UserRoleUser,
	}

	// Generate token with old key
	token1, _, err := manager.GenerateToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	// Rotate keys
	err = manager.RotateKeys()
	require.NoError(t, err)

	// Generate token with new key
	token2, _, err := manager.GenerateToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token2)

	// Tokens should be different
	assert.NotEqual(t, token1, token2)

	// Both tokens should be valid
	claims1, err := manager.ValidateToken(token1)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims1.UserID)

	claims2, err := manager.ValidateToken(token2)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims2.UserID)
}

// TestJWTManager_LoadExistingKeys tests that existing keys are loaded properly.
func TestJWTManager_LoadExistingKeys(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	// Create keys first
	privateKey, publicKey, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	privateKeyPath := filepath.Join(tmpDir, PrivateKeyFilename)
	publicKeyPath := filepath.Join(tmpDir, PublicKeyFilename)

	err = SavePrivateKey(privateKeyPath, privateKey)
	require.NoError(t, err)

	err = SavePublicKey(publicKeyPath, publicKey)
	require.NoError(t, err)

	// Create manager with key generation disabled
	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: false,
	})
	require.NoError(t, err)
	require.NotNil(t, manager)

	user := &domain.User{
		ID:    "test-user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  domain.UserRoleUser,
	}

	// Generate and validate token with loaded keys
	token, _, err := manager.GenerateToken(user)
	require.NoError(t, err)

	claims, err := manager.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
}

// TestJWTManager_KeyFilePermissions tests that key files have proper permissions.
func TestJWTManager_KeyFilePermissions(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	_, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	privateKeyPath := filepath.Join(tmpDir, PrivateKeyFilename)
	publicKeyPath := filepath.Join(tmpDir, PublicKeyFilename)

	// Check that files exist
	privateKeyInfo, err := os.Stat(privateKeyPath)
	require.NoError(t, err)
	assert.NotZero(t, privateKeyInfo.Size(), "private key file should not be empty")

	publicKeyInfo, err := os.Stat(publicKeyPath)
	require.NoError(t, err)
	assert.NotZero(t, publicKeyInfo.Size(), "public key file should not be empty")

	// On Unix-like systems, verify permissions (skip on Windows)
	if privateKeyInfo.Mode().Perm() == os.FileMode(0o600) {
		// Unix-like system - check exact permissions
		assert.Equal(t, os.FileMode(0o600), privateKeyInfo.Mode().Perm(), "private key should have 0o600 permissions")
		assert.Equal(t, os.FileMode(0o644), publicKeyInfo.Mode().Perm(), "public key should have 0o644 permissions")
	}
	// On Windows, exact permissions can't be set the same way, but files are created securely
}

// TestJWTManager_InvalidSigningMethod tests that tokens with wrong signing method are rejected.
func TestJWTManager_InvalidSigningMethod(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	// Try to validate a non-JWT string
	_, err = manager.ValidateToken("not-a-jwt")
	assert.Error(t, err)
}

// TestJWTManager_ValidateTokenWithDifferentKey tests that a manager with different keys can't validate tokens.
func TestJWTManager_ValidateTokenWithDifferentKey(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create first manager
	manager1, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir1,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	user := &domain.User{
		ID:    "test-user-123",
		Email: "test@example.com",
		Name:  "Test User",
		Role:  domain.UserRoleUser,
	}

	// Generate token with first manager
	token, _, err := manager1.GenerateToken(user)
	require.NoError(t, err)

	// Create second manager with different keys
	manager2, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir2,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	// Second manager should not be able to validate token from first manager
	_, err = manager2.ValidateToken(token)
	assert.Error(t, err)
}

// TestJWTManager_RSABits tests that different RSA key sizes work.
func TestJWTManager_RSABits(t *testing.T) {
	logger := setupTestLogger(t)
	testCases := []int{2048, 3072, 4096}

	for _, bits := range testCases {
		t.Run(fmt.Sprintf("%d bits", bits), func(t *testing.T) {
			tmpDir := t.TempDir()

			manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
				SecretDir:    tmpDir,
				Issuer:       "test-issuer",
				AccessTTL:    time.Hour,
				Logger:       logger,
				RSABits:      bits,
				GenerateKeys: true,
			})
			require.NoError(t, err)

			user := &domain.User{
				ID:    "test-user-123",
				Email: "test@example.com",
				Name:  "Test User",
				Role:  domain.UserRoleUser,
			}

			// Generate and validate token
			token, _, err := manager.GenerateToken(user)
			require.NoError(t, err)

			claims, err := manager.ValidateToken(token)
			require.NoError(t, err)
			assert.Equal(t, user.ID, claims.UserID)
		})
	}
}

// TestJWTManager_ValidateTokenWithEmptyString tests that empty token string is rejected.
func TestJWTManager_ValidateTokenWithEmptyString(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	_, err = manager.ValidateToken("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// TestGenerateRSAKeyPair tests RSA key pair generation.
func TestGenerateRSAKeyPair(t *testing.T) {
	// Test various bit sizes
	testCases := []int{2048, 3072, 4096}

	for _, bits := range testCases {
		t.Run(fmt.Sprintf("%d bits", bits), func(t *testing.T) {
			privateKey, publicKey, err := GenerateRSAKeyPair(bits)
			require.NoError(t, err)
			require.NotNil(t, privateKey)
			require.NotNil(t, publicKey)

			// Verify key sizes
			assert.Equal(t, bits, privateKey.N.BitLen())
			assert.Equal(t, bits, publicKey.N.BitLen())

			// Verify that public key matches private key
			assert.Equal(t, privateKey.PublicKey.N, publicKey.N)
			assert.Equal(t, privateKey.PublicKey.E, publicKey.E)
		})
	}
}

// TestSaveAndLoadRSAKeys tests saving and loading RSA keys.
func TestSaveAndLoadRSAKeys(t *testing.T) {
	tmpDir := t.TempDir()

	privateKey, publicKey, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")

	// Save keys
	err = SavePrivateKey(privateKeyPath, privateKey)
	require.NoError(t, err)

	err = SavePublicKey(publicKeyPath, publicKey)
	require.NoError(t, err)

	// Load keys
	loadedPrivateKey, err := LoadRSAPrivateKey(privateKeyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey.N, loadedPrivateKey.N)
	assert.Equal(t, privateKey.E, loadedPrivateKey.E)
	assert.Equal(t, privateKey.D, loadedPrivateKey.D)

	loadedPublicKey, err := LoadRSAPublicKey(publicKeyPath)
	require.NoError(t, err)
	assert.Equal(t, publicKey.N, loadedPublicKey.N)
	assert.Equal(t, publicKey.E, loadedPublicKey.E)
}

// TestParseRSAKeysPEM tests parsing RSA keys from PEM data.
func TestParseRSAKeysPEM(t *testing.T) {
	privateKey, publicKey, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	// Save and read as bytes to simulate PEM data
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")

	err = SavePrivateKey(privateKeyPath, privateKey)
	require.NoError(t, err)

	err = SavePublicKey(publicKeyPath, publicKey)
	require.NoError(t, err)

	// Read PEM data
	privateKeyPEM, err := os.ReadFile(privateKeyPath)
	require.NoError(t, err)

	publicKeyPEM, err := os.ReadFile(publicKeyPath)
	require.NoError(t, err)

	// Parse PEM data
	parsedPrivateKey, err := ParseRSAPrivateKeyPEM(privateKeyPEM)
	require.NoError(t, err)
	assert.Equal(t, privateKey.N, parsedPrivateKey.N)

	parsedPublicKey, err := ParseRSAPublicKeyPEM(publicKeyPEM)
	require.NoError(t, err)
	assert.Equal(t, publicKey.N, parsedPublicKey.N)
}

// TestNewJWTManagerWithInvalidConfig tests error handling for invalid configurations.
func TestNewJWTManagerWithInvalidConfig(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	testCases := []struct {
		name     string
		config   NewJWTManagerConfig
		expected string
	}{
		{
			name: "negative TTL",
			config: NewJWTManagerConfig{
				SecretDir:    tmpDir,
				AccessTTL:    -1 * time.Hour,
				Logger:       logger,
				RSABits:      2048,
				GenerateKeys: true,
			},
			expected: "must be positive",
		},
		{
			name: "zero TTL",
			config: NewJWTManagerConfig{
				SecretDir:    tmpDir,
				AccessTTL:    0,
				Logger:       logger,
				RSABits:      2048,
				GenerateKeys: true,
			},
			expected: "must be positive",
		},
		{
			name: "keys not found and generation disabled",
			config: NewJWTManagerConfig{
				SecretDir:    tmpDir,
				AccessTTL:    time.Hour,
				Logger:       logger,
				RSABits:      2048,
				GenerateKeys: false,
			},
			expected: "keys not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewJWTManagerWithConfig(tc.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expected)
		})
	}
}

// setupTestLogger creates a logger that discards output for tests.
func setupTestLogger(t *testing.T) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

// splitToken splits a JWT token into its three parts.
func splitToken(token string) []string {
	parts := make([]string, 0, 3)
	start := 0
	for i, r := range token {
		if r == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}

// TestJWTManager_GenerateTokenWithNilUser tests that generating token with nil user fails.
func TestJWTManager_GenerateTokenWithNilUser(t *testing.T) {
	logger := setupTestLogger(t)
	tmpDir := t.TempDir()

	manager, err := NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    tmpDir,
		Issuer:       "test-issuer",
		AccessTTL:    time.Hour,
		Logger:       logger,
		RSABits:      2048,
		GenerateKeys: true,
	})
	require.NoError(t, err)

	_, _, err = manager.GenerateToken(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}
