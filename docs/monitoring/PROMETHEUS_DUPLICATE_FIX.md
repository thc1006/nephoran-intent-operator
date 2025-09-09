# Prometheus Duplicate Metrics Registration Fix

This document explains the comprehensive fix implemented to resolve the "panic: duplicate metrics collector registration attempted" error that was occurring in tests.

## Problem Description

The error was caused by multiple attempts to register the same Prometheus metrics collectors, particularly in test environments where:

1. **Sequential test runs** would try to register the same metrics
2. **Concurrent test execution** could cause race conditions
3. **Package initialization** functions (`init()`) running multiple times
4. **Controller startup** attempting to re-register metrics after restarts
5. **TestMain functions** not properly isolating metrics between test runs

## Root Causes

### 1. Direct use of `prometheus.MustRegister()`
```go
// PROBLEMATIC CODE
prometheus.MustRegister(
    myCounter,
    myHistogram,
    myGauge,
)
```

This would panic on the second registration attempt.

### 2. Lack of Test Isolation

Tests were sharing the same global Prometheus registry, causing conflicts when:
- Multiple tests registered the same metric names
- Tests ran concurrently
- Test cleanup didn't properly reset registry state

### 3. No Centralized Registry Management

Different components were managing their own metrics registration logic, leading to:
- Inconsistent error handling
- Duplicate registration attempts
- Lack of coordination between components

## Solution Overview

The fix implements a **Centralized Global Registry** system with the following key components:

1. **`pkg/monitoring/registry.go`** - Centralized registry management
2. **`pkg/monitoring/testing.go`** - Test utilities for metrics isolation
3. **Updated components** to use centralized safe registration
4. **Comprehensive tests** to validate the fix

## Key Components

### 1. Global Registry Manager (`pkg/monitoring/registry.go`)

```go
type GlobalRegistry struct {
    production prometheus.Registerer
    test       prometheus.Registerer
    testMode   bool
    mu         sync.RWMutex
    initFlags  map[string]bool // Track initialized components
}
```

**Features:**
- **Singleton pattern** ensures single instance across application
- **Mode switching** between production and test registries
- **Component tracking** prevents duplicate initialization
- **Thread-safe** operations with proper locking

### 2. Safe Registration

```go
func (gr *GlobalRegistry) SafeRegister(component string, collector prometheus.Collector) error {
    // Check if already registered for this component/mode
    // Use appropriate registry (production or test)
    // Handle AlreadyRegisteredError gracefully
    // Track successful registrations
}
```

**Benefits:**
- **No panics** on duplicate registration
- **Component-based tracking** allows different components to register same metric names
- **Mode-aware** registration (production vs test isolation)

### 3. Test Utilities (`pkg/monitoring/testing.go`)

```go
func SetupTestMetrics(t *testing.T) {
    gr := GetGlobalRegistry()
    gr.SetTestMode(true)
    gr.ResetForTest()
    t.Cleanup(func() {
        TeardownTestMetrics()
    })
}
```

**Features:**
- **Automatic cleanup** using `t.Cleanup()`
- **Test isolation** with separate registries
- **Registry reset** between tests

## Updated Components

### Before (Problematic)

```go
// audit/query.go
prometheus.MustRegister(
    metrics.queriesTotal,
    metrics.queryDuration,
    // ... more metrics
)
```

### After (Fixed)

```go
// audit/query.go
gr := monitoring.GetGlobalRegistry()
gr.SafeRegister("audit-query-queries", metrics.queriesTotal)
gr.SafeRegister("audit-query-duration", metrics.queryDuration)
// ... more metrics with unique component names
```

## Components Updated

1. **`pkg/audit/query.go`** - Audit system metrics
2. **`pkg/llm/dynamic_context_manager.go`** - LLM context metrics
3. **`pkg/monitoring/health/enhanced_checker.go`** - Health check metrics
4. **`pkg/controllers/e2nodeset_controller.go`** - Controller metrics
5. **`pkg/monitoring/alerting/`** - Alert manager metrics
6. **`pkg/monitoring/metrics.go`** - Core monitoring metrics
7. **`pkg/llm/prometheus_metrics.go`** - LLM Prometheus integration

## Usage Patterns

### For Production Code

```go
package mypackage

import "github.com/thc1006/nephoran-intent-operator/pkg/monitoring"

func initMetrics() {
    gr := monitoring.GetGlobalRegistry()
    
    myCounter := prometheus.NewCounter(prometheus.CounterOpts{
        Name: "my_counter_total",
        Help: "Total number of my events",
    })
    
    // Safe registration - won't panic on duplicates
    err := gr.SafeRegister("my-component", myCounter)
    if err != nil {
        // Handle error (rare, usually means system issue)
        log.Error(err, "Failed to register metrics")
    }
}
```

