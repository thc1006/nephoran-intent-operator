# Optimized Test Runner for Windows PowerShell
# Purpose: Efficiently run Go tests with proper resource management

param(
    [string]$TestMode = "quick",  # quick, full, race, benchmark
    [int]$Parallel = 4,
    [int]$Timeout = 10,
    [switch]$Coverage,
    [switch]$Verbose,
    [switch]$FailFast
)

$ErrorActionPreference = "Stop"
$startTime = Get-Date

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Optimized Test Runner" -ForegroundColor Cyan
Write-Host "Mode: $TestMode | Parallel: $Parallel | Timeout: ${Timeout}m" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Set environment variables for optimal performance
$env:GOMAXPROCS = $Parallel
$env:GOGC = 100
$env:GOMEMLIMIT = "2GiB"

# Clean up any existing test cache if needed
if ($TestMode -eq "full") {
    Write-Host "Cleaning test cache..." -ForegroundColor Yellow
    go clean -testcache
}

# Build test command based on mode
$testCmd = "go test"
$testArgs = @()

switch ($TestMode) {
    "quick" {
        Write-Host "Running quick validation tests..." -ForegroundColor Green
        $testArgs += "-short"
        $testArgs += "-timeout=${Timeout}m"
    }
    
    "full" {
        Write-Host "Running full test suite..." -ForegroundColor Green
        $testArgs += "-timeout=${Timeout}m"
        $testArgs += "-count=1"  # Disable test caching
    }
    
    "race" {
        Write-Host "Running race detection tests..." -ForegroundColor Green
        $testArgs += "-race"
        $testArgs += "-timeout=${Timeout}m"
        $Parallel = [Math]::Max(1, $Parallel / 2)  # Reduce parallelism for race tests
    }
    
    "benchmark" {
        Write-Host "Running benchmarks..." -ForegroundColor Green
        $testArgs += "-bench=."
        $testArgs += "-benchmem"
        $testArgs += "-timeout=${Timeout}m"
    }
    
    default {
        Write-Host "Unknown test mode: $TestMode" -ForegroundColor Red
        exit 1
    }
}

# Add common flags
$testArgs += "-parallel=$Parallel"

if ($Coverage) {
    $testArgs += "-coverprofile=coverage.out"
    $testArgs += "-covermode=atomic"
}

if ($Verbose) {
    $testArgs += "-v"
}

if ($FailFast) {
    $testArgs += "-failfast"
}

$testArgs += "./..."

# Create results directory
$resultsDir = "test-results"
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir | Out-Null
}

$timestamp = Get-Date -Format "yyyy-MM-dd_HH-mm-ss"
$logFile = Join-Path $resultsDir "test_${TestMode}_${timestamp}.log"

Write-Host "Executing: $testCmd $($testArgs -join ' ')" -ForegroundColor Cyan
Write-Host "Log file: $logFile" -ForegroundColor Gray

# Run tests with output capture
$testProcess = Start-Process -FilePath "go" -ArgumentList (@("test") + $testArgs) -NoNewWindow -PassThru -RedirectStandardOutput $logFile -RedirectStandardError "$logFile.err"

# Monitor test execution
$elapsed = 0
while (-not $testProcess.HasExited) {
    Start-Sleep -Seconds 5
    $elapsed += 5
    
    # Show progress
    Write-Host "." -NoNewline -ForegroundColor Gray
    
    # Check for timeout
    if ($elapsed -gt ($Timeout * 60)) {
        Write-Host ""
        Write-Host "Test execution exceeded timeout, terminating..." -ForegroundColor Red
        $testProcess.Kill()
        break
    }
}

Write-Host ""

# Check results
$exitCode = $testProcess.ExitCode
$duration = (Get-Date) - $startTime

if ($exitCode -eq 0) {
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "Tests PASSED" -ForegroundColor Green
    Write-Host "Duration: $($duration.ToString('mm\:ss'))" -ForegroundColor Green
    
    if ($Coverage) {
        Write-Host "Generating coverage report..." -ForegroundColor Cyan
        go tool cover -html=coverage.out -o coverage.html
        Write-Host "Coverage report: coverage.html" -ForegroundColor Cyan
    }
} else {
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "Tests FAILED (exit code: $exitCode)" -ForegroundColor Red
    Write-Host "Duration: $($duration.ToString('mm\:ss'))" -ForegroundColor Red
    Write-Host "Check log file: $logFile" -ForegroundColor Yellow
    
    # Show last few lines of error
    if (Test-Path "$logFile.err") {
        Write-Host "Last error lines:" -ForegroundColor Yellow
        Get-Content "$logFile.err" -Tail 10 | ForEach-Object { Write-Host $_ -ForegroundColor Red }
    }
}

# Parse and display test summary
Write-Host ""
Write-Host "Test Summary:" -ForegroundColor Cyan
$logContent = Get-Content $logFile -Tail 50
$passCount = ($logContent | Select-String "PASS" | Measure-Object).Count
$failCount = ($logContent | Select-String "FAIL" | Measure-Object).Count
$skipCount = ($logContent | Select-String "SKIP" | Measure-Object).Count

Write-Host "  Passed: $passCount" -ForegroundColor Green
Write-Host "  Failed: $failCount" -ForegroundColor $(if ($failCount -gt 0) { "Red" } else { "Gray" })
Write-Host "  Skipped: $skipCount" -ForegroundColor Gray

# Performance metrics
if ($Verbose) {
    Write-Host ""
    Write-Host "Performance Metrics:" -ForegroundColor Cyan
    Write-Host "  GOMAXPROCS: $env:GOMAXPROCS"
    Write-Host "  GOGC: $env:GOGC"
    Write-Host "  GOMEMLIMIT: $env:GOMEMLIMIT"
    
    # Try to extract timing information
    $timingInfo = $logContent | Select-String "\d+\.\d+s" | Select-Object -Last 5
    if ($timingInfo) {
        Write-Host "  Recent test timings:" -ForegroundColor Gray
        $timingInfo | ForEach-Object { Write-Host "    $_" -ForegroundColor Gray }
    }
}

Write-Host "========================================" -ForegroundColor Cyan

exit $exitCode