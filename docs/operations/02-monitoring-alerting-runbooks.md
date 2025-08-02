# Nephoran Intent Operator - Monitoring & Alerting Runbooks

## Overview

This document provides comprehensive monitoring setup, alerting configuration, and incident response runbooks for the Nephoran Intent Operator. The monitoring stack includes Prometheus, Grafana, Jaeger, and custom telecom-specific metrics for complete system observability.

## Monitoring Architecture

### Core Components
- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **Jaeger**: Distributed tracing
- **AlertManager**: Alert routing and notification
- **Custom Metrics**: Business and technical KPIs

### Key Metrics Categories
1. **System Health Metrics**: Pod status, resource utilization, network connectivity
2. **Business Metrics**: Intent processing rates, success rates, cost per intent
3. **Performance Metrics**: Response times, throughput, queue depths
4. **Security Metrics**: Authentication failures, policy violations, audit events
5. **O-RAN Metrics**: Interface utilization, policy enforcement, network function health

## Monitoring Stack Setup

### 1.1 Prometheus Configuration

**Production Prometheus Configuration:**
```yaml
# prometheus-production-config.yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  retention: 90d
  external_labels:
    cluster: 'nephoran-production'
    environment: 'production'

scrape_configs:
  # Nephoran Intent Operator controllers
  - job_name: 'nephoran-controllers'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - nephoran-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_part_of]
        action: keep
        regex: nephoran-intent-operator
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
    scrape_interval: 15s
    metrics_path: /metrics

  # LLM Processor service metrics
  - job_name: 'llm-processor'
    static_configs:
      - targets: ['llm-processor.nephoran-system.svc.cluster.local:8080']
    scrape_interval: 15s
    metrics_path: /metrics
    
  # RAG API service metrics
  - job_name: 'rag-api'
    static_configs:
      - targets: ['rag-api.nephoran-system.svc.cluster.local:8080']
    scrape_interval: 30s
    metrics_path: /metrics

  # Weaviate vector database metrics
  - job_name: 'weaviate'
    static_configs:
      - targets: ['weaviate.nephoran-system.svc.cluster.local:2112']
    scrape_interval: 30s
    metrics_path: /metrics

  # Kubernetes cluster metrics
  - job_name: 'kubernetes-nodes'
    kubernetes_sd_configs:
      - role: node
    relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
    scrape_interval: 30s

  # O-RAN interface metrics
  - job_name: 'oran-adaptors'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - nephoran-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: oran-adaptor
    scrape_interval: 15s

rule_files:
  - "/etc/prometheus/rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager.nephoran-monitoring.svc.cluster.local:9093
```

### 1.2 AlertManager Configuration

