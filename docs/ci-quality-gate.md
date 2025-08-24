# CI Quality Gate Behavior

## Overview
The quality metrics calculation job in CI has different behavior depending on the event type:

- **Pull Requests**: Gate warns on quality issues but does not fail the job
- **Push to main**: Gate blocks on quality issues and fails the job

## Technical Details

### Exit Codes
- `0`: Quality gates passed
- `123`: Quality metrics below threshold (converted to warning on PRs)
- Other non-zero: Actual errors (fail in both cases)

### Workflow Configuration
- Uses `continue-on-error: ${{ github.event_name == 'pull_request' }}` to allow PR jobs to continue
- Special handling in the quality gate step converts exit code `123` to success on PRs with warning annotation
- CI status check excludes quality failures for PRs but includes them for main branch pushes

This ensures developers get early feedback on quality issues without blocking their workflow, while maintaining strict quality standards on the main branch.