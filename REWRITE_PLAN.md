# Git History Rewrite Plan - Nephoran Intent Operator

## Executive Summary

This plan outlines the complete process for rewriting git history to remove large binary files (~300MB) and migrate appropriate assets to Git LFS. The repository has already removed these files from the working tree, but they persist in git history, bloating the repository size.

## Current State Analysis

### Large Files in Git History
- **Total Size to Remove**: ~300MB
- **Primary Culprits**:
  - `bin/nephio-bridge`: 68.2MB (multiple versions)
  - `bin/oran-adaptor`: 66.4MB (multiple versions)
  - `bin/kpt`: 48.9MB
  - `kind`: 10.1MB
  - `bin/llm-processor`: 9.3MB (multiple versions)
  - Go module zips in history: ~40MB total

### Files for LFS Migration
Currently, no active binary files require LFS migration. The `.gitattributes` is properly configured for future binary files.

## Phase 1: Preparation and Backup

### 1.1 Create Full Backup
```bash
# Create mirror clone for backup
git clone --mirror https://github.com/thc1006/nephoran-intent-operator.git nephoran-backup
cd nephoran-backup
git bundle create ../nephoran-full-backup-$(date +%Y%m%d).bundle --all
cd ..

# Verify backup
git clone nephoran-full-backup-$(date +%Y%m%d).bundle test-restore
cd test-restore && git log --oneline -5 && cd ..
rm -rf test-restore
```

### 1.2 Tag Current State
```bash
# In main repository
git tag -a pre-history-rewrite-$(date +%Y%m%d) -m "Backup tag before history rewrite"
git push origin pre-history-rewrite-$(date +%Y%m%d)
```

### 1.3 Notify Team
- Send notification about maintenance window
- Request all team members to push their work
- Announce repository will be temporarily read-only

## Phase 2: History Rewrite

### 2.1 Install Required Tools
```bash
# Option A: git-filter-repo (recommended)
pip install git-filter-repo

# Option B: BFG Repo-Cleaner
wget https://repo1.maven.org/maven2/com/madgag/bfg/1.14.0/bfg-1.14.0.jar
alias bfg='java -jar bfg-1.14.0.jar'
```

### 2.2 Dry Run Strategy

#### Using git-filter-repo (Recommended)
```bash
# Clone fresh copy for dry run
git clone https://github.com/thc1006/nephoran-intent-operator.git nephoran-dry-run
cd nephoran-dry-run

# Analyze repository first
git filter-repo --analyze
# Check generated reports in .git/filter-repo/analysis/

# Dry run with --dry-run flag
git filter-repo --dry-run \
  --path bin/nephio-bridge --invert-paths \
  --path bin/oran-adaptor --invert-paths \
  --path bin/kpt --invert-paths \
  --path bin/llm-processor --invert-paths \
  --path kind --invert-paths \
  --path 'pkg/mod/cache/download/' --invert-paths \
  --path-glob '*.zip' --invert-paths \
  --path-glob 'golang.org/x/text/language/tables.go' --invert-paths

# Review what would be removed
cat .git/filter-repo/args
```

### 2.3 Actual History Rewrite

#### Using git-filter-repo
```bash
# Create clean working copy
git clone https://github.com/thc1006/nephoran-intent-operator.git nephoran-clean
cd nephoran-clean

# Remove large binaries from history
git filter-repo \
  --path bin/nephio-bridge --invert-paths \
  --path bin/oran-adaptor --invert-paths \
  --path bin/kpt --invert-paths \
  --path bin/llm-processor --invert-paths \
  --path kind --invert-paths \
  --path 'pkg/mod/' --invert-paths

# Remove go module cache files
git filter-repo \
  --path-glob '**/golang.org/x/text@*.zip' --invert-paths \
  --path-glob '**/github.com/weaviate/weaviate@*.zip' --invert-paths \
  --path-glob '**/google.golang.org/genproto@*.zip' --invert-paths \
  --path-glob '**/go.mongodb.org/mongo-driver@*.zip' --invert-paths

# Remove obsolete archive directory
git filter-repo --path archive/ --invert-paths

# Remove old cleanup summary files
git filter-repo \
  --path-glob '*_SUMMARY.md' --invert-paths \
  --path-glob '*_ANALYSIS.md' --invert-paths \
  --path-glob 'CLEANUP_*.md' --invert-paths
```

