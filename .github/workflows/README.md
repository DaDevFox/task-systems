# CI/CD Workflows

This document describes the CI/CD workflows for the task-systems monorepo.

## Comprehensive CI/CD Workflow

The main workflow is `comprehensive-ci.yml` which provides:

### Features

1. **Smart Change Detection** - Only runs tests/builds for projects that have changed
2. **Protobuf Generation** - Generates protobuf files matching the PowerShell script standard
3. **Multi-Project Testing** - Tests Go and .NET projects separately
4. **Security Scanning** - Runs security scans on Go projects
5. **Code Quality Analysis** - Analyzes cognitive complexity for maintainability
6. **Docker Builds** - Builds and pushes Docker images with semantic versioning
7. **Semantic Versioning** - Follows SCOPES.md for version bumping

### Required Secrets

For Docker builds and pushes to work, you need to set these GitHub secrets:

- `DOCKER_USERNAME` - Your Docker Hub username
- `DOCKER_TOKEN` - A Docker Hub access token (not password)

### Change Detection Logic

The workflow only runs relevant jobs based on what files have changed:

- **Go Projects**: Triggered by changes to `*.go` files, `go.mod`, `go.sum`, or `*.proto` files
- **.NET Projects**: Triggered by changes to `*.cs`, `*.csproj`, or frontend files
- **Docker Builds**: Triggered by changes to Dockerfiles or when Go projects change
- **Protobuf**: Triggered by changes to `*.proto` files

### Semantic Versioning

Versions are determined by analyzing commit messages according to SCOPES.md:

- **Major Version Bump**: Breaking changes with `[!]` modifier
- **Minor Version Bump**: `FEAT(...)` commits
- **Patch Version Bump**: `ENH_(...)` or `FIX_(...)` commits
- **Development Builds**: Other changes get timestamped dev versions

### Project Scopes

The workflow maps projects to scopes from SCOPES.md:

- `tasker-core` → `TASK`
- `inventory-core/*` → `INV_`
- `home-manager/*` → `WKFL`
- `user-core` → `USER`
- Others → `ALL_`

### Protobuf Generation

Protobuf files are generated to match the structure defined in `generate-proto.ps1`:

- `{project}/pkg/proto/{service}/v1/*.pb.go`
- Supports multiple proto files per project (like home-manager)
- Generated files are uploaded as artifacts for use by other jobs

### Docker Images

Docker images are built for projects with Dockerfiles and pushed to Docker Hub on main branch pushes:

- Format: `{username}/{project-name}:latest` and `{username}/{project-name}:v{version}`
- Images include proper OCI labels with version info
- Git tags are automatically created for production versions

### Code Quality

The workflow includes:

- **Cognitive Complexity Analysis** - Reports functions with complexity ≥ 15
- **Security Scanning** - Uses Gosec to detect security issues
- **Test Coverage** - Generates coverage reports for all projects
- **Linting** - Runs `go vet` and format checks

### Artifacts

The following artifacts are generated and stored:

- **Coverage Reports** - Test coverage for all projects (30 days)
- **Test Results** - .NET test results in TRX format (30 days)
- **Binaries** - Built executables from Go projects (7 days)
- **Code Quality** - Complexity analysis and Code Climate reports (30 days)

### Example Usage

1. **Feature Development**: Create a commit like `FEAT(TASK): add new task scheduling`
2. **Bug Fix**: Create a commit like `FIX_(INV_): resolve inventory sync issue`
3. **Breaking Change**: Create a commit like `ENH_(WKFL): [!]remove legacy API support`

The CI/CD will automatically:
- Run tests for affected projects
- Generate appropriate version numbers
- Build and push Docker images
- Create Git tags for releases

### Local Development

To replicate the protobuf generation locally, use:
```powershell
.\generate-proto.ps1 -Verbose
```

This ensures your local development matches the CI/CD environment.
