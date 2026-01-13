# Task Management System

A production-ready task management system with user-partitioned task organization, DAG visualization, and staging workflows.

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PowerShell (for Windows deployment scripts)

### Deploy with Docker
```powershell
# Clone and deploy
git clone <repository-url>
cd tasker-core
.\deploy.ps1
```

### Manual Build
```powershell
# Generate protobuf code
buf generate

# Build binaries
go build -o bin/server.exe ./cmd/server
go build -o bin/client.exe ./cmd/client

# Run server
.\bin\server.exe --port 8080

# Use client
.\bin\client.exe --server localhost:8080 --help
```

## 📋 System Overview

The system manages tasks through a structured 4-stage workflow:

### Stage 0: Pending
- Tasks created via endpoints/CLI land here by default
- Transitional stage, tasks move to inbox automatically

### Stage 1: Inbox  
- **Maximum 5 tasks** - operations blocked if exceeded
- Tasks have **no location information** by design
- Cannot be geo-tagged, resource-tagged, or assigned
- Default stage for `list` command

### Stage 2: Staging Area
- Tasks become "flowlets" with dependencies
- **Inflow/Outflow relationships** (predecessor/successor tasks)
- **Location required** - inherited from dependencies or explicitly set
- **Points system** for tracking sub-work
- Supports hierarchical location tagging

### Stage 3: Active Work
- Subset of staging tasks selected for execution
- **Scheduling information** (start time, due dates)
- **Work intervals** tracking

## 🔧 Core Features

### Task ID Resolution
- **Minimum unique prefixes** - use `a1b2` instead of full UUIDs
- **User-partitioned** - each user has independent ID space
- **Fuzzy matching** in CLI for task selection

### User Management
- **Email-based lookup** - find users by email or ID
- **User context** - all operations scoped to specific users
- **Independent task spaces** per user

### Staging & Dependencies
- **Location inheritance** - tasks inherit location from dependencies
- **Dependency chains** - A → B → C relationships
- **Location parsing** - `project/backend/api` becomes `["project", "backend", "api"]`
- **Fuzzy picker** - interactive task selection or manual location entry

### DAG Visualization
- **Topological sorting** - dependency-aware task ordering
- **Minimum prefix highlighting** - visual distinction of short IDs
- **Compact and detailed views**
- **Level-based display** showing dependency depth

## 🎯 Command Line Interface

### User Management
```bash
# Create user
tasker user create user@example.com "User Name"

# Get user by email or ID
tasker user get user@example.com
tasker user get a1b2c3d4
```

### Task Management
```bash
# Add task (goes to inbox)
tasker --user <user-id> add "Task Name" -d "Description"

# List tasks (defaults to inbox)
tasker --user <user-id> list
tasker --user <user-id> list staging
tasker --user <user-id> list pending

# Stage task with location
tasker --user <user-id> stage <task-id> --location project --location backend

# Stage task with dependency (inherits location)
tasker --user <user-id> stage <source-task> <destination-task>

# Interactive staging (fuzzy picker or location entry)
tasker --user <user-id> stage <task-id>
```

### Task Operations
```bash
# Start task
tasker --user <user-id> start <task-id>

# Complete task  
tasker --user <user-id> complete <task-id>

# Get task details
tasker --user <user-id> get <task-id>
```

### Visualization
```bash
# View dependency graph
tasker --user <user-id> dag

# Compact view
tasker --user <user-id> dag --compact
```

## 🏗️ System Architecture

### Backend Components
- **gRPC Server** - Core service API
- **Task Repository** - Persistent storage interface
- **User Repository** - User management
- **ID Resolver** - User-partitioned Trie for task IDs
- **DAG Renderer** - Dependency visualization

