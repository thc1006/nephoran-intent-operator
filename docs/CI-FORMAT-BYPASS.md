# CI Format Bypass Strategy

This document outlines the safe bypass strategy for formatting issues in the Nephoran Intent Operator CI pipeline.

## 🎯 Purpose

The format bypass allows urgent merges when only code formatting issues are blocking CI, while maintaining safety by still checking for critical errors.

## 🚦 When to Use

**✅ SAFE TO BYPASS:**
- Only formatting issues (gofmt, gofumpt, gci, whitespace)
- No critical errors or runtime issues
- Urgent hotfixes or critical feature merges
- Time-sensitive integrations

**❌ NOT SAFE TO BYPASS:**
- Critical errors (compilation failures, runtime errors)
- Security vulnerabilities (gosec errors)
- Logic errors (govet errors)
- Type checking failures

## 🔧 How to Use

### Method 1: Commit Message Keywords

Add any of these keywords to your commit message:
```bash
git commit -m "[bypass-format] fix: urgent hotfix for production issue"
git commit -m "[format-bypass] feat: critical feature for integration"
git commit -m "[bypass-lint] fix: urgent merge needed"
git commit -m "[skip-lint] chore: emergency deployment fix"
```

### Method 2: Emergency Keywords

For broader bypasses (use carefully):
```bash
git commit -m "[emergency] fix: critical production issue"
git commit -m "[urgent-merge] feat: time-sensitive integration"
git commit -m "[hotfix] fix: security patch"
```

### Method 3: Safety Check Script

Run the safety checker before using bypass:

**PowerShell (Windows):**
```powershell
.\scripts\format-bypass-check.ps1
```

**Bash (Linux/Mac):**
```bash
./scripts/format-bypass-check.sh
```

## 🛡️ Safety Mechanisms

### 1. Issue Classification
The CI automatically classifies issues into:
- **Critical Errors:** Must be fixed (compilation, runtime, security)
- **Other Issues:** Should be reviewed (logic, style warnings)
- **Formatting Issues:** Safe to bypass (formatting, imports, whitespace)

### 2. Bypass Safety Check
```yaml
critical_count=0     # ✅ Safe to bypass
format_count=5       # ⚠️  Need formatting fixes
other_count=2        # 📋 Review recommended
```

### 3. Automated Classification
The CI pipeline automatically:
- Counts critical vs. formatting issues
- Shows bypass safety status
- Provides fix recommendations
- Blocks unsafe bypasses

## 📊 CI Pipeline Behavior

### Normal Mode
```yaml
quality:
  continue-on-error: true    # Soft-fail on formatting
  steps:
    - lint-analysis          # Full golangci-lint
    - classify-issues        # Separate critical vs. format
    - safety-assessment      # Determine bypass safety
```

### Emergency Mode (with bypass keywords)
```yaml
emergency-quality:
  steps:
    - build-test            # Ensure code compiles
    - go-vet               # Critical issues only
    - skip-formatting      # Bypass format checks
```

## 🔍 Issue Types & Bypass Safety

| Linter | Issue Type | Bypass Safe | Description |
|--------|------------|-------------|-------------|
| `govet` | Critical | ❌ No | Runtime issues, logic errors |
| `errcheck` | Critical | ❌ No | Unhandled errors |
| `gosec` | Critical | ❌ No | Security vulnerabilities |
| `staticcheck` | Critical | ❌ No | Bug-prone patterns |
| `typecheck` | Critical | ❌ No | Type system violations |
| `gofmt` | Format | ✅ Yes | Code formatting |
| `gofumpt` | Format | ✅ Yes | Enhanced formatting |
| `gci` | Format | ✅ Yes | Import organization |
| `whitespace` | Format | ✅ Yes | Whitespace issues |
| `godot` | Format | ✅ Yes | Comment formatting |

## 🛠️ Post-Bypass Fixes

After using format bypass, fix issues in follow-up:

### 1. Auto-Fix Commands
```bash
# Format code
gofmt -w .

# Fix imports (adjust for your module)
gci write --skip-generated \
  -s standard \
  -s default \
  -s "prefix(github.com/nephio-project/nephoran-intent-operator)" \
  .

# Auto-fix lint issues
golangci-lint run --fix
```

### 2. Manual Review
```bash
# Check what needs fixing
golangci-lint run

# Use bypass config to see only critical issues
golangci-lint run -c .golangci-bypass.yml

# Full check with all linters
golangci-lint run -c .golangci.yml
```

### 3. Follow-up Commit
```bash
git add .
git commit -m "fix: resolve formatting issues from bypassed CI

- Applied gofmt formatting
- Organized imports with gci
- Fixed whitespace issues
- Resolved golangci-lint formatting warnings

Follows up on: [bypass-format] commit abc123"
```

## 📝 Best Practices

### 1. Use Sparingly
- Only for urgent situations
- Always fix formatting in follow-up
- Document reason for bypass

### 2. Safety First
- Always run safety check script first
- Never bypass critical errors
- Review bypass usage in PRs

### 3. Team Communication
- Notify team of bypass usage
- Include reason in commit message
- Schedule formatting fix immediately

### 4. Monitoring
- Track bypass usage frequency
- Review bypass patterns in retrospectives
- Adjust strategy if overused

## 🚨 Troubleshooting

### Bypass Not Working?
1. Check commit message format
2. Ensure keywords are correct: `[bypass-format]`
3. Verify CI detected the bypass
4. Check emergency-check job logs

### Still Failing with Critical Errors?
1. Run safety check script
2. Fix critical errors first
3. Use bypass only for formatting
4. Contact team if unsure

### Format Issues After Merge?
1. Run auto-fix commands
2. Create follow-up PR
3. Use pre-commit hooks to prevent future issues

## 📋 Configuration Files

- **`.golangci.yml`** - Full lint configuration
- **`.golangci-bypass.yml`** - Bypass configuration (formatting disabled)
- **`.github/workflows/ci.yml`** - CI pipeline with bypass logic
- **`scripts/format-bypass-check.ps1`** - PowerShell safety checker
- **`scripts/format-bypass-check.sh`** - Bash safety checker

## 🔗 Related Documentation

- [golangci-lint Configuration](.golangci.yml)
- [CI Pipeline Documentation](.github/workflows/ci.yml)
- [Contributing Guidelines](../CONTRIBUTING.md)
- [Code Style Guide](../docs/CODE_STYLE.md)

---

**Remember:** Format bypass is a safety valve, not a shortcut. Always prioritize code quality and use bypass only when absolutely necessary for urgent situations.