# Getting Started with Nephoran Intent Operator

## Overview

The Nephoran Intent Operator is an innovative research project that explores how Large Language Models can transform natural language network operation intents into Kubernetes-based telecommunications infrastructure deployments. This guide will help you set up and experiment with the system in a development environment.

## Prerequisites

### Software Requirements
- **Go 1.23+** - Core language for the operator
- **Docker** - For containerization
- **Kubernetes cluster** - kind, minikube, or cloud provider
- **kubectl** - Kubernetes command-line tool
- **Helm 3.x** - Package manager for Kubernetes
- **Python 3.8+** - For RAG API components

### API Keys and Credentials
- **OpenAI API Key** - Required for LLM integration
- **Git Access Token** - Optional, for GitOps features

### System Requirements
- 4GB RAM minimum (8GB recommended)
- 2 CPU cores minimum
- 10GB free disk space

## Quick Start (15 minutes)

### Step 1: Clone and Setup

```bash
git clone https://github.com/your-repo/nephoran-intent-operator
cd nephoran-intent-operator

# Set required environment variables
export OPENAI_API_KEY="sk-your-openai-key-here"
export LOG_LEVEL="info"  # Optional: debug, info, warn, error
```

### Step 2: Prepare Your Kubernetes Cluster

Using **kind** (recommended for local development):

```bash
# Create a kind cluster
kind create cluster --name nephoran-dev --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

# Verify cluster is ready
kubectl cluster-info --context kind-nephoran-dev
```

Using **minikube**:

```bash
minikube start --driver=docker --memory=4096 --cpus=2
```

### Step 3: Install Custom Resource Definitions

```bash
# Install the CRDs that define our custom resources
kubectl apply -f deployments/crds/
```

Verify CRDs are installed:

```bash
kubectl get crd | grep nephoran
```

You should see:
- `networkintents.nephoran.com`
- `e2nodesets.nephoran.com`
- `managedelements.nephoran.com`

### Step 4: Deploy Core Components

Using **Helm** (recommended):

```bash
# Create namespace
kubectl create namespace nephoran-system

# Install Helm chart with basic configuration
helm install nephoran-operator ./deployments/helm/nephoran-operator \
  --namespace nephoran-system \
  --set llmProcessor.env.OPENAI_API_KEY="$OPENAI_API_KEY" \
  --set rag.enabled=false \
  --set ml.enabled=false
```

Using **kubectl** (alternative):

```bash
# Create secrets for API keys
kubectl create secret generic llm-secrets \
  --from-literal=openai-api-key="$OPENAI_API_KEY" \
  --namespace nephoran-system

# Deploy components
kubectl apply -f deployments/kubernetes/
```

### Step 5: Verify Installation

Check that all pods are running:

```bash
kubectl get pods -n nephoran-system
```

Expected output:
```
NAME                                 READY   STATUS    RESTARTS   AGE
llm-processor-xxx                   1/1     Running   0          2m
nephio-bridge-xxx                   1/1     Running   0          2m
oran-adaptor-xxx                    1/1     Running   0          2m
```

Check services are accessible:

```bash
kubectl get svc -n nephoran-system
```

Test health endpoints:

```bash
kubectl port-forward -n nephoran-system svc/llm-processor 8080:8080 &
curl http://localhost:8080/health
```

## Your First Network Intent

### Create a Simple Intent

```bash
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: my-first-intent
  namespace: default
spec:
  intent: "Deploy a basic 5G AMF function for development testing"
  priority: "medium"
  target_environment: "development"
EOF
```

### Monitor Processing

Watch the intent being processed:

```bash
# Check intent status
kubectl get networkintent my-first-intent -o yaml

# Watch controller logs
kubectl logs -n nephoran-system deployment/nephio-bridge -f

# Check for created resources
kubectl get configmaps | grep amf
```

### Expected Behavior

1. **Intent Creation** - NetworkIntent resource is created
2. **Controller Detection** - Nephio bridge controller detects the new intent
3. **LLM Processing** - Controller calls LLM processor to interpret the intent
4. **Resource Creation** - Controller creates ConfigMaps and other resources to simulate deployment
5. **Status Update** - Intent status is updated with processing results

