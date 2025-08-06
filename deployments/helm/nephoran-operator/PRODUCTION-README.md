# Nephoran Intent Operator - Production Helm Chart

## Overview

This Helm chart deploys the Nephoran Intent Operator in production-ready configuration with enterprise-grade security, scalability, and operational excellence features.

## Architecture

The Nephoran Intent Operator consists of five core components:

1. **Nephoran Operator Controller** - Main Kubernetes controller with webhook server
2. **LLM Processor** - Natural language processing service with GPT-4o-mini integration
3. **Nephio Bridge** - GitOps orchestration service for Nephio R5 integration
4. **RAG API** - Retrieval-Augmented Generation service with vector search
5. **Weaviate** - Vector database for telecommunications domain knowledge

## Production Features

### High Availability
- Multi-replica deployments with anti-affinity rules
- Leader election for controllers
- PodDisruptionBudgets ensure minimum availability during disruptions
- Rolling update strategy with zero-downtime deployments

### Security Hardening
- **Pod Security Standards**: Restricted security context with non-root users
- **Network Policies**: Default deny-all with specific allow rules
- **RBAC**: Minimal required permissions per component
- **Secrets Management**: External Secrets Operator integration
- **Security Contexts**: `runAsNonRoot`, `readOnlyRootFilesystem`, dropped capabilities

### Scalability
- **Horizontal Pod Autoscaling**: CPU/Memory + custom metrics based scaling
- **KEDA Integration**: Advanced scaling with Prometheus metrics
- **Resource Optimization**: Realistic resource requests/limits per workload
- **Node Affinity**: Optimized pod placement across node types

### Observability
- **Prometheus Metrics**: Comprehensive business and technical metrics
- **Jaeger Tracing**: Distributed request tracing across all components  
- **Structured Logging**: JSON-formatted logs with correlation IDs
- **Health Checks**: Liveness, readiness, and startup probes

### Operational Excellence
- **GitOps Integration**: Nephio R5 package orchestration
- **Disaster Recovery**: Automated backups and restore procedures
- **Circuit Breakers**: Fault isolation and automatic recovery
- **Configuration Management**: Externalized configuration via ConfigMaps

## Quick Start

### Prerequisites

```bash
# Add required Helm repositories
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add kedacore https://kedacore.github.io/charts
helm repo update

# Install dependencies
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring --create-namespace
helm install keda kedacore/keda -n keda-system --create-namespace
```

### Production Deployment

```bash
# Create namespace
kubectl create namespace nephoran-system

# Install with production values
helm install nephoran-operator . \
  -n nephoran-system \
  -f values-production.yaml \
  --wait --timeout=10m
```

## Configuration

### Core Components

#### Nephoran Operator Controller
- **Replicas**: 2 (leader election enabled)
- **Resources**: 200m-1000m CPU, 256Mi-1Gi memory
- **Leader Election**: 15s lease duration, 10s renew deadline
- **Health Checks**: 30s liveness, 10s readiness, startup probe

#### LLM Processor  
- **Replicas**: 3-10 (HPA enabled)
- **Resources**: 500m-2000m CPU, 512Mi-2Gi memory
- **Autoscaling**: 70% CPU, 80% memory thresholds
- **Models**: GPT-4o-mini (primary), Mistral-8x22b (fallback)

#### RAG API
- **Replicas**: 2-8 (HPA enabled) 
- **Resources**: 500m-2000m CPU, 1Gi-4Gi memory
- **Vector Search**: Sub-200ms P95 latency target
- **Caching**: 300s TTL, 1000 entry capacity

#### Weaviate Database
- **Replicas**: 3 (clustered)
- **Resources**: 1000m-4000m CPU, 2Gi-8Gi memory
- **Storage**: 500Gi persistent volumes (fast-ssd)
- **Backup**: Daily automated backups

### Security Configuration

```yaml
# Security contexts applied to all components
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  runAsGroup: 65534
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault

# Network policies with default deny
networkPolicy:
  enabled: true
  denyAll: true
  # Specific allow rules defined per component
```

### Node Requirements

```yaml
# Recommended node labels and taints
nodeSelector:
  # Control plane nodes
  nephoran.com/node-type: "control-plane"
  
  # AI processing nodes (GPU optional)
  nephoran.com/node-type: "ai-processing"
  
  # Database nodes (high IOPS storage)
  nephoran.com/node-type: "database"
  
  # Orchestration nodes
  nephoran.com/node-type: "orchestration"

tolerations:
  - key: "nephoran.com/ai-workload"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
```

## Monitoring & Alerting

### Key Metrics

