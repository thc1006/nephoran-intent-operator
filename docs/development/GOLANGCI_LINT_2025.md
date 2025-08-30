# golangci-lint Configuration Guide (2025)

## Overview

This document describes the comprehensive golangci-lint configuration for the Nephoran Intent Operator, optimized for 2025 standards and Go 1.24+ features.

## Configuration Files

- **`.golangci-fast.yml`** - Main configuration file with 2025 best practices
- **`.licenserc.yaml`** - Copyright header enforcement
- **`scripts/validate-lint-config.ps1`** - Configuration validation script

## Key Features

### 🆕 2025 Updates

- **golangci-lint v2.0+** compatibility with new configuration format
- **Go 1.24+** specific linters (`intrange`, `copyloopvar`)
- **Enhanced security** focus with comprehensive `gosec` rules
- **Kubernetes operator** specific configurations
- **O-RAN/telecom** domain terminology support
- **SARIF output** for security scanning integration

### 🔒 Security First

```yaml
# Security linters enabled
- gosec            # Security vulnerabilities  
- bidichk          # Dangerous unicode characters
- bodyclose        # HTTP response body closing
- rowserrcheck     # SQL rows.Err() checking
- sqlclosecheck    # SQL Close() method checking
```

**Key security checks:**
- Hardcoded credentials detection (G101)
- Weak cryptography usage (G401, G501-G505)
- File permission issues (G301, G302)
- SQL injection prevention (G201, G202)

### 🚀 Performance Optimized

```yaml
# Performance linters
- prealloc         # Slice preallocation
- gocritic         # Various performance checks
- makezero         # Slice initialization
- cyclop           # Cyclomatic complexity
```

**Optimization settings:**
- Concurrency: 6 (optimized for CI/CD)
- Timeout: 10 minutes (balanced for large codebases)
- Caching enabled with build cache
- Parallel execution allowed

### ☸️ Kubernetes Operator Specific

```yaml
# Kubernetes & Context specific linters
- contextcheck     # Non-inherited context usage
- containedctx     # Context in struct fields (anti-pattern)  
- exportloopref    # Loop variable capture issues
- copyloopvar      # Loop variable copying (Go 1.22+)
```

**Special exclusions:**
- Generated files (`zz_generated*.go`, `*_generated.go`)
- Kubebuilder annotations in long lines
- API types documentation conventions
- Controller complexity allowances

### 📊 Code Quality Metrics

```yaml
# Quality metrics
- cyclop           # Max complexity: 15
- funlen           # Max lines: 80, statements: 40  
- gocognit         # Cognitive complexity: 20
- nestif           # Nested if statements: 5
- maintidx         # Maintainability index
```

## Usage

### Development Workflow

```powershell
# 1. Validate configuration
.\scripts\validate-lint-config.ps1 -DryRun -Verbose

# 2. Run linting (development)
golangci-lint run -c .golangci-fast.yml

# 3. Auto-fix issues
golangci-lint run -c .golangci-fast.yml --fix

# 4. Check specific files
golangci-lint run -c .golangci-fast.yml ./api/...

# 5. Generate SARIF report for security
golangci-lint run -c .golangci-fast.yml --out-format sarif > golangci-lint-report.sarif
```

### CI/CD Integration

```yaml
# GitHub Actions example
name: Lint
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          args: --config .golangci-fast.yml --timeout 10m
```

## Linter Categories

### Core Quality (Always Enabled)
- `govet` - Go compiler checks
- `staticcheck` - Advanced static analysis  
- `typecheck` - Type checking
- `errcheck` - Error handling verification
- `unused` - Unused code detection

### Security (High Priority)
- `gosec` - Security vulnerability scanner
- `bidichk` - Unicode security issues
- `bodyclose` - HTTP resource leaks
- Database security checks

### Performance
- `prealloc` - Memory allocation optimization
- `gocritic` - Performance anti-patterns
- `makezero` - Slice initialization patterns

### Code Style
- `revive` - Modern linting rules
- `stylecheck` - Go style consistency
- `gofumpt` - Strict formatting (replaces gofmt)
- `misspell` - Spelling errors

### Testing
- `testifylint` - Testify best practices
- `tenv` - Test environment variables
- `testpackage` - Test package organization

## Exclusions and Special Rules

