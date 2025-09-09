# Cross-Platform Race Detection Test Script
# Handles CGO setup for both Windows and Linux/macOS

param(
    [int]$TimeoutMinutes = 15,
    [switch]$Verbose,
    [switch]$SkipBuildCheck
)

Write-Host "[RACE DETECTION] Test Suite" -ForegroundColor Green
Write-Host "================================" -ForegroundColor Green
Write-Host ""

# Platform Detection
$IsWindows = $PSVersionTable.PSVersion.Major -ge 5 -and $PSVersionTable.Platform -ne 'Unix'
$IsLinux = $PSVersionTable.Platform -eq 'Unix' -and (Test-Path "/etc/os-release")
$IsMacOS = $PSVersionTable.Platform -eq 'Unix' -and (Test-Path "/System/Library")

Write-Host "[SYSTEM INFO]" -ForegroundColor Cyan
Write-Host "  Platform: $(if ($IsWindows) { 'Windows' } elseif ($IsLinux) { 'Linux' } elseif ($IsMacOS) { 'macOS' } else { 'Unknown' })" -ForegroundColor Gray
Write-Host "  PowerShell Version: $($PSVersionTable.PSVersion)" -ForegroundColor Gray
Write-Host "  Go Version: $(go version)" -ForegroundColor Gray
Write-Host ""

# Function to detect C compiler
function Find-CCompiler {
    $compilers = @()
    
    if ($IsWindows) {
        # Windows compiler detection
        $windowsCompilers = @(
            @{ Name = "TDM-GCC"; Cmd = "gcc"; TestArgs = "--version" },
            @{ Name = "MinGW"; Cmd = "mingw32-gcc"; TestArgs = "--version" },
            @{ Name = "MSYS2"; Cmd = "gcc"; TestArgs = "--version"; Path = "C:\msys64\mingw64\bin" },
            @{ Name = "Clang"; Cmd = "clang"; TestArgs = "--version" },
            @{ Name = "MSVC"; Cmd = "cl"; TestArgs = "" }
        )
        
        foreach ($compiler in $windowsCompilers) {
            try {
                if ($compiler.Path -and (Test-Path $compiler.Path)) {
                    $env:PATH = "$($compiler.Path);$env:PATH"
                }
                
                $result = & $compiler.Cmd $compiler.TestArgs 2>&1
                if ($LASTEXITCODE -eq 0) {
                    $compilers += $compiler
                    Write-Host "  [FOUND] $($compiler.Name) ($($compiler.Cmd))" -ForegroundColor Green
                }
            }
            catch {
                # Compiler not found
            }
        }
    }
    else {
        # Unix-like compiler detection
        $unixCompilers = @(
            @{ Name = "GCC"; Cmd = "gcc"; TestArgs = "--version" },
            @{ Name = "Clang"; Cmd = "clang"; TestArgs = "--version" }
        )
        
        foreach ($compiler in $unixCompilers) {
            try {
                $result = & $compiler.Cmd $compiler.TestArgs 2>&1
                if ($LASTEXITCODE -eq 0) {
                    $compilers += $compiler
                    Write-Host "  [FOUND] $($compiler.Name) ($($compiler.Cmd))" -ForegroundColor Green
                }
            }
            catch {
                # Compiler not found
            }
        }
    }
    
    return $compilers
}

# Check for C compiler
Write-Host "[COMPILER CHECK] Checking for C Compiler..." -ForegroundColor Cyan
$compilers = Find-CCompiler

if ($compilers.Count -eq 0) {
    Write-Host ""
    Write-Host "[WARNING] No C compiler found!" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Race detection requires CGO, which needs a C compiler." -ForegroundColor Yellow
    Write-Host ""
    
    if ($IsWindows) {
        Write-Host "[INSTALL] To enable race detection on Windows, install one of:" -ForegroundColor Cyan
        Write-Host "  1. TDM-GCC (recommended):" -ForegroundColor Gray
        Write-Host "     https://jmeubank.github.io/tdm-gcc/" -ForegroundColor Blue
        Write-Host ""
        Write-Host "  2. MinGW-w64:" -ForegroundColor Gray
        Write-Host "     https://www.mingw-w64.org/downloads/" -ForegroundColor Blue
        Write-Host ""
        Write-Host "  3. MSYS2:" -ForegroundColor Gray
        Write-Host "     https://www.msys2.org/" -ForegroundColor Blue
        Write-Host "     After install, run: pacman -S mingw-w64-x86_64-gcc" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  4. Install via Chocolatey:" -ForegroundColor Gray
        Write-Host "     choco install mingw" -ForegroundColor Blue
        Write-Host ""
        Write-Host "  5. Install via Scoop:" -ForegroundColor Gray
        Write-Host "     scoop install gcc" -ForegroundColor Blue
    }
    elseif ($IsLinux) {
        Write-Host "[INSTALL] To enable race detection on Linux, install GCC:" -ForegroundColor Cyan
        Write-Host "  Ubuntu/Debian: sudo apt-get install build-essential" -ForegroundColor Blue
        Write-Host "  RHEL/CentOS: sudo yum groupinstall 'Development Tools'" -ForegroundColor Blue
        Write-Host "  Fedora: sudo dnf groupinstall 'Development Tools'" -ForegroundColor Blue
        Write-Host "  Arch: sudo pacman -S base-devel" -ForegroundColor Blue
    }
    elseif ($IsMacOS) {
        Write-Host "[INSTALL] To enable race detection on macOS:" -ForegroundColor Cyan
        Write-Host "  Install Xcode Command Line Tools:" -ForegroundColor Blue
        Write-Host "  xcode-select --install" -ForegroundColor Blue
    }
    
    Write-Host ""
    Write-Host "[CONTINUE] Continuing with standard tests (no race detection)..." -ForegroundColor Yellow
    $CGOEnabled = $false
}
else {
    Write-Host "[OK] C compiler(s) available for CGO" -ForegroundColor Green
    $CGOEnabled = $true
}

