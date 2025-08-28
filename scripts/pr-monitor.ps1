#!/usr/bin/env pwsh
<#
.SYNOPSIS
    PR Monitor - Monitor CI status after push and provide actionable feedback
.DESCRIPTION
    Comprehensive PR monitoring with:
    - Real-time CI status tracking
    - Failure analysis and suggestions
    - Auto-retry capabilities for transient failures
    - Integration with fix engine for quick resolution
    - Slack/Teams notifications (optional)
#>
param(
    [Parameter(Position = 0)]
    [ValidateSet("watch", "status", "retry", "notify", "analyze")]
    [string]$Command = "watch",
    
    [string]$PullRequestNumber,
    [string]$Branch,
    [int]$TimeoutMinutes = 30,
    [switch]$AutoRetry,
    [switch]$Notifications,
    [string]$WebhookUrl
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$MonitorDir = Join-Path $ProjectRoot ".pr-monitor"
$LogFile = Join-Path $MonitorDir "monitor.log"

# CI status definitions matching GitHub Actions
$CIJobs = @(
    @{ name = "Detect Changes"; key = "changes"; critical = $false }
    @{ name = "Repository Hygiene"; key = "hygiene"; critical = $false }
    @{ name = "Generate CRDs"; key = "crds"; critical = $false }
    @{ name = "Unit Tests"; key = "unit-tests"; critical = $true }
    @{ name = "Lint"; key = "lint"; critical = $true }
    @{ name = "Security Scan"; key = "security"; critical = $false }
    @{ name = "Docker Build"; key = "docker-build"; critical = $true }
    @{ name = "Build Validation"; key = "build-validation"; critical = $true }
    @{ name = "All Checks Passed"; key = "success"; critical = $true }
)

function Initialize-Monitor {
    if (-not (Test-Path $MonitorDir)) {
        New-Item -Path $MonitorDir -ItemType Directory -Force | Out-Null
    }
    
    Write-MonitorLog "Monitor initialized for $(Split-Path -Leaf $ProjectRoot)" "INFO"
}

function Write-MonitorLog {
    param([string]$Message, [string]$Level = "INFO")
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logEntry = "[$timestamp] [$Level] $Message"
    
    # Console output with colors
    $color = switch ($Level) {
        "SUCCESS" { "Green" }
        "WARNING" { "Yellow" }
        "ERROR" { "Red" }
        "INFO" { "White" }
        default { "Gray" }
    }
    
    Write-Host $logEntry -ForegroundColor $color
    
    # File logging
    $logEntry | Add-Content -Path $LogFile -Force
}

function Get-CurrentBranch {
    try {
        $branch = git rev-parse --abbrev-ref HEAD 2>$null
        if ($branch -and $branch -ne "HEAD") {
            return $branch
        }
    } catch {}
    return $null
}

function Get-CurrentCommit {
    try {
        return git rev-parse HEAD 2>$null
    } catch {}
    return $null
}

function Test-GitHubCLI {
    if (-not (Get-Command "gh" -ErrorAction SilentlyContinue)) {
        Write-MonitorLog "GitHub CLI (gh) not found. Please install it to use PR monitoring." "ERROR"
        Write-Host ""
        Write-Host "Install GitHub CLI:" -ForegroundColor Cyan
        Write-Host "  Windows: winget install GitHub.CLI" -ForegroundColor Gray
        Write-Host "  Or visit: https://cli.github.com/" -ForegroundColor Gray
        return $false
    }
    
    try {
        $null = gh auth status 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-MonitorLog "GitHub CLI not authenticated. Run 'gh auth login' first." "ERROR"
            return $false
        }
    } catch {
        Write-MonitorLog "GitHub CLI authentication check failed." "ERROR"
        return $false
    }
    
    return $true
}

function Get-PRInfo {
    param([string]$Branch, [string]$PRNumber)
    
    try {
        if ($PRNumber) {
            $prData = gh pr view $PRNumber --json "number,title,state,headRefName,headRefOid" | ConvertFrom-Json
        } else {
            # Find PR for current branch
            $prData = gh pr list --head $Branch --json "number,title,state,headRefName,headRefOid" --limit 1 | ConvertFrom-Json
            if ($prData -is [array] -and $prData.Count -gt 0) {
                $prData = $prData[0]
            }
        }
        
        if (-not $prData) {
            Write-MonitorLog "No pull request found for branch: $Branch" "WARNING"
            return $null
        }
        
        return $prData
    } catch {
        Write-MonitorLog "Error getting PR info: $($_.Exception.Message)" "ERROR"
        return $null
    }
}

