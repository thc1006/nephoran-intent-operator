# Gitleaks Configuration Notes

## Important: Regex vs Glob Pattern Gotcha

### Issue
Gitleaks uses **RE2 regular expressions**, not glob patterns, for path matching. This is a common source of confusion and errors.

### ❌ Incorrect (Glob Pattern)
```toml
paths = [
    '**/testdata/**',      # This will NOT work!
    '*.test.go',           # This will NOT work!
]
```

### ✅ Correct (RE2 Regex)
```toml
paths = [
    '''(^|/)testdata(/|$)''',  # Matches testdata directory anywhere
    '''.*_test\.go$''',         # Matches files ending with _test.go
]
```

## Common Pattern Translations

| Glob Pattern | RE2 Regex | Description |
|-------------|-----------|-------------|
| `**/testdata/**` | `(^|/)testdata(/|$)` | Matches testdata directory anywhere |
| `*.md` | `.*\.md$` | Matches files ending with .md |
| `**/vendor/**` | `(^|/)vendor(/|$)` | Matches vendor directory anywhere |
| `test*.json` | `test.*\.json$` | Matches files starting with test and ending with .json |

## SARIF Output Requirements

When using gitleaks with SARIF output format, ensure:

1. **Create output directory first**:
   ```bash
   mkdir -p security-reports
   ```

2. **Use proper flags**:
   ```bash
   gitleaks detect --source . \
     --redact \              # Redact secrets in output
     -v \                    # Verbose output
     --exit-code=2 \         # Only fail on actual leaks
     --report-format sarif \ # Use SARIF format
     --report-path security-reports/gitleaks.sarif
   ```

## Workflow Configuration

All GitHub Actions workflows should:
1. Create the output directory before running gitleaks
2. Use SARIF format for GitHub Security integration
3. Use `--redact` flag to prevent exposing secrets in logs
4. Use `--exit-code=2` to differentiate between errors (1) and actual leaks (2)

## Testing Configuration

Test your `.gitleaks.toml` configuration locally:

```bash
# Install gitleaks
brew install gitleaks  # macOS
# or download from https://github.com/gitleaks/gitleaks/releases

# Test configuration
gitleaks detect --source . --config .gitleaks.toml --verbose

# Generate SARIF report
mkdir -p security-reports
gitleaks detect --source . \
  --config .gitleaks.toml \
  --report-format sarif \
  --report-path security-reports/gitleaks.sarif
```

## References
- [Gitleaks Documentation](https://github.com/gitleaks/gitleaks)
- [RE2 Syntax](https://github.com/google/re2/wiki/Syntax)
- [SARIF Specification](https://sarifweb.azurewebsites.net/)