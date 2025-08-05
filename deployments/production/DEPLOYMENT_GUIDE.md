# Nephoran Intent Operator - Production Deployment Guide

This guide provides comprehensive instructions for deploying the Nephoran Intent Operator with enhanced O-RAN components in production environments.

## Overview

The production deployment includes:

- **Enhanced O-RAN Adaptor**: Complete A1, E2, O1, O2 interface support
- **Resource Limits & Security**: Production-grade resource constraints and security contexts
- **High Availability**: Pod disruption budgets, anti-affinity rules, and horizontal autoscaling
- **Monitoring & Observability**: Prometheus metrics, Grafana dashboards, distributed tracing
- **Security**: Network policies, RBAC, TLS encryption, read-only filesystems
- **Resilience**: Circuit breakers, retry mechanisms, health checks

## Prerequisites

### Kubernetes Cluster Requirements

- **Kubernetes Version**: 1.24+
- **Cluster Resources**: Minimum 8 CPU cores, 16GB RAM across worker nodes
- **Storage**: Support for persistent volumes (for monitoring components)
- **Networking**: CNI with NetworkPolicy support (Calico, Cilium, etc.)

### Required Components

```bash
# Install cert-manager for TLS certificate management
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Install Prometheus Operator for monitoring
kubectl apply -f https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.68.0/bundle.yaml

# Install KEDA for advanced autoscaling (optional)
kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.12.0/keda-2.12.0.yaml
```

### Node Labels

Label nodes for optimal scheduling:

```bash
# Label nodes for O-RAN workloads
kubectl label nodes <node-name> nephoran.com/node-type=oran-processing
kubectl label nodes <node-name> nephoran.com/oran-workload=true

# Label nodes for AI processing (if separate)
kubectl label nodes <node-name> nephoran.com/node-type=ai-processing
```

## Deployment Steps

### 1. Create Namespace and Basic Resources

```bash
# Apply the complete O-RAN deployment
kubectl apply -f deployments/production/oran-complete-deployment.yaml
```

This creates:
- `nephoran-oran` namespace with resource quotas and limits
- Service accounts and RBAC configurations
- Network policies for security
- Priority classes for scheduling

### 2. Configure Secrets (Production)

#### Git Integration Secrets

Configure Git authentication for GitOps workflows:

```bash
# Create Git token secret for GitOps integration
kubectl create secret generic git-token-secret \
  --from-literal=token="your-github-personal-access-token" \
  -n nephoran-oran

# Verify the secret was created
kubectl get secret git-token-secret -n nephoran-oran -o yaml
```

#### O-RAN Interface Secrets

Update the secrets with production values:

```bash
# Edit the credentials secret
kubectl edit secret oran-credentials -n nephoran-oran

# Update with actual production credentials:
# - o1_username/o1_password: O1 interface credentials
# - a1_api_key: A1 RIC authentication
# - e2_api_key: E2 RIC authentication
# - o2_api_key: O2 IMS authentication
# - ric_auth_token: RIC service authentication
# - ims_auth_token: IMS service authentication
```

### 3. Configure TLS Certificates

For production, use cert-manager to manage certificates:

```yaml
# Create a ClusterIssuer for internal CA
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: oran-ca-issuer
spec:
  ca:
    secretName: oran-ca-key-pair
---
# Create a Certificate resource
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: oran-adaptor-tls
  namespace: nephoran-oran
spec:
  secretName: oran-tls-certs
  issuerRef:
    name: oran-ca-issuer
    kind: ClusterIssuer
  commonName: oran-adaptor.nephoran-oran.svc.cluster.local
  dnsNames:
  - oran-adaptor
  - oran-adaptor.nephoran-oran
  - oran-adaptor.nephoran-oran.svc
  - oran-adaptor.nephoran-oran.svc.cluster.local
```

### 4. Configure GitOps Integration

Configure the nephio-bridge deployment to use Git token secrets:

```bash
# Update the nephio-bridge deployment with Git configuration
kubectl patch deployment nephio-bridge -n nephoran-oran --patch '
spec:
  template:
    spec:
      containers:
      - name: nephio-bridge
        env:
        - name: GIT_REPO_URL
          value: "https://github.com/your-org/your-control-repo.git"
        - name: GIT_TOKEN_PATH
          value: "/etc/git-secret/token"
        - name: GIT_BRANCH
          value: "main"
        volumeMounts:
        - name: git-token
          mountPath: "/etc/git-secret"
          readOnly: true
      volumes:
      - name: git-token
        secret:
          secretName: git-token-secret
          defaultMode: 0400
'

# Verify the patch was applied
kubectl describe deployment nephio-bridge -n nephoran-oran
```

### 5. Configure External Services

Update the ConfigMap with actual O-RAN component endpoints:

