# Nephoran Intent Operator - Production Deployment Guide

## Overview

This comprehensive guide provides step-by-step procedures for deploying the Nephoran Intent Operator in production environments across major cloud providers (AWS, GCP, Azure). The system is a production-ready cloud-native orchestration platform that bridges natural language network operations with O-RAN compliant network function deployments.

## Architecture Overview

The Nephoran Intent Operator consists of five core layers:
1. **Intent Processing Layer**: NetworkIntent, E2NodeSet, ManagedElement controllers
2. **AI/ML Processing Layer**: LLM Processor, RAG API, Weaviate vector database  
3. **GitOps Orchestration Layer**: Nephio Bridge, Package Generation, Porch API
4. **O-RAN Interface Layer**: A1, O1, O2 interface adaptors
5. **Monitoring & Observability Layer**: Prometheus, Grafana, Jaeger, custom metrics

## Prerequisites

### Infrastructure Requirements

#### Minimum Production Requirements
- **Kubernetes**: v1.25+ with CSI driver support
- **CPU**: 32 cores across cluster nodes
- **Memory**: 128GB total cluster memory
- **Storage**: 1TB+ with high-performance SSD (NVMe preferred)
- **Network**: 10Gbps+ bandwidth for O-RAN interface processing

#### Cloud Provider Specifications

**AWS Requirements:**
- EKS cluster with managed node groups
- Instance types: `m5.2xlarge` or higher
- Storage classes: `gp3`, `io2` for high IOPS
- Load balancer: ALB with SSL termination
- IAM roles: EKS service role and node group roles

**GCP Requirements:**
- GKE cluster with autopilot or standard mode
- Machine types: `e2-standard-8` or higher
- Storage classes: `pd-ssd`, `pd-extreme` for high IOPS
- Load balancer: Global HTTPS load balancer
- Service accounts: GKE service account with required permissions

**Azure Requirements:**
- AKS cluster with managed identity
- VM sizes: `Standard_D8s_v3` or higher
- Storage classes: `managed-premium`, `ultra-disk`
- Load balancer: Application Gateway with SSL
- Service principals: AKS service principal with cluster admin rights

### Software Dependencies

#### Required Tools
```bash
# Core deployment tools
kubectl >= 1.25.0
helm >= 3.8.0
kustomize >= 4.5.0
docker >= 20.10.0

# Security and compliance tools
trivy >= 0.35.0
cosign >= 1.13.0
kubesec >= 2.11.0

# Monitoring and observability
prometheus-operator >= 0.60.0
grafana >= 9.0.0
jaeger-operator >= 1.38.0
```

#### Environment Variables
```bash
# Core configuration
export NEPHORAN_NAMESPACE="nephoran-system"
export NEPHORAN_VERSION="v2.1.0"
export OPENAI_API_KEY="your-openai-api-key"

# Cloud provider specific
export CLOUD_PROVIDER="aws|gcp|azure"
export REGION="us-east-1|us-central1|eastus"
export CLUSTER_NAME="nephoran-production"

# Security configuration
export ENABLE_ISTIO="true"
export ENABLE_SECURITY_POLICIES="true"
export BACKUP_ENABLED="true"
```

## Production Deployment Procedures

### Phase 1: Infrastructure Preparation

#### 1.1 Cloud Infrastructure Setup

**AWS Deployment:**
```bash
#!/bin/bash
# AWS infrastructure setup
CLUSTER_NAME="nephoran-production"
REGION="us-east-1"
NODE_GROUP_NAME="nephoran-workers"

# Create EKS cluster
eksctl create cluster \
  --name $CLUSTER_NAME \
  --region $REGION \
  --node-type m5.2xlarge \
  --nodes 6 \
  --nodes-min 3 \
  --nodes-max 20 \
  --managed \
  --enable-ssm \
  --asg-access \
  --external-dns-access \
  --full-ecr-access \
  --alb-ingress-access

# Create storage classes
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3-encrypted
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  encrypted: "true"
  iops: "3000"
  throughput: "125"
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
EOF

# Install AWS Load Balancer Controller
helm repo add eks https://aws.github.io/eks-charts
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  --set clusterName=$CLUSTER_NAME \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller \
  -n kube-system
```

