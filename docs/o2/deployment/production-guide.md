# O2 IMS Production Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying the O2 Infrastructure Management Service in production environments, covering security, high availability, performance optimization, and operational best practices.

## Prerequisites

### Infrastructure Requirements

#### Minimum System Requirements

| Component | Minimum | Recommended | High Availability |
|-----------|---------|-------------|-------------------|
| **CPU** | 2 cores | 4 cores | 8 cores |
| **Memory** | 4GB | 8GB | 16GB |
| **Storage** | 100GB SSD | 500GB SSD | 1TB NVMe |
| **Network** | 1Gbps | 10Gbps | 25Gbps |

#### Kubernetes Cluster Requirements

- **Version**: Kubernetes 1.24+ (1.28+ recommended)
- **Nodes**: Minimum 3 worker nodes for HA
- **Storage Class**: Dynamic provisioning supported
- **Network Plugin**: CNI-compliant (Calico, Cilium, or Flannel)
- **Service Mesh**: Istio 1.18+ (optional but recommended)

#### Database Requirements

- **PostgreSQL**: 14.0+ (recommended) or 13.0+
- **High Availability**: PostgreSQL cluster with streaming replication
- **Storage**: High-performance SSD with IOPS > 3000
- **Memory**: Minimum 8GB for PostgreSQL instance
- **Backup**: Automated backup with point-in-time recovery

### External Dependencies

#### Required Services

- **Container Registry**: Docker Hub, Harbor, or cloud provider registry
- **Load Balancer**: External load balancer for ingress
- **Certificate Authority**: For TLS certificate management
- **Identity Provider**: OAuth2/OIDC provider (optional)

#### Optional Services

- **Monitoring Stack**: Prometheus, Grafana, AlertManager
- **Logging Stack**: ELK or EFK stack
- **Tracing**: Jaeger or Zipkin
- **Service Mesh**: Istio for advanced traffic management

## Production Architecture

### High Availability Architecture

```
                              ┌─────────────────┐
                              │   Load Balancer │
                              │   (External)    │
                              └─────────┬───────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
              ┌─────▼─────┐       ┌─────▼─────┐       ┌─────▼─────┐
              │ O2 IMS    │       │ O2 IMS    │       │ O2 IMS    │
              │ Instance 1│       │ Instance 2│       │ Instance 3│
              └─────┬─────┘       └─────┬─────┘       └─────┬─────┘
                    │                   │                   │
                    └───────────────────┼───────────────────┘
                                        │
                              ┌─────────▼───────────┐
                              │   PostgreSQL HA     │
                              │   (Primary/Replica) │
                              └─────────────────────┘
```

### Network Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         DMZ / Edge Zone                             │
├─────────────────────────────────────────────────────────────────────┤
│  External Load Balancer (HAProxy/F5/Cloud LB)                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ TLS Termination │ WAF │ Rate Limiting │ DDoS Protection   │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────┬───────────────────────────────────────────┘
                          │ Encrypted Traffic
┌─────────────────────────▼───────────────────────────────────────────┐
│                    Application Zone                                 │
├─────────────────────────────────────────────────────────────────────┤
│  Kubernetes Cluster                                                │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Ingress Controller (NGINX/Istio Gateway)                   │    │
│  └─────────────────────────┬───────────────────────────────────┘    │
│                            │                                        │
│  ┌─────────────────────────▼───────────────────────────────────┐    │
│  │ O2 IMS Service Pods (3+ replicas)                          │    │
│  │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │    │
│  │ │  Instance 1 │ │  Instance 2 │ │  Instance 3 │           │    │
│  │ └─────────────┘ └─────────────┘ └─────────────┘           │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────┬───────────────────────────────────────────┘
                          │ Internal Network
┌─────────────────────────▼───────────────────────────────────────────┐
│                      Data Zone                                      │
├─────────────────────────────────────────────────────────────────────┤
│  PostgreSQL HA Cluster                                             │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Primary │ Replica 1 │ Replica 2 │ Backup Storage          │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

## Deployment Methods

### Method 1: Helm Chart Deployment (Recommended)

#### 1.1 Install Helm Repository

```bash
# Add Nephoran Helm repository
helm repo add nephoran https://charts.nephoran.com
helm repo update

# Verify repository
helm search repo nephoran/o2-ims
```

