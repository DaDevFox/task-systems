package middleware

import (
	"context"
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
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	if !ok {
		return principalFromMetadata(ctx)
	}

	return principal, true
}

// PrincipalUserIDFromContext returns the authenticated caller user ID when one is present.
func PrincipalUserIDFromContext(ctx context.Context) (string, bool) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return "", false
	}

	if !principal.Authenticated {
		return "", false
	}

	userID := strings.TrimSpace(principal.UserID)
	if userID == "" {
		return "", false
	}

	return userID, true
}

func principalFromMetadata(ctx context.Context) (Principal, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Principal{}, false
	}

	userID := strings.TrimSpace(firstMetadataValue(md, PrincipalUserIDHeader))
	if userID == "" {
		return Principal{}, false
	}

	return Principal{
		UserID:        userID,
		Email:         strings.TrimSpace(firstMetadataValue(md, PrincipalEmailHeader)),
		Authenticated: true,
	}, true
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
