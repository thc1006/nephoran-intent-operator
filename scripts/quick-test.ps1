#!/usr/bin/env pwsh
# Quick test script for immediate feedback

param(
    [string]$Test = "",     # Specific test to run
    [switch]$Verbose       # Verbose output
)

$Red = "`e[31m"
$Green = "`e[32m"
$Yellow = "`e[33m"
$Reset = "`e[0m"

Write-Host "${Yellow}🧪 Quick Test Runner${Reset}"
Write-Host ""

# Clean test cache
go clean -testcache

if ($Test) {
    Write-Host "Running specific test: ${Yellow}$Test${Reset}"
    if ($Verbose) {
        go test -v -race -run $Test ./...
    } else {
        go test -run $Test ./...
    }
} else {
    Write-Host "Running all tests with race detection..."
    go test -race -short ./...
}

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "${Green}✅ Tests passed!${Reset}"
} else {
    Write-Host ""  
    Write-Host "${Red}❌ Tests failed!${Reset}"
    Write-Host "Use: ${Yellow}.\scripts\test-local-ci.ps1 -Fast -Verbose${Reset} for detailed analysis"
}