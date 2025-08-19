# Conductor Loop - Production Deployment Guide

This guide provides comprehensive instructions for deploying the Conductor Loop component in a production environment with security hardening, monitoring, and compliance features.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Security Configuration](#security-configuration)
4. [Deployment Options](#deployment-options)
5. [Configuration Management](#configuration-management)
6. [Monitoring and Alerting](#monitoring-and-alerting)
7. [Security Hardening](#security-hardening)
8. [Operations and Maintenance](#operations-and-maintenance)
9. [Troubleshooting](#troubleshooting)
10. [Compliance and Auditing](#compliance-and-auditing)

## Overview

The Conductor Loop is a critical component of the Nephoran Intent Operator that monitors intent files and orchestrates their processing through Porch. This production deployment includes:

- **Multi-stage secure container builds** with distroless base images
- **Comprehensive security scanning** and vulnerability management
- **Zero-trust network policies** and access controls
- **Production-grade monitoring** with Prometheus and Grafana
- **Automated CI/CD pipelines** with security gates
- **Compliance frameworks** (SOC2, ISO27001, GDPR)
- **Disaster recovery** and backup strategies

## Prerequisites

### Infrastructure Requirements

- **Kubernetes Cluster**: v1.28+ with Pod Security Standards enabled
- **Container Registry**: With vulnerability scanning support
- **Monitoring Stack**: Prometheus, Grafana, AlertManager
- **Security Tools**: OPA Gatekeeper, Falco (optional)
- **Certificate Management**: cert-manager for TLS certificates

### Security Requirements

- **RBAC**: Properly configured role-based access control
- **Network Policies**: Calico or similar CNI with NetworkPolicy support
- **Pod Security Standards**: Restricted profile enforcement
- **Secrets Management**: External secrets operator or Vault integration
- **Image Security**: Container image scanning and signing

### Resource Requirements

| Component | CPU | Memory | Storage | Replicas |
|-----------|-----|--------|---------|----------|
| Conductor Loop | 100m-1000m | 128Mi-512Mi | 10Gi | 2 |
| Redis Cache | 100m-500m | 64Mi-256Mi | 5Gi | 1 |
| Monitoring | 100m-1000m | 256Mi-1Gi | 50Gi | 1 |

## Security Configuration

### 1. Container Security

The Conductor Loop uses a multi-stage Dockerfile with security hardening:

```dockerfile
# Security features enabled:
# - Distroless base image (gcr.io/distroless/static-debian12:nonroot)
# - Non-root user (65532:65532)
# - Read-only root filesystem
# - No shell or package manager in runtime image
# - Minimal attack surface
```

Build the secure container:

```bash
# Production build with security scanning
docker build --target runtime \
  --build-arg VERSION=1.0.0 \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg VCS_REF=$(git rev-parse HEAD) \
  -t conductor-loop:1.0.0 \
  -f cmd/conductor-loop/Dockerfile .

# Security scan the image
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy:latest image --severity HIGH,CRITICAL conductor-loop:1.0.0
```

### 2. Kubernetes Security

Apply comprehensive security policies:

```bash
# Apply namespace with Pod Security Standards
kubectl apply -f deployments/kubernetes/production/conductor-loop-deployment.yaml

# Apply additional security policies
kubectl apply -f deployments/kubernetes/security/security-policies.yaml

# Verify security configuration
kubectl describe ns nephoran-conductor
kubectl get networkpolicies -n nephoran-conductor
kubectl get psp,podsecuritypolicy
```

### 3. RBAC Configuration

The deployment uses minimal RBAC permissions:

```yaml
# Permissions granted:
# - Read ConfigMaps (configuration only)
# - Read Secrets (credentials only)
# - Create Events (audit logging)
# - Get Pods (health checks only)

# No cluster-wide permissions
# No write access to core resources
# No exec or port-forward permissions
```

## Deployment Options

### Option 1: Kubernetes Native Deployment

For production Kubernetes environments:

```bash
# 1. Create namespace and apply security policies
kubectl apply -f deployments/kubernetes/production/conductor-loop-deployment.yaml

# 2. Verify deployment
kubectl get pods -n nephoran-conductor
kubectl logs -f deployment/conductor-loop -n nephoran-conductor

# 3. Run health checks
kubectl exec -n nephoran-conductor deployment/conductor-loop -- \
  wget --spider http://localhost:8080/healthz
```

### Option 2: Docker Compose (Development/Testing)

For local development with security features:

```bash
# Use the secure compose configuration
docker-compose -f deployments/docker-compose/docker-compose.secure.yml up -d

# Run with security scanning
docker-compose -f deployments/docker-compose/docker-compose.secure.yml \
  --profile security up --abort-on-container-exit

# Production-like deployment
docker-compose -f deployments/docker-compose/docker-compose.secure.yml \
  --profile production up -d
```

### Option 3: CI/CD Pipeline Deployment

Automated deployment through GitHub Actions:

```bash
# Trigger deployment workflow
gh workflow run conductor-loop-cicd.yml \
  -f deploy_environment=production \
  -f security_scan_level=comprehensive

# Monitor deployment status
gh run list --workflow=conductor-loop-cicd.yml
```

## Configuration Management

### 1. Application Configuration

The production configuration is located at:
- `deployments/config/production/conductor-loop-config.json`

Key security settings:

```json
{
  "security": {
    "rate_limiting": {
      "enabled": true,
      "requests_per_minute": 60,
      "burst_size": 10
    },
    "file_validation": {
      "max_file_size": "10MB",
      "max_files_per_batch": 100,
      "scan_for_malware": true
    },
    "input_sanitization": {
      "enabled": true,
      "max_input_length": 10000
    }
  }
}
```

### 2. Environment Variables

Security-sensitive environment variables are managed through Kubernetes Secrets:

```bash
# Create secrets from environment file
kubectl create secret generic conductor-loop-credentials \
  --from-env-file=.env \
  -n nephoran-conductor

# Verify secret creation
kubectl get secret conductor-loop-credentials -n nephoran-conductor -o yaml
```

### 3. ConfigMap Management

Configuration updates can be applied without pod restarts:

```bash
# Update configuration
kubectl apply -f deployments/config/production/conductor-loop-config.json

# Force configuration reload (if supported)
kubectl patch deployment conductor-loop -n nephoran-conductor \
  -p '{"spec":{"template":{"metadata":{"annotations":{"config-hash":"'$(date +%s)'"}}}}}'
```

## Monitoring and Alerting

### 1. Metrics Collection

Prometheus metrics are exposed on port 9090:

```bash
# Verify metrics endpoint
kubectl port-forward -n nephoran-conductor svc/conductor-loop 9090:9090
curl http://localhost:9090/metrics
```

Key metrics monitored:

- `conductor_loop_files_processed_total` - Total files processed
- `conductor_loop_processing_duration_seconds` - Processing time distribution
- `conductor_loop_queue_depth` - Current queue depth
- `conductor_loop_errors_total` - Error count by type
- `conductor_loop_security_events_total` - Security events

### 2. Alerting Rules

Critical alerts are configured in `deployments/monitoring/conductor-loop-alerts.yml`:

```yaml
# Sample critical alerts:
# - Service unavailable (>30s downtime)
# - High error rate (>10% errors)
# - Memory leak detection
# - Security violations
# - Queue backup (>100 files)
```

Apply alerting rules:

```bash
kubectl apply -f deployments/monitoring/conductor-loop-alerts.yml
```

### 3. Dashboards

Grafana dashboards are available for:

- **Application Performance**: Response times, throughput, errors
- **Infrastructure Metrics**: CPU, memory, disk, network
- **Security Dashboard**: Authentication failures, rate limiting, threats
- **Business Metrics**: Processing volumes, SLA compliance

## Security Hardening

### 1. Runtime Security

Security measures implemented:

```yaml
# Container security context
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  runAsGroup: 65532
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: [ALL]
  seccompProfile:
    type: RuntimeDefault
```

### 2. Network Security

Network policies provide micro-segmentation:

```bash
# Verify network policies
kubectl get networkpolicies -n nephoran-conductor

# Test network connectivity
kubectl run test-pod --rm -i --image=busybox \
  -- nc -zv conductor-loop.nephoran-conductor.svc.cluster.local 8080
```

### 3. Image Security

Container images are scanned for vulnerabilities:

```bash
# Scan during CI/CD
govulncheck ./...
gosec -severity medium ./...
trivy image conductor-loop:1.0.0

# Policy enforcement with OPA Gatekeeper
kubectl apply -f deployments/security/opa-policies/
```

### 4. Secrets Management

Sensitive data is properly protected:

```bash
# Secrets are never logged
# Environment variables are encrypted at rest
# TLS certificates are automatically rotated
# Database credentials use short-lived tokens
```

## Operations and Maintenance

### 1. Health Monitoring

Health check endpoints:

- `/healthz` - Liveness probe (port 8080)
- `/ready` - Readiness probe (port 8080)
- `/startup` - Startup probe (port 8080)

Monitor health status:

```bash
# Check pod health
kubectl get pods -n nephoran-conductor -o wide

# View health check logs
kubectl logs -n nephoran-conductor deployment/conductor-loop | grep health
```

### 2. Log Management

Structured logging with security audit:

```bash
# View application logs
kubectl logs -n nephoran-conductor deployment/conductor-loop -f

# Search for security events
kubectl logs -n nephoran-conductor deployment/conductor-loop | \
  grep -E "security|auth|error"

# Export logs for analysis
kubectl logs -n nephoran-conductor deployment/conductor-loop \
  --since=1h > conductor-loop-logs.json
```

### 3. Backup and Recovery

Data backup strategy:

```bash
# Backup persistent data
kubectl exec -n nephoran-conductor deployment/conductor-loop -- \
  tar -czf /tmp/backup-$(date +%Y%m%d).tar.gz /data

# Test backup restoration
kubectl cp nephoran-conductor/conductor-loop-pod:/tmp/backup.tar.gz ./backup.tar.gz
```

### 4. Scaling and Performance

Horizontal scaling configuration:

```yaml
# HPA configuration
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: conductor-loop-hpa
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
```

## Troubleshooting

### Common Issues

#### 1. Pod Startup Failures

```bash
# Check pod status and events
kubectl describe pod -n nephoran-conductor conductor-loop-xxx

# Common causes:
# - Image pull failures (check registry access)
# - Configuration errors (validate ConfigMap/Secrets)
# - Resource constraints (check resource quotas)
# - Security policy violations (check PSP/PSS)
```

#### 2. File Processing Issues

```bash
# Check file permissions
kubectl exec -n nephoran-conductor deployment/conductor-loop -- \
  ls -la /data/handoff/

# Verify Porch connectivity
kubectl exec -n nephoran-conductor deployment/conductor-loop -- \
  wget -qO- http://porch-server:7007/health

# Monitor processing metrics
curl http://localhost:9090/metrics | grep conductor_loop_
```

#### 3. Security Policy Violations

```bash
# Check security events
kubectl get events -n nephoran-conductor --field-selector type=Warning

# Review network policy denials
kubectl logs -n kube-system deployment/calico-node | grep denied

# Validate RBAC permissions
kubectl auth can-i --list --as=system:serviceaccount:nephoran-conductor:conductor-loop
```

### Performance Optimization

#### 1. Resource Tuning

```bash
# Monitor resource usage
kubectl top pods -n nephoran-conductor

# Adjust resource limits based on usage
kubectl patch deployment conductor-loop -n nephoran-conductor \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"conductor-loop","resources":{"limits":{"memory":"1Gi"}}}]}}}}'
```

#### 2. Concurrency Tuning

Adjust worker count based on workload:

```json
{
  "file_handling": {
    "max_workers": 4,
    "debounce_duration": "250ms"
  }
}
```

## Compliance and Auditing

### 1. Security Compliance

The deployment satisfies multiple compliance frameworks:

- **SOC 2 Type II**: Access controls, data protection, monitoring
- **ISO 27001**: Information security management system
- **GDPR**: Data protection and privacy requirements

### 2. Audit Logging

Comprehensive audit trail:

```bash
# View audit logs
kubectl logs -n nephoran-conductor deployment/conductor-loop | \
  jq 'select(.level == "audit")'

# Export audit data
kubectl logs -n nephoran-conductor deployment/conductor-loop \
  --since=24h | jq 'select(.audit == true)' > audit-$(date +%Y%m%d).json
```

### 3. Compliance Validation

Regular compliance scanning:

```bash
# Run CIS Kubernetes benchmark
kubectl apply -f deployments/security/security-config.yaml

# Monitor compliance job
kubectl logs -n nephoran-conductor job/security-compliance-scan
```

### 4. Security Reporting

Generate security reports:

```bash
# Vulnerability report
trivy image --format json conductor-loop:1.0.0 > vulnerability-report.json

# Security metrics export
curl http://localhost:9090/api/v1/query?query=conductor_loop_security_events_total

# Compliance status
kubectl get configmap production-readiness-checklist -n nephoran-conductor -o yaml
```

## Production Readiness Checklist

Before going to production, verify:

### Security ✓
- [ ] Security policies configured and enforced
- [ ] RBAC permissions minimized and tested
- [ ] Network policies implemented and validated
- [ ] Secrets properly managed and rotated
- [ ] Container security hardened (non-root, read-only, etc.)
- [ ] Vulnerability scanning enabled and passing
- [ ] Security monitoring and alerting configured

### Reliability ✓
- [ ] Health checks configured and responding
- [ ] Resource limits and requests properly set
- [ ] Pod disruption budget configured
- [ ] High availability configured (multiple replicas)
- [ ] Backup strategy implemented and tested
- [ ] Disaster recovery procedures documented and tested

### Monitoring ✓
- [ ] Metrics collection enabled and validated
- [ ] Alerting rules configured and tested
- [ ] Dashboards deployed and accessible
- [ ] Log aggregation setup and retention configured
- [ ] Performance baselines established

### Compliance ✓
- [ ] Audit logging enabled and validated
- [ ] Data retention policies configured
- [ ] Privacy controls implemented
- [ ] Compliance frameworks validated
- [ ] Documentation completed and reviewed

---

## Support and Contact

For deployment support and questions:

- **Documentation**: https://docs.nephoran.com/conductor-loop
- **Issues**: https://github.com/thc1006/nephoran-intent-operator/issues
- **Security**: security@nephoran.com
- **Operations**: ops@nephoran.com

---

**Last Updated**: 2025-01-01  
**Version**: 1.0.0  
**Deployment Guide Version**: 1.0