# Comprehensive Regression Testing Framework

The Nephoran Intent Operator includes a sophisticated regression testing framework that prevents quality degradation by automatically detecting performance, functional, security, and production readiness regressions. This framework integrates seamlessly with the existing validation suite and provides comprehensive monitoring of system quality over time.

## Overview

The regression testing framework provides:

- **Automated Baseline Management**: Establishes and maintains quality baselines
- **Multi-Category Regression Detection**: Monitors performance, functional, security, and production metrics
- **Statistical Trend Analysis**: Identifies patterns and predicts quality trajectories  
- **Intelligent Alerting**: Sends targeted notifications for detected regressions
- **Comprehensive Reporting**: Generates dashboards, metrics, and historical analysis
- **CI/CD Integration**: Provides fail-fast capabilities for continuous deployment pipelines

## Architecture

### Core Components

1. **RegressionFramework**: Main orchestration component
2. **BaselineManager**: Handles baseline creation, storage, and versioning
3. **RegressionDetectionEngine**: Analyzes results against baselines to detect regressions
4. **TrendAnalyzer**: Performs statistical analysis and trend prediction
5. **AlertSystem**: Manages notifications and alerts
6. **RegressionDashboard**: Generates reports and visualizations

### Integration Points

- **ValidationSuite**: Leverages existing 90/100 point validation framework
- **CI/CD Pipelines**: Provides automated quality gates
- **Monitoring Systems**: Exports metrics in Prometheus format
- **Alerting Systems**: Integrates with Slack, email, and webhooks

## Configuration

### Regression Thresholds

```go
type RegressionConfig struct {
    // Regression detection thresholds
    PerformanceThreshold  float64 // > 10% degradation
    FunctionalThreshold   float64 // > 5% decrease in pass rates  
    SecurityThreshold     float64 // Any new vulnerabilities
    ProductionThreshold   float64 // Any availability violations
    OverallScoreThreshold float64 // < 5% decrease in total score
    
    // Baseline management
    BaselineStoragePath   string
    AutoBaselineUpdate    bool
    BaselineRetention     int // Number of baselines to keep
    
    // Alert configuration
    EnableAlerting        bool
    AlertWebhookURL       string
    AlertSlackChannel     string
    AlertEmailRecipients  []string
}
```

### Environment Variables

```bash
# Regression Testing Configuration
REGRESSION_BASELINE_ID=""              # Specific baseline ID to compare against
REGRESSION_FAIL_ON_DETECTION="true"    # Fail tests when regressions detected  
REGRESSION_ALERT_WEBHOOK=""            # Webhook URL for alerts
TEST_ARTIFACTS_PATH="./regression-artifacts" # Output path for reports
```

## Usage

### Quick Start

1. **Setup Environment**:
   ```bash
   make regression-setup
   ```

2. **Establish Baseline**:
   ```bash
   make regression-baseline
   ```

3. **Run Regression Tests**:
   ```bash
   make regression-test
   ```

4. **Generate Dashboard**:
   ```bash
   make regression-dashboard
   ```

### Detailed Workflow

#### 1. Baseline Creation

When no baseline exists, establish one:

```bash
# Create initial baseline
make regression-baseline

# The system will:
# - Run comprehensive validation (targeting 90/100 points)
# - Create baseline snapshot with all metrics
# - Store baseline with versioning information
# - Generate baseline statistics
```

#### 2. Regression Testing

Run regression detection against existing baseline:

```bash
# Run regression testing
make regression-test

# Or specify baseline ID
REGRESSION_BASELINE_ID="2024-01-15T10-30-45-a1b2c3d4" make regression-test
```

The system will:
- Load specified or latest baseline
- Execute current validation suite  
- Compare results against baseline using statistical analysis
- Detect regressions across all categories
- Generate comprehensive regression report
- Trigger alerts if regressions detected

#### 3. Trend Analysis

Generate trend analysis from historical data:

```bash
# Analyze trends over last 30 days
make regression-trends

# The system will:
# - Load all baselines from last 30 days
# - Perform statistical trend analysis
# - Identify patterns and anomalies
# - Generate predictions
# - Export trend visualizations
```

#### 4. Dashboard Generation

Create comprehensive regression dashboard:

```bash
# Generate dashboard with all visualizations
make regression-dashboard

# Outputs:
# - HTML dashboard with interactive charts
# - JSON data for API consumption
# - Prometheus metrics export
# - CSV files for analysis
```

### CI/CD Integration

#### GitHub Actions Example

```yaml
name: Regression Testing
on:
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 6 * * *' # Daily at 6 AM UTC

jobs:
  regression-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.24.1
          
      - name: Setup Regression Testing
        run: make regression-setup
        
      - name: Run Regression Tests
        run: make regression-ci
        env:
          REGRESSION_BASELINE_ID: ${{ secrets.REGRESSION_BASELINE_ID }}
          REGRESSION_ALERT_WEBHOOK: ${{ secrets.SLACK_WEBHOOK_URL }}
          
      - name: Upload Artifacts
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: regression-reports
          path: regression-artifacts/
```

#### Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    
    environment {
        REGRESSION_BASELINE_ID = credentials('regression-baseline-id')
        REGRESSION_ALERT_WEBHOOK = credentials('slack-webhook-url')
    }
    
    stages {
        stage('Setup') {
            steps {
                sh 'make regression-setup'
            }
        }
        
        stage('Regression Test') {
            steps {
                sh 'make regression-ci'
            }
            post {
                always {
                    archiveArtifacts artifacts: 'regression-artifacts/**/*'
                    publishHTML([
                        allowMissing: false,
                        alwaysLinkToLastBuild: true,
                        keepAll: true,
                        reportDir: 'regression-artifacts/dashboard',
                        reportFiles: 'regression-dashboard.html',
                        reportName: 'Regression Dashboard'
                    ])
                }
            }
        }
    }
}
```

## Regression Detection Categories

### Performance Regressions

Monitors key performance indicators:

- **P95 Latency**: > 10% increase triggers regression
- **P99 Latency**: > 10% increase triggers regression  
- **Throughput**: > 10% decrease triggers regression
- **Availability**: Any decrease below 99.95% triggers regression

**Example Detection**:
```json
{
  "metric_name": "P95 Latency",
  "baseline_value": 1500000000,
  "current_value": 1800000000,
  "degradation_percent": 20.0,
  "severity": "medium",
  "impact": "Latency increased by 20.0% (1.5s → 1.8s), potentially affecting user experience",
  "recommendation": "Profile application performance and optimize critical paths"
}
```

### Functional Regressions

Tracks test pass rates and functionality:

- **Overall Score**: > 5% decrease in functional points
- **Category-Specific**: Individual test category degradations
- **New Failures**: Tests that previously passed but now fail
- **Test Stability**: Consistency of test results over time

**Example Detection**:
```json
{
  "test_category": "intent-processing",
  "baseline_pass_rate": 95.0,
  "current_pass_rate": 87.5,
  "failed_tests": ["TestIntentValidation", "TestLLMIntegration"],
  "new_failures": ["TestIntentValidation"],
  "severity": "high",
  "impact": "intent-processing functionality degraded by 7.5%"
}
```

### Security Regressions

Monitors security posture:

- **New Vulnerabilities**: Any new security findings
- **Severity Increases**: Existing vulnerabilities becoming more severe
- **Compliance Failures**: New compliance violations
- **Security Score**: Decrease in overall security rating

**Example Detection**:
```json
{
  "finding": {
    "type": "Container Vulnerability",
    "severity": "high",
    "description": "CVE-2024-1234: Remote code execution vulnerability"
  },
  "is_new_vulnerability": true,
  "impact": "New high security vulnerability detected",
  "recommendation": "Update container base image and rebuild"
}
```

### Production Readiness Regressions

Tracks deployment confidence factors:

- **Availability Targets**: SLA compliance monitoring
- **Reliability Metrics**: MTBF and MTTR tracking
- **Monitoring Coverage**: Observability completeness
- **Disaster Recovery**: Recovery capability assessment

## Alert System

### Alert Types

1. **Regression Alerts**: Triggered when regressions detected
2. **Threshold Alerts**: When metrics exceed defined thresholds
3. **Anomaly Alerts**: For unusual patterns or outliers
4. **Trend Alerts**: Early warnings for degrading trends

### Alert Channels

#### Slack Integration

```json
{
  "channel": "#quality-alerts",
  "username": "Nephoran Quality Bot",
  "text": "<!channel> Quality Regression Detected (HIGH)",
  "attachments": [{
    "color": "#FF6600",
    "title": "Performance Regression: P95 Latency",
    "text": "Latency increased by 25.0% (1.5s → 1.875s)",
    "fields": [
      {"title": "Severity", "value": "HIGH", "short": true},
      {"title": "Impact", "value": "User experience degradation", "short": true}
    ],
    "actions": [{
      "type": "button",
      "text": "View Dashboard",
      "url": "https://dashboard.example.com/regression"
    }]
  }]
}
```

#### Email Alerts

Detailed email reports with:
- Executive summary of regressions
- Detailed breakdown by category
- Recommended actions with priorities
- Links to dashboards and runbooks

#### Webhook Integration

Generic webhook payload for custom integrations:
```json
{
  "alert_type": "regression",
  "severity": "high",
  "system": "Nephoran Intent Operator",
  "timestamp": "2024-01-15T10:30:45Z",
  "regressions": {
    "performance": 2,
    "functional": 1,
    "security": 0,
    "production": 1
  },
  "dashboard_url": "https://dashboard.example.com",
  "runbook_url": "https://runbook.example.com/regression"
}
```

## Dashboard and Reporting

### HTML Dashboard

Interactive dashboard featuring:

- **Overall Status**: Health indicator and trend direction
- **Metric Visualizations**: Time-series charts for key metrics  
- **Regression Timeline**: Historical regression events
- **Trend Analysis**: Statistical analysis with predictions
- **Alert Summary**: Active and recent alerts
- **Baseline Information**: Current and historical baselines

### JSON API

Structured data export for programmatic access:

```bash
# Access dashboard data
curl http://localhost:8080/regression-dashboard.json

