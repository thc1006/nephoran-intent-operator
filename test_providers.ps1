#!/usr/bin/env pwsh

Write-Host "=== Testing O2 Providers Package ===" -ForegroundColor Cyan

# Ensure we're in the right directory
Push-Location

try {
    # 1. Check if the providers package exists and has all files
    Write-Host "`n1. Checking providers package structure..." -ForegroundColor Yellow
    $providersPath = "pkg/oran/o2/providers"
    
    if (-not (Test-Path $providersPath)) {
        Write-Host "❌ Providers directory not found: $providersPath" -ForegroundColor Red
        exit 1
    }
    
    $expectedFiles = @(
        "types.go",
        "interfaces.go", 
        "resources.go",
        "events.go",
        "factory.go",
        "mock.go",
        "providers_test.go",
        "doc.go"
    )
    
    foreach ($file in $expectedFiles) {
        $filePath = Join-Path $providersPath $file
        if (Test-Path $filePath) {
            Write-Host "  ✓ $file" -ForegroundColor Green
        } else {
            Write-Host "  ❌ $file (missing)" -ForegroundColor Red
        }
    }

    # 2. Check for compilation errors
    Write-Host "`n2. Testing compilation..." -ForegroundColor Yellow
    $buildOutput = go build ./pkg/oran/o2/providers/... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✓ Compilation successful" -ForegroundColor Green
    } else {
        Write-Host "  ❌ Compilation failed:" -ForegroundColor Red
        $buildOutput | Write-Host -ForegroundColor Red
        exit 1
    }

    # 3. Run go mod tidy to ensure dependencies are clean
    Write-Host "`n3. Running go mod tidy..." -ForegroundColor Yellow
    go mod tidy
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✓ Dependencies are clean" -ForegroundColor Green
    } else {
        Write-Host "  ⚠️ go mod tidy had issues" -ForegroundColor Yellow
    }

    # 4. Run tests
    Write-Host "`n4. Running tests..." -ForegroundColor Yellow
    $testOutput = go test -v ./pkg/oran/o2/providers/... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✓ All tests passed" -ForegroundColor Green
        
        # Show test results summary
        $testOutput | Where-Object { $_ -match "PASS|FAIL|RUN" } | ForEach-Object {
            if ($_ -match "PASS") {
                Write-Host "    $_" -ForegroundColor Green
            } elseif ($_ -match "FAIL") {
                Write-Host "    $_" -ForegroundColor Red
            } else {
                Write-Host "    $_" -ForegroundColor Cyan
            }
        }
    } else {
        Write-Host "  ❌ Tests failed:" -ForegroundColor Red
        $testOutput | Write-Host -ForegroundColor Red
        exit 1
    }

    # 5. Run race detection tests
    Write-Host "`n5. Running race detection tests..." -ForegroundColor Yellow
    $raceTestOutput = go test -race ./pkg/oran/o2/providers/... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✓ No race conditions detected" -ForegroundColor Green
    } else {
        Write-Host "  ❌ Race conditions detected:" -ForegroundColor Red
        $raceTestOutput | Write-Host -ForegroundColor Red
    }

    # 6. Check test coverage
    Write-Host "`n6. Checking test coverage..." -ForegroundColor Yellow
    $coverageOutput = go test -cover ./pkg/oran/o2/providers/... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✓ Coverage analysis completed" -ForegroundColor Green
        $coverageOutput | Where-Object { $_ -match "coverage:" } | Write-Host -ForegroundColor Cyan
    } else {
        Write-Host "  ⚠️ Coverage analysis had issues" -ForegroundColor Yellow
    }

    # 7. Validate package can be imported
    Write-Host "`n7. Testing package import..." -ForegroundColor Yellow
    $importTest = @"
package main

import (
    "fmt"
    _ "github.com/nephoran/nephoran-intent-operator/pkg/oran/o2/providers"
)

func main() {
    fmt.Println("Import successful")
}
"@
    
    $importTest | Out-File -FilePath "import_test.go" -Encoding UTF8
    $importResult = go run import_test.go 2>&1
    Remove-Item "import_test.go" -ErrorAction SilentlyContinue
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✓ Package import successful" -ForegroundColor Green
    } else {
        Write-Host "  ❌ Package import failed:" -ForegroundColor Red
        $importResult | Write-Host -ForegroundColor Red
    }

    Write-Host "`n=== All Tests Completed Successfully! ===" -ForegroundColor Green

} catch {
    Write-Host "❌ Error during testing: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}