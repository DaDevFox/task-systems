package middleware

import (
	"context"
	"fmt"
	"errors"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	PrincipalUserIDHeader = "x-user-id"
	PrincipalEmailHeader  = "x-user-email"
)

// Principal describes the authenticated caller identity made available by the
// protected system boundary.
type Principal struct {
	UserID        string
	Email         string
	Authenticated bool
}

type principalKey struct{}

// ContextWithPrincipal stores the authenticated caller identity in context.
func ContextWithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFromContext retrieves the authenticated caller identity from context.
// It first checks for an explicitly stored principal, then falls back to
// trusted gRPC metadata injected by Envoy.
func PrincipalFromContext(ctx context.Context) (Principal, error) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	if !ok {
		return principalFromMetadata(ctx)
	}

	return principal, nil
}

// PrincipalUserIDFromContext returns the authenticated caller user ID when one is present.
func PrincipalUserIDFromContext(ctx context.Context) (string, error) {
	principal, err := PrincipalFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to extract principal from context: %w", err)
	}

	if !principal.Authenticated {
		return "", errors.New("unauthenticated principal")
	}

	userID := strings.TrimSpace(principal.UserID)
	if userID == "" {
		return "", errors.New("user ID was empty string")
	}

	return userID, nil
}

func principalFromMetadata(ctx context.Context) (Principal, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Principal{}, errors.New("failed to extract metadata from context")
	}

	userID := strings.TrimSpace(firstMetadataValue(md, PrincipalUserIDHeader))
	if userID == "" {
		return Principal{}, errors.New("user ID was empty string")
	}

	return Principal{
		UserID:        userID,
		Email:         strings.TrimSpace(firstMetadataValue(md, PrincipalEmailHeader)),
		Authenticated: true,
	}, nil
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
