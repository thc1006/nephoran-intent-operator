#!/usr/bin/env pwsh

# Deep scan of pkg/oran/o2/providers to identify all compilation errors
Write-Host "=== Scanning O2 Providers Package for Errors ===" -ForegroundColor Cyan

$ErrorActionPreference = "Continue"
$providersPath = "pkg/oran/o2/providers"

if (-not (Test-Path $providersPath)) {
    Write-Host "ERROR: Providers path does not exist: $providersPath" -ForegroundColor Red
    exit 1
}

# 1. Find all Go files in providers
Write-Host "`n1. Go files in providers package:" -ForegroundColor Yellow
Get-ChildItem -Path $providersPath -Recurse -Filter "*.go" | ForEach-Object {
    Write-Host "  - $($_.FullName.Replace((Get-Location), '.'))"
}

# 2. Run go mod tidy to ensure clean dependencies
Write-Host "`n2. Running go mod tidy..." -ForegroundColor Yellow
go mod tidy

# 3. Check for missing types and interfaces
Write-Host "`n3. Checking for compilation errors..." -ForegroundColor Yellow
$buildOutput = go build -v ./pkg/oran/o2/providers/... 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "BUILD ERRORS FOUND:" -ForegroundColor Red
    $buildOutput | Write-Host -ForegroundColor Red
} else {
    Write-Host "No build errors found." -ForegroundColor Green
}

# 4. Run tests to identify runtime issues
Write-Host "`n4. Running tests..." -ForegroundColor Yellow
$testOutput = go test -v ./pkg/oran/o2/providers/... 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "TEST ERRORS FOUND:" -ForegroundColor Red
    $testOutput | Write-Host -ForegroundColor Red
} else {
    Write-Host "All tests passed." -ForegroundColor Green
}

# 5. Check for undefined types using grep
Write-Host "`n5. Scanning for undefined types and interfaces..." -ForegroundColor Yellow
$grepPatterns = @(
    "undefined:",
    "cannot use.*as.*in",
    "unknown field",
    "missing method",
    "interface.*not implemented"
)

foreach ($pattern in $grepPatterns) {
    Write-Host "Searching for pattern: $pattern" -ForegroundColor Gray
    $results = Select-String -Path "$providersPath\*.go" -Pattern $pattern -AllMatches 2>$null
    if ($results) {
        $results | ForEach-Object { Write-Host "  $($_.Filename):$($_.LineNumber): $($_.Line)" -ForegroundColor Yellow }
    }
}

Write-Host "`n=== Scan Complete ===" -ForegroundColor Cyan