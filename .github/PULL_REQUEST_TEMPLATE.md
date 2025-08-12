# Pull Request

## Summary

Provide a brief description of the changes in this pull request.

## Related Issue

- Closes #[issue number]
- Addresses #[issue number]
- Related to #[issue number]

## Type of Change

Please delete options that are not relevant:

- [ ] 🐛 Bug fix (non-breaking change which fixes an issue)
- [ ] ✨ New feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📖 Documentation update (changes to documentation only)
- [ ] 🧪 Test improvements (adding missing tests or correcting existing tests)
- [ ] ♻️ Code refactoring (no functional changes, no api changes)
- [ ] ⚡ Performance improvements
- [ ] 🎨 Style changes (formatting, renaming)
- [ ] 🔧 Chore (maintenance tasks, dependency updates)

## Changes Made

Describe the specific changes made in this pull request:

- Change 1: Description
- Change 2: Description  
- Change 3: Description

## Technical Details

### Architecture Changes

- [ ] No architecture changes
- [ ] Modified existing components
- [ ] Added new components
- [ ] Changed APIs or interfaces
- [ ] Modified data models
- [ ] Updated dependencies

**If checked, please describe:**

### Performance Impact

- [ ] No performance impact expected
- [ ] Performance improvements expected
- [ ] Potential performance degradation (explained below)
- [ ] Performance impact unknown/not measured

**Performance Analysis:**

### Security Considerations

- [ ] No security impact
- [ ] Improves security
- [ ] Potential security implications (reviewed below)
- [ ] Security impact unknown

**Security Analysis:**

## Testing

### Test Coverage

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated  
- [ ] End-to-end tests added/updated
- [ ] Manual testing completed
- [ ] No tests needed (explain why)

**Test Details:**
- New test files: 
- Modified test files:
- Test coverage percentage:

### Manual Testing

**Test Environment:**
- Kubernetes version:
- Operating system:
- Other relevant details:

**Test Scenarios:**
- [ ] Happy path scenarios tested
- [ ] Error handling tested  
- [ ] Edge cases tested
- [ ] Performance testing completed
- [ ] Security testing completed

**Test Results:**
Describe the results of your manual testing.

## Documentation

- [ ] Code comments updated
- [ ] README updated (if needed)
- [ ] API documentation updated (if needed)
- [ ] User documentation updated (if needed)
- [ ] Migration guide updated (if breaking changes)
- [ ] Examples updated (if needed)
- [ ] No documentation changes needed

**Documentation Changes:**
List the documentation files that were modified.

## Backwards Compatibility

- [ ] Fully backwards compatible
- [ ] Backwards compatible with deprecation warnings
- [ ] Breaking change with migration path provided
- [ ] Breaking change without migration path (justify below)

**Compatibility Notes:**
If there are backwards compatibility implications, describe them here.

## Deployment Considerations

- [ ] No special deployment considerations
- [ ] Requires configuration changes
- [ ] Requires database migrations
- [ ] Requires new environment variables
- [ ] Requires new secrets/credentials
- [ ] Requires infrastructure changes

**Deployment Notes:**
Describe any special deployment requirements.

## Rollback Plan

**If this change causes issues, how can it be rolled back?**
- [ ] Simple revert of this PR
- [ ] Requires configuration rollback
- [ ] Requires data migration rollback
- [ ] Complex rollback (detailed below)

**Rollback Instructions:**

## Quality Assurance

### Code Quality

- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Code is well-commented, particularly in hard-to-understand areas
- [ ] Logging is appropriate and helpful
- [ ] Error handling is comprehensive

### Dependencies

- [ ] No new dependencies added
- [ ] New dependencies are justified and documented
- [ ] Dependencies are properly licensed
- [ ] Security implications of dependencies considered

### Configuration

- [ ] Configuration changes are documented
- [ ] Default values are sensible
- [ ] Configuration validation is implemented
- [ ] Environment-specific configurations considered

## Governance Checklist

**IMPORTANT: All PRs must comply with repository governance rules**

### Contract Compliance
- [ ] No modifications to `docs/contracts/` files (unless this is a contract update PR)
- [ ] Code changes comply with existing contract definitions
- [ ] If contract changes needed, separate PR opened to `integrate/mvp`

### CI/CD Integrity
- [ ] No modifications to `.github/workflows/ci.yml` (only Project Conductor role)
- [ ] No creation of `.github/workflows/ci.yaml` (use ci.yml only)
- [ ] CI pipeline passes all required checks

### Module Boundaries
- [ ] Changes stay within appropriate module boundaries
- [ ] No cross-module rewrites without coordination
- [ ] Interface contracts maintained

### Build System
- [ ] No unauthorized changes to Makefile targets
- [ ] Make targets still function correctly

## Review Checklist

Please confirm that you have:

- [ ] Read the [Contributing Guidelines](../../CONTRIBUTING.md)
- [ ] Followed the project's coding standards
- [ ] Written/updated tests for your changes
- [ ] Updated documentation as needed
- [ ] Considered performance and security implications
- [ ] Tested the changes locally
- [ ] Ensured CI checks will pass
- [ ] Verified governance checklist compliance

## Screenshots (if applicable)

Add screenshots to help explain your changes, especially for UI changes.

## Additional Notes

Any additional information that reviewers should know:

- Implementation trade-offs made
- Alternative approaches considered
- Future work that might be needed
- Known limitations or issues

## Reviewer Focus Areas

Please pay special attention to:

- [ ] Code architecture and design
- [ ] Performance implications
- [ ] Security considerations  
- [ ] Error handling and edge cases
- [ ] Documentation completeness
- [ ] Test coverage and quality
- [ ] Backwards compatibility
- [ ] User experience impact

---

**Thank you for your contribution! We appreciate you taking the time to improve the Nephoran Intent Operator.**