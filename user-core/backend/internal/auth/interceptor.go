package auth

import (
	"context"

	sharedauth "github.com/DaDevFox/task-systems/shared/auth"
	"github.com/sirupsen/logrus"
)

type Claims = sharedauth.Claims
type TokenVerifier = sharedauth.TokenVerifier
type Interceptor = sharedauth.Interceptor

func NewTokenVerifier(ctx context.Context, issuer string, audiences []string) (*TokenVerifier, error) {
	return sharedauth.NewTokenVerifier(ctx, issuer, audiences)
}

func NewInterceptorFromEnv(ctx context.Context, logger *logrus.Logger, defaultPublicMethods []string) (*Interceptor, error) {
	return sharedauth.NewInterceptorFromEnv(ctx, logger, defaultPublicMethods)
}

func DefaultPublicMethods() []string {
	return sharedauth.DefaultPublicMethods()
}

func DefaultUserPublicMethods() []string {
	return sharedauth.DefaultUserPublicMethods()
}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return sharedauth.WithClaims(ctx, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	return sharedauth.ClaimsFromContext(ctx)
}
