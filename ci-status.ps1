# CI Status Checker and Manual Intervention Helper

param(
    [switch]$ShowErrors = $false,
    [switch]$ShowLogs = $false,
    [switch]$QuickFix = $false,
    [int]$Iteration = 1
)

function Get-CIStatus {
    if (Test-Path "CI_FIX_LOOP.md") {
        $content = Get-Content "CI_FIX_LOOP.md" -Raw
        
        $status = if ($content -match "- \*\*Status\*\*: (.+)") { $matches[1] } else { "Unknown" }
        $iteration = if ($content -match "- \*\*Iteration\*\*: (\d+)") { $matches[1] } else { "0" }
        $started = if ($content -match "- \*\*Started\*\*: (.+)") { $matches[1] } else { "Not started" }
        $fixes = if ($content -match "- \*\*Fixes Applied\*\*: (\d+)") { $matches[1] } else { "0" }
        
        return @{
            Status = $status
            Iteration = $iteration
            Started = $started
            FixesApplied = $fixes
        }
    }
    return $null
}

function Show-ErrorSummary {
    $errorFiles = Get-ChildItem -Filter "ci-errors-iter-*.log" | Sort-Object Name
    
    if ($errorFiles.Count -eq 0) {
        Write-Host "No error files found." -ForegroundColor Green
        return
    }
    
    Write-Host "`n📊 ERROR SUMMARY:" -ForegroundColor White -BackgroundColor DarkRed
    
    foreach ($file in $errorFiles) {
        $iteration = $file.Name -replace 'ci-errors-iter-(\d+)\.log', '$1'
        $content = Get-Content $file.FullName -ErrorAction SilentlyContinue
        $errorCount = ($content | Where-Object { $_ -match "FAIL|ERROR|undefined|cannot|panic" }).Count
        
        Write-Host "  Iteration $iteration`: $errorCount errors ($($file.Name))" -ForegroundColor Red
        
        if ($ShowErrors -and $content) {
            Write-Host "    Errors:" -ForegroundColor Yellow
            $content | Where-Object { $_ -match "FAIL|ERROR|undefined|cannot|panic" } | 
                Select-Object -First 5 | ForEach-Object { Write-Host "      • $_" -ForegroundColor Gray }
            if ($errorCount -gt 5) {
                Write-Host "      ... and $($errorCount - 5) more errors" -ForegroundColor Gray
            }
        }
    }
}

function Show-SessionLogs {
    $logFiles = Get-ChildItem -Filter "ci-fix-loop-session-*.log" | Sort-Object LastWriteTime -Descending
    
    if ($logFiles.Count -eq 0) {
        Write-Host "No session logs found." -ForegroundColor Yellow
        return
    }
    
    Write-Host "`n📜 SESSION LOGS:" -ForegroundColor White -BackgroundColor DarkBlue
    
    $latestLog = $logFiles[0]
    Write-Host "Latest session: $($latestLog.Name)" -ForegroundColor Cyan
    
    $content = Get-Content $latestLog.FullName -ErrorAction SilentlyContinue
    if ($content) {
        $content | Select-Object -Last 10 | ForEach-Object { Write-Host "  $_" -ForegroundColor White }
    }
    
    if ($logFiles.Count -gt 1) {
        Write-Host "`nOther sessions:" -ForegroundColor Gray
        $logFiles[1..($logFiles.Count-1)] | ForEach-Object { 
            Write-Host "  • $($_.Name) ($('{0:yyyy-MM-dd HH:mm}' -f $_.LastWriteTime))" -ForegroundColor Gray 
        }
    }
}