## Understanding the Output

### Intent Status Fields

Check your intent's processing status:

```bash
kubectl get networkintent my-first-intent -o jsonpath='{.status}' | jq .
```

Key status fields:
- `phase` - Current processing phase (Pending, Processing, Deployed, Failed)
- `conditions` - Detailed condition information
- `deployedResources` - List of created Kubernetes resources
- `lastProcessedTime` - When the intent was last processed
- `processingErrors` - Any errors encountered

### Generated Resources

List resources created by your intent:

```bash
# ConfigMaps (simulating network functions)
kubectl get configmaps -l nephoran.com/intent=my-first-intent

# Check content of generated configs
kubectl get configmap amf-config-xxx -o yaml
```

## Common Use Cases and Examples

### Development Environment Setup

```yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: dev-environment
spec:
  intent: "Set up a complete 5G development environment with AMF, SMF, and test UPF"
  target_environment: "development"
  resources:
    cpu: "100m"
    memory: "256Mi"
```

### E2 Node Simulation

```yaml
apiVersion: nephoran.com/v1alpha1
kind: E2NodeSet
metadata:
  name: test-enb-cluster
spec:
  intent: "Create 3 simulated eNodeB instances for load testing"
  replica: 3
  nodeType: "eNodeB"
  configuration:
    cellId: "001"
    trackingAreaCode: "123"
```

### O-RAN Component Deployment

```yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: oran-near-rt-ric
spec:
  intent: "Deploy Near-RT RIC with basic xApps for traffic steering"
  components:
    - name: "near-rt-ric"
      type: "ric"
    - name: "traffic-steering-xapp"
      type: "xapp"
```

## Development Workflow

### Making Changes

1. **Modify Code** - Edit Go source files in `pkg/` or `cmd/`
2. **Build Locally** - `make build`
3. **Run Tests** - `go test ./pkg/controllers/...`
4. **Build Images** - `make docker-build`
5. **Update Deployment** - `kubectl rollout restart deployment/nephio-bridge -n nephoran-system`

### Debugging Common Issues

#### Intent Stuck in Pending

```bash
# Check controller logs
kubectl logs -n nephoran-system deployment/nephio-bridge --tail=50

# Check if LLM processor is responding
kubectl port-forward -n nephoran-system svc/llm-processor 8080:8080
curl http://localhost:8080/health
```

#### LLM API Errors

```bash
# Verify API key is set correctly
kubectl get secret llm-secrets -n nephoran-system -o yaml

# Check for quota/rate limiting issues
kubectl logs -n nephoran-system deployment/llm-processor | grep -i error
```

#### Resource Creation Failures

```bash
# Check RBAC permissions
kubectl auth can-i create configmaps --as=system:serviceaccount:nephoran-system:nephio-bridge

# Verify CRDs are properly installed
kubectl get crd networkintents.nephoran.com -o yaml
```

## Next Steps

### Explore Advanced Features

1. **RAG Integration** - Enable vector database for domain knowledge
2. **GitOps Workflow** - Connect with Nephio package management
3. **Multi-cluster Deployment** - Scale across environments
4. **Custom xApp Development** - Create your own O-RAN applications

### Learn More

- [Integration Patterns](integration-patterns.md) - Common integration scenarios
- [Troubleshooting Guide](troubleshooting.md) - Detailed problem resolution
- [API Reference](../docs/API_REFERENCE.md) - Complete API documentation
- [Architecture Overview](../docs/architecture.md) - System design deep dive

### Join the Community

- GitHub Issues - Report bugs and request features
- Discussions - Ask questions and share experiences
- Contribute - Submit pull requests and improvements

## Important Notes

- **Experimental Software** - This is a research project, not production-ready
- **Simulated Deployments** - Most network functions are simulated via ConfigMaps
- **API Costs** - Monitor your OpenAI API usage and costs
- **Resource Usage** - The system has minimal resource requirements but can grow with scale

---

**Warning**: This software is for educational and research purposes. Do not use in production environments without thorough testing and validation.