#### 1.2 Configure Production Values

Create `production-values.yaml`:

```yaml
# production-values.yaml
global:
  environment: "production"
  clusterDomain: "cluster.local"

# Application configuration
o2ims:
  image:
    repository: "registry.nephoran.com/o2-ims"
    tag: "v1.2.0"
    pullPolicy: "IfNotPresent"

  # Replica configuration for HA
  replicaCount: 3
  
  # Resource requirements
  resources:
    requests:
      cpu: "1000m"
      memory: "2Gi"
    limits:
      cpu: "2000m"
      memory: "4Gi"

  # Pod disruption budget
  podDisruptionBudget:
    enabled: true
    minAvailable: 2

  # Horizontal Pod Autoscaler
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80

  # Health checks
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
    initialDelaySeconds: 10
    periodSeconds: 5
    timeoutSeconds: 3
    failureThreshold: 2

  # Security context
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    fsGroup: 65534
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop:
        - ALL

  # Service configuration
  service:
    type: ClusterIP
    port: 8080
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"

# Database configuration
postgresql:
  enabled: true
  architecture: "replication"
  auth:
    postgresPassword: "secure_password_here"
    username: "o2user"
    password: "secure_user_password_here"
    database: "o2ims"

  primary:
    resources:
      requests:
        cpu: "1000m"
        memory: "2Gi"
      limits:
        cpu: "2000m"
        memory: "4Gi"
    
    persistence:
      enabled: true
      size: "100Gi"
      storageClass: "fast-ssd"
    
    # PostgreSQL configuration
    postgresql:
      maxConnections: 200
      sharedBuffers: "512MB"
      effectiveCacheSize: "1536MB"
      maintenanceWorkMem: "128MB"
      walBuffers: "16MB"

  readReplicas:
    replicaCount: 2
    resources:
      requests:
        cpu: "500m"
        memory: "1Gi"
      limits:
        cpu: "1000m"
        memory: "2Gi"

  # Backup configuration
  backup:
    enabled: true
    schedule: "0 2 * * *"
    retention: 30

# Ingress configuration
ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: "o2ims.your-domain.com"
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: "o2ims-tls"
      hosts:
        - "o2ims.your-domain.com"

# Monitoring configuration
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: "30s"
    scrapeTimeout: "10s"
  
  prometheusRule:
    enabled: true
    rules:
      - alert: "O2IMSHighLatency"
        expr: "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5"
        for: "2m"
        labels:
          severity: "warning"
        annotations:
          summary: "O2 IMS high latency detected"
      
      - alert: "O2IMSHighErrorRate"
        expr: "rate(http_requests_total{status=~\"5..\"}[5m]) > 0.1"
        for: "1m"
        labels:
          severity: "critical"
        annotations:
          summary: "O2 IMS high error rate detected"

# Logging configuration
logging:
  enabled: true
  level: "info"
  format: "json"
  output: "stdout"

# Security configuration
security:
  # Network policies
  networkPolicy:
    enabled: true
    egress:
      - to: []
        ports:
          - protocol: TCP
            port: 5432  # PostgreSQL
          - protocol: TCP
            port: 443   # HTTPS
          - protocol: TCP
            port: 53    # DNS
          - protocol: UDP
            port: 53    # DNS

  # Pod Security Policy
  podSecurityPolicy:
    enabled: true
    allowPrivilegeEscalation: false
    runAsNonRoot: true
    fsGroup:
      rule: RunAsAny
    runAsUser:
      rule: MustRunAsNonRoot
    seLinux:
      rule: RunAsAny
    volumes:
      - configMap
      - emptyDir
      - projected
      - secret
      - downwardAPI
      - persistentVolumeClaim

# Provider configurations
providers:
  kubernetes:
    enabled: true
    config:
      inCluster: true
  
  aws:
    enabled: false
    config:
      region: "us-east-1"
      # Credentials will be provided via IAM roles
  
  azure:
    enabled: false
  
  gcp:
    enabled: false
```

#### 1.3 Deploy to Production

```bash
# Create namespace
kubectl create namespace o2-system

# Install with production values
helm install o2-ims nephoran/o2-ims \
  --namespace o2-system \
  --values production-values.yaml \
  --wait \
  --timeout 10m

# Verify deployment
kubectl get pods -n o2-system
kubectl get svc -n o2-system
kubectl get ingress -n o2-system
```

