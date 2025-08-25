# Branch Protection Rules

This document defines the branch protection rules for the Nephoran Intent Operator repository.

## Protected Branches

### Main
Primary production branch with the following protections:

**Status Checks Required:**
- `ci-status` - Must pass before merge
- `hygiene` - Repository cleanliness check
- `build` - Binary compilation
- `test` - Unit and integration tests

**Merge Requirements:**
- ✅ Require status checks to pass before merging
- ✅ Require branches to be up to date before merging
- ✅ Include administrators in restrictions
- ⚠️ Code review not required (single developer mode)
  - Will enable when team grows

**Push Restrictions:**
- Direct pushes disabled (must use PRs)
- Force pushes disabled

### integrate/mvp
Integration branch for MVP features:

**Status Checks Required:**
- `integrate-mvp-gate` - Comprehensive validation gate
- `ci-status` - Core pipeline status
- All module-specific tests (when triggered)

**Merge Requirements:**
- ✅ Require status checks to pass
- ✅ Require linear history
- ✅ Dismiss stale reviews when new commits pushed
- ⚠️ Code review not required (single developer mode)

## GitHub Settings Configuration

To apply these rules via GitHub UI:

1. Go to Settings → Branches
2. Add rule for `main`:
   ```
   - Pattern: main
   - Required status checks: ci-status, hygiene, build, test
   - Dismiss stale reviews: Yes
   - Include administrators: Yes
   - Restrict force pushes: Yes
   ```

3. Add rule for `integrate/mvp`:
   ```
   - Pattern: integrate/mvp
   - Required status checks: integrate-mvp-gate, ci-status
   - Require linear history: Yes
   - Include administrators: Yes
   ```

## GitHub CLI Configuration

```bash
# Configure main branch protection
gh api repos/:owner/:repo/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["ci-status","hygiene","build","test"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews=null \
  --field restrictions=null

# Configure integrate/mvp branch protection
gh api repos/:owner/:repo/branches/integrate/mvp/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["integrate-mvp-gate","ci-status"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews=null \
  --field required_linear_history=true
```

## Rulesets (Alternative to Branch Protection)

For more granular control, use GitHub Rulesets:

### Ruleset: Production Protection
**Target:** `main`
- Restrict deletions
- Require status checks
- Block force pushes
- Require PRs before merge

### Ruleset: Integration Protection
**Target:** `integrate/*`
- Require linear history
- Require status checks
- Tag protection for releases

## Status Check Jobs

The following CI jobs serve as status checks:

| Job Name | Purpose | Required For |
|----------|---------|--------------|
| hygiene | Repository cleanliness | main |
| build | Compilation check | main |
| test | Test suite | main |
| ci-status | Overall CI gate | main, integrate/mvp |
| integrate-mvp-gate | Integration validation | integrate/mvp |

## Future Enhancements

When the team grows:
1. Enable required reviews (1-2 reviewers)
2. Enable CODEOWNERS enforcement
3. Add security scanning requirements
4. Add deployment environment protections