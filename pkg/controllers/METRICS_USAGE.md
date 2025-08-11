# Controller-Runtime Metrics Integration

This document describes how to integrate controller-runtime metrics with other controllers in the Nephoran Intent Operator.

## Overview

The `pkg/controllers/metrics.go` module provides a clean, reusable metrics system that:
- Uses controller-runtime's built-in metrics registry
- Respects the `METRICS_ENABLED` environment variable
- Provides standardized metrics for all controllers
- Is safe to use when metrics are disabled

## Available Metrics

- `networkintent_reconciles_total`: Total reconciliation count (labels: controller, namespace, name, result)
- `networkintent_reconcile_errors_total`: Total reconciliation errors (labels: controller, namespace, name, error_type)
- `networkintent_processing_duration_seconds`: Processing duration histograms (labels: controller, namespace, name, phase)
- `networkintent_status`: Current status gauge (labels: controller, namespace, name, phase)

## Usage Example

### 1. Add metrics to your controller struct

```go
type MyControllerReconciler struct {
    client.Client
    Scheme  *runtime.Scheme
    metrics *ControllerMetrics  // Add this field
}
```

### 2. Initialize metrics in constructor

```go
func NewMyControllerReconciler(client client.Client, scheme *runtime.Scheme) *MyControllerReconciler {
    return &MyControllerReconciler{
        Client:  client,
        Scheme:  scheme,
        metrics: NewControllerMetrics("mycontroller"), // Initialize here
    }
}
```

### 3. Record metrics in Reconcile method

```go
func (r *MyControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    startTime := time.Now()
    
    // Your reconciliation logic here
    result, err := r.doReconciliation(ctx, req)
    
    // Record metrics
    if err != nil {
        errorType := "unknown"
        if strings.Contains(err.Error(), "timeout") {
            errorType = "timeout"
        }
        r.metrics.RecordFailure(req.Namespace, req.Name, errorType)
    } else {
        r.metrics.RecordSuccess(req.Namespace, req.Name)
    }
    
    // Record total processing time
    duration := time.Since(startTime).Seconds()
    r.metrics.RecordProcessingDuration(req.Namespace, req.Name, "total", duration)
    
    return result, err
}
```

### 4. Record phase-specific metrics

```go
func (r *MyControllerReconciler) processPhase1(ctx context.Context, obj *MyResource) error {
    phaseStart := time.Now()
    
    // Set status to processing
    r.metrics.SetStatus(obj.Namespace, obj.Name, "phase1", StatusProcessing)
    
    // Do phase 1 work
    err := r.doPhase1Work(ctx, obj)
    
    // Record phase duration
    duration := time.Since(phaseStart).Seconds()
    r.metrics.RecordProcessingDuration(obj.Namespace, obj.Name, "phase1", duration)
    
    if err != nil {
        r.metrics.SetStatus(obj.Namespace, obj.Name, "phase1", StatusFailed)
        return err
    }
    
    r.metrics.SetStatus(obj.Namespace, obj.Name, "phase1", StatusReady)
    return nil
}
```

## Environment Variable

Metrics are controlled by the `METRICS_ENABLED` environment variable:

```bash
export METRICS_ENABLED=true   # Enable metrics
export METRICS_ENABLED=false  # Disable metrics (default)
```

When disabled, all metric calls are no-ops with minimal overhead.

## Status Values

Use the predefined constants for status metrics:

```go
const (
    StatusFailed     float64 = 0  // Operation failed
    StatusProcessing float64 = 1  // Currently processing
    StatusReady      float64 = 2  // Successfully completed
)
```

## Example: CNFDeployment Controller Integration

Here's how you could integrate metrics with the CNFDeployment controller:

```go
// In the struct
type CNFDeploymentReconciler struct {
    client.Client
    Scheme           *runtime.Scheme
    Recorder         record.EventRecorder
    CNFOrchestrator  *cnf.CNFOrchestrator
    LLMProcessor     *llm.Processor
    RAGClient        *rag.Client
    MetricsCollector *monitoring.MetricsCollector
    metrics          *ControllerMetrics  // Add this
}

// In the constructor
func NewCNFDeploymentReconciler(mgr ctrl.Manager, cnfOrchestrator *cnf.CNFOrchestrator) *CNFDeploymentReconciler {
    return &CNFDeploymentReconciler{
        Client:          mgr.GetClient(),
        Scheme:          mgr.GetScheme(),
        Recorder:        mgr.GetEventRecorderFor(CNFDeploymentControllerName),
        CNFOrchestrator: cnfOrchestrator,
        metrics:         NewControllerMetrics("cnfdeployment"), // Add this
    }
}

// In the Reconcile method
func (r *CNFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    startTime := time.Now()
    
    // Existing reconciliation logic...
    result, err := r.reconcileCNFDeployment(ctx, req)
    
    // Record metrics
    if err != nil {
        errorType := "deployment_error"
        if strings.Contains(err.Error(), "llm") {
            errorType = "llm_processing"
        } else if strings.Contains(err.Error(), "resource") {
            errorType = "resource_allocation"
        }
        r.metrics.RecordFailure(req.Namespace, req.Name, errorType)
    } else {
        r.metrics.RecordSuccess(req.Namespace, req.Name)
    }
    
    // Record processing duration
    duration := time.Since(startTime).Seconds()
    r.metrics.RecordProcessingDuration(req.Namespace, req.Name, "total", duration)
    
    return result, err
}
```

## Best Practices

1. **Always initialize metrics in the constructor**: This ensures metrics are available throughout the controller lifecycle.

2. **Use meaningful error types**: Categorize errors for better observability.

3. **Record phase-specific durations**: This helps identify bottlenecks in multi-phase processing.

4. **Set status appropriately**: Use the status gauge to track current state of resources.

5. **Handle metrics gracefully**: The metrics system is designed to be safe when disabled - don't add conditional logic.

6. **Use consistent naming**: Use lowercase, underscore-separated names for phases and error types.

## Testing

The metrics system includes comprehensive tests. You can test your integration:

```go
func TestMyControllerMetrics(t *testing.T) {
    os.Setenv("METRICS_ENABLED", "true")
    defer os.Unsetenv("METRICS_ENABLED")
    
    metrics := NewControllerMetrics("mycontroller")
    
    // Test metric recording
    metrics.RecordSuccess("default", "test")
    metrics.RecordProcessingDuration("default", "test", "phase1", 1.5)
    metrics.SetStatus("default", "test", "ready", StatusReady)
    
    // Metrics should not panic
}
```

This integration provides comprehensive observability for all controllers while maintaining clean separation of concerns and respecting the operator's configuration.