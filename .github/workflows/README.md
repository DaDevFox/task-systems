# Comprehensive CI/CD Pipeline

This directory contains the GitHub Actions workflows for the task-systems monorepo.

## Workflows

### `comprehensive-ci.yml`
The main CI/CD pipeline that:

1. **Detects Changes**: Identifies which Go and .NET projects have changed
2. **Protocol Buffer Generation**: Generates protobuf code for all projects using buf/protoc
3. **Go Testing**: Tests each Go project separately with proper dependency handling
4. **C# Testing**: Tests .NET projects with backend integration support
5. **Security Scanning**: Runs Gosec security analysis on Go projects
6. **Code Quality**: Analyzes cognitive complexity and posts PR comments for high-complexity functions
7. **Artifact Management**: Uploads coverage reports, binaries, and test results

## Project Structure Support

The pipeline automatically detects and tests these projects:

### Go Projects
- `tasker-core/` - Main task management system
- `tasker-core/backend/` - Task backend services
- `inventory-core/` - Inventory management system
- `inventory-core/backend/` - Inventory backend services
- `shared/` - Shared utilities and proto definitions
- `home-manager/backend/` - Home management backend

### .NET Projects  
- `inventory-core/frontend/` - Inventory frontend (C# Avalonia)

## Features

### Parallel Testing
Each project is tested in parallel with proper isolation and dependency handling.

### Smart Change Detection
Only tests projects that have changed, speeding up CI for focused changes.

### Protocol Buffer Support
Automatically generates protobuf code using buf and protoc before testing.

### Dependency Management
Handles local module replacements and complex dependency chains between projects.

### Coverage Reporting
Generates and uploads coverage reports for each project separately.

### Cognitive Complexity Analysis
Analyzes Go code for functions with high cognitive complexity (>15) and posts actionable feedback on PRs.

### Security Scanning
Runs Gosec security analysis on all Go projects and uploads SARIF reports.

## Usage

The pipeline runs automatically on:
- Pushes to `main` or `develop` branches
- Pull requests to `main` branch

### Artifacts

The following artifacts are generated:
- **Coverage Reports** (`coverage-{project}-{run_number}`): HTML and text coverage reports
- **Binaries** (`binaries-{project}-{run_number}`): Built executables for each project
- **Test Results** (`test-results-{project}-{run_number}`): .NET test results and logs
- **Code Quality** (`code-climate-report-{run_number}`): Complexity analysis results

### Cognitive Complexity Thresholds

Functions are flagged based on cognitive complexity:
- 🟢 **1-14**: Acceptable
- 🟡 **15-20**: Consider refactoring when adding features
- 🟠 **21-30**: Should be refactored for maintainability  
- 🔴 **31+**: Critical - requires immediate attention

## Troubleshooting

### Go Module Issues
If you see "missing go.sum entry" errors, the pipeline automatically handles local module replacements for the monorepo structure.

### Integration Test Failures
.NET integration tests may fail if the required Go backend isn't properly built. The pipeline builds Go binaries first and makes them available to .NET tests.

### Protocol Buffer Generation Failures
Proto generation errors are logged but don't fail the entire pipeline, allowing other tests to continue.

## Local Development

To run similar checks locally:

```bash
# Run Go tests for a specific project
cd tasker-core
go mod tidy
go test -v -race -coverprofile=coverage.out ./...

# Run .NET tests
cd inventory-core/frontend  
dotnet test --logger trx

# Check cognitive complexity
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
gocyclo -over 14 .
```
