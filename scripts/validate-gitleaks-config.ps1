# Validate gitleaks configuration in GitHub Actions workflows
# This script checks that all workflows properly configure gitleaks for SARIF generation

Write-Host "Validating gitleaks configuration in GitHub Actions workflows..." -ForegroundColor Cyan
Write-Host ""

$workflowPath = Join-Path $PSScriptRoot "..\\.github\\workflows"
$issues = @()
$fixed = @()

# Files to check
$workflows = @(
    "security-enhanced.yml",
    "production.yml",
    "deploy-production.yml"
)

foreach ($workflow in $workflows) {
    $filePath = Join-Path $workflowPath $workflow
    if (Test-Path $filePath) {
        $content = Get-Content $filePath -Raw
        
        Write-Host "Checking $workflow..." -ForegroundColor Yellow
        
        # Check for directory creation
        if ($content -match "mkdir -p security-reports") {
            Write-Host "  ✓ Creates security-reports directory" -ForegroundColor Green
            $fixed += "${workflow}: Creates output directory"
        } else {
            Write-Host "  ⚠ Missing directory creation" -ForegroundColor Red
            $issues += "${workflow}: Missing 'mkdir -p security-reports'"
        }
        
        # Check for proper gitleaks flags
        if ($content -match "gitleaks detect.*--redact.*--exit-code=2.*--report-format sarif") {
            Write-Host "  ✓ Uses proper gitleaks flags (--redact, --exit-code=2, SARIF format)" -ForegroundColor Green
            $fixed += "${workflow}: Proper gitleaks flags"
        } elseif ($content -match "args:.*--redact.*--exit-code=2.*--report-format sarif") {
            Write-Host "  ✓ Uses gitleaks action with proper flags" -ForegroundColor Green
            $fixed += "${workflow}: Gitleaks action configured correctly"
        } else {
            Write-Host "  ⚠ Missing proper gitleaks flags" -ForegroundColor Red
            $issues += "${workflow}: Missing proper gitleaks flags"
        }
        
        # Check for SARIF output path
        if ($content -match "security-reports/gitleaks\.sarif") {
            Write-Host "  ✓ Outputs to security-reports/gitleaks.sarif" -ForegroundColor Green
            $fixed += "${workflow}: Correct SARIF output path"
        } else {
            Write-Host "  ⚠ Incorrect or missing SARIF output path" -ForegroundColor Red
            $issues += "${workflow}: Incorrect SARIF output path"
        }
        
        Write-Host ""
    } else {
        Write-Host "$workflow not found" -ForegroundColor Red
        $issues += "${workflow}: File not found"
    }
}

Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host "VALIDATION SUMMARY" -ForegroundColor Cyan
Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host ""

if ($fixed.Count -gt 0) {
    Write-Host "Fixed configurations ($($fixed.Count)):" -ForegroundColor Green
    foreach ($fix in $fixed) {
        Write-Host "  ✓ $fix" -ForegroundColor Green
    }
    Write-Host ""
}

if ($issues.Count -gt 0) {
    Write-Host "Issues found ($($issues.Count)):" -ForegroundColor Red
    foreach ($issue in $issues) {
        Write-Host "  ✗ $issue" -ForegroundColor Red
    }
    Write-Host ""
    Write-Host "Please review and fix the issues above." -ForegroundColor Yellow
    exit 1
} else {
    Write-Host "All gitleaks configurations are properly set up!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Key improvements applied:" -ForegroundColor Cyan
    Write-Host "  1. All workflows create security-reports directory before running gitleaks" -ForegroundColor White
    Write-Host "  2. Using --redact flag to avoid exposing secrets in logs" -ForegroundColor White
    Write-Host "  3. Using --exit-code=2 to only fail on actual leaks (not errors)" -ForegroundColor White
    Write-Host "  4. Generating SARIF format for GitHub Security tab integration" -ForegroundColor White
    Write-Host "  5. Consistent output path: security-reports/gitleaks.sarif" -ForegroundColor White
    Write-Host ""
    exit 0
}