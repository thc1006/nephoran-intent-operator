# Nephoran Intent Operator - Monitoring Stack

This directory contains the Prometheus and Grafana monitoring stack configuration for the Nephoran Intent Operator.

## Overview

The monitoring stack provides comprehensive observability for the Nephoran Intent Operator, including:

- **Service Level Indicators (SLIs)** for production reliability
- **Performance metrics** for LLM processing, RAG queries, and O-RAN interfaces
- **Resource utilization** monitoring
- **Error tracking and alerting** capabilities

## Components

### Prometheus
- Metrics collection and storage
- Service discovery for Kubernetes
- Alerting rules (when configured with AlertManager)

### Grafana
- Visualization dashboards
- SLI monitoring
- Performance analysis

## Dashboards

1. **Nephoran Intent Operator - Key SLIs** (`grafana-dashboard-sli.yaml`)
   - Service Availability (Target: 99.9%)
   - Request Success Rate (Target: 99.5%)
   - P95 Latency (Target: <2s)
   - Error Budget Tracking
   - NetworkIntent Processing Time
   - E2NodeSet Reconciliation Time

2. **RAG System Overview** (`grafana-dashboards.yaml`)
   - Query rate and latency
   - Document processing queue
   - Cache hit rate
   - Embedding generation performance

3. **LLM Processor Performance** (`grafana-dashboards.yaml`)
   - Request rate and latency percentiles
   - Token usage tracking
   - Pod scaling metrics
   - Model performance by type

4. **Weaviate Vector Database Performance** (`grafana-dashboards.yaml`)
   - Vector operations rate
   - Query response time
   - Object count and memory usage
   - Disk usage by class

5. **Telecom Workload Analytics** (`grafana-dashboards.yaml`)
   - NetworkIntent processing metrics
   - O-RAN interface load
   - E2NodeSet scaling activity
   - System resource pressure

## Deployment

### Prerequisites

1. Kubernetes cluster with Prometheus Operator installed (optional but recommended)
2. Namespace `nephoran-system` created

### Deploy using Kustomize

```bash
# Deploy the monitoring stack
kubectl apply -k deployments/monitoring/

# Verify deployment
kubectl get pods -n nephoran-system -l app.kubernetes.io/component=monitoring
```

### Access Grafana

```bash
# Port-forward to access Grafana
kubectl port-forward -n nephoran-system svc/grafana 3000:3000

# Access Grafana at http://localhost:3000
# Default credentials: admin/admin
```

### Access Prometheus

```bash
# Port-forward to access Prometheus
kubectl port-forward -n nephoran-system svc/prometheus 9090:9090

# Access Prometheus at http://localhost:9090
```

## Key Metrics

### Service Level Indicators (SLIs)

| SLI | Target | Query |
|-----|--------|-------|
| Service Availability | 99.9% | `(1 - (sum(rate(nephoran_errors_total[5m])) / sum(rate(nephoran_requests_total[5m])))) * 100` |
| Request Success Rate | 99.5% | `(sum(rate(nephoran_requests_total{status="success"}[5m])) / sum(rate(nephoran_requests_total[5m]))) * 100` |
| P95 Latency | <2s | `histogram_quantile(0.95, sum(rate(nephoran_request_duration_seconds_bucket[5m])) by (le))` |
| LLM API Response Time | P95 <5s | `histogram_quantile(0.95, sum(rate(nephoran_llm_request_duration_seconds_bucket[5m])) by (le))` |
| RAG Query Success Rate | >98% | `(sum(rate(rag_queries_total{status="success"}[5m])) / sum(rate(rag_queries_total[5m]))) * 100` |
| O-RAN Interface Availability | 99.95% | `(1 - (sum(rate(nephoran_oran_interface_errors_total[5m])) / sum(rate(nephoran_oran_interface_requests_total[5m])))) * 100` |

### Performance Metrics

- **NetworkIntent Processing**: `nephoran_networkintent_processing_duration_seconds`
- **E2NodeSet Reconciliation**: `nephoran_e2nodeset_reconciliation_duration_seconds`
- **LLM Request Duration**: `nephoran_llm_request_duration_seconds`
- **RAG Query Latency**: `rag_query_latency_seconds`
- **O-RAN Interface Latency**: `nephoran_oran_interface_latency_seconds`

### Resource Metrics

- **Controller Memory Usage**: `container_memory_working_set_bytes{pod=~"nephoran-controller-.*"}`
- **CPU Usage**: `container_cpu_usage_seconds_total{pod=~"nephoran-.*"}`
- **Pod Replicas**: `kube_deployment_status_replicas{deployment=~"nephoran-.*"}`

## Alerting

To set up alerting, create PrometheusRule resources:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: nephoran-sli-alerts
  namespace: nephoran-system
spec:
  groups:
    - name: sli.rules
      interval: 30s
      rules:
        - alert: ServiceAvailabilityBelowSLO
          expr: |
            (1 - (sum(rate(nephoran_errors_total[5m])) / sum(rate(nephoran_requests_total[5m])))) * 100 < 99.9
          for: 5m
          labels:
            severity: critical
            team: platform
          annotations:
            summary: "Service availability below SLO"
            description: "Service availability is {{ $value }}%, below the 99.9% SLO"
```

## Customization

### Adding New Metrics

1. Instrument your code with Prometheus metrics
2. Expose metrics endpoint (typically `:2112/metrics`)
3. Add ServiceMonitor configuration
4. Create Grafana dashboard

### Modifying Dashboards

1. Edit dashboard JSON in ConfigMaps
2. Apply changes: `kubectl apply -f deployments/monitoring/grafana-dashboards.yaml`
3. Refresh Grafana to load updated dashboards

## Troubleshooting

### Metrics Not Appearing

1. Check ServiceMonitor is created: `kubectl get servicemonitor -n nephoran-system`
2. Verify service labels match ServiceMonitor selector
3. Check Prometheus targets: http://localhost:9090/targets
4. Verify metrics endpoint is accessible: `kubectl port-forward <pod> 2112:2112`

### Dashboard Not Loading

1. Check ConfigMap is created with label `grafana_dashboard: "1"`
2. Verify Grafana sidecar is configured to watch ConfigMaps
3. Check Grafana logs: `kubectl logs -n nephoran-system deployment/grafana`

## Production Considerations

1. **Retention**: Configure appropriate retention for metrics storage
2. **High Availability**: Deploy Prometheus in HA mode for production
3. **Storage**: Use persistent volumes for Prometheus data
4. **Security**: Enable authentication and TLS for external access
5. **Resource Limits**: Set appropriate resource requests and limits
6. **Backup**: Regular backup of Grafana dashboards and Prometheus data