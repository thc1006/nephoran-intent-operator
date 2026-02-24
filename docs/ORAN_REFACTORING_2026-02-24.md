# O-RAN Architecture Refactoring - 2026-02-24

## Overview

This document describes the comprehensive O-RAN architecture refactoring completed on 2026-02-24 to eliminate hardcoded URLs and standardize O-RAN interfaces according to the refactoring plan defined in `.claude/plans/shimmering-shimmying-rainbow.md`.

## Changes Implemented

### Phase 1: Core Architecture Fixes ✅ COMPLETED

#### 1.1 Configurable A1 API Path Format

**Problem**: The controller was hardcoded to use O-RAN SC RICPLT-specific API paths (`/A1-P/v2/policytypes/{typeId}/policies/{policyId}`), which are not compatible with O-RAN Alliance standard implementations.

**Solution**: Added configurable A1 API format support with two modes:

- **Standard Mode** (`A1_API_FORMAT=standard`): Uses O-RAN Alliance A1AP-v03.01 standard paths
  - Policy Creation: `PUT /v2/policies/{policyId}`
  - Policy Deletion: `DELETE /v2/policies/{policyId}`

- **Legacy Mode** (`A1_API_FORMAT=legacy`, **default**): Uses O-RAN SC RICPLT paths for backward compatibility
  - Policy Creation: `PUT /A1-P/v2/policytypes/{typeId}/policies/{policyId}`
  - Policy Deletion: `DELETE /A1-P/v2/policytypes/{typeId}/policies/{policyId}`

**Files Modified**:
- `controllers/networkintent_controller.go`:
  - Added `A1APIFormat` type and constants (`A1FormatStandard`, `A1FormatLegacy`)
  - Added `A1APIFormat` field to `NetworkIntentReconciler` struct
  - Updated `createA1Policy()` to support both formats via switch statement
  - Updated `deleteA1Policy()` to support both formats via switch statement
  - Updated `SetupWithManager()` to read `A1_API_FORMAT` environment variable

- `cmd/main.go`:
  - Added `--a1-api-format` flag with description
  - Added flag validation and propagation to reconciler
  - Added `A1_API_FORMAT` environment variable propagation

**Configuration Examples**:

```bash
# Using environment variable (legacy mode, default)
export A1_API_FORMAT=legacy
export A1_MEDIATOR_URL=http://service-ricplt-a1mediator-http.ricplt:10000

# Using environment variable (standard mode)
export A1_API_FORMAT=standard
export A1_MEDIATOR_URL=http://a1-policy-service:8080

# Using command-line flags
./manager \
  --a1-endpoint=http://service-ricplt-a1mediator-http.ricplt:10000 \
  --a1-api-format=legacy

# For O-RAN Alliance compliant implementations
./manager \
  --a1-endpoint=http://a1-policy-service:8080 \
  --a1-api-format=standard
```

#### 1.2 Finalizer Graceful Degradation ✅ ALREADY IMPLEMENTED

**Status**: This was already implemented in previous commits. The current code includes:

- `maxA1CleanupRetries = 3` constant
- `A1CleanupRetriesAnnotation` tracking retry count
- Automatic finalizer removal after max retries to prevent indefinite blocking
- Proper error logging and retry counter increment

**No changes needed** - existing implementation already meets requirements.

#### 1.3 ObservedEndpoints Status Recording ✅ COMPLETED

**Problem**: Operators had no visibility into which actual endpoints (A1, LLM, Porch) were contacted during reconciliation, making debugging difficult.

**Solution**: Added automatic recording of successfully contacted endpoints in `NetworkIntent.Status.ObservedEndpoints`.

**Files Modified**:
- `controllers/networkintent_controller.go`:
  - Added `metav1` import for timestamp types
  - Added `recordObservedEndpoint()` helper function
  - Updated `createA1Policy()` to record A1 endpoint on success
  - Updated LLM processing logic to record LLM endpoint on success

- `api/intent/v1alpha1/networkintent_types.go`:
  - **Already defined**: `ObservedEndpoint` struct with `Name`, `URL`, `LastContactedAt` fields
  - **Already defined**: `ObservedEndpoints []ObservedEndpoint` in `NetworkIntentStatus`

**Example Status Output**:

```yaml
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: test-scaling
status:
  phase: Deployed
  message: Intent deployed successfully via A1 policy
  observedEndpoints:
  - name: a1
    url: http://service-ricplt-a1mediator-http.ricplt:10000
    lastContactedAt: "2026-02-24T10:30:45Z"
  - name: llm
    url: http://ollama-service:11434/process
    lastContactedAt: "2026-02-24T10:30:42Z"
```