### Data Structure
```go
Task {
    stage: pending|inbox|staging|active
    location: []string  // hierarchical like folder structure
    points: []Point{title: string, val: uint}
    schedule: {work_intervals: [{start, stop, points_completed}], due: time}
    status: {updates: [{time: datetime, update: string}]}
    tags: map[string]interface{}  // user-configurable
    id: string  // 32-bit UUID with minimum unique prefix support
    inflows: []string   // dependency predecessors
    outflows: []string  // dependency successors
}
```

### Core Operations
- `add task` → pending → inbox (auto-transition)
- `stage task` → staging with location/dependencies
- `start task` → active with scheduling
- `complete task` → archived (outside system)

### Constraints
- **Inbox limit**: Maximum 5 tasks
- **No chain hopping**: Can't create A→C when A→B→C exists
- **Location requirement**: Staging tasks must have location
- **User isolation**: All operations scoped to user context

## 🐳 Production Deployment

### Docker Compose
```yaml
# Included docker-compose.yml
version: '3.8'
services:
  task-server:
    build: .
    ports: ["8080:8080"]
    healthcheck: ...
    restart: unless-stopped
```

### Environment Variables
- `PORT` - Server port (default: 8080)
- `LOG_LEVEL` - Logging level
- `MAX_INBOX_SIZE` - Inbox constraint (default: 5)

### Health Monitoring
- Health check endpoint: `/health`
- Docker health checks included
- Automatic restart policies

## 🧪 Testing

### Automated Tests
```powershell
# Quick functionality test
.\quick-test.ps1

# Comprehensive end-to-end test
.\test-e2e.ps1

# Unit tests
go test ./...

# Test with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage
- User management and resolution
- Task staging and dependencies  
- ID resolution with partial matching
- DAG visualization and minimum prefixes
- User partitioning and isolation
- Location parsing and inheritance

## 🔌 API Integration

### gRPC Service
```protobuf
service TaskService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc AddTask(AddTaskRequest) returns (AddTaskResponse);
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
  rpc MoveToStaging(MoveToStagingRequest) returns (MoveToStagingResponse);
  rpc StartTask(StartTaskRequest) returns (StartTaskResponse);
  rpc GetTaskDAG(GetTaskDAGRequest) returns (GetTaskDAGResponse);
  rpc ResolveTaskID(ResolveTaskIDRequest) returns (ResolveTaskIDResponse);
  rpc ResolveUserID(ResolveUserIDRequest) returns (ResolveUserIDResponse);
}
```

### Client Libraries
- Go client included in `cmd/client`
- gRPC definitions in `proto/taskcore/v1/`
- Easy integration with other languages via protobuf

## 📈 Scalability & Performance

### Design Considerations
- **User-partitioned storage** - O(1) user isolation
- **Trie-based ID resolution** - Fast prefix matching
- **Stateless server** - Horizontal scaling ready
- **Dependency-aware operations** - Maintains data consistency

### Performance Features
- Minimum unique prefixes reduce typing
- Fuzzy picker for interactive workflows
- Efficient DAG algorithms
- Compact visualization options

## 🛠️ Development

### Project Structure
```
tasker-core/
├── cmd/           # Executables (server, client)
├── internal/      # Private packages
│   ├── domain/    # Core business logic
│   ├── grpc/      # gRPC server implementation
│   ├── repository/# Data access layer
│   ├── service/   # Business services
│   └── idresolver/# ID resolution system
├── proto/         # Protobuf definitions
├── bin/           # Built binaries
└── *.ps1         # Deployment/test scripts
```

### Adding Features
1. Update protobuf definitions in `proto/`
2. Regenerate code: `buf generate`
3. Implement in `internal/grpc/` and `internal/service/`
4. Add CLI commands in `cmd/client/`
5. Add tests and update documentation

## 📄 License

[Your License Here]

## 🤝 Contributing

1. Fork the repository
2. Create feature branch
3. Add tests for new functionality  
4. Run full test suite: `.\test-e2e.ps1`
5. Submit pull request

---

**Production Ready**: This system is designed for production use with proper error handling, health checks, Docker deployment, comprehensive testing, and horizontal scaling capabilities.
