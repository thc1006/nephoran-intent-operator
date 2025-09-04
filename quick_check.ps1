Write-Host "=== Quick O2 Providers Check ===" -ForegroundColor Cyan

# Check directory existence and structure
$dirs = @("pkg", "pkg\oran", "pkg\oran\o2", "pkg\oran\o2\providers")
foreach ($dir in $dirs) {
    if (Test-Path $dir) {
        Write-Host "✓ $dir exists" -ForegroundColor Green
    } else {
        Write-Host "✗ $dir missing" -ForegroundColor Red
        Write-Host "Creating directory: $dir" -ForegroundColor Yellow
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
}

# List files in O2 directory if it exists
if (Test-Path "pkg\oran\o2") {
    Write-Host "`nFiles in pkg\oran\o2:" -ForegroundColor Yellow
    Get-ChildItem -Path "pkg\oran\o2" -Recurse | ForEach-Object {
        Write-Host "  $($_.FullName.Replace((Get-Location).Path, '.'))"
    }
}

# Try compilation
Write-Host "`nTesting compilation:" -ForegroundColor Yellow
$result = go build ./pkg/oran/o2/... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ No compilation errors" -ForegroundColor Green
} else {
    Write-Host "✗ Compilation errors found:" -ForegroundColor Red
    $result | Write-Host -ForegroundColor Red
}