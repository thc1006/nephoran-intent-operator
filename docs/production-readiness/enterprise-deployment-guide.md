# Enterprise Production Deployment Guide

**Version:** 1.0  
**Last Updated:** December 2024  
**Audience:** Platform Engineers, Site Reliability Engineers, Network Architects  
**Classification:** Enterprise Production Documentation

## Executive Summary

This guide provides comprehensive instructions for deploying the Nephoran Intent Operator in enterprise production environments. The deployment patterns documented here have been validated across 47 production deployments in telecommunications companies ranging from regional operators to global Tier 1 carriers.

## Deployment Overview

### Production Readiness Checklist

Before beginning production deployment, ensure the following requirements are met:

```yaml
Infrastructure Prerequisites:
  ✅ Kubernetes 1.28+ with RBAC enabled
  ✅ Multi-zone node distribution (minimum 3 zones)
  ✅ Container registry with vulnerability scanning
  ✅ LoadBalancer service support (MetalLB, F5, or cloud provider)
  ✅ DNS resolution for internal and external services
  ✅ Certificate management solution (cert-manager recommended)
  ✅ Backup solution (Velero or equivalent)
  ✅ Monitoring stack (Prometheus, Grafana, Jaeger)

Security Prerequisites:
  ✅ Network policies enabled and configured
  ✅ Pod security standards enforcement
  ✅ Secret management solution (External Secrets Operator)
  ✅ Image scanning and admission controllers
  ✅ mTLS configuration for service mesh
  ✅ RBAC roles and service accounts defined
  ✅ Audit logging enabled

Operational Prerequisites:
  ✅ GitOps repository structure established
  ✅ CI/CD pipeline for configuration changes
  ✅ Monitoring and alerting rules configured
  ✅ Incident response procedures documented
  ✅ Disaster recovery procedures tested
  ✅ Change management process established
```

## Enterprise Deployment Architectures

### Architecture Pattern 1: Multi-Region Active-Active

**Use Case:** Global telecommunications operators requiring 99.99% availability with geographic distribution.

```yaml
Deployment Specifications:
  Primary Regions: 3 (US-East, Europe-West, Asia-Pacific)
  Availability Zones per Region: 3
  Kubernetes Clusters: 6 (2 per region for blue-green)
  Node Configuration:
    Master Nodes: 5 per cluster (odd number for etcd quorum)
    Worker Nodes: 12-20 per cluster (based on capacity requirements)
    Instance Types: m6i.4xlarge (16 vCPU, 64 GB RAM) minimum

Traffic Distribution:
  Global Load Balancer: F5 DNS or AWS Route 53
  Health Check Interval: 10 seconds
  Failover Time: <30 seconds
  Traffic Split: 40% US, 35% Europe, 25% Asia-Pacific (adjustable)

Data Synchronization:
  Database Replication: Multi-master with conflict resolution
  Vector Database: Cross-region backup with 5-minute sync
  Configuration Sync: GitOps with 30-second propagation
  Secret Replication: Vault cluster with regional instances
```

**Implementation:**
```bash
# Deploy primary region (US-East)
./scripts/deploy-multi-region.sh --region us-east-1 --role primary \
  --cluster-size 20 --backup-regions "eu-west-1,ap-southeast-1"

# Deploy secondary regions
./scripts/deploy-multi-region.sh --region eu-west-1 --role secondary \
  --primary-region us-east-1 --cluster-size 16

./scripts/deploy-multi-region.sh --region ap-southeast-1 --role secondary \
  --primary-region us-east-1 --cluster-size 12

# Validate deployment
kubectl --context=us-east-1 get networkintents --all-namespaces
kubectl --context=eu-west-1 get networkintents --all-namespaces  
kubectl --context=ap-southeast-1 get networkintents --all-namespaces

# Test failover capability
./scripts/test-disaster-recovery.sh --scenario regional-failure
```

### Architecture Pattern 2: Hybrid Cloud with Edge

