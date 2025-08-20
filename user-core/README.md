# User-Core Service

Centralized user management service for the Task Systems ecosystem.

## Overview

User-Core provides centralized user authentication, authorization, and profile management for all services in the Task Systems suite (Tasker-Core, Home-Manager, Inventory-Core, and future services).

## Architecture

- **Backend**: Go with gRPC and Protocol Buffers
- **Storage**: BadgerDB for persistence
- **Logging**: Structured logging with Logrus
- **Authentication**: JWT tokens (future enhancement)

## Features

### v0 (Current)
- User creation and management
- User lookup by ID, email, or name
- Basic user profiles with notification preferences  
- User validation for other services
- Configuration management per user

### v1 (Future)
- JWT-based authentication
- Role-based access control (RBAC)
- OAuth integration
- Audit logging
- User groups and permissions

## API

The service provides a gRPC API with the following main methods:
- `CreateUser` - Create new user accounts
- `GetUser` - Retrieve user by ID/email/name
- `UpdateUser` - Update user profile and settings
- `ListUsers` - List all users (with filtering)
- `ValidateUser` - Check if user exists
- `DeleteUser` - Remove user account

## Integration

Other services integrate with User-Core via gRPC:
- **Tasker-Core**: User management, task ownership, notification preferences
- **Home-Manager**: User validation, task assignment, leaderboards
- **Inventory-Core**: Future project/store ownership and access control

## Development

```powershell
# Build
go build ./cmd/server

# Test
go test ./...

# Generate protobuf
./generate-proto.ps1

# Run server
./cmd/server/server
```

## Security

See SECURITY.md for authentication roadmap and security considerations.
