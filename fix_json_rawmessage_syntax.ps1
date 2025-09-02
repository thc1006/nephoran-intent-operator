# Fix json.RawMessage syntax errors across the codebase
# This script fixes malformed json.RawMessage("{}"){ patterns

Write-Host "Fixing json.RawMessage syntax errors..." -ForegroundColor Green

$rootDir = "C:\Users\tingy\dev\_worktrees\nephoran\feat-e2e"
$testFiles = Get-ChildItem -Path $rootDir -Recurse -Filter "*_test.go"

$totalFixed = 0
$totalFiles = 0

foreach ($file in $testFiles) {
    $content = Get-Content -Path $file.FullName -Raw
    $originalContent = $content
    
    # Fix pattern 1: json.RawMessage("{}"){  ->  map[string]interface{}{
    $content = $content -replace 'json\.RawMessage\("\{\}"\)\{', 'map[string]interface{}{'
    
    # Fix pattern 2: json.RawMessage("{}"),  where it should be map[string]interface{}
    # This is more complex - we need to look at context
    # For now, let's handle specific patterns
    
    # Fix pattern 3: json.RawMessage("{}"){"count": 3},  ->  map[string]interface{}{"count": 3},
    $content = $content -replace 'json\.RawMessage\("\{\}"\)\{([^}]+)\}', 'map[string]interface{}{$1}'
    
    if ($content -ne $originalContent) {
        Set-Content -Path $file.FullName -Value $content -NoNewline
        $totalFiles++
        
        # Count how many patterns were fixed in this file
        $matches1 = [regex]::Matches($originalContent, 'json\.RawMessage\("\{\}"\)\{')
        $matches2 = [regex]::Matches($originalContent, 'json\.RawMessage\("\{\}"\)\{[^}]+\}')
        $fixesInFile = $matches1.Count + $matches2.Count
        $totalFixed += $fixesInFile
        
        Write-Host "Fixed $fixesInFile json.RawMessage patterns in: $($file.FullName -replace [regex]::Escape($rootDir), '')" -ForegroundColor Yellow
    }
}

Write-Host "`nSummary:" -ForegroundColor Green
Write-Host "- Files modified: $totalFiles" -ForegroundColor White
Write-Host "- json.RawMessage patterns fixed: $totalFixed" -ForegroundColor White