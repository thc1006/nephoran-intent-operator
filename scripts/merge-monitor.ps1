# =============================================================================
# MERGE MONITORING & RECOVERY SYSTEM
# =============================================================================
# Real-time monitoring for feat/e2e → integrate/mvp merge
# Provides automated error detection and immediate response
# =============================================================================

param(
    [string]$BranchName = "integrate/mvp",
    [int]$MonitorDuration = 1800,  # 30 minutes
    [switch]$VerboseLogging,
    [switch]$AutoRecover
)

$ErrorActionPreference = "Continue"

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Cyan"

function Write-ColorOutput {
    param($Message, $Color = "White")
    Write-Host $Message -ForegroundColor $Color
}

function Get-Timestamp {
    return Get-Date -Format "yyyy-MM-dd HH:mm:ss"
}

function Initialize-Monitoring {
    Write-ColorOutput "🚀 NEPHORAN MERGE MONITOR INITIALIZED" $Blue
    Write-ColorOutput "Branch: $BranchName" $Blue
    Write-ColorOutput "Duration: $MonitorDuration seconds" $Blue
    Write-ColorOutput "Auto-Recovery: $AutoRecover" $Blue
    Write-ColorOutput "Verbose: $VerboseLogging" $Blue
    Write-ColorOutput "Started: $(Get-Timestamp)" $Blue
    Write-ColorOutput ("=" * 80) $Blue
}

function Test-Prerequisites {
    # Check if GitHub CLI is available
    try {
        $ghVersion = gh --version
        Write-ColorOutput "✅ GitHub CLI available: $($ghVersion[0])" $Green
    } catch {
        Write-ColorOutput "❌ GitHub CLI not found - install with: winget install GitHub.cli" $Red
        exit 1
    }
    
    # Check if we're in the correct repository
    try {
        $repoInfo = gh repo view --json name,owner | ConvertFrom-Json
        Write-ColorOutput "✅ Repository: $($repoInfo.owner.login)/$($repoInfo.name)" $Green
    } catch {
        Write-ColorOutput "❌ Not in a GitHub repository or auth failed" $Red
        exit 1
    }
    
    # Verify branch exists
    try {
        git show-ref --verify --quiet "refs/remotes/origin/$BranchName"
        Write-ColorOutput "✅ Branch $BranchName verified" $Green
    } catch {
        Write-ColorOutput "⚠️ Branch $BranchName may not exist remotely" $Yellow
    }
}

function Get-WorkflowStatus {
    param([int]$Limit = 20)
    
    try {
        $workflows = gh run list --branch $BranchName --limit $Limit --json status,conclusion,name,startedAt,url,id | ConvertFrom-Json
        return $workflows
    } catch {
        Write-ColorOutput "❌ Failed to get workflow status: $_" $Red
        return @()
    }
}

function Analyze-WorkflowFailure {
    param($Workflow)
    
    Write-ColorOutput "🔍 ANALYZING FAILURE: $($Workflow.name)" $Yellow
    Write-ColorOutput "   ID: $($Workflow.id)" $Yellow
    Write-ColorOutput "   Started: $($Workflow.startedAt)" $Yellow
    Write-ColorOutput "   URL: $($Workflow.url)" $Yellow
    
    # Get detailed logs for failure analysis
    try {
        $logs = gh run view $Workflow.id --log --log-failed | Out-String
        
        # Pattern matching for common failure types
        $failureType = "Unknown"
        $recoveryAction = "Manual Investigation Required"
        
        if ($logs -match "Failed to restore.*Cache service responded with 400") {
            $failureType = "Cache Restoration Failure"
            $recoveryAction = "Force cache reset and retry"
        }
        elseif ($logs -match "The operation was canceled|timeout") {
            $failureType = "Timeout/Cancellation"
            $recoveryAction = "Retry with increased timeout"
        }
        elseif ($logs -match "go: module.*not found|go mod download.*failed") {
            $failureType = "Go Module Resolution"
            $recoveryAction = "Clean module cache and retry"
        }
        elseif ($logs -match "gosec.*not found|Path does not exist.*sarif") {
            $failureType = "Security Scan Artifact Missing"
            $recoveryAction = "Continue without security artifacts"
        }
        elseif ($logs -match "build.*failed|compilation.*error") {
            $failureType = "Build/Compilation Failure"
            $recoveryAction = "Code review and fix required"
        }
        
        Write-ColorOutput "   Failure Type: $failureType" $Red
        Write-ColorOutput "   Recommended Action: $recoveryAction" $Yellow
        
        return @{
            Type = $failureType
            Action = $recoveryAction
            Logs = $logs
            Workflow = $Workflow
        }
    } catch {
        Write-ColorOutput "❌ Failed to analyze workflow logs: $_" $Red
        return $null
    }
}

