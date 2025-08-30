# Format Bypass Safety Checker (PowerShell Version)
# This script helps determine if it's safe to bypass formatting checks
# Usage: .\scripts\format-bypass-check.ps1

param(
    [switch]$Verbose
)

Write-Host "🔍 Format Bypass Safety Checker" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Check if golangci-lint is available
if (!(Get-Command "golangci-lint" -ErrorAction SilentlyContinue)) {
    Write-Host "❌ golangci-lint not found. Please install it first:" -ForegroundColor Red
    Write-Host "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" -ForegroundColor Yellow
    exit 1
}

Write-Host "📋 Running critical checks (non-formatting)..." -ForegroundColor Blue

# Create temp directory if needed
$tempDir = $env:TEMP
if (!(Test-Path $tempDir)) {
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
}

$bypassResults = "$tempDir\bypass-results.json"
$fullResults = "$tempDir\full-results.json"

try {
    # Run bypass configuration (without formatting linters)
    Write-Host "Running bypass lint check..." -ForegroundColor Gray
    $bypassExitCode = 0
    golangci-lint run -c .golangci-bypass.yml --out-format=json > $bypassResults 2>$null
    if ($LASTEXITCODE -ne 0) {
        $bypassExitCode = $LASTEXITCODE
    }

    # Run full configuration to see formatting issues
    Write-Host "Running full lint check..." -ForegroundColor Gray
    $fullExitCode = 0
    golangci-lint run --out-format=json > $fullResults 2>$null
    if ($LASTEXITCODE -ne 0) {
        $fullExitCode = $LASTEXITCODE
    }

    Write-Host ""
    Write-Host "📊 Results Analysis:" -ForegroundColor Yellow
    Write-Host "====================" -ForegroundColor Yellow

    $criticalIssues = 0
    $totalIssues = 0
    $formatIssues = 0

    # Analyze bypass results (critical issues)
    if (Test-Path $bypassResults) {
        try {
            $bypassJson = Get-Content $bypassResults -Raw | ConvertFrom-Json
            if ($bypassJson.Issues) {
                $criticalIssues = ($bypassJson.Issues | Where-Object { $_.Severity -eq "error" }).Count
                if (!$criticalIssues) { $criticalIssues = 0 }
            }
            Write-Host "Critical errors (non-formatting): $criticalIssues" -ForegroundColor $(if ($criticalIssues -eq 0) { "Green" } else { "Red" })
        } catch {
            Write-Host "Critical errors (non-formatting): Could not analyze" -ForegroundColor Gray
        }
    } else {
        Write-Host "Critical errors (non-formatting): Could not analyze" -ForegroundColor Gray
    }

    # Analyze full results 
    if (Test-Path $fullResults) {
        try {
            $fullJson = Get-Content $fullResults -Raw | ConvertFrom-Json
            if ($fullJson.Issues) {
                $totalIssues = $fullJson.Issues.Count
                $formatIssues = ($fullJson.Issues | Where-Object { $_.FromLinter -match "gofmt|gofumpt|gci|whitespace" }).Count
                if (!$formatIssues) { $formatIssues = 0 }
            }
            Write-Host "Total issues (full check): $totalIssues" -ForegroundColor Gray
            Write-Host "Formatting-only issues: $formatIssues" -ForegroundColor $(if ($formatIssues -eq 0) { "Green" } else { "Yellow" })
        } catch {
            Write-Host "Total issues (full check): Could not analyze" -ForegroundColor Gray
            Write-Host "Formatting-only issues: Could not analyze" -ForegroundColor Gray
        }
    } else {
        Write-Host "Total issues (full check): Could not analyze" -ForegroundColor Gray
    }

    Write-Host ""
    Write-Host "🎯 Safety Assessment:" -ForegroundColor Magenta
    Write-Host "=====================" -ForegroundColor Magenta

    if ($criticalIssues -eq 0) {
        Write-Host "✅ SAFE TO BYPASS" -ForegroundColor Green
        Write-Host ""
        Write-Host "No critical errors detected. You can safely use format bypass:" -ForegroundColor Green
        Write-Host ""
        Write-Host "   git commit -m `"[bypass-format] your commit message here`"" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Or use one of these bypass keywords in your commit message:" -ForegroundColor Gray
        Write-Host "   [bypass-format], [format-bypass], [bypass-lint], [skip-lint]" -ForegroundColor Gray
        Write-Host ""
        Write-Host "⚠️  Remember to fix formatting issues in a follow-up commit:" -ForegroundColor Yellow
        Write-Host "   golangci-lint run --fix" -ForegroundColor Cyan
        Write-Host "   gofmt -w ." -ForegroundColor Cyan
        Write-Host ""
        $safetyStatus = "SAFE"
    } else {
        Write-Host "❌ NOT SAFE TO BYPASS" -ForegroundColor Red
        Write-Host ""
        Write-Host "Critical errors found: $criticalIssues" -ForegroundColor Red
        Write-Host ""
        Write-Host "You must fix these critical issues before bypassing:" -ForegroundColor Yellow
        Write-Host ""
        
        # Show critical issues if available
        if ((Test-Path $bypassResults) -and $bypassJson -and $bypassJson.Issues) {
            Write-Host "Critical Issues:" -ForegroundColor Red
            $criticalList = $bypassJson.Issues | Where-Object { $_.Severity -eq "error" } | Select-Object -First 10
            foreach ($issue in $criticalList) {
                Write-Host "  - $($issue.Pos.Filename):$($issue.Pos.Line): $($issue.Text)" -ForegroundColor Red
            }
        }
        Write-Host ""
        Write-Host "Fix critical issues first, then you can bypass formatting issues." -ForegroundColor Yellow
        $safetyStatus = "UNSAFE"
    }

    Write-Host ""
    Write-Host "📝 Quick Fix Commands:" -ForegroundColor Blue
    Write-Host "======================" -ForegroundColor Blue
    Write-Host "Format code:      gofmt -w ." -ForegroundColor Cyan
    Write-Host "Fix imports:      gci write --skip-generated -s standard -s default -s `"prefix(github.com/nephio-project/nephoran-intent-operator)`" ." -ForegroundColor Cyan
    Write-Host "Auto-fix lint:    golangci-lint run --fix" -ForegroundColor Cyan
    Write-Host "Full lint check:  golangci-lint run" -ForegroundColor Cyan
    Write-Host "Bypass check:     golangci-lint run -c .golangci-bypass.yml" -ForegroundColor Cyan

} finally {
    # Cleanup
    if (Test-Path $bypassResults) { Remove-Item $bypassResults -Force }
    if (Test-Path $fullResults) { Remove-Item $fullResults -Force }
}

# Exit with appropriate code
if ($safetyStatus -eq "SAFE") {
    exit 0
} else {
    exit 1
}