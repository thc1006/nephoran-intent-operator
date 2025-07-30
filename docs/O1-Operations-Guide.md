# O1 Interface Operations and Maintenance Guide

## Overview

This guide provides comprehensive operational procedures for the O1 interface in the Nephoran Intent Operator. It covers deployment, configuration, monitoring, troubleshooting, and maintenance activities for production environments.

## Table of Contents

1. [Deployment Procedures](#deployment-procedures)
2. [Configuration Management](#configuration-management)
3. [Monitoring and Observability](#monitoring-and-observability)
4. [Troubleshooting Guide](#troubleshooting-guide)
5. [Maintenance Procedures](#maintenance-procedures)
6. [Performance Tuning](#performance-tuning)
7. [Security Operations](#security-operations)
8. [Backup and Recovery](#backup-and-recovery)
9. [Operational Playbooks](#operational-playbooks)
10. [Best Practices](#best-practices)

## Deployment Procedures

### Prerequisites

Before deploying the O1 interface, ensure the following prerequisites are met:

#### System Requirements
- **Kubernetes**: Version 1.25 or higher
- **Memory**: Minimum 512Mi per pod, recommended 2Gi
- **CPU**: Minimum 0.1 cores per pod, recommended 0.5 cores
- **Storage**: 10Gi persistent storage for YANG models and logs

#### Network Requirements
- **Connectivity**: Network access to target O-RAN network elements
- **Ports**: 
  - 830 (NETCONF over SSH) - outbound to network elements
  - 8080 (metrics) - inbound for monitoring
  - 8443 (webhooks) - inbound for Kubernetes admission control

#### Security Requirements
- **Credentials**: SSH credentials or certificates for network element access
- **RBAC**: Kubernetes service account with appropriate permissions
- **Network Policies**: Configured to allow required traffic flows

### Step-by-Step Deployment

#### 1. Prepare Namespace and RBAC

```bash
# Create namespace
kubectl create namespace nephoran-system

# Apply RBAC configuration
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nephoran-o1-adaptor
  namespace: nephoran-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-o1-adaptor
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets", "events"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: ["nephoran.com"]
  resources: ["managedelements", "networkintents", "e2nodesets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["nephoran.com"]
  resources: ["managedelements/status", "networkintents/status", "e2nodesets/status"]
  verbs: ["get", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nephoran-o1-adaptor
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nephoran-o1-adaptor
subjects:
- kind: ServiceAccount
  name: nephoran-o1-adaptor
  namespace: nephoran-system
EOF
```

#### 2. Configure Secrets and ConfigMaps

```bash
# Create SSH credentials secret
kubectl create secret generic o1-ssh-credentials \
  --from-literal=username=admin \
  --from-literal=password=your-secure-password \
  --from-file=private-key=path/to/private-key \
  --namespace=nephoran-system

# Create configuration
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: o1-adaptor-config
  namespace: nephoran-system
data:
  config.yaml: |
    o1:
      defaultPort: 830
      connectTimeout: 30s
      requestTimeout: 60s
      maxRetries: 3
      retryInterval: 5s
      yangModels:
        autoLoad: true
        modelPaths:
          - "/etc/yang-models"
      metrics:
        enabled: true
        port: 8080
        collectionInterval: 30s
      logging:
        level: "info"
        format: "json"
EOF
```

#### 3. Deploy O1 Adaptor

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-o1-adaptor
  namespace: nephoran-system
  labels:
    app: nephoran-o1-adaptor
    component: o1-interface
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nephoran-o1-adaptor
  template:
    metadata:
      labels:
        app: nephoran-o1-adaptor
        component: o1-interface
    spec:
      serviceAccountName: nephoran-o1-adaptor
      containers:
      - name: o1-adaptor
        image: nephoran/o1-adaptor:latest
        ports:
        - containerPort: 8080
          name: metrics
          protocol: TCP
        - containerPort: 8443
          name: webhook
          protocol: TCP
        env:
        - name: CONFIG_FILE
          value: "/etc/config/config.yaml"
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        volumeMounts:
        - name: config
          mountPath: /etc/config
          readOnly: true
        - name: ssh-credentials
          mountPath: /etc/ssh-credentials
          readOnly: true
        - name: yang-models
          mountPath: /etc/yang-models
          readOnly: true
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 1Gi
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
      volumes:
      - name: config
        configMap:
          name: o1-adaptor-config
      - name: ssh-credentials
        secret:
          secretName: o1-ssh-credentials
          defaultMode: 0600
      - name: yang-models
        configMap:
          name: yang-models
          optional: true
---
apiVersion: v1
kind: Service
metadata:
  name: nephoran-o1-adaptor
  namespace: nephoran-system
  labels:
    app: nephoran-o1-adaptor
spec:
  selector:
    app: nephoran-o1-adaptor
  ports:
  - name: metrics
    port: 8080
    targetPort: 8080
    protocol: TCP
  - name: webhook
    port: 8443
    targetPort: 8443
    protocol: TCP
  type: ClusterIP
EOF
```

#### 4. Verify Deployment

```bash
# Check deployment status
kubectl get deployment nephoran-o1-adaptor -n nephoran-system

# Check pod status
kubectl get pods -l app=nephoran-o1-adaptor -n nephoran-system

# Check logs
kubectl logs -l app=nephoran-o1-adaptor -n nephoran-system --tail=50

# Test health endpoints
kubectl port-forward svc/nephoran-o1-adaptor 8080:8080 -n nephoran-system &
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

### Configuration Validation

#### Network Element Connectivity Test

Create a test ManagedElement to validate connectivity:

```bash
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1alpha1
kind: ManagedElement
metadata:
  name: test-element
  namespace: nephoran-system
spec:
  deploymentName: "test-deployment"
  host: "192.168.1.100"
  port: 830
  credentials:
    secretRef:
      name: "o1-ssh-credentials"
  o1Config: |
    <config>
      <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
        <interface>
          <name>test-interface</name>
          <type>ianaift:ethernetCsmacd</type>
          <enabled>true</enabled>
        </interface>
      </interfaces>
    </config>
EOF

# Check status
kubectl get managedelement test-element -n nephoran-system -o yaml
```

## Configuration Management

### O1 Adaptor Configuration

#### Main Configuration File

The O1 adaptor uses a YAML configuration file with the following structure:

```yaml
# /etc/config/config.yaml
o1:
  # Connection settings
  defaultPort: 830
  connectTimeout: "30s"
  requestTimeout: "60s"
  maxRetries: 3
  retryInterval: "5s"
  
  # TLS configuration (optional)
  tls:
    enabled: false
    certFile: "/etc/tls/tls.crt"
    keyFile: "/etc/tls/tls.key"
    caFile: "/etc/tls/ca.crt"
    insecureSkipVerify: false
    minVersion: "1.2"
  
  # YANG model management
  yangModels:
    autoLoad: true
    modelPaths:
      - "/etc/yang-models"
      - "/usr/share/yang"
    customModels:
      - name: "vendor-specific-model"
        path: "/etc/custom-models/vendor.yang"
  
  # Connection pooling
  connectionPool:
    maxConnections: 10
    maxIdleConnections: 5
    connectionTimeout: "30s"
    idleTimeout: "5m"
  
  # Metrics and monitoring
  metrics:
    enabled: true
    port: 8080
    path: "/metrics"
    collectionInterval: "30s"
    retentionPeriod: "24h"
  
  # Logging configuration
  logging:
    level: "info"          # debug, info, warn, error
    format: "json"         # json, text
    output: "stdout"       # stdout, file, syslog
    timestampFormat: "2006-01-02T15:04:05.000Z07:00"
  
  # Circuit breaker settings
  circuitBreaker:
    enabled: true
    failureThreshold: 5
    recoveryTimeout: "1m"
    halfOpenMaxRequests: 3
  
  # Rate limiting
  rateLimiting:
    enabled: true
    requestsPerSecond: 10
    burst: 20
```

#### Environment-Specific Configurations

**Development Configuration:**
```yaml
o1:
  defaultPort: 830
  connectTimeout: "10s"
  requestTimeout: "30s"
  maxRetries: 2
  logging:
    level: "debug"
  metrics:
    collectionInterval: "10s"
```

**Production Configuration:**
```yaml
o1:
  defaultPort: 830
  connectTimeout: "30s"
  requestTimeout: "60s"
  maxRetries: 3
  tls:
    enabled: true
    minVersion: "1.3"
  logging:
    level: "info"
    format: "json"
  metrics:
    collectionInterval: "30s"
    retentionPeriod: "7d"
  circuitBreaker:
    enabled: true
    failureThreshold: 5
```

### Dynamic Configuration Updates

#### ConfigMap Updates

```bash
# Update configuration
kubectl patch configmap o1-adaptor-config -n nephoran-system \
  --type merge -p '{"data":{"config.yaml":"o1:\n  logging:\n    level: debug"}}'

# Trigger pod restart to apply changes
kubectl rollout restart deployment/nephoran-o1-adaptor -n nephoran-system

# Monitor rollout
kubectl rollout status deployment/nephoran-o1-adaptor -n nephoran-system
```

#### Runtime Configuration Validation

```bash
# Validate configuration syntax
kubectl exec -it deployment/nephoran-o1-adaptor -n nephoran-system -- \
  /root/manager --validate-config --config-file=/etc/config/config.yaml

# Check current configuration
kubectl exec -it deployment/nephoran-o1-adaptor -n nephoran-system -- \
  curl http://localhost:8080/config
```

## Monitoring and Observability

### Health Monitoring

#### Health Check Endpoints

The O1 adaptor exposes several health check endpoints:

- **`/healthz`**: Basic liveness check
- **`/readyz`**: Readiness check including dependency validation
- **`/metrics`**: Prometheus metrics endpoint
- **`/config`**: Current configuration dump
- **`/connections`**: Active connection status

#### Kubernetes Health Checks

```yaml
# Liveness probe configuration  
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30
  timeoutSeconds: 5
  failureThreshold: 3
  successThreshold: 1

# Readiness probe configuration
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 15
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
  successThreshold: 1
```

### Metrics Collection

#### Prometheus Metrics

The O1 adaptor exposes comprehensive metrics:

**Connection Metrics:**
- `o1_connections_total`: Total number of connections established
- `o1_connections_active`: Currently active connections
- `o1_connections_failed_total`: Failed connection attempts
- `o1_connection_duration_seconds`: Connection establishment time

**Operation Metrics:**
- `o1_operations_total`: Total operations performed (get-config, edit-config, etc.)
- `o1_operation_duration_seconds`: Operation execution time
- `o1_operation_errors_total`: Operation failures by type

**YANG Model Metrics:**
- `o1_yang_models_loaded`: Number of loaded YANG models
- `o1_yang_validation_errors_total`: Configuration validation failures
- `o1_yang_validation_duration_seconds`: Validation processing time

**Network Element Metrics:**
- `o1_managed_elements_total`: Total managed elements
- `o1_managed_elements_connected`: Currently connected elements
- `o1_alarms_active`: Active alarms by severity
- `o1_performance_metrics_collected_total`: Performance metrics collected

#### Monitoring Setup

**Prometheus Configuration:**
```yaml
# prometheus-config.yaml
scrape_configs:
  - job_name: 'nephoran-o1-adaptor'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
            - nephoran-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: nephoran-o1-adaptor
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        action: keep
        regex: metrics
    scrape_interval: 30s
    scrape_timeout: 10s
```

**Grafana Dashboard Configuration:**
```json
{
  "dashboard": {
    "id": null,
    "title": "Nephoran O1 Interface",
    "tags": ["nephoran", "o1", "netconf"],
    "panels": [
      {
        "title": "Connection Status",
        "type": "stat",
        "targets": [
          {
            "expr": "o1_connections_active",
            "legendFormat": "Active Connections"
          }
        ]
      },
      {
        "title": "Operation Success Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(o1_operations_total[5m]) - rate(o1_operation_errors_total[5m])",
            "legendFormat": "Success Rate"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(o1_operation_duration_seconds_bucket[5m]))",
            "legendFormat": "95th Percentile"
          }
        ]
      }
    ]
  }
}
```

### Logging and Tracing

#### Log Configuration

**Structured Logging Example:**
```json
{
  "timestamp": "2024-01-29T10:30:00.000Z",
  "level": "info",
  "logger": "o1-adaptor",
  "message": "NETCONF connection established",
  "component": "netconf-client",
  "managedElement": "o-cu-01",
  "host": "192.168.1.100",
  "port": 830,
  "sessionId": "12345",
  "duration": "2.3s",
  "capabilities": ["urn:ietf:params:netconf:base:1.0", "urn:o-ran:hardware:1.0"]
}
```

#### Log Aggregation

**Fluent Bit Configuration:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: nephoran-system
data:
  fluent-bit.conf: |
    [INPUT]
        Name tail
        Path /var/log/containers/*nephoran-o1-adaptor*.log
        Parser docker
        Tag o1.*
        Refresh_Interval 5
        
    [FILTER]
        Name kubernetes
        Match o1.*
        K8S-Logging.Parser On
        K8S-Logging.Exclude Off
        
    [OUTPUT]
        Name elasticsearch
        Match o1.*
        Host elasticsearch.logging.svc.cluster.local
        Port 9200
        Index nephoran-o1
        Type _doc
```

### Alerting

#### Alert Rules

**Prometheus Alert Rules:**
```yaml
groups:
  - name: nephoran-o1-alerts
    rules:
      - alert: O1AdaptorDown
        expr: up{job="nephoran-o1-adaptor"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "O1 Adaptor is down"
          description: "O1 Adaptor has been down for more than 1 minute"
          
      - alert: O1HighConnectionFailureRate
        expr: rate(o1_connections_failed_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High connection failure rate"
          description: "O1 connection failure rate is {{ $value }} per second"
          
      - alert: O1HighOperationLatency
        expr: histogram_quantile(0.95, rate(o1_operation_duration_seconds_bucket[5m])) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High operation latency"
          description: "95th percentile latency is {{ $value }}s"
          
      - alert: O1YANGValidationErrors
        expr: rate(o1_yang_validation_errors_total[5m]) > 0.05
        for: 3m
        labels:
          severity: warning
        annotations:
          summary: "High YANG validation error rate"
          description: "YANG validation errors occurring at {{ $value }} per second"
```

## Troubleshooting Guide

### Common Issues and Solutions

#### Connection Issues

**Problem:** Cannot establish NETCONF connection to network element

**Symptoms:**
- Connection timeout errors in logs
- `o1_connections_failed_total` metric increasing
- ManagedElement status shows "ConnectionFailed"

**Diagnostic Steps:**
```bash
# 1. Check network connectivity
kubectl exec -it deployment/nephoran-o1-adaptor -n nephoran-system -- \
  nc -zv <target-host> 830

# 2. Verify SSH connectivity
kubectl exec -it deployment/nephoran-o1-adaptor -n nephoran-system -- \
  ssh -o ConnectTimeout=10 admin@<target-host> -p 830

# 3. Check credentials
kubectl get secret o1-ssh-credentials -n nephoran-system -o yaml

# 4. Review connection logs
kubectl logs -l app=nephoran-o1-adaptor -n nephoran-system | grep -i "connection"
```

**Common Solutions:**
1. **Network Issues**: Check firewall rules, network policies, DNS resolution
2. **Authentication**: Verify credentials, SSH keys, host key verification
3. **NETCONF Support**: Ensure target device supports NETCONF subsystem
4. **Resource Limits**: Check if connection pool limits are exceeded

#### Configuration Validation Errors

**Problem:** YANG validation failures when applying configuration

**Symptoms:**
- "Configuration validation failed" errors
- `o1_yang_validation_errors_total` metric increasing
- ManagedElement status shows "ValidationFailed"

**Diagnostic Steps:**
```bash
# 1. Check YANG model registry
kubectl exec -it deployment/nephoran-o1-adaptor -n nephoran-system -- \
  curl http://localhost:8080/yang/models

# 2. Validate configuration manually
kubectl exec -it deployment/nephoran-o1-adaptor -n nephoran-system -- \
  /root/manager --validate-yang --model=o-ran-hardware --config=config.xml

# 3. Check model compatibility
kubectl logs -l app=nephoran-o1-adaptor -n nephoran-system | grep -i "yang\|validation"
```

**Common Solutions:**
1. **Schema Mismatch**: Update YANG models to match target device capabilities
2. **Configuration Format**: Ensure XML/JSON format is correct
3. **Mandatory Fields**: Verify all required fields are present
4. **Data Types**: Check data type compatibility (string, integer, boolean)

#### Performance Issues

**Problem:** Slow operation response times

**Symptoms:**
- High values for `o1_operation_duration_seconds` metric
- User reports of slow configuration changes
- Timeout errors in logs

**Diagnostic Steps:**
```bash
# 1. Check current connections
kubectl exec -it deployment/nephoran-o1-adaptor -n nephoran-system -- \
  curl http://localhost:8080/connections

# 2. Monitor resource usage
kubectl top pods -l app=nephoran-o1-adaptor -n nephoran-system

# 3. Check for connection pool exhaustion
kubectl logs -l app=nephoran-o1-adaptor -n nephoran-system | grep -i "pool\|timeout"
```

**Common Solutions:**
1. **Connection Pooling**: Increase connection pool size
2. **Resource Allocation**: Increase CPU/memory limits
3. **Timeout Configuration**: Adjust request timeouts
4. **Load Balancing**: Scale up replicas for higher load

### Diagnostic Tools

#### Connection Diagnostics

**Connection Status Script:**
```bash
#!/bin/bash
# connection-diagnostics.sh

NAMESPACE="nephoran-system"
DEPLOYMENT="nephoran-o1-adaptor"

echo "=== O1 Connection Diagnostics ==="

# Check deployment status
echo "1. Deployment Status:"
kubectl get deployment $DEPLOYMENT -n $NAMESPACE

# Check pod status
echo "2. Pod Status:"
kubectl get pods -l app=$DEPLOYMENT -n $NAMESPACE

# Check active connections
echo "3. Active Connections:"
kubectl exec deployment/$DEPLOYMENT -n $NAMESPACE -- \
  curl -s http://localhost:8080/connections | jq .

# Check recent errors
echo "4. Recent Errors:"
kubectl logs -l app=$DEPLOYMENT -n $NAMESPACE --tail=20 | grep -i error

# Check metrics
echo "5. Connection Metrics:"
kubectl exec deployment/$DEPLOYMENT -n $NAMESPACE -- \
  curl -s http://localhost:8080/metrics | grep o1_connections
```

#### YANG Model Diagnostics

**YANG Validation Script:**
```bash
#!/bin/bash
# yang-diagnostics.sh

NAMESPACE="nephoran-system"
DEPLOYMENT="nephoran-o1-adaptor"
CONFIG_FILE="${1:-config.xml}"

echo "=== YANG Model Diagnostics ==="

# List loaded models
echo "1. Loaded YANG Models:"
kubectl exec deployment/$DEPLOYMENT -n $NAMESPACE -- \
  curl -s http://localhost:8080/yang/models | jq '.models[] | {name, version, namespace}'

# Validate configuration
echo "2. Configuration Validation:"
if [ -f "$CONFIG_FILE" ]; then
  kubectl exec deployment/$DEPLOYMENT -n $NAMESPACE -- \
    /root/manager --validate-config --config-file=/dev/stdin < "$CONFIG_FILE"
else
  echo "Configuration file not provided or not found"
fi

# Check validation metrics
echo "3. Validation Metrics:"
kubectl exec deployment/$DEPLOYMENT -n $NAMESPACE -- \
  curl -s http://localhost:8080/metrics | grep yang_validation
```

### Log Analysis

#### Common Log Patterns

**Successful Connection:**
```
{"level":"info","msg":"NETCONF connection established","host":"192.168.1.100","port":830,"sessionId":"abc123"}
```

**Connection Failure:**
```
{"level":"error","msg":"Failed to establish NETCONF connection","host":"192.168.1.100","error":"connection timeout"}
```

**Configuration Applied:**
```
{"level":"info","msg":"Configuration applied successfully","managedElement":"o-cu-01","operation":"edit-config"}
```

**YANG Validation Error:**
```
{"level":"error","msg":"YANG validation failed","model":"o-ran-hardware","error":"mandatory field missing: name"}
```

#### Log Filtering and Analysis

**Useful Log Queries:**
```bash
# Connection-related logs
kubectl logs -l app=nephoran-o1-adaptor -n nephoran-system | grep -i "connection\|netconf"

# Error logs only
kubectl logs -l app=nephoran-o1-adaptor -n nephoran-system | grep '"level":"error"'

# Performance-related logs
kubectl logs -l app=nephoran-o1-adaptor -n nephoran-system | grep -i "timeout\|duration\|slow"

# YANG-related logs
kubectl logs -l app=nephoran-o1-adaptor -n nephoran-system | grep -i "yang\|validation\|schema"
```

## Maintenance Procedures

### Routine Maintenance Tasks

#### Daily Maintenance

**Daily Health Check Script:**
```bash
#!/bin/bash
# daily-health-check.sh

NAMESPACE="nephoran-system"
DATE=$(date +%Y-%m-%d)
REPORT_FILE="/tmp/o1-health-report-$DATE.txt"

echo "=== Daily O1 Health Check - $DATE ===" > $REPORT_FILE

# 1. Check deployment health
echo "1. Deployment Status:" >> $REPORT_FILE
kubectl get deployment nephoran-o1-adaptor -n $NAMESPACE >> $REPORT_FILE

# 2. Check pod health
echo "2. Pod Health:" >> $REPORT_FILE
kubectl get pods -l app=nephoran-o1-adaptor -n $NAMESPACE >> $REPORT_FILE

# 3. Check resource usage
echo "3. Resource Usage:" >> $REPORT_FILE
kubectl top pods -l app=nephoran-o1-adaptor -n $NAMESPACE >> $REPORT_FILE

# 4. Check error count (last 24 hours)
echo "4. Error Count (24h):" >> $REPORT_FILE
kubectl logs -l app=nephoran-o1-adaptor -n $NAMESPACE --since=24h | \
  grep -c '"level":"error"' >> $REPORT_FILE

# 5. Connection status
echo "5. Active Connections:" >> $REPORT_FILE
kubectl exec deployment/nephoran-o1-adaptor -n $NAMESPACE -- \
  curl -s http://localhost:8080/connections | jq '.activeConnections' >> $REPORT_FILE

# 6. Check recent alarms
echo "6. Recent Managed Element Status:" >> $REPORT_FILE
kubectl get managedelements -A --show-labels >> $REPORT_FILE

echo "Health check complete. Report saved to: $REPORT_FILE"
```

#### Weekly Maintenance

**Weekly Maintenance Tasks:**
1. **Log Rotation and Cleanup**
2. **Certificate Expiry Check**
3. **Performance Metrics Review**
4. **YANG Model Updates**
5. **Security Scan**

**Weekly Maintenance Script:**
```bash
#!/bin/bash
# weekly-maintenance.sh

NAMESPACE="nephoran-system"
DATE=$(date +%Y-%m-%d)

echo "=== Weekly O1 Maintenance - $DATE ==="

# 1. Check certificate expiry
echo "1. Checking certificate expiry..."
kubectl get secrets -n $NAMESPACE -o json | \
  jq -r '.items[] | select(.type=="kubernetes.io/tls") | 
    "\(.metadata.name): \(.data."tls.crt" | @base64d | split("\n")[0])"' | \
  while read cert; do
    echo "Checking certificate: $cert"
    # Certificate expiry check would go here
  done

# 2. Performance metrics review
echo "2. Performance metrics review..."
kubectl exec deployment/nephoran-o1-adaptor -n $NAMESPACE -- \
  curl -s http://localhost:8080/metrics | grep -E "o1_(connections|operations|yang)"

# 3. Check for YANG model updates
echo "3. Checking YANG models..."
kubectl exec deployment/nephoran-o1-adaptor -n $NAMESPACE -- \
  curl -s http://localhost:8080/yang/models | jq '.statistics'

# 4. Security scan (placeholder)
echo "4. Security scan..."
# Security scanning tools would be invoked here

echo "Weekly maintenance completed"
```

#### Monthly Maintenance

**Monthly Tasks:**
1. **Backup Configuration**
2. **Update YANG Models**
3. **Review and Update Documentation**
4. **Capacity Planning Review**
5. **Security Policy Review**

### Software Updates

#### O1 Adaptor Updates

**Update Procedure:**
```bash
#!/bin/bash
# update-o1-adaptor.sh

NEW_VERSION="${1:-latest}"
NAMESPACE="nephoran-system"

echo "=== Updating O1 Adaptor to version $NEW_VERSION ==="

# 1. Backup current configuration
echo "1. Backing up current configuration..."
kubectl get configmap o1-adaptor-config -n $NAMESPACE -o yaml > \
  "o1-config-backup-$(date +%Y%m%d-%H%M%S).yaml"

# 2. Update container image
echo "2. Updating container image..."
kubectl set image deployment/nephoran-o1-adaptor \
  o1-adaptor=nephoran/o1-adaptor:$NEW_VERSION -n $NAMESPACE

# 3. Wait for rollout to complete
echo "3. Waiting for rollout to complete..."
kubectl rollout status deployment/nephoran-o1-adaptor -n $NAMESPACE --timeout=300s

# 4. Verify health
echo "4. Verifying health after update..."
sleep 30
kubectl exec deployment/nephoran-o1-adaptor -n $NAMESPACE -- \
  curl -f http://localhost:8080/healthz

if [ $? -eq 0 ]; then
  echo "✅ Update completed successfully"
else
  echo "❌ Update failed - rolling back"
  kubectl rollout undo deployment/nephoran-o1-adaptor -n $NAMESPACE
  exit 1
fi
```

#### YANG Model Updates

**YANG Model Update Script:**
```bash
#!/bin/bash
# update-yang-models.sh

NAMESPACE="nephoran-system"
MODEL_SOURCE="${1:-/path/to/yang-models}"

echo "=== Updating YANG Models ==="

# 1. Create new ConfigMap with updated models
kubectl create configmap yang-models-new \
  --from-file=$MODEL_SOURCE \
  --dry-run=client -o yaml | kubectl apply -n $NAMESPACE -f -

# 2. Update deployment to use new ConfigMap
kubectl patch deployment nephoran-o1-adaptor -n $NAMESPACE \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/volumes/2/configMap/name", "value": "yang-models-new"}]'

# 3. Wait for rollout
kubectl rollout status deployment/nephoran-o1-adaptor -n $NAMESPACE

# 4. Verify models loaded
kubectl exec deployment/nephoran-o1-adaptor -n $NAMESPACE -- \
  curl -s http://localhost:8080/yang/models | jq '.statistics.totalModels'

# 5. Cleanup old ConfigMap
kubectl delete configmap yang-models -n $NAMESPACE 2>/dev/null || true
kubectl patch deployment nephoran-o1-adaptor -n $NAMESPACE \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/volumes/2/configMap/name", "value": "yang-models"}]'

echo "YANG models updated successfully"
```

## Performance Tuning

### Connection Tuning

#### Connection Pool Configuration

```yaml
# Optimized connection pool settings
o1:
  connectionPool:
    maxConnections: 20          # Increase for high-throughput environments
    maxIdleConnections: 10      # Keep more connections alive
    connectionTimeout: "30s"    # Reasonable timeout
    idleTimeout: "10m"          # Longer idle timeout for persistent connections
    keepAlive: true             # Enable TCP keep-alive
    keepAliveInterval: "30s"    # Keep-alive probe interval
```

#### SSH Connection Optimization

```yaml
# SSH-specific optimizations
o1:
  ssh:
    compression: true           # Enable SSH compression
    cipherSuites:              # Use faster ciphers
      - "aes128-gcm@openssh.com"
      - "chacha20-poly1305@openssh.com"
    kexAlgorithms:             # Prefer ECDH key exchange
      - "curve25519-sha256"
      - "ecdh-sha2-nistp256"
    tcpKeepAlive: true         # Enable TCP keep-alive
    serverAliveInterval: 30    # Send keep-alive every 30s
    serverAliveCountMax: 3     # Allow 3 missed keep-alives
```

### Memory and CPU Optimization

#### Resource Allocation Guidelines

**Development Environment:**
```yaml
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

**Production Environment:**
```yaml
resources:
  requests:
    cpu: 200m
    memory: 512Mi
  limits:
    cpu: 1000m
    memory: 1Gi
```

**High-Load Environment:**
```yaml
resources:
  requests:
    cpu: 500m
    memory: 1Gi
  limits:
    cpu: 2000m
    memory: 2Gi
```

#### JVM Tuning (if applicable)

```yaml
env:
  - name: JAVA_OPTS
    value: "-Xms512m -Xmx1g -XX:+UseG1GC -XX:MaxGCPauseMillis=100"
```

### Scaling Configuration

#### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: nephoran-o1-adaptor-hpa
  namespace: nephoran-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nephoran-o1-adaptor
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
        name: o1_connections_active
      target:
        type: AverageValue
        averageValue: "50"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
```

This comprehensive operations guide provides all the necessary procedures for deploying, configuring, monitoring, troubleshooting, and maintaining the O1 interface in production environments. The guide includes practical scripts, configuration examples, and step-by-step procedures that operations teams can follow to ensure reliable operation of the O1 interface components.