# Cross-Platform CI/CD Solution for Conductor Loop

## Problem Analysis

The CI tests were failing on ubuntu-latest runners with the error:
```
fork/exec .../mock-porch.bat: exec format error
```

**Root Cause**: Windows batch files (.bat) were being executed on Linux systems, causing exec format errors.

## Solution Overview

This document outlines the comprehensive cross-platform solution implemented to resolve CI/CD compatibility issues across Windows, macOS, and Linux environments.

## Key Components

### 1. Cross-Platform Utility Package (`internal/platform/crossplatform.go`)

**Purpose**: Provides runtime OS detection and appropriate script generation.

**Key Features**:
- `runtime.GOOS` detection for OS-specific behavior
- Automatic script extension selection (`.bat` for Windows, `.sh` for Unix)
- Cross-platform executable permission handling
- Unified script generation API

**Key Functions**:
```go
// Get appropriate script extension for current OS
func GetScriptExtension() string

// Create cross-platform executable scripts
func CreateCrossPlatformScript(scriptPath string, scriptType ScriptType, opts ScriptOptions) error

// Check if file is executable on current platform
func IsExecutable(path string) bool

// Make file executable with platform-appropriate permissions
func MakeExecutable(path string) error
```

### 2. Enhanced Test Utilities (`internal/porch/testutil.go`)

**Before**: Hardcoded `.bat` file creation in some functions
**After**: Full cross-platform support with enhanced options

**Improvements**:
- Integration with platform utilities
- Advanced mock creation with custom behaviors
- Pattern-based failure simulation
- Timeout and delay support

**New Functions**:
```go
func CreateAdvancedMock(tempDir string, opts CrossPlatformMockOptions) (string, error)
func GetMockPorchPath(tempDir string) string
func ValidateMockExecutable(mockPath string) error
func CreateFailingMock(tempDir string, exitCode int, errorMessage string) (string, error)
func CreateSlowMock(tempDir string, delay time.Duration) (string, error)
func CreatePatternFailingMock(tempDir string, failPattern string) (string, error)
```

### 3. Comprehensive Test Helper Framework (`testdata/helpers/crossplatform_test_helpers.go`)

**Purpose**: Provides a complete testing environment that works across all platforms.

**Key Components**:
- `CrossPlatformTestEnvironment`: Manages test setup and teardown
- `PlatformSpecificTestRunner`: Runs tests only on appropriate platforms
- `CrossPlatformAssertions`: Platform-aware assertions
- Intent creation utilities

**Usage Example**:
```go
func TestCrossPlatformFeature(t *testing.T) {
    env := helpers.NewCrossPlatformTestEnvironment(t)
    env.SetupSimpleMockPorch()
    env.BuildConductorLoop()
    
    intent := helpers.CreateTestIntent("scale", 3)
    intentPath := env.CreateIntentFile("test.json", intent)
    
    args := env.GetDefaultArgs()
    cmd, err := env.RunConductorLoop(ctx, args...)
    // Test assertions...
}
```

### 4. Updated CI/CD Pipeline (`.github/workflows/conductor-loop-cicd.yml`)

**Enhancements**:
- Multi-platform testing matrix (ubuntu-latest, windows-latest, macos-latest)
- Platform-specific test commands
- Cross-platform path handling
- Enhanced timeout and error handling

**Before**:
```yaml
runs-on: ubuntu-latest
```

**After**:
```yaml
strategy:
  fail-fast: false
  matrix:
    os: [ubuntu-latest, windows-latest, macos-latest]
    go-version: ['1.24']
runs-on: ${{ matrix.os }}
```

## Technical Implementation Details

### Script Generation Logic

#### Windows Script Generation
- Uses `@echo off` for clean output
- Implements `timeout /t` for delays
- Uses `findstr` for pattern matching
- Proper error handling with `exit /b`
- PowerShell integration for precise timing

#### Unix Script Generation  
- Uses `#!/bin/bash` shebang
- Implements `sleep` for delays
- Uses `grep` for pattern matching
- Proper error handling with `exit`
- POSIX compliance for maximum compatibility

### File Permission Handling

#### Windows
- Relies on file extensions (`.exe`, `.bat`, `.cmd`)
- Uses standard file permissions (0644)
- No executable bit required

#### Unix (Linux/macOS)
- Sets executable permissions (0755)
- Checks executable bit with `info.Mode()&0111`
- Supports chmod operations

### Cross-Platform Path Handling

```go
// Platform-aware path construction
func GetScriptPath(dir, baseName string) string {
    return filepath.Join(dir, baseName+GetScriptExtension())
}

// Executable detection with extension support
func GetExecutablePath(dir, baseName string) string {
    if runtime.GOOS == "windows" {
        // Check for .exe, .bat, .cmd extensions
        for _, ext := range []string{".exe", ".bat", ".cmd"} {
            path := filepath.Join(dir, baseName+ext)
            if _, err := os.Stat(path); err == nil {
                return path
            }
        }
        return filepath.Join(dir, baseName+".exe")
    }
    return filepath.Join(dir, baseName)
}
```

