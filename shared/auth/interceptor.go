package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type claimsContextKeyType string

const claimsContextKey claimsContextKeyType = "zitadel_claims"

// Claims contains normalized token values used by service handlers.
type Claims struct {
	Subject   string
	Email     string
	Role      string
	Audience  []string
	ExpiresAt time.Time
	Raw       map[string]interface{}
}

// TokenVerifier validates Zitadel JWTs via OIDC discovery + JWKS.
type TokenVerifier struct {
	issuer           string
	allowedAudiences map[string]struct{}
	verifier         *oidc.IDTokenVerifier
}

// Interceptor validates Bearer tokens for gRPC unary calls.
type Interceptor struct {
	enabled       bool
	logger        *logrus.Logger
	verifier      *TokenVerifier
	publicMethods map[string]struct{}
}

func NewTokenVerifier(ctx context.Context, issuer string, audiences []string) (*TokenVerifier, error) {
	trimmedIssuer := strings.TrimSpace(issuer)
	if trimmedIssuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}

	allowed := map[string]struct{}{}
	for _, aud := range audiences {
		trimmed := strings.TrimSpace(aud)
		if trimmed == "" {
			continue
		}

		allowed[trimmed] = struct{}{}
	}

	if len(allowed) == 0 {
		return nil, fmt.Errorf("at least one audience is required")
	}

	provider, err := oidc.NewProvider(ctx, trimmedIssuer)
	if err != nil {
		return nil, fmt.Errorf("initialize oidc provider failed: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
	return &TokenVerifier{
		issuer:           trimmedIssuer,
		allowedAudiences: allowed,
		verifier:         verifier,
	}, nil
}

func (v *TokenVerifier) Verify(ctx context.Context, token string) (*Claims, error) {
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		return nil, fmt.Errorf("token is required")
	}

	idToken, err := v.verifier.Verify(ctx, trimmedToken)
	if err != nil {
		return nil, fmt.Errorf("verify token failed: %w", err)
	}

	rawClaims := map[string]interface{}{}
	claimsErr := idToken.Claims(&rawClaims)
	if claimsErr != nil {
		return nil, fmt.Errorf("decode claims failed: %w", claimsErr)
	}

	audience := extractAudience(rawClaims)
	if !v.isAudienceAllowed(audience) {
		return nil, fmt.Errorf("token audience is not allowed")
	}

	claims := &Claims{
		Subject:   extractStringClaim(rawClaims, "sub"),
		Email:     extractStringClaim(rawClaims, "email"),
		Role:      extractRole(rawClaims),
		Audience:  audience,
		ExpiresAt: idToken.Expiry,
		Raw:       rawClaims,
	}

	return claims, nil
}

func (v *TokenVerifier) isAudienceAllowed(audience []string) bool {
	for _, aud := range audience {
		_, ok := v.allowedAudiences[aud]
		if ok {
			return true
		}
	}

	return false
}

func NewInterceptorFromEnv(ctx context.Context, logger *logrus.Logger, defaultPublicMethods []string) (*Interceptor, error) {
	if logger == nil {
		logger = logrus.New()
	}

	enabled := parseBoolEnv("AUTH_ENABLED", true)
	if !enabled {
		logger.WithField("auth_enabled", false).Warn("auth interceptor disabled via environment")
		return &Interceptor{
			enabled:       false,
			logger:        logger,
			publicMethods: buildPublicMethods(defaultPublicMethods),
		}, nil
	}

	issuer := os.Getenv("ZITADEL_ISSUER")
	audiences := splitCSV(os.Getenv("ZITADEL_AUDIENCE"))
	verifier, err := NewTokenVerifier(ctx, issuer, audiences)
	if err != nil {
		return nil, fmt.Errorf("create token verifier failed: %w", err)
	}

	publicMethods := append(defaultPublicMethods, splitCSV(os.Getenv("AUTH_PUBLIC_METHODS"))...)

	return &Interceptor{
		enabled:       true,
		logger:        logger,
		verifier:      verifier,
		publicMethods: buildPublicMethods(publicMethods),
	}, nil
}

func DefaultPublicMethods() []string {
	return []string{
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
	}
}

func DefaultUserPublicMethods() []string {
	return append(
		DefaultPublicMethods(),
		"/usercore.v1.UserService/Authenticate",
		"/usercore.v1.UserService/RefreshToken",
		"/usercore.v1.UserService/ValidateToken",
	)
}

func (i *Interceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !i.enabled {
			return handler(ctx, req)
		}

		if i.isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		token, err := bearerTokenFromIncomingContext(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "authorization token is required")
		}

		claims, verifyErr := i.verifier.Verify(ctx, token)
		if verifyErr != nil {
			i.logger.WithError(verifyErr).WithField("full_method", info.FullMethod).Warn("token verification failed")
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		nextCtx := WithClaims(ctx, claims)
		return handler(nextCtx, req)
	}
}

func (i *Interceptor) Verifier() *TokenVerifier {
	return i.verifier
}

func (i *Interceptor) isPublicMethod(method string) bool {
	_, ok := i.publicMethods[method]
	return ok
}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	if !ok {
		return nil, false
	}

	return claims, true
}

func bearerTokenFromIncomingContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", fmt.Errorf("authorization metadata is missing")
	}

	raw := strings.TrimSpace(values[0])
	if raw == "" {
		return "", fmt.Errorf("authorization metadata is empty")
	}

	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("authorization metadata is malformed")
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("authorization scheme is unsupported")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("bearer token is empty")
	}

	return token, nil
}

func buildPublicMethods(methods []string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, method := range methods {
		trimmed := strings.TrimSpace(method)
		if trimmed == "" {
			continue
		}

		result[trimmed] = struct{}{}
	}

	return result
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}

	if raw == "1" || raw == "true" || raw == "yes" || raw == "on" {
		return true
	}

	if raw == "0" || raw == "false" || raw == "no" || raw == "off" {
		return false
	}

	return fallback
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		result = append(result, trimmed)
	}

	return result
}

func extractAudience(claims map[string]interface{}) []string {
	audValue, ok := claims["aud"]
	if !ok {
		return []string{}
	}

	single, ok := audValue.(string)
	if ok {
		return []string{single}
	}

	arrayValues, ok := audValue.([]interface{})
	if !ok {
		return []string{}
	}

	audience := make([]string, 0, len(arrayValues))
	for _, item := range arrayValues {
		value, castOK := item.(string)
		if !castOK {
			continue
		}

		audience = append(audience, value)
	}

	return audience
}

func extractStringClaim(claims map[string]interface{}, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return text
}

func extractRole(claims map[string]interface{}) string {
	directRole := extractStringClaim(claims, "role")
	if directRole != "" {
		return directRole
	}

	rolesValue, ok := claims["urn:zitadel:iam:org:project:roles"]
	if !ok {
		return ""
	}

	rolesMap, ok := rolesValue.(map[string]interface{})
	if !ok {
		return ""
	}

	for roleName := range rolesMap {
		return roleName
	}

	return ""
}
