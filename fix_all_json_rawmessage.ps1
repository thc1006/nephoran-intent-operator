# Fix all json.RawMessage syntax errors in the codebase

$rootDir = "C:\Users\tingy\dev\_worktrees\nephoran\feat-e2e"

# Find all .go files with json.RawMessage issues
$files = Get-ChildItem -Path $rootDir -Recurse -Include "*.go" | Where-Object { 
    (Get-Content $_.FullName -Raw) -match 'json\.RawMessage\("\{.*?\}"\)' 
}

Write-Host "Found $($files.Count) files with json.RawMessage syntax issues"

foreach ($file in $files) {
    Write-Host "Fixing: $($file.FullName)"
    
    $content = Get-Content $file.FullName -Raw
    
    # Replace json.RawMessage("{}") with json.RawMessage(`{}`)
    $content = $content -replace 'json\.RawMessage\("(\{[^}]*\})"\)', 'json.RawMessage(`$1`)'
    
    # Handle special cases like json.RawMessage("{}"){
    $content = $content -replace 'json\.RawMessage\("(\{[^}]*\})"\)\{', 'json.RawMessage(`$1`){' 
    
    # Handle cases like []json.RawMessage("{}"),
    $content = $content -replace '\[\]json\.RawMessage\("(\{[^}]*\})"\)', '[]json.RawMessage(`$1`)'
    
    # Handle cases with extra quote marks like json.RawMessage("{}"))
    $content = $content -replace 'json\.RawMessage\("(\{[^}]*\})"\)\)', 'json.RawMessage(`$1`))'
    
    # Handle cases like json.RawMessage("{}"),
    $content = $content -replace 'json\.RawMessage\("(\{[^}]*\})"\)\,', 'json.RawMessage(`$1`),'
    
    Set-Content $file.FullName -Value $content
}

Write-Host "All json.RawMessage syntax issues fixed!"