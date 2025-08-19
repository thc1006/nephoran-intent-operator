# PR: feat/porch-structured-patch → integrate/mvp

## Title
feat(porch): add structured KRM patch generator

## Summary
Implements structured KRM patch generator that creates kpt-compatible packages from intent JSON.

## Changes
- ✅ New `cmd/porch-structured-patch` with CLI flags: `--intent`, `--out`, `--apply`
- ✅ Generates complete kpt packages with Kptfile, deployment patch, and setters
- ✅ Validates intent JSON (scaling type, target, namespace, replicas)
- ✅ Optional `--apply` flag for porch-direct integration
- ✅ Windows-safe path handling
- ✅ Auto-generated README in each package

## Test Results
```bash
# Build
go build ./cmd/porch-structured-patch

# Generate patch
./porch-structured-patch.exe --intent ./examples/intent.json --out ./examples/packages/scaling

# Output
=== Patch Package Generated ===
Package: example-deployment-scaling-1755372079
Target: example-deployment
Namespace: default
Replicas: 3
Location: examples\packages\scaling\example-deployment-scaling-1755372079
================================
```

## Generated Package Structure
```
example-deployment-scaling-1755372079/
├── Kptfile                 # Package metadata
├── deployment-patch.yaml   # Replica patch
├── setters.yaml           # Configuration values
└── README.md              # Usage instructions
```

## Files Changed
- `cmd/porch-structured-patch/main.go` - Main CLI entry point
- `cmd/porch-structured-patch/BUILD-RUN-TEST.windows.md` - Build and test documentation
- `internal/patch/intent.go` - Intent parsing and validation
- `internal/patch/generator.go` - KRM package generation
- `internal/patch/intent_test.go` - Unit tests
- `examples/intent.json` - Example intent file
- `examples/packages/scaling/` - Generated example packages

## Testing
```bash
# Unit tests pass
go test ./internal/patch/... -v
=== RUN   TestLoadIntent
--- PASS: TestLoadIntent (0.03s)
=== RUN   TestLoadIntentValidation
--- PASS: TestLoadIntentValidation (0.12s)
PASS
```

## Definition of Done
✅ Reliably generates structured KRM patches
✅ Optional `--apply` calls porch-direct
✅ Unit tests passing
✅ Windows-compatible
✅ Documentation included

## Branch
- Source: `feat/porch-structured-patch`
- Target: `integrate/mvp`
- Commit: `d5080b39`