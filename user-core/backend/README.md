# User-Core Backend

Minimal Go service that backs Task Systems authentication and user management.

## Quick setup

1. **Bootstrap data** – copy the example seed (contains an `admin@example.com` account, password `Admin123!`).
   ```powershell
   Set-Location e:/Source/Systems/workspaces/user-core/user-core/backend
   Copy-Item ./config/bootstrap_users.example.textproto ./config/bootstrap_users.textproto -Force
   ```
2. **JWT configuration** – set the required env var (TTL values are optional; defaults are 15m access / 720h refresh, issuer `user-core`).
   ```powershell
   $env:JWT_SECRET = "super-secret-key"
   # Optional overrides:
   # $env:JWT_ACCESS_TTL = "30m"
   # $env:JWT_REFRESH_TTL = "720h"
   # $env:JWT_ISSUER = "user-core"
   ```
3. **Run the server** – BadgerDB files land in `./.data/badger` on first boot; the bootstrap file is only consumed when that directory is empty.
   ```powershell
   go run ./cmd/server --data-dir ./.data --config-dir ./config
   ```

## Smoke test with `grpcurl`

All RPCs require a bearer token; start by authenticating as the seeded admin.

```powershell
# 1) Authenticate and capture the access token
$auth = grpcurl -plaintext -d '{"identifier":"admin@example.com","password":"Admin123!"}' localhost:50051 usercore.v1.UserService/Authenticate |
    ConvertFrom-Json
$token = $auth.accessToken

# 2) Create a user (requires admin token)
grpcurl -plaintext -H "authorization: Bearer $token" -d '{
  "email": "demo@example.com",
  "name": "Demo User",
  "password": "ChangeMe123!",
  "role": "USER_ROLE_USER"
}' localhost:50051 usercore.v1.UserService/CreateUser

# 3) List users to confirm
grpcurl -plaintext -H "authorization: Bearer $token" -d '{}' localhost:50051 usercore.v1.UserService/ListUsers
```

> Tip: `grpcurl` is available via `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest`.

## How other services integrate

- gRPC endpoint: `usercore.v1.UserService` on port `50051` (override via `PORT` env var before launch).
- Common RPCs:
  - `Authenticate` / `ValidateToken` for issuing JWTs and validating claims.
  - `ValidateUser` and `BulkGetUsers` to resolve user IDs in downstream services.
  - `ListUsers` / `SearchUsers` for admin consoles.
- Services should present bearer tokens via `authorization: Bearer <jwt>` metadata; JWTs are signed with the shared `JWT_SECRET`.

## Dev utilities

```powershell
# Regenerate protobufs (run from repo root)
pwsh ./generate-proto.ps1

# Run backend tests
Set-Location e:/Source/Systems/workspaces/user-core/user-core/backend
go test ./...
```
