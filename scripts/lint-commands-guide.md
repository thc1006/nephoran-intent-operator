# Golangci-Lint Verification Strategy Commands Guide

This guide provides comprehensive commands for verifying golangci-lint fixes with fast feedback loops.

## Quick Reference

### Individual Linter Testing
```powershell
# Test a specific linter on current directory
.\scripts\lint-isolated.ps1 revive

# Test a specific linter on specific path  
.\scripts\lint-isolated.ps1 staticcheck ./pkg/controllers

# Test with additional arguments
.\scripts\lint-isolated.ps1 govet ./internal "--print-issued-lines=false"
```

```bash
# Linux/Mac equivalent
./scripts/lint-isolated.sh revive
./scripts/lint-isolated.sh staticcheck ./pkg/controllers
```

### Batch Linter Testing
```powershell
# Run all linters individually
.\scripts\lint-batch.ps1

# Run all linters on specific path
.\scripts\lint-batch.ps1 ./pkg

# Run with fail-fast mode
.\scripts\lint-batch.ps1 . -FailFast

# Run in parallel (faster)
.\scripts\lint-batch.ps1 . -Parallel
```

### Incremental File Testing
```powershell
# Test specific files
.\scripts\lint-incremental.ps1 "pkg/controllers/*.go"

# Test with specific linter
.\scripts\lint-incremental.ps1 "internal/loop/*.go" revive

# Get JSON output for automation
.\scripts\lint-incremental.ps1 "cmd/conductor/*.go" -JsonOutput

# Auto-fix issues
.\scripts\lint-incremental.ps1 "pkg/auth/*.go" -Fix
```

```bash
# Linux/Mac file checking
./scripts/lint-file-checker.sh "pkg/controllers/*.go"
./scripts/lint-file-checker.sh "internal/loop/processor.go" revive
```

### Verification and Testing
```powershell
# Verify fixes don't break functionality
.\scripts\verify-lint-fixes.ps1

# Run only build verification
.\scripts\verify-lint-fixes.ps1 -RunTests:$false -RunLint:$false

# Quick verification with 5-minute timeout
.\scripts\verify-lint-fixes.ps1 -Timeout 300

# Fail-fast mode
.\scripts\verify-lint-fixes.ps1 -FailFast

# Verbose output with detailed JSON report
.\scripts\verify-lint-fixes.ps1 -Verbose
```

```bash
# Linux/Mac regression testing
./scripts/lint-regression-test.sh unit
./scripts/lint-regression-test.sh integration
./scripts/lint-regression-test.sh all

# With environment variables
NUM_RUNS=10 VERBOSE=true ./scripts/lint-regression-test.sh
```

### Fast Feedback Loop
```powershell
# Interactive feedback loop
.\scripts\lint-feedback-loop.ps1

# Target specific linter
.\scripts\lint-feedback-loop.ps1 -LinterName revive -TargetPath ./pkg

# Auto-fix mode with max 5 iterations
.\scripts\lint-feedback-loop.ps1 -AutoFix -MaxIterations 5

# Non-interactive mode
.\scripts\lint-feedback-loop.ps1 -Interactive:$false -ContinueOnFail
```

```bash
# Linux/Mac workflow automation
./scripts/lint-workflow.sh full ./pkg
./scripts/lint-workflow.sh quick ./cmd
./scripts/lint-workflow.sh targeted ./internal revive
./scripts/lint-workflow.sh batch ./pkg/controllers
./scripts/lint-workflow.sh safe ./
```

### Comprehensive Verification
```powershell
# Full verification suite
.\scripts\lint-verify-all.ps1

# Custom configuration
.\scripts\lint-verify-all.ps1 -ConfigFile "./custom.golangci.yml" -TargetPath "./pkg"

# Fast check without cache
.\scripts\lint-verify-all.ps1 -CacheResults:$false -Timeout 300

# Generate detailed report
.\scripts\lint-verify-all.ps1 -GenerateReport -Verbose

# Fail-fast mode for CI
.\scripts\lint-verify-all.ps1 -FailFast
```

```bash
# Linux/Mac consistency checking  
./scripts/lint-consistency-check.sh

# Custom configuration
TARGET_PATH=./pkg CONFIG_FILE=./custom.yml ./scripts/lint-consistency-check.sh

# More runs for better statistics
NUM_RUNS=10 PARALLEL_RUNS=4 ./scripts/lint-consistency-check.sh
```

