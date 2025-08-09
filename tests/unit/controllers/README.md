# NetworkIntent Controller Tests

This directory contains comprehensive unit tests for the NetworkIntent controller, including both traditional unit tests and environment-based tests (envtest).

## Test Files

- `networkintent_controller_test.go` - Original comprehensive unit tests with mocked dependencies
- `networkintent_envtest.go` - **New envtest-based integration tests** that run against a real Kubernetes API server
- `e2nodeset_controller_test.go` - E2NodeSet controller tests
- `suite_test.go` - Test suite configuration

## NetworkIntent Envtest

The `networkintent_envtest.go` file provides comprehensive testing of the NetworkIntent controller using controller-runtime's envtest framework, which spins up a real Kubernetes API server and etcd for testing.

### Features Tested

1. **Phase Transitions**:
   - Non-empty intent → "Validated" phase
   - LLM enabled → "Processed" phase  
   - Empty intent → "Error" phase

2. **Environment Variable Behavior**:
   - `ENABLE_LLM_INTENT=true` - Enables LLM processing
   - `ENABLE_LLM_INTENT=1` - Alternative enablement
   - `ENABLE_LLM_INTENT=false` - Disables LLM processing
   - Unset variable - Defaults to disabled

3. **Error Handling**:
   - LLM service errors
   - Unreachable LLM endpoints
   - Network timeout scenarios

4. **Status Management**:
   - ObservedGeneration tracking
   - LastUpdateTime updates
   - Status message accuracy

### Running the Envtest

#### Prerequisites

```bash
# Install required tools
make deps

# Or install manually:
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
```

#### Method 1: Using Go Test
```bash
# Run from project root
cd tests/unit/controllers
go test -v networkintent_envtest.go

# With coverage
go test -v -coverprofile=coverage.out networkintent_envtest.go
```

#### Method 2: Using Ginkgo
```bash
# Run from project root
cd tests/unit/controllers
ginkgo -v networkintent_envtest.go

# Run specific test
ginkgo -v --focus="phase transition to Validated" networkintent_envtest.go

# Run with coverage
ginkgo -v --cover --coverprofile=coverage.out networkintent_envtest.go
```

#### Method 3: Using the Custom Runner
```bash
# Run from project root  
cd tests/unit/controllers
go run run_envtest.go

# With options
go run run_envtest.go -v --focus="ENABLE_LLM_INTENT"
go run run_envtest.go --dry-run  # Show commands only
```

#### Method 4: Using Make Targets
```bash
# From project root
make test  # Runs all tests including envtest
```

### Environment Variables

The envtest can be configured with these environment variables:

- `ENABLE_LLM_INTENT` - Controls LLM processing (true/false/1/0)
- `LLM_PROCESSOR_ENDPOINT` - LLM service endpoint (set by test's mock server)
- `KUBEBUILDER_ASSETS` - Path to kubebuilder test assets (auto-detected)
- `TEST_ENV` - Indicates test environment (set to "true")

### Test Structure

The envtest follows this structure:

```
BeforeSuite: Start envtest environment with real K8s API server
  ├── Add CRDs to scheme
  ├── Start envtest
  └── Create K8s client

BeforeEach: Setup test context and mock servers
  ├── Create test context with timeout
  ├── Start HTTP mock server for LLM
  ├── Set environment variables
  └── Create unique namespace

Describe: Test suites for different scenarios
  ├── "NetworkIntent Status Phase Transitions"
  ├── "ENABLE_LLM_INTENT Environment Variable Behavior"  
  ├── "LLM Integration Error Handling"
  └── "Status Updates and Observed Generation"

AfterEach: Cleanup test resources
  ├── Restore environment variables
  ├── Stop mock servers
  └── Cancel context

AfterSuite: Stop envtest environment
```

### Key Differences from Unit Tests

| Aspect | Unit Tests | Envtest |
|--------|------------|---------|
| API Server | Fake/Mock | Real Kubernetes |
| CRDs | Mocked | Actually installed |
| Controllers | Isolated | Full integration |
| Performance | Very fast | Slower but realistic |
| Flakiness | Minimal | Some inherent |
| Coverage | Deep mocking | End-to-end behavior |

### Debugging

To debug envtest issues:

1. **Enable verbose logging**:
   ```bash
   ginkgo -v networkintent_envtest.go
   ```

2. **Check CRD installation**:
   ```bash
   export KUBEBUILDER_ASSETS=/path/to/assets
   setup-envtest use 1.28.0 --print path
   ```

3. **Inspect test artifacts**:
   - Check `/tmp/` for envtest temporary files
   - Look for controller logs in test output
   - Verify namespace and resource creation

4. **Common issues**:
   - Missing KUBEBUILDER_ASSETS path
   - CRD directory not found
   - Port conflicts on CI systems
   - Timeout on slower systems

### Performance Considerations

- Envtest startup: ~5-10 seconds
- Individual test: ~100-500ms each
- Total suite runtime: ~30-60 seconds
- Memory usage: ~500MB-1GB

For faster iteration during development, consider running specific test focuses rather than the entire suite.

### CI Integration

The envtest integrates with the project's CI pipeline:

```yaml
# In CI, the test uses optimized settings:
- Shorter timeouts
- Reduced verbosity  
- Parallel execution disabled
- Cleanup verification
```

This is automatically handled when `CI=true` environment variable is set.