```bash
kubectl edit configmap oran-adaptor-config -n nephoran-oran

# Update these values:
# - a1_ric_url: Actual A1 RIC endpoint
# - e2_ric_url: Actual E2 RIC endpoint  
# - o1_netconf_host: Actual O1 managed element
# - o2_ims_url: Actual O2 IMS endpoint
```

### 6. Verify Deployment

```bash
# Check deployment status
kubectl get deployment oran-adaptor -n nephoran-oran

# Check pod status
kubectl get pods -n nephoran-oran -l app=oran-adaptor

# Check service status
kubectl get svc -n nephoran-oran

# Check HPA status
kubectl get hpa -n nephoran-oran

# Check logs
kubectl logs -n nephoran-oran -l app=oran-adaptor -f
```

## Configuration

### Resource Scaling

The deployment includes automatic scaling based on:

- **CPU Utilization**: Target 70%
- **Memory Utilization**: Target 80%  
- **Custom Metrics**: O-RAN specific metrics (connections, throughput, error rate)

Default scaling parameters:
- **Min Replicas**: 3 (production)
- **Max Replicas**: 20
- **Scale Up**: Fast (30s stabilization)
- **Scale Down**: Conservative (300s stabilization)

### Security Configuration

The deployment implements multiple security layers:

1. **Pod Security**:
   - Non-root user (UID 65532)
   - Read-only root filesystem
   - Dropped capabilities
   - SecComp profiles

2. **Network Security**:
   - NetworkPolicies restricting ingress/egress
   - TLS encryption for all interfaces
   - mTLS support (configurable)

3. **RBAC**:
   - Minimal required permissions
   - Service account isolation
   - Cluster-level resource access restrictions

### Monitoring Configuration

The deployment includes comprehensive monitoring:

1. **Metrics Collection**:
   - Prometheus ServiceMonitor
   - Custom O-RAN metrics
   - Resource utilization metrics
   - Health check metrics

2. **Dashboards**:
   - Grafana dashboard for O-RAN interfaces
   - Circuit breaker state monitoring
   - Throughput and error rate visualization

3. **Alerting** (configure separately):
   ```yaml
   # Example PrometheusRule for alerting
   apiVersion: monitoring.coreos.com/v1
   kind: PrometheusRule
   metadata:
     name: oran-adaptor-alerts
     namespace: nephoran-oran
   spec:
     groups:
     - name: oran-adaptor
       rules:
       - alert: ORANAdaptorDown
         expr: up{job="oran-adaptor"} == 0
         for: 1m
         labels:
           severity: critical
         annotations:
           summary: "O-RAN Adaptor is down"
       - alert: ORANHighErrorRate
         expr: nephoran_oran_error_rate_percent > 10
         for: 5m
         labels:
           severity: warning
         annotations:
           summary: "High error rate in O-RAN interface"
   ```

## Performance Tuning

### Resource Optimization

For different environments, adjust resources:

```yaml
# Development Environment
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi

# Staging Environment  
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

# Production Environment (current)
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi
```

### JVM Tuning (if applicable)

For Java-based components:

```yaml
env:
- name: JAVA_OPTS
  value: "-Xms512m -Xmx1536m -XX:+UseG1GC -XX:MaxGCPauseMillis=200"
- name: JVM_HEAP_SIZE
  value: "1536m"
```

## Troubleshooting

### Common Issues

1. **Pod Scheduling Issues**:
   ```bash
   # Check node resources
   kubectl describe nodes
   
   # Check pod events
   kubectl describe pod <pod-name> -n nephoran-oran
   
   # Check taints and tolerations
   kubectl get nodes -o json | jq '.items[].spec.taints'
   ```

2. **Network Connectivity**:
   ```bash
   # Test service connectivity
   kubectl exec -n nephoran-oran <pod-name> -- curl -k https://a1-ric.oran-production.svc.cluster.local:8443/healthz
   
   # Check NetworkPolicy
   kubectl describe networkpolicy -n nephoran-oran
   ```

3. **Certificate Issues**:
   ```bash
   # Check certificate status
   kubectl get certificate -n nephoran-oran
   
   # Check cert-manager logs
   kubectl logs -n cert-manager deploy/cert-manager
   ```

4. **Git Integration Issues**:
   ```bash
   # Check if Git token secret is mounted correctly
   kubectl exec -n nephoran-oran deployment/nephio-bridge -- ls -la /etc/git-secret/
   
   # Verify Git token is readable
   kubectl exec -n nephoran-oran deployment/nephio-bridge -- cat /etc/git-secret/token | wc -c
   
   # Test Git repository access
   kubectl exec -n nephoran-oran deployment/nephio-bridge -- \
     curl -H "Authorization: token $(cat /etc/git-secret/token)" \
     https://api.github.com/user
   
   # Check Git-related environment variables
   kubectl exec -n nephoran-oran deployment/nephio-bridge -- env | grep GIT
   ```