#### Alternative: Using BFG
```bash
# Clone mirror for BFG
git clone --mirror https://github.com/thc1006/nephoran-intent-operator.git nephoran-mirror
cd nephoran-mirror

# Remove folders
bfg --delete-folders bin --no-blob-protection
bfg --delete-folders archive --no-blob-protection
bfg --delete-folders '{pkg/mod,pkg/mod/cache}' --no-blob-protection

# Remove specific files
bfg --delete-files kind --no-blob-protection
bfg --delete-files '*.{zip}' --no-blob-protection

# Clean up
git reflog expire --expire=now --all
git gc --prune=now --aggressive
```

### 2.4 Verification
```bash
# Check repository size
du -sh .git
git count-objects -vH

# Verify files are gone from history
git log --all --oneline --name-only | grep -E "bin/|kind" | wc -l
# Should return 0

# Check for any remaining large blobs
git rev-list --objects --all | \
  git cat-file --batch-check='%(objecttype) %(objectname) %(objectsize) %(rest)' | \
  awk '/^blob/ && $3 > 1048576 {print $3, $4}' | \
  sort -k1 -n -r

# Run tests to ensure nothing broke
go test ./...
```

## Phase 3: Git LFS Setup (Future-Proofing)

### 3.1 Initialize Git LFS
```bash
# Install LFS in repository
git lfs install

# Track binary file patterns
git lfs track "*.exe"
git lfs track "*.dll"
git lfs track "*.so"
git lfs track "*.dylib"
git lfs track "*.png"
git lfs track "*.jpg"
git lfs track "*.jpeg"
git lfs track "*.gif"
git lfs track "*.pdf"
git lfs track "*.zip"
git lfs track "*.tar.gz"
git lfs track "*.tar"
git lfs track "bin/*"

# Commit LFS tracking
git add .gitattributes
git commit -m "chore: configure Git LFS tracking for binary files"
```

### 3.2 Migrate Existing Binaries to LFS (if any found)
```bash
# This will rewrite history to use LFS for matched files
git lfs migrate import --include="*.png,*.jpg,*.zip,*.tar.gz" --everything

# Verify migration
git lfs ls-files
```

## Phase 4: Push Strategy

### Option A: New Clean Branch (Recommended)
```bash
# Add new remote
git remote add origin-clean https://github.com/thc1006/nephoran-intent-operator.git

# Create new main-clean branch
git checkout -b main-clean

# Push with force (requires admin permissions)
git push origin-clean main-clean --force

# After verification, rename branches
git push origin-clean :main  # Delete old main
git push origin-clean main-clean:main  # Rename main-clean to main
```

### Option B: Temporary Branch Protection Relaxation
```bash
# 1. Go to GitHub Settings > Branches
# 2. Temporarily disable branch protection for 'main'
# 3. Push rewritten history
git push origin main --force-with-lease

# 4. Re-enable branch protection immediately
```

### Option C: Create New Repository
```bash
# 1. Create new repository: nephoran-intent-operator-clean
# 2. Push clean history
git remote add new-origin https://github.com/thc1006/nephoran-intent-operator-clean.git
git push new-origin --all
git push new-origin --tags

# 3. Archive old repository
# 4. Rename new repository to original name
```

## Phase 5: Post-Rewrite Tasks

### 5.1 Team Synchronization
```bash
# All team members must re-clone
git clone https://github.com/thc1006/nephoran-intent-operator.git nephoran-fresh

# Or force update existing clones
git fetch origin
git reset --hard origin/main
git clean -fdx
```

### 5.2 Update CI/CD
- Update any cached repository references
- Clear CI/CD caches
- Rebuild all containers
- Update deployment scripts with new commit SHAs if hardcoded

