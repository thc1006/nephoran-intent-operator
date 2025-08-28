# Conductor Loop - Operations Guide

This guide covers production deployment, monitoring, and operational procedures for the conductor-loop component.

## Table of Contents

- [Production Deployment](#production-deployment)
- [Monitoring and Alerting](#monitoring-and-alerting)
- [Performance Tuning](#performance-tuning)
- [Backup and Recovery](#backup-and-recovery)
- [Scaling Strategies](#scaling-strategies)
- [Security Hardening](#security-hardening)
- [Incident Response](#incident-response)
- [Maintenance Procedures](#maintenance-procedures)

## Production Deployment

### Infrastructure Requirements

**Minimum System Requirements**:
- **CPU**: 2 cores (4 cores recommended)
- **Memory**: 1GB RAM (2GB recommended)
- **Storage**: 20GB (SSD preferred)
- **Network**: 1Gbps (for high-throughput scenarios)

**Operating System Support**:
- Linux (Ubuntu 20.04+, RHEL 8+, CentOS 8+)
- Windows Server 2019+
- Kubernetes 1.24+

**Dependencies**:
- Porch CLI tool
- Persistent storage for handoff directory
- Network access to Porch package repository

### Deployment Architectures

#### 1. Standalone Deployment

**Single Instance Setup**:
```bash
# Create service user
sudo useradd -r -s /bin/false conductor-loop

# Create directories
sudo mkdir -p /opt/conductor-loop/{bin,data,logs,config}
sudo chown -R conductor-loop:conductor-loop /opt/conductor-loop

# Install binary
sudo cp conductor-loop /opt/conductor-loop/bin/
sudo chmod +x /opt/conductor-loop/bin/conductor-loop

# Create systemd service
sudo cp conductor-loop.service /etc/systemd/system/
sudo systemctl enable conductor-loop
sudo systemctl start conductor-loop
```

**Systemd Service Configuration**:
```ini
# /etc/systemd/system/conductor-loop.service
[Unit]
Description=Conductor Loop Service
After=network.target
Wants=network.target

[Service]
Type=simple
User=conductor-loop
Group=conductor-loop
WorkingDirectory=/opt/conductor-loop
ExecStart=/opt/conductor-loop/bin/conductor-loop \
    -handoff /opt/conductor-loop/data/handoff \
    -out /opt/conductor-loop/data/output \
    -porch /usr/local/bin/porch \
    -mode structured

# Restart configuration
Restart=always
RestartSec=10
StartLimitInterval=60
StartLimitBurst=3

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/conductor-loop/data

# Environment
Environment=CONDUCTOR_LOG_LEVEL=info
Environment=CONDUCTOR_METRICS_ENABLED=true

[Install]
WantedBy=multi-user.target
```

#### 2. High Availability Deployment

**Load Balancer Configuration**:
```nginx
# nginx.conf
upstream conductor-loop {
    least_conn;
    server conductor-1.example.com:8080 max_fails=3 fail_timeout=30s;
    server conductor-2.example.com:8080 max_fails=3 fail_timeout=30s;
    server conductor-3.example.com:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name conductor-loop.example.com;
    
    location /health {
        proxy_pass http://conductor-loop;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_connect_timeout 5s;
        proxy_read_timeout 10s;
    }
    
    location /upload {
        client_max_body_size 10M;
        proxy_pass http://conductor-loop;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

**Shared Storage Setup**:
```bash
# NFS shared storage for handoff directory
# Server side
sudo apt-get install nfs-kernel-server
echo "/shared/conductor-loop *(rw,sync,no_subtree_check)" >> /etc/exports
sudo exportfs -a
sudo systemctl restart nfs-kernel-server

# Client side (on each conductor-loop instance)
sudo apt-get install nfs-common
sudo mkdir -p /opt/conductor-loop/data/handoff
sudo mount -t nfs nfs-server:/shared/conductor-loop /opt/conductor-loop/data/handoff
```

#### 3. Kubernetes Deployment

**Complete Kubernetes Configuration**:
```yaml
# conductor-loop-namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: conductor-loop
  labels:
    name: conductor-loop
    istio-injection: enabled
---
# conductor-loop-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: conductor-loop-config
  namespace: conductor-loop
data:
  config.json: |
    {
      "porch_path": "/usr/local/bin/porch",
      "mode": "structured",
      "max_workers": 4,
      "debounce_duration": "500ms",
      "cleanup_after": "168h",
      "log_level": "info",
      "metrics_enabled": true
    }
---
# conductor-loop-pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: conductor-loop-data
  namespace: conductor-loop
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: nfs-client
  resources:
    requests:
      storage: 50Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: conductor-loop-logs
  namespace: conductor-loop
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: fast-ssd
  resources:
    requests:
      storage: 10Gi
```

**Deployment with Rolling Updates**:
```yaml
# conductor-loop-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: conductor-loop
  namespace: conductor-loop
  labels:
    app.kubernetes.io/name: conductor-loop
    app.kubernetes.io/version: "1.0.0"
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app.kubernetes.io/name: conductor-loop
  template:
    metadata:
      labels:
        app.kubernetes.io/name: conductor-loop
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: conductor-loop
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: conductor-loop
        image: ghcr.io/thc1006/conductor-loop:latest
        imagePullPolicy: IfNotPresent
        args:
          - "--config=/config/config.json"
          - "--handoff=/data/handoff"
          - "--out=/data/output"
        ports:
        - name: http-health
          containerPort: 8080
          protocol: TCP
        - name: http-metrics
          containerPort: 9090
          protocol: TCP
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONDUCTOR_LOG_LEVEL
          value: "info"
        - name: CONDUCTOR_METRICS_ENABLED
          value: "true"
        resources:
          limits:
            cpu: "2000m"
            memory: "2Gi"
            ephemeral-storage: "5Gi"
          requests:
            cpu: "500m"
            memory: "512Mi"
            ephemeral-storage: "1Gi"
        volumeMounts:
        - name: config-volume
          mountPath: /config
          readOnly: true
        - name: data-volume
          mountPath: /data
        - name: logs-volume
          mountPath: /logs
        livenessProbe:
          httpGet:
            path: /livez
            port: http-health
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: http-health
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        startupProbe:
          httpGet:
            path: /healthz
            port: http-health
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 5
          failureThreshold: 12
      volumes:
      - name: config-volume
        configMap:
          name: conductor-loop-config
      - name: data-volume
        persistentVolumeClaim:
          claimName: conductor-loop-data
      - name: logs-volume
        persistentVolumeClaim:
          claimName: conductor-loop-logs
```

**Service and Ingress**:
```yaml
# conductor-loop-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: conductor-loop
  namespace: conductor-loop
  labels:
    app.kubernetes.io/name: conductor-loop
spec:
  type: ClusterIP
  ports:
  - name: http-health
    port: 8080
    targetPort: http-health
    protocol: TCP
  - name: http-metrics
    port: 9090
    targetPort: http-metrics
    protocol: TCP
  selector:
    app.kubernetes.io/name: conductor-loop
---
# conductor-loop-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: conductor-loop
  namespace: conductor-loop
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - conductor-loop.example.com
    secretName: conductor-loop-tls
  rules:
  - host: conductor-loop.example.com
    http:
      paths:
      - path: /health
        pathType: Prefix
        backend:
          service:
            name: conductor-loop
            port:
              number: 8080
```

## Monitoring and Alerting

### Metrics Collection

**Prometheus Configuration**:
```yaml
# prometheus-conductor-loop.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: conductor-loop
  namespace: conductor-loop
  labels:
    app.kubernetes.io/name: conductor-loop
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: conductor-loop
  endpoints:
  - port: http-metrics
    interval: 30s
    path: /metrics
    scrapeTimeout: 10s
```

**Key Metrics**:
```yaml
# Conductor Loop specific metrics
conductor_files_processed_total{status="success|failed"}
conductor_files_processing_duration_seconds_bucket
conductor_active_workers
conductor_queue_size
conductor_state_entries_total
conductor_porch_execution_duration_seconds
conductor_porch_timeout_total
conductor_file_manager_operations_total{operation="move_processed|move_failed"}

# System metrics
process_cpu_seconds_total
process_memory_bytes
process_open_fds
go_memstats_heap_inuse_bytes
go_goroutines
```

**Grafana Dashboard Configuration**:
```json
{
  "dashboard": {
    "title": "Conductor Loop Operations",
    "panels": [
      {
        "title": "File Processing Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(conductor_files_processed_total[5m])",
            "legendFormat": "{{status}} files/sec"
          }
        ]
      },
      {
        "title": "Processing Duration",
        "type": "heatmap",
        "targets": [
          {
            "expr": "conductor_files_processing_duration_seconds_bucket",
            "format": "heatmap"
          }
        ]
      },
      {
        "title": "Active Workers vs Queue Size",
        "type": "graph",
        "targets": [
          {
            "expr": "conductor_active_workers",
            "legendFormat": "Active Workers"
          },
          {
            "expr": "conductor_queue_size",
            "legendFormat": "Queue Size"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "process_memory_bytes / 1024 / 1024",
            "legendFormat": "Memory MB"
          }
        ]
      }
    ]
  }
}
```

### Alerting Rules

**Prometheus Alerting Rules**:
```yaml
# conductor-loop-alerts.yaml
groups:
- name: conductor-loop.rules
  rules:
  # High failure rate
  - alert: ConductorLoopHighFailureRate
    expr: |
      (
        rate(conductor_files_processed_total{status="failed"}[5m]) /
        rate(conductor_files_processed_total[5m])
      ) > 0.1
    for: 2m
    labels:
      severity: warning
      component: conductor-loop
    annotations:
      summary: "High failure rate in conductor-loop"
      description: "Conductor loop has a failure rate of {{ $value | humanizePercentage }} over the last 5 minutes"
      runbook_url: "https://docs.example.com/runbooks/conductor-loop-failures"

  # Processing queue backed up
  - alert: ConductorLoopQueueBackup
    expr: conductor_queue_size > 50
    for: 5m
    labels:
      severity: warning
      component: conductor-loop
    annotations:
      summary: "Conductor loop queue is backed up"
      description: "Queue size is {{ $value }} files, processing may be delayed"

  # No files processed recently
  - alert: ConductorLoopNoActivity
    expr: |
      increase(conductor_files_processed_total[10m]) == 0 and
      conductor_queue_size > 0
    for: 10m
    labels:
      severity: critical
      component: conductor-loop
    annotations:
      summary: "Conductor loop has stopped processing files"
      description: "No files have been processed in the last 10 minutes despite queue not being empty"

  # High memory usage
  - alert: ConductorLoopHighMemoryUsage
    expr: process_memory_bytes / 1024 / 1024 > 1024
    for: 5m
    labels:
      severity: warning
      component: conductor-loop
    annotations:
      summary: "Conductor loop high memory usage"
      description: "Memory usage is {{ $value }}MB, may indicate a memory leak"

  # Porch execution timeouts
  - alert: ConductorLoopPorchTimeouts
    expr: rate(conductor_porch_timeout_total[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
      component: conductor-loop
    annotations:
      summary: "High rate of Porch execution timeouts"
      description: "Porch executions are timing out at {{ $value }} per second"

  # Service down
  - alert: ConductorLoopDown
    expr: up{job="conductor-loop"} == 0
    for: 1m
    labels:
      severity: critical
      component: conductor-loop
    annotations:
      summary: "Conductor loop service is down"
      description: "Conductor loop instance {{ $labels.instance }} is not responding"
```

### Log Management

**Structured Logging Configuration**:
```json
{
  "level": "info",
  "format": "json",
  "timestamp": true,
  "fields": {
    "service": "conductor-loop",
    "version": "1.0.0",
    "environment": "production"
  }
}
```

**Log Aggregation with Fluentd**:
```yaml
# fluentd-conductor-loop.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-conductor-loop
  namespace: conductor-loop
data:
  fluent.conf: |
    <source>
      @type tail
      path /logs/conductor-loop.log
      pos_file /tmp/conductor-loop.log.pos
      tag conductor-loop.application
      format json
      time_format %Y-%m-%dT%H:%M:%S.%NZ
    </source>
    
    <filter conductor-loop.**>
      @type parser
      key_name message
      reserve_data true
      <parse>
        @type json
      </parse>
    </filter>
    
    <match conductor-loop.**>
      @type elasticsearch
      host elasticsearch.logging.svc.cluster.local
      port 9200
      index_name conductor-loop
      type_name _doc
      include_tag_key true
      tag_key @log_name
    </match>
```

**Log Analysis Queries**:
```bash
# Elasticsearch queries for common issues

# High error rate analysis
GET conductor-loop/_search
{
  "query": {
    "bool": {
      "must": [
        {"range": {"@timestamp": {"gte": "now-1h"}}},
        {"match": {"level": "error"}}
      ]
    }
  },
  "aggs": {
    "error_types": {
      "terms": {"field": "error_type.keyword"}
    }
  }
}

# Processing time analysis
GET conductor-loop/_search
{
  "query": {
    "bool": {
      "must": [
        {"range": {"@timestamp": {"gte": "now-24h"}}},
        {"exists": {"field": "processing_duration"}}
      ]
    }
  },
  "aggs": {
    "duration_stats": {
      "stats": {"field": "processing_duration"}
    }
  }
}
```

## Performance Tuning

### Resource Optimization

**CPU Optimization**:
```bash
# Set CPU affinity for conductor-loop process
taskset -c 0,1 ./conductor-loop

# Enable CPU performance governor
echo performance > /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Optimize Go runtime
export GOMAXPROCS=4
export GOGC=100
```

**Memory Optimization**:
```bash
# Tune Go garbage collector
export GOGC=200  # Less frequent GC for throughput
export GOMEMLIMIT=1536MiB  # Set memory limit

# System memory tuning
echo 'vm.swappiness=10' >> /etc/sysctl.conf
echo 'vm.dirty_ratio=15' >> /etc/sysctl.conf
echo 'vm.dirty_background_ratio=5' >> /etc/sysctl.conf
sysctl -p
```

**Storage Optimization**:
```bash
# SSD optimizations
echo noop > /sys/block/sda/queue/scheduler
echo 1 > /sys/block/sda/queue/iosched/fifo_batch

# Mount options for performance
mount -o noatime,nodiratime,data=writeback /dev/sda1 /opt/conductor-loop/data
```

### Application Tuning

**Worker Pool Configuration**:
```go
// Optimal worker count based on workload
func calculateOptimalWorkers() int {
    cpuCount := runtime.NumCPU()
    
    // For I/O bound work (porch execution)
    ioWorkers := cpuCount * 2
    
    // For CPU bound work
    cpuWorkers := cpuCount
    
    // Mixed workload heuristic
    return cpuCount + (cpuCount / 2)
}
```

**Debounce Tuning**:
```yaml
# Configuration for different environments
development:
  debounce_duration: "1s"    # Slower for development
  
staging:
  debounce_duration: "500ms" # Default
  
production:
  debounce_duration: "200ms" # Faster for high throughput
  
high_throughput:
  debounce_duration: "100ms" # Minimal delay
```

**Batch Processing Configuration**:
```yaml
# Batch processing for high-volume scenarios
batch_processing:
  enabled: true
  batch_size: 10
  batch_timeout: "5s"
  max_concurrent_batches: 3
```

### Capacity Planning

**Throughput Calculations**:
```bash
# Baseline measurements
# Average porch execution time: 2s
# Worker pool size: 4
# Theoretical max throughput: 4 workers / 2s = 2 files/second = 7200 files/hour

# Account for overhead (file I/O, state management, etc.)
# Practical throughput: ~70% of theoretical = 5000 files/hour

# High-performance configuration
# Optimized porch execution: 1s
# Increased workers: 8
# SSD storage, optimized state management
# Expected throughput: 6-7 files/second = 20000+ files/hour
```

**Scaling Thresholds**:
```yaml
scaling_thresholds:
  # Scale up triggers
  queue_size_high: 20
  processing_time_p95_high: "10s"
  cpu_utilization_high: 80
  memory_utilization_high: 70
  
  # Scale down triggers  
  queue_size_low: 2
  processing_time_p95_low: "2s"
  cpu_utilization_low: 20
  memory_utilization_low: 30
  
  # Cooldown periods
  scale_up_cooldown: "2m"
  scale_down_cooldown: "5m"
```

## Backup and Recovery

### Data Backup Strategy

**Critical Data Components**:
1. **State File** (`.conductor-state.json`)
2. **Configuration Files**
3. **Intent Files** (input, processed, failed)
4. **Status Files**
5. **Application Logs**

**Backup Script**:
```bash
#!/bin/bash
# conductor-loop-backup.sh

set -euo pipefail

BACKUP_DIR="/backup/conductor-loop"
DATA_DIR="/opt/conductor-loop/data"
CONFIG_DIR="/opt/conductor-loop/config"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="conductor-loop_${TIMESTAMP}"

# Create backup directory
mkdir -p "${BACKUP_DIR}/${BACKUP_NAME}"

# Backup state file (most critical)
cp "${DATA_DIR}/handoff/.conductor-state.json" "${BACKUP_DIR}/${BACKUP_NAME}/" 2>/dev/null || true

# Backup configuration
cp -r "${CONFIG_DIR}" "${BACKUP_DIR}/${BACKUP_NAME}/"

# Backup unprocessed intent files
find "${DATA_DIR}/handoff" -name "intent-*.json" -exec cp {} "${BACKUP_DIR}/${BACKUP_NAME}/" \;

# Backup recent status files (last 24 hours)
find "${DATA_DIR}/handoff/status" -name "*.status" -mtime -1 -exec cp {} "${BACKUP_DIR}/${BACKUP_NAME}/" \;

# Create manifest
cat > "${BACKUP_DIR}/${BACKUP_NAME}/MANIFEST" << EOF
Backup Created: $(date)
Hostname: $(hostname)
Version: $(conductor-loop --version 2>/dev/null || echo "unknown")
Data Directory: ${DATA_DIR}
Config Directory: ${CONFIG_DIR}
EOF

# Compress backup
tar -czf "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" -C "${BACKUP_DIR}" "${BACKUP_NAME}"
rm -rf "${BACKUP_DIR}/${BACKUP_NAME}"

# Cleanup old backups (keep last 30 days)
find "${BACKUP_DIR}" -name "conductor-loop_*.tar.gz" -mtime +30 -delete

echo "Backup completed: ${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"
```

**Automated Backup with Cron**:
```bash
# Add to crontab
0 2 * * * /opt/conductor-loop/scripts/conductor-loop-backup.sh

# Verify backups
0 3 * * * /opt/conductor-loop/scripts/verify-backup.sh
```

### Disaster Recovery Procedures

**Recovery Priority Matrix**:
```
Priority 1 (RTO: 15 minutes):
- Restore service functionality
- Recover state file
- Resume intent processing

Priority 2 (RTO: 1 hour):
- Restore all unprocessed intents
- Recover processing history
- Restore monitoring/metrics

Priority 3 (RTO: 4 hours):
- Complete audit trail restoration
- Historical data recovery
- Performance optimization
```

**Recovery Procedures**:

#### 1. State File Corruption Recovery
```bash
#!/bin/bash
# recover-state.sh

# Stop service
systemctl stop conductor-loop

# Backup corrupted state
cp /opt/conductor-loop/data/handoff/.conductor-state.json \
   /tmp/corrupted-state-$(date +%s).json

# Restore from backup
LATEST_BACKUP=$(ls -t /backup/conductor-loop/conductor-loop_*.tar.gz | head -1)
tar -xzf "$LATEST_BACKUP" -C /tmp

# Extract state file
cp /tmp/conductor-loop_*/. conductor-state.json \
   /opt/conductor-loop/data/handoff/

# Start service
systemctl start conductor-loop

# Verify recovery
systemctl status conductor-loop
tail -f /opt/conductor-loop/logs/conductor-loop.log
```

#### 2. Complete Service Recovery
```bash
#!/bin/bash
# full-recovery.sh

# 1. Stop all related services
systemctl stop conductor-loop
systemctl stop nginx  # if using load balancer

# 2. Restore data from backup
LATEST_BACKUP=$(ls -t /backup/conductor-loop/conductor-loop_*.tar.gz | head -1)
echo "Restoring from: $LATEST_BACKUP"

# Create temporary restore directory
RESTORE_DIR="/tmp/conductor-restore-$(date +%s)"
mkdir -p "$RESTORE_DIR"
tar -xzf "$LATEST_BACKUP" -C "$RESTORE_DIR"

# 3. Restore state file
cp "$RESTORE_DIR"/conductor-loop_*/.conductor-state.json \
   /opt/conductor-loop/data/handoff/

# 4. Restore unprocessed intents
find "$RESTORE_DIR" -name "intent-*.json" -exec cp {} /opt/conductor-loop/data/handoff/ \;

# 5. Restore configuration
cp -r "$RESTORE_DIR"/conductor-loop_*/config/* /opt/conductor-loop/config/

# 6. Set proper permissions
chown -R conductor-loop:conductor-loop /opt/conductor-loop/data
chmod 755 /opt/conductor-loop/data/handoff

# 7. Start services
systemctl start conductor-loop
systemctl start nginx

# 8. Verify recovery
sleep 10
systemctl is-active conductor-loop
curl -f http://localhost:8080/healthz

echo "Recovery completed successfully"
```

#### 3. Kubernetes Recovery
```bash
#!/bin/bash
# k8s-recovery.sh

# Scale down deployment
kubectl scale deployment conductor-loop --replicas=0 -n conductor-loop

# Restore PVC data from backup
kubectl exec -n conductor-loop backup-pod -- \
  tar -xzf /backup/latest.tar.gz -C /data/handoff/

# Scale up deployment
kubectl scale deployment conductor-loop --replicas=3 -n conductor-loop

# Verify pods are ready
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=conductor-loop \
  -n conductor-loop --timeout=300s

# Check service health
kubectl port-forward -n conductor-loop service/conductor-loop 8080:8080 &
curl -f http://localhost:8080/healthz
```

### Backup Verification

**Automated Verification Script**:
```bash
#!/bin/bash
# verify-backup.sh

BACKUP_DIR="/backup/conductor-loop"
LATEST_BACKUP=$(ls -t "$BACKUP_DIR"/conductor-loop_*.tar.gz | head -1)

if [[ -z "$LATEST_BACKUP" ]]; then
    echo "ERROR: No backup files found"
    exit 1
fi

# Extract to temporary location
TEMP_DIR=$(mktemp -d)
tar -xzf "$LATEST_BACKUP" -C "$TEMP_DIR"

# Verify state file exists and is valid JSON
STATE_FILE=$(find "$TEMP_DIR" -name ".conductor-state.json")
if [[ -n "$STATE_FILE" ]]; then
    if jq empty "$STATE_FILE" 2>/dev/null; then
        echo "✓ State file is valid JSON"
    else
        echo "✗ State file is corrupted"
        exit 1
    fi
else
    echo "✗ State file not found in backup"
    exit 1
fi

# Verify manifest exists
MANIFEST=$(find "$TEMP_DIR" -name "MANIFEST")
if [[ -n "$MANIFEST" ]]; then
    echo "✓ Backup manifest found"
    cat "$MANIFEST"
else
    echo "⚠ No backup manifest found"
fi

# Cleanup
rm -rf "$TEMP_DIR"

echo "Backup verification completed successfully"
```

## Scaling Strategies

### Horizontal Scaling

#### Multi-Instance Deployment
```yaml
# Horizontal scaling configuration
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: conductor-loop-hpa
  namespace: conductor-loop
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: conductor-loop
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
  - type: Pods
    pods:
      metric:
        name: conductor_queue_size
      target:
        type: AverageValue
        averageValue: "10"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 120
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
```

#### Directory Sharding Strategy
```bash
# Shard intents across multiple instances
# Instance 1: Handles intent files with hash % 3 == 0
# Instance 2: Handles intent files with hash % 3 == 1  
# Instance 3: Handles intent files with hash % 3 == 2

# File distributor script
#!/bin/bash
# distribute-intents.sh

for intent_file in /shared/intent-queue/intent-*.json; do
    filename=$(basename "$intent_file")
    hash=$(echo "$filename" | sha256sum | cut -c1-8)
    shard=$((0x$hash % 3))
    
    mv "$intent_file" "/shared/shard-$shard/"
done
```

### Vertical Scaling

#### Resource Scaling Guidelines
```yaml
# Small deployment (< 100 files/hour)
resources:
  requests:
    cpu: "200m"
    memory: "256Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"

# Medium deployment (100-1000 files/hour)  
resources:
  requests:
    cpu: "500m"
    memory: "512Mi"
  limits:
    cpu: "1000m"
    memory: "1Gi"

# Large deployment (> 1000 files/hour)
resources:
  requests:
    cpu: "1000m"
    memory: "1Gi"
  limits:
    cpu: "2000m"
    memory: "2Gi"
```

### Edge Deployment Scaling

#### Multi-Region Architecture
```yaml
# Edge deployment configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: conductor-loop-edge-config
data:
  config.json: |
    {
      "mode": "edge",
      "upstream_sync": {
        "enabled": true,
        "endpoint": "https://central.conductor.example.com",
        "sync_interval": "30s",
        "batch_size": 50
      },
      "local_processing": {
        "enabled": true,
        "cache_ttl": "5m",
        "offline_mode": true
      }
    }
```

## Security Operations

### Security Monitoring and Alerting

#### Security Event Monitoring

**Real-Time Security Alerts**:
```yaml
# Security-focused Prometheus alerts
groups:
- name: conductor-loop-security
  rules:
  - alert: SecurityIncidentDetected
    expr: increase(conductor_security_events_total{severity="high"}[5m]) > 0
    for: 0m
    labels:
      severity: critical
      component: conductor-loop
    annotations:
      summary: "High-severity security event detected"
      description: "Security event of type {{ $labels.event_type }} detected"
      runbook_url: "https://docs.company.com/runbooks/security-incident-response"
      
  - alert: AnomalousFileProcessing
    expr: rate(conductor_files_processed_total[5m]) > (avg_over_time(rate(conductor_files_processed_total[5m])[1h]) * 3)
    for: 2m
    labels:
      severity: warning
      component: conductor-loop
    annotations:
      summary: "Anomalous file processing rate detected"
      description: "File processing rate is {{ $value }} times higher than normal"
      
  - alert: CommandInjectionAttempt
    expr: increase(conductor_command_injection_attempts_total[1m]) > 0
    for: 0m
    labels:
      severity: critical
      component: conductor-loop
    annotations:
      summary: "Command injection attempt detected"
      description: "Potential command injection attack detected"
      
  - alert: PathTraversalAttempt
    expr: increase(conductor_path_traversal_attempts_total[1m]) > 0
    for: 0m
    labels:
      severity: high
      component: conductor-loop
    annotations:
      summary: "Path traversal attempt detected"
      description: "Potential path traversal attack detected"
```

**Security Dashboard Configuration**:
```json
{
  "dashboard": {
    "title": "Conductor Loop Security Operations",
    "panels": [
      {
        "title": "Security Events Timeline",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(conductor_security_events_total[5m])",
            "legendFormat": "{{event_type}} - {{severity}}"
          }
        ]
      },
      {
        "title": "Failed Validation Attempts",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(rate(conductor_validation_failures_total[5m]))"
          }
        ]
      },
      {
        "title": "Resource Consumption",
        "type": "graph",
        "targets": [
          {
            "expr": "process_memory_bytes",
            "legendFormat": "Memory Usage"
          },
          {
            "expr": "rate(process_cpu_seconds_total[5m])",
            "legendFormat": "CPU Usage"
          }
        ]
      }
    ]
  }
}
```

#### Threat Detection

**Behavioral Analysis Engine**:
```bash
#!/bin/bash
# security-analysis.sh - Automated threat detection

# Analyze file processing patterns
echo "Analyzing file processing patterns..."
kubectl exec -n conductor-loop conductor-loop-0 -- \
  /usr/local/bin/analyze-patterns --window=1h --threshold=3

# Check for suspicious file names
echo "Checking for suspicious file patterns..."
kubectl exec -n conductor-loop conductor-loop-0 -- \
  find /data/handoff -name "*.json" -exec grep -l "\.\./" {} \;

# Monitor resource consumption
echo "Monitoring resource consumption..."
kubectl top pods -n conductor-loop --use-protocol-buffers

# Check for unusual network activity
echo "Checking network connections..."
kubectl exec -n conductor-loop conductor-loop-0 -- \
  netstat -tuln | grep -v "0.0.0.0:8080\|0.0.0.0:9090"
```

**Anomaly Detection Rules**:
```yaml
# anomaly-detection.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: anomaly-detection-rules
data:
  rules.yml: |
    rules:
      - name: unusual_file_size
        description: "Detect unusually large intent files"
        threshold: 
          metric: file_size_bytes
          operator: ">"
          value: 10485760  # 10MB
        action: quarantine
        
      - name: rapid_file_creation
        description: "Detect rapid file creation patterns"
        threshold:
          metric: files_created_per_minute
          operator: ">"
          value: 100
        action: throttle
        
      - name: processing_time_anomaly
        description: "Detect unusually long processing times"
        threshold:
          metric: processing_duration_seconds
          operator: ">"
          value: 300  # 5 minutes
        action: investigate
```

#### Security Incident Response

**Incident Response Playbooks**:
```bash
#!/bin/bash
# incident-response.sh

INCIDENT_TYPE="$1"
SEVERITY="$2"

case "$INCIDENT_TYPE" in
  "command_injection")
    echo "=== COMMAND INJECTION INCIDENT RESPONSE ==="
    
    # Immediate containment
    kubectl scale deployment conductor-loop --replicas=0
    
    # Quarantine files
    kubectl exec conductor-loop-0 -- \
      mv /data/handoff/*.json /data/quarantine/
    
    # Collect evidence
    kubectl exec conductor-loop-0 -- \
      tar -czf /tmp/evidence-$(date +%s).tar.gz /data/handoff /var/log/conductor-loop
    
    # Alert security team
    curl -X POST "https://security-alerts.company.com/api/incidents" \
      -H "Content-Type: application/json" \
      -d "{\"type\":\"command_injection\",\"severity\":\"$SEVERITY\",\"component\":\"conductor-loop\"}"
    ;;
    
  "resource_exhaustion")
    echo "=== RESOURCE EXHAUSTION INCIDENT RESPONSE ==="
    
    # Reduce resource allocation
    kubectl patch deployment conductor-loop -p \
      '{"spec":{"template":{"spec":{"containers":[{"name":"conductor-loop","resources":{"limits":{"memory":"256Mi","cpu":"200m"}}}]}}}}'
    
    # Clear processing queue
    kubectl exec conductor-loop-0 -- \
      rm -f /data/handoff/intent-*.json
    
    # Restart with limited resources
    kubectl rollout restart deployment/conductor-loop
    ;;
    
  "data_breach")
    echo "=== DATA BREACH INCIDENT RESPONSE ==="
    
    # Immediate shutdown
    kubectl delete deployment conductor-loop
    
    # Isolate data
    kubectl exec -n conductor-loop pvc-backup-pod -- \
      tar -czf /backup/breach-isolation-$(date +%s).tar.gz /data/handoff
    
    # Enable forensic mode
    kubectl apply -f forensic-mode-deployment.yaml
    ;;
esac
```

### Security Hardening

#### Enhanced Authentication and Authorization

#### RBAC Configuration
```yaml
# conductor-loop-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: conductor-loop
  namespace: conductor-loop
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: conductor-loop
  name: conductor-loop-role
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: conductor-loop-binding
  namespace: conductor-loop
subjects:
- kind: ServiceAccount
  name: conductor-loop
  namespace: conductor-loop
roleRef:
  kind: Role
  name: conductor-loop-role
  apiGroup: rbac.authorization.k8s.io
```

#### Network Security
```yaml
# NetworkPolicy for conductor-loop
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: conductor-loop-netpol
  namespace: conductor-loop
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: conductor-loop
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 9090  # metrics
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080  # health checks
  egress:
  - to: []  # Allow all egress for porch execution
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 80
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
```

### Secret Management

#### Sealed Secrets Configuration
```yaml
# conductor-loop-sealed-secret.yaml
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: conductor-loop-secrets
  namespace: conductor-loop
spec:
  encryptedData:
    porch-token: AgBy3i4OJSWK+PiTySYZZA9rO43cGDEQAx...
    database-url: AgAKAoiQm+/LMxJmjmNKgXUs8nxIkdNy...
  template:
    metadata:
      name: conductor-loop-secrets
      namespace: conductor-loop
    type: Opaque
```

#### Vault Integration
```bash
# Vault integration for secret management
vault kv put secret/conductor-loop/prod \
  porch-token="$(openssl rand -base64 32)" \
  database-url="postgresql://user:pass@db:5432/conductor"

# Init container to fetch secrets
vault agent -config=/vault/config/agent.hcl
```

### Container Security

#### Security Context Configuration
```yaml
# Enhanced security context
securityContext:
  runAsNonRoot: true
  runAsUser: 65534  # nobody user
  runAsGroup: 65534
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
  seccompProfile:
    type: RuntimeDefault
```

#### Image Security Scanning
```bash
# Trivy security scanning
trivy image ghcr.io/thc1006/conductor-loop:latest

# Snyk scanning
snyk container test ghcr.io/thc1006/conductor-loop:latest

# OPA Gatekeeper policies
kubectl apply -f security-policies/
```

## Incident Response

### Incident Classification

#### Severity Levels
```yaml
severity_levels:
  P0_Critical:
    description: "Complete service outage"
    response_time: "15 minutes"
    examples:
      - All conductor-loop instances down
      - Data corruption detected
      - Security breach confirmed
      
  P1_High:
    description: "Major functionality impaired"
    response_time: "1 hour"
    examples:
      - High failure rate (>50%)
      - Performance severely degraded
      - Single-point-of-failure component down
      
  P2_Medium:
    description: "Partial functionality impaired"
    response_time: "4 hours"
    examples:
      - Processing delays
      - Non-critical errors increasing
      - Monitoring alerts firing
      
  P3_Low:
    description: "Minor issues, no user impact"
    response_time: "24 hours"
    examples:
      - Log warnings
      - Performance trending down
      - Documentation needs updates
```

### Incident Response Playbooks

#### P0: Complete Service Outage
```bash
#!/bin/bash
# p0-outage-response.sh

echo "=== P0 INCIDENT: CONDUCTOR-LOOP OUTAGE ==="
echo "Incident start time: $(date)"

# 1. Immediate assessment
echo "1. Checking service status..."
kubectl get pods -n conductor-loop -o wide
kubectl get events -n conductor-loop --sort-by=.metadata.creationTimestamp

# 2. Check resource availability
echo "2. Checking cluster resources..."
kubectl top nodes
kubectl describe nodes | grep -A 5 "Allocated resources"

# 3. Check for obvious issues
echo "3. Checking for common issues..."
kubectl logs -n conductor-loop -l app.kubernetes.io/name=conductor-loop --tail=100

# 4. Attempt quick restart
echo "4. Attempting service restart..."
kubectl rollout restart deployment/conductor-loop -n conductor-loop
kubectl rollout status deployment/conductor-loop -n conductor-loop --timeout=300s

# 5. Verify recovery
echo "5. Verifying service recovery..."
sleep 30
curl -f http://conductor-loop.example.com/healthz

if [ $? -eq 0 ]; then
    echo "✓ Service restored successfully"
else
    echo "✗ Service still down, escalating to advanced recovery"
    ./advanced-recovery.sh
fi

echo "Incident end time: $(date)"
```

#### P1: High Failure Rate
```bash
#!/bin/bash
# p1-high-failure-rate.sh

echo "=== P1 INCIDENT: HIGH FAILURE RATE ==="

# 1. Gather metrics
echo "1. Gathering failure metrics..."
kubectl exec -n monitoring prometheus-0 -- \
  promtool query instant 'rate(conductor_files_processed_total{status="failed"}[5m])'

# 2. Check recent failed files
echo "2. Analyzing recent failures..."
kubectl exec -n conductor-loop conductor-loop-0 -- \
  find /data/handoff/failed -name "*.error.log" -mtime -1 | head -5 | \
  xargs cat

# 3. Check porch connectivity
echo "3. Testing porch connectivity..."
kubectl exec -n conductor-loop conductor-loop-0 -- \
  porch --help

# 4. Scale up resources temporarily
echo "4. Scaling up for improved throughput..."
kubectl scale deployment conductor-loop --replicas=5 -n conductor-loop

# 5. Monitor improvement
echo "5. Monitoring for improvement..."
for i in {1..5}; do
    echo "Check $i/5..."
    kubectl exec -n monitoring prometheus-0 -- \
      promtool query instant 'rate(conductor_files_processed_total{status="failed"}[1m])'
    sleep 60
done
```

### Communication Templates

#### Incident Notification Template
```markdown
**INCIDENT ALERT: Conductor Loop Service Issue**

**Incident ID**: INC-{{ timestamp }}
**Severity**: {{ severity }}
**Status**: {{ status }}
**Started**: {{ start_time }}

**Impact**:
{{ impact_description }}

**Current Actions**:
{{ current_actions }}

**Next Update**: {{ next_update_time }}

**Incident Commander**: {{ commander_name }}
**Contact**: {{ contact_info }}

---
*This is an automated incident notification*
```

#### Status Page Update Template
```json
{
  "incident": {
    "name": "Conductor Loop Performance Degradation",
    "status": "investigating",
    "impact": "minor",
    "started_at": "2024-08-15T10:30:00Z",
    "components": [
      {
        "id": "conductor-loop",
        "status": "degraded_performance"
      }
    ],
    "updates": [
      {
        "status": "investigating",
        "body": "We are investigating reports of slower intent processing times. The service remains functional but with reduced throughput.",
        "created_at": "2024-08-15T10:35:00Z"
      }
    ]
  }
}
```

## Maintenance Procedures

### Regular Maintenance Tasks

#### Daily Maintenance Checklist
```bash
#!/bin/bash
# daily-maintenance.sh

echo "=== DAILY MAINTENANCE: $(date) ==="

# 1. Check service health
echo "1. Checking service health..."
systemctl is-active conductor-loop
curl -s http://localhost:8080/healthz | jq .

# 2. Review error logs
echo "2. Reviewing error logs from last 24 hours..."
journalctl -u conductor-loop --since "24 hours ago" --priority=err | wc -l

# 3. Check disk usage
echo "3. Checking disk usage..."
df -h /opt/conductor-loop/data
du -sh /opt/conductor-loop/data/handoff/{processed,failed}

# 4. Review processing statistics
echo "4. Review processing statistics..."
find /opt/conductor-loop/data/handoff/processed -name "*.json" -mtime -1 | wc -l
find /opt/conductor-loop/data/handoff/failed -name "*.json" -mtime -1 | wc -l

# 5. Check backup status
echo "5. Checking backup status..."
ls -la /backup/conductor-loop/conductor-loop_$(date +%Y%m%d)*.tar.gz 2>/dev/null | wc -l

# 6. Performance metrics
echo "6. Performance metrics..."
ps aux | grep conductor-loop | grep -v grep | awk '{print "CPU:", $3"%, Memory:", $4"%"}'

echo "Daily maintenance completed"
```

#### Weekly Maintenance Checklist
```bash
#!/bin/bash
# weekly-maintenance.sh

echo "=== WEEKLY MAINTENANCE: $(date) ==="

# 1. Log rotation and cleanup
echo "1. Log rotation and cleanup..."
find /opt/conductor-loop/logs -name "*.log" -mtime +7 -delete
journalctl --vacuum-time=7d

# 2. State file optimization
echo "2. State file optimization..."
# Backup current state
cp /opt/conductor-loop/data/handoff/.conductor-state.json \
   /tmp/state-backup-$(date +%s).json

# Clean up old entries (older than 30 days)
systemctl stop conductor-loop
./cleanup-old-state-entries.sh
systemctl start conductor-loop

# 3. Performance analysis
echo "3. Performance analysis..."
./generate-weekly-performance-report.sh

# 4. Security updates check
echo "4. Checking for security updates..."
apt list --upgradable | grep -i security

# 5. Backup verification
echo "5. Backup verification..."
./verify-backup.sh

# 6. Certificate expiry check
echo "6. Certificate expiry check..."
openssl x509 -in /etc/ssl/certs/conductor-loop.crt -noout -dates

echo "Weekly maintenance completed"
```

#### Monthly Maintenance Procedures
```bash
#!/bin/bash
# monthly-maintenance.sh

echo "=== MONTHLY MAINTENANCE: $(date) ==="

# 1. Full system backup
echo "1. Performing full system backup..."
./full-system-backup.sh

# 2. Capacity planning review
echo "2. Capacity planning review..."
./generate-capacity-report.sh

# 3. Security audit
echo "3. Security audit..."
./security-audit.sh

# 4. Performance benchmarking
echo "4. Performance benchmarking..."
./performance-benchmark.sh

# 5. Dependency updates
echo "5. Checking dependency updates..."
go list -u -m all | grep -v "^github.com/thc1006/nephoran-intent-operator"

# 6. Documentation review
echo "6. Documentation review..."
./validate-documentation.sh

echo "Monthly maintenance completed"
```

### Update Procedures

#### Application Updates
```bash
#!/bin/bash
# update-conductor-loop.sh

NEW_VERSION="$1"
if [[ -z "$NEW_VERSION" ]]; then
    echo "Usage: $0 <new_version>"
    exit 1
fi

echo "=== UPDATING CONDUCTOR-LOOP TO VERSION $NEW_VERSION ==="

# 1. Pre-update backup
echo "1. Creating pre-update backup..."
./conductor-loop-backup.sh

# 2. Download new version
echo "2. Downloading new version..."
wget "https://github.com/thc1006/nephoran-intent-operator/releases/download/$NEW_VERSION/conductor-loop-linux-amd64" \
     -O /tmp/conductor-loop-new

# 3. Verify checksum
echo "3. Verifying checksum..."
wget "https://github.com/thc1006/nephoran-intent-operator/releases/download/$NEW_VERSION/checksums.txt" \
     -O /tmp/checksums.txt
sha256sum -c /tmp/checksums.txt --ignore-missing

# 4. Stop service
echo "4. Stopping service..."
systemctl stop conductor-loop

# 5. Backup current binary
echo "5. Backing up current binary..."
cp /opt/conductor-loop/bin/conductor-loop /opt/conductor-loop/bin/conductor-loop.backup

# 6. Install new binary
echo "6. Installing new binary..."
cp /tmp/conductor-loop-new /opt/conductor-loop/bin/conductor-loop
chmod +x /opt/conductor-loop/bin/conductor-loop

# 7. Start service
echo "7. Starting service..."
systemctl start conductor-loop

# 8. Verify update
echo "8. Verifying update..."
sleep 10
systemctl is-active conductor-loop
/opt/conductor-loop/bin/conductor-loop --version

# 9. Smoke test
echo "9. Running smoke test..."
curl -f http://localhost:8080/healthz

echo "Update completed successfully"
```

#### Kubernetes Rolling Updates
```bash
#!/bin/bash
# k8s-rolling-update.sh

NEW_IMAGE="$1"
if [[ -z "$NEW_IMAGE" ]]; then
    echo "Usage: $0 <new_image_tag>"
    exit 1
fi

echo "=== KUBERNETES ROLLING UPDATE TO $NEW_IMAGE ==="

# 1. Verify new image exists
echo "1. Verifying new image..."
docker pull "ghcr.io/thc1006/conductor-loop:$NEW_IMAGE"

# 2. Update deployment
echo "2. Updating deployment..."
kubectl set image deployment/conductor-loop \
  conductor-loop="ghcr.io/thc1006/conductor-loop:$NEW_IMAGE" \
  -n conductor-loop

# 3. Monitor rollout
echo "3. Monitoring rollout..."
kubectl rollout status deployment/conductor-loop -n conductor-loop --timeout=600s

# 4. Verify all pods are ready
echo "4. Verifying pod readiness..."
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=conductor-loop \
  -n conductor-loop --timeout=300s

# 5. Health check
echo "5. Performing health check..."
kubectl port-forward -n conductor-loop service/conductor-loop 8080:8080 &
PORTFWD_PID=$!
sleep 5
curl -f http://localhost:8080/healthz
kill $PORTFWD_PID

echo "Rolling update completed successfully"
```

This operations guide provides comprehensive coverage of production deployment, monitoring, and maintenance procedures for the conductor-loop component. It includes practical scripts, configuration examples, and step-by-step procedures for common operational tasks.