#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Runs comprehensive Windows-specific test coverage for nephoran-intent-operator

.DESCRIPTION
    This script runs all Windows-specific tests that validate the fixes implemented
    to resolve CI failures. It includes tests for:
    - PowerShell command separation and execution
    - Parent directory creation in status file operations
    - Windows path limits and validation
    - Concurrency stress tests for Windows filesystem behavior
    - Integration tests for the complete Windows CI pipeline

.PARAMETER TestType
    Type of tests to run: all, unit, integration, stress, or specific test names

.PARAMETER Verbose
    Enable verbose output for debugging

.PARAMETER Short
    Run only short tests (skip stress and long-running tests)

.PARAMETER Coverage
    Generate test coverage report

.EXAMPLE
    .\run-windows-comprehensive-tests.ps1
    Runs all Windows-specific tests

.EXAMPLE
    .\run-windows-comprehensive-tests.ps1 -TestType stress -Verbose
    Runs only stress tests with verbose output

.EXAMPLE
    .\run-windows-comprehensive-tests.ps1 -Short -Coverage
    Runs short tests with coverage report
#>

param(
    [Parameter()]
    [ValidateSet("all", "unit", "integration", "stress", "powershell", "pathutil", "concurrency")]
    [string]$TestType = "all",
    
    [Parameter()]
    [switch]$Verbose,
    
    [Parameter()]
    [switch]$Short,
    
    [Parameter()]
    [switch]$Coverage,
    
    [Parameter()]
    [switch]$FailFast,
    
    [Parameter()]
    [int]$Timeout = 600 # 10 minutes default timeout
)

# Ensure we're running on Windows
if ($PSVersionTable.PSEdition -eq "Core" -and $IsWindows -eq $false) {
    Write-Error "This script is designed to run on Windows only"
    exit 1
}
elseif ($PSVersionTable.PSEdition -eq "Desktop" -and $env:OS -ne "Windows_NT") {
    Write-Error "This script is designed to run on Windows only"
    exit 1
}

# Function to write colored output
function Write-TestHeader($Message) {
    Write-Host "`n=== $Message ===" -ForegroundColor Cyan
}

