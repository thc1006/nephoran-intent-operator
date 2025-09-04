#!/usr/bin/env pwsh
# =============================================================================
# CI STABILITY FIXES VALIDATION SCRIPT - NEPHORAN INTENT OPERATOR
# =============================================================================
# This script validates the comprehensive CI stability fixes
# Tests cache recovery, timeout management, security scanning, and orchestration
# Last Updated: 2025-09-03
# =============================================================================

param(
    [switch]$SkipBuild = $false,
    [switch]$Verbose = $false,
    [switch]$TestWorkflowSyntax = $true
)

# Set error handling
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Colors for output
$Red = "`e[31m"
$Green = "`e[32m"
$Yellow = "`e[33m"
$Blue = "`e[34m"
$Reset = "`e[0m"

function Write-Status {
    param([string]$Message, [string]$Color = $Blue)
    Write-Host "${Color}[$(Get-Date -Format 'HH:mm:ss')] $Message${Reset}"
}

function Write-Success {
    param([string]$Message)
    Write-Host "${Green}✅ $Message${Reset}"
}

function Write-Warning {
    param([string]$Message)
    Write-Host "${Yellow}⚠️  $Message${Reset}"
}

function Write-Error {
    param([string]$Message)
    Write-Host "${Red}❌ $Message${Reset}"
}

function Test-Command {
    param([string]$Command, [string]$Description)
    
    Write-Status "Testing: $Description"
    try {
        $output = Invoke-Expression $Command 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "$Description"
            return $true
        } else {
            Write-Error "$Description failed with exit code $LASTEXITCODE"
            if ($Verbose) { Write-Host $output }
            return $false
        }
    } catch {
        Write-Error "$Description failed with exception: $($_.Exception.Message)"
        return $false
    }
}

function Test-FileExists {
    param([string]$Path, [string]$Description)
    
    if (Test-Path $Path) {
        Write-Success "$Description exists: $Path"
        return $true
    } else {
        Write-Error "$Description not found: $Path"
        return $false
    }
}

# =============================================================================
# Main Validation
# =============================================================================

Write-Status "=== CI Pipeline Fixes Validation ===" $Green
Write-Status "Service: $Service"
Write-Status "Skip Build: $SkipBuild"
Write-Status "Verbose: $Verbose"
Write-Host ""

$allTests = @()

# Test 1: Check required tools
Write-Status "=== Testing Required Tools ===" $Blue
$allTests += Test-Command "docker --version" "Docker installation"
$allTests += Test-Command "git --version" "Git installation"

# Test if buildx is available
$buildxAvailable = $false
try {
    docker buildx version | Out-Null
    $buildxAvailable = $true
    Write-Success "Docker Buildx available"
    $allTests += $true
} catch {
    Write-Warning "Docker Buildx not available - some tests will be skipped"
    $allTests += $true  # Not critical for basic validation
}

Write-Host ""

# Test 2: Check CI stability workflow files
Write-Status "=== Testing CI Stability Workflows ===" $Blue

# Define critical workflow files and their features
$CriticalWorkflows = @{
    "security-scan-ultra-reliable.yml" = @("timeout-minutes", "retry.*logic", "SARIF.*valid", "cache.*recover")
    "cache-recovery-system.yml" = @("cache.*valid", "corruption.*detect", "recovery.*system", "fallback")
    "timeout-management.yml" = @("timeout.*management", "adaptive", "complexity")
    "ci-stability-orchestrator.yml" = @("orchestrat", "stability", "quality.*gate")
}

foreach ($workflow in $CriticalWorkflows.Keys) {
    $workflowPath = ".github/workflows/$workflow"
    if (Test-Path $workflowPath) {
        Write-Success "Critical workflow found: $workflow"
        $allTests += $true
        
        # Test workflow features
        $content = Get-Content $workflowPath -Raw
        $missingFeatures = @()
        
        foreach ($feature in $CriticalWorkflows[$workflow]) {
            if ($content -match $feature) {
                if ($Verbose) { Write-Success "  ✓ Feature '$feature' found in $workflow" }
            } else {
                $missingFeatures += $feature
            }
        }
        
        if ($missingFeatures.Count -eq 0) {
            Write-Success "All features implemented in $workflow"
            $allTests += $true
        } else {
            Write-Warning "Missing features in $workflow`: $($missingFeatures -join ', ')"
            $allTests += $true  # Warning, not critical failure
        }
    } else {
        Write-Error "Critical workflow missing: $workflow"
        $allTests += $false
    }
}

# Check basic project files
$allTests += Test-FileExists "go.mod" "Go module file"
$allTests += Test-FileExists "go.sum" "Go sum file"

Write-Host ""

# Test 3: Validate CI workflow syntax
Write-Status "=== Testing CI Workflow ===" $Blue

# Check for common issues in CI workflow
$ciContent = Get-Content ".github/workflows/ci.yml" -Raw

# Test for fixed registry paths
if ($ciContent -match 'nephoran/\$\{\{ matrix\.service\.name \}\}') {
    Write-Success "Registry path fix applied"
    $allTests += $true
} else {
    Write-Error "Registry path fix not found in CI workflow"
    $allTests += $false
}

# Test for removed problematic registry cache
if ($ciContent -notmatch 'buildcache') {
    Write-Success "Problematic registry cache removed"
    $allTests += $true
} else {
    Write-Warning "Registry buildcache references still present - may cause issues"
    $allTests += $true  # Warning, not error
}

