package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/security"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	errMsgIdentifierEmpty      = "identifier cannot be empty"
	errMsgAccessTokenEmpty     = "access token cannot be empty"
	errMsgRefreshTokenEmpty    = "refresh token cannot be empty"
	errMsgInvalidCredentials   = "invalid credentials"
	errMsgRefreshTokenInvalid  = "refresh token invalid"
	errMsgRefreshTokenExpired  = "refresh token expired"
)

var (
	// ErrInvalidCredentials signals authentication or password verification failure.
	ErrInvalidCredentials = errors.New(errMsgInvalidCredentials)
	// ErrRefreshTokenInvalid indicates an unknown refresh token.
	ErrRefreshTokenInvalid = errors.New(errMsgRefreshTokenInvalid)
	// ErrRefreshTokenExpired indicates a refresh token past its TTL.
	ErrRefreshTokenExpired = errors.New(errMsgRefreshTokenExpired)
)

// AuthenticateResult bundles tokens and user data returned after successful authentication.
type AuthenticateResult struct {
	AccessToken          string
	AccessTokenExpiresAt time.Time
	RefreshToken         string
	User                 *domain.User
}

// RefreshResult provides refreshed access credentials along with user data.
type RefreshResult struct {
	AccessToken          string
	AccessTokenExpiresAt time.Time
	RefreshToken         string
	User                 *domain.User
}

// ValidateTokenResult exposes validated JWT claims for downstream services.
type ValidateTokenResult struct {
	Claims *security.AccessTokenClaims
}

// AuthService coordinates credential verification, password management, and token lifecycle.
type AuthService struct {
	userRepo      repository.UserRepository
	logger        *logrus.Logger
	jwtManager    *security.JWTManager
	refreshStore  security.RefreshTokenStore
	refreshTTL    time.Duration
	bcryptCost    int
	passwordBytes int
}

// NewAuthService constructs an AuthService with sane defaults.
func NewAuthService(userRepo repository.UserRepository, logger *logrus.Logger, jwtManager *security.JWTManager, refreshStore security.RefreshTokenStore, refreshTTL time.Duration) *AuthService {
	if logger == nil {
		logger = logrus.New()
	}

	if refreshTTL <= 0 {
		refreshTTL = 24 * time.Hour * 30
	}

	return &AuthService{
		userRepo:      userRepo,
		logger:        logger,
		jwtManager:    jwtManager,
		refreshStore:  refreshStore,
		refreshTTL:    refreshTTL,
		bcryptCost:    bcrypt.DefaultCost,
		passwordBytes: 32,
	}
}

// Authenticate validates credentials and issues new tokens.
func (s *AuthService) Authenticate(ctx context.Context, identifier string, password string) (*AuthenticateResult, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"operation":  "authenticate",
		"identifier": identifier,
	})

	if identifier == "" {
		logger.Error(errMsgIdentifierEmpty)
		return nil, fmt.Errorf(errMsgIdentifierEmpty)
	}

	if password == "" {
		logger.Error(errMsgPasswordEmpty)
		return nil, fmt.Errorf(errMsgPasswordEmpty)
	}

	user, err := s.resolveUser(ctx, identifier)
	if err != nil {
		logger.WithError(err).Warn("failed to resolve user for authentication")
		return nil, ErrInvalidCredentials
	}

	hashErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if hashErr != nil {
		logger.Warn("password verification failed")
		return nil, ErrInvalidCredentials
	}

	accessToken, accessExpiry, err := s.jwtManager.GenerateToken(user)
	if err != nil {
		logger.WithError(err).Error("failed to generate access token")
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		logger.WithError(err).Error("failed to generate refresh token")
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	metadata := security.RefreshTokenMetadata{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.refreshTTL),
		IssuedAt:  time.Now(),
	}

	saveErr := s.refreshStore.Save(ctx, refreshToken, metadata)
	if saveErr != nil {
		logger.WithError(saveErr).Error("failed to persist refresh token")
		return nil, fmt.Errorf("failed to persist refresh token: %w", saveErr)
	}

	s.recordSuccessfulLogin(ctx, user)

	return &AuthenticateResult{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessExpiry,
		RefreshToken:         refreshToken,
		User:                 user,
	}, nil
}

