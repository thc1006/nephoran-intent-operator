# ULTRA SPEED COMPREHENSIVE LINTING - IMMEDIATE EXECUTION
# This script provides the optimal linting command and issue analysis

Write-Host "=== NEPHORAN ULTRA SPEED COMPREHENSIVE LINTING ===" -ForegroundColor Cyan
Write-Host "Optimized for Windows Git Bash with Go 1.24.x" -ForegroundColor Gray

# Ensure PATH includes Go binaries
$env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"

# Check golangci-lint installation
Write-Host "`nChecking golangci-lint installation..." -ForegroundColor Yellow
$lintVersion = & golangci-lint version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "Installing golangci-lint..." -ForegroundColor Yellow
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    $lintVersion = & golangci-lint version
}
Write-Host "Found: $lintVersion" -ForegroundColor Green

# Create output directory
New-Item -ItemType Directory -Force -Path "test-results" | Out-Null

Write-Host "`n=== PHASE 1: COMPILATION STATUS CHECK ===" -ForegroundColor Cyan
Write-Host "Checking if Go code compiles..." -ForegroundColor Yellow

# Quick compilation check
$compileResult = go build ./... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Compilation successful - can run full linting" -ForegroundColor Green
    $canRunFullLint = $true
} else {
    Write-Host "❌ Compilation errors found - running limited linting" -ForegroundColor Red
    Write-Host "Top 10 compilation errors:" -ForegroundColor Yellow
    $compileResult | Select-Object -First 10 | ForEach-Object { Write-Host "  $_" -ForegroundColor White }
    $canRunFullLint = $false
}

Write-Host "`n=== PHASE 2: OPTIMIZED LINTING EXECUTION ===" -ForegroundColor Cyan

if ($canRunFullLint) {
    Write-Host "Running comprehensive linting analysis..." -ForegroundColor Yellow
    
    # Full comprehensive command
    $lintCommand = @"
golangci-lint run \
  --enable-all \
  --disable=govet,exhaustive,gochecknoglobals,gochecknoinits,gomnd,dupl,funlen,gocyclo,cyclop,maintidx,lll,wsl \
  --timeout=15m \
  --issues-exit-code=0 \
  --max-issues-per-linter=0 \
  --max-same-issues=0 \
  --out-format=colored-line-number,json:test-results/comprehensive-issues.json,junit-xml:test-results/comprehensive-issues.xml \
  --sort-results \
  --print-issued-lines \
  --print-linter-name \
  --concurrency=0 \
  ./...
"@
    
    Invoke-Expression $lintCommand.Replace("`\", "")
    
} else {
    Write-Host "Running compilation-independent linting..." -ForegroundColor Yellow
    
    # Limited command that works without full compilation
    $limitedCommand = @"
golangci-lint run \
  --disable-all \
  --enable=misspell,gofumpt,gci,unconvert,predeclared \
  --timeout=5m \
  --issues-exit-code=0 \
  --max-issues-per-linter=0 \
  --out-format=colored-line-number,json:test-results/limited-issues.json \
  --sort-results \
  ./...
"@
    
    Invoke-Expression $limitedCommand.Replace("`\", "")
}

Write-Host "`n=== PHASE 3: ISSUE ANALYSIS ===" -ForegroundColor Cyan

# Analyze results if JSON was generated
$jsonFiles = @("test-results/comprehensive-issues.json", "test-results/limited-issues.json")
$jsonFile = $jsonFiles | Where-Object { Test-Path $_ } | Select-Object -First 1

if ($jsonFile) {
    Write-Host "Analyzing issues from: $jsonFile" -ForegroundColor Yellow
    
    try {
        $jsonContent = Get-Content $jsonFile | ConvertFrom-Json
        
        if ($jsonContent.Issues -and $jsonContent.Issues.Count -gt 0) {
            # Group by linter
            $issuesByLinter = $jsonContent.Issues | Group-Object FromLinter | Sort-Object Count -Descending
            
            Write-Host "`nISSUES BY LINTER:" -ForegroundColor Yellow
            foreach ($group in $issuesByLinter) {
                $color = if ($group.Count -gt 50) { "Red" } elseif ($group.Count -gt 10) { "Yellow" } else { "White" }
                Write-Host "  $($group.Name): $($group.Count)" -ForegroundColor $color
            }
            
            # Top problematic files
            Write-Host "`nTOP 15 FILES WITH ISSUES:" -ForegroundColor Yellow
            $issuesByFile = $jsonContent.Issues | Group-Object {$_.Pos.Filename} | Sort-Object Count -Descending | Select-Object -First 15
            foreach ($group in $issuesByFile) {
                $filename = Split-Path $group.Name -Leaf
                $color = if ($group.Count -gt 10) { "Red" } elseif ($group.Count -gt 5) { "Yellow" } else { "White" }
                Write-Host "  $filename`: $($group.Count)" -ForegroundColor $color
            }
            
            # Common patterns
            Write-Host "`nMOST COMMON ISSUES (Top 10):" -ForegroundColor Yellow
            $issuesByText = $jsonContent.Issues | Group-Object Text | Sort-Object Count -Descending | Select-Object -First 10
            foreach ($group in $issuesByText) {
                $shortText = if ($group.Name.Length -gt 80) { $group.Name.Substring(0, 77) + "..." } else { $group.Name }
                Write-Host "  [$($group.Count)x] $shortText" -ForegroundColor White
            }
            
            Write-Host "`n📊 TOTAL ISSUES FOUND: $($jsonContent.Issues.Count)" -ForegroundColor $(if ($jsonContent.Issues.Count -gt 100) { "Red" } else { "Yellow" })
            
        } else {
            Write-Host "✅ No linting issues found!" -ForegroundColor Green
        }
    }
    catch {
        Write-Host "Could not parse JSON results: $_" -ForegroundColor Red
    }
} else {
    Write-Host "No JSON output generated - check for errors above" -ForegroundColor Red
}

Write-Host "`n=== PHASE 4: NEXT STEPS RECOMMENDATIONS ===" -ForegroundColor Cyan

if (-not $canRunFullLint) {
    Write-Host "🔧 CRITICAL: Fix compilation errors first!" -ForegroundColor Red
    Write-Host "   1. Review compilation errors above" -ForegroundColor White
    Write-Host "   2. Fix constructor signature mismatches" -ForegroundColor White
    Write-Host "   3. Add missing method definitions" -ForegroundColor White
    Write-Host "   4. Fix struct field references" -ForegroundColor White
    Write-Host "   5. Re-run this script after fixes" -ForegroundColor White
} else {
    Write-Host "🎯 READY FOR SYSTEMATIC FIXES!" -ForegroundColor Green
    Write-Host "   1. Review JSON output in test-results/" -ForegroundColor White
    Write-Host "   2. Apply bulk fixes for common patterns" -ForegroundColor White
    Write-Host "   3. Use automated fixes where possible" -ForegroundColor White
    Write-Host "   4. Focus on high-count linter violations first" -ForegroundColor White
}

Write-Host "`n=== OUTPUT FILES GENERATED ===" -ForegroundColor Green
Get-ChildItem "test-results/" -Filter "*issues*" | ForEach-Object {
    Write-Host "📄 $($_.FullName)" -ForegroundColor White
}

Write-Host "`n=== ULTRA SPEED MODE COMPLETE ===" -ForegroundColor Cyan
Write-Host "Use the generated files for bulk analysis and systematic fixes!" -ForegroundColor Green