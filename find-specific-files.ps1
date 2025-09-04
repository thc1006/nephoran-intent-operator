# Look for the specific files mentioned in the error
$targetFiles = @("types.go", "sla_monitoring_architecture.go")

Write-Host "=== Looking for target files ==="
foreach ($file in $targetFiles) {
    Write-Host "`n--- Searching for: $file ---"
    $found = Get-ChildItem -Recurse -Filter $file -ErrorAction SilentlyContinue
    if ($found) {
        foreach ($f in $found) {
            Write-Host "Found: $($f.FullName)"
            Write-Host "Content preview:"
            Get-Content $f.FullName -Head 10 | ForEach-Object { Write-Host "  $_" }
        }
    } else {
        Write-Host "File not found: $file"
    }
}

Write-Host "`n=== All Go files in repository ==="
Get-ChildItem -Recurse -Filter "*.go" | Select-Object Name, Directory | Sort-Object Name