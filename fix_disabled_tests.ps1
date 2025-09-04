# Fix all DISABLED test functions across the codebase
# This script removes the DISABLED comment prefix from test functions

Write-Host "Fixing DISABLED test functions..." -ForegroundColor Green

$rootDir = "C:\Users\tingy\dev\_worktrees\nephoran\feat-e2e"
$testFiles = Get-ChildItem -Path $rootDir -Recurse -Filter "*_test.go"

$totalFixed = 0
$totalFiles = 0

foreach ($file in $testFiles) {
    $content = Get-Content -Path $file.FullName -Raw
    $originalContent = $content
    
    # Fix the DISABLED function pattern: "// DISABLED: func" -> "func"
    $content = $content -replace '// DISABLED:\s*func\s+', 'func '
    
    if ($content -ne $originalContent) {
        Set-Content -Path $file.FullName -Value $content -NoNewline
        $totalFiles++
        
        # Count how many functions were fixed in this file
        $matches = [regex]::Matches($originalContent, '// DISABLED:\s*func\s+')
        $totalFixed += $matches.Count
        
        Write-Host "Fixed $($matches.Count) functions in: $($file.FullName -replace [regex]::Escape($rootDir), '')" -ForegroundColor Yellow
    }
}

Write-Host "`nSummary:" -ForegroundColor Green
Write-Host "- Files modified: $totalFiles" -ForegroundColor White
Write-Host "- Functions fixed: $totalFixed" -ForegroundColor White

Write-Host "`nRunning go test to verify fixes..." -ForegroundColor Green