function Get-CIRuns {
    param([string]$Branch, [string]$CommitSha)
    
    try {
        $runs = gh run list --branch $Branch --limit 10 --json "id,status,conclusion,createdAt,headSha,workflowName,jobs_url" | ConvertFrom-Json
        
        # Filter to runs for the specific commit if provided
        if ($CommitSha) {
            $runs = $runs | Where-Object { $_.headSha -eq $CommitSha }
        }
        
        return $runs
    } catch {
        Write-MonitorLog "Error getting CI runs: $($_.Exception.Message)" "ERROR"
        return @()
    }
}

function Get-RunJobs {
    param([string]$RunId)
    
    try {
        $jobs = gh api "/repos/:owner/:repo/actions/runs/$RunId/jobs" | ConvertFrom-Json
        return $jobs.jobs
    } catch {
        Write-MonitorLog "Error getting run jobs: $($_.Exception.Message)" "WARNING"
        return @()
    }
}

function Get-JobStatus {
    param([object]$Job)
    
    $status = $Job.status
    $conclusion = $Job.conclusion
    
    if ($status -eq "completed") {
        switch ($conclusion) {
            "success" { return @{ status = "success"; icon = "✅"; color = "Green" } }
            "failure" { return @{ status = "failure"; icon = "❌"; color = "Red" } }
            "cancelled" { return @{ status = "cancelled"; icon = "⚠️"; color = "Yellow" } }
            "skipped" { return @{ status = "skipped"; icon = "⏭️"; color = "Gray" } }
            default { return @{ status = "unknown"; icon = "❓"; color = "Gray" } }
        }
    } else {
        switch ($status) {
            "in_progress" { return @{ status = "running"; icon = "🔄"; color = "Cyan" } }
            "queued" { return @{ status = "queued"; icon = "⏳"; color = "Yellow" } }
            default { return @{ status = "pending"; icon = "⏳"; color = "Yellow" } }
        }
    }
}

function Show-CIStatus {
    param([object]$PRInfo, [object[]]$Runs)
    
    Write-Host ""
    Write-Host "=== PR CI STATUS ===" -ForegroundColor Cyan
    Write-Host ""
    
    if ($PRInfo) {
        Write-Host "PR: #$($PRInfo.number) - $($PRInfo.title)" -ForegroundColor White
        Write-Host "Branch: $($PRInfo.headRefName)" -ForegroundColor Gray
        Write-Host "Commit: $($PRInfo.headRefOid.Substring(0,8))" -ForegroundColor Gray
        Write-Host ""
    }
    
    if ($Runs.Count -eq 0) {
        Write-Host "No CI runs found" -ForegroundColor Yellow
        return
    }
    
    # Show latest run
    $latestRun = $Runs | Sort-Object { [DateTime]::Parse($_.createdAt) } -Descending | Select-Object -First 1
    
    Write-Host "Latest CI Run: $($latestRun.workflowName)" -ForegroundColor White
    Write-Host "Status: $($latestRun.status)" -ForegroundColor $(
        switch ($latestRun.status) {
            "completed" { if ($latestRun.conclusion -eq "success") { "Green" } else { "Red" } }
            "in_progress" { "Cyan" }
            default { "Yellow" }
        }
    )
    
    if ($latestRun.conclusion) {
        Write-Host "Result: $($latestRun.conclusion)" -ForegroundColor $(
            switch ($latestRun.conclusion) {
                "success" { "Green" }
                "failure" { "Red" }
                default { "Yellow" }
            }
        )
    }
    
    # Get job details
    $jobs = Get-RunJobs -RunId $latestRun.id
    
    if ($jobs.Count -gt 0) {
        Write-Host ""
        Write-Host "Job Details:" -ForegroundColor Yellow
        
        foreach ($job in $jobs) {
            $jobStatus = Get-JobStatus -Job $job
            $criticalFlag = ""
            
            # Check if this is a critical job
            $criticalJob = $CIJobs | Where-Object { $job.name -like "*$($_.name)*" -or $job.name -like "*$($_.key)*" }
            if ($criticalJob -and $criticalJob.critical) {
                $criticalFlag = " [CRITICAL]"
            }
            
            Write-Host "  $($jobStatus.icon) $($job.name)$criticalFlag" -ForegroundColor $jobStatus.color
            
            if ($jobStatus.status -eq "failure") {
                Write-Host "    Duration: $([math]::Round(([DateTime]::Parse($job.completed_at) - [DateTime]::Parse($job.started_at)).TotalMinutes, 1)) min" -ForegroundColor Gray
            }
        }
    }
    
    Write-Host ""
}

