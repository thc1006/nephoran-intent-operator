#!/usr/bin/env pwsh
<#
.SYNOPSIS
    CI Progress Tracker - Track iterative fixes and CI progress
.DESCRIPTION
    Advanced progress tracking system for CI fixes with:
    - Fix history with success/failure tracking
    - Regression detection
    - Performance metrics
    - Automated recommendations
    - Integration with GitHub Actions
#>
param(
    [Parameter(Position = 0)]
    [ValidateSet("init", "record", "status", "history", "analyze", "export")]
    [string]$Command = "status",
    
    [string]$FixName,
    [string]$Result,
    [string]$Details,
    [switch]$Regression,
    [string]$ExportPath
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$ProgressDbPath = Join-Path $ProjectRoot ".ci-progress.json"
$ReportsDir = Join-Path $ProjectRoot ".ci-reports"

function Initialize-ProgressDB {
    $initialDB = @{
        version = "1.0"
        created = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        project = Split-Path -Leaf $ProjectRoot
        branch = (git rev-parse --abbrev-ref HEAD 2>$null) ?? "unknown"
        sessions = @()
        fixes = @{}
        metrics = @{
            total_sessions = 0
            total_fixes_attempted = 0
            total_fixes_successful = 0
            success_rate = 0.0
            average_session_duration = 0
        }
    }
    
    $initialDB | ConvertTo-Json -Depth 10 | Set-Content -Path $ProgressDbPath
    Write-Host "✅ Progress database initialized: $ProgressDbPath" -ForegroundColor Green
}

function Get-ProgressDB {
    if (-not (Test-Path $ProgressDbPath)) {
        Initialize-ProgressDB
    }
    
    return Get-Content -Path $ProgressDbPath -Raw | ConvertFrom-Json
}

function Save-ProgressDB {
    param([object]$DB)
    
    # Update metrics
    $DB.metrics.total_sessions = $DB.sessions.Count
    $DB.metrics.total_fixes_attempted = ($DB.sessions | ForEach-Object { $_.fixes.Count } | Measure-Object -Sum).Sum
    
    $successfulFixes = $DB.sessions | ForEach-Object { 
        $_.fixes | Where-Object { $_.result -eq "success" }
    }
    $DB.metrics.total_fixes_successful = $successfulFixes.Count
    
    if ($DB.metrics.total_fixes_attempted -gt 0) {
        $DB.metrics.success_rate = [math]::Round($DB.metrics.total_fixes_successful / $DB.metrics.total_fixes_attempted * 100, 2)
    }
    
    # Calculate average session duration
    $completedSessions = $DB.sessions | Where-Object { $_.end_time }
    if ($completedSessions.Count -gt 0) {
        $durations = $completedSessions | ForEach-Object {
            $start = [DateTime]::Parse($_.start_time)
            $end = [DateTime]::Parse($_.end_time)
            ($end - $start).TotalMinutes
        }
        $DB.metrics.average_session_duration = [math]::Round(($durations | Measure-Object -Average).Average, 1)
    }
    
    $DB | ConvertTo-Json -Depth 10 | Set-Content -Path $ProgressDbPath
}

function Start-NewSession {
    $db = Get-ProgressDB
    
    # End any active sessions
    $activeSessions = $db.sessions | Where-Object { -not $_.end_time }
    foreach ($session in $activeSessions) {
        $session.end_time = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        $session.status = "interrupted"
    }
    
    $sessionId = [System.Guid]::NewGuid().ToString("N").Substring(0, 8)
    $newSession = @{
        id = $sessionId
        start_time = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        end_time = $null
        status = "active"
        commit_hash = (git rev-parse --short HEAD 2>$null) ?? "unknown"
        branch = (git rev-parse --abbrev-ref HEAD 2>$null) ?? "unknown"
        fixes = @()
        ci_status = @{
            build = "unknown"
            lint = "unknown"
            tests = "unknown"
            security = "unknown"
        }
        notes = @()
    }
    
    $db.sessions += $newSession
    Save-ProgressDB -DB $db
    
    Write-Host "🚀 Started new CI session: $sessionId" -ForegroundColor Green
    return $sessionId
}

function Record-Fix {
    param(
        [string]$Name,
        [string]$Result,
        [string]$Details = "",
        [bool]$IsRegression = $false
    )
    
    $db = Get-ProgressDB
    $activeSession = $db.sessions | Where-Object { $_.status -eq "active" } | Select-Object -Last 1
    
    if (-not $activeSession) {
        $sessionId = Start-NewSession
        $activeSession = $db.sessions | Where-Object { $_.id -eq $sessionId }
    }
    
    $fix = @{
        name = $Name
        result = $Result.ToLower()
        timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        details = $Details
        is_regression = $IsRegression
        duration = 0  # Could be tracked if needed
    }
    
    $activeSession.fixes += $fix
    
    # Update global fix tracking
    if (-not $db.fixes.ContainsKey($Name)) {
        $db.fixes[$Name] = @{
            name = $Name
            attempts = 0
            successes = 0
            failures = 0
            last_attempt = $null
            success_rate = 0.0
            common_errors = @()
        }
    }
    
    $fixTracker = $db.fixes[$Name]
    $fixTracker.attempts++
    $fixTracker.last_attempt = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
    
    if ($Result.ToLower() -eq "success") {
        $fixTracker.successes++
    } else {
        $fixTracker.failures++
        if ($Details -and $fixTracker.common_errors -notcontains $Details) {
            $fixTracker.common_errors += $Details
        }
    }
    
    $fixTracker.success_rate = if ($fixTracker.attempts -gt 0) {
        [math]::Round($fixTracker.successes / $fixTracker.attempts * 100, 2)
    } else { 0.0 }
    
    Save-ProgressDB -DB $db
    
    $statusColor = if ($Result.ToLower() -eq "success") { "Green" } else { "Red" }
    $regressionFlag = if ($IsRegression) { " [REGRESSION]" } else { "" }
    Write-Host "📝 Recorded fix: $Name - $Result$regressionFlag" -ForegroundColor $statusColor
}

function Update-CIStatus {
    param([hashtable]$Status)
    
    $db = Get-ProgressDB
    $activeSession = $db.sessions | Where-Object { $_.status -eq "active" } | Select-Object -Last 1
    
    if ($activeSession) {
        $activeSession.ci_status = $Status
        Save-ProgressDB -DB $db
        Write-Host "📊 Updated CI status for session $($activeSession.id)" -ForegroundColor Cyan
    }
}

function Complete-Session {
    param([string]$Status = "completed")
    
    $db = Get-ProgressDB
    $activeSession = $db.sessions | Where-Object { $_.status -eq "active" } | Select-Object -Last 1
    
    if ($activeSession) {
        $activeSession.end_time = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        $activeSession.status = $Status
        Save-ProgressDB -DB $db
        
        $start = [DateTime]::Parse($activeSession.start_time)
        $end = [DateTime]::Parse($activeSession.end_time)
        $duration = ($end - $start).TotalMinutes
        
        Write-Host "✅ Completed session $($activeSession.id) in $([math]::Round($duration, 1)) minutes" -ForegroundColor Green
    }
}

function Show-Status {
    $db = Get-ProgressDB
    
    Write-Host ""
    Write-Host "=== CI PROGRESS TRACKER STATUS ===" -ForegroundColor Cyan
    Write-Host ""
    
    # Overall metrics
    Write-Host "OVERALL METRICS:" -ForegroundColor Yellow
    Write-Host "  Total Sessions: $($db.metrics.total_sessions)"
    Write-Host "  Fixes Attempted: $($db.metrics.total_fixes_attempted)"
    Write-Host "  Fixes Successful: $($db.metrics.total_fixes_successful)"
    Write-Host "  Success Rate: $($db.metrics.success_rate)%"
    Write-Host "  Avg Session Duration: $($db.metrics.average_session_duration) minutes"
    Write-Host ""
    
    # Active session
    $activeSession = $db.sessions | Where-Object { $_.status -eq "active" } | Select-Object -Last 1
    if ($activeSession) {
        Write-Host "ACTIVE SESSION: $($activeSession.id)" -ForegroundColor Green
        Write-Host "  Started: $($activeSession.start_time)"
        Write-Host "  Branch: $($activeSession.branch)"
        Write-Host "  Commit: $($activeSession.commit_hash)"
        Write-Host "  Fixes Applied: $($activeSession.fixes.Count)"
        
        # CI Status
        $ciStatus = $activeSession.ci_status
        Write-Host "  CI Status:"
        Write-Host "    Build: $(Get-StatusIcon $ciStatus.build)"
        Write-Host "    Lint: $(Get-StatusIcon $ciStatus.lint)"  
        Write-Host "    Tests: $(Get-StatusIcon $ciStatus.tests)"
        Write-Host "    Security: $(Get-StatusIcon $ciStatus.security)"
        
        if ($activeSession.fixes.Count -gt 0) {
            Write-Host ""
            Write-Host "  Recent Fixes:" -ForegroundColor Yellow
            $activeSession.fixes | Select-Object -Last 5 | ForEach-Object {
                $statusIcon = if ($_.result -eq "success") { "✅" } else { "❌" }
                Write-Host "    $statusIcon $($_.name) - $($_.result)"
            }
        }
    } else {
        Write-Host "No active session" -ForegroundColor Yellow
    }
    
    # Recent sessions
    $recentSessions = $db.sessions | Where-Object { $_.status -ne "active" } | Select-Object -Last 3
    if ($recentSessions.Count -gt 0) {
        Write-Host ""
        Write-Host "RECENT SESSIONS:" -ForegroundColor Yellow
        $recentSessions | ForEach-Object {
            $start = [DateTime]::Parse($_.start_time)
            $duration = if ($_.end_time) {
                $end = [DateTime]::Parse($_.end_time)
                [math]::Round(($end - $start).TotalMinutes, 1)
            } else { "ongoing" }
            
            $statusIcon = switch ($_.status) {
                "completed" { "✅" }
                "failed" { "❌" }
                "interrupted" { "⚠️" }
                default { "❓" }
            }
            
            Write-Host "  $statusIcon $($_.id) - $($_.fixes.Count) fixes - $duration min - $($_.status)"
        }
    }
    
    # Top failing fixes
    $topFailingFixes = $db.fixes.GetEnumerator() | Where-Object { $_.Value.failures -gt 0 } | Sort-Object { $_.Value.failures } -Descending | Select-Object -First 3
    if ($topFailingFixes.Count -gt 0) {
        Write-Host ""
        Write-Host "TOP FAILING FIXES:" -ForegroundColor Red
        $topFailingFixes | ForEach-Object {
            Write-Host "  ❌ $($_.Key) - $($_.Value.failures) failures ($(100-$_.Value.success_rate)% fail rate)"
            if ($_.Value.common_errors.Count -gt 0) {
                Write-Host "     Common error: $($_.Value.common_errors[0])" -ForegroundColor DarkRed
            }
        }
    }
    
    Write-Host ""
}

function Get-StatusIcon {
    param([string]$Status)
    
    switch ($Status.ToLower()) {
        "pass" { return "✅" }
        "fail" { return "❌" }
        "success" { return "✅" }
        "failure" { return "❌" }
        "unknown" { return "❓" }
        default { return "❓" }
    }
}

function Show-History {
    param([int]$Limit = 10)
    
    $db = Get-ProgressDB
    
    Write-Host ""
    Write-Host "=== CI SESSION HISTORY ===" -ForegroundColor Cyan
    Write-Host ""
    
    $sessions = $db.sessions | Sort-Object { [DateTime]::Parse($_.start_time) } -Descending | Select-Object -First $Limit
    
    foreach ($session in $sessions) {
        $start = [DateTime]::Parse($session.start_time)
        $duration = if ($session.end_time) {
            $end = [DateTime]::Parse($session.end_time)
            "$([math]::Round(($end - $start).TotalMinutes, 1)) min"
        } else { "ongoing" }
        
        $statusIcon = switch ($session.status) {
            "completed" { "✅" }
            "failed" { "❌" }
            "interrupted" { "⚠️" }
            "active" { "🔄" }
            default { "❓" }
        }
        
        Write-Host "$statusIcon Session $($session.id)" -ForegroundColor Cyan
        Write-Host "   Time: $($start.ToString('yyyy-MM-dd HH:mm:ss')) ($duration)"
        Write-Host "   Branch: $($session.branch) @ $($session.commit_hash)"
        Write-Host "   Status: $($session.status)"
        Write-Host "   Fixes: $($session.fixes.Count) total"
        
        if ($session.fixes.Count -gt 0) {
            $successful = ($session.fixes | Where-Object { $_.result -eq "success" }).Count
            $failed = ($session.fixes | Where-Object { $_.result -ne "success" }).Count
            Write-Host "   Results: $successful successful, $failed failed"
            
            # Show recent fixes for this session
            $session.fixes | Select-Object -Last 3 | ForEach-Object {
                $fixIcon = if ($_.result -eq "success") { "  ✅" } else { "  ❌" }
                Write-Host "$fixIcon $($_.name)"
            }
        }
        
        Write-Host ""
    }
}

function Export-Progress {
    param([string]$ExportPath)
    
    $db = Get-ProgressDB
    
    if (-not $ExportPath) {
        $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
        $ExportPath = Join-Path $ReportsDir "ci-progress-export-$timestamp.json"
    }
    
    # Ensure directory exists
    $exportDir = Split-Path -Parent $ExportPath
    if (-not (Test-Path $exportDir)) {
        New-Item -Path $exportDir -ItemType Directory -Force | Out-Null
    }
    
    # Create export data
    $exportData = @{
        export_timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        export_version = "1.0"
        source_db = $ProgressDbPath
        data = $db
        summary = @{
            total_sessions = $db.sessions.Count
            active_sessions = ($db.sessions | Where-Object { $_.status -eq "active" }).Count
            completed_sessions = ($db.sessions | Where-Object { $_.status -eq "completed" }).Count
            failed_sessions = ($db.sessions | Where-Object { $_.status -eq "failed" }).Count
            top_fixes = $db.fixes.GetEnumerator() | Sort-Object { $_.Value.attempts } -Descending | Select-Object -First 5 | ForEach-Object { 
                @{
                    name = $_.Key
                    attempts = $_.Value.attempts
                    success_rate = $_.Value.success_rate
                }
            }
        }
    }
    
    $exportData | ConvertTo-Json -Depth 15 | Set-Content -Path $ExportPath
    Write-Host "📤 Progress data exported to: $ExportPath" -ForegroundColor Green
    
    # Also create a summary report
    $summaryPath = $ExportPath -replace '\.json$', '-summary.txt'
    $summaryReport = @"
CI Progress Tracker - Summary Report
Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')

OVERALL STATISTICS:
- Total Sessions: $($exportData.summary.total_sessions)
- Completed Sessions: $($exportData.summary.completed_sessions)
- Failed Sessions: $($exportData.summary.failed_sessions)
- Success Rate: $($db.metrics.success_rate)%
- Average Session Duration: $($db.metrics.average_session_duration) minutes

TOP FIXES BY USAGE:
$($exportData.summary.top_fixes | ForEach-Object { "- $($_.name): $($_.attempts) attempts ($($_.success_rate)% success)" } | Out-String)

RECOMMENDATIONS:
$((Get-Recommendations -DB $db) | Out-String)
"@
    
    $summaryReport | Set-Content -Path $summaryPath
    Write-Host "📋 Summary report created: $summaryPath" -ForegroundColor Green
}

function Get-Recommendations {
    param([object]$DB)
    
    $recommendations = @()
    
    # Check for frequently failing fixes
    $frequentlyFailing = $DB.fixes.GetEnumerator() | Where-Object { 
        $_.Value.attempts -ge 3 -and $_.Value.success_rate -lt 50 
    }
    
    if ($frequentlyFailing.Count -gt 0) {
        $recommendations += "🔧 Consider manual review for frequently failing fixes:"
        $frequentlyFailing | ForEach-Object {
            $recommendations += "   - $($_.Key) ($(100-$_.Value.success_rate)% failure rate)"
        }
    }
    
    # Check session success patterns
    $recentSessions = $DB.sessions | Sort-Object { [DateTime]::Parse($_.start_time) } -Descending | Select-Object -First 5
    $successfulSessions = $recentSessions | Where-Object { $_.status -eq "completed" }
    
    if ($successfulSessions.Count -lt $recentSessions.Count * 0.6) {
        $recommendations += "⚠️  Recent session success rate is low - consider process improvements"
    }
    
    # Check for regression patterns
    $regressiveFixes = $DB.sessions | ForEach-Object { $_.fixes } | Where-Object { $_.is_regression }
    if ($regressiveFixes.Count -gt 0) {
        $recommendations += "🔄 Detected $($regressiveFixes.Count) regressions - review fix quality"
    }
    
    if ($recommendations.Count -eq 0) {
        $recommendations += "✅ No major issues detected - CI process is performing well"
    }
    
    return $recommendations
}

# Command execution
try {
    # Ensure reports directory exists
    if (-not (Test-Path $ReportsDir)) {
        New-Item -Path $ReportsDir -ItemType Directory -Force | Out-Null
    }
    
    switch ($Command) {
        "init" {
            Initialize-ProgressDB
        }
        
        "record" {
            if (-not $FixName -or -not $Result) {
                Write-Host "Error: -FixName and -Result parameters are required for record command" -ForegroundColor Red
                exit 1
            }
            Record-Fix -Name $FixName -Result $Result -Details $Details -IsRegression $Regression
        }
        
        "status" {
            Show-Status
        }
        
        "history" {
            Show-History
        }
        
        "analyze" {
            $db = Get-ProgressDB
            Write-Host ""
            Write-Host "=== CI PROGRESS ANALYSIS ===" -ForegroundColor Cyan
            Write-Host ""
            
            # Show recommendations
            Write-Host "RECOMMENDATIONS:" -ForegroundColor Yellow
            $recommendations = Get-Recommendations -DB $db
            $recommendations | ForEach-Object { Write-Host $_ }
            
            Write-Host ""
            
            # Show fix analysis
            if ($db.fixes.Count -gt 0) {
                Write-Host "FIX EFFECTIVENESS ANALYSIS:" -ForegroundColor Yellow
                $db.fixes.GetEnumerator() | Sort-Object { $_.Value.success_rate } | ForEach-Object {
                    $name = $_.Key
                    $stats = $_.Value
                    $color = if ($stats.success_rate -ge 80) { "Green" } elseif ($stats.success_rate -ge 50) { "Yellow" } else { "Red" }
                    Write-Host "  $name: $($stats.success_rate)% success ($($stats.attempts) attempts)" -ForegroundColor $color
                }
            }
        }
        
        "export" {
            Export-Progress -ExportPath $ExportPath
        }
        
        default {
            Write-Host "Unknown command: $Command" -ForegroundColor Red
            Write-Host "Available commands: init, record, status, history, analyze, export" -ForegroundColor Yellow
            exit 1
        }
    }
}
catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}