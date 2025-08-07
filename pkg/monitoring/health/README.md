# Enhanced Health Monitoring System

The Enhanced Health Monitoring System provides comprehensive, multi-tiered health assessment that integrates seamlessly with the Nephoran Intent Operator's SLA monitoring framework. This system offers advanced health checking capabilities with weighted scoring, contextual monitoring, predictive analysis, and deep SLA integration.

## Architecture Overview

The enhanced health system consists of five main components:

1. **Enhanced Health Checker** (`enhanced_checker.go`) - Multi-tiered health assessment with parallel execution
2. **Dependency Health Tracker** (`dependency_health.go`) - Comprehensive external dependency monitoring
3. **Health Aggregator** (`health_aggregator.go`) - Multi-dimensional health aggregation and analysis
4. **Health Predictor** (`health_predictor.go`) - Predictive health monitoring and early warning
5. **SLA Integration Bridge** (`integration_bridge.go`) - Seamless integration with SLA monitoring

## Key Features

### Multi-Tiered Health Assessment

The system provides four tiers of health monitoring:

- **System Tier**: Critical infrastructure components (Kubernetes API, core services)
- **Service Tier**: Application services and microservices
- **Component Tier**: Individual application components
- **Dependency Tier**: External dependencies (APIs, databases, storage)

Each tier has configurable weights and specific monitoring strategies optimized for different operational contexts.

### Contextual Health Monitoring

Health checks adapt to different operational contexts:

- **Startup**: Relaxed thresholds during service initialization
- **Steady State**: Normal operational monitoring
- **Shutdown**: Grace period monitoring during graceful shutdown
- **High Load**: Adjusted thresholds for high-traffic periods
- **Maintenance**: Special monitoring during maintenance windows

### Performance Optimization

- **Sub-100ms Execution**: Optimized for fast health check execution
- **Parallel Processing**: Worker pool architecture for concurrent checks
- **Intelligent Caching**: 30-second cache TTL for frequently accessed results
- **Circuit Breakers**: Protection against cascading failures
- **Resource Monitoring**: <10MB additional memory footprint

### Business Impact Weighting

Health components are weighted by business criticality:

- **Critical Weight (1.0)**: Revenue-impacting services
- **High Weight (0.8)**: Important operational services  
- **Normal Weight (0.5)**: Standard application services
- **Low Weight (0.2)**: Optional or development services

### Advanced Analytics

#### Trend Analysis
- Linear regression for health trend prediction
- Volatility detection for unstable services
- Seasonal pattern recognition
- Confidence scoring for trend reliability

#### Anomaly Detection
- Statistical outlier detection (3-sigma rule)
- Baseline model comparison
- Multiple detection algorithms
- Severity classification

#### Predictive Health Monitoring
- ML-based health prediction models
- Early warning system with configurable thresholds
- Resource exhaustion prediction
- Automated recovery recommendations

### Comprehensive Dependency Monitoring

#### External Service Health
- **LLM APIs**: OpenAI, Azure OpenAI, custom models
- **Databases**: PostgreSQL, MongoDB, Redis, vector databases
- **Message Queues**: Kafka, RabbitMQ, AWS SQS
- **Storage Systems**: S3, GCS, Azure Blob, local storage
- **Kubernetes API**: Cluster health and resource availability

#### Circuit Breaker Protection
- Per-dependency circuit breakers
- Configurable failure thresholds
- Automatic recovery detection
- State transition monitoring

#### Service Mesh Integration
- Istio, Linkerd, Consul Connect support
- Request success/error rate tracking
- Latency percentile monitoring (P50, P95, P99)
- Retry and timeout policy compliance

### Multi-Dimensional Aggregation

The system provides multiple perspectives on health:

#### Business Perspective
- Revenue impact weighting
- Customer impact assessment
- Feature availability tracking
- Compliance risk evaluation

