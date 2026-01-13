# Task Core Service

A backend service for managing tasks through different lifecycle stages (pending, inbox, staging, active, archived) with dependency management and extensible integrations.

## Architecture

The service implements a task management system with the following stages:

- **Stage 0 (Pending)**: New tasks from users or automated triggers
- **Stage 1 (Inbox)**: Tasks with limited capacity (default: 5 tasks max)
- **Stage 2 (Staging)**: Tasks with dependencies, locations, and points
- **Stage 3 (Active)**: Currently executing tasks with scheduling
- **Archived**: Completed tasks (leave the system)

## Features

- Task lifecycle management through defined stages
- Dependency tracking with inflow/outflow relationships
- Point-based work tracking
- Hierarchical location tagging
- Status history and updates
- gRPC API with protobuf definitions
- Inbox capacity constraints
- Extensible design for future integrations (Calendar, Email, SMS, etc.)

## Prerequisites

- Go 1.24.2 or later
- Protocol Buffers compiler (`protoc`)
- Buf CLI tool for proto management
- Make (optional, for using Makefile)

## Setup Instructions

### 1. Install Dependencies

```powershell
# Install Go dependencies
go mod download

# Install protobuf code generators
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install buf (Protocol Buffer management)
# On Windows, download from: https://github.com/bufbuild/buf/releases
# Or use chocolatey: choco install buf
```

### 2. Generate Protocol Buffer Code

```powershell
# Using buf (recommended)
buf generate

# Or using Make (if available)
make proto
```

### 3. Build the Service

```powershell
# Build the server binary
go build -o bin/task-server.exe ./cmd/server

# Or using Make
make build
```

### 4. Run the Service

```powershell
# Run with default settings (port 8080, inbox size 5)
./bin/task-server.exe

# Run with custom settings
./bin/task-server.exe -port 9090 -max-inbox-size 10

# Or using Make
make run
```

### 5. Run Tests

```powershell
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Or using Make
make test
make test-coverage
```

## Buf Configuration

The project uses Buf for Protocol Buffer management. Key files:

- `buf.yaml`: Buf workspace configuration
- `buf.gen.yaml`: Code generation configuration
- `proto/task.proto`: Task service definitions

### Buf Commands

```powershell
# Lint proto files
buf lint

# Generate code
buf generate

# Format proto files
buf format -w

# Check for breaking changes
buf breaking --against '.git#branch=main'
```

## API Endpoints

The gRPC service provides the following endpoints:

- `AddTask`: Create a new task (pending stage)
- `MoveToStaging`: Move task from pending/inbox to staging
- `StartTask`: Start a task (move to active stage)
- `StopTask`: Stop a task (partial completion)
- `CompleteTask`: Complete a task (move to archived)
- `MergeTasks`: Merge two tasks in the same chain
- `SplitTask`: Split a task into multiple tasks
- `AdvertiseTask`: Make one task outflow into many
- `StitchTasks`: Make multiple tasks outflow into one
- `ListTasks`: List tasks by stage
- `GetTask`: Get a task by ID

## Development Commands

```powershell
# Install all dependencies
make deps

# Generate protocol buffer code
make proto

# Build the application
make build

# Run tests
make test

# Run with development settings
make run-dev

# Clean generated files
make clean

# Run linters
make lint
```

## Docker Support

```powershell
# Build Docker image
docker build -t task-core:latest .

# Run in Docker
docker run -p 8080:8080 task-core:latest

# Or using Make
make docker-build
make docker-run
```

## Project Structure

```
tasker-core/
├── cmd/
│   └── server/          # Main server entry point
├── internal/
│   ├── domain/          # Domain models and business logic
│   ├── repository/      # Data persistence interfaces and implementations
│   ├── service/         # Business logic and orchestration
│   └── grpc/           # gRPC server implementation
├── proto/              # Protocol buffer definitions
├── buf.yaml           # Buf configuration
├── buf.gen.yaml       # Buf code generation config
├── go.mod             # Go module definition
├── Makefile           # Build automation
├── Dockerfile         # Container definition
└── README.md          # This file
```

## Testing

The project includes comprehensive tests for all layers:

- Domain model tests: `internal/domain/task_test.go`
- Repository tests: `internal/repository/memory_repository_test.go`
- Service tests: `internal/service/task_service_test.go`

Tests use the standard Go testing package with table-driven test patterns.

## Future Extensions

The service is designed to support future integrations:

- **Google Calendar**: For scheduled task materialization
- **Email/SMS/Push Notifications**: For task updates and reminders
- **Group Assignment System**: For coordinated task assignments
- **Advanced Scheduling**: For complex task scheduling logic

The modular architecture allows easy extension through the service layer without affecting core business logic.

## Configuration

Server configuration options:

- `-port`: Server port (default: 8080)
- `-max-inbox-size`: Maximum inbox capacity (default: 5)

## Health Monitoring

The Docker container includes a health check that verifies the gRPC server is responding on the configured port.

## Contributing

1. Follow Go conventions and best practices
2. Write tests for all new functionality
3. Use buf for protocol buffer management
4. Update documentation for API changes
5. Ensure all tests pass before submitting changes