Write-Host ""

# Function to run tests with proper CGO setup
function Invoke-GoTestWithCGO {
    param(
        [string]$TestPattern = "./...",
        [string]$TestName = "All Tests",
        [switch]$RaceDetection,
        [int]$Timeout = 10
    )
    
    Write-Host "[TEST] Running: $TestName" -ForegroundColor Cyan
    
    # Set up environment based on platform and CGO availability
    if ($RaceDetection -and $CGOEnabled) {
        if ($IsWindows) {
            # Windows: Set environment variable in current session
            $env:CGO_ENABLED = "1"
            Write-Host "  CGO_ENABLED=1 (Windows environment)" -ForegroundColor Gray
            
            # Build command with race flag
            $testCmd = "go test $TestPattern -race -timeout=${Timeout}m"
        }
        else {
            # Linux/macOS: Use inline environment variable
            $testCmd = "CGO_ENABLED=1 go test $TestPattern -race -timeout=${Timeout}m"
            Write-Host "  CGO_ENABLED=1 (Unix environment)" -ForegroundColor Gray
        }
    }
    else {
        if ($IsWindows) {
            $env:CGO_ENABLED = "0"
        }
        $testCmd = "go test $TestPattern -timeout=${Timeout}m"
        Write-Host "  CGO_ENABLED=0 (Standard tests)" -ForegroundColor Gray
    }
    
    if ($Verbose) {
        $testCmd += " -v"
    }
    
    Write-Host "  Command: $testCmd" -ForegroundColor Gray
    Write-Host ""
    
    # Execute the test command
    $startTime = Get-Date
    
    try {
        if ($IsWindows) {
            # Windows: Use Invoke-Expression
            $output = Invoke-Expression $testCmd 2>&1
            $exitCode = $LASTEXITCODE
        }
        else {
            # Unix: Use bash -c
            $output = bash -c "$testCmd" 2>&1
            $exitCode = $LASTEXITCODE
        }
        
        $duration = (Get-Date) - $startTime
        
        if ($exitCode -eq 0) {
            Write-Host "[PASS] $TestName completed in $($duration.TotalSeconds.ToString('F2'))s" -ForegroundColor Green
            return @{ Success = $true; Duration = $duration; Output = $output }
        }
        else {
            Write-Host "[FAIL] $TestName failed after $($duration.TotalSeconds.ToString('F2'))s" -ForegroundColor Red
            
            # Check for specific CGO/race errors
            $outputStr = $output -join "`n"
            if ($outputStr -match "requires cgo|CGO_ENABLED") {
                Write-Host "  Error: CGO not properly configured" -ForegroundColor Red
                Write-Host "  Please install a C compiler (see instructions above)" -ForegroundColor Yellow
            }
            
            if ($Verbose) {
                Write-Host "Output:" -ForegroundColor Gray
                $output | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            }
            
            return @{ Success = $false; Duration = $duration; Output = $output }
        }
    }
    catch {
        $duration = (Get-Date) - $startTime
        Write-Host "[ERROR] $TestName error: $($_.Exception.Message)" -ForegroundColor Red
        return @{ Success = $false; Duration = $duration; Output = $_.Exception.Message }
    }
}

# Run test suites
$testResults = @()

# Test 1: Build verification (without race detection)
if (-not $SkipBuildCheck) {
    Write-Host "[TEST 1] Build Verification" -ForegroundColor Magenta
    Write-Host "-------------------------------" -ForegroundColor Gray
    
    $buildResult = Invoke-GoTestWithCGO -TestPattern "./..." -TestName "Build Test" -RaceDetection:$false -Timeout 5
    $testResults += @{ Name = "Build Verification"; Result = $buildResult }
    
    Write-Host ""
}

# Test 2: Standard tests (without race detection)
Write-Host "[TEST 2] Standard Unit Tests" -ForegroundColor Magenta
Write-Host "-------------------------------" -ForegroundColor Gray

