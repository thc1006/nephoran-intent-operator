# CI Diagnostics Collection Guide

## Overview

This document describes the CI/CD diagnostics collection approach for the Nephoran Intent Operator project. Our strategy ensures that debugging information from failed CI runs is properly collected, stored, and accessible without polluting the repository.

## Key Principles

1. **No CI logs in repository**: CI logs and diagnostic files should never be committed to the repository
2. **Automatic artifact collection**: Failed builds automatically collect and upload diagnostics
3. **Time-limited retention**: Diagnostic artifacts are retained for 30 days in GitHub Actions
4. **Comprehensive information**: Collect enough information to debug issues without needing to re-run failed builds

## Diagnostic Collection Strategy

### What Gets Collected

When a CI job fails, the following diagnostics are automatically collected:

1. **Test Results**
   - Test output logs (`test-attempt-*.log`)
   - Coverage files (`coverage*.out`)
   - Test summaries and reports

2. **Environment Information**
   - Operating system details
   - Go version and environment variables
   - Runner specifications
   - Build timestamps and identifiers

3. **Module Dependencies**
   - Complete list of Go modules (`go list -m all`)
   - Dependency versions for reproducibility

4. **System Diagnostics**
   - Available disk space
   - Memory usage (Linux only)
   - System architecture and capabilities

### Where Diagnostics Are Stored

#### During CI Execution
- **Temporary directory**: `ci-diagnostics/` (created during workflow execution)
- **Test results**: `test-results/` (standard test output location)

#### After CI Completion
- **GitHub Actions Artifacts**: Uploaded automatically on failure
- **Retention period**: 30 days
- **Artifact naming**: `test-results-{os}-{run-id}`

### How to Access Diagnostics

1. **Via GitHub UI**:
   - Navigate to the failed workflow run
   - Click on "Summary" tab
   - Download artifacts from the "Artifacts" section

2. **Via GitHub CLI**:
   ```bash
   # List workflow runs
   gh run list --workflow=ci-enhanced.yml
   
   # Download artifacts from a specific run
   gh run download <run-id>
   ```

3. **Via API**:
   ```bash
   # Get artifacts for a workflow run
   curl -H "Authorization: token $GITHUB_TOKEN" \
     https://api.github.com/repos/thc1006/nephoran-intent-operator/actions/runs/<run-id>/artifacts
   ```

## Implementation Details

### Workflow Configuration

The CI workflow includes dedicated steps for diagnostic collection:

```yaml
- name: Collect CI Diagnostics on Failure
  if: failure()
  run: |
    # Collection script (see workflow for full implementation)
    mkdir -p ci-diagnostics
    # ... collect various diagnostics ...

- name: Upload test artifacts
  if: always()
  uses: actions/upload-artifact@v4
  with:
    name: test-results-${{ matrix.os }}-${{ github.run_id }}
    path: |
      test-results/
      ci-diagnostics/
    retention-days: 30
    if-no-files-found: warn
```

### .gitignore Configuration

The following patterns ensure CI logs are never committed:

```gitignore
# CI/CD artifacts
.github/workflows/*.log
.claude/CI_error/
.claude/ci-diagnostics/
CI_error/
ci-logs/
ci-diagnostics/
test-results/
```

## Best Practices

### For Developers

1. **Never commit CI logs**: Always check `.gitignore` is working before committing
2. **Use artifacts for debugging**: Download artifacts instead of adding debug commits
3. **Clean local directories**: Run `rm -rf ci-diagnostics/ test-results/` after local testing

### For CI Maintenance

1. **Monitor artifact storage**: GitHub has limits on artifact storage (varies by plan)
2. **Adjust retention periods**: Balance debugging needs with storage costs
3. **Enhance diagnostics**: Add new diagnostic information as needed, but avoid sensitive data

### Security Considerations

1. **No secrets in logs**: Ensure diagnostic collection never includes secrets or tokens
2. **Sanitize output**: Use `2>&1 || true` to prevent command failures from exposing sensitive errors
3. **Limited retention**: 30-day retention limits exposure of any accidentally included sensitive data

## Troubleshooting

### Common Issues

1. **Missing artifacts**: Check if the upload step ran (even on failure)
2. **Incomplete diagnostics**: Review the `continue-on-error: true` flags
3. **Large artifacts**: Consider compressing or filtering large log files

### Adding New Diagnostics

To add new diagnostic information:

1. Edit `.github/workflows/ci-enhanced.yml`
2. Add collection commands to the appropriate diagnostic step (Unix or Windows)
3. Ensure the new files are in `ci-diagnostics/` directory
4. Test locally before pushing

## Alternative Approaches

For projects requiring different approaches, consider:

1. **External logging services**: 
   - AWS CloudWatch
   - Google Cloud Logging
   - Datadog CI Visibility

2. **Issue creation**:
   - Automatically create GitHub issues for failures
   - Include diagnostic summaries in issue body

3. **Notification systems**:
   - Slack/Discord webhooks with failure summaries
   - Email notifications with attached logs

## Maintenance

This diagnostic system should be reviewed quarterly to ensure:
- Artifact storage is within limits
- Diagnostic information remains relevant
- No sensitive data is being collected
- Performance impact is minimal

## Contact

For questions or improvements to the CI diagnostics system:
- Create an issue in the repository
- Tag with `ci/cd` and `diagnostics` labels