**GCP Deployment:**
```bash
#!/bin/bash
# GCP infrastructure setup
PROJECT_ID="nephoran-production"
CLUSTER_NAME="nephoran-production"
REGION="us-central1"

# Create GKE cluster
gcloud container clusters create $CLUSTER_NAME \
  --project=$PROJECT_ID \
  --zone=$REGION-a \
  --machine-type=e2-standard-8 \
  --num-nodes=6 \
  --enable-autoscaling \
  --min-nodes=3 \
  --max-nodes=20 \
  --enable-autorepair \
  --enable-autoupgrade \
  --enable-network-policy \
  --enable-ip-alias \
  --enable-shielded-nodes \
  --disk-type=pd-ssd \
  --disk-size=100GB

# Create storage classes
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: pd-ssd-encrypted
provisioner: pd.csi.storage.gke.io
parameters:
  type: pd-ssd
  replication-type: regional-pd
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
EOF

# Install Google Cloud Load Balancer Controller
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/gke-networking-recipes/main/ingress/single-cluster/ingress-https/ingress-ssl.yaml
```

**Azure Deployment:**
```bash
#!/bin/bash
# Azure infrastructure setup
RESOURCE_GROUP="nephoran-production-rg"
CLUSTER_NAME="nephoran-production"
LOCATION="eastus"

# Create resource group
az group create --name $RESOURCE_GROUP --location $LOCATION

# Create AKS cluster
az aks create \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --node-count 6 \
  --min-count 3 \
  --max-count 20 \
  --enable-cluster-autoscaler \
  --node-vm-size Standard_D8s_v3 \
  --enable-managed-identity \
  --enable-addons monitoring \
  --enable-network-policy \
  --network-plugin azure \
  --generate-ssh-keys

# Get credentials
az aks get-credentials --resource-group $RESOURCE_GROUP --name $CLUSTER_NAME

# Create storage classes
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: managed-premium-encrypted
provisioner: disk.csi.azure.com
parameters:
  skuName: Premium_LRS
  kind: Managed
  cachingmode: ReadOnly
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
EOF
```

#### 1.2 Security Setup

**Network Policies and RBAC:**
```bash
#!/bin/bash
# Apply security baseline
kubectl apply -f deployments/security/security-rbac.yaml
kubectl apply -f deployments/security/security-network-policies.yaml
kubectl apply -f deployments/security/security-config.yaml
```

**Secrets Management:**
```bash
#!/bin/bash
# Create namespace
kubectl create namespace nephoran-system

# Create secrets
kubectl create secret generic openai-api-key \
  --from-literal=api-key="$OPENAI_API_KEY" \
  --namespace=nephoran-system

kubectl create secret generic weaviate-api-key \
  --from-literal=api-key="$(openssl rand -hex 32)" \
  --namespace=nephoran-system

kubectl create secret generic github-token \
  --from-literal=token="$GITHUB_TOKEN" \
  --namespace=nephoran-system
```

### Phase 2: Core Infrastructure Deployment

#### 2.1 Service Mesh Installation (Istio)

```bash
#!/bin/bash
# Install Istio service mesh
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# Install Istio with production settings
istioctl install --set values.global.meshID=nephoran \
  --set values.global.meshID=nephoran \
  --set values.global.network=nephoran-network \
  --set values.pilot.env.EXTERNAL_ISTIOD=true \
  --set values.global.proxy.resources.requests.cpu=100m \
  --set values.global.proxy.resources.requests.memory=128Mi \
  --set values.global.proxy.resources.limits.cpu=2000m \
  --set values.global.proxy.resources.limits.memory=1Gi

# Enable sidecar injection
kubectl label namespace nephoran-system istio-injection=enabled

# Apply Istio configurations
kubectl apply -f deployments/istio/
```

#### 2.2 Monitoring Stack Deployment