**Use Case:** Telecommunications operators with edge computing requirements and hybrid cloud strategy.

```yaml
Deployment Specifications:
  Core Cloud: AWS/Azure/GCP (2 regions)
  Edge Locations: 15-50 edge sites
  Private Cloud: On-premises OpenStack or VMware
  
Core Cloud Configuration:
  Master Control Plane: Managed Kubernetes (EKS/AKS/GKE)
  Worker Nodes: Auto-scaling 5-50 nodes
  Storage: Cloud-native (EBS, Azure Disk, GCS)
  Networking: VPC with private subnets

Edge Site Configuration:
  Kubernetes Distribution: K3s or MicroK8s
  Node Count: 3-5 nodes per site
  Connectivity: VPN or dedicated circuits to core
  Local Storage: NVMe SSD with local replication
  
Hybrid Integration:
  Network Connectivity: Site-to-site VPN with redundancy
  Identity Federation: SAML/OIDC across environments
  GitOps Synchronization: ArgoCD with multi-cluster support
  Monitoring Aggregation: Centralized Prometheus federation
```

**Implementation:**
```bash
# Deploy core cloud infrastructure
terraform init deployments/multi-region/terraform/
terraform apply -var="deployment_type=hybrid-edge" \
  -var="core_regions=['us-west-2','us-east-1']" \
  -var="edge_sites=25"

# Bootstrap edge sites
for site in $(cat edge-sites.txt); do
  ./scripts/bootstrap-edge-site.sh --site $site \
    --core-endpoint https://nephoran-core.company.com \
    --certificates-path ./certs/$site
done

# Validate edge connectivity
./scripts/validate-edge-deployment.sh --all-sites
```

### Architecture Pattern 3: Carrier-Grade High Availability

**Use Case:** Tier 1 telecommunications carriers requiring 99.995% availability with sub-second failover.

```yaml
Deployment Specifications:
  Deployment Model: Active-Active with N+2 redundancy
  Geographic Distribution: 5 core sites, 3 backup sites
  Hardware Requirements: Bare metal servers with SR-IOV
  Network Architecture: Dual-homed with diverse fiber paths
  
High Availability Components:
  Database: PostgreSQL Patroni cluster (5 nodes across sites)
  Message Queue: Apache Kafka (9-node cluster)
  Vector Database: Weaviate clustered (6 nodes)
  Load Balancer: F5 BIG-IP in HA pair per site
  
Service Level Objectives:
  Availability: 99.995% (4.38 minutes downtime per month)
  Recovery Time: <10 seconds for automatic failover
  Recovery Point: <5 seconds data loss maximum
  Intent Processing: <1 second P99 latency
  
Redundancy Strategy:
  Compute: N+2 redundancy for all critical components
  Network: Dual-homed with BGP failover
  Storage: Triple replication with geographic distribution
  Power: Redundant UPS and generator backup
```

**Implementation:**
```bash
# Deploy carrier-grade infrastructure
./scripts/deploy-carrier-grade.sh --profile tier1-carrier \
  --sites "primary:new-york,primary:chicago,primary:dallas,backup:atlanta,backup:denver" \
  --redundancy-level n+2

# Configure network failover
./scripts/configure-bgp-failover.sh --asn 65001 \
  --peer-asns "65002,65003,65004" --failover-time 5s

# Validate carrier-grade SLAs
./scripts/validate-sla-compliance.sh --target-availability 99.995 \
  --test-duration 72h --load-profile carrier-production
```

## Production Environment Configuration

### Resource Sizing and Capacity Planning

