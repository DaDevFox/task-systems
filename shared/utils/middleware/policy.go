package middleware

import (
	"context"
	"fmt"

	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
	"google.golang.org/protobuf/proto"
)

// UserGroupQuery is the shared query contract used to test caller eligibility.
type UserGroupQuery = pb.UserGroupQuery

// ParameterizedUserGroupQuery builds a user query from the incoming RPC request.
type ParameterizedUserGroupQuery func(ctx context.Context, request any) (UserGroupQuery, error)

// EligibilityChecker evaluates whether a user matches a query.
type EligibilityChecker interface {
	Test(ctx context.Context, query UserGroupQuery, userID string) (bool, error)
}

// Policy controls how a single RPC method is authorized.
type Policy struct {
	AllowUnauthed bool
	Query         UserGroupQuery
	QueryFactory  ParameterizedUserGroupQuery
}

func (p Policy) resolveQuery(ctx context.Context, request any) (UserGroupQuery, error) {
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

