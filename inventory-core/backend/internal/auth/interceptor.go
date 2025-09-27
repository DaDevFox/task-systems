package auth

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userpb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/usercore/v1"
)

const (
	authorizationMetadataKey = "authorization"
	bearerPrefix             = "bearer "
	metadataRequiredMessage  = "authorization metadata required"
)

// Interceptor validates incoming requests against the user-core service and injects claims into the context.
type Interceptor struct {
	logger     *logrus.Logger
	userClient userpb.UserServiceClient
	allowlist  map[string]struct{}
}

// Option customizes the interceptor behaviour.
type Option func(*Interceptor)

// WithAllowUnauthenticated registers fully qualified RPC method names that bypass authentication.
func WithAllowUnauthenticated(methods ...string) Option {
	return func(i *Interceptor) {
		for _, m := range methods {
			if strings.TrimSpace(m) == "" {
				continue
			}
			i.allowlist[m] = struct{}{}
		}
	}
}

// NewInterceptor constructs an authentication interceptor.
func NewInterceptor(logger *logrus.Logger, userClient userpb.UserServiceClient, opts ...Option) *Interceptor {
	if logger == nil {
		logger = logrus.New()
	}

	i := &Interceptor{
		logger:     logger,
		userClient: userClient,
		allowlist:  make(map[string]struct{}),
	}

	for _, opt := range opts {
		opt(i)
	}

	return i
}

// Unary returns a unary server interceptor enforcing authentication.
func (i *Interceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, ok := i.allowlist[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		claims, err := i.authenticate(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		ctxWithClaims := ContextWithClaims(ctx, claims)
		return handler(ctxWithClaims, req)
	}
}

func (i *Interceptor) authenticate(ctx context.Context, method string) (*Claims, error) {
	start := time.Now()
	entry := i.logger.WithField("method", method)

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		entry.Warn("missing metadata on request")
		return nil, status.Error(codes.Unauthenticated, metadataRequiredMessage)
	}

	values := md.Get(authorizationMetadataKey)
	if len(values) == 0 {
		entry.Warn("authorization header missing")
		return nil, status.Error(codes.Unauthenticated, metadataRequiredMessage)
	}

	token := strings.TrimSpace(values[0])
	if token == "" {
		entry.Warn("authorization header empty")
		return nil, status.Error(codes.Unauthenticated, metadataRequiredMessage)
	}

	if strings.HasPrefix(strings.ToLower(token), bearerPrefix) {
		token = strings.TrimSpace(token[len(bearerPrefix):])
	}

	if token == "" {
		entry.Warn("bearer token empty after trimming prefix")
		return nil, status.Error(codes.Unauthenticated, metadataRequiredMessage)
	}

	resp, err := i.userClient.ValidateToken(ctx, &userpb.ValidateTokenRequest{AccessToken: token})
	if err != nil {
		entry.WithError(err).WithField("duration", time.Since(start)).Warn("token validation rpc failed")
		return nil, status.Error(codes.Unauthenticated, "invalid access token")
	}

	if !resp.GetValid() {
		entry.WithField("duration", time.Since(start)).Warn("token invalid")
		return nil, status.Error(codes.Unauthenticated, "invalid access token")
	}

	claims := &Claims{
		UserID: resp.GetUserId(),
		Email:  resp.GetEmail(),
		Role:   NormalizeRole(resp.GetRole().String()),
	}

	entry.WithFields(logrus.Fields{
		"user_id":  claims.UserID,
		"role":     claims.Role,
		"duration": time.Since(start),
	}).Debug("request authenticated")

	return claims, nil
}
