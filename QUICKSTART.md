# 🚀 Nephoran Intent Operator - 15-Minute Quick Start Guide

> **Transform natural language into deployed network functions in under 15 minutes!**

This guide will take you from zero to your first deployed intent-driven network function in exactly 15 minutes. Follow along with our interactive tutorial and embedded terminal recordings.

---

## ⏱️ Time Breakdown

| Phase | Duration | Activity |
|-------|----------|----------|
| Prerequisites | 2 min | Tool installation check |
| Environment Setup | 5 min | Local Kubernetes cluster |
| First Intent | 5 min | Deploy AMF network function |
| Validation | 2 min | Verify deployment |
| Troubleshooting | 1 min | Common fixes |

---

## 📋 Prerequisites Check (2 minutes)

### Required Tools

Before starting, ensure you have the following tools installed:

```bash
# Check Docker (required)
docker --version
# Expected: Docker version 20.10+ 

# Check kubectl (required)
kubectl version --client
# Expected: Client Version: v1.27+

# Check Git (required)
git --version
# Expected: git version 2.30+

# Check Go (optional, for development)
go version
# Expected: go version go1.21+
```

### Quick Install Commands

If any tools are missing, use these quick install commands:

<details>
<summary>🐧 Linux/WSL</summary>

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Kind (Kubernetes in Docker)
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```
</details>

<details>
<summary>🍎 macOS</summary>

```bash
# Install with Homebrew
brew install docker kubectl kind

# Start Docker Desktop
open -a Docker
```
</details>

<details>
<summary>🪟 Windows</summary>

```powershell
# Install with Chocolatey
choco install docker-desktop kubernetes-cli kind -y

# Or use winget
winget install Docker.DockerDesktop
winget install Kubernetes.kubectl
```
</details>

### ✅ Prerequisites Validation

Run this validation script to ensure everything is ready:

```bash
#!/bin/bash
echo "🔍 Checking prerequisites..."

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Check function
check_tool() {
    if command -v $1 &> /dev/null; then
        echo -e "${GREEN}✓${NC} $1 is installed"
        return 0
    else
        echo -e "${RED}✗${NC} $1 is not installed"
        return 1
    fi
}

# Run checks
READY=true
check_tool docker || READY=false
check_tool kubectl || READY=false
check_tool git || READY=false

if $READY; then
    echo -e "\n${GREEN}🎉 All prerequisites met! You're ready to start.${NC}"
else
    echo -e "\n${RED}⚠️  Please install missing tools before continuing.${NC}"
    exit 1
fi
```

---

## 🛠️ Environment Setup (5 minutes)

### Step 1: Clone the Repository (30 seconds)

```bash
# Clone the repository
git clone https://github.com/thc1006/nephoran-intent-operator.git
cd nephoran-intent-operator

# Verify you're in the right directory
pwd
# Expected output: .../nephoran-intent-operator
```

### Step 2: Create Local Kubernetes Cluster (2 minutes)

We'll use Kind (Kubernetes in Docker) for a quick local setup:

```bash
# Create the Kind cluster configuration
cat <<EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: nephoran-quickstart
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

# Create the cluster
kind create cluster --config=kind-config.yaml

# Verify cluster is running
kubectl cluster-info
kubectl get nodes
```

Expected output:
```
NAME                           STATUS   ROLES           AGE   VERSION
nephoran-quickstart-control-plane   Ready    control-plane   1m    v1.27.3
nephoran-quickstart-worker          Ready    <none>          1m    v1.27.3
nephoran-quickstart-worker2         Ready    <none>          1m    v1.27.3
```

### Step 3: Install CRDs and Core Components (2 minutes)

```bash
# Create namespace
kubectl create namespace nephoran-system

# Install CRDs
kubectl apply -f deployments/crds/

# Verify CRDs are installed
kubectl get crds | grep nephoran
```

Expected output:
```
networkintents.nephoran.com     2024-01-20T10:00:00Z
managedelements.nephoran.com    2024-01-20T10:00:00Z
e2nodesets.nephoran.com         2024-01-20T10:00:00Z
```

### Step 4: Deploy Core Services (30 seconds)

For the quickstart, we'll deploy a minimal configuration:

```bash
# Create secrets for LLM processor (using demo values)
kubectl create secret generic llm-secrets \
  --from-literal=openai-api-key=demo-key-replace-in-production \
  -n nephoran-system

