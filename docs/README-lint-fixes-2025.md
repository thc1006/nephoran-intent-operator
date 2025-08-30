# Go Linting Fixes for 2025 - Complete Guide

## Overview

This guide provides comprehensive templates and tools for fixing common golangci-lint issues based on 2025 best practices and Go 1.24+ requirements. It includes ready-to-apply solutions for the most common linting errors in Kubernetes operators and telecommunications software.

## 🚀 Quick Start

### Option 1: Automated Script (Recommended)
```bash
# Linux/macOS
./scripts/apply-lint-fixes-2025.sh all

# Windows PowerShell
.\scripts\apply-lint-fixes-2025.ps1 all
```

### Option 2: Manual Application
```bash
# Check requirements
./scripts/apply-lint-fixes-2025.sh check

# Create backup
./scripts/apply-lint-fixes-2025.sh backup

# Apply specific fixes
./scripts/apply-lint-fixes-2025.sh imports format comments errors

# Run linter and generate report
./scripts/apply-lint-fixes-2025.sh lint report
```

## 📋 What's New in 2025

### Go 1.24 Features Supported
- **Generic Type Aliases**: Full support for parameterized type aliases
- **Tool Directive in go.mod**: Version-controlled tool dependencies
- **omitzero Struct Field Option**: Enhanced JSON marshaling control
- **Enhanced Performance**: Optimized runtime and compiler improvements

### Updated Linter Configuration
- **golangci-lint v2.x compatibility**: New configuration format
- **35+ enabled linters**: Comprehensive code quality checks
- **Security-first approach**: Enhanced security linting for telecom
- **Context-aware patterns**: Modern Go concurrency patterns
- **Performance optimizations**: Preallocation and efficiency checks

## 🛠️ Tools and Files

### Core Files
- `docs/lint-fix-templates-2025.md` - Template patterns and examples
- `.golangci-2025.yml` - Modern linter configuration
- `scripts/apply-lint-fixes-2025.sh` - Linux/macOS automation script
- `scripts/apply-lint-fixes-2025.ps1` - Windows PowerShell script
- `scripts/lint-fixes-2025.go` - Go-based AST manipulation tool

### Prerequisites
```bash
# Install required tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3
go install golang.org/x/tools/cmd/goimports@latest
go install mvdan.cc/gofumpt@latest
```

## 📖 Common Fix Patterns

### 1. Package Documentation (ST1000, package-comments)
```go
// ❌ Bad: No package comment
package clients

// ✅ Good: Complete package documentation
// Package clients provides mTLS-enabled client factories for O-RAN network services.
// It supports secure communication with LLM processors, RAG services, Git repositories,
// and monitoring systems in Kubernetes-based telecommunications environments.
package clients
```

### 2. Exported Function Documentation (revive, exported)
```go
// ❌ Bad: No documentation
func NewMTLSClientFactory(config *ClientFactoryConfig) (*MTLSClientFactory, error) {

// ✅ Good: Complete function documentation
// NewMTLSClientFactory creates a new mTLS client factory for secure service communication.
// It initializes the factory with CA management and identity provisioning capabilities.
//
// The factory supports auto-provisioning of service identities and certificate rotation
// for O-RAN network services in Kubernetes environments.
//
// Returns an error if the configuration is invalid or required dependencies are missing.
func NewMTLSClientFactory(config *ClientFactoryConfig) (*MTLSClientFactory, error) {
```

### 3. Error Wrapping (errorlint)
```go
// ❌ Bad: Error formatting without wrapping
return fmt.Errorf("failed to create client: %s", err.Error())

// ✅ Good: Error wrapping with %w
return fmt.Errorf("failed to create mTLS HTTP client for %s service: %w", serviceType, err)
```

### 4. Context Patterns (contextcheck)
```go
// ✅ Good: Context-first parameter pattern
func (f *MTLSClientFactory) CreateClient(ctx context.Context, serviceType ServiceType, options ...Option) (*Client, error) {
	// Check context before expensive operations
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context canceled before client creation: %w", err)
	}
	
	// Use context in HTTP requests
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	return f.createClientInternal(ctx, serviceType, options...)
}
```