### For Tests

```go
func TestMyFeature(t *testing.T) {
    // Setup isolated metrics for this test
    monitoring.SetupTestMetrics(t)
    // Cleanup is automatic via t.Cleanup()
    
    // Your test code here - metrics registration won't conflict
    initMetrics()
    
    // Test continues...
}
```

### For Test Suites with TestMain

```go
func TestMain(m *testing.M) {
    // Optional: Setup global test environment
    gr := monitoring.GetGlobalRegistry()
    gr.SetTestMode(true)
    
    // Run tests
    code := m.Run()
    
    // Cleanup
    monitoring.TeardownTestMetrics()
    
    os.Exit(code)
}
```

## Backward Compatibility

The fix maintains backward compatibility:

1. **Existing `safeRegister` function** is marked as deprecated but still works
2. **Legacy registration patterns** continue to function
3. **No breaking changes** to public APIs
4. **Gradual migration** possible - components can be updated incrementally

## Testing

The fix includes comprehensive tests:

1. **`TestDuplicateRegistrationFix`** - Validates no panics on duplicates
2. **`TestRealWorldScenarios`** - Tests common problematic patterns
3. **`TestGlobalRegistry`** - Tests registry management
4. **`TestConcurrentAccess`** - Tests thread safety
5. **`TestSetupTeardownTestMetrics`** - Tests cleanup mechanisms

### Running the Tests

```bash
# Test the fix
go test ./pkg/monitoring -v -run="TestDuplicateRegistrationFix"

# Test all monitoring package
go test ./pkg/monitoring -v

# Test LLM integration
go test ./pkg/llm -run="TestPrometheusMetricsBasic" -v
```

## Benefits

1. **No More Panics** - `SafeRegister` handles duplicates gracefully
2. **Test Isolation** - Tests run in isolated metric registries
3. **Concurrent Safety** - Thread-safe operations with proper locking
4. **Component Tracking** - Prevents duplicate initialization per component
5. **Easy Testing** - Simple test setup/teardown utilities
6. **Centralized Management** - Single point of control for all metrics
7. **Production Ready** - Maintains production performance and reliability

## Migration Guide

### Step 1: Update Imports

```go
import "github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
```

### Step 2: Replace Direct Registration

```go
// OLD
prometheus.MustRegister(myMetric)

// NEW
gr := monitoring.GetGlobalRegistry()
gr.SafeRegister("my-component", myMetric)
```

### Step 3: Update Tests

```go
func TestMyFunction(t *testing.T) {
    monitoring.SetupTestMetrics(t) // Add this line
    
    // Rest of test unchanged
}
```

### Step 4: Optional TestMain

```go
func TestMain(m *testing.M) {
    // Setup
    gr := monitoring.GetGlobalRegistry()
    gr.SetTestMode(true)
    
    // Run
    code := m.Run()
    
    // Cleanup
    monitoring.TeardownTestMetrics()
    
    os.Exit(code)
}
```

## Best Practices

1. **Use unique component names** for `SafeRegister` to avoid conflicts
2. **Always call `SetupTestMetrics(t)`** in tests that use metrics
3. **Prefer `SafeRegister` over `MustRegister`** for new code
4. **Use descriptive component names** like "controller-reconciles" not just "metrics"
5. **Test metric registration** in your component tests
6. **Group related metrics** under the same component name

## Troubleshooting

### If you still see duplicate registration errors:

1. **Check for direct `prometheus.MustRegister` calls** - replace with `SafeRegister`
2. **Ensure tests call `SetupTestMetrics(t)`**
3. **Verify component names are unique** across your codebase
4. **Check for concurrent initialization** in your init functions

### Debug logging:

```go
// Enable verbose logging to see registration attempts
gr := monitoring.GetGlobalRegistry()
// Check if component already registered
if gr.IsComponentRegistered("my-component") {
    log.Info("Component already registered")
}
```

## Performance Impact

The fix has minimal performance impact:
- **Registration** happens once per component (startup time)
- **Runtime metrics collection** unchanged
- **Memory overhead** is negligible (small tracking map)
- **Lock contention** is minimal (registration is infrequent)

## Conclusion

This fix provides a robust, thread-safe solution to Prometheus metrics registration that:
- Eliminates panic conditions
- Provides proper test isolation
- Maintains backward compatibility
- Scales to large codebases
- Includes comprehensive testing

The centralized approach makes metrics management more predictable and maintainable across the entire Nephoran codebase.
