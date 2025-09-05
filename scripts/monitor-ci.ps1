# Ultra-Fast CI Monitoring Script for PR 177
# Continuous monitoring with 30-minute timeout and 30-second intervals

param(
    [int]$PRNumber = 177,
    [int]$TimeoutMinutes = 30,
    [int]$CheckInterval = 30
)

$MaxChecks = ($TimeoutMinutes * 60) / $CheckInterval
$CheckCount = 0

Write-Host "🚀 ULTRA-FAST CI MONITORING STARTED FOR PR #$PRNumber" -ForegroundColor Cyan
Write-Host "⏱️  Timeout: $TimeoutMinutes minutes" -ForegroundColor Yellow
Write-Host "🔄 Check interval: $CheckInterval seconds" -ForegroundColor Yellow
Write-Host "📊 Max checks: $MaxChecks" -ForegroundColor Yellow
Write-Host "==================================" -ForegroundColor Gray

function Log-WithTimestamp {
    param([string]$Message, [string]$Color = "White")
    $timestamp = Get-Date -Format "HH:mm:ss"
    Write-Host "[$timestamp] $Message" -ForegroundColor $Color
}

function Get-CIStatus {
    try {
        $result = gh pr checks $PRNumber --json name,state,link,workflow,completedAt | ConvertFrom-Json
        return $result
    }
    catch {
        Log-WithTimestamp "❌ Error getting CI status: $_" "Red"
        return @()
    }
}

function Format-Status {
    param([string]$State, [string]$CompletedAt)
    
    switch ($State) {
        "SUCCESS" { return "✅ SUCCESS" }
        "FAILURE" { return "❌ FAILURE" }
        "CANCELLED" { return "⚠️  CANCELLED" }
        "SKIPPED" { return "⏭️  SKIPPED" }
        "NEUTRAL" { return "➖ NEUTRAL" }
        "PENDING" { 
            if ([string]::IsNullOrEmpty($CompletedAt)) {
                return "🔄 IN PROGRESS"
            } else {
                return "⏳ PENDING"
            }
        }
        default { return "❓ $State" }
    }
}

function Check-CIStatus {
    $checks = Get-CIStatus
    $totalChecks = $checks.Count
    
    Log-WithTimestamp "📋 CI STATUS CHECK #$($CheckCount + 1)" "Cyan"
    Write-Host "────────────────────────────────────────────────" -ForegroundColor Gray
    
    if ($totalChecks -eq 0) {
        Log-WithTimestamp "⚠️  No CI checks found yet..." "Yellow"
        return "waiting"
    }
    
    $completedChecks = 0
    $successChecks = 0
    $failedChecks = 0
    $inProgressChecks = 0
    $queuedChecks = 0
    
    # Process each check
    foreach ($check in $checks) {
        $formattedStatus = Format-Status $check.state $check.completedAt
        $name = $check.name.PadRight(40)
        $workflow = $check.workflow.PadRight(20)
        Write-Host "$name [$workflow] $formattedStatus"
        
        switch ($check.state) {
            "SUCCESS" {
                $completedChecks++
                $successChecks++
            }
            { $_ -in @("FAILURE", "CANCELLED") } {
                $completedChecks++
                $failedChecks++
            }
            "SKIPPED" {
                $completedChecks++
            }
            "PENDING" {
                if ([string]::IsNullOrEmpty($check.completedAt)) {
                    $inProgressChecks++
                } else {
                    $queuedChecks++
                }
            }
        }
    }
    
    Write-Host "────────────────────────────────────────────────" -ForegroundColor Gray
    Log-WithTimestamp "📊 SUMMARY: $successChecks✅ $failedChecks❌ $inProgressChecks🔄 $queuedChecks⏳ ($completedChecks/$totalChecks complete)" "White"
    
    # Check for failures
    if ($failedChecks -gt 0) {
        Log-WithTimestamp "🚨 ALERT: $failedChecks job(s) FAILED! Immediate action required!" "Red"
        foreach ($check in $checks) {
            if ($check.state -in @("FAILURE", "CANCELLED")) {
                Write-Host "❌ FAILED: $($check.name) [$($check.workflow)] - $($check.link)" -ForegroundColor Red
            }
        }
        return "failed"
    }
    
    # Check if all completed successfully
    if ($completedChecks -eq $totalChecks -and $successChecks -eq $totalChecks) {
        Log-WithTimestamp "🎉 SUCCESS: ALL CI JOBS PASSED! 🎉" "Green"
        return "success"
    }
    
    # Still in progress
    return "in_progress"
}

# Main monitoring loop
Log-WithTimestamp "🔍 Starting continuous CI monitoring..." "Cyan"

while ($CheckCount -lt $MaxChecks) {
    $CheckCount++
    
    $status = Check-CIStatus
    
    switch ($status) {
        "success" {
            Log-WithTimestamp "🏆 MONITORING COMPLETE: ALL CI JOBS SUCCESSFUL! 🏆" "Green"
            exit 0
        }
        "failed" {
            Log-WithTimestamp "💥 FAILURES DETECTED - CONTINUING MONITORING FOR FIXES..." "Red"
        }
    }
    
    # Calculate remaining time
    $remainingChecks = $MaxChecks - $CheckCount
    $remainingMinutes = [math]::Floor(($remainingChecks * $CheckInterval) / 60)
    
    if ($remainingChecks -gt 0) {
        Log-WithTimestamp "⏰ Next check in ${CheckInterval}s (${remainingMinutes}m remaining, check $CheckCount/$MaxChecks)" "Yellow"
        Start-Sleep $CheckInterval
    }
}

Log-WithTimestamp "⏰ TIMEOUT REACHED: Monitoring stopped after $TimeoutMinutes minutes" "Yellow"
Log-WithTimestamp "📋 Final status check..." "Cyan"
Check-CIStatus | Out-Null

exit 1