### 5. Unused Variables (unused, ineffassign)
```go
// ❌ Bad: Unused parameter
func (f *MTLSClientFactory) processRequest(ctx context.Context, data []byte) error {
	// data parameter not used
	return f.doSomething(ctx)
}

// ✅ Good: Use underscore for intentionally unused
func (f *MTLSClientFactory) processRequest(ctx context.Context, _ []byte) error {
	return f.doSomething(ctx)
}
```

### 6. Security Patterns (gosec)
```go
// ❌ Bad: Weak random number generator
import "math/rand"
token := rand.Int63()

// ✅ Good: Cryptographically secure random
import "crypto/rand"
import "math/big"

func generateSecureToken() (int64, error) {
	max := big.NewInt(1 << 62)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, fmt.Errorf("failed to generate secure random number: %w", err)
	}
	return n.Int64(), nil
}
```

## 🔧 Automated Tools Usage

### Script Commands
```bash
# Check prerequisites
./scripts/apply-lint-fixes-2025.sh check

# Create backup before changes
./scripts/apply-lint-fixes-2025.sh backup

# Apply specific types of fixes
./scripts/apply-lint-fixes-2025.sh imports    # Fix import formatting
./scripts/apply-lint-fixes-2025.sh format     # Apply strict formatting
./scripts/apply-lint-fixes-2025.sh comments   # Add missing package comments
./scripts/apply-lint-fixes-2025.sh errors     # Fix error wrapping patterns

# Run linter with auto-fix
./scripts/apply-lint-fixes-2025.sh lint

# Generate detailed report
./scripts/apply-lint-fixes-2025.sh report

# Validate go.mod
./scripts/apply-lint-fixes-2025.sh mod

# Run tests to ensure fixes work
./scripts/apply-lint-fixes-2025.sh test

# Apply all fixes at once
./scripts/apply-lint-fixes-2025.sh all

# Restore from backup if needed
./scripts/apply-lint-fixes-2025.sh restore
```

### Go AST Tool Usage
```bash
# Compile the Go tool
go build -o lint-fixer scripts/lint-fixes-2025.go

# Apply fixes to directory
./lint-fixer ./pkg/

# Show quick fix for specific linter
./lint-fixer help ST1000
./lint-fixer help errorlint
./lint-fixer help missing-doc-exported
```

## 🎯 Linter Configuration Highlights

### Key Enabled Linters (2025)
```yaml
# Core quality
- revive          # Comprehensive replacement for golint
- staticcheck     # Advanced static analysis
- govet           # Standard Go vet checks
- gosimple        # Simplify code suggestions
- unused          # Find unused code
- typecheck       # Type-checking errors

# Error handling
- errcheck        # Check for unchecked errors
- errorlint       # Error wrapping with %w
- contextcheck    # Context usage patterns
- nilerr          # Return nil even if error is not nil

# Security (critical for telecom)
- gosec           # Security vulnerabilities
- G101-G110       # Specific security checks

# Performance
- ineffassign     # Ineffective assignments
- prealloc        # Preallocate slices
- gocritic        # Comprehensive performance checks
- noctx           # HTTP requests without context
- bodyclose       # HTTP response body close

# Modern Go features (Go 1.18+)
- copyloopvar     # Loop variable copying (automatic in Go 1.22+)
- intrange        # Integer range patterns
- testifylint     # Testify usage patterns

# Kubernetes/Cloud-native specific
- containedctx    # Context contained in struct
- fatcontext      # Fat context usage
```

### Performance Optimizations
```yaml
run:
  timeout: 20m
  concurrency: 0      # Use all CPU cores
  modules-download-mode: readonly
  
linters-settings:
  gocritic:
    enabled-tags:
      - diagnostic
      - style  
      - performance
      - experimental
      - opinionated
```

## 🚨 Common Issues and Solutions

### Issue: "ST1000: at least one file in a package should have a package comment"
**Solution:**
```go
// Add at the top of any .go file in the package:
// Package <name> provides <description of what the package does>.
package <name>
```

### Issue: "should have comment or be unexported"
**Solution:**
```go
// Add documentation for exported functions:
// FunctionName does something specific and returns a result.
// It handles errors by returning them wrapped with additional context.
func FunctionName() error {
```

