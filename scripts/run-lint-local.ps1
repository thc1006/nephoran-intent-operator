# Run golangci-lint locally with exact CI configuration
# Reproduces the exact linting process used in CI

param(
    [switch]$Fast = $false,
    [switch]$Fix = $false,
    [string]$Config = "",
    [int]$Timeout = 15,
    [switch]$Verbose = $false,
    [switch]$ShowStats = $false
)

$ErrorActionPreference = "Stop"

# Determine which config to use
$ConfigFile = if ($Config) {
    $Config
} elseif ($Fast) {
    ".golangci-fast.yml"
} else {
    ".golangci.yml"
}

# Check if config file exists
if (-not (Test-Path $ConfigFile)) {
    Write-Host "❌ Config file not found: $ConfigFile" -ForegroundColor Red
    Write-Host "Available configs:"
    Get-ChildItem -Name ".golangci*.yml" | ForEach-Object { Write-Host "  $_" }
    exit 1
}

Write-Host "🔍 Running golangci-lint with configuration: $ConfigFile" -ForegroundColor Green

# Verify golangci-lint is installed
try {
    $version = & golangci-lint version 2>$null
    Write-Host "✅ Using: $version" -ForegroundColor Green
} catch {
    Write-Host "❌ golangci-lint not found. Please run: .\scripts\install-golangci-lint.ps1" -ForegroundColor Red
    exit 1
}

# Build command arguments
$args = @(
    "run"
    "--config=$ConfigFile"
    "--timeout=$($Timeout)m"
)

if ($Fix) {
    $args += "--fix"
    Write-Host "🔧 Auto-fix mode enabled" -ForegroundColor Yellow
}

if ($Verbose) {
    $args += "--verbose"
}

# Add output formats for detailed reporting
$args += "--out-format=colored-line-number"
$args += "--print-issued-lines"
$args += "--print-linter-name"
$args += "--sort-results"

# Show command being executed
Write-Host "🚀 Executing: golangci-lint $($args -join ' ')" -ForegroundColor Cyan
Write-Host ""

# Record start time
$startTime = Get-Date

# Execute golangci-lint and capture both exit code and output
$output = ""
$exitCode = 0

try {
    # Run the command and capture output
    $process = Start-Process -FilePath "golangci-lint" -ArgumentList $args -NoNewWindow -PassThru -Wait -RedirectStandardOutput "lint-output.tmp" -RedirectStandardError "lint-error.tmp"
    $exitCode = $process.ExitCode
    
    # Read output files
    if (Test-Path "lint-output.tmp") {
        $output = Get-Content "lint-output.tmp" -Raw
    }
    if (Test-Path "lint-error.tmp") {
        $errorOutput = Get-Content "lint-error.tmp" -Raw
        if ($errorOutput) {
            $output += "`n$errorOutput"
        }
    }
    
    # Display output
    if ($output) {
        Write-Host $output
    }
    
} catch {
    Write-Host "❌ Failed to execute golangci-lint: $($_.Exception.Message)" -ForegroundColor Red
    $exitCode = 1
} finally {
    # Clean up temporary files
    Remove-Item "lint-output.tmp" -ErrorAction SilentlyContinue
    Remove-Item "lint-error.tmp" -ErrorAction SilentlyContinue
}

# Calculate execution time
$endTime = Get-Date
$duration = $endTime - $startTime

# Show results
Write-Host ""
Write-Host "===============================================" -ForegroundColor Cyan
Write-Host "🏁 GOLANGCI-LINT RESULTS" -ForegroundColor Cyan
Write-Host "===============================================" -ForegroundColor Cyan
Write-Host "Configuration: $ConfigFile"
Write-Host "Duration: $($duration.TotalSeconds.ToString('F2'))s"
Write-Host "Exit Code: $exitCode"

if ($exitCode -eq 0) {
    Write-Host "✅ No linting issues found!" -ForegroundColor Green
    if ($ShowStats) {
        Write-Host ""
        Write-Host "📊 STATISTICS:"
        & golangci-lint run --config=$ConfigFile --print-linters-list | Select-Object -First 20
    }
} else {
    Write-Host "❌ Linting issues detected (exit code: $exitCode)" -ForegroundColor Red
    
    # Parse and show summary if verbose
    if ($output) {
        $issueCount = ($output -split "`n" | Where-Object { $_ -match "level=|:.*:" }).Count
        Write-Host "📊 Total issues found: $issueCount"
        
        # Show most common linters with issues
        $linterCounts = @{}
        $output -split "`n" | Where-Object { $_ -match "\((\w+)\)$" } | ForEach-Object {
            if ($_ -match "\((\w+)\)$") {
                $linter = $Matches[1]
                $linterCounts[$linter] = ($linterCounts[$linter] ?? 0) + 1
            }
        }
        
        if ($linterCounts.Count -gt 0) {
            Write-Host ""
            Write-Host "📋 Top issues by linter:"
            $linterCounts.GetEnumerator() | Sort-Object Value -Descending | Select-Object -First 5 | ForEach-Object {
                Write-Host "  $($_.Key): $($_.Value) issues"
            }
        }
    }
}

Write-Host ""
Write-Host "💡 TIPS:"
Write-Host "  - Use -Fast to run with optimized config (.golangci-fast.yml)"
Write-Host "  - Use -Fix to auto-fix issues where possible"
Write-Host "  - Use -Verbose for detailed output"
Write-Host "  - Use -ShowStats to see enabled linters"
Write-Host ""

exit $exitCode