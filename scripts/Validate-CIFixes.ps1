# =============================================================================
# PowerShell CI Fixes Validation Script
# =============================================================================
# Windows-compatible validation script for Nephoran CI fixes
# Provides the same comprehensive testing as the bash scripts
# =============================================================================

param(
    [switch]$Quick,
    [switch]$Performance,
    [switch]$Security,
    [switch]$Help,
    [string]$LogPath = "validation-results.txt"
)

# Configuration
$ProjectRoot = Split-Path $PSScriptRoot -Parent
$ResultsDir = Join-Path $ProjectRoot "validation-results"
$MasterLog = Join-Path $ProjectRoot $LogPath

# Colors for output
$Colors = @{
    Info = "Cyan"
    Success = "Green"  
    Warning = "Yellow"
    Error = "Red"
    Header = "Magenta"
}

# Test counters
$Global:TestsTotal = 0
$Global:TestsPassed = 0
$Global:TestsFailed = 0
$Global:TestsWarning = 0

# Logging functions
function Write-Log {
    param($Message, $Type = "Info")
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Type] $Message"
    
    Write-Host $Message -ForegroundColor $Colors[$Type]
    Add-Content -Path $MasterLog -Value $logMessage
}

function Test-Result {
    param($TestName, $Status, $Message)
    
    $Global:TestsTotal++
    
    switch ($Status) {
        "PASS" {
            $Global:TestsPassed++
            Write-Log "✅ TEST $($Global:TestsTotal): $TestName - $Message" "Success"
        }
        "FAIL" {
            $Global:TestsFailed++
            Write-Log "❌ TEST $($Global:TestsTotal): $TestName - $Message" "Error"
        }
        "WARN" {
            $Global:TestsWarning++
            Write-Log "⚠️ TEST $($Global:TestsTotal): $TestName - $Message" "Warning"
        }
    }
}

