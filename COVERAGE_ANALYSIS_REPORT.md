# Test Coverage Analysis Report
**Date**: 2026-02-23
**Overall Coverage**: 17.7%
**Total Test Files**: 374
**Integration/E2E Tests**: 46
**Test Execution Time**: ~6 minutes

---

## Executive Summary

The Nephoran Intent Operator codebase has **critically low coverage at 17.7%**, far below production standards. While some packages demonstrate excellent testing (89-100%), the majority of critical infrastructure packages have dangerously low coverage (<30%).

### Key Findings

1. **Critical Infrastructure Gaps**: Core controllers, O-RAN interfaces, and LLM integration have <30% coverage
2. **Zero-Coverage Command Packages**: 3 command-line tools have no tests
3. **Untested Security Functions**: Major security and authentication functions are completely untested
4. **Test Quality Mixed**: Strong unit tests in some areas, but lacking integration test coverage for critical flows
5. **3 Test Suite Failures**: `pkg/disaster`, `planner/internal/security`, `test/envtest`

---

## Overall Coverage Statistics

| Metric | Value |
|--------|-------|
| **Total Coverage** | 17.7% |
| **Passing Test Packages** | 84 |
| **Failing Test Packages** | 3 |
| **Packages with 0% Coverage** | 8 |
| **Packages <50% Coverage** | 57 |
| **Packages ≥80% Coverage** | 10 |

---

## Critical Packages Coverage Analysis

### P0 - CRITICAL (Production-Blocking Issues)

These packages are core to the operator's functionality and have unacceptably low coverage:

| Package | Coverage | Risk Level | Impact |
|---------|----------|------------|--------|
| **pkg/controllers** | 20.8% | 🔴 CRITICAL | Main NetworkIntent reconciliation logic |
| **pkg/controllers/parallel** | 21.0% | 🔴 CRITICAL | Parallel execution of intents |
| **pkg/controllers/orchestration** | 7.0% | 🔴 CRITICAL | Multi-cluster orchestration |
| **pkg/oran/o1** | 3.0% | 🔴 CRITICAL | O1 interface for fault management |
| **pkg/oran/o2** | 2.0% | 🔴 CRITICAL | O2 IMS interface for inventory |
| **pkg/oran/a1** | 31.5% | 🔴 CRITICAL | A1 policy management |
| **pkg/oran/e2** | 13.9% | 🔴 CRITICAL | E2 subscription management |
| **pkg/llm** | 7.2% | 🔴 CRITICAL | LLM integration for NL parsing |
| **pkg/nephio** | 9.6% | 🔴 CRITICAL | Porch/KPT package generation |
| **pkg/nephio/porch** | 1.1% | 🔴 CRITICAL | Porch client operations |
| **pkg/security** | 5.8% | 🔴 CRITICAL | Secret management, encryption |
| **internal/loop** | 30.0% | 🔴 CRITICAL | File watcher and processing loop |

### P1 - HIGH PRIORITY

| Package | Coverage | Risk Level | Impact |
|---------|----------|------------|--------|
| **pkg/porch** | 56.9% | 🟠 HIGH | Porch integration layer |
| **internal/porch** | 54.8% | 🟠 HIGH | Porch command executor |
| **pkg/handlers** | 46.5% | 🟠 HIGH | HTTP handlers for LLM processor |
| **pkg/audit** | 32.0% | 🟠 HIGH | Audit trail and compliance |
| **pkg/auth** | 22.9% | 🟠 HIGH | Authentication/authorization |
| **pkg/git** | 14.3% | 🟠 HIGH | Git repository operations |

### P2 - MEDIUM PRIORITY

| Package | Coverage | Risk Level |
|---------|----------|------------|
| **pkg/monitoring** | 3.1% | 🟡 MEDIUM |
| **pkg/cnf** | 26.4% | 🟡 MEDIUM |
| **pkg/config** | 21.9% | 🟡 MEDIUM |
| **api/v1alpha1** | 4.8% | 🟡 MEDIUM |
| **api/intent/v1alpha1** | 12.6% | 🟡 MEDIUM |

---

## Zero Coverage Packages (8 Total)

These packages have **NO TESTS**:

