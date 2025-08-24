# PowerShell script to generate protobuf code using protoc


# Create the output directory for generated Go files
New-Item -ItemType Directory -Force -Path "backend/pkg/proto/taskcore/v1" | Out-Null

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


protoc --go_out=backend/pkg/proto --go_opt=paths=source_relative `
    --go-grpc_out=backend/pkg/proto --go-grpc_opt=paths=source_relative `
    --proto_path=proto `
    --proto_path="$env:GOPATH/pkg/mod/google.golang.org/protobuf@v1.34.2" `
    proto/task.proto

if ($LASTEXITCODE -eq 0) {
    Write-Host "Protocol buffer code generated successfully"
}
else {
    Write-Error "Failed to generate protocol buffer code"
    exit 1
}
