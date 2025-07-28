# Weaviate Operations Runbook for Nephoran Intent Operator

## Overview

This comprehensive operations runbook provides detailed procedures for deploying, managing, and troubleshooting the Weaviate vector database cluster within the Nephoran Intent Operator's RAG system. It covers day-to-day operations, incident response, maintenance procedures, and disaster recovery scenarios.

## Table of Contents

1. [Deployment Procedures](#deployment-procedures)
2. [Daily Operations](#daily-operations)
3. [Monitoring and Alerting](#monitoring-and-alerting)
4. [Troubleshooting Guide](#troubleshooting-guide)
5. [Maintenance Procedures](#maintenance-procedures)
6. [Backup and Recovery](#backup-and-recovery)
7. [Disaster Recovery](#disaster-recovery)
8. [Performance Tuning](#performance-tuning)
9. [Security Operations](#security-operations)
10. [Incident Response](#incident-response)

## Deployment Procedures

### Pre-Deployment Checklist

**Infrastructure Requirements:**
- [ ] Kubernetes cluster v1.25+ with at least 3 worker nodes
- [ ] Storage class `fast-ssd` configured for high-performance storage
- [ ] Standard storage class for backup volumes
- [ ] Network policies enabled on cluster
- [ ] Prometheus monitoring stack deployed (optional but recommended)

**Resources Verification:**
```bash
# Check cluster resources
kubectl top nodes
kubectl get storageclass
kubectl get nodes -o wide

# Verify minimum resources per node
# - 16GB+ RAM per node
# - 4+ CPU cores per node
# - 500GB+ available storage
```

**Environment Setup:**
```bash
# Set required environment variables
export OPENAI_API_KEY="your-openai-api-key-here"
export ENVIRONMENT="production"  # or staging/development
export NAMESPACE="nephoran-system"
export WEAVIATE_VERSION="1.28.1"
export BACKUP_SCHEDULE="0 2 * * *"  # Daily at 2 AM UTC
```

### Automated Deployment

**Quick Start Deployment:**
```bash
# Navigate to deployment directory
cd deployments/weaviate

# Run automated deployment script
chmod +x deploy-weaviate.sh
./deploy-weaviate.sh

# Verify deployment
make verify-weaviate-deployment
```

**Manual Deployment Steps:**

1. **Create Namespace and Secrets:**
```bash
# Create namespace
kubectl create namespace nephoran-system
kubectl label namespace nephoran-system name=nephoran-system

# Create OpenAI API secret
kubectl create secret generic openai-api-key \
  --from-literal=api-key="$OPENAI_API_KEY" \
  --namespace=nephoran-system

# Create Weaviate API secret
kubectl create secret generic weaviate-api-key \
  --from-literal=api-key="nephoran-rag-key-production" \
  --namespace=nephoran-system
```

2. **Deploy Core Components:**
```bash
# Deploy RBAC
kubectl apply -f rbac.yaml

# Deploy network policies
kubectl apply -f network-policy.yaml

# Deploy storage resources
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: weaviate-pvc
  namespace: nephoran-system
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 500Gi
  storageClassName: fast-ssd
EOF

kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: weaviate-backup-pvc
  namespace: nephoran-system
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 200Gi
  storageClassName: standard
EOF
```

3. **Deploy Weaviate Cluster:**
```bash
# Deploy main application
kubectl apply -f weaviate-deployment.yaml

# Wait for deployment to be ready
kubectl wait --for=condition=available deployment/weaviate \
  --namespace=nephoran-system --timeout=600s

# Verify pods are running
kubectl get pods -n nephoran-system -l app=weaviate
```

4. **Deploy Supporting Services:**
```bash
# Deploy monitoring
kubectl apply -f monitoring.yaml

# Deploy backup automation
kubectl apply -f backup-cronjob.yaml

# Initialize schemas
python3 telecom-schema.py
```

### Post-Deployment Verification

**Health Checks:**
```bash
# Check deployment status
kubectl get deployments -n nephoran-system
kubectl get pods -n nephoran-system -l app=weaviate

# Verify services
kubectl get services -n nephoran-system -l app=weaviate

# Test connectivity
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
curl http://localhost:8080/v1/.well-known/ready
curl http://localhost:8080/v1/.well-known/live
```

**Schema Verification:**
```bash
# Check schema creation
curl http://localhost:8080/v1/schema | jq '.classes[] | .class'

# Verify object count (should be 0 initially)
curl http://localhost:8080/v1/objects | jq '.objects | length'
```

## Daily Operations

### Morning Health Check Routine (10 minutes)

**System Status Verification:**
```bash
#!/bin/bash
# daily-health-check.sh

echo "=== Nephoran Weaviate Daily Health Check $(date) ==="

# Check pod status
echo "Pod Status:"
kubectl get pods -n nephoran-system -l app=weaviate -o wide

# Check resource usage
echo "Resource Usage:"
kubectl top pods -n nephoran-system -l app=weaviate

# Check persistent volume status
echo "Storage Status:"
kubectl get pvc -n nephoran-system

# Test API connectivity
echo "API Health:"
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 5

if curl -s http://localhost:8080/v1/.well-known/ready > /dev/null; then
    echo "✅ Weaviate API is healthy"
else
    echo "❌ Weaviate API is unhealthy"
fi

# Check backup status
echo "Backup Status:"
kubectl get cronjobs -n nephoran-system -l component=backup

# Kill port-forward
kill $PORT_FORWARD_PID 2>/dev/null

echo "=== Health Check Complete ==="
```

**Key Metrics to Monitor:**
- Pod readiness and liveness status
- Memory usage (should be <80% of limit)
- CPU usage (should be <70% of limit)
- Storage usage (should be <85% of capacity)
- API response time (<500ms for health checks)

### Knowledge Base Management

**Check Knowledge Base Statistics:**
```bash
# Get object count by class
curl -s http://localhost:8080/v1/objects \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq '.objects | group_by(.class) | map({class: .[0].class, count: length})'

# Get schema information
curl -s http://localhost:8080/v1/schema \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq '.classes[] | {class: .class, properties: (.properties | length)}'
```

**Add New Documents:**
```bash
# Upload documents via RAG API
curl -X POST http://rag-api:5001/knowledge/upload \
  -F "files=@new_telecom_spec.pdf" \
  -F "files=@updated_oran_doc.md"

# Or populate from directory
curl -X POST http://rag-api:5001/knowledge/populate \
  -H "Content-Type: application/json" \
  -d '{"path": "/app/knowledge_base/new_docs"}'
```

## Monitoring and Alerting

### Key Performance Indicators (KPIs)

**Availability Metrics:**
- Cluster uptime: >99.9%
- API response success rate: >99.5%
- Pod restart rate: <1 restart/day per pod

**Performance Metrics:**
- Query latency (p95): <500ms
- Object ingestion rate: >100 objects/minute
- Memory utilization: 60-80% of allocated
- CPU utilization: 40-70% of allocated

**Business Metrics:**
- Knowledge base size: Track growth over time
- Query patterns: Most frequent search terms
- Cache hit rate: >70% for RAG queries

### Prometheus Queries

**Essential Alerts:**
```yaml
# Weaviate Pod Down
- alert: WeaviatePodDown
  expr: up{job="weaviate"} == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "Weaviate pod is down"
    description: "Weaviate pod has been down for more than 1 minute"

# High Memory Usage
- alert: WeaviateHighMemoryUsage
  expr: container_memory_usage_bytes{pod=~"weaviate-.*"} / container_spec_memory_limit_bytes > 0.85
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Weaviate high memory usage"
    description: "Weaviate memory usage is above 85%"

# Query Latency High
- alert: WeaviateSlowQueries
  expr: histogram_quantile(0.95, weaviate_query_duration_seconds_bucket) > 5
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Weaviate slow queries"
    description: "95th percentile query latency is above 5 seconds"
```

### Grafana Dashboard Configuration

**Key Panels:**
1. **Cluster Overview**: Pod status, resource usage, uptime
2. **Query Performance**: Latency histograms, throughput metrics
3. **Storage Metrics**: Disk usage, IOPS, backup status
4. **Business Metrics**: Object counts, search patterns

**Dashboard JSON Export:**
```bash
# Export current dashboard configuration
kubectl get configmap weaviate-grafana-dashboard -n monitoring -o jsonpath='{.data.dashboard\.json}' > weaviate-dashboard.json
```

## Troubleshooting Guide

### Common Issues and Solutions

#### 1. Pods Not Starting

**Symptoms:**
- Pods stuck in Pending or CrashLoopBackOff state
- Events showing scheduling or resource issues

**Diagnosis:**
```bash
# Check pod status and events
kubectl describe pod -n nephoran-system -l app=weaviate

# Check resource availability
kubectl describe nodes
kubectl top nodes

# Check storage issues
kubectl get pvc -n nephoran-system
kubectl describe pvc weaviate-pvc -n nephoran-system
```

**Common Causes and Solutions:**

**Resource Constraints:**
```bash
# Check if nodes have sufficient resources
kubectl describe nodes | grep -A 5 "Allocated resources"

# Solution: Scale cluster or adjust resource requests
kubectl patch deployment weaviate -n nephoran-system \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"weaviate","resources":{"requests":{"memory":"2Gi","cpu":"500m"}}}]}}}}'
```

**Storage Issues:**
```bash
# Check storage class availability
kubectl get storageclass

# Check PVC binding
kubectl get pvc -n nephoran-system

# Solution: Create missing storage class or fix PVC
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp3
allowVolumeExpansion: true
EOF
```

#### 2. API Connectivity Issues

**Symptoms:**
- Health check endpoints returning errors
- RAG API unable to connect to Weaviate
- Timeout errors in application logs

**Diagnosis:**
```bash
# Test direct connectivity
kubectl exec -it deployment/weaviate -n nephoran-system -- curl localhost:8080/v1/.well-known/ready

# Check service configuration
kubectl get svc weaviate -n nephoran-system -o yaml

# Check network policies
kubectl get networkpolicy -n nephoran-system
```

**Solutions:**

**Service Discovery Issues:**
```bash
# Verify service endpoints
kubectl get endpoints weaviate -n nephoran-system

# Test DNS resolution
kubectl run test-dns --image=busybox --rm -i --tty \
  --namespace=nephoran-system \
  -- nslookup weaviate
```

**Network Policy Blocking:**
```bash
# Temporarily disable network policies for testing
kubectl delete networkpolicy weaviate-netpol -n nephoran-system

# If this fixes the issue, update network policy rules
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: weaviate-netpol
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
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # For OpenAI API
EOF
```

#### 3. Performance Issues

**Symptoms:**
- High query latency (>2 seconds)
- Memory usage constantly high (>90%)
- Frequent pod restarts due to OOM

**Diagnosis:**
```bash
# Check resource usage over time
kubectl top pods -n nephoran-system -l app=weaviate --sort-by=memory

# Check query performance metrics
curl -s http://localhost:8080/v1/meta | jq '.moduleConfig'

# Check for slow queries in logs
kubectl logs -n nephoran-system -l app=weaviate | grep -i "slow\|timeout\|error"
```

**Solutions:**

**Memory Optimization:**
```bash
# Increase memory limits
kubectl patch deployment weaviate -n nephoran-system \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"weaviate","resources":{"limits":{"memory":"32Gi"}}}]}}}}'

# Tune vector cache
kubectl set env deployment/weaviate -n nephoran-system \
  VECTOR_CACHE_MAX_OBJECTS=2000000
```

**Query Optimization:**
```bash
# Update vector index configuration
curl -X PATCH http://localhost:8080/v1/schema/TelecomKnowledge \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer nephoran-rag-key-production" \
  -d '{
    "vectorIndexConfig": {
      "ef": 128,
      "efConstruction": 256,
      "maxConnections": 32
    }
  }'
```

#### 4. Data Consistency Issues

**Symptoms:**
- Missing objects after restart
- Inconsistent search results
- Schema validation errors

**Diagnosis:**
```bash
# Check object counts
curl -s http://localhost:8080/v1/objects \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq '.objects | length'

# Validate schema
curl -s http://localhost:8080/v1/schema \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq '.classes[] | {class: .class, vectorizer: .vectorizer}'

# Check for data corruption
kubectl exec -it deployment/weaviate -n nephoran-system -- \
  find /var/lib/weaviate -name "*.db" -exec file {} \;
```

**Solutions:**

**Schema Recreation:**
```bash
# Backup current data first
kubectl exec -it deployment/weaviate -n nephoran-system -- \
  tar -czf /tmp/weaviate-backup.tar.gz /var/lib/weaviate

# Recreate schema
python3 telecom-schema.py --recreate

# Re-populate data if needed
curl -X POST http://rag-api:5001/knowledge/populate \
  -H "Content-Type: application/json" \
  -d '{"path": "/app/knowledge_base", "force_recreate": true}'
```

## Maintenance Procedures

### Weekly Maintenance (30 minutes)

**Performance Review:**
```bash
#!/bin/bash
# weekly-maintenance.sh

echo "=== Weekly Weaviate Maintenance $(date) ==="

# Resource usage analysis
echo "Resource Usage Trends:"
kubectl top pods -n nephoran-system -l app=weaviate --sort-by=memory
kubectl top pods -n nephoran-system -l app=weaviate --sort-by=cpu

# Storage analysis
echo "Storage Usage:"
kubectl exec -it deployment/weaviate -n nephoran-system -- df -h /var/lib/weaviate

# Query performance metrics
echo "Query Performance:"
curl -s http://localhost:8080/v1/meta \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq '.moduleConfig'

# Knowledge base statistics
echo "Knowledge Base Stats:"
curl -s http://localhost:8080/v1/objects \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq '.objects | group_by(.class) | map({class: .[0].class, count: length})'

# Check for any warnings in logs
echo "Recent Warnings:"
kubectl logs -n nephoran-system -l app=weaviate --since=168h | grep -i warn | tail -20

echo "=== Maintenance Complete ==="
```

**Cleanup Tasks:**
```bash
# Clean up old backup files (keep last 30 days)
kubectl create job cleanup-old-backups --image=busybox \
  --command -- sh -c "find /var/lib/weaviate/backups -type f -mtime +30 -delete"

# Optimize vector indices
curl -X POST http://localhost:8080/v1/schema/TelecomKnowledge/shards/_optimize \
  -H "Authorization: Bearer nephoran-rag-key-production"
```

### Monthly Maintenance (2 hours)

**Version Updates:**
```bash
# Check for new Weaviate version
echo "Current version:"
kubectl get deployment weaviate -n nephoran-system -o jsonpath='{.spec.template.spec.containers[0].image}'

# Update to new version (example)
kubectl set image deployment/weaviate weaviate=semitechnologies/weaviate:1.29.0 -n nephoran-system

# Monitor rollout
kubectl rollout status deployment/weaviate -n nephoran-system
```

**Security Updates:**
```bash
# Rotate API keys
NEW_API_KEY="nephoran-rag-key-$(date +%Y%m%d)"
kubectl patch secret weaviate-api-key -n nephoran-system \
  -p "{\"data\":{\"api-key\":\"$(echo -n $NEW_API_KEY | base64)\"}}"

# Update applications to use new key
kubectl rollout restart deployment/rag-api -n nephoran-system
```

**Performance Tuning:**
```bash
# Analyze query patterns and optimize
curl -s http://localhost:8080/v1/meta \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq '.usage'

# Adjust HPA based on usage patterns
kubectl patch hpa weaviate-hpa -n nephoran-system \
  -p '{"spec":{"metrics":[{"type":"Resource","resource":{"name":"cpu","target":{"type":"Utilization","averageUtilization":65}}}]}}'
```

## Backup and Recovery

### Automated Backup System

**Backup Configuration:**
```yaml
# backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: weaviate-backup
  namespace: nephoran-system
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM UTC
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: curlimages/curl:latest
            env:
            - name: WEAVIATE_URL
              value: "http://weaviate:8080"
            - name: BACKUP_ID
              value: "nephoran-backup-$(date +%Y%m%d-%H%M%S)"
            command:
            - /bin/sh
            - -c
            - |
              echo "Starting backup: $BACKUP_ID"
              
              # Create backup
              curl -X POST "$WEAVIATE_URL/v1/backups/filesystem" \
                -H "Authorization: Bearer nephoran-rag-key-production" \
                -H "Content-Type: application/json" \
                -d "{\"id\":\"$BACKUP_ID\",\"include\":[\"TelecomKnowledge\"]}"
              
              # Wait for completion
              while true; do
                STATUS=$(curl -s "$WEAVIATE_URL/v1/backups/filesystem/$BACKUP_ID" \
                  -H "Authorization: Bearer nephoran-rag-key-production" | \
                  jq -r '.status')
                if [ "$STATUS" = "SUCCESS" ]; then
                  echo "Backup completed successfully"
                  break
                elif [ "$STATUS" = "FAILED" ]; then
                  echo "Backup failed"
                  exit 1
                fi
                echo "Backup in progress... Status: $STATUS"
                sleep 30
              done
          restartPolicy: OnFailure
```

**Manual Backup:**
```bash
# Trigger immediate backup
BACKUP_ID="manual-backup-$(date +%Y%m%d-%H%M%S)"

curl -X POST http://localhost:8080/v1/backups/filesystem \
  -H "Authorization: Bearer nephoran-rag-key-production" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"$BACKUP_ID\",\"include\":[\"TelecomKnowledge\"]}"

# Check backup status
curl -s http://localhost:8080/v1/backups/filesystem/$BACKUP_ID \
  -H "Authorization: Bearer nephoran-rag-key-production" | jq '.status'
```

### Recovery Procedures

**Point-in-Time Recovery:**
```bash
# List available backups
curl -s http://localhost:8080/v1/backups/filesystem \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq '.backups[] | {id: .id, status: .status, created: .created}'

# Restore from specific backup
BACKUP_ID="nephoran-backup-20241128-020000"

curl -X POST http://localhost:8080/v1/backups/filesystem/$BACKUP_ID/restore \
  -H "Authorization: Bearer nephoran-rag-key-production" \
  -H "Content-Type: application/json" \
  -d '{"include":["TelecomKnowledge"]}'
```

**Complete Cluster Recovery:**
```bash
#!/bin/bash
# cluster-recovery.sh

echo "Starting cluster recovery procedure..."

# 1. Deploy fresh cluster
kubectl delete deployment weaviate -n nephoran-system
kubectl apply -f weaviate-deployment.yaml

# 2. Wait for pods to be ready
kubectl wait --for=condition=available deployment/weaviate \
  --namespace=nephoran-system --timeout=600s

# 3. Restore schema
python3 telecom-schema.py

# 4. Restore data from latest backup
LATEST_BACKUP=$(curl -s http://localhost:8080/v1/backups/filesystem \
  -H "Authorization: Bearer nephoran-rag-key-production" | \
  jq -r '.backups | sort_by(.created) | .[-1].id')

echo "Restoring from backup: $LATEST_BACKUP"

curl -X POST http://localhost:8080/v1/backups/filesystem/$LATEST_BACKUP/restore \
  -H "Authorization: Bearer nephoran-rag-key-production" \
  -H "Content-Type: application/json" \
  -d '{"include":["TelecomKnowledge"]}'

echo "Recovery complete"
```

## Disaster Recovery

### RTO/RPO Targets

- **Recovery Time Objective (RTO)**: 2 hours
- **Recovery Point Objective (RPO)**: 24 hours (daily backups)
- **Availability Target**: 99.9% uptime

### Multi-Region Setup

**Primary-Secondary Configuration:**
```yaml
# secondary-region-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: weaviate-secondary
  namespace: nephoran-system-dr
spec:
  replicas: 2
  selector:
    matchLabels:
      app: weaviate-secondary
  template:
    metadata:
      labels:
        app: weaviate-secondary
        role: disaster-recovery
    spec:
      containers:
      - name: weaviate
        image: semitechnologies/weaviate:1.28.1
        env:
        - name: CLUSTER_HOSTNAME
          value: "weaviate-dr"
        - name: PERSISTENCE_DATA_PATH
          value: "/var/lib/weaviate"
        # ... other configuration
```

**Cross-Region Backup Sync:**
```bash
#!/bin/bash
# sync-backups-cross-region.sh

# Sync backups to secondary region
aws s3 sync /var/lib/weaviate/backups \
  s3://nephoran-backups-secondary/weaviate/ \
  --delete \
  --region us-west-2

# Verify sync
aws s3 ls s3://nephoran-backups-secondary/weaviate/ --recursive
```

### Failover Procedures

**Automated Failover (via External DNS/Load Balancer):**
```yaml
# external-dns-failover.yaml
apiVersion: v1
kind: Service
metadata:
  name: weaviate-global
  annotations:
    external-dns.alpha.kubernetes.io/hostname: weaviate.nephoran.com
    external-dns.alpha.kubernetes.io/target: weaviate-primary.us-east-1.nephoran.com
spec:
  type: ExternalName
  externalName: weaviate.nephoran-system.svc.cluster.local
```

**Manual Failover:**
```bash
#!/bin/bash
# manual-failover.sh

echo "Initiating failover to secondary region..."

# 1. Update DNS to point to secondary
kubectl patch service weaviate-global \
  -p '{"spec":{"externalName":"weaviate-secondary.us-west-2.nephoran.com"}}'

# 2. Scale up secondary cluster
kubectl scale deployment weaviate-secondary --replicas=3 -n nephoran-system-dr

# 3. Update application configuration
kubectl set env deployment/rag-api \
  WEAVIATE_URL="http://weaviate-secondary.nephoran-system-dr.svc.cluster.local:8080"

# 4. Verify secondary is operational
curl http://weaviate-secondary:8080/v1/.well-known/ready

echo "Failover complete"
```

## Performance Tuning

### Vector Index Optimization

**HNSW Index Parameters:**
```bash
# Optimize for high recall and accuracy
curl -X PATCH http://localhost:8080/v1/schema/TelecomKnowledge \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer nephoran-rag-key-production" \
  -d '{
    "vectorIndexConfig": {
      "ef": 256,              # Higher for better recall
      "efConstruction": 512,  # Higher for better index quality
      "maxConnections": 64,   # Higher for better connectivity
      "vectorCacheMaxObjects": 2000000,
      "dynamicEfMin": 128,
      "dynamicEfMax": 512
    }
  }'
```

**Memory Optimization:**
```bash
# Tune JVM settings for large datasets
kubectl patch deployment weaviate -n nephoran-system \
  -p '{
    "spec": {
      "template": {
        "spec": {
          "containers": [{
            "name": "weaviate",
            "env": [
              {"name": "GOGC", "value": "100"},
              {"name": "GOMEMLIMIT", "value": "12GiB"}
            ]
          }]
        }
      }
    }
  }'
```

### Query Performance Tuning

**Hybrid Search Optimization:**
```python
# Optimal hybrid search parameters for telecom domain
result = client.query.get("TelecomKnowledge", ["content", "title"]) \
    .with_hybrid(
        query="5G AMF registration procedure",
        alpha=0.7,  # Favor vector search for technical queries
        properties=["content^2", "keywords^1.5"]  # Weight important fields
    ) \
    .with_additional(["score", "explainScore"]) \
    .with_limit(10) \
    .do()
```

**Batch Operations:**
```python
# Optimize bulk ingestion
with client.batch as batch:
    batch.batch_size = 100
    batch.dynamic = True
    batch.timeout_retries = 3
    batch.callback = weaviate.util.check_batch_result
    
    for doc in documents:
        batch.add_data_object(doc.properties, "TelecomKnowledge")
```

## Security Operations

### Access Control Management

**API Key Rotation:**
```bash
#!/bin/bash
# rotate-api-keys.sh

# Generate new API key
NEW_KEY="nephoran-rag-$(openssl rand -hex 16)"

# Update secret
kubectl patch secret weaviate-api-key -n nephoran-system \
  --type='json' \
  -p="[{\"op\":\"replace\",\"path\":\"/data/api-key\",\"value\":\"$(echo -n $NEW_KEY | base64)\"}]"

# Rolling restart of dependent services
kubectl rollout restart deployment/rag-api -n nephoran-system
kubectl rollout restart deployment/llm-processor -n nephoran-system

# Verify new key works
sleep 30
curl -H "Authorization: Bearer $NEW_KEY" \
  http://localhost:8080/v1/.well-known/ready

echo "API key rotation complete"
```

**Network Security Audit:**
```bash
# Verify network policies are enforced
kubectl get networkpolicy -n nephoran-system -o yaml

# Test unauthorized access (should be blocked)
kubectl run test-unauthorized --image=curlimages/curl --rm -i \
  --namespace=default \
  -- curl -m 5 http://weaviate.nephoran-system:8080/v1/schema

# Verify TLS configuration
openssl s_client -connect weaviate.nephoran.com:443 -servername weaviate.nephoran.com
```

### Compliance and Auditing

**Access Logging:**
```bash
# Enable audit logging
kubectl patch deployment weaviate -n nephoran-system \
  -p '{
    "spec": {
      "template": {
        "spec": {
          "containers": [{
            "name": "weaviate",
            "env": [
              {"name": "LOG_LEVEL", "value": "debug"},
              {"name": "ENABLE_AUDIT_LOG", "value": "true"}
            ]
          }]
        }
      }
    }
  }'

# Collect audit logs
kubectl logs -n nephoran-system -l app=weaviate | grep -i audit > weaviate-audit.log
```

## Incident Response

### Severity Levels

**P0 - Critical (Response: Immediate, Resolution: 2 hours)**
- Complete service outage
- Data corruption or loss
- Security breach

**P1 - High (Response: 1 hour, Resolution: 8 hours)**
- Partial service degradation
- Performance issues affecting >50% of queries
- Backup failures

**P2 - Medium (Response: 4 hours, Resolution: 24 hours)**
- Minor performance issues
- Single pod failures with redundancy
- Configuration issues

**P3 - Low (Response: 1 business day, Resolution: 5 business days)**
- Enhancement requests
- Documentation updates
- Non-critical maintenance

### Incident Response Playbook

**P0 Incident Response:**
```bash
#!/bin/bash
# p0-incident-response.sh

echo "P0 INCIDENT RESPONSE ACTIVATED"
echo "Timestamp: $(date)"

# 1. Immediate triage
kubectl get pods -n nephoran-system -l app=weaviate
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | tail -20

# 2. Service status check
curl -w "HTTP %{http_code} - Response time: %{time_total}s\n" \
  http://localhost:8080/v1/.well-known/ready

# 3. Collect diagnostic information
kubectl describe deployment weaviate -n nephoran-system > incident-deploy-describe.txt
kubectl logs -n nephoran-system -l app=weaviate --since=1h > incident-logs.txt
kubectl top pods -n nephoran-system > incident-resources.txt

# 4. Attempt automated recovery
kubectl rollout restart deployment/weaviate -n nephoran-system

# 5. Notify stakeholders
echo "Incident detected in Weaviate cluster. Investigation in progress." | \
  mail -s "P0 Incident - Weaviate Cluster" ops-team@nephoran.com

echo "Initial response complete. Continue with detailed investigation."
```

### Post-Incident Review

**Template for Post-Incident Report:**
```markdown
# Incident Report - Weaviate Cluster Outage

## Incident Summary
- **Date**: 2024-11-28
- **Duration**: 45 minutes
- **Severity**: P0
- **Impact**: Complete RAG system unavailable

## Timeline
- **14:30 UTC**: First alert received
- **14:32 UTC**: On-call engineer engaged
- **14:35 UTC**: Root cause identified (memory exhaustion)
- **14:40 UTC**: Recovery initiated
- **15:15 UTC**: Service fully restored

## Root Cause
Memory leak in vector index causing OOM kills.

## Resolution
- Restarted affected pods
- Applied memory limit increase
- Implemented memory monitoring alert

## Action Items
1. [ ] Implement proactive memory monitoring
2. [ ] Update runbook with memory pressure handling
3. [ ] Schedule performance review meeting
```

---

This operations runbook provides comprehensive procedures for managing the Weaviate vector database cluster in production. Regular updates should be made based on operational experience and evolving requirements.

For additional support, refer to:
- [Weaviate Official Documentation](https://docs.weaviate.io/)
- [Kubernetes Operations Guide](https://kubernetes.io/docs/concepts/cluster-administration/)
- [Nephoran Intent Operator CLAUDE.md](./CLAUDE.md)