### 5.3 Documentation
```bash
# Create HISTORY_REWRITE_COMPLETED.md
cat > HISTORY_REWRITE_COMPLETED.md << EOF
# History Rewrite Completed - $(date +%Y-%m-%d)

## Changes Made
- Removed ~300MB of binary files from git history
- Configured Git LFS for future binary files
- Cleaned obsolete archive and summary files

## New Repository Size
- Before: XXX MB
- After: XXX MB
- Reduction: XX%

## Required Actions for Team Members
1. Re-clone the repository
2. Update any local scripts referencing old commit SHAs
3. Rebase any in-progress work on new history

## Backup Location
- Bundle: nephoran-full-backup-$(date +%Y%m%d).bundle
- Tag: pre-history-rewrite-$(date +%Y%m%d)
EOF

git add HISTORY_REWRITE_COMPLETED.md
git commit -m "docs: document history rewrite completion"
git push origin main
```

## Risk Assessment and Mitigation

### Risks
1. **Data Loss**: Accidental removal of needed files
   - **Mitigation**: Full backup bundle, tagged state, mirror clone
   
2. **Broken References**: External tools referencing old commit SHAs
   - **Mitigation**: Document all SHA changes, update CI/CD configs
   
3. **Team Disruption**: Developers with uncommitted work
   - **Mitigation**: Advance notice, maintenance window, clear instructions
   
4. **CI/CD Failures**: Build pipelines failing due to history change
   - **Mitigation**: Test builds before team notification, update caches

5. **LFS Quota**: GitHub LFS storage limits
   - **Mitigation**: Monitor usage, purchase additional storage if needed

### Rollback Plan

#### Immediate Rollback (Within 1 Hour)
```bash
# If issues discovered immediately
git push origin pre-history-rewrite-$(date +%Y%m%d):main --force
```

#### Full Rollback (Any Time)
```bash
# Restore from bundle
git clone nephoran-full-backup-$(date +%Y%m%d).bundle nephoran-restored
cd nephoran-restored
git remote add origin https://github.com/thc1006/nephoran-intent-operator.git
git push origin --all --force
git push origin --tags --force
```

## Success Metrics

### Expected Outcomes
- [ ] Repository size reduced by >70% (~300MB reduction)
- [ ] Clone time reduced by >50%
- [ ] No large blobs (>1MB) in git history except documentation
- [ ] Git LFS properly configured for future binaries
- [ ] All tests passing
- [ ] CI/CD pipelines green
- [ ] Team successfully migrated to clean history

## Timeline

### Estimated Duration: 4-6 Hours

1. **Preparation** (1 hour)
   - Create backups
   - Notify team
   - Set up tools

2. **Dry Run** (1 hour)
   - Test commands
   - Verify removals
   - Check for issues

3. **Execution** (1-2 hours)
   - Run history rewrite
   - Configure LFS
   - Verify results

4. **Push and Migration** (1-2 hours)
   - Push to remote
   - Update CI/CD
   - Team migration

5. **Verification** (1 hour)
   - Run tests
   - Check builds
   - Document completion

## Commands Summary Card

```bash
# Quick Reference - Copy and Run

# 1. Backup
git clone --mirror <repo-url> backup && cd backup
git bundle create ../backup.bundle --all && cd ..

# 2. Rewrite (git-filter-repo)
git filter-repo --path bin/ --invert-paths --path kind --invert-paths

# 3. LFS Setup
git lfs install
git lfs track "*.{exe,dll,so,png,jpg,zip}"
git lfs migrate import --include="*.png,*.jpg,*.zip" --everything

# 4. Push
git push origin main --force-with-lease

# 5. Verify
git count-objects -vH
git lfs ls-files
```

## Final Checklist

### Pre-Execution
- [ ] Backup bundle created and verified
- [ ] Pre-rewrite tag pushed
- [ ] Team notified with 24-hour notice
- [ ] Dry run completed successfully
- [ ] CI/CD update plan ready

### Post-Execution
- [ ] History rewrite completed
- [ ] LFS configured
- [ ] Force push successful
- [ ] Team migration instructions sent
- [ ] CI/CD pipelines updated and green
- [ ] Documentation committed
- [ ] Backup retained for 30 days

---

**Document Status**: READY FOR REVIEW  
**Author**: Repository Hygiene Team  
**Review Required By**: DevOps Lead, Security Lead  
**Approval Required By**: Repository Admin