# Apply the quickstart deployment
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: nephoran-quickstart-config
  namespace: nephoran-system
data:
  config.yaml: |
    mode: demo
    llm:
      provider: mock
      mockResponses: true
    rag:
      enabled: false
    telemetry:
      enabled: true
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-controller
  namespace: nephoran-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nephoran-controller
  template:
    metadata:
      labels:
        app: nephoran-controller
    spec:
      containers:
      - name: controller
        image: ghcr.io/thc1006/nephoran-intent-operator:latest
        imagePullPolicy: IfNotPresent
        env:
        - name: DEMO_MODE
          value: "true"
        - name: LOG_LEVEL
          value: "info"
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 9090
          name: health
EOF

# Wait for deployment to be ready
kubectl wait --for=condition=available --timeout=120s \
  deployment/nephoran-controller -n nephoran-system
```

### 🎯 Environment Ready Checkpoint

Run this verification:

```bash
echo "🔍 Verifying environment setup..."
kubectl get all -n nephoran-system
```

You should see:
- ✅ 1 deployment running
- ✅ 1 pod in Running state
- ✅ CRDs installed

---

## 🎮 Deploy Your First Intent (5 minutes)

### Step 1: Create Your First Network Intent (1 minute)

Let's deploy an AMF (Access and Mobility Management Function) using natural language:

```bash
# Create the intent file
cat <<EOF > my-first-intent.yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: deploy-amf-quickstart
  namespace: default
  labels:
    tutorial: quickstart
    component: amf
spec:
  intent: |
    Deploy a production-ready AMF (Access and Mobility Management Function) 
    for a 5G core network with the following requirements:
    - High availability with 3 replicas
    - Auto-scaling enabled (min: 3, max: 10 pods)
    - Resource limits: 2 CPU cores, 4GB memory per pod
    - Enable prometheus monitoring on port 9090
    - Configure for urban area with expected 100k UE connections
    - Set up with standard 3GPP interfaces (N1, N2, N11)
EOF

# Apply the intent
kubectl apply -f my-first-intent.yaml
```

### Step 2: Watch the Magic Happen (2 minutes)

Watch as the Nephoran Intent Operator processes your natural language intent:

```bash
# Watch the intent processing
kubectl get networkintent deploy-amf-quickstart -w

# In another terminal, watch the events
kubectl get events --field-selector involvedObject.name=deploy-amf-quickstart -w

# Check controller logs
kubectl logs -n nephoran-system deployment/nephoran-controller -f
```

You'll see the following stages:
1. 📝 **Intent Received** - Natural language captured
2. 🤖 **LLM Processing** - Intent analyzed and translated
3. 📦 **Package Generation** - Kubernetes resources created
4. 🚀 **Deployment Initiated** - Resources applied to cluster
5. ✅ **Deployment Complete** - AMF running

### Step 3: Explore Generated Resources (2 minutes)

The intent creates multiple Kubernetes resources:

```bash
# View the generated resources
kubectl get all -l generated-from=deploy-amf-quickstart

# Inspect the generated deployment
kubectl describe deployment amf-deployment

# Check the generated ConfigMap
kubectl get configmap amf-config -o yaml

# View the HPA (Horizontal Pod Autoscaler)
kubectl get hpa amf-hpa
```

---

## ✅ Validation & Success Criteria (2 minutes)

### Automated Validation Script

Run this comprehensive validation:

```bash
#!/bin/bash
cat <<'SCRIPT' > validate-quickstart.sh
#!/bin/bash

echo "🔍 Running Nephoran Quickstart Validation..."
echo "==========================================="

ERRORS=0
WARNINGS=0

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check CRDs
echo -e "\n📋 Checking CRDs..."
if kubectl get crd networkintents.nephoran.com &>/dev/null; then
    echo -e "${GREEN}✓${NC} NetworkIntent CRD installed"
else
    echo -e "${RED}✗${NC} NetworkIntent CRD missing"
    ((ERRORS++))
