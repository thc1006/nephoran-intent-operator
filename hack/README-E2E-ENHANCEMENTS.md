# E2E Test Harness Enhancements

## Overview

The E2E test harness (`hack/run-e2e.sh`) has been significantly enhanced to provide robust, production-ready testing with comprehensive validation and error handling.

## Key Enhancements

### 1. Robust Wait/Verify Logic
- **Enhanced `wait_for_condition()`**: Improved timeout handling with diagnostic information
- **`wait_for_http_health()`**: HTTP endpoint health checks with configurable expected responses
- **`wait_for_process_health()`**: Process health verification with application-specific checks
- **Periodic status updates**: Progress feedback every 30 seconds for long-running operations

### 2. Component Health Checks
- **Intent-ingest verification**: Process health + HTTP endpoint validation
- **Conductor-loop verification**: Process health + handoff directory monitoring
- **Webhook manager validation**: Deployment readiness + configuration verification
- **CRD establishment checks**: Proper CustomResourceDefinition availability

### 3. KRM Patch Generation Verification
- **Handoff directory monitoring**: Automatic creation and permission validation
- **Artifact counting**: JSON/YAML file detection and validation
- **Content verification**: Scaling-specific artifact validation
- **JSON syntax validation**: Structure verification for generated patches

### 4. Enhanced Error Handling
- **Graceful cleanup**: Proper process termination on exit/failure
- **Diagnostic information**: Timeout causes and failure details
- **Resource cleanup**: Test intent removal and failed pod cleanup
- **Component tracking**: PID management for all local processes

### 5. Success Criteria Validation
- **5-point validation score**: Comprehensive pipeline health assessment
- **Final E2E pipeline validation**: Overall system health check
- **Clear PASS/FAIL reporting**: Detailed test result summaries
- **Troubleshooting guidance**: Helpful hints for debugging failures

### 6. Schema Validation Fixes
- **API group alignment**: Fixed webhook configuration to use `nephoran.com/v1`
- **Contract compliance**: Aligned samples with `docs/contracts/intent.schema.json`
- **Webhook path correction**: Updated validation endpoint path
- **Sample format standardization**: Consistent NetworkIntent format

### 7. Windows Compatibility
- **Git Bash detection**: Automatic environment detection and PATH enhancement
- **Binary detection**: `.exe` suffix handling for Windows binaries
- **Path normalization**: Consistent absolute path handling
- **Process management**: Windows-compatible process health checks

## Usage

### Basic E2E Test Run
```bash
./hack/run-e2e.sh
```

### Development Mode (Preserve Cluster)
```bash
SKIP_CLEANUP=true ./hack/run-e2e.sh
```

### Verbose Output
```bash
VERBOSE=true ./hack/run-e2e.sh
```

### Custom Configuration
```bash
CLUSTER_NAME=my-test \
NAMESPACE=test-ns \
TIMEOUT=600 \
./hack/run-e2e.sh
```

## Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| `CLUSTER_NAME` | `nephoran-e2e` | Kind cluster name |
| `NAMESPACE` | `nephoran-system` | Kubernetes namespace |
| `INTENT_INGEST_MODE` | `local` | `local` or `sidecar` |
| `CONDUCTOR_MODE` | `local` | `local` or `in-cluster` |
| `PORCH_MODE` | `structured-patch` | `structured-patch` or `direct` |
| `SKIP_CLEANUP` | `false` | Preserve cluster after test |
| `VERBOSE` | `false` | Enable detailed output |
| `TIMEOUT` | `300` | Default timeout in seconds |

## Test Workflow

1. **Tool Detection**: Verify required binaries (kind, kubectl, go, curl)
2. **Cluster Setup**: Create/verify kind cluster
3. **CRD Installation**: Apply and wait for CRD establishment
4. **Namespace Setup**: Create namespace with webhook labels
5. **Webhook Deployment**: Deploy and verify webhook manager
6. **Intent-ingest Component**: Build, start, and verify local service
7. **Conductor-loop Component**: Build, start, and verify watcher
8. **Webhook Validation**: Test valid/invalid NetworkIntent creation
9. **Intent Processing**: Verify JSON intent processing (if local mode)
10. **Sample Scenarios**: Test predefined sample intents
11. **KRM Patch Verification**: Check for generated artifacts
12. **Final Pipeline Validation**: 5-point health assessment
13. **Cleanup**: Stop processes and optionally remove cluster

