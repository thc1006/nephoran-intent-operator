#!/usr/bin/env powershell

<#
.SYNOPSIS
    Comprehensive E2E test runner for Nephoran Intent Operator

.DESCRIPTION
    This script orchestrates the execution of all end-to-end tests for the Nephoran Intent Operator.
    It handles test environment setup, service dependencies, and comprehensive test reporting.

.PARAMETER TestSuite
    Specify which test suite to run (all, basic, integration, workflow, stress)

.PARAMETER SkipServiceChecks
    Skip service dependency checks (useful for unit test environments)

.PARAMETER Verbose
    Enable verbose output for debugging

.PARAMETER OutputDir
    Directory for test results and reports

.PARAMETER Timeout
    Test timeout in minutes (default: 30)

.EXAMPLE
    .\run_e2e_tests.ps1 -TestSuite all -Verbose
    
.EXAMPLE
    .\run_e2e_tests.ps1 -TestSuite basic -SkipServiceChecks -OutputDir ".\test-results"
#>

param(
    [ValidateSet("all", "basic", "integration", "workflow", "stress", "cleanup")]
    [string]$TestSuite = "basic",
    
    [switch]$SkipServiceChecks,
    [switch]$Verbose,
    [switch]$GenerateReport,
    [switch]$CleanupOnly,
    
    [string]$OutputDir = ".\test-results\e2e",
    [int]$Timeout = 30,
    [string]$Namespace = "default",
    [string]$KubeConfig = $null
)

# Set error handling
$ErrorActionPreference = "Stop"
$global:TestResults = @()
$global:StartTime = Get-Date

# Logging functions
function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"
    
    switch($Level) {
        "ERROR" { Write-Host $logMessage -ForegroundColor Red }
        "WARN"  { Write-Host $logMessage -ForegroundColor Yellow }
        "SUCCESS" { Write-Host $logMessage -ForegroundColor Green }
        default { Write-Host $logMessage }
    }
    
    if($Verbose) {
        $logMessage | Out-File -FilePath "$OutputDir\e2e-test.log" -Append
    }
}

function Show-Banner {
    Write-Host @"
╔═══════════════════════════════════════════════════════╗
║         Nephoran Intent Operator E2E Tests           ║
║                                                       ║
║  Test Suite: $($TestSuite.ToUpper().PadRight(10))                        ║
║  Namespace:  $($Namespace.PadRight(10))                        ║
║  Timeout:    $($Timeout) minutes                          ║
╚═══════════════════════════════════════════════════════╝
"@ -ForegroundColor Cyan
}

function Test-Prerequisites {
    Write-Log "Checking prerequisites..."
    
    $prerequisites = @(
        @{ Name = "Go"; Command = "go version"; Required = $true },
        @{ Name = "kubectl"; Command = "kubectl version --client"; Required = $true },
        @{ Name = "Ginkgo"; Command = "ginkgo version"; Required = $true }
    )
    
    $missing = @()
    
    foreach($prereq in $prerequisites) {
        try {
            $null = Invoke-Expression $prereq.Command 2>$null
            Write-Log "✓ $($prereq.Name) is available" "SUCCESS"
        }
        catch {
            if($prereq.Required) {
                $missing += $prereq.Name
                Write-Log "✗ $($prereq.Name) is missing or not accessible" "ERROR"
            } else {
                Write-Log "⚠ $($prereq.Name) is not available (optional)" "WARN"
            }
        }
    }
    
    if($missing.Count -gt 0) {
        throw "Missing required prerequisites: $($missing -join ', ')"
    }
}

function Test-KubernetesConnection {
    Write-Log "Testing Kubernetes connection..."
    
    try {
        if($KubeConfig) {
            $env:KUBECONFIG = $KubeConfig
        }
        
        $clusterInfo = kubectl cluster-info 2>&1
        if($LASTEXITCODE -ne 0) {
            throw "kubectl cluster-info failed: $clusterInfo"
        }
        
        Write-Log "✓ Kubernetes cluster is accessible" "SUCCESS"
        
        # Check namespace exists
        $namespaceCheck = kubectl get namespace $Namespace 2>&1
        if($LASTEXITCODE -ne 0) {
            Write-Log "Creating namespace $Namespace..." "WARN"
            kubectl create namespace $Namespace
            if($LASTEXITCODE -ne 0) {
                throw "Failed to create namespace $Namespace"
            }
        }
        
        Write-Log "✓ Namespace $Namespace is available" "SUCCESS"
    }
    catch {
        throw "Kubernetes connection failed: $_"
    }
}

