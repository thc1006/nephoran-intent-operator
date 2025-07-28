# Weaviate Deployment Guide for Nephoran Intent Operator

## Overview

This guide provides comprehensive instructions for deploying a production-ready Weaviate vector database cluster optimized for the Nephoran Intent Operator's RAG system. The deployment includes advanced security, monitoring, backup automation, and telecom-specific optimizations.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Nephoran System Architecture                 │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │ RAG API     │    │ LLM         │    │ Intent Controller   │  │
│  │ Service     │◄──►│ Processor   │◄──►│                     │  │
│  └─────────────┘    └─────────────┘    └─────────────────────┘  │
│         │                   │                        │          │
│         └───────────────────┼────────────────────────┘          │
│                             ▼                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              Weaviate Vector Database Cluster              │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │ │
│  │  │ Weaviate    │  │ Weaviate    │  │ Weaviate            │ │ │
│  │  │ Pod 1       │  │ Pod 2       │  │ Pod 3               │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘ │ │
│  │                                                             │ │
│  │  Storage Classes: TelecomKnowledge, IntentPatterns,        │ │
│  │                   NetworkFunctions                         │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                             │                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    Supporting Services                     │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │ │
│  │  │ Monitoring  │  │ Backup      │  │ Network Policies    │ │ │
│  │  │ (Prometheus)│  │ Automation  │  │ & Security          │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

### System Requirements

- **Kubernetes Cluster**: Version 1.25+ with at least 3 nodes
- **Storage**: 500GB+ high-performance SSD storage per cluster
- **Memory**: 16GB+ RAM per node for Weaviate pods
- **CPU**: 4+ cores per node for optimal performance
- **Network**: Pod-to-pod communication with cluster networking

### Required Tools

```bash
# Install required CLI tools
kubectl --version  # >= 1.25
curl --version
python3 --version  # >= 3.8
jq --version
```

### Environment Variables

```bash
export OPENAI_API_KEY="your-openai-api-key-here"
export ENVIRONMENT="production"  # or staging/development
export NAMESPACE="nephoran-system"
```

### Storage Classes

Ensure your cluster has appropriate storage classes:

```bash
# Check available storage classes
kubectl get storageclass

# For AWS EKS
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
EOF
```

## Quick Start Deployment

### 1. Automated Deployment

```bash
cd deployments/weaviate
chmod +x deploy-weaviate.sh
./deploy-weaviate.sh
```

### 2. Manual Step-by-Step Deployment

```bash
# 1. Create namespace
kubectl create namespace nephoran-system
kubectl label namespace nephoran-system name=nephoran-system

# 2. Create secrets
kubectl create secret generic openai-api-key \
  --from-literal=api-key="$OPENAI_API_KEY" \
  --namespace=nephoran-system

# 3. Deploy RBAC
kubectl apply -f rbac.yaml

# 4. Deploy network policies
kubectl apply -f network-policy.yaml

# 5. Deploy Weaviate cluster
kubectl apply -f weaviate-deployment.yaml

# 6. Wait for readiness
kubectl wait --for=condition=available deployment/weaviate \
  --namespace=nephoran-system --timeout=600s

# 7. Deploy monitoring
kubectl apply -f monitoring.yaml

# 8. Deploy backup automation
kubectl apply -f backup-cronjob.yaml

# 9. Initialize schemas
python3 telecom-schema.py
```

## Configuration

### Production Tuning

The deployment includes several production optimizations:

#### Resource Allocation
```yaml
resources:
  requests:
    memory: "4Gi"
    cpu: "1000m"
    ephemeral-storage: "10Gi"
  limits:
    memory: "16Gi"
    cpu: "4000m"
    ephemeral-storage: "50Gi"
```

#### High Availability
- **Pod Anti-Affinity**: Ensures pods are distributed across nodes
- **Pod Disruption Budget**: Maintains minimum availability during updates
- **Horizontal Pod Autoscaler**: Scales based on CPU/memory usage

