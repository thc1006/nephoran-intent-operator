# O2 IMS Troubleshooting Guide

## Overview

This comprehensive troubleshooting guide provides step-by-step solutions for common issues encountered when deploying and operating the O2 Infrastructure Management Service. It covers diagnostic procedures, root cause analysis, and remediation steps for various scenarios.

## Quick Diagnostic Checklist

Before diving into specific issues, run this quick diagnostic checklist:

```bash
#!/bin/bash
# quick-diagnostics.sh

echo "=== O2 IMS Quick Diagnostics ==="

# 1. Check pod status
echo "1. Checking pod status..."
kubectl get pods -n o2-system -o wide

# 2. Check service endpoints
echo "2. Checking service endpoints..."
kubectl get endpoints -n o2-system

# 3. Check ingress status
echo "3. Checking ingress status..."
kubectl get ingress -n o2-system

# 4. Check database connectivity
echo "4. Checking database connectivity..."
kubectl exec -it deployment/o2-ims -n o2-system -- nc -zv postgresql-primary 5432

# 5. Check recent events
echo "5. Checking recent events..."
kubectl get events -n o2-system --sort-by='.lastTimestamp' | tail -20

# 6. Quick health check
echo "6. Quick health check..."
kubectl port-forward -n o2-system svc/o2-ims 8080:8080 &
sleep 3
curl -s http://localhost:8080/health | jq '.' || echo "Health check failed"
pkill -f "port-forward"

echo "=== Diagnostics Complete ==="
```

## Common Issues and Solutions

### 1. Service Startup Issues

#### 1.1 Pods Failing to Start (CrashLoopBackOff)

**Symptoms:**
- Pods in `CrashLoopBackOff` state
- Repeated restarts
- Application logs showing startup errors

**Diagnostic Steps:**

```bash
# Check pod status and events
kubectl get pods -n o2-system
kubectl describe pod <pod-name> -n o2-system

# Check container logs
kubectl logs <pod-name> -n o2-system -c o2-ims
kubectl logs <pod-name> -n o2-system -c o2-ims --previous

# Check resource limits
kubectl describe pod <pod-name> -n o2-system | grep -A 5 "Limits\|Requests"
```

**Common Causes and Solutions:**

1. **Configuration Issues:**
   ```bash
   # Check ConfigMap
   kubectl get configmap o2-ims-config -n o2-system -o yaml
   
   # Validate configuration syntax
   kubectl exec -it <pod-name> -n o2-system -- cat /etc/config/config.yaml | yq eval
   ```

2. **Secret Mount Issues:**
   ```bash
   # Check secret existence
   kubectl get secrets -n o2-system
   
   # Verify secret contents (base64 decode)
   kubectl get secret <secret-name> -n o2-system -o yaml
   
   # Check if secret is properly mounted
   kubectl exec -it <pod-name> -n o2-system -- ls -la /etc/secrets/
   ```

3. **Database Connectivity Issues:**
   ```bash
   # Test database connection
   kubectl exec -it <pod-name> -n o2-system -- nc -zv postgresql-primary 5432
   
   # Check database credentials
   kubectl exec -it <pod-name> -n o2-system -- psql -h postgresql-primary -U o2user -d o2ims -c "SELECT 1;"
   ```

4. **Insufficient Resources:**
   ```bash
   # Check node resources
   kubectl describe nodes | grep -A 5 "Allocated resources"
   
   # Check resource requests vs limits
   kubectl top pods -n o2-system
   ```

**Resolution Steps:**

```bash
# 1. Fix configuration
kubectl edit configmap o2-ims-config -n o2-system

# 2. Restart deployment
kubectl rollout restart deployment/o2-ims -n o2-system

# 3. Monitor rollout
kubectl rollout status deployment/o2-ims -n o2-system

# 4. Verify health
kubectl get pods -n o2-system
curl -f http://o2ims.your-domain.com/health
```

#### 1.2 ImagePullBackOff Errors

**Symptoms:**
- Pods stuck in `ImagePullBackOff` state
- Cannot pull container image

**Diagnostic Steps:**

```bash
# Check image pull status
kubectl describe pod <pod-name> -n o2-system | grep -A 10 "Events"

# Check image registry access
kubectl get secret -n o2-system | grep registry
```