```bash
#!/bin/bash
# Deploy monitoring infrastructure
kubectl apply -f deployments/monitoring/complete-monitoring-stack.yaml

# Wait for monitoring components
kubectl wait --for=condition=available deployment/prometheus-server \
  -n nephoran-monitoring --timeout=600s
kubectl wait --for=condition=available deployment/grafana \
  -n nephoran-monitoring --timeout=600s
kubectl wait --for=condition=available deployment/jaeger \
  -n nephoran-monitoring --timeout=600s
```

#### 2.3 Storage Layer Deployment

```bash
#!/bin/bash
# Deploy Weaviate vector database
helm repo add weaviate https://weaviate.github.io/weaviate-helm
helm install weaviate weaviate/weaviate \
  --namespace nephoran-system \
  --set image.tag=1.26.1 \
  --set replicas=3 \
  --set persistence.enabled=true \
  --set persistence.size=500Gi \
  --set persistence.storageClass=gp3-encrypted \
  --set resources.requests.memory=8Gi \
  --set resources.requests.cpu=2000m \
  --set resources.limits.memory=16Gi \
  --set resources.limits.cpu=4000m \
  --set authentication.apikey.enabled=true \
  --set modules.text2vec-openai.enabled=true \
  --set modules.generative-openai.enabled=true \
  --values deployments/weaviate/values.yaml

# Wait for Weaviate to be ready
kubectl wait --for=condition=ready pod -l app=weaviate \
  -n nephoran-system --timeout=600s
```

### Phase 3: Application Deployment

#### 3.1 Core Controllers Deployment

```bash
#!/bin/bash
# Deploy CRDs first
kubectl apply -f deployments/crds/

# Wait for CRDs to be established
kubectl wait --for condition=established crd/networkintents.nephoran.com
kubectl wait --for condition=established crd/e2nodesets.nephoran.com
kubectl wait --for condition=established crd/managedelements.nephoran.com

# Deploy controllers
kubectl apply -k deployments/kustomize/overlays/production/nephio-bridge/
kubectl apply -k deployments/kustomize/overlays/production/oran-adaptor/

# Verify controller deployment
kubectl wait --for=condition=available deployment/nephio-bridge \
  -n nephoran-system --timeout=600s
kubectl wait --for=condition=available deployment/oran-adaptor \
  -n nephoran-system --timeout=600s
```

#### 3.2 AI/ML Processing Services

```bash
#!/bin/bash
# Deploy LLM Processor service
kubectl apply -k deployments/kustomize/overlays/production/llm-processor/

# Deploy RAG API service
kubectl apply -k deployments/kustomize/overlays/production/rag-api/

# Wait for services to be ready
kubectl wait --for=condition=available deployment/llm-processor \
  -n nephoran-system --timeout=600s
kubectl wait --for=condition=available deployment/rag-api \
  -n nephoran-system --timeout=600s

# Verify service health
kubectl exec deployment/llm-processor -n nephoran-system -- \
  curl -f http://localhost:8080/healthz
kubectl exec deployment/rag-api -n nephoran-system -- \
  curl -f http://localhost:8080/health
```

#### 3.3 Auto-Scaling Configuration

```bash
#!/bin/bash
# Install KEDA for advanced auto-scaling
helm repo add kedacore https://kedacore.github.io/charts
helm install keda kedacore/keda \
  --namespace keda-system \
  --create-namespace \
  --set prometheus.metricServer.enabled=true \
  --set prometheus.operator.enabled=true

# Apply HPA and KEDA configurations
kubectl apply -f deployments/kustomize/base/optimized-hpa/
kubectl apply -f deployments/kustomize/base/keda/

# Verify auto-scaling setup
kubectl get hpa -n nephoran-system
kubectl get scaledobjects -n nephoran-system
```

### Phase 4: Validation and Testing

#### 4.1 System Health Validation

