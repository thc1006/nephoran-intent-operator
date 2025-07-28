# Weaviate Troubleshooting Guide
## Comprehensive Problem Resolution for Nephoran Intent Operator

### Overview

This troubleshooting guide provides systematic approaches to diagnosing and resolving common issues with the Weaviate vector database deployment in the Nephoran Intent Operator system. The guide is organized by problem category with step-by-step resolution procedures.

### Quick Diagnostic Commands

#### System Health Check Script

```bash
#!/bin/bash
# weaviate-health-check.sh - Quick system diagnostic

NAMESPACE="nephoran-system"
echo "=== Weaviate Health Check ==="
echo "Timestamp: $(date)"

# 1. Pod Status
echo -e "\n1. Pod Status:"
kubectl get pods -n $NAMESPACE -l app=weaviate -o wide

# 2. Service Status
echo -e "\n2. Service Status:"
kubectl get svc -n $NAMESPACE -l app=weaviate

# 3. Endpoints
echo -e "\n3. Service Endpoints:"
kubectl get endpoints -n $NAMESPACE -l app=weaviate

# 4. Resource Usage
echo -e "\n4. Resource Usage:"
kubectl top pods -n $NAMESPACE -l app=weaviate 2>/dev/null || echo "Metrics not available"

# 5. Recent Events
echo -e "\n5. Recent Events:"
kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | tail -5

# 6. Storage Status
echo -e "\n6. Storage Status:"
kubectl get pvc -n $NAMESPACE

# 7. API Health
echo -e "\n7. API Health Test:"
kubectl run weaviate-test --image=curlimages/curl --rm -i --restart=Never \
  -- curl -s -f "http://weaviate.$NAMESPACE.svc.cluster.local:8080/v1/.well-known/ready" \
  && echo "✅ API is healthy" || echo "❌ API health check failed"
```

### Common Issues and Solutions

## 1. Pod Startup and Initialization Issues

### Issue: Pods Stuck in Pending State

**Symptoms:**
- Pods remain in `Pending` status
- Events show scheduling issues
- No containers have started

**Diagnostic Commands:**
```bash
# Check pod status and events
kubectl describe pods -n nephoran-system -l app=weaviate

# Check node resources
kubectl describe nodes | grep -A 5 "Allocatable"

# Check resource requests vs available
kubectl top nodes
```

**Common Causes and Solutions:**

#### Insufficient Node Resources

**Diagnosis:**
```bash
# Check if nodes have sufficient resources
kubectl describe nodes | grep -A 10 "Non-terminated Pods"
kubectl get nodes -o yaml | grep -A 5 "allocatable"
```

**Resolution:**
```bash
# Scale down resource requests temporarily
kubectl patch deployment weaviate -n nephoran-system -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "weaviate",
          "resources": {
            "requests": {
              "memory": "2Gi",
              "cpu": "500m"
            }
          }
        }]
      }
    }
  }
}'

# Or add more nodes to cluster
# kubectl scale nodes --replicas=5  # Cloud provider specific
```

#### Storage Class Issues

**Diagnosis:**
```bash
# Check storage class availability
kubectl get storageclass
kubectl describe storageclass gp3-encrypted
```

**Resolution:**
```bash
# Create missing storage class
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3-encrypted
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  encrypted: "true"
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
EOF
```

#### Pod Security Policy Issues

**Diagnosis:**
```bash
# Check for PSP violations
kubectl get events -n nephoran-system | grep -i "security\|policy"
kubectl describe podsecuritypolicy
```

**Resolution:**
```bash
# Update security context in deployment
kubectl patch deployment weaviate -n nephoran-system -p '
{
  "spec": {
    "template": {
      "spec": {
        "securityContext": {
          "runAsNonRoot": true,
          "runAsUser": 1000,
          "fsGroup": 1000
        }
      }
    }
  }
}'
```

### Issue: Pods Stuck in Init Container State

**Symptoms:**
- Pods show `Init:0/1` status
- Init containers fail to complete
- Application container never starts

**Diagnostic Commands:**
```bash
# Check init container logs
kubectl logs -n nephoran-system -l app=weaviate -c init-container

# Check init container status
kubectl describe pods -n nephoran-system -l app=weaviate | grep -A 10 "Init Containers"
```

