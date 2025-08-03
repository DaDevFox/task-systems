# Task Systems Restructuring Project

## Core Architecture Principles
- **Information-as-data**: Creative representations of voluminous and structured data (mind maps, timelines, etc.)
- **Simplicity despite ostensible complexity**: Subvert the "many stub" problem through pub/sub triggers and modular design
- **Pub/Sub Event-Driven**: Avoid tight coupling between services using event-driven architecture

## Three-Project System Architecture

### 1. Task-Core (Individual Task Management)
**Status**: ✅ Recently implemented  
**Purpose**: "I am work central" - Core task planning and execution

#### Key Features:
- Simple binary state: unexecuted or executed
- Quick capture with intuitive results
- Execution tracking: "doing" button (timer start), "done" button (timer end)
- Activity tracking for "hot" tasks (shouldn't stay here long)
- Rich task lifecycle: PENDING → INBOX → STAGING → ACTIVE → ARCHIVED
- Point-based work tracking and progress measurement
- Dependency management (inflows/outflows)
- gRPC API with comprehensive task operations

#### Technology Stack:
- **Backend**: Go with gRPC
- **Database**: BadgerDB (embedded key-value store)
- **Protocol**: Protobuf (already defined in `proto/task.proto`)

### 2. Inventory-Core (NEW - to be created)
**Status**: 🚧 Needs implementation  
**Purpose**: Store, update (predict), and serve inventory levels

#### Key Features:
- Rich unit information for conversions (kg ↔ lbs, liters ↔ cups, etc.)
- Inventory tracking with "consumption behavior" patterns
- Adaptive/predictive algorithm for inventory forecasting
- Unit conversion engine
- Consumption pattern analysis
- Low-stock alerts and triggers
- Historical consumption data
- ML/Statistical prediction module (potentially Python integration)

#### Technology Stack:
- **Backend**: Go with gRPC
- **Database**: BadgerDB or PostgreSQL (for time-series data)
- **Protocol**: Protobuf
- **ML Module**: Python service (optional, for advanced predictions)

#### Protobuf Structure (to be created):
```proto
service InventoryService {
  rpc AddInventoryItem(AddInventoryItemRequest) returns (AddInventoryItemResponse);
  rpc UpdateInventoryLevel(UpdateInventoryLevelRequest) returns (UpdateInventoryLevelResponse);
  rpc GetInventoryStatus(GetInventoryStatusRequest) returns (GetInventoryStatusResponse);
  rpc PredictConsumption(PredictConsumptionRequest) returns (PredictConsumptionResponse);
  rpc SetConsumptionBehavior(SetConsumptionBehaviorRequest) returns (SetConsumptionBehaviorResponse);
}
```

### 3. Home-Manager (Group Workflows & Orchestration)
**Status**: 🔄 Needs refactoring  
**Purpose**: Bridges inventory and task planning + adds group management

#### Key Features:
- Dispatches tasks based on intervals and inventory level triggers
- Group management: algorithmic task assignment to members
- Pipeline management: work stacking while tasks await completion
- Event-driven triggers from inventory and task systems
- Notification system (Gotify, NTFY, Email)
- Web dashboard for monitoring

#### Refactoring Plan:
1. **Remove direct task management** → Delegate to task-core via gRPC
2. **Extract inventory logic** → Move to inventory-core
3. **Focus on orchestration** → Event handling, group assignment, pipeline management
4. **Maintain "pile" concept** as abstraction over inventory items

#### Technology Stack:
- **Backend**: Go with gRPC
- **Frontend**: Vue.js (existing)
- **Events**: In-memory pub/sub (for now) → Future: NATS/Redis
- **Protocol**: Protobuf (refactor existing)

## Implementation Roadmap

### Phase 1: Create Inventory-Core 🆕
1. **Setup project structure**
   ```
   inventory-core/
   ├── cmd/server/main.go
   ├── internal/
   │   ├── service/inventory_service.go
   │   ├── repository/inventory_repository.go
   │   ├── domain/inventory.go
   │   └── prediction/consumption_predictor.go
   ├── proto/inventory.proto
   └── go.mod
   ```

2. **Define Protobuf schema**
   - InventoryItem (id, name, current_level, unit, max_capacity)
   - ConsumptionBehavior (pattern, rate, seasonal_factors)
   - InventoryStatus (items, low_stock_alerts, predictions)

3. **Implement core services**
   - CRUD operations for inventory items
   - Level tracking and updates
   - Basic consumption prediction
   - Unit conversion system

### Phase 2: Refactor Home-Manager 🔄
1. **Remove task management logic**
   - Delete direct task creation/manipulation
   - Replace with gRPC calls to task-core

2. **Remove inventory management logic**
   - Extract to inventory-core
   - Replace "piles" with inventory-core client calls

3. **Focus on orchestration**
   - Event-driven triggers
   - Group assignment algorithms
   - Pipeline management
   - Notification coordination

4. **Update protobuf definitions**
   - Remove task/pile definitions
   - Add orchestration-specific messages
   - Group management structures

### Phase 3: Integration & Event System 🔗
1. **Implement pub/sub event system**
   - Inventory level triggers → Home-manager
   - Task completion events → Home-manager
   - Schedule-based triggers

2. **Create service clients**
   - Home-manager → Task-core client
   - Home-manager → Inventory-core client

3. **Testing & integration**
   - End-to-end workflow testing
   - Performance optimization
   - Error handling and resilience

### Phase 4: Frontend & ML Enhancement 🎨
1. **Unified frontend** (future consideration)
   - Single dashboard accessing all three services
   - Real-time updates via WebSocket/Server-Sent Events

2. **ML prediction service** (Python module)
   - Advanced consumption forecasting
   - Seasonal pattern recognition
   - Integration with inventory-core

## Event-Driven Architecture

### Key Events:
1. **InventoryLevelChanged** (inventory-core → home-manager)
2. **TaskCompleted** (task-core → home-manager)
3. **ScheduleTrigger** (home-manager internal)
4. **GroupTaskAssigned** (home-manager → task-core)

### Pub/Sub Pattern Benefits:
- **Eliminates "many stub" problem**: New services can subscribe to events without modifying existing services
- **Loose coupling**: Services can evolve independently
- **Scalability**: Easy to add new event consumers
- **Resilience**: Services can handle events asynchronously

## Directory Structure After Refactoring

```
task-systems/
├── task-core/              # Individual task management (existing)
├── inventory-core/         # NEW: Inventory tracking & prediction
├── home-manager/           # Refactored: Group workflows & orchestration
├── shared/                 # Common protobuf definitions
└── docs/                   # Architecture documentation
```

## Development Guidelines

1. **gRPC-first**: All inter-service communication via gRPC
2. **Event-driven**: Use pub/sub for loose coupling
3. **Protobuf schemas**: Version all service interfaces
4. **Domain separation**: Clear boundaries between services
5. **Testing**: Unit tests + integration tests for each service
6. **Documentation**: Keep protobuf files well-documented

## Migration Strategy

1. **Parallel development**: Build inventory-core alongside existing system
2. **Gradual migration**: Move home-manager features incrementally
3. **Feature flags**: Toggle between old/new implementations
4. **Data migration**: Scripts to move existing data to new services
5. **Rollback plan**: Maintain ability to revert if needed

This restructuring will create a robust, scalable system that embodies the core principles while maintaining the functionality users depend on.