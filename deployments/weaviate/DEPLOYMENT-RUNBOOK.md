# Weaviate Vector Database Deployment Runbook
## Nephoran Intent Operator Production Deployment Guide

### Overview

This comprehensive runbook provides step-by-step procedures for deploying, validating, and maintaining the Weaviate vector database cluster for the Nephoran Intent Operator. This document covers production-grade deployment with high availability, security, monitoring, and disaster recovery.

### Prerequisites and Pre-Deployment Validation

#### System Requirements Verification

**Minimum Cluster Requirements:**
```bash
# Verify cluster resources
kubectl top nodes
kubectl describe nodes | grep -A 5 "Capacity:\|Allocatable:"

# Required minimum:
# - 3+ worker nodes
# - 16+ GB RAM per node
# - 100+ GB available storage per node
# - Kubernetes v1.25+
```

**Storage Class Validation:**
```bash
# Check available storage classes
kubectl get storageclass

# Verify required storage classes exist
kubectl get storageclass gp3-encrypted 2>/dev/null || echo "ERROR: gp3-encrypted storage class not found"
kubectl get storageclass fast-ssd 2>/dev/null || echo "WARNING: fast-ssd storage class not found"

# Test storage provisioning
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: gp3-encrypted
EOF

# Verify PVC is bound
kubectl get pvc test-pvc
kubectl delete pvc test-pvc
```

**Network Policy Support:**
```bash
# Test network policy support
kubectl get networkpolicies -A
kubectl api-resources | grep networkpolicies
```

**Required CLI Tools:**
```bash
# Verify tool versions
echo "=== Tool Version Check ==="
kubectl version --client --short
helm version --short
jq --version
curl --version
yq --version

# Required versions:
# kubectl: v1.25+
# helm: v3.8+
# jq: 1.6+
# curl: 7.68+
```

#### Environment Variable Setup

```bash
# Set deployment environment variables
export ENVIRONMENT="production"  # or staging/development
export NAMESPACE="nephoran-system"
export OPENAI_API_KEY="your-openai-api-key"
export WEAVIATE_API_KEY="nephoran-rag-key-production-$(date +%s)"
export DEPLOYMENT_DATE=$(date +%Y%m%d-%H%M%S)
export BACKUP_RETENTION_DAYS="30"

# Validate required variables
for var in ENVIRONMENT NAMESPACE OPENAI_API_KEY WEAVIATE_API_KEY; do
  if [ -z "${!var}" ]; then
    echo "ERROR: $var is not set"
    exit 1
  fi
done

echo "✅ Environment variables validated"
```

### Step 1: Pre-Deployment Setup

#### Create Namespace and Labels

```bash
# Create the namespace with proper labels
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | \
kubectl apply -f -

kubectl label namespace $NAMESPACE \
  name=$NAMESPACE \
  environment=$ENVIRONMENT \
  component=weaviate \
  managed-by=nephoran-operator

# Verify namespace
kubectl get namespace $NAMESPACE -o yaml
```

#### Create Required Secrets

```bash
# Create OpenAI API key secret
kubectl create secret generic openai-api-key \
  --from-literal=api-key="$OPENAI_API_KEY" \
  --namespace=$NAMESPACE \
  --dry-run=client -o yaml | \
kubectl apply -f -

# Create Weaviate API key secret
kubectl create secret generic weaviate-api-key \
  --from-literal=api-key="$WEAVIATE_API_KEY" \
  --namespace=$NAMESPACE \
  --dry-run=client -o yaml | \
kubectl apply -f -

# Create TLS certificates (if using custom certs)
# kubectl create secret tls weaviate-tls \
#   --cert=path/to/tls.crt \
#   --key=path/to/tls.key \
#   --namespace=$NAMESPACE

# Verify secrets
kubectl get secrets -n $NAMESPACE
```

#### Deploy Storage Classes (if needed)

```bash
# Deploy optimized storage classes for different environments
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3-encrypted
  annotations:
    storageclass.kubernetes.io/is-default-class: "false"
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
  encrypted: "true"
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Retain
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3-backup
  annotations:
    storageclass.kubernetes.io/is-default-class: "false"
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "1000"
  throughput: "125"
  encrypted: "true"
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Retain
EOF
```

### Step 2: Core Weaviate Deployment

#### Deploy RBAC Configuration

```bash
# Apply RBAC policies
kubectl apply -f rbac.yaml -n $NAMESPACE

# Verify RBAC
kubectl get serviceaccounts,roles,rolebindings -n $NAMESPACE
```

#### Deploy Network Policies