**Resolution:**
```bash
# Skip init container if not critical
kubectl patch deployment weaviate -n nephoran-system -p '
{
  "spec": {
    "template": {
      "spec": {
        "initContainers": []
      }
    }
  }
}'

# Or fix init container configuration
kubectl edit deployment weaviate -n nephoran-system
```

### Issue: CrashLoopBackOff

**Symptoms:**
- Pods continuously restart
- `CrashLoopBackOff` status
- Application fails to start properly

**Diagnostic Commands:**
```bash
# Check container logs
kubectl logs -n nephoran-system -l app=weaviate --previous

# Check current logs
kubectl logs -n nephoran-system -l app=weaviate -f

# Check resource limits
kubectl describe pods -n nephoran-system -l app=weaviate | grep -A 5 "Limits"
```

**Common Causes and Solutions:**

#### Memory Issues

**Diagnosis:**
```bash
# Check for OOMKilled status
kubectl describe pods -n nephoran-system -l app=weaviate | grep -i "killed\|oom"

# Check memory usage patterns
kubectl top pods -n nephoran-system -l app=weaviate --sort-by=memory
```

**Resolution:**
```bash
# Increase memory limits
kubectl patch deployment weaviate -n nephoran-system -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "weaviate",
          "resources": {
            "limits": {
              "memory": "8Gi"
            },
            "requests": {
              "memory": "4Gi"
            }
          }
        }]
      }
    }
  }
}'
```

#### Configuration Issues

**Diagnosis:**
```bash
# Check configuration
kubectl get configmaps -n nephoran-system
kubectl describe configmap weaviate-config -n nephoran-system

# Check secrets
kubectl get secrets -n nephoran-system
```

**Resolution:**
```bash
# Recreate configuration with correct values
kubectl create configmap weaviate-config \
  --from-literal=AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  --from-literal=PERSISTENCE_DATA_PATH=/var/lib/weaviate \
  --from-literal=DEFAULT_VECTORIZER_MODULE=text2vec-openai \
  --namespace=nephoran-system \
  --dry-run=client -o yaml | kubectl apply -f -
```

## 2. Network Connectivity Issues

### Issue: Service Not Accessible

**Symptoms:**
- Cannot reach Weaviate API endpoints
- Connection timeouts or refused connections
- Service discovery failures

**Diagnostic Commands:**
```bash
# Test internal connectivity
kubectl run test-pod --image=curlimages/curl --rm -i --restart=Never \
  -- curl -v "http://weaviate.nephoran-system.svc.cluster.local:8080/v1/.well-known/ready"

# Check DNS resolution
kubectl run test-dns --image=busybox --rm -i --restart=Never \
  -- nslookup weaviate.nephoran-system.svc.cluster.local

# Check service configuration
kubectl describe svc weaviate -n nephoran-system
```

**Resolution Steps:**

#### DNS Resolution Issues

**Diagnosis:**
```bash
# Test cluster DNS
kubectl run test-dns --image=busybox --rm -i --restart=Never \
  -- nslookup kubernetes.default.svc.cluster.local

# Check CoreDNS status
kubectl get pods -n kube-system -l k8s-app=kube-dns
```

**Resolution:**
```bash
# Restart CoreDNS if needed
kubectl rollout restart deployment/coredns -n kube-system

# Check CoreDNS configuration
kubectl get configmap coredns -n kube-system -o yaml
```

#### Network Policy Blocking

**Diagnosis:**
```bash
# Check network policies
kubectl get networkpolicy -n nephoran-system
kubectl describe networkpolicy weaviate-network-policy -n nephoran-system

# Test without network policies
kubectl delete networkpolicy weaviate-network-policy -n nephoran-system
# Test connectivity, then recreate policy
```

**Resolution:**
```bash
# Update network policy to allow required traffic
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: weaviate-network-policy
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
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
    - podSelector:
        matchLabels:
          component: rag-api
    - podSelector:
        matchLabels:
          component: llm-processor
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # HTTPS to OpenAI
    - protocol: UDP
      port: 53   # DNS
EOF
```

### Issue: Load Balancer Not Working

**Symptoms:**
- External traffic cannot reach services
- Load balancer shows unhealthy backends
- Intermittent connectivity

