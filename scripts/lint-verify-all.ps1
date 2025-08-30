#!/usr/bin/env pwsh
# Comprehensive verification script to ensure all linter rules pass consistently
# Usage: .\lint-verify-all.ps1 [options]

param(
    [string]$ConfigFile = "./.golangci.yml",
    [string]$TargetPath = "./...",
    [int]$Timeout = 600,  # 10 minutes
    [switch]$Verbose = $false,
    [switch]$GenerateReport = $true,
    [switch]$FailFast = $false,
    [switch]$CacheResults = $true,
    [string]$OutputFormat = "colored-line-number"
)

$ErrorActionPreference = "Continue"

# Configuration
$Colors = @{
    Success = "Green"
    Error = "Red"
    Warning = "Yellow"
    Info = "Cyan"
    Debug = "Gray"
    Header = "Magenta"
}

function Write-ColorOutput {
    param([string]$Message, [string]$Color = "White")
    Write-Host $Message -ForegroundColor $Colors.$Color
}

function Test-Prerequisites {
    $missing = @()
    
    if (-not (Get-Command "golangci-lint" -ErrorAction SilentlyContinue)) {
        $missing += "golangci-lint"
    }
    
    if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
        $missing += "go"
    }
    
    if (-not (Test-Path $ConfigFile)) {
        $missing += "golangci config file at $ConfigFile"
    }
    
    return $missing
}

function Get-EnabledLinters {
    param([string]$ConfigPath)
    
    try {
        $configContent = Get-Content $ConfigPath -Raw
        if ($configContent -match "enable:\s*\n((?:\s*-\s*\w+\s*\n?)+)") {
            $linterSection = $matches[1]
            $linters = @()
            $linterSection -split "`n" | ForEach-Object {
                if ($_ -match "\s*-\s*(\w+)") {
                    $linters += $matches[1]
                }
            }
            return $linters
        }
    } catch {
        Write-ColorOutput "Warning: Could not parse config file, using default linters" "Warning"
    }
    
    # Default linters from our config
    return @("revive", "staticcheck", "govet", "ineffassign", "errcheck", "gocritic", "misspell", "unparam", "unconvert", "prealloc", "gosec")
}

function Invoke-LinterVerification {
    param(
        [string]$LinterName,
        [string]$Path,
        [int]$TimeoutSeconds,
        [string]$Format
    )
    
    $startTime = Get-Date
    
    try {
        $cmd = @(
            "golangci-lint", "run",
            "--disable-all",
            "--enable", $LinterName,
            "--timeout", "${TimeoutSeconds}s",
            "--out-format", $Format
        )
        
        if (-not $CacheResults) {
            $cmd += "--no-cache"
        }
        
        $cmd += $Path
        
        if ($Verbose) {
            Write-ColorOutput "Command: $($cmd -join ' ')" "Debug"
        }
        
        $output = & $cmd[0] $cmd[1..($cmd.Length-1)] 2>&1
        $exitCode = $LASTEXITCODE
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        
        # Parse issues count
        $issueCount = 0
        if ($output -is [array]) {
            $issueCount = @($output | Where-Object { $_ -match ':(line \d+|col \d+)' }).Count
        }
        
        return @{
            Linter = $LinterName
            Success = ($exitCode -eq 0)
            ExitCode = $exitCode
            Duration = $duration
            Output = if ($output) { $output -join "`n" } else { "" }
            IssueCount = $issueCount
        }
        
    } catch {
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        
        return @{
            Linter = $LinterName
            Success = $false
            ExitCode = 1
            Duration = $duration
            Output = "Error: $_"
            IssueCount = -1
        }
    }
}

function Invoke-FullLintCheck {
    param([string]$Path, [int]$TimeoutSeconds, [string]$Format)
    
    $startTime = Get-Date
    
    try {
        $cmd = @(
            "golangci-lint", "run",
            "--timeout", "${TimeoutSeconds}s",
            "--out-format", $Format
        )
        
        if (-not $CacheResults) {
            $cmd += "--no-cache"
        }
        
        $cmd += $Path
        
        if ($Verbose) {
            Write-ColorOutput "Full check command: $($cmd -join ' ')" "Debug"
        }
        
        $output = & $cmd[0] $cmd[1..($cmd.Length-1)] 2>&1
        $exitCode = $LASTEXITCODE
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        
        return @{
            Success = ($exitCode -eq 0)
            ExitCode = $exitCode
            Duration = $duration
            Output = if ($output) { $output -join "`n" } else { "" }
        }
        
    } catch {
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        
        return @{
            Success = $false
            ExitCode = 1
            Duration = $duration
            Output = "Error: $_"
        }
    }
}

# Start verification
Write-ColorOutput "🔍 GOLANGCI-LINT COMPREHENSIVE VERIFICATION" "Header"
Write-ColorOutput "=" * 50 "Header"
Write-ColorOutput "Config: $ConfigFile" "Debug"
Write-ColorOutput "Target: $TargetPath" "Debug"
Write-ColorOutput "Timeout: $Timeout seconds" "Debug"
Write-ColorOutput "Output format: $OutputFormat" "Debug"
Write-ColorOutput ""

# Check prerequisites
Write-ColorOutput "📋 Checking prerequisites..." "Info"
$missing = Test-Prerequisites
if ($missing.Count -gt 0) {
    Write-ColorOutput "Missing prerequisites:" "Error"
    $missing | ForEach-Object { Write-ColorOutput "  - $_" "Error" }
    exit 1
}
Write-ColorOutput "  ✅ All prerequisites satisfied" "Success"
Write-ColorOutput ""