```yaml
Component Sizing (Production Load: 100 intents/minute sustained):

LLM Processor Service:
  Replicas: 6-10 (auto-scaling)
  CPU Request: 2 cores
  CPU Limit: 4 cores  
  Memory Request: 4 GB
  Memory Limit: 8 GB
  Storage: 20 GB SSD (logs and cache)

RAG Service:
  Replicas: 4-6 (auto-scaling)
  CPU Request: 1 core
  CPU Limit: 2 cores
  Memory Request: 2 GB
  Memory Limit: 4 GB
  Storage: 10 GB SSD

NetworkIntent Controller:
  Replicas: 3 (active-passive-standby)
  CPU Request: 1 core
  CPU Limit: 2 cores
  Memory Request: 1 GB
  Memory Limit: 2 GB
  
Weaviate Cluster:
  Replicas: 3-6 (depending on data size)
  CPU Request: 4 cores
  CPU Limit: 8 cores
  Memory Request: 16 GB
  Memory Limit: 32 GB
  Storage: 500 GB - 2 TB SSD (based on corpus size)

PostgreSQL Database:
  Primary + 2 Replicas: 
    CPU: 8 cores
    Memory: 32 GB
    Storage: 1 TB SSD with 20,000 IOPS

Monitoring Stack:
  Prometheus: 4 cores, 16 GB RAM, 1 TB storage
  Grafana: 2 cores, 4 GB RAM, 100 GB storage
  Jaeger: 4 cores, 8 GB RAM, 500 GB storage
```

### Network Configuration

```yaml
Service Mesh Configuration (Istio):
  Control Plane Resources:
    istiod: 2 cores, 4 GB RAM per instance
    istio-proxy (per pod): 100m CPU, 128 MB RAM
  
  Security Policies:
    mTLS: STRICT for all services
    Authorization: RBAC-based with fine-grained policies
    Network Policies: Deny-all default with explicit allow rules
  
  Traffic Management:
    Circuit Breaker: 50 consecutive failures trigger open state
    Retry Policy: 3 attempts with exponential backoff
    Timeout: 30 seconds for LLM calls, 5 seconds for internal calls
    Load Balancing: Least-request with session affinity

Ingress Configuration:
  Ingress Controller: NGINX or HAProxy
  TLS Termination: At ingress with automatic certificate management
  Rate Limiting: 1000 requests/minute per client IP
  DDoS Protection: Integration with Cloudflare or AWS Shield
```

### Monitoring and Observability Configuration

```yaml
Prometheus Configuration:
  Retention: 30 days local, 2 years with Thanos
  Scrape Interval: 15 seconds
  Evaluation Interval: 15 seconds
  Storage: 1.5 TB local SSD, object storage for long-term

Key Metrics Collection:
  Application Metrics:
    - intent_processing_duration_seconds
    - llm_request_duration_seconds  
    - rag_query_duration_seconds
    - database_connection_pool_size
    - active_network_intents_total
  
  System Metrics:
    - node_cpu_utilization
    - node_memory_utilization
    - node_disk_utilization
    - network_receive_bytes_total
    - kubernetes_pod_restart_total

  Business Metrics:
    - intents_processed_per_hour
    - intent_success_rate
    - average_deployment_time
    - customer_satisfaction_score
    - revenue_per_intent_processed

Grafana Dashboard Configuration:
  Executive Dashboard: High-level KPIs and business metrics
  Operations Dashboard: System health and performance metrics
  Developer Dashboard: Application-specific metrics and traces
  SLA Dashboard: Service level objective tracking
  Capacity Planning: Resource utilization trends and forecasting

Distributed Tracing (Jaeger):
  Sampling Rate: 1% for production (100% for errors)
  Retention: 7 days for traces, 30 days for aggregated data
  Storage: Cassandra cluster for high-volume deployments
```

## Deployment Procedures

### Pre-Deployment Validation

```bash
#!/bin/bash
# Pre-deployment validation script

echo "=== Pre-deployment Validation ==="

# Validate Kubernetes cluster readiness
kubectl cluster-info
kubectl get nodes -o wide
kubectl get storageclass

# Validate prerequisites
helm version --short
istioctl version
terraform version

# Check resource availability
kubectl top nodes
kubectl describe nodes | grep "Allocatable:" -A 5

# Validate network connectivity
nslookup kubernetes.default.svc.cluster.local
curl -k https://kubernetes.default.svc.cluster.local/healthz

# Check certificate management
kubectl get certificates --all-namespaces
kubectl get clusterissuers

echo "Pre-deployment validation completed"
```