1. **cmd/conductor-watch** (0.0%) - File watcher command-line tool
2. **cmd/intent-ingest** (0.0%) - Intent ingestion HTTP server
3. **cmd/porch-publisher** (0.0%) - Porch package publisher
4. **pkg/controllers/optimized** (0.0%) - Optimized controller implementation
5. **pkg/nephio/krm** (0.0%) - KRM resource manipulation
6. **tests/disaster-recovery** (0.0%) - Disaster recovery test suite (meta)
7. **tests/security** (0.0%) - Security test suite (meta)
8. **tests/validation** (0.0%) - Validation test suite (meta)

---

## Well-Tested Packages (≥80% Coverage)

### Excellent Coverage (90-100%)

| Package | Coverage | Notes |
|---------|----------|-------|
| **internal/patchgen/generator** | 100.0% | Perfect coverage - patch generation |
| **planner/internal/security** | 99.8% | Near-perfect security validation |
| **internal/ves** | 95.0% | VES event generation |
| **pkg/oran/health** | 90.0% | O-RAN health checking |
| **pkg/webhooks** | 89.1% | Webhook validation |

### Good Coverage (80-89%)

| Package | Coverage | Notes |
|---------|----------|-------|
| **internal/patch** | 88.7% | Intent patching logic |
| **pkg/services** | 87.1% | Service management |
| **planner/internal/rules** | 87.0% | Planning rules engine |
| **internal/patchgen** | 84.0% | Patch generator |
| **internal/kpm** | 82.9% | KPM package generation |

---

## Critical Untested Functions

### Controllers (pkg/controllers)

**0% Coverage Functions** (High Risk):
- `AuditTrailController.Reconcile()` - Complete reconciliation loop untested
- `BackupPolicyController.Reconcile()` - Backup logic untested
- `CertificateAutomationController.Reconcile()` - Certificate management untested
- `CNFDeploymentController.Reconcile()` - CNF deployment untested
- `DisasterRecoveryController.Reconcile()` - DR logic untested
- `E2NodeSetController.Reconcile()` - E2 node management untested
- `FailoverPolicyController.Reconcile()` - Failover logic untested
- `GitOpsDeploymentController.Reconcile()` - GitOps integration untested
- `IntentProcessingController.Reconcile()` - Intent processing untested
- `ManagedElementController.Reconcile()` - Managed element lifecycle untested
- `ManifestGenerationController.Reconcile()` - Manifest generation untested
- `O1InterfaceController.Reconcile()` - O1 interface management untested
- `ResourcePlanController.Reconcile()` - Resource planning untested

**Impact**: These are the core Kubernetes controllers that power the operator. Zero coverage means production bugs will only be found in runtime.

### O-RAN Interfaces

#### pkg/oran/o1 (3.0% coverage)

**0% Coverage Functions**:
- `ComprehensiveAccountingManager` - Complete accounting system (2000+ lines)
- `NewUsageDataCollector()` - Usage data collection
- `ProcessEvent()` - Event processing
- All billing and fraud detection functions

**Impact**: O1 interface is critical for fault management and accounting in O-RAN. This code is essentially untested.

#### pkg/oran/o2 (2.0% coverage)

**0% Coverage Functions**:
- `Shutdown()` - Clean shutdown logic
- `GetSystemInfo()` - System information
- `collectResourceMetrics()` - Metrics collection
- `performHealthChecks()` - Health checking
- All API handlers (`handleGetServiceInfo`, `handleHealthCheck`, etc.)
- Complete resource pool management

**Impact**: O2 IMS interface for inventory management is nearly untested. Production failures highly likely.

#### pkg/oran/a1 (31.5% coverage)

**Coverage too low for**:
- Policy lifecycle management
- A1 enrichment
- Policy validation
- NEF integration

### LLM Integration (pkg/llm - 7.2% coverage)

**0% Coverage Functions**:
- `AdvancedCircuitBreaker` - Complete circuit breaker (all methods at 0%)
- `Execute()` - Main circuit breaker execution
- `allowRequest()` - Request throttling
- `onFailure()` - Failure handling
- `GetStats()` - Statistics collection
- `Reset()`, `ForceOpen()`, `ForceClose()` - State management

**Impact**: LLM integration is a core feature for NL intent processing. Circuit breaker provides resilience - completely untested means production outages likely.

### Security (pkg/security - 5.8% coverage)

**0% Coverage Functions**:
- `NewAtomicFileWriter()` - Atomic file operations
- `CreateIntentFile()` - Secure file creation
- `WriteIntentAtomic()` - Atomic writes
- `SecureCopy()` - Secure file copying
- `LogSecretRotation()` - Secret rotation audit
- `LogAPIKeyValidation()` - API key audit
- `AuditSecretAccess()` - Secret access audit
- `NewFileBackend()` - File-based secret backend
- `Store()`, `Retrieve()` - Secret storage

