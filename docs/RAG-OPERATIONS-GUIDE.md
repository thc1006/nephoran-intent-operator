# Nephoran RAG Pipeline - Operations Guide

## Overview

This operations guide provides comprehensive instructions for deploying, monitoring, and maintaining the Nephoran Intent Operator's RAG (Retrieval-Augmented Generation) pipeline in production environments. The guide covers deployment strategies, monitoring setup, performance optimization, troubleshooting procedures, and maintenance best practices.

## Table of Contents

1. [Production Deployment](#production-deployment)
2. [Monitoring and Observability](#monitoring-and-observability)
3. [Performance Optimization](#performance-optimization)
4. [Security Operations](#security-operations)
5. [Backup and Recovery](#backup-and-recovery)
6. [Troubleshooting Guide](#troubleshooting-guide)
7. [Maintenance Procedures](#maintenance-procedures)
8. [Disaster Recovery](#disaster-recovery)
9. [Scaling Operations](#scaling-operations)
10. [Migration Procedures](#migration-procedures)

## Production Deployment

### Prerequisites

#### System Requirements

**Minimum Requirements (Development/Testing):**
- **Kubernetes Cluster**: v1.25+ with 3 worker nodes
- **CPU**: 8 cores total (2 cores per node minimum)
- **Memory**: 24GB total (8GB per node minimum)
- **Storage**: 200GB SSD storage for Weaviate
- **Network**: 1Gbps internal networking

**Recommended Production:**
- **Kubernetes Cluster**: v1.27+ with 6+ worker nodes
- **CPU**: 24+ cores total (4+ cores per node)
- **Memory**: 96GB+ total (16GB+ per node)
- **Storage**: 1TB+ NVMe SSD with backup storage
- **Network**: 10Gbps internal networking with redundancy

**High Availability Setup:**
- **Kubernetes Cluster**: 3+ master nodes, 9+ worker nodes
- **CPU**: 48+ cores total
- **Memory**: 192GB+ total
- **Storage**: 2TB+ primary + 1TB+ backup per AZ
- **Network**: Multi-AZ with load balancing

#### Required Tools and Services

```bash
# Core tools
kubectl --version    # >= 1.25
helm --version      # >= 3.10
docker --version    # >= 20.10

# Monitoring stack
prometheus --version # >= 2.40
grafana --version   # >= 9.0

# External services
# - OpenAI API access with sufficient quotas
# - Redis cluster (can be deployed in-cluster)
# - Container registry (Docker Hub, ECR, GCR, etc.)
```

### Deployment Strategies

#### 1. Quick Start Deployment

For rapid deployment in staging or development environments:

```bash
#!/bin/bash
# quick-deploy-rag.sh

set -e

# Set environment variables
export NAMESPACE="nephoran-system"
export OPENAI_API_KEY="your-openai-api-key"
export ENVIRONMENT="staging"

# Create namespace
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Deploy RAG system components
echo "Deploying RAG system components..."

# 1. Deploy secrets
kubectl create secret generic openai-api-key \
  --from-literal=api-key="${OPENAI_API_KEY}" \
  --namespace=${NAMESPACE} \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Deploy Weaviate
kubectl apply -f deployments/weaviate/weaviate-deployment.yaml -n ${NAMESPACE}

# 3. Deploy Redis
kubectl apply -f deployments/redis/redis-deployment.yaml -n ${NAMESPACE}

# 4. Deploy RAG API
kubectl apply -f deployments/kustomize/base/rag-api/ -n ${NAMESPACE}

# 5. Deploy LLM Processor
kubectl apply -f deployments/kustomize/base/llm-processor/ -n ${NAMESPACE}

# Wait for deployments
echo "Waiting for deployments to be ready..."
kubectl wait --for=condition=available deployment/weaviate -n ${NAMESPACE} --timeout=600s
kubectl wait --for=condition=available deployment/redis -n ${NAMESPACE} --timeout=300s
kubectl wait --for=condition=available deployment/rag-api -n ${NAMESPACE} --timeout=600s
kubectl wait --for=condition=available deployment/llm-processor -n ${NAMESPACE} --timeout=300s

# Initialize Weaviate schema
echo "Initializing Weaviate schema..."
kubectl run weaviate-init --image=python:3.11-slim --rm -i --tty \
  --env="WEAVIATE_URL=http://weaviate.${NAMESPACE}.svc.cluster.local:8080" \
  --namespace=${NAMESPACE} \
  -- python3 -c "
import requests
import json

# Telecom knowledge schema
schema = {
    'class': 'TelecomDocument',
    'description': 'Telecommunications knowledge documents',
    'vectorizer': 'none',
    'properties': [
        {'name': 'content', 'dataType': ['text']},
        {'name': 'title', 'dataType': ['string']},
        {'name': 'source', 'dataType': ['string']},
        {'name': 'category', 'dataType': ['string']},
        {'name': 'version', 'dataType': ['string']},
        {'name': 'keywords', 'dataType': ['string[]']},
        {'name': 'networkFunction', 'dataType': ['string[]']},
        {'name': 'technology', 'dataType': ['string[]']},
        {'name': 'useCase', 'dataType': ['string[]']},
        {'name': 'confidence', 'dataType': ['number']},
    ]
}

response = requests.post('${WEAVIATE_URL}/v1/schema', json=schema)
print(f'Schema creation: {response.status_code}')
"

echo "RAG system deployment completed!"
echo "Access the RAG API at: kubectl port-forward svc/rag-api 5001:5001 -n ${NAMESPACE}"
echo "Access Weaviate at: kubectl port-forward svc/weaviate 8080:8080 -n ${NAMESPACE}"
```

#### 2. Production Deployment with Helm

Create a comprehensive Helm chart deployment:

```yaml
# values-production.yaml
global:
  namespace: nephoran-system
  environment: production
  imageRegistry: your-registry.com
  imageTag: v1.0.0

weaviate:
  enabled: true
  replicaCount: 3
  resources:
    requests:
      cpu: 2000m
      memory: 8Gi
    limits:
      cpu: 4000m
      memory: 16Gi
  persistence:
    enabled: true
    size: 500Gi
    storageClass: fast-ssd
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80
  monitoring:
    enabled: true
    serviceMonitor: true
  backup:
    enabled: true
    schedule: "0 2 * * *"
    retention: "30d"

redis:
  enabled: true
  architecture: cluster
  cluster:
    nodes: 6
    replicas: 1
  master:
    resources:
      requests:
        cpu: 1000m
        memory: 2Gi
      limits:
        cpu: 2000m
        memory: 4Gi
  replica:
    resources:
      requests:
        cpu: 500m
        memory: 1Gi
      limits:
        cpu: 1000m
        memory: 2Gi
  auth:
    enabled: true
    password: "secure-redis-password"
  persistence:
    enabled: true
    size: 100Gi

ragApi:
  enabled: true
  replicaCount: 2
  image:
    repository: nephoran/rag-api
    tag: v1.0.0
  resources:
    requests:
      cpu: 1000m
      memory: 2Gi
    limits:
      cpu: 2000m
      memory: 4Gi
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 8
    targetCPUUtilizationPercentage: 70
  config:
    openai:
      model: gpt-4o-mini
      embeddingModel: text-embedding-3-large
      maxTokens: 4096
      temperature: 0.0
    weaviate:
      batchSize: 100
      timeout: 30s
    cache:
      ttl: 3600
      maxSize: 10000

llmProcessor:
  enabled: true
  replicaCount: 2
  image:
    repository: nephoran/llm-processor
    tag: v1.0.0
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 1000m
      memory: 2Gi
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 6

monitoring:
  enabled: true
  prometheus:
    enabled: true
    retention: 30d
    storage: 200Gi
  grafana:
    enabled: true
    persistence:
      enabled: true
      size: 10Gi
  alertmanager:
    enabled: true
    config:
      slack:
        webhookUrl: "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
      email:
        smtpServer: "smtp.company.com:587"

networkPolicies:
  enabled: true
  ingress:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
    hosts:
      - host: rag-api.company.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: rag-api-tls
        hosts:
          - rag-api.company.com

securityContext:
  runAsNonRoot: true
  runAsUser: 1001
  fsGroup: 1001
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true

podDisruptionBudget:
  enabled: true
  minAvailable: 1

podSecurityPolicy:
  enabled: true
```

Deploy with Helm:

```bash
# Add Helm repositories
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install production deployment
helm install nephoran-rag ./chart/nephoran-rag \
  --namespace nephoran-system \
  --create-namespace \
  --values values-production.yaml \
  --wait \
  --timeout 20m

# Verify deployment
kubectl get all -n nephoran-system
kubectl get pvc -n nephoran-system
kubectl get ingress -n nephoran-system
```

#### 3. Multi-Environment Deployment

Set up multiple environments with GitOps:

```yaml
# environments/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: nephoran-production

resources:
  - ../../base

patchesStrategicMerge:
  - replica-patches.yaml
  - resource-patches.yaml
  - config-patches.yaml

configMapGenerator:
  - name: rag-config
    files:
      - config/production.yaml
    options:
      disableNameSuffixHash: true

secretGenerator:
  - name: api-keys
    envs:
      - secrets/production.env
    options:
      disableNameSuffixHash: true

images:
  - name: nephoran/rag-api
    newTag: v1.0.0
  - name: nephoran/llm-processor
    newTag: v1.0.0

replicas:
  - name: rag-api
    count: 4
  - name: llm-processor
    count: 3
  - name: weaviate
    count: 5
```

```yaml
# environments/production/replica-patches.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rag-api
spec:
  replicas: 4
  template:
    spec:
      containers:
      - name: rag-api
        resources:
          requests:
            cpu: 1000m
            memory: 2Gi
          limits:
            cpu: 2000m
            memory: 4Gi

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: weaviate
spec:
  replicas: 5
  template:
    spec:
      containers:
      - name: weaviate
        resources:
          requests:
            cpu: 2000m
            memory: 8Gi
          limits:
            cpu: 4000m
            memory: 16Gi
```

### Environment-Specific Configurations

#### Development Environment

```yaml
# config/development.yaml
rag:
  documentLoader:
    batchSize: 5
    maxConcurrency: 2
    processingTimeout: 30s
  chunking:
    chunkSize: 500
    chunkOverlap: 50
  embedding:
    batchSize: 20
    rateLimitRPM: 1000
  weaviate:
    replicas: 1
    resources:
      requests:
        cpu: 500m
        memory: 1Gi
  cache:
    enabled: true
    ttl: 1h
    maxSize: 100
  monitoring:
    enabled: true
    level: debug
```

#### Staging Environment

```yaml
# config/staging.yaml
rag:
  documentLoader:
    batchSize: 10
    maxConcurrency: 5
    processingTimeout: 60s
  chunking:
    chunkSize: 1000
    chunkOverlap: 100
  embedding:
    batchSize: 50
    rateLimitRPM: 2000
  weaviate:
    replicas: 2
    resources:
      requests:
        cpu: 1000m
        memory: 4Gi
  cache:
    enabled: true
    ttl: 2h
    maxSize: 1000
  monitoring:
    enabled: true
    level: info
```

#### Production Environment

```yaml
# config/production.yaml
rag:
  documentLoader:
    batchSize: 20
    maxConcurrency: 10
    processingTimeout: 120s
  chunking:
    chunkSize: 1000
    chunkOverlap: 200
    preserveHierarchy: true
    useSemanticBoundaries: true
  embedding:
    batchSize: 100
    rateLimitRPM: 5000
    enableCaching: true
  weaviate:
    replicas: 5
    resources:
      requests:
        cpu: 2000m
        memory: 8Gi
      limits:
        cpu: 4000m
        memory: 16Gi
    backup:
      enabled: true
      schedule: "0 2 * * *"
      retention: "30d"
  cache:
    enabled: true
    ttl: 4h
    maxSize: 10000
    compression: true
  monitoring:
    enabled: true
    level: info
    alerts: true
```

## Monitoring and Observability

### Prometheus Metrics

The RAG system exposes comprehensive metrics for monitoring:

#### Core System Metrics

```yaml
# ServiceMonitor for RAG API
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: rag-api-metrics
  namespace: nephoran-system
spec:
  selector:
    matchLabels:
      app: rag-api
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
    honorLabels: true
```

**Key Metrics:**

```promql
# Query Processing Metrics
rag_query_total{intent_type="configuration",status="success"} 
rag_query_duration_seconds{intent_type="configuration",quantile="0.95"}
rag_query_errors_total{intent_type="configuration",error_type="retrieval_failed"}

# Document Processing Metrics  
rag_document_processing_total{status="success"}
rag_document_processing_duration_seconds{quantile="0.99"}
rag_chunks_total{document_source="3GPP"}

# Embedding Metrics
rag_embedding_requests_total{model="text-embedding-3-large",status="success"}
rag_embedding_duration_seconds{model="text-embedding-3-large",quantile="0.95"}
rag_embedding_tokens_total{model="text-embedding-3-large"}

# Cache Metrics
rag_cache_hits_total{cache_type="query_result"}
rag_cache_misses_total{cache_type="query_result"}  
rag_cache_hit_rate{cache_type="embedding"}

# Weaviate Metrics
weaviate_queries_total{class="TelecomDocument",status="success"}
weaviate_query_duration_seconds{class="TelecomDocument",quantile="0.95"}
weaviate_objects_total{class="TelecomDocument"}

# Resource Metrics
rag_memory_usage_bytes{component="rag_api"}
rag_cpu_usage_percent{component="weaviate"}
rag_disk_usage_bytes{component="weaviate"}
```

#### Custom Business Metrics

```go
// Business metrics implementation
type BusinessMetrics struct {
    // Intent processing metrics
    IntentsByType     *prometheus.CounterVec
    IntentSuccessRate *prometheus.GaugeVec
    IntentLatency     *prometheus.HistogramVec
    
    // Knowledge base metrics
    DocumentCount     *prometheus.GaugeVec
    KnowledgeQuality  *prometheus.GaugeVec
    SourceDistribution *prometheus.GaugeVec
    
    // User interaction metrics
    QueryComplexity   *prometheus.HistogramVec
    UserSatisfaction  *prometheus.GaugeVec
    ResponseAccuracy  *prometheus.GaugeVec
}

func NewBusinessMetrics() *BusinessMetrics {
    return &BusinessMetrics{
        IntentsByType: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "nephoran_intents_total",
                Help: "Total number of processed intents by type",
            },
            []string{"intent_type", "domain", "complexity"},
        ),
        IntentSuccessRate: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "nephoran_intent_success_rate",
                Help: "Success rate of intent processing",
            },
            []string{"intent_type", "time_window"},
        ),
        KnowledgeQuality: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "nephoran_knowledge_quality_score",
                Help: "Average quality score of knowledge base documents",
            },
            []string{"source", "category"},
        ),
    }
}
```

### Grafana Dashboards

#### 1. RAG System Overview Dashboard

```json
{
  "dashboard": {
    "id": null,
    "title": "Nephoran RAG System Overview",
    "tags": ["nephoran", "rag", "telecom"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Query Processing Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(rag_query_total[5m])",
            "legendFormat": "Queries/sec"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "reqps",
            "thresholds": {
              "steps": [
                {"color": "green", "value": null},
                {"color": "yellow", "value": 10},
                {"color": "red", "value": 50}
              ]
            }
          }
        }
      },
      {
        "id": 2,
        "title": "Average Query Latency",
        "type": "stat",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(rag_query_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "s",
            "thresholds": {
              "steps": [
                {"color": "green", "value": null},
                {"color": "yellow", "value": 2},
                {"color": "red", "value": 5}
              ]
            }
          }
        }
      },
      {
        "id": 3,
        "title": "Query Success Rate",
        "type": "gauge",
        "targets": [
          {
            "expr": "rate(rag_query_total{status=\"success\"}[5m]) / rate(rag_query_total[5m]) * 100",
            "legendFormat": "Success Rate %"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percent",
            "min": 0,
            "max": 100,
            "thresholds": {
              "steps": [
                {"color": "red", "value": null},
                {"color": "yellow", "value": 95},
                {"color": "green", "value": 99}
              ]
            }
          }
        }
      },
      {
        "id": 4,
        "title": "Cache Hit Rate",
        "type": "timeseries",
        "targets": [
          {
            "expr": "rag_cache_hits_total / (rag_cache_hits_total + rag_cache_misses_total) * 100",
            "legendFormat": "{{cache_type}} Cache Hit Rate"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percent",
            "custom": {
              "drawStyle": "line",
              "fillOpacity": 10
            }
          }
        }
      },
      {
        "id": 5,
        "title": "Intent Types Distribution",
        "type": "piechart",
        "targets": [
          {
            "expr": "sum by (intent_type) (rate(rag_query_total[1h]))",
            "legendFormat": "{{intent_type}}"
          }
        ]
      },
      {
        "id": 6,
        "title": "Weaviate Cluster Status",
        "type": "table",
        "targets": [
          {
            "expr": "weaviate_cluster_healthy",
            "legendFormat": "Cluster Health"
          },
          {
            "expr": "weaviate_objects_total",
            "legendFormat": "Total Objects"
          },
          {
            "expr": "weaviate_vector_dimensions",
            "legendFormat": "Vector Dimensions"
          }
        ]
      },
      {
        "id": 7,
        "title": "Resource Usage",
        "type": "timeseries",
        "targets": [
          {
            "expr": "rag_memory_usage_bytes{component=\"rag_api\"} / 1024 / 1024 / 1024",
            "legendFormat": "RAG API Memory (GB)"
          },
          {
            "expr": "rag_memory_usage_bytes{component=\"weaviate\"} / 1024 / 1024 / 1024",
            "legendFormat": "Weaviate Memory (GB)"
          },
          {
            "expr": "rag_cpu_usage_percent{component=\"rag_api\"}",
            "legendFormat": "RAG API CPU %"
          },
          {
            "expr": "rag_cpu_usage_percent{component=\"weaviate\"}",
            "legendFormat": "Weaviate CPU %"
          }
        ]
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

#### 2. Performance Analytics Dashboard

```json
{
  "dashboard": {
    "title": "RAG Performance Analytics",
    "panels": [
      {
        "id": 1,
        "title": "Query Latency Breakdown",
        "type": "timeseries",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(rag_retrieval_duration_seconds_bucket[5m]))",
            "legendFormat": "Retrieval (95th)"
          },
          {
            "expr": "histogram_quantile(0.95, rate(rag_enhancement_duration_seconds_bucket[5m]))",
            "legendFormat": "Enhancement (95th)"
          },
          {
            "expr": "histogram_quantile(0.95, rate(rag_reranking_duration_seconds_bucket[5m]))",
            "legendFormat": "Reranking (95th)"
          },
          {
            "expr": "histogram_quantile(0.95, rate(rag_context_assembly_duration_seconds_bucket[5m]))",
            "legendFormat": "Context Assembly (95th)"
          }
        ]
      },
      {
        "id": 2,
        "title": "Embedding Generation Performance",
        "type": "timeseries",
        "targets": [
          {
            "expr": "rate(rag_embedding_requests_total[5m])",
            "legendFormat": "Requests/sec"
          },
          {
            "expr": "histogram_quantile(0.95, rate(rag_embedding_duration_seconds_bucket[5m]))",
            "legendFormat": "Latency (95th)"
          },
          {
            "expr": "rate(rag_embedding_tokens_total[5m])",
            "legendFormat": "Tokens/sec"
          }
        ]
      },
      {
        "id": 3,
        "title": "Knowledge Base Growth",
        "type": "timeseries",
        "targets": [
          {
            "expr": "weaviate_objects_total{class=\"TelecomDocument\"}",
            "legendFormat": "Total Documents"
          },
          {
            "expr": "increase(rag_document_processing_total{status=\"success\"}[1d])",
            "legendFormat": "Documents Added (Daily)"
          }
        ]
      }
    ]
  }
}
```

### Alerting Rules

```yaml
# alerts/rag-alerts.yaml
groups:
- name: rag-system-alerts
  rules:
  
  # High-level service alerts
  - alert: RAGSystemDown
    expr: up{job="rag-api"} == 0
    for: 1m
    labels:
      severity: critical
      component: rag-system
    annotations:
      summary: "RAG API service is down"
      description: "RAG API has been down for more than 1 minute"
      runbook_url: "https://docs.company.com/runbooks/rag-system-down"

  - alert: WeaviateClusterUnhealthy
    expr: weaviate_cluster_healthy == 0
    for: 2m
    labels:
      severity: critical
      component: weaviate
    annotations:
      summary: "Weaviate cluster is unhealthy"
      description: "Weaviate cluster has been unhealthy for more than 2 minutes"

  # Performance alerts
  - alert: HighQueryLatency
    expr: histogram_quantile(0.95, rate(rag_query_duration_seconds_bucket[5m])) > 5
    for: 3m
    labels:
      severity: warning
      component: rag-api
    annotations:
      summary: "High query latency detected"
      description: "95th percentile query latency is {{ $value }}s, above 5s threshold"

  - alert: LowCacheHitRate
    expr: rag_cache_hit_rate{cache_type="query_result"} < 0.7
    for: 5m
    labels:
      severity: warning
      component: cache
    annotations:
      summary: "Low cache hit rate"
      description: "Query result cache hit rate is {{ $value | humanizePercentage }}, below 70%"

  # Resource alerts
  - alert: HighMemoryUsage
    expr: rag_memory_usage_bytes{component="weaviate"} / rag_memory_limit_bytes{component="weaviate"} > 0.85
    for: 2m
    labels:
      severity: warning
      component: weaviate
    annotations:
      summary: "High memory usage in Weaviate"
      description: "Weaviate memory usage is {{ $value | humanizePercentage }}, above 85%"

  - alert: DiskSpaceLow
    expr: rag_disk_usage_bytes{component="weaviate"} / rag_disk_capacity_bytes{component="weaviate"} > 0.9
    for: 1m
    labels:
      severity: critical
      component: weaviate
    annotations:
      summary: "Low disk space in Weaviate"
      description: "Weaviate disk usage is {{ $value | humanizePercentage }}, above 90%"

  # Business logic alerts
  - alert: IntentProcessingFailureRate
    expr: rate(rag_query_total{status="failed"}[10m]) / rate(rag_query_total[10m]) > 0.05
    for: 3m
    labels:
      severity: warning
      component: intent-processing
    annotations:
      summary: "High intent processing failure rate"
      description: "Intent failure rate is {{ $value | humanizePercentage }}, above 5%"

  - alert: EmbeddingAPIErrors
    expr: rate(rag_embedding_requests_total{status="failed"}[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
      component: embedding-service
    annotations:
      summary: "High embedding API error rate"
      description: "Embedding API errors: {{ $value }} requests/sec"

  # Data quality alerts
  - alert: LowDocumentQuality
    expr: avg(nephoran_knowledge_quality_score) < 0.7
    for: 10m
    labels:
      severity: warning
      component: knowledge-base
    annotations:
      summary: "Low document quality in knowledge base"
      description: "Average document quality score is {{ $value }}, below 0.7"

  - alert: StaleKnowledgeBase
    expr: time() - rag_last_document_processed_timestamp > 86400 * 7  # 7 days
    for: 0m
    labels:
      severity: info
      component: knowledge-base
    annotations:
      summary: "Knowledge base hasn't been updated recently"
      description: "Last document processed {{ $value | humanizeDuration }} ago"
```

### Log Management

#### Structured Logging Configuration

```yaml
# logging-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: logging-config
  namespace: nephoran-system
data:
  log-config.yaml: |
    version: 1
    disable_existing_loggers: false
    
    formatters:
      json:
        class: pythonjsonlogger.jsonlogger.JsonFormatter
        format: '%(asctime)s %(name)s %(levelname)s %(message)s'
      detailed:
        format: '%(asctime)s - %(name)s - %(levelname)s - %(module)s - %(funcName)s - %(message)s'
    
    handlers:
      console:
        class: logging.StreamHandler
        level: INFO
        formatter: json
        stream: ext://sys.stdout
      
      file:
        class: logging.handlers.RotatingFileHandler
        level: DEBUG
        formatter: detailed
        filename: /var/log/rag/rag-system.log
        maxBytes: 104857600  # 100MB
        backupCount: 5
    
    loggers:
      rag:
        level: INFO
        handlers: [console, file]
        propagate: false
      
      rag.query:
        level: DEBUG
        handlers: [console, file]
        propagate: false
      
      rag.embedding:
        level: INFO
        handlers: [console, file]
        propagate: false
      
      weaviate:
        level: WARNING
        handlers: [console]
        propagate: false
    
    root:
      level: INFO
      handlers: [console]
```

#### Log Aggregation with Fluentd

```yaml
# fluentd-rag-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-rag-config
data:
  fluent.conf: |
    <match kubernetes.var.log.containers.rag-api-**>
      @type elasticsearch
      host elasticsearch.logging.svc.cluster.local
      port 9200
      index_name rag-api-logs
      type_name _doc
      
      <buffer>
        @type file
        path /var/log/fluentd-buffers/rag-api.buffer
        flush_mode interval
        flush_interval 10s
        chunk_limit_size 10M
        queue_limit_length 32
        retry_max_interval 30
        retry_forever true
      </buffer>
    </match>

    <match kubernetes.var.log.containers.weaviate-**>
      @type elasticsearch
      host elasticsearch.logging.svc.cluster.local
      port 9200
      index_name weaviate-logs
      type_name _doc
      
      <filter>
        @type parser
        key_name message
        reserve_data true
        <parse>
          @type json
        </parse>
      </filter>
    </match>
```

## Performance Optimization

### System-Level Optimizations

#### 1. Kubernetes Resource Optimization

```yaml
# resource-optimization.yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: rag-system-limits
  namespace: nephoran-system
spec:
  limits:
  - default:
      cpu: "1000m"
      memory: "2Gi"
    defaultRequest:
      cpu: "500m"
      memory: "1Gi"
    type: Container
  - max:
      cpu: "4000m"
      memory: "16Gi"
    min:
      cpu: "100m"
      memory: "128Mi"
    type: Container

---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: rag-system-quota
  namespace: nephoran-system
spec:
  hard:
    requests.cpu: "20"
    requests.memory: "40Gi"
    limits.cpu: "40"
    limits.memory: "80Gi"
    persistentvolumeclaims: "10"
    requests.storage: "2Ti"
```

#### 2. Node Affinity and Pod Distribution

```yaml
# node-affinity.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: weaviate
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-type
                operator: In
                values: ["memory-optimized"]
              - key: storage-type
                operator: In
                values: ["nvme-ssd"]
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values: ["weaviate"]
              topologyKey: kubernetes.io/hostname
      tolerations:
      - key: "high-memory"
        operator: "Equal"
        value: "true"
        effect: "NoSchedule"
```

### Application-Level Optimizations

#### 1. Connection Pool Optimization

```go
// connection-pool-config.go
type OptimizedConnectionConfig struct {
    Weaviate struct {
        MaxIdleConns        int           `yaml:"max_idle_conns"`
        MaxOpenConns        int           `yaml:"max_open_conns"`
        ConnMaxLifetime     time.Duration `yaml:"conn_max_lifetime"`
        ConnMaxIdleTime     time.Duration `yaml:"conn_max_idle_time"`
        KeepAlive           time.Duration `yaml:"keep_alive"`
        TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout"`
    } `yaml:"weaviate"`
    
    Redis struct {
        PoolSize     int           `yaml:"pool_size"`
        MinIdleConns int           `yaml:"min_idle_conns"`
        MaxRetries   int           `yaml:"max_retries"`
        DialTimeout  time.Duration `yaml:"dial_timeout"`
        ReadTimeout  time.Duration `yaml:"read_timeout"`
        WriteTimeout time.Duration `yaml:"write_timeout"`
        PoolTimeout  time.Duration `yaml:"pool_timeout"`
    } `yaml:"redis"`
    
    OpenAI struct {
        MaxIdleConns     int           `yaml:"max_idle_conns"`
        MaxConnsPerHost  int           `yaml:"max_conns_per_host"`
        IdleConnTimeout  time.Duration `yaml:"idle_conn_timeout"`
        RequestTimeout   time.Duration `yaml:"request_timeout"`
        RateLimitBuffer  float64       `yaml:"rate_limit_buffer"`
    } `yaml:"openai"`
}

// Production configuration
func GetProductionConnectionConfig() *OptimizedConnectionConfig {
    return &OptimizedConnectionConfig{
        Weaviate: struct {
            MaxIdleConns        int           `yaml:"max_idle_conns"`
            MaxOpenConns        int           `yaml:"max_open_conns"`
            ConnMaxLifetime     time.Duration `yaml:"conn_max_lifetime"`
            ConnMaxIdleTime     time.Duration `yaml:"conn_max_idle_time"`
            KeepAlive           time.Duration `yaml:"keep_alive"`
            TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout"`
        }{
            MaxIdleConns:        50,
            MaxOpenConns:        100,
            ConnMaxLifetime:     30 * time.Minute,
            ConnMaxIdleTime:     5 * time.Minute,
            KeepAlive:           30 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
        },
        Redis: struct {
            PoolSize     int           `yaml:"pool_size"`
            MinIdleConns int           `yaml:"min_idle_conns"`
            MaxRetries   int           `yaml:"max_retries"`
            DialTimeout  time.Duration `yaml:"dial_timeout"`
            ReadTimeout  time.Duration `yaml:"read_timeout"`
            WriteTimeout time.Duration `yaml:"write_timeout"`
            PoolTimeout  time.Duration `yaml:"pool_timeout"`
        }{
            PoolSize:     100,
            MinIdleConns: 20,
            MaxRetries:   3,
            DialTimeout:  5 * time.Second,
            ReadTimeout:  3 * time.Second,
            WriteTimeout: 3 * time.Second,
            PoolTimeout:  4 * time.Second,
        },
        OpenAI: struct {
            MaxIdleConns     int           `yaml:"max_idle_conns"`
            MaxConnsPerHost  int           `yaml:"max_conns_per_host"`
            IdleConnTimeout  time.Duration `yaml:"idle_conn_timeout"`
            RequestTimeout   time.Duration `yaml:"request_timeout"`
            RateLimitBuffer  float64       `yaml:"rate_limit_buffer"`
        }{
            MaxIdleConns:    20,
            MaxConnsPerHost: 10,
            IdleConnTimeout: 90 * time.Second,
            RequestTimeout:  30 * time.Second,
            RateLimitBuffer: 0.9, // Use 90% of rate limit
        },
    }
}
```

#### 2. Caching Strategy Optimization

```go
// multi-level-cache.go
type MultiLevelCache struct {
    L1Cache    *lru.Cache        // In-memory LRU
    L2Cache    *redis.Client     // Redis distributed cache
    L3Cache    *FileCache        // Disk-based cache
    
    config     *CacheConfig
    metrics    *CacheMetrics
    compressor Compressor
}

type CacheConfig struct {
    L1 struct {
        MaxSize      int           `yaml:"max_size"`
        TTL          time.Duration `yaml:"ttl"`
        EvictionPolicy string      `yaml:"eviction_policy"`
    } `yaml:"l1"`
    
    L2 struct {
        TTL         time.Duration `yaml:"ttl"`
        MaxSize     int64         `yaml:"max_size"`
        Compression bool          `yaml:"compression"`
        Pipeline    bool          `yaml:"pipeline"`
    } `yaml:"l2"`
    
    L3 struct {
        Directory   string        `yaml:"directory"`
        MaxSize     int64         `yaml:"max_size"`
        TTL         time.Duration `yaml:"ttl"`
        Compression bool          `yaml:"compression"`
    } `yaml:"l3"`
    
    Prefetch struct {
        Enabled     bool          `yaml:"enabled"`
        Workers     int           `yaml:"workers"`
        BatchSize   int           `yaml:"batch_size"`
        Threshold   float64       `yaml:"threshold"`
    } `yaml:"prefetch"`
}

func (mlc *MultiLevelCache) Get(key string) (interface{}, bool, error) {
    // Try L1 cache first (fastest)
    if value, exists := mlc.L1Cache.Get(key); exists {
        mlc.metrics.RecordHit("L1")
        return value, true, nil
    }
    
    // Try L2 cache (Redis)
    ctx := context.Background()
    if data, err := mlc.L2Cache.Get(ctx, key).Bytes(); err == nil {
        mlc.metrics.RecordHit("L2")
        
        var value interface{}
        if mlc.config.L2.Compression {
            value, err = mlc.compressor.Decompress(data)
        } else {
            err = json.Unmarshal(data, &value)
        }
        
        if err != nil {
            return nil, false, err
        }
        
        // Promote to L1
        mlc.L1Cache.Add(key, value)
        return value, true, nil
    }
    
    // Try L3 cache (disk)
    if value, exists, err := mlc.L3Cache.Get(key); err == nil && exists {
        mlc.metrics.RecordHit("L3")
        
        // Promote to L2 and L1
        mlc.L2Cache.Set(ctx, key, value, mlc.config.L2.TTL)
        mlc.L1Cache.Add(key, value)
        return value, true, nil
    }
    
    mlc.metrics.RecordMiss()
    return nil, false, nil
}

func (mlc *MultiLevelCache) Set(key string, value interface{}, ttl time.Duration) error {
    // Store in all levels
    mlc.L1Cache.Add(key, value)
    
    ctx := context.Background()
    var data []byte
    var err error
    
    if mlc.config.L2.Compression {
        data, err = mlc.compressor.Compress(value)
    } else {
        data, err = json.Marshal(value)
    }
    
    if err != nil {
        return err
    }
    
    // L2 cache (Redis)
    mlc.L2Cache.Set(ctx, key, data, ttl)
    
    // L3 cache (disk) - async
    go func() {
        mlc.L3Cache.Set(key, value, ttl)
    }()
    
    return nil
}
```

#### 3. Query Optimization Strategies

```go
// query-optimizer.go
type QueryOptimizer struct {
    analyzer    *QueryAnalyzer
    planner     *ExecutionPlanner
    cache       *QueryCache
    metrics     *QueryMetrics
}

type OptimizedQuery struct {
    Original        string
    Optimized       string
    ExecutionPlan   *ExecutionPlan
    EstimatedCost   float64
    CacheKey        string
}

type ExecutionPlan struct {
    Strategy        string        // parallel, sequential, hybrid
    Partitions      []QueryPart   // For parallel execution
    CacheStrategy   string        // preload, lazy, none
    Timeout         time.Duration
    RetryPolicy     *RetryPolicy
}

func (qo *QueryOptimizer) OptimizeQuery(request *EnhancedSearchRequest) (*OptimizedQuery, error) {
    // Analyze query characteristics
    analysis := qo.analyzer.AnalyzeQuery(request.Query)
    
    // Generate cache key
    cacheKey := qo.generateCacheKey(request)
    
    // Check for cached query plan
    if plan, exists := qo.cache.GetExecutionPlan(cacheKey); exists {
        return &OptimizedQuery{
            Original:      request.Query,
            Optimized:     plan.OptimizedQuery,
            ExecutionPlan: plan,
            CacheKey:      cacheKey,
        }, nil
    }
    
    // Create execution plan based on query characteristics
    plan := qo.createExecutionPlan(analysis, request)
    
    // Optimize query text
    optimizedQuery := qo.optimizeQueryText(request.Query, analysis)
    
    optimized := &OptimizedQuery{
        Original:        request.Query,
        Optimized:       optimizedQuery,
        ExecutionPlan:   plan,
        EstimatedCost:   qo.estimateCost(plan),
        CacheKey:        cacheKey,
    }
    
    // Cache the plan
    qo.cache.SetExecutionPlan(cacheKey, plan)
    
    return optimized, nil
}

func (qo *QueryOptimizer) createExecutionPlan(analysis *QueryAnalysis, request *EnhancedSearchRequest) *ExecutionPlan {
    plan := &ExecutionPlan{
        Timeout:     30 * time.Second,
        RetryPolicy: &RetryPolicy{MaxRetries: 3, BackoffMultiplier: 2.0},
    }
    
    // Determine execution strategy
    if analysis.Complexity > 0.8 && analysis.HasMultipleConcepts {
        // Complex query - use parallel execution
        plan.Strategy = "parallel"
        plan.Partitions = qo.partitionQuery(analysis, request)
        plan.Timeout = 60 * time.Second
    } else if analysis.IsSimple && analysis.HasExactMatch {
        // Simple query - use cache-first strategy
        plan.Strategy = "cache_first"
        plan.CacheStrategy = "preload"
        plan.Timeout = 10 * time.Second
    } else {
        // Standard query - use hybrid approach
        plan.Strategy = "hybrid"
        plan.CacheStrategy = "lazy"
    }
    
    return plan
}
```

### Infrastructure Optimizations

#### 1. Storage Optimization

```yaml
# storage-optimization.yaml
# High-performance storage class for Weaviate
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: weaviate-high-performance
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp3
  iops: "16000"      # Maximum IOPS for gp3
  throughput: "1000"  # Maximum throughput for gp3
  fsType: ext4
  encrypted: "true"
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer

---
# Redis storage optimization
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: redis-optimized
provisioner: kubernetes.io/aws-ebs
parameters:
  type: io2
  iops: "10000"
  fsType: ext4
  encrypted: "true"
allowVolumeExpansion: true

---
# Weaviate StatefulSet with optimized storage
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: weaviate
spec:
  serviceName: weaviate-headless
  replicas: 5
  template:
    spec:
      containers:
      - name: weaviate
        image: semitechnologies/weaviate:1.22.4
        env:
        - name: QUERY_DEFAULTS_LIMIT
          value: "100"
        - name: QUERY_MAXIMUM_RESULTS
          value: "10000"
        - name: PERSISTENCE_DATA_PATH
          value: "/var/lib/weaviate"
        - name: DEFAULT_VECTORIZER_MODULE
          value: "none"
        - name: ENABLE_MODULES
          value: ""
        volumeMounts:
        - name: weaviate-data
          mountPath: /var/lib/weaviate
        - name: weaviate-backup
          mountPath: /var/lib/weaviate/backups
  volumeClaimTemplates:
  - metadata:
      name: weaviate-data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: weaviate-high-performance
      resources:
        requests:
          storage: 500Gi
  - metadata:
      name: weaviate-backup
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: standard
      resources:
        requests:
          storage: 200Gi
```

#### 2. Network Optimization

```yaml
# network-optimization.yaml
# Network policy for optimized traffic flow
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: rag-network-optimization
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app: rag-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
    - podSelector:
        matchLabels:
          app: llm-processor
    ports:
    - protocol: TCP
      port: 5001
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: weaviate
    ports:
    - protocol: TCP
      port: 8080
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
  - to: []  # Allow external OpenAI API calls
    ports:
    - protocol: TCP
      port: 443

---
# Service mesh configuration for optimized routing
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: rag-api-routing
spec:
  hosts:
  - rag-api
  http:
  - match:
    - headers:
        request-type:
          exact: batch
    route:
    - destination:
        host: rag-api
        subset: batch-optimized
      weight: 100
    timeout: 120s
  - route:
    - destination:
        host: rag-api
        subset: standard
      weight: 100
    timeout: 30s

---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: rag-api-subsets
spec:
  host: rag-api
  subsets:
  - name: standard
    labels:
      version: standard
  - name: batch-optimized
    labels:
      version: batch-optimized
    trafficPolicy:
      connectionPool:
        tcp:
          maxConnections: 100
        http:
          http1MaxPendingRequests: 100
          maxRequestsPerConnection: 2
      loadBalancer:
        simple: LEAST_CONN
```

## Security Operations

### Security Configuration

#### 1. Pod Security Standards

```yaml
# pod-security.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted

---
apiVersion: v1
kind: SecurityContext
metadata:
  name: rag-security-context
spec:
  runAsNonRoot: true
  runAsUser: 1001
  runAsGroup: 1001
  fsGroup: 1001
  seccompProfile:
    type: RuntimeDefault
  capabilities:
    drop:
    - ALL
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false

---
# Security policies for RAG components
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rag-api
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1001
        fsGroup: 1001
      containers:
      - name: rag-api
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1001
          capabilities:
            drop:
            - ALL
        volumeMounts:
        - name: tmp
          mountPath: /tmp
        - name: cache
          mountPath: /app/cache
      volumes:
      - name: tmp
        emptyDir: {}
      - name: cache
        emptyDir: {}
```

#### 2. Network Security Policies

```yaml
# network-security.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: rag-security-policy
  namespace: nephoran-system
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  
  # Default deny all
  ingress: []
  egress: []

---
# Allow internal communication
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: rag-internal-communication
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app: rag-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: llm-processor
    - podSelector:
        matchLabels:
          app: nephio-bridge
    ports:
    - protocol: TCP
      port: 5001
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: weaviate
    ports:
    - protocol: TCP
      port: 8080
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
  # Allow OpenAI API access
  - to: []
    ports:
    - protocol: TCP
      port: 443
    namespaceSelector: {}

---
# Weaviate security policy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: weaviate-security-policy
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app: weaviate
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: rag-api
    ports:
    - protocol: TCP
      port: 8080
  - from:
    - podSelector:
        matchLabels:
          app: weaviate  # Allow inter-cluster communication
    ports:
    - protocol: TCP
      port: 7000
    - protocol: TCP
      port: 7001
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: weaviate
    ports:
    - protocol: TCP
      port: 7000
    - protocol: TCP
      port: 7001
  # Allow DNS resolution
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
```

#### 3. Secret Management

```yaml
# secret-management.yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vault-backend
  namespace: nephoran-system
spec:
  provider:
    vault:
      server: "https://vault.company.com"
      path: "secret"
      version: "v2"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "nephoran-rag"

---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: rag-secrets
  namespace: nephoran-system
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: rag-secrets
    creationPolicy: Owner
    template:
      type: Opaque
  data:
  - secretKey: openai-api-key
    remoteRef:
      key: rag/openai
      property: api-key
  - secretKey: weaviate-api-key
    remoteRef:
      key: rag/weaviate
      property: api-key
  - secretKey: redis-password
    remoteRef:
      key: rag/redis
      property: password

---
# API key rotation job
apiVersion: batch/v1
kind: CronJob
metadata:
  name: api-key-rotation
  namespace: nephoran-system
spec:
  schedule: "0 2 1 * *"  # Monthly on the 1st at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: key-rotator
            image: vault:latest
            command:
            - /bin/sh
            - -c
            - |
              # Rotate OpenAI API key
              NEW_KEY=$(vault kv get -field=new-key secret/rag/openai-rotation)
              vault kv put secret/rag/openai api-key="$NEW_KEY"
              
              # Update Kubernetes secret
              kubectl patch secret rag-secrets -n nephoran-system \
                --type='json' \
                -p='[{"op": "replace", "path": "/data/openai-api-key", "value": "'$(echo -n "$NEW_KEY" | base64)'"}]'
              
              # Rolling restart RAG API pods
              kubectl rollout restart deployment/rag-api -n nephoran-system
          restartPolicy: OnFailure
```

### Security Monitoring

#### 1. Security Metrics and Alerts

```yaml
# security-alerts.yaml
groups:
- name: security-alerts
  rules:
  - alert: UnauthorizedAPIAccess
    expr: increase(rag_api_requests_total{status_code=~"401|403"}[5m]) > 10
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "Unauthorized API access attempts detected"
      description: "{{ $value }} unauthorized requests in the last 5 minutes"

  - alert: SuspiciousQueryPatterns
    expr: rate(rag_query_total{query_length > 10000}[10m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "Suspicious query patterns detected"
      description: "Abnormally long queries detected: {{ $value }} queries/sec"

  - alert: FailedSecretAccess
    expr: increase(vault_secret_access_failures[5m]) > 0
    for: 0m
    labels:
      severity: critical
    annotations:
      summary: "Failed secret access detected"
      description: "{{ $value }} failed secret access attempts"

  - alert: NetworkPolicyViolation
    expr: increase(cilium_policy_verdict_total{verdict="DENIED"}[5m]) > 0
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "Network policy violations detected"
      description: "{{ $value }} denied network connections"
```

#### 2. Security Scanning

```yaml
# security-scanning.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: security-scan
  namespace: nephoran-system
spec:
  schedule: "0 1 * * *"  # Daily at 1 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: trivy-scanner
            image: aquasec/trivy:latest
            command:
            - /bin/sh
            - -c
            - |
              # Scan RAG API image
              trivy image --exit-code 1 --severity HIGH,CRITICAL \
                --format json --output /tmp/rag-api-scan.json \
                nephoran/rag-api:latest
              
              # Scan Weaviate image
              trivy image --exit-code 1 --severity HIGH,CRITICAL \
                --format json --output /tmp/weaviate-scan.json \
                semitechnologies/weaviate:1.22.4
              
              # Upload results to security dashboard
              curl -X POST https://security-dashboard.company.com/api/scans \
                -H "Authorization: Bearer $SECURITY_TOKEN" \
                -F "rag-api=@/tmp/rag-api-scan.json" \
                -F "weaviate=@/tmp/weaviate-scan.json"
            env:
            - name: SECURITY_TOKEN
              valueFrom:
                secretKeyRef:
                  name: security-secrets
                  key: dashboard-token
            volumeMounts:
            - name: scan-results
              mountPath: /tmp
          volumes:
          - name: scan-results
            emptyDir: {}
          restartPolicy: OnFailure
```

This comprehensive operations guide provides production-ready procedures for deploying, monitoring, and maintaining the Nephoran RAG pipeline. The guide includes detailed configurations, monitoring setups, performance optimizations, security measures, and troubleshooting procedures that enable reliable operation at scale.