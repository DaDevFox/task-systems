# Protobuf Generation Guide

This project uses Protocol Buffers (protobuf) to define service interfaces and data structures. All generated `.pb.go` files are **git-ignored** and regenerated automatically during builds.

## Directory Structure

All protobuf files follow a standardized directory structure:

```
{project}/
├── proto/                           # Source .proto files
│   ├── {service}.proto
│   └── ...
└── pkg/proto/{service}/v1/         # Generated Go code (git-ignored)
    ├── {service}.pb.go
    └── {service}_grpc.pb.go
```

### Project Layout

- **tasker-core**: `pkg/proto/taskcore/v1/`
- **inventory-core**: `pkg/proto/inventory/v1/`
- **shared**: `pkg/proto/events/v1/`
- **home-manager**: `backend/pkg/proto/hometasker/v1/`

## Local Development

### Prerequisites

1. **Install protoc**: 
   - Ubuntu/Debian: `sudo apt-get install protobuf-compiler`
   - macOS: `brew install protobuf`
   - Windows: Download from [protobuf releases](https://github.com/protocolbuffers/protobuf/releases)

2. **Install Go protobuf plugins**:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

### Generate Protobuf Files

**Option 1: Use the provided scripts**
```bash
# Linux/macOS
./generate-proto.sh

# Windows PowerShell  
./generate-proto.ps1 -Verbose
```

**Option 2: Manual generation**
```bash
# For individual projects
cd tasker-core
protoc --go_out=pkg/proto --go_opt=paths=source_relative \
       --go-grpc_out=pkg/proto --go-grpc_opt=paths=source_relative \
       --proto_path=proto \
       proto/task.proto
```

## CI/CD Integration

The GitHub Actions workflow automatically:

1. **Detects protobuf changes** in `.proto` files
2. **Installs protoc** and Go plugins
3. **Generates all protobuf files** using the standardized structure
4. **Uploads artifacts** for use by build jobs
5. **Downloads artifacts** in Go build jobs

### Workflow Dependencies

```yaml
jobs:
  setup-proto:
    # Generates protobuf files when .proto files change
    
  test-go-projects:
    needs: [detect-changes, setup-proto]
    # Downloads and uses generated protobuf artifacts
```

## Import Paths

All Go code should use the standardized import paths:

```go
// Correct
import pb "github.com/DaDevFox/task-systems/tasker-core/pkg/proto/taskcore/v1"
import eventspb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
import inventorypb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"

// Incorrect - old paths
import pb "github.com/DaDevFox/task-systems/task-core"
import "github.com/DaDevFox/task-systems/shared/proto/events/v1"
```

## Module Dependencies

Each `go.mod` file includes local replacements for monorepo development:

```go
// For root-level modules
replace github.com/DaDevFox/task-systems/shared => ../shared
replace github.com/DaDevFox/task-systems/tasker-core => ../tasker-core

// For backend modules  
replace github.com/DaDevFox/task-systems/shared => ../../shared
replace github.com/DaDevFox/task-systems/tasker-core => ../../tasker-core
```

## Troubleshooting

### Common Issues

1. **Import not found**: Ensure protobuf files are generated and paths match
2. **Module not found**: Check `go.mod` local replacements
3. **Version conflicts**: Run `go mod tidy` after changes

### Regenerate Everything

```bash
# Clean generated files (they're git-ignored anyway)
find . -name "*.pb.go" -delete

# Regenerate all
./generate-proto.sh  # or .ps1 on Windows
```

### Debug Build Issues

```bash
# Check if protobuf files exist
find . -name "*.pb.go" -type f

# Verify import paths in generated files
grep -r "package.*v1" */pkg/proto/
```

## Best Practices

1. **Never commit** `.pb.go` files (they're git-ignored)
2. **Always regenerate** locally after `.proto` changes
3. **Use standardized paths** for all imports
4. **Test builds** work without committed protobuf files
5. **Update both scripts** when adding new services