### Method 2: GitOps Deployment

#### 2.1 ArgoCD Configuration

Create `argocd-application.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: o2-ims-production
  namespace: argocd
spec:
  project: default
  
  source:
    repoURL: https://github.com/your-org/o2-ims-config
    targetRevision: HEAD
    path: production
  
  destination:
    server: https://kubernetes.default.svc
    namespace: o2-system
  
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - PruneLast=true
  
  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas
```

#### 2.2 Flux Configuration

Create `flux-kustomization.yaml`:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: o2-ims-production
  namespace: flux-system
spec:
  interval: 5m
  path: ./production
  prune: true
  sourceRef:
    kind: GitRepository
    name: o2-ims-config
  validation: client
  healthChecks:
    - apiVersion: apps/v1
      kind: Deployment
      name: o2-ims
      namespace: o2-system
```

## Configuration Management

### Environment-Specific Configuration

#### Production ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: o2-ims-config
  namespace: o2-system
data:
  config.yaml: |
    server:
      address: "0.0.0.0"
      port: 8080
      tls:
        enabled: true
        certFile: "/etc/tls/tls.crt"
        keyFile: "/etc/tls/tls.key"
      
      # Timeouts
      readTimeout: "30s"
      writeTimeout: "30s"
      idleTimeout: "60s"
      
      # Rate limiting
      rateLimit:
        enabled: true
        requestsPerSecond: 100
        burst: 200

    database:
      type: "postgresql"
      host: "postgresql-primary.o2-system.svc.cluster.local"
      port: 5432
      database: "o2ims"
      username: "o2user"
      # Password from secret
      maxConnections: 50
      maxIdleConnections: 10
      connectionMaxLifetime: "1h"
      
      # SSL configuration
      sslMode: "require"
      sslCert: "/etc/ssl/client.crt"
      sslKey: "/etc/ssl/client.key"
      sslRootCert: "/etc/ssl/ca.crt"

    providers:
      kubernetes:
        enabled: true
        config:
          inCluster: true
          qps: 50
          burst: 100
          timeout: "30s"
      
      aws:
        enabled: true
        config:
          region: "us-east-1"
          maxRetries: 3
          timeout: "30s"
          # Use IAM roles for authentication

    monitoring:
      metrics:
        enabled: true
        path: "/metrics"
        interval: "30s"
      
      tracing:
        enabled: true
        endpoint: "http://jaeger-collector.monitoring:14268/api/traces"
        samplingRate: 0.1
      
      healthChecks:
        enabled: true
        interval: "10s"
        timeout: "5s"

    logging:
      level: "info"
      format: "json"
      output: "stdout"
      fields:
        service: "o2-ims"
        version: "1.2.0"
        environment: "production"

    security:
      authentication:
        enabled: true
        providers:
          - type: "oauth2"
            name: "corporate-sso"
            config:
              issuerURL: "https://auth.company.com"
              clientID: "o2-ims-production"
              # Client secret from secret
      
      authorization:
        enabled: true
        rbac:
          enabled: true
          configFile: "/etc/rbac/rbac.yaml"
      
      cors:
        enabled: true
        allowedOrigins:
          - "https://console.company.com"
          - "https://monitoring.company.com"
        allowedMethods:
          - "GET"
          - "POST"
          - "PUT"
          - "DELETE"
        allowedHeaders:
          - "Authorization"
          - "Content-Type"
          - "X-Correlation-ID"

    features:
      # Feature flags for gradual rollout
      multiCloudSupport: true
      advancedMonitoring: true
      experimentalAPIs: false
```

#### Secrets Management

```yaml
# Database credentials
apiVersion: v1
kind: Secret
metadata:
  name: postgresql-credentials
  namespace: o2-system
type: Opaque
data:
  password: <base64-encoded-password>

---
# OAuth2 client secret
apiVersion: v1
kind: Secret
metadata:
  name: oauth2-credentials
  namespace: o2-system
type: Opaque
data:
  client-secret: <base64-encoded-client-secret>

---
# TLS certificates
apiVersion: v1
kind: Secret
metadata:
  name: o2-ims-tls
  namespace: o2-system
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-certificate>
  tls.key: <base64-encoded-private-key>
```

