#!/usr/bin/env pwsh

# Generate Protocol Buffers for User-Core service
# This script generates Go code from .proto files using buf

Write-Host "Generating Protocol Buffers for User-Core..." -ForegroundColor Green

# Ensure buf is installed
if (!(Get-Command "buf" -ErrorAction SilentlyContinue)) {
    Write-Host "Error: buf is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install buf from: https://buf.build/docs/installation" -ForegroundColor Yellow
    exit 1
}

# Clean existing generated files
Write-Host "Cleaning existing generated files..." -ForegroundColor Yellow
if (Test-Path "backend/pkg/proto") {
    Remove-Item -Recurse -Force "backend/pkg/proto"
}

# Create output directory
New-Item -ItemType Directory -Force -Path "backend/pkg/proto" | Out-Null

# Generate Go code
Write-Host "Generating Go protobuf code..." -ForegroundColor Yellow

# Prefer direct protoc invocation for reproducible builds (avoid relying on buf CLI here)
# Detect protoc availability and fall back to buf if protoc is unavailable
$protocCmd = (Get-Command protoc -ErrorAction SilentlyContinue)
$bufCmd = (Get-Command buf -ErrorAction SilentlyContinue)

if ($protocCmd) {
    Write-Host "Found protoc at: $($protocCmd.Source)" -ForegroundColor Cyan
    $protocPath = $protocCmd.Source
    # Ensure Go plugins exist
    $protocGenGo = (Get-Command protoc-gen-go -ErrorAction SilentlyContinue)
    $protocGenGoGrpc = (Get-Command protoc-gen-go-grpc -ErrorAction SilentlyContinue)
    if (-not $protocGenGo) {
        Write-Host "Installing protoc-gen-go..." -ForegroundColor Yellow
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        $protocGenGo = (Get-Command protoc-gen-go -ErrorAction SilentlyContinue)
    }
    if (-not $protocGenGoGrpc) {
        Write-Host "Installing protoc-gen-go-grpc..." -ForegroundColor Yellow
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
        $protocGenGoGrpc = (Get-Command protoc-gen-go-grpc -ErrorAction SilentlyContinue)
    }

    $protoFiles = @(
        "proto/usercore/v1/user.proto",
        "proto/usercore/v1/bootstrap_users.proto"
    )

    $protoOut = "backend/pkg/proto"
    if (-not (Test-Path $protoOut)) { New-Item -ItemType Directory -Force -Path $protoOut | Out-Null }

    $protocArgs = @(
        "--go_out=paths=source_relative:$protoOut",
        "--go-grpc_out=paths=source_relative:$protoOut",
        "--plugin=protoc-gen-go=$($protocGenGo.Source)",
        "--plugin=protoc-gen-go-grpc=$($protocGenGoGrpc.Source)",
        "--proto_path=proto"
    ) + $protoFiles

    Write-Host "Running protoc with args: $($protocArgs -join ' ')" -ForegroundColor Magenta
    & $protocPath $protocArgs
    $exit = $LASTEXITCODE
}
elseif ($bufCmd) {
    Write-Host "protoc not found, falling back to buf generate" -ForegroundColor Yellow
    & buf generate
    $exit = $LASTEXITCODE
}
else {
    Write-Host "Error: neither protoc nor buf found in PATH. Install protoc or buf." -ForegroundColor Red
    exit 1
}

# Check if generation was successful
if ($exit -eq 0) {
    Write-Host "Protocol buffer generation completed successfully!" -ForegroundColor Green
    
    # Show generated files
    Write-Host "`nGenerated files:" -ForegroundColor Cyan
    Get-ChildItem -Recurse "backend/pkg/proto" -Name
}
else {
    Write-Host "Protocol buffer generation failed! Exit code: $exit" -ForegroundColor Red
    exit $exit
}
