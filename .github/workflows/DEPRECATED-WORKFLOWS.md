# Deprecated Workflows - Migration Guide

## 🚨 WORKFLOW DEPRECATION NOTICE

The following GitHub Actions workflows have been **deprecated** as of 2025-01-XX and will be removed in a future update. Please migrate to the new 2025-optimized workflows.

## Deprecated Workflows

### Replaced by `ci-2025.yml` (Comprehensive CI)
- ❌ `ci-optimized.yml` 
- ❌ `ci-production.yml`
- ❌ `ci-timeout-fix.yml`
- ❌ `main-ci-optimized.yml`
- ❌ `ubuntu-ci.yml`
- ❌ `parallel-tests.yml`

### Replaced by `pr-validation.yml` (Fast PR Checks)  
- ❌ `pr-ci-fast.yml`
- ❌ `dev-fast.yml`
- ❌ `dev-fast-fixed.yml`

### Replaced by `production-ci.yml` (Production Builds)
- ❌ `ci-production.yml`
- ❌ `emergency-merge.yml`

### Keep as-is (Specialized)
- ✅ `debug-ghcr-auth.yml` (debugging only)

## Migration Benefits

### 🎯 Issues Resolved in New Workflows

1. **Cache Key Generation Failures**
   - ✅ Robust cache key generation with fallbacks
   - ✅ SHA-based keys with validation
   - ✅ Prevents empty cache key errors

2. **File Permission Issues**
   - ✅ Proper cache directory cleanup
   - ✅ Explicit permission setting (chmod 755)
   - ✅ Sudo cleanup for cache conflicts

3. **Tar Extraction Errors**
   - ✅ Clean cache directories before restore
   - ✅ Prevent overlapping cache operations
   - ✅ Manual cache control vs auto-caching

4. **Build Timeout Issues**
   - ✅ Comprehensive timeout configurations (15-25 minutes)
   - ✅ Per-component timeout strategies
   - ✅ Intelligent build grouping

5. **Go 1.25.0 Optimization**
   - ✅ Consistent Go 1.25.0 across all workflows
   - ✅ Latest GitHub Actions (checkout@v4, setup-go@v5, cache@v4)
   - ✅ Optimized build flags for Go 1.25

6. **Security & 2025 Best Practices**
   - ✅ Minimal required permissions
   - ✅ OIDC-ready configuration
   - ✅ Comprehensive security scanning
   - ✅ Vulnerability assessment

7. **Ubuntu Linux Focus**
   - ✅ Removed all cross-platform code
   - ✅ Ubuntu-optimized configurations
   - ✅ Linux-specific optimizations

## New Workflow Structure

```
.github/workflows/
├── ci-2025.yml          # 🆕 Comprehensive CI for all branches
├── pr-validation.yml    # 🆕 Fast PR validation (<5 min)
├── production-ci.yml    # 🆕 Production-ready builds (main/integrate)
└── debug-ghcr-auth.yml  # ✅ Keep for debugging
```

## Quick Migration Commands

### For Feature Branches
Replace any manual workflow triggers with:
```bash
# Triggers pr-validation.yml automatically
git push origin feature/your-branch

# For comprehensive testing
gh workflow run ci-2025.yml
```

### For Production Branches (main/integrate/mvp)
```bash
# Triggers production-ci.yml automatically
git push origin main

# Manual production build with options
gh workflow run production-ci.yml \
  --field build_mode=full \
  --field enable_benchmarks=true
```

## Key Improvements Summary

| Issue | Old Workflows | New Workflows |
|-------|---------------|---------------|
| Cache failures | ❌ Empty keys, conflicts | ✅ Robust SHA-based keys |
| Build timeouts | ❌ 10-15 min limits | ✅ 15-25 min with intelligence |
| Go version | ❌ Mixed 1.24/1.25 | ✅ Consistent 1.25.0 |
| Permissions | ❌ Overly broad | ✅ Minimal required |
| Platform support | ❌ Cross-platform complexity | ✅ Ubuntu-only optimization |
| Error handling | ❌ Fail-fast | ✅ Graceful degradation |
| Security | ❌ Basic | ✅ Comprehensive scanning |
| Reliability | ❌ Frequent failures | ✅ Production-grade stability |

## Timeline for Deprecation

- **2025-01-XX**: New workflows introduced
- **2025-02-XX**: Old workflows marked deprecated (this notice)  
- **2025-03-XX**: Old workflows will be removed from repository

## Support

If you encounter issues with the new workflows, please:
1. Check this migration guide first
2. Review the new workflow logs for detailed error information
3. Open an issue with specific error details
4. Tag @deployment-team for urgent production issues

---

**Note**: The new workflows are designed to be more reliable, faster, and secure. They address all known issues from the previous generation while providing better visibility and control.