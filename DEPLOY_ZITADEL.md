# Zitadel Deployment (Unified gRPC Auth)

This repository now uses a shared gRPC auth interceptor in `shared/auth` and thin wrappers in:

- `user-core/backend/internal/auth`
- `inventory-core/backend/internal/auth`
- `tasker-core/backend/internal/auth`

All three backends validate incoming Bearer tokens against Zitadel OIDC/JWKS.

## Environment contract

Each backend uses the same auth environment variables:

- `AUTH_ENABLED` (default: `true`)
- `ZITADEL_ISSUER` (for compose: `http://zitadel:8080`)
- `ZITADEL_AUDIENCE` (comma-separated allowed audiences)
- `AUTH_PUBLIC_METHODS` (optional extra public gRPC methods)

`user-core` keeps legacy JWT issuance RPCs for compatibility (`Authenticate`, `RefreshToken`).
For local compatibility, it still expects `JWT_SECRET` and related JWT env vars.

## Full stack compose

Root compose file: `docker-compose.yml`

Includes:

- `postgres` for Zitadel
- `zitadel`
- `user-core`
- `inventory-core`
- `tasker-core`

Run:

```pwsh
docker compose up --build
```

## Optional nginx gRPC edge

Nginx config: `deploy/nginx/grpc.conf`

Run with edge profile:

```pwsh
docker compose --profile edge up --build
```

This exposes one gRPC endpoint on `localhost:50050` and routes by gRPC service path:

- `/usercore.v1.UserService/` -> `user-core:50051`
- `/inventory.v1.InventoryService/` -> `inventory-core:50052`
- `/taskcore.v1.TaskService/` -> `tasker-core:50053`

## Notes

- Reflection methods are public by default.
- `user-core` marks `Authenticate`, `RefreshToken`, and `ValidateToken` as public by default.
- For tests that do not provision Zitadel, set `AUTH_ENABLED=false`.
