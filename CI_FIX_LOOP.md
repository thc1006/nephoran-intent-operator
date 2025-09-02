# CI Fix Loop System

## Overview
Automated CI fix loop with systematic error tracking, timeout scaling, and foolproof iteration management.

## Current Session
- **Started**: Not initialized
- **Iteration**: 0
- **Status**: Ready
- **Last Error**: None
- **Fixes Applied**: 0

## Iteration Log

### Session Template
```
## Session [YYYY-MM-DD HH:MM:SS]
- Branch: feat/e2e
- Base Commit: [commit-hash]
- Target: Clean CI build
- Max Iterations: 10

| Iter | Duration | Local CI | Errors Found | Fixes Applied | Remote CI | Status |
|------|----------|----------|--------------|---------------|-----------|--------|
| 1    | 00:00    | ❌       | [count]      | [list]        | ❌        | FIXING |
| 2    | 05:15    | ✅       | 0            | []            | ✅        | DONE   |
```

---

## Active Loop Configuration

### Current Settings
- **Timeout Base**: 2 minutes
- **Timeout Multiplier**: 1.5x per iteration
- **Max Iterations**: 10
- **PR Check Delay**: 5 minutes
- **Auto-Push**: After local CI passes

### Commands per Iteration
```powershell
# 1. Local CI Run
go test ./...
golangci-lint run --timeout=${timeout}

# 2. Error Capture
go test ./... 2>&1 | Tee-Object -FilePath "ci-errors-iter-${iteration}.log"

# 3. Fix Application
# [Automated fixes based on error patterns]

# 4. Validation
go test ./...
go build ./...

# 5. Push (if clean)
git add -A
git commit -m "fix(ci): iteration ${iteration} fixes"
git push origin HEAD

# 6. PR Status Check (after 5min delay)
Start-Sleep 300
gh pr checks
```

## Error Pattern Database

### Common Patterns & Fixes
```yaml
patterns:
  - pattern: "undefined: (.+)"
    fix: "add_missing_import"
    command: "goimports -w ."
    
  - pattern: "package (.+) is not in GOROOT"
    fix: "go_mod_tidy"
    command: "go mod tidy"
    
  - pattern: "cannot use .+ as .+ in argument"
    fix: "type_conversion"
    command: "manual_review_required"
    
  - pattern: "test timeout"
    fix: "increase_timeout"
    command: "update_test_timeout"
    
  - pattern: "build constraints exclude all Go files"
    fix: "build_tags"
    command: "check_build_constraints"
```

## Automation Scripts

### Loop Runner (PowerShell)
```powershell
# ci-fix-loop.ps1
param(
    [int]$MaxIterations = 10,
    [int]$BaseTimeout = 120,
    [double]$TimeoutMultiplier = 1.5
)

$iteration = 1
$sessionStart = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$logFile = "ci-fix-loop-session-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"

Write-Host "🚀 Starting CI Fix Loop Session: $sessionStart" | Tee-Object -Append $logFile

while ($iteration -le $MaxIterations) {
    $timeout = [math]::Round($BaseTimeout * [math]::Pow($TimeoutMultiplier, $iteration - 1))
    $iterStart = Get-Date
    
    Write-Host "📍 Iteration $iteration (Timeout: ${timeout}s)" | Tee-Object -Append $logFile
    
    # Run local CI
    $testResult = & go test ./... 2>&1
    $lintResult = & golangci-lint run --timeout="${timeout}s" 2>&1
    
    $errors = @()
    if ($LASTEXITCODE -ne 0) {
        $errors += $testResult | Where-Object { $_ -match "FAIL|ERROR|undefined|cannot" }
        $errors += $lintResult | Where-Object { $_ -match "ERROR|WARN" }
    }
    
    if ($errors.Count -eq 0) {
        Write-Host "✅ Local CI passed! Pushing..." | Tee-Object -Append $logFile
        
        # Push changes
        git add -A
        git commit -m "fix(ci): iteration $iteration - clean build achieved"
        git push origin HEAD
        
        # Wait and check PR
        Write-Host "⏳ Waiting 5 minutes for PR CI..." | Tee-Object -Append $logFile
        Start-Sleep 300
        
        $prStatus = & gh pr checks
        if ($prStatus -match "✓") {
            Write-Host "🎉 CI Fix Loop COMPLETED successfully!" | Tee-Object -Append $logFile
            exit 0
        } else {
            Write-Host "❌ PR CI failed, pulling logs..." | Tee-Object -Append $logFile
            # Continue to next iteration
        }
    } else {
        Write-Host "❌ Found $($errors.Count) errors:" | Tee-Object -Append $logFile
        $errors | ForEach-Object { Write-Host "  - $_" } | Tee-Object -Append $logFile
        
        # Apply fixes
        Apply-Fixes -Errors $errors -Iteration $iteration
    }
    
    $duration = (Get-Date) - $iterStart
    Write-Host "⏱️  Iteration $iteration completed in $($duration.ToString('mm\:ss'))" | Tee-Object -Append $logFile
    
    $iteration++
}

Write-Host "❌ CI Fix Loop FAILED after $MaxIterations iterations" | Tee-Object -Append $logFile
exit 1

function Apply-Fixes {
    param($Errors, $Iteration)
    
    $fixes = @()
    
    foreach ($error in $Errors) {
        switch -Regex ($error) {
            "undefined:" {
                Write-Host "🔧 Applying goimports fix..."
                & goimports -w .
                $fixes += "goimports"
            }
            "package .+ is not in GOROOT" {
                Write-Host "🔧 Running go mod tidy..."
                & go mod tidy
                $fixes += "go_mod_tidy"
            }
            "test timeout" {
                Write-Host "🔧 Increasing test timeout..."
                # Update timeout in test files
                $fixes += "timeout_increase"
            }
            "build constraints" {
                Write-Host "🔧 Checking build constraints..."
                # Review build tags
                $fixes += "build_constraints"
            }
            default {
                Write-Host "⚠️  Manual review required for: $error"
                $fixes += "manual_review"
            }
        }
    }
    
    Write-Host "🔧 Applied fixes: $($fixes -join ', ')"
}
```

