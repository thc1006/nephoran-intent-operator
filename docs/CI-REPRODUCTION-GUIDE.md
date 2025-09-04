# CI Reproduction Guide

Complete guide for reproducing GitHub Actions CI locally with exact environment matching.

## Quick Start

### 1. Install CI Tools (Required)
```powershell
# Install exact versions used in CI
./scripts/install-ci-tools.ps1

# Verify installation
go version          # Should be 1.24.6
golangci-lint version  # Should be v1.64.3
controller-gen --version  # Should be v0.19.0
```

### 2. Run All CI Jobs Locally
```powershell
# Full CI pipeline (recommended)
./scripts/run-ci-locally.ps1

# Fast mode (skip slow checks)
./scripts/run-ci-locally.ps1 -FastMode

# Individual jobs
./scripts/run-ci-locally.ps1 -Job lint
./scripts/run-ci-locally.ps1 -Job test
./scripts/run-ci-locally.ps1 -Job build
./scripts/run-ci-locally.ps1 -Job deps
```

### 3. Run with GitHub Actions (Optional)
```powershell
# Install act tool
./scripts/install-act.ps1

# Run workflows locally in containers
act  # Run all workflows
act -W .github/workflows/main-ci.yml  # Specific workflow
```

## Environment Details

### Exact CI Versions
Our local setup matches CI exactly:

| Tool | Version | Source |
|------|---------|---------|
| Go | 1.24.6 | `.github/workflows/*.yml` |
| golangci-lint | v1.64.3 | Environment variable |
| controller-gen | v0.19.0 | Environment variable |
| Kubernetes (envtest) | 1.31.0 | Environment variable |

### Environment Variables
```powershell
# Core CI environment (automatically set)
$env:GO_VERSION = "1.24.6"
$env:CGO_ENABLED = "0"
$env:GOPROXY = "https://proxy.golang.org,direct"
$env:GOSUMDB = "sum.golang.org"
$env:GOMAXPROCS = "2"

# Mode flags
$env:FAST_MODE = "false"      # Override with -FastMode
$env:SKIP_SECURITY = "false"  # Override with -SkipSecurity
```

## Job Descriptions

### 1. Dependency Security (`deps`)
**Script**: `scripts/ci-jobs/dependency-security.ps1`
**Mirrors**: `.github/workflows/main-ci.yml` → `dependency-security` job

**What it does**:
- ✅ Verifies `go.sum` integrity  
- ✅ Downloads and caches all dependencies
- ✅ Runs vulnerability scanning with `govulncheck`
- ✅ Generates cache keys for other jobs
- ⚠️ Skips `go mod tidy` due to import migration

**Key outputs**:
- `.cache-keys.json` - Cache information
- Vulnerability scan results

### 2. Build & Code Quality (`build`)
**Script**: `scripts/ci-jobs/build.ps1`
**Mirrors**: `.github/workflows/main-ci.yml` → `build-and-quality` job

**What it does**:
- 🔧 Installs `controller-gen` locally
- 🏗️ Generates CRDs and RBAC manifests  
- 📝 Verifies code formatting (`go fmt`)
- 🔍 Runs static analysis (`go vet`)
- 🏗️ Builds all packages and executables

**Key outputs**:
- `bin/` - Built executables
- Generated CRDs in `config/crd/bases/`

### 3. Comprehensive Testing (`test`)
**Script**: `scripts/ci-jobs/test.ps1`  
**Mirrors**: `.github/workflows/main-ci.yml` → `testing` job

**What it does**:
- 🔧 Sets up `envtest` Kubernetes binaries
- 🧪 Runs unit tests with race detection
- 📊 Generates coverage reports (optional)
- 🏃 Runs benchmarks (optional)
- ⚡ Fast mode: essential tests only

**Key outputs**:
- `test-results/coverage.out` - Coverage data
- `test-results/coverage.html` - HTML coverage report
- `test-results/benchmark-output.txt` - Benchmark results

### 4. Advanced Linting (`lint`)
**Script**: `scripts/ci-jobs/lint.ps1`
**Mirrors**: `.github/workflows/main-ci.yml` → `linting` job

**What it does**:
- 🔧 Uses exact `golangci-lint` version from CI
- 📋 Runs comprehensive linting with `.golangci.yml`
- 📊 Generates multiple output formats (JSON, XML, SARIF)
- 🔍 Provides detailed issue analysis and summaries

**Key outputs**:
- `lint-reports/golangci-lint.json` - Structured lint results
- `lint-reports/lint-output.txt` - Human-readable output
- `lint-reports/checkstyle.xml` - XML format for tools

## Advanced Usage

### Running Individual Scripts
```powershell
# Dependency checks only
./scripts/ci-jobs/dependency-security.ps1 -FastMode

# Build with verbose output  
./scripts/ci-jobs/build.ps1 -Verbose

# Tests with coverage and benchmarks
./scripts/ci-jobs/test.ps1 -WithCoverage -WithBenchmarks

# Detailed linting with multiple formats
./scripts/ci-jobs/lint.ps1 -DetailedOutput -Config .golangci.yml
```

### GitHub Actions with act

#### Setup
```powershell
# Install act
./scripts/install-act.ps1

# Configure secrets (optional)
cp .github/secrets-template.txt .secrets
# Edit .secrets with your GitHub token
```

#### Usage
```powershell
# List available workflows
act --list

# Dry run to see what would execute
act --dry-run

# Run specific workflow  
act -W .github/workflows/main-ci.yml

# Run single job
act -j lint

# Run with secrets file
act --secret-file .secrets
```