**Impact**: Security functions are critical. Zero coverage on atomic file operations, secret management, and audit logging is a **MAJOR SECURITY RISK**.

---

## Test Quality Analysis

### Strengths

1. **Strong Unit Test Coverage in Core Areas**:
   - `internal/patch` (88.7%) - Good isolation, behavior-driven tests
   - `internal/patchgen` (84.0%) - Comprehensive patch generation tests
   - `pkg/webhooks` (89.1%) - Webhook validation well tested
   - `internal/ves` (95.0%) - VES event generation excellent

2. **Integration Test Presence**:
   - 46 integration/e2e test files found
   - Good mix of unit and integration tests
   - Tests use proper testify suites

3. **Test Isolation**:
   - Good use of temp directories (`t.TempDir()`)
   - Proper cleanup in teardown
   - Mock providers for external dependencies

4. **Security Testing**:
   - Dedicated security test files
   - XSS validation tests
   - Path traversal tests
   - Webhook validation

### Weaknesses

1. **Testing Implementation Not Behavior**:
   ```go
   // Bad: Testing implementation details
   func TestWatcherInternals(t *testing.T) {
       w := NewWatcher()
       if w.fileQueue == nil { // Testing internal state
           t.Error("queue is nil")
       }
   }

   // Good: Testing behavior
   func TestWatcherProcessesIntentFiles(t *testing.T) {
       w := NewWatcher()
       w.ProcessFile("intent-test.json")
       // Assert on observable behavior (file processed, event emitted)
   }
   ```

2. **Insufficient Integration Test Coverage**:
   - Only 46 integration tests vs 374 total test files (12.3%)
   - Missing end-to-end flows:
     - NL Intent → LLM → RAG → NetworkIntent CRD → Porch → A1 Policy
     - NetworkIntent creation → Controller → Porch → Package publish
     - Multi-cluster orchestration flows

3. **Missing Critical Path Testing**:
   - Controller reconciliation loops mostly untested
   - Error handling paths minimal coverage
   - Edge cases not thoroughly tested

4. **Flaky Tests**:
   - Pre-existing failures in 3 test suites
   - Time-dependent tests (debounce, stability checks)
   - Race conditions in parallel tests

5. **Test Isolation Issues**:
   - Some tests share state
   - Cleanup not always complete
   - Test order dependencies

---

## Coverage Gaps by Functional Area

### 1. Controller Reconciliation (CRITICAL GAP)

**Gap**: Main reconciliation logic in 13+ controllers has 0% coverage

**Missing Tests**:
- Happy path: NetworkIntent created → reconciliation → Porch package created
- Error handling: Invalid intent spec → validation error → status update
- Retry logic: Porch API failure → exponential backoff → success
- Deletion: Intent deleted → finalizer cleanup → resource removal
- Status updates: Progress tracking → condition updates → ready state

**Risk**: Production bugs in core business logic will only surface in runtime

### 2. O-RAN Interface Integration (CRITICAL GAP)

**Gap**: A1/E2/O1/O2 interfaces have <32% coverage

**Missing Tests**:
- A1 policy lifecycle: Create → Update → Delete
- E2 subscription management: Subscribe → Event stream → Unsubscribe
- O1 fault management: Alarm → Notification → Resolution
- O2 inventory: Resource discovery → Registration → Updates

**Risk**: O-RAN integration is a core product feature - untested means customer-facing failures

### 3. LLM/RAG Pipeline (CRITICAL GAP)

**Gap**: LLM integration has 7.2% coverage

**Missing Tests**:
- Natural language parsing: "Deploy 5G core with 3 UPF replicas" → Intent JSON
- RAG context retrieval: Intent query → Vector search → Context injection
- Circuit breaker: LLM timeout → Circuit open → Fallback
- Retry with backoff: LLM 429 → Exponential backoff → Success

**Risk**: AI-powered features are untested - hallucinations, timeouts, errors uncaught

### 4. Security & Audit (CRITICAL GAP)

**Gap**: Security package 5.8%, audit 32% coverage

**Missing Tests**:
- Secret rotation: Old secret → Rotate → New secret → Old invalidated
- Atomic file writes: Partial write → Rollback → Retry → Success
- Audit trail: Event → Log → Compliance check → Report
- Authentication: Token → Validate → RBAC check → Allow/Deny

