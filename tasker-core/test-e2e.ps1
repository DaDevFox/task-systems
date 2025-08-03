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

    # Test 3: List tasks
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

    # Test 7: Test help commands
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

    # Test 8: Test DAG help
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
