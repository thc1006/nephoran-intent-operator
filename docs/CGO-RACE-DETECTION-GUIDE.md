# CGO and Race Detection Configuration Guide

## Overview

This guide addresses the critical CGO (C Go) and race detection setup issues that prevent proper concurrency testing in Go applications. Race detection is essential for identifying data races and concurrency bugs before they reach production.

## Problem Statement

The following errors indicate CGO configuration issues:
```
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
/usr/bin/bash: line 1: CGO_ENABLED=1: command not found
```

**Root Cause**: Race detection in Go requires CGO, which needs a C compiler. The environment variable `CGO_ENABLED` must be set correctly for your platform.

## Solution

### Platform-Specific CGO Setup

#### Windows PowerShell
```powershell
# Set CGO environment variable
$env:CGO_ENABLED = "1"

# Run tests with race detection
go test -race ./...

# Or inline for single command
$env:CGO_ENABLED="1"; go test -race ./...
```

#### Windows Command Prompt (CMD)
```cmd
REM Set CGO environment variable
set CGO_ENABLED=1

REM Run tests with race detection
go test -race ./...

REM Or combined
set CGO_ENABLED=1 && go test -race ./...
```

#### Linux/macOS (Bash/Zsh)
```bash
# Export for session
export CGO_ENABLED=1
go test -race ./...

# Or inline for single command
CGO_ENABLED=1 go test -race ./...
```

#### GitHub Actions / CI
```yaml
env:
  CGO_ENABLED: '1'
  
steps:
  - name: Run race detection tests
    run: go test -race ./...
```

## Installing C Compilers

### Windows

Race detection requires a C compiler. Choose one of these options:

#### Option 1: TDM-GCC (Recommended)
- Download from: https://jmeubank.github.io/tdm-gcc/
- Run installer, select default options
- Restart terminal after installation

#### Option 2: MinGW-w64
- Download from: https://www.mingw-w64.org/downloads/
- Add to PATH: `C:\mingw64\bin`

#### Option 3: MSYS2
- Download from: https://www.msys2.org/
- After install, run: `pacman -S mingw-w64-x86_64-gcc`

#### Option 4: Chocolatey
```powershell
choco install mingw
```

#### Option 5: Scoop
```powershell
scoop install gcc
```

### Linux

#### Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install build-essential
```

#### RHEL/CentOS/Fedora
```bash
# Fedora
sudo dnf groupinstall 'Development Tools'

# RHEL/CentOS
sudo yum groupinstall 'Development Tools'
```

#### Arch Linux
```bash
sudo pacman -S base-devel
```

#### Alpine Linux
```bash
apk add build-base
```

### macOS
```bash
# Install Xcode Command Line Tools
xcode-select --install
```

## Verification Scripts

### Quick CGO Check
Run the provided script to verify your CGO setup:

```powershell
# Windows PowerShell
.\scripts\check-cgo-setup.ps1

# Linux/macOS
./scripts/check-cgo-setup.sh
```

### Comprehensive Race Detection Test
```powershell
# Windows PowerShell
.\scripts\test-race-detection.ps1

# Linux/macOS
./scripts/test-race-detection.sh
```

## CI/CD Configuration

### GitHub Actions Workflow

See `.github/workflows/race-detection.yml` for a complete CI setup that:
- Installs necessary C compilers
- Configures CGO properly
- Runs race detection tests
- Handles multiple platforms

### Key CI Environment Variables
```yaml
env:
  GO_VERSION: '1.22'
  CGO_ENABLED: '1'
  GORACE: "halt_on_error=1"  # Stop on first race detection
```

## Troubleshooting

### Common Issues

#### 1. "gcc not found" on Windows
- **Solution**: Install TDM-GCC or MinGW (see installation section)
- **Verify**: Run `gcc --version` in a new terminal

#### 2. "CGO_ENABLED=1: command not found" in bash
- **Issue**: Incorrect syntax
- **Solution**: Use `CGO_ENABLED=1 go test` (no semicolon)

#### 3. Race detector not finding races
- **Verify CGO**: Run `go env CGO_ENABLED`
- **Check compiler**: Run `gcc --version` or `clang --version`
- **Test with known race**: Use provided test scripts

#### 4. Slow test execution with race detection
- **Expected**: Race detection adds 2-10x overhead
- **Solution**: Use `-timeout` flag appropriately
- **Optimize**: Run race tests on critical packages only

### Debug Commands

```bash
# Check Go environment
go env CGO_ENABLED
go env CC
go env CXX

# Verify C compiler
gcc --version
clang --version

# Test CGO compilation
cat > test.go << 'EOF'
package main
// #include <stdio.h>
import "C"
func main() { C.printf("CGO works!\n") }
EOF
CGO_ENABLED=1 go run test.go
```

## Best Practices

1. **Always enable CGO for race tests**: Race detection won't work without it
2. **Use appropriate timeouts**: Race tests run slower (2-10x)
3. **Test locally before CI**: Ensure your development environment has CGO
4. **Document compiler requirements**: Add to project README
5. **Cache CI dependencies**: Speed up builds by caching compiler tools

## Performance Considerations

Race detection adds overhead:
- **Memory**: 5-10x increase
- **CPU**: 2-10x slower execution
- **Recommendation**: Run race tests separately from regular tests

```bash
# Regular tests (fast)
go test ./...

# Race detection tests (slower, separate job)
CGO_ENABLED=1 go test -race ./...
```

## Project-Specific Configuration

This project includes:
- **Scripts**: `scripts/test-race-detection.ps1` and `.sh` for cross-platform testing
- **CI Workflow**: `.github/workflows/race-detection.yml` for automated testing
- **Quick Check**: `scripts/check-cgo-setup.ps1` for verification

## Summary

Proper CGO configuration is essential for:
- Race condition detection
- Memory safety verification
- Concurrent code validation
- Production-ready code quality

Follow this guide to ensure your development and CI environments are properly configured for comprehensive Go testing with race detection enabled.