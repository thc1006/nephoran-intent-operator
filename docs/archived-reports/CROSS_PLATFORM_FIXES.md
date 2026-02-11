# Cross-Platform Compatibility Fixes for CI Workflows

## Summary of Changes

This document outlines the fixes implemented to resolve cross-platform compatibility issues in the CI workflows and test infrastructure.

## Issues Fixed

### 1. Timeout Command Issue on macOS

**Problem**: The `timeout` command is not available by default on macOS runners in GitHub Actions, causing CI failures.

**Solution**: Implemented cross-platform timeout detection in CI workflows.

**Files Modified**:
- `.github/workflows/conductor-loop.yml`
- `.github/workflows/conductor-loop-cicd.yml`

**Changes**:
```bash
# Use cross-platform timeout solution
if command -v gtimeout >/dev/null 2>&1; then
  TIMEOUT_CMD="gtimeout 300"
elif command -v timeout >/dev/null 2>&1; then
  TIMEOUT_CMD="timeout 300"
else
  # Fallback: use Go's built-in timeout
  TIMEOUT_CMD=""
fi

if [ -n "$TIMEOUT_CMD" ]; then
  $TIMEOUT_CMD go test -v -race -timeout=5m ...
else
  go test -v -race -timeout=5m ...
fi
```

The solution:
1. Checks for `gtimeout` (GNU timeout, available via Homebrew on macOS)
2. Falls back to `timeout` (available on Linux and some systems)
3. Uses Go's built-in `-timeout` flag as fallback

### 2. Mock Script Execution Issues

**Problem**: Tests were creating Windows `.bat` files but trying to run them on Unix systems, causing "exec format error".

**Solution**: Updated mock script creation to be truly cross-platform.

**Files Modified**:
- `testdata/helpers/test_helpers.go`
- `internal/porch/executor_test.go`
- `internal/porch/executor_security_test.go`

**Key Changes**:

1. **Cross-Platform Mock Script Creation**:
   ```go
   func (tf *TestFixtures) CreateMockPorch(t testing.TB, exitCode int, stdout, stderr string, sleepDuration time.Duration) string {
       if runtime.GOOS == "windows" {
           ext = ".bat"
           // Windows batch script with PowerShell for precise timing
           sleepCmd = fmt.Sprintf("powershell -command \"Start-Sleep -Milliseconds %d\"", int(sleep.Milliseconds()))
       } else {
           ext = ".sh"
           // Unix shell script
           sleepCmd = fmt.Sprintf("sleep %v", sleep.Seconds())
       }
   }
   ```

2. **Fixed Mock Executable Path Resolution**:
   ```go
   func (tf *TestFixtures) GetMockExecutable(name string) string {
       execPath := filepath.Join(tf.MockExecutablesDir, name)
       if filepath.Ext(name) == "" {
           if runtime.GOOS == "windows" {
               execPath += ".bat"
           } else {
               execPath += ".sh"
           }
       }
       return execPath
   }
   ```

3. **Created Shell Script Versions of Mock Executables**:
   - `testdata/mock-executables/mock-porch-success.sh`
   - `testdata/mock-executables/mock-porch-failure.sh`
   - `testdata/mock-executables/mock-porch-timeout.sh`

### 3. Enhanced Test Helper Functions

**Added**:
- `CreateCrossPlatformMockScript()` - Creates platform-appropriate mock scripts
- `createWindowsMockScript()` - Windows-specific batch file creation
- `createUnixMockScript()` - Unix-specific shell script creation

## Platform-Specific Considerations

### Windows
- Uses `.bat` extension for scripts
- Uses `powershell -command "Start-Sleep -Milliseconds X"` for precise timing
- Handles stdout/stderr redirection with proper batch syntax

### Unix/Linux/macOS
- Uses `.sh` extension for scripts
- Uses `sleep X.Y` for timing
- Uses standard bash script syntax
- Made scripts executable with `chmod +x`

## Testing

All cross-platform fixes have been tested and verified:

1. **Timeout Tests**: Mock scripts with precise timing now work correctly on both platforms
2. **Mock Script Execution**: No more "exec format error" - proper scripts are created for each platform
3. **CI Compatibility**: Workflows now handle different timeout command availability

## Benefits

1. **CI Stability**: Tests now run successfully on Linux, macOS, and Windows runners
2. **Developer Experience**: Local testing works consistently across platforms
3. **Maintainability**: Clear separation of platform-specific code with fallbacks
4. **Performance**: More precise timing in mock scripts using PowerShell on Windows

## Future Considerations

- Consider using Go-based timeout mechanisms instead of shell commands for even better cross-platform compatibility
- Monitor CI runs to ensure consistent behavior across all supported platforms
- Add integration tests specifically for cross-platform scenarios

## Files Added/Modified

### Modified Files:
- `.github/workflows/conductor-loop.yml`
- `.github/workflows/conductor-loop-cicd.yml` 
- `testdata/helpers/test_helpers.go`
- `internal/porch/executor_test.go`
- `internal/porch/executor_security_test.go`

### New Files:
- `testdata/mock-executables/mock-porch-success.sh`
- `testdata/mock-executables/mock-porch-failure.sh`
- `testdata/mock-executables/mock-porch-timeout.sh`
- `CROSS_PLATFORM_FIXES.md` (this document)

All changes are backward-compatible and maintain existing functionality while adding cross-platform support.