**Solutions:**

1. **Registry Authentication:**
   ```bash
   # Create registry secret if missing
   kubectl create secret docker-registry regcred \
     --namespace=o2-system \
     --docker-server=registry.nephoran.com \
     --docker-username=<username> \
     --docker-password=<password>
   
   # Update deployment to use secret
   kubectl patch deployment o2-ims -n o2-system -p '
   {
     "spec": {
       "template": {
         "spec": {
           "imagePullSecrets": [{"name": "regcred"}]
         }
       }
     }
   }'
   ```

2. **Image Tag Issues:**
   ```bash
   # Check if image exists
   docker pull registry.nephoran.com/o2-ims:v1.2.0
   
   # Update to correct tag
   kubectl set image deployment/o2-ims -n o2-system o2-ims=registry.nephoran.com/o2-ims:v1.2.1
   ```

### 2. Database Connection Issues

#### 2.1 Connection Timeout/Refused

**Symptoms:**
- Database connection timeouts
- "Connection refused" errors
- High connection latency

**Diagnostic Steps:**

```bash
# Check database pod status
kubectl get pods -l app=postgresql -n o2-system

# Check database service
kubectl get svc -l app=postgresql -n o2-system

# Test connectivity from application pod
kubectl exec -it deployment/o2-ims -n o2-system -- telnet postgresql-primary 5432

# Check database logs
kubectl logs postgresql-primary-0 -n o2-system | tail -50
```

**Common Causes and Solutions:**

1. **PostgreSQL Not Ready:**
   ```bash
   # Check PostgreSQL startup
   kubectl describe pod postgresql-primary-0 -n o2-system
   
   # Wait for readiness
   kubectl wait --for=condition=ready pod/postgresql-primary-0 -n o2-system --timeout=300s
   ```

2. **Network Policy Blocking:**
   ```bash
   # Check network policies
   kubectl get networkpolicy -n o2-system
   
   # Test without network policy (temporarily)
   kubectl delete networkpolicy o2-ims-network-policy -n o2-system
   ```

3. **DNS Resolution Issues:**
   ```bash
   # Test DNS resolution
   kubectl exec -it deployment/o2-ims -n o2-system -- nslookup postgresql-primary.o2-system.svc.cluster.local
   
   # Check CoreDNS
   kubectl get pods -n kube-system -l k8s-app=kube-dns
   ```

#### 2.2 Max Connections Reached

**Symptoms:**
- "too many connections" errors
- Connection pool exhaustion
- Application unable to connect

**Diagnostic Steps:**

```bash
# Check current connection count
kubectl exec postgresql-primary-0 -n o2-system -- psql -U postgres -c "
  SELECT count(*) as total_connections,
         state,
         application_name
  FROM pg_stat_activity
  GROUP BY state, application_name
  ORDER BY total_connections DESC;"

# Check max_connections setting
kubectl exec postgresql-primary-0 -n o2-system -- psql -U postgres -c "SHOW max_connections;"

# Check connection configuration in app
kubectl exec -it deployment/o2-ims -n o2-system -- cat /etc/config/config.yaml | grep -A 5 database
```

**Solutions:**

1. **Increase PostgreSQL max_connections:**
   ```bash
   # Edit PostgreSQL configuration
   kubectl edit postgresql postgresql-primary -n o2-system
   
   # Or update via ConfigMap
   kubectl patch configmap postgresql-primary-configuration -n o2-system --patch='
   {
     "data": {
       "postgresql.conf": "max_connections = 200\nshared_buffers = 128MB\n"
     }
   }'
   
   # Restart PostgreSQL
   kubectl rollout restart statefulset/postgresql-primary -n o2-system
   ```

2. **Implement Connection Pooling:**
   ```yaml
   # pgbouncer-deployment.yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: pgbouncer
     namespace: o2-system
   spec:
     replicas: 2
     selector:
       matchLabels:
         app: pgbouncer
     template:
       metadata:
         labels:
           app: pgbouncer
       spec:
         containers:
         - name: pgbouncer
           image: pgbouncer/pgbouncer:latest
           ports:
           - containerPort: 6432
           env:
           - name: DATABASES_HOST
             value: postgresql-primary
           - name: DATABASES_PORT
             value: "5432"
           - name: POOL_MODE
             value: transaction
           - name: MAX_CLIENT_CONN
             value: "200"
           - name: DEFAULT_POOL_SIZE
             value: "20"
   ```

