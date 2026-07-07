**NOTE**: all methods (excepting direct dependencies of this middleware itself) SHOULD use this

# RECEIVE

## Config

As place-able on an RPC method, it can (optionally) take the following config vars:
`bool allowUnauthed` [NOT RECOMMENDED] allow completely unauthenticated requests (not from a user in the protected service boundary)
`UserQuery elgible` blanket query whose results indicate elgibility to use the target method
`ParameterizedUserQuery elgibleByRequestData` query who may be modified by the request and whose results indicate elgibility to use the target method

## Function

Evaluate elgibility to request an RPC method based on the user sending/possibly the request data itself (see config). Return an unauthorized response if not.

> main idea: this handling is written here (once), not at the start of every RPC request method. Should be convenient, but also better for refactoring later to add logging, metrics, etc

### elgibility eval

if allowUnauthed, skip

read the user from the request header into `caling user`

if ParameterizedUserQuery, make UserQuery ParameterizedUserQuery.run(request data)
if UserQuery, run `UserService.Test(UserQuery, calling user)`

if inelgible based on test, return gRPC status unauthorized

## Result

gRPC status unauthorized if inelgible, pass to method otherwise

# SEND
