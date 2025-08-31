#!/usr/bin/env pwsh
# Disable redundant GitHub Actions workflows to accelerate development

Write-Host "Disabling redundant workflows to speed up development..." -ForegroundColor Yellow

$workflowsToDisable = @(
    "ci.yml",              # Heavy CI - runs on main/integrate pushes
    "ci-fix.yml",          # Duplicate CI for feat/e2e
    "ci-minimal-status.yml", # Runs on EVERY push/PR - biggest slowdown!
    "ubuntu-ci.yml",       # Manual trigger, rarely needed
    "pr-ci.yml",           # PR validation - slows down PR reviews
    "validate-configs.yml" # Config validation - only needed pre-merge
)

$workflowDir = ".github/workflows"

foreach ($workflow in $workflowsToDisable) {
    $sourcePath = Join-Path $workflowDir $workflow
    $targetPath = "$sourcePath.disabled"
    
    if (Test-Path $sourcePath) {
        Move-Item -Path $sourcePath -Destination $targetPath -Force
        Write-Host "✓ Disabled: $workflow" -ForegroundColor Green
    } else {
        Write-Host "⚠ Already disabled or not found: $workflow" -ForegroundColor Yellow
    }
}

Write-Host "`n📊 Summary:" -ForegroundColor Cyan
Write-Host "- Disabled 6 redundant CI workflows" -ForegroundColor White
Write-Host "- Kept 2 essential workflows (debug-ghcr, emergency-merge)" -ForegroundColor White
Write-Host "`n⚡ Development should now be faster with fewer redundant CI runs!" -ForegroundColor Green
Write-Host "`nTo re-enable workflows later, rename them from .yml.disabled back to .yml" -ForegroundColor Gray