function Analyze-Failures {
    param([object[]]$Jobs)
    
    $failures = $jobs | Where-Object { $_.conclusion -eq "failure" }
    
    if ($failures.Count -eq 0) {
        return @()
    }
    
    Write-MonitorLog "Analyzing $($failures.Count) failed jobs..." "INFO"
    
    $analysis = @()
    
    foreach ($failure in $failures) {
        $jobAnalysis = @{
            name = $failure.name
            suggestions = @()
            auto_fixable = $false
            commands = @()
        }
        
        # Analyze based on job name patterns
        switch -Regex ($failure.name) {
            ".*[Ll]int.*" {
                $jobAnalysis.suggestions += "Linting failures detected"
                $jobAnalysis.auto_fixable = $true
                $jobAnalysis.commands += ".\scripts\ci-mirror.ps1 fix -AutoFix"
                $jobAnalysis.commands += ".\scripts\auto-fix-engine.ps1 apply -DryRun:$false"
            }
            
            ".*[Tt]est.*" {
                $jobAnalysis.suggestions += "Unit test failures - check test logs for specific failures"
                $jobAnalysis.auto_fixable = $false
                $jobAnalysis.commands += ".\scripts\ci-mirror.ps1 verify -SkipTests:$false"
            }
            
            ".*[Bb]uild.*" {
                $jobAnalysis.suggestions += "Build compilation errors"
                $jobAnalysis.auto_fixable = $true
                $jobAnalysis.commands += "go mod tidy"
                $jobAnalysis.commands += "go build ./..."
            }
            
            ".*[Ss]ecurity.*" {
                $jobAnalysis.suggestions += "Security scan issues - review vulnerability report"
                $jobAnalysis.auto_fixable = $false
                $jobAnalysis.commands += "make scan-vulnerabilities"
            }
            
            ".*[Dd]ocker.*" {
                $jobAnalysis.suggestions += "Docker build failures - check Dockerfile and build context"
                $jobAnalysis.auto_fixable = $false
                $jobAnalysis.commands += "make docker-build"
            }
            
            default {
                $jobAnalysis.suggestions += "Unknown failure type - check job logs for details"
            }
        }
        
        $analysis += $jobAnalysis
    }
    
    return $analysis
}

function Show-FailureAnalysis {
    param([object[]]$Analysis)
    
    if ($Analysis.Count -eq 0) {
        return
    }
    
    Write-Host "=== FAILURE ANALYSIS ===" -ForegroundColor Red
    Write-Host ""
    
    $autoFixableCount = ($Analysis | Where-Object { $_.auto_fixable }).Count
    
    foreach ($failure in $Analysis) {
        $fixableIcon = if ($failure.auto_fixable) { "🔧" } else { "🔍" }
        Write-Host "$fixableIcon $($failure.name)" -ForegroundColor $(if ($failure.auto_fixable) { "Yellow" } else { "Red" })
        
        foreach ($suggestion in $failure.suggestions) {
            Write-Host "   • $suggestion" -ForegroundColor Gray
        }
        
        if ($failure.commands.Count -gt 0) {
            Write-Host "   Suggested commands:" -ForegroundColor Cyan
            foreach ($command in $failure.commands) {
                Write-Host "     $command" -ForegroundColor White
            }
        }
        
        Write-Host ""
    }
    
    if ($autoFixableCount -gt 0) {
        Write-Host "💡 $autoFixableCount job(s) may be auto-fixable. Run:" -ForegroundColor Green
        Write-Host "   .\scripts\pr-monitor.ps1 retry -AutoRetry" -ForegroundColor White
    }
}

