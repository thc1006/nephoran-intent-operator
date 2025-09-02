#!/usr/bin/env pwsh
# =====================================================================================
# Run CI Locally - Complete Reproduction of GitHub Actions
# =====================================================================================  
# This script orchestrates all CI jobs locally to exactly match GitHub Actions
# Supports both individual jobs and full pipeline execution
# =====================================================================================

param(
    [ValidateSet("all", "deps", "build", "test", "lint", "security")]
    [string]$Job = "all",
    
    [switch]$FastMode,
    [switch]$SkipSecurity,
    [switch]$WithCoverage,
    [switch]$WithBenchmarks,
    [switch]$DetailedLint,
    [switch]$Verbose,
    [switch]$StopOnFailure,
    [string]$Config = ".golangci.yml"
)

$ErrorActionPreference = "Stop"

Write-Host "=== Nephoran CI Local Reproduction ===" -ForegroundColor Green
Write-Host "Job: $Job" -ForegroundColor Cyan
Write-Host "Fast Mode: $FastMode" -ForegroundColor Cyan
Write-Host "Skip Security: $SkipSecurity" -ForegroundColor Cyan
Write-Host "Stop on Failure: $StopOnFailure" -ForegroundColor Cyan
Write-Host ""

# Ensure we're in the right directory
if (!(Test-Path "go.mod")) {
    Write-Host "❌ Must run from repository root (go.mod not found)" -ForegroundColor Red
    exit 1
}

# Create results directory
if (!(Test-Path "ci-results")) {
    New-Item -ItemType Directory -Path "ci-results" -Force | Out-Null
}

# Track job results
$jobResults = @{}
$overallStartTime = Get-Date

# Function to run a job and track results
function Invoke-CIJob {
    param(
        [string]$JobName,
        [string]$ScriptPath,
        [scriptblock]$ScriptArgs
    )
    
    Write-Host ""
    Write-Host "════════════════════════════════════════════════════════════════════" -ForegroundColor Blue
    Write-Host "🚀 Starting Job: $JobName" -ForegroundColor Blue  
    Write-Host "════════════════════════════════════════════════════════════════════" -ForegroundColor Blue
    Write-Host ""
    
    $jobStart = Get-Date
    $logFile = "ci-results\$JobName-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
    
    try {
        # Execute job with logging
        & $ScriptPath @ScriptArgs 2>&1 | Tee-Object -FilePath $logFile
        $exitCode = $LASTEXITCODE
        
        $jobDuration = (Get-Date) - $jobStart
        
        if ($exitCode -eq 0) {
            $jobResults[$JobName] = @{
                Status = "SUCCESS"
                Duration = $jobDuration
                LogFile = $logFile
                ExitCode = 0
            }
            Write-Host ""
            Write-Host "✅ Job '$JobName' completed successfully in $([math]::Round($jobDuration.TotalMinutes, 1)) minutes" -ForegroundColor Green
        } else {
            $jobResults[$JobName] = @{
                Status = "FAILURE"  
                Duration = $jobDuration
                LogFile = $logFile
                ExitCode = $exitCode
            }
            Write-Host ""
            Write-Host "❌ Job '$JobName' failed with exit code $exitCode after $([math]::Round($jobDuration.TotalMinutes, 1)) minutes" -ForegroundColor Red
            
            if ($StopOnFailure) {
                Write-Host "🛑 Stopping pipeline due to failure (StopOnFailure=true)" -ForegroundColor Red
                return $false
            }
        }
    }
    catch {
        $jobDuration = (Get-Date) - $jobStart
        $jobResults[$JobName] = @{
            Status = "ERROR"
            Duration = $jobDuration
            LogFile = $logFile
            ExitCode = -1
            Error = $_.Exception.Message
        }
        Write-Host ""  
        Write-Host "💥 Job '$JobName' crashed: $_" -ForegroundColor Red
        
        if ($StopOnFailure) {
            Write-Host "🛑 Stopping pipeline due to error (StopOnFailure=true)" -ForegroundColor Red
            return $false
        }
    }
    
    return $true
}

# Job definitions matching CI workflow order
$jobs = @{
    "deps" = @{
        Name = "Dependency Security"
        Script = "scripts\ci-jobs\dependency-security.ps1"
        Args = @{
            FastMode = $FastMode
            SkipSecurity = $SkipSecurity
            Verbose = $Verbose
        }
    }
    "build" = @{
        Name = "Build & Code Quality"  
        Script = "scripts\ci-jobs\build.ps1"
        Args = @{
            Verbose = $Verbose
        }
    }
    "test" = @{
        Name = "Comprehensive Testing"
        Script = "scripts\ci-jobs\test.ps1" 
        Args = @{
            FastMode = $FastMode
            WithRace = !$FastMode
            WithCoverage = $WithCoverage
            WithBenchmarks = $WithBenchmarks
        }
    }
    "lint" = @{
        Name = "Advanced Linting"
        Script = "scripts\ci-jobs\lint.ps1"
        Args = @{
            Config = $Config
            Verbose = $Verbose  
            DetailedOutput = $DetailedLint
        }
    }
}

# Execute requested jobs
$success = $true

