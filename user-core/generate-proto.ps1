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
if (Test-Path "pkg/proto") {
    Remove-Item -Recurse -Force "pkg/proto"
}

# Create output directory
New-Item -ItemType Directory -Force -Path "pkg/proto" | Out-Null

# Generate Go code
Write-Host "Generating Go protobuf code..." -ForegroundColor Yellow
buf generate

# Check if generation was successful
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Protocol buffer generation completed successfully!" -ForegroundColor Green
    
    # Show generated files
    Write-Host "`nGenerated files:" -ForegroundColor Cyan
    if (Test-Path "pkg/proto") {
        Get-ChildItem -Recurse "pkg/proto" -Name
    } else {
    Write-Host "✗ pkg/proto directory does not exist!" -ForegroundColor Red
  }
} else {
    Write-Host "✗ Protocol buffer generation failed!" -ForegroundColor Red
    exit 1
}