### 3. Performance Issues

#### 3.1 High API Response Time

**Symptoms:**
- Slow API responses (>2 seconds)
- Timeout errors
- Poor user experience

**Diagnostic Steps:**

```bash
# Check metrics endpoint
curl -s http://o2ims.your-domain.com/metrics | grep http_request_duration

# Check resource utilization
kubectl top pods -n o2-system
kubectl top nodes

# Check database performance
kubectl exec postgresql-primary-0 -n o2-system -- psql -U postgres -c "
  SELECT query, calls, total_time, mean_time, stddev_time
  FROM pg_stat_statements
  WHERE calls > 100
  ORDER BY mean_time DESC
  LIMIT 10;"
```

**Solutions:**

1. **Database Query Optimization:**
   ```sql
   -- Identify slow queries
   SELECT query, calls, total_time, mean_time
   FROM pg_stat_statements
   WHERE mean_time > 1000  -- > 1 second
   ORDER BY mean_time DESC;
   
   -- Check missing indexes
   SELECT schemaname, tablename, attname, n_distinct, correlation
   FROM pg_stats
   WHERE schemaname = 'public'
     AND n_distinct > 100
     AND correlation < 0.1;
   
   -- Add indexes as needed
   CREATE INDEX CONCURRENTLY idx_resource_pools_provider ON resource_pools(provider);
   CREATE INDEX CONCURRENTLY idx_resource_instances_pool_id ON resource_instances(resource_pool_id);
   ```

2. **Application Performance Tuning:**
   ```yaml
   # Increase resource limits
   resources:
     requests:
       cpu: "2000m"
       memory: "4Gi"
     limits:
       cpu: "4000m"
       memory: "8Gi"
   
   # Enable connection pooling
   env:
   - name: DB_MAX_OPEN_CONNS
     value: "50"
   - name: DB_MAX_IDLE_CONNS
     value: "10"
   - name: DB_CONN_MAX_LIFETIME
     value: "1h"
   ```

3. **Horizontal Scaling:**
   ```bash
   # Scale up replicas
   kubectl scale deployment o2-ims -n o2-system --replicas=5
   
   # Enable autoscaling
   kubectl autoscale deployment o2-ims -n o2-system --cpu-percent=70 --min=3 --max=10
   ```

#### 3.2 Memory Issues (OOMKilled)

**Symptoms:**
- Pods being killed due to memory limits
- OutOfMemory events
- Frequent restarts

**Diagnostic Steps:**

```bash
# Check memory usage
kubectl top pods -n o2-system

# Check events for OOMKilled
kubectl get events -n o2-system --field-selector reason=OOMKilling

# Check container limits
kubectl describe pod <pod-name> -n o2-system | grep -A 5 "Limits"

# Check memory pressure on nodes
kubectl describe nodes | grep -A 5 "System Info\|Memory"
```

**Solutions:**

1. **Increase Memory Limits:**
   ```bash
   # Update deployment
   kubectl patch deployment o2-ims -n o2-system -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "o2-ims",
               "resources": {
                 "requests": {"memory": "2Gi"},
                 "limits": {"memory": "4Gi"}
               }
             }
           ]
         }
       }
     }
   }'
   ```

2. **Memory Profiling:**
   ```bash
   # Enable pprof (development only)
   kubectl port-forward deployment/o2-ims -n o2-system 6060:6060
   
   # Generate heap profile
   go tool pprof http://localhost:6060/debug/pprof/heap
   
   # Analyze memory usage
   (pprof) top10
   (pprof) list <function-name>
   ```