function Send-Notification {
    param(
        [string]$Title,
        [string]$Message,
        [string]$Status,
        [string]$WebhookUrl
    )
    
    if (-not $WebhookUrl) {
        return
    }
    
    $color = switch ($Status) {
        "success" { "good" }
        "failure" { "danger" }
        "warning" { "warning" }
        default { "#439FE0" }
    }
    
    $payload = @{
        attachments = @(
            @{
                color = $color
                title = $Title
                text = $Message
                footer = "PR Monitor"
                ts = [int][double]::Parse((Get-Date -UFormat "%s"))
            }
        )
    } | ConvertTo-Json -Depth 3
    
    try {
        $headers = @{ "Content-Type" = "application/json" }
        Invoke-RestMethod -Uri $WebhookUrl -Method Post -Body $payload -Headers $headers | Out-Null
        Write-MonitorLog "Notification sent successfully" "SUCCESS"
    } catch {
        Write-MonitorLog "Failed to send notification: $($_.Exception.Message)" "WARNING"
    }
}

function Start-CIWatch {
    param(
        [string]$Branch,
        [string]$PRNumber,
        [int]$TimeoutMinutes = 30,
        [bool]$AutoRetry = $false
    )
    
    Write-MonitorLog "Starting CI watch for branch: $Branch" "INFO"
    
    if (-not $Branch) {
        $Branch = Get-CurrentBranch
        if (-not $Branch) {
            throw "Could not determine branch to monitor"
        }
    }
    
    $commit = Get-CurrentCommit
    $prInfo = Get-PRInfo -Branch $Branch -PRNumber $PRNumber
    
    if (-not $prInfo) {
        Write-MonitorLog "No PR found, monitoring CI runs for branch: $Branch" "WARNING"
    }
    
    $startTime = Get-Date
    $endTime = $startTime.AddMinutes($TimeoutMinutes)
    $lastStatus = ""
    $checkCount = 0
    
    Write-Host "🔍 Monitoring CI for branch '$Branch'..." -ForegroundColor Cyan
    Write-Host "Timeout: $TimeoutMinutes minutes" -ForegroundColor Gray
    Write-Host ""
    
    do {
        $checkCount++
        $currentTime = Get-Date
        
        # Get CI runs
        $runs = Get-CIRuns -Branch $Branch -CommitSha $commit
        
        if ($runs.Count -gt 0) {
            $latestRun = $runs | Sort-Object { [DateTime]::Parse($_.createdAt) } -Descending | Select-Object -First 1
            $currentStatus = "$($latestRun.status):$($latestRun.conclusion)"
            
            # Only show update if status changed or every 10th check
            if ($currentStatus -ne $lastStatus -or $checkCount % 10 -eq 0) {
                Clear-Host
                Show-CIStatus -PRInfo $prInfo -Runs @($latestRun)
                
                $elapsed = [math]::Round(($currentTime - $startTime).TotalMinutes, 1)
                Write-Host "Elapsed: $elapsed min | Next check in 30s | Timeout: $([math]::Round(($endTime - $currentTime).TotalMinutes, 1)) min remaining" -ForegroundColor Gray
                
                $lastStatus = $currentStatus
            }
            
            # Check if CI is complete
            if ($latestRun.status -eq "completed") {
                $jobs = Get-RunJobs -RunId $latestRun.id
                
                if ($latestRun.conclusion -eq "success") {
                    Write-MonitorLog "🎉 CI passed successfully!" "SUCCESS"
                    
                    if ($Notifications -and $WebhookUrl) {
                        Send-Notification -Title "CI Passed" -Message "All checks passed for PR #$($prInfo.number)" -Status "success" -WebhookUrl $WebhookUrl
                    }
                    
                    return $true
                } else {
                    Write-MonitorLog "❌ CI failed with conclusion: $($latestRun.conclusion)" "ERROR"
                    
                    # Analyze failures
                    $failureAnalysis = Analyze-Failures -Jobs $jobs
                    Show-FailureAnalysis -Analysis $failureAnalysis
                    
                    if ($Notifications -and $WebhookUrl) {
                        $failureMsg = "CI failed: $($failureAnalysis.Count) job(s) failed"
                        Send-Notification -Title "CI Failed" -Message $failureMsg -Status "failure" -WebhookUrl $WebhookUrl
                    }
                    
                    # Auto-retry if enabled and fixes are available
                    if ($AutoRetry) {
                        $autoFixable = $failureAnalysis | Where-Object { $_.auto_fixable }
                        if ($autoFixable.Count -gt 0) {
                            Write-Host ""
                            Write-Host "🔧 Auto-retry enabled - attempting fixes..." -ForegroundColor Yellow
                            
                            # Apply fixes
                            & (Join-Path $PSScriptRoot "auto-fix-engine.ps1") apply -MaxFixes 5 -Force
                            
                            # Re-run verification
                            $verifyResult = & (Join-Path $PSScriptRoot "ci-mirror.ps1") verify
                            
                            if ($LASTEXITCODE -eq 0) {
                                Write-Host "✅ Fixes applied successfully - push to trigger new CI run" -ForegroundColor Green
                                return $false  # Indicate retry needed
                            }
                        }
                    }
                    
                    return $false
                }
            }
        } else {
            if ($checkCount -eq 1) {
                Write-Host "⏳ Waiting for CI runs to start..." -ForegroundColor Yellow
            }
        }
        
        # Wait before next check
        Start-Sleep -Seconds 30
        
    } while ($currentTime -lt $endTime)
    
    Write-MonitorLog "⏰ Timeout reached ($TimeoutMinutes minutes)" "WARNING"
    
    if ($Notifications -and $WebhookUrl) {
        Send-Notification -Title "CI Timeout" -Message "CI monitoring timed out after $TimeoutMinutes minutes" -Status "warning" -WebhookUrl $WebhookUrl
    }
    
    return $false
}

