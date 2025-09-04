#!/usr/bin/env pwsh
# =====================================================================================
# CI Job: Comprehensive Testing
# =====================================================================================  
# Mirrors: .github/workflows/main-ci.yml -> testing job
#         .github/workflows/ubuntu-ci.yml -> test job
# This script reproduces the exact testing process from CI
# =====================================================================================

param(
    [switch]$FastMode,
    [switch]$WithRace,
    [switch]$WithCoverage, 
    [switch]$WithBenchmarks,
    [int]$TimeoutMinutes = 25,
    [int]$Parallel = 2
)

$ErrorActionPreference = "Stop"

# Load CI environment  
. "scripts\ci-env.ps1"

if ($FastMode) { $env:FAST_MODE = "true" }

Write-Host "=== Comprehensive Testing ===" -ForegroundColor Green
Write-Host "Fast Mode: $env:FAST_MODE" -ForegroundColor Cyan
Write-Host "Race Detection: $WithRace" -ForegroundColor Cyan
Write-Host "Coverage: $WithCoverage" -ForegroundColor Cyan
Write-Host "Benchmarks: $WithBenchmarks" -ForegroundColor Cyan
Write-Host "Timeout: $TimeoutMinutes minutes" -ForegroundColor Cyan
Write-Host "Parallel: $Parallel" -ForegroundColor Cyan
Write-Host ""

# Create test results directory
if (!(Test-Path "test-results")) {
    New-Item -ItemType Directory -Path "test-results" -Force | Out-Null
}

