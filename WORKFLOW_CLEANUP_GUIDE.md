# GitHub Actions Workflow Cleanup Implementation Guide

## Overview

This guide provides step-by-step instructions for consolidating the 24 existing GitHub Actions workflows down to 8 essential workflows while maintaining all critical functionality.

## Pre-Implementation Checklist

- [ ] Create a backup branch: `git checkout -b workflow-cleanup-backup && git push origin workflow-cleanup-backup`
- [ ] Review current workflow dependencies and integrations
- [ ] Ensure all necessary secrets are configured
- [ ] Notify team members about the planned changes

## Implementation Steps

### Phase 1: Backup and Branch Creation (Estimated time: 15 minutes)

```bash
# 1. Create backup and working branches
git checkout -b workflow-cleanup-backup
git push origin workflow-cleanup-backup
git checkout -b workflow-consolidation

# 2. Make cleanup script executable
chmod +x scripts/cleanup-workflows.sh

# 3. Run the automated cleanup script
./scripts/cleanup-workflows.sh
```

### Phase 2: Manual Verification and Testing (Estimated time: 2 hours)

1. **Verify New Workflows**
   ```bash
   # Check that new workflows are properly formatted
   find .github/workflows -name "*.yml" | xargs -I {} yaml-lint {}
   
   # Validate workflow syntax (if you have act installed)
   act --list
   ```

2. **Review Workflow Dependencies**
   - Check that all referenced secrets exist in GitHub Settings
   - Verify that all required GitHub Apps/permissions are configured
   - Ensure environment variables are properly set

3. **Test Key Workflows**
   ```bash
   # Test main CI pipeline (dry run)
   git add .github/workflows/main-ci.yml
   git commit -m "feat: add consolidated main CI pipeline"
   git push origin workflow-consolidation
   
   # Create a test PR to verify the workflow triggers
   ```

### Phase 3: Gradual Migration (Estimated time: 1 day)

1. **Day 1 Morning: Deploy Core Workflows**
   - Add `main-ci.yml`
   - Add `security.yml` 
   - Test with a small PR

2. **Day 1 Afternoon: Deploy Supporting Workflows**
   - Add `docs.yml`
   - Add `release.yml`
   - Add `dependabot.yml`

3. **Day 2: Deploy Specialized Workflows**
   - Replace `nightly.yml` with `nightly-simple.yml`
   - Simplify `production.yml` (manual review required)
   - Remove all deprecated workflows

### Phase 4: Clean Up and Documentation (Estimated time: 4 hours)

1. **Remove Old Workflows**
   ```bash
   # Remove redundant workflows (the script handles this)
   ./scripts/cleanup-workflows.sh
   
   # Commit the changes
   git add .github/workflows/
   git commit -m "feat: consolidate GitHub Actions workflows from 24 to 8"
   ```

2. **Update Documentation**
   - Update README.md with new workflow descriptions
   - Update any CI/CD documentation
   - Update contributor guidelines

3. **Final Testing**
   - Create test PRs to verify all workflows
   - Test release process (use a test tag)
   - Verify security scans run correctly

## New Workflow Structure

| Workflow | File | Triggers | Purpose |
|----------|------|----------|---------|
| **Main CI/CD** | `main-ci.yml` | Push to main, PRs | Core build, test, lint, deploy |
| **Security Scanning** | `security.yml` | Daily, PRs, pushes | SAST, dependency scan, secrets |
| **Documentation** | `docs.yml` | Doc changes, releases | Build and deploy docs |
| **Release Management** | `release.yml` | Tags, manual | Create releases, build assets |
| **Dependency Updates** | `dependabot.yml` | Weekly schedule | Update dependencies |
| **Nightly Build** | `nightly-simple.yml` | Daily 2 AM UTC | Basic build and test |
| **Production Deploy** | `production.yml` | Release tags | Production deployment |
| **Dependabot Config** | `dependabot.yml` | Auto dependency PRs | Manage dependencies |

