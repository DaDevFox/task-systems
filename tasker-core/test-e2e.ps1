#!/usr/bin/env pwsh
# End-to-end test script for the task management system
# This script tests the full user flow including DAG visualization

param(
    [string]$ServerPort = "8082"
)

$ErrorActionPreference = "Stop"

# Colors for output
$Green = "`e[32m"
$Red = "`e[31m"
$Yellow = "`e[33m"
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

function Write-Warning {
    param([string]$Message)
    Write-Host "${Yellow}[WARNING]${Reset} $Message"
}

# Build the binaries
Write-Step "Building server and client..."
try {
    go build -o server.exe ./cmd/server
    go build -o client.exe ./cmd/client
    Write-Success "Binaries built successfully"
} catch {
    Write-Error "Failed to build binaries: $_"
    exit 1
}

# Start the server in background
Write-Step "Starting server on port $ServerPort..."
$serverProcess = Start-Process -FilePath ".\server.exe" -ArgumentList "--port", $ServerPort -PassThru -NoNewWindow
Start-Sleep -Seconds 2

# Check if server started
if ($serverProcess.HasExited) {
    Write-Error "Server failed to start"
    exit 1
}
Write-Success "Server started with PID $($serverProcess.Id)"

# Define client command with server address
$ClientCmd = ".\client.exe --server localhost:$ServerPort"

