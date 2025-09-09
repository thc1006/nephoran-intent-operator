# EMERGENCY CI FIX PLAN - ITERATION 5

## 🚨 CRITICAL CI FAILURES IDENTIFIED

### ROOT CAUSES:
1. **GO_VERSION MISMATCHES**: 17 workflows using outdated Go 1.22.7 vs go.mod requiring 1.24.6
2. **CONCURRENCY CONFLICTS**: Multiple different concurrency group patterns causing job cancellations  
3. **CACHE KEY CONFLICTS**: 5+ different cache key formats causing cache thrashing
4. **RACE DETECTION FAILURES**: CGO_ENABLED=0 conflicts with race detection requirements
5. **ACTION VERSION INCOMPATIBILITIES**: Mixed action versions causing workflow failures

### EMERGENCY FIXES:

#### PHASE 1: STANDARDIZE GO VERSION (IMMEDIATE)
```yaml
# STANDARDIZE ALL WORKFLOWS TO:
env:
  GO_VERSION: "1.24.6"  # Match go.mod toolchain
```

#### PHASE 2: UNIFY CONCURRENCY GROUPS 
```yaml
# STANDARDIZE ALL WORKFLOWS TO:
concurrency:
  group: nephoran-ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

#### PHASE 3: STANDARDIZE CACHE KEYS
```yaml
# STANDARDIZE ALL WORKFLOWS TO:
key: nephoran-v5-${{ runner.os }}-go${{ env.GO_VERSION }}-${{ hashFiles('**/go.sum') }}-${{ hashFiles('go.mod') }}
```

#### PHASE 4: FIX RACE DETECTION
```yaml
# FOR RACE DETECTION JOBS ONLY:
env:
  CGO_ENABLED: "1"
steps:
  - name: Install build dependencies
    run: sudo apt-get update && sudo apt-get install -y gcc libc6-dev
```

#### PHASE 5: UPDATE ACTION VERSIONS
```yaml
# STANDARDIZE ALL WORKFLOWS TO:
- uses: actions/checkout@v4
- uses: actions/setup-go@v5  
- uses: actions/cache@v4
- uses: actions/upload-artifact@v4
- uses: actions/download-artifact@v4
```

### FILES REQUIRING IMMEDIATE FIXES:
1. `.github/workflows/ci-production.yml` ✓ (already correct GO_VERSION)
2. `.github/workflows/ci-reliability-optimized.yml` ✓ (already correct GO_VERSION)
3. `.github/workflows/ci-2025.yml` ✓ (already correct GO_VERSION)
4. `.github/workflows/ci-ultra-optimized-2025.yml` ❌ (needs GO_VERSION fix)
5. `.github/workflows/cache-recovery-system.yml` ❌ (needs GO_VERSION fix)
6. `.github/workflows/ci-timeout-fix.yml` ❌ (needs GO_VERSION fix)
7. All other workflows with GO_VERSION: "1.22.7" ❌

### SUCCESS CRITERIA:
- All workflows use GO_VERSION: "1.24.6"
- Unified concurrency groups prevent job cancellations
- Standardized cache keys eliminate conflicts
- Race detection jobs have CGO_ENABLED=1 with gcc installed
- All action versions are latest stable

### EXECUTION ORDER:
1. Fix GO_VERSION in all workflows (highest priority)
2. Standardize concurrency groups
3. Unify cache key formats
4. Fix race detection configuration
5. Test with single workflow first, then propagate