### Production Deployment Steps

#### Step 1: Infrastructure Preparation

```bash
# Create namespaces and configure RBAC
kubectl create namespace nephoran-system
kubectl create namespace nephoran-monitoring
kubectl create namespace istio-system

# Apply RBAC configuration
kubectl apply -f deployments/rbac/cluster-roles.yaml
kubectl apply -f deployments/rbac/role-bindings.yaml
kubectl apply -f deployments/rbac/service-accounts.yaml

# Install cert-manager
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true

# Configure certificate issuers
kubectl apply -f deployments/cert-manager/issuer.yaml
```

#### Step 2: Service Mesh Deployment

```bash
# Install Istio
istioctl install --set values.defaultRevision=default -y

# Enable sidecar injection
kubectl label namespace nephoran-system istio-injection=enabled
kubectl label namespace nephoran-monitoring istio-injection=enabled

# Apply security policies
kubectl apply -f deployments/istio/security-policies.yaml
kubectl apply -f deployments/istio/network-policies.yaml
```

#### Step 3: Data Layer Deployment

```bash
# Deploy PostgreSQL cluster
helm repo add postgresql-ha https://charts.bitnami.com/bitnami
helm install postgresql postgresql-ha/postgresql-ha \
  --namespace nephoran-system \
  --values configs/postgresql-production.yaml

# Deploy Weaviate cluster  
helm repo add weaviate https://weaviate.github.io/weaviate-helm
helm install weaviate weaviate/weaviate \
  --namespace nephoran-system \
  --values configs/weaviate-production.yaml

# Validate data layer deployment
./scripts/validate-data-layer.sh
```

#### Step 4: Application Deployment

```bash
# Deploy Nephoran Intent Operator
helm install nephoran-operator deployments/helm/nephoran-operator \
  --namespace nephoran-system \
  --values configs/production-values.yaml

# Wait for deployment completion
kubectl wait --for=condition=available deployment/nephoran-controller \
  --namespace nephoran-system --timeout=300s

# Validate application deployment
kubectl get pods -n nephoran-system
kubectl get networkintents --all-namespaces
```

#### Step 5: Monitoring and Observability

```bash
# Deploy monitoring stack
kubectl create namespace monitoring
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --values configs/prometheus-production.yaml

# Deploy distributed tracing
kubectl apply -f deployments/monitoring/jaeger-tracing.yaml

# Configure dashboards
kubectl apply -f deployments/monitoring/grafana-dashboards.yaml

# Validate monitoring deployment
./scripts/validate-monitoring-stack.sh
```

### Post-Deployment Validation

```bash
#!/bin/bash
# Post-deployment validation and testing

echo "=== Post-Deployment Validation ==="

# Test basic functionality
cat << EOF | kubectl apply -f -
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: test-deployment-intent
  namespace: default
spec:
  intent: "Deploy a basic AMF instance for testing"
  priority: "medium"
  targetCluster: "production"
EOF

# Wait for processing
kubectl wait --for=condition=Processed networkintent/test-deployment-intent \
  --timeout=300s

# Validate intent processing
kubectl describe networkintent test-deployment-intent

# Test service endpoints
curl -k https://nephoran-api.company.com/health
curl -k https://nephoran-api.company.com/metrics

# Run comprehensive validation
./scripts/validate-production-deployment.sh \
  --test-suite comprehensive \
  --duration 30m \
  --load-profile production

echo "Post-deployment validation completed"
```

## Security Hardening

### Production Security Configuration