## Testing Strategy

### Unit Tests
- Platform utility tests (`internal/platform/crossplatform_test.go`)
- Test helper validation (`testdata/helpers/crossplatform_test_helpers_test.go`)
- Mock script behavior verification

### Integration Tests
- Cross-platform workflow validation
- Multi-OS CI pipeline testing
- Performance benchmarking

### Platform-Specific Tests
```go
runner := helpers.NewPlatformSpecificTestRunner(t)

runner.RunOnWindows(func() {
    // Windows-specific test logic
})

runner.RunOnUnix(func() {
    // Unix-specific test logic
})

runner.RunOnLinux(func() {
    // Linux-specific test logic
})
```

## Benefits of This Solution

### 1. **Complete Cross-Platform Compatibility**
- Works seamlessly on Windows, macOS, and Linux
- Automatic OS detection and adaptation
- No manual platform-specific configuration required

### 2. **Robust CI/CD Pipeline**
- Multi-platform testing matrix
- Early detection of platform-specific issues
- Comprehensive test coverage across environments

### 3. **Maintainable Codebase**
- Centralized platform logic
- Reusable test utilities
- Clear separation of concerns

### 4. **Developer Experience**
- Simple API for cross-platform testing
- Comprehensive test helpers
- Platform-aware assertions

### 5. **Production Readiness**
- Zero-downtime deployment compatibility
- Container-ready (works in distroless containers)
- Security-focused (proper permission handling)

## Migration Guide

### For Existing Tests

**Before**:
```go
// Old hardcoded approach
mockFile := filepath.Join(tempDir, "mock-porch.bat")
mockScript := `@echo off
echo processing completed
exit /b 0`
os.WriteFile(mockFile, []byte(mockScript), 0755)
```

**After**:
```go
// New cross-platform approach
mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
    ExitCode: 0,
    Stdout:   "processing completed",
})
```

### For New Tests

```go
func TestNewFeature(t *testing.T) {
    env := helpers.NewCrossPlatformTestEnvironment(t)
    env.SetupSimpleMockPorch()
    
    // Your test logic here...
}
```

## Verification Commands

### Local Testing
```bash
# Test on current platform
go test -v ./internal/platform
go test -v ./internal/porch
go test -v ./testdata/helpers

# Test conductor-loop functionality
go test -v ./cmd/conductor-loop -run TestOnceMode
```

### CI Pipeline Testing
The updated CI workflow automatically tests across:
- Ubuntu Latest (Linux)
- Windows Latest 
- macOS Latest

## File Structure Summary

```
internal/
├── platform/
│   ├── crossplatform.go           # Core cross-platform utilities
│   └── crossplatform_test.go      # Platform utility tests
└── porch/
    ├── testutil.go                # Enhanced test utilities
    └── testutil_test.go           # Test utility validation

testdata/
└── helpers/
    ├── crossplatform_test_helpers.go      # Comprehensive test framework
    └── crossplatform_test_helpers_test.go # Framework validation

cmd/conductor-loop/
└── main_test.go                   # Updated to use cross-platform utilities

.github/workflows/
└── conductor-loop-cicd.yml        # Multi-platform CI pipeline
```

## Best Practices

### 1. **Always Use Cross-Platform Utilities**
```go
// ✅ Good
mockPath := platform.GetScriptPath(tempDir, "mock-porch")
err := platform.CreateCrossPlatformScript(mockPath, platform.MockPorchScript, opts)

// ❌ Avoid
mockPath := filepath.Join(tempDir, "mock-porch.bat") // Windows-specific
```

### 2. **Leverage Test Environment Helpers**
```go
// ✅ Good
env := helpers.NewCrossPlatformTestEnvironment(t)
env.SetupMockPorch(0, "success", "")

// ❌ Avoid
tempDir := t.TempDir()
// Manual directory setup...
```

### 3. **Use Platform-Specific Testing When Needed**
```go
runner := helpers.NewPlatformSpecificTestRunner(t)
runner.RunOnWindows(func() {
    // Windows-specific behavior testing
})
```

### 4. **Validate Executables Properly**
```go
assertions := helpers.NewCrossPlatformAssertions(t)
assertions.AssertExecutableExists(mockPath)
assertions.AssertScriptWorks(mockPath)
```

## Conclusion

This comprehensive cross-platform solution eliminates the CI/CD compatibility issues while providing a robust, maintainable, and extensible framework for future development. The implementation follows deployment engineering best practices with:

- **Automation**: No manual platform-specific steps required
- **Consistency**: Single API works across all platforms  
- **Reliability**: Comprehensive testing and validation
- **Maintainability**: Clear separation of concerns and reusable components
- **Performance**: Minimal overhead with efficient implementation

The solution is production-ready and integrates seamlessly with the existing conductor-loop architecture while preparing the codebase for future multi-platform deployments.