## Workflow Examples

### 1. Fix Specific Issues Iteratively
```powershell
# Step 1: Identify problem linters
.\scripts\lint-batch.ps1 ./pkg

# Step 2: Focus on worst offender
.\scripts\lint-feedback-loop.ps1 -LinterName revive -TargetPath ./pkg -AutoFix

# Step 3: Verify the fix doesn't break anything
.\scripts\verify-lint-fixes.ps1 -TestPattern "./pkg/..."

# Step 4: Final comprehensive check
.\scripts\lint-verify-all.ps1 -TargetPath "./pkg/..."
```

### 2. Pre-commit Hook Workflow
```bash
#!/bin/bash
# Pre-commit hook example

# Quick check on staged files
staged_files=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$')
if [[ -n "$staged_files" ]]; then
    echo "$staged_files" | xargs ./scripts/lint-file-checker.sh
    
    if [[ $? -ne 0 ]]; then
        echo "Lint issues found. Run auto-fix?"
        read -p "Auto-fix issues? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo "$staged_files" | xargs golangci-lint run --fix
        fi
        exit 1
    fi
fi
```

### 3. CI Pipeline Integration
```yaml
# GitHub Actions example
name: Lint Verification
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.24'
      
      # Fast individual checks
      - name: Quick Lint Batch
        run: ./scripts/lint-batch.ps1 -FailFast
        
      # Comprehensive verification
      - name: Full Verification
        run: ./scripts/lint-verify-all.ps1 -FailFast -GenerateReport
        
      # Upload results
      - name: Upload Lint Results
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: lint-results
          path: ./test-results/
```

### 4. Development Workflow
```powershell
# Daily development routine

# Morning: Check current state
.\scripts\lint-verify-all.ps1 -TargetPath "./pkg/..." -GenerateReport

# During development: Quick incremental checks
.\scripts\lint-incremental.ps1 "pkg/controllers/networkintent_*.go" -Fix

# Before commit: Comprehensive verification
.\scripts\verify-lint-fixes.ps1 -TestPattern "./pkg/controllers/..."

# Weekly: Consistency check
.\scripts\lint-consistency-check.sh
```

## Performance Optimization Tips

### 1. Parallel Execution
- Use `-Parallel` flag in batch operations
- Set `PARALLEL_JOBS` environment variable for regression tests
- Adjust `PARALLEL_RUNS` for consistency checks

### 2. Targeted Testing
- Test only modified files/packages during development
- Use specific linter names for focused fixing
- Leverage incremental scripts for fast feedback

### 3. Caching
- Enable cache for repeated runs (default)
- Disable cache (`-CacheResults:$false`) for CI environments
- Clear cache periodically: `golangci-lint cache clean`

### 4. Timeout Management
- Use shorter timeouts (30-60s) for quick checks  
- Longer timeouts (10-15min) for comprehensive runs
- Adjust based on codebase size and CI environment

## Troubleshooting

### Common Issues
```powershell
# Linter not found
golangci-lint --version
# Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Configuration issues
.\scripts\lint-verify-all.ps1 -ConfigFile ./.golangci.yml -Verbose

# Performance issues
.\scripts\lint-batch.ps1 -Parallel
.\scripts\lint-consistency-check.sh | grep "variance"

# False positives
.\scripts\lint-isolated.ps1 revive ./problematic-file.go
```

### Debugging Commands
```bash
# Enable verbose logging
VERBOSE=true ./scripts/lint-workflow.sh full

# Check configuration parsing
yq eval '.linters.enable[]' .golangci.yml

# Test single file
golangci-lint run --disable-all --enable=revive ./specific-file.go

# Check golangci-lint version compatibility  
golangci-lint --version
go version
```

## Integration with IDEs

### VS Code
```json
// settings.json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": [
        "--fast",
        "--enable=revive,staticcheck,govet"
    ]
}
```

### GoLand/IntelliJ
1. File → Settings → Go → Linter
2. Select golangci-lint
3. Add custom args: `--enable=revive,staticcheck,govet`

This comprehensive command guide provides all the tools needed for effective golangci-lint verification with fast feedback loops.