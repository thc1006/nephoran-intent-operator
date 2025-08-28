#!/usr/bin/env pwsh
<#
.SYNOPSIS
    CI Bulletproof - Ultimate CI verification and fix system
.DESCRIPTION
    Master orchestration script that provides bulletproof CI verification:
    - Pre-push verification that mirrors CI exactly
    - Automated fix application with rollback safety
    - Iterative fix loops with regression detection  
    - PR monitoring with intelligent failure analysis
    - Comprehensive progress tracking and reporting
    
    This script ensures you NEVER have CI failures again.
.PARAMETER Mode
    Operation mode: verify, fix, monitor, status, reset
.PARAMETER AutoFix
    Automatically apply all available fixes
.PARAMETER MaxIterations
    Maximum fix iterations (default: 3)
.PARAMETER TimeoutMinutes
    CI monitoring timeout (default: 30)
.PARAMETER Force
    Force operations even if risky
.PARAMETER Interactive
    Enable interactive mode for fix selection
.EXAMPLE
    .\scripts\ci-bulletproof.ps1 verify
    Run complete pre-push verification
.EXAMPLE  
    .\scripts\ci-bulletproof.ps1 fix -AutoFix -MaxIterations 5
    Apply fixes automatically with up to 5 iterations
.EXAMPLE
    .\scripts\ci-bulletproof.ps1 monitor -TimeoutMinutes 45
    Monitor CI status after push for 45 minutes
