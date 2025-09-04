# CI Reproduction Scripts - Quick Reference

## 🚀 Quick Start

1. **Install Tools**: `./install-ci-tools.ps1`
2. **Run All CI**: `./run-ci-locally.ps1`  
3. **Optional Act Setup**: `./install-act.ps1`

## 📋 Script Overview

| Script | Purpose | Mirrors CI Job |
|--------|---------|----------------|
| `install-ci-tools.ps1` | Install exact CI tool versions | N/A (setup) |
| `install-act.ps1` | Setup GitHub Actions local runner | N/A (setup) |
| `ci-env.ps1` | Load CI environment variables | N/A (helper) |
| `run-ci-locally.ps1` | Main orchestrator - run all jobs | Full pipeline |
| `ci-jobs/dependency-security.ps1` | Deps & security checks | `dependency-security` |
| `ci-jobs/build.ps1` | Build & code quality | `build-and-quality` |
| `ci-jobs/test.ps1` | Comprehensive testing | `testing` |
| `ci-jobs/lint.ps1` | Advanced linting | `linting` |

## ⚡ Common Commands

```powershell
# Full CI pipeline
./run-ci-locally.ps1

# Fast development checks  
./run-ci-locally.ps1 -FastMode

# Individual jobs
./run-ci-locally.ps1 -Job lint
./run-ci-locally.ps1 -Job test -WithCoverage
./run-ci-locally.ps1 -Job build -Verbose

# With act (GitHub Actions locally)
act --dry-run                    # See what would run
act -W .github/workflows/main-ci.yml  # Run specific workflow
act -j lint                      # Run single job
```

## 🔧 Tool Versions (Matches CI Exactly)

- **Go**: 1.24.6
- **golangci-lint**: v1.64.3  
- **controller-gen**: v0.19.0
- **Kubernetes (envtest)**: 1.31.0

## 📁 Generated Artifacts

| Directory | Contents |
|-----------|----------|
| `test-results/` | Test outputs, coverage reports |
| `lint-reports/` | Lint results (JSON, XML, SARIF) |
| `ci-results/` | Job logs and execution summaries |
| `bin/` | Built executables |

## 🐛 Troubleshooting

| Issue | Solution |
|-------|----------|
| "golangci-lint not found" | `./install-ci-tools.ps1 -Force` |
| Tests fail with envtest | Check `KUBEBUILDER_ASSETS` env var |
| go.sum integrity errors | Expected during migration (CI handles this) |
| Docker not available | Use scripts without act, or install Docker |

## 📖 Full Documentation

See [`../docs/CI-REPRODUCTION-GUIDE.md`](../docs/CI-REPRODUCTION-GUIDE.md) for complete details.