```bash
# Apply network security policies
kubectl apply -f network-policy.yaml -n $NAMESPACE

# Verify network policies
kubectl get networkpolicies -n $NAMESPACE
kubectl describe networkpolicy weaviate-network-policy -n $NAMESPACE
```

#### Deploy Weaviate Cluster

```bash
# Deploy the main Weaviate cluster
echo "Deploying Weaviate cluster..."
kubectl apply -f weaviate-deployment.yaml -n $NAMESPACE

# Monitor deployment progress
echo "Monitoring deployment progress..."
kubectl rollout status deployment/weaviate -n $NAMESPACE --timeout=600s

# Check pod status
kubectl get pods -n $NAMESPACE -l app=weaviate
```

#### Validate Core Deployment

```bash
# Wait for all pods to be ready
echo "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=weaviate -n $NAMESPACE --timeout=300s

# Check pod distribution across nodes
echo "Pod distribution:"
kubectl get pods -n $NAMESPACE -l app=weaviate -o wide

# Verify service endpoints
kubectl get endpoints weaviate -n $NAMESPACE
kubectl describe service weaviate -n $NAMESPACE
```

### Step 3: Backup System Deployment

#### Deploy Backup Infrastructure

```bash
# Deploy backup system
kubectl apply -f backup-system.yaml -n $NAMESPACE

# Verify backup components
kubectl get cronjobs -n $NAMESPACE -l component=backup
kubectl get configmaps -n $NAMESPACE -l component=backup
kubectl get secrets -n $NAMESPACE -l component=backup
```

#### Test Backup System

```bash
# Trigger initial backup job
kubectl create job weaviate-backup-initial \
  --from=cronjob/weaviate-daily-backup \
  --namespace=$NAMESPACE

# Monitor backup job
kubectl wait --for=condition=complete job/weaviate-backup-initial -n $NAMESPACE --timeout=600s
kubectl logs job/weaviate-backup-initial -n $NAMESPACE
```

### Step 4: Monitoring and Observability

#### Deploy Monitoring Stack

```bash
# Deploy enhanced monitoring
kubectl apply -f monitoring-enhanced.yaml -n $NAMESPACE

# Verify monitoring components
kubectl get servicemonitors,prometheusrules -n $NAMESPACE
kubectl get cronjobs -n $NAMESPACE -l component=monitoring
```

#### Validate Metrics Endpoint

```bash
# Port forward to check metrics
kubectl port-forward svc/weaviate 8080:8080 -n $NAMESPACE &
METRICS_PID=$!

sleep 5

# Test metrics endpoint
curl -s "http://localhost:8080/v1/meta" | jq '.'
curl -s "http://localhost:8080/metrics" | head -20

# Cleanup port forward
kill $METRICS_PID
```

### Step 5: Schema Initialization

#### Deploy Schema Configuration

```bash
# Create schema initialization job
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: weaviate-schema-init-$DEPLOYMENT_DATE
  namespace: $NAMESPACE
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
        - name: WEAVIATE_API_KEY
          valueFrom:
            secretKeyRef:
              name: weaviate-api-key
              key: api-key
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: openai-api-key
              key: api-key
        volumeMounts:
        - name: schema-script
          mountPath: /scripts
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "200m"
      volumes:
      - name: schema-script
        configMap:
          name: telecom-schema-script
      restartPolicy: Never
  backoffLimit: 3
EOF

# Wait for schema initialization
kubectl wait --for=condition=complete job/weaviate-schema-init-$DEPLOYMENT_DATE -n $NAMESPACE --timeout=300s

# Check schema initialization logs
kubectl logs job/weaviate-schema-init-$DEPLOYMENT_DATE -n $NAMESPACE
```

### Step 6: Integration Testing and Validation

#### Comprehensive System Validation

```bash
# Run the automated deployment validation script
chmod +x deploy-and-validate.sh
./deploy-and-validate.sh

# Additional manual validation
echo "=== Manual Validation Steps ==="

# 1. API Health Check
kubectl port-forward svc/weaviate 8080:8080 -n $NAMESPACE &
HEALTH_PID=$!
sleep 5

echo "Testing API endpoints..."
curl -f "http://localhost:8080/v1/.well-known/ready" || echo "❌ Readiness check failed"
curl -f "http://localhost:8080/v1/.well-known/live" || echo "❌ Liveness check failed"
curl -s "http://localhost:8080/v1/schema" | jq '.classes | length' || echo "❌ Schema check failed"

kill $HEALTH_PID

# 2. Backup Validation
echo "Validating backup system..."
kubectl get cronjobs -n $NAMESPACE -l component=backup
kubectl describe cronjob weaviate-daily-backup -n $NAMESPACE

# 3. Monitoring Validation
echo "Validating monitoring..."
kubectl get servicemonitor weaviate-monitor -n $NAMESPACE
kubectl get prometheusrule weaviate-alerts -n $NAMESPACE
```