## Key Changes and Benefits

### Before (24 workflows)
- Redundant CI pipelines
- Multiple security scan workflows
- Fragmented documentation builds  
- Complex dependency management
- Maintenance overhead

### After (8 workflows)
- Single consolidated CI/CD pipeline
- Unified security scanning
- Streamlined documentation
- Automated dependency management
- Reduced complexity by 70%

## Migration Validation Checklist

- [ ] All workflows use latest action versions (v4, v5)
- [ ] Node.js standardized on v20
- [ ] Go version standardized on 1.24
- [ ] Security scans still function correctly
- [ ] Documentation builds successfully
- [ ] Release process works end-to-end
- [ ] Notifications work (Slack, webhooks)
- [ ] Artifact retention policies are appropriate

## Troubleshooting Common Issues

### Issue: Workflow syntax errors
**Solution:**
```bash
# Install yamllint
pip install yamllint

# Check all workflow files
find .github/workflows -name "*.yml" -exec yamllint {} \;
```

### Issue: Missing secrets or environment variables
**Solution:**
1. Check GitHub Settings → Secrets and variables → Actions
2. Verify all required secrets exist:
   - `GCP_SA_KEY`
   - `SLACK_WEBHOOK_URL`
   - `RELEASE_WEBHOOK_URL`
   - `SONAR_TOKEN` (if using SonarCloud)

### Issue: Permissions errors
**Solution:**
1. Verify GITHUB_TOKEN permissions in workflow files
2. Check repository settings for Actions permissions
3. Ensure required permissions are granted:
   - `contents: read/write`
   - `pages: write`
   - `id-token: write`

### Issue: Build failures after consolidation
**Solution:**
1. Check the build logs for specific error messages
2. Compare with previous working workflows in backup
3. Ensure all required tools are installed in the workflow
4. Verify environment variables are properly set

## Rollback Procedure

If issues arise after deployment:

```bash
# 1. Switch to backup branch
git checkout workflow-cleanup-backup

# 2. Create rollback branch
git checkout -b rollback-workflows

# 3. Restore from git history if needed (backup removed)
git checkout HEAD~10 -- .github/workflows/

# 4. Remove new workflows
rm .github/workflows/main-ci.yml
rm .github/workflows/security.yml
rm .github/workflows/docs.yml
rm .github/workflows/release.yml
rm .github/workflows/dependabot.yml
rm .github/workflows/nightly-simple.yml

# 5. Commit and push
git add .github/workflows/
git commit -m "rollback: restore original workflow configuration"
git push origin rollback-workflows

# 6. Create emergency PR to main
```

## Performance Impact

### Expected Improvements
- **CI/CD Runtime**: ~30% faster due to parallel jobs and reduced redundancy
- **Maintenance Time**: ~70% reduction in workflow maintenance
- **Resource Usage**: ~40% reduction in GitHub Actions minutes
- **Developer Experience**: Cleaner, more predictable CI/CD

### Monitoring Metrics
- Track workflow execution times
- Monitor success/failure rates
- Measure time to deployment
- Track developer satisfaction

## Post-Migration Tasks

1. **Week 1**: Monitor all workflows closely
2. **Week 2**: Gather developer feedback
3. **Month 1**: Analyze performance metrics
4. **Month 3**: Consider further optimizations

## Support and Contact

If you encounter issues during the migration:

1. **Check Documentation**: Review this guide and workflow comments
2. **Check Logs**: GitHub Actions logs provide detailed error information  
3. **Team Support**: Contact the DevOps team for assistance
4. **Rollback**: Use the rollback procedure if critical issues arise

## Success Criteria

The migration is considered successful when:

- [ ] All 8 workflows function correctly
- [ ] CI/CD pipeline time improves by 20%+
- [ ] No critical functionality is lost
- [ ] Developer workflow remains smooth
- [ ] Security scans continue to function
- [ ] Documentation builds successfully
- [ ] Release process works end-to-end