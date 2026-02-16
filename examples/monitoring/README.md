# Nephoran Intent Operator - Monitoring Examples

This directory contains comprehensive monitoring examples and configurations for the Nephoran Intent Operator, providing production-ready observability through Prometheus metrics and Grafana dashboards.

## 📋 Contents

| File | Purpose | Description |
|------|---------|-------------|
| **[prometheus-queries.yaml](prometheus-queries.yaml)** | Query Examples | 50+ optimized Prometheus queries for monitoring, troubleshooting, and business intelligence |
| **[prometheus-alerts.yaml](prometheus-alerts.yaml)** | Alert Rules | Production-ready alerting rules with appropriate thresholds and severities |
| **[grafana-dashboard.json](grafana-dashboard.json)** | Dashboard Config | Comprehensive Grafana dashboard with 20+ panels covering all system aspects |
| **[prometheus-scrape-config.yaml](prometheus-scrape-config.yaml)** | Scrape Config | Complete Prometheus configuration for all Nephoran components |

## 🚀 Quick Setup

### 1. Enable Metrics in Nephoran Intent Operator

```bash
# Set environment variable to enable metrics collection
export METRICS_ENABLED=true

# Optional: Restrict metrics access (recommended for production)
export METRICS_ALLOWED_IPS="10.0.0.50,10.0.0.51"  # Monitoring servers only

# Apply to your deployment
kubectl set env deployment/nephoran-intent-operator -n nephoran-system METRICS_ENABLED=true
```

### 2. Configure Prometheus Scraping

**Option A: Add to existing Prometheus configuration**
```bash
# Append to your prometheus.yml
cat prometheus-scrape-config.yaml >> /etc/prometheus/prometheus.yml
```

**Option B: Kubernetes ServiceMonitor (if using Prometheus Operator)**
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nephoran-intent-operator
  namespace: nephoran-system
spec:
  selector:
    matchLabels:
      app: nephoran-intent-operator
  endpoints:
  - port: metrics
    interval: 15s
    path: /metrics
```

### 3. Deploy Alert Rules

```bash
# Apply Prometheus alerting rules
kubectl create configmap nephoran-alerts --from-file=prometheus-alerts.yaml -n monitoring
```

### 4. Import Grafana Dashboard

1. Open Grafana Web UI
2. Navigate to **Dashboards** → **Import**
3. Upload `grafana-dashboard.json`
4. Configure data source (Prometheus)
5. Save dashboard

**Or via API:**
```bash
curl -X POST \
  http://grafana:3000/api/dashboards/db \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d @grafana-dashboard.json
```

## 📊 Dashboard Overview

The Grafana dashboard provides comprehensive monitoring across six main sections:

### 🎯 System Overview
- **Service Status**: Real-time service availability
- **Overall Success Rate**: Combined LLM and controller success metrics  
- **System Throughput**: Intents processed per minute
- **P95 Processing Time**: Performance latency tracking
- **Cache Hit Rate**: LLM cache efficiency
- **Active NetworkIntents**: Resource status distribution

### 🧠 LLM Performance
- **Request Rate by Model**: Traffic distribution across AI models
- **Processing Duration**: P50/P95/P99 latency percentiles
- **Error Rate by Model**: Failure analysis and troubleshooting
- **Cache Performance**: Hit/miss rates for cost optimization
- **Fallback & Retry Rates**: Reliability and resilience monitoring

### 🎮 Controller Operations  
- **NetworkIntent Processing Rate**: Controller throughput and success rates
- **Processing Duration by Phase**: Detailed timing breakdown (LLM, validation, deployment)
- **Status Distribution**: Visual representation of intent states
- **Error Rate by Type**: Categorized failure analysis
- **Top Slow NetworkIntents**: Performance bottleneck identification

### 💻 System Resources
- **CPU Usage**: Processor utilization trends
- **Memory Usage**: RAM consumption monitoring
- **Goroutines**: Go runtime health indicators

### 📈 Business Intelligence
- **Hourly Intent Volume**: Usage patterns and trends
- **Model Usage Distribution**: AI model adoption analysis
- **Top Active Namespaces**: Resource utilization by tenant

### 📋 SLA Monitoring
- **Availability SLA**: 99.95% uptime tracking
- **Latency SLA Compliance**: Sub-2-second processing verification
- **Error Budget Consumption**: Monthly error allowance tracking
- **Throughput SLA**: 45 intents/minute capacity monitoring

## 🔍 Key Monitoring Queries

### System Health Quick Check
```promql
# Overall system availability
up{job="nephoran-intent-operator"}

# System-wide success rate
(rate(nephoran_llm_requests_total{status="success"}[5m]) + 
 rate(networkintent_reconciles_total{result="success"}[5m])) /
(rate(nephoran_llm_requests_total[5m]) + 
 rate(networkintent_reconciles_total[5m])) * 100