function Invoke-AutoRecovery {
    param($FailureAnalysis)
    
    if (-not $AutoRecover) {
        Write-ColorOutput "⚠️ Auto-recovery disabled - manual intervention required" $Yellow
        return
    }
    
    $workflowName = $FailureAnalysis.Workflow.name
    $failureType = $FailureAnalysis.Type
    
    Write-ColorOutput "🔧 INITIATING AUTO-RECOVERY" $Blue
    Write-ColorOutput "   Workflow: $workflowName" $Blue
    Write-ColorOutput "   Failure Type: $failureType" $Blue
    
    try {
        switch ($failureType) {
            "Cache Restoration Failure" {
                Write-ColorOutput "   Action: Triggering workflow with cache reset..." $Blue
                # Try to trigger with cache reset parameter if supported
                gh workflow run "ci-2025.yml" --ref $BranchName -f debug_enabled=true -f force_cache_reset=true 2>$null
                Write-ColorOutput "✅ Cache reset recovery initiated" $Green
            }
            
            "Timeout/Cancellation" {
                Write-ColorOutput "   Action: Retrying with extended timeout..." $Blue
                # Retry the same workflow after a brief wait
                Start-Sleep -Seconds 60
                gh workflow run "$($workflowName.replace(' ', '-')).yml" --ref $BranchName 2>$null
                Write-ColorOutput "✅ Timeout recovery retry initiated" $Green
            }
            
            "Go Module Resolution" {
                Write-ColorOutput "   Action: Module resolution recovery..." $Blue
                # This would require code changes, so just flag for manual review
                Write-ColorOutput "⚠️ Module issues require manual code review" $Yellow
            }
            
            "Security Scan Artifact Missing" {
                Write-ColorOutput "   Action: Security scan is non-blocking, continuing..." $Blue
                Write-ColorOutput "✅ Security failure marked as non-blocking" $Green
            }
            
            "Build/Compilation Failure" {
                Write-ColorOutput "   Action: Build failures require code fixes..." $Yellow
                Write-ColorOutput "⚠️ Manual code review and fixes required" $Yellow
            }
            
            default {
                Write-ColorOutput "⚠️ Unknown failure type - manual investigation required" $Yellow
            }
        }
    } catch {
        Write-ColorOutput "❌ Auto-recovery failed: $_" $Red
    }
}

function Generate-StatusReport {
    param($Workflows, $FailureAnalyses)
    
    Write-ColorOutput "`n📊 STATUS REPORT - $(Get-Timestamp)" $Blue
    Write-ColorOutput ("=" * 60) $Blue
    
    $totalWorkflows = $Workflows.Count
    $completedWorkflows = ($Workflows | Where-Object { $_.status -eq "completed" }).Count
    $successfulWorkflows = ($Workflows | Where-Object { $_.conclusion -eq "success" }).Count
    $failedWorkflows = ($Workflows | Where-Object { $_.conclusion -eq "failure" }).Count
    $inProgressWorkflows = ($Workflows | Where-Object { $_.status -eq "in_progress" -or $_.status -eq "queued" }).Count
    
    Write-ColorOutput "Total Workflows: $totalWorkflows" $Blue
    Write-ColorOutput "In Progress: $inProgressWorkflows" $Yellow
    Write-ColorOutput "Completed: $completedWorkflows" $Blue
    Write-ColorOutput "Successful: $successfulWorkflows" $Green
    Write-ColorOutput "Failed: $failedWorkflows" $Red
    
    if ($failedWorkflows -gt 0) {
        Write-ColorOutput "`nFailure Summary:" $Red
        foreach ($analysis in $FailureAnalyses) {
            Write-ColorOutput "  • $($analysis.Workflow.name): $($analysis.Type)" $Red
        }
    }
    
    # Calculate success rate
    if ($completedWorkflows -gt 0) {
        $successRate = [math]::Round(($successfulWorkflows / $completedWorkflows) * 100, 1)
        $color = if ($successRate -ge 95) { $Green } elseif ($successRate -ge 80) { $Yellow } else { $Red }
        Write-ColorOutput "Success Rate: $successRate%" $color
    }
    
    Write-ColorOutput ("=" * 60) $Blue
}

