#!/usr/bin/env pwsh

# Build script for User-Core service
# Handles protobuf generation and Go compilation

Write-Host "Building User-Core service..." -ForegroundColor Green

# Ensure we're in the right directory
$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $projectRoot

# Step 1: Generate protobuf files
Write-Host "Generating protobuf files..." -ForegroundColor Yellow
& ./generate-proto.ps1
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Protobuf generation failed!" -ForegroundColor Red
    exit 1
}

# Step 2: Build backend
Write-Host "Building backend..." -ForegroundColor Yellow
Set-Location backend

# Update dependencies
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Failed to update dependencies!" -ForegroundColor Red
    exit 1
}

# Build server binary
Write-Host "Compiling server binary..." -ForegroundColor Yellow
go build -o ../bin/user-core-server.exe ./cmd/server
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Server compilation failed!" -ForegroundColor Red
    exit 1
}

Set-Location ..

# Step 3: Run tests
Write-Host "Running tests..." -ForegroundColor Yellow
Set-Location backend
go test ./... -v
if ($LASTEXITCODE -ne 0) {
    Write-Host "⚠ Tests failed, but build completed" -ForegroundColor Yellow
}
else {
    Write-Host "✓ All tests passed!" -ForegroundColor Green
}

Set-Location ..

Write-Host "✓ User-Core service built successfully!" -ForegroundColor Green
Write-Host "Binary location: ./bin/user-core-server.exe" -ForegroundColor Cyan