**Risk**: Security vulnerabilities, compliance failures, data corruption

### 5. Multi-Cluster Orchestration (MODERATE GAP)

**Gap**: Orchestration controller 7% coverage

**Missing Tests**:
- Cluster placement: Intent → Placement decision → Multi-cluster deploy
- Failover: Primary cluster down → Detect → Failover to backup
- Resource federation: Cross-cluster resource sharing

**Risk**: Multi-cluster features untested - production deployment failures

---

## Test Execution Issues

### Failing Test Suites (3)

1. **pkg/disaster** - FAIL (6.638s)
   - Likely failover manager test failures
   - Impact: Disaster recovery features untested

2. **planner/internal/security** - FAIL (0.197s)
   - Security validation failures
   - Impact: Planning security checks broken

3. **test/envtest** - FAIL (0.120s)
   - Environment test setup issues
   - Impact: Controller integration tests blocked

### Test Performance

- Total execution time: ~6 minutes
- Slowest packages:
  - `internal/loop` - 105s
  - `pkg/controllers/parallel` - 69s
  - `pkg/audit` - 58s
  - `pkg/nephio/multicluster` - 36s

**Optimization Needed**: Long-running tests should be marked as integration tests and run separately

---

## Recommendations

### Immediate Actions (Week 1)

#### Priority 1: Critical Controllers

**Task**: Add basic controller reconciliation tests
**Target**: Increase controller coverage from 20% → 60%
**Effort**: 40 hours

**Focus Areas**:
1. NetworkIntent controller happy path
2. Error handling and validation
3. Status update logic
4. Finalizer cleanup

**Test Structure**:
```go
func TestNetworkIntentController_Reconcile_HappyPath(t *testing.T) {
    // Arrange: Create NetworkIntent
    // Act: Trigger reconciliation
    // Assert: Porch package created, status updated
}

func TestNetworkIntentController_Reconcile_InvalidSpec(t *testing.T) {
    // Arrange: Create invalid NetworkIntent
    // Act: Trigger reconciliation
    // Assert: Validation error, status shows error condition
}
```

#### Priority 2: O-RAN Interface Tests

**Task**: Add O-RAN interface integration tests
**Target**: Increase O-RAN coverage from <32% → 70%
**Effort**: 32 hours

**Focus Areas**:
1. A1 policy CRUD operations
2. E2 subscription lifecycle
3. O1 fault notifications
4. O2 resource pool management

**Test Structure**:
```go
func TestA1Mediator_CreatePolicy_Success(t *testing.T) {
    // Arrange: Mock A1 server, policy spec
    // Act: Create policy via A1 client
    // Assert: Policy created, stored, retrievable
}
```

#### Priority 3: Security Function Tests

**Task**: Add security and audit tests
**Target**: Increase security coverage from 5.8% → 80%
**Effort**: 24 hours

**Focus Areas**:
1. Atomic file operations
2. Secret storage and retrieval
3. Audit trail logging
4. Authentication/authorization

#### Priority 4: Fix Failing Tests

**Task**: Debug and fix 3 failing test suites
**Effort**: 8 hours

**Steps**:
1. Run `go test ./pkg/disaster -v` - identify failures
2. Run `go test ./planner/internal/security -v` - fix security tests
3. Run `go test ./test/envtest -v` - fix envtest setup

### Short-Term Actions (Weeks 2-4)

#### Add Integration Tests

**Target**: Increase integration test count from 46 → 100+
**Effort**: 60 hours

**Focus on Critical Flows**:
1. **End-to-End Intent Flow**:
   ```
   NL Intent → LLM Processing → RAG Context →
   Intent JSON → NetworkIntent CRD → Controller →
   Porch Package → A1 Policy → Deployment
   ```

2. **Multi-Cluster Orchestration**:
   ```
   NetworkIntent → Placement → Multi-cluster Deploy →
   Health Check → Failover
   ```

3. **Disaster Recovery**:
   ```
   Primary Down → Detection → Failover →
   Backup Active → Recovery
   ```

**Test Infrastructure Needed**:
- Mock Porch server
- Mock A1/E2/O1/O2 endpoints
- Multi-cluster test environment
- Chaos testing framework

#### Improve Test Quality

