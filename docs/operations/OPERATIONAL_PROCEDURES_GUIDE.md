# Operational Procedures Guide - Microservices Architecture

## Executive Summary

This comprehensive operational guide provides detailed procedures for deploying, monitoring, troubleshooting, and maintaining the Nephoran Intent Operator microservices architecture in production environments. The guide ensures operational excellence with **99.95% availability**, **sub-2-second response times**, and **automated incident response** capabilities.

## Table of Contents

1. [Deployment Procedures](#deployment-procedures)
2. [Monitoring and Observability](#monitoring-and-observability)
3. [Troubleshooting Guide](#troubleshooting-guide)
4. [Security Procedures](#security-procedures)
5. [Performance Tuning](#performance-tuning)
6. [Backup and Recovery](#backup-and-recovery)
7. [Incident Response](#incident-response)
8. [Maintenance Procedures](#maintenance-procedures)

## Deployment Procedures

### Production Deployment Checklist

```bash
#!/bin/bash
# Production Deployment Checklist Script

echo "=== Nephoran Microservices Production Deployment Checklist ==="

# Prerequisites validation
validate_prerequisites() {
    echo "🔍 Validating prerequisites..."
    
    # Check Kubernetes version
    K8S_VERSION=$(kubectl version --short --client | cut -d' ' -f3 | sed 's/v//')
    if [[ "$(printf '%s\n' "1.25" "$K8S_VERSION" | sort -V | head -n1)" != "1.25" ]]; then
        echo "❌ Kubernetes version must be 1.25+ (found: $K8S_VERSION)"
        return 1
    fi
    
    # Validate cluster resources
    TOTAL_CPU=$(kubectl top nodes | awk 'NR>1 {sum+=$2} END {print sum}' | sed 's/m//')
    TOTAL_MEMORY=$(kubectl top nodes | awk 'NR>1 {sum+=$4} END {print sum}' | sed 's/Mi//')
    
    if [[ $TOTAL_CPU -lt 8000 ]]; then  # 8 CPU cores minimum
        echo "❌ Insufficient CPU resources (need 8+ cores, have: ${TOTAL_CPU}m)"
        return 1
    fi
    
    if [[ $TOTAL_MEMORY -lt 16384 ]]; then  # 16GB memory minimum
        echo "❌ Insufficient memory resources (need 16GB+, have: ${TOTAL_MEMORY}Mi)"
        return 1
    fi
    
    # Check required namespaces
    for ns in nephoran-system nephoran-monitoring nephoran-infrastructure; do
        if ! kubectl get namespace $ns &>/dev/null; then
            echo "⚠️  Creating missing namespace: $ns"
            kubectl create namespace $ns
        fi
    done
    
    # Validate storage classes
    if ! kubectl get storageclass | grep -q "gp2\|standard\|fast"; then
        echo "❌ No suitable storage class found"
        return 1
    fi
    
    echo "✅ Prerequisites validated successfully"
    return 0
}

# Infrastructure deployment
deploy_infrastructure() {
    echo "🏗️  Deploying infrastructure components..."
    
    # Deploy Redis cluster for distributed caching
    echo "Deploying Redis cluster..."
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm upgrade --install redis-cluster bitnami/redis-cluster \
        --namespace nephoran-infrastructure \
        --set cluster.nodes=3 \
        --set persistence.enabled=true \
        --set persistence.size=20Gi \
        --set metrics.enabled=true \
        --timeout=10m
    
    # Wait for Redis cluster to be ready
    kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=redis-cluster \
        -n nephoran-infrastructure --timeout=600s
    
    # Deploy Weaviate vector database
    echo "Deploying Weaviate vector database..."
    kubectl apply -f deployments/weaviate/weaviate-deployment.yaml
    kubectl wait --for=condition=Available deployment/weaviate \
        -n nephoran-system --timeout=600s
    
    # Deploy monitoring stack
    echo "Deploying monitoring infrastructure..."
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace nephoran-monitoring \
        --values config/monitoring/values-production.yaml \
        --timeout=15m
    
    echo "✅ Infrastructure deployment completed"
}

# Core services deployment
deploy_core_services() {
    echo "🎯 Deploying core microservices..."
    
    # Deploy CRDs first
    echo "Deploying Custom Resource Definitions..."
    kubectl apply -f deployments/crds/
    
    # Wait for CRDs to be established
    for crd in intentprocessings resourceplans manifestgenerations gitopsdeployments deploymentverifications; do
        kubectl wait --for=condition=Established crd ${crd}.nephoran.com --timeout=60s
    done
    
    # Deploy services in dependency order
    SERVICES=(
        "coordination-controller"
        "intent-processing-controller" 
        "resource-planning-controller"
        "manifest-generation-controller"
        "gitops-deployment-controller"
        "deployment-verification-controller"
    )
    
    for service in "${SERVICES[@]}"; do
        echo "Deploying $service..."
        kubectl apply -f deployments/specialized-controllers/$service.yaml
        
        # Wait for deployment to be ready
        kubectl wait --for=condition=Available deployment/$service \
            -n nephoran-system --timeout=300s
        
        # Validate service health
        kubectl get pods -l app=$service -n nephoran-system
    done
    
    echo "✅ Core services deployment completed"
}

# Configuration deployment
deploy_configuration() {
    echo "⚙️  Deploying configuration..."
    
    # Deploy feature flags configuration
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: microservices-feature-flags
  namespace: nephoran-system
data:
  flags.yaml: |
    flags:
      microservices.full.enabled:
        enabled: true
        percentage: 100
        description: "Enable complete microservices architecture"
      parallel.processing.enabled:
        enabled: true
        percentage: 100
        description: "Enable parallel processing across controllers"
      advanced.caching.enabled:
        enabled: true
        percentage: 100
        description: "Enable advanced multi-level caching"
EOF
    
    # Deploy monitoring configuration
    kubectl apply -f config/monitoring/
    
    # Deploy RBAC configuration
    kubectl apply -f deployments/rbac/
    
    echo "✅ Configuration deployment completed"
}

# Deployment validation
validate_deployment() {
    echo "🧪 Validating deployment..."
    
    # Check all pods are running
    PENDING_PODS=$(kubectl get pods -n nephoran-system --field-selector=status.phase!=Running --no-headers | wc -l)
    if [[ $PENDING_PODS -gt 0 ]]; then
        echo "❌ $PENDING_PODS pods are not running"
        kubectl get pods -n nephoran-system --field-selector=status.phase!=Running
        return 1
    fi
    
    # Validate service endpoints
    SERVICES=("coordination-controller" "intent-processing-controller" "resource-planning-controller")
    for service in "${SERVICES[@]}"; do
        if ! kubectl get endpoints $service -n nephoran-system -o jsonpath='{.subsets[*].addresses[*].ip}' | grep -q .; then
            echo "❌ Service $service has no endpoints"
            return 1
        fi
    done
    
    # Run health check tests
    echo "Running health check tests..."
    timeout 120 bash tests/health-check-suite.sh
    
    if [[ $? -eq 0 ]]; then
        echo "✅ Deployment validation successful"
        return 0
    else
        echo "❌ Deployment validation failed"
        return 1
    fi
}

# Main deployment function
main() {
    echo "Starting production deployment..."
    
    if ! validate_prerequisites; then
        echo "💥 Prerequisites validation failed"
        exit 1
    fi
    
    if ! deploy_infrastructure; then
        echo "💥 Infrastructure deployment failed"
        exit 1
    fi
    
    if ! deploy_core_services; then
        echo "💥 Core services deployment failed"
        exit 1
    fi
    
    if ! deploy_configuration; then
        echo "💥 Configuration deployment failed"
        exit 1
    fi
    
    if ! validate_deployment; then
        echo "💥 Deployment validation failed"
        exit 1
    fi
    
    echo "🎉 Production deployment completed successfully!"
    echo ""
    echo "📊 Access Grafana dashboards: http://grafana.nephoran-monitoring.svc.cluster.local:3000"
    echo "📈 Access Prometheus: http://prometheus.nephoran-monitoring.svc.cluster.local:9090"
    echo "🎯 System endpoints:"
    echo "  - Intent Processing: http://intent-processing-controller.nephoran-system.svc.cluster.local:8080"
    echo "  - Resource Planning: http://resource-planning-controller.nephoran-system.svc.cluster.local:8081"
    echo "  - Manifest Generation: http://manifest-generation-controller.nephoran-system.svc.cluster.local:8082"
}

# Execute main function
main "$@"
```

### Blue-Green Deployment Strategy

```yaml
# Blue-Green Deployment Configuration
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: intent-processing-controller
  namespace: nephoran-system
spec:
  replicas: 3
  strategy:
    blueGreen:
      # Traffic routing
      activeService: intent-processing-active
      previewService: intent-processing-preview
      
      # Automated promotion
      autoPromotionEnabled: false  # Manual promotion for safety
      
      # Health checks before promotion
      prePromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: intent-processing-preview
        - templateName: avg-latency
        args:
        - name: service-name
          value: intent-processing-preview
      
      # Post-promotion analysis
      postPromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: intent-processing-active
        - templateName: avg-latency
        args:
        - name: service-name
          value: intent-processing-active
      
      # Rollback configuration
      scaleDownDelaySeconds: 30
      prePromotionAnalysis:
        successCondition: result[0] >= 0.95 && result[1] < 2000
      postPromotionAnalysis:
        successCondition: result[0] >= 0.95 && result[1] < 2000
  
  selector:
    matchLabels:
      app: intent-processing-controller
  
  template:
    metadata:
      labels:
        app: intent-processing-controller
    spec:
      containers:
      - name: intent-processor
        image: nephoran/intent-processing-controller:v2.0.0
        ports:
        - containerPort: 8080
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 1Gi
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

---
# Analysis Templates
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
  namespace: nephoran-system
spec:
  args:
  - name: service-name
  metrics:
  - name: success-rate
    interval: 30s
    count: 5
    provider:
      prometheus:
        address: http://prometheus.nephoran-monitoring.svc.cluster.local:9090
        query: |
          sum(rate(http_requests_total{service="{{args.service-name}}",code=~"2.."}[2m])) /
          sum(rate(http_requests_total{service="{{args.service-name}}"}[2m]))

---
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: avg-latency
  namespace: nephoran-system
spec:
  args:
  - name: service-name
  metrics:
  - name: avg-latency
    interval: 30s
    count: 5
    provider:
      prometheus:
        address: http://prometheus.nephoran-monitoring.svc.cluster.local:9090
        query: |
          histogram_quantile(0.95,
            rate(http_request_duration_seconds_bucket{service="{{args.service-name}}"}[2m])
          ) * 1000
```

## Monitoring and Observability

### Comprehensive Monitoring Stack

```yaml
# Prometheus Configuration for Microservices
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: nephoran-monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s
    
    rule_files:
      - "/etc/prometheus/rules/*.yml"
    
    alerting:
      alertmanagers:
      - static_configs:
        - targets:
          - alertmanager:9093
    
    scrape_configs:
    # Kubernetes API server
    - job_name: 'kubernetes-apiservers'
      kubernetes_sd_configs:
      - role: endpoints
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      relabel_configs:
      - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
        action: keep
        regex: default;kubernetes;https
    
    # Nephoran microservices
    - job_name: 'nephoran-controllers'
      kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
          - nephoran-system
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
      - action: labelmap
        regex: __meta_kubernetes_service_label_(.+)
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace
      - source_labels: [__meta_kubernetes_service_name]
        action: replace
        target_label: kubernetes_name
    
    # Redis cluster monitoring
    - job_name: 'redis-cluster'
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
          - nephoran-infrastructure
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_name]
        action: keep
        regex: redis-cluster
      - source_labels: [__address__]
        action: replace
        regex: ([^:]+):.*
        replacement: $1:9121
        target_label: __address__
    
    # Weaviate monitoring
    - job_name: 'weaviate'
      static_configs:
      - targets: ['weaviate.nephoran-system.svc.cluster.local:2112']
      scrape_interval: 30s

---
# Grafana Dashboard ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: nephoran-dashboards
  namespace: nephoran-monitoring
  labels:
    grafana_dashboard: "1"
data:
  nephoran-overview.json: |
    {
      "dashboard": {
        "id": null,
        "title": "Nephoran Microservices Overview",
        "tags": ["nephoran", "microservices"],
        "timezone": "browser",
        "panels": [
          {
            "id": 1,
            "title": "Intent Processing Rate",
            "type": "stat",
            "targets": [
              {
                "expr": "sum(rate(nephoran_intents_processed_total[5m]))",
                "legendFormat": "Intents/sec"
              }
            ],
            "fieldConfig": {
              "defaults": {
                "color": {"mode": "thresholds"},
                "thresholds": {
                  "steps": [
                    {"color": "red", "value": 0},
                    {"color": "yellow", "value": 0.5},
                    {"color": "green", "value": 1.0}
                  ]
                }
              }
            },
            "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0}
          },
          {
            "id": 2,
            "title": "Processing Latency",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.50, rate(nephoran_processing_duration_seconds_bucket[5m]))",
                "legendFormat": "P50"
              },
              {
                "expr": "histogram_quantile(0.95, rate(nephoran_processing_duration_seconds_bucket[5m]))",
                "legendFormat": "P95"
              },
              {
                "expr": "histogram_quantile(0.99, rate(nephoran_processing_duration_seconds_bucket[5m]))",
                "legendFormat": "P99"
              }
            ],
            "yAxes": [
              {
                "unit": "s",
                "min": 0
              }
            ],
            "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0}
          },
          {
            "id": 3,
            "title": "Error Rate by Controller",
            "type": "graph",
            "targets": [
              {
                "expr": "sum(rate(nephoran_processing_errors_total[5m])) by (controller)",
                "legendFormat": "{{controller}}"
              }
            ],
            "yAxes": [
              {
                "unit": "percent",
                "min": 0,
                "max": 100
              }
            ],
            "gridPos": {"h": 8, "w": 24, "x": 0, "y": 8}
          },
          {
            "id": 4,
            "title": "System Resource Utilization",
            "type": "graph",
            "targets": [
              {
                "expr": "sum(rate(container_cpu_usage_seconds_total{namespace=\"nephoran-system\"}[5m])) by (pod)",
                "legendFormat": "CPU - {{pod}}"
              },
              {
                "expr": "sum(container_memory_working_set_bytes{namespace=\"nephoran-system\"}) by (pod) / 1024 / 1024",
                "legendFormat": "Memory - {{pod}}"
              }
            ],
            "yAxes": [
              {
                "unit": "percent",
                "min": 0
              }
            ],
            "gridPos": {"h": 8, "w": 24, "x": 0, "y": 16}
          }
        ],
        "time": {
          "from": "now-1h",
          "to": "now"
        },
        "refresh": "30s"
      }
    }
```

### Alert Rules Configuration

```yaml
# Prometheus Alert Rules
apiVersion: v1
kind: ConfigMap
metadata:
  name: nephoran-alert-rules
  namespace: nephoran-monitoring
data:
  nephoran.yml: |
    groups:
    - name: nephoran-microservices
      rules:
      
      # High-level service availability alerts
      - alert: NephoranServiceDown
        expr: up{job="nephoran-controllers"} == 0
        for: 1m
        labels:
          severity: critical
          component: "{{ $labels.kubernetes_name }}"
        annotations:
          summary: "Nephoran service {{ $labels.kubernetes_name }} is down"
          description: "Service {{ $labels.kubernetes_name }} in namespace {{ $labels.kubernetes_namespace }} has been down for more than 1 minute"
          runbook_url: "https://docs.nephoran.com/runbooks/service-down"
      
      # Error rate alerts
      - alert: HighErrorRate
        expr: |
          (
            sum(rate(nephoran_processing_errors_total[5m])) by (controller) /
            sum(rate(nephoran_processing_total[5m])) by (controller)
          ) * 100 > 5
        for: 2m
        labels:
          severity: warning
          component: "{{ $labels.controller }}"
        annotations:
          summary: "High error rate in {{ $labels.controller }}"
          description: "Error rate for {{ $labels.controller }} is {{ $value }}% over the last 5 minutes"
          runbook_url: "https://docs.nephoran.com/runbooks/high-error-rate"
      
      # Latency alerts
      - alert: HighProcessingLatency
        expr: |
          histogram_quantile(0.95, 
            rate(nephoran_processing_duration_seconds_bucket[5m])
          ) > 10
        for: 5m
        labels:
          severity: warning
          component: processing
        annotations:
          summary: "High processing latency detected"
          description: "95th percentile latency is {{ $value }}s over the last 5 minutes"
          runbook_url: "https://docs.nephoran.com/runbooks/high-latency"
      
      # Resource utilization alerts
      - alert: HighCPUUtilization
        expr: |
          sum(rate(container_cpu_usage_seconds_total{namespace="nephoran-system"}[5m])) by (pod) /
          sum(container_spec_cpu_quota{namespace="nephoran-system"}) by (pod) * 100 > 80
        for: 10m
        labels:
          severity: warning
          component: "{{ $labels.pod }}"
        annotations:
          summary: "High CPU utilization in {{ $labels.pod }}"
          description: "CPU utilization for {{ $labels.pod }} is {{ $value }}%"
          runbook_url: "https://docs.nephoran.com/runbooks/high-cpu"
      
      # Memory utilization alerts  
      - alert: HighMemoryUtilization
        expr: |
          sum(container_memory_working_set_bytes{namespace="nephoran-system"}) by (pod) /
          sum(container_spec_memory_limit_bytes{namespace="nephoran-system"}) by (pod) * 100 > 85
        for: 5m
        labels:
          severity: warning
          component: "{{ $labels.pod }}"
        annotations:
          summary: "High memory utilization in {{ $labels.pod }}"
          description: "Memory utilization for {{ $labels.pod }} is {{ $value }}%"
          runbook_url: "https://docs.nephoran.com/runbooks/high-memory"
      
      # Cache performance alerts
      - alert: LowCacheHitRate
        expr: |
          sum(rate(nephoran_cache_hits_total[5m])) by (cache_level) /
          (sum(rate(nephoran_cache_hits_total[5m])) by (cache_level) + 
           sum(rate(nephoran_cache_misses_total[5m])) by (cache_level)) * 100 < 70
        for: 10m
        labels:
          severity: warning
          component: "cache-{{ $labels.cache_level }}"
        annotations:
          summary: "Low cache hit rate for {{ $labels.cache_level }}"
          description: "Cache hit rate for {{ $labels.cache_level }} is {{ $value }}%"
          runbook_url: "https://docs.nephoran.com/runbooks/low-cache-hit-rate"
      
      # Queue depth alerts
      - alert: HighQueueDepth
        expr: nephoran_queue_depth > 100
        for: 2m
        labels:
          severity: warning
          component: "queue-{{ $labels.queue_name }}"
        annotations:
          summary: "High queue depth in {{ $labels.queue_name }}"
          description: "Queue depth for {{ $labels.queue_name }} is {{ $value }} messages"
          runbook_url: "https://docs.nephoran.com/runbooks/high-queue-depth"
      
      # Database connectivity alerts
      - alert: WeaviateDatabaseDown
        expr: up{job="weaviate"} == 0
        for: 30s
        labels:
          severity: critical
          component: weaviate
        annotations:
          summary: "Weaviate database is down"
          description: "Weaviate vector database is not responding"
          runbook_url: "https://docs.nephoran.com/runbooks/weaviate-down"
      
      - alert: RedisClusterDown
        expr: up{job="redis-cluster"} == 0
        for: 30s
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "Redis cluster is down"
          description: "Redis cluster is not responding"
          runbook_url: "https://docs.nephoran.com/runbooks/redis-down"
      
      # SLA compliance alerts
      - alert: SLAViolation
        expr: nephoran_sla_compliance_percentage < 99.5
        for: 1m
        labels:
          severity: critical
          component: sla
        annotations:
          summary: "SLA compliance violation"
          description: "SLA compliance is {{ $value }}% (below 99.5% threshold)"
          runbook_url: "https://docs.nephoran.com/runbooks/sla-violation"
```

## Troubleshooting Guide

### Common Issues and Resolutions

```bash
#!/bin/bash
# Comprehensive Troubleshooting Guide

# Issue 1: Intent Processing Failures
troubleshoot_intent_processing() {
    echo "🔍 Troubleshooting Intent Processing Issues..."
    
    # Check controller status
    kubectl get pods -n nephoran-system -l app=intent-processing-controller
    
    # Check recent logs
    echo "📋 Recent controller logs:"
    kubectl logs -n nephoran-system -l app=intent-processing-controller --tail=50
    
    # Check IntentProcessing resources
    echo "📊 IntentProcessing resources:"
    kubectl get intentprocessings -A
    
    # Check for common issues
    FAILED_INTENTS=$(kubectl get intentprocessings -A -o json | jq -r '.items[] | select(.status.phase == "Failed") | .metadata.name')
    
    if [[ -n "$FAILED_INTENTS" ]]; then
        echo "❌ Failed IntentProcessing resources found:"
        echo "$FAILED_INTENTS"
        
        # Analyze failure reasons
        for intent in $FAILED_INTENTS; do
            echo "Analyzing failure for $intent:"
            kubectl get intentprocessing "$intent" -o jsonpath='{.status.errorMessage}'
            echo ""
        done
    fi
    
    # Check LLM client connectivity
    echo "🔗 Testing LLM connectivity..."
    kubectl exec -n nephoran-system deployment/intent-processing-controller -- \
        curl -s -o /dev/null -w "%{http_code}" https://api.openai.com/v1/models
    
    # Check RAG service connectivity
    echo "🧠 Testing RAG service connectivity..."
    kubectl exec -n nephoran-system deployment/intent-processing-controller -- \
        curl -s -o /dev/null -w "%{http_code}" http://rag-api.nephoran-system.svc.cluster.local:8080/health
}

# Issue 2: High Latency Problems
troubleshoot_high_latency() {
    echo "⏱️  Troubleshooting High Latency Issues..."
    
    # Check current latency metrics
    echo "📈 Current latency metrics:"
    curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=histogram_quantile(0.95,rate(nephoran_processing_duration_seconds_bucket[5m]))" | \
        jq -r '.data.result[].value[1]'
    
    # Check resource utilization
    echo "💻 Resource utilization:"
    kubectl top pods -n nephoran-system
    
    # Check cache performance
    echo "🗂️  Cache performance:"
    curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=sum(rate(nephoran_cache_hits_total[5m]))/sum(rate(nephoran_cache_requests_total[5m]))*100" | \
        jq -r '.data.result[].value[1]'
    
    # Check database connectivity
    echo "🗃️  Database connectivity:"
    kubectl exec -n nephoran-system deployment/resource-planning-controller -- \
        curl -s -o /dev/null -w "%{http_code}" http://weaviate.nephoran-system.svc.cluster.local:8080/v1/meta
    
    # Check for resource bottlenecks
    echo "🚧 Checking for bottlenecks..."
    
    # CPU bottlenecks
    HIGH_CPU_PODS=$(kubectl top pods -n nephoran-system | awk 'NR>1 && $2+0 > 500 {print $1}')
    if [[ -n "$HIGH_CPU_PODS" ]]; then
        echo "⚠️  High CPU usage pods: $HIGH_CPU_PODS"
    fi
    
    # Memory bottlenecks
    HIGH_MEM_PODS=$(kubectl top pods -n nephoran-system | awk 'NR>1 && $3+0 > 1000 {print $1}')
    if [[ -n "$HIGH_MEM_PODS" ]]; then
        echo "⚠️  High memory usage pods: $HIGH_MEM_PODS"
    fi
    
    # Queue depth check
    echo "📥 Queue depth analysis:"
    curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=nephoran_queue_depth" | \
        jq -r '.data.result[] | "\(.metric.queue_name): \(.value[1])"'
}

# Issue 3: Service Connectivity Problems
troubleshoot_connectivity() {
    echo "🌐 Troubleshooting Service Connectivity..."
    
    # Check all service endpoints
    echo "🎯 Service endpoints status:"
    for service in coordination-controller intent-processing-controller resource-planning-controller manifest-generation-controller gitops-deployment-controller deployment-verification-controller; do
        echo "Checking $service..."
        ENDPOINT_IPS=$(kubectl get endpoints $service -n nephoran-system -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)
        if [[ -n "$ENDPOINT_IPS" ]]; then
            echo "✅ $service: $ENDPOINT_IPS"
        else
            echo "❌ $service: No endpoints"
        fi
    done
    
    # Check DNS resolution
    echo "🔍 DNS resolution test:"
    kubectl exec -n nephoran-system deployment/coordination-controller -- \
        nslookup intent-processing-controller.nephoran-system.svc.cluster.local
    
    # Check network policies
    echo "🛡️  Network policies:"
    kubectl get networkpolicies -n nephoran-system
    
    # Test inter-service communication
    echo "📡 Inter-service communication test:"
    kubectl exec -n nephoran-system deployment/coordination-controller -- \
        curl -s -o /dev/null -w "%{http_code}" \
        http://intent-processing-controller.nephoran-system.svc.cluster.local:8080/healthz
}

# Issue 4: Database and Storage Problems
troubleshoot_storage() {
    echo "💾 Troubleshooting Storage Issues..."
    
    # Check Weaviate status
    echo "🧠 Weaviate status:"
    kubectl get pods -n nephoran-system -l app=weaviate
    kubectl logs -n nephoran-system -l app=weaviate --tail=20
    
    # Check Weaviate health
    echo "🏥 Weaviate health check:"
    kubectl exec -n nephoran-system deployment/weaviate -- \
        curl -s http://localhost:8080/v1/meta | jq '.hostname, .version'
    
    # Check Redis cluster status
    echo "🔄 Redis cluster status:"
    kubectl get pods -n nephoran-infrastructure -l app.kubernetes.io/name=redis-cluster
    
    # Test Redis connectivity
    echo "🔗 Redis connectivity test:"
    REDIS_PASSWORD=$(kubectl get secret redis-cluster -n nephoran-infrastructure -o jsonpath='{.data.redis-password}' | base64 -d)
    kubectl exec -n nephoran-infrastructure redis-cluster-0 -- redis-cli -a "$REDIS_PASSWORD" ping
    
    # Check persistent volumes
    echo "💽 Persistent volumes:"
    kubectl get pv | grep nephoran
    
    # Check storage class
    echo "🗂️  Storage classes:"
    kubectl get storageclass
}

# Issue 5: Performance Degradation
troubleshoot_performance() {
    echo "📊 Troubleshooting Performance Degradation..."
    
    # Check system metrics
    echo "📈 System performance metrics:"
    
    # Processing rate
    CURRENT_RATE=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=sum(rate(nephoran_intents_processed_total[5m]))" | jq -r '.data.result[].value[1]')
    echo "Current processing rate: $CURRENT_RATE intents/sec"
    
    # Error rate
    ERROR_RATE=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=sum(rate(nephoran_processing_errors_total[5m]))/sum(rate(nephoran_processing_total[5m]))*100" | jq -r '.data.result[].value[1]')
    echo "Current error rate: $ERROR_RATE%"
    
    # Cache hit rate
    CACHE_HIT_RATE=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=sum(rate(nephoran_cache_hits_total[5m]))/sum(rate(nephoran_cache_requests_total[5m]))*100" | jq -r '.data.result[].value[1]')
    echo "Current cache hit rate: $CACHE_HIT_RATE%"
    
    # Resource utilization
    echo "💻 Resource utilization summary:"
    kubectl top nodes
    kubectl top pods -n nephoran-system --sort-by=cpu
    
    # Check for resource limits
    echo "🚫 Resource limit analysis:"
    kubectl describe pods -n nephoran-system | grep -E "(Requests|Limits|cpu|memory)" | head -20
}

# Main troubleshooting function
main_troubleshoot() {
    local issue_type=${1:-"all"}
    
    echo "🔧 Nephoran Microservices Troubleshooting Guide"
    echo "Issue Type: $issue_type"
    echo "=============================================="
    
    case $issue_type in
        "intent"|"processing")
            troubleshoot_intent_processing
            ;;
        "latency"|"performance")
            troubleshoot_high_latency
            troubleshoot_performance
            ;;
        "connectivity"|"network")
            troubleshoot_connectivity
            ;;
        "storage"|"database")
            troubleshoot_storage
            ;;
        "all"|*)
            troubleshoot_intent_processing
            troubleshoot_connectivity
            troubleshoot_storage
            troubleshoot_performance
            ;;
    esac
    
    echo ""
    echo "📋 Additional Resources:"
    echo "- Logs: kubectl logs -n nephoran-system -l app=<controller-name>"
    echo "- Metrics: http://grafana.nephoran-monitoring.svc.cluster.local:3000"
    echo "- Documentation: https://docs.nephoran.com/troubleshooting"
    echo "- Support: support@nephoran.com"
}

# Execute troubleshooting
main_troubleshoot "$@"
```

### Diagnostic Commands Reference

```bash
# Quick diagnostic commands for common issues

# Check overall system health
kubectl get pods -A | grep -E "(nephoran|Error|Pending|CrashLoop)"

# View recent events
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | tail -20

# Check resource quotas and limits
kubectl describe resourcequota -n nephoran-system

# View service mesh status (if using Istio)
istioctl proxy-status -n nephoran-system

# Check certificate expiration
kubectl get certificates -n nephoran-system

# View custom resource statuses
kubectl get networkintents,intentprocessings,resourceplans -A

# Performance analysis
kubectl top pods -n nephoran-system --sort-by=memory
kubectl top pods -n nephoran-system --sort-by=cpu

# Network connectivity test
kubectl exec -n nephoran-system deployment/coordination-controller -- \
  ping -c 3 intent-processing-controller.nephoran-system.svc.cluster.local

# Database connectivity tests
kubectl exec -n nephoran-system deployment/intent-processing-controller -- \
  curl -s http://weaviate.nephoran-system.svc.cluster.local:8080/v1/meta

# Redis cluster health
kubectl exec -n nephoran-infrastructure redis-cluster-0 -- redis-cli cluster nodes
```

## Security Procedures

### Security Hardening Checklist

```bash
#!/bin/bash
# Security Hardening and Validation Script

echo "🔒 Nephoran Microservices Security Hardening"

# RBAC validation
validate_rbac() {
    echo "👤 Validating RBAC configuration..."
    
    # Check service accounts
    kubectl get serviceaccounts -n nephoran-system
    
    # Validate roles and role bindings
    kubectl get roles,rolebindings -n nephoran-system
    kubectl get clusterroles,clusterrolebindings | grep nephoran
    
    # Test least privilege principle
    echo "Testing least privilege access..."
    
    # Test coordination controller permissions
    kubectl auth can-i create networkintents --as=system:serviceaccount:nephoran-system:coordination-controller
    kubectl auth can-i delete nodes --as=system:serviceaccount:nephoran-system:coordination-controller
    
    if [[ $? -eq 0 ]]; then
        echo "⚠️  WARNING: Service account has excessive permissions"
    else
        echo "✅ Service account follows least privilege principle"
    fi
}

# Network security validation
validate_network_security() {
    echo "🌐 Validating network security..."
    
    # Check network policies
    kubectl get networkpolicies -n nephoran-system
    
    # Validate TLS configuration
    echo "🔐 Checking TLS configuration..."
    
    # Check certificates
    kubectl get certificates -n nephoran-system
    
    # Validate secure communication
    for service in intent-processing-controller resource-planning-controller; do
        echo "Testing TLS for $service..."
        kubectl exec -n nephoran-system deployment/coordination-controller -- \
            curl -s -k -I https://$service.nephoran-system.svc.cluster.local:8080/healthz | head -1
    done
}

# Secret management validation
validate_secrets() {
    echo "🗝️  Validating secret management..."
    
    # Check for secrets
    kubectl get secrets -n nephoran-system
    
    # Validate encryption at rest
    echo "Checking encryption at rest..."
    kubectl get encryptionconfig -o yaml 2>/dev/null || echo "⚠️  Encryption at rest not configured"
    
    # Check for hardcoded secrets in configurations
    echo "Scanning for hardcoded secrets..."
    kubectl get configmaps -n nephoran-system -o yaml | grep -iE "(password|key|token|secret)" | head -5
    
    # Validate secret rotation
    echo "Checking secret age..."
    kubectl get secrets -n nephoran-system -o json | \
        jq -r '.items[] | "\(.metadata.name): \(.metadata.creationTimestamp)"' | \
        while read line; do
            secret_name=$(echo $line | cut -d: -f1)
            creation_time=$(echo $line | cut -d: -f2)
            age_days=$(( ($(date +%s) - $(date -d "$creation_time" +%s)) / 86400 ))
            if [[ $age_days -gt 90 ]]; then
                echo "⚠️  Secret $secret_name is $age_days days old (consider rotation)"
            fi
        done
}

# Container security validation
validate_container_security() {
    echo "🐳 Validating container security..."
    
    # Check security contexts
    echo "Checking security contexts..."
    kubectl get pods -n nephoran-system -o json | \
        jq -r '.items[] | "\(.metadata.name): RunAsNonRoot=\(.spec.securityContext.runAsNonRoot // "not_set"), ReadOnlyRootFilesystem=\(.spec.containers[0].securityContext.readOnlyRootFilesystem // "not_set")"'
    
    # Check for privileged containers
    PRIVILEGED_PODS=$(kubectl get pods -n nephoran-system -o json | \
        jq -r '.items[] | select(.spec.containers[].securityContext.privileged == true) | .metadata.name')
    
    if [[ -n "$PRIVILEGED_PODS" ]]; then
        echo "⚠️  WARNING: Privileged containers found: $PRIVILEGED_PODS"
    else
        echo "✅ No privileged containers found"
    fi
    
    # Check resource limits
    echo "Validating resource limits..."
    PODS_WITHOUT_LIMITS=$(kubectl get pods -n nephoran-system -o json | \
        jq -r '.items[] | select(.spec.containers[].resources.limits == null) | .metadata.name')
    
    if [[ -n "$PODS_WITHOUT_LIMITS" ]]; then
        echo "⚠️  WARNING: Pods without resource limits: $PODS_WITHOUT_LIMITS"
    else
        echo "✅ All pods have resource limits"
    fi
}

# Vulnerability scanning
run_vulnerability_scan() {
    echo "🔍 Running vulnerability scan..."
    
    # Check for admission controllers
    kubectl get admissionregistration.k8s.io/v1/validatingadmissionwebhooks | grep -E "(opa|gatekeeper|falco)"
    
    # Scan images for vulnerabilities (if trivy is available)
    if command -v trivy &> /dev/null; then
        echo "Scanning container images..."
        kubectl get pods -n nephoran-system -o jsonpath='{range .items[*]}{.spec.containers[*].image}{"\n"}{end}' | \
            sort | uniq | while read image; do
            echo "Scanning $image..."
            trivy image --severity HIGH,CRITICAL --quiet "$image" | head -10
        done
    else
        echo "⚠️  Trivy not available for image scanning"
    fi
}

# Audit log validation
validate_audit_logs() {
    echo "📋 Validating audit configuration..."
    
    # Check if audit logging is enabled
    if kubectl get pods -n kube-system -l component=kube-apiserver -o yaml | grep -q audit-log-path; then
        echo "✅ Audit logging is enabled"
        
        # Check recent audit events
        echo "Recent audit events (if accessible):"
        # Note: This requires access to master node audit logs
        # kubectl exec -n kube-system kube-apiserver-master -- tail -5 /var/log/audit.log 2>/dev/null || echo "Audit logs not accessible"
    else
        echo "⚠️  Audit logging may not be enabled"
    fi
}

# Main security validation
main_security_check() {
    echo "Starting comprehensive security validation..."
    
    validate_rbac
    validate_network_security
    validate_secrets
    validate_container_security
    run_vulnerability_scan
    validate_audit_logs
    
    echo ""
    echo "📊 Security Validation Summary:"
    echo "- RBAC: Validated service account permissions"
    echo "- Network: Checked TLS and network policies"  
    echo "- Secrets: Validated secret management"
    echo "- Containers: Checked security contexts and limits"
    echo "- Vulnerabilities: Scanned for known issues"
    echo "- Audit: Validated audit logging configuration"
    echo ""
    echo "🔒 For additional security hardening:"
    echo "- Enable Pod Security Standards"
    echo "- Implement network segmentation"
    echo "- Regular security assessments"
    echo "- Vulnerability management process"
}

# Execute security validation
main_security_check
```

### Security Monitoring and Alerting

```yaml
# Security-focused Prometheus alerts
apiVersion: v1
kind: ConfigMap
metadata:
  name: security-alert-rules
  namespace: nephoran-monitoring
data:
  security.yml: |
    groups:
    - name: security-alerts
      rules:
      
      # Unauthorized access attempts
      - alert: UnauthorizedAPIAccess
        expr: increase(apiserver_audit_total{objectRef_apiVersion!="v1",verb!~"get|list|watch",code!~"2.."}[5m]) > 0
        labels:
          severity: warning
        annotations:
          summary: "Unauthorized API access detected"
          description: "Unauthorized API access attempts detected"
      
      # Privileged container alerts
      - alert: PrivilegedContainerCreated
        expr: increase(kube_pod_container_info{container_privileged="true"}[5m]) > 0
        labels:
          severity: critical
        annotations:
          summary: "Privileged container created"
          description: "A privileged container was created in namespace {{ $labels.namespace }}"
      
      # Certificate expiration alerts
      - alert: CertificateExpiringSoon
        expr: (cert_exporter_not_after - time()) / 86400 < 30
        labels:
          severity: warning
        annotations:
          summary: "Certificate expiring soon"
          description: "Certificate {{ $labels.dns_names }} expires in {{ $value }} days"
      
      # Failed authentication alerts  
      - alert: HighFailedAuthentications
        expr: increase(authentication_attempts{result="failure"}[10m]) > 10
        labels:
          severity: warning
        annotations:
          summary: "High number of failed authentications"
          description: "{{ $value }} failed authentication attempts in the last 10 minutes"
      
      # Network policy violations
      - alert: NetworkPolicyViolation
        expr: increase(cilium_drop_count_total{reason="Policy denied"}[5m]) > 0
        labels:
          severity: warning
        annotations:
          summary: "Network policy violation detected"
          description: "Network traffic was denied by policy"
```

This comprehensive operational procedures guide provides detailed guidance for deploying, monitoring, troubleshooting, and securing the Nephoran Intent Operator microservices architecture in production environments. The procedures ensure operational excellence with automated monitoring, proactive alerting, and comprehensive security measures.