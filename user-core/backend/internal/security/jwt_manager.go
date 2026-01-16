package security

import (
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultRSABits is the default RSA key size in bits
	DefaultRSABits = 2048
	// CurrentKeyID is the key ID for the current active key
	CurrentKeyID = "current"
	// DefaultSecretsDir is the default directory for storing keys
	DefaultSecretsDir = "./secrets"
	// PrivateKeyFilename is the filename for the private key
	PrivateKeyFilename = "jwt_private.pem"
	// PublicKeyFilename is the filename for the public key
	PublicKeyFilename = "jwt_public.pem"
)

// AccessTokenClaims captures JWT claims for access tokens
// Embeds jwt.RegisteredClaims to leverage standard fields
// Role is stored as the string version of the domain role for readability across services
// and to avoid coupling downstream services to numeric enum ordinals
// The claim keys follow a short convention to minimize token size while remaining descriptive
// uid -> user ID, email -> user email, role -> user role string
// Additional fields can be added as future enhancements (e.g., tenant, permissions)
// without breaking compatibility when using JWT registered claim conventions
// Issuer and audience handling is deferred to configuration in the JWT manager
// to keep claim generation focused on user-centric data
// Future OAuth integrations can reuse these claims or extend them as needed
// while still going through the manager for signing and validation
// This ensures centralized security hardening for token logic
// and provides a single place to inject features like key rotation.
type AccessTokenClaims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager handles access token issuance and validation
// It encapsulates signing key material, token TTL, and issuer configuration
// and centralizes structured logging for token operations.
// Supports RSA-based asymmetric signing with RS256 and key rotation.
type JWTManager struct {
	privateKey      *rsa.PrivateKey
	currentKeyID    string
	publicKeys      map[string]*rsa.PublicKey // keyID -> public key for verification
	publicKeysMutex sync.RWMutex
	issuer          string
	accessTTL       time.Duration
	logger          *logrus.Logger
	secretsDir      string
}

// NewJWTManager constructs a JWTManager with RSA asymmetric signing.
// It loads existing RSA keys from the secrets directory or generates new ones if they don't exist.
func NewJWTManager(secret string, issuer string, accessTTL time.Duration, logger *logrus.Logger) (*JWTManager, error) {
	return NewJWTManagerWithConfig(NewJWTManagerConfig{
		SecretDir:    DefaultSecretsDir,
		Issuer:       issuer,
		AccessTTL:    accessTTL,
		Logger:       logger,
		RSABits:      DefaultRSABits,
		GenerateKeys: true,
	})
}

// NewJWTManagerWithConfig constructs a JWTManager with custom configuration.
func NewJWTManagerWithConfig(config NewJWTManagerConfig) (*JWTManager, error) {
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	if config.AccessTTL <= 0 {
		config.Logger.WithField("access_ttl", config.AccessTTL).Error("jwt access ttl must be positive")
		return nil, fmt.Errorf("jwt access ttl must be positive")
	}

	if config.Issuer == "" {
		config.Issuer = "user-core"
	}

	if config.RSABits < 2048 {
		config.Logger.WithField("rsa_bits", config.RSABits).Warn("RSA key size less than 2048 bits is not recommended")
	}

	if config.SecretDir == "" {
		config.SecretDir = DefaultSecretsDir
	}

	manager := &JWTManager{
		currentKeyID: CurrentKeyID,
		publicKeys:   make(map[string]*rsa.PublicKey),
		issuer:       config.Issuer,
		accessTTL:    config.AccessTTL,
		logger:       config.Logger,
		secretsDir:   config.SecretDir,
	}

	// Load or generate keys
	err := manager.initializeKeys(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keys: %w", err)
	}

	return manager, nil
}

// NewJWTManagerConfig provides configuration options for JWTManager.
type NewJWTManagerConfig struct {
	SecretDir    string        // Directory to store/load keys
	Issuer       string        // Token issuer
	AccessTTL    time.Duration // Token time-to-live
	Logger       *logrus.Logger
	RSABits      int  // RSA key size in bits
	GenerateKeys bool // Whether to generate keys if they don't exist
}

