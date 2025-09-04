# 2025 Go Testing Best Practices Implementation

This document describes the implementation of modern Go testing best practices applied to the Nephoran Intent Operator project, focusing on Phase 2 dependency resolution issues and comprehensive testing strategies.

## Overview

The testing improvements implement:

- **envtest Integration**: Realistic Kubernetes API testing with controller-runtime patterns
- **Go 1.24+ Context Patterns**: Proper context handling with cancellation and timeouts
- **Modern Constructor Patterns**: Flexible dependency injection for testing
- **Ginkgo/Gomega Integration**: BDD-style testing with controller-runtime best practices
- **Comprehensive Test Organization**: Test pyramid approach with proper cleanup

## Key Files Modified/Created

### 1. O2 API Server Constructor Updates

**File**: `pkg/oran/o2/api_server.go`

```go
// Updated constructor signature for 2025 patterns
func NewO2APIServer(config *O2IMSConfig, logger *logging.StructuredLogger, k8sClient interface{}) (*O2APIServer, error)

// Backward compatibility wrapper
func NewO2APIServerWithConfig(config *O2IMSConfig) (*O2APIServer, error)

// Testing support method
func (s *O2APIServer) GetRouter() *mux.Router
```

**Benefits**:
- Explicit dependency injection for better testing
- Backward compatibility maintained
- Support for mock dependencies in tests

### 2. Test Environment Setup

**File**: `tests/integration/test_utils.go`

```go
// Enhanced CreateTestNamespace with envtest patterns
func CreateTestNamespace() *corev1.Namespace

// Environment setup with automatic CI/dev detection
func SetupTestEnvironment() error

// Proper cleanup with context handling
func TeardownTestEnvironment()
```

**Features**:
- Automatic namespace creation in real Kubernetes API
- Environment-aware configuration (CI vs development)
- Proper resource cleanup with context handling

### 3. Comprehensive Test Suite

**File**: `tests/integration/suite_test.go`

```go
// BeforeSuite with envtest setup
var _ = BeforeSuite(func() {
    ctx, cancel = context.WithCancel(context.Background())
    testEnv, err = setupEnvtestEnvironment()
    // ...
})

// Context helpers for Go 1.24+ patterns
func TestContext(timeout time.Duration) (context.Context, context.CancelFunc)
func TestContextWithDeadline(deadline time.Time) (context.Context, context.CancelFunc)
```

**Advantages**:
- Centralized test environment management
- Consistent context patterns across all tests
- Automatic resource cleanup

### 4. Modern Test Examples

**File**: `tests/integration/controllers/modern_test_example.go`

Demonstrates:
- Context-aware test execution
- Proper resource lifecycle management
- Eventually patterns with timeouts
- Error handling best practices
- CRUD operation testing

### 5. envtest Utilities

**File**: `hack/testtools/envtest_setup.go`

Provides:
- Comprehensive test environment options
- CI/development mode detection
- CRD installation and verification
- Resource cleanup utilities
- Health checking and metrics

## Usage Examples

### Basic Test Setup

```go
var _ = Describe("My Component", func() {
    var (
        namespace  *corev1.Namespace
        testCtx    context.Context
        testCancel context.CancelFunc
    )

    BeforeEach(func() {
        // Create context with timeout
        testCtx, testCancel = TestContext(5 * time.Minute)
        
        // Create namespace using envtest
        namespace = CreateTestNamespaceWithContext(testCtx)
    })

    AfterEach(func() {
        // Cleanup resources
        CleanupTestNamespaceWithContext(testCtx, namespace)
        testCancel()
    })

    It("should work with modern patterns", func(ctx SpecContext) {
        // Test implementation with context
    }, SpecTimeout(2*time.Minute))
})
```

### O2 API Server Testing

```go
func TestO2APIServer(t *testing.T) {
    logger := logging.NewLogger("test", "debug")
    config := &o2.O2IMSConfig{
        ServerAddress: "127.0.0.1",
        ServerPort:    0,
        TLSEnabled:    false,
        // ... other config
    }

    // Use modern constructor
    o2Server, err := o2.NewO2APIServer(config, logger, nil)
    require.NoError(t, err)
    defer o2Server.Shutdown(context.Background())

    // Get router for testing
    router := o2Server.GetRouter()
    server := httptest.NewServer(router)
    defer server.Close()

    // Perform tests...
}
```

### Resource Management

```go
It("should handle resource lifecycle", func(ctx SpecContext) {
    By("creating resource with context")
    intent := &nephoranv1.NetworkIntent{
        // ... spec
    }
    
    Expect(k8sClient.Create(testCtx, intent)).To(Succeed())
    
    By("waiting for resource ready")
    Eventually(func(g Gomega) {
        retrieved := &nephoranv1.NetworkIntent{}
        err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(intent), retrieved)
        g.Expect(err).NotTo(HaveOccurred())
    }).WithContext(testCtx).
      WithTimeout(30*time.Second).
      Should(Succeed())
})
```

