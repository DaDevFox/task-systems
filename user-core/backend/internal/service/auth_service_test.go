package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/repository"
	"github.com/DaDevFox/task-systems/user-core/backend/internal/security"
	"github.com/sirupsen/logrus"
)

func TestAuthServiceAuthenticateAndRefresh(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	repo := repository.NewInMemoryUserRepository()
	userService := NewUserService(repo, logger)

	ctx := context.Background()
	createParams := CreateUserParams{
		Email:    "auth@test.dev",
		Name:     "Auth Tester",
		Password: "Str0ngPass!",
	}

	user, err := userService.CreateUser(ctx, createParams)
	switch {
	case err != nil:
		t.Fatalf("expected user creation success, got error: %v", err)
	case user == nil:
		t.Fatal("expected created user, got nil")
	}

	jwtManager, err := security.NewJWTManager("super-secret", "test-suite", time.Minute, logger)
	switch {
	case err != nil:
		t.Fatalf("expected jwt manager creation success, got error: %v", err)
	case jwtManager == nil:
		t.Fatal("expected jwt manager instance, got nil")
	}

	refreshStore := security.NewInMemoryRefreshTokenStore(logger)
	authService := NewAuthService(repo, logger, jwtManager, refreshStore, time.Hour)

	authResult, err := authService.Authenticate(ctx, createParams.Email, createParams.Password)
	switch {
	case err != nil:
		t.Fatalf("expected authentication success, got error: %v", err)
	case authResult == nil:
		t.Fatal("expected authentication result, got nil")
	case authResult.AccessToken == "":
		t.Fatal("expected non-empty access token")
	case authResult.RefreshToken == "":
		t.Fatal("expected non-empty refresh token")
	}

	validateResult, err := authService.ValidateToken(ctx, authResult.AccessToken)
	switch {
	case err != nil:
		t.Fatalf("expected token validation success, got error: %v", err)
	case validateResult == nil:
		t.Fatal("expected validation result, got nil")
	case validateResult.Claims == nil:
		t.Fatal("expected validation claims, got nil")
	case validateResult.Claims.UserID != authResult.User.ID:
		t.Fatalf("expected user ID %s, got %s", authResult.User.ID, validateResult.Claims.UserID)
	}

	refreshResult, err := authService.RefreshToken(ctx, authResult.RefreshToken)
	switch {
	case err != nil:
		t.Fatalf("expected refresh success, got error: %v", err)
	case refreshResult == nil:
		t.Fatal("expected refresh result, got nil")
	case refreshResult.AccessToken == "":
		t.Fatal("expected refreshed access token, got empty string")
	case refreshResult.RefreshToken == "":
		t.Fatal("expected rotated refresh token, got empty string")
	}
}

func TestAuthServiceUpdatePassword(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	repo := repository.NewInMemoryUserRepository()
	userService := NewUserService(repo, logger)

	ctx := context.Background()
	createParams := CreateUserParams{
		Email:    "rotate@test.dev",
		Name:     "Rotator",
		Password: "Sup3rPass!",
	}

	user, err := userService.CreateUser(ctx, createParams)
	switch {
	case err != nil:
		t.Fatalf("failed to create user: %v", err)
	case user == nil:
		t.Fatal("expected user instance, got nil")
	}

	jwtManager, err := security.NewJWTManager("super-secret", "test-suite", time.Minute, logger)
	switch {
	case err != nil:
		t.Fatalf("failed to create jwt manager: %v", err)
	case jwtManager == nil:
		t.Fatal("expected jwt manager instance, got nil")
	}

	refreshStore := security.NewInMemoryRefreshTokenStore(logger)
	authService := NewAuthService(repo, logger, jwtManager, refreshStore, time.Hour)

	_, err = authService.Authenticate(ctx, createParams.Email, createParams.Password)
	switch {
	case err != nil:
		t.Fatalf("expected authentication to succeed, got error: %v", err)
	}

	newPassword := "N3wSup3rPass!!"
	err = authService.UpdatePassword(ctx, user.ID, createParams.Password, newPassword)
	switch {
	case err != nil:
		t.Fatalf("expected password update success, got error: %v", err)
	}

	_, err = authService.Authenticate(ctx, createParams.Email, createParams.Password)
	switch {
	case !errors.Is(err, ErrInvalidCredentials):
		t.Fatalf("expected invalid credentials error, got: %v", err)
	}

	authResult, err := authService.Authenticate(ctx, createParams.Email, newPassword)
	switch {
	case err != nil:
		t.Fatalf("expected authentication success with new password, got error: %v", err)
	case authResult == nil:
		t.Fatal("expected auth result with new password, got nil")
	}
}
