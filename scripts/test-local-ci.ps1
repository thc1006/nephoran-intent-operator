#!/usr/bin/env pwsh

# =============================================================================
# Local CI Testing Script - Replicates CI Failures Locally  
# =============================================================================
# This script runs the exact same tests that are failing in your CI
# Based on your specific test failures, it focuses on:
# 1. File permission issues
# 2. Exit code validation 
# 3. Handoff directory validation
# 4. JSON input validation
# 5. Concurrent processing
# 6. TLS server logging
# 7. Provider configuration
# =============================================================================

param(
    [switch]$Fast,           # Run only failing tests
    [switch]$Verbose,        # Verbose output
    [switch]$SkipBuild,      # Skip build step
    [switch]$Coverage,       # Generate coverage report
    [string]$TestFilter = "",# Filter specific tests
    [switch]$Help            # Show help
)

if ($Help) {
    Write-Host @"
Local CI Testing Script - Fix Issues Before Push

USAGE:
  .\scripts\test-local-ci.ps1 [OPTIONS]

OPTIONS:
  -Fast         Run only the tests that are currently failing in CI
  -Verbose      Show detailed test output  
  -SkipBuild    Skip the build step (if binaries already exist)
  -Coverage     Generate coverage reports
  -TestFilter   Run specific test pattern (e.g., "TestFilePermission")
  -Help         Show this help message

EXAMPLES:
  # Quick run of failing tests only
  .\scripts\test-local-ci.ps1 -Fast

  # Run specific failing test with verbose output
  .\scripts\test-local-ci.ps1 -TestFilter "TestFilePermissionValidation" -Verbose

  # Full test run with coverage (like CI)
  .\scripts\test-local-ci.ps1 -Coverage

  # Debug concurrent processing issue
  .\scripts\test-local-ci.ps1 -TestFilter "TestConcurrentFileProcessing" -Verbose
"@
    exit 0
}

# Script configuration
$ErrorActionPreference = "Continue"
$OriginalLocation = Get-Location

# Test environment setup
$env:CGO_ENABLED = "1"
$env:GOPROXY = "https://proxy.golang.org,direct"
$env:GOMAXPROCS = "4"
$env:DEBUG = "true"
$env:LLM_ALLOWED_ORIGINS = "http://localhost:3000,http://localhost:8080"

# Colors for output
$Red = "`e[31m"
$Green = "`e[32m"  
$Yellow = "`e[33m"
$Blue = "`e[34m"
$Magenta = "`e[35m"
$Cyan = "`e[36m"
$White = "`e[37m"
$Reset = "`e[0m"

function Write-Header {
    param([string]$Title)
    Write-Host "${Cyan}=============================================================================${Reset}"
    Write-Host "${Cyan}$Title${Reset}"
    Write-Host "${Cyan}=============================================================================${Reset}"
}

function Write-Success {
    param([string]$Message)
    Write-Host "${Green}✅ $Message${Reset}"
}

function Write-Error {
    param([string]$Message)
    Write-Host "${Red}❌ $Message${Reset}"
}

function Write-Warning {
    param([string]$Message)
    Write-Host "${Yellow}⚠️  $Message${Reset}"
}

function Write-Info {
    param([string]$Message)
    Write-Host "${Blue}ℹ️  $Message${Reset}"
}

# Specific failing tests from your CI
$FailingTests = @(
    "TestFilePermissionValidation",
    "TestOnceMode_ExitCodes",
    "TestMain_ExitCodes", 
    "TestValidateHandoffDir_Integration",
    "TestConcurrentFileProcessing",
    "TestTLSServerStartup",
    "TestProviderFactoryConfiguration"
)

