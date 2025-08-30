# Golangci-Lint Verification Strategy

## Overview

This directory contains a comprehensive verification strategy for golangci-lint fixes with fast feedback loops designed for the Nephoran Intent Operator project. The strategy focuses on iterative fixing, regression testing, and consistent verification.

## Prerequisites

Before using these scripts, ensure you have:

1. **golangci-lint** installed:
   ```bash
   # Install latest version
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   
   # Verify installation
   golangci-lint --version
   ```

2. **Go toolchain** (1.24+):
   ```bash
   go version
   ```

3. **PowerShell** (Windows) or **Bash** (Linux/macOS) for script execution

4. **jq** (for JSON processing in shell scripts):
   ```bash
   # Ubuntu/Debian
   sudo apt-get install jq
   
   # macOS
   brew install jq
   
   # Windows
   winget install stedolan.jq
   ```

## Script Categories

### 1. Isolated Testing Scripts
**Purpose**: Run individual linters to isolate specific issues

- **`lint-isolated.ps1`** - Windows PowerShell script for individual linter execution
- **`lint-isolated.sh`** - Linux/macOS bash script for individual linter execution
- **`lint-batch.ps1`** - Run all linters individually with parallel support

**Usage Examples**:
```powershell
# Test specific linter
.\scripts\lint-isolated.ps1 revive ./pkg

# Run all linters individually  
.\scripts\lint-batch.ps1 -Parallel

# Fail-fast mode for CI
.\scripts\lint-batch.ps1 -FailFast
```

### 2. Incremental Testing Scripts  
**Purpose**: Test specific files or small file sets during development

- **`lint-incremental.ps1`** - Incremental linting with pattern matching
- **`lint-file-checker.sh`** - Unix shell script for file-specific checks

**Usage Examples**:
```powershell  
# Test specific files
.\scripts\lint-incremental.ps1 "pkg/controllers/*.go"

# Auto-fix issues
.\scripts\lint-incremental.ps1 "internal/loop/*.go" -Fix

# JSON output for automation
.\scripts\lint-incremental.ps1 "cmd/*" -JsonOutput
```

### 3. Functionality Verification Scripts
**Purpose**: Ensure lint fixes don't break existing functionality

- **`verify-lint-fixes.ps1`** - Comprehensive build/test/lint verification  
- **`lint-regression-test.sh`** - Unix regression testing with multiple test suites

**Usage Examples**:
```powershell
# Full verification
.\scripts\verify-lint-fixes.ps1

# Quick build-only check
.\scripts\verify-lint-fixes.ps1 -RunTests:$false -RunLint:$false

# Verbose reporting
.\scripts\verify-lint-fixes.ps1 -Verbose
```

```bash
# Regression test suites
./scripts/lint-regression-test.sh unit
./scripts/lint-regression-test.sh integration
./scripts/lint-regression-test.sh all
```

### 4. Fast Feedback Loop Scripts
**Purpose**: Iterative fixing with interactive workflows

- **`lint-feedback-loop.ps1`** - Interactive PowerShell feedback loop
- **`lint-workflow.sh`** - Automated Unix workflow with multiple strategies

**Usage Examples**:
```powershell
# Interactive fixing session
.\scripts\lint-feedback-loop.ps1

# Auto-fix mode
.\scripts\lint-feedback-loop.ps1 -AutoFix -LinterName revive
```

```bash
# Different workflow strategies
./scripts/lint-workflow.sh full ./pkg
./scripts/lint-workflow.sh quick ./cmd
./scripts/lint-workflow.sh targeted ./internal revive
./scripts/lint-workflow.sh batch ./pkg/controllers
./scripts/lint-workflow.sh safe ./
```

### 5. Comprehensive Verification Scripts
**Purpose**: Final verification that all rules pass consistently

- **`lint-verify-all.ps1`** - Complete verification with reporting
- **`lint-consistency-check.sh`** - Multi-run consistency validation

**Usage Examples**:
```powershell
# Full verification with report
.\scripts\lint-verify-all.ps1 -GenerateReport

# CI mode with fail-fast
.\scripts\lint-verify-all.ps1 -FailFast
```

```bash
# Consistency testing
NUM_RUNS=10 ./scripts/lint-consistency-check.sh

# Custom configuration
TARGET_PATH=./pkg CONFIG_FILE=./custom.yml ./scripts/lint-consistency-check.sh  
```

## Workflow Strategies

### Strategy 1: Development Workflow (Fast Feedback)
```bash
# 1. Quick check during development
./scripts/lint-incremental.ps1 "modified-files*.go" -Fix

# 2. Verify changes don't break build  
./scripts/verify-lint-fixes.ps1 -RunTests:$false

# 3. Pre-commit verification
./scripts/lint-verify-all.ps1 -TargetPath "./modified-package/..."
```

### Strategy 2: Problem Isolation Workflow
```bash
# 1. Identify problematic linters
./scripts/lint-batch.ps1

# 2. Focus on worst offender
./scripts/lint-feedback-loop.ps1 -LinterName staticcheck -AutoFix

# 3. Verify fix
./scripts/verify-lint-fixes.ps1
```

### Strategy 3: CI/CD Pipeline Integration
```yaml
# Example GitHub Actions workflow
- name: Lint Verification
  run: |
    # Quick individual checks
    ./scripts/lint-batch.ps1 -FailFast -Parallel
    
    # Comprehensive verification  
    ./scripts/lint-verify-all.ps1 -FailFast -GenerateReport
    
    # Regression testing
    ./scripts/lint-regression-test.sh all
```

