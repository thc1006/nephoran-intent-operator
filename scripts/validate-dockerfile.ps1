#!/usr/bin/env pwsh
# Dockerfile Validation Script (2025)
# Validates Docker build configuration and stage targets

param(
    [string]$DockerfilePath = "Dockerfile.fast-2025",
    [string]$Service = "intent-ingest", 
    [string]$Target = "runtime"
)

Write-Host "=== Dockerfile Validation Script ===" -ForegroundColor Cyan
Write-Host "Dockerfile: $DockerfilePath" -ForegroundColor Yellow
Write-Host "Service: $Service" -ForegroundColor Yellow
Write-Host "Target: $Target" -ForegroundColor Yellow
Write-Host

# Check if Dockerfile exists
if (-not (Test-Path $DockerfilePath)) {
    Write-Host "ERROR: Dockerfile not found: $DockerfilePath" -ForegroundColor Red
    exit 1
}

Write-Host "OK: Dockerfile exists: $DockerfilePath" -ForegroundColor Green

# Parse Dockerfile for stages
$stages = @()
$content = Get-Content $DockerfilePath
foreach ($line in $content) {
    if ($line -match '^FROM .* AS (\w+)') {
        $stages += $matches[1]
    }
}

Write-Host "Found stages in ${DockerfilePath}:" -ForegroundColor Cyan
foreach ($stage in $stages) {
    if ($stage -eq $Target) {
        Write-Host "  [TARGET] $stage" -ForegroundColor Green
    } else {
        Write-Host "  [STAGE] $stage" -ForegroundColor White
    }
}

# Validate target exists
if ($Target -in $stages) {
    Write-Host "OK: Target stage '$Target' found in Dockerfile" -ForegroundColor Green
} else {
    Write-Host "ERROR: Target stage '$Target' not found in Dockerfile" -ForegroundColor Red
    Write-Host "Available stages: $($stages -join ', ')" -ForegroundColor Yellow
    exit 1
}

Write-Host
Write-Host "Validation completed for $DockerfilePath" -ForegroundColor Green
Write-Host "Ready for CI/CD build with target: $Target" -ForegroundColor Cyan