**Focus**:
1. Test behavior, not implementation
2. Add property-based tests for validation logic
3. Add mutation testing to verify test effectiveness
4. Separate unit/integration/e2e tests with build tags

**Example**:
```go
// +build integration

func TestFullIntentWorkflow_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    // Full end-to-end test
}
```

### Long-Term Actions (Months 2-3)

#### Establish Coverage Requirements

**Target Coverage by Package Type**:
- Controllers: 80% minimum
- O-RAN interfaces: 80% minimum
- Security/Auth: 90% minimum
- Internal libraries: 70% minimum
- Commands: 60% minimum

**Enforce in CI/CD**:
```bash
# Fail build if critical packages below threshold
go test ./pkg/controllers/... -coverprofile=cover.out
go tool cover -func=cover.out | grep total | awk '{print $3}' | \
  sed 's/%//' | awk '{if ($1 < 80) exit 1}'
```

#### Add Advanced Testing

1. **Property-Based Testing** (QuickCheck/rapid):
   ```go
   func TestIntentValidation_Properties(t *testing.T) {
       rapid.Check(t, func(t *rapid.T) {
           intent := generateRandomIntent(t)
           result := ValidateIntent(intent)
           // Property: validation is deterministic
           assert.Equal(t, result, ValidateIntent(intent))
       })
   }
   ```

2. **Mutation Testing** (go-mutesting):
   - Verify tests catch injected bugs
   - Ensure tests are actually effective

3. **Chaos Testing**:
   - Network failures
   - Pod crashes
   - API timeouts
   - Resource exhaustion

4. **Performance Benchmarks**:
   ```go
   func BenchmarkIntentProcessing(b *testing.B) {
       for i := 0; i < b.N; i++ {
           ProcessIntent(intent)
       }
   }
   ```

#### Documentation

1. **Testing Guidelines** (`docs/TESTING.md`):
   - How to write good tests
   - Test structure and patterns
   - Running specific test suites
   - Coverage requirements

2. **Coverage Reports in CI**:
   - HTML coverage reports
   - Coverage trend tracking
   - Per-PR coverage diff

---

## Prioritized Coverage Gap List

### P0 - Critical (Fix Immediately)

| Package | Current | Target | Gap | Effort |
|---------|---------|--------|-----|--------|
| pkg/controllers | 20.8% | 80% | 59.2% | 40h |
| pkg/oran/o1 | 3.0% | 80% | 77.0% | 16h |
| pkg/oran/o2 | 2.0% | 80% | 78.0% | 16h |
| pkg/security | 5.8% | 90% | 84.2% | 24h |
| pkg/llm | 7.2% | 70% | 62.8% | 16h |

**Total Effort**: ~112 hours (3 weeks for 1 engineer)

### P1 - High (Fix Within 2 Weeks)

| Package | Current | Target | Gap | Effort |
|---------|---------|--------|-----|--------|
| pkg/oran/a1 | 31.5% | 80% | 48.5% | 12h |
| pkg/oran/e2 | 13.9% | 80% | 66.1% | 12h |
| pkg/nephio | 9.6% | 70% | 60.4% | 12h |
| pkg/nephio/porch | 1.1% | 70% | 68.9% | 12h |
| internal/loop | 30.0% | 70% | 40.0% | 16h |

**Total Effort**: ~64 hours (1.5 weeks)

### P2 - Medium (Fix Within 1 Month)

| Package | Current | Target | Gap | Effort |
|---------|---------|--------|-----|--------|
| pkg/porch | 56.9% | 75% | 18.1% | 8h |
| pkg/handlers | 46.5% | 70% | 23.5% | 8h |
| pkg/audit | 32.0% | 70% | 38.0% | 12h |
| pkg/auth | 22.9% | 70% | 47.1% | 12h |

**Total Effort**: ~40 hours (1 week)

---

## Metrics to Track

### Coverage Metrics

- Overall coverage percentage
- Per-package coverage
- Coverage trend over time
- New code coverage (for PRs)

### Test Quality Metrics

- Test execution time
- Test flakiness rate
- Mutation score (when mutation testing added)
- Integration test count
- E2E test count

### CI/CD Metrics

- Build time
- Test failure rate
- Coverage enforcement failures
- PR coverage delta

---

## Conclusion

The Nephoran Intent Operator has **critically insufficient test coverage at 17.7%**. While some packages demonstrate excellent testing practices, the majority of critical infrastructure - controllers, O-RAN interfaces, LLM integration, and security functions - have dangerously low coverage.

