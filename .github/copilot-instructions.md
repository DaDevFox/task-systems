# INSTRUCTIONS

## Core expectations

- Follow `STYLE.md` (no `else`, early returns, structured log fields, wrap errors with context).

## Architecture cheat sheet

- Monorepo houses several services (`user-core`, `inventory-core`, `tasker-core`, `workflows`, `shared\events`). In any service `[service]/backend` is the Go service; `[service]/pkg/proto` holds generated protobuf code, `[service]/proto` has the source `.proto` files, and `[service]/frontend` contains the frontend code (usually Javascript or C#).
- Persistence: Services use BadgerDB where needed or an in-memory repository -- BadgerDB is preferred, Bolt for smaller services which need less complexity. We'd like to add SQL database support in future.
- This code is meant to run as a set of related services on a small number of machines in a self-hosted environment. There is no single point of failure, and each service is designed to be horizontally scalable. Code should be built into Docker images and run as containers with orchestration.
- Command line flags are the current standard for configuration (including discovery of other services). A move to a service like Hashicorp Consul in future is anticipated.
- Services communicate via gRPC (internal APIs) and protobuf-encoded messages. There is no REST API.
- Each service has a bootstrap process to initialize data on first run. This is usually done via a textproto file containing initial data (e.g., admin users). See "Bootstrap & configuration" below for details.

## Essential workflows

- Generate protobufs via the repo-root script only:
  ```pwsh
  pwsh ./generate-proto.ps1
  ```
- Run backend tests from `[service]/backend`:
  ```pwsh
  go test ./...
  ```
- Start the server from `[service]/backend` (ensure Go toolchain ≥1.23):
  ```pwsh
  go run ./cmd/server [more flags]
  ```

## Bootstrap & configuration

For services which require initial data (e.g., admin users, units for inventory), the following applies:

- First boot (no repository satisfying minimum requirements found) should require a textproto seed file containing data which meets minimum requirements (e.g. at least one admin user) which is loaded into the newly created DB/repository on first boot. Default location + flag is `--config-dir/` + `bootstrap_[service short name].textproto`; an example lives at `user-core/backend/config/bootstrap_users.example.textproto` with a ready bcrypt hash (`$2a$04$2hv4siT.AyzPNbr8Sz3TKOmpbq6hIl1DzaQPxTmDNRLfqROzq6iia` as is necessary for that service).
- `[service]/backend/internal/bootstrap` handles file parsing; keep conversions aligned with the latest protobuf schema (`[service]/proto/[service]/v1/bootstrap_[service short name].proto`).
- JWT settings come from env vars (`JWT_SECRET`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `JWT_ISSUER`). Calls to auth services will fail hard if these aren’t provided, so guard any new code accordingly.

## Patterns to follow

- Repository interfaces live in `internal/repository/interfaces.go`; new persistence layers must satisfy the general `[service]Repository` and be registered in `[service]_repository.go`, but we still follow a dependency injection pattern.
- Services (`internal/service`) accept repositories + loggers; never instantiate repositories directly inside services—inject dependencies through constructors.
- gRPC layer resides in `internal/grpc`; reuse conversion helpers in `conversions.go` to map domain ↔ proto.
- Tests should stick to table-driven style or switch statements for assertions (see `user-core/internal/repository/memory_repository_test.go`). Prefer `repository.NewInMemory[service]Repository()` for unit tests.

## When editing proto schemas

- Update the textproto bootstrapping schema alongside `*.proto` files it uses constructs from.
- After editing `.proto` files, rerun `generate-proto.ps1` and commit both proto changes and regenerated Go code (unless ignored). Keep textproto examples in sync to prevent bootstrap regressions.

## Review reminders

- Surface any cross-service impact (shared proto changes affect other backends; call out follow-up tasks).
- Ensure new commands/scripts work on Windows PowerShell (`pwsh`) since that’s the default shell.
- Document non-obvious setup steps (e.g., new required flags/env vars) in the relevant README if the change alters day-one developer experience.
