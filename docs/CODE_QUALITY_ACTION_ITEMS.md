# Code Quality Action Items - Quick Reference

**Date**: 2026-02-23
**Status**: Active remediation tracking

---

## Phase 1: Critical Fixes (Target: 2 weeks)

### Week 1

- [ ] **A1: Fix context.TODO() in production code** (3 hours)
  - [ ] `pkg/disaster/backup_manager.go` - Add ctx parameter
  - [ ] `pkg/disaster/failover_manager.go` - Add ctx parameter
  - [ ] `pkg/disaster/restore_manager.go` - Add ctx parameter
  - [ ] `pkg/context/enhanced_context.go` - Review usage
  - [ ] `pkg/security/ca/certificate_pool.go` - Review usage

- [ ] **A2: Add HTTP client timeouts** (2 hours)
  - [ ] `controllers/networkintent_controller.go` - Add timeout
  - [ ] `pkg/oran/a1/server.go` - Add timeout
  - [ ] `pkg/porch/client.go` - Add timeout
  - [ ] Audit remaining 15 files

- [ ] **A3: Fix unchecked error returns** (1 hour)
  - [ ] `cmd/llm-processor/service_manager.go` - 6 json.Encoder errors
  - [ ] `cmd/performance-comparison/main.go` - 3 errors
  - [ ] `cmd/quality-metrics/main.go` - 2 errors

### Week 2

- [ ] **A4: Add disaster recovery tests** (1 week)
  - [ ] `pkg/disaster/backup_manager_test.go` - Create
  - [ ] `pkg/disaster/failover_manager_test.go` - Create
  - [ ] `pkg/disaster/restore_manager_test.go` - Create
  - [ ] Target: 80% coverage for pkg/disaster/*

- [ ] **A5: Security audit** (1 hour)
  - [ ] Run `git-secrets` scanner
  - [ ] Run `govulncheck ./...`
  - [ ] Run `gosec ./...`
  - [ ] Review findings and create issues

---

## Phase 2: Code Health (Target: 4 weeks)

### Weeks 3-4

- [ ] **B1: Consolidate LLM clients** (1 week)
  - [ ] Audit all client implementations
  - [ ] Design unified client interface
  - [ ] Migrate to single implementation
  - [ ] Remove duplicate files
  - [ ] Update all consumers

- [ ] **B2: Refactor internal/loop/watcher.go** (1 week)
  - [ ] Extract validator.go (800 lines)
  - [ ] Extract processor.go (600 lines)
  - [ ] Extract metrics.go (300 lines)
  - [ ] Extract debouncer.go (200 lines)
  - [ ] Update tests

### Weeks 5-6

- [ ] **B3: OpenAI provider decision** (3 days)
  - [ ] Evaluate: Implement vs Remove
  - [ ] If implement: Add OpenAI SDK integration
  - [ ] If remove: Delete file, update docs
  - [ ] Remove 16 TODOs

- [ ] **B4: Delete dead code** (5 minutes)
  - [ ] Delete `cmd/test-lint/main.go`
  - [ ] Remove from Makefile
  - [ ] Update documentation

- [ ] **B5: Goroutine lifecycle audit** (2 days)
  - [ ] Review all 602 `go func` calls
  - [ ] Ensure context cancellation
  - [ ] Add proper shutdown
  - [ ] Document lifecycle

---

## Phase 3: Technical Debt (Target: 8 weeks)

### Ongoing

- [ ] **C1: TODO remediation** (2 months, 20 TODOs/week)
  - Week 7-8: High-priority TODOs (87 items)
  - Week 9-12: Medium-priority (200 items)
  - Week 13-14: Low-priority cleanup (148 items)

- [ ] **C2: Logging standardization** (1 week)
  - [ ] Replace all `log.*` with `slog`
  - [ ] Add structured fields
  - [ ] Create logging guidelines
  - [ ] Update all packages

- [ ] **C3: Replace time.Sleep** (1 week)
  - [ ] Audit 233 instances
  - [ ] Replace with proper sync primitives
  - [ ] Document acceptable uses

- [ ] **C4: Increase test coverage** (1 month)
  - Current: ~19%
  - Target: 40%
  - Priority: pkg/disaster, pkg/oran, pkg/nephio

- [ ] **C5: Dependency optimization** (1 week)
  - [ ] Add build tags for cloud providers
  - [ ] Remove unused dependencies
  - [ ] Update go.mod

---

## Quick Wins (Can do today)

- [ ] Delete `cmd/test-lint/main.go` (5 min)
- [ ] Fix 3 context.TODO() in test files (15 min)
- [ ] Add timeout to controller HTTP client (10 min)
- [ ] Run golangci-lint and fix top 10 issues (1 hour)

---

## Automated Checks to Add

### Pre-commit Hooks
```bash
# Add to .git/hooks/pre-commit
golangci-lint run --new-from-rev=HEAD~1
go test ./... -short
govulncheck ./...
```

### CI/CD Pipeline
```yaml
- name: Code Quality Gates
  run: |
    # Fail if TODO count increases
    # Fail if test coverage decreases
    # Fail if new context.TODO() added
    # Fail if HTTP clients lack timeouts
```

---

## Success Metrics

| Week | TODOs Remaining | Test Coverage | context.TODO() | Critical Issues |
|------|----------------|---------------|----------------|-----------------|
| 0 (now) | 435 | 19% | 26 | 52 |
| 2 | 415 | 25% | 5 (prod) | 0 |
| 6 | 350 | 30% | 0 (prod) | 0 |
| 14 | 200 | 40% | 0 (all) | 0 |

---

## Notes

- Priority: P0 (critical) > P1 (high) > P2 (medium)
- Effort: Hours (h), Days (d), Weeks (w), Months (m)
- Track progress in weekly team meetings
- Update this document as items complete

---

**Last Updated**: 2026-02-23
**Owner**: Development Team
**Review Frequency**: Weekly