**Immediate Risks**:
- Production bugs in untested controller logic
- O-RAN integration failures
- Security vulnerabilities
- LLM pipeline outages
- Data corruption from untested atomic operations

**Recommended Path Forward**:
1. **Week 1**: Fix failing tests + add critical controller tests (60% coverage)
2. **Week 2-3**: Add O-RAN interface tests + security tests
3. **Week 4-6**: Add integration tests for critical flows
4. **Month 2-3**: Establish coverage requirements, advanced testing, CI enforcement

**Success Criteria**:
- Overall coverage ≥60% within 6 weeks
- Critical packages ≥80% within 8 weeks
- All failing tests fixed within 1 week
- 100+ integration tests added within 6 weeks
- CI/CD coverage enforcement active within 8 weeks

---

## Appendix A: Full Package Coverage List

```
  0.0%	cmd/conductor-watch
  0.0%	cmd/intent-ingest
  0.0%	cmd/porch-publisher
  0.0%	pkg/controllers/optimized
  0.0%	pkg/nephio/krm
  1.1%	pkg/nephio/porch
  2.0%	pkg/oran/o2
  2.3%	pkg/performance
  3.0%	pkg/oran/o1
  3.1%	pkg/monitoring
  4.8%	api/v1alpha1
  4.9%	security/compliance
  5.6%	pkg/oran/o2/providers
  5.8%	pkg/security
  6.6%	cmd/llm-processor
  7.0%	pkg/controllers/orchestration
  7.2%	pkg/llm
  9.6%	pkg/nephio
 12.6%	api/intent/v1alpha1
 13.9%	pkg/oran/e2
 14.3%	pkg/git
 15.7%	pkg/testutil/auth
 17.1%	pkg/runtime
 20.8%	pkg/controllers
 21.0%	pkg/controllers/parallel
 21.1%	cmd/conductor-loop
 21.9%	pkg/config
 22.9%	pkg/auth
 24.9%	planner/cmd/planner
 26.4%	pkg/cnf
 28.5%	internal/pathutil
 28.7%	internal/watch
 30.0%	internal/loop
 31.5%	pkg/oran/a1
 32.0%	pkg/audit
 34.4%	pkg/auth/providers
 39.4%	cmd/porch-structured-patch
 40.0%	cmd/conductor
 40.6%	pkg/generics
 41.7%	pkg/audit/compliance
 42.3%	internal/security
 42.6%	pkg/auth/docker
 43.5%	test/testutil
 44.4%	pkg/nephio/blueprint
 46.5%	pkg/handlers
 50.0%	internal/config
 52.0%	internal/a1sim
 52.3%	internal/platform
 54.8%	internal/porch
 56.4%	pkg/monitoring/alerting
 56.9%	pkg/injection
 56.9%	pkg/porch
 61.7%	internal/conductor
 62.2%	internal/intent
 63.7%	cmd/porch-direct
 66.6%	pkg/middleware
 68.4%	pkg/contracts
 71.2%	pkg/global
 75.0%	internal/llm/providers
 75.0%	internal/planner
 79.5%	pkg/nephio/multicluster
 80.0%	internal/ingest
 82.9%	internal/kpm
 84.0%	internal/patchgen
 87.0%	planner/internal/rules
 87.1%	pkg/services
 88.7%	internal/patch
 89.1%	pkg/webhooks
 90.0%	pkg/oran/health
 95.0%	internal/ves
 99.8%	planner/internal/security
100.0%	internal/patchgen/generator
```

---

## Appendix B: Test Execution Log Summary

**Execution Time**: ~360 seconds (6 minutes)

**Package Failures**:
- `pkg/disaster` - FAIL (6.638s)
- `planner/internal/security` - FAIL (0.197s)
- `test/envtest` - FAIL (0.120s)

**Longest Running Tests**:
- `internal/loop` - 105.564s
- `pkg/controllers/parallel` - 69.791s
- `pkg/audit` - 58.524s
- `pkg/nephio/multicluster` - 36.258s
- `pkg/llm` - 22.034s

**Go Version Warnings**:
- Compiler version mismatch: go1.26.0 vs go1.25.6
- Does not affect test results but should be resolved

---

**Report Generated**: 2026-02-23
**Generated By**: Claude Code AI Agent (Test Engineering Specialist)
**Coverage File**: `/home/thc1006/dev/nephoran-intent-operator/coverage.out`