```bash
#!/bin/bash
# Run comprehensive health check
./scripts/monitoring-operations.sh health-check

# Verify all pods are running
kubectl get pods -n nephoran-system -o wide
kubectl get pods -n nephoran-monitoring -o wide

# Check service endpoints
kubectl get svc -n nephoran-system
kubectl get ingress -n nephoran-system
```

#### 4.2 Integration Testing

```bash
#!/bin/bash
# Test intent processing pipeline
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: production-validation-test
  namespace: nephoran-system
spec:
  description: "Deploy AMF with 3 replicas for production validation"
  priority: high
  intentType: deployment
EOF

# Monitor processing
kubectl wait --for=condition=Ready networkintent/production-validation-test \
  -n nephoran-system --timeout=300s

# Verify status
kubectl describe networkintent production-validation-test -n nephoran-system
```

#### 4.3 Performance Validation

```bash
#!/bin/bash
# Run performance benchmark
./scripts/ops/performance-benchmark-suite.sh --environment=production

# Run load test
./scripts/ops/execute-production-load-test.sh --duration=300 --concurrent-users=50

# Verify performance metrics meet SLAs
kubectl port-forward -n nephoran-monitoring svc/grafana 3000:3000 &
# Access Grafana at http://localhost:3000 for performance validation
```

## Environment-Specific Configurations

### Production Environment Variables

```bash
# Production configuration template
cat > production.env <<EOF
# Core system configuration
NEPHORAN_ENVIRONMENT=production
NEPHORAN_LOG_LEVEL=INFO
NEPHORAN_DEBUG=false

# Performance and scaling
MAX_CONCURRENT_INTENTS=100
LLM_REQUEST_TIMEOUT=60s
RAG_CACHE_TTL=3600s
VECTOR_SEARCH_TIMEOUT=10s

# Security settings
ENABLE_MTLS=true
SECURITY_SCAN_ENABLED=true
AUDIT_LOGGING=true
COMPLIANCE_MODE=strict

# High availability
MIN_REPLICAS=3
MAX_REPLICAS=20
BACKUP_RETENTION_DAYS=90
ENABLE_CROSS_REGION_BACKUP=true

# Monitoring and alerting
PROMETHEUS_RETENTION=90d
METRICS_SCRAPE_INTERVAL=15s
ALERT_MANAGER_ENABLED=true
DISTRIBUTED_TRACING=true

# Resource limits
CPU_REQUEST=1000m
CPU_LIMIT=4000m
MEMORY_REQUEST=2Gi
MEMORY_LIMIT=8Gi
STORAGE_REQUEST=100Gi
EOF
```

### Resource Quotas and Limits

```yaml
# Production resource quotas
apiVersion: v1
kind: ResourceQuota
metadata:
  name: nephoran-production-quota
  namespace: nephoran-system
spec:
  hard:
    requests.cpu: "50"
    requests.memory: 200Gi
    requests.storage: 2Ti
    limits.cpu: "100"
    limits.memory: 400Gi
    persistentvolumeclaims: "20"
    services: "50"
    secrets: "50"
    configmaps: "100"
---
apiVersion: v1
kind: LimitRange
metadata:
  name: nephoran-production-limits
  namespace: nephoran-system
spec:
  limits:
  - default:
      cpu: "2000m"
      memory: "4Gi"
    defaultRequest:
      cpu: "500m"
      memory: "1Gi"
    type: Container
  - max:
      cpu: "8000m"
      memory: "16Gi"
    min:
      cpu: "100m"
      memory: "128Mi"
    type: Container
```

## Security Hardening

### 4.1 Network Security

```bash
#!/bin/bash
# Apply comprehensive network policies
kubectl apply -f deployments/security/security-network-policies.yaml

# Verify network policy enforcement
kubectl get networkpolicies -n nephoran-system
kubectl describe networkpolicy default-deny-all -n nephoran-system
```

### 4.2 Pod Security Standards

```yaml
# Pod Security Standards enforcement
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
    security.nephoran.com/tier: production
```

### 4.3 Image Security