**Production AlertManager Setup:**
```yaml
# alertmanager-production-config.yaml
global:
  smtp_smarthost: 'smtp.company.com:587'
  smtp_from: 'nephoran-alerts@company.com'
  smtp_auth_username: 'nephoran-alerts@company.com'
  smtp_auth_password: 'alert-password'

templates:
  - '/etc/alertmanager/templates/*.tmpl'

route:
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
    # Critical alerts - immediate escalation
    - match:
        severity: critical
      receiver: 'critical-alerts'
      group_wait: 0s
      repeat_interval: 5m
      
    # High severity - operations team
    - match:
        severity: high
      receiver: 'operations-team'
      group_wait: 30s
      repeat_interval: 15m
      
    # Medium severity - standard alerts
    - match:
        severity: medium
      receiver: 'standard-alerts'
      group_wait: 2m
      repeat_interval: 1h
      
    # Low severity - information only
    - match:
        severity: low
      receiver: 'info-alerts'
      group_wait: 5m
      repeat_interval: 4h

receivers:
  - name: 'default'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#nephoran-alerts'
        title: 'Nephoran Alert'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

  - name: 'critical-alerts'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#nephoran-critical'
        title: '🚨 CRITICAL: {{ .GroupLabels.alertname }}'
        text: |
          *Cluster:* {{ .GroupLabels.cluster }}
          *Service:* {{ .GroupLabels.service }}
          *Alert:* {{ range .Alerts }}{{ .Annotations.summary }}{{ end }}
          *Description:* {{ range .Alerts }}{{ .Annotations.description }}{{ end }}
          *Runbook:* {{ range .Alerts }}{{ .Annotations.runbook_url }}{{ end }}
    pagerduty_configs:
      - routing_key: 'YOUR-PAGERDUTY-INTEGRATION-KEY'
        description: 'Critical Nephoran Alert: {{ .GroupLabels.alertname }}'
        severity: 'critical'
    email_configs:
      - to: 'oncall@company.com'
        subject: 'CRITICAL: Nephoran Alert - {{ .GroupLabels.alertname }}'
        body: |
          Critical alert in Nephoran Intent Operator:
          
          Alert: {{ .GroupLabels.alertname }}
          Cluster: {{ .GroupLabels.cluster }}
          Service: {{ .GroupLabels.service }}
          
          {{ range .Alerts }}
          Summary: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Runbook: {{ .Annotations.runbook_url }}
          {{ end }}

  - name: 'operations-team'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#nephoran-operations'
        title: '⚠️ HIGH: {{ .GroupLabels.alertname }}'
        text: |
          *Alert:* {{ range .Alerts }}{{ .Annotations.summary }}{{ end }}
          *Description:* {{ range .Alerts }}{{ .Annotations.description }}{{ end }}
    email_configs:
      - to: 'operations-team@company.com'
        subject: 'HIGH: Nephoran Alert - {{ .GroupLabels.alertname }}'

  - name: 'standard-alerts'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#nephoran-alerts'
        title: 'ℹ️ MEDIUM: {{ .GroupLabels.alertname }}'

  - name: 'info-alerts'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#nephoran-info'
        title: '💡 INFO: {{ .GroupLabels.alertname }}'

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'high'
    equal: ['alertname', 'cluster', 'service']
  - source_match:
      severity: 'high'
    target_match:
      severity: 'medium'
    equal: ['alertname', 'cluster', 'service']
```

## Alert Rules and Thresholds

### 2.1 Critical System Alerts

```yaml
# critical-alerts.yml
groups:
- name: nephoran-critical-alerts
  rules:
  # System availability
  - alert: NephoranSystemDown
    expr: up{job=~"nephoran.*"} == 0
    for: 1m
    labels:
      severity: critical
      component: system
    annotations:
      summary: "Nephoran component {{ $labels.job }} is down"
      description: "{{ $labels.job }} has been down for more than 1 minute"
      runbook_url: "https://docs.nephoran.com/runbooks/system-down"

  # Intent processing failure
  - alert: IntentProcessingFailureHigh
    expr: rate(nephoran_networkintent_failed_total[5m]) / rate(nephoran_networkintent_total[5m]) > 0.1
    for: 2m
    labels:
      severity: critical
      component: intent-processing
    annotations:
      summary: "High intent processing failure rate"
      description: "Intent processing failure rate is {{ $value | humanizePercentage }}, exceeding 10% threshold"
      runbook_url: "https://docs.nephoran.com/runbooks/intent-processing-failures"

  # Memory usage critical
  - alert: PodMemoryUsageCritical
    expr: container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.9
    for: 5m
    labels:
      severity: critical
      component: resource-management
    annotations:
      summary: "Pod {{ $labels.pod }} memory usage critical"
      description: "Pod {{ $labels.pod }} memory usage is {{ $value | humanizePercentage }}, exceeding 90%"
      runbook_url: "https://docs.nephoran.com/runbooks/memory-issues"

  # Vector database unavailable
  - alert: WeaviateUnavailable
    expr: up{job="weaviate"} == 0
    for: 2m
    labels:
      severity: critical
      component: weaviate
    annotations:
      summary: "Weaviate vector database is unavailable"
      description: "Weaviate has been unavailable for more than 2 minutes"
      runbook_url: "https://docs.nephoran.com/runbooks/weaviate-down"

  # LLM service failure
  - alert: LLMProcessorDown
    expr: up{job="llm-processor"} == 0
    for: 1m
    labels:
      severity: critical
      component: llm-processor
    annotations:
      summary: "LLM Processor service is down"
      description: "LLM Processor has been down for more than 1 minute"
      runbook_url: "https://docs.nephoran.com/runbooks/llm-processor-down"
```

### 2.2 Performance and SLA Alerts

