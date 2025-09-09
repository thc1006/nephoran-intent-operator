# Branch Protection Configuration - Post CI Consolidation

## 🎯 Required Changes for Clean CI

After consolidating to a single workflow, update GitHub branch protection rules to reference the correct status checks.

## 📋 Branch Protection Settings

### **Branches to Protect**
- `main`
- `integrate/mvp`
- `integrate/**` (pattern)

### **Required Status Checks** 

#### **Primary (Required)**
```
✅ Integration Status
```
*This is the final status from nephoran-ci-consolidated-2025.yml*

#### **Optional (Recommended)**
```
- 🔧 Environment Setup
- 📦 Dependency Resolution  
- ⚡ Fast Validation
- 🏗️ Full Build & Integration
```

#### **Remove These Old Status Checks**
```
❌ CI Status Check (from ci-optimized.yml - now disabled)
❌ Main CI (Optimized) - DISABLED
❌ Production CI Pipeline  
❌ CI Reliability Optimized
❌ Kubernetes Operator CI 2025
❌ Nephoran Production CI
❌ PR Validation
❌ Parallel Test Execution
❌ Ultra-Optimized Go CI
```

## 🔧 How to Update (GitHub Web UI)

1. **Navigate**: Repository → Settings → Branches
2. **Edit Branch Protection Rule** for `main`
3. **Required Status Checks**:
   - ✅ Enable "Require status checks to pass before merging"
   - ✅ Enable "Require branches to be up to date before merging"  
   - **Add**: `Integration Status`
   - **Remove**: All old workflow status checks listed above
4. **Repeat** for `integrate/mvp` and `integrate/**`

## 🔧 Alternative: GitHub CLI

```bash
# Update main branch protection
gh api repos/:owner/:repo/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["Integration Status"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1}' \
  --field restrictions=null

# Update integrate/mvp branch protection  
gh api repos/:owner/:repo/branches/integrate%2Fmvp/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["Integration Status"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1}' \
  --field restrictions=null
```

## ⚠️ Important Notes

### **Status Check Name Mapping**
The workflow job names become the status check names:
- `integration-status` job → "Integration Status" status check
- `setup` job → "🔧 Environment Setup" status check  
- `dependencies` job → "📦 Dependency Resolution" status check

### **Testing Status Checks**
1. Create a test PR after updating branch protection
2. Verify only the new status checks appear
3. Confirm PR can merge when "Integration Status" passes

### **Emergency Override**
If branch protection blocks critical fixes:
1. Temporarily disable "Required status checks" in branch protection
2. Merge the fix
3. Re-enable protections immediately after

## 📊 Expected Result
- ✅ Clean CI status: 1 workflow, 1 final status check
- ✅ No more "6 failed checks" from old workflows  
- ✅ Faster PR feedback (no conflicting workflows)
- ✅ Reliable CI (25min timeout vs 5min failures)