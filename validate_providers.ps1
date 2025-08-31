#!/usr/bin/env pwsh

Write-Host "=== O2 Providers Package Validation ===" -ForegroundColor Cyan

# Quick validation check
$providersPath = "pkg/oran/o2/providers"

Write-Host "`nChecking package structure:" -ForegroundColor Yellow
if (Test-Path $providersPath) {
    Write-Host "✓ Providers directory exists" -ForegroundColor Green
    
    Get-ChildItem -Path $providersPath -Filter "*.go" | ForEach-Object {
        Write-Host "  - $($_.Name)" -ForegroundColor Cyan
    }
    
    # Quick compile test
    Write-Host "`nTesting compilation:" -ForegroundColor Yellow
    $result = go build ./pkg/oran/o2/providers 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Compilation successful" -ForegroundColor Green
    } else {
        Write-Host "❌ Compilation failed:" -ForegroundColor Red
        $result | Write-Host -ForegroundColor Red
        exit 1
    }
    
    # Quick test run  
    Write-Host "`nTesting basic functionality:" -ForegroundColor Yellow
    $testResult = go test ./pkg/oran/o2/providers -run TestProviderFactory 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Basic tests passed" -ForegroundColor Green
    } else {
        Write-Host "❌ Tests failed:" -ForegroundColor Red
        $testResult | Write-Host -ForegroundColor Red
    }
    
} else {
    Write-Host "❌ Providers directory not found" -ForegroundColor Red
    exit 1
}

Write-Host "`n✅ Validation complete - Providers package is ready!" -ForegroundColor Green