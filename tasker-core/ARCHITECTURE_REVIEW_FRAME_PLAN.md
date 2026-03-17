# Tasker Objective Frame Plan

## Goal
Provide a clean, modular, and extensible Go/protobuf frame for the full `tasker-core/OBJECTIVE.md` scope, while preserving current behavior and enabling incremental implementation.

## Proto Plan
The objective frame is represented in:
- `tasker-core/proto/taskcore/v1/objective_frame.proto`

This introduces schema contracts for:
- Domains and access boundaries (`TaskDomain`)
- Factory-origin assignment policies (`TaskFactoryFrame`)
- Required/awaitable results (`TaskResultRequirement`)
- Awaitable resources (`TaskResourceDependency`)
- Trait definitions (`TraitDefinition`)
- System descriptors with hook support (`SystemDescriptor`, `HookAction`)
- Explicit veto contracts (`ActionVeto`, `HookDecision`)
- Snapshot container for architecture review (`TaskObjectiveFrame`)

## Go Frame
Service-level extensibility frame is introduced in:
- `backend/internal/service/task_systems.go`
- `backend/internal/service/task_objective_frame.go`
- `backend/internal/service/task_service_system_hooks.go`

### Extension points
- Register systems dynamically (`RegisterSystem`)
- Enumerate registered systems (`RegisteredSystems`)
- Before/after hooks for task actions
- System veto with explicit `system` and `message` via `ActionDeniedError`
- Trait index for fast lookup (`TasksByTraitValue`)

### Objective structure decomposition
The frame separates responsibilities into distinct units:
- Domains
- Factories
- Results
- Resources
- Traits
- Systems/hooks

## Immediate Refactor Delta
`TaskService` now delegates extensibility concerns to:
- `SystemRegistry` (hook dispatch and trait index)
- `ObjectiveFrame` (architecture-level definitions)

and invokes hooks in core actions:
- add task
- move to staging
- start
- stop
- complete
- update tags

## Next decomposition steps (planned)
1. Split `task_service.go` action methods into dedicated files by responsibility:
   - `task_actions_lifecycle.go`
   - `task_actions_dependency.go`
   - `task_actions_merge_split.go`
   - `task_users.go`
   - `task_calendar.go`
2. Add persistent repositories for objective frame entities (domain/factory/result/resource/trait).
3. Add gRPC endpoints to manage objective frame entities and system registration metadata.
4. Implement result/resource satisfaction engine and completion guards.
5. Implement first-class system plugins (Pomodoro, 3-cycler, worked-time/calendar policy).
