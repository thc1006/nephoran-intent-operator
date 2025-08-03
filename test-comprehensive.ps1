#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Comprehensive test execution script for Nephoran Intent Operator
.DESCRIPTION
    This script runs all test suites including unit tests, integration tests, 
    CRD validation tests, LLM processor tests, and Windows compatibility tests.
.PARAMETER TestCategory
    Specific test category to run (controller, integration, api, service, windows)
.PARAMETER Verbose
    Enable verbose test output
.PARAMETER FailFast
    Stop on first test failure
.PARAMETER Coverage
    Generate test coverage report
.EXAMPLE
    .\test-comprehensive.ps1 -TestCategory controller -Verbose
    .\test-comprehensive.ps1 -Coverage -FailFast
#>

param(
    [Parameter(HelpMessage="Test category to run: controller, integration, api, service, windows, all")]
    [ValidateSet("controller", "integration", "api", "service", "windows", "all")]
    [string]$TestCategory = "all",
    
    [Parameter(HelpMessage="Enable verbose test output")]
    [switch]$Verbose,
    
    [Parameter(HelpMessage="Stop on first test failure")]
    [switch]$FailFast,
    
    [Parameter(HelpMessage="Generate test coverage report")]
    [switch]$Coverage,
    
    [Parameter(HelpMessage="Skip environment validation")]
    [switch]$SkipValidation,
    
    [Parameter(HelpMessage="Test timeout in minutes")]
    [int]$TimeoutMinutes = 10
)

# Set error handling
$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# Color output functions
function Write-ColorOutput($Message, $Color = "White") {
    Write-Host $Message -ForegroundColor $Color
}

function Write-Success($Message) {
    Write-ColorOutput "✅ $Message" "Green"
}

function Write-Error($Message) {
    Write-ColorOutput "❌ $Message" "Red"
}

function Write-Warning($Message) {
    Write-ColorOutput "⚠️  $Message" "Yellow"
}

function Write-Info($Message) {
    Write-ColorOutput "ℹ️  $Message" "Cyan"
}

function Write-Header($Message) {
    Write-Host ""
    Write-ColorOutput "═══════════════════════════════════════════════════════════════════" "Magenta"
    Write-ColorOutput " $Message" "Magenta"
    Write-ColorOutput "═══════════════════════════════════════════════════════════════════" "Magenta"
    Write-Host ""
}

# Validate prerequisites
function Test-Prerequisites {
    Write-Header "Validating Prerequisites"
    
    $errors = @()
    
    # Check Go installation
    try {
        $goVersion = go version 2>$null
        if ($goVersion -match "go(\d+\.\d+)") {
            Write-Success "Go version: $goVersion"
        } else {
            $errors += "Go is not installed or not in PATH"
        }
    } catch {
        $errors += "Go is not installed or not accessible"
    }
    
    # Check required Go tools
    $tools = @("ginkgo", "gocov", "gocov-html")
    foreach ($tool in $tools) {
        try {
            $null = Get-Command $tool -ErrorAction Stop 2>$null
            Write-Success "$tool is available"
        } catch {
            if ($Coverage -and $tool -like "gocov*") {
                Write-Warning "$tool not found - coverage reporting may not work"
            } elseif ($tool -eq "ginkgo") {
                Write-Info "Installing ginkgo..."
                try {
                    go install -a github.com/onsi/ginkgo/v2/ginkgo@latest
                    Write-Success "ginkgo installed successfully"
                } catch {
                    $errors += "Failed to install ginkgo"
                }
            }
        }
    }
    
    # Check Kubernetes tools (if not skipping validation)
    if (-not $SkipValidation) {
        try {
            $kubectlVersion = kubectl version --client 2>$null
            Write-Success "kubectl is available"
        } catch {
            Write-Warning "kubectl not found - some integration tests may fail"
        }
    }
    
    # Check environment variables
    $envVars = @("KUBEBUILDER_ASSETS")
    foreach ($var in $envVars) {
        $value = [Environment]::GetEnvironmentVariable($var)
        if ($value) {
            Write-Success "$var is set"
        } else {
            Write-Warning "$var is not set - may cause test failures"
        }
    }
    
    if ($errors.Count -gt 0) {
        Write-Error "Prerequisites validation failed:"
        foreach ($error in $errors) {
            Write-Host "  • $error" -ForegroundColor Red
        }
        exit 1
    }
    
    Write-Success "All prerequisites validated successfully"
}

