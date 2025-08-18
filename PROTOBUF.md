# Protobuf Generation Guide

This project uses Protocol Buffers (protobuf) to define service interfaces and data structures across both **Go** and **C#** projects. All generated files are **git-ignored** and regenerated automatically during builds.

## Directory Structure

All protobuf files follow standardized directory structures for each language:

### Go Projects
```
{project}/
├── proto/                           # Source .proto files
│   ├── {service}.proto
│   └── ...
└── pkg/proto/{service}/v1/         # Generated Go code (git-ignored)
    ├── {service}.pb.go
    └── {service}_grpc.pb.go
```

### C# Projects  
```
{project}/
├── proto/                           # Source .proto files (symlinked/copied)
│   ├── {service}.proto
│   └── ...
└── frontend/
    ├── src/Generated/Proto/{Service}/V1/  # Generated C# code (git-ignored)
    │   ├── {Service}.cs
    │   └── {Service}Grpc.cs
    └── src/
        └── {ProjectName}/
            └── {ProjectName}.csproj   # Contains <Protobuf> references
```

### Project-Specific Layout

| Project | Language | Generated Location |
|---------|----------|-------------------|
| **tasker-core** | Go | `pkg/proto/taskcore/v1/` |
| **inventory-core** | Go | `pkg/proto/inventory/v1/` |
| **inventory-core** | C# | `frontend/src/Generated/Proto/Inventory/V1/` |
| **shared** | Go | `pkg/proto/events/v1/` |
| **home-manager** | Go | `backend/pkg/proto/hometasker/v1/` |

## Local Development

### Prerequisites

