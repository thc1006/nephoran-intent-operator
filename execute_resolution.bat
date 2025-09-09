@echo off
echo EXECUTING ULTRA-SPEED CONFLICT RESOLUTION...

REM Accept theirs for all conflict files
git checkout --theirs api/v1alpha1/zz_generated.deepcopy.go
git checkout --theirs pkg/auth/security_test.go.disabled
git checkout --theirs pkg/auth/session_manager_comprehensive_test.go.disabled
git checkout --theirs pkg/controllers/networkintent_cleanup_edge_cases_test.go
git checkout --theirs pkg/nephio/blueprint/manager_comprehensive_test.go
git checkout --theirs pkg/nephio/krm/runtime_comprehensive_test.go
git checkout --theirs pkg/nephio/multicluster/chaos_resilience_test.go
git checkout --theirs pkg/nephio/multicluster/performance_test.go
git checkout --theirs pkg/oran/a1/validation_test.go
git checkout --theirs pkg/oran/o2/helper_types.go
git checkout --theirs pkg/oran/o2/o2_manager_test.go
git checkout --theirs pkg/oran/o2/providers/providers_test.go
git checkout --theirs pkg/rag/minimal_interface.go
git checkout --theirs pkg/rag/rag_pipeline.go
git checkout --theirs pkg/testutil/auth/testing.go
git checkout --theirs test/integration/porch/intent_reconciliation_test.go
git checkout --theirs test/integration/porch/resilience_test.go
git checkout --theirs test/integration/porch_integration_test.go
git checkout --theirs test/integration/system_test.go
git checkout --theirs tests/integration/audit_e2e_test.go
git checkout --theirs tests/integration/audit_test_types.go
git checkout --theirs tests/integration/availability_tracking_test.go
git checkout --theirs tests/o2/integration/api_endpoints_test.go
git checkout --theirs tests/o2/integration/multi_cloud_test.go
git checkout --theirs tests/performance/production_load_test.go
git checkout --theirs tests/performance/race_benchmarks_test.go
git checkout --theirs tests/unit/llm_test.go
git checkout --theirs tests/validation/performance_comprehensive_test.go
git checkout --theirs tests/validation/sla_validation_test.go

REM Add all resolved files
git add -A

echo CONTINUING REBASE...
git rebase --continue

echo CONFLICT RESOLUTION COMPLETE!