```yaml
# performance-alerts.yml
groups:
- name: nephoran-performance-alerts
  rules:
  # Response time SLA violation
  - alert: IntentProcessingLatencyHigh
    expr: histogram_quantile(0.95, rate(nephoran_llm_request_duration_seconds_bucket[5m])) > 3.0
    for: 2m
    labels:
      severity: high
      component: performance
    annotations:
      summary: "Intent processing latency exceeds SLA"
      description: "P95 latency is {{ $value }}s, exceeding 3.0s SLA threshold"
      runbook_url: "https://docs.nephoran.com/runbooks/performance-issues"

  # Throughput degradation
  - alert: IntentProcessingThroughputLow
    expr: rate(nephoran_networkintent_processed_total[5m]) < 5
    for: 5m
    labels:
      severity: high
      component: performance
    annotations:
      summary: "Intent processing throughput below target"
      description: "Current throughput is {{ $value }} intents/min, below 5/min target"
      runbook_url: "https://docs.nephoran.com/runbooks/throughput-issues"

  # Queue depth high
  - alert: LLMProcessingQueueHigh
    expr: nephoran_llm_processing_queue_depth > 50
    for: 3m
    labels:
      severity: medium
      component: llm-processor
    annotations:
      summary: "LLM processing queue depth high"
      description: "Queue depth is {{ $value }}, indicating processing bottleneck"
      runbook_url: "https://docs.nephoran.com/runbooks/queue-management"

  # Cache performance degradation
  - alert: RAGCacheHitRateLow
    expr: nephoran_rag_cache_hit_rate < 0.7
    for: 10m
    labels:
      severity: medium
      component: rag-api
    annotations:
      summary: "RAG API cache hit rate low"
      description: "Cache hit rate is {{ $value | humanizePercentage }}, below 70% target"
      runbook_url: "https://docs.nephoran.com/runbooks/cache-optimization"
```

### 2.3 Business and Cost Alerts

```yaml
# business-alerts.yml
groups:
- name: nephoran-business-alerts
  rules:
  # Cost management
  - alert: LLMCostExceedsDaily
    expr: increase(nephoran_llm_cost_dollars_total[24h]) > 100
    for: 1h
    labels:
      severity: high
      component: cost-management
    annotations:
      summary: "Daily LLM costs exceed budget"
      description: "24-hour LLM costs are ${{ $value }}, exceeding $100 daily budget"
      runbook_url: "https://docs.nephoran.com/runbooks/cost-management"

  # Success rate SLA
  - alert: IntentSuccessRateBelowSLA
    expr: rate(nephoran_networkintent_success_total[15m]) / rate(nephoran_networkintent_total[15m]) < 0.95
    for: 5m
    labels:
      severity: high
      component: business-sla
    annotations:
      summary: "Intent success rate below SLA"
      description: "Success rate is {{ $value | humanizePercentage }}, below 95% SLA"
      runbook_url: "https://docs.nephoran.com/runbooks/sla-violations"

  # Deployment automation rate
  - alert: AutomationRateLow
    expr: rate(nephoran_automated_deployments_total[1h]) / rate(nephoran_total_deployments_total[1h]) < 0.8
    for: 30m
    labels:
      severity: medium
      component: automation
    annotations:
      summary: "Deployment automation rate below target"
      description: "Automation rate is {{ $value | humanizePercentage }}, below 80% target"
      runbook_url: "https://docs.nephoran.com/runbooks/automation-optimization"
```

## Grafana Dashboard Configuration

### 3.1 Executive Dashboard