// RefreshToken exchanges a valid refresh token for a new access token (and rotated refresh token).
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	logger := s.logger.WithField("operation", "refresh_token")

	if refreshToken == "" {
		logger.Error(errMsgRefreshTokenEmpty)
		return nil, fmt.Errorf(errMsgRefreshTokenEmpty)
	}

	metadata, err := s.refreshStore.Get(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, security.ErrRefreshTokenNotFound) {
			logger.Warn("refresh token not found")
			return nil, ErrRefreshTokenInvalid
		}

		if errors.Is(err, security.ErrRefreshTokenExpired) {
			logger.Warn("refresh token expired")
			return nil, ErrRefreshTokenExpired
		}

		logger.WithError(err).Error("failed to load refresh token")
		return nil, fmt.Errorf("failed to load refresh token: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, metadata.UserID)
	if err != nil {
		logger.WithError(err).Error("failed to load user for refresh token")
		return nil, fmt.Errorf("failed to load user for refresh token: %w", err)
	}

	accessToken, accessExpiry, err := s.jwtManager.GenerateToken(user)
	if err != nil {
		logger.WithError(err).Error("failed to generate access token")
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	rotatedToken, err := s.generateRefreshToken()
	if err != nil {
		logger.WithError(err).Error("failed to generate refresh token")
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	rotationMetadata := security.RefreshTokenMetadata{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.refreshTTL),
		IssuedAt:  time.Now(),
	}

	saveErr := s.refreshStore.Save(ctx, rotatedToken, rotationMetadata)
	if saveErr != nil {
		logger.WithError(saveErr).Error("failed to persist rotated refresh token")
		return nil, fmt.Errorf("failed to persist refresh token: %w", saveErr)
	}

	deleteErr := s.refreshStore.Delete(ctx, refreshToken)
	if deleteErr != nil {
		logger.WithError(deleteErr).Warn("failed to delete previous refresh token")
	}

	return &RefreshResult{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessExpiry,
		RefreshToken:         rotatedToken,
		User:                 user,
	}, nil
}

// ValidateToken verifies an access token and returns its claims.
func (s *AuthService) ValidateToken(ctx context.Context, accessToken string) (*ValidateTokenResult, error) {
	logger := s.logger.WithField("operation", "validate_token")

	if accessToken == "" {
		logger.Error(errMsgAccessTokenEmpty)
		return nil, fmt.Errorf(errMsgAccessTokenEmpty)
	}

	claims, err := s.jwtManager.ValidateToken(accessToken)
	if err != nil {
		logger.WithError(err).Warn("token validation failed")
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	return &ValidateTokenResult{Claims: claims}, nil
}

// UpdatePassword updates a user's password after verifying the current password.
func (s *AuthService) UpdatePassword(ctx context.Context, userID string, currentPassword string, newPassword string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"operation": "update_password",
		"user_id":   userID,
	})

	if userID == "" {
		logger.Error(errMsgUserIDEmpty)
		return fmt.Errorf(errMsgUserIDEmpty)
	}

	if currentPassword == "" {
		logger.Error(errMsgPasswordEmpty)
		return fmt.Errorf(errMsgPasswordEmpty)
	}

	if len(newPassword) < security.MinPasswordLength {
		logger.WithField("min_length", security.MinPasswordLength).Error("new password too short")
		return fmt.Errorf("new password must be at least %d characters", security.MinPasswordLength)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("failed to load user for password update")
		return fmt.Errorf("failed to load user: %w", err)
	}

	compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword))
	if compareErr != nil {
		logger.Warn("current password verification failed")
		return ErrInvalidCredentials
	}

	hashBytes, hashErr := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if hashErr != nil {
		logger.WithError(hashErr).Error("failed to hash new password")
		return fmt.Errorf("failed to hash new password: %w", hashErr)
	}

	user.PasswordHash = string(hashBytes)
	now := time.Now()
	user.UpdatedAt = now

	updateErr := s.userRepo.Update(ctx, user)
	if updateErr != nil {
		logger.WithError(updateErr).Error("failed to persist password update")
		return fmt.Errorf("failed to update password: %w", updateErr)
	}

	logger.Info("password updated successfully")
	return nil
}

func (s *AuthService) generateRefreshToken() (string, error) {
	buffer := make([]byte, s.passwordBytes)
	_, err := rand.Read(buffer)
	if err != nil {
		s.logger.WithError(err).Error("failed to read crypto random bytes")
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(buffer)
	return token, nil
}

func (s *AuthService) resolveUser(ctx context.Context, identifier string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, identifier)
	if err == nil {
		return user, nil
	}

	if errors.Is(err, repository.ErrUserNotFound) {
		userByID, byIDErr := s.userRepo.GetByID(ctx, identifier)
		if byIDErr == nil {
			return userByID, nil
		}

		return nil, byIDErr
	}

	return nil, err
}

func (s *AuthService) recordSuccessfulLogin(ctx context.Context, user *domain.User) {
	now := time.Now()
	user.LastLogin = &now
	user.UpdatedAt = now

	saveErr := s.userRepo.Update(ctx, user)
	if saveErr != nil {
		s.logger.WithError(saveErr).Warn("failed to persist last login timestamp")
	}
}