5. **Performance Issues**:
   ```bash
   # Check resource usage
   kubectl top pods -n nephoran-oran
   
   # Check HPA status
   kubectl describe hpa -n nephoran-oran
   
   # Check metrics
   kubectl port-forward -n nephoran-oran svc/oran-adaptor 8082:8082
   curl http://localhost:8082/metrics
   ```

### Health Checks

The deployment includes comprehensive health checks:

1. **Startup Probe**: 5-minute window for slow startup
2. **Liveness Probe**: Detects deadlocks and restarts pods
3. **Readiness Probe**: Ensures traffic only goes to ready pods

Health check endpoints:
- `/healthz`: Liveness check
- `/readyz`: Readiness check  
- `/metrics`: Prometheus metrics

### Log Analysis

```bash
# View real-time logs
kubectl logs -n nephoran-oran -l app=oran-adaptor -f

# View logs from all replicas
kubectl logs -n nephoran-oran -l app=oran-adaptor --prefix=true

# View previous container logs
kubectl logs -n nephoran-oran <pod-name> --previous

# Search for specific errors
kubectl logs -n nephoran-oran -l app=oran-adaptor | grep -i error
```

## Maintenance

### Rolling Updates

```bash
# Update image version
kubectl set image deployment/oran-adaptor -n nephoran-oran oran-adaptor=thc1006/nephoran-oran-adaptor:v1.1.0

# Check rollout status
kubectl rollout status deployment/oran-adaptor -n nephoran-oran

# Rollback if needed
kubectl rollout undo deployment/oran-adaptor -n nephoran-oran
```

### Backup and Recovery

```bash
# Backup configurations
kubectl get all,configmap,secret,networkpolicy,pdb,hpa,servicemonitor -n nephoran-oran -o yaml > oran-backup.yaml

# Restore from backup
kubectl apply -f oran-backup.yaml
```

### Scaling Operations

```bash
# Manual scaling
kubectl scale deployment oran-adaptor -n nephoran-oran --replicas=5

# Update HPA limits
kubectl patch hpa oran-adaptor-hpa -n nephoran-oran -p '{"spec":{"maxReplicas":30}}'
```

## Security Hardening

### Additional Security Measures

1. **Pod Security Standards**:
   ```yaml
   apiVersion: v1
   kind: Namespace
   metadata:
     name: nephoran-oran
     labels:
       pod-security.kubernetes.io/enforce: restricted
       pod-security.kubernetes.io/audit: restricted
       pod-security.kubernetes.io/warn: restricted
   ```

2. **Runtime Security**:
   - Consider using Falco for runtime security monitoring
   - Implement image scanning in CI/CD pipeline
   - Regular security updates for base images

3. **Network Segmentation**:
   - Use Istio service mesh for advanced traffic management
   - Implement zero-trust networking principles
   - Regular NetworkPolicy audits

## Integration with External Systems

### A1 Interface Integration

```yaml
# A1 RIC Configuration
a1_ric_url: "https://a1-ric.oran-production.svc.cluster.local:8443"
a1_api_version: "v1"
a1_timeout: "30s"
a1_tls_enabled: "true"
a1_mtls_enabled: "true"
```

### E2 Interface Integration

```yaml
# E2 RIC Configuration
e2_ric_url: "https://e2-ric.oran-production.svc.cluster.local:38443"
e2_timeout: "30s"
e2_heartbeat_interval: "30s"
e2_tls_enabled: "true"
e2_mtls_enabled: "true"
```

### O1 Interface Integration

```yaml
# O1 NETCONF Configuration
o1_netconf_host: "o1-managed-element.oran-production.svc.cluster.local"
o1_netconf_port: "830"
o1_tls_enabled: "true"
o1_yang_models_path: "/etc/yang-models"
```

### O2 Interface Integration

```yaml
# O2 IMS Configuration
o2_ims_url: "https://o2-ims.oran-production.svc.cluster.local:8443"
o2_api_version: "v1"
o2_timeout: "60s"
o2_tls_enabled: "true"
o2_mtls_enabled: "true"
```

## Support and Documentation

For additional support:

1. **Logs**: Enable debug logging for detailed troubleshooting
2. **Metrics**: Use Prometheus and Grafana for monitoring
3. **Tracing**: Configure Jaeger for distributed tracing
4. **Health Checks**: Monitor all health endpoints

## Conclusion

This production deployment provides a robust, scalable, and secure foundation for the Nephoran Intent Operator's O-RAN components. Regular monitoring, maintenance, and security updates are essential for optimal performance in production environments.