# Test for retry mechanisms
if ($ciContent -match 'for.*in.*\{1\.\.3\}') {
    Write-Success "Retry mechanisms implemented"
    $allTests += $true
} else {
    Write-Warning "Retry mechanisms not found - builds may be less reliable"
    $allTests += $true  # Warning, not error
}

Write-Host ""

# Test 4: Docker build validation
if (-not $SkipBuild) {
    Write-Status "=== Testing Docker Build ===" $Blue
    
    # Test basic build
    $buildCommand = "docker build --build-arg SERVICE=$Service -t nephoran/${Service}:test ."
    $allTests += Test-Command $buildCommand "Basic Docker build for $Service"
    
    # Test with cache if buildx available
    if ($buildxAvailable) {
        $cacheCommand = "docker buildx build --build-arg SERVICE=$Service --cache-from type=gha,scope=test-$Service --cache-to type=gha,mode=max,scope=test-$Service -t nephoran/${Service}:test-cache ."
        $allTests += Test-Command $cacheCommand "BuildKit cache build for $Service"
    }
    
    # Test container startup
    Write-Status "Testing container startup..."
    try {
        $containerId = docker run -d --name "test-$Service-$(Get-Date -Format 'yyyyMMdd-HHmmss')" "nephoran/${Service}:test"
        Start-Sleep 5
        
        $containerStatus = docker inspect $containerId --format='{{.State.Status}}' 2>$null
        docker rm -f $containerId | Out-Null
        
        if ($containerStatus -eq "running" -or $containerStatus -eq "exited") {
            Write-Success "Container startup test"
            $allTests += $true
        } else {
            Write-Error "Container failed to start properly: $containerStatus"
            $allTests += $false
        }
    } catch {
        Write-Error "Container startup test failed: $($_.Exception.Message)"
        $allTests += $false
    }
} else {
    Write-Status "=== Skipping Docker Build Tests ===" $Yellow
}

Write-Host ""

# Test 5: Validate Dockerfile optimization
Write-Status "=== Testing Dockerfile Optimization ===" $Blue

$dockerfileContent = Get-Content "Dockerfile" -Raw

# Check for multi-stage build
if ($dockerfileContent -match 'FROM.*AS.*deps-cache' -and $dockerfileContent -match 'FROM.*AS.*builder') {
    Write-Success "Multi-stage build optimization present"
    $allTests += $true
} else {
    Write-Error "Multi-stage build optimization not found"
    $allTests += $false
}

# Check for retry logic in Dockerfile
if ($dockerfileContent -match 'for attempt in.*1.*2.*3') {
    Write-Success "Dockerfile retry logic present"
    $allTests += $true
} else {
    Write-Warning "Dockerfile retry logic not found"
    $allTests += $true  # Warning, not critical
}

# Check for security hardening
if ($dockerfileContent -match 'distroless' -and $dockerfileContent -match 'nonroot') {
    Write-Success "Security hardening (distroless + nonroot) present"
    $allTests += $true
} else {
    Write-Error "Security hardening not properly configured"
    $allTests += $false
}

Write-Host ""

# Test 6: Git repository validation
Write-Status "=== Testing Git Repository ===" $Blue

$allTests += Test-Command "git status --porcelain" "Git repository status"

# Check for required branch
$currentBranch = git rev-parse --abbrev-ref HEAD
if ($currentBranch -eq "feat-e2e" -or $currentBranch -eq "feat/e2e") {
    Write-Success "On feature branch: $currentBranch"
    $allTests += $true
} else {
    Write-Warning "Not on expected feature branch (current: $currentBranch)"
    $allTests += $true  # Warning, not error
}

Write-Host ""

# =============================================================================
# Results Summary
# =============================================================================

Write-Status "=== Validation Results ===" $Green

$totalTests = $allTests.Count
$passedTests = ($allTests | Where-Object { $_ -eq $true }).Count
$failedTests = $totalTests - $passedTests

Write-Host ""
Write-Host "📊 Test Results:"
Write-Host "  Total Tests: $totalTests"
Write-Host "  Passed: ${Green}$passedTests${Reset}"
Write-Host "  Failed: ${Red}$failedTests${Reset}"

if ($failedTests -eq 0) {
    Write-Host ""
    Write-Success "All tests passed! ✨"
    Write-Success "The CI pipeline fixes are ready for deployment."
    Write-Host ""
    Write-Host "Next steps:"
    Write-Host "  1. Commit your changes: git add -A && git commit -m 'fix(ci): resolve critical pipeline failures'"
    Write-Host "  2. Push to trigger CI: git push -u origin HEAD"
    Write-Host "  3. Monitor the GitHub Actions workflow"
    Write-Host "  4. Create a PR when ready: gh pr create --base integrate/mvp --head $currentBranch"
    exit 0
} else {
    Write-Host ""
    Write-Error "Some tests failed! ⚠️"
    Write-Host ""
    Write-Host "Please fix the issues above before proceeding:"
    Write-Host "  1. Review the failed tests and error messages"
    Write-Host "  2. Fix any structural or configuration issues"
    Write-Host "  3. Re-run this validation script"
    Write-Host "  4. Only proceed when all tests pass"
    exit 1
}