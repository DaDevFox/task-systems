package auth

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey struct{}

var claimsContextKey contextKey

// ContextWithClaims embeds authenticated claims into the provided context.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext extracts claims if present.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	if ctx == nil {
		return nil, false
	}

	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok && claims != nil
}

// RequireClaims ensures authentication has occurred for the request.
func RequireClaims(ctx context.Context) (*Claims, error) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	return claims, nil
}

// RequireRole asserts that the authenticated caller possesses at least one required role.
func RequireRole(ctx context.Context, roles ...Role) (*Claims, error) {
	claims, err := RequireClaims(ctx)
	if err != nil {
		return nil, err
	}

	if len(roles) == 0 || claims.HasAnyRole(roles...) {
		return claims, nil
	}

	return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
}