```bash
#!/bin/bash
# Scan container images for vulnerabilities
for image in llm-processor rag-api nephio-bridge oran-adaptor; do
  trivy image --severity HIGH,CRITICAL \
    us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran/$image:latest
done

# Verify image signatures (if using signed images)
for image in llm-processor rag-api nephio-bridge oran-adaptor; do
  cosign verify \
    us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran/$image:latest
done
```

## Compliance and Audit

### 4.1 SOC 2 Compliance

```bash
#!/bin/bash
# Run SOC 2 compliance validation
./scripts/security/execute-security-audit.sh --compliance=soc2

# Generate compliance report
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' > compliance-events.log
kubectl logs -l app.kubernetes.io/part-of=nephoran-intent-operator \
  -n nephoran-system --since=24h > compliance-logs.txt
```

### 4.2 Telecom Regulatory Compliance

```bash
#!/bin/bash
# Run telecom-specific compliance checks
./scripts/oran-compliance-validator.sh --standard=o-ran-alliance

# Validate GDPR compliance for EU deployments
if [[ "$REGION" =~ ^eu- ]]; then
  ./scripts/security/execute-security-audit.sh --compliance=gdpr
fi
```

## Troubleshooting

### Common Deployment Issues

**Issue 1: Weaviate fails to start**
```bash
# Check storage class and PVC
kubectl get pvc -n nephoran-system
kubectl describe pvc weaviate-data-0 -n nephoran-system

# Check resource constraints
kubectl describe pod -l app=weaviate -n nephoran-system | grep -A 10 "Events:"

# Solution: Verify storage class exists and has sufficient capacity
kubectl get storageclass
```

**Issue 2: LLM Processor connection errors**
```bash
# Check OpenAI API key secret
kubectl get secret openai-api-key -n nephoran-system -o yaml

# Verify network connectivity
kubectl exec deployment/llm-processor -n nephoran-system -- \
  curl -v https://api.openai.com/v1/models

# Check service mesh configuration
kubectl get destinationrule -n nephoran-system
```

**Issue 3: Auto-scaling not working**
```bash
# Check HPA status
kubectl describe hpa -n nephoran-system

# Verify metrics server
kubectl get pods -n kube-system | grep metrics-server

# Check KEDA configuration
kubectl get scaledobjects -n nephoran-system -o yaml
```

## Post-Deployment Operations

### 4.1 Initial Configuration

```bash
#!/bin/bash
# Initialize knowledge base
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: knowledge-base-init
  namespace: nephoran-system
spec:
  template:
    spec:
      containers:
      - name: kb-init
        image: us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran/rag-api:latest
        command: ["python3", "/app/scripts/populate_knowledge_base.py"]
        env:
        - name: WEAVIATE_URL
          value: "http://weaviate:8080"
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: openai-api-key
              key: api-key
      restartPolicy: Never
  backoffLimit: 3
EOF

# Wait for knowledge base initialization
kubectl wait --for=condition=complete job/knowledge-base-init \
  -n nephoran-system --timeout=1800s
```

### 4.2 Monitoring Setup

```bash
#!/bin/bash
# Configure Grafana dashboards
kubectl port-forward -n nephoran-monitoring svc/grafana 3000:3000 &
GRAFANA_PID=$!

# Wait for port forward
sleep 5

# Import dashboards
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @deployments/monitoring/nephoran-production-dashboards.yaml

# Setup alerting
curl -X POST http://admin:admin@localhost:3000/api/alert-notifications \
  -H "Content-Type: application/json" \
  -d @deployments/monitoring/enhanced-alerting-configuration.yaml

# Cleanup
kill $GRAFANA_PID
```

### 4.3 Backup Configuration

```bash
#!/bin/bash
# Setup automated backups
kubectl apply -f deployments/kubernetes/disaster-recovery-cronjobs.yaml

# Verify backup schedule
kubectl get cronjobs -n nephoran-system
kubectl describe cronjob nephoran-backup-daily -n nephoran-system
```

This production deployment guide provides comprehensive procedures for deploying the Nephoran Intent Operator across major cloud providers with enterprise-grade security, monitoring, and compliance capabilities.