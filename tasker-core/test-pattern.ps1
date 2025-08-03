cd e:\Source\Systems\task-systems\tasker-core
$help = .\client.exe --help 2>&1
Write-Host "Help content:"
Write-Host $help
Write-Host "---"
if ($help -match 'dag.*View task dependency graph') {
    Write-Host "PATTERN FOUND"
} else {
    Write-Host "PATTERN NOT FOUND"
}