# Get specific metric trends  
curl http://localhost:8080/metrics/p95_latency.json

# Retrieve regression history
curl http://localhost:8080/regression-history.json?days=30
```

### Prometheus Metrics

Exportable metrics for monitoring systems:

```
# System quality metrics
nephoran_regression_total_score 92
nephoran_quality_score 89.5

# Performance metrics  
nephoran_p95_latency_seconds 0.00150
nephoran_throughput_intents_per_minute 47.3
nephoran_availability_percent 99.97

# Regression counters
nephoran_regressions_detected_total 15
nephoran_regressions_this_week 2
```

### CSV Exports

Data exports for analysis:
- `regression-history.csv`: Historical regression events
- `performance-metrics.csv`: Time-series performance data
- `baseline-comparison.csv`: Baseline-to-baseline comparisons

## Best Practices

### Baseline Management

1. **Regular Updates**: Update baselines after successful releases
2. **Environment Isolation**: Maintain separate baselines per environment
3. **Version Tagging**: Tag baselines with release versions
4. **Retention Policy**: Keep reasonable number of historical baselines

### Threshold Tuning

1. **Start Conservative**: Begin with strict thresholds, adjust based on experience
2. **Statistical Significance**: Ensure thresholds account for normal variation
3. **Business Context**: Align thresholds with business requirements
4. **Regular Review**: Periodically review and adjust thresholds

### CI/CD Integration

1. **Fail Fast**: Configure CI to fail on critical regressions
2. **Parallel Execution**: Run regression tests in parallel with other validations
3. **Artifact Preservation**: Always preserve test artifacts for analysis
4. **Dashboard Integration**: Link dashboards to CI/CD results

### Alert Management

1. **Alert Fatigue**: Avoid excessive alerting with proper threshold tuning
2. **Escalation**: Define clear escalation paths for different severities
3. **Documentation**: Maintain runbooks for common regression types
4. **Response Time**: Define SLAs for regression response and resolution

## Troubleshooting

### Common Issues

#### No Baseline Found
```bash
Error: baseline not found
Solution: Run 'make regression-baseline' to create initial baseline
```

#### Regression Detection False Positives
```bash
Issue: Too many regression alerts
Solution: 
- Review threshold configuration
- Analyze statistical significance
- Consider environmental factors
- Adjust baseline update frequency
```

#### Performance Issues
```bash
Issue: Regression tests taking too long
Solution:
- Enable parallel test execution
- Optimize validation suite performance  
- Use incremental regression testing
- Consider sampling for large datasets
```

#### Alert Delivery Failures
```bash
Issue: Alerts not being delivered
Solution:
- Verify webhook URLs and credentials
- Check network connectivity
- Review alert configuration
- Test alert system independently
```

### Debug Commands

```bash
# Check regression testing status
make regression-status

# Test alert system
make regression-alert-test

# View detailed logs
GINKGO_ARGS="-v" make regression-test

# Generate debug report
DEBUG=true make regression-test
```

## Advanced Features

### Custom Metrics

Add custom metrics to regression detection:

```go
// Add custom performance metric
baseline.PerformanceBaselines["custom_metric"] = &PerformanceBaseline{
    MetricName:      "Custom Response Time",
    AverageValue:    customMetricValue,
    Threshold:       customThreshold,
    Unit:           "milliseconds",
}
```

### Statistical Models

The framework supports multiple statistical models:

- **Linear Regression**: For trend analysis
- **Moving Averages**: For smoothing volatile metrics
- **Standard Deviation**: For anomaly detection
- **Confidence Intervals**: For prediction accuracy

### Integration APIs

```go
// Custom integration example
regressionFramework := validation.NewRegressionFramework(config, validationConfig)

// Execute regression test
detection, err := regressionFramework.ExecuteRegressionTest(ctx)
if err != nil {
    log.Fatal(err)
}

// Check for regressions
if detection.HasRegression {
    // Handle regression detection
    handleRegressions(detection)
}

// Generate custom dashboard
dashboard := validation.NewRegressionDashboard(regressionFramework, config)
dashboard.GenerateDashboard("./custom-reports")
```

## Support and Maintenance

### Monitoring Framework Health

The regression testing framework includes self-monitoring:

- **Framework Performance**: Tracks execution time and resource usage
- **Detection Accuracy**: Monitors false positive/negative rates
- **Alert Delivery**: Confirms successful alert delivery
- **Baseline Quality**: Validates baseline integrity

### Updates and Maintenance

- **Framework Updates**: Update through standard dependency management
- **Threshold Adjustment**: Regular review and tuning of detection thresholds
- **Baseline Maintenance**: Periodic baseline cleanup and optimization
- **Alert Configuration**: Regular review of alert channels and recipients

For additional support, see the main project documentation or open an issue in the project repository.