fi

# Check controller
echo -e "\n🎮 Checking Controller..."
if kubectl get deployment nephoran-controller -n nephoran-system &>/dev/null; then
    READY=$(kubectl get deployment nephoran-controller -n nephoran-system -o jsonpath='{.status.readyReplicas}')
    if [ "$READY" -ge 1 ]; then
        echo -e "${GREEN}✓${NC} Controller running ($READY replicas ready)"
    else
        echo -e "${RED}✗${NC} Controller not ready"
        ((ERRORS++))
    fi
else
    echo -e "${RED}✗${NC} Controller deployment not found"
    ((ERRORS++))
fi

# Check intent
echo -e "\n📝 Checking Network Intent..."
if kubectl get networkintent deploy-amf-quickstart &>/dev/null; then
    STATUS=$(kubectl get networkintent deploy-amf-quickstart -o jsonpath='{.status.phase}')
    if [ "$STATUS" = "Deployed" ] || [ "$STATUS" = "Ready" ]; then
        echo -e "${GREEN}✓${NC} Intent processed successfully (Status: $STATUS)"
    else
        echo -e "${YELLOW}⚠${NC} Intent status: $STATUS"
        ((WARNINGS++))
    fi
else
    echo -e "${RED}✗${NC} Intent not found"
    ((ERRORS++))
fi

# Check generated resources
echo -e "\n📦 Checking Generated Resources..."
RESOURCES=$(kubectl get all -l generated-from=deploy-amf-quickstart 2>/dev/null | wc -l)
if [ $RESOURCES -gt 1 ]; then
    echo -e "${GREEN}✓${NC} Generated resources found ($((RESOURCES-1)) resources)"
    kubectl get all -l generated-from=deploy-amf-quickstart --no-headers
else
    echo -e "${YELLOW}⚠${NC} No generated resources found yet"
    ((WARNINGS++))
fi

# Summary
echo -e "\n==========================================="
echo "📊 VALIDATION SUMMARY"
echo "==========================================="

if [ $ERRORS -eq 0 ]; then
    if [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}🎉 PERFECT!${NC} All checks passed!"
        echo "Your Nephoran Intent Operator is working correctly."
    else
        echo -e "${YELLOW}✅ SUCCESS${NC} with $WARNINGS warnings"
        echo "The system is functional but may need minor adjustments."
    fi
else
    echo -e "${RED}❌ ISSUES FOUND${NC}: $ERRORS errors, $WARNINGS warnings"
    echo "Please check the troubleshooting section below."
fi

exit $ERRORS
SCRIPT

chmod +x validate-quickstart.sh
./validate-quickstart.sh
```

### Success Criteria Checklist

- [ ] ✅ Kind cluster running with 3 nodes
- [ ] ✅ Nephoran CRDs installed
- [ ] ✅ Controller pod running in nephoran-system namespace
- [ ] ✅ NetworkIntent created and processed
- [ ] ✅ Status shows "Deployed" or "Ready"
- [ ] ✅ Generated resources visible in cluster

---

## 🔧 Troubleshooting (1 minute)

### Common Issues & Quick Fixes

<details>
<summary>❌ "Error: cluster nephoran-quickstart not found"</summary>

```bash
# Recreate the cluster
kind delete cluster --name nephoran-quickstart
kind create cluster --config=kind-config.yaml
```
</details>

<details>
<summary>❌ "ImagePullBackOff" for controller pod</summary>

```bash
# Build and load local image
make docker-build
kind load docker-image nephoran-intent-operator:latest --name nephoran-quickstart

# Update deployment to use local image
kubectl set image deployment/nephoran-controller \
  controller=nephoran-intent-operator:latest \
  -n nephoran-system
```
</details>

<details>
<summary>❌ Intent stuck in "Processing" state</summary>

```bash
# Check controller logs for errors
kubectl logs -n nephoran-system deployment/nephoran-controller --tail=50

# If using real OpenAI API, verify secret:
kubectl get secret llm-secrets -n nephoran-system -o yaml

# Restart controller
kubectl rollout restart deployment/nephoran-controller -n nephoran-system
```
</details>

<details>
<summary>❌ No resources generated from intent</summary>

```bash
# Check if intent has errors
kubectl describe networkintent deploy-amf-quickstart