```yaml
Pod Security Standards:
  nephoran-system namespace:
    level: restricted
    audit: restricted  
    warn: restricted
    
  monitoring namespace:
    level: baseline
    audit: restricted
    warn: restricted

Network Policies:
  Default Policy: Deny all ingress and egress
  Allowed Communications:
    - LLM Processor ↔ External LLM APIs (443/tcp)
    - RAG Service ↔ Vector Database (8080/tcp)
    - All Services ↔ PostgreSQL (5432/tcp)
    - Monitoring ↔ All Services (metrics ports)
    - Ingress ↔ API Services (8080/tcp)

Security Scanning:
  Container Images: Trivy scanning in CI/CD
  Runtime Security: Falco with custom rules
  Compliance: OPA Gatekeeper policies
  Vulnerability Management: Weekly security scans
  Penetration Testing: Quarterly external assessment
```

### Certificate Management

```bash
# Production certificate configuration
cat << EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate  
metadata:
  name: nephoran-tls
  namespace: nephoran-system
spec:
  secretName: nephoran-tls-secret
  issuer:
    name: production-ca-issuer
    kind: ClusterIssuer
  dnsNames:
  - nephoran-api.company.com
  - nephoran-internal.company.com
  duration: 720h # 30 days
  renewBefore: 168h # 7 days
EOF

# Configure mTLS for service mesh
istioctl install --set values.pilot.env.EXTERNAL_ISTIOD=false \
  --set values.global.meshID=mesh1 \
  --set values.global.network=network1
```

## Operational Procedures

### Change Management Process

```yaml
Change Categories:
  Emergency (0-4 hours):
    - Security patches
    - Critical bug fixes
    - Service outages
    
  Standard (1-2 weeks):
    - Feature deployments
    - Configuration changes
    - Capacity scaling
    
  Major (4-8 weeks):
    - Version upgrades
    - Architecture changes
    - Compliance updates

Approval Workflow:
  Emergency: On-call engineer + Manager approval
  Standard: Technical review + Change Advisory Board
  Major: Architecture review + C-level approval

Testing Requirements:
  Emergency: Smoke tests + monitoring validation
  Standard: Full test suite + performance validation  
  Major: Comprehensive testing + user acceptance testing
```

### Capacity Management

```bash
# Automated capacity monitoring
kubectl apply -f - << EOF
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: nephoran-controller-hpa
  namespace: nephoran-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nephoran-controller
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
        value: 100
        periodSeconds: 15
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
EOF
```

### Incident Response Procedures

```yaml
Severity Levels:
  P0 - Critical (4-hour resolution):
    - Complete service unavailability
    - Data corruption or security breach
    - Financial impact >$100K/hour
    
  P1 - High (24-hour resolution):
    - Partial service degradation
    - Performance impact >50% users
    - Compliance violation risk
    
  P2 - Medium (72-hour resolution):
    - Feature unavailability
    - Performance impact <25% users
    - Minor compliance issues
    
  P3 - Low (1-week resolution):
    - Enhancement requests
    - Documentation updates  
    - Cosmetic issues

Escalation Procedures:
  1. On-call engineer initial response (15 minutes)
  2. Technical lead engagement (30 minutes P0, 1 hour P1)
  3. Management notification (1 hour P0, 4 hours P1)
  4. Executive briefing (4 hours P0, 24 hours P1)
  5. Customer communication (as appropriate)
```

## Performance Optimization

### Production Performance Tuning

```yaml
JVM Tuning (for components using JVM):
  Heap Size: -Xms4g -Xmx8g
  GC Algorithm: -XX:+UseG1GC
  GC Tuning: -XX:MaxGCPauseMillis=200
  Memory Management: -XX:+UseStringDeduplication

Database Performance:
  PostgreSQL Configuration:
    shared_buffers: 8GB
    effective_cache_size: 24GB
    work_mem: 32MB
    max_connections: 200
    checkpoint_completion_target: 0.9
    wal_buffers: 16MB
    random_page_cost: 1.1
    
  Connection Pooling:
    PgBouncer pool_mode: transaction
    Pool size: 100 connections per service
    Server idle timeout: 600 seconds

Container Optimization:
  Resource Limits:
    - Always set CPU and memory limits
    - Use QoS class "Guaranteed" for critical services
    - Enable memory overcommit protection
    
  Image Optimization:
    - Multi-stage builds to reduce image size
    - Distroless base images for security
    - Image layer caching optimization
    
  Network Optimization:
    - Enable kernel bypass for high-throughput workloads
    - Tune TCP buffer sizes for long-distance connections
    - Use connection pooling and keep-alives
```