function Write-TestSuccess($Message) {
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-TestError($Message) {
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-TestInfo($Message) {
    Write-Host "ℹ $Message" -ForegroundColor Yellow
}

# Test configuration
$TestResults = @{
    Passed = 0
    Failed = 0
    Skipped = 0
    Total = 0
    StartTime = Get-Date
    Errors = @()
}

# Test packages and their descriptions
$TestPackages = @{
    "powershell" = @{
        Package = "./internal/platform"
        Pattern = "*powershell*test.go"
        Description = "PowerShell command separation and execution tests"
        Tags = @("windows")
    }
    "pathutil" = @{
        Package = "./internal/pathutil"
        Pattern = "*windows*path*test.go"
        Description = "Windows path validation and limits tests"
        Tags = @("windows")
    }
    "parent-dir" = @{
        Package = "./internal/loop"
        Pattern = "*parent*dir*test.go"
        Description = "Parent directory creation tests"
        Tags = @("windows")
    }
    "concurrency" = @{
        Package = "./internal/loop"
        Pattern = "*concurrency*stress*test.go"
        Description = "Windows filesystem concurrency stress tests"
        Tags = @("windows")
        IsStress = $true
    }
    "integration" = @{
        Package = "./internal/integration"
        Pattern = "*windows*ci*integration*test.go"
        Description = "End-to-end Windows CI pipeline integration tests"
        Tags = @("windows", "integration")
        IsIntegration = $true
    }
    "compatibility" = @{
        Package = "./pkg/controllers"
        Pattern = "*windows*compatibility*test.go"
        Description = "Windows compatibility tests for controllers"
        Tags = @("windows")
    }
    "race-conditions" = @{
        Package = "./internal/loop"
        Pattern = "*windows*race*test.go"
        Description = "Windows race condition and retry logic tests"
        Tags = @("windows")
    }
}

function Run-TestPackage($PackageName, $PackageInfo) {
    Write-TestHeader "Running $($PackageInfo.Description)"
    
    # Build test command
    $testArgs = @("test")
    
    # Add package
    $testArgs += $PackageInfo.Package
    
    # Add build tags
    if ($PackageInfo.Tags) {
        $tagString = $PackageInfo.Tags -join ","
        $testArgs += "-tags", $tagString
    }
    
    # Add verbose flag if requested
    if ($Verbose) {
        $testArgs += "-v"
    }
    
    # Add short flag if requested
    if ($Short -and $PackageInfo.IsStress) {
        Write-TestInfo "Skipping stress tests in short mode: $PackageName"
        $TestResults.Skipped++
        return $true
    }
    
    if ($Short -and $PackageInfo.IsIntegration) {
        Write-TestInfo "Skipping integration tests in short mode: $PackageName"
        $TestResults.Skipped++
        return $true
    }
    
    if ($Short) {
        $testArgs += "-short"
    }
    
    # Add coverage if requested
    if ($Coverage) {
        $coverageFile = "coverage-$PackageName.out"
        $testArgs += "-coverprofile=$coverageFile", "-covermode=atomic"
    }
    
    # Add timeout
    $testArgs += "-timeout", "${Timeout}s"
    
    # Add fail fast if requested
    if ($FailFast) {
        $testArgs += "-failfast"
    }
    
    # Run the test
    Write-TestInfo "Executing: go $($testArgs -join ' ')"
    
    $process = Start-Process -FilePath "go" -ArgumentList $testArgs -NoNewWindow -PassThru -RedirectStandardOutput "test-output-$PackageName.log" -RedirectStandardError "test-error-$PackageName.log"
    
    # Wait for completion with timeout
    $completed = $process.WaitForExit($Timeout * 1000)
    
    if (-not $completed) {
        Write-TestError "Test timed out after $Timeout seconds: $PackageName"
        $process.Kill()
        $TestResults.Failed++
        $TestResults.Errors += "Timeout: $PackageName"
        return $false
    }
    
    $exitCode = $process.ExitCode
    
    # Read outputs
    $stdout = Get-Content "test-output-$PackageName.log" -ErrorAction SilentlyContinue
    $stderr = Get-Content "test-error-$PackageName.log" -ErrorAction SilentlyContinue
    
    if ($exitCode -eq 0) {
        Write-TestSuccess "$PackageName tests passed"
        $TestResults.Passed++
        
        # Show summary from output
        $summaryLine = $stdout | Where-Object { $_ -match "^PASS" -or $_ -match "ok\s+" } | Select-Object -Last 1
        if ($summaryLine) {
            Write-TestInfo $summaryLine
        }
        
        return $true
    } else {
        Write-TestError "$PackageName tests failed (exit code: $exitCode)"
        $TestResults.Failed++
        $TestResults.Errors += "$PackageName (exit code: $exitCode)"
        
        # Show error details
        if ($stderr) {
            Write-Host "Error output:" -ForegroundColor Red
            $stderr | ForEach-Object { Write-Host "  $_" -ForegroundColor DarkRed }
        }
        
        # Show test failures from stdout
        $failureLines = $stdout | Where-Object { $_ -match "FAIL:" -or $_ -match "--- FAIL:" }
        if ($failureLines) {
            Write-Host "Test failures:" -ForegroundColor Red
            $failureLines | ForEach-Object { Write-Host "  $_" -ForegroundColor DarkRed }
        }
        
        return $false
    }
}

function Run-AllTests() {
    Write-TestHeader "Windows Comprehensive Test Suite"
    Write-TestInfo "Test Type: $TestType"
    Write-TestInfo "Short Mode: $Short"
    Write-TestInfo "Coverage: $Coverage"
    Write-TestInfo "Verbose: $Verbose"
    Write-TestInfo "Timeout: $Timeout seconds"
    
    # Verify Go is available
    try {
        $goVersion = go version 2>$null
        Write-TestInfo "Go Version: $goVersion"
    }
    catch {
        Write-TestError "Go is not available. Please install Go and ensure it's in your PATH."
        exit 1
    }
    
    # Clean up old test outputs
    Get-ChildItem -Path "." -Filter "test-*.log" | Remove-Item -Force -ErrorAction SilentlyContinue
    Get-ChildItem -Path "." -Filter "coverage-*.out" | Remove-Item -Force -ErrorAction SilentlyContinue
    
    # Determine which tests to run
    $testsToRun = @()
    
    switch ($TestType) {
        "all" {
            $testsToRun = $TestPackages.Keys
        }
        "unit" {
            $testsToRun = @("powershell", "pathutil", "parent-dir", "compatibility")
        }
        "integration" {
            $testsToRun = @("integration")
        }
        "stress" {
            $testsToRun = @("concurrency", "race-conditions")
        }
        default {
            if ($TestPackages.ContainsKey($TestType)) {
                $testsToRun = @($TestType)
            } else {
                Write-TestError "Unknown test type: $TestType"
                Write-TestInfo "Available test types: $($TestPackages.Keys -join ', ')"
                exit 1
            }
        }
    }
    
    Write-TestInfo "Running test packages: $($testsToRun -join ', ')"
    
    # Run tests
    foreach ($testName in $testsToRun) {
        $TestResults.Total++
        $packageInfo = $TestPackages[$testName]
        
        try {
            Run-TestPackage $testName $packageInfo
        }
        catch {
            Write-TestError "Exception running $testName : $_"
            $TestResults.Failed++
            $TestResults.Errors += "$testName (exception: $_)"
        }
    }
}

function Show-TestSummary() {
    $duration = (Get-Date) - $TestResults.StartTime
    
    Write-TestHeader "Test Results Summary"
    Write-Host "Duration: $($duration.ToString('mm\:ss'))" -ForegroundColor White
    Write-Host "Total:    $($TestResults.Total)" -ForegroundColor White
    Write-Host "Passed:   $($TestResults.Passed)" -ForegroundColor Green
    Write-Host "Failed:   $($TestResults.Failed)" -ForegroundColor $(if ($TestResults.Failed -gt 0) { "Red" } else { "Green" })
    Write-Host "Skipped:  $($TestResults.Skipped)" -ForegroundColor Yellow
    
    if ($TestResults.Errors.Count -gt 0) {
        Write-Host "`nErrors:" -ForegroundColor Red
        $TestResults.Errors | ForEach-Object { Write-Host "  - $_" -ForegroundColor DarkRed }
    }
    
    # Generate coverage report if requested
    if ($Coverage) {
        Write-TestHeader "Generating Coverage Report"
        $coverageFiles = Get-ChildItem -Path "." -Filter "coverage-*.out"
        
        if ($coverageFiles.Count -gt 0) {
            # Merge coverage files
            Write-TestInfo "Merging coverage files..."
            $mergedCoverage = "coverage-merged.out"
            
            # Write the first line (mode) from first file
            $firstFile = Get-Content $coverageFiles[0].FullName
            $firstFile[0] | Out-File -FilePath $mergedCoverage -Encoding ASCII
            
            # Append all coverage lines (skip mode line) from all files
            foreach ($file in $coverageFiles) {
                $lines = Get-Content $file.FullName
                $lines[1..($lines.Length-1)] | Out-File -FilePath $mergedCoverage -Append -Encoding ASCII
            }
            
            # Generate HTML coverage report
            Write-TestInfo "Generating HTML coverage report..."
            go tool cover -html=$mergedCoverage -o coverage-report.html
            Write-TestSuccess "Coverage report generated: coverage-report.html"
            
            # Show coverage summary
            $coverageText = go tool cover -func=$mergedCoverage | Select-Object -Last 1
            Write-TestInfo "Coverage Summary: $coverageText"
        }
        else {
            Write-TestInfo "No coverage files found"
        }
    }
    
    # Show recommendation
    if ($TestResults.Failed -gt 0) {
        Write-Host "`nRecommendation: Review failed tests and fix issues before CI/CD pipeline execution." -ForegroundColor Red
        Write-Host "For detailed logs, check test-*.log files." -ForegroundColor Yellow
    }
    else {
        Write-Host "`nAll Windows-specific tests passed! The fixes are working correctly." -ForegroundColor Green
    }
}

function Show-TestDocumentation() {
    Write-TestHeader "Windows Test Coverage Documentation"
    
    Write-Host @"
This comprehensive test suite validates Windows-specific fixes for:

1. PowerShell Command Separation Issues
   - Tests fix for '50echo' concatenation that caused CI failures
   - Validates proper PowerShell -NoProfile -Command formatting
   - Ensures milliseconds parameter is correctly formatted

2. Parent Directory Creation
   - Tests automatic status directory creation
   - Validates deep nested directory handling
   - Ensures atomic file operations with parent directory creation

3. Windows Path Validation and Limits
   - Tests Windows MAX_PATH limits (248 chars without \\?\\ prefix)
   - Validates reserved filename handling (CON, PRN, etc.)
   - Tests invalid character detection and UNC path support

4. Concurrency Stress Testing
   - High-concurrency IsProcessed operations
   - Concurrent status file creation
   - File system race condition handling
   - Memory pressure testing

5. Integration Testing
   - End-to-end CI pipeline simulation
   - PowerShell script execution in CI context
   - Concurrent CI job simulation
   - Windows environment validation

Test Coverage Areas:
- PowerShell regression prevention
- Windows filesystem behavior
- Path handling edge cases
- Concurrency safety
- CI pipeline reliability

Usage:
  .\run-windows-comprehensive-tests.ps1                    # Run all tests
  .\run-windows-comprehensive-tests.ps1 -TestType unit     # Run unit tests only
  .\run-windows-comprehensive-tests.ps1 -Short             # Skip long-running tests
  .\run-windows-comprehensive-tests.ps1 -Coverage          # Generate coverage report
  .\run-windows-comprehensive-tests.ps1 -Verbose -FailFast # Detailed output, stop on first failure

"@ -ForegroundColor White
}

# Main execution
try {
    if ($args -contains "-help" -or $args -contains "--help" -or $args -contains "/?" -or $args -contains "-h") {
        Show-TestDocumentation
        exit 0
    }
    
    Write-Host "Windows Comprehensive Test Suite for nephoran-intent-operator" -ForegroundColor Magenta
    Write-Host "=" * 60 -ForegroundColor Magenta
    
    # Verify we're in the correct directory
    if (-not (Test-Path "go.mod")) {
        Write-TestError "go.mod not found. Please run this script from the repository root."
        exit 1
    }
    
    # Check if this is the correct repository
    $goModContent = Get-Content "go.mod" -Raw
    if ($goModContent -notmatch "nephoran-intent-operator") {
        Write-TestError "This doesn't appear to be the nephoran-intent-operator repository."
        exit 1
    }
    
    Run-AllTests
    Show-TestSummary
    
    # Exit with appropriate code
    if ($TestResults.Failed -gt 0) {
        exit 1
    }
    else {
        exit 0
    }
}
catch {
    Write-TestError "Unexpected error: $_"
    Write-TestError $_.ScriptStackTrace
    exit 1
}
finally {
    # Clean up temporary files
    Get-ChildItem -Path "." -Filter "test-*.log" | Remove-Item -Force -ErrorAction SilentlyContinue
}