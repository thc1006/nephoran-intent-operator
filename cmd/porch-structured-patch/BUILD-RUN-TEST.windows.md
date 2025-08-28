# BUILD-RUN-TEST for porch-structured-patch (Windows)

## Prerequisites
- Go 1.24+
- PowerShell
- Working directory: Repository root

## Build Steps

```powershell
# Build the command
go build -o porch-structured-patch.exe .\cmd\porch-structured-patch
```

## Run Steps

### 1. Basic Run (generate patches)

```powershell
# Create output directory
New-Item -ItemType Directory -Force -Path .\examples\packages\scaling | Out-Null

# Run the patch generator
.\porch-structured-patch.exe --intent .\examples\intent.json --out .\examples\packages\scaling

# Verify generated files
Get-ChildItem .\examples\packages\scaling\ -Recurse
```

### 2. Run with custom output directory

```powershell
# Create custom output directory
New-Item -ItemType Directory -Force -Path .\temp\patches | Out-Null

# Run with custom output
.\porch-structured-patch.exe --intent .\examples\intent.json --out .\temp\patches

# Check files
Get-ChildItem .\temp\patches\ -Recurse
```

### 3. Run with apply flag (calls porch-direct if available)

```powershell
.\porch-structured-patch.exe --intent .\examples\intent.json --out .\examples\packages\scaling --apply
```

## Expected Output

### Console Output
```
=== Patch Package Generated ===
Package: example-deployment-scaling-<timestamp>
Target: example-deployment
Namespace: default
Replicas: 3
Location: .\examples\packages\scaling\example-deployment-scaling-<timestamp>
================================
```

### Generated Files Structure
```
examples\packages\scaling\
└── example-deployment-scaling-<timestamp>\
    ├── Kptfile                  # KRM package metadata
    ├── deployment-patch.yaml    # Deployment patch with replicas
    └── setters.yaml            # ConfigMap with setter values
```

### File Contents

**Kptfile:**
```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example-deployment-scaling-<timestamp>
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: Scaling patch for example-deployment
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/apply-setters:v0.2.0
    configMap:
      replicas: "3"
```

**deployment-patch.yaml:**
```yaml
# kpt-file: deployment-patch.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
  namespace: default
spec:
  replicas: 3
```

**setters.yaml:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: setters
  namespace: default
data:
  replicas: "3"
  target: example-deployment
  namespace: default
```

## Test Commands

### Test 1: Verify build
```powershell
go test .\internal\patch\...
```

### Test 2: Invalid intent
```powershell
# Create invalid intent
@'
{
  "intent_type": "invalid",
  "target": "test",
  "namespace": "default",
  "replicas": 3
}
'@ | Out-File -Encoding UTF8 .\test-invalid.json

# Should fail with error
.\porch-structured-patch.exe --intent .\test-invalid.json --out .\temp
# Expected: Error: failed to load intent: unsupported intent_type: invalid (expected 'scaling')
```

### Test 3: Missing required field
```powershell
# Create intent missing namespace
@'
{
  "intent_type": "scaling",
  "target": "test",
  "replicas": 3
}
'@ | Out-File -Encoding UTF8 .\test-missing.json

# Should fail with error
.\porch-structured-patch.exe --intent .\test-missing.json --out .\temp
# Expected: Error: failed to load intent: namespace is required
```

### Test 4: Verify generated patch structure
```powershell
# Generate patches
.\porch-structured-patch.exe --intent .\examples\intent.json --out .\test-output

# Check all expected files exist
$packageDir = Get-ChildItem .\test-output\ -Directory | Select-Object -First 1
Test-Path "$($packageDir.FullName)\Kptfile"
Test-Path "$($packageDir.FullName)\deployment-patch.yaml"
Test-Path "$($packageDir.FullName)\setters.yaml"

# All should return True
```

## Cleanup

```powershell
# Remove test files and directories
Remove-Item -Recurse -Force .\examples\packages\scaling\* -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force .\temp -ErrorAction SilentlyContinue
Remove-Item -Force .\test-*.json -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force .\test-output -ErrorAction SilentlyContinue
Remove-Item -Force .\porch-structured-patch.exe -ErrorAction SilentlyContinue
```

## Notes

- The `--apply` flag will attempt to call `porch-direct` if available
- If `porch-direct` is not available, it will print instructions for manual application
- Package names include timestamp to ensure uniqueness
- All paths are Windows-safe (using filepath.Join)