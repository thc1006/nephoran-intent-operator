# CI Fix Loop - golangci-lint v2 Failures (URGENT)

## Objective
**ELIMINATE ALL golangci-lint v2 CI failures permanently** through iterative local debugging, fixing, and PR validation.

## Strategy
1. **Local Mirror CI**: Run exact CI commands locally
2. **Multi-Agent Research**: Use search-specialist before every fix
3. **Iterative Loop**: Fix → Test → Push → Wait 5min → Check PR → Repeat
4. **Zero Tolerance**: Continue until ALL CI jobs pass

## Current Status: STARTING

### Target CI Job
- **Job Name**: "Code Quality (golangci-lint v2)"
- **Command**: `golangci-lint run --timeout=10m --out-format=github-actions ./...`
- **Current State**: FAILING

---

## Iteration Log

### Iteration #1 - Initial Assessment
**Started**: 2025-08-28 (URGENT MODE)
**Status**: IN PROGRESS
**Goal**: Mirror CI environment and capture all golangci-lint errors

#### Step 1: Local CI Mirror Setup
```bash
# Exact CI environment replication
go version  # Should match CI: go1.24.x
golangci-lint version  # Should match CI: v1.64.7+
golangci-lint run --timeout=10m --out-format=github-actions ./...
```

#### Step 2: Error Capture
(Errors will be logged here)

#### Step 3: Research & Fix
(Each error researched with search-specialist first)

#### Step 4: Validation
- [ ] Local golangci-lint passes
- [ ] Push to PR
- [ ] Wait 5 minutes
- [ ] Check all CI jobs
- [ ] Repeat if any failures

---

## Multi-Agent Coordination Plan

### Phase 1: Local Mirroring
- **search-specialist**: Research golangci-lint v2 exact CI setup
- **devops-troubleshooter**: Mirror CI environment locally
- **error-detective**: Capture and categorize all errors

### Phase 2: Systematic Fixes
- **golang-pro**: Fix Go-specific linting issues
- **security-auditor**: Fix security-related linting issues  
- **code-reviewer**: Fix code quality issues
- **performance-engineer**: Fix performance-related issues

### Phase 3: PR Validation Loop
- **deployment-engineer**: Handle push and PR monitoring
- **context-manager**: Coordinate multi-iteration state
- **debugger**: Debug any CI-specific failures

---

## Success Criteria
✅ `golangci-lint run --timeout=10m --out-format=github-actions ./...` = 0 errors locally  
✅ All CI jobs pass in PR  
✅ No timeout issues  
✅ No environment-specific failures  

## Timeout Strategy
- Start: 10m (current CI timeout)
- Increase: 15m, 20m, 30m as needed per iteration
- Final: Lock in optimal timeout for CI

**STATUS**: LAUNCHING MULTI-AGENT ASSAULT 🚀