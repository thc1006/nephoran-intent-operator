#!/usr/bin/env pwsh
# Test optimization script with timeouts and parallel execution
# ITERATION #4 - AGENT #5: Performance Optimization

param(
    [string]$Package = "./...",
    [string]$Timeout = "10m",
    [switch]$Verbose,
    [switch]$Short,
    [switch]$Race,
    [int]$Parallel = 4,
    [switch]$SkipSlow
)

$ErrorActionPreference = "Continue"

Write-Host "=== Optimized Test Runner ===" -ForegroundColor Cyan
Write-Host "Timeout: $Timeout"
Write-Host "Parallel: $Parallel"
Write-Host ""

# Set environment variables for better test performance
$env:GOMAXPROCS = $Parallel
$env:GO_TEST_TIMEOUT_SCALE = "0.5"  # Reduce individual test timeouts

# Build test command
$testArgs = @(
    "test",
    "-timeout=$Timeout",
    "-count=1",
    "-parallel=$Parallel"
)

if ($Short) {
    $testArgs += "-short"
    Write-Host "Running in short mode (skipping slow tests)" -ForegroundColor Yellow
}

if ($Race) {
    $testArgs += "-race"
    Write-Host "Race detection enabled" -ForegroundColor Yellow
}

if ($Verbose) {
    $testArgs += "-v"
}

# Categories of tests by expected duration
$testCategories = @{
    "Quick" = @(
        "./api/...",
        "./pkg/generics/...",
        "./pkg/utils/...",
        "./internal/utils/..."
    )
    "Medium" = @(
        "./cmd/conductor/...",
        "./cmd/conductor-loop/...",
        "./cmd/conductor-watch/...",
        "./cmd/intent-ingest/...",
        "./cmd/llm-processor/...",
        "./internal/conductor/...",
        "./internal/loop/...",
        "./pkg/llm/...",
        "./pkg/nephio/...",
        "./pkg/porch/..."
    )
    "Slow" = @(
        "./planner/...",
        "./pkg/audit/...",
        "./pkg/auth/...",
        "./pkg/controllers/...",
        "./pkg/monitoring/...",
        "./pkg/oran/..."
    )
    "VerySlow" = @(
        "./tests/integration/...",
        "./tests/performance/...",
        "./tests/e2e/...",
        "./tests/disaster-recovery/...",
        "./tests/compliance/...",
        "./tests/chaos/...",
        "./tests/security/...",
        "./test/integration/..."
    )
}

# Track results
$results = @{}
$totalStart = Get-Date

# Function to run tests for a package
function Run-PackageTests {
    param(
        [string]$Package,
        [string]$Category,
        [string]$CategoryTimeout
    )
    
    Write-Host "`nTesting $Package ($Category)..." -ForegroundColor Green
    
    $pkgArgs = $testArgs + @("-timeout=$CategoryTimeout", $Package)
    
    $start = Get-Date
    $output = & go @pkgArgs 2>&1
    $exitCode = $LASTEXITCODE
    $duration = (Get-Date) - $start
    
    $status = if ($exitCode -eq 0) { "PASS" } else { "FAIL" }
    $color = if ($exitCode -eq 0) { "Green" } else { "Red" }
    
    Write-Host "  $status in $($duration.TotalSeconds.ToString('F2'))s" -ForegroundColor $color
    
    if ($exitCode -ne 0 -and $Verbose) {
        Write-Host $output -ForegroundColor Red
    }
    
    return @{
        Package = $Package
        Category = $Category
        Status = $status
        Duration = $duration
        ExitCode = $exitCode
    }
}

# Run tests by category with appropriate timeouts
Write-Host "`n=== Running Quick Tests (timeout: 1m) ===" -ForegroundColor Cyan
foreach ($pkg in $testCategories["Quick"]) {
    $result = Run-PackageTests -Package $pkg -Category "Quick" -CategoryTimeout "1m"
    $results[$pkg] = $result
}

Write-Host "`n=== Running Medium Tests (timeout: 2m) ===" -ForegroundColor Cyan
foreach ($pkg in $testCategories["Medium"]) {
    $result = Run-PackageTests -Package $pkg -Category "Medium" -CategoryTimeout "2m"
    $results[$pkg] = $result
}

if (-not $SkipSlow) {
    Write-Host "`n=== Running Slow Tests (timeout: 5m) ===" -ForegroundColor Cyan
    foreach ($pkg in $testCategories["Slow"]) {
        $result = Run-PackageTests -Package $pkg -Category "Slow" -CategoryTimeout "5m"
        $results[$pkg] = $result
    }
    
    if (-not $Short) {
        Write-Host "`n=== Running Very Slow Tests (timeout: 10m) ===" -ForegroundColor Cyan
        foreach ($pkg in $testCategories["VerySlow"]) {
            $result = Run-PackageTests -Package $pkg -Category "VerySlow" -CategoryTimeout "10m"
            $results[$pkg] = $result
        }
    }
}

# Summary
$totalDuration = (Get-Date) - $totalStart
Write-Host "`n=== Test Summary ===" -ForegroundColor Cyan
Write-Host "Total Duration: $($totalDuration.TotalSeconds.ToString('F2'))s"

$passed = $results.Values | Where-Object { $_.Status -eq "PASS" } | Measure-Object
$failed = $results.Values | Where-Object { $_.Status -eq "FAIL" } | Measure-Object

Write-Host "Passed: $($passed.Count)" -ForegroundColor Green
Write-Host "Failed: $($failed.Count)" -ForegroundColor $(if ($failed.Count -gt 0) { "Red" } else { "Green" })

if ($failed.Count -gt 0) {
    Write-Host "`nFailed Packages:" -ForegroundColor Red
    $results.Values | Where-Object { $_.Status -eq "FAIL" } | ForEach-Object {
        Write-Host "  - $($_.Package) ($($_.Category))" -ForegroundColor Red
    }
    exit 1
}

Write-Host "`nAll tests passed!" -ForegroundColor Green
exit 0