#### Performance Baseline Testing

```bash
# Create performance test job
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: weaviate-perf-test-$DEPLOYMENT_DATE
  namespace: $NAMESPACE
spec:
  template:
    spec:
      containers:
      - name: perf-test
        image: curlimages/curl:latest
        command:
        - /bin/sh
        - -c
        - |
          echo "Starting performance baseline test..."
          WEAVIATE_URL="http://weaviate:8080"
          
          # Test API response time
          for i in {1..10}; do
            START=\$(date +%s%N)
            curl -s "\$WEAVIATE_URL/v1/meta" > /dev/null
            END=\$(date +%s%N)
            DURATION=\$((\$END - \$START))
            DURATION_MS=\$((\$DURATION / 1000000))
            echo "Request \$i: \${DURATION_MS}ms"
          done
          
          echo "Performance test completed"
      restartPolicy: Never
  backoffLimit: 1
EOF

# Wait for performance test
kubectl wait --for=condition=complete job/weaviate-perf-test-$DEPLOYMENT_DATE -n $NAMESPACE --timeout=120s
kubectl logs job/weaviate-perf-test-$DEPLOYMENT_DATE -n $NAMESPACE
```

### Step 7: Post-Deployment Configuration

#### Configure Auto-Scaling

```bash
# Deploy Horizontal Pod Autoscaler
cat <<EOF | kubectl apply -f -
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: weaviate-hpa
  namespace: $NAMESPACE
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: weaviate
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 300
      policies:
      - type: Pods
        value: 2
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 600
      policies:
      - type: Pods
        value: 1
        periodSeconds: 60
EOF

# Verify HPA
kubectl get hpa weaviate-hpa -n $NAMESPACE
```

#### Configure Pod Disruption Budget

```bash
# Deploy PDB for high availability
cat <<EOF | kubectl apply -f -
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: weaviate-pdb
  namespace: $NAMESPACE
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: weaviate
EOF

# Verify PDB
kubectl get pdb weaviate-pdb -n $NAMESPACE
```

### Step 8: Knowledge Base Population

#### Automated Knowledge Base Initialization

```bash
# Run knowledge base population
if [ -f "../../populate-knowledge-base.ps1" ]; then
  echo "Running knowledge base population..."
  pwsh ../../populate-knowledge-base.ps1
else
  echo "Manual knowledge base population required"
  echo "Please run: kubectl port-forward svc/weaviate 8080:8080 -n $NAMESPACE"
  echo "Then execute the knowledge base population script"
fi
```

### Step 9: Final Validation and Sign-off

#### Comprehensive Health Check

```bash
echo "=== Final Deployment Validation ==="

# 1. Pod Health
echo "1. Pod Health Status:"
kubectl get pods -n $NAMESPACE -l app=weaviate -o wide

# 2. Service Status
echo "2. Service Status:"
kubectl get services -n $NAMESPACE

# 3. Storage Status  
echo "3. Storage Status:"
kubectl get pvc -n $NAMESPACE

# 4. Backup System
echo "4. Backup System:"
kubectl get cronjobs -n $NAMESPACE -l component=backup

# 5. Monitoring
echo "5. Monitoring Stack:"
kubectl get servicemonitors,prometheusrules -n $NAMESPACE

# 6. Auto-scaling
echo "6. Auto-scaling Configuration:"
kubectl get hpa -n $NAMESPACE

# 7. Network Policies
echo "7. Network Security:"
kubectl get networkpolicies -n $NAMESPACE

# 8. Overall System Status
echo "8. Overall System Status:"
kubectl top pods -n $NAMESPACE
kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | tail -10
```

#### Generate Deployment Report

