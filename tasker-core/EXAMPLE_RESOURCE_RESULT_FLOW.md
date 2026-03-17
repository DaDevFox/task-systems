# Example Resource/Result Completion Flow

This flow demonstrates objective-congruent completion behavior:
- API resource dependency check (`GET` + regex validation)
- File resource dependency check (local file path must exist)
- File result proof requirement check (`complete=true` + file exists)
- System settings update on user (e.g. `calendar.api_key`)

## Prerequisites

1. Start backend server:

```powershell
Set-Location tasker-core/backend
$env:AUTH_ENABLED='true'
$env:ZITADEL_ISSUER='<your-zitadel-issuer-url>'
$env:ZITADEL_AUDIENCE='<your-zitadel-audience>'
go run ./cmd/server --port 8080
```

2. Create a sample attachment and package it (compressed local store option):

```powershell
Set-Location tasker-core/backend
"proof payload" | Set-Content ./sample-proof.txt
go run ./cmd/enhanced_client_v3 -- pack-attachment ./sample-proof.txt ./attachments
```

3. Set a system setting for calendar sync:

```powershell
go run ./cmd/enhanced_client_v3 -- user setting-set <user-id-or-email> calendar.api_key demo-key
```

## Demo Flow

Run the end-to-end completion flow command:

```powershell
go run ./cmd/enhanced_client_v3 -- demo-completion-flow "Demo Objective" core <assignee-id> https://httpbin.org/get "url" ./attachments/<packed-file>.gz
```

Expected behavior:
- Task is created.
- Resource requirements are attached (API + file).
- File proof result is attached and marked complete.
- `CompleteTask` succeeds only when all checks pass.

If an API endpoint is down, regex mismatches, or file path is missing, completion is blocked and task status is set to `blocked`.

## Zitadel/User Integration Notes

- User creation and lookup RPCs are claims-aware.
- When authenticated claims are present, `CreateUser` uses claim subject as user ID and validates claim email alignment.
- `GetUser` can default to claim subject when no explicit identifier is provided.