**Diagnostic Commands:**
```bash
# Check service type and external IP
kubectl get svc weaviate -n nephoran-system -o wide

# Check endpoints
kubectl get endpoints weaviate -n nephoran-system

# Check ingress if applicable
kubectl get ingress -n nephoran-system
```

**Resolution:**
```bash
# Update service to use correct type
kubectl patch svc weaviate -n nephoran-system -p '
{
  "spec": {
    "type": "ClusterIP",
    "ports": [{
      "name": "http",
      "port": 8080,
      "targetPort": 8080,
      "protocol": "TCP"
    }]
  }
}'
```

## 3. Storage and Persistence Issues

### Issue: PVC Mounting Failures

**Symptoms:**
- Pods fail to start due to volume mount issues
- PVC stuck in `Pending` state
- Storage-related errors in pod events

**Diagnostic Commands:**
```bash
# Check PVC status
kubectl get pvc -n nephoran-system
kubectl describe pvc weaviate-pvc -n nephoran-system

# Check storage class
kubectl describe storageclass gp3-encrypted

# Check available disk space on nodes
kubectl describe nodes | grep -A 5 "Allocatable"
```

**Resolution Steps:**

#### PVC Provisioning Issues

**Diagnosis:**
```bash
# Check CSI driver status
kubectl get csidriver
kubectl get pods -n kube-system | grep csi

# Check storage class parameters
kubectl get storageclass gp3-encrypted -o yaml
```

**Resolution:**
```bash
# Recreate PVC with correct storage class
kubectl delete pvc weaviate-pvc -n nephoran-system
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: weaviate-pvc
  namespace: nephoran-system
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: gp3-encrypted
EOF
```

#### Disk Space Issues

**Diagnosis:**
```bash
# Check node disk usage
kubectl get pods -n nephoran-system -l app=weaviate -o wide
# Note which nodes pods are on, then:
kubectl describe node <node-name> | grep -A 5 "Conditions"
```

**Resolution:**
```bash
# Clean up unused volumes
kubectl get pv | grep Released | awk '{print $1}' | xargs kubectl delete pv

# Expand existing PVC
kubectl patch pvc weaviate-pvc -n nephoran-system -p '
{
  "spec": {
    "resources": {
      "requests": {
        "storage": "200Gi"
      }
    }
  }
}'
```

### Issue: Data Corruption or Loss

**Symptoms:**
- Weaviate reports data inconsistencies
- Schema or objects missing after restart
- Backup restoration fails

**Diagnostic Commands:**
```bash
# Check Weaviate data consistency
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
curl -s "http://localhost:8080/v1/schema" | jq '.classes | length'
curl -s "http://localhost:8080/v1/meta" | jq '.'

# Check backup status
kubectl get jobs -n nephoran-system -l component=backup
kubectl logs -l component=backup -n nephoran-system --tail=50
```

**Resolution:**
```bash
# Restore from latest backup
kubectl create job weaviate-restore-$(date +%s) \
  --from=cronjob/weaviate-daily-backup \
  --namespace=nephoran-system

# Monitor restoration
kubectl logs job/weaviate-restore-$(date +%s) -n nephoran-system -f

# Reinitialize schema if needed
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: schema-reinit-$(date +%s)
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
      volumes:
      - name: schema-script
        configMap:
          name: telecom-schema-script
      restartPolicy: Never
EOF
```

## 4. Performance Issues

### Issue: Slow Query Response Times

**Symptoms:**
- Query latency > 5 seconds
- Timeouts on complex searches
- High CPU/memory usage during queries

**Diagnostic Commands:**
```bash
# Monitor resource usage during queries
kubectl top pods -n nephoran-system -l app=weaviate --sort-by=cpu

# Check query performance
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
time curl -s "http://localhost:8080/v1/graphql" \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"query": "{Get{TelecomKnowledge(limit: 10){content title}}}"}'
```

**Performance Tuning:**

#### Optimize Vector Index Parameters

```bash
# Update HNSW parameters via API
curl -X PATCH "http://localhost:8080/v1/schema/TelecomKnowledge" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WEAVIATE_API_KEY" \
  -d '{
    "vectorIndexConfig": {
      "ef": 64,
      "efConstruction": 128,
      "maxConnections": 16,
      "vectorCacheMaxObjects": 500000
    }
  }'
```

#### Scale Resources