function Start-QuickFix {
    param([int]$Iteration)
    
    Write-Host "🚀 Starting Quick Fix for Iteration $Iteration..." -ForegroundColor Green
    
    # Common fixes
    $fixes = @()
    
    Write-Host "🔧 Applying common fixes..." -ForegroundColor Blue
    
    # 1. Go imports
    Write-Host "  • Running goimports..." -ForegroundColor Cyan
    try {
        & goimports -w . 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { $fixes += "goimports" }
    } catch {
        Write-Host "    ⚠️ goimports not found, skipping..." -ForegroundColor Yellow
    }
    
    # 2. Go mod tidy
    Write-Host "  • Running go mod tidy..." -ForegroundColor Cyan
    & go mod tidy
    if ($LASTEXITCODE -eq 0) { $fixes += "go_mod_tidy" }
    
    # 3. Go fmt
    Write-Host "  • Running go fmt..." -ForegroundColor Cyan
    & go fmt ./...
    if ($LASTEXITCODE -eq 0) { $fixes += "go_fmt" }
    
    # 4. Go vet
    Write-Host "  • Running go vet..." -ForegroundColor Cyan
    $vetOutput = & go vet ./... 2>&1
    if ($LASTEXITCODE -eq 0) { 
        $fixes += "go_vet" 
    } else {
        Write-Host "    ⚠️ Go vet found issues:" -ForegroundColor Yellow
        $vetOutput | Select-Object -First 3 | ForEach-Object { Write-Host "      $_" -ForegroundColor Gray }
    }
    
    Write-Host "✅ Quick fixes applied: $($fixes -join ', ')" -ForegroundColor Green
    
    # Test the result
    Write-Host "🧪 Testing fixes..." -ForegroundColor Yellow
    $testOutput = & go test ./... -timeout=60s 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Quick fix successful! Ready to commit." -ForegroundColor Green
        
        Write-Host "`n📤 Commit these changes? (y/n): " -ForegroundColor Yellow -NoNewline
        $response = Read-Host
        
        if ($response -eq 'y' -or $response -eq 'Y' -or $response -eq 'yes') {
            git add -A
            git commit -m "fix(ci): manual quick fix iteration $Iteration - $($fixes -join ', ')"
            Write-Host "✅ Changes committed." -ForegroundColor Green
            
            Write-Host "`n📤 Push changes? (y/n): " -ForegroundColor Yellow -NoNewline
            $pushResponse = Read-Host
            
            if ($pushResponse -eq 'y' -or $pushResponse -eq 'Y' -or $pushResponse -eq 'yes') {
                git push origin HEAD
                Write-Host "✅ Changes pushed." -ForegroundColor Green
            }
        }
    } else {
        Write-Host "❌ Quick fix failed. Manual intervention still required." -ForegroundColor Red
        Write-Host "Test output:" -ForegroundColor Red
        $testOutput | Select-Object -First 10 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
    }
}

function Show-Commands {
    Write-Host "`n🔧 USEFUL COMMANDS:" -ForegroundColor White -BackgroundColor DarkBlue
    Write-Host "  Start CI Fix Loop:   .\ci-fix-loop.ps1" -ForegroundColor Cyan
    Write-Host "  Dry Run:             .\ci-fix-loop.ps1 -DryRun" -ForegroundColor Cyan
    Write-Host "  Show Errors:         .\ci-status.ps1 -ShowErrors" -ForegroundColor Cyan
    Write-Host "  Show Logs:           .\ci-status.ps1 -ShowLogs" -ForegroundColor Cyan
    Write-Host "  Quick Fix:           .\ci-status.ps1 -QuickFix" -ForegroundColor Cyan
    Write-Host "  Manual Test:         go test ./..." -ForegroundColor Cyan
    Write-Host "  Check PR:            gh pr checks" -ForegroundColor Cyan
    Write-Host "  PR Status:           gh pr view --json statusCheckRollup" -ForegroundColor Cyan
}

# Main execution
Write-Host "🔍 CI Fix Loop Status Checker" -ForegroundColor White -BackgroundColor DarkBlue

$status = Get-CIStatus
if ($status) {
    Write-Host "`n📊 CURRENT STATUS:" -ForegroundColor White -BackgroundColor DarkGreen
    Write-Host "  • Status: $($status.Status)" -ForegroundColor Green
    Write-Host "  • Iteration: $($status.Iteration)" -ForegroundColor Green
    Write-Host "  • Started: $($status.Started)" -ForegroundColor Green
    Write-Host "  • Fixes Applied: $($status.FixesApplied)" -ForegroundColor Green
} else {
    Write-Host "`n⚠️  CI_FIX_LOOP.md not found. Loop not initialized." -ForegroundColor Yellow
}

if ($ShowErrors -or $ShowLogs) {
    if ($ShowErrors) { Show-ErrorSummary }
    if ($ShowLogs) { Show-SessionLogs }
} elseif ($QuickFix) {
    Start-QuickFix -Iteration $Iteration
} else {
    # Default: show summary
    Show-ErrorSummary
    Show-Commands
    
    # Check if we should show recent errors
    $recentErrors = Get-ChildItem -Filter "ci-errors-iter-*.log" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($recentErrors) {
        Write-Host "`n🔍 Most recent errors (use -ShowErrors for full details):" -ForegroundColor Yellow
        $content = Get-Content $recentErrors.FullName -ErrorAction SilentlyContinue
        $errors = $content | Where-Object { $_ -match "FAIL|ERROR|undefined|cannot|panic" } | Select-Object -First 3
        $errors | ForEach-Object { Write-Host "  • $_" -ForegroundColor Red }
    }
}

# Check Git status
$gitStatus = & git status --porcelain 2>$null
if ($gitStatus) {
    Write-Host "`n📝 Uncommitted changes:" -ForegroundColor Yellow
    $gitStatus | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
}