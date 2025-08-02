# Simplified PowerShell script to generate protobuf code

# Create the output directory
New-Item -ItemType Directory -Force -Path "proto/taskcore/v1"

# Create google/protobuf directory for well-known types
New-Item -ItemType Directory -Force -Path "proto/google/protobuf"

# Download the timestamp.proto file
Write-Host "Downloading Google protobuf well-known types..."
$timestampUrl = "https://raw.githubusercontent.com/protocolbuffers/protobuf/main/src/google/protobuf/timestamp.proto"
Invoke-WebRequest -Uri $timestampUrl -OutFile "proto/google/protobuf/timestamp.proto"

# Check if protoc-gen-go and protoc-gen-go-grpc are installed
$goPath = go env GOPATH
$protoc_gen_go = Join-Path $goPath "bin/protoc-gen-go.exe"
$protoc_gen_go_grpc = Join-Path $goPath "bin/protoc-gen-go-grpc.exe"

if (-not (Test-Path $protoc_gen_go)) {
    Write-Host "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
}

if (-not (Test-Path $protoc_gen_go_grpc)) {
    Write-Host "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
}

# Generate Go code
Write-Host "Generating protobuf code..."

$env:PATH += ";$goPath\bin"

protoc --go_out=. --go_opt=module=github.com/DaDevFox/task-systems/task-core `
       --go-grpc_out=. --go-grpc_opt=module=github.com/DaDevFox/task-systems/task-core `
       --proto_path=proto `
       proto/task.proto

if ($LASTEXITCODE -eq 0) {
    Write-Host "Protocol buffer code generated successfully"
} else {
    Write-Error "Failed to generate protocol buffer code"
    exit 1
}
