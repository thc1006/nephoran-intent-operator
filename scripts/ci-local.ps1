# ci-local.ps1 - Local CI equivalence for Windows
param(
    [switch]$Fast
)

$ErrorActionPreference = "Stop"

Write-Host "=== CI-LOCAL: Windows version ===" -ForegroundColor Cyan

try {
    if (-not $Fast) {
        Write-Host ">> go mod tidy / vet / build" -ForegroundColor Green
        go mod tidy
        if ($LASTEXITCODE -ne 0) { throw "go mod tidy failed" }
        
        go vet ./...
        if ($LASTEXITCODE -ne 0) { throw "go vet failed" }
        
        go build ./...
        if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    }

    Write-Host ">> unit tests" -ForegroundColor Green
    go test -count=1 ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed" }

    Write-Host ">> golangci-lint (if available)" -ForegroundColor Green
    if (Get-Command golangci-lint -ErrorAction SilentlyContinue) {
        golangci-lint run --timeout=5m
        if ($LASTEXITCODE -ne 0) { throw "golangci-lint failed" }
    } else {
        Write-Host "⚠️ golangci-lint not found, skipping" -ForegroundColor Yellow
    }

    Write-Host "✅ ci-local completed - ready for push" -ForegroundColor Green
    Write-Host "=== CI-LOCAL: SUCCESS ===" -ForegroundColor Cyan

} catch {
    Write-Host "❌ CI-LOCAL FAILED: $_" -ForegroundColor Red
    Write-Host "=== CI-LOCAL: FAILURE ===" -ForegroundColor Red
    exit 1
}