3. **Optimize Application Memory:**
   ```go
   // Example optimizations in application code
   
   // Use object pooling for frequent allocations
   var bufferPool = sync.Pool{
       New: func() interface{} {
           return make([]byte, 4096)
       },
   }
   
   // Limit goroutine concurrency
   semaphore := make(chan struct{}, 100)
   
   // Use context with timeout to prevent memory leaks
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

### 4. Network and Ingress Issues

#### 4.1 External Access Issues

**Symptoms:**
- Cannot reach service from outside cluster
- DNS resolution failures
- SSL/TLS certificate issues

**Diagnostic Steps:**

```bash
# Check ingress configuration
kubectl get ingress -n o2-system -o yaml

# Check ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller

# Test internal service access
kubectl port-forward -n o2-system svc/o2-ims 8080:8080
curl http://localhost:8080/health

# Check DNS resolution
nslookup o2ims.your-domain.com

# Check certificate status
kubectl get certificate -n o2-system
kubectl describe certificate o2-ims-tls -n o2-system
```

**Solutions:**

1. **Fix Ingress Configuration:**
   ```yaml
   # correct-ingress.yaml
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: o2-ims-ingress
     namespace: o2-system
     annotations:
       kubernetes.io/ingress.class: "nginx"
       nginx.ingress.kubernetes.io/ssl-redirect: "true"
       cert-manager.io/cluster-issuer: "letsencrypt-prod"
   spec:
     tls:
     - hosts:
       - o2ims.your-domain.com
       secretName: o2-ims-tls
     rules:
     - host: o2ims.your-domain.com
       http:
         paths:
         - path: /
           pathType: Prefix
           backend:
             service:
               name: o2-ims
               port:
                 number: 8080
   ```

2. **Certificate Issues:**
   ```bash
   # Check cert-manager
   kubectl get clusterissuer
   kubectl get certificate -n o2-system
   
   # Force certificate renewal
   kubectl delete certificate o2-ims-tls -n o2-system
   kubectl apply -f ingress-with-tls.yaml
   
   # Check certificate details
   openssl s_client -connect o2ims.your-domain.com:443 -servername o2ims.your-domain.com
   ```

3. **LoadBalancer Issues:**
   ```bash
   # Check external IP assignment
   kubectl get svc -n ingress-nginx
   
   # Check cloud provider load balancer
   # For AWS:
   aws elbv2 describe-load-balancers --region us-east-1
   
   # For GCP:
   gcloud compute forwarding-rules list
   ```

#### 4.2 Internal Service Communication

**Symptoms:**
- Services cannot communicate within cluster
- DNS resolution issues
- Network policy blocking traffic

**Diagnostic Steps:**

```bash
# Test DNS resolution
kubectl exec -it deployment/o2-ims -n o2-system -- nslookup postgresql-primary.o2-system.svc.cluster.local

# Test service connectivity
kubectl exec -it deployment/o2-ims -n o2-system -- nc -zv postgresql-primary 5432

# Check network policies
kubectl get networkpolicy -n o2-system

# Check service endpoints
kubectl get endpoints -n o2-system
```

**Solutions:**

1. **Fix Network Policies:**
   ```yaml
   # Allow database access
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: o2-ims-to-db
     namespace: o2-system
   spec:
     podSelector:
       matchLabels:
         app: o2-ims
     policyTypes:
     - Egress
     egress:
     - to:
       - podSelector:
           matchLabels:
             app: postgresql
       ports:
       - protocol: TCP
         port: 5432
   ```

2. **DNS Configuration:**
   ```bash
   # Check CoreDNS configuration
   kubectl get configmap coredns -n kube-system -o yaml
   
   # Test DNS from pod
   kubectl exec -it deployment/o2-ims -n o2-system -- cat /etc/resolv.conf
   ```

### 5. Authentication and Authorization Issues

#### 5.1 OAuth2 Authentication Failures

**Symptoms:**
- 401 Unauthorized errors
- OAuth2 token validation failures
- Authentication provider errors

**Diagnostic Steps:**

```bash
# Check OAuth2 configuration
kubectl get configmap o2-ims-config -n o2-system -o yaml | grep -A 10 oauth2

# Check OAuth2 secret
kubectl get secret oauth2-credentials -n o2-system -o yaml

# Test token validation
curl -H "Authorization: Bearer <token>" http://o2ims.your-domain.com/o2ims/v1/