### Issue: "Error strings should not be capitalized"
**Solution:**
```go
// ❌ Bad
return errors.New("This error message is capitalized")

// ✅ Good  
return errors.New("this error message is not capitalized")
```

### Issue: "printf: non-constant format string in call to fmt.Errorf"
**Solution:**
```go
// ❌ Bad
return fmt.Errorf(message, args...)

// ✅ Good
return fmt.Errorf("operation failed: %s", message)
```

### Issue: "ineffective assignment to field X"
**Solution:**
```go
// ❌ Bad
config.Field = config.Field

// ✅ Good - Remove the ineffective line or make it meaningful
if needsUpdate {
    config.Field = newValue
}
```

## 📊 Metrics and Reporting

### Generated Reports
- `lint-report-YYYYMMDD_HHMMSS.json` - Detailed JSON report
- `golangci-report.json` - Standard golangci-lint JSON output
- `.last-backup` - Path to last backup for restoration

### Viewing Reports
```bash
# If jq is installed, you can analyze reports:
cat lint-report-*.json | jq '.Issues | group_by(.FromLinter) | sort_by(length) | reverse'

# Count issues by severity
cat lint-report-*.json | jq '.Issues | group_by(.Severity) | map({severity: .[0].Severity, count: length})'

# Find most common issues
cat lint-report-*.json | jq '.Issues | group_by(.Text) | sort_by(length) | reverse | .[:10] | map({issue: .[0].Text, count: length})'
```

## 🔄 Integration with CI/CD

### GitHub Actions Example
```yaml
name: Lint with 2025 Standards
on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          
      - name: Install tools
        run: |
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3
          go install golang.org/x/tools/cmd/goimports@latest
          go install mvdan.cc/gofumpt@latest
          
      - name: Apply lint fixes
        run: ./scripts/apply-lint-fixes-2025.sh all
        
      - name: Check for changes
        run: |
          if [[ -n $(git status --porcelain) ]]; then
            echo "Linting fixes were applied. Please commit these changes."
            git diff
            exit 1
          fi
```

### Pre-commit Hook
```bash
#!/bin/sh
# .git/hooks/pre-commit
./scripts/apply-lint-fixes-2025.sh lint
```

## 🎓 Best Practices

### 1. **Always Create Backups**
```bash
# Before making any changes
./scripts/apply-lint-fixes-2025.sh backup
```

### 2. **Run Tests After Fixes**
```bash
# Ensure fixes don't break functionality
go test ./...
```

### 3. **Use Specific Fixes for Development**
```bash
# During development, apply specific fixes
./scripts/apply-lint-fixes-2025.sh imports format
```

### 4. **Review Changes Before Commit**
```bash
# Check what was changed
git diff
```

### 5. **Use Configuration Comments**
```go
//nolint:gosec // G204: Command executed with variable, but input is validated
cmd := exec.Command("kubectl", args...)
```

## 🆘 Troubleshooting

### Script Fails with "Missing required tools"
**Solution:** Install the missing tools as shown in the error message.

### Tests Fail After Applying Fixes
**Solution:** 
```bash
# Restore from backup
./scripts/apply-lint-fixes-2025.sh restore

# Apply fixes gradually
./scripts/apply-lint-fixes-2025.sh imports
go test ./...
./scripts/apply-lint-fixes-2025.sh format  
go test ./...
```

### Linter Reports Issues Not Fixed by Script
**Solution:** Check the manual fix templates in `docs/lint-fix-templates-2025.md` and apply the appropriate pattern.

### Permission Denied on Script
**Solution:**
```bash
chmod +x scripts/apply-lint-fixes-2025.sh
```

## 📚 Additional Resources

- [Go 1.24 Release Notes](https://tip.golang.org/doc/go1.24)
- [golangci-lint v2 Documentation](https://golangci-lint.run/docs/)
- [Effective Go Guide](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

## 🤝 Contributing

When contributing to the project:

1. Run the linting fixes before submitting PRs:
   ```bash
   ./scripts/apply-lint-fixes-2025.sh all
   ```

2. Add appropriate documentation for exported functions

3. Follow the error wrapping patterns with `%w`

4. Use context-first parameter ordering

5. Ensure tests pass after applying fixes

This comprehensive guide ensures your Go code meets 2025 standards and passes modern linter requirements while maintaining functionality and readability.