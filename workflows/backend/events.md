# events.md

## Event: WorkflowTriggered
- **Schema:**
```json
{
  "workflowId": "string",
  "timestamp": "RFC3339",
  "userId": "string"
}
```
- **Source:** API/Operator

## Event: TaskCompleted
- **Schema:**
```json
{
  "taskId": "string",
  "status": "success|error",
  "timestamp": "RFC3339"
}
```
- **Source:** Worker

## Security
- All event traffic sent encrypted via TLS
- No sensitive data in event payloads