## Security Hardening

### Container Security

#### 1. Security Context Configuration

```yaml
securityContext:
  # Pod-level security context
  runAsNonRoot: true
  runAsUser: 65534
  runAsGroup: 65534
  fsGroup: 65534
  seccompProfile:
    type: RuntimeDefault

containers:
  - name: o2-ims
    securityContext:
      # Container-level security context
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      runAsNonRoot: true
      runAsUser: 65534
      capabilities:
        drop:
          - ALL
        add:
          - NET_BIND_SERVICE  # Only if needed for privileged ports
```

#### 2. Network Security

```yaml
# Network Policy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: o2-ims-network-policy
  namespace: o2-system
spec:
  podSelector:
    matchLabels:
      app: o2-ims
  
  policyTypes:
    - Ingress
    - Egress
  
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-system
      ports:
        - protocol: TCP
          port: 8080
    
    # Allow monitoring scraping
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - protocol: TCP
          port: 8080
  
  egress:
    # Allow database access
    - to:
        - podSelector:
            matchLabels:
              app: postgresql
      ports:
        - protocol: TCP
          port: 5432
    
    # Allow external provider APIs
    - to: []
      ports:
        - protocol: TCP
          port: 443
    
    # Allow DNS
    - to: []
      ports:
        - protocol: TCP
          port: 53
        - protocol: UDP
          port: 53
```

#### 3. Pod Security Standards

```yaml
# Pod Security Policy (if using PSP)
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: o2-ims-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
```

### TLS Configuration

#### 1. Certificate Management

```bash
# Using cert-manager for automatic certificate management
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: o2-ims-tls
  namespace: o2-system
spec:
  secretName: o2-ims-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - o2ims.your-domain.com
    - api.o2ims.your-domain.com
```

#### 2. Internal TLS (Service Mesh)

```yaml
# Istio PeerAuthentication
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: o2-ims-mtls
  namespace: o2-system
spec:
  selector:
    matchLabels:
      app: o2-ims
  mtls:
    mode: STRICT

---
# Istio DestinationRule
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: o2-ims-dr
  namespace: o2-system
spec:
  host: o2-ims.o2-system.svc.cluster.local
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
```

## Monitoring & Observability

### Prometheus Configuration

#### 1. ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: o2-ims-metrics
  namespace: o2-system
spec:
  selector:
    matchLabels:
      app: o2-ims
  endpoints:
    - port: metrics
      interval: 30s
      scrapeTimeout: 10s
      path: /metrics
      honorLabels: true
```

#### 2. PrometheusRule

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: o2-ims-alerts
  namespace: o2-system
spec:
  groups:
    - name: o2-ims.rules
      rules:
        - alert: O2IMSInstanceDown
          expr: up{job="o2-ims"} == 0
          for: 1m
          labels:
            severity: critical
          annotations:
            summary: "O2 IMS instance is down"
            description: "O2 IMS instance {{ $labels.instance }} has been down for more than 1 minute"

        - alert: O2IMSHighLatency
          expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{job="o2-ims"}[5m])) > 0.5
          for: 2m
          labels:
            severity: warning
          annotations:
            summary: "O2 IMS high latency"
            description: "95% of requests are taking longer than 500ms"

        - alert: O2IMSHighErrorRate
          expr: rate(http_requests_total{job="o2-ims",code=~"5.."}[5m]) > 0.1
          for: 1m
          labels:
            severity: critical
          annotations:
            summary: "O2 IMS high error rate"
            description: "Error rate is {{ $value | humanizePercentage }}"

        - alert: O2IMSHighMemoryUsage
          expr: container_memory_usage_bytes{pod=~"o2-ims-.*"} / container_spec_memory_limit_bytes > 0.85
          for: 2m
          labels:
            severity: warning
          annotations:
            summary: "O2 IMS high memory usage"
            description: "Memory usage is above 85%"

        - alert: O2IMSDatabaseConnectionHigh
          expr: pg_stat_activity_count{datname="o2ims"} > 40
          for: 2m
          labels:
            severity: warning
          annotations:
            summary: "O2 IMS database connection count high"
            description: "Database connection count is {{ $value }}"
```

### Grafana Dashboard

Create `grafana-dashboard.json`:

```json
{
  "dashboard": {
    "id": null,
    "title": "O2 IMS Production Dashboard",
    "tags": ["o2-ims", "production"],
    "timezone": "browser",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{job=\"o2-ims\"}[5m])",
            "legendFormat": "{{ method }} {{ code }}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket{job=\"o2-ims\"}[5m]))",
            "legendFormat": "50th percentile"
          },
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{job=\"o2-ims\"}[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "container_memory_usage_bytes{pod=~\"o2-ims-.*\"}",
            "legendFormat": "{{ pod }}"
          }
        ]
      },
      {
        "title": "CPU Usage",
        "type": "graph", 
        "targets": [
          {
            "expr": "rate(container_cpu_usage_seconds_total{pod=~\"o2-ims-.*\"}[5m])",
            "legendFormat": "{{ pod }}"
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

## Backup and Disaster Recovery

### Database Backup Strategy

#### 1. Automated Backups

```yaml
# CronJob for database backups
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgresql-backup
  namespace: o2-system
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: postgres-backup
              image: postgres:14
              env:
                - name: PGHOST
                  value: "postgresql-primary"
                - name: PGUSER
                  value: "o2user"
                - name: PGPASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: postgresql-credentials
                      key: password
                - name: PGDATABASE
                  value: "o2ims"
              command:
                - /bin/bash
                - -c
                - |
                  BACKUP_FILE="/backup/o2ims-$(date +%Y%m%d-%H%M%S).sql"
                  pg_dump -h $PGHOST -U $PGUSER -d $PGDATABASE > $BACKUP_FILE
                  gzip $BACKUP_FILE
                  
                  # Upload to S3 or other storage
                  aws s3 cp ${BACKUP_FILE}.gz s3://your-backup-bucket/o2ims/
                  
                  # Keep only last 30 days locally
                  find /backup -name "o2ims-*.sql.gz" -mtime +30 -delete
              volumeMounts:
                - name: backup-storage
                  mountPath: /backup
          volumes:
            - name: backup-storage
              persistentVolumeClaim:
                claimName: backup-pvc
          restartPolicy: OnFailure
```

#### 2. Point-in-Time Recovery Setup

```bash
# PostgreSQL configuration for PITR
# Add to postgresql.conf
archive_mode = on
archive_command = 'test ! -f /backup/archive/%f && cp %p /backup/archive/%f'
wal_level = replica
max_wal_senders = 3
checkpoint_completion_target = 0.9
```

### Application State Backup

#### 1. Configuration Backup

```bash
#!/bin/bash
# backup-config.sh

BACKUP_DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="/backup/config/${BACKUP_DATE}"

mkdir -p $BACKUP_DIR

# Backup ConfigMaps
kubectl get configmaps -n o2-system -o yaml > $BACKUP_DIR/configmaps.yaml

# Backup Secrets (encrypted)
kubectl get secrets -n o2-system -o yaml > $BACKUP_DIR/secrets.yaml

# Backup Custom Resources
kubectl get resourcepools -n o2-system -o yaml > $BACKUP_DIR/resourcepools.yaml
kubectl get resourcetypes -n o2-system -o yaml > $BACKUP_DIR/resourcetypes.yaml

# Backup RBAC
kubectl get roles,rolebindings,clusterroles,clusterrolebindings -o yaml > $BACKUP_DIR/rbac.yaml

# Compress and upload
tar -czf $BACKUP_DIR.tar.gz -C /backup/config $BACKUP_DATE
aws s3 cp $BACKUP_DIR.tar.gz s3://your-backup-bucket/o2ims/config/

# Cleanup
rm -rf $BACKUP_DIR $BACKUP_DIR.tar.gz
```

### Disaster Recovery Procedures

#### 1. Full System Recovery

```bash
#!/bin/bash
# disaster-recovery.sh

set -e

echo "Starting O2 IMS disaster recovery..."

# 1. Restore infrastructure
kubectl create namespace o2-system

# 2. Restore database
kubectl apply -f manifests/postgresql-ha.yaml
kubectl wait --for=condition=ready pod -l app=postgresql -n o2-system --timeout=300s

