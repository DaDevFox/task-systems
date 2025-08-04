# Home Manager Orchestration Integration

## Overview

This document describes the integration of the orchestration layer into the home-manager backend, enabling event-driven coordination between inventory-core and task-core services.

## Architecture

### Service Coordination

```
┌─────────────────┐    Events    ┌──────────────────┐
│  inventory-core │ ──────────► │   home-manager   │
│    Service      │              │  Orchestration   │
└─────────────────┘              │     Layer        │
                                 └──────────────────┘
┌─────────────────┐    Events               │
│   task-core     │ ◄──────────────────────┘
│    Service      │
└─────────────────┘
```

### Event Flow Examples

1. **Inventory Low Stock → Task Creation**
   ```
   inventory-core: Item level drops below threshold
   → INVENTORY_LEVEL_CHANGED event
   → home-manager orchestration: ProcessInventoryLevelChange()
   → task-core: Create restock task
   ```

2. **Task Completion → Inventory Update**
   ```
   task-core: Task marked as complete
   → TASK_COMPLETED event  
   → home-manager orchestration: ProcessTaskCompletion()
   → inventory-core: Update consumption levels
   ```

3. **Scheduled Inventory Checks**
   ```
   Schedule trigger: Daily inventory check
   → SCHEDULE_TRIGGER event
   → home-manager orchestration: handleScheduledInventoryCheck()
   → Report on low stock and empty items
   ```

## Components

### OrchestrationService (`orchestration/orchestration_service.go`)

**Purpose**: Coordinates between inventory-core and task-core services.

**Key Methods**:
- `ProcessTaskCompletion()`: Handles task completion and potential inventory updates
- `ProcessInventoryLevelChange()`: Creates restocking tasks when items are low
- `Close()`: Graceful cleanup of service connections

### EventHandler (`orchestration/event_handler.go`)

**Purpose**: Processes events from external services and triggers appropriate actions.

**Event Types Handled**:
- `INVENTORY_LEVEL_CHANGED`: Inventory levels changed
- `TASK_COMPLETED`: Tasks marked as complete
- `TASK_ASSIGNED`: Tasks assigned to users
- `SCHEDULE_TRIGGER`: Scheduled operations trigger

### Service Clients (`clients/service_clients.go`)

**Purpose**: gRPC clients for communicating with external services.

**Clients**:
- `InventoryClient`: Inventory operations (get status, update levels)
- `TaskClient`: Task operations (create, complete, get tasks)

## Configuration

### Environment Variables

- `INVENTORY_SERVICE_ADDR`: Address of inventory-core service (default: `localhost:50053`)
- `TASK_SERVICE_ADDR`: Address of task-core service (default: `localhost:50054`)

### Service Addresses

The orchestration layer automatically connects to the configured service addresses. If connection fails, the system continues in legacy mode with warnings logged.

## Integration Points

### Main Application (`main.go`)

The main application now:

1. **Initializes Event Bus**: Creates shared event bus for inter-service communication
2. **Sets up Orchestration**: Creates orchestration service with service clients
3. **Subscribes to Events**: Registers event handlers for relevant event types
4. **Graceful Shutdown**: Properly closes all connections on shutdown

### Legacy Engine Integration

The existing engine continues to run alongside the orchestration layer during the transition period. This allows for:

- **Gradual Migration**: Move functionality piece by piece
- **Fallback Capability**: Continue operation if external services are unavailable
- **Testing**: Validate new orchestration against existing behavior

## Usage Examples

### Starting the System

```bash
# Set service addresses (optional)
export INVENTORY_SERVICE_ADDR="localhost:50053"
export TASK_SERVICE_ADDR="localhost:50054"

# Start home-manager
go run main.go
```

### Event Publishing

```go
// Example: Publishing an inventory level change event
eventBus.PublishInventoryLevelChanged(
    ctx,
    "item-123",
    "Milk",
    2.5,  // previous level
    1.0,  // new level
    "L",  // unit
    2.0,  // threshold
)
```

### Service Operations

```go
// Example: Using orchestration service directly
err := orchestrationSvc.ProcessTaskCompletion(ctx, "task-456", "user-789")
if err != nil {
    log.WithError(err).Error("failed to process task completion")
}
```

## Testing

### Unit Tests

- `orchestration/orchestration_test.go`: Tests orchestration service creation and basic functionality
- `clients/service_clients_test.go`: Tests service client creation and connection handling
- `main_test.go`: Tests main application integration points

### Integration Tests

Run all tests:
```bash
go test ./... -v
```

### Manual Testing

1. **Start External Services**: Run inventory-core and task-core services
2. **Start Home Manager**: Run the home-manager with orchestration enabled
3. **Trigger Events**: Perform actions that generate events (create tasks, update inventory)
4. **Verify Coordination**: Check logs for event processing and cross-service operations

## Monitoring and Logging

### Log Levels

- `INFO`: Normal operation events (service startup, event processing)
- `WARN`: Non-fatal issues (service connection failures, continuing in legacy mode)
- `ERROR`: Serious issues (event processing failures, service errors)

### Key Log Fields

- `task_id`, `user_id`: Task-related operations
- `item_id`, `item_name`, `level`: Inventory-related operations
- `event_type`, `source_service`: Event processing
- `orchestration_service`: Service coordination activities

## Next Steps

1. **Replace Legacy Engine**: Gradually move engine functionality to orchestration
2. **Add More Event Types**: Support additional coordination scenarios
3. **Frontend Integration**: Expose orchestration status through REST APIs
4. **Real-time Dashboards**: Show live coordination status and metrics
5. **Enhanced Error Handling**: Add retry logic and circuit breakers for service calls

## Migration Strategy

### Phase 1: ✅ Parallel Operation
- Orchestration runs alongside existing engine
- Event handlers process coordination logic
- Legacy engine continues core functionality

### Phase 2: Gradual Replacement
- Move specific engine triggers to orchestration
- Replace direct pile/task logic with service calls
- Maintain backward compatibility

### Phase 3: Full Migration
- Remove legacy engine components
- Pure event-driven orchestration
- Complete separation of concerns

## Troubleshooting

### Common Issues

1. **Service Connection Failures**
   - Check service addresses in environment variables
   - Verify external services are running
   - Review network connectivity

2. **Event Processing Errors**
   - Check event payload format
   - Verify service client authentication
   - Review orchestration service logs

3. **Legacy Engine Conflicts**
   - Monitor for duplicate operations
   - Check for race conditions
   - Review transition logic

### Debug Tips

- Enable debug logging: `log.SetLevel(log.DebugLevel)`
- Use environment variables to control service addresses
- Test with mock services for isolated testing
