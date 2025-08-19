# feat(porch-direct): add intent-driven KRM package generator CLI

## What and Why

- **What**: Adds `porch-direct` CLI tool that reads intent JSON files and generates KRM packages for Porch API, supporting both dry-run mode (writes to `./out/`) and live mode (calls Porch API)
- **Why**: Bridges the gap between high-level scaling intents and low-level Porch package management, enabling automated KRM generation from declarative intents
- **Scope**: New CLI tool with minimal impact - no changes to `docs/contracts/*`, no CI modifications, no public API changes to existing code

## Tests Added

### Unit Tests
```powershell
# Run all tests (15 test cases)
go test ./cmd/porch-direct -v
go test ./pkg/porch -v

# Key test scenarios:
# - Valid intent parsing and package generation
# - Invalid intent validation (negative replicas, missing fields)
# - Idempotency verification
# - Dry-run mode output validation
```

### Manual Testing
```powershell
# Build
go build .\cmd\porch-direct

# Test dry-run mode
.\porch-direct.exe --intent .\examples\intent.json --repo my-repo --package nf-sim \
  --workspace ws-001 --namespace ran-a --porch http://localhost:9443 --dry-run

# Verify outputs
Get-ChildItem -Path .\out -Recurse
# Expected: porch-package-request.json, overlays/deployment.yaml, overlays/configmap.yaml
```

## Risk Analysis

### Risks
- **Low Risk**: New isolated CLI tool with no modifications to existing functionality
- **No Breaking Changes**: Additive feature only; no changes to contracts, CI, or public APIs
- **Dependencies**: Uses only standard library + existing internal packages

### Rollback Plan
```bash
# Simple removal if issues arise (no dependencies on this tool)
git revert <commit-hash>
rm -rf cmd/porch-direct pkg/porch
```

### Mitigation
- Dry-run mode provides safe testing without Porch server interaction
- Comprehensive validation prevents malformed intent processing
- Clear separation from existing code paths

## Files Changed

### New Files (No Impact on Existing Code)
- `cmd/porch-direct/main.go` - CLI entry point
- `cmd/porch-direct/main_test.go` - Unit tests
- `pkg/porch/client.go` - Porch API client
- `pkg/porch/types.go` - Type definitions
- `examples/intent.json` - Sample intent file
- `BUILD-RUN-TEST.windows.md` - Windows documentation

### Minor Additions (Backward Compatible)
- `internal/generator/package.go` - Added `GetFiles()` method (new, non-breaking)
- `internal/intent/types.go` - Added `NetworkIntent` type alias (backward compatible)

### No Changes To
- ✅ `docs/contracts/*` - All contracts unchanged
- ✅ `.github/workflows/ci.yml` - No CI modifications
- ✅ Public APIs - No renamed identifiers or breaking changes

## Verification Checklist

- [x] Builds successfully: `go build ./cmd/porch-direct`
- [x] Tests pass: `go test ./...`
- [x] Dry-run generates expected outputs in `./out/`
- [x] No modifications to contracts or CI
- [x] No public API breaking changes
- [x] Documentation included

---

## Suggested Squash Commit Message

```
feat(porch-direct): add intent-driven KRM package generator CLI

Implement porch-direct CLI tool for generating KRM packages from intent JSON files.
Supports dry-run mode (writes to ./out/) and live Porch API integration.

- Add CLI with flags: --intent, --repo, --package, --workspace, --namespace, --porch, --dry-run
- Implement Porch client with package creation/update capabilities
- Add comprehensive test suite (15 test cases)
- Include Windows-specific documentation and examples
- No changes to contracts, CI, or existing public APIs

Tested: Unit tests pass, dry-run mode verified
Risk: Low - isolated new feature with no dependencies