- **Intent Processing Rate**: Target 45 intents/minute
- **LLM Response Latency**: P95 < 2 seconds
- **Vector Search Latency**: P95 < 200ms
- **System Availability**: 99.95% uptime SLA
- **Error Rates**: < 0.1% for critical operations

### Critical Alerts

- `NephoranOperatorDown`: Any component unavailable > 5 minutes
- `LLMProcessorHighLatency`: P95 latency > 5 seconds
- `WeaviateDatabaseDown`: Vector database unreachable > 2 minutes
- `HighIntentFailureRate`: Intent failures > 10%

### Grafana Dashboards

```bash
# Import production dashboards
kubectl apply -f dashboards/nephoran-overview.json
kubectl apply -f dashboards/llm-processor-metrics.json
kubectl apply -f dashboards/weaviate-performance.json
```

## Backup & Disaster Recovery

### Automated Backups

```yaml
backup:
  enabled: true
  schedule: "0 2 * * *"  # Daily at 2 AM
  retention: "30d"
  storage:
    type: "s3"
    bucket: "nephoran-backups-prod"
    region: "us-central1"
```

### Recovery Procedures

```bash
# Restore from backup
kubectl create job --from=cronjob/nephoran-backup-restore restore-$(date +%Y%m%d)

# Verify restoration
helm test nephoran-operator -n nephoran-system
```

## Scaling Operations

### Manual Scaling

```bash
# Scale LLM Processor
kubectl scale deployment nephoran-operator-llm-processor --replicas=5

# Scale RAG API  
kubectl scale deployment nephoran-operator-rag-api --replicas=4
```

### Custom Metrics Scaling

```yaml
# KEDA ScaledObject for advanced scaling
triggers:
  - type: prometheus
    metadata:
      query: |
        sum(rate(llm_processor_requests_pending_total[2m]))
      threshold: "10"
```

## Troubleshooting

### Common Issues

#### High Memory Usage
```bash
# Check memory metrics
kubectl top pods -n nephoran-system

# Review memory limits
kubectl describe hpa nephoran-operator-llm-processor
```

#### Vector Search Latency
```bash
# Check Weaviate cluster health
kubectl exec -it nephoran-operator-weaviate-0 -- \
  curl http://localhost:8080/v1/.well-known/ready

# Review vector search metrics
kubectl logs -f deployment/nephoran-operator-rag-api | grep vector_search
```

#### Intent Processing Failures
```bash
# Check controller logs
kubectl logs -f deployment/nephoran-operator-controller | grep "intent-processing"

# Verify Nephio Bridge connectivity
kubectl exec -it nephoran-operator-nephio-bridge-0 -- \
  curl http://porch-server.porch-system.svc.cluster.local:7007/ready
```

### Debug Mode

```bash
# Enable debug logging
kubectl patch deployment nephoran-operator-controller -p \
  '{"spec":{"template":{"spec":{"containers":[{"name":"manager","args":["--zap-log-level=debug"]}]}}}}'
```

## Security Considerations

### Certificate Management
- Webhook certificates auto-rotated by cert-manager
- TLS 1.2+ enforced for all communications
- mTLS between internal components

### Secrets Management
```yaml
# External Secrets Operator integration
secrets:
  external:
    enabled: true
    secretStore: "vault-backend"
```

### Network Security
- All egress traffic through authenticated proxies
- Inter-pod communication encrypted
- External API calls over HTTPS only

## Performance Tuning

### JVM/Runtime Optimization
```yaml
# Go runtime optimization
env:
  - name: GOMEMLIMIT
    value: "1800MiB"
  - name: GOMAXPROCS
    value: "2"
```

### Database Tuning
```yaml
# Weaviate performance settings
config:
  GOMEMLIMIT: "6GiB"
  GOMAXPROCS: "3"
  LIMIT_RESOURCES: "true"
```

## Support & Maintenance

### Health Checks
```bash
# Overall system health
kubectl get pods -n nephoran-system
kubectl get hpa -n nephoran-system
kubectl get pdb -n nephoran-system

# Component-specific health
curl -k https://nephoran-api.production.example.com/api/v1/llm/health
```

### Log Aggregation
```bash
# Centralized logging with structured output
kubectl logs -f -l app.kubernetes.io/name=nephoran-operator --all-containers=true
```

### Maintenance Windows
- Rolling updates during low-traffic periods
- Database maintenance coordinated with backup schedules
- Certificate renewals automated via cert-manager

## License

This production deployment guide is part of the Nephoran Intent Operator project. Please refer to the main LICENSE file for terms and conditions.

---

For additional support, please contact the Nephoran development team or refer to the main project documentation at [docs.nephoran.com](https://docs.nephoran.com).