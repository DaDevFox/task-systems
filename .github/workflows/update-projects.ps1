# This script updates the manual workflow dispatch options in the comprehensive-ci.yml file.
# It finds all .project.yml files, extracts their 'release-name', and updates the options list.
# To be run from the root of the repository.

$ErrorActionPreference = 'Stop'

# The workflow file to update
$workflowFile = ".github/workflows/comprehensive-ci.yml"

Write-Host "Finding project release names..."

# Find all .project.yml files and get their release-name property
# A simple regex is used to avoid requiring external modules.
$releaseNames = Get-ChildItem -Path . -Recurse -Filter ".project.yml" | ForEach-Object {
    (Get-Content $_.FullName) -match '^\s*release-name:\s*(.*)' | ForEach-Object { $Matches[1].Trim() }
} | Sort-Object

# Create the YAML-formatted string for the options
$options = @('all') + $releaseNames
$optionsString = ($options | ForEach-Object { "          - $_" }) -join "`n"

Write-Host "Updating workflow file: $workflowFile"

# Read the entire workflow file
$workflowContent = Get-Content $workflowFile -Raw

# Regex to find the options block and replace it.
# (?sm) allows . to match newlines and ^/$ to match start/end of lines.
$pattern = "(?sm)(project:\s*description:.*`n\s*required:.*`n\s*default:.*`n\s*type:\s*choice`n\s*options:`n)(?:          - .*\s*`n?)*"
$replacement = "`$1$optionsString`n"

$newWorkflowContent = $workflowContent -replace $pattern, $replacement

$newWorkflowContent | Set-Content -Path $workflowFile -Encoding utf8

Write-Host "Successfully updated project options in $workflowFile"
Write-Host "Please review and commit the changes."