# 3. Restore database data
LATEST_BACKUP=$(aws s3 ls s3://your-backup-bucket/o2ims/ | sort | tail -n 1 | awk '{print $4}')
aws s3 cp s3://your-backup-bucket/o2ims/$LATEST_BACKUP /tmp/

kubectl exec -i postgresql-primary-0 -n o2-system -- psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS o2ims;"
kubectl exec -i postgresql-primary-0 -n o2-system -- psql -U postgres -d postgres -c "CREATE DATABASE o2ims;"
kubectl exec -i postgresql-primary-0 -n o2-system -- pg_restore -U postgres -d o2ims < /tmp/$LATEST_BACKUP

# 4. Restore application
helm install o2-ims nephoran/o2-ims \
  --namespace o2-system \
  --values production-values.yaml \
  --wait

# 5. Restore configuration
kubectl apply -f backup/config/configmaps.yaml
kubectl apply -f backup/config/secrets.yaml
kubectl apply -f backup/config/resourcepools.yaml
kubectl apply -f backup/config/resourcetypes.yaml

# 6. Verify recovery
kubectl get pods -n o2-system
curl -f http://o2ims.your-domain.com/o2ims/v1/

echo "Disaster recovery completed successfully!"
```

#### 2. Recovery Testing

```bash
#!/bin/bash
# test-recovery.sh

# Regular recovery testing (monthly)
echo "Testing disaster recovery procedures..."

# Create test namespace
kubectl create namespace o2-system-dr-test

# Deploy from backup
# ... recovery steps in test namespace ...

# Verify functionality
kubectl port-forward -n o2-system-dr-test svc/o2-ims 8080:8080 &
sleep 5

# Test API endpoints
curl -f http://localhost:8080/o2ims/v1/ || exit 1
curl -f http://localhost:8080/health || exit 1

# Cleanup
kubectl delete namespace o2-system-dr-test

echo "Recovery test completed successfully!"
```

## Performance Optimization

### Database Optimization

#### 1. PostgreSQL Configuration

```sql
-- postgresql.conf optimizations
shared_buffers = '2GB'                    # 25% of RAM
effective_cache_size = '6GB'              # 75% of RAM
maintenance_work_mem = '512MB'
checkpoint_completion_target = 0.9
wal_buffers = '16MB'
default_statistics_target = 100
random_page_cost = 1.1                    # For SSD storage
effective_io_concurrency = 200            # For SSD storage

-- Connection pooling
max_connections = 200
shared_preload_libraries = 'pg_stat_statements'
```

#### 2. Database Monitoring

```sql
-- Enable query statistics
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Monitor slow queries
SELECT query, calls, total_time, mean_time, rows
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;

-- Monitor connection usage
SELECT count(*) as active_connections,
       state
FROM pg_stat_activity
GROUP BY state;
```

### Application Optimization

#### 1. Connection Pooling

```yaml
# PgBouncer configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: pgbouncer-config
  namespace: o2-system
data:
  pgbouncer.ini: |
    [databases]
    o2ims = host=postgresql-primary port=5432 dbname=o2ims
    
    [pgbouncer]
    listen_port = 6432
    listen_addr = 0.0.0.0
    auth_type = md5
    auth_file = /etc/pgbouncer/userlist.txt
    
    # Connection pooling
    pool_mode = transaction
    max_client_conn = 200
    default_pool_size = 20
    min_pool_size = 5
    reserve_pool_size = 5
    
    # Timeouts
    server_connect_timeout = 15
    server_login_retry = 15
    query_timeout = 0
    query_wait_timeout = 120
```

#### 2. Caching Strategy

```yaml
# Redis cache configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: o2-system
data:
  redis.conf: |
    # Memory optimization
    maxmemory 2gb
    maxmemory-policy allkeys-lru
    
    # Persistence
    save 900 1
    save 300 10
    save 60 10000
    
    # Performance
    tcp-keepalive 60
    timeout 0
```

### Resource Optimization

#### 1. JVM Tuning (if using Java components)

```yaml
env:
  - name: JAVA_OPTS
    value: >-
      -Xms2g
      -Xmx4g
      -XX:+UseG1GC
      -XX:G1HeapRegionSize=16m
      -XX:+UseStringDeduplication
      -XX:+OptimizeStringConcat
      -XX:+UseCompressedOops
      -server
```

#### 2. Go Application Tuning

```yaml
env:
  - name: GOGC
    value: "100"          # GC frequency
  - name: GOMAXPROCS
    value: "4"            # Match container CPU limit
  - name: GOMEMLIMIT
    value: "3GiB"         # 75% of container memory limit
```

## Operational Procedures

### Deployment Procedures

#### 1. Rolling Update

```bash
#!/bin/bash
# rolling-update.sh

set -e

NEW_VERSION="v1.3.0"
NAMESPACE="o2-system"

echo "Starting rolling update to version $NEW_VERSION..."

# Pre-update checks
kubectl get pods -n $NAMESPACE
kubectl get svc -n $NAMESPACE

# Update Helm release
helm upgrade o2-ims nephoran/o2-ims \
  --namespace $NAMESPACE \
  --set image.tag=$NEW_VERSION \
  --values production-values.yaml \
  --wait \
  --timeout 10m

# Verify update
kubectl rollout status deployment/o2-ims -n $NAMESPACE

# Post-update verification
curl -f http://o2ims.your-domain.com/o2ims/v1/ | jq '.version'

echo "Rolling update completed successfully!"
```

#### 2. Blue-Green Deployment

```bash
#!/bin/bash
# blue-green-deployment.sh

set -e

CURRENT_ENV="blue"
NEW_ENV="green"
NEW_VERSION="v1.3.0"

echo "Starting blue-green deployment..."

# Deploy new version to green environment
helm install o2-ims-$NEW_ENV nephoran/o2-ims \
  --namespace o2-system-$NEW_ENV \
  --create-namespace \
  --set image.tag=$NEW_VERSION \
  --values production-values.yaml \
  --wait

# Test green environment
curl -f http://o2ims-$NEW_ENV.your-domain.com/health

# Switch traffic
kubectl patch service ingress-controller -n ingress-system -p \
  '{"spec":{"selector":{"version":"'$NEW_ENV'"}}}'

# Verify traffic switch
sleep 30
curl -f http://o2ims.your-domain.com/o2ims/v1/ | jq '.version'

echo "Blue-green deployment completed successfully!"
echo "Remember to cleanup old environment after validation"
```

### Maintenance Procedures

#### 1. Scheduled Maintenance

```bash
#!/bin/bash
# maintenance-window.sh

set -e

echo "Starting maintenance window..."

# 1. Scale down to single replica for maintenance
kubectl scale deployment o2-ims -n o2-system --replicas=1

# 2. Put service in maintenance mode
kubectl patch configmap o2-ims-config -n o2-system --patch='
{
  "data": {
    "maintenance-mode": "true"
  }
}'

# 3. Restart to pick up maintenance mode
kubectl rollout restart deployment/o2-ims -n o2-system
kubectl rollout status deployment/o2-ims -n o2-system

# 4. Perform maintenance tasks
echo "Performing maintenance tasks..."

# Database maintenance
kubectl exec -it postgresql-primary-0 -n o2-system -- psql -U postgres -d o2ims -c "VACUUM ANALYZE;"
kubectl exec -it postgresql-primary-0 -n o2-system -- psql -U postgres -d o2ims -c "REINDEX DATABASE o2ims;"

# Certificate renewal (if needed)
kubectl get certificate -n o2-system

# Clean up old resources
kubectl delete pod -n o2-system --field-selector=status.phase=Failed
kubectl delete job -n o2-system --field-selector=status.successful=1

# 5. Exit maintenance mode
kubectl patch configmap o2-ims-config -n o2-system --patch='
{
  "data": {
    "maintenance-mode": "false"
  }
}'

# 6. Scale back up
kubectl scale deployment o2-ims -n o2-system --replicas=3
kubectl rollout status deployment/o2-ims -n o2-system

# 7. Verify service health
sleep 30
curl -f http://o2ims.your-domain.com/health

echo "Maintenance window completed successfully!"
```

#### 2. Database Maintenance

```sql
-- Database maintenance queries
-- Run during maintenance window

-- Vacuum and analyze all tables
VACUUM ANALYZE;

-- Reindex all indexes
REINDEX DATABASE o2ims;

-- Update table statistics
ANALYZE;

-- Check for bloated tables
SELECT schemaname, tablename, 
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
       pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Clean up old data (if applicable)
DELETE FROM audit_log WHERE created_at < NOW() - INTERVAL '90 days';
DELETE FROM metrics_data WHERE timestamp < NOW() - INTERVAL '7 days';
```

## Troubleshooting Guide

### Common Issues

#### 1. Service Startup Issues

**Symptoms**: Pods failing to start, CrashLoopBackOff

**Diagnosis**:
```bash
# Check pod status
kubectl get pods -n o2-system

# Check pod logs
kubectl logs -f deployment/o2-ims -n o2-system

# Check pod events
kubectl describe pod <pod-name> -n o2-system

# Check configuration
kubectl get configmap o2-ims-config -n o2-system -o yaml
```

**Solutions**:
1. Verify database connectivity
2. Check configuration values
3. Verify secrets are properly mounted
4. Check resource limits

#### 2. Database Connection Issues

**Symptoms**: Connection timeouts, max connections reached

**Diagnosis**:
```bash
# Check database pod status
kubectl get pods -l app=postgresql -n o2-system

# Check connection count
kubectl exec postgresql-primary-0 -n o2-system -- psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Check active queries
kubectl exec postgresql-primary-0 -n o2-system -- psql -U postgres -c "SELECT pid, state, query FROM pg_stat_activity WHERE state != 'idle';"
```

**Solutions**:
1. Increase max_connections in PostgreSQL
2. Implement connection pooling
3. Check for connection leaks in application
4. Scale up database resources

#### 3. High Latency Issues

**Symptoms**: Slow API responses, timeout errors

**Diagnosis**:
```bash
# Check metrics
curl http://o2ims.your-domain.com/metrics | grep http_request_duration

# Check resource usage
kubectl top pods -n o2-system

# Check database performance
kubectl exec postgresql-primary-0 -n o2-system -- psql -U postgres -c "SELECT query, calls, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

**Solutions**:
1. Optimize database queries
2. Implement caching
3. Scale horizontally
4. Optimize network configuration

### Log Analysis

#### 1. Centralized Logging

```bash
# If using ELK stack
# Search for errors in last hour
curl -X GET "elasticsearch:9200/logstash-*/_search" -H 'Content-Type: application/json' -d'
{
  "query": {
    "bool": {
      "must": [
        {"match": {"kubernetes.namespace": "o2-system"}},
        {"match": {"level": "error"}},
        {"range": {"@timestamp": {"gte": "now-1h"}}}
      ]
    }
  },
  "sort": [{"@timestamp": {"order": "desc"}}],
  "size": 100
}'
```

#### 2. Log Aggregation Queries

```bash
# Count errors by type
kubectl logs deployment/o2-ims -n o2-system --since=1h | \
  jq -r 'select(.level=="error") | .error' | \
  sort | uniq -c | sort -nr

# Extract performance metrics from logs
kubectl logs deployment/o2-ims -n o2-system --since=1h | \
  jq -r 'select(.duration_ms) | "\(.timestamp) \(.method) \(.path) \(.duration_ms)ms"' | \
  awk '$4 > 1000'  # Requests taking more than 1 second
```

### Performance Debugging

#### 1. Memory Profiling

```bash
# Enable pprof endpoint (development only)
kubectl port-forward deployment/o2-ims -n o2-system 6060:6060

# Generate memory profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Generate CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

#### 2. Database Query Analysis

```sql
-- Find slow queries
SELECT query, calls, total_time, mean_time, min_time, max_time
FROM pg_stat_statements
WHERE calls > 100
ORDER BY mean_time DESC
LIMIT 20;

-- Find queries with high I/O
SELECT query, calls, shared_blks_hit, shared_blks_read,
       (shared_blks_read * 100.0 / (shared_blks_hit + shared_blks_read)) AS cache_miss_ratio
FROM pg_stat_statements
WHERE shared_blks_read > 0
ORDER BY shared_blks_read DESC
LIMIT 20;
```

## Conclusion

This production deployment guide provides comprehensive coverage of deploying and operating the O2 IMS service in production environments. Regular review and updates of these procedures ensure continued operational excellence and system reliability.

For additional support and updates, refer to:
- [API Documentation](../../api/index.md)
- [Operations Guide](../operations/)
- [Troubleshooting Guide](../operations/troubleshooting-guide.md)
- [GitHub Repository](https://github.com/nephoran/nephoran-intent-operator)