try {
    Write-Header "🧪 Local CI Testing - Fix Issues Before Push"
    
    # Check Go version
    Write-Info "Checking Go version..."
    $goVersion = go version
    Write-Host "Go version: $goVersion"
    
    if (-not $goVersion.Contains("go1.24")) {
        Write-Warning "Expected Go 1.24.x, found: $goVersion"
        Write-Warning "CI uses Go 1.24.6 - version mismatch may cause different behavior"
    }

    # Clean test cache first
    Write-Info "Cleaning test cache..."
    go clean -testcache

    # Build step (unless skipped)
    if (-not $SkipBuild) {
        Write-Header "🔨 Building Components"
        Write-Info "Building all components..."
        
        $buildResult = & go build ./...
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Build failed! Fix build errors first."
            exit 1
        }
        Write-Success "Build completed successfully"
    }

    # Create test results directory
    $testDir = "test-results-local"
    if (Test-Path $testDir) {
        Remove-Item $testDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $testDir -Force | Out-Null

    # Determine which tests to run
    if ($Fast) {
        Write-Header "🚀 Running Failing Tests Only (Fast Mode)"
        $testPattern = $FailingTests -join "|"
        Write-Info "Test pattern: $testPattern"
    } elseif ($TestFilter) {
        Write-Header "🎯 Running Filtered Tests"
        $testPattern = $TestFilter
        Write-Info "Test filter: $testPattern"
    } else {
        Write-Header "🧪 Running Full Test Suite"
        $testPattern = ".*"
    }

    # Build test command
    $testCmd = @("go", "test")
    
    if ($Verbose) {
        $testCmd += "-v"
    }
    
    # Add race detection (like CI)
    $testCmd += "-race"
    
    # Timeout (like CI)
    $testCmd += "-timeout=10m"
    
    # Parallelism (like CI)
    $testCmd += "-parallel=4"
    
    # Coverage if requested
    if ($Coverage) {
        $testCmd += "-coverprofile=$testDir/coverage.out"
        $testCmd += "-covermode=atomic"
    }
    
    # Count=1 to avoid test caching
    $testCmd += "-count=1"
    
    # Test pattern
    if ($Fast -or $TestFilter) {
        $testCmd += "-run=$testPattern"
    }
    
    # JSON output for parsing
    $testCmd += "-json"
    
    # Target all packages
    $testCmd += "./..."

    Write-Info "Running command: $($testCmd -join ' ')"
    Write-Info "This replicates your CI environment exactly"
    Write-Host ""

    # Run tests and capture output
    $testOutput = & $testCmd[0] $testCmd[1..($testCmd.Length-1)] 2>&1
    $testExitCode = $LASTEXITCODE
    
    # Save raw output
    $testOutput | Out-File -FilePath "$testDir/test-output.log" -Encoding utf8
    
    # Parse JSON output to identify specific failures
    Write-Header "📊 Test Results Analysis"
    
    $failedTests = @()
    $passedTests = @()
    
    foreach ($line in $testOutput) {
        if ($line -match '"Action":"fail".*"Test":"([^"]+)"') {
            $failedTests += $matches[1]
        } elseif ($line -match '"Action":"pass".*"Test":"([^"]+)"') {
            $passedTests += $matches[1]
        }
    }
    
    # Display results
    if ($passedTests.Count -gt 0) {
        Write-Success "Passed tests: $($passedTests.Count)"
        if ($Verbose) {
            foreach ($test in $passedTests) {
                Write-Host "  ${Green}✓${Reset} $test"
            }
        }
    }
    
    if ($failedTests.Count -gt 0) {
        Write-Error "Failed tests: $($failedTests.Count)"
        foreach ($test in $failedTests) {
            Write-Host "  ${Red}✗${Reset} $test"
        }
        Write-Host ""
        
        # Show specific guidance for known failing tests
        Write-Header "🔧 Specific Fix Guidance for Failed Tests"
        
        foreach ($test in $failedTests) {
            switch -Regex ($test) {
                "TestFilePermissionValidation" {
                    Write-Host "${Yellow}📁 File Permission Issue:${Reset}"
                    Write-Host "   • Check file creation in your code uses 0o644 permissions"
                    Write-Host "   • Look for: os.WriteFile(path, data, 0o644)"
                    Write-Host "   • Current: 0640 (0x1a0), Expected: 0644 (0x1a4)"
                    Write-Host ""
                }
                "TestOnceMode_ExitCodes|TestMain_ExitCodes" {
                    Write-Host "${Yellow}🚪 Exit Code Issue:${Reset}"
                    Write-Host "   • Your main function should return exit code 8 when files fail"
                    Write-Host "   • Add: os.Exit(8) when someFilesFailed is true"
                    Write-Host "   • Current: exit code 0, Expected: exit code 8"
                    Write-Host ""
                }
                "TestValidateHandoffDir" {
                    Write-Host "${Yellow}📂 Handoff Directory Validation:${Reset}"
                    Write-Host "   • Invalid directories should return an error"
                    Write-Host "   • Add validation: if !isValidHandoffDir(dir) { return error }"
                    Write-Host "   • Current: no error returned, Expected: error for invalid dirs"
                    Write-Host ""
                }
                "TestConcurrentFileProcessing" {
                    Write-Host "${Yellow}🔄 Concurrent Processing Issue:${Reset}"
                    Write-Host "   • Files are not being processed exactly once"
                    Write-Host "   • Check worker synchronization and add locking if needed"
                    Write-Host "   • Current: 48 files processed, Expected: 50 files"
                    Write-Host ""
                }
                "TestTLSServerStartup" {
                    Write-Host "${Yellow}🔐 TLS Server Logging Issue:${Reset}"
                    Write-Host "   • Missing expected log messages during server startup"
                    Write-Host "   • Add: log.Info(`"Server starting (HTTP only)`")"
                    Write-Host "   • Add: log.Info(`"Server starting with TLS`")"
                    Write-Host ""
                }
                "TestProviderFactoryConfiguration" {
                    Write-Host "${Yellow}🏭 Provider Configuration Issue:${Reset}"
                    Write-Host "   • Invalid provider types should return an error"
                    Write-Host "   • Add: if !isValidProviderType(type) { return error }"
                    Write-Host "   • Current: no error for invalid types, Expected: error"
                    Write-Host ""
                }
            }
        }
    }
    
    # Coverage report generation
    if ($Coverage -and (Test-Path "$testDir/coverage.out")) {
        Write-Header "📈 Coverage Report Generation"
        
        Write-Info "Generating HTML coverage report..."
        go tool cover -html="$testDir/coverage.out" -o "$testDir/coverage.html"
        
        Write-Info "Generating function coverage summary..."
        $coverageText = go tool cover -func="$testDir/coverage.out"
        $coverageText | Out-File -FilePath "$testDir/coverage-summary.txt" -Encoding utf8
        
        # Extract total coverage
        $totalCoverage = ($coverageText | Select-String "total:" | ForEach-Object { $_.ToString().Split()[-1] })
        if ($totalCoverage) {
            Write-Success "Total Coverage: $totalCoverage"
            Write-Info "HTML Report: $testDir/coverage.html"
            Write-Info "Summary: $testDir/coverage-summary.txt"
        }
    }
    
    # Final status
    Write-Header "🎯 Final Status"
    
    if ($testExitCode -eq 0) {
        Write-Success "All tests passed! ✨"
        Write-Success "You can now push to your PR with confidence"
        
        if ($Fast) {
            Write-Info "Recommendation: Run full test suite before final push"
            Write-Info "Command: .\scripts\test-local-ci.ps1 -Coverage"
        }
    } else {
        Write-Error "Some tests failed! ⚠️"
        Write-Error "Fix the issues above before pushing to your PR"
        Write-Info "Test output saved to: $testDir/test-output.log"
        
        if (-not $Fast) {
            Write-Info "Tip: Use -Fast flag to focus on failing tests only"
            Write-Info "Command: .\scripts\test-local-ci.ps1 -Fast -Verbose"
        }
    }
    
    Write-Host ""
    Write-Host "${Magenta}📋 Next Steps:${Reset}"
    Write-Host "1. Fix the failing tests using the guidance above"
    Write-Host "2. Re-run this script to verify fixes"
    Write-Host "3. Use -Fast flag for quick iteration"
    Write-Host "4. Run full test suite before pushing: .\scripts\test-local-ci.ps1 -Coverage"
    Write-Host ""
    
    exit $testExitCode

} catch {
    Write-Error "Script failed with error: $($_.Exception.Message)"
    Write-Error "Stack trace: $($_.ScriptStackTrace)"
    exit 1
} finally {
    Set-Location $OriginalLocation
}