#>
param(
    [Parameter(Position = 0)]
    [ValidateSet("verify", "fix", "monitor", "status", "reset", "help")]
    [string]$Mode = "verify",
    
    [switch]$AutoFix,
    [int]$MaxIterations = 3,
    [int]$TimeoutMinutes = 30,
    [switch]$Force,
    [switch]$Interactive,
    [switch]$SkipTests,
    [switch]$Verbose,
    [string]$WebhookUrl
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BulletproofDir = Join-Path $ProjectRoot ".ci-bulletproof"
$SessionFile = Join-Path $BulletproofDir "session.json"

# Script paths
$CIMirrorScript = Join-Path $PSScriptRoot "ci-mirror.ps1"
$ProgressTrackerScript = Join-Path $PSScriptRoot "ci-progress-tracker.ps1"  
$AutoFixEngineScript = Join-Path $PSScriptRoot "auto-fix-engine.ps1"
$PRMonitorScript = Join-Path $PSScriptRoot "pr-monitor.ps1"

function Initialize-BulletproofSession {
    if (-not (Test-Path $BulletproofDir)) {
        New-Item -Path $BulletproofDir -ItemType Directory -Force | Out-Null
    }
    
    $session = @{
        version = "1.0"
        started = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        mode = $Mode
        branch = (git rev-parse --abbrev-ref HEAD 2>$null) ?? "unknown"
        commit = (git rev-parse --short HEAD 2>$null) ?? "unknown"
        parameters = @{
            auto_fix = $AutoFix.IsPresent
            max_iterations = $MaxIterations
            timeout_minutes = $TimeoutMinutes
            force = $Force.IsPresent
            interactive = $Interactive.IsPresent
            skip_tests = $SkipTests.IsPresent
        }
        phases = @{
            verification = @{ status = "pending"; started = $null; completed = $null; result = $null }
            fix_application = @{ status = "pending"; started = $null; completed = $null; iterations = 0 }
            monitoring = @{ status = "pending"; started = $null; completed = $null; result = $null }
        }
        metrics = @{
            total_duration = 0
            fixes_applied = 0
            ci_passes = 0
            ci_failures = 0
        }
        final_status = "in_progress"
    }
    
    $session | ConvertTo-Json -Depth 10 | Set-Content -Path $SessionFile
    Write-BulletproofLog "Bulletproof session initialized: $($session.started)" "SUCCESS"
    
    return $session
}

function Get-BulletproofSession {
    if (Test-Path $SessionFile) {
        return Get-Content -Path $SessionFile -Raw | ConvertFrom-Json
    }
    return $null
}

function Update-BulletproofSession {
    param([object]$Session)
    $Session | ConvertTo-Json -Depth 10 | Set-Content -Path $SessionFile
}

function Write-BulletproofLog {
    param([string]$Message, [string]$Level = "INFO", [switch]$NoNewline)
    
    $timestamp = Get-Date -Format "HH:mm:ss"
    $color = switch ($Level) {
        "SUCCESS" { "Green" }
        "WARNING" { "Yellow" }
        "ERROR" { "Red" }
        "INFO" { "Cyan" }
        "DEBUG" { "Gray" }
        default { "White" }
    }
    
    $prefix = "🛡️  BULLETPROOF"
    if ($NoNewline) {
        Write-Host "[$timestamp] [$prefix] $Message" -ForegroundColor $color -NoNewline
    } else {
        Write-Host "[$timestamp] [$prefix] $Message" -ForegroundColor $color
    }
}

function Show-Header {
    param([string]$Title, [string]$Subtitle = "")
    
    Clear-Host
    Write-Host ""
    Write-Host "🛡️ " -NoNewline -ForegroundColor Cyan
    Write-Host "CI BULLETPROOF SYSTEM" -ForegroundColor White -BackgroundColor DarkBlue
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor DarkCyan
    Write-Host "$Title" -ForegroundColor Yellow
    if ($Subtitle) {
        Write-Host "$Subtitle" -ForegroundColor Gray
    }
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor DarkCyan
    Write-Host ""
}

function Show-Status {
    $session = Get-BulletproofSession
    
    Show-Header -Title "SYSTEM STATUS" -Subtitle "Current session and system state"
    
    if ($session) {
        $duration = ([DateTime]::Parse($session.started) - (Get-Date)).TotalMinutes * -1
        
        Write-Host "SESSION INFO:" -ForegroundColor Yellow
        Write-Host "  Started: $($session.started)"
        Write-Host "  Duration: $([math]::Round($duration, 1)) minutes"
        Write-Host "  Mode: $($session.mode)"
        Write-Host "  Branch: $($session.branch) @ $($session.commit)"
        Write-Host "  Status: $($session.final_status)" -ForegroundColor $(
            switch ($session.final_status) {
                "completed" { "Green" }
                "failed" { "Red" }
                "in_progress" { "Cyan" }
                default { "Gray" }
            }
        )
        Write-Host ""
        
        Write-Host "PHASE STATUS:" -ForegroundColor Yellow
        foreach ($phaseKey in $session.phases.PSObject.Properties.Name) {
            $phase = $session.phases.$phaseKey
            $statusIcon = switch ($phase.status) {
                "completed" { "✅" }
                "failed" { "❌" }
                "in_progress" { "🔄" }
                "skipped" { "⏭️" }
                default { "⏳" }
            }
            
            $phaseName = ($phaseKey -replace "_", " ").ToUpper()
            Write-Host "  $statusIcon $phaseName" -ForegroundColor White
            
            if ($phase.result) {
                Write-Host "      Result: $($phase.result)" -ForegroundColor Gray
            }
            if ($phaseKey -eq "fix_application" -and $phase.iterations -gt 0) {
                Write-Host "      Iterations: $($phase.iterations)" -ForegroundColor Gray
            }
        }
        
        Write-Host ""
        Write-Host "METRICS:" -ForegroundColor Yellow
        Write-Host "  Fixes Applied: $($session.metrics.fixes_applied)"
        Write-Host "  CI Passes: $($session.metrics.ci_passes)"
        Write-Host "  CI Failures: $($session.metrics.ci_failures)"
    } else {
        Write-Host "No active session found" -ForegroundColor Yellow
    }
    
    Write-Host ""
    
    # Show progress tracker status
    Write-Host "PROGRESS TRACKER:" -ForegroundColor Yellow
    & $ProgressTrackerScript status
    
    Write-Host ""
    
    # Show fix engine status  
    Write-Host "FIX ENGINE:" -ForegroundColor Yellow
    & $AutoFixEngineScript analyze
}

function Invoke-VerificationPhase {
    param([object]$Session)
    
    Write-BulletproofLog "Starting verification phase..." "INFO"
    
    $Session.phases.verification.status = "in_progress"
    $Session.phases.verification.started = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
    Update-BulletproofSession -Session $Session
    
    try {
        $verifyArgs = @("verify")
        if ($SkipTests) { $verifyArgs += "-SkipTests" }
        if ($Verbose) { $verifyArgs += "-Verbose" }
        
        Write-BulletproofLog "Running CI mirror verification..." "INFO"
        & $CIMirrorScript @verifyArgs
        
        $verifySuccess = $LASTEXITCODE -eq 0
        
        $Session.phases.verification.status = if ($verifySuccess) { "completed" } else { "failed" }
        $Session.phases.verification.completed = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        $Session.phases.verification.result = if ($verifySuccess) { "passed" } else { "failed" }
        
        if ($verifySuccess) {
            Write-BulletproofLog "✅ Verification phase PASSED - ready to push!" "SUCCESS"
            $Session.metrics.ci_passes++
        } else {
            Write-BulletproofLog "❌ Verification phase FAILED - fixes needed" "ERROR"
            $Session.metrics.ci_failures++
        }
        
        Update-BulletproofSession -Session $Session
        return $verifySuccess
        
    } catch {
        $Session.phases.verification.status = "failed"
        $Session.phases.verification.completed = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        $Session.phases.verification.result = "error: $($_.Exception.Message)"
        Update-BulletproofSession -Session $Session
        
        Write-BulletproofLog "Verification phase error: $($_.Exception.Message)" "ERROR"
        throw
    }
}

function Invoke-FixPhase {
    param([object]$Session)
    
    Write-BulletproofLog "Starting fix application phase..." "INFO"
    
    $Session.phases.fix_application.status = "in_progress" 
    $Session.phases.fix_application.started = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
    Update-BulletproofSession -Session $Session
    
    $iteration = 0
    $maxIterations = $MaxIterations
    $allFixesSuccessful = $false
    
    do {
        $iteration++
        Write-BulletproofLog "Fix iteration $iteration/$maxIterations" "INFO"
        
        $Session.phases.fix_application.iterations = $iteration
        Update-BulletproofSession -Session $Session
        
        # Apply fixes
        try {
            $fixArgs = @("apply")
            if (-not $Interactive) { $fixArgs += "-AutoFix" }
            if ($Force) { $fixArgs += "-Force" }
            if ($Verbose) { $fixArgs += "-Verbose" }
            $fixArgs += "-MaxFixes", "10"
            
            Write-BulletproofLog "Applying automated fixes..." "INFO"
            & $AutoFixEngineScript @fixArgs
            
            $fixesApplied = $LASTEXITCODE -eq 0
            
            if ($fixesApplied) {
                $Session.metrics.fixes_applied++
                
                # Verify fixes worked
                Write-BulletproofLog "Verifying applied fixes..." "INFO"
                $verifySuccess = Invoke-VerificationPhase -Session $Session
                
                if ($verifySuccess) {
                    $allFixesSuccessful = $true
                    Write-BulletproofLog "✅ Fix iteration $iteration successful!" "SUCCESS"
                    break
                } else {
                    Write-BulletproofLog "⚠️ Fix iteration $iteration didn't resolve all issues" "WARNING"
                }
            } else {
                Write-BulletproofLog "⚠️ No fixes were applied in iteration $iteration" "WARNING"
            }
            
        } catch {
            Write-BulletproofLog "Fix iteration $iteration error: $($_.Exception.Message)" "ERROR"
        }
        
        # Brief pause between iterations
        if ($iteration -lt $maxIterations) {
            Write-BulletproofLog "Waiting 5 seconds before next iteration..." "INFO"
            Start-Sleep -Seconds 5
        }
        
    } while ($iteration -lt $maxIterations -and -not $allFixesSuccessful)
    
    $Session.phases.fix_application.status = if ($allFixesSuccessful) { "completed" } else { "failed" }
    $Session.phases.fix_application.completed = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
    Update-BulletproofSession -Session $Session
    
    if ($allFixesSuccessful) {
        Write-BulletproofLog "✅ Fix phase completed successfully after $iteration iterations" "SUCCESS"
    } else {
        Write-BulletproofLog "❌ Fix phase failed after $iteration iterations" "ERROR"
    }
    
    return $allFixesSuccessful
}

function Invoke-MonitoringPhase {
    param([object]$Session)
    
    Write-BulletproofLog "Starting monitoring phase..." "INFO"
    
    $Session.phases.monitoring.status = "in_progress"
    $Session.phases.monitoring.started = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
    Update-BulletproofSession -Session $Session
    
    try {
        $monitorArgs = @("watch", "-TimeoutMinutes", $TimeoutMinutes)
        if ($AutoFix) { $monitorArgs += "-AutoRetry" }
        if ($WebhookUrl) { 
            $monitorArgs += "-Notifications"
            $monitorArgs += "-WebhookUrl", $WebhookUrl
        }
        
        Write-BulletproofLog "Starting CI monitoring (timeout: $TimeoutMinutes minutes)..." "INFO"
        & $PRMonitorScript @monitorArgs
        
        $monitorSuccess = $LASTEXITCODE -eq 0
        
        $Session.phases.monitoring.status = if ($monitorSuccess) { "completed" } else { "failed" }
        $Session.phases.monitoring.completed = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        $Session.phases.monitoring.result = if ($monitorSuccess) { "ci_passed" } else { "ci_failed" }
        
        if ($monitorSuccess) {
            Write-BulletproofLog "🎉 Monitoring phase PASSED - CI successful!" "SUCCESS"
            $Session.metrics.ci_passes++
        } else {
            Write-BulletproofLog "❌ Monitoring phase detected CI failure" "ERROR"  
            $Session.metrics.ci_failures++
        }
        
        Update-BulletproofSession -Session $Session
        return $monitorSuccess
        
    } catch {
        $Session.phases.monitoring.status = "failed"
        $Session.phases.monitoring.completed = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ" 
        $Session.phases.monitoring.result = "error: $($_.Exception.Message)"
        Update-BulletproofSession -Session $Session
        
        Write-BulletproofLog "Monitoring phase error: $($_.Exception.Message)" "ERROR"
        throw
    }
}

function Show-CompletionSummary {
    param([object]$Session)
    
    Show-Header -Title "SESSION COMPLETE" -Subtitle "Final results and recommendations"
    
    $startTime = [DateTime]::Parse($Session.started)
    $endTime = Get-Date
    $totalDuration = ($endTime - $startTime).TotalMinutes
    
    $Session.metrics.total_duration = [math]::Round($totalDuration, 1)
    
    Write-Host "EXECUTION SUMMARY:" -ForegroundColor Yellow
    Write-Host "  Total Duration: $($Session.metrics.total_duration) minutes"
    Write-Host "  Fixes Applied: $($Session.metrics.fixes_applied)"
    Write-Host "  CI Passes: $($Session.metrics.ci_passes)"
    Write-Host "  CI Failures: $($Session.metrics.ci_failures)"
    Write-Host ""
    
    # Determine final status
    $allPhasesCompleted = $Session.phases.verification.status -eq "completed" -and 
                         ($Session.phases.fix_application.status -in @("completed", "skipped")) -and
                         ($Session.phases.monitoring.status -in @("completed", "skipped"))
    
    $Session.final_status = if ($allPhasesCompleted) { "completed" } else { "failed" }
    
    Write-Host "FINAL STATUS: " -NoNewline -ForegroundColor Yellow
    Write-Host $Session.final_status.ToUpper() -ForegroundColor $(
        if ($Session.final_status -eq "completed") { "Green" } else { "Red" }
    )
    
    Write-Host ""
    
    # Show recommendations
    Write-Host "RECOMMENDATIONS:" -ForegroundColor Cyan
    
    if ($Session.final_status -eq "completed") {
        Write-Host "  ✅ All systems operational - ready for development" -ForegroundColor Green
        Write-Host "  📊 Monitor progress: .\scripts\ci-progress-tracker.ps1 status" -ForegroundColor White
    } else {
        Write-Host "  🔧 Issues remain - consider manual review" -ForegroundColor Red
        Write-Host "  📋 Check logs: .\scripts\ci-bulletproof.ps1 status" -ForegroundColor White
        Write-Host "  🔄 Retry: .\scripts\ci-bulletproof.ps1 fix -Force" -ForegroundColor White
    }
    
    Write-Host ""
    Write-Host "ARTIFACTS:" -ForegroundColor Yellow
    Write-Host "  Session Log: $SessionFile" -ForegroundColor Gray
    Write-Host "  Progress DB: .ci-progress.json" -ForegroundColor Gray
    Write-Host "  Fix Engine State: .fix-engine/state.json" -ForegroundColor Gray
    
    Update-BulletproofSession -Session $Session
    
    Write-Host ""
}

function Show-Help {
    Show-Header -Title "HELP & USAGE" -Subtitle "Complete guide to bulletproof CI system"
    
    Write-Host "OVERVIEW:" -ForegroundColor Yellow
    Write-Host "  The CI Bulletproof system ensures you never have CI failures by providing:"
    Write-Host "  • Pre-push verification that mirrors CI exactly"
    Write-Host "  • Automated fix application with safety checks"
    Write-Host "  • Intelligent PR monitoring with failure analysis"
    Write-Host "  • Comprehensive progress tracking and rollback"
    Write-Host ""
    
    Write-Host "USAGE:" -ForegroundColor Yellow
    Write-Host "  .\scripts\ci-bulletproof.ps1 <mode> [options]" -ForegroundColor White
    Write-Host ""
    
    Write-Host "MODES:" -ForegroundColor Yellow
    Write-Host "  verify    Run complete pre-push verification" -ForegroundColor White
    Write-Host "  fix       Apply automated fixes iteratively" -ForegroundColor White  
    Write-Host "  monitor   Monitor CI status after push" -ForegroundColor White
    Write-Host "  status    Show current system status" -ForegroundColor White
    Write-Host "  reset     Reset all system state" -ForegroundColor White
    Write-Host "  help      Show this help message" -ForegroundColor White
    Write-Host ""
    
    Write-Host "OPTIONS:" -ForegroundColor Yellow
    Write-Host "  -AutoFix           Apply fixes automatically without prompts" -ForegroundColor White
    Write-Host "  -MaxIterations N   Maximum fix iterations (default: 3)" -ForegroundColor White
    Write-Host "  -TimeoutMinutes N  CI monitoring timeout (default: 30)" -ForegroundColor White
    Write-Host "  -Force             Force operations even if risky" -ForegroundColor White
    Write-Host "  -Interactive       Enable interactive mode for fix selection" -ForegroundColor White
    Write-Host "  -SkipTests         Skip tests for faster verification" -ForegroundColor White
    Write-Host "  -Verbose           Enable verbose output" -ForegroundColor White
    Write-Host "  -WebhookUrl URL    Slack/Teams webhook for notifications" -ForegroundColor White
    Write-Host ""
    
    Write-Host "WORKFLOW EXAMPLES:" -ForegroundColor Green
    Write-Host ""
    Write-Host "1. PRE-PUSH VERIFICATION:" -ForegroundColor Cyan
    Write-Host "   .\scripts\ci-bulletproof.ps1 verify" -ForegroundColor White
    Write-Host "   → Runs complete CI verification locally" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "2. AUTO-FIX ALL ISSUES:" -ForegroundColor Cyan  
    Write-Host "   .\scripts\ci-bulletproof.ps1 fix -AutoFix -MaxIterations 5" -ForegroundColor White
    Write-Host "   → Applies up to 5 rounds of automated fixes" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "3. INTERACTIVE FIX MODE:" -ForegroundColor Cyan
    Write-Host "   .\scripts\ci-bulletproof.ps1 fix -Interactive" -ForegroundColor White
    Write-Host "   → Lets you choose which fixes to apply" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "4. MONITOR CI AFTER PUSH:" -ForegroundColor Cyan
    Write-Host "   .\scripts\ci-bulletproof.ps1 monitor -TimeoutMinutes 45" -ForegroundColor White
    Write-Host "   → Monitors CI for 45 minutes with intelligent failure analysis" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "5. COMPLETE WORKFLOW:" -ForegroundColor Cyan
    Write-Host "   .\scripts\ci-bulletproof.ps1 verify && " -NoNewline -ForegroundColor White
    Write-Host "git push && " -NoNewline -ForegroundColor White
    Write-Host ".\scripts\ci-bulletproof.ps1 monitor" -ForegroundColor White
    Write-Host "   → Verify → Push → Monitor (bulletproof workflow)" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "TROUBLESHOOTING:" -ForegroundColor Red
    Write-Host "  • If verification fails: Run fix mode to resolve issues"
    Write-Host "  • If fixes don't work: Check status for detailed analysis"
    Write-Host "  • If system is stuck: Use reset to clear all state"
    Write-Host "  • For rollback: Use fix engine rollback functionality"
    Write-Host ""
}

function Reset-System {
    Write-BulletproofLog "Resetting CI Bulletproof system..." "WARNING"
    
    $confirm = Read-Host "This will reset all system state. Continue? [y/N]"
    if ($confirm.ToLower() -ne "y") {
        Write-BulletproofLog "Reset cancelled" "INFO"
        return
    }
    
    # Reset session
    if (Test-Path $SessionFile) {
        Remove-Item $SessionFile -Force
        Write-BulletproofLog "Session state reset" "SUCCESS"
    }
    
    # Reset progress tracker
    & $ProgressTrackerScript init
    
    # Reset fix engine  
    & $AutoFixEngineScript reset
    
    Write-BulletproofLog "🎯 System reset complete - ready for fresh start!" "SUCCESS"
}

# Main execution
try {
    Set-Location $ProjectRoot
    
    switch ($Mode) {
        "help" {
            Show-Help
            exit 0
        }
        
        "status" {
            Show-Status
            exit 0
        }
        
        "reset" {
            Reset-System
            exit 0
        }
        
        "verify" {
            $session = Initialize-BulletproofSession
            
            Show-Header -Title "VERIFICATION MODE" -Subtitle "Pre-push CI verification"
            
            $verifySuccess = Invoke-VerificationPhase -Session $session
            
            Show-CompletionSummary -Session $session
            
            if ($verifySuccess) {
                Write-BulletproofLog "🎉 Verification PASSED - safe to push!" "SUCCESS"
                exit 0
            } else {
                Write-BulletproofLog "❌ Verification FAILED - run fix mode" "ERROR"
                exit 1
            }
        }
        
        "fix" {
            $session = Initialize-BulletproofSession
            
            Show-Header -Title "FIX MODE" -Subtitle "Automated fix application with verification"
            
            # Always run verification first to establish baseline
            Write-BulletproofLog "Running initial verification..." "INFO"
            $initialVerifySuccess = Invoke-VerificationPhase -Session $session
            
            if ($initialVerifySuccess) {
                Write-BulletproofLog "Initial verification passed - no fixes needed!" "SUCCESS"
            } else {
                $fixSuccess = Invoke-FixPhase -Session $session
                
                if (-not $fixSuccess) {
                    Write-BulletproofLog "Some issues may require manual attention" "WARNING"
                }
            }
            
            Show-CompletionSummary -Session $session
            
            # Final verification check
            $finalSuccess = $session.phases.verification.result -eq "passed"
            
            if ($finalSuccess) {
                Write-BulletproofLog "🎉 All issues resolved - ready to push!" "SUCCESS"
                exit 0
            } else {
                Write-BulletproofLog "❌ Issues remain - check status for details" "ERROR"
                exit 1
            }
        }
        
        "monitor" {
            $session = Initialize-BulletproofSession
            
            Show-Header -Title "MONITORING MODE" -Subtitle "CI status monitoring with intelligent analysis"
            
            # Skip other phases, go straight to monitoring
            $session.phases.verification.status = "skipped"
            $session.phases.fix_application.status = "skipped"
            
            $monitorSuccess = Invoke-MonitoringPhase -Session $session
            
            Show-CompletionSummary -Session $session
            
            if ($monitorSuccess) {
                Write-BulletproofLog "🎉 CI monitoring PASSED!" "SUCCESS"
                exit 0
            } else {
                Write-BulletproofLog "❌ CI monitoring detected failures" "ERROR"
                exit 1
            }
        }
        
        default {
            Write-BulletproofLog "Unknown mode: $Mode" "ERROR"
            Show-Help
            exit 1
        }
    }
    
} catch {
    Write-BulletproofLog "Fatal error: $($_.Exception.Message)" "ERROR"
    
    if ($Verbose) {
        Write-BulletproofLog $_.ScriptStackTrace "DEBUG"
    }
    
    $session = Get-BulletproofSession
    if ($session) {
        $session.final_status = "failed"
        Update-BulletproofSession -Session $session
    }
    
    exit 1
}