### Strategy 4: Large Codebase Strategy
```bash
# 1. Batch processing for large codebases
./scripts/lint-workflow.sh batch ./

# 2. Consistency validation
./scripts/lint-consistency-check.sh

# 3. Safe mode with backups
./scripts/lint-workflow.sh safe ./pkg
```

## Configuration

### Golangci-Lint Configuration (.golangci.yml)
The scripts automatically read from `.golangci.yml` and respect:

- **Enabled linters** from `linters.enable` section
- **Exclusions** from `issues.exclude-files` and `issues.exclude-rules`
- **Timeout** settings (overrideable via script parameters)
- **Output format** preferences

### Environment Variables
```bash
# Regression testing configuration
export NUM_RUNS=5                # Number of consistency runs
export PARALLEL_JOBS=4           # Parallel test execution
export VERBOSE=true              # Enable verbose output
export FAIL_FAST=true           # Stop on first failure

# Target configuration
export TARGET_PATH=./pkg         # Default target path
export CONFIG_FILE=./.golangci.yml  # Config file location
```

## Performance Optimization

### 1. Parallel Execution
- Use `-Parallel` flag in PowerShell scripts
- Set `PARALLEL_RUNS` environment variable for shell scripts
- Adjust based on CPU cores and I/O capacity

### 2. Targeted Testing
- Test only modified files during development
- Use specific linters for focused fixing
- Leverage file patterns for incremental testing

### 3. Caching Strategy
```bash
# Enable caching (default)
golangci-lint run --build-cache

# Disable for CI environments
./scripts/lint-verify-all.ps1 -CacheResults:$false

# Clear cache when needed
golangci-lint cache clean
```

### 4. Timeout Management
```bash
# Quick checks (30-60s)
./scripts/lint-isolated.ps1 revive --timeout=30s

# Comprehensive checks (10-15min)  
./scripts/lint-verify-all.ps1 -Timeout 900

# CI optimized (5min)
./scripts/lint-batch.ps1 -Timeout 300
```

## Reporting and Analytics

### Generated Reports
All scripts generate timestamped reports in `./test-results/`:

- **`lint-verification-YYYYMMDD_HHMMSS.json`** - Comprehensive verification results
- **`lint-consistency-YYYYMMDD_HHMMSS.json`** - Consistency test results  
- **`lint-feedback-YYYYMMDD_HHMMSS.json`** - Feedback loop session results
- **`lint-regression-YYYYMMDD_HHMMSS.json`** - Regression test results

### Report Structure
```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "success": true,
  "summary": {
    "totalLinters": 11,
    "passedLinters": 11,
    "failedLinters": 0,
    "totalDuration": 45.67
  },
  "linterResults": [...],
  "configuration": {...}
}
```

## Troubleshooting

### Common Issues

1. **Golangci-lint not found**:
   ```bash
   # Install
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   
   # Add to PATH
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

2. **Configuration issues**:
   ```bash
   # Validate configuration
   golangci-lint run --help
   
   # Test configuration
   ./scripts/lint-verify-all.ps1 -Verbose
   ```

3. **Performance issues**:
   ```bash
   # Enable parallel processing
   ./scripts/lint-batch.ps1 -Parallel
   
   # Reduce timeout for quick checks
   ./scripts/lint-isolated.ps1 revive . --timeout=30s
   
   # Clear cache
   golangci-lint cache clean
   ```

4. **False positives**:
   ```bash
   # Test individual linter
   ./scripts/lint-isolated.ps1 problematic-linter ./specific-file.go
   
   # Check exclusions
   grep -A 10 "exclude-files" .golangci.yml
   ```

### Debug Mode
Enable verbose logging in all scripts:
```powershell
# PowerShell
.\scripts\lint-verify-all.ps1 -Verbose

# Bash  
VERBOSE=true ./scripts/lint-consistency-check.sh
```

## Integration Examples

### Pre-commit Hook
```bash
#!/bin/bash
# .git/hooks/pre-commit
set -e

staged_files=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' | tr '\n' ' ')
if [[ -n "$staged_files" ]]; then
    echo "Running lint checks on staged files..."
    ./scripts/lint-incremental.ps1 "$staged_files" -Fix
    
    # Restage fixed files
    git add $staged_files
fi
```

### IDE Integration (VS Code)
```json
// .vscode/tasks.json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "lint-quick",
            "type": "shell",
            "command": "./scripts/lint-incremental.ps1 '${workspaceFolder}/${relativeFile}' -Fix",
            "group": "build"
        },
        {
            "label": "lint-verify",
            "type": "shell", 
            "command": "./scripts/verify-lint-fixes.ps1",
            "group": "test"
        }
    ]
}
```

### Makefile Integration
```makefile
# Makefile
.PHONY: lint-quick lint-verify lint-fix lint-ci

lint-quick:
	./scripts/lint-batch.ps1

lint-verify:  
	./scripts/verify-lint-fixes.ps1

lint-fix:
	./scripts/lint-feedback-loop.ps1 -AutoFix -Interactive:$$false

lint-ci:
	./scripts/lint-verify-all.ps1 -FailFast -GenerateReport
```

## Best Practices

### 1. Development Workflow
- Use incremental scripts during active development
- Run verification scripts before commits
- Use feedback loops for persistent issues

### 2. CI/CD Integration
- Always use `-FailFast` in CI environments  
- Generate reports for debugging
- Cache golangci-lint binaries and results

### 3. Team Collaboration  
- Share verification reports for debugging
- Use consistent timeout values across team
- Document custom exclusions and their reasons

### 4. Performance Optimization
- Run quick checks frequently, comprehensive checks less often
- Use parallel execution for batch operations
- Target specific paths/files when possible

This verification strategy provides comprehensive tooling for maintaining high code quality while minimizing developer friction through fast feedback loops and iterative fixing approaches.