#### Technical Perspective
- System performance metrics
- Resource utilization tracking
- Error rate monitoring
- Latency distribution analysis

#### User Experience Perspective
- User journey health tracking
- End-to-end flow monitoring
- Feature functionality verification
- Performance from user viewpoint

### User Journey Monitoring

Track complete user journeys with health correlation:

```go
// Example: Intent Processing Journey
intentJourney := &UserJourney{
    ID:          "intent_processing",
    Name:        "Intent Processing Journey",
    Description: "Complete flow from intent submission to deployment",
    Steps: []JourneyStep{
        {
            ID:           "intent_validation",
            Dependencies: []string{"kubernetes-api"},
            MaxLatency:   time.Second,
            Required:     true,
            Weight:       0.2,
        },
        {
            ID:           "llm_processing", 
            Dependencies: []string{"llm-processor", "rag-api"},
            MaxLatency:   10 * time.Second,
            Required:     true,
            Weight:       0.5,
        },
        // Additional steps...
    },
}
```

### Predictive Health Analysis

#### Machine Learning Models
- **Linear Regression**: Basic trend prediction
- **Moving Average**: Short-term forecasting
- **Exponential Smoothing**: Weighted historical data
- **ARIMA**: Time series analysis (planned)
- **Neural Networks**: Complex pattern recognition (planned)

#### Early Warning System
- Configurable prediction horizons (1-24 hours)
- Multi-level alert thresholds
- Confidence-based alerting
- Automated escalation rules

#### Resource Exhaustion Detection
- Memory usage prediction
- CPU utilization forecasting
- Disk space monitoring
- Network bandwidth tracking
- Connection pool exhaustion

### SLA Integration Bridge

#### Health-SLA Correlation
- Automatic correlation discovery
- Statistical significance testing
- Impact coefficient calculation
- Multi-metric correlation analysis

#### Availability Calculation Methods
- **Health-Weighted**: Incorporates health scores into availability
- **Health-Gated**: Uses health as availability gate
- **Health-Adjusted**: Adjusts availability based on health trends
- **Traditional**: Standard uptime-based calculation

#### Composite Scoring
- Weighted combination of health and SLA metrics
- Business impact consideration
- Multiple scoring algorithms
- Stakeholder-specific views

## Configuration

### Basic Health Check Registration

```go
// Create enhanced health checker
checker := NewEnhancedHealthChecker("my-service", "v1.0", logger)

// Register a business-critical health check
config := &CheckConfig{
    Name:            "llm-processor",
    Tier:            TierService,
    Weight:          WeightCritical,
    Interval:        30 * time.Second,
    Timeout:         5 * time.Second,
    Enabled:         true,
    LatencyThreshold: 2 * time.Second,
}

checkFunc := func(ctx context.Context, context HealthContext) *EnhancedCheck {
    // Implement health check logic
    return &EnhancedCheck{
        Status:  health.StatusHealthy,
        Score:   1.0,
        Message: "Service operational",
    }
}

checker.RegisterEnhancedCheck(config, checkFunc)
```

### Dependency Health Monitoring

```go
// Create dependency health tracker
tracker := NewDependencyHealthTracker("my-service", kubeClient, logger)

// Register external API dependency
depConfig := &DependencyConfig{
    Name:         "openai-api",
    Type:         DepTypeLLMAPI,
    Category:     CatExternal,
    Criticality:  CriticalityEssential,
    Endpoint:     "https://api.openai.com",
    Timeout:      15 * time.Second,
    HealthCheckConfig: HealthCheckConfig{
        Method: "GET",
        Path:   "/v1/engines",
        ExpectedStatus: []int{200},
    },
    CircuitBreakerConfig: CircuitBreakerConfig{
        Enabled:          true,
        MaxRequests:      10,
        Interval:         30 * time.Second,
        FailureThreshold: 0.6,
    },
}

tracker.RegisterDependency(depConfig)
```

### Health Aggregation Setup

