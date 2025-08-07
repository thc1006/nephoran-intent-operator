# O2 Infrastructure Management Service Production Readiness Guide

## Overview

This comprehensive guide provides the requirements, checklist, and validation procedures for deploying the O2 Infrastructure Management Service (IMS) in production environments. Following these guidelines ensures enterprise-grade reliability, security, and performance for telecommunications deployments.

## Production Readiness Criteria

### Core Requirements

#### 1. High Availability (99.95% SLA)

**Requirements:**
- Multi-zone deployment across at least 3 availability zones
- Minimum 3 replicas for API services
- Pod disruption budgets configured
- Anti-affinity rules to prevent single points of failure
- Load balancer with health checks

**Configuration:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: o2ims-api
  namespace: nephoran-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: o2ims-api
  template:
    metadata:
      labels:
        app: o2ims-api
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - o2ims-api
            topologyKey: kubernetes.io/hostname
      containers:
      - name: o2ims-api
        image: nephoran/o2ims:v1.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8443
          name: https
        resources:
          requests:
            cpu: 1000m
            memory: 2Gi
          limits:
            cpu: 2000m
            memory: 4Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
        env:
        - name: O2IMS_ENVIRONMENT
          value: "production"
        - name: O2IMS_LOG_LEVEL
          value: "info"
        volumeMounts:
        - name: config
          mountPath: /etc/o2ims
        - name: tls-certs
          mountPath: /etc/ssl/certs/o2ims
      volumes:
      - name: config
        configMap:
          name: o2ims-config
      - name: tls-certs
        secret:
          secretName: o2ims-tls-certs
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: o2ims-pdb
  namespace: nephoran-system
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: o2ims-api
```

#### 2. Security Hardening

**Requirements:**
- TLS encryption for all communications
- OAuth 2.0 authentication with JWT tokens
- Role-based access control (RBAC)
- Network policies for traffic segmentation
- Security contexts with non-root users
- Regular security scanning

**TLS Configuration:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: o2ims-tls-certs
  namespace: nephoran-system
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTi...  # Base64 encoded certificate
  tls.key: LS0tLS1CRUdJTi...  # Base64 encoded private key
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: o2ims-network-policy
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app: o2ims-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: istio-system
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 8443
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
  - to: []
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 80
```