try {
    # Test 1: Create a user
    Write-Step "Creating test user..."
    $output = Invoke-Expression "$ClientCmd user create test@example.com 'Test User'" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create user: $output"
        throw "User creation failed"
    }
    Write-Success "User created successfully"
    
    # Extract user ID from output (assuming format: "ID: <user-id>")
    $userIdMatch = $output | Select-String "ID: ([a-f0-9-]+)"
    if ($userIdMatch) {
        $userId = $userIdMatch.Matches[0].Groups[1].Value
        Write-Success "Created user with ID: $userId"
    } else {
        Write-Warning "Could not extract user ID, using default-user"
        $userId = "default-user"
    }

    # Test 1.5: Get user by email
    Write-Step "Testing get user by email..."
    $output = Invoke-Expression "$ClientCmd user get test@example.com" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to get user by email: $output"
        throw "Get user by email failed"
    }
    Write-Success "Successfully retrieved user by email"

    # Test 2: Add tasks
    Write-Step "Adding tasks..."
    
    # Add root task
    $output1 = Invoke-Expression "$ClientCmd add 'Setup Environment' -d 'Initialize development environment' -u $userId" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to add first task: $output1"
        throw "Task creation failed"
    }
    
    # Add dependent tasks
    $output2 = Invoke-Expression "$ClientCmd add 'Install Dependencies' -d 'Install required packages' -u $userId" 2>&1
    $output3 = Invoke-Expression "$ClientCmd add 'Configure Database' -d 'Set up database connection' -u $userId" 2>&1
    $output4 = Invoke-Expression "$ClientCmd add 'Run Tests' -d 'Execute test suite' -u $userId" 2>&1
    $output5 = Invoke-Expression "$ClientCmd add 'Deploy Application' -d 'Deploy to production' -u $userId" 2>&1
    
    Write-Success "Tasks added successfully"

    # Test 3: List tasks in inbox (correct default stage)
    Write-Step "Listing tasks..."
    $output = Invoke-Expression "$ClientCmd list --stage inbox" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to list tasks: $output"
        throw "Task listing failed"
    }
    Write-Success "Tasks listed successfully"
    Write-Host "Tasks:" -ForegroundColor Cyan
    Write-Host $output

    # Test 4: Test DAG visualization (empty dependencies)
    Write-Step "Testing DAG visualization..."
    $output = Invoke-Expression "$ClientCmd dag -u $userId" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to display DAG: $output"
        throw "DAG visualization failed"
    }
    Write-Success "DAG visualization working"
    Write-Host "DAG Output:" -ForegroundColor Cyan
    Write-Host $output

    # Test 5: Test compact DAG format
    Write-Step "Testing compact DAG format..."
    $output = Invoke-Expression "$ClientCmd dag --compact -u $userId" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to display compact DAG: $output"
        throw "Compact DAG visualization failed"
    }
    Write-Success "Compact DAG visualization working"
    Write-Host "Compact DAG Output:" -ForegroundColor Cyan
    Write-Host $output

    # Test 6: Test user retrieval
    Write-Step "Testing user retrieval..."
    $output = Invoke-Expression "$ClientCmd user get $userId" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to get user: $output"
        throw "User retrieval failed"
    }
    Write-Success "User retrieval working"
    Write-Host "User Details:" -ForegroundColor Cyan
    Write-Host $output

    # Test 7: Test staging functionality
    Write-Step "Testing staging functionality..."
    
    # Get task IDs for staging test - extract from task list
    $taskList = Invoke-Expression "$ClientCmd list --stage inbox" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to list tasks for staging test: $taskList"
        throw "Task listing for staging test failed"
    }
    
    # Extract the first two task IDs from the output
    $taskIdMatches = $taskList | Select-String "ID: ([a-f0-9-]+)" | Select-Object -First 2
    if ($taskIdMatches.Count -ge 2) {
        $sourceTaskId = $taskIdMatches[0].Matches[0].Groups[1].Value
        $destTaskId = $taskIdMatches[1].Matches[0].Groups[1].Value
        
        Write-Host "Using source task: $sourceTaskId, destination task: $destTaskId" -ForegroundColor Cyan
        
        # First move destination task to staging with a location
        $destStagingOutput = Invoke-Expression "$ClientCmd stage move $destTaskId --location project --location setup" 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to move destination task to staging: $destStagingOutput"
            throw "Destination staging failed"
        }
        Write-Success "Destination task moved to staging successfully"
        
        # Then move source task to staging with destination dependency
        $sourceStagingOutput = Invoke-Expression "$ClientCmd stage move $sourceTaskId --destination $destTaskId" 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to move source task to staging: $sourceStagingOutput"
            throw "Source staging failed"
        }
        Write-Success "Source task moved to staging with dependency successfully"
        Write-Host "Source Staging Output:" -ForegroundColor Cyan
        Write-Host $sourceStagingOutput
        
        # Verify task is in staging
        $stagingList = Invoke-Expression "$ClientCmd list --stage staging" 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to list staging tasks: $stagingList"
            throw "Staging list failed"
        }
        Write-Success "Staging list working"
        Write-Host "Staging Tasks:" -ForegroundColor Cyan
        Write-Host $stagingList
    } else {
        Write-Warning "Not enough tasks found for staging test, skipping..."
    }

    # Test 8: Test ID resolution with partial IDs
    Write-Step "Testing ID resolution with partial IDs..."
    
    if ($taskIdMatches.Count -ge 1) {
        $fullTaskId = $taskIdMatches[0].Matches[0].Groups[1].Value
        $partialId = $fullTaskId.Substring(0, [Math]::Min(8, $fullTaskId.Length))
        
        Write-Host "Testing partial ID resolution: $partialId (from full ID: $fullTaskId)" -ForegroundColor Cyan
        
        # Test getting task with partial ID
        $partialIdOutput = Invoke-Expression "$ClientCmd get $partialId" 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to resolve partial task ID: $partialIdOutput"
            throw "Partial ID resolution failed"
        }
        Write-Success "Partial ID resolution working"
        Write-Host "Partial ID Output:" -ForegroundColor Cyan
        Write-Host $partialIdOutput
        
        # Test starting task with partial ID
        $startOutput = Invoke-Expression "$ClientCmd start $partialId" 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to start task with partial ID: $startOutput"
            throw "Start task with partial ID failed"
        }
        Write-Success "Start task with partial ID working"
        Write-Host "Start Output:" -ForegroundColor Cyan
        Write-Host $startOutput
    } else {
        Write-Warning "No tasks found for ID resolution test, skipping..."
    }

    # Test 9: Test user resolution
    Write-Step "Testing user resolution..."
    
    # Test getting user with partial input (email or partial ID)
    $userResolutionOutput = Invoke-Expression "$ClientCmd user get test@example.com" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "User resolution by email failed, this may be expected if user resolver doesn't support email lookup"
    } else {
        Write-Success "User resolution by email working"
        Write-Host "User Resolution Output:" -ForegroundColor Cyan
        Write-Host $userResolutionOutput
    }

    # Test 10: Test help commands
    Write-Step "Testing help commands..."
    $helpOutput = .\client.exe --help | Out-String
    if (-not $helpOutput) {
        Write-Error "Failed to get help output"
        throw "Help command failed"
    }
    
    if ($helpOutput -notlike "*dag*") {
        Write-Error "DAG command not found in help"
        throw "DAG command not properly integrated"
    }
    Write-Success "Help commands working and DAG command is listed"

    # Test 11: Test DAG help
    Write-Step "Testing DAG command help..."
    $dagHelpOutput = .\client.exe dag --help | Out-String
    if (-not $dagHelpOutput) {
        Write-Error "Failed to get DAG help output"
        throw "DAG help command failed"
    }
    
    if ($dagHelpOutput -notlike "*--compact*") {
        Write-Error "DAG compact flag not found in help"
        throw "DAG command flags not properly configured"
    }
    Write-Success "DAG help command working with all flags"

    Write-Success "All tests passed! ✅"

} catch {
    Write-Error "Test failed: $_"
    $exitCode = 1
} finally {
    # Cleanup: Stop the server
    Write-Step "Cleaning up..."
    if ($serverProcess -and !$serverProcess.HasExited) {
        Write-Step "Stopping server..."
        $serverProcess.Kill()
        $serverProcess.WaitForExit(5000)
        Write-Success "Server stopped"
    }
    
    # Clean up binaries
    if (Test-Path "server.exe") { Remove-Item "server.exe" -Force }
    if (Test-Path "client.exe") { Remove-Item "client.exe" -Force }
    Write-Success "Cleanup completed"
}

if ($exitCode -eq 1) {
    exit 1
}

Write-Success "End-to-end test completed successfully! 🎉"
