# CI Reproduction Loop - Iterative linting with error capture
# Runs golangci-lint repeatedly until all issues are resolved

param(
    [switch]$Fast = $false,
    [switch]$AutoFix = $false,
    [int]$MaxIterations = 10,
    [int]$DelaySeconds = 2,
    [switch]$WatchMode = $false,
    [string]$LogFile = "ci-loop.log"
)

$ErrorActionPreference = "Continue"  # Allow continuing on errors

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Cyan = "Cyan"
$Blue = "Blue"

# Initialize log file
$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
"=== CI Loop Started: $timestamp ===" | Out-File -FilePath $LogFile -Encoding utf8
"Configuration: Fast=$Fast, AutoFix=$AutoFix, MaxIterations=$MaxIterations" | Out-File -FilePath $LogFile -Encoding utf8 -Append

function Write-LoggedHost {
    param($Message, $Color = "White")
    Write-Host $Message -ForegroundColor $Color
    "$(Get-Date -Format 'HH:mm:ss') $Message" | Out-File -FilePath $LogFile -Encoding utf8 -Append
}

function Show-Banner {
    param($Title, $Color = $Cyan)
    
    $border = "=" * 60
    Write-Host ""
    Write-Host $border -ForegroundColor $Color
    Write-Host $Title -ForegroundColor $Color
    Write-Host $border -ForegroundColor $Color
    Write-Host ""
}

function Parse-LintResults {
    param($Output)
    
    $results = @{
        TotalIssues = 0
        IssuesByLinter = @{}
        IssuesByFile = @{}
        CriticalErrors = @()
        Warnings = @()
    }
    
    if (-not $Output) { return $results }
    
    $lines = $Output -split "`n"
    
    foreach ($line in $lines) {
        # Match golangci-lint output format: file:line:col: message (linter)
        if ($line -match '^([^:]+):(\d+):(\d+):\s*(.+?)\s*\((\w+)\)') {
            $file = $Matches[1]
            $lineNum = $Matches[2]
            $col = $Matches[3]
            $message = $Matches[4]
            $linter = $Matches[5]
            
            $results.TotalIssues++
            
            # Count by linter
            if ($results.IssuesByLinter.ContainsKey($linter)) {
                $results.IssuesByLinter[$linter]++
            } else {
                $results.IssuesByLinter[$linter] = 1
            }
            
            # Count by file
            if ($results.IssuesByFile.ContainsKey($file)) {
                $results.IssuesByFile[$file]++
            } else {
                $results.IssuesByFile[$file] = 1
            }
            
            # Categorize severity
            if ($linter -in @("typecheck", "staticcheck", "govet", "gosec")) {
                $results.CriticalErrors += "$file:$lineNum - $message ($linter)"
            } else {
                $results.Warnings += "$file:$lineNum - $message ($linter)"
            }
        }
    }
    
    return $results
}

function Show-IterationSummary {
    param($Iteration, $Results, $Duration, $ExitCode)
    
    Write-Host ""
    Write-Host "📊 ITERATION $Iteration SUMMARY" -ForegroundColor $Cyan
    Write-Host "Duration: $($Duration.TotalSeconds.ToString('F2'))s"
    Write-Host "Exit Code: $ExitCode"
    Write-Host "Total Issues: $($Results.TotalIssues)"
    
    if ($Results.TotalIssues -gt 0) {
        if ($Results.CriticalErrors.Count -gt 0) {
            Write-Host "❌ Critical Errors: $($Results.CriticalErrors.Count)" -ForegroundColor $Red
        }
        if ($Results.Warnings.Count -gt 0) {
            Write-Host "⚠️ Warnings: $($Results.Warnings.Count)" -ForegroundColor $Yellow
        }
        
        Write-Host ""
        Write-Host "🔍 Top Issues by Linter:"
        $Results.IssuesByLinter.GetEnumerator() | 
            Sort-Object Value -Descending | 
            Select-Object -First 5 | 
            ForEach-Object {
                Write-Host "  $($_.Key): $($_.Value)"
            }
            
        Write-Host ""
        Write-Host "📁 Top Files with Issues:"
        $Results.IssuesByFile.GetEnumerator() | 
            Sort-Object Value -Descending | 
            Select-Object -First 5 | 
            ForEach-Object {
                $fileName = Split-Path $_.Key -Leaf
                Write-Host "  $fileName: $($_.Value)"
            }
    }
}

function Wait-ForUserInput {
    if (-not $WatchMode) {
        Write-Host ""
        Write-Host "🔄 Press ENTER to continue, 'q' to quit, 'f' to toggle fix mode..." -ForegroundColor $Yellow
        $input = Read-Host
        
        switch ($input.ToLower()) {
            'q' { 
                Write-LoggedHost "User requested quit" $Yellow
                return $false 
            }
            'f' { 
                $script:AutoFix = -not $script:AutoFix
                Write-LoggedHost "Auto-fix toggled: $script:AutoFix" $Yellow
            }
        }
    } else {
        Write-Host "⏳ Waiting $DelaySeconds seconds before next iteration..." -ForegroundColor $Yellow
        Start-Sleep -Seconds $DelaySeconds
    }
    
    return $true
}

# Main execution
Show-Banner "🔄 CI REPRODUCTION LOOP STARTED"

