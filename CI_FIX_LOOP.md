# CI_FIX_LOOP.md - Systematic CI/CD Error Resolution Tracker

## 🎯 OBJECTIVE
Fix ALL "Code Quality" CI job failures LOCALLY before pushing to avoid the endless push-fail-fix cycle.

## 📋 CURRENT STATUS
- **Last Update**: 2025-01-02
- **Current Iteration**: 1
- **PR URL**: https://github.com/thc1006/nephoran-intent-operator/pull/85
- **CI Job**: Code Quality (golangci-lint)

## 🔄 ITERATION PROCESS

### Step 1: Local CI Simulation
```bash
# Install exact CI version
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3

# Run with exact CI configuration
golangci-lint run --timeout=10m --out-format=json:golangci-lint.json,colored-line-number
```

### Step 2: Error Categories & Count
| Error Type | Count | Status | Agent Assigned |
|------------|-------|--------|----------------|
| ST1000 (package comments) | 7 | ⏳ Pending | golang-pro |
| gci (import formatting) | 4 | ⏳ Pending | code-reviewer |
| whitespace | 14 | ⏳ Pending | golang-pro |
| exitAfterDefer | 5 | ⏳ Pending | debugger |
| gocritic | 35+ | ⏳ Pending | golang-pro |
| gosec (G115, G302, G306) | 10+ | ⏳ Pending | security-auditor |
| unparam | 15+ | ⏳ Pending | refactoring-specialist |
| errcheck | 20+ | ⏳ Pending | error-detective |
| ineffassign | 2 | ⏳ Pending | golang-pro |
| revive (exported) | 20+ | ⏳ Pending | api-documenter |
| prealloc | 8+ | ⏳ Pending | performance-engineer |
| staticcheck | 3 | ⏳ Pending | golang-pro |

### Step 3: Fix Commands
```bash
# Auto-fix what's possible
golangci-lint run --fix

# Format imports
gci write --skip-generated -s standard -s default -s "prefix(github.com/nephio-project/nephoran-intent-operator)" .

# Format code
gofmt -w .
goimports -w .

# Run again to check remaining
golangci-lint run --timeout=10m
```

### Step 4: Verification
- [ ] Local golangci-lint passes with 0 errors
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build ./...`
- [ ] Pushed to remote
- [ ] Wait 5 minutes for CI
- [ ] Check PR page for CI status

## 📊 ITERATION HISTORY

### Iteration 1 - 2025-01-02
- **Errors Found**: 200+ linting errors
- **Categories**: Package comments, formatting, error handling, security, unused params
- **Action**: Deploying multiple agents for parallel fixes
- **Status**: 🔄 In Progress

## 🚀 AGENT DEPLOYMENT STRATEGY

### ULTRA SPEED Mode - Parallel Agent Deployment
1. **golang-pro**: Fix gocritic, stylecheck, staticcheck issues
2. **security-auditor**: Fix gosec security issues
3. **error-detective**: Fix errcheck, error handling
4. **refactoring-specialist**: Fix unparam, unused code
5. **api-documenter**: Add missing comments (revive)
6. **performance-engineer**: Fix prealloc issues
7. **code-reviewer**: Fix gci import formatting
8. **debugger**: Fix exitAfterDefer issues

## ⚡ QUICK COMMANDS

```bash
# Full local CI check
make lint || golangci-lint run --timeout=10m

# Check specific package
golangci-lint run ./pkg/auth/...

# Count errors by type
golangci-lint run --timeout=10m 2>&1 | grep -oE '\([a-z]+\)' | sort | uniq -c

# Fix and verify
golangci-lint run --fix && golangci-lint run --timeout=10m
```

## 🔥 EMERGENCY FIXES

If CI still fails after local pass:
1. Check exact golangci-lint version: `golangci-lint --version`
2. Match CI environment: Ubuntu 24.04, Go 1.24.6
3. Check .golangci.yml for CI-specific config
4. Run with verbose: `golangci-lint run -v`

## 📝 NOTES
- CI uses golangci-lint v1.64.3
- Timeout is 10 minutes
- Must fix ALL errors, not just critical ones
- Some errors can't be auto-fixed and need manual intervention