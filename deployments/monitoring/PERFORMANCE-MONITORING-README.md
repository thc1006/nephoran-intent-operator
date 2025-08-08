# Nephoran Intent Operator - Performance Monitoring & Benchmarking

## Overview

This comprehensive performance monitoring system provides real-time validation of all claimed performance metrics for the Nephoran Intent Operator. The system integrates advanced benchmarking, statistical validation, and regression detection to ensure continuous performance assurance.

## Architecture Components

### 1. Performance Benchmarking Framework
- **Intent Processing Benchmarks**: Validates sub-2-second P95 latency claims
- **Statistical Validator**: Provides 95% confidence intervals and significance testing
- **Distributed Load Tester**: Tests realistic telecom workloads with 200+ concurrent users
- **Regression Detector**: Automated detection of performance degradations
- **Enhanced Profiler**: Leverages Go 1.24+ features for detailed performance analysis

### 2. Monitoring Stack
- **Prometheus**: High-frequency metrics collection (1s intervals for benchmarks)
- **Grafana**: Real-time dashboards with performance claims validation
- **Alerting**: Comprehensive alerts for SLA violations and regressions

### 3. Performance Claims Validated

| Claim | Target | Monitoring |
|-------|--------|------------|
| Intent Processing P95 Latency | ≤2 seconds | `benchmark:intent_processing_latency_p95` |
| Concurrent User Capacity | 200+ users | `benchmark_concurrent_users_current` |
| Throughput | 45 intents/minute | `benchmark:intent_processing_rate_1m` |
| Service Availability | 99.95% | `benchmark:availability_5m` |
| RAG Retrieval P95 Latency | ≤200ms | `benchmark:rag_latency_p95` |
| Cache Hit Rate | 87% | `benchmark:cache_hit_rate_5m` |

## Deployment

### Quick Start

```bash
# Deploy the complete monitoring stack
./scripts/deploy-performance-monitoring.sh deploy

# Access URLs (replace localhost with your cluster IP)
# Prometheus: http://localhost:30090
# Grafana: http://localhost:30030 (admin/nephoran-benchmarking-2024)
```

### Manual Deployment

```bash
# Create namespace
kubectl create namespace nephoran-monitoring

# Deploy Prometheus with benchmarking configuration
kubectl apply -f deployments/monitoring/performance-benchmarking-prometheus.yaml

# Deploy Grafana with performance dashboards
kubectl apply -f deployments/monitoring/performance-benchmarking-grafana.yaml

# Port-forward for local access
kubectl port-forward -n nephoran-monitoring svc/performance-benchmarking-grafana 3000:3000
kubectl port-forward -n nephoran-monitoring svc/performance-benchmarking-prometheus 9090:9090
```

## Dashboard Overview

### 1. Performance Benchmarking - Real-time Validation
**URL**: `/d/nephoran-performance-benchmarking`

**Key Features**:
- Real-time gauge displays for all 6 performance claims
- Overall performance score (0-100%) aggregation
- Historical trend analysis
- Live statistical validation results
- Benchmark execution status
- Performance regression detection
- Load testing scenario results

**Panels**:
- **CLAIM 1**: Intent Processing P95 Latency (≤2s)
- **CLAIM 2**: Concurrent User Capacity (200+)
- **CLAIM 3**: Throughput Target (45/min)
- **CLAIM 4**: Service Availability SLA (99.95%)
- **CLAIM 5**: RAG Retrieval P95 Latency (≤200ms)
- **CLAIM 6**: Cache Hit Rate Target (87%)

### 2. Statistical Validation Dashboard
**URL**: `/d/nephoran-statistical-validation`

