# Porch API v1alpha1 Client Implementation

This package provides a production-ready Porch API v1alpha1 client for the Nephoran Intent Operator, replacing the previous stub implementations with full CRUD operations for Porch resources.

## Overview

The Porch client implements comprehensive support for:
- Repository management (Git, OCI)
- Package revision lifecycle (Draft → Proposed → Published)
- Package content operations (upload/download KRM resources)
- KRM function execution and pipeline rendering
- Workflow management and approval processes
- Health monitoring and observability

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client API    │    │  Circuit        │    │  Rate Limiter   │
│   (PorchClient) │────│  Breaker        │────│  (golang.org/x/ │
└─────────────────┘    └─────────────────┘    │  time/rate)     │
         │                       │             └─────────────────┘
         ▼                       │
┌─────────────────┐              │             ┌─────────────────┐
│   Kubernetes    │              │             │     Cache      │
│   Dynamic       │              │             │   (TTL-based)   │
│   Client        │              │             └─────────────────┘
└─────────────────┘              │
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌─────────────────┐
│  Porch Server   │    │   Prometheus    │
│  (REST API)     │    │   Metrics       │
└─────────────────┘    └─────────────────┘
```

## Key Features

### 1. Production-Ready Implementation
- **Real Porch API calls** using Kubernetes dynamic client
- **Proper resource group**: `porch.kpt.dev/v1alpha1`
- **Authentication support**: Bearer tokens, basic auth, kubeconfig
- **TLS configuration**: Custom CA, client certificates
- **Error handling**: Comprehensive error types and recovery

### 2. Resilience Patterns
- **Circuit breaker**: Prevents cascade failures with configurable thresholds
- **Rate limiting**: Token bucket algorithm with burst support
- **Retry logic**: Exponential backoff for transient failures
- **Graceful degradation**: Partial functionality during component failures

### 3. Performance Optimization
- **Intelligent caching**: TTL-based with LRU eviction
- **Connection pooling**: Reuses HTTP connections
- **Batch operations**: Multiple resources in single requests
- **Streaming support**: Server-Sent Events for real-time updates

### 4. Observability
- **Prometheus metrics**: Request latency, error rates, cache hits
- **Structured logging**: With correlation IDs and context
- **Distributed tracing**: Integration ready
- **Health checks**: Readiness and liveness probes

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
    logger := zap.New(zap.UseDevMode(true))
    
    // Create client with default configuration
    client, err := porch.NewClient(porch.ClientOptions{
        Config:         porch.DefaultPorchConfig(),
        Logger:         logger,
        MetricsEnabled: true,
        CacheEnabled:   true,
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()

    ctx := context.Background()

    // List repositories
    repos, err := client.ListRepositories(ctx, &porch.ListOptions{})
    if err != nil {
        panic(err)
    }
    fmt.Printf("Found %d repositories\n", len(repos.Items))

    // Create package revision
    pkg := &porch.PackageRevision{
        Spec: porch.PackageRevisionSpec{
            PackageName: "example-package",
            Repository:  "default",
            Revision:    "v1.0.0",
            Lifecycle:   porch.PackageRevisionLifecycleDraft,
            Resources: []porch.KRMResource{
                {
                    APIVersion: "v1",
                    Kind:       "ConfigMap",
                    Metadata: map[string]interface{}{
                        "name": "example-config",
                    },
                    Data: map[string]interface{}{
                        "key": "value",
                    },
                },
            },
        },
    }

    created, err := client.CreatePackageRevision(ctx, pkg)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Created package: %s\n", created.Name)
}
```

### Advanced Configuration

```go
config := &porch.Config{
    Endpoint: "https://porch.company.com:8443",
    AuthConfig: &porch.AuthConfig{
        Type:  "bearer",
        Token: "eyJhbGciOiJSUzI1NiIs...",
    },
    TLSConfig: &porch.TLSConfig{
        CAFile:   "/etc/ssl/ca.pem",
        CertFile: "/etc/ssl/client.pem",
        KeyFile:  "/etc/ssl/client.key",
    },
    PorchConfig: &porch.PorchConfig{
        CircuitBreaker: &porch.CircuitBreakerConfig{
            Enabled:          true,
            FailureThreshold: 5,
            Timeout:          30 * time.Second,
        },
        RateLimit: &porch.RateLimitConfig{
            Enabled:           true,
            RequestsPerSecond: 10.0,
            Burst:             20,
        },
    },
}

client, err := porch.NewClient(porch.ClientOptions{
    Config:         config,
    Logger:         logger,
    MetricsEnabled: true,
    CacheEnabled:   true,
})
```

### Package Lifecycle Management