```json
{
  "dashboard": {
    "title": "Nephoran Executive Dashboard",
    "tags": ["nephoran", "executive", "business"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "System Availability SLA",
        "type": "stat",
        "targets": [
          {
            "expr": "avg(up{job=~\"nephoran.*\"})",
            "legendFormat": "Availability"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percentunit",
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 0.95},
                {"color": "green", "value": 0.99}
              ]
            }
          }
        },
        "gridPos": {"h": 8, "w": 6, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "Daily Intent Processing Volume",
        "type": "stat", 
        "targets": [
          {
            "expr": "increase(nephoran_networkintent_total[24h])",
            "legendFormat": "Intents Processed"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "short",
            "color": {"mode": "value"},
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 100},
                {"color": "green", "value": 500}
              ]
            }
          }
        },
        "gridPos": {"h": 8, "w": 6, "x": 6, "y": 0}
      },
      {
        "id": 3,
        "title": "Intent Success Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(nephoran_networkintent_success_total[24h]) / rate(nephoran_networkintent_total[24h])",
            "legendFormat": "Success Rate"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percentunit",
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 0.95},
                {"color": "green", "value": 0.98}
              ]
            }
          }
        },
        "gridPos": {"h": 8, "w": 6, "x": 12, "y": 0}
      },
      {
        "id": 4,
        "title": "Daily Cost Tracking",
        "type": "stat",
        "targets": [
          {
            "expr": "increase(nephoran_llm_cost_dollars_total[24h])",
            "legendFormat": "Daily Cost"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "currencyUSD",
            "thresholds": {
              "steps": [
                {"color": "green", "value": 0},
                {"color": "yellow", "value": 50},
                {"color": "red", "value": 100}
              ]
            }
          }
        },
        "gridPos": {"h": 8, "w": 6, "x": 18, "y": 0}
      },
      {
        "id": 5,
        "title": "Intent Processing Trends",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(nephoran_networkintent_processed_total[5m])",
            "legendFormat": "Processing Rate"
          },
          {
            "expr": "rate(nephoran_networkintent_success_total[5m])",
            "legendFormat": "Success Rate"
          },
          {
            "expr": "rate(nephoran_networkintent_failed_total[5m])",
            "legendFormat": "Failure Rate"
          }
        ],
        "yAxes": [
          {"label": "Intents/sec", "min": 0},
          {"label": "Rate", "min": 0, "max": 1}
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8}
      },
      {
        "id": 6,
        "title": "Response Time Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(nephoran_llm_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.95, rate(nephoran_llm_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P95"
          },
          {
            "expr": "histogram_quantile(0.99, rate(nephoran_llm_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P99"
          }
        ],
        "yAxes": [
          {"label": "Seconds", "min": 0}
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8}
      }
    ],
    "time": {"from": "now-24h", "to": "now"},
    "refresh": "30s"
  }
}
```

### 3.2 Technical Operations Dashboard

```json
{
  "dashboard": {
    "title": "Nephoran Technical Operations",
    "tags": ["nephoran", "operations", "technical"],
    "panels": [
      {
        "id": 1,
        "title": "Pod Resource Utilization",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(container_cpu_usage_seconds_total{namespace=\"nephoran-system\"}[5m])",
            "legendFormat": "{{ pod }} CPU"
          },
          {
            "expr": "container_memory_usage_bytes{namespace=\"nephoran-system\"} / 1024 / 1024",
            "legendFormat": "{{ pod }} Memory (MB)"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "Network I/O",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(container_network_receive_bytes_total{namespace=\"nephoran-system\"}[5m])",
            "legendFormat": "{{ pod }} RX"
          },
          {
            "expr": "rate(container_network_transmit_bytes_total{namespace=\"nephoran-system\"}[5m])",
            "legendFormat": "{{ pod }} TX"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0}
      },
      {
        "id": 3,
        "title": "Error Rates by Component",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(nephoran_errors_total[5m])",
            "legendFormat": "{{ component }} errors"
          }
        ],
        "gridPos": {"h": 8, "w": 24, "x": 0, "y": 8}
      }
    ]
  }
}
```

## Incident Response Runbooks

### 4.1 Critical System Down (P1)

**Alert:** `NephoranSystemDown`

**Impact:** Complete system unavailability affecting all network operations

**Response Time:** Immediate (0-5 minutes)

**Escalation Path:**
1. Level 1: Operations team (immediate)
2. Level 2: Engineering team (5 minutes)
3. Level 3: Engineering management (15 minutes)