#### Security Hardening
- **Pod Security Context**: Non-root execution, read-only filesystem where possible
- **Network Policies**: Restricted ingress/egress traffic
- **RBAC**: Least-privilege access controls
- **Secret Management**: Proper secret rotation and access

### Environment-Specific Configurations

#### Development
```bash
export ENVIRONMENT="development"
# Reduced resources, single replica, local storage
```

#### Staging
```bash
export ENVIRONMENT="staging"
# Production-like setup with reduced scale
```

#### Production
```bash
export ENVIRONMENT="production"
# Full scale with all production features
```

## Schema Management

### Telecom Knowledge Schema

The deployment creates three main schema classes:

1. **TelecomKnowledge**: Core telecommunications documentation
2. **IntentPatterns**: Natural language intent recognition
3. **NetworkFunctions**: 5G/O-RAN network function definitions

### Schema Properties

#### TelecomKnowledge Class
- `content`: Main document content
- `source`: Organization (3GPP, O-RAN, ETSI)
- `specification`: Spec ID (TS 23.501, O-RAN.WG1)
- `version`: Specification version
- `release`: 3GPP Release or O-RAN version
- `category`: Document type
- `domain`: Technical domain (RAN, Core, etc.)
- `keywords`: Technical keywords
- `networkFunctions`: Referenced NFs
- `interfaces`: Referenced interfaces

### Adding Custom Data

```python
import weaviate
from datetime import datetime

client = weaviate.Client("http://weaviate:8080")

# Add telecom knowledge
data_object = {
    "content": "Your telecom specification content...",
    "title": "Document Title",
    "source": "3GPP",
    "specification": "TS 23.501",
    "version": "17.9.0",
    "release": "Rel-17",
    "category": "Architecture",
    "domain": "Core",
    "keywords": ["NF", "SBA", "API"],
    "networkFunctions": ["AMF", "SMF"],
    "interfaces": ["N1", "N2"],
    "useCase": "eMBB",
    "priority": 8,
    "confidence": 0.95,
    "lastUpdated": datetime.utcnow().isoformat() + "Z"
}

client.data_object.create(data_object, "TelecomKnowledge")
```

## Monitoring and Observability

### Metrics

The deployment exposes comprehensive metrics:

- **Cluster Health**: Pod status, resource usage
- **Query Performance**: Latency, throughput, error rates
- **Storage Usage**: Disk utilization, backup status
- **Business Metrics**: Knowledge base size, query patterns

### Dashboards

Grafana dashboards are automatically provisioned:

- **Weaviate Overview**: Cluster health and performance
- **Query Analytics**: Search patterns and performance
- **Storage Monitoring**: Capacity and backup status
- **Telecom Metrics**: Domain-specific KPIs

### Alerting

Critical alerts are configured for:

- Pod/cluster unavailability
- High resource usage (>85% memory, >80% CPU)
- Query latency degradation
- Backup failures
- Knowledge base inconsistencies

### Accessing Monitoring

```bash
# Port forward to Grafana (if deployed)
kubectl port-forward svc/grafana 3000:3000 -n monitoring

# Access Prometheus metrics directly
kubectl port-forward svc/weaviate 2112:2112 -n nephoran-system
curl http://localhost:2112/metrics
```

## Backup and Recovery

### Automated Backups

Daily backups are configured via CronJob:

```yaml
schedule: "0 2 * * *"  # Daily at 2 AM UTC
retention: "30d"       # Keep for 30 days
```

### Manual Backup

```bash
# Trigger immediate backup
kubectl create job weaviate-backup-manual \
  --from=cronjob/weaviate-backup \
  --namespace=nephoran-system

# Check backup status
kubectl logs -l component=backup -n nephoran-system
```

### Disaster Recovery

