#!/usr/bin/env pwsh
# Script for running all linters individually to isolate issues
# Usage: .\lint-batch.ps1 [path]

param(
    [string]$Path = ".",
    [switch]$FailFast = $false,
    [switch]$Parallel = $false
)

$ErrorActionPreference = "Stop"

# Available linters from .golangci.yml
$Linters = @(
    "revive", "staticcheck", "govet", "ineffassign", "errcheck", 
    "gocritic", "misspell", "unparam", "unconvert", "prealloc", "gosec"
)

$Results = @{}
$TotalStartTime = Get-Date

Write-Host "Running all linters individually on path: $Path" -ForegroundColor Cyan
Write-Host "Fail fast: $FailFast" -ForegroundColor Gray
Write-Host "Parallel: $Parallel" -ForegroundColor Gray
Write-Host ""

function Run-SingleLinter {
    param($LinterName, $TargetPath)
    
    $startTime = Get-Date
    try {
        $output = & "./scripts/lint-isolated.ps1" $LinterName $TargetPath 2>&1
        $exitCode = $LASTEXITCODE
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        
        return @{
            Linter = $LinterName
            ExitCode = $exitCode
            Duration = $duration
            Output = $output -join "`n"
            Success = ($exitCode -eq 0)
        }
    } catch {
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        return @{
            Linter = $LinterName
            ExitCode = 1
            Duration = $duration
            Output = "ERROR: $_"
            Success = $false
        }
    }
}

if ($Parallel) {
    # Run linters in parallel using PowerShell jobs
    Write-Host "Running linters in parallel..." -ForegroundColor Yellow
    
    $jobs = @()
    foreach ($linter in $Linters) {
        $job = Start-Job -ScriptBlock {
            param($LinterName, $TargetPath, $ScriptPath)
            Set-Location (Split-Path $ScriptPath)
            & "$ScriptPath/lint-isolated.ps1" $LinterName $TargetPath 2>&1
            return @{
                Linter = $LinterName
                ExitCode = $LASTEXITCODE
            }
        } -ArgumentList $linter, $Path, $PWD
        $jobs += @{ Job = $job; Linter = $linter }
    }
    
    # Wait for all jobs and collect results
    foreach ($jobInfo in $jobs) {
        $result = Receive-Job -Job $jobInfo.Job -Wait
        Remove-Job -Job $jobInfo.Job
        
        $Results[$jobInfo.Linter] = @{
            Linter = $jobInfo.Linter
            ExitCode = $result.ExitCode
            Success = ($result.ExitCode -eq 0)
            Output = $result -join "`n"
        }
    }
} else {
    # Run linters sequentially
    foreach ($linter in $Linters) {
        Write-Host "Running $linter..." -ForegroundColor Yellow
        
        $result = Run-SingleLinter -LinterName $linter -TargetPath $Path
        $Results[$linter] = $result
        
        if ($result.Success) {
            Write-Host "  ✅ $linter passed" -ForegroundColor Green
        } else {
            Write-Host "  ❌ $linter failed" -ForegroundColor Red
            if ($FailFast) {
                Write-Host "Fail fast enabled, stopping execution" -ForegroundColor Red
                break
            }
        }
    }
}

$TotalEndTime = Get-Date
$TotalDuration = ($TotalEndTime - $TotalStartTime).TotalSeconds

# Summary
Write-Host "`n" + "="*60 -ForegroundColor Cyan
Write-Host "LINTER BATCH RESULTS SUMMARY" -ForegroundColor Cyan
Write-Host "="*60 -ForegroundColor Cyan

$PassedCount = 0
$FailedCount = 0

foreach ($linter in $Linters) {
    if ($Results.ContainsKey($linter)) {
        $result = $Results[$linter]
        if ($result.Success) {
            Write-Host "✅ $($result.Linter)" -ForegroundColor Green
            $PassedCount++
        } else {
            Write-Host "❌ $($result.Linter)" -ForegroundColor Red
            $FailedCount++
        }
    } else {
        Write-Host "⚠️  $linter (not run)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "Passed: $PassedCount" -ForegroundColor Green
Write-Host "Failed: $FailedCount" -ForegroundColor Red
Write-Host "Total Duration: $($TotalDuration.ToString('F2')) seconds" -ForegroundColor Gray

# Detailed results for failed linters
if ($FailedCount -gt 0) {
    Write-Host "`nDETAILED FAILURE RESULTS:" -ForegroundColor Red
    Write-Host "-"*40 -ForegroundColor Red
    
    foreach ($linter in $Linters) {
        if ($Results.ContainsKey($linter) -and -not $Results[$linter].Success) {
            $result = $Results[$linter]
            Write-Host "`n$($result.Linter):" -ForegroundColor Red
            Write-Host $result.Output
        }
    }
}

# Exit with appropriate code
exit $FailedCount