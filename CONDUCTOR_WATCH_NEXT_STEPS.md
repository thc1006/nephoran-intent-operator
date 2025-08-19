# Conductor-Watch Next Steps

## Current Implementation (MVP) ✅

The conductor-watch controller successfully:
- Watches NetworkIntent CRs across configured namespaces
- Parses intent strings to structured JSON
- Generates intent files matching `docs/contracts/intent.schema.json`
- Executes porch CLI for KRM generation
- Provides comprehensive logging without status updates

## Follow-up PR Items

### 1. Status Updates
- [ ] Add status updates to NetworkIntent CR after processing
- [ ] Track processing state: `Pending`, `Processing`, `Completed`, `Failed`
- [ ] Include last processed timestamp
- [ ] Add error messages for failed intents
- [ ] Update `ObservedGeneration` to track spec changes

```go
// Example status update
networkIntent.Status.Phase = "Completed"
networkIntent.Status.ObservedGeneration = networkIntent.Generation
networkIntent.Status.LastUpdateTime = metav1.Now()
networkIntent.Status.LastMessage = "Successfully generated KRM"
r.Status().Update(ctx, &networkIntent)
```

### 2. Deduplication Logic
- [ ] Track processed intents by Generation/ResourceVersion
- [ ] Skip reprocessing if spec hasn't changed
- [ ] Implement hash-based change detection
- [ ] Add annotation for last processed hash

```go
// Example deduplication
lastHash := networkIntent.Annotations["conductor-watch/last-hash"]
currentHash := hashIntent(networkIntent.Spec)
if lastHash == currentHash {
    return ctrl.Result{}, nil // Skip duplicate
}
```

### 3. Exponential Backoff
- [ ] Implement proper exponential backoff for retries
- [ ] Track retry count in annotations
- [ ] Increase delay between retries: 30s, 1m, 2m, 5m, 10m
- [ ] Max retry limit before giving up

```go
// Example backoff
retryCount := getRetryCount(networkIntent)
backoffDuration := time.Duration(math.Pow(2, float64(retryCount))) * 30 * time.Second
if backoffDuration > 10*time.Minute {
    backoffDuration = 10*time.Minute
}
return ctrl.Result{RequeueAfter: backoffDuration}, nil
```

### 4. Enhanced Porch Integration
- [ ] Support real porch/kpt commands (not just echo)
- [ ] Parse porch output for better error reporting
- [ ] Support different porch backends (kpt, direct API)
- [ ] Add porch health checks

### 5. Metrics and Observability
- [ ] Add Prometheus metrics for:
  - Total intents processed
  - Success/failure rates
  - Processing duration
  - Porch execution time
- [ ] Add structured logging with trace IDs
- [ ] OpenTelemetry tracing support

### 6. Advanced Intent Parsing
- [ ] Support more intent types beyond scaling:
  - Network policies
  - Resource limits
  - Affinity rules
  - Service mesh configuration
- [ ] Natural language processing improvements
- [ ] Intent validation against business rules

### 7. Production Readiness
- [ ] Leader election for HA deployments
- [ ] Graceful shutdown with intent draining
- [ ] Resource limits and requests
- [ ] Security context and RBAC refinement
- [ ] Webhook for intent validation

### 8. Testing Enhancements
- [ ] Integration tests with real Kubernetes cluster
- [ ] E2E tests with actual porch CLI
- [ ] Performance benchmarks
- [ ] Chaos testing for resilience

## Implementation Priority

1. **Phase 1**: Status updates + Deduplication (Critical for production)
2. **Phase 2**: Exponential backoff + Enhanced error handling
3. **Phase 3**: Metrics and observability
4. **Phase 4**: Advanced features and optimizations

## Breaking Changes to Consider

- Moving from log-only to status updates requires RBAC changes
- Deduplication may change reconciliation behavior
- Status schema additions need CRD updates

## Migration Path

1. Deploy with feature flags for new behavior
2. Run in parallel with existing system
3. Gradually enable features per namespace
4. Full rollout after validation

## Dependencies

- Requires Porch/KPT CLI in container image
- NetworkIntent CRD v1 or later
- Kubernetes 1.26+ for certain controller-runtime features
- Proper RBAC for status updates

## Security Considerations

- Validate intent strings against injection attacks
- Restrict porch CLI execution permissions
- Audit log all intent processing
- Rate limiting for reconciliation

## Performance Targets

- P50 latency: < 1s for intent processing
- P99 latency: < 5s including porch execution
- Throughput: 100+ intents/minute
- Memory: < 100MB for controller pod