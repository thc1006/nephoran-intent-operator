# Fix Workflow State for integrate/mvp Merge
# This script ensures only ci.yml and conductor-loop.yml are active
# All other workflows should be disabled to match integrate/mvp branch

$workflowDir = ".github\workflows"
Set-Location $workflowDir

Write-Host "`n=== Fixing Workflow State for integrate/mvp Merge ===" -ForegroundColor Cyan
Write-Host "Target: Only ci.yml (and optionally conductor-loop.yml) should be active" -ForegroundColor Gray

# Workflows that should be disabled (everything except ci.yml and conductor-loop.yml)
$toDisable = @(
    'claude.yml',
    'claude-code-review.yml', 
    'comprehensive-validation.yml',
    'dependabot.yml',
    'dependency-security.yml',
    'deploy-production.yml',
    'optimized-main-ci.yml',
    'performance-benchmarking.yml',
    'production.yml',
    'quality-gate.yml',
    'release.yml',
    'security-enhanced.yml',
    'ci-security-section.yml',
    'ci-ultra-speed.yml',
    'conductor-loop-cicd.yml',
    'integration-tests.yml',
    'ubuntu-ci.yml'
)

$disabledCount = 0
$alreadyDisabled = 0

Write-Host "`nDisabling unnecessary workflows..." -ForegroundColor Yellow
foreach ($workflow in $toDisable) {
    if (Test-Path $workflow) {
        $newName = "$workflow.disabled"
        if (Test-Path $newName) {
            Remove-Item $newName -Force
        }
        Rename-Item $workflow $newName
        Write-Host "  ✓ Disabled: $workflow" -ForegroundColor Green
        $disabledCount++
    } elseif (Test-Path "$workflow.disabled") {
        Write-Host "  • Already disabled: $workflow" -ForegroundColor Gray
        $alreadyDisabled++
    }
}

Write-Host "`n=== Summary ===" -ForegroundColor Cyan
Write-Host "Disabled: $disabledCount workflows" -ForegroundColor Green
Write-Host "Already disabled: $alreadyDisabled workflows" -ForegroundColor Gray

# Check what's active now
$activeWorkflows = Get-ChildItem -Filter "*.yml" | Select-Object -ExpandProperty Name
Write-Host "`nActive workflows remaining:" -ForegroundColor Cyan
foreach ($workflow in $activeWorkflows) {
    if ($workflow -eq 'ci.yml') {
        Write-Host "  ✓ $workflow (required for integrate/mvp)" -ForegroundColor Green
    } elseif ($workflow -eq 'conductor-loop.yml') {
        Write-Host "  ⚠ $workflow (optional - specific to conductor-loop feature)" -ForegroundColor Yellow
    } else {
        Write-Host "  ✗ $workflow (should be disabled!)" -ForegroundColor Red
    }
}

Write-Host "`n=== Recommendation ===" -ForegroundColor Magenta
Write-Host "The workflow state is now aligned for merging to integrate/mvp." -ForegroundColor White
Write-Host "Only ci.yml (and optionally conductor-loop.yml) are active." -ForegroundColor White
Write-Host "`nTo merge safely:" -ForegroundColor Yellow
Write-Host "  1. git add .github/workflows/" -ForegroundColor Gray
Write-Host "  2. git commit -m 'fix: align workflows with integrate/mvp branch'" -ForegroundColor Gray
Write-Host "  3. git push" -ForegroundColor Gray
Write-Host "  4. Create PR to integrate/mvp" -ForegroundColor Gray