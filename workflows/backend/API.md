# API.md

## Endpoints

### POST /api/workflows/trigger
Triggers an automation workflow.
- **Request:**
```json
{
  "workflowId": "string",
  "params": {"any": "object"}
}
```
- **Response:**
```json
{
  "status": "queued|started|error",
  "details": "string"
}
```
- **Errors:** 400 (bad input), 401 (auth fail), 500 (internal)

## Security
- Auth: Authelia SSO JWT required
- Input: Server-side validation for all trigger params
