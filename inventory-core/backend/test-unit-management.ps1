# Test script to demonstrate unit management functionality
# This script shows how the unit management endpoints work

Write-Host "=== Unit Management Test Script ===" -ForegroundColor Green

Write-Host "`nBuilding the server..." -ForegroundColor Yellow
go build -o server.exe ./cmd/server
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

Write-Host "Build successful!" -ForegroundColor Green

Write-Host "`nRunning unit management integration tests..." -ForegroundColor Yellow
go test -v ./test/integration/ -run TestUnitManagement

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ Unit Management Implementation Complete!" -ForegroundColor Green
    Write-Host "The following functionality has been implemented and tested:" -ForegroundColor Cyan
    Write-Host "  • ListUnits RPC - retrieves all unit definitions" -ForegroundColor White
    Write-Host "  • AddUnit RPC - creates new units with validation" -ForegroundColor White
    Write-Host "  • GetUnit RPC - retrieves individual unit details" -ForegroundColor White
    Write-Host "  • UpdateUnit RPC - modifies existing units" -ForegroundColor White
    Write-Host "  • DeleteUnit RPC - removes units with dependency checks" -ForegroundColor White
    Write-Host "  • Input validation and error handling" -ForegroundColor White
    Write-Host "  • Default units (kg, g, L, mL, count) pre-populated" -ForegroundColor White
}
else {
    Write-Host "`n❌ Tests failed!" -ForegroundColor Red
}

# Clean up
if (Test-Path "server.exe") {
    Remove-Item "server.exe"
}
