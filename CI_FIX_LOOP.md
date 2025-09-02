# CI_FIX_LOOP.md - Systematic CI Lint Fix Tracker

## Mission
Fix ALL "CI (Optimized Matrix) / Lint" failures locally BEFORE pushing to avoid the endless push-fail-fix loop.

## Current Status
- **PR**: #169
- **Branch**: feat/e2e
- **Date Started**: 2025-09-02
- **Current Iteration**: 1

## Local Lint Command
```bash
# Run the exact same lint that CI runs
golangci-lint run --timeout=10m --config=.golangci.yml ./...
```

## Error Tracking

### Iteration 1 - Starting Fresh
- [ ] Run local lint check
- [ ] Capture ALL errors
- [ ] Research fixes with search-specialist
- [ ] Apply fixes with multiple agents
- [ ] Verify locally until ZERO errors
- [ ] Push ONLY when local is clean
- [ ] Check CI status
- [ ] If CI fails, capture new errors and repeat

## Fixed Issues Log
(Will be updated with each successful fix)

## Remaining Issues
(To be populated after first local run)

## Research Notes
(Solutions verified by search-specialist will be documented here)

## Commands Used
```bash
# Install golangci-lint if needed
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run lint locally
golangci-lint run --timeout=10m ./...

# Run with specific linters
golangci-lint run --enable-all --timeout=10m ./...

# Run only on changed files
golangci-lint run --new-from-rev=main --timeout=10m ./...
```

## Success Criteria
- ✅ Local lint passes with ZERO errors
- ✅ All tests compile
- ✅ CI Lint job passes
- ✅ No more push-fail-fix loops

---
Last Updated: 2025-09-02 - Iteration 1 Starting