// initializeKeys loads existing keys or generates new ones.
func (m *JWTManager) initializeKeys(config NewJWTManagerConfig) error {
	// Ensure secrets directory exists
	if err := os.MkdirAll(m.secretsDir, 0o700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	privateKeyPath := filepath.Join(m.secretsDir, PrivateKeyFilename)
	publicKeyPath := filepath.Join(m.secretsDir, PublicKeyFilename)

	// Check if keys already exist
	privateKeyExists, _ := fileExists(privateKeyPath)
	publicKeyExists, _ := fileExists(publicKeyPath)

	if privateKeyExists && publicKeyExists {
		m.logger.WithField("secrets_dir", m.secretsDir).Info("Loading existing RSA keys")

		privateKey, err := LoadRSAPrivateKey(privateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load private key: %w", err)
		}

		publicKey, err := LoadRSAPublicKey(publicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load public key: %w", err)
		}

		m.privateKey = privateKey
		m.addPublicKey(m.currentKeyID, publicKey)

		m.logger.Info("RSA keys loaded successfully")
		return nil
	}

	// Keys don't exist, generate new ones
	if !config.GenerateKeys {
		return fmt.Errorf("keys not found and key generation is disabled")
	}

	m.logger.WithFields(logrus.Fields{
		"rsa_bits":    config.RSABits,
		"secrets_dir": m.secretsDir,
		"private_key": privateKeyPath,
		"public_key":  publicKeyPath,
	}).Info("Generating new RSA key pair")

	privateKey, publicKey, err := GenerateRSAKeyPair(config.RSABits)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	// Save keys with secure permissions
	if err := SavePrivateKey(privateKeyPath, privateKey); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	if err := SavePublicKey(publicKeyPath, publicKey); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	m.privateKey = privateKey
	m.addPublicKey(m.currentKeyID, publicKey)

	m.logger.Info("RSA keys generated and saved successfully")
	return nil
}

// addPublicKey adds a public key to the validation key set with thread safety.
func (m *JWTManager) addPublicKey(keyID string, publicKey *rsa.PublicKey) {
	m.publicKeysMutex.Lock()
	defer m.publicKeysMutex.Unlock()
	m.publicKeys[keyID] = publicKey
}

// getPublicKey retrieves a public key by ID with thread safety.
func (m *JWTManager) getPublicKey(keyID string) (*rsa.PublicKey, bool) {
	m.publicKeysMutex.RLock()
	defer m.publicKeysMutex.RUnlock()
	key, exists := m.publicKeys[keyID]
	return key, exists
}

// GenerateToken creates a signed JWT for the provided user using RS256.
func (m *JWTManager) GenerateToken(user *domain.User) (string, time.Time, error) {
	if user == nil {
		m.logger.Error("user cannot be nil for token generation")
		return "", time.Time{}, fmt.Errorf("user cannot be nil")
	}

	expiresAt := time.Now().Add(m.accessTTL)
	now := time.Now()

	claims := AccessTokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        m.currentKeyID, // Store key ID for key rotation support
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signedToken, err := token.SignedString(m.privateKey)
	if err != nil {
		m.logger.WithError(err).Error("failed to sign jwt")
		return "", time.Time{}, fmt.Errorf("failed to sign jwt: %w", err)
	}

	return signedToken, expiresAt, nil
}

// ValidateToken parses and verifies a signed JWT, returning claims on success.
// It validates tokens signed with RS256 using the public key.
// Supports key rotation by checking the key ID in the token's claims.
func (m *JWTManager) ValidateToken(tokenString string) (*AccessTokenClaims, error) {
	if tokenString == "" {
		m.logger.Error("token string cannot be empty")
		return nil, fmt.Errorf("token string cannot be empty")
	}

	parsedToken, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if token.Method != jwt.SigningMethodRS256 {
			m.logger.WithField("alg", token.Method.Alg()).Error("unexpected signing method")
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}

		// Extract key ID from token if present
		var keyID string
		if claims, ok := token.Claims.(*AccessTokenClaims); ok {
			keyID = claims.ID
		}

		// Use specific key if key ID is present, otherwise use current key
		if keyID != "" {
			if publicKey, exists := m.getPublicKey(keyID); exists {
				m.logger.WithField("key_id", keyID).Debug("validating token with specific key")
				return publicKey, nil
			}
			m.logger.WithField("key_id", keyID).Warn("key ID not found, falling back to current key")
		}

		// Fall back to current key
		publicKey, exists := m.getPublicKey(m.currentKeyID)
		if !exists {
			m.logger.Error("no public key available for validation")
			return nil, fmt.Errorf("no public key available for validation")
		}

		return publicKey, nil
	})

	if err != nil {
		m.logger.WithError(err).Warn("failed to parse jwt")
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := parsedToken.Claims.(*AccessTokenClaims)
	if !ok {
		m.logger.Error("failed to cast jwt claims")
		return nil, fmt.Errorf("failed to cast token claims")
	}

	if !parsedToken.Valid {
		m.logger.Warn("jwt token invalid")
		return nil, fmt.Errorf("token is invalid")
	}

	return claims, nil
}

// RotateKeys generates a new RSA key pair for signing new tokens while
// keeping old keys for validating existing tokens. This is useful for
// periodic key rotation without invalidating existing tokens.
func (m *JWTManager) RotateKeys() error {
	m.logger.Info("Rotating RSA keys")

	// Generate new key pair
	privateKey, publicKey, err := GenerateRSAKeyPair(DefaultRSABits)
	if err != nil {
		return fmt.Errorf("failed to generate new RSA key pair: %w", err)
	}

	// Save with new filenames to preserve old keys
	oldKeyID := m.currentKeyID
	newKeyID := fmt.Sprintf("key_%d", time.Now().Unix())

	privateKeyPath := filepath.Join(m.secretsDir, fmt.Sprintf("jwt_%s_private.pem", newKeyID))
	publicKeyPath := filepath.Join(m.secretsDir, fmt.Sprintf("jwt_%s_public.pem", newKeyID))

	if err := SavePrivateKey(privateKeyPath, privateKey); err != nil {
		return fmt.Errorf("failed to save new private key: %w", err)
	}

	if err := SavePublicKey(publicKeyPath, publicKey); err != nil {
		return fmt.Errorf("failed to save new public key: %w", err)
	}

	// Update manager state
	m.publicKeysMutex.Lock()
	// Keep old public key for validation
	if oldPublicKey, exists := m.publicKeys[oldKeyID]; exists {
		m.publicKeys[oldKeyID] = oldPublicKey
	}
	m.publicKeys[newKeyID] = publicKey
	m.currentKeyID = newKeyID
	m.privateKey = privateKey
	m.publicKeysMutex.Unlock()

	m.logger.WithFields(logrus.Fields{
		"old_key_id": oldKeyID,
		"new_key_id": newKeyID,
	}).Info("RSA keys rotated successfully")

	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
