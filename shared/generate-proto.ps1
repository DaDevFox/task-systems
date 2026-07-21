#!/usr/bin/env pwsh


# Generate protobuf files for shared components using buf
Write-Host "Generating shared protobuf files..."

# Clean existing generated files
if (Test-Path "proto") {
    Remove-Item -Recurse -Force "proto"
}

# Ensure output directory exists
New-Item -ItemType Directory -Force -Path "proto" | Out-Null

# Use buf generate instead of protoc directly
buf generate

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Protobuf generation completed successfully"
    Write-Host "`nGenerated files:" -ForegroundColor Cyan
    if (Test-Path "proto") {
        Get-ChildItem -Recurse "proto" -Name
    }
    else {
        Write-Host "✗ proto directory does not exist!" -ForegroundColor Red
    }
}
else {
    Write-Host "❌ Protobuf generation failed"
    exit 1
}