$standardResult = Invoke-GoTestWithCGO -TestPattern "./..." -TestName "Standard Tests" -RaceDetection:$false -Timeout 10
$testResults += @{ Name = "Standard Tests"; Result = $standardResult }

Write-Host ""

# Test 3: Race detection tests (if CGO available)
if ($CGOEnabled) {
    Write-Host "[TEST 3] Race Detection Tests" -ForegroundColor Magenta
    Write-Host "-------------------------------" -ForegroundColor Gray
    
    # Test specific packages known to have race conditions
    $racePackages = @(
        @{ Path = "./pkg/security/..."; Name = "Security Package" },
        @{ Path = "./pkg/controllers/..."; Name = "Controllers" },
        @{ Path = "./pkg/llm/..."; Name = "LLM Package" },
        @{ Path = "./internal/loop/..."; Name = "Loop Package" }
    )
    
    foreach ($pkg in $racePackages) {
        if (Test-Path ($pkg.Path -replace '\.\.\.$', '')) {
            $raceResult = Invoke-GoTestWithCGO -TestPattern $pkg.Path -TestName "$($pkg.Name) Race Tests" -RaceDetection -Timeout $TimeoutMinutes
            $testResults += @{ Name = "$($pkg.Name) Race"; Result = $raceResult }
            Write-Host ""
        }
    }
}
else {
    Write-Host "[TEST 3] Race Detection Tests - SKIPPED (No C compiler)" -ForegroundColor Yellow
    Write-Host ""
}

# Test 4: Specific race condition patterns
if ($CGOEnabled) {
    Write-Host "[TEST 4] Known Race Patterns" -ForegroundColor Magenta
    Write-Host "-------------------------------" -ForegroundColor Gray
    
    # Create a test file to verify race detection works
    $testFile = @'
package main

import (
    "testing"
    "sync"
)

func TestRaceDetection(t *testing.T) {
    var counter int
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter++ // Intentional race condition
        }()
    }
    
    wg.Wait()
    t.Logf("Counter value: %d", counter)
}
'@
    
    $tempTestPath = Join-Path $env:TEMP "race_test.go"
    Set-Content -Path $tempTestPath -Value $testFile
    
    Write-Host "  Testing race detector functionality..." -ForegroundColor Gray
    
    if ($IsWindows) {
        $env:CGO_ENABLED = "1"
        $testOutput = go test -race $tempTestPath 2>&1
    }
    else {
        $testOutput = bash -c "CGO_ENABLED=1 go test -race $tempTestPath" 2>&1
    }
    
    if ($testOutput -match "WARNING: DATA RACE|race detected") {
        Write-Host "[OK] Race detector is working correctly" -ForegroundColor Green
    }
    else {
        Write-Host "[WARNING] Race detector may not be functioning properly" -ForegroundColor Yellow
    }
    
    Remove-Item $tempTestPath -ErrorAction SilentlyContinue
    Write-Host ""
}

# Summary
Write-Host "[SUMMARY] TEST RESULTS" -ForegroundColor Green
Write-Host "=======================" -ForegroundColor Green
Write-Host ""

$totalTests = $testResults.Count
$passedTests = ($testResults | Where-Object { $_.Result.Success }).Count
$failedTests = $totalTests - $passedTests

Write-Host "Results:" -ForegroundColor Cyan
Write-Host "  Total Tests: $totalTests" -ForegroundColor Gray
Write-Host "  Passed: $passedTests" -ForegroundColor Green
Write-Host "  Failed: $failedTests" -ForegroundColor $(if ($failedTests -gt 0) { "Red" } else { "Gray" })
Write-Host "  Success Rate: $([math]::Round(($passedTests/$totalTests)*100, 1))%" -ForegroundColor $(if ($failedTests -eq 0) { "Green" } else { "Yellow" })

Write-Host ""
Write-Host "⏱️  Test Durations:" -ForegroundColor Cyan
foreach ($test in $testResults) {
    $status = if ($test.Result.Success) { "PASS" } else { "FAIL" }
    $duration = $test.Result.Duration.TotalSeconds.ToString('F2')
    $statusColor = if ($test.Result.Success) { "Green" } else { "Red" }
    Write-Host "  [$status] $($test.Name): ${duration}s" -ForegroundColor $statusColor
}

Write-Host ""

if (-not $CGOEnabled) {
    Write-Host "[TIP] Install a C compiler to enable race detection tests" -ForegroundColor Yellow
    Write-Host "        This helps catch concurrency bugs before production!" -ForegroundColor Yellow
    Write-Host ""
}

if ($failedTests -eq 0) {
    Write-Host "[SUCCESS] All tests passed successfully!" -ForegroundColor Green
    exit 0
}
else {
    Write-Host "[WARNING] Some tests failed. Review the output above for details." -ForegroundColor Yellow
    exit 1
}