# Setup test environment
function Initialize-TestEnvironment {
    Write-Header "Initializing Test Environment"
    
    # Set test environment variables
    $env:TEST_CATEGORY = $TestCategory
    $env:TEST_TIMEOUT = "$($TimeoutMinutes)m"
    $env:GINKGO_EDITOR_INTEGRATION = "true"
    
    # Create test artifacts directory
    $testDir = Join-Path $PWD "test-artifacts"
    if (-not (Test-Path $testDir)) {
        New-Item -ItemType Directory -Path $testDir -Force | Out-Null
        Write-Success "Created test artifacts directory: $testDir"
    }
    
    # Setup kubebuilder assets if needed
    if (-not $env:KUBEBUILDER_ASSETS) {
        Write-Info "Setting up kubebuilder test assets..."
        try {
            $assetsPath = Join-Path $testDir "kubebuilder-assets"
            if (-not (Test-Path $assetsPath)) {
                # Install kubebuilder test assets
                $setupEnvTest = "sigs.k8s.io/controller-runtime/tools/setup-envtest@latest"
                go install $setupEnvTest
                
                $setupCmd = Get-Command "setup-envtest" -ErrorAction SilentlyContinue
                if ($setupCmd) {
                    $assetsOutput = & setup-envtest use --bin-dir $assetsPath 1.29.0 --print path
                    $env:KUBEBUILDER_ASSETS = $assetsOutput.Trim()
                    Write-Success "Kubebuilder assets configured: $env:KUBEBUILDER_ASSETS"
                }
            } else {
                $env:KUBEBUILDER_ASSETS = $assetsPath
                Write-Success "Using existing kubebuilder assets: $assetsPath"
            }
        } catch {
            Write-Warning "Failed to setup kubebuilder assets: $($_.Exception.Message)"
        }
    }
    
    Write-Success "Test environment initialized successfully"
}

# Run specific test category
function Invoke-TestCategory($Category, $TestPaths) {
    Write-Header "Running $Category Tests"
    
    $ginkgoArgs = @()
    
    # Add verbosity
    if ($Verbose) {
        $ginkgoArgs += "-v"
    }
    
    # Add fail fast
    if ($FailFast) {
        $ginkgoArgs += "--fail-fast"
    }
    
    # Add timeout
    $ginkgoArgs += "--timeout=$($TimeoutMinutes)m"
    
    # Add coverage if requested
    if ($Coverage) {
        $coverageFile = "test-artifacts/coverage-$Category.out"
        $ginkgoArgs += "--cover"
        $ginkgoArgs += "--coverprofile=$coverageFile"
    }
    
    # Add test paths
    $ginkgoArgs += $TestPaths
    
    Write-Info "Executing: ginkgo $($ginkgoArgs -join ' ')"
    
    try {
        $startTime = Get-Date
        & ginkgo @ginkgoArgs
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "$Category tests completed successfully in $($duration.TotalSeconds.ToString('F2')) seconds"
            return $true
        } else {
            Write-Error "$Category tests failed with exit code $LASTEXITCODE"
            return $false
        }
    } catch {
        Write-Error "$Category tests failed: $($_.Exception.Message)"
        return $false
    }
}