# Command execution
try {
    Initialize-Monitor
    
    if (-not (Test-GitHubCLI)) {
        exit 1
    }
    
    switch ($Command) {
        "watch" {
            $result = Start-CIWatch -Branch $Branch -PRNumber $PullRequestNumber -TimeoutMinutes $TimeoutMinutes -AutoRetry:$AutoRetry
            exit $(if ($result) { 0 } else { 1 })
        }
        
        "status" {
            $currentBranch = Get-CurrentBranch
            $Branch = $Branch ?? $currentBranch
            
            if (-not $Branch) {
                throw "Could not determine branch"
            }
            
            $prInfo = Get-PRInfo -Branch $Branch -PRNumber $PullRequestNumber
            $runs = Get-CIRuns -Branch $Branch
            
            Show-CIStatus -PRInfo $prInfo -Runs $runs
        }
        
        "retry" {
            Write-MonitorLog "Attempting to fix CI issues and retry..." "INFO"
            
            # Run fix engine
            & (Join-Path $PSScriptRoot "auto-fix-engine.ps1") apply -MaxFixes 5 -Force:$Force
            
            # Verify fixes
            $verifyResult = & (Join-Path $PSScriptRoot "ci-mirror.ps1") verify
            
            if ($LASTEXITCODE -eq 0) {
                Write-Host ""
                Write-Host "✅ Fixes applied successfully!" -ForegroundColor Green
                Write-Host "💡 Push changes to trigger new CI run:" -ForegroundColor Cyan
                Write-Host "   git add -A && git commit -m 'fix: apply automated CI fixes' && git push" -ForegroundColor White
            } else {
                Write-Host ""
                Write-Host "❌ Fixes did not resolve all issues" -ForegroundColor Red
                Write-Host "💡 Manual review may be required" -ForegroundColor Yellow
            }
        }
        
        "notify" {
            if (-not $WebhookUrl) {
                Write-Host "Error: -WebhookUrl parameter required for notify command" -ForegroundColor Red
                exit 1
            }
            
            Send-Notification -Title "Test Notification" -Message "PR Monitor notification test" -Status "info" -WebhookUrl $WebhookUrl
        }
        
        "analyze" {
            $currentBranch = Get-CurrentBranch
            $Branch = $Branch ?? $currentBranch
            
            $runs = Get-CIRuns -Branch $Branch
            
            if ($runs.Count -gt 0) {
                $latestRun = $runs[0]
                $jobs = Get-RunJobs -RunId $latestRun.id
                $failureAnalysis = Analyze-Failures -Jobs $jobs
                
                Show-FailureAnalysis -Analysis $failureAnalysis
            } else {
                Write-Host "No CI runs found to analyze" -ForegroundColor Yellow
            }
        }
        
        default {
            Write-Host "Unknown command: $Command" -ForegroundColor Red
            Write-Host "Available commands: watch, status, retry, notify, analyze" -ForegroundColor Yellow
            exit 1
        }
    }
}
catch {
    Write-MonitorLog "Fatal error: $($_.Exception.Message)" "ERROR"
    exit 1
}