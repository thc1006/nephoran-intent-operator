# 📊 Prometheus Metrics Documentation

## Overview

The Nephoran Intent Operator provides comprehensive observability through Prometheus metrics, enabling detailed monitoring of LLM processing, controller operations, and system performance. Metrics are designed following Prometheus best practices with appropriate metric types, clear naming conventions, and meaningful labels for high-cardinality analysis.

**Key Features:**
- Conditional metrics registration via `METRICS_ENABLED` environment variable
- High-performance metrics collection with minimal overhead
- Multi-dimensional labels for detailed analysis
- Integration with existing MetricsCollector for backward compatibility
- Thread-safe operations with proper synchronization

## Configuration

### Environment Variable Control

Metrics collection is controlled by the `METRICS_ENABLED` environment variable:

```bash
# Enable metrics (required for Prometheus scraping)
export METRICS_ENABLED=true

# Disable metrics (default behavior)
export METRICS_ENABLED=false
```

### Metrics Endpoint

When enabled, metrics are exposed on the standard controller-runtime metrics endpoint:

```
GET /metrics
```

**Default Port**: `8080` (configurable via controller-runtime settings)

**Security**: Access can be restricted using `METRICS_ALLOWED_IPS` configuration.

## LLM Client Metrics

Located in `pkg/llm/prometheus_metrics.go`, these metrics provide comprehensive visibility into LLM processing operations.

### 📈 Counter Metrics

#### `nephoran_llm_requests_total`
**Type**: Counter  
**Description**: Total number of LLM requests processed  
**Labels**: 
- `model`: LLM model used (e.g., "gpt-4o-mini", "claude-3", "mistral-8x22b")
- `status`: Request outcome ("success", "error", "timeout", "rate_limited")

**Example Values**:
```
nephoran_llm_requests_total{model="gpt-4o-mini", status="success"} 1,247
nephoran_llm_requests_total{model="gpt-4o-mini", status="error"} 23
nephoran_llm_requests_total{model="claude-3", status="timeout"} 5
```

#### `nephoran_llm_errors_total`
**Type**: Counter  
**Description**: Total number of LLM processing errors  
**Labels**:
- `model`: LLM model that generated the error
- `error_type`: Error classification ("api_error", "timeout", "rate_limit", "authentication", "circuit_breaker_rejected", "processing_error", "request_failure")

**Example Values**:
```
nephoran_llm_errors_total{model="gpt-4o-mini", error_type="rate_limit"} 15
nephoran_llm_errors_total{model="gpt-4o-mini", error_type="timeout"} 8
nephoran_llm_errors_total{model="claude-3", error_type="api_error"} 3
```

#### `nephoran_llm_cache_hits_total`
**Type**: Counter  
**Description**: Total number of cache hits for LLM responses  
**Labels**:
- `model`: LLM model for cached response

**Example Values**:
```
nephoran_llm_cache_hits_total{model="gpt-4o-mini"} 892
nephoran_llm_cache_hits_total{model="claude-3"} 156
```

#### `nephoran_llm_cache_misses_total`
**Type**: Counter  
**Description**: Total number of cache misses requiring LLM API calls  
**Labels**:
- `model`: LLM model for missed cache lookup

**Example Values**:
```
nephoran_llm_cache_misses_total{model="gpt-4o-mini"} 378
nephoran_llm_cache_misses_total{model="claude-3"} 89
```

#### `nephoran_llm_fallback_attempts_total`
**Type**: Counter  
**Description**: Total number of fallback attempts when primary model fails  
**Labels**:
- `original_model`: Primary model that failed
- `fallback_model`: Secondary model used for fallback

**Example Values**:
```
nephoran_llm_fallback_attempts_total{original_model="gpt-4o-mini", fallback_model="claude-3"} 12
nephoran_llm_fallback_attempts_total{original_model="claude-3", fallback_model="mistral-8x22b"} 5
```

#### `nephoran_llm_retry_attempts_total`
**Type**: Counter  
**Description**: Total number of retry attempts for failed requests  
**Labels**:
- `model`: LLM model for retry attempts

**Example Values**:
```
nephoran_llm_retry_attempts_total{model="gpt-4o-mini"} 67
nephoran_llm_retry_attempts_total{model="claude-3"} 23
```

### 📊 Histogram Metrics

