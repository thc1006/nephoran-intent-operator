#!/usr/bin/env pwsh
# Script to verify that linting fixes don't break functionality
# Usage: .\verify-lint-fixes.ps1 [options]

param(
    [string]$TestPattern = "./...",
    [switch]$RunBuild = $true,
    [switch]$RunTests = $true,
    [switch]$RunLint = $true,
    [switch]$Verbose = $false,
    [int]$Timeout = 600,  # 10 minutes default
    [switch]$FailFast = $false
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Colors for output
$Colors = @{
    Success = "Green"
    Error = "Red"
    Warning = "Yellow"
    Info = "Cyan"
    Debug = "Gray"
}

function Write-ColorOutput {
    param([string]$Message, [string]$Color = "White")
    Write-Host $Message -ForegroundColor $Colors.$Color
}

function Test-CommandExists {
    param([string]$Command)
    return $null -ne (Get-Command $Command -ErrorAction SilentlyContinue)
}

# Verification results
$Results = @{
    Build = @{ Success = $false; Duration = 0; Output = "" }
    Tests = @{ Success = $false; Duration = 0; Output = ""; TestCount = 0 }
    Lint = @{ Success = $false; Duration = 0; Output = ""; IssueCount = 0 }
}

Write-ColorOutput "🔍 LINT FIX VERIFICATION SUITE" "Info"
Write-ColorOutput "================================" "Info"
Write-ColorOutput "Test Pattern: $TestPattern" "Debug"
Write-ColorOutput "Timeout: $Timeout seconds" "Debug"
Write-ColorOutput "Fail Fast: $FailFast" "Debug"
Write-ColorOutput ""

# Step 1: Build Verification
if ($RunBuild) {
    Write-ColorOutput "📦 Step 1: Build Verification" "Info"
    Write-ColorOutput "-" * 30 "Debug"
    
    $buildStartTime = Get-Date
    
    try {
        Write-ColorOutput "Building Go modules..." "Debug"
        $buildOutput = go build -v $TestPattern 2>&1
        $buildExitCode = $LASTEXITCODE
        
        $buildEndTime = Get-Date
        $buildDuration = ($buildEndTime - $buildStartTime).TotalSeconds
        
        $Results.Build.Duration = $buildDuration
        $Results.Build.Output = $buildOutput -join "`n"
        
        if ($buildExitCode -eq 0) {
            Write-ColorOutput "  ✅ Build successful ($($buildDuration.ToString('F2'))s)" "Success"
            $Results.Build.Success = $true
        } else {
            Write-ColorOutput "  ❌ Build failed ($($buildDuration.ToString('F2'))s)" "Error"
            $Results.Build.Success = $false
            
            if ($Verbose -or $FailFast) {
                Write-ColorOutput "Build output:" "Error"
                $Results.Build.Output | ForEach-Object { Write-ColorOutput "  $_" "Error" }
            }
            
            if ($FailFast) {
                Write-ColorOutput "Fail fast enabled, stopping verification" "Error"
                exit 1
            }
        }
    } catch {
        Write-ColorOutput "  ❌ Build error: $_" "Error"
        $Results.Build.Success = $false
        if ($FailFast) { exit 1 }
    }
    
    Write-ColorOutput ""
}

# Step 2: Test Verification
if ($RunTests) {
    Write-ColorOutput "🧪 Step 2: Test Verification" "Info"
    Write-ColorOutput "-" * 30 "Debug"
    
    $testStartTime = Get-Date
    
    try {
        Write-ColorOutput "Running Go tests..." "Debug"
        
        # Use go test with json output for better parsing
        $testArgs = @("test", "-v", "-timeout", "${Timeout}s", "-json", $TestPattern)
        if ($Verbose) {
            Write-ColorOutput "Command: go $($testArgs -join ' ')" "Debug"
        }
        
        $testOutput = & go @testArgs 2>&1
        $testExitCode = $LASTEXITCODE
        
        $testEndTime = Get-Date
        $testDuration = ($testEndTime - $testStartTime).TotalSeconds
        
        $Results.Tests.Duration = $testDuration
        $Results.Tests.Output = $testOutput -join "`n"
        
        # Parse test results
        $testLines = $testOutput | Where-Object { $_ -match '^{' } | ForEach-Object { $_ | ConvertFrom-Json -ErrorAction SilentlyContinue }
        $passedTests = @($testLines | Where-Object { $_.Action -eq "pass" -and $_.Test })
        $failedTests = @($testLines | Where-Object { $_.Action -eq "fail" -and $_.Test })
        $skippedTests = @($testLines | Where-Object { $_.Action -eq "skip" -and $_.Test })
        
        $Results.Tests.TestCount = $passedTests.Count + $failedTests.Count + $skippedTests.Count
        
        if ($testExitCode -eq 0 -and $failedTests.Count -eq 0) {
            Write-ColorOutput "  ✅ All tests passed ($($testDuration.ToString('F2'))s)" "Success"
            Write-ColorOutput "    Passed: $($passedTests.Count), Skipped: $($skippedTests.Count)" "Debug"
            $Results.Tests.Success = $true
        } else {
            Write-ColorOutput "  ❌ Tests failed ($($testDuration.ToString('F2'))s)" "Error"
            Write-ColorOutput "    Passed: $($passedTests.Count), Failed: $($failedTests.Count), Skipped: $($skippedTests.Count)" "Error"
            $Results.Tests.Success = $false
            
            if ($Verbose -or $FailFast) {
                if ($failedTests.Count -gt 0) {
                    Write-ColorOutput "Failed tests:" "Error"
                    $failedTests | ForEach-Object { 
                        Write-ColorOutput "  - $($_.Package)::$($_.Test)" "Error"
                    }
                }
            }
            
            if ($FailFast) {
                Write-ColorOutput "Fail fast enabled, stopping verification" "Error"
                exit 1
            }
        }
    } catch {
        Write-ColorOutput "  ❌ Test error: $_" "Error"
        $Results.Tests.Success = $false
        if ($FailFast) { exit 1 }
    }
    
    Write-ColorOutput ""
}

# Step 3: Lint Verification
if ($RunLint) {
    Write-ColorOutput "🔍 Step 3: Lint Verification" "Info"
    Write-ColorOutput "-" * 30 "Debug"
    
    if (-not (Test-CommandExists "golangci-lint")) {
        Write-ColorOutput "  ⚠️ golangci-lint not found, skipping lint verification" "Warning"
    } else {
        $lintStartTime = Get-Date
        
        try {
            Write-ColorOutput "Running golangci-lint..." "Debug"
            
            $lintArgs = @("run", "--timeout", "${Timeout}s", $TestPattern)
            if ($Verbose) {
                Write-ColorOutput "Command: golangci-lint $($lintArgs -join ' ')" "Debug"
            }
            
            $lintOutput = & golangci-lint @lintArgs 2>&1
            $lintExitCode = $LASTEXITCODE
            
            $lintEndTime = Get-Date
            $lintDuration = ($lintEndTime - $lintStartTime).TotalSeconds
            
            $Results.Lint.Duration = $lintDuration
            $Results.Lint.Output = $lintOutput -join "`n"
            
            # Count issues (rough estimate)
            $issueLines = @($lintOutput | Where-Object { $_ -match ':(line \d+|col \d+)' })
            $Results.Lint.IssueCount = $issueLines.Count
            
            if ($lintExitCode -eq 0) {
                Write-ColorOutput "  ✅ No lint issues found ($($lintDuration.ToString('F2'))s)" "Success"
                $Results.Lint.Success = $true
            } else {
                Write-ColorOutput "  ❌ Lint issues found ($($lintDuration.ToString('F2'))s)" "Error"
                Write-ColorOutput "    Approximate issue count: $($Results.Lint.IssueCount)" "Error"
                $Results.Lint.Success = $false
                
                if ($Verbose -or $FailFast) {
                    Write-ColorOutput "Lint output (first 20 lines):" "Error"
                    $Results.Lint.Output -split "`n" | Select-Object -First 20 | ForEach-Object { 
                        Write-ColorOutput "  $_" "Error" 
                    }
                }
                
                if ($FailFast) {
                    Write-ColorOutput "Fail fast enabled, stopping verification" "Error"
                    exit 1
                }
            }
        } catch {
            Write-ColorOutput "  ❌ Lint error: $_" "Error"
            $Results.Lint.Success = $false
            if ($FailFast) { exit 1 }
        }
    }
    
    Write-ColorOutput ""
}

# Final Summary
Write-ColorOutput "📊 VERIFICATION SUMMARY" "Info"
Write-ColorOutput "======================" "Info"

$allSuccess = $true
$totalDuration = 0

if ($RunBuild) {
    $status = if ($Results.Build.Success) { "✅ PASS" } else { "❌ FAIL"; $allSuccess = $false }
    Write-ColorOutput "Build:    $status ($($Results.Build.Duration.ToString('F2'))s)" $(if ($Results.Build.Success) { "Success" } else { "Error" })
    $totalDuration += $Results.Build.Duration
}

if ($RunTests) {
    $status = if ($Results.Tests.Success) { "✅ PASS" } else { "❌ FAIL"; $allSuccess = $false }
    Write-ColorOutput "Tests:    $status ($($Results.Tests.Duration.ToString('F2'))s, $($Results.Tests.TestCount) tests)" $(if ($Results.Tests.Success) { "Success" } else { "Error" })
    $totalDuration += $Results.Tests.Duration
}

if ($RunLint) {
    $status = if ($Results.Lint.Success) { "✅ PASS" } else { "❌ FAIL"; $allSuccess = $false }
    Write-ColorOutput "Lint:     $status ($($Results.Lint.Duration.ToString('F2'))s, $($Results.Lint.IssueCount) issues)" $(if ($Results.Lint.Success) { "Success" } else { "Error" })
    $totalDuration += $Results.Lint.Duration
}

Write-ColorOutput ""
Write-ColorOutput "Total Duration: $($totalDuration.ToString('F2'))s" "Debug"
Write-ColorOutput "Overall Status: $(if ($allSuccess) { '✅ ALL CHECKS PASSED' } else { '❌ SOME CHECKS FAILED' })" $(if ($allSuccess) { "Success" } else { "Error" })

# Generate JSON report if verbose
if ($Verbose) {
    $report = @{
        timestamp = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ssZ")
        success = $allSuccess
        totalDuration = $totalDuration
        results = $Results
        configuration = @{
            testPattern = $TestPattern
            runBuild = $RunBuild
            runTests = $RunTests  
            runLint = $RunLint
            timeout = $Timeout
            failFast = $FailFast
        }
    }
    
    Write-ColorOutput "`nDetailed JSON Report:" "Info"
    $report | ConvertTo-Json -Depth 5
}

exit $(if ($allSuccess) { 0 } else { 1 })