**RBAC Configuration:**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: o2ims-service-account
  namespace: nephoran-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: o2ims-cluster-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["networking.k8s.io"]
  resources: ["networkpolicies", "ingresses"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["config.porch.kpt.dev"]
  resources: ["packagerevisions", "packages"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: o2ims-cluster-role-binding
subjects:
- kind: ServiceAccount
  name: o2ims-service-account
  namespace: nephoran-system
roleRef:
  kind: ClusterRole
  name: o2ims-cluster-role
  apiGroup: rbac.authorization.k8s.io
```

#### 3. Performance and Scalability

**SLA Requirements:**
- API Response Time P50 < 100ms
- API Response Time P95 < 500ms  
- API Response Time P99 < 1000ms
- Throughput > 1000 RPS
- Support 200+ concurrent intents

**Auto-scaling Configuration:**
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: o2ims-hpa
  namespace: nephoran-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: o2ims-api
  minReplicas: 3
  maxReplicas: 20
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
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

**Vertical Pod Autoscaling:**
```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: o2ims-vpa
  namespace: nephoran-system
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: o2ims-api
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: o2ims-api
      maxAllowed:
        cpu: 4
        memory: 8Gi
      minAllowed:
        cpu: 500m
        memory: 1Gi
```

#### 4. Data Persistence and Backup

**Requirements:**
- Persistent storage for configuration and state
- Automated backup with 7-day retention
- Point-in-time recovery capability
- Cross-region backup replication

**Persistent Storage:**
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: o2ims-data
  namespace: nephoran-system
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: premium-ssd
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: o2ims-config
  namespace: nephoran-system
data:
  config.yaml: |
    database:
      type: postgresql
      host: postgres.database.svc.cluster.local
      port: 5432
      database: o2ims
      ssl: require
      maxConnections: 100
      connectionTimeout: 30s
      
    storage:
      type: persistent
      path: /data/o2ims
      backup:
        enabled: true
        schedule: "0 2 * * *"
        retention: "7d"
        destination: s3://backup-bucket/o2ims
        
    performance:
      cacheSize: 1000
      cacheTTL: 5m
      maxConcurrentRequests: 500
      requestTimeout: 30s
```

**Backup CronJob:**
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: o2ims-backup
  namespace: nephoran-system
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: nephoran/backup-tool:v1.0.0
            command:
            - /bin/sh
            - -c
            - |
              set -e
              echo "Starting O2 IMS backup at $(date)"
              
              # Database backup
              pg_dump -h postgres.database.svc.cluster.local \
                      -U $DB_USER -d o2ims \
                      -f /backup/o2ims-$(date +%Y%m%d-%H%M%S).sql
              
              # Configuration backup
              kubectl get configmaps -n nephoran-system -o yaml > /backup/configmaps-$(date +%Y%m%d-%H%M%S).yaml
              kubectl get secrets -n nephoran-system -o yaml > /backup/secrets-$(date +%Y%m%d-%H%M%S).yaml
              
              # Upload to S3
              aws s3 sync /backup/ s3://backup-bucket/o2ims/$(date +%Y/%m/%d)/
              
              # Cleanup old local files
              find /backup -type f -mtime +1 -delete
              
              echo "Backup completed at $(date)"
            env:
            - name: DB_USER
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: username
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: password
            volumeMounts:
            - name: backup-storage
              mountPath: /backup
          volumes:
          - name: backup-storage
            persistentVolumeClaim:
              claimName: backup-pvc
          restartPolicy: OnFailure
```

#### 5. Monitoring and Observability

**Requirements:**
- Prometheus metrics collection
- Grafana dashboards
- Jaeger distributed tracing
- Centralized logging with ELK stack
- Custom alerting rules

**ServiceMonitor Configuration:**
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: o2ims-service-monitor
  namespace: nephoran-system
  labels:
    app: o2ims-api
spec:
  selector:
    matchLabels:
      app: o2ims-api
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
    scrapeTimeout: 10s
---
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: o2ims-alerts
  namespace: nephoran-system
spec:
  groups:
  - name: o2ims.rules
    rules:
    - alert: O2IMSHighLatency
      expr: histogram_quantile(0.95, o2ims_api_request_duration_seconds_bucket) > 1.0
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "O2 IMS API high latency"
        description: "95th percentile latency is {{ $value }} seconds"
        
    - alert: O2IMSHighErrorRate
      expr: rate(o2ims_api_requests_total{status=~"5.."}[5m]) / rate(o2ims_api_requests_total[5m]) > 0.05
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "O2 IMS high error rate"
        description: "Error rate is {{ $value | humanizePercentage }}"
        
    - alert: O2IMSDown
      expr: up{job="o2ims-api"} == 0
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "O2 IMS is down"
        description: "O2 IMS has been down for more than 1 minute"
```

**Grafana Dashboard JSON:**
```json
{
  "dashboard": {
    "title": "O2 IMS Production Dashboard",
    "tags": ["o2ims", "production", "infrastructure"],
    "panels": [
      {
        "title": "API Request Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(rate(o2ims_api_requests_total[5m]))",
            "legendFormat": "RPS"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 100},
                {"color": "green", "value": 500}
              ]
            }
          }
        }
      },
      {
        "title": "Response Time Percentiles",
        "type": "timeseries",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, o2ims_api_request_duration_seconds_bucket)",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.95, o2ims_api_request_duration_seconds_bucket)",
            "legendFormat": "P95"
          },
          {
            "expr": "histogram_quantile(0.99, o2ims_api_request_duration_seconds_bucket)",
            "legendFormat": "P99"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "timeseries",
        "targets": [
          {
            "expr": "rate(o2ims_api_requests_total{status=~\"4..\"}[5m])",
            "legendFormat": "4xx"
          },
          {
            "expr": "rate(o2ims_api_requests_total{status=~\"5..\"}[5m])",
            "legendFormat": "5xx"
          }
        ]
      },
      {
        "title": "Pod CPU Usage",
        "type": "timeseries",
        "targets": [
          {
            "expr": "rate(container_cpu_usage_seconds_total{pod=~\"o2ims-api-.*\"}[5m])",
            "legendFormat": "{{pod}}"
          }
        ]
      },
      {
        "title": "Pod Memory Usage",
        "type": "timeseries",
        "targets": [
          {
            "expr": "container_memory_usage_bytes{pod=~\"o2ims-api-.*\"}",
            "legendFormat": "{{pod}}"
          }
        ]
      }
    ]
  }
}
```

## Production Deployment Procedure

### Pre-Deployment Checklist

#### Infrastructure Requirements

1. **Kubernetes Cluster:**
   - Version: 1.24+
   - Nodes: Minimum 3 worker nodes
   - Resources: 32 vCPU, 64GB RAM, 1TB SSD per node
   - Network: Pod and Service CIDR configured
   - Storage: CSI driver with snapshot support

2. **External Dependencies:**
   - PostgreSQL 13+ database cluster
   - Redis 7+ for caching
   - Load balancer (AWS ALB, GCP LB, etc.)
   - DNS provider for custom domains
   - Certificate management (Let's Encrypt, internal CA)

3. **Networking:**
   - Ingress controller (NGINX, Istio)
   - Service mesh (Istio) for mTLS
   - Network policies for micro-segmentation

#### Security Prerequisites

```bash
# 1. Create namespace with security labels
kubectl create namespace nephoran-system
kubectl label namespace nephoran-system \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted

# 2. Create service account and RBAC
kubectl apply -f rbac/service-accounts.yaml
kubectl apply -f rbac/cluster-roles.yaml
kubectl apply -f rbac/role-bindings.yaml

# 3. Generate TLS certificates
./scripts/generate-tls-certs.sh nephoran-system

# 4. Create secrets
kubectl create secret generic o2ims-database-credentials \
  --from-literal=username=o2ims \
  --from-literal=password="$(openssl rand -base64 32)" \
  -n nephoran-system

# 5. Set up OAuth 2.0 configuration
kubectl create secret generic o2ims-oauth-config \
  --from-literal=client-id=o2ims-production \
  --from-literal=client-secret="$(openssl rand -base64 32)" \
  --from-literal=issuer-url=https://auth.company.com/realms/production \
  -n nephoran-system
```

### Deployment Steps

#### 1. Deploy Database

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: postgres-cluster
  namespace: database
spec:
  instances: 3
  primaryUpdateStrategy: unsupervised
  
  postgresql:
    parameters:
      max_connections: "200"
      shared_preload_libraries: "pg_stat_statements"
      pg_stat_statements.max: "10000"
      
  bootstrap:
    initdb:
      database: o2ims
      owner: o2ims
      secret:
        name: postgres-credentials
        
  storage:
    size: 500Gi
    storageClass: premium-ssd
    
  monitoring:
    enabled: true
    
  backup:
    retentionPolicy: "30d"
    barmanObjectStore:
      destinationPath: s3://backup-bucket/postgres
      s3Credentials:
        accessKeyId:
          name: backup-credentials
          key: access-key
        secretAccessKey:
          name: backup-credentials
          key: secret-key
```

#### 2. Deploy O2 IMS

```bash
# Deploy configuration
kubectl apply -f config/configmaps.yaml
kubectl apply -f config/secrets.yaml

# Deploy the application
kubectl apply -f deployments/production/o2ims-deployment.yaml
kubectl apply -f deployments/production/o2ims-service.yaml
kubectl apply -f deployments/production/o2ims-ingress.yaml

# Deploy monitoring
kubectl apply -f monitoring/servicemonitor.yaml
kubectl apply -f monitoring/prometheus-rules.yaml

# Deploy autoscaling
kubectl apply -f scaling/hpa.yaml
kubectl apply -f scaling/vpa.yaml
```

#### 3. Verify Deployment

```bash
# Run production readiness check
./scripts/o2-production-readiness-check.sh nephoran-system

# Verify all components are healthy
kubectl get pods -n nephoran-system
kubectl get services -n nephoran-system
kubectl get ingress -n nephoran-system

# Check metrics
curl -k https://o2ims.company.com/metrics

# Test API endpoints
curl -k https://o2ims.company.com/o2ims/v1/
curl -k https://o2ims.company.com/health
curl -k https://o2ims.company.com/ready
```

## Performance Tuning

### JVM/Runtime Tuning

```yaml
env:
- name: GO_GCFLAGS
  value: "-N -l"
- name: GOMAXPROCS
  value: "4"
- name: GOGC
  value: "100"
- name: GOMEMLIMIT
  value: "3500MiB"
```

### Database Optimization

```sql
-- PostgreSQL configuration for O2 IMS
-- postgresql.conf settings

# Memory
shared_buffers = 1GB
work_mem = 64MB
maintenance_work_mem = 256MB

# Checkpoints
checkpoint_completion_target = 0.9
wal_buffers = 64MB

# Connections
max_connections = 200
max_prepared_transactions = 200

# Logging
log_statement = 'mod'
log_min_duration_statement = 1000

-- Create optimized indexes
CREATE INDEX CONCURRENTLY idx_resource_pools_provider ON resource_pools(provider);
CREATE INDEX CONCURRENTLY idx_resource_pools_location ON resource_pools(location);
CREATE INDEX CONCURRENTLY idx_resources_state ON resources(resource_state);
CREATE INDEX CONCURRENTLY idx_deployments_status ON deployments(status);

-- Analyze tables for query optimization
ANALYZE resource_pools;
ANALYZE resource_types;
ANALYZE resources;
ANALYZE deployments;
```

### Caching Strategy

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: nephoran-system
data:
  redis.conf: |
    maxmemory 2gb
    maxmemory-policy allkeys-lru
    save 900 1
    save 300 10
    save 60 10000
    
    # Performance tuning
    tcp-keepalive 300
    timeout 0
    tcp-backlog 511
    
    # Memory optimization
    hash-max-ziplist-entries 512
    hash-max-ziplist-value 64
    list-max-ziplist-size -2
    set-max-intset-entries 512
```

## Security Hardening

### Pod Security Standards

```yaml
apiVersion: v1
kind: Pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    runAsGroup: 65534
    fsGroup: 65534
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: o2ims-api
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      runAsNonRoot: true
      runAsUser: 65534
      capabilities:
        drop:
        - ALL
        add:
        - NET_BIND_SERVICE
    volumeMounts:
    - name: tmp
      mountPath: /tmp
    - name: var-cache
      mountPath: /var/cache
  volumes:
  - name: tmp
    emptyDir: {}
  - name: var-cache
    emptyDir: {}
```

### Network Security

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: o2ims-authz
  namespace: nephoran-system
spec:
  selector:
    matchLabels:
      app: o2ims-api
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account"]
  - to:
    - operation:
        methods: ["GET", "POST", "PUT", "PATCH", "DELETE"]
        paths: ["/o2ims/v1/*", "/health", "/ready", "/metrics"]
---
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: o2ims-mtls
  namespace: nephoran-system
spec:
  selector:
    matchLabels:
      app: o2ims-api
  mtls:
    mode: STRICT
```

## Disaster Recovery

### Multi-Region Setup

```yaml
# Primary region deployment
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: o2ims-primary
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/company/o2ims-config
    path: overlays/production/primary
    targetRevision: HEAD
  destination:
    server: https://primary-cluster.company.com
    namespace: nephoran-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
---
# Secondary region deployment
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: o2ims-secondary
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/company/o2ims-config
    path: overlays/production/secondary
    targetRevision: HEAD
  destination:
    server: https://secondary-cluster.company.com
    namespace: nephoran-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Recovery Procedures

```bash
#!/bin/bash
# disaster-recovery-playbook.sh

# Step 1: Assess primary region status
echo "Checking primary region health..."
kubectl --kubeconfig primary-cluster.kubeconfig get pods -n nephoran-system

# Step 2: Promote secondary region if primary is down
if ! kubectl --kubeconfig primary-cluster.kubeconfig get pods -n nephoran-system >/dev/null 2>&1; then
    echo "Primary region down. Promoting secondary..."
    
    # Update DNS to point to secondary region
    aws route53 change-resource-record-sets \
        --hosted-zone-id Z1234567890ABC \
        --change-batch file://dns-failover.json
    
    # Scale up secondary region
    kubectl --kubeconfig secondary-cluster.kubeconfig \
        scale deployment o2ims-api -n nephoran-system --replicas=5
    
    # Restore database from latest backup
    ./scripts/restore-database.sh secondary
    
    echo "Failover complete. Monitor secondary region."
fi

# Step 3: Recovery timeline
echo "Recovery Steps:"
echo "1. Fix primary region issues (ETA: 30 minutes)"
echo "2. Restore primary database (ETA: 15 minutes)"
echo "3. Sync data from secondary to primary (ETA: 10 minutes)"
echo "4. Failback to primary region (ETA: 5 minutes)"
```

## Compliance and Auditing

### Audit Logging

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: audit-policy
  namespace: kube-system
data:
  audit-policy.yaml: |
    apiVersion: audit.k8s.io/v1
    kind: Policy
    rules:
    - level: Request
      namespaces: ["nephoran-system"]
      resources:
      - group: ""
        resources: ["secrets", "configmaps"]
      - group: "apps"
        resources: ["deployments"]
    - level: Metadata
      namespaces: ["nephoran-system"]
      omitStages:
      - RequestReceived
```

### SOC 2 Compliance

```yaml
# Security controls for SOC 2 compliance
apiVersion: v1
kind: ConfigMap
metadata:
  name: security-controls
  namespace: nephoran-system
data:
  controls.yaml: |
    access_control:
      authentication: "OAuth 2.0 with MFA required"
      authorization: "RBAC with principle of least privilege"
      session_timeout: "30 minutes"
      
    data_protection:
      encryption_at_rest: "AES-256"
      encryption_in_transit: "TLS 1.3"
      key_management: "External KMS (AWS KMS/HashiCorp Vault)"
      
    monitoring:
      log_retention: "7 years"
      audit_trails: "All administrative actions"
      intrusion_detection: "Real-time alerting"
      
    availability:
      rpo: "15 minutes"
      rto: "1 hour"
      backup_frequency: "Every 6 hours"
      testing_frequency: "Monthly"
```

## Performance Benchmarking

### Load Testing

```bash
#!/bin/bash
# load-test.sh

# Install dependencies
go install github.com/rakyll/hey@latest

# Test scenarios
echo "Running load tests..."

# Scenario 1: Normal load (100 RPS for 5 minutes)
hey -n 30000 -c 100 -q 100 -t 30 \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    https://o2ims.company.com/o2ims/v1/resourcePools

# Scenario 2: Peak load (500 RPS for 2 minutes)
hey -n 60000 -c 500 -q 500 -t 30 \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    https://o2ims.company.com/o2ims/v1/resourcePools

# Scenario 3: Burst load (1000 RPS for 30 seconds)
hey -n 30000 -c 1000 -q 1000 -t 30 \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    https://o2ims.company.com/o2ims/v1/resourcePools

# Scenario 4: Complex queries
hey -n 10000 -c 100 -t 30 \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    "https://o2ims.company.com/o2ims/v1/resourcePools?filter=provider,eq,kubernetes;location,in,us-west-2,us-east-1&limit=50&sort=name,asc"

echo "Load testing complete. Check Grafana dashboards for results."
```

### Expected Performance Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| P50 Response Time | < 100ms | 95% compliance |
| P95 Response Time | < 500ms | 98% compliance |
| P99 Response Time | < 1000ms | 99% compliance |
| Throughput | > 1000 RPS | Sustained |
| Error Rate | < 0.1% | 4xx and 5xx |
| Availability | 99.95% | Monthly uptime |
| MTTR | < 5 minutes | Incident response |
| MTBF | > 720 hours | Between failures |

## Cost Optimization

### Resource Right-Sizing

```yaml
# Cost-optimized resource configuration
resources:
  requests:
    cpu: 800m      # Reduced from 1000m
    memory: 1.5Gi  # Reduced from 2Gi
  limits:
    cpu: 1.5       # Reduced from 2
    memory: 3Gi    # Reduced from 4Gi

# Spot instances for non-critical workloads
nodeSelector:
  node-type: spot
tolerations:
- key: spot
  operator: Equal
  value: "true"
  effect: NoSchedule
```

### Storage Optimization

```yaml
# Tiered storage strategy
apiVersion: v1
kind: StorageClass
metadata:
  name: o2ims-hot-storage
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
---
apiVersion: v1
kind: StorageClass
metadata:
  name: o2ims-cold-storage
provisioner: ebs.csi.aws.com
parameters:
  type: sc1  # Cold HDD for backups
```

## Maintenance Procedures

### Rolling Updates

```bash
#!/bin/bash
# rolling-update.sh

VERSION="$1"
if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

echo "Starting rolling update to version $VERSION..."

# Pre-update health check
kubectl rollout status deployment/o2ims-api -n nephoran-system

# Update image
kubectl set image deployment/o2ims-api \
    o2ims-api=nephoran/o2ims:$VERSION \
    -n nephoran-system

# Monitor rollout
kubectl rollout status deployment/o2ims-api -n nephoran-system --timeout=300s

# Post-update verification
sleep 30
./scripts/o2-production-readiness-check.sh nephoran-system

if [[ $? -eq 0 ]]; then
    echo "Rolling update to $VERSION completed successfully"
else
    echo "Rolling update failed. Initiating rollback..."
    kubectl rollout undo deployment/o2ims-api -n nephoran-system
    exit 1
fi
```

### Certificate Rotation

```bash
#!/bin/bash
# cert-rotation.sh

echo "Starting certificate rotation..."

# Generate new certificates
./scripts/generate-tls-certs.sh nephoran-system

# Update secret with new certificates
kubectl create secret tls o2ims-tls-certs-new \
    --cert=tls.crt \
    --key=tls.key \
    -n nephoran-system

# Rolling restart to pick up new certificates
kubectl patch deployment o2ims-api -n nephoran-system \
    -p '{"spec":{"template":{"spec":{"volumes":[{"name":"tls-certs","secret":{"secretName":"o2ims-tls-certs-new"}}]}}}}'

kubectl rollout status deployment/o2ims-api -n nephoran-system

# Cleanup old certificates
kubectl delete secret o2ims-tls-certs -n nephoran-system
kubectl create secret tls o2ims-tls-certs \
    --cert=tls.crt \
    --key=tls.key \
    -n nephoran-system

echo "Certificate rotation completed"
```

This comprehensive production readiness guide ensures enterprise-grade deployment of the O2 Infrastructure Management Service with all necessary security, performance, and operational requirements met for telecommunications production environments.