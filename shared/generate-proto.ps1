#!/usr/bin/env pwsh


# Generate protobuf files for shared components using buf
Write-Host "Generating shared protobuf files..."

# Clean existing generated files
if (Test-Path "pkg/proto") {
    Remove-Item -Recurse -Force "pkg/proto"
}

# Ensure output directory exists
New-Item -ItemType Directory -Force -Path "pkg/proto" | Out-Null

# Use buf generate instead of protoc directly
buf generate

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Protobuf generation completed successfully"
    Write-Host "`nGenerated files:" -ForegroundColor Cyan
    if (Test-Path "pkg/proto") {
        Get-ChildItem -Recurse "pkg/proto" -Name
    }
    else {
        Write-Host "✗ pkg/proto directory does not exist!" -ForegroundColor Red
    }
}
else {
    Write-Host "❌ Protobuf generation failed"
    exit 1
}