```bash
# Increase pod resources
kubectl patch deployment weaviate -n nephoran-system -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "weaviate",
          "resources": {
            "requests": {
              "memory": "8Gi",
              "cpu": "2000m"
            },
            "limits": {
              "memory": "16Gi",
              "cpu": "4000m"
            }
          }
        }]
      }
    }
  }
}'

# Scale number of replicas
kubectl scale deployment weaviate --replicas=3 -n nephoran-system
```

### Issue: High Memory Usage

**Symptoms:**
- Memory usage consistently > 90%
- Frequent garbage collection
- OOMKilled restarts

**Diagnostic Commands:**
```bash
# Monitor memory patterns
kubectl top pods -n nephoran-system -l app=weaviate --sort-by=memory

# Check memory limits vs usage
kubectl describe pods -n nephoran-system -l app=weaviate | grep -A 5 "Requests\|Limits"
```

**Resolution:**
```bash
# Optimize memory settings
kubectl set env deployment/weaviate -n nephoran-system \
  GOGC=100 \
  GOMEMLIMIT=12GiB \
  WEAVIATE_MEMORY_LIMIT=12GiB

# Tune vector cache
curl -X PATCH "http://localhost:8080/v1/schema/TelecomKnowledge" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WEAVIATE_API_KEY" \
  -d '{
    "vectorIndexConfig": {
      "vectorCacheMaxObjects": 250000,
      "cleanupIntervalSeconds": 120
    }
  }'
```

## 5. Authentication and Authorization Issues

### Issue: API Authentication Failures

**Symptoms:**
- 401 Unauthorized responses
- Authentication errors in logs
- Cannot access protected endpoints

**Diagnostic Commands:**
```bash
# Check secret existence and content
kubectl get secret weaviate-api-key -n nephoran-system -o yaml
kubectl get secret weaviate-api-key -n nephoran-system -o jsonpath='{.data.api-key}' | base64 -d

# Test authentication
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
API_KEY=$(kubectl get secret weaviate-api-key -n nephoran-system -o jsonpath='{.data.api-key}' | base64 -d)
curl -H "Authorization: Bearer $API_KEY" "http://localhost:8080/v1/meta"
```

**Resolution:**
```bash
# Recreate API key secret
NEW_API_KEY="nephoran-rag-key-$(date +%s)"
kubectl create secret generic weaviate-api-key \
  --from-literal=api-key="$NEW_API_KEY" \
  --namespace=nephoran-system \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart deployment to pick up new secret
kubectl rollout restart deployment/weaviate -n nephoran-system
```

### Issue: RBAC Permission Errors

**Symptoms:**
- Service account permission denied
- Cannot create/read resources
- RBAC-related errors in logs

**Diagnostic Commands:**
```bash
# Check service account and bindings
kubectl get sa -n nephoran-system
kubectl get rolebindings,clusterrolebindings -n nephoran-system

# Test permissions
kubectl auth can-i get pods --as=system:serviceaccount:nephoran-system:weaviate
```

**Resolution:**
```bash
# Update RBAC permissions
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: nephoran-system
  name: weaviate-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints", "configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: weaviate-rolebinding
  namespace: nephoran-system
subjects:
- kind: ServiceAccount
  name: weaviate
  namespace: nephoran-system
roleRef:
  kind: Role
  name: weaviate-role
  apiGroup: rbac.authorization.k8s.io
EOF
```

## 6. Backup and Recovery Issues

### Issue: Backup Jobs Failing

**Symptoms:**
- Backup CronJobs report failures
- No recent successful backups
- Backup storage issues

**Diagnostic Commands:**
```bash
# Check backup job status
kubectl get cronjobs -n nephoran-system -l component=backup
kubectl get jobs -n nephoran-system -l component=backup | head -10

# Check recent backup logs
kubectl logs -n nephoran-system -l component=backup --tail=100
```

**Resolution:**
```bash
# Test manual backup
kubectl create job test-backup-$(date +%s) \
  --from=cronjob/weaviate-daily-backup \
  --namespace=nephoran-system

# Check and fix backup configuration
kubectl get configmap weaviate-backup-config -n nephoran-system -o yaml

# Update backup script if needed
kubectl patch configmap weaviate-backup-config -n nephoran-system -p '
{
  "data": {
    "backup-script.sh": "#!/bin/bash\nset -e\necho \"Starting backup...\"\ncurl -X POST http://weaviate:8080/v1/backups/filesystem -H \"Content-Type: application/json\" -d {\"id\":\"backup-$(date +%Y%m%d-%H%M%S)\"}\necho \"Backup completed\""
  }
}'
```