function Test-ServiceDependencies {
    if($SkipServiceChecks) {
        Write-Log "Skipping service dependency checks" "WARN"
        return
    }
    
    Write-Log "Checking service dependencies..."
    
    $services = @(
        @{ Name = "Controller"; URL = "http://localhost:8080/health"; Optional = $false },
        @{ Name = "LLM Processor"; URL = "http://localhost:8080/health"; Optional = $true },
        @{ Name = "RAG Service"; URL = "http://localhost:8001/health"; Optional = $true }
    )
    
    foreach($service in $services) {
        try {
            $response = Invoke-WebRequest -Uri $service.URL -TimeoutSec 5 -UseBasicParsing
            if($response.StatusCode -eq 200) {
                Write-Log "✓ $($service.Name) is healthy" "SUCCESS"
            } else {
                Write-Log "⚠ $($service.Name) returned status $($response.StatusCode)" "WARN"
            }
        }
        catch {
            if($service.Optional) {
                Write-Log "⚠ $($service.Name) is not available (tests will be skipped)" "WARN"
            } else {
                Write-Log "✗ $($service.Name) is not healthy: $_" "ERROR"
            }
        }
    }
}

function Initialize-TestEnvironment {
    Write-Log "Initializing test environment..."
    
    # Create output directory
    if(!(Test-Path $OutputDir)) {
        New-Item -Path $OutputDir -ItemType Directory -Force | Out-Null
        Write-Log "Created output directory: $OutputDir"
    }
    
    # Set environment variables for tests
    $env:E2E_TEST_NAMESPACE = $Namespace
    $env:E2E_TEST_TIMEOUT = ($Timeout * 60) # Convert to seconds
    $env:E2E_OUTPUT_DIR = $OutputDir
    
    Write-Log "✓ Test environment initialized" "SUCCESS"
}

function Invoke-TestSuite {
    param([string]$Suite)
    
    Write-Log "Executing test suite: $Suite"
    
    $testDir = Join-Path $PSScriptRoot "."
    $suiteResults = @{
        Name = $Suite
        StartTime = Get-Date
        Tests = @()
        Success = $false
        Duration = 0
    }
    
    try {
        Push-Location $testDir
        
        # Define test suites
        $suiteMapping = @{
            "basic" = @(
                "networkintent_lifecycle_test.go",
                "health_monitoring_test.go"
            )
            "integration" = @(
                "llm_processor_integration_test.go",
                "rag_service_integration_test.go"
            )
            "workflow" = @(
                "full_workflow_test.go",
                "error_handling_timeout_test.go"
            )
            "stress" = @(
                "concurrency_cleanup_test.go"
            )
            "cleanup" = @(
                "concurrency_cleanup_test.go"
            )
            "all" = @(
                "networkintent_lifecycle_test.go",
                "health_monitoring_test.go",
                "llm_processor_integration_test.go", 
                "rag_service_integration_test.go",
                "full_workflow_test.go",
                "error_handling_timeout_test.go",
                "concurrency_cleanup_test.go"
            )
        }
        
        $testFiles = $suiteMapping[$Suite]
        if(-not $testFiles) {
            throw "Unknown test suite: $Suite"
        }
        
        foreach($testFile in $testFiles) {
            Write-Log "Running test file: $testFile"
            
            $testResult = @{
                File = $testFile
                StartTime = Get-Date
                Success = $false
                Output = ""
                Duration = 0
            }
            
            try {
                # Run individual test file
                $ginkgoCmd = "ginkgo run --timeout=$($Timeout)m --json-report=$OutputDir\$($testFile).json --junit-report=$OutputDir\$($testFile).xml $testFile"
                
                if($Verbose) {
                    $ginkgoCmd += " -v"
                }
                
                Write-Log "Executing: $ginkgoCmd"
                $output = Invoke-Expression $ginkgoCmd 2>&1
                
                $testResult.Output = $output -join "`n"
                $testResult.Success = $LASTEXITCODE -eq 0
                $testResult.Duration = ((Get-Date) - $testResult.StartTime).TotalSeconds
                
                if($testResult.Success) {
                    Write-Log "✓ $testFile completed successfully" "SUCCESS"
                } else {
                    Write-Log "✗ $testFile failed" "ERROR"
                    if($Verbose) {
                        Write-Log "Test output: $($testResult.Output)"
                    }
                }
            }
            catch {
                $testResult.Output = $_.ToString()
                $testResult.Duration = ((Get-Date) - $testResult.StartTime).TotalSeconds
                Write-Log "✗ $testFile encountered error: $_" "ERROR"
            }
            
            $suiteResults.Tests += $testResult
        }
        
        $suiteResults.Success = ($suiteResults.Tests | Where-Object { -not $_.Success }).Count -eq 0
        $suiteResults.Duration = ((Get-Date) - $suiteResults.StartTime).TotalSeconds
        
    }
    finally {
        Pop-Location
    }
    
    return $suiteResults
}

