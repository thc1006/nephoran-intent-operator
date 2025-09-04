#!/usr/bin/env pwsh
# =====================================================================================
# CI Job: Advanced Linting
# =====================================================================================
# Mirrors: .github/workflows/main-ci.yml -> linting job
#         .github/workflows/ubuntu-ci.yml -> lint job  
# This script reproduces the exact linting process from CI
# =====================================================================================

param(
    [string]$Config = ".golangci.yml",
    [switch]$Verbose,
    [switch]$DetailedOutput,
    [int]$TimeoutMinutes = 12
)

$ErrorActionPreference = "Stop"

# Load CI environment
. "scripts\ci-env.ps1"

Write-Host "=== Advanced Linting ===" -ForegroundColor Green
Write-Host "Config: $Config" -ForegroundColor Cyan
Write-Host "Timeout: $TimeoutMinutes minutes" -ForegroundColor Cyan
Write-Host ""

# Create reports directory
if (!(Test-Path "lint-reports")) {
    New-Item -ItemType Directory -Path "lint-reports" -Force | Out-Null
}

# Step 1: Verify golangci-lint installation (mirrors CI setup)
Write-Host "[1/4] Verifying golangci-lint installation..." -ForegroundColor Blue
try {
    $lintPath = Get-Command "golangci-lint" -ErrorAction Stop
    $version = & golangci-lint version 2>&1 | Out-String
    Write-Host "✅ golangci-lint found: $($lintPath.Source)" -ForegroundColor Green
    Write-Host "Version: $($version.Trim())" -ForegroundColor Gray
    
    # Verify expected version
    if ($version -match $env:GOLANGCI_LINT_VERSION.Replace("v", "")) {
        Write-Host "✅ Version matches CI: $env:GOLANGCI_LINT_VERSION" -ForegroundColor Green
    } else {
        Write-Host "⚠️ Version mismatch - CI uses: $env:GOLANGCI_LINT_VERSION" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "❌ golangci-lint not found" -ForegroundColor Red
    Write-Host "Run: scripts\install-ci-tools.ps1" -ForegroundColor Yellow
    exit 1
}

# Step 2: Verify configuration (mirrors CI config verify)
Write-Host "[2/4] Verifying linting configuration..." -ForegroundColor Blue
if (Test-Path $Config) {
    Write-Host "✅ Configuration file found: $Config" -ForegroundColor Green
    try {
        golangci-lint config verify --config=$Config
        Write-Host "✅ Configuration is valid" -ForegroundColor Green
    }
    catch {
        Write-Host "⚠️ Configuration verification failed - using anyway" -ForegroundColor Yellow
    }
} else {
    Write-Host "⚠️ Configuration file not found: $Config" -ForegroundColor Yellow
    Write-Host "Using default golangci-lint configuration" -ForegroundColor Gray
    $Config = ""
}

# Step 3: Show enabled linters (mirrors CI behavior)
Write-Host "[3/4] Checking enabled linters..." -ForegroundColor Blue
try {
    Write-Host "Enabled linters:" -ForegroundColor Gray
    $lintersOutput = golangci-lint linters 2>&1 | Out-String
    $enabledLinters = $lintersOutput -split "`n" | Where-Object { $_ -match "^\s*\w+:" } | Select-Object -First 10
    foreach ($linter in $enabledLinters) {
        Write-Host "  $linter" -ForegroundColor Gray
    }
    if ($enabledLinters.Count -eq 10) {
        Write-Host "  ... (showing first 10)" -ForegroundColor Gray
    }
}
catch {
    Write-Host "⚠️ Could not list linters" -ForegroundColor Yellow
}

# Step 4: Run golangci-lint (mirrors CI exact command)
Write-Host "[4/4] Running golangci-lint analysis..." -ForegroundColor Blue

# Build arguments exactly like CI
$args = @(
    "run"
    "--timeout=$($TimeoutMinutes)m"
)

if ($Config) {
    $args += "--config=$Config"
}

if ($Verbose -or $DetailedOutput) {
    $args += "--verbose"
    $args += "--print-issued-lines=true"
    $args += "--print-linter-name=true" 
    $args += "--sort-results=true"
}

# Output formats (mirrors CI multi-format output)
$outputFormats = @()
$outputFormats += "colored-line-number:stdout"

if ($DetailedOutput) {
    $outputFormats += "json:lint-reports/golangci-lint.json"
    $outputFormats += "checkstyle:lint-reports/checkstyle.xml"
    $outputFormats += "sarif:lint-reports/golangci-lint.sarif"
}

if ($outputFormats.Count -gt 0) {
    $args += "--out-format=$($outputFormats -join ',')"
}

$args += "./..."

Write-Host "Command: golangci-lint $($args -join ' ')" -ForegroundColor Gray
Write-Host ""

# Execute linting (capture exit code like CI)
try {
    $startTime = Get-Date
    & golangci-lint @args 2>&1 | Tee-Object -FilePath "lint-reports/lint-output.txt"
    $lintExitCode = $LASTEXITCODE
    $duration = (Get-Date) - $startTime
    
    Write-Host ""
    Write-Host "=== Lint Results Summary ===" -ForegroundColor Green
    Write-Host "Duration: $([math]::Round($duration.TotalSeconds, 1)) seconds" -ForegroundColor Gray
    Write-Host "Exit code: $lintExitCode" -ForegroundColor Gray
    
    # Parse results if JSON output was generated (mirrors CI parsing)
    if ((Test-Path "lint-reports/golangci-lint.json") -and $DetailedOutput) {
        try {
            $jsonContent = Get-Content "lint-reports/golangci-lint.json" -Raw | ConvertFrom-Json
            $totalIssues = if ($jsonContent.Issues) { $jsonContent.Issues.Count } else { 0 }
            
            Write-Host "Total issues found: $totalIssues" -ForegroundColor Cyan
            
            if ($totalIssues -gt 0) {
                Write-Host ""
                Write-Host "Issues by linter:" -ForegroundColor Yellow
                $jsonContent.Issues | Group-Object FromLinter | Sort-Object Count -Descending | ForEach-Object {
                    Write-Host "  $($_.Name): $($_.Count)" -ForegroundColor Gray
                }
                
                Write-Host ""
                Write-Host "Top files with issues:" -ForegroundColor Yellow
                $jsonContent.Issues | Group-Object { $_.Pos.Filename } | Sort-Object Count -Descending | Select-Object -First 5 | ForEach-Object {
                    Write-Host "  $($_.Name): $($_.Count) issues" -ForegroundColor Gray
                }
            }
        }
        catch {
            Write-Host "⚠️ Could not parse JSON results" -ForegroundColor Yellow
        }
    }
    
    # Report final status (mirrors CI logic)
    if ($lintExitCode -eq 0) {
        Write-Host "✅ No linting issues found!" -ForegroundColor Green
    } elseif ($lintExitCode -eq 1) {
        Write-Host "⚠️ Linting issues found (see details above)" -ForegroundColor Yellow
        
        # Show some example issues from output
        if (Test-Path "lint-reports/lint-output.txt") {
            $outputLines = Get-Content "lint-reports/lint-output.txt"
            $issueLines = $outputLines | Where-Object { $_ -match "^\S+\.\w+:" } | Select-Object -First 5
            if ($issueLines) {
                Write-Host ""
                Write-Host "Sample issues:" -ForegroundColor Yellow
                foreach ($line in $issueLines) {
                    Write-Host "  $line" -ForegroundColor Red
                }
                if ($outputLines.Count -gt 5) {
                    Write-Host "  ... (see lint-reports/lint-output.txt for full output)" -ForegroundColor Gray
                }
            }
        }
    } else {
        Write-Host "❌ Linter failed with error code $lintExitCode" -ForegroundColor Red
    }
    
    # Show generated reports
    Write-Host ""
    if (Test-Path "lint-reports") {
        $reports = Get-ChildItem "lint-reports" -File
        if ($reports) {
            Write-Host "Generated reports:" -ForegroundColor Green
            foreach ($report in $reports) {
                $size = [math]::Round($report.Length / 1KB, 1)
                Write-Host "  $($report.Name) ($size KB)" -ForegroundColor Gray
            }
        }
    }
    
    # Exit with same code as linter (matches CI behavior)
    exit $lintExitCode
}
catch {
    Write-Host "❌ Linting execution failed: $_" -ForegroundColor Red
    exit 2
}