# Check application logs
kubectl logs deployment/o2-ims -n o2-system | grep -i auth
```

**Solutions:**

1. **Fix OAuth2 Configuration:**
   ```yaml
   # oauth2-config.yaml
   authentication:
     enabled: true
     providers:
       - type: oauth2
         name: corporate-sso
         config:
           issuerURL: "https://auth.company.com"
           clientID: "o2-ims-production"
           clientSecret: "from-secret"
           scopes: ["openid", "profile", "email"]
           redirectURL: "https://o2ims.your-domain.com/auth/callback"
   ```

2. **Token Validation Issues:**
   ```bash
   # Verify issuer URL accessibility
   curl https://auth.company.com/.well-known/openid-configuration
   
   # Check token validity
   jwt-cli decode <token>
   
   # Verify clock synchronization
   kubectl exec -it deployment/o2-ims -n o2-system -- date
   ```

#### 5.2 RBAC Permission Issues

**Symptoms:**
- 403 Forbidden errors
- Access denied to resources
- Insufficient permissions

**Diagnostic Steps:**

```bash
# Check current user permissions
kubectl auth can-i get resourcepools --as=system:serviceaccount:o2-system:o2-ims

# Check RBAC configuration
kubectl get role,rolebinding,clusterrole,clusterrolebinding -n o2-system

