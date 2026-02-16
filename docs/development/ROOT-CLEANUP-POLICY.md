# Root Directory Cleanup Policy

## Overview

This document defines the policy for managing top-level (root) repository files and directories. The goal is to maintain a clean, predictable repository structure while allowing necessary exceptions.

## Allowlist System

### Files

- **`ci/root-allowlist.txt`**: Defines all permitted root entries (files and directories)
- **`ci/root-cleanup-candidates.txt`**: Tracks cleanup candidates, completed migrations, and intentional exceptions
- **`scripts/validate-root-allowlist.sh`**: Automated validation enforced in PR gate

### Validation

The root allowlist is validated on every PR via the `Basic Validation` CI check:

```bash
./scripts/validate-root-allowlist.sh
```

**Enforcement:** Blocking - PRs fail if unexpected root entries are detected.

## Exception Categories

### 1. Intentional Root Exceptions

Files that **must** remain at root due to technical or UX requirements. These are approved exceptions with documented unblock criteria.

**Current Exceptions:**

| File | Rationale | Unblock Criteria |
|------|-----------|------------------|
| `CLAUDE_AGENTS_ANALYSIS.md` | Core AI agent orchestration manifest referenced by `CLAUDE.md` boot sequence; root placement ensures reliable discovery during agent initialization. | Revisit when: (1) Agent system refactored to use dedicated CLI discovery mechanism, OR (2) `CLAUDE.md` imports support path resolution beyond simple `@`-prefixed references. |
| `QUICKSTART.md` | Primary user entry point optimized for immediate discoverability on GitHub landing page; critical for first-time user onboarding. | Revisit when: (1) Repository adopts CLI-guided onboarding flow that supersedes markdown quick-start, OR (2) GitHub repository templates support custom landing page routing. |

### 2. Cleanup Candidates (Blocked)

Files awaiting unblock signals (e.g., coordinated updates to dependent systems).

**Status:** `candidate-blocked` in `ci/root-cleanup-candidates.txt`

### 3. Completed Migrations

Files already successfully moved from root to structured paths.

**Examples:**
- `security-reports/` → `docs/security/reports/` (PR #336)
- `archive/` → `examples/archive/` (PR #337)

## Adding New Root Files

### Approval Process

1. **Check allowlist first**: All new root files must be added to `ci/root-allowlist.txt`
2. **Justify the placement**:
   - **Tooling requirement**: Config files for CI/dev tools (`.golangci.yml`, `Makefile`, etc.)
   - **Standard convention**: Files expected at root by ecosystem (`.gitignore`, `LICENSE`, `README.md`)
   - **Critical UX**: User-facing entry points requiring maximum discoverability
3. **Propose structured alternative**: If the file can logically live in a subdirectory, prefer that
4. **Document exception criteria**: If root placement is required, add to `ci/root-cleanup-candidates.txt` with unblock criteria

### Example: Adding `.newconfig.yaml`

#### Good (structured path):
```bash
# Add to config/ directory instead
touch config/newconfig.yaml
# No allowlist change needed
```

#### Acceptable (root with justification):
```bash
# 1. Add to allowlist
echo ".newconfig.yaml" >> ci/root-allowlist.txt
sort -u ci/root-allowlist.txt -o ci/root-allowlist.txt

# 2. Document in candidates file
echo ".newconfig.yaml|config/newconfig.yaml|intentional-root-exception|Required by XYZ tool which only reads from root.|Revisit when XYZ tool supports configurable paths." >> ci/root-cleanup-candidates.txt
```

## Removing Root Exceptions

When unblock criteria are met, follow this process:

1. **Create migration PR**:
   - Move file to proposed target path
   - Update all references (imports, links, tests, CI configs)
   - Run full validation suite
2. **Update tracking files**:
   - Remove entry from `ci/root-allowlist.txt`
   - Update status in `ci/root-cleanup-candidates.txt` to `completed`
3. **Document in PR description**:
   - Link to original exception rationale
   - Explain what changed to enable the migration
   - List all updated references

## Maintenance

### Quarterly Review

Review all `intentional-root-exception` entries to check if unblock criteria have been met:

- Q1 (Jan-Mar): Security/compliance team
- Q2 (Apr-Jun): Developer experience team
- Q3 (Jul-Sep): Infrastructure team
- Q4 (Oct-Dec): Documentation team

### Metrics

Track root directory health:

- **Total root entries**: Target < 70 (current: ~67)
- **Intentional exceptions**: Target < 5 (current: 2)
- **Blocked candidates**: Target < 3 (current: 0)

## Enforcement

### CI Validation

```yaml
# .github/workflows/pr-validation.yml
- name: Validate Root Allowlist
  run: ./scripts/validate-root-allowlist.sh
```

**Failure modes:**
- ❌ New root file without allowlist entry
- ❌ Allowlist entry for non-existent file
- ✅ All root entries match allowlist (including approved exceptions)

### Manual Override (Emergency)

If an urgent root file is needed temporarily:

```bash
# Add to allowlist immediately
echo "emergency-file.txt" >> ci/root-allowlist.txt

# Create follow-up issue to move/remove within 7 days
gh issue create --title "Remove emergency-file.txt from root" \
  --label "tech-debt" --milestone "Next Sprint"
```

## References

- **Validation Script**: [`scripts/validate-root-allowlist.sh`](../../scripts/validate-root-allowlist.sh)
- **Allowlist**: [`ci/root-allowlist.txt`](../../ci/root-allowlist.txt)
- **Candidates Tracking**: [`ci/root-cleanup-candidates.txt`](../../ci/root-cleanup-candidates.txt)
- **CI Workflow**: [`.github/workflows/pr-validation.yml`](../../.github/workflows/pr-validation.yml)

## History

- **2026-02-13**: Phase 3D - Finalized exception policy with unblock criteria
- **2026-02-13**: Phase 3C - Migrated `archive/` to `examples/archive/`
- **2026-02-13**: Phase 3B - Migrated `security-reports/` to `docs/security/reports/`
- **2026-02-13**: Phase 3A - Established allowlist policy and validation tooling