# Get enabled linters
Write-ColorOutput "🔧 Reading linter configuration..." "Info"
$enabledLinters = Get-EnabledLinters -ConfigPath $ConfigFile
Write-ColorOutput "  Enabled linters: $($enabledLinters.Count)" "Debug"
if ($Verbose) {
    $enabledLinters | ForEach-Object { Write-ColorOutput "    - $_" "Debug" }
}
Write-ColorOutput ""

# Individual linter verification
Write-ColorOutput "🧪 Running individual linter verification..." "Info"
$linterResults = @()
$passedCount = 0
$failedCount = 0
$totalIssues = 0

foreach ($linter in $enabledLinters) {
    Write-ColorOutput "Testing $linter..." "Debug"
    
    $result = Invoke-LinterVerification -LinterName $linter -Path $TargetPath -TimeoutSeconds $Timeout -Format $OutputFormat
    $linterResults += $result
    
    if ($result.Success) {
        Write-ColorOutput "  ✅ $linter passed ($($result.Duration.ToString('F2'))s)" "Success"
        $passedCount++
    } else {
        Write-ColorOutput "  ❌ $linter failed ($($result.Duration.ToString('F2'))s, $($result.IssueCount) issues)" "Error"
        $failedCount++
        $totalIssues += $result.IssueCount
        
        if ($Verbose) {
            $result.Output -split "`n" | Select-Object -First 5 | ForEach-Object {
                Write-ColorOutput "    $_" "Warning"
            }
        }
        
        if ($FailFast) {
            Write-ColorOutput "Fail fast enabled, stopping verification" "Error"
            exit 1
        }
    }
}

Write-ColorOutput ""

# Full configuration check
Write-ColorOutput "🔄 Running full configuration check..." "Info"
$fullResult = Invoke-FullLintCheck -Path $TargetPath -TimeoutSeconds $Timeout -Format $OutputFormat

if ($fullResult.Success) {
    Write-ColorOutput "  ✅ Full configuration check passed ($($fullResult.Duration.ToString('F2'))s)" "Success"
} else {
    Write-ColorOutput "  ❌ Full configuration check failed ($($fullResult.Duration.ToString('F2'))s)" "Error"
    if ($Verbose -and $fullResult.Output) {
        Write-ColorOutput "Full check output:" "Warning"
        $fullResult.Output -split "`n" | Select-Object -First 10 | ForEach-Object {
            Write-ColorOutput "  $_" "Warning"
        }
    }
}

Write-ColorOutput ""

# Summary
Write-ColorOutput "📊 VERIFICATION SUMMARY" "Header"
Write-ColorOutput "=" * 30 "Header"

$overallSuccess = ($failedCount -eq 0 -and $fullResult.Success)
$totalDuration = ($linterResults | Measure-Object -Property Duration -Sum).Sum + $fullResult.Duration

Write-ColorOutput "Individual Linters:" "Info"
Write-ColorOutput "  ✅ Passed: $passedCount" "Success"
Write-ColorOutput "  ❌ Failed: $failedCount" "Error"
Write-ColorOutput "  📊 Total issues: $totalIssues" "Debug"

Write-ColorOutput "Full Configuration:" "Info"
Write-ColorOutput "  $(if ($fullResult.Success) { '✅ Passed' } else { '❌ Failed' })" $(if ($fullResult.Success) { "Success" } else { "Error" })

Write-ColorOutput "Performance:" "Info"
Write-ColorOutput "  Total duration: $($totalDuration.ToString('F2'))s" "Debug"
Write-ColorOutput "  Average per linter: $(($totalDuration / $enabledLinters.Count).ToString('F2'))s" "Debug"

Write-ColorOutput "Overall Status: $(if ($overallSuccess) { '✅ ALL CHECKS PASSED' } else { '❌ SOME CHECKS FAILED' })" $(if ($overallSuccess) { "Success" } else { "Error" })

# Generate report
if ($GenerateReport) {
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $reportFile = "./test-results/lint-verification-$timestamp.json"
    
    $report = @{
        timestamp = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ssZ")
        success = $overallSuccess
        configuration = @{
            configFile = $ConfigFile
            targetPath = $TargetPath
            timeout = $Timeout
            outputFormat = $OutputFormat
            cacheResults = $CacheResults
        }
        summary = @{
            individualLinters = @{
                passed = $passedCount
                failed = $failedCount
                totalIssues = $totalIssues
            }
            fullConfiguration = @{
                passed = $fullResult.Success
                duration = $fullResult.Duration
            }
            performance = @{
                totalDuration = $totalDuration
                averagePerLinter = $totalDuration / $enabledLinters.Count
            }
        }
        linterResults = $linterResults
        fullResult = $fullResult
    }
    
    try {
        New-Item -Path (Split-Path $reportFile) -ItemType Directory -Force -ErrorAction SilentlyContinue | Out-Null
        $report | ConvertTo-Json -Depth 10 | Out-File -FilePath $reportFile -Encoding UTF8
        Write-ColorOutput "`n📁 Report saved to: $reportFile" "Info"
    } catch {
        Write-ColorOutput "Warning: Failed to save report: $_" "Warning"
    }
}

# Exit with appropriate code
exit $(if ($overallSuccess) { 0 } else { 1 })