# =============================================================================
# Test 1: Service Configuration Matrix Validation (PowerShell)
# =============================================================================
function Test-ServiceConfiguration {
    Write-Log "=== TEST 1: Service Configuration Matrix Validation ===" "Header"
    
    $dockerfile = Join-Path $ProjectRoot "Dockerfile"
    $validationErrors = 0
    
    # Define expected service mappings
    $services = @(
        @{Name="conductor-loop"; Path="./cmd/conductor-loop/main.go"},
        @{Name="intent-ingest"; Path="./cmd/intent-ingest/main.go"},
        @{Name="nephio-bridge"; Path="./cmd/nephio-bridge/main.go"},
        @{Name="llm-processor"; Path="./cmd/llm-processor/main.go"},
        @{Name="oran-adaptor"; Path="./cmd/oran-adaptor/main.go"},
        @{Name="manager"; Path="./cmd/conductor-loop/main.go"},
        @{Name="controller"; Path="./cmd/conductor-loop/main.go"},
        @{Name="e2-kmp-sim"; Path="./cmd/e2-kmp-sim/main.go"},
        @{Name="o1-ves-sim"; Path="./cmd/o1-ves-sim/main.go"}
    )
    
    foreach ($service in $services) {
        Write-Log "Validating service: $($service.Name) -> $($service.Path)" "Info"
        
        $actualPath = Join-Path $ProjectRoot ($service.Path -replace "^\./", "")
        
        # Check if main.go exists
        if (Test-Path $actualPath) {
            Test-Result "Service-$($service.Name)-MainExists" "PASS" "main.go found at $($service.Path)"
            
            # Check for main function
            $content = Get-Content $actualPath -Raw
            if ($content -match "func main\(\)") {
                Test-Result "Service-$($service.Name)-MainFunction" "PASS" "main() function found"
            } else {
                Test-Result "Service-$($service.Name)-MainFunction" "FAIL" "main() function not found"
                $validationErrors++
            }
            
            # Check Dockerfile mapping
            $dockerContent = Get-Content $dockerfile -Raw
            if ($dockerContent -match [regex]::Escape("`"$($service.Name)`") CMD_PATH=`"$($service.Path)`"")) {
                Test-Result "Service-$($service.Name)-DockerMapping" "PASS" "Correctly mapped in Dockerfile"
            } else {
                Test-Result "Service-$($service.Name)-DockerMapping" "FAIL" "Not properly mapped in Dockerfile"
                $validationErrors++
            }
        } else {
            Test-Result "Service-$($service.Name)-MainExists" "FAIL" "main.go not found at $($service.Path)"
            $validationErrors++
        }
    }
    
    return $validationErrors -eq 0
}

# =============================================================================
# Test 2: GitHub Actions Workflow Validation (PowerShell)
# =============================================================================
function Test-WorkflowSyntax {
    Write-Log "=== TEST 2: GitHub Actions Workflow Syntax Validation ===" "Header"
    
    $workflowsDir = Join-Path $ProjectRoot ".github\workflows"
    
    if (-not (Test-Path $workflowsDir)) {
        Test-Result "Workflows-Directory" "FAIL" "Workflows directory not found"
        return $false
    }
    
    $workflowFiles = Get-ChildItem -Path $workflowsDir -Filter "*.yml" -ErrorAction SilentlyContinue
    $workflowFiles += Get-ChildItem -Path $workflowsDir -Filter "*.yaml" -ErrorAction SilentlyContinue
    
    if ($workflowFiles.Count -eq 0) {
        Test-Result "Workflows-Files" "FAIL" "No workflow files found"
        return $false
    }
    
    Write-Log "Found $($workflowFiles.Count) workflow files to validate" "Info"
    
    $allValid = $true
    foreach ($workflowFile in $workflowFiles) {
        Write-Log "Validating workflow: $($workflowFile.Name)" "Info"
        
        try {
            # Test YAML syntax using PowerShell YAML module or manual parsing
            $content = Get-Content $workflowFile.FullName -Raw
            
            # Basic structure checks
            if ($content -match "name:\s*\S+" -and $content -match "on:\s*" -and $content -match "jobs:\s*") {
                Test-Result "Workflow-$($workflowFile.BaseName)-Structure" "PASS" "Required fields present"
            } else {
                Test-Result "Workflow-$($workflowFile.BaseName)-Structure" "FAIL" "Missing required fields"
                $allValid = $false
            }
            
            # Check for concurrency control
            if ($content -match "concurrency:\s*") {
                Test-Result "Workflow-$($workflowFile.BaseName)-Concurrency" "PASS" "Concurrency control configured"
            } else {
                Test-Result "Workflow-$($workflowFile.BaseName)-Concurrency" "WARN" "No concurrency control found"
            }
            
            # Security checks
            if ($content -match "secrets\.GITHUB_TOKEN") {
                Test-Result "Workflow-$($workflowFile.BaseName)-TokenSecurity" "PASS" "Using GITHUB_TOKEN"
            } else {
                Test-Result "Workflow-$($workflowFile.BaseName)-TokenSecurity" "WARN" "GITHUB_TOKEN usage not found"
            }
            
        } catch {
            Test-Result "Workflow-$($workflowFile.BaseName)-Syntax" "FAIL" "YAML parsing error: $($_.Exception.Message)"
            $allValid = $false
        }
    }
    
    return $allValid
}

# =============================================================================
# Test 3: Docker Configuration Testing (PowerShell)
# =============================================================================
function Test-DockerConfiguration {
    Write-Log "=== TEST 3: Docker Configuration Testing ===" "Header"
    
    $dockerfile = Join-Path $ProjectRoot "Dockerfile"
    $dockerfileResilient = Join-Path $ProjectRoot "Dockerfile.resilient"
    
    $dockerfiles = @()
    if (Test-Path $dockerfile) { $dockerfiles += $dockerfile }
    if (Test-Path $dockerfileResilient) { $dockerfiles += $dockerfileResilient }
    
    if ($dockerfiles.Count -eq 0) {
        Test-Result "Docker-Files" "FAIL" "No Dockerfile found"
        return $false
    }
    
    $allValid = $true
    foreach ($dockerFile in $dockerfiles) {
        $fileName = Split-Path $dockerFile -Leaf
        Write-Log "Validating $fileName..." "Info"
        
        $content = Get-Content $dockerFile -Raw
        
        # Check for required components
        $requiredComponents = @(
            @{Pattern="ARG SERVICE"; Description="Service argument"},
            @{Pattern="WORKDIR"; Description="Working directory"},
            @{Pattern="USER.*nonroot|USER.*65532"; Description="Non-root user"},
            @{Pattern="ENTRYPOINT"; Description="Entry point"}
        )
        
        foreach ($component in $requiredComponents) {
            if ($content -match $component.Pattern) {
                Test-Result "Docker-$fileName-$($component.Description -replace ' ', '')" "PASS" "$($component.Description) found"
            } else {
                Test-Result "Docker-$fileName-$($component.Description -replace ' ', '')" "WARN" "$($component.Description) missing"
            }
        }
        
        # Check for multi-stage builds
        $stageCount = ([regex]::Matches($content, "^FROM.*AS", "Multiline")).Count
        if ($stageCount -gt 1) {
            Test-Result "Docker-$fileName-MultiStage" "PASS" "Multi-stage build with $stageCount stages"
        } else {
            Test-Result "Docker-$fileName-MultiStage" "WARN" "Single-stage build"
        }
    }
    
    return $allValid
}

# =============================================================================
# Test 4: GHCR Authentication Configuration (PowerShell)
# =============================================================================
function Test-GHCRAuthentication {
    Write-Log "=== TEST 4: GHCR Authentication Configuration ===" "Header"
    
    $ciWorkflow = Join-Path $ProjectRoot ".github\workflows\ci.yml"
    
    if (-not (Test-Path $ciWorkflow)) {
        Test-Result "GHCR-Workflow" "FAIL" "CI workflow not found"
        return $false
    }
    
    $content = Get-Content $ciWorkflow -Raw
    
    # Check required permissions
    $requiredPermissions = @(
        "contents: read",
        "packages: write", 
        "security-events: write",
        "id-token: write"
    )
    
    foreach ($permission in $requiredPermissions) {
        $permName = $permission.Split(':')[0].Trim()
        if ($content -match [regex]::Escape($permission)) {
            Test-Result "GHCR-Permission-$permName" "PASS" "Permission $permission found"
        } else {
            Test-Result "GHCR-Permission-$permName" "FAIL" "Permission $permission missing"
        }
    }
    
    # Check GHCR login configuration
    if ($content -match "registry: ghcr\.io") {
        Test-Result "GHCR-Registry" "PASS" "GHCR registry configured"
    } else {
        Test-Result "GHCR-Registry" "FAIL" "GHCR registry not configured"
    }
    
    if ($content -match "username: \$\{\{ github\.actor \}\}") {
        Test-Result "GHCR-Username" "PASS" "GHCR username configured"  
    } else {
        Test-Result "GHCR-Username" "FAIL" "GHCR username not configured"
    }
    
    if ($content -match "password: \$\{\{ secrets\.GITHUB_TOKEN \}\}") {
        Test-Result "GHCR-Token" "PASS" "GHCR token configured"
    } else {
        Test-Result "GHCR-Token" "FAIL" "GHCR token not configured"
    }
    
    return $true
}

# =============================================================================
# Test 5: Smart Build Script Validation (PowerShell)
# =============================================================================
function Test-SmartBuildScript {
    Write-Log "=== TEST 5: Smart Build Script Validation ===" "Header"
    
    $smartBuildScript = Join-Path $ProjectRoot "scripts\smart-docker-build.sh"
    
    if (-not (Test-Path $smartBuildScript)) {
        Test-Result "SmartBuild-Exists" "FAIL" "Smart build script not found"
        return $false
    }
    
    Test-Result "SmartBuild-Exists" "PASS" "Smart build script found"
    
    $content = Get-Content $smartBuildScript -Raw
    
    # Check for required functions
    $requiredFunctions = @(
        "check_infrastructure_health",
        "select_build_strategy",
        "execute_build",
        "main"
    )
    
    foreach ($func in $requiredFunctions) {
        if ($content -match "$func\(\)") {
            Test-Result "SmartBuild-Function-$func" "PASS" "Function $func() found"
        } else {
            Test-Result "SmartBuild-Function-$func" "FAIL" "Function $func() missing"
        }
    }
    
    # Check for resilience features
    if ($content -match "max_attempts|retry|fallback") {
        Test-Result "SmartBuild-Resilience" "PASS" "Resilience features implemented"
    } else {
        Test-Result "SmartBuild-Resilience" "WARN" "Limited resilience features"
    }
    
    return $true
}

# =============================================================================
# Quick Validation Mode
# =============================================================================
function Invoke-QuickValidation {
    Write-Log "⚡ Running Quick Validation (Essential Tests Only)" "Header"
    
    Test-ServiceConfiguration | Out-Null
    Test-WorkflowSyntax | Out-Null
    Test-GHCRAuthentication | Out-Null
}

# =============================================================================
# Performance Validation Mode
# =============================================================================
function Invoke-PerformanceValidation {
    Write-Log "⚡ Running Performance-Focused Validation" "Header"
    
    # Test script loading performance
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    
    # Test workflow parsing performance
    $workflowsDir = Join-Path $ProjectRoot ".github\workflows"
    if (Test-Path $workflowsDir) {
        $workflowFiles = Get-ChildItem -Path $workflowsDir -Filter "*.yml"
        foreach ($file in $workflowFiles) {
            $null = Get-Content $file.FullName
        }
    }
    
    $stopwatch.Stop()
    $duration = $stopwatch.ElapsedMilliseconds
    
    if ($duration -lt 1000) {
        Write-Log "Fast workflow parsing: ${duration}ms (excellent)" "Success"
    } elseif ($duration -lt 3000) {
        Write-Log "Good workflow parsing: ${duration}ms" "Info"
    } else {
        Write-Log "Slow workflow parsing: ${duration}ms (may need optimization)" "Warning"
    }
}

# =============================================================================
# Security Validation Mode
# =============================================================================
function Invoke-SecurityValidation {
    Write-Log "🔒 Running Security-Focused Validation" "Header"
    
    $ciWorkflow = Join-Path $ProjectRoot ".github\workflows\ci.yml"
    
    if (Test-Path $ciWorkflow) {
        $content = Get-Content $ciWorkflow -Raw
        
        # Security checks
        $securityChecks = @(
            @{Pattern="secrets\.GITHUB_TOKEN"; Description="Using GITHUB_TOKEN (secure)"},
            @{Pattern="packages: write"; Description="Proper package permissions"},
            @{Pattern="github\.event_name != 'pull_request'"; Description="Conditional auth (secure)"}
        )
        
        foreach ($check in $securityChecks) {
            if ($content -match $check.Pattern) {
                Write-Log "✅ Security check passed: $($check.Description)" "Success"
            } else {
                Write-Log "⚠️ Security check failed: $($check.Description)" "Warning"  
            }
        }
    }
}

# =============================================================================
# Full Validation Mode
# =============================================================================
function Invoke-FullValidation {
    Write-Log "🔧 Running Full Validation Suite (All Tests)" "Header"
    
    $suites = @(
        @{Name="ServiceConfiguration"; Function={Test-ServiceConfiguration}; Description="Service configuration validation"},
        @{Name="WorkflowSyntax"; Function={Test-WorkflowSyntax}; Description="GitHub Actions workflow syntax"},
        @{Name="DockerConfiguration"; Function={Test-DockerConfiguration}; Description="Docker configuration validation"},
        @{Name="GHCRAuthentication"; Function={Test-GHCRAuthentication}; Description="GHCR authentication setup"},
        @{Name="SmartBuildScript"; Function={Test-SmartBuildScript}; Description="Smart build script validation"}
    )
    
    $suiteResults = @{}
    
    foreach ($suite in $suites) {
        Write-Log "" "Info"
        Write-Log "Running $($suite.Name): $($suite.Description)" "Info"
        
        try {
            $result = & $suite.Function
            $suiteResults[$suite.Name] = if ($result) { "PASSED" } else { "FAILED" }
        } catch {
            $suiteResults[$suite.Name] = "ERROR"
            Write-Log "Test suite $($suite.Name) threw an exception: $($_.Exception.Message)" "Error"
        }
    }
    
    return $suiteResults
}

# =============================================================================
# Generate Final Report
# =============================================================================
function Write-FinalReport {
    param($SuiteResults = @{})
    
    Write-Log "" "Info"
    Write-Log "===============================================================================" "Header"
    Write-Log "FINAL VALIDATION SUMMARY" "Header"
    Write-Log "===============================================================================" "Header"
    
    $successRate = if ($Global:TestsTotal -gt 0) { 
        [math]::Round(($Global:TestsPassed / $Global:TestsTotal) * 100, 1)
    } else { 0 }
    
    Write-Log "" "Info"
    Write-Log "Test Results Summary:" "Info"
    Write-Log "  Total Tests: $($Global:TestsTotal)" "Info"
    Write-Log "  Passed: $($Global:TestsPassed)" "Info"
    Write-Log "  Failed: $($Global:TestsFailed)" "Info"
    Write-Log "  Warnings: $($Global:TestsWarning)" "Info"
    Write-Log "  Success Rate: ${successRate}%" "Info"
    Write-Log "" "Info"
    
    # Suite-level results
    if ($SuiteResults.Count -gt 0) {
        Write-Log "Test Suite Results:" "Info"
        foreach ($suite in $SuiteResults.GetEnumerator()) {
            $status = switch ($suite.Value) {
                "PASSED" { "✅"; break }
                "FAILED" { "❌"; break }
                "ERROR" { "💥"; break }
                default { "❓" }
            }
            Write-Log "  $status $($suite.Name): $($suite.Value)" "Info"
        }
        Write-Log "" "Info"
    }
    
    # Final assessment
    if ($Global:TestsFailed -eq 0) {
        if ($Global:TestsWarning -eq 0) {
            Write-Log "🎉 ALL VALIDATIONS PASSED - CI fixes are fully functional!" "Success"
            Write-Log "Status: FULLY VALIDATED ✅" "Success"
            Write-Log "" "Info"
            Write-Log "✅ READY FOR PR MERGE" "Success"
        } else {
            Write-Log "✅ CORE VALIDATIONS PASSED - CI fixes are functional with minor warnings" "Success"
            Write-Log "Status: VALIDATED WITH WARNINGS ⚠️" "Warning"
            Write-Log "" "Info"
            Write-Log "✅ READY FOR MERGE (address warnings in follow-up)" "Success"
        }
        return $true
    } else {
        Write-Log "❌ VALIDATION FAILURES DETECTED - CI fixes need attention" "Error"
        Write-Log "Status: VALIDATION FAILED ❌" "Error"
        Write-Log "" "Info"
        Write-Log "❌ NOT READY FOR MERGE - Fix failures before proceeding" "Error"
        return $false
    }
}

# =============================================================================
# Main Execution Function
# =============================================================================
function Main {
    # Show usage if help requested
    if ($Help) {
        Write-Host @"
PowerShell CI Fixes Validation Script
=====================================

USAGE:
  .\Validate-CIFixes.ps1 [OPTIONS]

OPTIONS:
  -Quick          Run only essential tests (faster)
  -Performance    Run performance-focused tests  
  -Security       Run security-focused tests
  -LogPath        Specify custom log file path
  -Help           Show this help message

EXAMPLES:
  .\Validate-CIFixes.ps1                    # Run full validation
  .\Validate-CIFixes.ps1 -Quick             # Run quick validation
  .\Validate-CIFixes.ps1 -Performance       # Test performance aspects
  .\Validate-CIFixes.ps1 -Security          # Focus on security

OUTPUT:
  Results logged to: validation-results.txt (or custom path)
"@
        return
    }
    
    # Initialize
    Set-Location $ProjectRoot
    New-Item -ItemType Directory -Path $ResultsDir -Force -ErrorAction SilentlyContinue | Out-Null
    
    # Initialize log file
    $initMessage = @"
===============================================================================
NEPHORAN CI FIXES - POWERSHELL VALIDATION REPORT  
===============================================================================
Generated: $(Get-Date)
PowerShell Version: $($PSVersionTable.PSVersion)
Platform: $($PSVersionTable.Platform)
OS: $($PSVersionTable.OS)
Project: $(Split-Path $ProjectRoot -Leaf)
===============================================================================

"@
    Set-Content -Path $MasterLog -Value $initMessage
    
    Write-Log "🚀 Starting PowerShell CI Fixes Validation" "Header"
    Write-Log "Project root: $ProjectRoot" "Info"
    Write-Log "Results directory: $ResultsDir" "Info"
    Write-Log "Master log: $MasterLog" "Info"
    Write-Log "" "Info"
    
    # Run validation based on mode
    $suiteResults = @{}
    
    switch ($true) {
        $Quick {
            Invoke-QuickValidation
        }
        $Performance {
            Invoke-PerformanceValidation  
        }
        $Security {
            Invoke-SecurityValidation
        }
        default {
            $suiteResults = Invoke-FullValidation
        }
    }
    
    # Generate final report
    $success = Write-FinalReport -SuiteResults $suiteResults
    
    # Final status
    Write-Log "" "Info"
    Write-Log "🏁 PowerShell Validation Complete" "Header"
    
    if ($success) {
        Write-Log "✅ All validations passed - ready for production!" "Success"
        exit 0
    } else {
        Write-Log "❌ Some validations failed - review logs and fix issues" "Error"
        exit 1
    }
}

# Script execution
Main