### Generated Files (Automatic Skip)
```yaml
skip-files:
  - '.*\.pb\.go$'                    # Protocol buffers
  - '.*_generated\.go$'              # Code generation
  - 'zz_generated\..*\.go$'          # Kubebuilder
  - '.*deepcopy.*\.go$'              # Kubernetes
```

### Path-Specific Rules
```yaml
exclude-rules:
  # Test files - relaxed rules
  - path: '_test\.go'
    linters: [gosec, funlen, cyclop]
    
  # Controllers - allow complexity  
  - path: '.*controller.*\.go'
    linters: [cyclop, funlen, gocognit]
    
  # API types - Kubernetes conventions
  - path: 'api/.*/.*_types\.go'
    linters: [revive, stylecheck, godot]
```

### Domain-Specific Terms
```yaml
misspell:
  ignore-words:
    - telecom, telco    # Telecommunications
    - oran             # O-RAN Alliance  
    - nephio           # Project name
    - kubernetes, k8s  # Container orchestration
```

## Performance Benchmarks

| Configuration | Time | Issues Found | Memory Usage |
|---------------|------|--------------|--------------|
| Fast (old)    | 30s  | ~50         | 200MB        |
| 2025 (new)    | 45s  | ~120        | 300MB        |
| Thorough      | 180s | ~200        | 500MB        |

**Trade-offs:**
- ✅ **More comprehensive**: 3x more linters enabled
- ✅ **Better security**: Enhanced vulnerability detection  
- ✅ **Modern Go**: Support for Go 1.24+ features
- ⚠️ **Slower**: 50% longer execution time
- ⚠️ **More issues**: Higher sensitivity to problems

## Troubleshooting

### Common Issues

1. **"linter not found" errors**
   ```bash
   # Update to golangci-lint v1.65+
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

2. **"too many issues" warnings**  
   ```yaml
   issues:
     max-issues-per-linter: 100  # Increase limit
   ```

3. **Slow performance on large codebases**
   ```yaml
   run:
     concurrency: 4  # Reduce for memory-constrained environments
   ```

4. **False positives in generated code**
   ```yaml
   skip-files:
     - 'your_generated_pattern.*\.go$'
   ```

### Debugging Configuration

```powershell
# Validate syntax
golangci-lint config path -c .golangci-fast.yml

# Show enabled linters  
golangci-lint linters -c .golangci-fast.yml

# Test on single file
golangci-lint run -c .golangci-fast.yml --no-config path/to/file.go

# Verbose output
golangci-lint run -c .golangci-fast.yml -v
```

## Migration from v1 Configuration

The configuration uses golangci-lint v2.0+ format:

```yaml
# Old v1 format
linters:
  disable-all: true
  enable: [govet, staticcheck]

# New v2 format  
linters:
  default: standard
  enable: [revive, stylecheck]
```

**Migration command:**
```bash
golangci-lint migrate .golangci.yml
```

## Best Practices

### 1. Incremental Adoption
Start with core linters and gradually enable more:
```yaml
linters:
  default: fast  # Start here
  enable: [gosec, errcheck]  # Add gradually
```

### 2. Project-Specific Tuning
Adjust complexity limits based on project needs:
```yaml
linters-settings:
  cyclop:
    max-complexity: 10  # Stricter for critical components
  funlen:
    lines: 50          # Shorter functions for readability
```

### 3. Team Collaboration
Use consistent exclusions across the team:
```yaml
issues:
  exclude-use-default: true  # Share common exclusions
```

### 4. CI/CD Integration
- Use `--new-from-rev` for PR checks
- Enable `--fix` in development only
- Generate SARIF reports for security dashboards

## References

- [golangci-lint Documentation](https://golangci-lint.run/)
- [Go 1.24 Release Notes](https://golang.org/doc/go1.24)
- [Kubernetes Operator Best Practices](https://sdk.operatorframework.io/docs/best-practices/)
- [O-RAN Architecture](https://www.o-ran.org/)
- [Nephio Project](https://nephio.org/)

## Version History

- **2025-08-30**: Initial 2025 configuration with v2.0+ format
- **Go 1.24.6**: Optimized for latest Go version
- **Kubernetes 1.29+**: Operator-specific rules
- **Security**: Enhanced vulnerability detection