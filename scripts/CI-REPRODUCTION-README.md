# Local CI Reproduction Scripts

This directory contains scripts to reproduce the exact CI pipeline locally for the Nephoran Intent Operator project. These scripts ensure consistency between local development and CI environments.

## 🚀 Quick Start

### 1. Setup Environment

```powershell
# Install golangci-lint v1.64.3 (exact CI version)
.\scripts\install-golangci-lint.ps1

# Install nektos/act for GitHub Actions simulation (optional)
.\scripts\install-act.ps1
```

### 2. Run Linting (Most Common)

```powershell
# Fast linting (recommended for development)
.\scripts\run-lint-local.ps1 -Fast

# Full linting (matches CI exactly)
.\scripts\run-lint-local.ps1

# Auto-fix issues where possible
.\scripts\run-lint-local.ps1 -Fast -Fix
```

### 3. Complete CI Reproduction

```powershell
# Run all CI jobs locally
.\scripts\reproduce-ci.ps1

# Run specific job only
.\scripts\reproduce-ci.ps1 -Job linting

# Fast mode (skip slow operations)
.\scripts\reproduce-ci.ps1 -Fast
```

## 📋 Available Scripts

### Core Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `install-golangci-lint.ps1` | Install exact CI version of golangci-lint | `.\install-golangci-lint.ps1` |
| `run-lint-local.ps1` | Execute linting with CI config | `.\run-lint-local.ps1 -Fast` |
| `ci-loop.ps1` | Iterative linting until zero errors | `.\ci-loop.ps1 -AutoFix` |
| `reproduce-ci.ps1` | Complete CI pipeline simulation | `.\reproduce-ci.ps1 -Job all` |
| `install-act.ps1` | Install GitHub Actions runner | `.\install-act.ps1` |

## 🔧 Detailed Usage

### golangci-lint Installation

```powershell
# Install latest compatible version
.\scripts\install-golangci-lint.ps1

# Install specific version
.\scripts\install-golangci-lint.ps1 -Version "v1.64.3"

# Force reinstall
.\scripts\install-golangci-lint.ps1 -Force

# Install to custom directory
.\scripts\install-golangci-lint.ps1 -InstallDir "C:\tools\bin"
```

**Requirements:**
- Go 1.24+ installed
- Internet connection for download
- Windows PowerShell 5.0+

### Local Linting

```powershell
# Basic usage - runs with .golangci.yml
.\scripts\run-lint-local.ps1

# Fast mode - uses .golangci-fast.yml (optimized config)
.\scripts\run-lint-local.ps1 -Fast

# Auto-fix mode - attempts to fix issues automatically
.\scripts\run-lint-local.ps1 -Fix

# Custom config file
.\scripts\run-lint-local.ps1 -Config ".golangci-custom.yml"

# Increase timeout for large codebases
.\scripts\run-lint-local.ps1 -Timeout 30

# Verbose output with linter statistics
.\scripts\run-lint-local.ps1 -Verbose -ShowStats
```

**Output:**
- Colored terminal output matching CI format
- Exit code 0 for success, non-zero for failures
- Detailed error information with file locations
- Performance statistics and timing

### CI Loop (Iterative Fixing)

```powershell
# Basic loop - manual iteration
.\scripts\ci-loop.ps1

# Auto-fix mode - attempts fixes between iterations
.\scripts\ci-loop.ps1 -AutoFix

# Watch mode - automatic iteration with delays
.\scripts\ci-loop.ps1 -WatchMode -DelaySeconds 5

# Fast mode with limited iterations
.\scripts\ci-loop.ps1 -Fast -MaxIterations 5

# All options combined
.\scripts\ci-loop.ps1 -Fast -AutoFix -WatchMode -MaxIterations 10
```

**Features:**
- Progress tracking and trend analysis
- Error categorization (critical vs warnings)
- Top issues by linter and file
- Interactive controls (quit, toggle fix mode)
- Comprehensive logging to `ci-loop.log`

### Complete CI Reproduction

```powershell
# Run all CI jobs
.\scripts\reproduce-ci.ps1

# Run specific job
.\scripts\reproduce-ci.ps1 -Job "linting"
.\scripts\reproduce-ci.ps1 -Job "testing"
.\scripts\reproduce-ci.ps1 -Job "build-and-quality"
.\scripts\reproduce-ci.ps1 -Job "dependency-security"

# Fast mode (skip slow operations)
.\scripts\reproduce-ci.ps1 -Fast

# Skip security scans
.\scripts\reproduce-ci.ps1 -SkipSecurity

# Use GitHub Actions simulation
.\scripts\reproduce-ci.ps1 -ActMode

# Custom Go version
.\scripts\reproduce-ci.ps1 -GoVersion "1.24.6"
```

**Available Jobs:**
- `dependency-security`: Go module verification, vulnerability scanning
- `build-and-quality`: Code generation, formatting, static analysis, build
- `testing`: Unit tests, integration tests, coverage analysis
- `linting`: golangci-lint execution with full configuration
- `all`: Execute all jobs in sequence

### GitHub Actions Simulation

