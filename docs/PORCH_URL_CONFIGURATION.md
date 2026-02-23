# Porch URL Configuration Guide

**Last Updated**: 2026-02-23
**Task**: #30 - Remove hardcoded porch-server:8080 from test utilities

---

## Overview

As of Task #30, all hardcoded `porch-server:8080` URLs have been removed from test utilities. The Porch endpoint is now fully configurable via environment variables, with sensible defaults for different environments.

## Environment Variables

The following environment variables are used to configure the Porch endpoint (checked in priority order):

1. **`PORCH_SERVER_URL`** - Primary environment variable (highest priority)
2. **`PORCH_ENDPOINT`** - Alternative environment variable

If neither is set, the system uses context-appropriate defaults:

- **Local/Mock environments**: `http://localhost:7007`
- **Real Kubernetes clusters**: `http://porch-server.porch-system.svc.cluster.local:7007`

## Default Port Change

**Important**: The default Porch port has been updated from `8080` to `7007` to align with standard Porch deployment configurations.

- **Old default (deprecated)**: `http://porch-server:8080`
- **New default**: `http://porch-server.porch-system.svc.cluster.local:7007`

## Usage Examples

### Running Tests Locally

```bash
# Use default localhost endpoint (http://localhost:7007)
go test ./test/integration/...

# Use custom local endpoint
PORCH_SERVER_URL=http://localhost:9090 go test ./test/integration/...

# Use in-cluster service (when testing from within cluster)
PORCH_SERVER_URL=http://porch-server.porch-system.svc.cluster.local:7007 \
  go test ./test/integration/...
```

### Running Tests in CI/CD

```bash
# For CI environments where Porch is on a different host
export PORCH_SERVER_URL=http://porch.test.svc.cluster.local:7007
go test ./test/integration/... -v

# Or using PORCH_ENDPOINT (alternative)
export PORCH_ENDPOINT=http://porch-server:7007
go test ./test/integration/... -v
```

### Production Deployment

Set the environment variable in your deployment manifests:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-operator
spec:
  template:
    spec:
      containers:
      - name: operator
        env:
        - name: PORCH_SERVER_URL
          value: "http://porch-server.porch-system.svc.cluster.local:7007"
```

## Test Behavior

### Automatic Skipping

Integration tests that require Porch will automatically skip if:

1. Porch is not available at the configured endpoint
2. The health check fails (timeout or connection error)

Example skip message:

```
--- SKIP: TestPorchIntegration (0.01s)
    Skipping: Porch server not available at http://porch-server.porch-system.svc.cluster.local:7007 - connection refused
    Set PORCH_SERVER_URL or PORCH_ENDPOINT environment variable to override
```

### Test Logging

Tests will log the Porch URL being used:

```
=== RUN   TestPorchIntegration
    porch_integration_test.go:115: Using Porch URL: http://porch-server.porch-system.svc.cluster.local:7007
```

## Files Modified

### Test Utilities

- **`test/testutil/k8s_config.go`**
  - Updated `getTestPorchEndpoint()` to check environment variables
  - Added comprehensive documentation
  - Changed default ports from 8080 to 7007

- **`test/testutil/k8s_config_test.go`**
  - Updated test cases for new defaults
  - Added tests for both `PORCH_SERVER_URL` and `PORCH_ENDPOINT`
  - Improved test isolation with proper env var cleanup

### Integration Tests

- **`test/integration/porch_integration_test.go`**
  - Added `getPorchURL()` helper function
  - Updated `checkPorchAvailability()` to accept URL parameter
  - Modified `setupTestEnvironment()` to use configurable URL
  - Added package-level documentation
  - Improved skip messages with configuration hints

- **`test/integration/system_test.go`**
  - Added `getSystemTestPorchURL()` helper function
  - Updated to use environment-based configuration
  - Added package-level documentation

### Validation Tests

- **`pkg/validation/endpoint_validator_test.go`**
  - Added documentation comment clarifying test examples
  - Noted that hardcoded URLs in tests are just validation examples

## Migration Guide

If you have existing test scripts or CI/CD pipelines that relied on the old hardcoded defaults:

### Before (hardcoded)

```bash
# Tests assumed porch-server:8080 was always available
go test ./test/integration/...
```

### After (configurable)

```bash
# Option 1: Use default (recommended for in-cluster testing)
go test ./test/integration/...

# Option 2: Override for custom environments
PORCH_SERVER_URL=http://my-porch-server:7007 go test ./test/integration/...

# Option 3: Use port forwarding for local development
kubectl port-forward -n porch-system svc/porch-server 7007:7007 &
go test ./test/integration/...
```

## Benefits

1. **Flexibility**: Tests can run in any environment without code changes
2. **No Hardcoding**: Production code is properly configurable
3. **Better Defaults**: Context-aware defaults (localhost vs. in-cluster)
4. **Clear Documentation**: Environment variables documented in code and tests
5. **Automatic Skipping**: Tests skip gracefully when Porch is unavailable
6. **Port Alignment**: Consistent use of standard Porch port (7007)

## Troubleshooting

### Issue: Tests skip with "Porch server not available"

**Solution**: Set the correct Porch URL for your environment:

```bash
# Check where Porch is running
kubectl get svc -n porch-system

# Set the appropriate URL
export PORCH_SERVER_URL=http://porch-server.porch-system.svc.cluster.local:7007
```

### Issue: Connection refused on localhost:7007

**Solution**: Set up port forwarding or use in-cluster URL:

```bash
# Option 1: Port forward
kubectl port-forward -n porch-system svc/porch-server 7007:7007

# Option 2: Run tests from within cluster
PORCH_SERVER_URL=http://porch-server.porch-system.svc.cluster.local:7007 \
  go test ./test/integration/...
```

### Issue: Wrong port (8080 vs 7007)

**Solution**: Porch's standard port is 7007. If your deployment uses 8080, set it explicitly:

```bash
export PORCH_SERVER_URL=http://porch-server:8080
```

## Related Documentation

- **ARCHITECTURAL_HEALTH_ASSESSMENT.md**: Original issue identification (Task #30)
- **test/testutil/k8s_config.go**: Implementation details
- **test/integration/porch_integration_test.go**: Test usage examples

---

**Implementation Date**: 2026-02-23
**Completion Status**: ✅ Complete
**Tests Passing**: ✅ All test utilities and integration tests passing
