# EMERGENCY GO VERSION FIX SCRIPT
# Fixes all workflows with outdated Go versions causing CI failures

$workflows = @(
    ".github/workflows/ci-stability-orchestrator.yml",
    ".github/workflows/container-build-2025.yml",
    ".github/workflows/debug-ghcr-auth.yml",
    ".github/workflows/dev-fast-fixed.yml", 
    ".github/workflows/dev-fast.yml",
    ".github/workflows/emergency-merge.yml",
    ".github/workflows/final-integration-validation.yml",
    ".github/workflows/main-ci-optimized.yml",
    ".github/workflows/nephoran-ci-consolidated-2025.yml",
    ".github/workflows/nephoran-master-orchestrator.yml",
    ".github/workflows/parallel-tests.yml",
    ".github/workflows/pr-ci-fast.yml",
    ".github/workflows/production-ci.yml",
    ".github/workflows/security-enhanced-ci.yml",
    ".github/workflows/security-scan-optimized.yml",
    ".github/workflows/security-scan-ultra-reliable.yml",
    ".github/workflows/security-scan.yml",
    ".github/workflows/ubuntu-ci.yml"
)

Write-Host "🚨 EMERGENCY FIX: Updating GO_VERSION from 1.22.x to 1.24.6 in $($workflows.Count) workflows"

$totalFixed = 0

foreach ($workflow in $workflows) {
    if (Test-Path $workflow) {
        Write-Host "Fixing: $workflow"
        
        # Read content
        $content = Get-Content $workflow -Raw
        
        # Replace GO_VERSION patterns
        $originalContent = $content
        $content = $content -replace 'GO_VERSION:\s*"1\.22\.[0-9]+"', 'GO_VERSION: "1.24.6"  # EMERGENCY FIX - Match go.mod toolchain'
        
        if ($content -ne $originalContent) {
            # Write back
            Set-Content $workflow -Value $content -NoNewline
            Write-Host "  ✅ Fixed GO_VERSION in $workflow"
            $totalFixed++
        } else {
            Write-Host "  ⚠️ No GO_VERSION pattern found in $workflow"
        }
    } else {
        Write-Host "  ❌ File not found: $workflow"
    }
}

Write-Host ""
Write-Host "🎯 EMERGENCY FIX COMPLETE: Updated GO_VERSION in $totalFixed workflows"
Write-Host "📋 Summary: All workflows now use GO_VERSION: 1.24.6 (matching go.mod toolchain)"