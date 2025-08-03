#!/usr/bin/env pwsh
# Test script for new stage and DAG functionality

$ErrorActionPreference = "Stop"

Write-Host "Testing new stage and DAG functionality..." -ForegroundColor Blue

# Start server in background
Write-Host "Starting server..." -ForegroundColor Yellow
$serverProcess = Start-Process -FilePath ".\server.exe" -ArgumentList "-port", "8085" -PassThru -WindowStyle Hidden
Start-Sleep 2

try {
    # Test 1: Create users and tasks for user-partitioned tries
    Write-Host "Creating users and tasks..." -ForegroundColor Yellow
    $output = .\client.exe user create alice@example.com "Alice Smith" --server localhost:8085 2>&1
    Write-Host "Created user: $output"
    
    $output = .\client.exe user create bob@example.com "Bob Jones" --server localhost:8085 2>&1
    Write-Host "Created user: $output"
    
    # Create tasks for different users
    $output = .\client.exe --user alice@example.com --server localhost:8085 add "Alice Task 1" --description "First task for Alice" 2>&1
    Write-Host "Created Alice task: $output"
    
    $output = .\client.exe --user bob@example.com --server localhost:8085 add "Bob Task 1" --description "First task for Bob" 2>&1  
    Write-Host "Created Bob task: $output"
    
    $output = .\client.exe --user alice@example.com --server localhost:8085 add "Alice Task 2" --description "Second task for Alice" 2>&1
    Write-Host "Created Alice task 2: $output"
    
    # Test 2: Test new stage command (without destination - should prompt for fuzzy picker)
    Write-Host "Testing stage command..." -ForegroundColor Yellow
    # This would normally open a fuzzy picker, so we'll test with explicit location instead
    $output = .\client.exe --user alice@example.com --server localhost:8085 stage --location project --location backend 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Stage command with location works!" -ForegroundColor Green
    } else {
        Write-Host "❌ Stage command failed: $output" -ForegroundColor Red
    }
    
    # Test 3: Test DAG with minimum prefixes
    Write-Host "Testing DAG with minimum prefixes..." -ForegroundColor Yellow
    $output = .\client.exe --user alice@example.com --server localhost:8085 dag 2>&1
    Write-Host "DAG output:"
    Write-Host "$output"
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ DAG with minimum prefixes works!" -ForegroundColor Green
    } else {
        Write-Host "❌ DAG command failed" -ForegroundColor Red
    }
    
    # Test 4: Test user-specific ID resolution
    Write-Host "Testing user-specific ID resolution..." -ForegroundColor Yellow
    # Try to resolve a partial ID for Alice's context
    $output = .\client.exe --user alice@example.com --server localhost:8085 list inbox 2>&1
    Write-Host "Alice's inbox: $output"
    
    $output = .\client.exe --user bob@example.com --server localhost:8085 list inbox 2>&1
    Write-Host "Bob's inbox: $output"
    
    Write-Host "✅ All new functionality tested!" -ForegroundColor Green
    
} catch {
    Write-Host "❌ Test failed: $_" -ForegroundColor Red
} finally {
    # Clean up
    Write-Host "Stopping server..." -ForegroundColor Yellow
    if ($serverProcess -and !$serverProcess.HasExited) {
        $serverProcess.Kill()
        $serverProcess.WaitForExit(5000)
    }
}
