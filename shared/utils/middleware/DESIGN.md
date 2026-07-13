# Structure

All services use the shared middleware layer for authorization after the request has already been authenticated at the system boundary.

Envoy is the protected system boundary for the monorepo services. It is responsible for external traffic handling, TLS termination, authentication integration with Zitadel, and translating authenticated external requests into internal gRPC calls.

# Boundary

## Envoy

Envoy sits in front of all protected services and is the only component that deals with outside callers directly.

Its responsibilities are:

- authenticate the caller with Zitadel-backed identity information
- reject unauthenticated traffic before it reaches any service
- normalize the authenticated identity into trusted request metadata for internal gRPC calls
- preserve the caller identity and any request-scoped claims needed by downstream services

## Shared Middleware

The shared middleware package does not perform authentication.

It assumes the request has already crossed the protected boundary and that Envoy has attached trusted identity information to the request context or metadata.

Its responsibilities are:

- extract the caller identity from the internal request context
- evaluate whether the caller is eligible to invoke the target RPC
- support a blanket query for method-level authorization
- support a parameterized query factory for request-aware authorization
- return gRPC `PermissionDenied` when the caller is authenticated but not eligible
- return gRPC `Unauthenticated` only when trusted identity is missing after the boundary

# User Service

The user service is the canonical implementation of the query model used by other services.

It provides the query evaluation mechanism used by shared middleware, including:

- `UserGroupQuery` for fixed eligibility rules
- `ParameterizedUserGroupQuery` for request-derived eligibility rules
- `Test(UserGroupQuery, User)` semantics that return a boolean eligibility result

The middleware must not duplicate query evaluation logic. It should only orchestrate request-level policy and delegate the actual user eligibility evaluation to the user service.

# Request Flow

1. External request arrives at Envoy.
2. Envoy authenticates the caller with Zitadel.
3. Envoy forwards the authenticated request to an internal gRPC service and attaches trusted identity metadata.
4. The shared middleware reads the authenticated user identity from the request context or metadata.
5. The shared middleware resolves the configured `UserGroupQuery` or `ParameterizedUserGroupQuery` for the RPC method.
6. The shared middleware calls the user-service eligibility check.
7. The RPC handler runs only if the caller is authenticated and eligible.

# Config

As placeable on an RPC method, the shared middleware can optionally take:

`bool allowUnauthed` [NOT RECOMMENDED] allow requests that bypass the protected system boundary

`UserGroupQuery eligible` blanket query whose results indicate eligibility to use the target method

`ParameterizedUserGroupQuery eligibleByRequestData` query that may be derived from the incoming request and whose results indicate eligibility to use the target method

`string principalMetadataKey` optional override for where Envoy places the authenticated identity in internal request metadata

# Decisions

authentication stays outside the shared middleware and outside service code

JWT validation, session handling, redirects, and login flows remain Zitadel responsibilities at the boundary

Envoy is the only component allowed to translate external authenticated identity into trusted internal identity

the shared middleware only enforces authorization after identity has already been established

per-RPC policy should be configured declaratively so services can describe eligibility without reimplementing the same checks inline

# Implementation

## V1

Envoy terminates external traffic and forwards trusted caller identity to internal services.

The shared middleware provides a unary interceptor and stream interceptor that both use the same policy evaluation path.

Authorization is method-scoped and supports both:

- fixed query policies for straightforward membership checks
- request-derived query policies for operations where the target user or resource comes from the RPC input

## Notes

This design intentionally avoids importing Zitadel authentication middleware into internal services.

That middleware is HTTP-oriented and should remain at the edge, where redirects and browser-based login flows make sense.

If a service needs authenticated identity, it should receive that identity from Envoy in trusted internal metadata rather than performing its own external authentication.
