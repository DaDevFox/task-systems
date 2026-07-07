# Structure

All methods use the [Standard Middleware](./shared/utils/middleware/OBJECTIVE.md) with allowUnauthed=false by default

# User Service

Basic:

`User GetByEmail(string)`
`User GetById(string)`
`User Resolve(string)` returns any of the above that succeed (in order)
`bool Update(User)`

`streaming User ResolveStreaming(streaming string)`

Groups:

`bool GetMembership(User, string group)`
`Mode GetPrivilegeMode(User, string group)`
`bool GetPrivilege(User, string group, Mode)`

@Middleware(query: ParameterizedByGroup(IsOwner(User, group)))
`void SetPrivilege(User, string group, Mode)`
@Middleware(query: ParameterizedByGroup(IsOwnerOrAdmin(User, group)))
`void SetMember(User, string group, bool)`

> API/proto thing: `UserQuery` is `Specification`, which is `string group` + `Mode?` (none means just member is fine), joined together with `Not` (unary), `And`, `Or` (binary) operations.

`bool List(UserQuery)` run a query, return the output
`bool Test(UserQuery, User)` run a query, understanding the goal is only to check if a user is included in the result (enables optimizations?)

# Decisions

no setting validation on the consuming service side: lots of overhead (more anti-DI/coupled multichanges required to add new services) for a small use case

baggage is imported by the user service in the proto, defined on consuming service side: anti-DI/coupled multichange requirement, yes, but the only one.

audit trails is V2 (see [OBJECTIVES.md])

# Implementation

## V1

`List` runs a full query, `Test` returns early if the user is found, basic DFS traversal (NOT BFS, leaf nodes must be people, thus we should prioritize seeing those earlier for the `Test` speedup from an early return)

everything else is basic CRUD

## Decisions

database options:
[akrylysov/pogreb](https://github.com/akrylysov/pogreb) - slower insert (fine for this app specifically) reaping huge read speed reward
to eval: boltdb, levelsdb(, tiledb???)
badgerdb (ofc/standard) - indexes keys only, values stored in contiguous append-only log (fast insert/read/delete isolated), batch processing ok, relational ops slow??? unclear on that, pending investigation
