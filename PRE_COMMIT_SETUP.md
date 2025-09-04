# Pre-commit Hooks for Nephoran Intent Operator

## Overview

This setup provides comprehensive pre-commit hooks to prevent invalid golangci-lint configurations from being committed to the repository. The hooks are designed specifically for the Nephoran Intent Operator's DevOps pipeline.

## Files Created

### 1. Pre-commit Configuration
- **`.pre-commit-config.yaml`** - Main pre-commit configuration with multiple validation hooks
- **`.yamllint.yml`** - YAML linting configuration optimized for golangci-lint configs

### 2. Hook Scripts
- **`scripts/hooks/validate-golangci-config.sh`** - Comprehensive golangci-lint config validation
- **`scripts/hooks/check-lint-compatibility.sh`** - Version compatibility checking
- **`scripts/install-hooks.sh`** - Automated installer for pre-commit framework

### 3. Makefile Integration
Added new targets to the Makefile:
- `make install-hooks` - Install pre-commit hooks
- `make validate-configs` - Validate golangci-lint configurations
- `make test-hooks` - Test hook installation
- `make run-hooks` - Run hooks on staged files
- `make run-hooks-all` - Run hooks on all files
- `make update-hooks` - Update hook versions
- `make uninstall-hooks` - Remove hooks

## Features

### Validation Capabilities
1. **YAML Syntax Validation** - Ensures golangci-lint configs are valid YAML
2. **Configuration Structure** - Validates golangci-lint can parse the config
3. **Linter Compatibility** - Checks for deprecated or invalid linters
4. **Version Compatibility** - Verifies golangci-lint version compatibility
5. **Go Version Alignment** - Ensures Go version consistency
6. **Performance Checks** - Warns about high timeout/concurrency values
7. **Dry Run Testing** - Tests configuration with actual linting

### Pre-commit Hooks Active
- ✅ `validate-golangci-config` - Validates .golangci*.yml files
- ✅ `syntax-check-golangci` - Checks golangci-lint can parse config
- ✅ `lint-config-compatibility` - Checks version compatibility
- ✅ `go-fmt` - Formats Go code
- ✅ `go-vet` - Runs go vet
- ✅ `go-mod-tidy` - Ensures go.mod/go.sum are clean
- ✅ `golangci-lint-fast` - Quick lint on changed files
- ✅ `yamllint` - YAML syntax validation
- ✅ Generic checks - File size, merge conflicts, etc.

## Installation

### Automatic Installation
```bash
# Run the installer script
chmod +x scripts/install-hooks.sh
./scripts/install-hooks.sh

# Or use Make
make install-hooks
```

### Manual Installation
```bash
# Install pre-commit framework
pip3 install pre-commit

# Install golangci-lint (if not already installed)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3

# Install hooks from configuration
pre-commit install
```

## Usage

### Automatic Usage
Hooks run automatically on every `git commit`. They will:
1. Validate any changed .golangci*.yml files
2. Format and lint Go code
3. Check for common issues
4. Prevent commit if validation fails

### Manual Usage
```bash
# Run all hooks on staged files
pre-commit run

# Run all hooks on all files
pre-commit run --all-files

# Run specific hook
pre-commit run validate-golangci-config

# Test hooks installation
make test-hooks

# Validate configurations manually
make validate-configs
```

### Bypassing Hooks (Use Sparingly)
```bash
# Skip all hooks
git commit --no-verify

# Skip specific hooks
SKIP=golangci-lint-fast git commit

# Skip multiple hooks
SKIP=golangci-lint-fast,go-fmt git commit
```

## Configuration

### Expected Versions
- **golangci-lint**: v1.64.3 (CI compatible)
- **Go**: 1.24+ (as per project requirements)
- **pre-commit**: 2.20.0+

### Customization
Edit `.pre-commit-config.yaml` to:
- Add/remove hooks
- Modify hook arguments
- Change file patterns
- Adjust hook behavior

## Troubleshooting

### Common Issues

1. **golangci-lint not found**
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3
   export PATH="$(go env GOPATH)/bin:$PATH"
   ```

2. **pre-commit not found**
   ```bash
   pip3 install pre-commit
   # On some systems: pip install pre-commit
   ```

3. **Hooks not running**
   ```bash
   pre-commit install  # Reinstall hooks
   pre-commit run --all-files  # Test manually
   ```

4. **YAML validation issues**
   - The validation uses golangci-lint itself as the primary validator
   - yamllint is optional and falls back to basic validation
   - Check `.golangci-fast.yml` syntax manually if issues persist

### Debug Commands
```bash
# Check hook status
pre-commit run --all-files --dry-run

# Validate specific config
scripts/hooks/validate-golangci-config.sh .golangci-fast.yml

# Test compatibility
scripts/hooks/check-lint-compatibility.sh

# Check golangci-lint directly
golangci-lint config path -c .golangci-fast.yml
golangci-lint linters -c .golangci-fast.yml
```

## Benefits

1. **Prevents Broken Configs** - Invalid configurations can't be committed
2. **CI Pipeline Stability** - Ensures CI won't fail due to config issues
3. **Version Consistency** - Maintains compatibility across development environment
4. **Developer Experience** - Fast feedback loop with local validation
5. **Code Quality** - Automatic formatting and linting integration
6. **Team Consistency** - Standardized validation across all developers

## Integration with CI

The hooks are designed to complement the existing CI pipeline:
- Local validation prevents CI failures
- Same golangci-lint version used locally and in CI
- Fast local feedback reduces development cycle time
- Comprehensive validation ensures quality standards

## Maintenance

### Updating Hooks
```bash
# Update to latest versions
pre-commit autoupdate

# Or use Make
make update-hooks
```

### Monitoring
- Check hook performance regularly
- Update golangci-lint version when CI updates
- Monitor for new linter compatibility issues
- Keep pre-commit framework updated

## Security

- All scripts validate input and use safe practices
- No secrets or sensitive data in configurations
- Version pinning prevents supply chain attacks
- Read-only operations for most validations

---

✅ **Pre-commit hooks successfully configured for Nephoran Intent Operator!**

The setup prevents invalid golangci-lint configurations from ever reaching the repository, ensuring stable CI/CD pipeline operation.