# Look for events
kubectl get events --sort-by='.lastTimestamp' | tail -20

# Verify RBAC permissions
kubectl auth can-i create deployments --as=system:serviceaccount:nephoran-system:default
```
</details>

### Quick Debug Commands

```bash
# Get all relevant logs
kubectl logs -n nephoran-system -l app=nephoran-controller --tail=100

# Describe intent for detailed status
kubectl describe networkintent deploy-amf-quickstart

# Check all resources in namespace
kubectl get all -n nephoran-system

# Export debug bundle
kubectl cluster-info dump --output-directory=/tmp/nephoran-debug
```

---

## 🎯 Next Steps

### Congratulations! You've successfully:
- ✅ Set up a local Kubernetes environment
- ✅ Deployed the Nephoran Intent Operator
- ✅ Created your first intent-driven network function
- ✅ Validated the deployment

### What's Next?

#### 1. **Try More Complex Intents** (5 min)
```bash
# Deploy a complete 5G core network slice
kubectl apply -f examples/networkintent-example.yaml
```

#### 2. **Enable Production Features** (10 min)
- Configure real OpenAI API key
- Deploy Weaviate vector database
- Enable RAG for enhanced processing

#### 3. **Explore Advanced Features**
- [Network Slicing Guide](docs/getting-started.md#network-slicing)
- [O-RAN Integration](docs/ORAN-COMPLIANCE-CERTIFICATION.md)
- [Multi-cluster Deployments](deployments/multi-region/README.md)

#### 4. **Join the Community**
- 📖 [Full Documentation](README.md)
- 🐛 [Report Issues](https://github.com/thc1006/nephoran-intent-operator/issues)
- 💬 [Discussions](https://github.com/thc1006/nephoran-intent-operator/discussions)
- 🤝 [Contributing Guide](CONTRIBUTING.md)

---

## 📹 Terminal Recordings

For visual learners, we've prepared asciinema recordings of each major step:

### Prerequisites Check
```bash
# Play the recording
asciinema play https://asciinema.org/a/nephoran-prereq-check
```

### Environment Setup
```bash
# Play the recording
asciinema play https://asciinema.org/a/nephoran-env-setup
```

### First Intent Deployment
```bash
# Play the recording
asciinema play https://asciinema.org/a/nephoran-first-intent
```

---

## 📊 Performance Metrics

Your quickstart deployment should achieve:

| Metric | Expected Value | Your Value |
|--------|---------------|------------|
| Cluster Setup Time | < 2 min | ___________ |
| CRD Installation | < 30 sec | ___________ |
| Controller Startup | < 1 min | ___________ |
| Intent Processing | < 30 sec | ___________ |
| Total Time | < 15 min | ___________ |

---

## 🆘 Need Help?

If you encounter any issues:

1. Check the [Troubleshooting Guide](docs/troubleshooting.md)
2. Search [existing issues](https://github.com/thc1006/nephoran-intent-operator/issues)
3. Join our [Discord community](https://discord.gg/nephoran)
4. Create a [new issue](https://github.com/thc1006/nephoran-intent-operator/issues/new) with:
   - Your environment details (OS, Kubernetes version)
   - Steps to reproduce
   - Error messages and logs
   - Output of `validate-quickstart.sh`

---

## 🏁 Quick Reference Card

```bash
# Essential Commands Cheat Sheet
# ==============================

# Cluster Management
kind create cluster --config=kind-config.yaml    # Create cluster
kind delete cluster --name nephoran-quickstart   # Delete cluster
kubectl cluster-info                             # Check cluster

# Nephoran Operations
kubectl apply -f my-intent.yaml                  # Deploy intent
kubectl get networkintents                       # List intents
kubectl describe networkintent <name>            # Intent details
kubectl logs -n nephoran-system -l app=nephoran-controller  # Logs

# Troubleshooting
kubectl get events --sort-by='.lastTimestamp'    # Recent events
kubectl get all -n nephoran-system               # System components
./validate-quickstart.sh                         # Run validation
```

---

**🚀 You're now ready to transform network operations with intent-driven automation!**
