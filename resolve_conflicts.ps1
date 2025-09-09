#!/usr/bin/env pwsh
# Ultra-speed batch conflict resolution script
# Accept ALL incoming changes (--theirs) for listed files

$conflictFiles = @(
    "api/v1alpha1/zz_generated.deepcopy.go",
    "pkg/auth/security_test.go.disabled", 
    "pkg/auth/session_manager_comprehensive_test.go.disabled",
    "pkg/controllers/networkintent_cleanup_edge_cases_test.go",
    "pkg/nephio/blueprint/manager_comprehensive_test.go",
    "pkg/nephio/krm/runtime_comprehensive_test.go",
    "pkg/nephio/multicluster/chaos_resilience_test.go",
    "pkg/nephio/multicluster/performance_test.go",
    "pkg/oran/a1/validation_test.go",
    "pkg/oran/o2/helper_types.go",
    "pkg/oran/o2/o2_manager_test.go",
    "pkg/oran/o2/providers/providers_test.go",
    "pkg/rag/minimal_interface.go",
    "pkg/rag/rag_pipeline.go",
    "pkg/testutil/auth/testing.go",
    "test/integration/porch/intent_reconciliation_test.go",
    "test/integration/porch/resilience_test.go",
    "test/integration/porch_integration_test.go",
    "test/integration/system_test.go",
    "tests/integration/audit_e2e_test.go",
    "tests/integration/audit_test_types.go",
    "tests/integration/availability_tracking_test.go",
    "tests/o2/integration/api_endpoints_test.go",
    "tests/o2/integration/multi_cloud_test.go",
    "tests/performance/production_load_test.go",
    "tests/performance/race_benchmarks_test.go",
    "tests/unit/llm_test.go",
    "tests/validation/performance_comprehensive_test.go",
    "tests/validation/sla_validation_test.go"
)

Write-Host "🚀 ULTRA-SPEED CONFLICT RESOLUTION: Accepting --theirs for $($conflictFiles.Count) files" -ForegroundColor Yellow

# Batch resolve conflicts
foreach ($file in $conflictFiles) {
    if (Test-Path $file) {
        Write-Host "  ✅ Resolving: $file" -ForegroundColor Green
        git checkout --theirs $file
        git add $file
    } else {
        Write-Host "  ⚠️  Not found: $file" -ForegroundColor Yellow
    }
}

Write-Host "🎯 Conflict resolution complete! Continuing rebase..." -ForegroundColor Cyan

# Continue the rebase operation
git rebase --continue

Write-Host "✨ Rebase continuation executed!" -ForegroundColor Green