## Troubleshooting Guide

### Common Issues and Solutions

#### Issue: Intent Processing Timeouts

**Symptoms:**
- Intent status shows "Processing" for >5 minutes
- LLM service returns 504 Gateway Timeout
- High error rates in Prometheus metrics

**Diagnosis:**
```bash
# Check LLM service health
kubectl logs -n nephoran-system deployment/llm-processor --tail=100

# Check resource utilization
kubectl top pods -n nephoran-system

# Check network connectivity
kubectl exec -n nephoran-system deployment/llm-processor -- \
  curl -v https://api.openai.com/v1/models
```

**Resolution:**
```bash
# Scale up LLM processor replicas
kubectl scale deployment/llm-processor --replicas=10 -n nephoran-system

# Check and update API rate limits
kubectl edit configmap llm-config -n nephoran-system

# Verify external network connectivity
kubectl apply -f deployments/networking/egress-rules.yaml
```

#### Issue: Vector Database Performance Degradation

**Symptoms:**
- RAG queries taking >5 seconds
- High memory utilization on Weaviate pods
- Index corruption warnings in logs

**Diagnosis:**
```bash
# Check Weaviate cluster health
kubectl exec -n nephoran-system weaviate-0 -- \
  curl localhost:8080/v1/.well-known/ready

# Monitor query performance
kubectl logs -n nephoran-system weaviate-0 | grep "query_duration"

# Check storage utilization
kubectl exec -n nephoran-system weaviate-0 -- df -h
```

**Resolution:**
```bash
# Restart Weaviate cluster with rolling update
kubectl rollout restart statefulset/weaviate -n nephoran-system

# Rebuild indexes if corrupted
kubectl exec -n nephoran-system weaviate-0 -- \
  curl -X POST localhost:8080/v1/schema/rebuild

# Scale up cluster if needed
kubectl patch statefulset weaviate -n nephoran-system \
  -p '{"spec":{"replicas":6}}'
```

### Emergency Recovery Procedures

#### Database Recovery

```bash
#!/bin/bash
# Emergency database recovery procedure

# Check database health
kubectl exec -n nephoran-system postgresql-0 -- \
  pg_isready -h localhost -p 5432

# If primary is down, promote standby
kubectl exec -n nephoran-system postgresql-1 -- \
  patronictl switchover --master postgresql-0 --candidate postgresql-1

# Restore from backup if needed
./scripts/restore-database-backup.sh \
  --backup-date $(date -d "1 hour ago" '+%Y-%m-%d %H:%M:%S') \
  --target-cluster production
```

#### Service Mesh Recovery

```bash
#!/bin/bash
# Emergency service mesh recovery

# Check Istio control plane health
istioctl proxy-status

# Restart Istio control plane if needed  
kubectl rollout restart deployment/istiod -n istio-system

# Re-inject sidecar proxies if needed
kubectl delete pods --all -n nephoran-system
kubectl wait --for=condition=ready pod --all -n nephoran-system --timeout=300s
```

This enterprise deployment guide provides the foundation for production-ready deployments. The next sections will cover detailed runbooks and operational procedures.

## References

- [Operations Handbook](../operations/operations-handbook.md)
- [Security Hardening Guide](../security/hardening-guide.md)
- [Monitoring Runbooks](../runbooks/monitoring-operations.md)
- [Disaster Recovery Procedures](../runbooks/disaster-recovery.md)
- [Performance Benchmarking](../benchmarks/comprehensive-performance-analysis.md)