## Validation Points

The enhanced harness validates:

1. ✅ **Admission webhook operational**: ValidatingWebhookConfiguration exists
2. ✅ **Intent-ingest health**: Process running and HTTP endpoint responding
3. ✅ **Conductor-loop health**: Process running and watching handoff directory
4. ✅ **NetworkIntent creation**: CRD accepts valid intents, rejects invalid ones
5. ✅ **KRM artifact generation**: Files created in handoff directory

**Scoring**: 4-5/5 = Success, 2-3/5 = Partial Success, 0-1/5 = Failure

## Troubleshooting

### Common Issues

**Webhook not found**
```bash
kubectl get validatingwebhookconfigurations
kubectl describe validatingwebhookconfigurations nephoran
```

**Intent-ingest startup issues**
```bash
VERBOSE=true SKIP_CLEANUP=true ./hack/run-e2e.sh
# Check /tmp/intent-ingest binary and PROJECT_ROOT paths
```

**No KRM patches generated**
```bash
# Check handoff directory
ls -la handoff/
# Verify conductor-loop is watching
ps aux | grep conductor-loop
```

**Schema validation failures**
```bash
# Verify contract schema
cat docs/contracts/intent.schema.json
# Check webhook configuration
kubectl get validatingwebhookconfigurations -o yaml
```

### Debug Mode
```bash
SKIP_CLEANUP=true VERBOSE=true ./hack/run-e2e.sh
# Cluster preserved for manual inspection
kubectl get all -n nephoran-system
kubectl logs -n nephoran-system deployment/webhook-manager
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kind Cluster  │    │  Intent-ingest  │    │ Conductor-loop  │
│                 │    │   (local HTTP)  │    │   (file watch)  │
│ ┌─────────────┐ │    │                 │    │                 │
│ │   Webhook   │◀┼────┤   Port: 8080    │    │                 │
│ │   Manager   │ │    │   /healthz      │    │                 │
│ │             │ │    │   /intent       │    │                 │
│ └─────────────┘ │    └─────────────────┘    └─────────────────┘
│                 │             │                       │
│ ┌─────────────┐ │             ▼                       ▼
│ │ NetworkIntent│ │    ┌─────────────────┐    ┌─────────────────┐
│ │    CRDs     │ │    │   JSON Schema   │    │  Handoff Dir    │
│ │             │ │    │   Validation    │    │  *.json/*.yaml  │
│ └─────────────┘ │    └─────────────────┘    └─────────────────┘
└─────────────────┘
```

## Files Modified/Created

### Enhanced Files
- `hack/run-e2e.sh` - Main test harness with all enhancements
- `config/webhook/manifests.yaml` - Fixed API group alignment

### New Files
- `hack/test-e2e-enhancements.sh` - Validation script for enhancements
- `hack/README-E2E-ENHANCEMENTS.md` - This documentation

### Contract Alignment
- Webhook configuration aligned with `nephoran.com/v1` API group
- Samples updated to use correct NetworkIntent schema
- Intent processing aligned with `docs/contracts/intent.schema.json`

## Next Steps

1. **Run full E2E test**: `./hack/run-e2e.sh`
2. **Integrate with CI**: Add to GitHub Actions workflow
3. **Performance testing**: Test with larger intent volumes
4. **Porch integration**: Connect with actual Porch instance
5. **Monitoring integration**: Add metrics collection

## Support

For issues or questions about the E2E harness:
1. Check the troubleshooting section above
2. Run the validation script: `./hack/test-e2e-enhancements.sh`
3. Enable debug mode: `SKIP_CLEANUP=true VERBOSE=true ./hack/run-e2e.sh`
4. Review component logs and process status