### Issue: Recovery Process Failures

**Symptoms:**
- Cannot restore from backups
- Data inconsistency after restore
- Recovery jobs timing out

**Diagnostic Commands:**
```bash
# List available backups
kubectl exec -it deployment/weaviate -n nephoran-system -- ls -la /var/lib/weaviate/backups/

# Check backup integrity
kubectl exec -it deployment/weaviate -n nephoran-system -- \
  weaviate-backup verify --backup-id latest
```

**Resolution:**
```bash
# Perform clean recovery
kubectl scale deployment weaviate --replicas=0 -n nephoran-system
kubectl delete pvc weaviate-pvc -n nephoran-system

# Recreate PVC and restore data
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: weaviate-pvc
  namespace: nephoran-system
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 100Gi
  storageClassName: gp3-encrypted
EOF

# Scale back up and wait for restore
kubectl scale deployment weaviate --replicas=1 -n nephoran-system
kubectl wait --for=condition=ready pod -l app=weaviate -n nephoran-system --timeout=300s

# Trigger restore
kubectl create job restore-$(date +%s) \
  --from=cronjob/weaviate-backup-restore \
  --namespace=nephoran-system
```

## 7. Integration Issues

### Issue: RAG API Cannot Connect to Weaviate

**Symptoms:**
- RAG API health checks failing
- Connection refused errors
- Intent processing failures

**Diagnostic Commands:**
```bash
# Test connectivity from RAG API pod
kubectl exec -it deployment/rag-api -n nephoran-system -- \
  curl -v "http://weaviate:8080/v1/.well-known/ready"

# Check service discovery
kubectl get endpoints weaviate -n nephoran-system
kubectl describe svc weaviate -n nephoran-system
```

**Resolution:**
```bash
# Update RAG API configuration
kubectl patch deployment rag-api -n nephoran-system -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "rag-api",
          "env": [{
            "name": "WEAVIATE_URL",
            "value": "http://weaviate.nephoran-system.svc.cluster.local:8080"
          }]
        }]
      }
    }
  }
}'

# Restart RAG API to pick up changes
kubectl rollout restart deployment/rag-api -n nephoran-system
```

### Issue: OpenAI API Integration Problems

**Symptoms:**
- Vector embedding failures
- OpenAI API errors in logs
- Schema creation fails

**Diagnostic Commands:**
```bash
# Check OpenAI API key
kubectl get secret openai-api-key -n nephoran-system -o jsonpath='{.data.api-key}' | base64 -d

# Test OpenAI connectivity from pod
kubectl exec -it deployment/weaviate -n nephoran-system -- \
  curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  "https://api.openai.com/v1/models"
```

**Resolution:**
```bash
# Update OpenAI API key
kubectl create secret generic openai-api-key \
  --from-literal=api-key="sk-your-new-api-key" \
  --namespace=nephoran-system \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart Weaviate to pick up new key
kubectl rollout restart deployment/weaviate -n nephoran-system
```

## 8. Monitoring and Alerting Issues

### Issue: Metrics Not Available

**Symptoms:**
- Prometheus metrics missing
- Grafana dashboards empty
- No monitoring data

**Diagnostic Commands:**
```bash
# Check ServiceMonitor
kubectl get servicemonitor weaviate-monitor -n nephoran-system
kubectl describe servicemonitor weaviate-monitor -n nephoran-system

# Test metrics endpoint
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
curl -s "http://localhost:8080/metrics" | head -20
```

**Resolution:**
```bash
# Recreate ServiceMonitor
cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: weaviate-monitor
  namespace: nephoran-system
  labels:
    app: weaviate
spec:
  selector:
    matchLabels:
      app: weaviate
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
EOF
```

### Issue: Alerts Not Firing

**Symptoms:**
- No alerts despite obvious issues
- Alert manager not receiving alerts
- Notification channels not working

**Diagnostic Commands:**
```bash
# Check PrometheusRule
kubectl get prometheusrule weaviate-alerts -n nephoran-system
kubectl describe prometheusrule weaviate-alerts -n nephoran-system

# Check Prometheus targets
kubectl port-forward svc/prometheus 9090:9090 -n monitoring &
# Visit http://localhost:9090/targets
```

