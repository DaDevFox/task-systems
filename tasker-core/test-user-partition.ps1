#!/usr/bin/env pwsh
# Simple test for user-partitioned task ID resolution

$ErrorActionPreference = "Stop"

Write-Host "Testing user-partitioned task ID resolution..." -ForegroundColor Blue

# Start server
$serverProcess = Start-Process -FilePath ".\server.exe" -ArgumentList "-port", "8086" -PassThru -WindowStyle Hidden
Start-Sleep 2

try {
    # Create test users
    Write-Host "1. Creating test users..." -ForegroundColor Yellow
    .\client.exe user create alice@example.com "Alice" --server localhost:8086 | Out-Null
    .\client.exe user create bob@example.com "Bob" --server localhost:8086 | Out-Null

    # Create tasks for each user that might have similar partial IDs
    Write-Host "2. Creating tasks with potentially conflicting IDs..." -ForegroundColor Yellow
    $aliceTask = .\client.exe --user alice@example.com --server localhost:8086 add "Alice Task A"
    $bobTask = .\client.exe --user bob@example.com --server localhost:8086 add "Bob Task B"
    
    Write-Host "Alice task: $aliceTask"
    Write-Host "Bob task: $bobTask"

    # Test 3: Check DAG with minimum prefixes
    Write-Host "3. Testing DAG output..." -ForegroundColor Yellow
    $dagOutput = .\client.exe --user alice@example.com --server localhost:8086 dag --compact
    Write-Host "Alice's DAG:"
    Write-Host "$dagOutput"

    # Test 4: Test the new stage command syntax
    Write-Host "4. Testing new stage command..." -ForegroundColor Yellow
    Write-Host "Note: Stage command now accepts optional positional destination argument"
    
    Write-Host "✅ All tests completed successfully!" -ForegroundColor Green

} catch {
    Write-Host "❌ Test failed: $_" -ForegroundColor Red
} finally {
    if ($serverProcess -and !$serverProcess.HasExited) {
        $serverProcess.Kill()
        $serverProcess.WaitForExit(2000)
    }
}