#### Full Cluster Recovery
```bash
# 1. Deploy new cluster
./deploy-weaviate.sh --cleanup

# 2. Restore from backup
kubectl run restore-job --image=curlimages/curl \
  --env="BACKUP_ID=nephoran-backup-20241128-020000" \
  --namespace=nephoran-system \
  -- /scripts/restore-from-backup.sh
```

#### Data Recovery
```bash
# List available backups
kubectl exec -it deployment/weaviate -n nephoran-system -- \
  ls -la /var/lib/weaviate/backups/

# Restore specific backup
curl -X POST \
  -H "Authorization: Bearer $(kubectl get secret weaviate-api-key -n nephoran-system -o jsonpath='{.data.api-key}' | base64 -d)" \
  -H "Content-Type: application/json" \
  -d '{"id":"backup-20241128-020000","include":["TelecomKnowledge"]}' \
  "http://localhost:8080/v1/backups/filesystem/backup-20241128-020000/restore"
```

## Troubleshooting

### Common Issues

#### 1. Pods Not Starting

```bash
# Check pod status
kubectl get pods -n nephoran-system -o wide

# Describe problem pods
kubectl describe pod weaviate-xxx -n nephoran-system

# Check logs
kubectl logs weaviate-xxx -n nephoran-system

# Common causes:
# - Insufficient resources
# - Storage class issues
# - Secret not found
# - Network policy blocking traffic
```

#### 2. Storage Issues

```bash
# Check PVC status
kubectl get pvc -n nephoran-system

# Check storage class
kubectl get storageclass

# Check node disk space
kubectl describe nodes | grep -A 5 "Allocated resources"

# Fix: Expand PVC or add storage
kubectl patch pvc weaviate-pvc -n nephoran-system \
  -p '{"spec":{"resources":{"requests":{"storage":"750Gi"}}}}'
```

#### 3. API Connectivity Issues

```bash
# Test service connectivity
kubectl run test-pod --image=curlimages/curl --rm -i \
  --namespace=nephoran-system \
  -- curl -v http://weaviate:8080/v1/.well-known/ready

# Check network policies
kubectl get networkpolicy -n nephoran-system

# Test DNS resolution
kubectl run test-dns --image=curlimages/curl --rm -i \
  --namespace=nephoran-system \
  -- nslookup weaviate
```

#### 4. Performance Issues

```bash
# Check resource usage
kubectl top pods -n nephoran-system

# Check HPA status
kubectl get hpa -n nephoran-system

# Check query performance
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
curl "http://localhost:8080/v1/meta" | jq
```

#### 5. Schema Issues

```bash
# Check schema status
curl "http://localhost:8080/v1/schema" | jq

# Reinitialize schemas
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: schema-reinit
  namespace: nephoran-system
spec:
  template:
    spec:
      containers:
      - name: schema-init
        image: python:3.11-slim
        command: ["python3", "/scripts/telecom-schema.py"]
        env:
        - name: WEAVIATE_URL
          value: "http://weaviate:8080"
        volumeMounts:
        - name: script
          mountPath: /scripts
      volumes:
      - name: script
        configMap:
          name: telecom-schema-script
      restartPolicy: Never
EOF
```

### Debug Commands

```bash
# Get comprehensive cluster info
kubectl cluster-info dump --namespace=nephoran-system

# Check all resources
kubectl get all -n nephoran-system

# View recent events
kubectl get events -n nephoran-system --sort-by='.firstTimestamp'

# Export configuration for analysis
kubectl get deployment weaviate -n nephoran-system -o yaml > weaviate-config.yaml

# Network debugging
kubectl run netshoot --image=nicolaka/netshoot --rm -i --tty

# Resource usage over time
kubectl top nodes
kubectl top pods -n nephoran-system --sort-by=memory
```

### Log Analysis

```bash
# Stream all logs
kubectl logs -f -l app=weaviate -n nephoran-system

# Get logs with timestamps
kubectl logs --timestamps=true deployment/weaviate -n nephoran-system

# Filter for errors
kubectl logs deployment/weaviate -n nephoran-system | grep -i error

# Export logs for analysis
kubectl logs deployment/weaviate -n nephoran-system > weaviate.log
```