# Check service account
kubectl get serviceaccount o2-ims -n o2-system -o yaml
```

**Solutions:**

```yaml
# rbac-fix.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: o2-ims
  namespace: o2-system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: o2-ims-cluster-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["o2ims.nephoran.com"]
  resources: ["resourcepools", "resourcetypes", "resourceinstances"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: o2-ims-cluster-role-binding
subjects:
- kind: ServiceAccount
  name: o2-ims
  namespace: o2-system
roleRef:
  kind: ClusterRole
  name: o2-ims-cluster-role
  apiGroup: rbac.authorization.k8s.io
```

### 6. Monitoring and Alerting Issues

#### 6.1 Metrics Not Available

**Symptoms:**
- Prometheus not scraping metrics
- Missing metrics in Grafana
- Monitoring dashboards empty

**Diagnostic Steps:**

```bash
# Check metrics endpoint
curl http://o2ims.your-domain.com/metrics

# Check ServiceMonitor
kubectl get servicemonitor -n o2-system

# Check Prometheus targets
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Open http://localhost:9090/targets

# Check service labels
kubectl get service o2-ims -n o2-system --show-labels
```

**Solutions:**

1. **Fix ServiceMonitor:**
   ```yaml
   apiVersion: monitoring.coreos.com/v1
   kind: ServiceMonitor
   metadata:
     name: o2-ims-metrics
     namespace: o2-system
     labels:
       app: o2-ims
   spec:
     selector:
       matchLabels:
         app: o2-ims
     endpoints:
     - port: metrics
       interval: 30s
       path: /metrics
   ```

2. **Check Service Configuration:**
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: o2-ims
     namespace: o2-system
     labels:
       app: o2-ims
   spec:
     selector:
       app: o2-ims
     ports:
     - name: http
       port: 8080
       targetPort: 8080
     - name: metrics
       port: 8080
       targetPort: 8080
   ```

#### 6.2 Alerts Not Firing

**Symptoms:**
- Expected alerts not triggering
- AlertManager not receiving alerts
- Notification channels not working

**Diagnostic Steps:**

```bash
# Check PrometheusRule
kubectl get prometheusrule -n o2-system

# Check alert status in Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Open http://localhost:9090/alerts

# Check AlertManager
kubectl port-forward -n monitoring svc/alertmanager 9093:9093
# Open http://localhost:9093

# Test alert conditions
kubectl exec -n monitoring prometheus-0 -- promtool query instant 'up{job="o2-ims"} == 0'
```

**Solutions:**

```yaml
# prometheus-rule-fix.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: o2-ims-alerts
  namespace: o2-system
  labels:
    app: o2-ims
    role: alert-rules
spec:
  groups:
  - name: o2-ims.rules
    rules:
    - alert: O2IMSInstanceDown
      expr: up{job="o2-ims"} == 0
      for: 1m
      labels:
        severity: critical
        service: o2-ims
      annotations:
        summary: "O2 IMS instance is down"
        description: "O2 IMS instance {{ $labels.instance }} has been down for more than 1 minute"
```

## Advanced Troubleshooting

### Distributed Tracing Analysis

```bash
# Check Jaeger traces
kubectl port-forward -n monitoring svc/jaeger-query 16686:16686
# Open http://localhost:16686

# Search for slow requests
# Filter by service: o2-ims
# Duration > 2s

# Analyze trace spans for bottlenecks
```

### Network Traffic Analysis

```bash
# Capture network traffic (requires tcpdump)
kubectl exec -it deployment/o2-ims -n o2-system -- tcpdump -i any -w /tmp/traffic.pcap

# Analyze with Wireshark or tcpdump
kubectl cp o2-system/<pod-name>:/tmp/traffic.pcap ./traffic.pcap
wireshark traffic.pcap
```

### Memory Leak Detection

```bash
# Monitor memory usage over time
watch kubectl top pods -n o2-system

# Generate heap dumps
kubectl exec -it deployment/o2-ims -n o2-system -- curl http://localhost:6060/debug/pprof/heap > heap.pprof
go tool pprof heap.pprof

# Compare heap dumps over time
go tool pprof -base heap1.pprof heap2.pprof
```

## Emergency Procedures

### Service Unavailable Emergency Response

```bash
#!/bin/bash
# emergency-response.sh

echo "=== O2 IMS Emergency Response ==="

# 1. Immediate assessment
kubectl get pods -n o2-system
kubectl get svc -n o2-system
kubectl get ingress -n o2-system

# 2. Check recent events
kubectl get events -n o2-system --sort-by='.lastTimestamp' | tail -20

# 3. Scale up if needed
kubectl scale deployment o2-ims -n o2-system --replicas=5

# 4. Emergency rollback if recent deployment
kubectl rollout history deployment/o2-ims -n o2-system
kubectl rollout undo deployment/o2-ims -n o2-system

# 5. Bypass problematic components
kubectl patch configmap o2-ims-config -n o2-system --patch='
{
  "data": {
    "emergency-mode": "true"
  }
}'

# 6. Notify stakeholders
# Send notifications via Slack, PagerDuty, etc.

echo "Emergency response completed. Monitor service recovery."
```

### Data Recovery Procedures

```bash
#!/bin/bash
# data-recovery.sh

echo "=== Database Recovery Procedure ==="

# 1. Check database status
kubectl get pods -l app=postgresql -n o2-system

# 2. Check available backups
aws s3 ls s3://your-backup-bucket/o2ims/ --recursive | tail -10

# 3. Stop application to prevent writes
kubectl scale deployment o2-ims -n o2-system --replicas=0

# 4. Restore from latest backup
LATEST_BACKUP=$(aws s3 ls s3://your-backup-bucket/o2ims/ | sort | tail -n 1 | awk '{print $4}')
aws s3 cp s3://your-backup-bucket/o2ims/$LATEST_BACKUP /tmp/

# 5. Restore database
kubectl exec -i postgresql-primary-0 -n o2-system -- psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS o2ims;"
kubectl exec -i postgresql-primary-0 -n o2-system -- psql -U postgres -d postgres -c "CREATE DATABASE o2ims;"
kubectl exec -i postgresql-primary-0 -n o2-system -- pg_restore -U postgres -d o2ims < /tmp/$LATEST_BACKUP

# 6. Restart application
kubectl scale deployment o2-ims -n o2-system --replicas=3

# 7. Verify recovery
kubectl wait --for=condition=ready pod -l app=o2-ims -n o2-system --timeout=300s
curl -f http://o2ims.your-domain.com/health

echo "Data recovery completed successfully!"
```

## Support Escalation

### When to Escalate

Escalate issues to the development team when:

1. **Critical Service Impact:**
   - Complete service outage > 5 minutes
   - Data corruption detected
   - Security breach suspected

2. **Complex Technical Issues:**
   - Memory leaks or performance degradation
   - Database corruption
   - Integration failures with cloud providers

3. **Configuration Issues:**
   - O-RAN compliance violations
   - Authentication/authorization failures
   - Network policy conflicts

### Escalation Information to Provide

```bash
#!/bin/bash
# collect-support-info.sh

TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
SUPPORT_DIR="/tmp/o2ims-support-${TIMESTAMP}"

mkdir -p $SUPPORT_DIR

echo "Collecting support information..."

# System information
kubectl version > $SUPPORT_DIR/version.txt
kubectl get nodes -o wide > $SUPPORT_DIR/nodes.txt

# O2 IMS specific information
kubectl get all -n o2-system > $SUPPORT_DIR/o2-resources.txt
kubectl describe pods -n o2-system > $SUPPORT_DIR/pod-details.txt
kubectl get events -n o2-system > $SUPPORT_DIR/events.txt

# Logs
kubectl logs deployment/o2-ims -n o2-system > $SUPPORT_DIR/o2ims-logs.txt
kubectl logs deployment/o2-ims -n o2-system --previous > $SUPPORT_DIR/o2ims-logs-previous.txt

# Configuration
kubectl get configmap o2-ims-config -n o2-system -o yaml > $SUPPORT_DIR/config.yaml

# Database information
kubectl exec postgresql-primary-0 -n o2-system -- psql -U postgres -c "SELECT version();" > $SUPPORT_DIR/db-version.txt
kubectl exec postgresql-primary-0 -n o2-system -- psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;" > $SUPPORT_DIR/db-connections.txt

# Performance data
kubectl top pods -n o2-system > $SUPPORT_DIR/resource-usage.txt
curl -s http://o2ims.your-domain.com/metrics > $SUPPORT_DIR/metrics.txt

# Create archive
tar -czf o2ims-support-${TIMESTAMP}.tar.gz -C /tmp o2ims-support-${TIMESTAMP}

echo "Support information collected in: o2ims-support-${TIMESTAMP}.tar.gz"
echo "Please attach this file to your support request"
```

## Preventive Measures

### Regular Health Checks

```bash
#!/bin/bash
# health-check.sh

# Schedule this script to run every 5 minutes via cron

echo "$(date): Starting O2 IMS health check"

# Check API health
if ! curl -f -s http://o2ims.your-domain.com/health > /dev/null; then
  echo "ALERT: O2 IMS health check failed"
  # Send alert
fi

# Check database connectivity
if ! kubectl exec deployment/o2-ims -n o2-system -- nc -zv postgresql-primary 5432 > /dev/null 2>&1; then
  echo "ALERT: Database connectivity failed"
  # Send alert
fi

# Check certificate expiry
CERT_EXPIRY=$(kubectl get certificate o2-ims-tls -n o2-system -o jsonpath='{.status.notAfter}')
DAYS_TO_EXPIRY=$(( ($(date -d "$CERT_EXPIRY" +%s) - $(date +%s)) / 86400 ))
if [ $DAYS_TO_EXPIRY -lt 30 ]; then
  echo "ALERT: Certificate expires in $DAYS_TO_EXPIRY days"
  # Send alert
fi

echo "$(date): Health check completed"
```

### Capacity Monitoring

```bash
#!/bin/bash
# capacity-monitor.sh

# Check resource utilization
CPU_USAGE=$(kubectl top pods -n o2-system --no-headers | awk '{sum+=$2} END {print sum}' | sed 's/m//')
MEMORY_USAGE=$(kubectl top pods -n o2-system --no-headers | awk '{sum+=$3} END {print sum}' | sed 's/Mi//')

# Alert thresholds
CPU_THRESHOLD=70
MEMORY_THRESHOLD=80

if [ $CPU_USAGE -gt $CPU_THRESHOLD ]; then
  echo "ALERT: High CPU usage: ${CPU_USAGE}%"
fi

if [ $MEMORY_USAGE -gt $MEMORY_THRESHOLD ]; then
  echo "ALERT: High memory usage: ${MEMORY_USAGE}%"
fi
```

## Conclusion

This troubleshooting guide covers the most common issues encountered with the O2 IMS service. Regular monitoring, proper alerting, and proactive maintenance are key to preventing issues and ensuring optimal service performance.

For additional support:
- Check the [deployment guide](../deployment/production-guide.md) for configuration best practices
- Review [API documentation](../../api/index.md) for usage details
- Submit issues to the [GitHub repository](https://github.com/nephoran/nephoran-intent-operator/issues)
