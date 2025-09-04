# CI Fix Loop Automation Script
# Systematic CI error fixing with iteration tracking

param(
    [int]$MaxIterations = 10,
    [int]$BaseTimeout = 120,
    [double]$TimeoutMultiplier = 1.5,
    [switch]$DryRun = $false
)

# Initialize session
$iteration = 1
$sessionStart = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$logFile = "ci-fix-loop-session-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
$fixesApplied = @()

Write-Host "🚀 Starting CI Fix Loop Session: $sessionStart" -ForegroundColor Green
Write-Host "📊 Configuration: MaxIter=$MaxIterations, BaseTimeout=${BaseTimeout}s, Multiplier=$TimeoutMultiplier" -ForegroundColor Cyan

# Update CI_FIX_LOOP.md session start
function Update-SessionStatus {
    param($Status, $ErrorCount = 0, $Fixes = @())
    
    $content = Get-Content "CI_FIX_LOOP.md" -Raw
    $content = $content -replace "- \*\*Started\*\*: .*", "- **Started**: $sessionStart"
    $content = $content -replace "- \*\*Iteration\*\*: .*", "- **Iteration**: $iteration"
    $content = $content -replace "- \*\*Status\*\*: .*", "- **Status**: $Status"
    $content = $content -replace "- \*\*Fixes Applied\*\*: .*", "- **Fixes Applied**: $($fixesApplied.Count)"
    
    Set-Content "CI_FIX_LOOP.md" $content
}

function Apply-Fixes {
    param([string[]]$Errors, [int]$Iteration)
    
    $fixes = @()
    
    Write-Host "🔧 Analyzing $($Errors.Count) errors for fixes..." -ForegroundColor Yellow
    
    foreach ($error in $Errors) {
        Write-Host "  🔍 $error" -ForegroundColor Gray
        
        switch -Regex ($error) {
            "undefined:|not declared|undeclared name" {
                Write-Host "    ↳ 🔧 Applying goimports fix..." -ForegroundColor Blue
                if (-not $DryRun) {
                    & goimports -w .
                    if ($LASTEXITCODE -eq 0) { $fixes += "goimports" }
                }
            }
            "package .+ is not in GOROOT|cannot find package|no Go files" {
                Write-Host "    ↳ 🔧 Running go mod tidy..." -ForegroundColor Blue
                if (-not $DryRun) {
                    & go mod tidy
                    if ($LASTEXITCODE -eq 0) { $fixes += "go_mod_tidy" }
                }
            }
            "test timeout|context deadline exceeded" {
                Write-Host "    ↳ 🔧 Increasing test timeout..." -ForegroundColor Blue
                # This requires specific test file modification
                $fixes += "timeout_increase_needed"
            }
            "build constraints exclude all Go files|build tag" {
                Write-Host "    ↳ 🔧 Checking build constraints..." -ForegroundColor Blue
                # Review and fix build tags
                $fixes += "build_constraints_review"
            }
            "cannot use .+ as .+ in argument|type .+ is not an expression" {
                Write-Host "    ↳ ⚠️ Type conversion issue - manual review needed" -ForegroundColor Magenta
                $fixes += "manual_type_fix"
            }
            "missing return|not enough arguments|too many arguments" {
                Write-Host "    ↳ ⚠️ Function signature mismatch - manual review needed" -ForegroundColor Magenta
                $fixes += "manual_function_fix"
            }
            default {
                Write-Host "    ↳ ❓ Unknown error pattern - manual review required" -ForegroundColor Red
                $fixes += "manual_review"
            }
        }
    }
    
    # Apply additional common fixes
    if ($fixes -contains "goimports" -or $fixes -contains "go_mod_tidy") {
        Write-Host "  🔧 Running additional cleanup..." -ForegroundColor Blue
        if (-not $DryRun) {
            & go fmt ./...
            & go vet ./...
        }
    }
    
    return $fixes
}

function Test-LocalCI {
    param([int]$TimeoutSeconds)
    
    Write-Host "🧪 Running local CI (timeout: ${TimeoutSeconds}s)..." -ForegroundColor Cyan
    
    $testErrors = @()
    $lintErrors = @()
    
    # Run tests with timeout
    $testJob = Start-Job -ScriptBlock { 
        param($timeout)
        $env:GOPROXY = "direct"
        & go test ./... -timeout="${timeout}s" 2>&1
    } -ArgumentList $TimeoutSeconds
    
    Wait-Job $testJob -Timeout $TimeoutSeconds | Out-Null
    $testResult = Receive-Job $testJob
    Remove-Job $testJob -Force
    
    if ($testResult -match "FAIL|ERROR|undefined|cannot|panic") {
        $testErrors = $testResult | Where-Object { $_ -match "FAIL|ERROR|undefined|cannot|panic" }
    }
    
    # Run linter
    $lintResult = & golangci-lint run --timeout="${TimeoutSeconds}s" 2>&1
    if ($LASTEXITCODE -ne 0) {
        $lintErrors = $lintResult | Where-Object { $_ -match "ERROR|WARN" }
    }
    
    $allErrors = $testErrors + $lintErrors
    return @{
        Success = ($allErrors.Count -eq 0)
        Errors = $allErrors
        TestOutput = $testResult
        LintOutput = $lintResult
    }
}