#### `nephoran_llm_processing_duration_seconds`
**Type**: Histogram  
**Description**: Duration of LLM processing requests  
**Labels**:
- `model`: LLM model used for processing
- `status`: Request outcome status

**Buckets**: `[0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0]` seconds

**Generated Metrics**:
```
nephoran_llm_processing_duration_seconds_bucket{model="gpt-4o-mini", status="success", le="0.5"} 234
nephoran_llm_processing_duration_seconds_bucket{model="gpt-4o-mini", status="success", le="1.0"} 456
nephoran_llm_processing_duration_seconds_bucket{model="gpt-4o-mini", status="success", le="2.0"} 789
nephoran_llm_processing_duration_seconds_sum{model="gpt-4o-mini", status="success"} 1,234.56
nephoran_llm_processing_duration_seconds_count{model="gpt-4o-mini", status="success"} 1,247
```

## Controller Metrics

Located in `pkg/controllers/metrics.go`, these metrics provide insight into NetworkIntent controller operations and resource states.

### 📈 Counter Metrics

#### `networkintent_reconciles_total`
**Type**: Counter  
**Description**: Total number of NetworkIntent reconciliation operations  
**Labels**:
- `controller`: Controller name (typically "networkintent")
- `namespace`: Kubernetes namespace of the NetworkIntent
- `name`: Name of the NetworkIntent resource
- `result`: Reconciliation outcome ("success", "error", "requeue")

**Example Values**:
```
networkintent_reconciles_total{controller="networkintent", namespace="default", name="deploy-amf", result="success"} 15
networkintent_reconciles_total{controller="networkintent", namespace="production", name="core-network", result="error"} 3
networkintent_reconciles_total{controller="networkintent", namespace="default", name="edge-deployment", result="requeue"} 7
```

#### `networkintent_reconcile_errors_total`
**Type**: Counter  
**Description**: Total number of NetworkIntent reconciliation errors  
**Labels**:
- `controller`: Controller name
- `namespace`: Kubernetes namespace
- `name`: NetworkIntent resource name
- `error_type`: Error classification ("llm_processing", "validation", "deployment", "timeout", "resource_conflict")

**Example Values**:
```
networkintent_reconcile_errors_total{controller="networkintent", namespace="default", name="deploy-amf", error_type="llm_processing"} 2
networkintent_reconcile_errors_total{controller="networkintent", namespace="production", name="core-network", error_type="validation"} 1
networkintent_reconcile_errors_total{controller="networkintent", namespace="test", name="sample-intent", error_type="timeout"} 4
```

### 📊 Histogram Metrics

#### `networkintent_processing_duration_seconds`
**Type**: Histogram  
**Description**: Duration of NetworkIntent processing phases  
**Labels**:
- `controller`: Controller name
- `namespace`: Kubernetes namespace
- `name`: NetworkIntent resource name
- `phase`: Processing phase ("llm_processing", "validation", "package_generation", "deployment", "total")

**Buckets**: Default Prometheus buckets `[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]` seconds

**Example Values**:
```
networkintent_processing_duration_seconds_bucket{controller="networkintent", namespace="default", name="deploy-amf", phase="llm_processing", le="1.0"} 12
networkintent_processing_duration_seconds_bucket{controller="networkintent", namespace="default", name="deploy-amf", phase="total", le="5.0"} 15
networkintent_processing_duration_seconds_sum{controller="networkintent", namespace="default", name="deploy-amf", phase="total"} 67.8
```

### 📏 Gauge Metrics

#### `networkintent_status`
**Type**: Gauge  
**Description**: Current status of NetworkIntent resources  
**Labels**:
- `controller`: Controller name
- `namespace`: Kubernetes namespace
- `name`: NetworkIntent resource name
- `phase`: Processing phase or component

**Status Values**:
- `0`: Failed state
- `1`: Processing state
- `2`: Ready/Success state

**Example Values**:
```
networkintent_status{controller="networkintent", namespace="default", name="deploy-amf", phase="overall"} 2
networkintent_status{controller="networkintent", namespace="production", name="core-network", phase="llm_processing"} 1
networkintent_status{controller="networkintent", namespace="test", name="failed-intent", phase="overall"} 0
```

## Advanced Features

### Metrics Integration Layer

The `MetricsIntegrator` provides seamless integration between the existing `MetricsCollector` and new Prometheus metrics:

