#!/usr/bin/env pwsh

Write-Host "=== Checking O2 Providers Structure ===" -ForegroundColor Cyan

# Check if the providers directory exists
$providersPath = "pkg/oran/o2/providers"
Write-Host "Checking path: $providersPath"

if (Test-Path $providersPath) {
    Write-Host "✓ Providers directory exists" -ForegroundColor Green
    
    # List all files
    Write-Host "`nFiles in providers directory:"
    Get-ChildItem -Path $providersPath -Recurse | ForEach-Object {
        $relativePath = $_.FullName.Replace((Get-Location).Path + "\", "")
        Write-Host "  $relativePath"
    }
} else {
    Write-Host "✗ Providers directory does not exist" -ForegroundColor Red
    
    # Check if parent directories exist
    Write-Host "Checking parent directories:"
    @("pkg", "pkg/oran", "pkg/oran/o2") | ForEach-Object {
        if (Test-Path $_) {
            Write-Host "  ✓ $_ exists" -ForegroundColor Green
        } else {
            Write-Host "  ✗ $_ missing" -ForegroundColor Red
        }
    }
}

# Also check for any Go compilation errors in the entire o2 package
Write-Host "`n=== Checking O2 Package Compilation ===" -ForegroundColor Cyan
$buildResult = go build ./pkg/oran/o2/... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ O2 package compiles successfully" -ForegroundColor Green
} else {
    Write-Host "✗ O2 package has compilation errors:" -ForegroundColor Red
    $buildResult | Write-Host -ForegroundColor Red
}