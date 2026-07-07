package middleware

import (
	"context"
	"fmt"
	"strings"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Config controls the shared authorization middleware behavior.
type Config struct {
	PrincipalFromContext PrincipalExtractor
	Checker              EligibilityChecker
	Policies             map[string]Policy
	DefaultPolicy        *Policy
}

// PrincipalExtractor returns the authenticated caller identity from context.
type PrincipalExtractor func(context.Context) (Principal, bool)

// NewUnaryServerInterceptor builds a gRPC unary interceptor that authorizes callers
// after the Zitadel middleware has attached the principal to the context.
func NewUnaryServerInterceptor(cfg Config) grpc.UnaryServerInterceptor {
	config := normalizeConfig(cfg)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := config.authorize(ctx, info.FullMethod, req); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// NewStreamServerInterceptor builds a gRPC stream interceptor that authorizes the
// caller on the first received stream message.
func NewStreamServerInterceptor(cfg Config) grpc.StreamServerInterceptor {
	config := normalizeConfig(cfg)

	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapped := &authorizationStream{
			ServerStream: stream,
			config:       config,
			fullMethod:   info.FullMethod,
		}

		return handler(srv, wrapped)
	}
}

func normalizeConfig(cfg Config) Config {
	if cfg.PrincipalFromContext == nil {
		cfg.PrincipalFromContext = PrincipalFromContext
	}

	if cfg.Policies == nil {
		cfg.Policies = make(map[string]Policy)
	}

	return cfg
}

func (cfg Config) authorize(ctx context.Context, fullMethod string, req any) error {
	policy := cfg.policyForMethod(fullMethod)
	if policy.AllowUnauthed {
		return nil
	}

	principal, ok := cfg.PrincipalFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "authentication required")
	}

	if !principal.Authenticated {
		return status.Error(codes.Unauthenticated, "authentication required")
	}

	userID := strings.TrimSpace(principal.UserID)
	if userID == "" {
		return status.Error(codes.Unauthenticated, "authenticated user ID is required")
	}

	query, err := policy.resolveQuery(ctx, req)
	if err != nil {
		return status.Error(codes.PermissionDenied, fmt.Sprintf("authorization policy rejected request: %v", err))
	}

	if query == nil {
		return nil
	}

	if cfg.Checker == nil {
		return status.Error(codes.Internal, "authorization checker is required")
	}

	allowed, err := cfg.Checker.Test(ctx, query, userID)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("authorization test failed: %v", err))
	}

	if allowed {
		return nil
	}

	return status.Error(codes.PermissionDenied, "caller is not eligible for this method")
}

func (cfg Config) policyForMethod(fullMethod string) Policy {
	if policy, ok := cfg.Policies[fullMethod]; ok {
		return policy
	}

	if cfg.DefaultPolicy != nil {
		return *cfg.DefaultPolicy
	}

	return Policy{}
}

type authorizationStream struct {
	grpc.ServerStream
	config     Config
	fullMethod string
	authorized bool
	message    any
}

func (s *authorizationStream) RecvMsg(message any) error {
	err := s.ServerStream.RecvMsg(message)
	if err != nil {
		return err
	}

	if s.authorized {
		return nil
	}

	err = s.config.authorize(s.Context(), s.fullMethod, message)
	if err != nil {
		return err
	}

	s.authorized = true
	return nil
}