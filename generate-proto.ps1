# PowerShell script to generate protobuf files for all projects
# This script ensures consistent protobuf generation across all services

param(
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

Write-Host "Generating protobuf files for all projects..." -ForegroundColor Green

# TODO: return output path + display dynamically at end of script
# Function to generate protobuf with standardized paths
function Generate-Go-Proto {
    param(
        [string]$Project,
        [string]$Service,
        [string]$SourceDir,
        [string]$ProtoDir,
        [string[]]$ProtoFiles
    )
    
    if ($Verbose) {
        Write-Host "Generating protobuf for $Project ($Service)..." -ForegroundColor Yellow
    }

    if (-not (Test-Path $Project)) {
        Write-Warning "Project directory $Project not found, skipping..."
        return
    }

    Push-Location $Project

    try {
        # Print current directory for debugging
        Write-Host "Current directory: $(Get-Location); SourceDir: $SourceDir, ProtoDir: $ProtoDir" -ForegroundColor Magenta

        # Ensure pkg/proto exists before running protoc
        $protoOutDir = Join-Path $SourceDir "pkg/proto"
        if (-not (Test-Path $protoOutDir)) {
            New-Item -ItemType Directory -Force -Path $protoOutDir | Out-Null
        }


        # Find all .proto files in the proto directory
        if (-not (Test-Path $ProtoDir)) {
            Write-Warning "No proto directory found in $Project, skipping..."
            return
        }
        $allProtoFiles = Get-ChildItem -Path $ProtoDir -Filter *.proto -Recurse | Select-Object -ExpandProperty FullName
        if (-not $allProtoFiles) {
            Write-Warning "No .proto files found in $ProtoDir for $Project, skipping..."
            return
        }


        # Check if any of the expected proto files exist
        $filesExist = $false
        foreach ($file in $ProtoFiles) {
            if (Test-Path (Join-Path $ProtoDir $file)) {
                $filesExist = $true
                break
            }
        }
        if (-not $filesExist) {
            Write-Warning "None of the expected proto files found for $Project, skipping..."
            return
        }


        # Generate Go protobuf files
        if ($Verbose) {
            Write-Host "  Running protoc for Go: $($ProtoFiles -join ', ')..." -ForegroundColor Cyan
        }

        # Dynamically find protoc and its include directory
        $protocPath = (Get-Command protoc).Source
        if (-not $protocPath) {
            throw "protoc not found in PATH"
        }
        $protocDir = Split-Path $protocPath -Parent
        $protocInclude = Join-Path (Split-Path $protocDir -Parent) "include"

        $protocArgs = @(
            "--go_out=$protoOutDir"
            "--go_opt=paths=source_relative"
            "--go-grpc_out=$protoOutDir" 
            "--go-grpc_opt=paths=source_relative"
            "--proto_path=$ProtoDir"
            "--proto_path=$protocInclude"
        ) + $ProtoFiles

        & protoc $protocArgs

        if ($LASTEXITCODE -ne 0) {
            throw "Protoc Go generation failed for $Project"
        }

        # No move needed: protoc will generate files in the correct subdirectory based on go_package

        # Generate C# protobuf files if frontend directory exists
        if (Test-Path "frontend") {
            if ($Verbose) {
                Write-Host "  C# protobuf generation handled by MSBuild (.csproj files)" -ForegroundColor Cyan
            }
            # For C#, we rely on MSBuild and the existing <Protobuf> items in .csproj files
            # The standardized structure will be:
            # frontend/src/Generated/Proto/{Service}/V1/*.cs (when we update .csproj files)
            # For now, C# generation happens during build via Grpc.Tools package
            if ($Verbose) {
                Write-Host "    ✓ C# generation configured via MSBuild" -ForegroundColor Green
            }
        }

        Write-Host "  ✓ Generated protobuf files for $Project" -ForegroundColor Green
    }
    catch {
        Write-Error "Error generating protobuf for $Project`: $_"
        throw
    }
    finally {
        Pop-Location
    }
}

try {
    # Generate for tasker-core
    Generate-Go-Proto -Project "tasker-core" -Service "taskcore" -SourceDir "backend" -ProtoDir "proto" -ProtoFiles @("taskcore/v1/task.proto")

    # Generate for inventory-core  
    Generate-Go-Proto -Project "inventory-core" -Service "inventory" -SourceDir "backend" -ProtoDir "proto" -ProtoFiles @("inventory/v1/inventory.proto")

    # Generate for shared
    Generate-Go-Proto -Project "shared" -Service "events" -SourceDir "./" -ProtoDir "proto" -ProtoFiles @("events/v1/events.proto")

    # Generate for home-manager (has multiple proto files)
    if ((Test-Path "home-manager") -and (Test-Path "home-manager/proto/hometasker/v1/config.proto")) {
        Write-Host "Generating protobuf for home-manager (hometasker)..." -ForegroundColor Yellow
        
        Push-Location "home-manager"
        
        try {
            $protoOutDir = "backend/pkg/proto"
            $targetDir = "backend/pkg/proto/hometasker/v1"
            if (-not (Test-Path $protoOutDir)) {
                New-Item -ItemType Directory -Force -Path $protoOutDir | Out-Null
            }
            New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
            

            # Dynamically find protoc and its include directory
            $protocPath = (Get-Command protoc).Source
            if (-not $protocPath) {
                throw "protoc not found in PATH"
            }
            $protocDir = Split-Path $protocPath -Parent
            $protocInclude = Join-Path (Split-Path $protocDir -Parent) "include"

            $protocArgs = @(
                "--go_out=backend/pkg/proto"
                "--go_opt=paths=source_relative"
                "--go-grpc_out=backend/pkg/proto"
                "--go-grpc_opt=paths=source_relative" 
                "--proto_path=proto"
                "--proto_path=$protocInclude"
                "proto/hometasker/v1/config.proto"
                "proto/hometasker/v1/cooking.proto"
                "proto/hometasker/v1/hometasker_service.proto"
                "proto/hometasker/v1/state.proto"
                "proto/hometasker/v1/tasks.proto"
            )
            
            & protoc $protocArgs
            
            if ($LASTEXITCODE -ne 0) {
                throw "Protoc generation failed for home-manager"
            }
            
            Get-ChildItem -Path "backend/pkg/proto" -Filter "*.pb.go" -Recurse |
            Where-Object { $_.FullName -notlike "*/v1/*" } |
            ForEach-Object {
                $destination = Join-Path $targetDir $_.Name
                Move-Item $_.FullName $destination -Force
            }
            
            Write-Host "  ✓ Generated protobuf files for home-manager" -ForegroundColor Green
            
        }
        finally {
            Pop-Location
        }
    }

    Write-Host ""
    Write-Host "✓ Protobuf generation complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Generated files structure:" -ForegroundColor Cyan
    Write-Host "  tasker-core/backend/pkg/proto/taskcore/v1/*.pb.go"
    Write-Host "  inventory-core/backend/pkg/proto/inventory/v1/*.pb.go"  
    Write-Host "  shared/pkg/proto/events/v1/*.pb.go"
    Write-Host "  home-manager/backend/pkg/proto/hometasker/v1/*.pb.go"
    Write-Host ""
    Write-Host "Note: These generated files are git-ignored and will be regenerated in CI/CD." -ForegroundColor Yellow

}
catch {
    Write-Error "Protobuf generation failed: $_"
    exit 1
}
