#!/usr/bin/env pwsh

Write-Host "=== O2 Package Debug Analysis ===" -ForegroundColor Cyan

# 1. Check directory structure
Write-Host "`n1. Directory structure check:" -ForegroundColor Yellow
$paths = @("pkg", "pkg/oran", "pkg/oran/o2", "pkg/oran/o2/providers")
foreach ($path in $paths) {
    if (Test-Path $path) {
        Write-Host "  ✓ $path" -ForegroundColor Green
        if ($path -eq "pkg/oran/o2") {
            Write-Host "    Contents of pkg/oran/o2/:"
            Get-ChildItem -Path $path | ForEach-Object {
                Write-Host "      - $($_.Name) ($($_.Mode))"
            }
        }
    } else {
        Write-Host "  ✗ $path (missing)" -ForegroundColor Red
    }
}

# 2. Find all Go files related to O2
Write-Host "`n2. All O2-related Go files:" -ForegroundColor Yellow
if (Test-Path "pkg/oran/o2") {
    Get-ChildItem -Path "pkg/oran/o2" -Recurse -Filter "*.go" | ForEach-Object {
        $relativePath = $_.FullName.Replace((Get-Location).Path + "\", "")
        Write-Host "  - $relativePath" -ForegroundColor Cyan
    }
} else {
    Write-Host "  No O2 directory found" -ForegroundColor Red
}

# 3. Try to compile and capture specific errors
Write-Host "`n3. Compilation attempt:" -ForegroundColor Yellow
try {
    $output = go build -x ./pkg/oran/o2/... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✓ Compilation successful" -ForegroundColor Green
    } else {
        Write-Host "  ✗ Compilation failed:" -ForegroundColor Red
        $output | ForEach-Object {
            if ($_ -match "error|undefined|cannot|missing") {
                Write-Host "    ERROR: $_" -ForegroundColor Red
            }
        }
    }
} catch {
    Write-Host "  Exception during compilation: $($_.Exception.Message)" -ForegroundColor Red
}

# 4. Check for provider interface definitions anywhere in the codebase
Write-Host "`n4. Searching for provider interfaces:" -ForegroundColor Yellow
$providerFiles = Get-ChildItem -Recurse -Filter "*.go" | Select-String -Pattern "(interface.*Provider|Provider.*interface)" -List
if ($providerFiles) {
    Write-Host "  Found provider interfaces in:"
    $providerFiles | ForEach-Object {
        Write-Host "    - $($_.Filename)" -ForegroundColor Cyan
    }
} else {
    Write-Host "  No provider interfaces found" -ForegroundColor Yellow
}

Write-Host "`n=== Debug Complete ===" -ForegroundColor Cyan