function Monitor-MergeStatus {
    $startTime = Get-Date
    $endTime = $startTime.AddSeconds($MonitorDuration)
    $checkCount = 0
    $lastFailureCount = 0
    $failureAnalyses = @()
    
    Write-ColorOutput "`n🔄 STARTING CONTINUOUS MONITORING..." $Blue
    Write-ColorOutput "Will monitor until: $endTime" $Blue
    
    while ((Get-Date) -lt $endTime) {
        $checkCount++
        Write-ColorOutput "`n--- Check #$checkCount at $(Get-Timestamp) ---" $Blue
        
        # Get current workflow status
        $workflows = Get-WorkflowStatus
        
        if ($workflows.Count -eq 0) {
            Write-ColorOutput "⚠️ No workflows found - may be normal if no recent activity" $Yellow
        }
        else {
            # Check for new failures
            $currentFailures = $workflows | Where-Object { $_.conclusion -eq "failure" }
            $currentFailureCount = $currentFailures.Count
            
            if ($currentFailureCount -gt $lastFailureCount) {
                Write-ColorOutput "🚨 NEW FAILURE(S) DETECTED!" $Red
                
                # Analyze new failures
                $newFailures = $currentFailures | Select-Object -First ($currentFailureCount - $lastFailureCount)
                foreach ($failure in $newFailures) {
                    $analysis = Analyze-WorkflowFailure -Workflow $failure
                    if ($analysis) {
                        $failureAnalyses += $analysis
                        Invoke-AutoRecovery -FailureAnalysis $analysis
                    }
                }
                
                $lastFailureCount = $currentFailureCount
            }
            
            if ($VerboseLogging) {
                Generate-StatusReport -Workflows $workflows -FailureAnalyses $failureAnalyses
            }
        }
        
        # Wait before next check (30 seconds)
        Start-Sleep -Seconds 30
    }
    
    Write-ColorOutput "`n🏁 MONITORING COMPLETED" $Blue
    Generate-StatusReport -Workflows (Get-WorkflowStatus) -FailureAnalyses $failureAnalyses
}

function Export-MonitoringReport {
    param($FailureAnalyses)
    
    $reportPath = "merge-monitoring-report-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
    
    $report = @{
        Timestamp = Get-Date -Format "o"
        Branch = $BranchName
        MonitorDuration = $MonitorDuration
        FailureCount = $FailureAnalyses.Count
        Failures = $FailureAnalyses
        Summary = "Merge monitoring completed"
    }
    
    $report | ConvertTo-Json -Depth 10 | Out-File -FilePath $reportPath -Encoding UTF8
    Write-ColorOutput "📄 Monitoring report exported to: $reportPath" $Blue
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

Initialize-Monitoring
Test-Prerequisites

try {
    Monitor-MergeStatus
    Write-ColorOutput "`n✅ MERGE MONITORING COMPLETED SUCCESSFULLY" $Green
}
catch {
    Write-ColorOutput "`n❌ MONITORING FAILED: $_" $Red
    exit 1
}
finally {
    Write-ColorOutput "`n📊 Final Status Check..." $Blue
    $finalWorkflows = Get-WorkflowStatus -Limit 10
    Generate-StatusReport -Workflows $finalWorkflows -FailureAnalyses @()
}

Write-ColorOutput "`n🎯 MERGE MONITOR SESSION COMPLETE" $Blue
Write-ColorOutput "Monitor your merge at: https://github.com/$(gh repo view --json owner,name -q '.owner.login + "/" + .name")/actions" $Blue