## Environment Detection

The testing framework automatically detects the environment:

```go
// Automatic CI detection
if testtools.IsRunningInCI() {
    opts = testtools.CITestEnvironmentOptions()
} else {
    opts = testtools.DevelopmentTestEnvironmentOptions()
}
```

**CI Mode Features**:
- Reduced timeouts for faster execution
- Lower resource limits
- Less verbose logging
- Optimized for parallel execution

**Development Mode Features**:
- Extended timeouts for debugging
- Verbose logging enabled
- Metrics and profiling enabled
- Better error reporting

## Best Practices Applied

### 1. Test Pyramid Approach
- **Unit Tests**: Fast, isolated, many
- **Integration Tests**: Medium speed, realistic dependencies, fewer
- **E2E Tests**: Slow, full system, minimal

### 2. Arrange-Act-Assert Pattern
```go
// Arrange
namespace := CreateTestNamespace()
intent := &nephoranv1.NetworkIntent{...}

// Act
err := k8sClient.Create(ctx, intent)

// Assert
Expect(err).NotTo(HaveOccurred())
```

### 3. Context-Aware Execution
- All operations use proper context handling
- Timeouts are consistently applied
- Cancellation is properly handled

### 4. Resource Cleanup
- Automatic cleanup with defer patterns
- Context-aware cleanup operations
- Proper error handling during cleanup

### 5. Deterministic Tests
- No race conditions
- Proper synchronization with Eventually
- Consistent test data

## Running Tests

### Development
```bash
# Run with development settings
go test ./tests/integration/... -v

# Run specific test
go test ./tests/integration/controllers -run TestModern2025Patterns
```

### CI Environment
```bash
# Run with CI optimizations
CI=true go test ./tests/integration/... -timeout=10m

# Parallel execution
go test ./tests/... -parallel 4 -timeout=15m
```

### Using Test Runner
```bash
# Run comprehensive demo
go run cmd/test-runner-2025/main.go
```

## Key Improvements

### Phase 2 Dependency Resolution Fixes

1. **O2 API Server Constructor**: Fixed signature mismatches using flexible parameter patterns
2. **Test Utilities**: Replaced fake clients with envtest for realistic API testing
3. **Context Handling**: Applied Go 1.24+ context patterns throughout test setup
4. **Environment Setup**: Implemented proper envtest Environment initialization

### Testing Quality Enhancements

1. **Realistic Testing**: envtest provides actual Kubernetes API server
2. **Proper Cleanup**: Context-aware resource cleanup prevents test pollution
3. **Environment Detection**: Automatic CI/development mode selection
4. **Error Handling**: Comprehensive error handling in test setup and teardown
5. **Performance**: Optimized test execution for both CI and development

## Migration Guide

### From Old Patterns
```go
// Old: Fake client usage
fakeClient := fake.NewSimpleClientset()

// New: envtest usage
testEnv, _ := testtools.SetupTestEnvironment()
k8sClient := testEnv.K8sClient
```

### From Old Constructor
```go
// Old: Single parameter constructor
server, err := o2.NewO2APIServer(config)

// New: Explicit dependencies
server, err := o2.NewO2APIServer(config, logger, k8sClient)

// Or use compatibility wrapper
server, err := o2.NewO2APIServerWithConfig(config)
```

### From Basic Context Usage
```go
// Old: Basic context
ctx := context.Background()

// New: Timeout context with cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

## Troubleshooting

### Common Issues

1. **CRD Not Found**: Ensure CRD paths are correctly configured in test options
2. **Context Timeout**: Increase timeout for slow operations in development mode
3. **Resource Cleanup**: Use proper cleanup functions to prevent test pollution
4. **envtest Binary**: Ensure Kubernetes test binaries are installed

### Debug Tips

1. Enable verbose logging: `opts.VerboseLogging = true`
2. Use development mode: `opts.DevelopmentMode = true`
3. Check test environment setup: `testEnv.VerifyCRDInstallation()`
4. Monitor resource cleanup: Enable cleanup logging

## Conclusion

These 2025 Go testing best practices provide:

- **Reliability**: Tests run consistently across environments
- **Speed**: Optimized execution for CI and development
- **Maintainability**: Clear patterns and proper organization
- **Realism**: Tests use actual Kubernetes API servers
- **Quality**: Comprehensive coverage with proper cleanup

The implementation resolves Phase 2 dependency issues while establishing a solid foundation for future testing improvements.