#### For Go Projects
1. **Install protoc**: 
   - Ubuntu/Debian: `sudo apt-get install protobuf-compiler`
   - macOS: `brew install protobuf`
   - Windows: Download from [protobuf releases](https://github.com/protocolbuffers/protobuf/releases)

2. **Install Go protobuf plugins**:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

#### For C# Projects
C# protobuf generation is handled automatically by **MSBuild** using the `Grpc.Tools` NuGet package. No additional setup required.

### Generate Protobuf Files

**Option 1: Use the provided scripts**
```bash
# Linux/macOS
./generate-proto.sh

# Windows PowerShell  
./generate-proto.ps1 -Verbose
```

**Option 2: Manual generation (Go)**
```bash
# For individual Go projects
cd tasker-core
protoc --go_out=pkg/proto --go_opt=paths=source_relative \
       --go-grpc_out=pkg/proto --go-grpc_opt=paths=source_relative \
       --proto_path=proto \
       --proto_path=/usr/include \
       proto/task.proto
```

**Option 3: Manual generation (C#)**
C# projects use MSBuild integration via `.csproj` files:
```xml
<ItemGroup>
  <Protobuf Include="..\..\..\proto\inventory.proto" GrpcServices="Client" ProtoRoot="..\..\.." />
</ItemGroup>
```

## CI/CD Integration

The GitHub Actions workflow automatically handles both Go and C# protobuf generation:

### Go Projects
1. **Detects protobuf changes** in `.proto` files
2. **Installs protoc** and Go plugins
3. **Generates all protobuf files** using the standardized structure
4. **Uploads artifacts** for use by build jobs
5. **Downloads artifacts** in Go build jobs

### C# Projects
1. **Builds with MSBuild** which automatically handles protobuf compilation
2. **Uses Grpc.Tools package** for C# code generation
3. **Integrates with existing .csproj configuration**

### Workflow Dependencies

```yaml
jobs:
  setup-proto:
    # Generates Go protobuf files when .proto files change
    
  test-go-projects:
    needs: [detect-changes, setup-proto]
    # Downloads and uses generated Go protobuf artifacts
    
  test-dotnet-projects:
    needs: [detect-changes, setup-proto, test-go-projects] 
    # Builds C# projects with automatic protobuf compilation
```

## Import Paths

### Go Import Paths
All Go code should use the standardized import paths:

```go
// Correct - standardized paths
import pb "github.com/DaDevFox/task-systems/tasker-core/pkg/proto/taskcore/v1"
import eventspb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
import inventorypb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"

// Incorrect - old paths
import pb "github.com/DaDevFox/task-systems/task-core"
import "github.com/DaDevFox/task-systems/shared/proto/events/v1"
```

### C# Namespaces
C# generated code uses the protobuf package declarations:

```csharp
// Generated from proto package: inventory.core.v1
using InventoryCore.V1;
using Grpc.Core;

// Usage
var client = new InventoryService.InventoryServiceClient(channel);
var response = await client.GetInventoryItemAsync(request);
```

## Module Dependencies

### Go Modules
Each `go.mod` file includes local replacements for monorepo development:

```go
// For root-level modules
replace github.com/DaDevFox/task-systems/shared => ../shared
replace github.com/DaDevFox/task-systems/tasker-core => ../tasker-core

// For backend modules  
replace github.com/DaDevFox/task-systems/shared => ../../shared
replace github.com/DaDevFox/task-systems/tasker-core => ../../tasker-core
```

### C# Project References
C# projects reference protobuf files using relative paths:

```xml
<ItemGroup>
  <Protobuf Include="..\..\..\proto\inventory.proto" GrpcServices="Client" ProtoRoot="..\..\.." />
</ItemGroup>
```

## Git Integration

Generated files are automatically ignored:

```gitignore
# Go generated files
*.pb.go
**/*.pb.go

# C# generated files
*.pb.cs
**/*.pb.cs
*Grpc.cs
**/*Grpc.cs
```

## Troubleshooting

### Common Issues

1. **Go - Import not found**: Ensure protobuf files are generated and paths match
2. **Go - Module not found**: Check `go.mod` local replacements
3. **C# - Build errors**: Check `.csproj` Protobuf includes and NuGet packages
4. **Version conflicts**: Run `go mod tidy` or `dotnet restore` after changes

### Regenerate Everything

```bash
# Clean all generated files (they're git-ignored anyway)
find . -name "*.pb.go" -delete
find . -name "*.pb.cs" -delete
find . -name "*Grpc.cs" -delete

# Regenerate all
./generate-proto.sh  # or .ps1 on Windows

# For C#, also clean and rebuild
dotnet clean
dotnet build
```

### Debug Build Issues

```bash
# Check if Go protobuf files exist
find . -name "*.pb.go" -type f

# Verify Go import paths in generated files
grep -r "package.*v1" */pkg/proto/

# Check C# generated files
find . -name "*.pb.cs" -type f
find . -name "*Grpc.cs" -type f
```

## Language-Specific Notes

### Go
- **Thread-safe**: Generated Go code is thread-safe
- **Memory efficient**: Uses efficient serialization
- **Type safety**: Strong typing with compile-time checks

### C#
- **Async support**: Full async/await support via Grpc.Net.Client
- **JSON integration**: Easy JSON serialization with Google.Protobuf
- **LINQ compatible**: Generated types work with LINQ queries

## Best Practices

### General
1. **Never commit** generated files (`.pb.go`, `.pb.cs`, `*Grpc.cs`)
2. **Always regenerate** locally after `.proto` changes
3. **Use standardized paths** for all imports
4. **Test builds** work without committed protobuf files
5. **Update both scripts** when adding new services

### Go-Specific
1. **Use interface segregation** - define minimal service interfaces
2. **Handle context properly** - always pass context through gRPC calls
3. **Error wrapping** - wrap gRPC errors with context

### C#-Specific
1. **Use dependency injection** - register gRPC clients in DI container
2. **Configure retry policies** - handle transient failures
3. **Use cancellation tokens** - support operation cancellation

## Adding New Services

### For Go Projects
1. Create `.proto` file in `{project}/proto/`
2. Update `generate-proto.sh` and `generate-proto.ps1`
3. Add import paths using pattern: `{project}/pkg/proto/{service}/v1`
4. Update `go.mod` with necessary dependencies

### For C# Projects  
1. Create `.proto` file in `{project}/proto/`
2. Add `<Protobuf>` reference to `.csproj` files
3. Install necessary NuGet packages (`Grpc.Net.Client`, `Google.Protobuf`)
4. Generated files will use namespace from proto package declaration
