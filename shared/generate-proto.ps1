#!/usr/bin/env pwsh


# Generate protobuf files for shared components using buf
Write-Host "Generating shared protobuf files..."

# Clean existing generated files
if (Test-Path "backend/pkg/proto") {
    Remove-Item -Recurse -Force "backend/pkg/proto"
}

# Ensure output directory exists
New-Item -ItemType Directory -Force -Path "backend/pkg/proto" | Out-Null

# Use buf generate instead of protoc directly
buf generate

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Protobuf generation completed successfully"
    Write-Host "`nGenerated files:" -ForegroundColor Cyan
    if (Test-Path "backend/pkg/proto") {
        Get-ChildItem -Recurse "backend/pkg/proto" -Name
    }
    else {
        Write-Host "✗ backend/pkg/proto directory does not exist!" -ForegroundColor Red
    }
}
else {
    Write-Host "❌ Protobuf generation failed"
    exit 1
}