**Resolution:**
```bash
# Update alerting rules
cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: weaviate-alerts
  namespace: nephoran-system
spec:
  groups:
  - name: weaviate
    rules:
    - alert: WeaviatePodDown
      expr: up{job="weaviate"} == 0
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "Weaviate pod is down"
        description: "Weaviate pod has been down for more than 1 minute"
    - alert: WeaviateHighMemoryUsage
      expr: (container_memory_usage_bytes{pod=~"weaviate-.*"} / container_spec_memory_limit_bytes{pod=~"weaviate-.*"}) > 0.9
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Weaviate high memory usage"
        description: "Weaviate memory usage is above 90%"
EOF
```

## Emergency Recovery Procedures

### Complete System Recovery

**When to Use:** Total system failure, data corruption, or cluster compromise

```bash
#!/bin/bash
# emergency-recovery.sh

NAMESPACE="nephoran-system"
BACKUP_ID="latest"

echo "=== EMERGENCY WEAVIATE RECOVERY ==="
echo "This will completely recreate the Weaviate deployment"
read -p "Continue? (yes/NO): " confirm
[[ $confirm != "yes" ]] && exit 1

# 1. Scale down deployment
kubectl scale deployment weaviate --replicas=0 -n $NAMESPACE

# 2. Delete PVCs (this will delete data!)
kubectl delete pvc -l app=weaviate -n $NAMESPACE

# 3. Recreate from backups
kubectl apply -f weaviate-deployment.yaml
kubectl apply -f backup-system.yaml

# 4. Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=weaviate -n $NAMESPACE --timeout=600s

# 5. Restore from backup
kubectl create job emergency-restore-$(date +%s) \
  --from=cronjob/weaviate-daily-backup \
  --namespace=$NAMESPACE

echo "Recovery initiated. Monitor with:"
echo "kubectl logs -f job/emergency-restore-$(date +%s) -n $NAMESPACE"
```

### Disaster Recovery Checklist

- [ ] Verify backup integrity before restoration
- [ ] Document the incident and root cause
- [ ] Update monitoring and alerting rules
- [ ] Test disaster recovery procedures regularly
- [ ] Maintain updated contact information for escalation
- [ ] Review and update this troubleshooting guide

### Support and Escalation

#### Internal Escalation Path

1. **Level 1:** Application team (0-2 hours)
2. **Level 2:** Platform team (2-8 hours)
3. **Level 3:** Senior engineering (8-24 hours)
4. **Level 4:** External vendor support (24+ hours)

#### External Resources

- **Weaviate Documentation:** https://docs.weaviate.io/
- **Kubernetes Troubleshooting:** https://kubernetes.io/docs/tasks/debug-application-cluster/
- **OpenAI API Status:** https://status.openai.com/
- **Community Support:** Weaviate Slack Community

#### Log Collection for Support

```bash
# Collect comprehensive logs for support ticket
mkdir -p /tmp/weaviate-support-$(date +%Y%m%d)
cd /tmp/weaviate-support-$(date +%Y%m%d)

# System information
kubectl get nodes -o wide > nodes.txt
kubectl get pods -A -o wide > all-pods.txt
kubectl get events -A --sort-by='.lastTimestamp' > events.txt

# Weaviate specific
kubectl get all -n nephoran-system > weaviate-resources.txt
kubectl describe pods -n nephoran-system -l app=weaviate > pod-descriptions.txt
kubectl logs -n nephoran-system -l app=weaviate --previous > previous-logs.txt
kubectl logs -n nephoran-system -l app=weaviate > current-logs.txt

# Configuration
kubectl get configmaps -n nephoran-system -o yaml > configmaps.yaml
kubectl get secrets -n nephoran-system -o yaml > secrets.yaml

# Create tarball
cd ..
tar -czf weaviate-support-$(date +%Y%m%d).tar.gz weaviate-support-$(date +%Y%m%d)/
echo "Support bundle created: weaviate-support-$(date +%Y%m%d).tar.gz"
```

This comprehensive troubleshooting guide should help resolve most issues encountered with the Weaviate deployment. Keep this document updated as new issues and solutions are discovered.