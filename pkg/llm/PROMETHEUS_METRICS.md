# Prometheus Metrics Integration for LLM Client

This document describes the clean Prometheus metrics implementation for the Nephoran Intent Operator's LLM client.

## Overview

The implementation provides comprehensive Prometheus metrics for LLM operations while maintaining compatibility with the existing custom metrics system. Metrics are conditionally enabled based on the `METRICS_ENABLED` environment variable, following Go best practices and established patterns.

## Architecture

### Core Components

1. **PrometheusMetrics**: Contains all Prometheus metric definitions
2. **MetricsIntegrator**: Integrates Prometheus metrics with existing MetricsCollector
3. **Client Integration**: Enhanced LLM client with automatic metrics recording

### Design Principles

- **Environment Variable Gating**: Metrics only registered when `METRICS_ENABLED=true`
- **Thread-Safe**: All operations are safe for concurrent use
- **No Breaking Changes**: Existing functionality remains unchanged
- **Clean Integration**: Follows existing codebase patterns

## Metrics Definition

### Counter Metrics

| Metric Name | Description | Labels |
|-------------|-------------|---------|
| `nephoran_llm_requests_total` | Total LLM requests | `model`, `status` |
| `nephoran_llm_errors_total` | Total LLM errors | `model`, `error_type` |
| `nephoran_llm_cache_hits_total` | Total cache hits | `model` |
| `nephoran_llm_cache_misses_total` | Total cache misses | `model` |
| `nephoran_llm_retry_attempts_total` | Total retry attempts | `model` |
| `nephoran_llm_fallback_attempts_total` | Total fallback attempts | `original_model`, `fallback_model` |

### Histogram Metrics

| Metric Name | Description | Labels | Buckets |
|-------------|-------------|---------|---------|
| `nephoran_llm_processing_duration_seconds` | Processing duration | `model`, `status` | 0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0 |

### Label Values

- **model**: LLM model name (e.g., `gpt-4o-mini`, `mistral-8x22b`)
- **status**: `success` or `error`
- **error_type**: Categorized error types:
  - `circuit_breaker_open`
  - `timeout`
  - `connection_refused`
  - `dns_resolution`
  - `tls_error`
  - `authentication_error`
  - `authorization_error`
  - `rate_limit_exceeded`
  - `server_error`
  - `server_unavailable`
  - `parsing_error`
  - `unknown_error`

## Implementation Details

### Environment Variable Control

```go
func isMetricsEnabled() bool {
    return os.Getenv("METRICS_ENABLED") == "true"
}
```

Metrics are only registered when `METRICS_ENABLED=true`, providing fine-grained control over monitoring overhead.

### Thread-Safe Registration

```go
var (
    prometheusOnce sync.Once
    prometheusMetrics *PrometheusMetrics
)
```

Uses `sync.Once` to ensure metrics are registered exactly once, preventing duplicate registration errors.

### Graceful Degradation

```go
func (pm *PrometheusMetrics) RecordRequest(model, status string, duration time.Duration) {
    if !pm.isRegistered() {
        return
    }
    // ... record metrics
}
```

All metric recording methods safely handle unregistered state without panics.

### Error Categorization

The implementation includes intelligent error categorization:

```go
func (c *Client) categorizeError(err error) string {
    errMsg := err.Error()
    switch {
    case strings.Contains(errMsg, "circuit breaker is open"):
        return "circuit_breaker_open"
    case strings.Contains(errMsg, "timeout"):
        return "timeout"
    // ... additional categorizations
    }
}
```

## Integration Points

### Client Integration

The LLM client automatically records metrics during request processing:

1. **Request Metrics**: Recorded in `updateMetrics()` function
2. **Error Classification**: Enhanced in `ProcessIntent()` with error categorization  
3. **Cache Operations**: Automatically tracked during cache hits/misses
4. **Circuit Breaker Events**: Detected and categorized appropriately

### Dual Metrics System

The implementation maintains both systems:

- **Existing MetricsCollector**: Continues to work unchanged
- **Prometheus Metrics**: Added alongside for monitoring integration

```go
type MetricsIntegrator struct {
    collector         *MetricsCollector
    prometheusMetrics *PrometheusMetrics
}
```

## Usage Examples

### Basic Setup

```bash
export METRICS_ENABLED=true
```

```go
client := NewClient("https://api.openai.com/v1/chat/completions")
response, err := client.ProcessIntent(ctx, "Deploy 5G AMF")
// Metrics automatically recorded
```

### Metric Access

```go
// Get comprehensive metrics including Prometheus status
metrics := client.metricsIntegrator.GetComprehensiveMetrics()
fmt.Printf("Prometheus enabled: %v\n", metrics["prometheus_enabled"])
```

### Manual Metric Recording

```go
integrator.RecordLLMRequest("gpt-4o-mini", "success", 500*time.Millisecond, 150)
integrator.RecordCacheOperation("gpt-4o-mini", "get", true)
integrator.RecordRetryAttempt("gpt-4o-mini")
```

## Monitoring Integration

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'nephoran-llm'
    static_configs:
      - targets: ['llm-processor:8080']
    scrape_interval: 15s
    metrics_path: /metrics
```

### Example Queries

```promql
# Request rate by model
rate(nephoran_llm_requests_total[5m])

# Error rate percentage
rate(nephoran_llm_errors_total[5m]) / rate(nephoran_llm_requests_total[5m]) * 100

# Average processing time
histogram_quantile(0.95, nephoran_llm_processing_duration_seconds_bucket)

# Cache hit rate
rate(nephoran_llm_cache_hits_total[5m]) / (rate(nephoran_llm_cache_hits_total[5m]) + rate(nephoran_llm_cache_misses_total[5m]))
```

## Testing

The implementation includes comprehensive tests covering:

- Environment variable gating
- Metrics registration behavior
- Recording functionality
- Error categorization
- Integration scenarios

Run tests with:
```bash
go test -run TestPrometheusMetrics -v
```

## Performance Considerations

- **Zero Overhead**: When disabled, metrics have minimal performance impact
- **Efficient Recording**: Uses prometheus/client_golang optimized collectors
- **Memory Usage**: Bounded by cardinality of model names and error types
- **Registration Cost**: One-time cost during initialization

## Security

- **No Sensitive Data**: Metrics don't expose request content or API keys
- **Configurable Exposure**: Controlled via environment variables
- **Standard Labels**: Only includes safe categorization labels

## Compatibility

- **Go Version**: Compatible with Go 1.21+
- **Prometheus**: Uses prometheus/client_golang v1.14+
- **Kubernetes**: Integrates with controller-runtime metrics registry
- **Existing Code**: No breaking changes to existing functionality

## Future Enhancements

Potential future improvements:

1. **Model-Specific Buckets**: Custom histogram buckets per model type
2. **Token Metrics**: Enhanced token usage tracking per request
3. **Cost Metrics**: Integration with token cost calculations
4. **SLO Metrics**: Service Level Objective tracking
5. **Request Size Metrics**: Payload size distribution tracking

## Troubleshooting

### Common Issues

1. **Metrics Not Appearing**
   - Verify `METRICS_ENABLED=true`
   - Check application logs for registration errors
   - Confirm `/metrics` endpoint accessibility

2. **Duplicate Registration Errors**
   - Ensure singleton pattern is working correctly
   - Check for multiple client initializations

3. **High Cardinality**
   - Monitor label cardinality, especially model names
   - Consider label value limits if needed

### Debug Information

```go
metrics := client.metricsIntegrator.GetComprehensiveMetrics()
fmt.Printf("Debug info: %+v\n", metrics)
```

This provides comprehensive debugging information about the metrics system state.