```go
// Create health aggregator
aggregator := NewHealthAggregator("my-service", checker, tracker, logger)

// Configure business weights
aggregator.SetBusinessWeight("llm-processor", BusinessWeight{
    Component:        "llm-processor",
    Weight:          1.0,
    RevenueImpact:   0.8,
    UserImpact:      0.9,
    ComplianceImpact: 0.6,
})

// Perform aggregated health check
result, err := aggregator.AggregateHealth(ctx)
if err != nil {
    log.Error("Health aggregation failed", "error", err)
}

fmt.Printf("Overall Health: %s (Score: %.2f)\n", 
    result.OverallStatus, result.OverallScore)
```

### Predictive Health Configuration

```go
// Create health predictor
predictor := NewHealthPredictor("my-service", aggregator, tracker, logger)

// Predict health for the next 4 hours
prediction, err := predictor.PredictHealth(ctx, "llm-processor", 4*time.Hour)
if err != nil {
    log.Error("Health prediction failed", "error", err)
}

// Check for early warnings
for _, warning := range prediction.Warnings {
    if warning.Severity == SeverityCritical {
        log.Warn("Critical health issue predicted", 
            "component", warning.Component,
            "predicted_time", warning.PredictedTime,
            "confidence", warning.Confidence)
    }
}
```

### SLA Integration Setup

```go
// Create SLA integration bridge
bridge := NewSLAIntegrationBridge("my-service", checker, aggregator, predictor, tracker, logger)

// Configure SLA target with health correlation
slaTarget := &SLATarget{
    ID:         "availability",
    Name:       "System Availability",
    MetricType: SLAMetricAvailability,
    Target:     99.95,
    Threshold:  99.9,
    HealthCorrelation: &HealthCorrelation{
        Components:          []string{"llm-processor", "kubernetes-api"},
        CorrelationStrength: 0.8,
        CorrelationType:     CorrelationDirect,
        ImpactFunction:      ImpactLinear,
    },
    HealthImpactWeight: 0.3,
}

// Calculate composite health-SLA score
compositeScore, err := bridge.CalculateCompositeScore(ctx, "balanced")
if err != nil {
    log.Error("Composite score calculation failed", "error", err)
}

fmt.Printf("Composite Score: %.2f (Health: %.2f, SLA: %.2f)\n",
    compositeScore.OverallScore,
    compositeScore.HealthScore,
    compositeScore.SLAScore)
```

## Metrics and Observability

The system provides comprehensive Prometheus metrics:

### Enhanced Health Checker Metrics
- `enhanced_health_check_duration_seconds` - Health check execution time
- `enhanced_health_check_success_total` - Successful health checks
- `enhanced_health_score` - Weighted health scores by tier
- `enhanced_health_state_transitions_total` - Health state changes

### Dependency Health Metrics
- `dependency_health_status` - Dependency health status
- `dependency_response_time_seconds` - Dependency response times
- `dependency_availability_rate` - Dependency availability
- `dependency_circuit_breaker_state` - Circuit breaker states

### Predictive Health Metrics
- `health_prediction_accuracy` - Model prediction accuracy
- `early_warnings_generated_total` - Early warnings by type
- `anomalies_detected_total` - Detected anomalies
- `model_performance_metrics` - ML model performance

### SLA Integration Metrics
- `health_sla_correlation_coefficient` - Health-SLA correlations
- `sla_compliance_status` - SLA compliance by target
- `composite_health_sla_score` - Composite scores
- `integrated_health_sla_alerts_total` - Integrated alerts

## Integration with Existing Systems

### Kubernetes Integration
The system integrates with existing Kubernetes health checks:

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: my-service
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /readyz
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

### Prometheus Integration
Metrics are automatically registered and exposed:

```yaml
# prometheus.yml
scrape_configs:
- job_name: 'nephoran-health'
  static_configs:
  - targets: ['localhost:8080']
  metrics_path: '/metrics'
  scrape_interval: 15s
```

