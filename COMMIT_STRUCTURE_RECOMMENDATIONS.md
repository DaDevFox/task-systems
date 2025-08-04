# Commit Structure Recommendations

Based on the current codebase analysis, here are recommendations for structuring commits to ensure single responsibility and better code organization.

## Current State Analysis

The codebase has successfully implemented:
- Backend folder restructuring for tasker-core and inventory-core
- Orchestration layer in home-manager
- gRPC service clients
- Event-driven architecture foundations
- Protocol buffer configurations

## Recommended Commit Structure

### 1. Fix: Clean up main.go duplicate code (Lines: ~15)
**Current Issue**: `home-manager/backend/main.go` had duplicate package declarations and imports from incomplete merge.

**What was fixed**:
- Lines 1-89: Removed duplicate legacy main function and imports
- Kept the updated orchestration-based main function

**Single Responsibility**: Code cleanup and deduplication

### 2. Refactor: Extract gRPC server setup (Lines: ~40-50)
**Potential Split**: The `startGRPCServer` and `serveGRPCAndHTTP` functions in main.go could be extracted:

```go
// New file: home-tasker/server/grpc_server.go
package server

type GRPCServerManager struct {
    port     string
    state    *pb.SystemState
    grpcServer *grpc.Server
}

func NewGRPCServerManager(port string, state *pb.SystemState) *GRPCServerManager
func (g *GRPCServerManager) Start() error
func (g *GRPCServerManager) Stop() error
```

**Lines that could be moved**:
- Lines 112-130: `startGRPCServer` function
- Lines 132-150: `serveGRPCAndHTTP` function
- Related imports

**Benefits**: 
- Separates server management from main application logic
- Makes testing easier
- Follows single responsibility principle

### 3. Enhance: Add error handling and retry logic to service clients (Lines: ~30-40)
**Current Gap**: Service clients need better error handling and connection management.

**Recommended additions to `clients/service_clients.go`**:
```go
// Lines 20-30: Add connection retry logic
func NewInventoryClientWithRetry(address string, maxRetries int) (*InventoryClient, error)

// Lines 60-70: Add circuit breaker pattern
type CircuitBreaker struct {
    maxFailures int
    timeout     time.Duration
    failures    int
    lastFailure time.Time
}

// Lines 80-90: Add health checking
func (c *InventoryClient) HealthCheck(ctx context.Context) error
```

### 4. Refactor: Extract configuration management (Lines: ~25-35)
**Current Issue**: Configuration and environment variable handling is scattered.

**Recommended new file**: `home-tasker/config/service_config.go`
```go
type ServiceConfig struct {
    InventoryServiceAddr string
    TaskServiceAddr      string
    GRPCPort            string
    HTTPPort            string
}

func LoadServiceConfig() *ServiceConfig
func (c *ServiceConfig) Validate() error
```

**Lines to move from main.go**:
- Lines 47-48: `getEnvOrDefault` function
- Lines 43-44: Service address configuration

### 5. Add: Enhanced logging and metrics (Lines: ~50-60)
**Missing Feature**: Structured logging and metrics collection for orchestration.

**Recommended additions**:
```go
// New file: home-tasker/observability/metrics.go
type OrchestrationMetrics struct {
    taskCompletions    prometheus.Counter
    inventoryUpdates   prometheus.Counter
    orchestrationErrors prometheus.Counter
}

// New file: home-tasker/observability/logger.go
func NewStructuredLogger(service string) *logrus.Logger
func LogOrchestrationEvent(logger *logrus.Logger, event, taskID, userID string)
```

### 6. Enhance: Event handling with persistence (Lines: ~40-50)
**Current Gap**: Event handling is in-memory only.

**Recommended enhancements to `orchestration/event_handler.go`**:
```go
// Lines 30-40: Add event persistence
type EventStore interface {
    StoreEvent(event *eventspb.Event) error
    GetEvents(since time.Time) ([]*eventspb.Event, error)
}

// Lines 50-60: Add event replay capability
func (h *EventHandler) ReplayEvents(since time.Time) error

// Lines 70-80: Add dead letter queue for failed events
func (h *EventHandler) HandleFailedEvent(event *eventspb.Event, err error)
```

### 7. Add: Integration tests for orchestration (Lines: ~100-120)
**Missing**: End-to-end integration tests for the orchestration layer.

**Recommended new file**: `home-tasker/test/integration/orchestration_integration_test.go`
```go
func TestOrchestrationTaskToInventoryFlow(t *testing.T)
func TestInventoryLevelChangeCreatesTask(t *testing.T)
func TestServiceFailureHandling(t *testing.T)
```

## Buildability Requirements

Each commit should:
1. **Build successfully**: `go build` should pass
2. **Pass existing tests**: `go test ./...` should pass
3. **Maintain API compatibility**: No breaking changes to existing interfaces
4. **Include necessary imports**: All dependencies properly declared

## Exceptions for Multi-Commit Features

Some features may require multiple commits that are only complete when combined:

### Example: Circuit Breaker Implementation
1. **Commit 1**: Add circuit breaker interface and basic implementation
2. **Commit 2**: Integrate circuit breaker into inventory client  
3. **Commit 3**: Add circuit breaker configuration and metrics

**Note**: Commit 1 might not be fully functional but establishes the foundation.

## Current Code Quality Assessment

### Strengths
- ✅ Good separation of concerns between clients, orchestration, and main
- ✅ Proper error handling in most functions
- ✅ Comprehensive test coverage for individual components
- ✅ Clean gRPC client implementations

### Areas for Improvement
- ⚠️ Main function is doing too many things (80+ lines)
- ⚠️ Configuration scattered across files
- ⚠️ Limited observability (metrics, structured logging)
- ⚠️ No graceful degradation when services are unavailable
- ⚠️ Event handling is basic (no persistence, replay, or dead letter queue)

## Recommended Next Actions

1. **Immediate** (< 20 lines each):
   - Extract environment variable handling to config package
   - Add health check endpoints to service clients
   - Add structured logging to orchestration events

2. **Short-term** (20-50 lines each):
   - Extract gRPC server management to separate package
   - Add retry logic and circuit breakers to service clients
   - Implement event persistence layer

3. **Medium-term** (50-100 lines each):
   - Add comprehensive metrics and observability
   - Implement event replay and recovery mechanisms
   - Add integration tests for cross-service scenarios

Each of these can be implemented as focused commits that build and test successfully.
