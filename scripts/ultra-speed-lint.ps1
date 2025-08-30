# ULTRA SPEED COMPREHENSIVE LINTING SCRIPT
# Designed to find ALL remaining issues for bulk fixing

Write-Host "=== ULTRA SPEED COMPREHENSIVE LINTING ===" -ForegroundColor Cyan

# Ensure test-results directory exists
New-Item -ItemType Directory -Force -Path "test-results" | Out-Null

# Core comprehensive lint command with maximum verbosity
$lintCommand = @"
golangci-lint run `
    --config=.golangci-thorough.yml `
    --timeout=15m `
    --verbose `
    --print-stats `
    --print-resources-usage `
    --issues-exit-code=0 `
    --max-issues-per-linter=0 `
    --max-same-issues=0 `
    --whole-files `
    --show-stats `
    --out-format=colored-line-number,json:test-results/golangci-comprehensive.json,junit-xml:test-results/golangci-comprehensive.xml,code-climate:test-results/golangci-comprehensive-climate.json `
    --sort-results `
    --print-issued-lines `
    --print-linter-name `
    --allow-parallel-runners `
    --concurrency=0 `
    ./...
"@

Write-Host "Running comprehensive lint analysis..." -ForegroundColor Yellow
Write-Host "Command: $lintCommand" -ForegroundColor Gray

# Execute the command
Invoke-Expression $lintCommand

# Parse JSON output for categorization
if (Test-Path "test-results/golangci-comprehensive.json") {
    Write-Host "`n=== ISSUE CATEGORIZATION ===" -ForegroundColor Cyan
    
    $jsonContent = Get-Content "test-results/golangci-comprehensive.json" | ConvertFrom-Json
    
    # Group by linter
    $issuesByLinter = $jsonContent.Issues | Group-Object FromLinter | Sort-Object Count -Descending
    
    Write-Host "Issues by Linter:" -ForegroundColor Yellow
    foreach ($group in $issuesByLinter) {
        Write-Host "  $($group.Name): $($group.Count)" -ForegroundColor White
    }
    
    # Group by file pattern
    Write-Host "`nTop 20 Files with Issues:" -ForegroundColor Yellow
    $issuesByFile = $jsonContent.Issues | Group-Object {$_.Pos.Filename} | Sort-Object Count -Descending | Select-Object -First 20
    foreach ($group in $issuesByFile) {
        $filename = Split-Path $group.Name -Leaf
        Write-Host "  $filename`: $($group.Count)" -ForegroundColor White
    }
    
    # Common issue patterns
    Write-Host "`nCommon Issue Patterns:" -ForegroundColor Yellow
    $issuesByText = $jsonContent.Issues | Group-Object Text | Sort-Object Count -Descending | Select-Object -First 15
    foreach ($group in $issuesByText) {
        $shortText = if ($group.Name.Length -gt 80) { $group.Name.Substring(0, 77) + "..." } else { $group.Name }
        Write-Host "  [$($group.Count)x] $shortText" -ForegroundColor White
    }
    
    Write-Host "`nTotal Issues Found: $($jsonContent.Issues.Count)" -ForegroundColor Red
}

Write-Host "`n=== OUTPUT FILES GENERATED ===" -ForegroundColor Green
Write-Host "- test-results/golangci-comprehensive.json (structured data)" -ForegroundColor White
Write-Host "- test-results/golangci-comprehensive.xml (JUnit format)" -ForegroundColor White  
Write-Host "- test-results/golangci-comprehensive-climate.json (Code Climate)" -ForegroundColor White
Write-Host "`nUse these files for bulk analysis and automated fixes." -ForegroundColor Cyan