# NOTE

All protected methods should use the shared authorization middleware after Envoy has already authenticated the caller at the boundary.

# RECEIVE

## Config

As place-able on an RPC method, it can (optionally) take the following config vars:
`bool allowUnauthed` [NOT RECOMMENDED] allow requests that bypass the protected system boundary
`UserQuery eligible` blanket query whose results indicate eligibility to use the target method
`ParameterizedUserQuery eligibleByRequestData` query that may be derived from the request and whose results indicate eligibility to use the target method

## Function

Evaluate eligibility to request an RPC method based on the caller identity and possibly the request data itself (see config). Return an unauthorized response if not.

> main idea: this handling is written here (once), not at the start of every RPC request method. Should be convenient, but also better for refactoring later to add logging, metrics, etc

### elgibility eval

if allowUnauthed, skip

read the trusted caller identity from Envoy-provided request metadata/context

if ParameterizedUserQuery, make UserQuery ParameterizedUserQuery.run(request data)
if UserQuery, run `UserService.Test(UserQuery, calling user)`

if inelgible based on test, return gRPC status unauthorized

## Result

gRPC status unauthorized if ineligible, pass to method otherwise

# SEND

encode info required for receive, pulling form Zitadel for authenticated user info
