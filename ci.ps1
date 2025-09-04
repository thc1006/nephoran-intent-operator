# CI Fix Loop Launcher - Simple Interface

param(
    [string]$Action = "status",
    [switch]$DryRun = $false,
    [int]$MaxIterations = 10
)

function Show-Help {
    Write-Host "🔧 CI Fix Loop System" -ForegroundColor White -BackgroundColor DarkBlue
    Write-Host ""
    Write-Host "Usage: .\ci.ps1 [action] [options]" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Actions:" -ForegroundColor White
    Write-Host "  start       Start automated CI fix loop" -ForegroundColor Green
    Write-Host "  status      Show current status and recent errors (default)" -ForegroundColor Yellow
    Write-Host "  quickfix    Apply common fixes manually" -ForegroundColor Blue
    Write-Host "  errors      Show detailed error analysis" -ForegroundColor Red
    Write-Host "  logs        Show session logs" -ForegroundColor Cyan
    Write-Host "  test        Run local tests only" -ForegroundColor Magenta
    Write-Host "  clean       Clean up log files" -ForegroundColor Gray
    Write-Host "  help        Show this help" -ForegroundColor White
    Write-Host ""
    Write-Host "Options:" -ForegroundColor White
    Write-Host "  -DryRun           Don't make actual changes (for start/quickfix)" -ForegroundColor Gray
    Write-Host "  -MaxIterations N  Set maximum iterations (default: 10)" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor White
    Write-Host "  .\ci.ps1 start                # Start the automated loop" -ForegroundColor Cyan
    Write-Host "  .\ci.ps1 start -DryRun        # Preview what would happen" -ForegroundColor Cyan
    Write-Host "  .\ci.ps1 status               # Check current status" -ForegroundColor Cyan
    Write-Host "  .\ci.ps1 quickfix             # Apply manual fixes" -ForegroundColor Cyan
    Write-Host "  .\ci.ps1 test                 # Just run tests" -ForegroundColor Cyan
}

switch ($Action.ToLower()) {
    "start" {
        Write-Host "🚀 Starting CI Fix Loop..." -ForegroundColor Green
        if ($DryRun) {
            .\ci-fix-loop.ps1 -MaxIterations $MaxIterations -DryRun
        } else {
            .\ci-fix-loop.ps1 -MaxIterations $MaxIterations
        }
    }
    
    "status" {
        Write-Host "📊 Checking CI Status..." -ForegroundColor Yellow
        .\ci-status.ps1
    }
    
    "quickfix" {
        Write-Host "🔧 Applying Quick Fixes..." -ForegroundColor Blue
        if ($DryRun) {
            Write-Host "🧪 DRY RUN: Would apply quick fixes" -ForegroundColor Cyan
        } else {
            .\ci-status.ps1 -QuickFix
        }
    }
    
    "errors" {
        Write-Host "🔍 Analyzing Errors..." -ForegroundColor Red
        .\ci-status.ps1 -ShowErrors
    }
    
    "logs" {
        Write-Host "📜 Showing Session Logs..." -ForegroundColor Cyan
        .\ci-status.ps1 -ShowLogs
    }
    
    "test" {
        Write-Host "🧪 Running Local Tests..." -ForegroundColor Magenta
        Write-Host "Running go test..." -ForegroundColor Cyan
        $testStart = Get-Date
        go test ./... -timeout=120s
        $testEnd = Get-Date
        $duration = $testEnd - $testStart
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Tests passed in $($duration.ToString('mm\:ss'))" -ForegroundColor Green
            
            Write-Host "Running golangci-lint..." -ForegroundColor Cyan
            golangci-lint run --timeout=120s
            
            if ($LASTEXITCODE -eq 0) {
                Write-Host "✅ Linter passed" -ForegroundColor Green
                Write-Host "🎉 Local CI is clean!" -ForegroundColor Green -BackgroundColor DarkGreen
            } else {
                Write-Host "❌ Linter failed" -ForegroundColor Red
            }
        } else {
            Write-Host "❌ Tests failed in $($duration.ToString('mm\:ss'))" -ForegroundColor Red
        }
    }
    
    "clean" {
        Write-Host "🧹 Cleaning up log files..." -ForegroundColor Gray
        $errorFiles = Get-ChildItem -Filter "ci-errors-iter-*.log"
        $sessionFiles = Get-ChildItem -Filter "ci-fix-loop-session-*.log"
        
        if ($errorFiles.Count -gt 0) {
            Write-Host "  Removing $($errorFiles.Count) error files..." -ForegroundColor Gray
            $errorFiles | Remove-Item -Force
        }
        
        if ($sessionFiles.Count -gt 0) {
            Write-Host "  Removing $($sessionFiles.Count) session files..." -ForegroundColor Gray
            $sessionFiles | Remove-Item -Force
        }
        
        Write-Host "✅ Cleanup complete" -ForegroundColor Green
    }
    
    "help" {
        Show-Help
    }
    
    default {
        Write-Host "❌ Unknown action: $Action" -ForegroundColor Red
        Write-Host ""
        Show-Help
        exit 1
    }
}