```go
// Example usage in controller
integrator := NewMetricsIntegrator(existingCollector)
integrator.RecordLLMRequest("gpt-4o-mini", "success", 1.2*time.Second, 150)
integrator.RecordCacheOperation("gpt-4o-mini", "lookup", true)
integrator.RecordFallbackAttempt("gpt-4o-mini", "claude-3")
```

### Thread Safety

All metrics operations are thread-safe with proper synchronization:
- Prometheus metrics use atomic operations internally
- Custom metrics include mutex-protected access
- MetricsIntegrator coordinates between systems safely

### Performance Considerations

- **Minimal Overhead**: Metrics collection adds < 1ms latency per operation
- **Memory Efficient**: Prometheus metrics use shared label instances
- **High Cardinality Support**: Designed for 10,000+ unique label combinations
- **Conditional Registration**: Zero overhead when `METRICS_ENABLED=false`

## Common Queries & Alerts

### Essential Prometheus Queries

#### LLM Performance Monitoring

```promql
# Average LLM request duration by model
rate(nephoran_llm_processing_duration_seconds_sum[5m]) / 
rate(nephoran_llm_processing_duration_seconds_count[5m])

# LLM error rate by model
rate(nephoran_llm_errors_total[5m]) / 
rate(nephoran_llm_requests_total[5m]) * 100

# Cache hit rate
rate(nephoran_llm_cache_hits_total[5m]) / 
(rate(nephoran_llm_cache_hits_total[5m]) + rate(nephoran_llm_cache_misses_total[5m])) * 100

# 95th percentile processing time
histogram_quantile(0.95, rate(nephoran_llm_processing_duration_seconds_bucket[5m]))
```

#### Controller Operations

```promql
# NetworkIntent processing success rate
rate(networkintent_reconciles_total{result="success"}[5m]) / 
rate(networkintent_reconciles_total[5m]) * 100

# Average reconciliation duration
rate(networkintent_processing_duration_seconds_sum{phase="total"}[5m]) / 
rate(networkintent_processing_duration_seconds_count{phase="total"}[5m])

# Failed NetworkIntents count
sum by (namespace) (networkintent_status{phase="overall"} == 0)

# Processing NetworkIntents count
sum by (namespace) (networkintent_status{phase="overall"} == 1)
```

#### System Health

```promql
# Top error-prone NetworkIntents
topk(10, rate(networkintent_reconcile_errors_total[1h]))

# Slowest LLM models
topk(5, rate(nephoran_llm_processing_duration_seconds_sum[10m]) / 
         rate(nephoran_llm_processing_duration_seconds_count[10m]))

# Fallback frequency by model
rate(nephoran_llm_fallback_attempts_total[5m])
```

### Recommended Alerting Rules

```yaml
groups:
- name: nephoran_intent_operator
  rules:
  - alert: HighLLMErrorRate
    expr: rate(nephoran_llm_errors_total[5m]) / rate(nephoran_llm_requests_total[5m]) > 0.05
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High LLM error rate detected"
      description: "LLM error rate is {{ $value | humanizePercentage }} for model {{ $labels.model }}"

  - alert: SlowLLMProcessing
    expr: histogram_quantile(0.95, rate(nephoran_llm_processing_duration_seconds_bucket[5m])) > 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Slow LLM processing detected"
      description: "95th percentile LLM processing time is {{ $value }}s"

  - alert: NetworkIntentProcessingFailures
    expr: rate(networkintent_reconcile_errors_total[5m]) > 0.1
    for: 3m
    labels:
      severity: critical
    annotations:
      summary: "NetworkIntent processing failures"
      description: "High rate of NetworkIntent failures: {{ $value }} errors/sec"

  - alert: LowCacheHitRate
    expr: rate(nephoran_llm_cache_hits_total[10m]) / (rate(nephoran_llm_cache_hits_total[10m]) + rate(nephoran_llm_cache_misses_total[10m])) < 0.3
    for: 10m
    labels:
      severity: info
    annotations:
      summary: "Low LLM cache hit rate"
      description: "Cache hit rate is {{ $value | humanizePercentage }}, consider increasing cache size"
```

## Grafana Integration

### Dashboard Creation

The metrics are designed for optimal visualization in Grafana dashboards. Key visualization patterns:

