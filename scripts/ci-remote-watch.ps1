# ci-remote-watch.ps1 - Monitor remote CI and download logs
param(
    [int]$TimeoutSeconds = 1800  # 30 minutes default
)

$ErrorActionPreference = "Stop"

Write-Host "=== REMOTE CI MONITORING ===" -ForegroundColor Cyan

try {
    # Get current branch
    $branch = git rev-parse --abbrev-ref HEAD
    Write-Host "Branch: $branch" -ForegroundColor Green

    # Get latest run ID for this branch  
    $runJson = gh run list -b $branch -L 1 --json databaseId,status,conclusion,url,displayTitle | ConvertFrom-Json
    $runId = $runJson[0].databaseId
    Write-Host "Monitoring run: $runId" -ForegroundColor Green
    Write-Host "Title: $($runJson[0].displayTitle)" -ForegroundColor Gray

    # Watch with timeout
    Write-Host "⏰ Watching for $TimeoutSeconds seconds..." -ForegroundColor Yellow
    
    # Create timeout job
    $job = Start-Job -ScriptBlock {
        param($runId)
        gh run watch --exit-status $runId
    } -ArgumentList $runId

    if (Wait-Job $job -Timeout $TimeoutSeconds) {
        $result = Receive-Job $job
        $exitCode = $job.State -eq "Failed" ? 1 : 0
    } else {
        Write-Host "⚠️ Timeout reached, stopping watch" -ForegroundColor Yellow
        Stop-Job $job
        $exitCode = 2
    }

    Remove-Job $job -Force

    # Download logs regardless
    Write-Host "📥 Downloading logs..." -ForegroundColor Green
    if (-not (Test-Path .ci_logs)) {
        New-Item -ItemType Directory -Path .ci_logs -Force | Out-Null
    }

    try {
        gh run view $runId --log-failed > .ci_logs/failed.log
        gh run view $runId --log > .ci_logs/full.log  
        gh run download $runId --dir .ci_logs
        Write-Host "✅ Logs saved to .ci_logs/" -ForegroundColor Green
    } catch {
        Write-Host "⚠️ Failed to download some logs: $_" -ForegroundColor Yellow
    }

    if ($exitCode -eq 0) {
        Write-Host "✅ CI PASSED" -ForegroundColor Green
    } else {
        Write-Host "❌ CI FAILED - logs available in .ci_logs/" -ForegroundColor Red
        exit $exitCode
    }

} catch {
    Write-Host "❌ MONITORING FAILED: $_" -ForegroundColor Red
    exit 1
}