### Debugging Failed Jobs

#### 1. Check Logs
```powershell
# View job-specific logs
Get-Content ci-results/Build*.log | Select-Object -Last 50

# View test output
Get-Content test-results/test-output.txt | Select-Object -Last 30
```

#### 2. Run with Verbose Output
```powershell
./scripts/run-ci-locally.ps1 -Job test -Verbose
```

#### 3. Environment Issues
```powershell
# Reload environment
. scripts/ci-env.ps1

# Check tool versions
go version
golangci-lint version  
controller-gen --version
```

## Configuration Files

### Linting Configuration
- **Primary**: `.golangci.yml` (comprehensive, matches CI)
- **Fast**: `.golangci-fast.yml` (for quick checks)  
- **Minimal**: `.golangci-minimal.yml` (basic checks only)

### Environment Files
- **CI Environment**: `scripts/ci-env.ps1` (loaded automatically)
- **Cache Info**: `.cache-keys.json` (generated by deps job)
- **Envtest**: `.env.ci` (created by tool installation)

## Troubleshooting

### Common Issues

#### 1. "golangci-lint not found"
```powershell
# Reinstall tools
./scripts/install-ci-tools.ps1 -Force

# Check PATH
$env:PATH -split ';' | Where-Object { $_ -match 'local.*bin' }
```

#### 2. "go.sum integrity check failed"  
This is expected during the import path migration. The CI handles this gracefully by skipping `go mod tidy`.

#### 3. "controller-gen not found"
```powershell
# Build job installs it automatically, or run manually:
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.19.0
```

#### 4. Tests fail with "KUBEBUILDER_ASSETS not set"
```powershell
# Reinstall with envtest setup
./scripts/install-ci-tools.ps1 -Force
```

#### 5. Docker not available for act
```powershell
# Install Docker Desktop
# Or use scripts without act:
./scripts/run-ci-locally.ps1
```

### Performance Tuning

#### Fast Development Workflow
```powershell
# Quick lint check during development
./scripts/run-ci-locally.ps1 -Job lint -FastMode

# Essential tests only
./scripts/run-ci-locally.ps1 -Job test -FastMode  

# Skip security scans
./scripts/run-ci-locally.ps1 -SkipSecurity
```

#### Full Pre-Push Validation
```powershell
# Complete CI validation (recommended before push)
./scripts/run-ci-locally.ps1 -WithCoverage -DetailedLint
```

## Integration with Development Workflow

### 1. Pre-Commit Hooks
```powershell
# Quick checks before commit
./scripts/run-ci-locally.ps1 -Job lint -FastMode
./scripts/run-ci-locally.ps1 -Job build
```

### 2. Pre-Push Validation
```powershell  
# Full validation before push
./scripts/run-ci-locally.ps1 -StopOnFailure
```

### 3. Development Loop
```powershell
# During active development
./scripts/run-ci-locally.ps1 -Job test -FastMode

# Before creating PR
./scripts/run-ci-locally.ps1 -WithCoverage
```

## Architecture

### Script Organization
```
scripts/
├── install-ci-tools.ps1      # Tool installation
├── install-act.ps1           # act setup  
├── ci-env.ps1               # Environment variables
├── run-ci-locally.ps1       # Main orchestrator
└── ci-jobs/                 # Individual job scripts
    ├── dependency-security.ps1
    ├── build.ps1
    ├── test.ps1
    └── lint.ps1
```

### Data Flow
1. **install-ci-tools.ps1** → Installs tools, creates `.env.ci`
2. **ci-env.ps1** → Loads environment variables
3. **dependency-security.ps1** → Creates `.cache-keys.json`
4. **Other jobs** → Use cache keys, generate artifacts
5. **run-ci-locally.ps1** → Orchestrates everything

## Comparison with CI

| Aspect | GitHub Actions | Local Reproduction | Notes |
|--------|---------------|-------------------|-------|
| **OS** | Ubuntu 22.04 | Windows 11 | Scripts handle differences |
| **Go** | 1.24.6 | 1.24.6 | ✅ Exact match |  
| **golangci-lint** | v1.64.3 | v1.64.3 | ✅ Exact match |
| **controller-gen** | v0.19.0 | v0.19.0 | ✅ Exact match |
| **Environment** | Ubuntu paths | Windows paths | Scripts adapt paths |
| **Caching** | GitHub cache | Local cache | Different but equivalent |
| **Containers** | Native | Docker (act) | act provides containers |
| **Secrets** | GitHub vault | Local files | act loads from .secrets |

## FAQ

### Q: Why use PowerShell instead of Bash?
**A**: The target environment is Windows, and PowerShell provides better Windows integration, error handling, and cross-platform compatibility.

### Q: Can I run this on Linux/macOS?  
**A**: The scripts are PowerShell Core compatible and should work on Linux/macOS with PowerShell installed. The CI targets Linux anyway.

### Q: How accurate is the local reproduction?
**A**: Very accurate. We use the exact same tool versions, environment variables, and command sequences as CI. The main difference is the OS, which is handled by the scripts.

### Q: What if CI changes?
**A**: Update the version constants in `scripts/ci-env.ps1` and `scripts/install-ci-tools.ps1` to match the new CI configuration.

### Q: Can I add custom linting rules?
**A**: Yes, modify `.golangci.yml` or create custom config files and use `-Config` parameter with the lint job.

---

**Need help?** Check the logs in `ci-results/` or run individual jobs with `-Verbose` for detailed output.