1. **Time Series Panels**: For rate-based metrics and duration trends
2. **Gauge Panels**: For current status values and percentages
3. **Heatmap Panels**: For latency distribution analysis
4. **Table Panels**: For detailed resource status views

### Pre-configured Dashboard

A comprehensive Grafana dashboard JSON is provided in the next section with:
- LLM performance overview
- Controller operations monitoring  
- Error tracking and debugging
- Cache efficiency metrics
- System health indicators

## Troubleshooting

### Common Issues

#### Metrics Not Appearing
1. Verify `METRICS_ENABLED=true` environment variable
2. Check metrics endpoint accessibility: `curl http://localhost:8080/metrics`
3. Confirm Prometheus scrape configuration targets the correct endpoint
4. Review controller logs for metrics registration errors

#### High Cardinality Warnings
1. Monitor label cardinality using `prometheus_tsdb_symbol_table_size_bytes`
2. Limit NetworkIntent names and namespaces in high-volume environments
3. Consider label aggregation for dashboards vs. detailed metrics

#### Performance Impact
1. Metrics collection overhead should be < 1% of total CPU usage
2. Memory usage increases proportionally with unique label combinations
3. Disable metrics in resource-constrained environments

### Debugging Queries

```promql
# Check metrics registration
up{job="nephoran-intent-operator"}

# Verify data freshness
time() - timestamp(nephoran_llm_requests_total)

# Label cardinality analysis
count by (__name__)({__name__=~"nephoran_.*"})
```

## Migration Guide

### From Legacy MetricsCollector

The new Prometheus metrics complement the existing `MetricsCollector` without replacement:

1. **Gradual Migration**: Both systems operate simultaneously
2. **Dual Recording**: `MetricsIntegrator` updates both systems
3. **Prometheus Preference**: New dashboards should use Prometheus metrics
4. **Legacy Support**: Existing integrations continue working unchanged

### Version Compatibility

- **v1.0+**: Full Prometheus metrics support
- **v0.9**: Legacy MetricsCollector only
- **Migration Path**: Update environment variables and scrape configuration

## API Reference

### PrometheusMetrics Methods

```go
func (pm *PrometheusMetrics) RecordRequest(model, status string, duration time.Duration)
func (pm *PrometheusMetrics) RecordError(model, errorType string)
func (pm *PrometheusMetrics) RecordCacheHit(model string)
func (pm *PrometheusMetrics) RecordCacheMiss(model string)
func (pm *PrometheusMetrics) RecordFallbackAttempt(originalModel, fallbackModel string)
func (pm *PrometheusMetrics) RecordRetryAttempt(model string)
```

### ControllerMetrics Methods

```go
func (m *ControllerMetrics) RecordReconcileTotal(namespace, name, result string)
func (m *ControllerMetrics) RecordReconcileError(namespace, name, errorType string)
func (m *ControllerMetrics) RecordProcessingDuration(namespace, name, phase string, duration float64)
func (m *ControllerMetrics) SetStatus(namespace, name, phase string, status float64)
func (m *ControllerMetrics) RecordSuccess(namespace, name string)
func (m *ControllerMetrics) RecordFailure(namespace, name, errorType string)
```

## Best Practices

### Label Design
- Keep label cardinality reasonable (< 10,000 combinations per metric)
- Use meaningful label names that support filtering and aggregation
- Avoid high-cardinality labels like timestamps or UUIDs

### Alert Design
- Set appropriate alert thresholds based on system SLOs
- Include runbook links in alert annotations
- Use tiered severity levels (info, warning, critical)

### Dashboard Design
- Group related metrics into logical dashboard sections
- Use consistent time ranges and refresh intervals
- Include contextual links between related views
- Provide drill-down capabilities for investigation

### Performance Optimization
- Use recording rules for frequently-queried complex expressions
- Implement appropriate retention policies for historical data
- Monitor Prometheus resource usage and scale accordingly
- Consider metric federation for multi-cluster deployments

---

**Next Steps:**
- Review the example [Grafana Dashboard JSON](examples/monitoring/grafana-dashboard.json)
- Set up [Prometheus Alerting Rules](examples/monitoring/prometheus-alerts.yaml)
- Configure [Prometheus Scrape Configuration](examples/monitoring/prometheus-scrape-config.yaml)
- Explore [Production Monitoring Guide](docs/PRODUCTION_MONITORING.md)