### Grafana Dashboards
Pre-built dashboard templates are available for:
- Overall health overview
- Dependency health monitoring
- Predictive health trends
- SLA compliance tracking
- Business impact assessment

## Best Practices

### Health Check Implementation
1. **Keep checks fast**: Target <100ms execution time
2. **Implement circuit breakers**: Protect against downstream failures
3. **Use appropriate timeouts**: Balance accuracy vs performance
4. **Weight by business impact**: Prioritize critical components
5. **Monitor dependencies**: Include all external services

### Performance Optimization
1. **Enable caching**: Use intelligent caching for frequent checks
2. **Parallel execution**: Leverage worker pools for concurrent checks
3. **Resource monitoring**: Track memory and CPU usage
4. **Batch operations**: Group related health checks
5. **Graceful degradation**: Maintain core functionality during partial failures

### Alerting Strategy
1. **Multi-level thresholds**: Warning, critical, emergency levels
2. **Contextual alerting**: Different rules for different contexts
3. **Correlation-based alerts**: Combine health and SLA conditions
4. **Escalation policies**: Automatic escalation for unresolved issues
5. **Alert suppression**: Prevent alert storms during incidents

## Troubleshooting

### Common Issues

#### Health Checks Timing Out
```go
// Increase timeout or optimize check logic
config.Timeout = 10 * time.Second

// Add circuit breaker protection
config.CircuitBreakerConfig.Enabled = true
```

#### High Memory Usage
```go
// Enable garbage collection for historical data
config.HistoryMaxPoints = 50  // Reduce from default 100

// Adjust cache settings
config.CacheExpiry = 10 * time.Second  // Reduce from 30s
```

#### Prediction Accuracy Issues
```go
// Increase training data size
config.MinTrainingDataSize = 100  // Increase from 50

// Adjust model algorithm
model.Algorithm = AlgorithmExponentialSmoothing  // Try different algorithm
```

#### SLA Correlation Problems
```go
// Verify component mapping
correlation.Components = []string{"actual-service-name"}

// Check correlation strength
correlation.CorrelationStrength = 0.6  // Reduce if too aggressive
```

### Debug Logging
Enable debug logging for detailed troubleshooting:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

checker := NewEnhancedHealthChecker("service", "v1.0", logger)
```

### Performance Monitoring
Monitor system performance with built-in metrics:

```bash
# Check health check latency
curl http://localhost:8080/metrics | grep enhanced_health_check_duration

# Monitor memory usage
curl http://localhost:8080/metrics | grep process_resident_memory_bytes

# Check prediction accuracy
curl http://localhost:8080/metrics | grep health_prediction_accuracy
```

## Future Enhancements

### Planned Features
- **Advanced ML Models**: Neural networks, ensemble methods
- **Service Mesh Deep Integration**: Envoy proxy metrics, traffic analysis
- **Multi-Cloud Health**: Cross-cloud dependency monitoring
- **AI-Powered Root Cause Analysis**: Automated incident analysis
- **Custom Health DSL**: Domain-specific language for health rules

### Roadmap
- **Q1 2024**: Advanced ML models and improved prediction accuracy
- **Q2 2024**: Enhanced service mesh integration and traffic analysis
- **Q3 2024**: Multi-cloud health monitoring and edge deployments
- **Q4 2024**: AI-powered root cause analysis and automated remediation

## Contributing

The enhanced health monitoring system is designed to be extensible. To contribute:

1. **Add New Health Check Types**: Implement new dependency types
2. **Enhance Prediction Models**: Add new ML algorithms
3. **Improve Aggregation**: Add new aggregation methods
4. **Extend SLA Integration**: Add new correlation types
5. **Optimize Performance**: Improve execution speed and memory usage

See the main project CONTRIBUTING.md for detailed contribution guidelines.

## License

This component is part of the Nephoran Intent Operator and is licensed under the same terms as the main project.