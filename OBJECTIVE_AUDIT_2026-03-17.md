# Objective and Style Audit (2026-03-17)

## Scope
- STYLE.md alignment across service implementations
- OBJECTIVE.md minimum implementation alignment across services
- Architecture consistency check (Go + Protobuf + gRPC for non-frontend projects)

## Service Status

### user-core
- Status: PARTIAL PASS
- Architecture: PASS (Go backend + protobuf + gRPC present)
- Objective alignment: PARTIAL (core auth/group/user semantics present; usability/dashboard obligations still mostly frontend-dependent)
- Style gaps: known `else` usage and mixed logging patterns in some paths

### inventory-core
- Status: PARTIAL PASS
- Architecture: PASS (Go backend + protobuf + gRPC present)
- Objective alignment: PARTIAL (core store/resource/unit behavior exists; full reporting/trigger UX remains incomplete)
- Style gaps: known `else` usage in backend and mixed log style in some repositories

### tasker-core
- Status: IMPROVED / PARTIAL PASS
- Architecture: PASS (Go backend + protobuf + gRPC present)
- Objective alignment: PARTIAL->IMPROVED (CLI maturity improved; new TUI dashboard command added with table/hierarchy/timeline/dag views)
- Style gaps reduced in key paths (server bootstrap logging and core task service logging improved)
- Remaining gaps: broad semantic objective areas still incomplete (domains/results/resources/systems), no full graphical frontend yet

### workflows
- Status: PARTIAL PASS
- Architecture: PASS (Go backend + protobuf + gRPC present)
- Objective alignment: PARTIAL (orchestration intent represented; objective is high-level)

### shared
- Status: PASS
- Architecture: PASS (shared Go modules + protobuf contracts)
- Objective alignment: PARTIAL PASS (contracts/utilities exist; pruning/doc quality should be continually reviewed)

### home-manager (backend)
- Status: FAIL (incomplete implementation)
- Architecture: FAIL CURRENTLY (backend objective exists but no substantive Go/gRPC implementation found)
- Recommendation: explicitly mark as planned/inactive or scaffold minimal Go + protobuf + gRPC backend to satisfy monorepo baseline

## Concrete Remediation Applied In This Pass
- tasker-core/backend/cmd/enhanced_client_v3/main.go
  - Added `tui` command with objective-aligned terminal views:
    - table (A)
    - hierarchy (B)
    - timeline (C)
    - DAG (D)
  - Replaced placeholder task selection with fuzzy interactive selector
  - Added resolver-safe current user selection without else/else-if for init path

- tasker-core/backend/internal/service/task_service.go
  - Replaced printf-based operational errors with structured logger fields
  - Refactored destination/location branch to avoid else-if chain
  - Removed unreachable duplicate update block in StartTask

- tasker-core/backend/cmd/server/main.go
  - Removed stdlib log usage in favor of structured logrus fields and errors

- generate-proto.ps1
  - Fixed protobuf generation path handling to pass absolute output directories to `protoc` while running from service proto directories
  - Verified repo-root protobuf generation now succeeds for tasker-core, inventory-core, user-core, shared, and workflows

- user-core/backend/internal/bootstrap/bootstrap.go
- user-core/backend/internal/grpc/conversions.go
- user-core/backend/internal/bootstrap/bootstrap_test.go
  - Repaired malformed protobuf imports and restored valid Go syntax

- workflows/backend/clients/service_clients.go
  - Corrected tasker protobuf import path to backend-generated package

- inventory-core/backend/go.mod
- workflows/backend/go.mod
  - Added/adjusted local replace directives to resolve local modules during workspace builds

## Verification Snapshot
- tasker-core/backend: PASS (`go test ./...`)
- user-core: PASS after import repair (`go test ./...`)
- shared: PASS (`go test ./...`)
- inventory-core/backend: FAIL (pre-existing corruption in shared event bus file + malformed integration test source)
- workflows/backend: FAIL (pre-existing compile/test drift in state/server/engine packages)

## Blocking Findings Requiring Separate Repair Scope
1. `shared/events/eventbus.go` currently starts with a free function and no `package` declaration, causing downstream build failures.
2. `inventory-core/backend/test/integration/unit_management_test.go` is structurally corrupted (duplicated/garbled tokens).
3. `workflows/backend/state/loader.go` references missing identifiers/imports (`filepath`, `fmt`, `securePath`) and multiple workflows packages have API/schema drift (`TaskHistory` field references and test signature mismatches).

## Remaining Work (High Priority)
1. Continue STYLE normalization in tasker-core backend (`else` removal, structured logging in remaining files).
2. Decide lifecycle for `enhanced_client_v2` (remove, mark legacy, or complete).
3. Define and implement minimal home-manager backend baseline (Go + protobuf + gRPC) or formal de-scope.
4. Build tasker frontend/dashboard if objective requires non-terminal experience beyond CLI/TUI.
5. Add objective coverage tests/checklists to avoid drift.
