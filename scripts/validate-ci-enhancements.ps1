#!/usr/bin/env pwsh
# validate-ci-enhancements.ps1 - Validate Windows CI enhancements

param(
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

function Write-Status {
    param([string]$Message, [string]$Status = "Info")
    $color = switch ($Status) {
        "Success" { "Green" }
        "Warning" { "Yellow" }
        "Error" { "Red" }
        default { "White" }
    }
    Write-Host $Message -ForegroundColor $color
}

function Test-WorkflowFile {
    param([string]$FilePath, [string]$Name)
    
    Write-Status "Testing $Name..." "Info"
    
    if (-not (Test-Path $FilePath)) {
        Write-Status "❌ File not found: $FilePath" "Error"
        return $false
    }
    
    $content = Get-Content $FilePath -Raw
    
    # Check for key enhancements
    $checks = @(
        @{pattern="timeout-minutes:\s*3[0-9]"; description="Extended timeout (30+ minutes)"},
        @{pattern="WINDOWS_TEMP_BASE"; description="Windows temp directory management"},
        @{pattern="continue-on-error:\s*true"; description="Error handling configuration"},
        @{pattern="uses:\s*actions/upload-artifact@v4"; description="Enhanced artifact upload"},
        @{pattern="shell:\s*pwsh"; description="PowerShell shell usage"}
    )
    
    $allPassed = $true
    foreach ($check in $checks) {
        if ($content -match $check.pattern) {
            Write-Status "  ✅ $($check.description)" "Success"
        } else {
            Write-Status "  ❌ Missing: $($check.description)" "Error"
            $allPassed = $false
        }
    }
    
    return $allPassed
}

function Test-ScriptFile {
    param([string]$FilePath, [string]$Name)
    
    Write-Status "Testing $Name..." "Info"
    
    if (-not (Test-Path $FilePath)) {
        Write-Status "❌ Script not found: $FilePath" "Error"
        return $false
    }
    
    # Test script syntax
    try {
        $null = Get-Command $FilePath -ErrorAction Stop
        Write-Status "  ✅ Script syntax valid" "Success"
    } catch {
        Write-Status "  ❌ Script syntax error: $_" "Error"
        return $false
    }
    
    # Check for key functions
    $content = Get-Content $FilePath -Raw
    $functions = @("Test-Prerequisites", "Setup-Environment", "Run-Tests", "Cleanup")
    
    $allFound = $true
    foreach ($func in $functions) {
        if ($content -match "function $func") {
            Write-Status "  ✅ Function found: $func" "Success"
        } else {
            Write-Status "  ❌ Missing function: $func" "Error"
            $allFound = $false
        }
    }
    
    return $allFound
}

function Test-Documentation {
    param([string]$FilePath, [string]$Name)
    
    Write-Status "Testing $Name..." "Info"
    
    if (-not (Test-Path $FilePath)) {
        Write-Status "❌ Documentation not found: $FilePath" "Error"
        return $false
    }
    
    $content = Get-Content $FilePath -Raw
    
    # Check for key sections
    $sections = @(
        "## Key Improvements",
        "## Environment Variables", 
        "## Test Strategy",
        "## Local Testing",
        "## Troubleshooting"
    )
    
    $allFound = $true
    foreach ($section in $sections) {
        if ($content -match [regex]::Escape($section)) {
            Write-Status "  ✅ Section found: $section" "Success"
        } else {
            Write-Status "  ❌ Missing section: $section" "Error"
            $allFound = $false
        }
    }
    
    return $allFound
}

function Test-ProjectStructure {
    Write-Status "Testing project structure..." "Info"
    
    $requiredFiles = @(
        @{path="go.mod"; description="Go module file"},
        @{path="cmd\conductor-loop\main.go"; description="Conductor loop main"},
        @{path="internal\loop\watcher.go"; description="Watcher implementation"},
        @{path="scripts"; description="Scripts directory"},
        @{path="docs"; description="Documentation directory"}
    )
    
    $allFound = $true
    foreach ($file in $requiredFiles) {
        if (Test-Path $file.path) {
            Write-Status "  ✅ $($file.description)" "Success"
        } else {
            Write-Status "  ❌ Missing: $($file.description)" "Error"
            $allFound = $false
        }
    }
    
    return $allFound
}

function Test-GoEnvironment {
    Write-Status "Testing Go environment..." "Info"
    
    try {
        $goVersion = go version
        Write-Status "  ✅ Go version: $goVersion" "Success"
        
        # Test basic Go commands
        go env GOOS | Out-Null
        Write-Status "  ✅ Go environment accessible" "Success"
        
        # Test module validation
        go mod verify | Out-Null
        Write-Status "  ✅ Go modules verified" "Success"
        
        return $true
    } catch {
        Write-Status "  ❌ Go environment error: $_" "Error"
        return $false
    }
}

function Main {
    Write-Status "🔍 Validating Windows CI Enhancements" "Info"
    Write-Status "Started at: $(Get-Date)" "Info"
    Write-Status "" "Info"
    
    $tests = @(
        @{
            name = "Project Structure"
            func = { Test-ProjectStructure }
        },
        @{
            name = "Go Environment"
            func = { Test-GoEnvironment }
        },
        @{
            name = "Main CI Workflow"
            func = { Test-WorkflowFile -FilePath ".github\workflows\ci.yml" -Name "Main CI" }
        },
        @{
            name = "Enhanced Windows CI Workflow"
            func = { Test-WorkflowFile -FilePath ".github\workflows\windows-ci-enhanced.yml" -Name "Enhanced Windows CI" }
        },
        @{
            name = "Cross-Platform Workflow"
            func = { Test-WorkflowFile -FilePath ".github\workflows\cross-platform.yml" -Name "Cross-Platform" }
        },
        @{
            name = "Local Test Script"
            func = { Test-ScriptFile -FilePath "scripts\test-windows-ci.ps1" -Name "Local Test Script" }
        },
        @{
            name = "Enhancement Documentation"
            func = { Test-Documentation -FilePath "docs\ci\WINDOWS-CI-ENHANCEMENTS.md" -Name "Enhancement Documentation" }
        }
    )
    
    $allPassed = $true
    $results = @()
    
    foreach ($test in $tests) {
        $testStart = Get-Date
        $success = & $test.func
        $testEnd = Get-Date
        $duration = ($testEnd - $testStart).TotalMilliseconds
        
        $results += @{
            name = $test.name
            success = $success
            duration = $duration
        }
        
        if ($success) {
            Write-Status "✅ $($test.name) passed ($([math]::Round($duration, 0))ms)" "Success"
        } else {
            Write-Status "❌ $($test.name) failed ($([math]::Round($duration, 0))ms)" "Error"
            $allPassed = $false
        }
        Write-Status "" "Info"
    }
    
    # Summary
    Write-Status "=== Validation Summary ===" "Info"
    Write-Status "Total tests: $($tests.Count)" "Info"
    Write-Status "Passed: $(($results | Where-Object {$_.success}).Count)" "Success"
    Write-Status "Failed: $(($results | Where-Object {-not $_.success}).Count)" "Error"
    
    if ($Verbose) {
        Write-Status "" "Info"
        Write-Status "Detailed Results:" "Info"
        foreach ($result in $results) {
            $status = if ($result.success) { "PASS" } else { "FAIL" }
            $duration = [math]::Round($result.duration, 0)
            Write-Status "  $status - $($result.name) (${duration}ms)" "Info"
        }
    }
    
    Write-Status "" "Info"
    if ($allPassed) {
        Write-Status "🎉 All validations passed! Windows CI enhancements are ready." "Success"
        Write-Status "" "Info"
        Write-Status "Next steps:" "Info"
        Write-Status "  1. Test locally: .\scripts\test-windows-ci.ps1" "Info"
        Write-Status "  2. Commit changes and push to trigger CI" "Info"
        Write-Status "  3. Monitor CI execution in GitHub Actions" "Info"
        exit 0
    } else {
        Write-Status "❌ Some validations failed. Please fix the issues above." "Error"
        exit 1
    }
}

# Run validation
if ($MyInvocation.InvocationName -ne '.') {
    Main
}