# Main loop
while ($iteration -le $MaxIterations) {
    $timeout = [math]::Round($BaseTimeout * [math]::Pow($TimeoutMultiplier, $iteration - 1))
    $iterStart = Get-Date
    
    Write-Host "`n📍 Iteration $iteration/$MaxIterations (Timeout: ${timeout}s)" -ForegroundColor White -BackgroundColor DarkBlue
    Update-SessionStatus "RUNNING - Iteration $iteration"
    
    # Save error output to file
    $errorFile = "ci-errors-iter-$iteration.log"
    
    # Run local CI
    $ciResult = Test-LocalCI -TimeoutSeconds $timeout
    
    # Log results
    $ciResult.TestOutput + $ciResult.LintOutput | Out-File $errorFile -Encoding UTF8
    
    if ($ciResult.Success) {
        Write-Host "✅ Local CI passed!" -ForegroundColor Green
        
        if (-not $DryRun) {
            Write-Host "📤 Committing and pushing..." -ForegroundColor Yellow
            git add -A
            git commit -m "fix(ci): iteration $iteration - clean build achieved`n`nFixes applied: $($fixesApplied -join ', ')"
            git push origin HEAD
            
            Write-Host "⏳ Waiting 5 minutes for PR CI..." -ForegroundColor Yellow
            Start-Sleep 300
            
            Write-Host "🔍 Checking PR status..." -ForegroundColor Cyan
            $prStatus = & gh pr checks 2>&1
            
            if ($prStatus -match "✓|pass|success") {
                Write-Host "🎉 CI Fix Loop COMPLETED successfully!" -ForegroundColor Green -BackgroundColor DarkGreen
                Update-SessionStatus "✅ COMPLETED"
                
                # Final summary
                Write-Host "`n📊 FINAL SUMMARY:" -ForegroundColor White -BackgroundColor DarkGreen
                Write-Host "  • Total Iterations: $iteration" -ForegroundColor Green
                Write-Host "  • Total Fixes Applied: $($fixesApplied.Count)" -ForegroundColor Green
                Write-Host "  • Session Duration: $((Get-Date) - [DateTime]$sessionStart)" -ForegroundColor Green
                Write-Host "  • Fixes: $($fixesApplied -join ', ')" -ForegroundColor Green
                
                exit 0
            } else {
                Write-Host "❌ PR CI still failing:" -ForegroundColor Red
                Write-Host $prStatus -ForegroundColor Red
                Write-Host "📥 Pulling remote changes and continuing..." -ForegroundColor Yellow
                git pull origin $(git rev-parse --abbrev-ref HEAD) --no-rebase
            }
        } else {
            Write-Host "🧪 DRY RUN: Would push changes now" -ForegroundColor Cyan
        }
    } else {
        Write-Host "❌ Found $($ciResult.Errors.Count) errors:" -ForegroundColor Red
        $ciResult.Errors | ForEach-Object { Write-Host "  • $_" -ForegroundColor Red }
        
        # Apply fixes
        $iterationFixes = Apply-Fixes -Errors $ciResult.Errors -Iteration $iteration
        $fixesApplied += $iterationFixes
        
        Write-Host "🔧 Applied fixes this iteration: $($iterationFixes -join ', ')" -ForegroundColor Blue
    }
    
    $duration = (Get-Date) - $iterStart
    Write-Host "⏱️ Iteration $iteration completed in $($duration.ToString('mm\:ss'))" -ForegroundColor Gray
    
    # Log to main session file
    "Iteration $iteration | Duration: $($duration.ToString('mm\:ss')) | Errors: $($ciResult.Errors.Count) | Fixes: $($iterationFixes -join ',')" | Add-Content $logFile
    
    $iteration++
}

# Failed after max iterations
Write-Host "❌ CI Fix Loop FAILED after $MaxIterations iterations" -ForegroundColor Red -BackgroundColor DarkRed
Update-SessionStatus "❌ FAILED"

Write-Host "`n📊 FAILURE SUMMARY:" -ForegroundColor White -BackgroundColor DarkRed
Write-Host "  • Total Iterations: $MaxIterations" -ForegroundColor Red
Write-Host "  • Total Fixes Attempted: $($fixesApplied.Count)" -ForegroundColor Red
Write-Host "  • Session Duration: $((Get-Date) - [DateTime]$sessionStart)" -ForegroundColor Red
Write-Host "  • Last Error Log: $errorFile" -ForegroundColor Red
Write-Host "  • Session Log: $logFile" -ForegroundColor Red

Write-Host "`n🔧 MANUAL INTERVENTION REQUIRED:" -ForegroundColor Yellow
Write-Host "  1. Review error logs: Get-Content '$errorFile'" -ForegroundColor Yellow
Write-Host "  2. Check session log: Get-Content '$logFile'" -ForegroundColor Yellow
Write-Host "  3. Apply manual fixes and re-run" -ForegroundColor Yellow

exit 1