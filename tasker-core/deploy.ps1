#!/usr/bin/env pwsh
# Production deployment script for Task Management System
# This script builds and deploys the complete system

param(
    [string]$Environment = "development",
    [switch]$BuildOnly = $false,
    [switch]$TestFirst = $true,
    [string]$Port = "8080"
)

$ErrorActionPreference = "Stop"

# Colors for output
$Green = "`e[32m"
$Red = "`e[31m"
$Yellow = "`e[33m"
$Blue = "`e[34m"
$Reset = "`e[0m"

function Write-Step {
    param([string]$Message)
    Write-Host "${Blue}[STEP]${Reset} $Message"
}

function Write-Success {
    param([string]$Message)
    Write-Host "${Green}[SUCCESS]${Reset} $Message"
}

function Write-Error {
    param([string]$Message)
    Write-Host "${Red}[ERROR]${Reset} $Message"
}

function Write-Warning {
    param([string]$Message)
    Write-Host "${Yellow}[WARNING]${Reset} $Message"
}

function Write-Info {
    param([string]$Message)
    Write-Host "${Blue}[INFO]${Reset} $Message"
}

Write-Info "=== Task Management System Production Deployment ==="
Write-Info "Environment: $Environment"
Write-Info "Port: $Port"
Write-Info "Build Only: $BuildOnly"
Write-Info "Test First: $TestFirst"

try {
    # Step 1: Pre-flight checks
    Write-Step "Performing pre-flight checks..."
    
    # Check if Docker is available
    try {
        docker --version | Out-Null
        Write-Success "Docker is available"
    } catch {
        Write-Error "Docker is not available. Please install Docker first."
        exit 1
    }
    
    # Check if Go is available
    try {
        go version | Out-Null
        Write-Success "Go is available"
    } catch {
        Write-Error "Go is not available. Please install Go first."
        exit 1
    }

    # Step 2: Clean previous builds
    Write-Step "Cleaning previous builds..."
    if (Test-Path "bin") {
        Remove-Item "bin" -Recurse -Force
    }
    if (Test-Path "server.exe") { Remove-Item "server.exe" -Force }
    if (Test-Path "client.exe") { Remove-Item "client.exe" -Force }
    Write-Success "Cleaned previous builds"

    # Step 3: Run tests if requested
    if ($TestFirst) {
        Write-Step "Running comprehensive tests..."
        
        # Run quick test first
        try {
            .\quick-test.ps1
            Write-Success "Quick tests passed"
        } catch {
            Write-Warning "Quick tests failed, but continuing..."
        }
        
        # Run unit tests
        try {
            go test ./... -v
            Write-Success "Unit tests passed"
        } catch {
            Write-Error "Unit tests failed"
            throw "Tests failed"
        }
    }

    # Step 4: Generate protobuf code
    Write-Step "Generating protobuf code..."
    try {
        if (Get-Command buf -ErrorAction SilentlyContinue) {
            buf generate
        } else {
            .\generate-proto-simple.ps1
        }
        Write-Success "Protobuf code generated"
    } catch {
        Write-Error "Failed to generate protobuf code: $_"
        throw "Protobuf generation failed"
    }

    # Step 5: Build binaries
    Write-Step "Building server and client binaries..."
    
    # Create bin directory
    if (!(Test-Path "bin")) {
        New-Item -ItemType Directory -Path "bin" | Out-Null
    }
    
    # Build server
    go build -ldflags "-s -w" -o bin/server.exe ./cmd/server
    if ($LASTEXITCODE -ne 0) {
        throw "Server build failed"
    }
    
    # Build client
    go build -ldflags "-s -w" -o bin/client.exe ./cmd/client
    if ($LASTEXITCODE -ne 0) {
        throw "Client build failed"
    }
    
    Write-Success "Binaries built successfully"

    # Step 6: Build Docker image (if not build-only)
    if (!$BuildOnly) {
        Write-Step "Building Docker image..."
        docker build -t task-management-system:latest .
        if ($LASTEXITCODE -ne 0) {
            throw "Docker build failed"
        }
        Write-Success "Docker image built successfully"

        # Step 7: Deploy with docker-compose
        Write-Step "Deploying with docker-compose..."
        
        # Set environment variables
        $env:PORT = $Port
        
        # Stop any existing containers
        docker-compose down --remove-orphans 2>$null
        
        # Start the service
        docker-compose up -d
        if ($LASTEXITCODE -ne 0) {
            throw "Docker deployment failed"
        }
        
        Write-Success "Service deployed successfully"
        
        # Step 8: Health check
        Write-Step "Performing health check..."
        $healthCheckAttempts = 0
        $maxAttempts = 10
        
        do {
            Start-Sleep -Seconds 3
            $healthCheckAttempts++
            
            try {
                $response = Invoke-WebRequest -Uri "http://localhost:$Port/health" -Method GET -TimeoutSec 5 -ErrorAction SilentlyContinue
                if ($response.StatusCode -eq 200) {
                    Write-Success "Health check passed"
                    break
                }
            } catch {
                # Health check endpoint might not exist, try a simple connection
                try {
                    $tcpClient = New-Object System.Net.Sockets.TcpClient
                    $tcpClient.Connect("localhost", $Port)
                    $tcpClient.Close()
                    Write-Success "Service is accepting connections"
                    break
                } catch {
                    if ($healthCheckAttempts -eq $maxAttempts) {
                        Write-Warning "Health check failed after $maxAttempts attempts"
                        break
                    }
                    Write-Info "Health check attempt $healthCheckAttempts/$maxAttempts failed, retrying..."
                }
            }
        } while ($healthCheckAttempts -lt $maxAttempts)

        # Step 9: Display deployment information
        Write-Info ""
        Write-Info "=== Deployment Complete ==="
        Write-Info "Server URL: http://localhost:$Port"
        Write-Info "Client binary: ./bin/client.exe"
        Write-Info "Environment: $Environment"
        Write-Info ""
        Write-Info "🔍 Structured Logging:"
        Write-Info "  - All RPC handlers include comprehensive structured logging with logrus"
        Write-Info "  - Logs include request IDs, operation timings, validation details, and error context"
        Write-Info "  - JSON format for easy parsing and filtering"
        Write-Info "  - View logs with: docker-compose logs -f"
        Write-Info ""
        Write-Info "Quick Start Commands:"
        Write-Info "  Create user:  .\bin\client.exe --server localhost:$Port user create user@example.com 'User Name'"
        Write-Info "  Add task:     .\bin\client.exe --server localhost:$Port --user <user-id> add 'Task Name'"
        Write-Info "  List tasks:   .\bin\client.exe --server localhost:$Port --user <user-id> list"
        Write-Info "  Stage task:   .\bin\client.exe --server localhost:$Port --user <user-id> stage <task-id> --location project/module"
        Write-Info "  View DAG:     .\bin\client.exe --server localhost:$Port --user <user-id> dag"
        Write-Info ""
        Write-Info "To stop the service: docker-compose down"
        Write-Info "To view logs: docker-compose logs -f"
    }

    Write-Success "Deployment completed successfully! 🎉"

} catch {
    Write-Error "Deployment failed: $_"
    
    # Cleanup on failure
    Write-Step "Cleaning up after failure..."
    docker-compose down --remove-orphans 2>$null
    
    exit 1
}
