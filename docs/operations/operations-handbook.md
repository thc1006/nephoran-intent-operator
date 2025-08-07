# Nephoran Intent Operator - Operations Handbook

## Table of Contents

1. [Service Level Objectives (SLOs)](#service-level-objectives-slos)
2. [Monitoring and Alerting](#monitoring-and-alerting)
3. [Upgrade Procedures](#upgrade-procedures)
4. [Operational Runbooks](#operational-runbooks)
5. [Emergency Response](#emergency-response)
6. [Performance Optimization](#performance-optimization)
7. [Troubleshooting Reference](#troubleshooting-reference)

---

## Service Level Objectives (SLOs)

### 1.1 System Availability SLOs

| Service Component | Target SLO | Error Budget | Measurement Window | SLI Definition |
|-------------------|------------|---------------|-------------------|----------------|
| **Overall System** | 99.95% | 0.05% (21.6 min/month) | 30 days | HTTP 200 responses / total requests |
| **Intent Processing** | 99.9% | 0.1% (43.2 min/month) | 30 days | Successfully processed intents / total intents |
| **LLM Service** | 99.8% | 0.2% (86.4 min/month) | 30 days | LLM API success rate |
| **RAG System** | 99.9% | 0.1% (43.2 min/month) | 30 days | Vector search success rate |
| **O-RAN Interface** | 99.95% | 0.05% (21.6 min/month) | 30 days | O-RAN API availability |

**SLO Monitoring Configuration:**
```yaml
# slo-monitoring-rules.yaml
groups:
  - name: nephoran_slo_rules
    interval: 30s
    rules:
      # Overall system availability
      - record: sli:system_availability:rate5m
        expr: |
          (
            sum(rate(http_requests_total{code!~"5.."}[5m])) /
            sum(rate(http_requests_total[5m]))
          ) * 100
      
      # Error budget calculation
      - record: slo:error_budget:remaining
        expr: |
          (
            1 - (
              (1 - (sli:system_availability:rate5m / 100)) /
              (1 - 0.9995)
            )
          ) * 100
```

### 1.2 Performance SLOs

| Metric | P50 Target | P95 Target | P99 Target | Critical Threshold |
|--------|------------|------------|------------|-------------------|
| **Intent Processing Latency** | < 1.5s | < 3s | < 5s | > 10s |
| **LLM Response Time** | < 800ms | < 2s | < 5s | > 8s |
| **RAG Query Latency** | < 200ms | < 500ms | < 1s | > 2s |
| **API Gateway Response** | < 100ms | < 300ms | < 500ms | > 1s |
| **Database Query Time** | < 50ms | < 150ms | < 300ms | > 500ms |

**Performance SLI Measurement Strategy:**
```bash
#!/bin/bash
# sli-measurement-script.sh

# Define PromQL queries using heredocs for clarity
INTENT_P50_QUERY=$(cat <<'EOF'
histogram_quantile(0.5, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))
EOF
)

INTENT_P95_QUERY=$(cat <<'EOF'
histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))
EOF
)

INTENT_P99_QUERY=$(cat <<'EOF'
histogram_quantile(0.99, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))
EOF
)

LLM_P50_QUERY=$(cat <<'EOF'
histogram_quantile(0.5, sum(rate(llm_request_duration_seconds_bucket[5m])) by (le))
EOF
)

LLM_P95_QUERY=$(cat <<'EOF'
histogram_quantile(0.95, sum(rate(llm_request_duration_seconds_bucket[5m])) by (le))
EOF
)

LLM_P99_QUERY=$(cat <<'EOF'
histogram_quantile(0.99, sum(rate(llm_request_duration_seconds_bucket[5m])) by (le))
EOF
)

# Intent Processing Latency (P50, P95, P99)
INTENT_P50=$(promtool query instant "$INTENT_P50_QUERY")
INTENT_P95=$(promtool query instant "$INTENT_P95_QUERY")
INTENT_P99=$(promtool query instant "$INTENT_P99_QUERY")

# LLM Response Time
LLM_P50=$(promtool query instant "$LLM_P50_QUERY")
LLM_P95=$(promtool query instant "$LLM_P95_QUERY")
LLM_P99=$(promtool query instant "$LLM_P99_QUERY")

# Generate SLO report
cat << EOF > /tmp/slo-report-$(date +%Y%m%d).json
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "intent_processing": {
    "p50": ${INTENT_P50},
    "p95": ${INTENT_P95},
    "p99": ${INTENT_P99}
  },
  "llm_service": {
    "p50": ${LLM_P50},
    "p95": ${LLM_P95},
    "p99": ${LLM_P99}
  }
}
EOF
```

### 1.3 Error Budget Calculations

**Error Budget Tracking:**
```yaml
# error-budget-alerts.yaml
groups:
  - name: error_budget_alerts
    rules:
      # Critical: Error budget < 10%
      - alert: ErrorBudgetCritical
        expr: slo:error_budget:remaining < 10
        for: 0s
        labels:
          severity: critical
          team: sre
        annotations:
          summary: "Error budget critically low"
          description: "Error budget is {{ $value }}% remaining. Immediate action required."
          runbook_url: "https://docs.internal/runbooks/error-budget-critical"
      
      # Warning: Error budget < 25%
      - alert: ErrorBudgetWarning
        expr: slo:error_budget:remaining < 25
        for: 5m
        labels:
          severity: warning
          team: sre
        annotations:
          summary: "Error budget low"
          description: "Error budget is {{ $value }}% remaining. Review ongoing work."
          runbook_url: "https://docs.internal/runbooks/error-budget-warning"
      
      # Burn rate alerts
      - alert: ErrorBudgetBurnRateHigh
        expr: |
          (
            1 - sli:system_availability:rate5m / 100
          ) > (
            14.4 * (1 - 0.9995)
          )
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error budget burn rate"
          description: "Current burn rate will exhaust monthly budget in < 2 days"
```

---

## Monitoring and Alerting

### 2.1 Complete Alert Rules Configuration

**Critical System Alerts:**
```yaml
# critical-system-alerts.yaml
groups:
  - name: nephoran_critical_alerts
    interval: 30s
    rules:
      # System availability
      - alert: SystemDown
        expr: up{job="nephoran-api-gateway"} == 0
        for: 30s
        labels:
          severity: critical
          team: sre
          page: "true"
        annotations:
          summary: "Nephoran system is down"
          description: "API Gateway is unreachable"
          runbook_url: "https://docs.internal/runbooks/system-down"
          escalation: "immediate"
      
      # High memory usage
      - alert: HighMemoryUsage
        expr: |
          (
            container_memory_working_set_bytes{container!="",container!="POD"} /
            container_spec_memory_limit_bytes{container!="",container!="POD"} * 100
          ) > 85
        for: 5m
        labels:
          severity: warning
          component: "{{ $labels.container }}"
        annotations:
          summary: "High memory usage in {{ $labels.container }}"
          description: "Memory usage is {{ $value }}% in {{ $labels.container }}"
          runbook_url: "https://docs.internal/runbooks/high-memory"
      
      # High CPU usage
      - alert: HighCPUUsage
        expr: |
          rate(container_cpu_usage_seconds_total{container!="",container!="POD"}[5m]) * 100 > 80
        for: 10m
        labels:
          severity: warning
          component: "{{ $labels.container }}"
        annotations:
          summary: "High CPU usage in {{ $labels.container }}"
          description: "CPU usage is {{ $value }}% in {{ $labels.container }}"
          runbook_url: "https://docs.internal/runbooks/high-cpu"
      
      # Network intent processing failures
      - alert: IntentProcessingFailure
        expr: |
          rate(intent_processing_errors_total[5m]) > 0.1
        for: 3m
        labels:
          severity: critical
          component: intent-controller
        annotations:
          summary: "Intent processing failures detected"
          description: "Error rate: {{ $value }} failures/second"
          runbook_url: "https://docs.internal/runbooks/intent-failures"
      
      # O-RAN interface failures
      - alert: ORANInterfaceDown
        expr: |
          sum(up{job=~"oran-.*"}) / count(up{job=~"oran-.*"}) < 0.8
        for: 2m
        labels:
          severity: critical
          component: oran
        annotations:
          summary: "O-RAN interfaces degraded"
          description: "{{ $value }}% of O-RAN interfaces are down"
          runbook_url: "https://docs.internal/runbooks/oran-interface-down"
      
      # Weaviate vector database issues
      - alert: WeaviateUnavailable
        expr: up{job="weaviate"} == 0
        for: 1m
        labels:
          severity: critical
          component: weaviate
        annotations:
          summary: "Weaviate vector database is down"
          description: "RAG system will be impacted"
          runbook_url: "https://docs.internal/runbooks/weaviate-down"
      
      # Kubernetes cluster health
      - alert: KubernetesNodeNotReady
        expr: kube_node_status_condition{condition="Ready",status="true"} == 0
        for: 5m
        labels:
          severity: critical
          component: kubernetes
        annotations:
          summary: "Kubernetes node {{ $labels.node }} not ready"
          description: "Node has been not ready for more than 5 minutes"
          runbook_url: "https://docs.internal/runbooks/k8s-node-not-ready"
```

**Business Logic Alerts:**
```yaml
# business-logic-alerts.yaml
groups:
  - name: nephoran_business_alerts
    interval: 60s
    rules:
      # Cost management
      - alert: HighLLMCosts
        expr: |
          rate(llm_token_costs_usd_total[1h]) * 24 > 1000
        for: 15m
        labels:
          severity: warning
          team: finance
        annotations:
          summary: "High LLM operational costs"
          description: "Daily cost projection: ${{ $value }}"
          runbook_url: "https://docs.internal/runbooks/high-costs"
      
      # Intent processing volume
      - alert: IntentVolumeAnomaly
        expr: |
          abs(
            rate(intent_processing_total[10m]) -
            avg_over_time(rate(intent_processing_total[10m])[24h:10m])
          ) > 
          2 * stddev_over_time(rate(intent_processing_total[10m])[24h:10m])
        for: 10m
        labels:
          severity: info
          component: analytics
        annotations:
          summary: "Unusual intent processing volume"
          description: "Volume deviation: {{ $value }} from 24h average"
      
      # Network function deployment success rate
      - alert: NetworkFunctionDeploymentFailure
        expr: |
          (
            rate(network_function_deployments_failed_total[10m]) /
            rate(network_function_deployments_total[10m])
          ) > 0.1
        for: 5m
        labels:
          severity: warning
          component: deployment
        annotations:
          summary: "High network function deployment failure rate"
          description: "{{ $value }}% of deployments are failing"
          runbook_url: "https://docs.internal/runbooks/deployment-failures"
```

### 2.2 Grafana Dashboard Specifications

**Executive Dashboard Configuration:**
```json
{
  "dashboard": {
    "title": "Nephoran Intent Operator - Executive Dashboard",
    "tags": ["nephoran", "executive", "slo"],
    "refresh": "5m",
    "panels": [
      {
        "title": "System Availability (SLO: 99.95%)",
        "type": "stat",
        "targets": [
          {
            "expr": "sli:system_availability:rate5m",
            "legendFormat": "Current Availability"
          }
        ],
        "fieldConfig": {
          "overrides": [
            {
              "matcher": {"id": "byName", "options": "Current Availability"},
              "properties": [
                {"id": "color", "value": {"mode": "thresholds"}},
                {"id": "thresholds", "value": {
                  "steps": [
                    {"color": "red", "value": 0},
                    {"color": "yellow", "value": 99.5},
                    {"color": "green", "value": 99.95}
                  ]
                }}
              ]
            }
          ]
        }
      },
      {
        "title": "Intent Processing Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(intent_processing_total[5m])",
            "legendFormat": "Intents/sec"
          }
        ]
      },
      {
        "title": "Error Budget Remaining",
        "type": "gauge",
        "targets": [
          {
            "expr": "slo:error_budget:remaining",
            "legendFormat": "Error Budget %"
          }
        ],
        "fieldConfig": {
          "min": 0,
          "max": 100,
          "thresholds": {
            "steps": [
              {"color": "red", "value": 0},
              {"color": "yellow", "value": 10},
              {"color": "green", "value": 25}
            ]
          }
        }
      },
      {
        "title": "Cost Per Intent (24h trend)",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(llm_token_costs_usd_total[1h]) / rate(intent_processing_total[1h])",
            "legendFormat": "$/intent"
          }
        ]
      }
    ]
  }
}
```

**Technical Operations Dashboard:**
```json
{
  "dashboard": {
    "title": "Nephoran Intent Operator - Technical Operations",
    "tags": ["nephoran", "technical", "operations"],
    "refresh": "30s",
    "panels": [
      {
        "title": "Component Health Status",
        "type": "table",
        "targets": [
          {
            "expr": "up{job=~\"llm-processor|rag-api|oran-adaptor|nephio-bridge\"}",
            "legendFormat": "{{job}}",
            "format": "table",
            "instant": true
          }
        ],
        "transformations": [
          {
            "id": "organize",
            "options": {
              "excludeByName": {"Time": true},
              "renameByName": {
                "job": "Component",
                "Value": "Status"
              }
            }
          }
        ],
        "overrides": [
          {
            "matcher": {"id": "byName", "options": "Status"},
            "properties": [
              {
                "id": "custom.displayMode",
                "value": "color-background"
              },
              {
                "id": "mappings",
                "value": [
                  {"options": {"1": {"color": "green", "text": "UP"}}, "type": "value"},
                  {"options": {"0": {"color": "red", "text": "DOWN"}}, "type": "value"}
                ]
              }
            ]
          }
        ]
      },
      {
        "title": "Request Latency by Service",
        "type": "heatmap",
        "targets": [
          {
            "expr": "sum(rate(http_request_duration_seconds_bucket[5m])) by (le, service)",
            "legendFormat": "{{service}}"
          }
        ]
      },
      {
        "title": "Active Network Intents by Status",
        "type": "piechart",
        "targets": [
          {
            "expr": "count by (status) (kube_customresource_info{customresource_kind=\"NetworkIntent\"})",
            "legendFormat": "{{status}}"
          }
        ]
      }
    ]
  }
}
```

### 2.3 Alert Routing and Escalation

**AlertManager Configuration:**
```yaml
# alertmanager-config.yaml
global:
  smtp_smarthost: 'smtp.company.com:587'
  smtp_from: 'nephoran-alerts@company.com'

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'component']

route:
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'default'
  
  routes:
    # Critical system alerts - immediate page
    - match:
        severity: critical
        page: "true"
      receiver: 'pager-critical'
      group_wait: 10s
      repeat_interval: 30m
    
    # SRE team alerts
    - match:
        team: sre
      receiver: 'sre-team'
      group_interval: 2m
    
    # Business hours only
    - match:
        severity: warning
      receiver: 'business-hours'
      active_time_intervals:
        - business-hours
    
    # Finance team for cost alerts
    - match:
        team: finance
      receiver: 'finance-team'

receivers:
  - name: 'default'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/XXXXXXXXX'
        channel: '#nephoran-alerts'
        
  - name: 'pager-critical'
    pagerduty_configs:
      - routing_key: 'PAGERDUTY_INTEGRATION_KEY'
        severity: '{{ .GroupLabels.severity }}'
        description: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/XXXXXXXXX'
        channel: '#nephoran-critical'
        text: 'CRITICAL: {{ .GroupLabels.alertname }}'
        
  - name: 'sre-team'
    email_configs:
      - to: 'sre-team@company.com'
        subject: '[SRE] {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Runbook: {{ .Annotations.runbook_url }}
          {{ end }}
          
  - name: 'finance-team'
    email_configs:
      - to: 'finance-team@company.com'
        subject: '[COST ALERT] {{ .GroupLabels.alertname }}'

time_intervals:
  - name: business-hours
    time_intervals:
      - times:
        - start_time: '09:00'
          end_time: '17:00'
        weekdays: ['monday:friday']
        location: 'America/New_York'
```

---

## Upgrade Procedures

### 3.1 Zero-Downtime Upgrade Strategy

**Upgrade Preparation Checklist:**
```bash
#!/bin/bash
# upgrade-preparation.sh

set -euo pipefail

UPGRADE_VERSION=${1:-""}
ROLLBACK_VERSION=$(kubectl get deployment llm-processor -n nephoran-system -o jsonpath='{.spec.template.spec.containers[0].image}' | cut -d: -f2)
UPGRADE_LOG="/tmp/nephoran-upgrade-$(date +%Y%m%d-%H%M%S).log"

echo "🚀 Starting Nephoran Intent Operator upgrade preparation" | tee -a $UPGRADE_LOG

# 1. Validate current system health
echo "📊 Validating system health..."
UNHEALTHY_PODS=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
if [ $UNHEALTHY_PODS -gt 0 ]; then
    echo "❌ Found $UNHEALTHY_PODS unhealthy pods. Aborting upgrade." | tee -a $UPGRADE_LOG
    kubectl get pods -n nephoran-system | grep -v Running | tee -a $UPGRADE_LOG
    exit 1
fi

# 2. Check active intents
ACTIVE_INTENTS=$(kubectl get networkintents -A --no-headers | grep -c "Processing" || echo "0")
if [ $ACTIVE_INTENTS -gt 10 ]; then
    echo "⚠️ Warning: $ACTIVE_INTENTS intents currently processing. Consider waiting." | tee -a $UPGRADE_LOG
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Upgrade aborted by user." | tee -a $UPGRADE_LOG
        exit 1
    fi
fi

# 3. Create backup
echo "💾 Creating pre-upgrade backup..."
kubectl create configmap upgrade-rollback-info-$(date +%Y%m%d-%H%M%S) \
  -n nephoran-system \
  --from-literal=previous_version="$ROLLBACK_VERSION" \
  --from-literal=upgrade_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --from-literal=operator="$(whoami)"

# 4. Scale down non-essential services
echo "📉 Scaling down non-essential services..."
kubectl scale deployment grafana --replicas=0 -n nephoran-monitoring
kubectl scale deployment jaeger-collector --replicas=1 -n nephoran-monitoring

# 5. Enable maintenance mode
echo "🔧 Enabling maintenance mode..."
kubectl patch configmap maintenance-config -n nephoran-system \
  --patch '{"data":{"maintenance_mode":"true","maintenance_start":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}}'

echo "✅ Upgrade preparation completed successfully!" | tee -a $UPGRADE_LOG
echo "📝 Upgrade log: $UPGRADE_LOG"
```

**Rolling Update Execution:**
```bash
#!/bin/bash
# execute-rolling-upgrade.sh

set -euo pipefail

UPGRADE_VERSION=${1:-""}
UPDATE_STRATEGY=${2:-"rolling"}  # rolling, canary, blue-green
UPGRADE_LOG="/tmp/nephoran-upgrade-$(date +%Y%m%d-%H%M%S).log"

if [ -z "$UPGRADE_VERSION" ]; then
    echo "Usage: $0 <version> [strategy]"
    echo "Strategies: rolling (default), canary, blue-green"
    exit 1
fi

echo "🔄 Starting $UPDATE_STRATEGY upgrade to version $UPGRADE_VERSION" | tee -a $UPGRADE_LOG

case $UPDATE_STRATEGY in
    "rolling")
        perform_rolling_upgrade
        ;;
    "canary")
        perform_canary_upgrade
        ;;
    "blue-green")
        perform_blue_green_upgrade
        ;;
    *)
        echo "❌ Unknown upgrade strategy: $UPDATE_STRATEGY"
        exit 1
        ;;
esac

function perform_rolling_upgrade() {
    echo "🔄 Performing rolling upgrade..."
    
    # Update images with rolling strategy
    COMPONENTS=("llm-processor" "rag-api" "oran-adaptor" "nephio-bridge")
    
    for component in "${COMPONENTS[@]}"; do
        echo "📦 Upgrading $component to $UPGRADE_VERSION..."
        
        kubectl set image deployment/$component \
            $component=nephoran/$component:$UPGRADE_VERSION \
            -n nephoran-system
        
        echo "⏳ Waiting for $component rollout..."
        kubectl rollout status deployment/$component -n nephoran-system --timeout=300s
        
        # Verify health after each component
        sleep 30
        check_component_health $component
    done
}

function perform_canary_upgrade() {
    echo "🐤 Performing canary upgrade..."
    
    # Deploy canary version alongside current
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-processor-canary
  namespace: nephoran-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: llm-processor
      version: canary
  template:
    spec:
      containers:
      - name: llm-processor
        image: nephoran/llm-processor:$UPGRADE_VERSION
        # ... rest of container spec
EOF

    # Configure traffic split (10% to canary)
    kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: llm-processor-canary
  namespace: nephoran-system
spec:
  http:
  - match:
    - headers:
        canary:
          exact: "true"
    route:
    - destination:
        host: llm-processor
        subset: canary
      weight: 100
  - route:
    - destination:
        host: llm-processor
        subset: stable
      weight: 90
    - destination:
        host: llm-processor
        subset: canary
      weight: 10
EOF

    # Monitor canary for 10 minutes
    monitor_canary_health
    
    # If successful, promote canary
    promote_canary
}

function check_component_health() {
    local component=$1
    echo "🏥 Checking health of $component..."
    
    # Wait for pods to be ready
    kubectl wait --for=condition=ready pod -l app=$component -n nephoran-system --timeout=300s
    
    # Check health endpoint
    local pod=$(kubectl get pods -n nephoran-system -l app=$component -o jsonpath='{.items[0].metadata.name}')
    local health_check=$(kubectl exec $pod -n nephoran-system -- curl -s -f http://localhost:8080/healthz || echo "FAILED")
    
    if [ "$health_check" == "FAILED" ]; then
        echo "❌ Health check failed for $component"
        trigger_rollback
        exit 1
    fi
    
    echo "✅ $component is healthy"
}

function trigger_rollback() {
    echo "🔄 Triggering automatic rollback..."
    
    for component in "${COMPONENTS[@]}"; do
        echo "⏪ Rolling back $component..."
        kubectl rollout undo deployment/$component -n nephoran-system
        kubectl rollout status deployment/$component -n nephoran-system --timeout=300s
    done
    
    # Disable maintenance mode
    kubectl patch configmap maintenance-config -n nephoran-system \
        --patch '{"data":{"maintenance_mode":"false","maintenance_end":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}}'
    
    echo "❌ Rollback completed. System restored to previous version."
    exit 1
}
```

### 3.2 Version Compatibility Matrix

| Component | v2.1.x | v2.2.x | v2.3.x | Breaking Changes |
|-----------|---------|---------|---------|------------------|
| **LLM Processor** | ✅ | ✅ | ✅ | None |
| **RAG API** | ✅ | ✅ | ⚠️ | Schema changes in v2.3.0 |
| **O-RAN Adaptor** | ✅ | ⚠️ | ❌ | API changes in v2.2.0, removed in v2.3.0 |
| **Nephio Bridge** | ✅ | ✅ | ✅ | None |
| **Weaviate** | 1.20.x | 1.21.x | 1.22.x | Schema migration required |
| **Kubernetes** | 1.25+ | 1.26+ | 1.27+ | None |

**Compatibility Check Script:**
```bash
#!/bin/bash
# compatibility-check.sh

check_version_compatibility() {
    local current_version=$(kubectl get deployment llm-processor -n nephoran-system -o jsonpath='{.spec.template.spec.containers[0].image}' | cut -d: -f2)
    local target_version=$1
    
    echo "Current version: $current_version"
    echo "Target version: $target_version"
    
    # Check major version compatibility
    local current_major=$(echo $current_version | cut -d. -f1)
    local target_major=$(echo $target_version | cut -d. -f1)
    
    if [ $target_major -gt $current_major ]; then
        echo "⚠️ Major version upgrade detected. Manual intervention may be required."
        return 1
    fi
    
    # Check for breaking changes
    case "$target_version" in
        "2.2."*)
            echo "⚠️ v2.2.x introduces O-RAN Adaptor API changes"
            ;;
        "2.3."*)
            echo "❌ v2.3.x removes O-RAN Adaptor. Migration required."
            return 1
            ;;
    esac
    
    return 0
}
```

### 3.3 Database Migration Procedures

**Weaviate Schema Migration:**
```bash
#!/bin/bash
# weaviate-schema-migration.sh

migrate_weaviate_schema() {
    local backup_name="weaviate-backup-$(date +%Y%m%d-%H%M%S)"
    
    echo "📊 Starting Weaviate schema migration..."
    
    # 1. Create backup
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -X POST "http://localhost:8080/v1/backups/filesystem" \
        -H "Content-Type: application/json" \
        -d '{"id":"'$backup_name'","include":["NetworkIntent","TelecomKnowledge"]}'
    
    # 2. Export current schema
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -s http://localhost:8080/v1/schema > /tmp/current-schema.json
    
    # 3. Apply new schema
    kubectl apply -f migrations/weaviate-schema-v${TARGET_VERSION}.yaml
    
    # 4. Verify migration
    local migration_status=$(kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -s http://localhost:8080/v1/schema | jq -r '.classes | length')
    
    if [ $migration_status -lt 2 ]; then
        echo "❌ Schema migration failed. Restoring backup..."
        restore_weaviate_backup $backup_name
        exit 1
    fi
    
    echo "✅ Weaviate schema migration completed successfully"
}

restore_weaviate_backup() {
    local backup_name=$1
    echo "🔄 Restoring Weaviate backup: $backup_name"
    
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -X POST "http://localhost:8080/v1/backups/filesystem/$backup_name/restore" \
        -H "Content-Type: application/json" \
        -d '{"include":["NetworkIntent","TelecomKnowledge"]}'
}
```

---

## Operational Runbooks

### 4.1 Incident Response Procedures

**P1 Critical Incident Response:**
```bash
#!/bin/bash
# p1-incident-response.sh

# Incident ID generation
INCIDENT_ID="INC-$(date +%Y%m%d-%H%M%S)"
INCIDENT_LOG="/tmp/incident-$INCIDENT_ID.log"
INCIDENT_CHANNEL="nephoran-incidents"

echo "🚨 P1 INCIDENT RESPONSE INITIATED: $INCIDENT_ID" | tee -a $INCIDENT_LOG

# 1. Immediate assessment
assess_system_status() {
    echo "📊 System Status Assessment:" | tee -a $INCIDENT_LOG
    
    # Check all critical components
    kubectl get pods -n nephoran-system -o wide | tee -a $INCIDENT_LOG
    kubectl get nodes -o wide | tee -a $INCIDENT_LOG
    
    # Check external dependencies
    curl -s -o /dev/null -w "LLM API: %{http_code} (%{time_total}s)\n" https://api.openai.com/v1/models | tee -a $INCIDENT_LOG
    kubectl exec deployment/weaviate -n nephoran-system -- curl -s -f http://localhost:8080/v1/live | tee -a $INCIDENT_LOG
    
    # Check recent alerts
    promtool query instant 'ALERTS{alertstate="firing"}' | tee -a $INCIDENT_LOG
}

# 2. Immediate containment
immediate_containment() {
    echo "🛡️ Immediate Containment Actions:" | tee -a $INCIDENT_LOG
    
    # Enable circuit breakers
    kubectl patch configmap llm-config -n nephoran-system \
        --patch '{"data":{"circuit_breaker_enabled":"true","circuit_breaker_threshold":"1"}}'
    
    # Scale up healthy components
    kubectl scale deployment llm-processor --replicas=5 -n nephoran-system
    kubectl scale deployment rag-api --replicas=3 -n nephoran-system
    
    # Enable maintenance page if system is completely down
    local system_health=$(kubectl get pods -n nephoran-system --no-headers | grep Running | wc -l)
    if [ $system_health -eq 0 ]; then
        enable_maintenance_page
    fi
}

# 3. Communication
incident_communication() {
    # Post to incident channel
    curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"🚨 P1 INCIDENT '${INCIDENT_ID}': System impact detected. War room: #'${INCIDENT_CHANNEL}'-'${INCIDENT_ID}'"}' \
        $SLACK_WEBHOOK_URL
    
    # Page on-call
    curl -X POST https://events.pagerduty.com/v2/enqueue \
        -H 'Authorization: Token '$PAGERDUTY_TOKEN \
        -H 'Content-Type: application/json' \
        -d '{
            "routing_key": "'$PAGERDUTY_ROUTING_KEY'",
            "event_action": "trigger",
            "dedup_key": "'$INCIDENT_ID'",
            "payload": {
                "summary": "P1 Incident: Nephoran System Impact",
                "source": "nephoran-monitoring",
                "severity": "critical"
            }
        }'
}

# 4. Recovery actions
recovery_actions() {
    echo "🔧 Recovery Actions:" | tee -a $INCIDENT_LOG
    
    # Check if this is a known issue
    check_known_issues
    
    # Try automated recovery procedures
    execute_recovery_runbooks
    
    # If automation fails, escalate
    if ! verify_system_recovery; then
        escalate_to_engineering
    fi
}

# Execute incident response
assess_system_status
immediate_containment
incident_communication
recovery_actions

echo "📝 Incident log saved to: $INCIDENT_LOG"
```

**Common Troubleshooting Scenarios:**

1. **High Intent Processing Latency:**
```bash
# Latency Investigation Runbook
#!/bin/bash
# runbook-high-latency.sh

# Check component response times
echo "🔍 Investigating high latency..."

# Define PromQL queries
LLM_P95_QUERY='histogram_quantile(0.95, sum(rate(llm_request_duration_seconds_bucket[5m])) by (le))'
RAG_P95_QUERY='histogram_quantile(0.95, sum(rate(rag_query_latency_seconds_bucket[5m])) by (le))'

# LLM service latency
LLM_P95=$(promtool query instant "$LLM_P95_QUERY")
echo "LLM P95 Latency: ${LLM_P95}s"

# RAG query latency  
RAG_P95=$(promtool query instant "$RAG_P95_QUERY")
echo "RAG P95 Latency: ${RAG_P95}s"

# Check for resource constraints
kubectl top pods -n nephoran-system --sort-by=cpu
kubectl top pods -n nephoran-system --sort-by=memory

# Check connection pools
kubectl exec deployment/rag-api -n nephoran-system -- \
    curl -s http://localhost:8080/metrics | grep connection_pool

# Mitigation actions
if (( $(echo "$LLM_P95 > 3" | bc -l) )); then
    echo "🚀 Scaling LLM processor..."
    kubectl scale deployment llm-processor --replicas=10 -n nephoran-system
fi

if (( $(echo "$RAG_P95 > 1" | bc -l) )); then
    echo "🔄 Clearing RAG cache..."
    kubectl exec deployment/rag-api -n nephoran-system -- \
        curl -X POST http://localhost:8080/admin/cache/clear
fi
```

2. **Vector Database Issues:**
```bash
# Weaviate Troubleshooting Runbook
#!/bin/bash
# runbook-weaviate-issues.sh

# Check Weaviate health
WEAVIATE_LIVE=$(kubectl exec deployment/weaviate -n nephoran-system -- curl -s -f http://localhost:8080/v1/live || echo "FAILED")
WEAVIATE_READY=$(kubectl exec deployment/weaviate -n nephoran-system -- curl -s -f http://localhost:8080/v1/ready || echo "FAILED")

if [ "$WEAVIATE_LIVE" == "FAILED" ]; then
    echo "❌ Weaviate is not responding. Restarting..."
    kubectl rollout restart deployment/weaviate -n nephoran-system
    kubectl rollout status deployment/weaviate -n nephoran-system --timeout=300s
fi

# Check schema integrity
SCHEMA_CLASSES=$(kubectl exec deployment/weaviate -n nephoran-system -- \
    curl -s http://localhost:8080/v1/schema | jq -r '.classes | length')

if [ "$SCHEMA_CLASSES" -lt 2 ]; then
    echo "❌ Schema corruption detected. Restoring from backup..."
    restore_weaviate_schema
fi

# Check disk space
kubectl exec deployment/weaviate -n nephoran-system -- df -h /var/lib/weaviate

# Check memory usage
kubectl exec deployment/weaviate -n nephoran-system -- \
    curl -s http://localhost:8080/v1/nodes | jq '.nodes[0].stats.memoryUsagePercentage'
```

### 4.2 Performance Tuning Guidelines

**LLM Service Optimization:**
```bash
# llm-performance-tuning.sh

# Batch size optimization
kubectl patch configmap llm-config -n nephoran-system \
    --patch '{"data":{
        "batch_size":"32",
        "batch_timeout":"500ms",
        "max_concurrent_requests":"100",
        "circuit_breaker_timeout":"30s"
    }}'

# Connection pool tuning
kubectl patch deployment llm-processor -n nephoran-system \
    --patch '{
        "spec": {
            "template": {
                "spec": {
                    "containers": [{
                        "name": "llm-processor",
                        "env": [
                            {"name": "HTTP_MAX_IDLE_CONNS", "value": "100"},
                            {"name": "HTTP_MAX_IDLE_CONNS_PER_HOST", "value": "10"},
                            {"name": "HTTP_TIMEOUT", "value": "30s"}
                        ]
                    }]
                }
            }
        }
    }'

# Memory optimization
kubectl patch deployment llm-processor -n nephoran-system \
    --patch '{
        "spec": {
            "template": {
                "spec": {
                    "containers": [{
                        "name": "llm-processor",
                        "resources": {
                            "requests": {
                                "memory": "512Mi",
                                "cpu": "500m"
                            },
                            "limits": {
                                "memory": "2Gi", 
                                "cpu": "2000m"
                            }
                        },
                        "env": [
                            {"name": "GOGC", "value": "400"},
                            {"name": "GOMEMLIMIT", "value": "1800MiB"}
                        ]
                    }]
                }
            }
        }
    }'
```

**RAG System Optimization:**
```bash
# rag-performance-tuning.sh

# Cache configuration optimization
kubectl patch configmap rag-config -n nephoran-system \
    --patch '{"data":{
        "memory_cache_size": "512MB",
        "memory_cache_ttl": "1h",
        "redis_cache_ttl": "24h",
        "vector_cache_size": "1000",
        "query_cache_enabled": "true"
    }}'

# Weaviate optimization
kubectl exec deployment/weaviate -n nephoran-system -- \
    curl -X PUT http://localhost:8080/v1/schema/NetworkIntent \
    -H "Content-Type: application/json" \
    -d '{
        "vectorIndexConfig": {
            "ef": 128,
            "efConstruction": 256,
            "maxConnections": 32
        }
    }'

# Connection pool optimization
kubectl patch deployment rag-api -n nephoran-system \
    --patch '{
        "spec": {
            "template": {
                "spec": {
                    "containers": [{
                        "name": "rag-api",
                        "env": [
                            {"name": "WEAVIATE_POOL_SIZE", "value": "20"},
                            {"name": "WEAVIATE_POOL_TIMEOUT", "value": "30s"},
                            {"name": "REDIS_POOL_SIZE", "value": "10"}
                        ]
                    }]
                }
            }
        }
    }'
```

### 4.3 Capacity Planning Recommendations

**Resource Scaling Guidelines:**
```bash
# capacity-planning.sh

calculate_resource_requirements() {
    local intent_rate=${1:-10}  # intents per second
    local peak_multiplier=${2:-3}
    
    echo "📊 Capacity Planning for ${intent_rate} intents/sec (${peak_multiplier}x peak)"
    
    # LLM Processor scaling
    local llm_replicas=$(echo "($intent_rate * $peak_multiplier) / 2" | bc)
    echo "LLM Processor replicas needed: $llm_replicas"
    
    # RAG API scaling
    local rag_replicas=$(echo "($intent_rate * $peak_multiplier) / 5" | bc)
    echo "RAG API replicas needed: $rag_replicas"
    
    # Memory requirements
    local memory_per_intent="50Mi"
    local total_memory=$(echo "$intent_rate * 50 * $peak_multiplier" | bc)
    echo "Total memory needed: ${total_memory}Mi"
    
    # Generate HPA configurations
    generate_hpa_config $intent_rate $peak_multiplier
}

generate_hpa_config() {
    local base_rate=$1
    local peak_multiplier=$2
    
    cat << EOF > /tmp/hpa-config.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-processor
  namespace: nephoran-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
  minReplicas: $(echo "$base_rate / 2" | bc)
  maxReplicas: $(echo "$base_rate * $peak_multiplier / 2" | bc)
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Pods
    pods:
      metric:
        name: llm_requests_per_second
      target:
        type: AverageValue
        averageValue: "2"
EOF

    echo "Generated HPA configuration: /tmp/hpa-config.yaml"
}
```

---

## Emergency Response

### 5.1 Emergency Contact Matrix

| Incident Type | Primary Contact | Secondary Contact | Escalation |
|---------------|----------------|-------------------|------------|
| **System Down** | SRE On-Call (+1-555-SRE-TEAM) | Engineering Manager | CTO |
| **Security Breach** | Security Team (+1-555-SEC-TEAM) | CISO | Legal/Compliance |
| **Data Loss** | Data Protection Officer | Backup Operations | CTO |
| **High Costs** | Finance Team | Engineering Director | CFO |
| **Compliance Issue** | Compliance Officer | Legal Team | General Counsel |

### 5.2 Emergency Procedures Decision Tree

```
INCIDENT DETECTED
        |
        v
   Is system responding?
        |
    Yes  |  No
        |   |
        |   v
        |   SYSTEM DOWN (P1)
        |   - Page on-call immediately
        |   - Enable maintenance page  
        |   - Execute disaster recovery
        |   - Notify executives
        |
        v
   Performance degraded?
        |
    Yes  |  No
        |   |
        |   v
        |   INVESTIGATE (P3)
        |   - Check metrics
        |   - Review logs
        |   - Create ticket
        |
        v
   Error rate > 10%?
        |
    Yes  |  No
        |   |
        |   v
        |   PERFORMANCE ISSUE (P2)
        |   - Scale services
        |   - Check resources
        |   - Monitor closely
        |
        v
   DATA CORRUPTION (P1)
   - Stop processing
   - Isolate affected systems
   - Restore from backup
   - Investigate root cause
```

### 5.3 Disaster Recovery Procedures

**Complete System Recovery:**
```bash
#!/bin/bash
# disaster-recovery.sh

DR_SCENARIO=${1:-"complete-failure"}
RECOVERY_LOG="/tmp/dr-$(date +%Y%m%d-%H%M%S).log"

echo "🆘 DISASTER RECOVERY INITIATED: $DR_SCENARIO" | tee -a $RECOVERY_LOG

case $DR_SCENARIO in
    "complete-failure")
        complete_system_recovery
        ;;
    "data-corruption")
        data_recovery
        ;;
    "security-breach")
        security_incident_recovery
        ;;
    *)
        echo "Unknown DR scenario: $DR_SCENARIO"
        exit 1
        ;;
esac

complete_system_recovery() {
    echo "🔄 Complete system recovery initiated..." | tee -a $RECOVERY_LOG
    
    # 1. Restore infrastructure
    terraform apply -var="disaster_recovery=true" infra/
    
    # 2. Restore Kubernetes cluster
    kubectl apply -f disaster-recovery/cluster-backup.yaml
    
    # 3. Restore application state
    restore_application_state
    
    # 4. Verify system health
    verify_disaster_recovery
}

restore_application_state() {
    echo "📦 Restoring application state..." | tee -a $RECOVERY_LOG
    
    # Restore Weaviate data
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -X POST "http://localhost:8080/v1/backups/filesystem/latest/restore" \
        -H "Content-Type: application/json" \
        -d '{"include":["NetworkIntent","TelecomKnowledge"]}'
    
    # Restore configuration
    kubectl apply -f backups/latest/configmaps/
    kubectl apply -f backups/latest/secrets/
    
    # Restore persistent data
    velero restore create --from-backup nephoran-daily-$(date -d "yesterday" +%Y%m%d)
}

verify_disaster_recovery() {
    echo "✅ Verifying disaster recovery..." | tee -a $RECOVERY_LOG
    
    # Wait for all pods to be ready
    kubectl wait --for=condition=ready pod --all -n nephoran-system --timeout=600s
    
    # Test critical paths
    test_intent_processing
    test_rag_queries
    test_oran_interfaces
    
    # Validate data integrity
    validate_data_integrity
    
    echo "✅ Disaster recovery completed successfully!" | tee -a $RECOVERY_LOG
}
```

---

## Performance Optimization

### 6.1 Performance Monitoring and Benchmarking

**Automated Performance Testing:**
```bash
#!/bin/bash
# performance-benchmark.sh

BENCHMARK_DURATION=${1:-"300"}  # 5 minutes default
CONCURRENT_USERS=${2:-"50"}
RESULTS_DIR="/tmp/perf-results-$(date +%Y%m%d-%H%M%S)"

mkdir -p $RESULTS_DIR

echo "🚀 Starting performance benchmark - Duration: ${BENCHMARK_DURATION}s, Users: $CONCURRENT_USERS"

# 1. Intent processing benchmark
hey -z ${BENCHMARK_DURATION}s -c $CONCURRENT_USERS \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -m POST \
    -d '{"intent":"Deploy high-availability 5G AMF with auto-scaling","priority":"high"}' \
    http://api.nephoran.local/v1/intents > $RESULTS_DIR/intent-processing.txt

# 2. RAG query benchmark  
hey -z ${BENCHMARK_DURATION}s -c $((CONCURRENT_USERS/2)) \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -m POST \
    -d '{"query":"5G AMF configuration best practices","limit":10}' \
    http://rag-api.nephoran.local/v1/search > $RESULTS_DIR/rag-queries.txt

# 3. Generate performance report
generate_performance_report $RESULTS_DIR

generate_performance_report() {
    local results_dir=$1
    
    cat << EOF > $results_dir/performance-report.md
# Performance Benchmark Report
Generated: $(date)

## Intent Processing Performance
$(cat $results_dir/intent-processing.txt | grep -A 10 "Summary:")

## RAG Query Performance  
$(cat $results_dir/rag-queries.txt | grep -A 10 "Summary:")

## Resource Utilization During Test
### CPU Usage
$(kubectl top nodes)

### Memory Usage
$(kubectl top pods -n nephoran-system --sort-by=memory)

## Recommendations
$(generate_performance_recommendations)
EOF

    echo "📊 Performance report generated: $results_dir/performance-report.md"
}

generate_performance_recommendations() {
    local intent_p95=$(cat $RESULTS_DIR/intent-processing.txt | grep "95%" | awk '{print $2}')
    local rag_p95=$(cat $RESULTS_DIR/rag-queries.txt | grep "95%" | awk '{print $2}')
    
    if (( $(echo "$intent_p95 > 3000" | bc -l) )); then
        echo "- Scale LLM processor replicas"
        echo "- Optimize batch processing"
        echo "- Consider caching strategy"
    fi
    
    if (( $(echo "$rag_p95 > 500" | bc -l) )); then
        echo "- Optimize vector database indexing"
        echo "- Increase cache hit rate"
        echo "- Scale RAG API replicas"
    fi
}
```

### 6.2 Optimization Strategies

**Memory Optimization:**
```bash
# memory-optimization.sh

# JVM-like Go memory optimization
kubectl patch deployment llm-processor -n nephoran-system \
    --patch '{
        "spec": {
            "template": {
                "spec": {
                    "containers": [{
                        "name": "llm-processor",
                        "env": [
                            {"name": "GOGC", "value": "200"},
                            {"name": "GOMEMLIMIT", "value": "1800MiB"},
                            {"name": "GOMAXPROCS", "value": "2"}
                        ]
                    }]
                }
            }
        }
    }'

# CPU optimization
kubectl patch deployment rag-api -n nephoran-system \
    --patch '{
        "spec": {
            "template": {
                "spec": {
                    "containers": [{
                        "name": "rag-api", 
                        "env": [
                            {"name": "GOMAXPROCS", "value": "4"},
                            {"name": "WORKER_POOL_SIZE", "value": "100"}
                        ]
                    }]
                }
            }
        }
    }'
```

---

## Troubleshooting Reference

### 7.1 Quick Diagnostic Commands

```bash
# System health check
kubectl get pods -n nephoran-system -o wide
kubectl get events -n nephoran-system --sort-by='.lastTimestamp'
kubectl top pods -n nephoran-system --sort-by=memory

# Performance metrics
INTENT_RATE_QUERY='rate(intent_processing_total[5m])'
LLM_P95_QUERY='histogram_quantile(0.95, sum(rate(llm_request_duration_seconds_bucket[5m])) by (le))'

promtool query instant "$INTENT_RATE_QUERY"
promtool query instant "$LLM_P95_QUERY"

# Log aggregation
kubectl logs -n nephoran-system -l app=llm-processor --tail=100
kubectl logs -n nephoran-system -l app=rag-api --since=1h

# Network connectivity
kubectl exec -it deployment/llm-processor -n nephoran-system -- curl -v https://api.openai.com/v1/models
kubectl exec -it deployment/rag-api -n nephoran-system -- curl -v http://weaviate:8080/v1/live
```

### 7.2 Common Issue Resolution

| Issue | Symptoms | Resolution | Prevention |
|-------|----------|------------|------------|
| **High Latency** | P95 > 5s | Scale replicas, check resources | Monitor SLOs, capacity planning |
| **Memory Leaks** | OOMKilled pods | Restart pods, tune GOGC | Memory profiling, load testing |
| **API Timeouts** | 504 responses | Check upstream services | Circuit breakers, timeouts |
| **Vector DB Slow** | RAG queries > 1s | Optimize indexes, clear cache | Regular maintenance, monitoring |
| **Token Costs** | Budget exceeded | Review usage patterns | Cost alerts, rate limiting |

---

**Document Version**: v2.1.0  
**Last Updated**: January 2025  
**Next Review**: March 2025  
**Owner**: SRE Team  

For questions about this operations handbook, contact: sre-team@company.com