**Response Procedure:**
```bash
#!/bin/bash
# P1 Incident Response - System Down

INCIDENT_ID="P1-$(date +%Y%m%d-%H%M%S)"
echo "🚨 P1 INCIDENT: $INCIDENT_ID - System Down"

# Step 1: Immediate assessment
kubectl get pods -n nephoran-system --no-headers | grep -v Running
kubectl get nodes -o wide
kubectl top nodes

# Step 2: Check recent events
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | tail -20

# Step 3: Identify failing components
for deployment in llm-processor rag-api nephio-bridge oran-adaptor weaviate; do
  echo "Checking $deployment..."
  kubectl get deployment $deployment -n nephoran-system -o json | \
    jq '.status | {replicas, readyReplicas, unavailableReplicas}'
done

# Step 4: Automatic recovery attempts
kubectl rollout restart deployment/llm-processor -n nephoran-system
kubectl rollout restart deployment/rag-api -n nephoran-system
kubectl rollout restart deployment/nephio-bridge -n nephoran-system

# Step 5: Scale up if needed
kubectl scale deployment llm-processor --replicas=5 -n nephoran-system
kubectl scale deployment rag-api --replicas=3 -n nephoran-system

# Step 6: Monitor recovery
kubectl wait --for=condition=available deployment/llm-processor \
  -n nephoran-system --timeout=300s
kubectl wait --for=condition=available deployment/rag-api \
  -n nephoran-system --timeout=300s

# Step 7: Validate functionality
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: incident-test-$INCIDENT_ID
  namespace: nephoran-system
spec:
  description: "Test intent for incident validation"
  priority: high
EOF

# Step 8: Notify stakeholders
curl -X POST "$SLACK_WEBHOOK_URL" \
  -H 'Content-type: application/json' \
  --data "{\"text\":\"🚨 P1 INCIDENT $INCIDENT_ID: System recovery initiated. Status: $(kubectl get deployments -n nephoran-system -o jsonpath='{.items[*].status.readyReplicas}')\"}"

echo "✅ P1 response complete - Incident ID: $INCIDENT_ID"
```

### 4.2 Performance Degradation (P2)

**Alert:** `IntentProcessingLatencyHigh`

**Impact:** SLA violations affecting user experience

**Response Time:** 15 minutes

**Response Procedure:**
```bash
#!/bin/bash
# P2 Incident Response - Performance Degradation

INCIDENT_ID="P2-$(date +%Y%m%d-%H%M%S)"
echo "⚠️  P2 INCIDENT: $INCIDENT_ID - Performance Degradation"

# Step 1: Gather performance metrics
CURRENT_P95=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=histogram_quantile(0.95,rate(nephoran_llm_request_duration_seconds_bucket[5m]))" | jq -r '.data.result[0].value[1]')
CACHE_HIT_RATE=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=nephoran_rag_cache_hit_rate" | jq -r '.data.result[0].value[1]')

echo "Current P95 Latency: ${CURRENT_P95}s"
echo "Cache Hit Rate: ${CACHE_HIT_RATE}"

# Step 2: Check resource utilization
kubectl top pods -n nephoran-system

# Step 3: Analyze bottlenecks
kubectl describe hpa -n nephoran-system

# Step 4: Immediate optimizations
if (( $(echo "$CURRENT_P95 > 3.0" | bc -l) )); then
  echo "Triggering aggressive scaling..."
  kubectl patch hpa llm-processor -n nephoran-system --type merge \
    -p='{"spec":{"metrics":[{"type":"Resource","resource":{"name":"cpu","target":{"type":"Utilization","averageUtilization":50}}}]}}'
fi

if (( $(echo "$CACHE_HIT_RATE < 0.6" | bc -l) )); then
  echo "Clearing and warming cache..."
  kubectl exec deployment/rag-api -n nephoran-system -- \
    curl -X DELETE http://localhost:8080/cache/clear
  kubectl exec deployment/rag-api -n nephoran-system -- \
    curl -X POST http://localhost:8080/cache/warm
fi

# Step 5: Monitor improvement
sleep 180
NEW_P95=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=histogram_quantile(0.95,rate(nephoran_llm_request_duration_seconds_bucket[5m]))" | jq -r '.data.result[0].value[1]')
echo "Updated P95 Latency: ${NEW_P95}s"

echo "✅ P2 response complete - Incident ID: $INCIDENT_ID"
```

### 4.3 Cost Budget Exceeded (P3)

**Alert:** `LLMCostExceedsDaily`

**Impact:** Budget overrun affecting operational costs

**Response Time:** 1 hour