## Performance Optimization

### Vector Index Tuning

```python
# Optimize for telecom workloads
vector_config = {
    "ef": 128,                    # Search accuracy
    "efConstruction": 256,        # Build accuracy
    "maxConnections": 32,         # Index density
    "vectorCacheMaxObjects": 1000000,  # Cache size
    "dynamicEfMin": 64,
    "dynamicEfMax": 256,
}
```

### Query Optimization

```python
# Use hybrid search for best results
result = client.query.get("TelecomKnowledge", ["content", "title"]) \
    .with_hybrid(
        query="5G AMF registration procedure",
        alpha=0.7  # Balance between keyword and vector search
    ) \
    .with_additional(["score"]) \
    .with_limit(10) \
    .do()
```

### Resource Scaling

```bash
# Scale deployment
kubectl scale deployment weaviate --replicas=5 -n nephoran-system

# Update HPA
kubectl patch hpa weaviate-hpa -n nephoran-system \
  -p '{"spec":{"maxReplicas":10}}'

# Increase resource limits
kubectl patch deployment weaviate -n nephoran-system \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"weaviate","resources":{"limits":{"memory":"32Gi","cpu":"8000m"}}}]}}}}'
```

## Security Best Practices

### Network Security
- Use network policies to restrict traffic
- Implement service mesh for advanced security
- Regular security scanning of images

### Access Control
- Rotate API keys regularly
- Use least-privilege RBAC
- Implement audit logging

### Data Protection
- Enable encryption at rest
- Use secure backup storage
- Implement data retention policies

## Integration with Nephoran System

### RAG API Integration

```python
# Example RAG API integration
from weaviate_client import WeaviateClient

client = WeaviateClient("http://weaviate.nephoran-system.svc.cluster.local:8080")

def process_intent(user_intent: str) -> dict:
    # Search knowledge base
    results = client.query.get("TelecomKnowledge") \
        .with_near_text({"concepts": [user_intent]}) \
        .with_limit(5) \
        .do()
    
    # Process with LLM
    context = "\n".join([r["content"] for r in results["data"]["Get"]["TelecomKnowledge"]])
    
    return {
        "structured_output": llm_process(user_intent, context),
        "sources": results,
        "confidence": calculate_confidence(results)
    }
```

### Health Check Integration

```bash
# Add to Kubernetes readiness probe
livenessProbe:
  httpGet:
    path: /v1/.well-known/live
    port: 8080
  initialDelaySeconds: 120
  periodSeconds: 30
```

## Maintenance

### Regular Tasks

```bash
# Weekly: Check cluster health
kubectl get all -n nephoran-system
kubectl top pods -n nephoran-system

# Monthly: Review metrics and alerts
# Quarterly: Update Weaviate version
# Annually: Review and rotate secrets
```

### Updates and Upgrades

```bash
# Update Weaviate version
kubectl set image deployment/weaviate \
  weaviate=semitechnologies/weaviate:1.29.0 \
  -n nephoran-system

# Monitor rollout
kubectl rollout status deployment/weaviate -n nephoran-system

# Rollback if needed
kubectl rollout undo deployment/weaviate -n nephoran-system
```

## Support and Resources

### Documentation
- [Weaviate Documentation](https://docs.weaviate.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [3GPP Specifications](https://www.3gpp.org/specifications)
- [O-RAN Specifications](https://www.o-ran.org/specifications)

### Community
- [Weaviate Community](https://weaviate.io/community)
- [Kubernetes Community](https://kubernetes.io/community/)

### Troubleshooting Support
- Check deployment logs: `/tmp/weaviate-deploy-*.log`
- Review monitoring dashboards
- Consult alerting system
- Contact system administrators

---

**Note**: This deployment is optimized for production use with the Nephoran Intent Operator. For development or testing, consider reducing resource allocations and replica counts.