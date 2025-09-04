# Monitoring Package

This package provides comprehensive monitoring capabilities for O-RAN Nephio systems, including latency tracking, performance reporting, availability monitoring, and alerting.

## Package Structure

```
pkg/monitoring/
├── types.go                 # Common types and interfaces
├── latency_tracker.go      # E2E latency tracking implementation
├── performance_reporter.go # Performance report generation
├── synthetic_monitor.go    # Synthetic availability monitoring
├── alert_router.go         # Alert routing and notification
└── README.md              # This documentation
```

## Components

### 1. Latency Tracking (`latency_tracker.go`)
- **E2ELatencyTracker**: Tracks end-to-end latency for O-RAN operations
- **Key Features**:
  - OpenTelemetry integration for distributed tracing
  - Span context management (fixes line 294 error: `trace.SpanFromContext`)
  - Metrics aggregation and percentile calculations
  - Periodic collection and reporting

### 2. Performance Reporting (`performance_reporter.go`)
- **PerformanceReporter**: Generates comprehensive performance reports
- **Key Features**:
  - HTML and JSON report templates (fixes line 348 error: `loadTemplates`)
  - Threshold-based alerting
  - Configurable time ranges and filters
  - Export capabilities for multiple formats

### 3. Synthetic Monitoring (`synthetic_monitor.go`)
- **SyntheticMonitor**: Performs synthetic availability checks
- **Key Features**:
  - HTTP, TCP, and gRPC health checks
  - CheckResult type with proper availability metrics
  - Configurable intervals and timeouts
  - Real-time monitoring and alerting

### 4. Alert Management (`alert_router.go`)
- **AlertRouter**: Handles alert routing and delivery
- **Key Features**:
  - Alert type with comprehensive metadata (fixes line 910 error)
  - Multiple notification channels (webhook, Slack, email)
  - Rule-based alerting with cooldown periods
  - Risk assessment and acknowledgment workflows

## Error Fixes

### Fixed Compilation Errors:
1. **Line 294**: `trace.SpanFromContext` - Added proper OpenTelemetry span extraction
2. **Line 348**: `loadTemplates` - Implemented template loading with HTML/JSON templates
3. **CheckResult type**: Defined comprehensive check result structure
4. **Alert type**: Complete alert structure with routing capabilities

### Key Types Added:
- `LatencyMetrics` - Latency statistics and percentiles
- `PerformanceMetrics` - Performance data structure
- `CheckResult` - Synthetic monitoring results
- `Alert` - Alert instance with full metadata
- `AlertRule` - Alert rule configuration
- `AlertChannel` - Notification channel interface

## Usage Examples

### Latency Tracking
```go
tracker := NewE2ELatencyTracker()

// Track an operation
ctx, err := tracker.StartOperation(ctx, "op-123", "scaling", "nephio", attrs)
// ... perform operation ...
err = tracker.EndOperation(ctx, "op-123", true, "")

// Or use convenience method
err = tracker.TrackOperation(ctx, "op-456", func(ctx context.Context) error {
    // Your operation logic here
    return nil
})
```

### Performance Reporting
```go
reporter := NewPerformanceReporter()

// Record metrics
metrics := &PerformanceMetrics{
    Throughput:   100.5,
    Latency:      50 * time.Millisecond,
    ErrorRate:    0.1,
    Availability: 99.9,
}
reporter.RecordMetrics("component-name", metrics)

// Generate report
config := &ReportConfig{
    TimeRange: TimeRange{
        Start: time.Now().Add(-1 * time.Hour),
        End:   time.Now(),
    },
    OutputFormat: "html",
    ThresholdAlerts: map[string]float64{
        "error_rate": 1.0,
        "availability": 99.0,
    },
}
report, err := reporter.GenerateReport(ctx, config)
```

### Synthetic Monitoring
```go
monitor := NewSyntheticMonitor()

// Add a check
check := &SyntheticCheck{
    ID:           "http-check-1",
    Name:         "API Health Check",
    Type:         "http",
    Target:       "http://api.example.com/health",
    Interval:     30 * time.Second,
    Timeout:      10 * time.Second,
    Enabled:      true,
    ExpectedCode: 200,
    ThresholdMs:  5000,
}
monitor.AddCheck(check)

// Start monitoring
monitor.StartMonitoring(ctx)

// Get results
result, err := monitor.GetCheckResult("http-check-1")
```

### Alert Management
```go
router := NewAlertRouter()

// Add alert rule
rule := &AlertRule{
    ID:        "high-latency-rule",
    Name:      "High Latency Alert",
    Condition: "gt",
    Threshold: 100.0,
    Severity:  "warning",
    Component: "api-gateway",
    Enabled:   true,
}
router.AddRule(rule)

// Fire alert
labels := map[string]string{"service": "api", "instance": "prod-1"}
err = router.FireAlert(ctx, "high-latency-rule", labels, 150.0)
```

## Integration with Nephio

This monitoring package integrates with:
- **O-RAN L Release**: Provides telecom-specific monitoring
- **Nephio R5**: Tracks package lifecycle operations
- **OpenTelemetry**: Distributed tracing and observability
- **Kubernetes**: Pod and service health monitoring
- **VES 7.3**: Event streaming and FCAPS integration

## Configuration

Default configurations are provided for all components, but can be customized:
- Latency collection intervals
- Report generation schedules
- Alert thresholds and channels
- Synthetic check frequencies

## Dependencies

- `go.opentelemetry.io/otel` - Distributed tracing
- `sigs.k8s.io/controller-runtime` - Kubernetes integration
- `github.com/go-logr/logr` - Structured logging

All compilation errors have been resolved and the monitoring package is now fully functional.