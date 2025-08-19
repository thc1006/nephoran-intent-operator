# Conductor-Watch Testing Scripts

This directory contains Windows PowerShell scripts for testing the conductor-watch functionality.

## Scripts

### `watcher-smoke.ps1`

Comprehensive smoke test that demonstrates the complete conductor-watch workflow with Kubernetes integration.

**Prerequisites:**
- kubectl configured and connected to a Kubernetes cluster
- NetworkIntent CRD installed (`kubectl apply -f deployments/crds/`)
- Go 1.24+ for building the binary

**Usage:**
```powershell
# Full smoke test with 30s timeout
.\hack\watcher-smoke.ps1

# Quick test with custom settings
.\hack\watcher-smoke.ps1 -Timeout 15 -Namespace test-ns -SkipBuild

# Run without building (if binary exists)
.\hack\watcher-smoke.ps1 -SkipBuild
```

**What it tests:**
1. **Pre-flight checks**: CRD existence, namespace, kubectl connectivity
2. **Build process**: Compiles conductor-watch binary
3. **File monitoring**: Starts conductor-watch with proper configuration
4. **NetworkIntent processing**: Creates and applies test NetworkIntent resource
5. **Handoff verification**: Confirms intent files are created and processed
6. **Log analysis**: Verifies expected conductor-watch behaviors

### `watcher-basic-test.ps1`

Basic functionality test that doesn't require Kubernetes - only tests file monitoring and processing.

**Prerequisites:**
- Go 1.24+ for building the binary
- PowerShell 5.1+

**Usage:**
```powershell
# Basic test (no Kubernetes required)
.\hack\watcher-basic-test.ps1

# Quick test with shorter timeout
.\hack\watcher-basic-test.ps1 -Timeout 5
```

**What it tests:**
1. **Binary build**: Compiles conductor-watch successfully
2. **File monitoring**: Detects new intent JSON files
3. **Schema validation**: Validates against intent schema
4. **Processing pipeline**: Generates scaling patches
5. **Log patterns**: Verifies expected log messages

## Makefile Targets

The following Make targets are available for conductor-watch operations:

```bash
# Build conductor-watch binary
make conductor-watch-build

# Run conductor-watch locally
make conductor-watch-run

# Run conductor-watch in dry-run mode
make conductor-watch-run-dry

# Run comprehensive smoke test
make conductor-watch-smoke

# Run quick smoke test (15s timeout)
make conductor-watch-smoke-quick

# Run basic test (no Kubernetes)
make conductor-watch-test-basic

# Clean up artifacts
make conductor-watch-clean
```

## Expected Outputs

### Successful Basic Test
```
✅ Conductor-watch built successfully: .\conductor-loop.exe
✅ Conductor-watch started (PID: 12345)
✅ Created test file: .\handoff\intent-20250816123456-basic-test-1.json
✅ Found porch output patterns - files are being processed!
   ✓ wrote: output
   ✓ scaling-patch.yaml
✅ Found 4/4 expected log patterns
   ✓ Starting conductor-loop
   ✓ Watching directory
   ✓ Intent file detected
   ✓ Successfully processed
✅ All expected log patterns found - conductor-watch is working correctly!
```

### Successful Smoke Test
```
✅ NetworkIntent CRD exists
✅ Namespace 'ran-a' exists
✅ Conductor-watch built successfully: .\conductor-loop.exe
✅ Conductor-watch started (PID: 12345)
✅ Test NetworkIntent applied successfully
✅ Handoff file created: intent-20250816123456-smoke-test.json
✅ Conductor-watch is still running
```

## Troubleshooting

### Common Issues

1. **"NetworkIntent CRD not found"**
   ```bash
   kubectl apply -f deployments/crds/networkintent_crd.yaml
   ```

2. **"Cannot connect to Kubernetes cluster"**
   ```bash
   kubectl cluster-info
   kubectl config current-context
   ```

3. **"Binary not found after build"**
   - Check Go installation: `go version`
   - Verify in correct directory
   - Check file permissions

4. **"Validation failed: invalid json"**
   - Check intent JSON format matches `docs/contracts/intent.schema.json`
   - Verify no trailing whitespace or BOM issues
   - Check file encoding (should be UTF-8)

5. **"No handoff file created"**
   - Verify NetworkIntent controller is running
   - Check NetworkIntent status: `kubectl describe networkintent test-name`
   - Check controller logs for errors

### Debug Commands

```powershell
# Check conductor-watch version and help
.\conductor-loop.exe --help

# Manual test with verbose logging
.\conductor-loop.exe -handoff-dir .\handoff -schema .\docs\contracts\intent.schema.json -batch-interval 1s

# Check handoff directory contents
Get-ChildItem .\handoff -Filter "intent-*.json"

# Check for error files
Get-ChildItem .\handoff\errors

# Validate intent JSON manually
Get-Content .\handoff\intent-test.json | ConvertFrom-Json
```

## Log Patterns

### Expected Log Messages
```
[conductor-loop] Starting conductor-loop:
[conductor-loop]   Watching: C:\...\handoff
[conductor-loop]   Errors: C:\...\handoff\errors
[conductor-loop]   Mode: structured
[conductor-loop]   Batch: size=1, interval=2s
[conductor-loop] Watching directory: C:\...\handoff
[conductor-loop] [CREATE] Intent file detected: intent-*.json
[conductor-loop] Processing batch of 1 files
[conductor-loop] Successfully processed: intent-*.json
```

### Expected Output Messages
```
wrote: output\scaling-patch.yaml
next: (optional) kpt live init/apply under ./output
```

## Files Created

The smoke tests create the following files:

- `conductor-loop.exe` - Built binary
- `handoff/intent-*-smoke-test*.json` - Test NetworkIntent files  
- `handoff/intent-*-basic-test*.json` - Basic test files
- `output/scaling-patch.yaml` - Generated scaling patches
- `handoff/.processed` - Processing state tracking
- `handoff/errors/*.error` - Error files (if validation fails)

All test files are automatically cleaned up by the scripts.

## Integration with CI/CD

These scripts can be integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions step
- name: Run Conductor-Watch Smoke Test
  run: |
    pwsh -ExecutionPolicy Bypass -File hack/watcher-smoke.ps1 -Timeout 30
  shell: pwsh
```

For environments without Kubernetes:
```yaml
- name: Run Basic Conductor-Watch Test  
  run: |
    pwsh -ExecutionPolicy Bypass -File hack/watcher-basic-test.ps1 -Timeout 10
  shell: pwsh
```