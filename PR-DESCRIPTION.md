# PR: feat(porch): add structured KRM patch generator for scaling intents

**Base:** `integrate/mvp`  
**Head:** `feat/porch-structured-patch`

## What and Why

- **Implements structured KRM patch generator** that transforms scaling intent JSON into kpt-compatible packages, enabling declarative replica management through Porch/Nephio workflows
- **Provides Windows-safe CLI tool** (`cmd/porch-structured-patch`) with validated intent parsing and automatic package structure generation including Kptfile, deployment patches, and setters
- **Bridges intent-based orchestration with KRM toolchain** by producing standard kpt packages that integrate seamlessly with existing GitOps pipelines and Porch package management

## Tests Added

### Unit Tests
```bash
# Run unit tests for intent validation and loading
go test ./internal/patch/... -v

# Expected output:
# === RUN   TestLoadIntent
# --- PASS: TestLoadIntent (0.03s)
# === RUN   TestLoadIntentValidation/invalid_intent_type
# === RUN   TestLoadIntentValidation/missing_target
# === RUN   TestLoadIntentValidation/missing_namespace
# === RUN   TestLoadIntentValidation/negative_replicas
# --- PASS: TestLoadIntentValidation (0.12s)
```

### Integration Test
```bash
# Build the tool
go build ./cmd/porch-structured-patch

# Run with example intent
./porch-structured-patch.exe --intent ./examples/intent.json --out ./examples/packages/scaling

# Verify generated package structure
ls -la ./examples/packages/scaling/example-deployment-scaling-*/
# Should contain: Kptfile, deployment-patch.yaml, setters.yaml, README.md
```

### Manual Validation
```bash
# Test error handling with invalid intent
echo '{"intent_type":"invalid","target":"test","namespace":"ns","replicas":1}' > bad.json
./porch-structured-patch.exe --intent bad.json --out ./temp
# Expected: Error: failed to load intent: unsupported intent_type: invalid (expected 'scaling')

# Test --apply flag (dry run if porch-direct unavailable)
./porch-structured-patch.exe --intent ./examples/intent.json --out ./temp --apply
# Expected: Warning about porch-direct or successful execution
```

## Risk Analysis

### Risks
- **Low Risk - New isolated component**: No modifications to existing modules, contracts, or CI configuration
- **Low Risk - Pure Go implementation**: No external binary dependencies, Windows-safe path handling
- **Low Risk - Validation layer**: Intent JSON validated before processing, prevents malformed patches

### Rollback Plan
1. **Immediate rollback**: Delete `cmd/porch-structured-patch` and `internal/patch` directories
2. **No data migration required**: Generated packages are ephemeral artifacts
3. **No service disruption**: Component is CLI-only, not part of runtime services
4. **Clean revert**: `git revert <commit-hash>` removes all changes cleanly

## Scope Confirmation

✅ **No changes to `docs/contracts/*`** - All contract schemas remain unchanged  
✅ **No changes to `.github/workflows/ci.yml`** - CI pipeline unmodified  
✅ **No public identifiers renamed** - New code only, no breaking changes  
✅ **Module path preserved** - Uses existing `github.com/thc1006/nephoran-intent-operator`

## Files Changed

```
+ cmd/porch-structured-patch/main.go                 (68 lines)
+ cmd/porch-structured-patch/BUILD-RUN-TEST.windows.md (185 lines)
+ internal/patch/generator.go                        (174 lines)
+ internal/patch/intent.go                           (44 lines)
+ internal/patch/intent_test.go                      (96 lines)
+ examples/intent.json                               (8 lines)
+ examples/packages/scaling/<generated-example>/     (4 files)
```

## Conventional Commit Title
```
feat(porch): add structured KRM patch generator for scaling intents
```

## Suggested Squash Commit Message
```
feat(porch): add structured KRM patch generator for scaling intents

Implements cmd/porch-structured-patch CLI tool that transforms scaling
intent JSON into kpt-compatible KRM packages. Generates Kptfile,
deployment patches, and setter configurations for declarative replica
management through Porch/Nephio workflows.

Key features:
- Validates intent JSON (type, target, namespace, replicas)
- Generates complete kpt package structure with metadata
- Windows-safe path handling for cross-platform compatibility
- Optional --apply flag for porch-direct integration
- Includes auto-generated README in each package

No changes to existing contracts, CI configuration, or public APIs.
Pure additive change with isolated module scope.

Tested on Windows with PowerShell. Unit tests cover intent validation
and error cases. Integration verified with example intent.json.
```

## Checklist

- [x] Code compiles without warnings (`go build ./cmd/porch-structured-patch`)
- [x] Unit tests pass (`go test ./internal/patch/...`)
- [x] Integration test successful (generated valid KRM packages)
- [x] Documentation included (`BUILD-RUN-TEST.windows.md`)
- [x] No breaking changes to existing code
- [x] No modifications to contracts or CI
- [x] Windows-compatible implementation verified
- [x] Follows conventional commit format