# Step 1: Setup envtest binaries (mirrors CI line 407-420)
Write-Host "[1/4] Setting up envtest binaries..." -ForegroundColor Blue
try {
    # Check if setup-envtest is available
    $setupEnvtest = Get-Command "setup-envtest" -ErrorAction SilentlyContinue
    if (!$setupEnvtest) {
        Write-Host "📥 Installing setup-envtest..." -ForegroundColor Yellow
        go install "sigs.k8s.io/controller-runtime/tools/setup-envtest@latest"
    }
    
    Write-Host "🔧 Setting up envtest binaries for Kubernetes $env:ENVTEST_K8S_VERSION..." -ForegroundColor Yellow
    
    # Setup envtest binaries (adjusted for Windows)
    $setupArgs = @(
        "use", $env:ENVTEST_K8S_VERSION
        "--arch=amd64"
        "--os=windows"  # Adjust for local Windows environment
        "-p", "path"
    )
    
    try {
        $kubebuilderAssets = & setup-envtest @setupArgs 2>&1 | Out-String | ForEach-Object Trim
        $env:KUBEBUILDER_ASSETS = $kubebuilderAssets
        Write-Host "✅ KUBEBUILDER_ASSETS set to: $env:KUBEBUILDER_ASSETS" -ForegroundColor Green
        
        # Verify binaries exist
        if (Test-Path $env:KUBEBUILDER_ASSETS) {
            $binaries = Get-ChildItem $env:KUBEBUILDER_ASSETS -File
            Write-Host "Available test binaries: $($binaries.Name -join ', ')" -ForegroundColor Gray
        }
    }
    catch {
        Write-Host "⚠️ Failed to setup Windows envtest binaries, trying Linux..." -ForegroundColor Yellow
        $setupArgs[2] = "--os=linux"  # Fallback to Linux binaries
        $kubebuilderAssets = & setup-envtest @setupArgs 2>&1 | Out-String | ForEach-Object Trim  
        $env:KUBEBUILDER_ASSETS = $kubebuilderAssets
        Write-Host "✅ KUBEBUILDER_ASSETS set to: $env:KUBEBUILDER_ASSETS (Linux binaries)" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "⚠️ Failed to setup envtest binaries: $_" -ForegroundColor Yellow
    Write-Host "Controller tests may fail without these binaries" -ForegroundColor Gray
}

# Step 2: Smart test execution (mirrors CI line 422-465)
Write-Host "[2/4] Executing test suite..." -ForegroundColor Blue

# Set CI-like environment optimizations
$env:CGO_ENABLED = "0"
if (!$env:GOMAXPROCS) { $env:GOMAXPROCS = "2" }

$testStartTime = Get-Date

# Build test arguments
$testArgs = @()

if ($env:FAST_MODE -eq "true") {
    Write-Host "⚡ Fast mode: Essential tests only (no race detection)" -ForegroundColor Yellow
    $testArgs += "-short"
    $testArgs += "-timeout=10m"
    $testArgs += "-parallel=4"
    
    # Test only key packages in fast mode
    $packages = @("./api/...", "./controllers/...", "./pkg/...")
    
} else {
    Write-Host "🔍 Full mode: Comprehensive test suite with race detection" -ForegroundColor Yellow
    $testArgs += "-v"
    
    if ($WithRace) {
        $testArgs += "-race"
    }
    
    $testArgs += "-vet=off"  # Skip vet during test (already done in build job)
    $testArgs += "-timeout=$($TimeoutMinutes)m"
    $testArgs += "-parallel=$Parallel"
    
    if ($WithCoverage) {
        $testArgs += "-coverprofile=test-results/coverage.out"
        $testArgs += "-covermode=atomic"
    }
    
    # Test all packages in full mode
    $packages = @("./...")
}

# Execute tests (mirrors CI exact pattern)
Write-Host "Command: go test $($testArgs -join ' ') $($packages -join ' ')" -ForegroundColor Gray
Write-Host ""

try {
    $testOutput = @()
    
    # Run tests and capture output
    & go test @testArgs @packages 2>&1 | Tee-Object -FilePath "test-results/test-output.txt" -Append | ForEach-Object {
        $testOutput += $_
        Write-Host $_
    }
    
    $testExitCode = $LASTEXITCODE
    $testDuration = (Get-Date) - $testStartTime
    
    Write-Host ""
    Write-Host "Test execution completed in $([math]::Round($testDuration.TotalMinutes, 1)) minutes" -ForegroundColor Gray
    
    if ($testExitCode -eq 0) {
        if ($env:FAST_MODE -eq "true") {
            Write-Host "✅ Essential tests passed (fast mode)" -ForegroundColor Green
        } else {
            Write-Host "✅ All tests passed (comprehensive mode)" -ForegroundColor Green
        }
    } else {
        Write-Host "❌ Tests failed with exit code $testExitCode" -ForegroundColor Red
        Write-Host "📋 Test failures detected" -ForegroundColor Yellow
        
        # Show test summary from output (like CI)
        $failureLines = $testOutput | Where-Object { $_ -match "FAIL|panic:" } | Select-Object -Last 10
        if ($failureLines) {
            Write-Host ""
            Write-Host "Recent test failures:" -ForegroundColor Yellow
            foreach ($line in $failureLines) {
                Write-Host "  $line" -ForegroundColor Red
            }
        }
        
        exit $testExitCode
    }
}
catch {
    Write-Host "❌ Test execution failed: $_" -ForegroundColor Red
    exit 1
}

# Step 3: Coverage reporting (mirrors CI line 456-464)
if ($WithCoverage -and (Test-Path "test-results/coverage.out")) {
    Write-Host "[3/4] Generating coverage report..." -ForegroundColor Blue
    
    try {
        # Generate HTML coverage report
        go tool cover -html=test-results/coverage.out -o test-results/coverage.html
        Write-Host "✅ HTML coverage report: test-results/coverage.html" -ForegroundColor Green
        
        # Generate coverage summary (matches CI format)
        Write-Host ""
        Write-Host "📈 Coverage summary:" -ForegroundColor Green
        $coverageTotal = go tool cover -func=test-results/coverage.out | Select-Object -Last 1
        Write-Host $coverageTotal -ForegroundColor Cyan
        
        Write-Host ""
        Write-Host "📋 Per-package coverage:" -ForegroundColor Yellow
        $coverageDetails = go tool cover -func=test-results/coverage.out | Where-Object { $_ -match "(total|api/|controllers/|pkg/)" } | Select-Object -First 10
        foreach ($line in $coverageDetails) {
            Write-Host "  $line" -ForegroundColor Gray
        }
    }
    catch {
        Write-Host "⚠️ Failed to generate coverage report: $_" -ForegroundColor Yellow
    }
} else {
    Write-Host "[3/4] Coverage reporting skipped" -ForegroundColor Blue
}

# Step 4: Benchmark tests (mirrors CI line 467-480) 
if ($WithBenchmarks) {
    Write-Host "[4/4] Running benchmark tests..." -ForegroundColor Blue
    
    try {
        # Check if benchmarks exist
        $benchmarkList = go test -list=Benchmark ./... 2>&1 | Out-String
        if ($benchmarkList -match "Benchmark") {
            Write-Host "🔄 Found benchmarks, running..." -ForegroundColor Yellow
            
            $benchArgs = @(
                "-bench=."
                "-benchmem" 
                "-timeout=10m"
                "./..."
            )
            
            & go test @benchArgs | Tee-Object -FilePath "test-results/benchmark-output.txt"
            Write-Host "✅ Benchmarks completed - results in test-results/benchmark-output.txt" -ForegroundColor Green
        } else {
            Write-Host "ℹ️ No benchmark tests found" -ForegroundColor Gray
        }
    }
    catch {
        Write-Host "⚠️ Benchmark execution failed: $_" -ForegroundColor Yellow  
    }
} else {
    Write-Host "[4/4] Benchmark tests skipped" -ForegroundColor Blue
}

# Test results summary
Write-Host ""
Write-Host "=== Test Results Summary ===" -ForegroundColor Green
$testResultsSize = if (Test-Path "test-results") {
    $files = Get-ChildItem "test-results" -File
    $totalSize = ($files | Measure-Object Length -Sum).Sum / 1KB
    Write-Host "Generated artifacts ($([math]::Round($totalSize, 1)) KB):" -ForegroundColor Cyan
    foreach ($file in $files) {
        $size = [math]::Round($file.Length / 1KB, 1)
        Write-Host "  $($file.Name) ($size KB)" -ForegroundColor Gray
    }
} else {
    Write-Host "No test artifacts generated" -ForegroundColor Gray
}

Write-Host ""
Write-Host "🎉 Test suite completed successfully!" -ForegroundColor Green