```go
// Create in Draft
pkg, _ := client.CreatePackageRevision(ctx, packageRevision)

// Validate
validation, _ := client.ValidatePackage(ctx, "package-name", "v1.0.0")
if !validation.Valid {
    // Handle validation errors
}

// Propose for review
_ = client.ProposePackageRevision(ctx, "package-name", "v1.0.0")

// Approve and publish
_ = client.ApprovePackageRevision(ctx, "package-name", "v1.0.0")
```

### Function Execution

```go
req := &porch.FunctionRequest{
    FunctionConfig: porch.FunctionConfig{
        Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
        ConfigMap: map[string]interface{}{
            "namespace": "production",
        },
    },
    Resources: []porch.KRMResource{
        // Input resources
    },
}

resp, err := client.RunFunction(ctx, req)
if err != nil {
    // Handle error
}

// Process transformed resources
for _, resource := range resp.Resources {
    // Handle each resource
}
```

## Resource Types

### Repository
```yaml
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: example-repo
spec:
  type: git
  url: https://github.com/example/packages.git
  branch: main
```

### PackageRevision
```yaml
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  name: example-package-v1
spec:
  packageName: example-package
  repository: example-repo
  revision: v1.0.0
  lifecycle: Draft
  resources:
  - apiVersion: v1
    kind: ConfigMap
    metadata:
      name: example-config
    data:
      key: value
```

## Configuration Options

### Circuit Breaker
```go
CircuitBreaker: &porch.CircuitBreakerConfig{
    Enabled:             true,      // Enable circuit breaker
    FailureThreshold:    5,         // Failures before opening
    SuccessThreshold:    3,         // Successes before closing
    Timeout:             30 * time.Second, // Time before half-open
    HalfOpenMaxCalls:    3,         // Max calls in half-open state
}
```

### Rate Limiting
```go
RateLimit: &porch.RateLimitConfig{
    Enabled:           true,        // Enable rate limiting
    RequestsPerSecond: 10.0,        // Sustained rate
    Burst:             20,          // Burst capacity
}
```

### Caching
```go
ClientOptions{
    CacheEnabled: true,             // Enable caching
    CacheSize:    1000,            // Max cache entries
    CacheTTL:     5 * time.Minute, // Cache entry lifetime
}
```

## Metrics

The client exposes Prometheus metrics:

- `porch_client_requests_total` - Total requests by operation and status
- `porch_client_request_duration_seconds` - Request latency histograms
- `porch_client_cache_hits_total` - Cache hit count
- `porch_client_cache_misses_total` - Cache miss count
- `porch_client_circuit_breaker_state` - Circuit breaker state (0=closed, 1=half-open, 2=open)
- `porch_client_errors_total` - Error count by operation and type

## Error Handling

The client provides typed errors for better error handling:

```go
import "errors"

_, err := client.GetRepository(ctx, "nonexistent")
if err != nil {
    var porchErr *porch.PorchError
    if errors.As(err, &porchErr) {
        switch porchErr.Type {
        case "RepositoryNotFound":
            // Handle not found
        case "ValidationFailed":
            // Handle validation error
        default:
            // Handle other errors
        }
    }
}
```

## Testing

Run the test suite:

```bash
# Unit tests
go test ./pkg/nephio/porch/

# With coverage
go test -coverprofile=coverage.out ./pkg/nephio/porch/

# Benchmarks
go test -bench=. ./pkg/nephio/porch/
```

## Integration with Nephoran Intent Operator

The Porch client integrates seamlessly with the Nephoran Intent Operator:

1. **Intent Processing**: LLM-generated intents create PackageRevisions
2. **Template Management**: Repository operations manage intent templates  
3. **Deployment**: Package approval triggers GitOps deployment
4. **Monitoring**: Metrics integration with operator observability

## Migration from Stubs

To migrate from the previous stub implementation:

1. **Update imports**: No changes needed
2. **Update configuration**: Add Porch-specific config
3. **Handle errors**: Real errors instead of nil returns
4. **Update tests**: Mock real Porch responses

## Development

### Adding New Operations

1. Define the operation in `types.go` interface
2. Implement the public method in `client.go`  
3. Implement the internal method with real Porch API calls
4. Add validation logic
5. Add tests in `client_test.go`
6. Update examples in `example_usage.go`

### Performance Tuning

- Adjust cache TTL based on update frequency
- Configure circuit breaker thresholds for your environment
- Set rate limits based on Porch server capacity
- Monitor metrics to identify bottlenecks

## Contributing

1. Follow Go best practices and idioms
2. Add comprehensive tests for new functionality
3. Update documentation and examples
4. Ensure metrics and logging are properly implemented
5. Test against real Porch server when possible

## License

Licensed under the Apache License, Version 2.0.