```

### Performance Monitoring
```promql
# P95 end-to-end processing time
histogram_quantile(0.95, 
  rate(networkintent_processing_duration_seconds_bucket{phase="total"}[5m]))

# Cache efficiency
rate(nephoran_llm_cache_hits_total[5m]) / 
(rate(nephoran_llm_cache_hits_total[5m]) + rate(nephoran_llm_cache_misses_total[5m]))
```

### Troubleshooting
```promql
# Top error-prone NetworkIntents
topk(10, rate(networkintent_reconcile_errors_total[1h]))

# High retry rates (potential issues)
rate(nephoran_llm_retry_attempts_total[5m]) > 0.1
```

## 🚨 Production Alerting Strategy

### Critical Alerts (Immediate Response)
- **Service Down**: Service unavailable > 1 minute
- **High Error Rate**: LLM errors > 10% for 2 minutes
- **NetworkIntent Failures**: Controller errors > 0.1/sec for 3 minutes  
- **Extreme Latency**: P95 processing > 30 seconds

### Warning Alerts (Action Recommended)
- **Slow Processing**: P95 LLM processing > 10 seconds
- **Low Cache Hit Rate**: Cache efficiency < 30%
- **High Fallback Rate**: Model fallbacks > 5%
- **Resource Constraints**: CPU > 80% or Memory > 2GB

### Info Alerts (Monitoring)
- **Low Throughput**: < 10 intents/minute for 30 minutes
- **Unusual Patterns**: Request volume 3x above average
- **Model Imbalance**: Usage ratio > 10:1 between models

## 🔧 Configuration Customization

### Environment-Specific Labels
Update scrape configurations for your environment:
```yaml
metric_relabel_configs:
  - target_label: environment
    replacement: 'staging'  # Change: production, staging, dev
  - target_label: cluster  
    replacement: 'nephoran-west'  # Your cluster identifier
```

### Alert Threshold Tuning
Adjust alert thresholds based on your SLA requirements:
```yaml
# Example: Adjust error rate threshold
expr: rate(nephoran_llm_errors_total[5m]) / rate(nephoran_llm_requests_total[5m]) > 0.05  # 5% instead of 10%
```

### Dashboard Variables
The dashboard includes template variables for dynamic filtering:
- **$namespace**: Filter by Kubernetes namespace
- **$model**: Filter by LLM model  
- **$interval**: Adjust time granularity

## 📚 Advanced Usage

### Multi-Cluster Monitoring
For monitoring multiple Nephoran deployments:
```yaml
# Add federation configuration
- job_name: 'federate'
  honor_labels: true
  metrics_path: '/federate'
  params:
    'match[]':
      - '{job=~"nephoran.*"}'
  static_configs:
    - targets: ['remote-prometheus:9090']
```

### Long-Term Storage
Configure Prometheus remote write for historical data:
```yaml
remote_write:
  - url: "https://your-tsdb-endpoint/write"
    basic_auth:
      username: "monitoring"
      password: "secure-password"
```

### Custom Metrics Integration
Extend monitoring with business-specific metrics:
```go
// Add custom metrics to your NetworkIntent controller
customMetric := prometheus.NewCounterVec(
  prometheus.CounterOpts{
    Name: "nephoran_custom_business_metric_total",
    Help: "Custom business logic metric",
  },
  []string{"business_unit", "operation"},
)
```

## 🔗 Related Documentation

- **[Complete Prometheus Metrics Documentation](../../PROMETHEUS_METRICS.md)**: Detailed metrics reference
- **[Production Monitoring Guide](../../docs/runbooks/monitoring-alerting-runbook.md)**: Operational procedures
- **[Performance Optimization](../../docs/archive/reports/performance-optimization.md)**: System tuning guidance
- **[Troubleshooting Guide](../../docs/runbooks/troubleshooting-guide.md)**: Issue resolution procedures

## 🤝 Contributing

To improve monitoring configurations:

1. **Test Changes**: Validate queries in Prometheus UI
2. **Update Documentation**: Keep README and comments current
3. **Version Dashboards**: Export updated JSON with version notes
4. **Share Best Practices**: Document optimization techniques

**Example Contribution:**
```bash
# Test new query
curl -G 'http://prometheus:9090/api/v1/query' \
  --data-urlencode 'query=your_new_query'

# Update dashboard
# Export from Grafana UI → Update JSON → Test import

# Submit changes
git add examples/monitoring/
git commit -m "feat: Add cost optimization metrics to dashboard"
```

---

**Questions or Issues?**
- 📖 [Complete Documentation](../../README.md#-monitoring--observability)
- 💬 [Discord Community](https://discord.gg/nephoran)
- 🐛 [Report Issues](https://github.com/thc1006/nephoran-intent-operator/issues)