```powershell
# Install act (one-time setup)
.\scripts\install-act.ps1

# Run CI with act
.\scripts\reproduce-ci.ps1 -ActMode

# Direct act usage
act -l                                    # List workflows
act -j linting                           # Run linting job
act -W .github/workflows/main-ci.yml     # Run specific workflow
```

**Requirements:**
- Docker Desktop installed and running
- nektos/act installed
- Linux container support

## 🎯 Configuration Files

### golangci-lint Configurations

| File | Purpose | Usage |
|------|---------|-------|
| `.golangci.yml` | Full CI configuration | Production linting |
| `.golangci-fast.yml` | Optimized for speed | Development linting |

**Fast Config Features:**
- Reduced timeout (10m vs 45m)
- Essential linters only
- Optimized for development speed
- Maintains code quality standards

### Environment Variables

The scripts automatically set CI-compatible environment variables:

```powershell
$env:CGO_ENABLED = "0"
$env:GOPROXY = "https://proxy.golang.org,direct"
$env:GOSUMDB = "sum.golang.org"
$env:GOLANGCI_LINT_VERSION = "v1.64.3"
$env:GO_VERSION = "1.24.6"
```

## 🔍 Troubleshooting

### Common Issues

**golangci-lint not found:**
```powershell
# Ensure it's installed
.\scripts\install-golangci-lint.ps1

# Check PATH
$env:PATH -split ';' | Where-Object { $_ -like "*go*" }

# Manual installation
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3
```

**Docker issues with act:**
```powershell
# Check Docker status
docker version

# Install Docker Desktop
winget install Docker.DockerDesktop

# Use alternative runner images
act -P ubuntu-latest=catthehacker/ubuntu:act-latest
```

**Permission issues:**
```powershell
# Run as Administrator if needed
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# Check PowerShell execution policy
Get-ExecutionPolicy -List
```

**Build failures:**
```powershell
# Clean and rebuild
go clean -cache
go mod download
go mod tidy

# Check Go version
go version  # Should be 1.24.x
```

### Debug Mode

All scripts support verbose output for troubleshooting:

```powershell
# Enable verbose output
.\scripts\run-lint-local.ps1 -Verbose

# Check log files
Get-Content ci-results/*.log

# Enable PowerShell debugging
$VerbosePreference = "Continue"
```

## 📊 Performance Optimization

### Caching Strategy

The scripts implement intelligent caching:

- **Go module cache**: `~/go/pkg/mod`
- **Go build cache**: `~/.cache/go-build`
- **golangci-lint cache**: Automatic with `--skip-cache=false`

### Parallel Execution

```powershell
# Enable parallel testing
$env:GOMAXPROCS = "4"
go test -parallel=4 ./...

# Parallel linting (built-in)
# golangci-lint automatically uses available CPU cores
```

### Resource Monitoring

```powershell
# Monitor resource usage during CI
Get-Process -Name "go" | Select-Object CPU, WorkingSet
Get-Process -Name "golangci-lint" | Select-Object CPU, WorkingSet
```

## 🚀 Integration Examples

### IDE Integration

**VS Code tasks.json:**
```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Lint Fast",
            "type": "shell",
            "command": ".\\scripts\\run-lint-local.ps1",
            "args": ["-Fast"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "panel": "new"
            }
        }
    ]
}
```

### Git Hooks

**pre-commit hook:**
```bash
#!/bin/sh
powershell.exe -File scripts/run-lint-local.ps1 -Fast
if [ $? -ne 0 ]; then
    echo "Linting failed. Commit aborted."
    exit 1
fi
```

### CI/CD Integration

**Azure DevOps:**
```yaml
- task: PowerShell@2
  displayName: 'Run Local CI Reproduction'
  inputs:
    filePath: 'scripts/reproduce-ci.ps1'
    arguments: '-Fast -Job linting'
```

## 📈 Metrics and Reporting

### Coverage Reports

```powershell
# Generate coverage HTML report
.\scripts\reproduce-ci.ps1 -Job testing
# Output: test-results/coverage.html
```

### Benchmark Reports

```powershell
# Run performance benchmarks
go test -bench=. -benchmem ./... | Tee-Object benchmarks.txt
```

### CI Timing Analysis

The scripts provide detailed timing information:

- Per-job execution time
- Linter performance breakdown  
- Test suite timing
- Overall pipeline duration

## 🔧 Customization

### Custom Linter Config

Create your own configuration:

```yaml
# .golangci-custom.yml
run:
  timeout: 5m
linters:
  enable:
    - revive
    - staticcheck
    - govet
```

```powershell
# Use custom config
.\scripts\run-lint-local.ps1 -Config ".golangci-custom.yml"
```

### Environment-Specific Settings

```powershell
# Development environment
$env:FAST_MODE = "true"
$env:SKIP_SECURITY = "true"

# Production-like environment
$env:FAST_MODE = "false"
$env:SKIP_SECURITY = "false"
```

## 📞 Support

For issues or questions:

1. Check the troubleshooting section above
2. Review CI logs in `.github/workflows/`
3. Examine local logs in `ci-results/`
4. Compare with working CI runs

**Common Commands for Support:**

```powershell
# System information
$PSVersionTable
go version
golangci-lint version
docker version

# Project status
git status
go mod tidy -v
go list -m all
```