package security

import (
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
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
type JWTManager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
	logger    *logrus.Logger
}

// NewJWTManager constructs a JWTManager, validating the provided configuration.
func NewJWTManager(secret string, issuer string, accessTTL time.Duration, logger *logrus.Logger) (*JWTManager, error) {
	if logger == nil {
		logger = logrus.New()
	}

	if secret == "" {
		logger.Error("jwt secret cannot be empty")
		return nil, fmt.Errorf("jwt secret cannot be empty")
	}

	if accessTTL <= 0 {
		logger.WithField("access_ttl", accessTTL).Error("jwt access ttl must be positive")
		return nil, fmt.Errorf("jwt access ttl must be positive")
	}

	if issuer == "" {
		issuer = "user-core"
	}

	return &JWTManager{
		secret:    []byte(secret),
		issuer:    issuer,
		accessTTL: accessTTL,
		logger:    logger,
	}, nil
}

// GenerateToken creates a signed JWT for the provided user.
func (m *JWTManager) GenerateToken(user *domain.User) (string, time.Time, error) {
	if user == nil {
		m.logger.Error("user cannot be nil for token generation")
		return "", time.Time{}, fmt.Errorf("user cannot be nil")
	}

	expiresAt := time.Now().Add(m.accessTTL)

	claims := AccessTokenClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		m.logger.WithError(err).Error("failed to sign jwt")
		return "", time.Time{}, fmt.Errorf("failed to sign jwt: %w", err)
	}

	return signedToken, expiresAt, nil
}

// ValidateToken parses and verifies a signed JWT, returning claims on success.
func (m *JWTManager) ValidateToken(tokenString string) (*AccessTokenClaims, error) {
	if tokenString == "" {
		m.logger.Error("token string cannot be empty")
		return nil, fmt.Errorf("token string cannot be empty")
	}

	parsedToken, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			m.logger.WithField("alg", token.Method.Alg()).Error("unexpected signing method")
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}

		return m.secret, nil
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