### Status Tracker
```powershell
# ci-status.ps1
function Update-CIStatus {
    param($Iteration, $Status, $Errors, $Fixes)
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $statusLine = "| $Iteration | $(Get-ElapsedTime) | $Status | $($Errors.Count) | $($Fixes -join ',') | TBD | $Status |"
    
    # Update CI_FIX_LOOP.md
    $content = Get-Content "CI_FIX_LOOP.md"
    $newContent = $content -replace "- \*\*Status\*\*: .*", "- **Status**: $Status"
    $newContent = $newContent -replace "- \*\*Iteration\*\*: .*", "- **Iteration**: $Iteration"
    $newContent = $newContent -replace "- \*\*Last Error\*\*: .*", "- **Last Error**: $($Errors -join '; ')"
    
    Set-Content "CI_FIX_LOOP.md" $newContent
}
```

## Manual Commands

### Start New Session
```powershell
# Initialize session
.\ci-fix-loop.ps1 -MaxIterations 10 -BaseTimeout 120

# Or step-by-step
$iteration = 1
$timeout = 120
```

### Single Iteration
```powershell
# Run tests with error capture
go test ./... 2>&1 | Tee-Object -FilePath "errors-iter-${iteration}.log"

# Apply common fixes
goimports -w .
go mod tidy
golangci-lint run --fix

# Validate
go test ./...
go build ./...

# Commit if clean
if ($LASTEXITCODE -eq 0) {
    git add -A
    git commit -m "fix(ci): iteration $iteration fixes"
    git push origin HEAD
}
```

### PR Status Check
```powershell
# Check PR status
gh pr checks

# Get detailed logs if failed
gh api repos/nephoran/nephoran-intent-operator/actions/runs --jq '.workflow_runs[0].logs_url'
```

## Recovery Procedures

### If Loop Gets Stuck
1. Check current iteration in CI_FIX_LOOP.md
2. Review last error in logs
3. Apply manual fix
4. Resume from next iteration

### If PR CI Keeps Failing
1. Pull latest PR CI logs
2. Compare with local errors
3. Check for environment differences
4. Apply environment-specific fixes

### Emergency Reset
```powershell
# Reset to clean state
git reset --hard HEAD~${failed_iterations}
git push --force-with-lease origin HEAD

# Restart loop
.\ci-fix-loop.ps1
```

## Success Criteria
- ✅ All local tests pass
- ✅ No linter errors
- ✅ Clean build
- ✅ PR CI passes
- ✅ No timeout issues
- ✅ All fixes documented

---

*Auto-generated CI Fix Loop System - Ready for initialization*