# Generate coverage reports
function New-CoverageReport {
    Write-Header "Generating Coverage Report"
    
    $coverageFiles = Get-ChildItem "test-artifacts/coverage-*.out" -ErrorAction SilentlyContinue
    if (-not $coverageFiles) {
        Write-Warning "No coverage files found"
        return
    }
    
    # Merge coverage files
    $mergedCoverage = "test-artifacts/coverage-merged.out"
    try {
        # Create header
        "mode: atomic" | Out-File -FilePath $mergedCoverage -Encoding utf8
        
        # Merge all coverage data
        foreach ($file in $coverageFiles) {
            $content = Get-Content $file | Select-Object -Skip 1
            $content | Out-File -FilePath $mergedCoverage -Append -Encoding utf8
        }
        
        Write-Success "Merged coverage data: $mergedCoverage"
        
        # Generate HTML report if gocov tools are available
        try {
            $htmlReport = "test-artifacts/coverage-report.html"
            gocov convert $mergedCoverage | gocov-html > $htmlReport
            Write-Success "HTML coverage report: $htmlReport"
            
            # Calculate coverage percentage
            $coveragePercent = gocov convert $mergedCoverage | gocov report | Select-String "Total Coverage:" | ForEach-Object { 
                $_.Line -replace ".*Total Coverage:\s*([0-9.]+)%.*", '$1' 
            }
            
            if ($coveragePercent) {
                Write-Success "Total Coverage: $coveragePercent%"
            }
        } catch {
            Write-Warning "Failed to generate HTML coverage report: $($_.Exception.Message)"
        }
    } catch {
        Write-Error "Failed to merge coverage data: $($_.Exception.Message)"
    }
}

# Main execution
function Main {
    $overallStartTime = Get-Date
    $testResults = @{}
    
    Write-Header "Nephoran Intent Operator - Comprehensive Test Suite"
    Write-Info "Test Category: $TestCategory"
    Write-Info "Coverage: $Coverage"
    Write-Info "Fail Fast: $FailFast"
    Write-Info "Timeout: $TimeoutMinutes minutes"
    Write-Info "Platform: $($PSVersionTable.Platform)"
    Write-Info "PowerShell: $($PSVersionTable.PSVersion)"
    
    # Validate prerequisites
    if (-not $SkipValidation) {
        Test-Prerequisites
    }
    
    # Initialize test environment
    Initialize-TestEnvironment
    
    # Define test categories and their paths
    $testCategories = @{
        "controller" = @("./pkg/controllers")
        "integration" = @("./pkg/controllers/integration_test.go", "./pkg/controllers/error_recovery_test.go")
        "api" = @("./pkg/controllers/crd_validation_test.go")
        "service" = @("./pkg/controllers/llm_processor_service_test.go", "./cmd/llm-processor")
        "windows" = @("./pkg/controllers/windows_compatibility_test.go")
    }
    
    # Determine which tests to run
    $categoriesToRun = @()
    if ($TestCategory -eq "all") {
        $categoriesToRun = $testCategories.Keys
    } else {
        $categoriesToRun = @($TestCategory)
    }
    
    # Run each test category
    foreach ($category in $categoriesToRun) {
        if (-not $testCategories.ContainsKey($category)) {
            Write-Warning "Unknown test category: $category"
            continue
        }
        
        $testPaths = $testCategories[$category]
        $result = Invoke-TestCategory $category $testPaths
        $testResults[$category] = $result
        
        if (-not $result -and $FailFast) {
            Write-Error "Stopping execution due to test failure (fail-fast enabled)"
            break
        }
    }
    
    # Generate coverage report if requested
    if ($Coverage) {
        New-CoverageReport
    }
    
    # Summary
    $overallEndTime = Get-Date
    $overallDuration = $overallEndTime - $overallStartTime
    
    Write-Header "Test Execution Summary"
    Write-Info "Total Duration: $($overallDuration.TotalMinutes.ToString('F2')) minutes"
    
    $successCount = 0
    $failureCount = 0
    
    foreach ($category in $testResults.Keys) {
        $result = $testResults[$category]
        if ($result) {
            Write-Success "$category tests: PASSED"
            $successCount++
        } else {
            Write-Error "$category tests: FAILED"
            $failureCount++
        }
    }
    
    Write-Host ""
    if ($failureCount -eq 0) {
        Write-Success "All test categories passed! ($successCount/$($testResults.Count))"
        exit 0
    } else {
        Write-Error "Some test categories failed! ($failureCount failed, $successCount passed)"
        exit 1
    }
}

# Execute main function
try {
    Main
} catch {
    Write-Error "Test execution failed: $($_.Exception.Message)"
    Write-Host $_.ScriptStackTrace -ForegroundColor Red
    exit 1
}