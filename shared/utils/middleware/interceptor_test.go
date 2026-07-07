package middleware

import (
	"context"
	"errors"
	"testing"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeChecker struct {
	allowed bool
	err     error
}

func (f fakeChecker) Test(ctx context.Context, query UserQuery, userID string) (bool, error) {
	return f.allowed, f.err
}

func TestUnaryInterceptorAllowsEligibleCaller(t *testing.T) {
	query, err := structpb.NewStruct(map[string]any{"scope": "test"})
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	interceptor := NewUnaryServerInterceptor(Config{
		Checker: fakeChecker{allowed: true},
		Policies: map[string]Policy{
			"/svc.Test/Method": {Query: query},
		},
	})

	ctx := ContextWithPrincipal(context.Background(), Principal{UserID: "user-1", Authenticated: true})
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Test/Method"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != "ok" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestUnaryInterceptorRejectsUnauthenticatedCaller(t *testing.T) {
	interceptor := NewUnaryServerInterceptor(Config{})

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Test/Method"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestUnaryInterceptorRejectsIneligibleCaller(t *testing.T) {
	query, err := structpb.NewStruct(map[string]any{"scope": "test"})
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	interceptor := NewUnaryServerInterceptor(Config{
		Checker: fakeChecker{allowed: false},
		Policies: map[string]Policy{
			"/svc.Test/Method": {Query: query},
		},
	})

	ctx := ContextWithPrincipal(context.Background(), Principal{UserID: "user-1", Authenticated: true})
	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc.Test/Method"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestPolicyQueryFactory(t *testing.T) {
	policy := Policy{
		QueryFactory: func(ctx context.Context, request any) (UserQuery, error) {
			if request == nil {
				return nil, errors.New("request missing")
			}

			return request.(UserQuery), nil
		},
	}

	query, err := structpb.NewStruct(map[string]any{"scope": "test"})
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	resolved, err := policy.resolveQuery(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved != query {
		t.Fatalf("expected query factory to return original query")
	}
}

func TestPrincipalFromContextReadsIncomingMetadata(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		PrincipalUserIDHeader, "user-123",
		PrincipalEmailHeader, "user@example.com",
	))

	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		t.Fatalf("expected principal from metadata")
	}

	if principal.UserID != "user-123" {
		t.Fatalf("unexpected user id: %s", principal.UserID)
	}

	if principal.Email != "user@example.com" {
		t.Fatalf("unexpected email: %s", principal.Email)
	}

	if !principal.Authenticated {
		t.Fatalf("expected authenticated principal")
	}
}
