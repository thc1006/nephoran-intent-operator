# Nephoran Intent Operator - Complete Deployment Guide

## Overview

This comprehensive guide provides step-by-step instructions for deploying the Nephoran Intent Operator in various environments. The system supports both local development deployments and production cloud deployments with enterprise-grade security and monitoring.

## Table of Contents

1. [Prerequisites and System Requirements](#prerequisites-and-system-requirements)
2. [Environment Preparation](#environment-preparation)
3. [Local Development Deployment](#local-development-deployment)
4. [Cloud Production Deployment](#cloud-production-deployment)
5. [Post-Deployment Validation](#post-deployment-validation)
6. [Monitoring and Observability Setup](#monitoring-and-observability-setup)
7. [Troubleshooting](#troubleshooting)
8. [Security Configuration](#security-configuration)
9. [Performance Optimization](#performance-optimization)

## Prerequisites and System Requirements

### Verified System Requirements (Tested August 2025)

#### Essential Software Dependencies
- **Go**: Version 1.23.0+ (toolchain go1.24.5) - Required for infrastructure optimizations
- **Docker**: Latest stable version - For multi-stage container builds with security optimization
- **kubectl**: Compatible with your target Kubernetes version - For cluster operations
- **Python**: Version 3.8+ - For RAG API components and Flask-based services
- **Git**: Latest version - For GitOps integration and repository operations
- **make**: Cross-platform build system with Windows/Linux support

#### Kubernetes Infrastructure Requirements
- **Cluster Version**: Kubernetes 1.28+
- **Minimum Resources**: 8 CPU cores, 16GB RAM, 100GB storage
- **Recommended Resources**: 16 CPU cores, 32GB RAM, 500GB storage
- **Storage Classes**: Dynamic provisioning support for PersistentVolumes
- **Network**: LoadBalancer or Ingress controller for external access

#### Cloud Provider Specific
For **Google Kubernetes Engine (GKE)**:
- GCP Project with billing enabled
- Artifact Registry API enabled
- Container Registry API enabled
- Kubernetes Engine API enabled
- Minimum node pool: 3 nodes (e2-standard-4 or larger)

#### Development Environment
- **IDE**: VSCode or GoLand recommended
- **Container Runtime**: Docker Desktop or compatible
- **Local Kubernetes**: Kind, Minikube, or Docker Desktop Kubernetes

### Dependency Verification

#### Automated Environment Validation
```bash
# Comprehensive environment validation (40+ checks)
./validate-environment.ps1

# Alternative manual verification
go version          # Should show go1.23.0+ 
docker --version    # Should show Docker 20.10+
kubectl version     # Should connect to cluster
python3 --version   # Should show Python 3.8+
make --version      # Should show GNU Make 4.0+
```

#### Infrastructure Validation
```bash
# Kubernetes cluster validation
kubectl cluster-info
kubectl get nodes
kubectl get storageclass

# Resource availability check
kubectl describe nodes | grep -E "Capacity|Allocatable"
```

## Environment Preparation

### 1. Development Environment Setup

#### Clone and Initialize Repository
```bash
# Clone the repository
git clone <repository-url>
cd nephoran-intent-operator

# Verify project structure
ls -la
```

#### Install Development Dependencies
```bash
# Automated setup (recommended)
make setup-dev

# Manual dependency installation
go mod download
pip3 install -r requirements-rag.txt

# Install development tools
make dev-setup
```

#### Build System Validation
```bash
# Validate build system integrity
make validate-build

# Test compilation (without deployment)
make build-all

# Validate Docker build capability
make docker-build-test
```

### 2. Kubernetes Cluster Preparation

#### Local Cluster Setup (Kind)
```bash
# Install Kind (if not available)
go install sigs.k8s.io/kind@latest

# Create optimized cluster for Nephoran
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: nephoran-local
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
- role: worker
- role: worker
EOF

# Verify cluster creation
kubectl cluster-info --context kind-nephoran-local
```

#### Local Cluster Setup (Minikube)
```bash
# Start Minikube with adequate resources
minikube start --cpus=4 --memory=8192 --disk-size=50g \
  --kubernetes-version=v1.28.0 \
  --addons=ingress,metrics-server

# Verify Minikube cluster
minikube status
kubectl get nodes
```

#### GKE Cluster Setup
```bash
# Set project variables
export GCP_PROJECT_ID="your-project-id"
export GCP_REGION="us-central1"
export CLUSTER_NAME="nephoran-production"

# Create GKE cluster with optimized configuration
gcloud container clusters create $CLUSTER_NAME \
  --project=$GCP_PROJECT_ID \
  --region=$GCP_REGION \
  --machine-type=e2-standard-4 \
  --num-nodes=3 \
  --enable-autoscaling \
  --min-nodes=3 \
  --max-nodes=10 \
  --enable-autorepair \
  --enable-autoupgrade \
  --enable-ip-alias \
  --network="default" \
  --subnetwork="default" \
  --enable-network-policy \
  --enable-stackdriver-kubernetes \
  --addons=HorizontalPodAutoscaling,HttpLoadBalancing,NetworkPolicy

# Get cluster credentials
gcloud container clusters get-credentials $CLUSTER_NAME \
  --region=$GCP_REGION --project=$GCP_PROJECT_ID
```

### 3. Container Registry Setup

#### Docker Registry Authentication
```bash
# For Docker Hub
docker login

# For Google Artifact Registry
gcloud auth configure-docker us-central1-docker.pkg.dev

# Create Artifact Registry repository
gcloud artifacts repositories create nephoran-repo \
  --repository-format=docker \
  --location=$GCP_REGION \
  --description="Nephoran Intent Operator Images"
```

## Local Development Deployment

### 1. Quick Start Deployment

#### Automated Local Deployment
```bash
# Complete automated deployment (recommended for development)
./deploy.sh local

# This script automatically:
# - Builds all 4 service binaries
# - Creates enterprise-grade Docker images
# - Loads images into local cluster
# - Deploys using Kustomize overlays
# - Configures monitoring stack
```

#### Manual Step-by-Step Deployment
```bash
# Step 1: Build all services (40% faster with parallel builds)
make build-all

# Step 2: Create Docker images with security optimization
make docker-build

# Step 3: Load images into local cluster
# For Kind
kind load docker-image nephoran/llm-processor:latest --name nephoran-local
kind load docker-image nephoran/nephio-bridge:latest --name nephoran-local
kind load docker-image nephoran/oran-adaptor:latest --name nephoran-local

# For Minikube
eval $(minikube docker-env)
make docker-build

# Step 4: Deploy using Kustomize
kubectl apply -k deployments/kustomize/overlays/local

# Step 5: Deploy vector database
make deploy-rag
```

### 2. Component-by-Component Deployment

#### CRD Installation
```bash
# Install Custom Resource Definitions
kubectl apply -f deployments/crds/

# Verify CRD installation
kubectl get crd | grep nephoran
kubectl describe crd networkintents.nephoran.com
kubectl describe crd e2nodesets.nephoran.com
```

#### RBAC Configuration
```bash
# Create service accounts and RBAC
kubectl apply -f deployments/kubernetes/nephio-bridge-sa.yaml
kubectl apply -f deployments/kubernetes/nephio-bridge-rbac.yaml

# Verify RBAC setup
kubectl get serviceaccounts -l app.kubernetes.io/part-of=nephoran
kubectl get clusterroles -l app.kubernetes.io/part-of=nephoran
```

#### Core Services Deployment
```bash
# Deploy main controller
kubectl apply -f deployments/kustomize/base/nephio-bridge/

# Deploy LLM processor with full configuration
kubectl apply -f deployments/kustomize/base/llm-processor/

# Deploy O-RAN adaptor
kubectl apply -f deployments/kustomize/base/oran-adaptor/

# Deploy RAG API service
kubectl apply -f deployments/kustomize/base/rag-api/
```

#### Vector Database (Weaviate) Deployment
```bash
# Deploy Weaviate with optimized configuration
kubectl apply -f deployments/weaviate/weaviate-deployment.yaml

# Wait for Weaviate to be ready
kubectl wait --for=condition=ready pod -l app=weaviate --timeout=300s

# Populate knowledge base with telecom documents
./populate-knowledge-base.ps1
```

### 3. Local Deployment Validation

#### Service Health Verification
```bash
# Check all pods are running
kubectl get pods -l app.kubernetes.io/part-of=nephoran

# Verify service endpoints
kubectl get services -l app.kubernetes.io/part-of=nephoran

# Health check endpoints
kubectl port-forward svc/llm-processor 8080:8080 &
curl http://localhost:8080/healthz

kubectl port-forward svc/rag-api 5001:5001 &
curl http://localhost:5001/readyz

# Stop port forwards
pkill -f "kubectl port-forward"
```

#### Functional Testing
```bash
# Test CRD functionality
kubectl apply -f examples/networkintent-example.yaml
kubectl get networkintents
kubectl describe networkintent example-intent

# Test E2NodeSet functionality
kubectl apply -f test-e2nodeset.yaml
kubectl get e2nodesets
kubectl get configmaps -l e2nodeset=test-nodes
```

## Cloud Production Deployment

### 1. GKE Production Deployment

#### Pre-deployment Configuration
```bash
# Configure deployment variables
export GCP_PROJECT_ID="your-production-project"
export GCP_REGION="us-central1"
export AR_REPO="nephoran-repo"
export CLUSTER_NAME="nephoran-production"

# Update Kustomize configuration
sed -i "s/your-gcp-project/$GCP_PROJECT_ID/g" \
  deployments/kustomize/overlays/remote/kustomization.yaml
```

#### Image Registry Setup
```bash
# Build and push images to Artifact Registry
make docker-build
make docker-tag-remote
make docker-push

# Verify images in registry
gcloud artifacts docker images list \
  us-central1-docker.pkg.dev/$GCP_PROJECT_ID/$AR_REPO
```

#### Kubernetes Secret Management
```bash
# Create image pull secret
kubectl create secret generic nephoran-regcred \
  --from-file=.dockerconfigjson=$HOME/.docker/config.json \
  --type=kubernetes.io/dockerconfigjson

# Create additional secrets for production
kubectl create secret generic llm-processor-config \
  --from-literal=openai-api-key="your-openai-key" \
  --from-literal=model-name="gpt-4"

kubectl create secret generic weaviate-config \
  --from-literal=weaviate-url="http://weaviate:8080" \
  --from-literal=weaviate-api-key="your-weaviate-key"
```

#### Production Deployment
```bash
# Deploy to GKE cluster
./deploy.sh remote

# Monitor deployment progress
kubectl rollout status deployment/nephio-bridge
kubectl rollout status deployment/llm-processor
kubectl rollout status deployment/oran-adaptor
kubectl rollout status deployment/rag-api
```

### 2. Security Configuration

#### Network Policies
```bash
# Apply network security policies
kubectl apply -f deployments/security/security-network-policies.yaml

# Verify network policies
kubectl get networkpolicies
kubectl describe networkpolicy nephoran-default-deny
```

#### mTLS Configuration
```bash
# Install Istio service mesh (if not present)
istioctl install --set values.defaultRevision=default

# Label namespace for Istio injection
kubectl label namespace default istio-injection=enabled

# Apply Istio security configuration
kubectl apply -f deployments/istio/
```

#### RBAC Hardening
```bash
# Apply production RBAC configuration
kubectl apply -f deployments/security/security-rbac.yaml

# Verify security policies
kubectl auth can-i --list --as=system:serviceaccount:default:nephio-bridge
```

### 3. High Availability Setup

#### Multi-Zone Deployment
```bash
# Configure pod anti-affinity for HA
kubectl patch deployment nephio-bridge -p '
{
  "spec": {
    "template": {
      "spec": {
        "affinity": {
          "podAntiAffinity": {
            "preferredDuringSchedulingIgnoredDuringExecution": [
              {
                "weight": 100,
                "podAffinityTerm": {
                  "labelSelector": {
                    "matchLabels": {
                      "app": "nephio-bridge"
                    }
                  },
                  "topologyKey": "topology.kubernetes.io/zone"
                }
              }
            ]
          }
        }
      }
    }
  }
}'
```

#### Auto-scaling Configuration
```bash
# Deploy HPA configurations
kubectl apply -f deployments/kustomize/base/llm-processor/hpa.yaml
kubectl apply -f deployments/kustomize/base/rag-api/hpa.yaml

# Deploy KEDA for advanced scaling
kubectl apply -f deployments/kustomize/base/keda/

# Verify autoscaling setup
kubectl get hpa
kubectl get scaledobjects
```

## Post-Deployment Validation

### 1. Comprehensive Health Checks

#### Service Readiness Verification
```bash
# Automated validation script
./scripts/validate-deployment.sh

# Manual health checks
kubectl get pods --all-namespaces | grep nephoran
kubectl get services --all-namespaces | grep nephoran
kubectl get ingress --all-namespaces | grep nephoran
```

#### API Endpoint Testing
```bash
# LLM Processor API validation
kubectl port-forward svc/llm-processor 8080:8080 &
curl -H "Content-Type: application/json" \
  -d '{"intent":"Deploy 3 E2 nodes for eMBB slice"}' \
  http://localhost:8080/process-intent

# RAG API validation
kubectl port-forward svc/rag-api 5001:5001 &
curl http://localhost:5001/healthz
curl -X POST "http://localhost:5001/query" \
  -H "Content-Type: application/json" \
  -d '{"query":"What is O-RAN E2 interface?"}'
```

#### End-to-End Workflow Testing
```bash
# Create test NetworkIntent
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: test-e2e-intent
  namespace: default
spec:
  intent: "Scale out E2 nodes to handle increased traffic in urban area"
  priority: "high"
EOF

# Monitor processing
kubectl get networkintents -w
kubectl describe networkintent test-e2e-intent
kubectl logs -f deployment/llm-processor | grep test-e2e-intent
```

### 2. Performance Validation

#### Load Testing
```bash
# Run performance benchmark suite
./scripts/performance-benchmark-suite.sh

# Monitor resource usage during testing
kubectl top nodes
kubectl top pods --all-namespaces
```

#### Monitoring Validation
```bash
# Verify metrics collection
kubectl port-forward svc/prometheus 9090:9090 &
curl http://localhost:9090/api/v1/query?query=up

# Check Grafana dashboards
kubectl port-forward svc/grafana 3000:3000 &
# Access Grafana at http://localhost:3000
```

## Monitoring and Observability Setup

### 1. Prometheus and Grafana Deployment

#### Monitoring Stack Installation
```bash
# Deploy complete monitoring stack
kubectl apply -f deployments/monitoring/complete-monitoring-stack.yaml

# Wait for monitoring components
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus --timeout=300s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana --timeout=300s
```

#### Custom Dashboards Configuration
```bash
# Import Nephoran-specific dashboards
kubectl apply -f deployments/monitoring/nephoran-production-dashboards.yaml

# Import telecom-specific metrics
kubectl apply -f deployments/monitoring/telecom-specific-metrics.yaml
```

### 2. Distributed Tracing

#### Jaeger Installation
```bash
# Deploy Jaeger tracing
kubectl apply -f deployments/monitoring/jaeger-tracing.yaml

# Verify Jaeger deployment
kubectl get pods -l app.kubernetes.io/name=jaeger
```

#### OpenTelemetry Configuration
```bash
# Configure OpenTelemetry instrumentation
kubectl apply -f deployments/monitoring/opentelemetry-configuration.yaml

# Verify tracing is working
kubectl port-forward svc/jaeger-query 16686:16686 &
# Access Jaeger UI at http://localhost:16686
```

### 3. Centralized Logging

#### ELK Stack Deployment
```bash
# Deploy Elasticsearch, Logstash, and Kibana
kubectl apply -f deployments/monitoring/elasticsearch-deployment.yaml
kubectl apply -f deployments/monitoring/logstash-deployment.yaml
kubectl apply -f deployments/monitoring/kibana-deployment.yaml

# Configure structured logging
kubectl apply -f deployments/monitoring/structured-logging-config.yaml
```

## Troubleshooting

### 1. Common Deployment Issues

#### Pod Startup Failures
```bash
# Check pod status and events
kubectl get pods -o wide
kubectl describe pod <failing-pod-name>

# Check resource constraints
kubectl describe nodes
kubectl get resourcequotas
kubectl get limitranges
```

#### Image Pull Errors
```bash
# Verify image pull secrets
kubectl get secrets | grep regcred
kubectl describe secret nephoran-regcred

# Test manual image pull
kubectl run test-pod --image=nephoran/llm-processor:latest --rm -it --restart=Never -- /bin/sh
```

#### Service Discovery Issues
```bash
# Check service endpoints
kubectl get endpoints
kubectl describe service llm-processor

# Test internal connectivity
kubectl run debug-pod --image=curlimages/curl --rm -it --restart=Never -- /bin/sh
# From inside pod: curl http://llm-processor:8080/healthz
```

### 2. Configuration Problems

#### CRD Validation Errors
```bash
# Check CRD installation
kubectl get crd | grep nephoran
kubectl describe crd networkintents.nephoran.com

# Validate CRD schema
kubectl apply --dry-run=client -f examples/networkintent-example.yaml
```

#### RBAC Permission Issues
```bash
# Check service account permissions
kubectl auth can-i --list --as=system:serviceaccount:default:nephio-bridge

# Debug RBAC issues
kubectl describe rolebinding -l app.kubernetes.io/part-of=nephoran
kubectl describe clusterrolebinding -l app.kubernetes.io/part-of=nephoran
```

### 3. Performance Issues

#### Resource Constraints
```bash
# Monitor resource usage
kubectl top nodes
kubectl top pods --sort-by=cpu
kubectl top pods --sort-by=memory

# Check resource requests and limits
kubectl describe pod <pod-name> | grep -A 5 "Requests:\|Limits:"
```

#### Network Performance
```bash
# Test network latency between services
kubectl exec -it <llm-processor-pod> -- ping rag-api
kubectl exec -it <llm-processor-pod> -- curl -w "@curl-format.txt" http://rag-api:5001/healthz
```

## Security Configuration

### 1. Security Hardening

#### Pod Security Standards
```bash
# Apply pod security policies
kubectl apply -f deployments/security/security-deployments.yaml

# Verify security contexts
kubectl describe pod <pod-name> | grep -A 10 "Security Context"
```

#### Secret Management
```bash
# Rotate secrets regularly
kubectl create secret generic llm-processor-config-new \
  --from-literal=openai-api-key="new-openai-key" \
  --dry-run=client -o yaml | kubectl apply -f -

# Update deployment to use new secret
kubectl patch deployment llm-processor -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "llm-processor",
            "envFrom": [
              {
                "secretRef": {
                  "name": "llm-processor-config-new"
                }
              }
            ]
          }
        ]
      }
    }
  }
}'
```

### 2. Compliance and Auditing

#### Security Scanning
```bash
# Run security audit
./scripts/execute-security-audit.sh

# Scan container images for vulnerabilities
./scripts/vulnerability-scanner.sh
```

#### Compliance Validation
```bash
# Generate compliance report
./scripts/oran-compliance-validator.sh

# Audit RBAC configuration
kubectl-who-can create networkintents
kubectl-who-can delete e2nodesets
```

## Performance Optimization

### 1. Resource Optimization

#### CPU and Memory Tuning
```bash
# Update resource requests and limits based on monitoring data
kubectl patch deployment llm-processor -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "llm-processor",
            "resources": {
              "requests": {
                "cpu": "500m",
                "memory": "1Gi"
              },
              "limits": {
                "cpu": "2000m",
                "memory": "4Gi"
              }
            }
          }
        ]
      }
    }
  }
}'
```

#### Storage Optimization
```bash
# Configure high-performance storage for Weaviate
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nephoran-fast-ssd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
  replication-type: regional-pd
allowVolumeExpansion: true
EOF
```

### 2. Scaling Configuration

#### Horizontal Scaling
```bash
# Configure advanced autoscaling
kubectl apply -f deployments/kustomize/base/optimized-hpa/enhanced-keda-scaling.yaml

# Monitor scaling events
kubectl get events --sort-by='.lastTimestamp' | grep HorizontalPodAutoscaler
```

#### Vertical Scaling
```bash
# Install Vertical Pod Autoscaler (if available)
kubectl apply -f https://github.com/kubernetes/autoscaler/releases/latest/download/vpa-release.yaml

# Configure VPA for Nephoran components
kubectl apply -f - <<EOF
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: llm-processor-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
  updatePolicy:
    updateMode: "Auto"
EOF
```

## Maintenance and Updates

### 1. Rolling Updates

#### Application Updates
```bash
# Update container images
kubectl set image deployment/llm-processor llm-processor=nephoran/llm-processor:v1.2.0
kubectl set image deployment/nephio-bridge nephio-bridge=nephoran/nephio-bridge:v1.2.0

# Monitor rollout
kubectl rollout status deployment/llm-processor
kubectl rollout status deployment/nephio-bridge
```

#### Configuration Updates
```bash
# Update ConfigMaps
kubectl create configmap llm-processor-config-v2 \
  --from-file=config.yaml=new-config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart deployments to pick up new configuration
kubectl rollout restart deployment/llm-processor
```

### 2. Backup and Recovery

#### Database Backup
```bash
# Backup Weaviate data
./deployments/weaviate/backup-validation.sh

# Backup Kubernetes resources
kubectl get all,configmaps,secrets,pvc -o yaml > nephoran-backup-$(date +%Y%m%d).yaml
```

#### Disaster Recovery Testing
```bash
# Test disaster recovery procedures
./scripts/disaster-recovery.sh --test-mode

# Validate recovery capabilities
./deployments/weaviate/backup-validation.sh --validate
```

This deployment guide provides comprehensive instructions for deploying the Nephoran Intent Operator in various environments. Follow the appropriate section based on your deployment target and requirements. For additional assistance, refer to the troubleshooting section or contact the development team.