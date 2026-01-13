# Structured Logging Implementation

## Overview

This document describes the comprehensive structured logging implementation using logrus that has been added to all RPC handlers in the task management system.

## Philosophy

The logging follows the structured logging philosophy of minimizing natural language text and maximizing machine-readable fields:

- **Minimize Human Text**: Log messages use concise, standardized terms like `rpc_start`, `rpc_success`, `rpc_validation_failed`
- **Maximize Structured Fields**: Rich context is provided through structured fields rather than embedded in messages
- **Consistent Patterns**: All handlers follow the same logging pattern for predictable parsing and monitoring

## Logging Structure

Each RPC handler implements the following logging pattern:

### 1. Request Start
```json
{
  "level": "info",
  "msg": "rpc_start",
  "rpc": "AddTask",
  "request_id": "add_task_1704067200000000000",
  "task_name": "Example Task",
  "user_id": "user123",
  "has_desc": true,
  "timestamp": "2024-01-01T00:00:00.000Z"
}
```

### 2. Validation Failures
```json
{
  "level": "error",
  "msg": "rpc_validation_failed",
  "rpc": "AddTask",
  "request_id": "add_task_1704067200000000000",
  "validation_error": "empty_task_name",
  "timestamp": "2024-01-01T00:00:00.000Z"
}
```

### 3. Service Call Failures
```json
{
  "level": "error",
  "msg": "rpc_service_call_failed",
  "rpc": "AddTask",
  "request_id": "add_task_1704067200000000000",
  "operation": "task_service_add_task_for_user",
  "duration": "15ms",
  "error": "database connection failed",
  "timestamp": "2024-01-01T00:00:00.000Z"
}
```

### 4. Successful Completion
```json
{
  "level": "info",
  "msg": "rpc_success",
  "rpc": "AddTask",
  "request_id": "add_task_1704067200000000000",
  "task_id": "abc12345",
  "task_stage": "pending",
  "task_status": "todo",
  "duration": "25ms",
  "timestamp": "2024-01-01T00:00:00.000Z"
}
```

## Key Fields

### Standard Fields (All Handlers)
- `rpc`: Handler name (e.g., "AddTask", "StartTask", "GetUser")
- `request_id`: Unique identifier for request tracing (format: `{rpc_name}_{unix_nano}`)
- `duration`: Request processing time
- `timestamp`: ISO 8601 timestamp

### Validation Fields
- `validation_error`: Specific validation failure type
- Input validation details (e.g., `empty_task_name`, `missing_from_id`)

### Operation Fields
- `operation`: Specific service/resolver operation called
- Context-specific data (e.g., `task_count`, `user_count`, `target_ids`)

### Result Fields
- Entity IDs and key properties
- Counts and summaries
- State information (stage, status)

## RPC Handler Coverage

### Task Management
- ✅ `AddTask`: Create new tasks
- ✅ `MoveToStaging`: Move tasks to staging with destination tracking
- ✅ `StartTask`: Start task execution
- ✅ `StopTask`: Stop tasks with completion tracking
- ✅ `CompleteTask`: Mark tasks complete
- ✅ `MergeTasks`: Merge operations with source/target tracking
- ✅ `SplitTask`: Split with new task tracking
- ✅ `AdvertiseTask`: Flow operations with target tracking
- ✅ `StitchTasks`: Flow operations with source tracking

### Task Querying
- ✅ `ListTasks`: List operations with filtering context
- ✅ `GetTask`: Individual task retrieval
- ✅ `GetTaskDAG`: DAG operations with resolver updates
- ✅ `UpdateTaskTags`: Tag modification with conversion tracking

### User Management
- ✅ `CreateUser`: User creation with settings tracking
- ✅ `GetUser`: User retrieval with lookup type differentiation
- ✅ `UpdateUser`: User modification tracking

### ID Resolution
- ✅ `ResolveTaskID`: Task ID resolution with suggestion tracking
- ✅ `ResolveUserID`: User ID resolution with suggestion tracking

### Helper Operations
- ✅ `updateResolvers`: Resolver refresh with data counts

## Debugging Benefits

1. **Request Tracing**: Each request has a unique ID for end-to-end tracking
2. **Performance Monitoring**: Duration tracking for all operations
3. **Error Context**: Rich error information with operation context
4. **Validation Details**: Specific validation failure identification
5. **State Transitions**: Clear tracking of entity state changes
6. **Dependency Tracking**: Flow operations show source/target relationships

## Usage Examples

### Filter by RPC Type
```bash
docker-compose logs -f | jq 'select(.rpc == "AddTask")'
```

### Monitor Performance
```bash
docker-compose logs -f | jq 'select(.duration and (.duration | tonumber > 100))'
```

### Track Request Journey
```bash
docker-compose logs -f | jq 'select(.request_id == "add_task_1704067200000000000")'
```

### Monitor Errors
```bash
docker-compose logs -f | jq 'select(.level == "error")'
```

### Track Validation Failures
```bash
docker-compose logs -f | jq 'select(.msg == "rpc_validation_failed")'
```

## Configuration

The logging system is configured in `internal/logging/logger.go`:
- JSON format for structured output
- ISO 8601 timestamps
- Configurable log levels
- Standard output destination

The system automatically uses structured logging without requiring configuration changes to individual handlers.