function New-TestReport {
    param($Results)
    
    if(-not $GenerateReport) {
        return
    }
    
    Write-Log "Generating test report..."
    
    $reportPath = Join-Path $OutputDir "e2e-test-report.html"
    $jsonReportPath = Join-Path $OutputDir "e2e-test-report.json"
    
    # Generate JSON report
    $Results | ConvertTo-Json -Depth 10 | Out-File -FilePath $jsonReportPath -Encoding UTF8
    
    # Generate HTML report
    $html = @"
<!DOCTYPE html>
<html>
<head>
    <title>Nephoran E2E Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f0f8ff; padding: 20px; border-radius: 5px; }
        .summary { background: #f9f9f9; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .test-suite { margin: 20px 0; border: 1px solid #ddd; border-radius: 5px; }
        .suite-header { background: #f5f5f5; padding: 15px; font-weight: bold; }
        .test-result { padding: 10px 15px; border-bottom: 1px solid #eee; }
        .success { color: #28a745; }
        .failure { color: #dc3545; }
        .duration { color: #6c757d; font-size: 0.9em; }
        .output { background: #f8f9fa; padding: 10px; margin: 10px 0; border-radius: 3px; font-family: monospace; font-size: 0.8em; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Nephoran Intent Operator - E2E Test Report</h1>
        <p><strong>Test Suite:</strong> $($Results.Name)</p>
        <p><strong>Execution Time:</strong> $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")</p>
        <p><strong>Duration:</strong> $([math]::Round($Results.Duration, 2)) seconds</p>
    </div>
    
    <div class="summary">
        <h2>Test Summary</h2>
        <p><strong>Total Tests:</strong> $($Results.Tests.Count)</p>
        <p><strong>Passed:</strong> <span class="success">$(($Results.Tests | Where-Object { $_.Success }).Count)</span></p>
        <p><strong>Failed:</strong> <span class="failure">$(($Results.Tests | Where-Object { -not $_.Success }).Count)</span></p>
        <p><strong>Overall Status:</strong> $(if($Results.Success) { '<span class="success">PASSED</span>' } else { '<span class="failure">FAILED</span>' })</p>
    </div>
    
    <div class="test-suite">
        <div class="suite-header">Test Results</div>
"@

    foreach($test in $Results.Tests) {
        $statusClass = if($test.Success) { "success" } else { "failure" }
        $statusText = if($test.Success) { "PASSED" } else { "FAILED" }
        
        $html += @"
        <div class="test-result">
            <div><strong>$($test.File)</strong> - <span class="$statusClass">$statusText</span> <span class="duration">($([math]::Round($test.Duration, 2))s)</span></div>
"@
        
        if(-not $test.Success -and $test.Output) {
            $html += "<div class='output'>$($test.Output -replace "`n", "<br>")</div>"
        }
        
        $html += "</div>"
    }
    
    $html += @"
    </div>
</body>
</html>
"@
    
    $html | Out-File -FilePath $reportPath -Encoding UTF8
    Write-Log "✓ Test report generated: $reportPath" "SUCCESS"
}

function Invoke-Cleanup {
    Write-Log "Performing cleanup..."
    
    try {
        # Clean up test resources
        kubectl delete networkintents --all -n $Namespace --ignore-not-found=true 2>$null
        
        # Clean up test namespaces if they were created
        $testNamespaces = kubectl get namespaces -o name | Select-String "test-"
        foreach($ns in $testNamespaces) {
            $nsName = $ns.ToString().Replace("namespace/", "")
            Write-Log "Cleaning up test namespace: $nsName"
            kubectl delete namespace $nsName --ignore-not-found=true 2>$null
        }
        
        Write-Log "✓ Cleanup completed" "SUCCESS"
    }
    catch {
        Write-Log "⚠ Cleanup encountered issues: $_" "WARN"
    }
}

function Show-Summary {
    param($Results)
    
    $duration = ((Get-Date) - $global:StartTime).TotalMinutes
    
    Write-Host "`n" -NoNewline
    Write-Host "╔═══════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║                    TEST SUMMARY                      ║" -ForegroundColor Cyan  
    Write-Host "╠═══════════════════════════════════════════════════════╣" -ForegroundColor Cyan
    Write-Host "║ Suite: $($Results.Name.ToUpper().PadRight(10))                                ║" -ForegroundColor Cyan
    Write-Host "║ Total Tests: $($Results.Tests.Count.ToString().PadLeft(2))                                ║" -ForegroundColor Cyan
    Write-Host "║ Passed: $((($Results.Tests | Where-Object { $_.Success }).Count).ToString().PadLeft(2))                                  ║" -ForegroundColor Cyan
    Write-Host "║ Failed: $((($Results.Tests | Where-Object { -not $_.Success }).Count).ToString().PadLeft(2))                                  ║" -ForegroundColor Cyan
    Write-Host "║ Duration: $([math]::Round($duration, 1).ToString().PadLeft(4)) minutes                          ║" -ForegroundColor Cyan
    Write-Host "║ Status: $(if($Results.Success) { "PASSED".PadRight(6) } else { "FAILED".PadRight(6) })                                ║" -ForegroundColor Cyan
    Write-Host "╚═══════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    
    if(-not $Results.Success) {
        Write-Host "`nFailed Tests:" -ForegroundColor Red
        $Results.Tests | Where-Object { -not $_.Success } | ForEach-Object {
            Write-Host "  - $($_.File)" -ForegroundColor Red
        }
    }
}

# Main execution flow
try {
    Show-Banner
    
    if($CleanupOnly) {
        Invoke-Cleanup
        exit 0
    }
    
    Write-Log "Starting E2E test execution..."
    
    # Prerequisites and setup
    Test-Prerequisites
    Test-KubernetesConnection
    Test-ServiceDependencies
    Initialize-TestEnvironment
    
    # Execute tests
    $results = Invoke-TestSuite -Suite $TestSuite
    $global:TestResults += $results
    
    # Generate reports
    New-TestReport -Results $results
    
    # Show summary
    Show-Summary -Results $results
    
    # Cleanup
    if($TestSuite -eq "cleanup" -or $TestSuite -eq "all") {
        Invoke-Cleanup
    }
    
    # Exit with appropriate code
    if($results.Success) {
        Write-Log "✓ All tests completed successfully" "SUCCESS"
        exit 0
    } else {
        Write-Log "✗ Some tests failed" "ERROR"
        exit 1
    }
}
catch {
    Write-Log "Fatal error during test execution: $_" "ERROR"
    
    if($Verbose) {
        Write-Log "Stack trace: $($_.ScriptStackTrace)"
    }
    
    exit 1
}
finally {
    # Always attempt cleanup on exit
    if($global:TestResults.Count -gt 0) {
        Write-Log "Test execution completed in $([math]::Round(((Get-Date) - $global:StartTime).TotalMinutes, 1)) minutes"
    }
}