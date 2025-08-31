# Check pkg directory structure
Write-Host "=== Current directory ==="
Get-Location

Write-Host "`n=== Directory tree ==="
Get-ChildItem -Recurse -Directory | Select-Object FullName

Write-Host "`n=== Go files containing 'monitoring' ==="
Get-ChildItem -Recurse -Filter "*.go" | Where-Object { $_.Name -like "*monitoring*" -or (Get-Content $_.FullName -Raw -ErrorAction SilentlyContinue) -like "*monitoring*" } | Select-Object FullName

Write-Host "`n=== All Go files in pkg ==="
Get-ChildItem -Path "pkg" -Recurse -Filter "*.go" -ErrorAction SilentlyContinue | Select-Object FullName