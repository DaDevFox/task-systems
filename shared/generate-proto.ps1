#!/usr/bin/env pwsh

# Generate protobuf files for shared components using buf
Write-Host "Generating shared protobuf files..."

# Use buf generate instead of protoc directly
buf generate

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Protobuf generation completed successfully"
} else {
    Write-Host "❌ Protobuf generation failed"
    exit 1
}