**Key Features**:
- Statistical confidence levels (target: 95%+)
- P-values for significance testing
- Effect sizes (Cohen's d) with magnitude classification
- Confidence intervals visualization
- Sample size adequacy validation

### 3. Performance Regression Analysis
**URL**: `/d/nephoran-regression-analysis`

**Key Features**:
- Performance stability score (0-100%)
- Regression detection alerts
- Baseline deviation analysis
- 24-hour performance change percentages
- Trend analysis across multiple time windows

## Metrics Reference

### Core Performance Metrics

```promql
# Intent processing latency (P50, P95, P99)
benchmark:intent_processing_latency_p95
benchmark:intent_processing_latency_p50
histogram_quantile(0.99, rate(nephoran_intent_processing_duration_seconds_bucket[5m]))

# Throughput metrics
benchmark:intent_processing_rate_1m    # Intents per minute
benchmark:intent_processing_rate_5m    # 5-minute average

# Availability calculation
benchmark:availability_5m = (total_requests - failed_requests) / total_requests * 100

# Cache performance
benchmark:cache_hit_rate_5m = cache_hits / (cache_hits + cache_misses) * 100

# RAG system performance
benchmark:rag_latency_p95              # RAG retrieval P95 latency
```

### Statistical Validation Metrics

```promql
# Confidence intervals
benchmark:intent_latency_confidence_interval_lower
benchmark:intent_latency_confidence_interval_upper

# Statistical tests
benchmark:intent_latency_significance_test  # t-test results
benchmark:normality_test_passed             # Shapiro-Wilk test
benchmark:sample_size_adequate              # Power analysis

# Effect size analysis
benchmark_effect_sizes                      # Cohen's d values
benchmark:statistical_power                 # Statistical power
```

### Regression Detection Metrics

```promql
# Performance changes
benchmark:intent_latency_24h_change_percent
benchmark:performance_stability_score

# Regression alerts
benchmark_regression_detected               # 0 or 1
benchmark:baseline_deviation_score         # Standard deviations from baseline
```

## Alerting Rules

### Critical Alerts

| Alert | Condition | Severity |
|-------|-----------|----------|
| `IntentProcessingLatencySLAViolation` | P95 > 2.0s | Critical |
| `AvailabilitySLAViolation` | < 99.95% | Critical |
| `OverallPerformanceClaimsFailure` | Score < 90% | Critical |
| `PerformanceRegressionDetected` | Regression flag = 1 | Critical |

### Warning Alerts

| Alert | Condition | Severity |
|-------|-----------|----------|
| `ConcurrentUserCapacityExceeded` | Users > 200 | Warning |
| `ThroughputBelowClaimedTarget` | < 45 intents/min | Warning |
| `RAGLatencyClaimViolation` | P95 > 200ms | Warning |
| `CacheHitRateClaimViolation` | < 87% | Warning |

### Statistical Alerts

| Alert | Condition | Severity |
|-------|-----------|----------|
| `StatisticalValidationLowConfidence` | < 95% confidence | Warning |
| `InsufficientSampleSize` | Sample size inadequate | Warning |
| `PerformanceInstabilityDetected` | Stability < 80% | Warning |

## Integration with Performance Test Framework

The monitoring system integrates with the comprehensive performance test framework:

### Test Components Integration

```bash
# Performance test components expose metrics on these ports:
performance-test-runner:8090       # Main benchmarking orchestrator
intent-processor-benchmark:8091    # Intent processing benchmarks
statistical-validator:8092         # Statistical validation service
distributed-load-tester:8093       # Load testing framework
regression-detector:8094           # Regression detection service
enhanced-profiler:8095             # Go profiler with pprof
validation-suite:8096              # Comprehensive validation suite
```

### Benchmark Execution Flow

1. **Benchmarking Orchestrator** runs comprehensive test suites
2. **Statistical Validator** analyzes results with 95% confidence
3. **Metrics Export** to Prometheus for real-time monitoring
4. **Dashboard Visualization** of all performance claims
5. **Alert Generation** on SLA violations or regressions
6. **Regression Detection** for continuous quality assurance

## Operational Procedures

### Daily Operations

1. **Morning Performance Review**:
   - Check overall performance score (target: >90%)
   - Review overnight regression alerts
   - Validate all 6 performance claims

2. **Performance Trend Analysis**:
   - Weekly stability score trending
   - Monthly performance improvement tracking
   - Quarterly baseline adjustment

3. **Alert Response**:
   - Critical alerts: Immediate response (<15 min)
   - Warning alerts: Response within 1 hour
   - Statistical alerts: Review within 4 hours

### Troubleshooting

#### Performance Degradation

1. Check **Performance Regression Analysis** dashboard
2. Identify affected components via **Performance Claims Historical Trend**
3. Review **Load Testing Scenario Results** for failure patterns
4. Examine **Statistical Validation Results** for significance

#### Alert Fatigue Mitigation

1. Adjust thresholds based on **Statistical Validation** confidence intervals
2. Implement alert suppression during maintenance windows
3. Use **Performance Stability Score** for alert prioritization

### Maintenance

#### Monthly Tasks

- Review and adjust performance baselines
- Update statistical validation parameters
- Performance test framework version updates
- Dashboard optimization based on usage patterns

#### Quarterly Tasks

- Comprehensive performance regression analysis
- Stakeholder reporting with executive dashboards
- Performance improvement roadmap updates
- Monitoring infrastructure scaling review

## Performance Validation Evidence

The system provides quantifiable evidence for all performance claims through:

1. **Statistical Rigor**: 95% confidence intervals, p-values, effect sizes
2. **Continuous Monitoring**: Real-time validation with 1-second granularity
3. **Regression Protection**: Automated detection with 24-hour baselines
4. **Load Testing**: Realistic telecom workload scenarios
5. **Comprehensive Coverage**: All 6 performance claims monitored simultaneously

## Configuration

### Prometheus Configuration

```yaml
# High-frequency scraping for benchmarks
scrape_interval: 1s
evaluation_interval: 5s

# Performance-specific job configurations
- job_name: 'performance-benchmarking'
  scrape_interval: 1s  # Real-time benchmarking
  metrics_path: '/metrics'
```

### Grafana Configuration

```yaml
# Datasource configuration
datasources:
- name: Prometheus-Benchmarking
  type: prometheus
  url: http://performance-benchmarking-prometheus:9090
  isDefault: true
  jsonData:
    timeInterval: 1s  # High-resolution queries
```

### Alert Configuration

```yaml
# Performance claims alert groups
groups:
- name: performance-claims-violations
  rules:
  - alert: IntentProcessingLatencySLAViolation
    expr: benchmark:intent_processing_latency_p95 > 2.0
    for: 30s  # Quick detection
    labels:
      severity: critical
```

## Support and Maintenance

### Monitoring Health

- **System Health**: Monitor monitoring system itself
- **Data Quality**: Ensure metrics collection continuity
- **Storage Management**: Prometheus retention and cleanup
- **Performance Impact**: Monitor overhead of monitoring system

### Scaling Considerations

- **Metrics Volume**: ~500 metrics per second during benchmarking
- **Storage Requirements**: ~10GB for 7-day retention
- **Query Performance**: Optimized for sub-second dashboard loading
- **Concurrent Users**: Supports 50+ simultaneous dashboard users

For additional support, refer to the operational runbooks in `/docs/operations/` or contact the Nephoran performance engineering team.