#!/usr/bin/env pwsh

# Execute the providers scan
& ".\scan_providers.ps1"

# Also check the specific providers directory structure
Write-Host "`n=== Directory Structure Analysis ===" -ForegroundColor Cyan
if (Test-Path "pkg/oran/o2/providers") {
    tree /F pkg\oran\o2\providers 2>$null
    if ($LASTEXITCODE -ne 0) {
        # Fallback for systems without tree command
        Get-ChildItem -Path "pkg/oran/o2/providers" -Recurse | ForEach-Object {
            $indent = "  " * ($_.FullName.Split('\').Count - (Get-Location).Path.Split('\').Count - 3)
            Write-Host "$indent$($_.Name)"
        }
    }
} else {
    Write-Host "Providers directory does not exist!" -ForegroundColor Red
}

# Check for any provider-related files in other locations
Write-Host "`n=== Searching for provider-related files ===" -ForegroundColor Cyan
Get-ChildItem -Recurse -Filter "*provider*" -Name | Where-Object { $_ -like "*.go" } | ForEach-Object {
    Write-Host "Found: $_" -ForegroundColor Yellow
}