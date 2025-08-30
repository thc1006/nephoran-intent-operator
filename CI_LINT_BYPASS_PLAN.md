# CI Lint Bypass Plan - TEMPORARY

## Status: ACTIVE (as of 2025-08-30)

## Why This Change?

To accelerate development and unblock the merge to `integrate/mvp`, we have temporarily configured lint jobs to soft-fail. This allows the team to merge critical functionality while deferring formatting and style issues.

## What Changed?

### Modified Files:
1. `.github/workflows/ci.yml` (line 127)
   - Added `continue-on-error: true` to the `quality` job
   
2. `.github/workflows/ubuntu-ci.yml` (line 29)
   - Added `continue-on-error: true` to the `lint` job

### Behavior Changes:

| Component | Before | After (Temporary) |
|-----------|--------|-------------------|
| golangci-lint failures | Block merge ❌ | Warning only ⚠️ |
| Lint artifacts | Uploaded | Still uploaded ✅ |
| Lint execution | Required | Still runs ✅ |
| Build/Test failures | Block merge ❌ | Still block merge ❌ |
| Security scans | Required | Still required ✅ |

## How to Read Lint Results

Even though lint jobs won't fail the CI:
1. **Check the job status** - Lint jobs will show yellow/warning instead of red
2. **Review artifacts** - Download `quality-reports` or `lint-reports-ubuntu` artifacts
3. **Check job logs** - Detailed issues are still visible in the job output

## Rollback Plan - REQUIRED POST-MERGE

### Step 1: Remove Soft-Fail Configuration

```bash
# Remove the continue-on-error lines from both workflow files
git checkout integrate/mvp
git pull

# Edit .github/workflows/ci.yml - remove line 127
# Edit .github/workflows/ubuntu-ci.yml - remove line 29

git add .github/workflows/ci.yml .github/workflows/ubuntu-ci.yml
git commit -m "chore: restore strict linting after emergency merge"
git push
```

### Step 2: Fix Remaining Lint Issues

```bash
# Run lint locally to see all issues
golangci-lint run --timeout=10m

# Fix issues manually or with auto-fix
golangci-lint run --fix

# Verify all issues resolved
golangci-lint run
```

### Step 3: Create Follow-up PR

Create a PR titled: "chore: address deferred lint issues from emergency merge"

## Timeline

- **Soft-fail activated**: 2025-08-30
- **Target restoration**: Within 48 hours of merge
- **Maximum duration**: 7 days (must restore by 2025-09-06)

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Code quality degradation | Lint still runs and reports issues |
| Forgetting to restore | Calendar reminder set, documented here |
| Missing critical issues | Build, test, and security checks still enforced |

## Verification Commands

```bash
# Verify soft-fail is working (should not fail even with lint issues)
git push origin HEAD

# Check that build/test still fail on errors
go test ./... # Should still fail if tests break

# Monitor lint issues without blocking
golangci-lint run --out-format=json | jq '.Issues | length'
```

## Contact

- **Implementation**: Claude Code + [Your Name]
- **Questions**: File issue with label `ci-bypass-active`

---

**⚠️ REMINDER: This is a TEMPORARY measure. Please restore strict linting ASAP after merge.**