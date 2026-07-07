package middleware

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// UserQuery is the shared query contract used to test caller eligibility.
type UserQuery = proto.Message

// ParameterizedUserQuery builds a user query from the incoming RPC request.
type ParameterizedUserQuery func(ctx context.Context, request any) (UserQuery, error)

// EligibilityChecker evaluates whether a user matches a query.
type EligibilityChecker interface {
	Test(ctx context.Context, query UserQuery, userID string) (bool, error)
}

// Policy controls how a single RPC method is authorized.
type Policy struct {
	AllowUnauthed bool
	Query         UserQuery
	QueryFactory   ParameterizedUserQuery
}

func (p Policy) resolveQuery(ctx context.Context, request any) (UserQuery, error) {
	if p.AllowUnauthed {
		return nil, nil
	}

	if p.QueryFactory != nil {
		query, err := p.QueryFactory(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("build user query: %w", err)
		}

		if query == nil {
			return nil, fmt.Errorf("user query factory returned nil")
		}

		return query, nil
	}

	if p.Query == nil {
		return nil, fmt.Errorf("user query is required")
	}

	return p.Query, nil
}