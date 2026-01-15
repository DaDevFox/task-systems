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

    $protocGenGoCmd = (Get-Command protoc-gen-go -ErrorAction SilentlyContinue)
    $protocGenGoGrpcCmd = (Get-Command protoc-gen-go-grpc -ErrorAction SilentlyContinue)

    if (-not $protocGenGoCmd) {
        Write-Host "protoc-gen-go not found; installing..." -ForegroundColor Yellow
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        $protocGenGoCmd = (Get-Command protoc-gen-go -ErrorAction SilentlyContinue)
    }
    if (-not $protocGenGoGrpcCmd) {
        Write-Host "protoc-gen-go-grpc not found; installing..." -ForegroundColor Yellow
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
        $protocGenGoGrpcCmd = (Get-Command protoc-gen-go-grpc -ErrorAction SilentlyContinue)
    }

    $protocGenGo = $protocGenGoCmd.Source
    $protocGenGoGrpc = $protocGenGoGrpcCmd.Source 


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
        # Locate protoc binary
        $protocPath = (Get-Command protoc -ErrorAction SilentlyContinue).Source
        if (-not $protocPath) {
            throw "protoc not found in PATH"
        }
        $protocDir = Split-Path $protocPath -Parent

        # Probe candidate include directories for well-known types (google/protobuf/*)
        $candidateIncludesRaw = @()
        try {
            $candidateIncludesRaw += (Join-Path $protocDir "include")
        } catch { }
        try {
            $candidateIncludesRaw += (Join-Path (Split-Path $protocDir -Parent) "include")
        } catch { }

        if ($env:GOPATH) { $candidateIncludesRaw += (Join-Path $env:GOPATH "pkg/mod") }
        $goModCache = (& go env GOMODCACHE 2>$null) -as [string]
        if ($goModCache) { $candidateIncludesRaw += $goModCache }
        try { $candidateIncludesRaw += (Join-Path $PWD "proto") } catch { }

        # Filter to existing directories and unique
        $candidateIncludes = $candidateIncludesRaw | Where-Object { $_ -and (Test-Path $_) } | Select-Object -Unique

        $foundInclude = $null
        foreach ($inc in $candidateIncludes) {
            if (-not $inc) { continue }
            if (-not (Test-Path $inc)) { continue }

            # Direct hit for well-known types
            $timestampProto = Join-Path $inc "google\protobuf\timestamp.proto"
            if (Test-Path $timestampProto) {
                $foundInclude = $inc
                break
            }

            # Search one level deeper for google/protobuf under subdirs (common in GOPATH/pkg/mod)
            try {
                $subdirs = Get-ChildItem -Path $inc -Directory -ErrorAction SilentlyContinue
                foreach ($sub in $subdirs) {
                    if (Test-Path (Join-Path $sub.FullName "google\protobuf\timestamp.proto")) {
                        $foundInclude = $sub.FullName
                        break
                    }
                }
                if ($foundInclude) { break }
            } catch { }
        }

        if (-not $foundInclude) {
            Write-Warning "Could not find protoc include dir containing google/protobuf/*. Attempting to download well-known types into .tools/protoc_include..." 

            $localIncludeRoot = Join-Path $PWD ".tools\protoc_include"
            $localGoogleDir = Join-Path $localIncludeRoot "google\protobuf"
            if (-not (Test-Path $localGoogleDir)) {
                New-Item -ItemType Directory -Force -Path $localGoogleDir | Out-Null
            }

            $wellKnownFiles = @(
                'timestamp.proto', 'duration.proto', 'empty.proto', 'wrappers.proto', 'any.proto', 'struct.proto', 'field_mask.proto', 'source_context.proto', 'descriptor.proto'
            )

            $baseRaw = 'https://raw.githubusercontent.com/protocolbuffers/protobuf/main/src/google/protobuf'
            $downloaded = $false
            foreach ($f in $wellKnownFiles) {
                $url = "$baseRaw/$f"
                $dest = Join-Path $localGoogleDir $f
                try {
                    Write-Host "Downloading $url -> $dest" -ForegroundColor Cyan
                    Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing -ErrorAction Stop
                    $downloaded = $true
                } catch {
                    Write-Warning ("Failed to download {0} from {1}: {2}" -f $f, $url, $_)
                }
            }

            if ($downloaded) {
                $foundInclude = $localIncludeRoot
                Write-Host "Downloaded well-known protos to $foundInclude" -ForegroundColor Green
            } else {
                Write-Warning "Failed to download well-known protos; protoc may still fail." 
            }
        }

        $protocArgs = @(
            "--go_out=$protoOutDir",
            "--go_opt=paths=source_relative",
            "--go-grpc_out=$protoOutDir",
            "--go-grpc_opt=paths=source_relative",
            "--plugin=protoc-gen-go=$protocGenGo",
            "--plugin=protoc-gen-go-grpc=$protocGenGoGrpc"
        )

        # Add proto_path entries: project proto dir first
        $protoPathEntries = @()
        $protoPathEntries += (Resolve-Path -Path $ProtoDir).Path
        if ($foundInclude) { $protoPathEntries += (Resolve-Path -Path $foundInclude).Path }

        # Add --proto_path entries: project proto dir, any found include for well-known types, and directories containing each proto file
        $protoPathEntries = @()
        try { $protoPathEntries += (Resolve-Path -Path $ProtoDir).Path } catch { $protoPathEntries += $ProtoDir }
        if ($foundInclude) { try { $protoPathEntries += (Resolve-Path -Path $foundInclude).Path } catch { $protoPathEntries += $foundInclude } }

        foreach ($pf in $ProtoFiles) {
            # Get the directory part of the proto file and add it as a proto_path so imports like "user.proto" resolve when referenced from the same folder
            $pd = Split-Path $pf -Parent
            if ($pd -and $pd -ne '') {
                $candidate = Join-Path $ProtoDir $pd
                if (Test-Path $candidate) { $protoPathEntries += (Resolve-Path -Path $candidate).Path }
            }
        }

        $protoPathEntries = $protoPathEntries | Select-Object -Unique
        foreach ($ppe in $protoPathEntries) { $protocArgs += "--proto_path=$ppe" }

        # Append proto filenames (relative paths from ProtoDir)
        $protocArgs = $protocArgs + $ProtoFiles

        # Run protoc from the ProtoDir to ensure relative imports resolve correctly
        $cwdBefore = Get-Location
        try {
            Set-Location -Path $ProtoDir
            $localProtoArgs = @()
            $localProtoArgs += "--go_out=$protoOutDir"
            $localProtoArgs += "--go_opt=paths=source_relative"
            $localProtoArgs += "--go-grpc_out=$protoOutDir"
            $localProtoArgs += "--go-grpc_opt=paths=source_relative"
            $localProtoArgs += "--plugin=protoc-gen-go=$protocGenGo"
            $localProtoArgs += "--plugin=protoc-gen-go-grpc=$protocGenGoGrpc"
            $localProtoArgs += "--proto_path=."
            if ($foundInclude) { $localProtoArgs += "--proto_path=$foundInclude" }
            $localProtoArgs += $ProtoFiles

            Write-Host "Invoking protoc (from $PWD): $protocPath $($localProtoArgs -join ' ')" -ForegroundColor Magenta
            & $protocPath $localProtoArgs
            if ($LASTEXITCODE -ne 0) {
                throw "Protoc Go generation failed for $Project"
            }
        } finally {
            Set-Location -Path $cwdBefore
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

    # Generate for user-core
    Generate-Go-Proto -Project "user-core" -Service "usercore" -SourceDir "backend" -ProtoDir "proto" -ProtoFiles @("usercore/v1/user.proto", "usercore/v1/bootstrap_users.proto")

    # Generate for shared
    Generate-Go-Proto -Project "shared" -Service "events" -SourceDir "./" -ProtoDir "proto" -ProtoFiles @("events/v1/events.proto")

    # Generate for workflows
    Generate-Go-Proto -Project "workflows" -Service "workflows" -SourceDir "backend" -ProtoDir "proto" -ProtoFiles @("workflows/v1/workflows_service.proto", "workflows/v1/cooking.proto", "workflows/v1/state.proto", "workflows/v1/config.proto", "workflows/v1/tasks.proto")

    Write-Host ""
    Write-Host "✓ Protobuf generation complete!" -ForegroundColor Green
    Write-Host ""
    # Add mock proto dependencies path for validation
$mockProtoPath = "/mocked/mocks/ps1-deps"
if (-not (Test-Path $mockProtoPath)) {
    New-Item -ItemType Directory -Force -Path $mockProtoPath | Out-Null
    Write-Host "Mock proto path created for validation: $mockProtoPath" -ForegroundColor Yellow
}

Write-Host "Generated files structure:" -ForegroundColor Cyan
    Write-Host "  tasker-core/backend/pkg/proto/taskcore/v1/*.pb.go"
    Write-Host "  inventory-core/backend/pkg/proto/inventory/v1/*.pb.go"  
    Write-Host "  user-core/backend/pkg/proto/usercore/v1/*.pb.go"
    Write-Host "  shared/pkg/proto/events/v1/*.pb.go"
    Write-Host "  workflows/backend/pkg/proto/workflows/v1/*.pb.go"
    Write-Host ""
    Write-Host "Note: These generated files are git-ignored and will be regenerated in CI/CD." -ForegroundColor Yellow

}
catch {
    Write-Error "Protobuf generation failed: $_"
    exit 1
}
