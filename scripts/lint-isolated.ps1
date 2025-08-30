#!/usr/bin/env pwsh
# Script for running specific golangci-lint rules in isolation
# Usage: .\lint-isolated.ps1 <linter> [path] [additional-args]

param(
    [Parameter(Mandatory=$true)]
    [string]$Linter,
    [string]$Path = ".",
    [string[]]$AdditionalArgs = @()
)

# Configuration
$ErrorActionPreference = "Stop"
$VerbosePreference = "Continue"

# Available linters from .golangci.yml
$ValidLinters = @(
    "revive", "staticcheck", "govet", "ineffassign", "errcheck", 
    "gocritic", "misspell", "unparam", "unconvert", "prealloc", "gosec"
)

# Validation
if ($Linter -notin $ValidLinters) {
    Write-Error "Invalid linter: $Linter. Valid options: $($ValidLinters -join ', ')"
    exit 1
}

# Build command
$cmd = @("golangci-lint", "run")
$cmd += "--disable-all"
$cmd += "--enable", $Linter
$cmd += "--timeout", "5m"
$cmd += "--out-format", "colored-line-number"
$cmd += "--print-issued-lines"
$cmd += "--print-linter-name"
$cmd += $Path

# Add any additional arguments
$cmd += $AdditionalArgs

Write-Host "Running isolated linter: $Linter" -ForegroundColor Cyan
Write-Host "Command: $($cmd -join ' ')" -ForegroundColor Gray
Write-Host "Path: $Path" -ForegroundColor Gray
Write-Host ""

# Execute command
try {
    $startTime = Get-Date
    & $cmd[0] $cmd[1..($cmd.Length-1)]
    $endTime = Get-Date
    $duration = $endTime - $startTime
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "`n✅ $Linter passed in $($duration.TotalSeconds.ToString('F2')) seconds" -ForegroundColor Green
    } else {
        Write-Host "`n❌ $Linter found issues in $($duration.TotalSeconds.ToString('F2')) seconds" -ForegroundColor Red
    }
    
    exit $LASTEXITCODE
} catch {
    Write-Error "Failed to run golangci-lint: $_"
    exit 1
}