**Response Procedure:**
```bash
#!/bin/bash
# P3 Incident Response - Cost Budget Exceeded

INCIDENT_ID="P3-$(date +%Y%m%d-%H%M%S)"
echo "💰 P3 INCIDENT: $INCIDENT_ID - Cost Budget Exceeded"

# Step 1: Analyze current costs
DAILY_COST=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=increase(nephoran_llm_cost_dollars_total[24h])" | jq -r '.data.result[0].value[1]')
HOURLY_RATE=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=rate(nephoran_llm_cost_dollars_total[1h])" | jq -r '.data.result[0].value[1]')

echo "Current 24h cost: \$${DAILY_COST}"
echo "Current hourly rate: \$${HOURLY_RATE}/hour"

# Step 2: Identify cost drivers
kubectl logs deployment/llm-processor -n nephoran-system --since=1h | grep "token_usage" | tail -20

# Step 3: Implement cost controls
kubectl patch configmap llm-processor-config -n nephoran-system --type merge \
  -p='{"data":{"cost_control_enabled":"true","max_tokens_per_request":"2048"}}'

# Step 4: Scale down non-critical processing
kubectl scale deployment rag-api --replicas=1 -n nephoran-system

# Step 5: Enable cost-optimized processing
kubectl patch configmap llm-processor-config -n nephoran-system --type merge \
  -p='{"data":{"prefer_cached_responses":"true","enable_cost_optimization":"true"}}'

# Step 6: Notify finance team
curl -X POST "$SLACK_WEBHOOK_URL" \
  -H 'Content-type: application/json' \
  --data "{\"text\":\"💰 COST ALERT $INCIDENT_ID: Daily LLM costs \$${DAILY_COST} exceed budget. Cost controls activated.\"}"

echo "✅ P3 response complete - Incident ID: $INCIDENT_ID"
```

## Capacity Planning and Scaling

### 5.1 Automated Capacity Analysis

```bash
#!/bin/bash
# capacity-analysis.sh - Weekly capacity planning analysis

ANALYSIS_DATE=$(date +%Y%m%d)
REPORT_FILE="/var/log/nephoran/capacity-analysis-$ANALYSIS_DATE.json"

# Collect metrics for analysis
WEEKLY_INTENT_VOLUME=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=increase(nephoran_networkintent_total[7d])" | jq -r '.data.result[0].value[1]')
PEAK_CPU_USAGE=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=max_over_time(rate(container_cpu_usage_seconds_total{namespace=\"nephoran-system\"}[5m])[7d])" | jq -r '.data.result[0].value[1]')
PEAK_MEMORY_USAGE=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=max_over_time(container_memory_usage_bytes{namespace=\"nephoran-system\"}[7d])" | jq -r '.data.result[0].value[1]')

# Calculate growth trends
GROWTH_RATE=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=(increase(nephoran_networkintent_total[7d]) - increase(nephoran_networkintent_total[7d] offset 7d)) / increase(nephoran_networkintent_total[7d] offset 7d)" | jq -r '.data.result[0].value[1]')

# Generate capacity recommendations
cat > "$REPORT_FILE" <<EOF
{
  "analysis_date": "$ANALYSIS_DATE",
  "current_metrics": {
    "weekly_intent_volume": $WEEKLY_INTENT_VOLUME,
    "peak_cpu_usage": $PEAK_CPU_USAGE,
    "peak_memory_usage": $PEAK_MEMORY_USAGE,
    "weekly_growth_rate": $GROWTH_RATE
  },
  "capacity_recommendations": {
    "scale_up_trigger": $(echo "$GROWTH_RATE > 0.2" | bc -l),
    "recommended_additional_nodes": $(echo "($GROWTH_RATE * 10)" | bc -l | cut -d. -f1),
    "storage_expansion_needed": $(echo "$PEAK_MEMORY_USAGE > 80000000000" | bc -l)
  }
}
EOF

echo "Capacity analysis complete: $REPORT_FILE"
```

### 5.2 Predictive Scaling Rules

```yaml
# predictive-scaling-rules.yml
groups:
- name: nephoran-predictive-scaling
  rules:
  # Predict high load based on historical patterns
  - alert: PredictedHighLoad
    expr: |
      (
        predict_linear(rate(nephoran_networkintent_total[30m])[30m:5m], 600) > 
        avg_over_time(rate(nephoran_networkintent_total[5m])[1d]) * 1.5
      )
    for: 5m
    labels:
      severity: medium
      component: capacity-planning
    annotations:
      summary: "High load predicted in next 10 minutes"
      description: "Predicted load: {{ $value }} intents/min (50% above average)"
      runbook_url: "https://docs.nephoran.com/runbooks/predictive-scaling"
```

This comprehensive monitoring and alerting guide provides the foundation for maintaining high availability and performance of the Nephoran Intent Operator in production environments.