```bash
# Create deployment summary report
cat > deployment-report-$DEPLOYMENT_DATE.md <<EOF
# Weaviate Deployment Report

**Date:** $(date)
**Environment:** $ENVIRONMENT  
**Namespace:** $NAMESPACE
**Deployment ID:** $DEPLOYMENT_DATE

## Deployment Summary

### Components Deployed
- ✅ Weaviate Cluster ($(kubectl get pods -n $NAMESPACE -l app=weaviate --no-headers | wc -l) replicas)
- ✅ Backup System ($(kubectl get cronjobs -n $NAMESPACE -l component=backup --no-headers | wc -l) jobs)
- ✅ Monitoring Stack ($(kubectl get servicemonitors -n $NAMESPACE --no-headers | wc -l) monitors)
- ✅ Network Policies ($(kubectl get networkpolicies -n $NAMESPACE --no-headers | wc -l) policies)
- ✅ Auto-scaling (HPA configured)

### Resource Allocation
\`\`\`
$(kubectl top pods -n $NAMESPACE 2>/dev/null || echo "Resource metrics not available")
\`\`\`

### Storage Configuration
\`\`\`
$(kubectl get pvc -n $NAMESPACE)
\`\`\`

### Health Status
\`\`\`
$(kubectl get pods -n $NAMESPACE -l app=weaviate)
\`\`\`

## Next Steps
1. Configure monitoring dashboards
2. Set up alerting notifications  
3. Schedule backup validation
4. Begin knowledge base population
5. Integration with Nephoran Intent Operator

## Support Information
- Deployment logs: kubectl logs -n $NAMESPACE -l app=weaviate
- Monitoring: kubectl port-forward svc/weaviate 8080:8080 -n $NAMESPACE
- Backup status: kubectl get cronjobs -n $NAMESPACE -l component=backup

EOF

echo "✅ Deployment report generated: deployment-report-$DEPLOYMENT_DATE.md"
```

### Troubleshooting Common Issues

#### Pod Startup Issues

```bash
# Check pod events
kubectl describe pods -n $NAMESPACE -l app=weaviate

# Check resource constraints
kubectl top nodes
kubectl describe nodes | grep -A 5 "Non-terminated Pods"

# Check storage issues
kubectl get pvc -n $NAMESPACE
kubectl describe pvc -n $NAMESPACE
```

#### Network Connectivity Issues

```bash
# Test DNS resolution
kubectl run test-dns --image=busybox --rm -it --restart=Never -- nslookup weaviate.$NAMESPACE.svc.cluster.local

# Test service connectivity
kubectl run test-connect --image=curlimages/curl --rm -it --restart=Never -- curl -v http://weaviate.$NAMESPACE.svc.cluster.local:8080/v1/.well-known/ready

# Check network policies
kubectl describe networkpolicy -n $NAMESPACE
```

#### Performance Issues

```bash
# Check resource utilization
kubectl top pods -n $NAMESPACE
kubectl describe hpa -n $NAMESPACE

# Check storage performance
kubectl describe pvc -n $NAMESPACE | grep -A 10 "Conditions"

# Check for resource limits
kubectl describe pods -n $NAMESPACE -l app=weaviate | grep -A 5 "Limits\|Requests"
```

### Rollback Procedures

#### Emergency Rollback

```bash
# If deployment fails, rollback using:
kubectl rollout undo deployment/weaviate -n $NAMESPACE

# Check rollback status
kubectl rollout status deployment/weaviate -n $NAMESPACE

# If needed, restore from backup
kubectl create job weaviate-restore-emergency \
  --from=cronjob/weaviate-daily-backup \
  --namespace=$NAMESPACE
```

### Maintenance Procedures

#### Regular Maintenance Tasks

```bash
# Weekly maintenance script
cat > weekly-maintenance.sh <<'EOF'
#!/bin/bash
NAMESPACE="nephoran-system"

echo "=== Weekly Weaviate Maintenance ==="

# 1. Check cluster health
kubectl get pods -n $NAMESPACE -l app=weaviate

# 2. Review resource usage
kubectl top pods -n $NAMESPACE

# 3. Check backup status
kubectl get jobs -n $NAMESPACE -l component=backup | head -10

# 4. Verify monitoring
kubectl get servicemonitors -n $NAMESPACE

# 5. Check for updates
kubectl get deployment weaviate -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].image}'

echo "Maintenance check completed"
EOF

chmod +x weekly-maintenance.sh
```

### Security Hardening Checklist

- [ ] Network policies applied and tested
- [ ] Secrets properly configured with rotation
- [ ] RBAC policies implement least privilege
- [ ] TLS encryption enabled for all communications
- [ ] Pod security contexts configured
- [ ] Audit logging enabled
- [ ] Vulnerability scanning configured
- [ ] Backup encryption enabled

### Sign-off and Handover

#### Deployment Checklist

- [ ] All pods running and healthy
- [ ] Services accessible and responding
- [ ] Backup system operational
- [ ] Monitoring stack deployed
- [ ] Network policies active
- [ ] Auto-scaling configured
- [ ] Performance baseline established
- [ ] Documentation updated
- [ ] Team training completed
- [ ] Runbook validated

**Deployment Completed By:** `________________`  
**Date:** `________________`  
**Reviewed By:** `________________`  
**Approved for Production:** `________________`

---

This deployment runbook ensures a consistent, reliable, and secure deployment of the Weaviate vector database cluster for the Nephoran Intent Operator. Regular updates to this document should be made as the deployment procedures evolve.