Write-LoggedHost "Starting CI reproduction loop..." $Green
Write-LoggedHost "Working Directory: $(Get-Location)"
Write-LoggedHost "Fast Mode: $Fast"
Write-LoggedHost "Auto-Fix: $AutoFix" 
Write-LoggedHost "Max Iterations: $MaxIterations"
Write-LoggedHost "Watch Mode: $WatchMode"

$iteration = 0
$allPassed = $false
$totalIssuesHistory = @()

while ($iteration -lt $MaxIterations) {
    $iteration++
    
    Show-Banner "🧪 ITERATION $iteration / $MaxIterations"
    
    Write-LoggedHost "Running golangci-lint (iteration $iteration)..." $Blue
    
    # Build arguments for lint script
    $lintArgs = @()
    if ($Fast) { $lintArgs += "-Fast" }
    if ($AutoFix) { $lintArgs += "-Fix" }
    $lintArgs += "-Verbose"
    
    # Execute linting
    $startTime = Get-Date
    
    try {
        # Capture both output and exit code
        $process = Start-Process -FilePath "powershell.exe" -ArgumentList @("-File", "scripts\run-lint-local.ps1") + $lintArgs -NoNewWindow -PassThru -Wait -RedirectStandardOutput "lint-temp.out" -RedirectStandardError "lint-temp.err"
        $exitCode = $process.ExitCode
        
        $output = ""
        if (Test-Path "lint-temp.out") {
            $output += Get-Content "lint-temp.out" -Raw
        }
        if (Test-Path "lint-temp.err") {
            $errorOutput = Get-Content "lint-temp.err" -Raw
            if ($errorOutput) {
                $output += "`n$errorOutput"
            }
        }
        
        # Clean up temp files
        Remove-Item "lint-temp.out" -ErrorAction SilentlyContinue
        Remove-Item "lint-temp.err" -ErrorAction SilentlyContinue
        
    } catch {
        Write-LoggedHost "❌ Failed to execute lint script: $($_.Exception.Message)" $Red
        $exitCode = 1
        $output = $_.Exception.Message
    }
    
    $endTime = Get-Date
    $duration = $endTime - $startTime
    
    # Parse results
    $results = Parse-LintResults -Output $output
    $totalIssuesHistory += $results.TotalIssues
    
    # Log full output
    "=== Iteration $iteration Output ===" | Out-File -FilePath $LogFile -Encoding utf8 -Append
    $output | Out-File -FilePath $LogFile -Encoding utf8 -Append
    
    # Show summary
    Show-IterationSummary -Iteration $iteration -Results $results -Duration $duration -ExitCode $exitCode
    
    if ($exitCode -eq 0) {
        Write-LoggedHost "✅ SUCCESS! No linting issues found." $Green
        $allPassed = $true
        break
    }
    
    # Show progress trend
    if ($totalIssuesHistory.Count -gt 1) {
        $previousIssues = $totalIssuesHistory[-2]
        $currentIssues = $totalIssuesHistory[-1]
        $change = $currentIssues - $previousIssues
        
        if ($change -lt 0) {
            Write-Host "📈 Progress: $([Math]::Abs($change)) fewer issues than last iteration" -ForegroundColor $Green
        } elseif ($change -gt 0) {
            Write-Host "📉 Regression: $change more issues than last iteration" -ForegroundColor $Red
        } else {
            Write-Host "➡️ No change in issue count" -ForegroundColor $Yellow
        }
    }
    
    # Show some critical errors if present
    if ($results.CriticalErrors.Count -gt 0) {
        Write-Host ""
        Write-Host "🚨 CRITICAL ERRORS (first 3):" -ForegroundColor $Red
        $results.CriticalErrors | Select-Object -First 3 | ForEach-Object {
            Write-Host "  $_" -ForegroundColor $Red
        }
    }
    
    # Check if we should continue
    if (-not (Wait-ForUserInput)) {
        break
    }
}

# Final results
Show-Banner "🏁 CI LOOP RESULTS"

if ($allPassed) {
    Write-LoggedHost "🎉 SUCCESS! All linting issues resolved after $iteration iterations." $Green
} else {
    Write-LoggedHost "⏸️ Loop completed after $iteration iterations. Issues may still remain." $Yellow
}

# Show trend analysis
if ($totalIssuesHistory.Count -gt 1) {
    Write-Host ""
    Write-Host "📊 TREND ANALYSIS:" -ForegroundColor $Cyan
    Write-Host "  Initial Issues: $($totalIssuesHistory[0])"
    Write-Host "  Final Issues: $($totalIssuesHistory[-1])"
    Write-Host "  Net Change: $(($totalIssuesHistory[-1] - $totalIssuesHistory[0]))"
    
    Write-Host ""
    Write-Host "📈 Issues per iteration: $($totalIssuesHistory -join ', ')"
}

$finalTimestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
"=== CI Loop Ended: $finalTimestamp ===" | Out-File -FilePath $LogFile -Encoding utf8 -Append
"Final Result: Passed=$allPassed, Iterations=$iteration" | Out-File -FilePath $LogFile -Encoding utf8 -Append

Write-Host ""
Write-Host "📋 Full log saved to: $LogFile" -ForegroundColor $Blue
Write-Host ""

if ($allPassed) {
    exit 0
} else {
    exit 1
}