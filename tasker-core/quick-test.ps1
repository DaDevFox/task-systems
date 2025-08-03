#!/usr/bin/env pwsh
# Quick test script for staging and ID resolution functionality

param(
    [string]$ServerPort = "8084"
)

$ErrorActionPreference = "Stop"

# Colors for output
$Green = "`e[32m"
$Red = "`e[31m"
$Blue = "`e[34m"
$Reset = "`e[0m"

function Write-Step {
    param([string]$Message)
    Write-Host "${Blue}[STEP]${Reset} $Message"
}

function Write-Success {
    param([string]$Message)
    Write-Host "${Green}[SUCCESS]${Reset} $Message"
}

function Write-Error {
    param([string]$Message)
    Write-Host "${Red}[ERROR]${Reset} $Message"
}

# Build the binaries
Write-Step "Building server and client..."
go build -o server.exe ./cmd/server
go build -o client.exe ./cmd/client
Write-Success "Binaries built successfully"

# Start the server in background
Write-Step "Starting server on port $ServerPort..."
$serverProcess = Start-Process -FilePath ".\server.exe" -ArgumentList "--port", $ServerPort -PassThru -NoNewWindow
Start-Sleep -Seconds 3

# Check if server started
if ($serverProcess.HasExited) {
    Write-Error "Server failed to start"
    exit 1
}
Write-Success "Server started with PID $($serverProcess.Id)"

# Define client command with server address
$ClientCmd = ".\client.exe --server localhost:$ServerPort"

try {
    Write-Step "Testing core functionality..."
    
    # Create user
    $userOutput = Invoke-Expression "$ClientCmd user create quicktest@example.com 'Quick Test User'" 2>&1
    $userId = ($userOutput | Select-String "ID: ([a-f0-9-]+)").Matches[0].Groups[1].Value
    Write-Success "Created user with ID: $userId"
    
    # Create tasks
    $task1Output = Invoke-Expression "$ClientCmd add 'Task One' -d 'First task' -u quicktest@example.com" 2>&1
    $task1Id = ($task1Output | Select-String "ID: ([a-f0-9-]+)").Matches[0].Groups[1].Value
    
    $task2Output = Invoke-Expression "$ClientCmd add 'Task Two' -d 'Second task' -u quicktest@example.com" 2>&1
    $task2Id = ($task2Output | Select-String "ID: ([a-f0-9-]+)").Matches[0].Groups[1].Value
    
    Write-Success "Created tasks: $task1Id, $task2Id"
    
    # Test partial ID resolution
    $partialId = $task1Id.Substring(0, 4)
    $getOutput = Invoke-Expression "$ClientCmd get $partialId" 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Partial ID resolution working ($partialId -> $task1Id)"
    } else {
        Write-Error "Partial ID resolution failed: $getOutput"
    }
    
    # Test staging with location
    $stagingOutput1 = Invoke-Expression "$ClientCmd stage move $task1Id --location project --location phase1" 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Task moved to staging with location"
    } else {
        Write-Error "Staging with location failed: $stagingOutput1"
    }
    
    # Test staging with destination
    $stagingOutput2 = Invoke-Expression "$ClientCmd stage move $task2Id --destination $task1Id" 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Task moved to staging with destination dependency"
    } else {
        Write-Error "Staging with destination failed: $stagingOutput2"
    }
    
    # Verify tasks are in staging
    $stagingList = Invoke-Expression "$ClientCmd list -u quicktest@example.com --stage staging" 2>&1
    $stagingCount = ($stagingList | Select-String "(\d+) total").Matches[0].Groups[1].Value
    
    if ($stagingCount -eq "2") {
        Write-Success "Both tasks correctly moved to staging"
    } else {
        Write-Error "Expected 2 tasks in staging, found $stagingCount"
    }
    
    Write-Success "All core functionality tests passed! ✅"

} catch {
    Write-Error "Test failed: $_"
    exit 1
} finally {
    # Cleanup: Stop the server
    Write-Step "Cleaning up..."
    if ($serverProcess -and !$serverProcess.HasExited) {
        $serverProcess.Kill()
        $serverProcess.WaitForExit(5000)
        Write-Success "Server stopped"
    }
    
    # Clean up binaries
    if (Test-Path "server.exe") { Remove-Item "server.exe" -Force }
    if (Test-Path "client.exe") { Remove-Item "client.exe" -Force }
    Write-Success "Cleanup completed"
}

Write-Success "Quick test completed successfully! 🎉"