### Phase 2: O-RAN Interface Updates ⏳ PENDING

#### 2.1 Standardize O2 IMS API Paths

**Status**: NOT YET IMPLEMENTED (Task #72)

**Target Files**: `pkg/oran/o2/`

**Required Changes**:
- Update API server routes to use `/o2ims_infrastructureInventory/v1/*` paths
- Update client code to use standard O2 IMS paths
- Maintain backward compatibility with legacy paths if needed

#### 2.2 Update Nephio Porch Integration

**Status**: PARTIALLY DONE (Porch endpoint already configurable via `PORCH_SERVER_URL`)

**Current State**:
- Porch endpoint is already configurable via `--porch-server` flag and `PORCH_SERVER_URL` environment variable
- No hardcoded `porch-server:8080` in main operator code

**Remaining Work** (if any):
- Verify `pkg/nephio/` uses correct Nephio R6 API groups
- Add Porch endpoint to ObservedEndpoints when contacted

### Phase 3: Cleanup ⏳ PENDING

#### 3.1 Remove Duplicate Type Definitions

**Status**: NOT YET IMPLEMENTED

**Target Files**: Multiple files with duplicate `A1PolicyType`, `A1PolicyInstance` definitions

#### 3.2 Remove Stub Implementations

**Status**: NOT YET IMPLEMENTED

**Candidates**: Functions with no actual logic, just placeholder returns

#### 3.3 Remove Outdated Example YAMLs

**Status**: NOT YET IMPLEMENTED

**Target**: Example YAMLs using old hardcoded paths

## Testing

### Unit Tests

```bash
# Test controller builds correctly
go build ./controllers/...
go build ./cmd/main.go

# Run controller tests
go test ./controllers/... -v

# Run full test suite
go test ./... -v
```

### Integration Tests

```bash
# Test with legacy format (default)
kubectl apply -f examples/networkintent-scaling.yaml
kubectl get networkintent test-scaling -o yaml | grep -A 10 observedEndpoints

# Test with standard format
export A1_API_FORMAT=standard
export A1_MEDIATOR_URL=http://a1-standard-service:8080
kubectl apply -f examples/networkintent-scaling.yaml
```

### Validation Checklist

- [x] Controller compiles without errors
- [x] A1 API format is configurable via environment variable
- [x] A1 API format is configurable via command-line flag
- [x] Default format is `legacy` for backward compatibility
- [x] ObservedEndpoints are recorded on successful A1 contact
- [x] ObservedEndpoints are recorded on successful LLM contact
- [x] Finalizer graceful degradation already implemented
- [ ] O2 IMS API paths standardized (Task #72)
- [ ] Porch endpoint added to ObservedEndpoints
- [ ] All tests passing
- [ ] Documentation updated

## Migration Guide

### For Existing Deployments (O-RAN SC RICPLT)

**No action required** - the default `A1_API_FORMAT=legacy` maintains full backward compatibility with existing O-RAN SC A1 Mediator deployments.

### For New Deployments (O-RAN Alliance Standard)

1. Set environment variable in deployment:
   ```yaml
   env:
   - name: A1_API_FORMAT
     value: "standard"
   - name: A1_MEDIATOR_URL
     value: "http://a1-policy-service:8080"
   ```

2. Or use command-line flags:
   ```yaml
   args:
   - --a1-endpoint=http://a1-policy-service:8080
   - --a1-api-format=standard
   ```

### Verification

After deployment, check NetworkIntent status:

```bash
kubectl get networkintent -n default test-intent -o jsonpath='{.status.observedEndpoints}' | jq
```

Expected output:
```json
[
  {
    "name": "a1",
    "url": "http://service-ricplt-a1mediator-http.ricplt:10000",
    "lastContactedAt": "2026-02-24T10:30:45Z"
  }
]
```

## Architecture Diagrams

### Before Refactoring

```
NetworkIntent Controller
    |
    +-- Hardcoded: http://service-ricplt-a1mediator-http.ricplt:10000/A1-P/v2/policytypes/100/policies/{id}
    |
    +-- Hardcoded: http://llm-processor:8080/process
    |
    +-- Configurable: Porch endpoint (PORCH_SERVER_URL)
```

### After Refactoring (Phase 1 Complete)

```
NetworkIntent Controller
    |
    +-- Configurable A1 Endpoint (A1_MEDIATOR_URL or --a1-endpoint)
    |     |
    |     +-- Format: Standard (A1_API_FORMAT=standard)
    |     |     → /v2/policies/{policyId}
    |     |
    |     +-- Format: Legacy (A1_API_FORMAT=legacy, default)
    |           → /A1-P/v2/policytypes/{typeId}/policies/{policyId}
    |
    +-- Configurable LLM Endpoint (LLM_PROCESSOR_URL or --llm-endpoint)
    |
    +-- Configurable Porch Endpoint (PORCH_SERVER_URL or --porch-server)
    |
    +-- Status Recording: ObservedEndpoints
          → Records actual endpoints contacted with timestamps
```

## Related Documentation

- [Architecture Refactoring Plan](../.claude/plans/shimmering-shimmying-rainbow.md)
- [A1 Integration Verification](A1_INTEGRATION_VERIFICATION.md)
- [CLAUDE.md - Project Instructions](../CLAUDE.md)
- [NetworkIntent CRD](../api/intent/v1alpha1/networkintent_types.go)

## Implementation Tasks

| Task | Status | Files Modified | Notes |
|------|--------|---------------|-------|
| Add A1APIFormat type and constants | ✅ Complete | `controllers/networkintent_controller.go` | Standard and Legacy modes |
| Update createA1Policy for both formats | ✅ Complete | `controllers/networkintent_controller.go` | Switch statement based on format |
| Update deleteA1Policy for both formats | ✅ Complete | `controllers/networkintent_controller.go` | Switch statement based on format |
| Add A1_API_FORMAT env var reading | ✅ Complete | `controllers/networkintent_controller.go` | In SetupWithManager() |
| Add --a1-api-format flag | ✅ Complete | `cmd/main.go` | With validation |
| Add recordObservedEndpoint helper | ✅ Complete | `controllers/networkintent_controller.go` | Upsert logic |
| Record A1 endpoint on success | ✅ Complete | `controllers/networkintent_controller.go` | In createA1Policy |
| Record LLM endpoint on success | ✅ Complete | `controllers/networkintent_controller.go` | In LLM processing |
| Add metav1 import for timestamps | ✅ Complete | `controllers/networkintent_controller.go` | Required for metav1.Now() |
| Standardize O2 IMS API paths | ⏳ Pending | `pkg/oran/o2/` | Task #72 |
| Add Porch to ObservedEndpoints | ⏳ Pending | Multiple files | When Porch is contacted |
| Remove duplicate types | ⏳ Pending | Multiple files | Phase 3 cleanup |
| Remove stub implementations | ⏳ Pending | Multiple files | Phase 3 cleanup |
| Update example YAMLs | ⏳ Pending | `examples/` | Phase 3 cleanup |

## Performance Impact

**Minimal to None**:
- A1 API path construction is a simple string format operation
- ObservedEndpoints recording is O(n) where n = number of endpoints (typically 2-3)
- No additional network calls introduced
- No changes to critical reconciliation loop logic

## Security Considerations

**Enhanced Security**:
- All endpoints still validated via existing SSRF validator
- Environment variable validation prevents invalid formats
- ObservedEndpoints provide audit trail of external communications
- No new attack surface introduced

## Backward Compatibility

**100% Backward Compatible**:
- Default `A1_API_FORMAT=legacy` maintains exact same behavior as before
- All existing deployments continue working without any changes
- Existing environment variables (`A1_MEDIATOR_URL`, etc.) still work
- New features are opt-in via explicit configuration

## Future Work

1. **Phase 2**: Standardize O2 IMS API paths (Task #72)
2. **Phase 3**: Code cleanup - remove duplicates and stubs
3. **Observability**: Add Prometheus metrics for endpoint contact frequency
4. **Validation**: Add CRD validation webhook to ensure valid A1_API_FORMAT values
5. **Documentation**: Add OpenAPI spec for both A1 standard and legacy formats

## References

- [O-RAN Alliance A1 Interface Specification (A1AP-v03.01)](https://specifications.o-ran.org/)
- [O-RAN SC A1 Mediator Documentation](https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-a1/en/latest/)
- [Kubernetes API Conventions - Status Subresource](https://kubernetes.io/docs/concepts/overview/kubernetes-api/)

---

**Last Updated**: 2026-02-24
**Author**: Claude Code AI Agent (Sonnet 4.5)
**Related Tasks**: #64, #70, #71, #72