if ($Job -eq "all") {
    # Full CI pipeline (matches workflow dependency order)
    $jobOrder = @("deps", "build", "test", "lint")
    
    Write-Host "🔥 Running full CI pipeline..." -ForegroundColor Yellow
    Write-Host "Jobs: $($jobOrder -join ' → ')" -ForegroundColor Gray
    
    foreach ($jobKey in $jobOrder) {
        $jobDef = $jobs[$jobKey]
        $result = Invoke-CIJob -JobName $jobDef.Name -ScriptPath $jobDef.Script -ScriptArgs $jobDef.Args
        if (!$result) {
            $success = $false
            break
        }
    }
}
elseif ($jobs.ContainsKey($Job)) {
    # Single job execution
    $jobDef = $jobs[$Job]
    Write-Host "🎯 Running single job: $($jobDef.Name)" -ForegroundColor Yellow
    $success = Invoke-CIJob -JobName $jobDef.Name -ScriptPath $jobDef.Script -ScriptArgs $jobDef.Args
}
else {
    Write-Host "❌ Unknown job: $Job" -ForegroundColor Red
    Write-Host "Available jobs: $($jobs.Keys -join ', '), all" -ForegroundColor Yellow
    exit 1
}

# Final results summary
$totalDuration = (Get-Date) - $overallStartTime

Write-Host ""
Write-Host "════════════════════════════════════════════════════════════════════" -ForegroundColor Blue
Write-Host "🏁 CI Pipeline Results" -ForegroundColor Blue
Write-Host "════════════════════════════════════════════════════════════════════" -ForegroundColor Blue
Write-Host ""
Write-Host "Total Duration: $([math]::Round($totalDuration.TotalMinutes, 1)) minutes" -ForegroundColor Cyan
Write-Host ""

# Job status table
$successCount = 0
$failureCount = 0
$errorCount = 0

foreach ($jobName in $jobResults.Keys) {
    $result = $jobResults[$jobName]
    $duration = [math]::Round($result.Duration.TotalMinutes, 1)
    
    switch ($result.Status) {
        "SUCCESS" {
            Write-Host "✅ $jobName - $($result.Status) ($duration min)" -ForegroundColor Green
            $successCount++
        }
        "FAILURE" {
            Write-Host "❌ $jobName - $($result.Status) ($duration min) - Exit: $($result.ExitCode)" -ForegroundColor Red
            Write-Host "   Log: $($result.LogFile)" -ForegroundColor Gray
            $failureCount++
        }
        "ERROR" {
            Write-Host "💥 $jobName - $($result.Status) ($duration min) - $($result.Error)" -ForegroundColor Red
            Write-Host "   Log: $($result.LogFile)" -ForegroundColor Gray
            $errorCount++
        }
    }
}

Write-Host ""
Write-Host "Summary: $successCount succeeded, $failureCount failed, $errorCount errored" -ForegroundColor Cyan

# Artifacts summary
Write-Host ""
Write-Host "📊 Generated Artifacts:" -ForegroundColor Yellow
$artifactDirs = @("test-results", "lint-reports", "ci-results", "bin")
foreach ($dir in $artifactDirs) {
    if (Test-Path $dir) {
        $files = Get-ChildItem $dir -File -Recurse
        $totalSize = ($files | Measure-Object Length -Sum).Sum / 1KB
        Write-Host "  $dir/: $($files.Count) files ($([math]::Round($totalSize, 1)) KB)" -ForegroundColor Gray
    }
}

# Cache information
if (Test-Path ".cache-keys.json") {
    Write-Host ""
    Write-Host "📦 Cache Information:" -ForegroundColor Yellow
    $cacheKeys = Get-Content ".cache-keys.json" | ConvertFrom-Json
    foreach ($key in $cacheKeys.PSObject.Properties) {
        Write-Host "  $($key.Name): $($key.Value)" -ForegroundColor Gray
    }
}

Write-Host ""
if ($success -and ($successCount -eq $jobResults.Count)) {
    Write-Host "🎉 All CI jobs completed successfully!" -ForegroundColor Green
    Write-Host "🚀 Ready for deployment or merge" -ForegroundColor Cyan
    
    # Suggest next steps
    Write-Host ""
    Write-Host "💡 Next steps:" -ForegroundColor Yellow
    Write-Host "  • Review artifacts in test-results/ and lint-reports/" -ForegroundColor Gray
    Write-Host "  • Run 'scripts\install-act.ps1' to test with GitHub Actions locally" -ForegroundColor Gray
    Write-Host "  • Push changes to trigger actual CI" -ForegroundColor Gray
    
    exit 0
} else {
    Write-Host "💀 CI pipeline failed - check logs above" -ForegroundColor Red
    Write-Host ""
    Write-Host "🔧 Troubleshooting:" -ForegroundColor Yellow
    Write-Host "  • Check individual job logs in ci-results/" -ForegroundColor Gray
    Write-Host "  • Run failed jobs individually: scripts\run-ci-locally.ps1 -Job <jobname>" -ForegroundColor Gray
    Write-Host "  • Use -Verbose for detailed output" -ForegroundColor Gray
    
    exit 1
}