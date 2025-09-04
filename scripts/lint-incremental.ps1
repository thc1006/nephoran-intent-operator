#!/usr/bin/env pwsh
# Script for incremental linting of individual files or small file sets
# Usage: .\lint-incremental.ps1 <file-pattern> [linter] [options]

param(
    [Parameter(Mandatory=$true)]
    [string]$FilePattern,
    [string]$Linter = "",
    [switch]$Verbose = $false,
    [switch]$Fix = $false,
    [switch]$JsonOutput = $false
)

$ErrorActionPreference = "Stop"

# Available linters
$AllLinters = @(
    "revive", "staticcheck", "govet", "ineffassign", "errcheck", 
    "gocritic", "misspell", "unparam", "unconvert", "prealloc", "gosec"
)

# Resolve file pattern to actual files
try {
    $Files = @(Get-ChildItem -Path $FilePattern -Recurse -File -Include "*.go" | Where-Object { 
        $_.FullName -notmatch "_test\.go$|_mock\.go$|\.mock\.go$|_generated\.go$|\.pb\.go$|zz_generated\."
    })
    
    if ($Files.Count -eq 0) {
        Write-Host "No Go files found matching pattern: $FilePattern" -ForegroundColor Yellow
        exit 0
    }
} catch {
    Write-Error "Failed to resolve file pattern: $FilePattern"
    exit 1
}

Write-Host "Found $($Files.Count) Go files to lint" -ForegroundColor Cyan
if ($Verbose) {
    $Files | ForEach-Object { Write-Host "  - $($_.FullName)" -ForegroundColor Gray }
}
Write-Host ""

# Determine which linters to run
$LintersToRun = if ($Linter) { @($Linter) } else { $AllLinters }

# Validate linter names
foreach ($l in $LintersToRun) {
    if ($l -notin $AllLinters) {
        Write-Error "Invalid linter: $l. Valid options: $($AllLinters -join ', ')"
        exit 1
    }
}

$Results = @{}
$OverallSuccess = $true

foreach ($linter in $LintersToRun) {
    Write-Host "Running $linter on $($Files.Count) files..." -ForegroundColor Yellow
    
    $startTime = Get-Date
    
    # Build golangci-lint command for specific files
    $cmd = @(
        "golangci-lint", "run",
        "--disable-all",
        "--enable", $linter,
        "--timeout", "2m"
    )
    
    if ($JsonOutput) {
        $cmd += "--out-format", "json"
    } else {
        $cmd += "--out-format", "colored-line-number"
        $cmd += "--print-issued-lines"
        $cmd += "--print-linter-name"
    }
    
    if ($Fix) {
        $cmd += "--fix"
    }
    
    # Add file paths
    foreach ($file in $Files) {
        $cmd += $file.FullName
    }
    
    try {
        if ($Verbose) {
            Write-Host "Command: $($cmd -join ' ')" -ForegroundColor Gray
        }
        
        $output = & $cmd[0] $cmd[1..($cmd.Length-1)] 2>&1
        $exitCode = $LASTEXITCODE
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        
        $Results[$linter] = @{
            ExitCode = $exitCode
            Duration = $duration
            Output = $output -join "`n"
            Success = ($exitCode -eq 0)
            FilesProcessed = $Files.Count
        }
        
        if ($exitCode -eq 0) {
            Write-Host "  ✅ $linter passed ($($duration.ToString('F2'))s)" -ForegroundColor Green
        } else {
            Write-Host "  ❌ $linter found issues ($($duration.ToString('F2'))s)" -ForegroundColor Red
            $OverallSuccess = $false
            
            if (-not $JsonOutput -and $output) {
                Write-Host "  Issues found:" -ForegroundColor Red
                $output | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
            }
        }
        
    } catch {
        Write-Host "  ❌ $linter failed with error: $_" -ForegroundColor Red
        $Results[$linter] = @{
            ExitCode = 1
            Duration = 0
            Output = "Error: $_"
            Success = $false
            FilesProcessed = $Files.Count
        }
        $OverallSuccess = $false
    }
}

# Summary
Write-Host "`n" + "="*50 -ForegroundColor Cyan
Write-Host "INCREMENTAL LINT SUMMARY" -ForegroundColor Cyan
Write-Host "="*50 -ForegroundColor Cyan
Write-Host "Files processed: $($Files.Count)" -ForegroundColor Gray
Write-Host "Pattern: $FilePattern" -ForegroundColor Gray
Write-Host ""

$passedCount = 0
$failedCount = 0

foreach ($linter in $LintersToRun) {
    if ($Results[$linter].Success) {
        Write-Host "✅ $linter ($(($Results[$linter].Duration).ToString('F2'))s)" -ForegroundColor Green
        $passedCount++
    } else {
        Write-Host "❌ $linter ($(($Results[$linter].Duration).ToString('F2'))s)" -ForegroundColor Red
        $failedCount++
    }
}

Write-Host ""
Write-Host "Passed: $passedCount, Failed: $failedCount" -ForegroundColor $(if ($OverallSuccess) { "Green" } else { "Red" })

# JSON output if requested
if ($JsonOutput) {
    $jsonResult = @{
        success = $OverallSuccess
        filesProcessed = $Files.Count
        pattern = $FilePattern
        results = $Results
        summary = @{
            passed = $passedCount
            failed = $failedCount
        }
    }
    
    Write-Host "`nJSON Results:" -ForegroundColor Cyan
    $jsonResult | ConvertTo-Json -Depth 5
}

exit $(if ($OverallSuccess) { 0 } else { 1 })