# Nephoran Intent Operator - Quick Start Guide

**Version**: 1.0
**Date**: 2026-02-21
**Kubernetes**: v1.35.1
**Status**: Production-Ready

---

## Overview

This quick start guide provides step-by-step instructions to use the Nephoran Intent Operator for AI-powered 5G network orchestration. The system translates natural language intents into executable 5G network configurations.

---

## Prerequisites

- Kubernetes cluster v1.35.1+ running
- kubectl configured and accessible
- System health validated (run `./scripts/health-check.sh`)

---

## 🚀 Getting Started in 5 Minutes

### Step 1: Verify System Health

```bash
# Run comprehensive health check
./scripts/health-check.sh

# Expected output: "✅ ALL CRITICAL COMPONENTS HEALTHY"
# Pass rate should be 100% (57/57 checks)
```

### Step 2: Access RAG Service

The RAG service processes natural language intents and converts them to structured Kubernetes manifests.

```bash
# Get RAG service pod name
export RAG_POD=$(kubectl get pod -n rag-service -o jsonpath='{.items[0].metadata.name}')

# Test RAG service health
kubectl exec -n rag-service $RAG_POD -- \
  curl -s http://localhost:8000/health | jq .

# View RAG statistics
kubectl exec -n rag-service $RAG_POD -- \
  curl -s http://localhost:8000/stats | jq .
```

### Step 3: Process Your First Intent

#### Example 1: Scale a Network Function

```bash
kubectl exec -n rag-service $RAG_POD -- \
  curl -s -X POST http://localhost:8000/process_intent \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Scale the UPF to 3 replicas for high throughput"
  }' | jq .
```

**Expected Response**:
```json
{
  "intent_id": "uuid-here",
  "original_intent": "Scale the UPF to 3 replicas for high throughput",
  "structured_output": {
    "type": "NetworkFunctionScale",
    "name": "UPF",
    "namespace": "default",
    "replicas": 3,
    "resource_adjustments": {
      "cpu_scale_factor": 1.0,
      "memory_scale_factor": 1.0
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 2417,
    "confidence_score": 0.4,
    "model_version": "llama3.1:8b-instruct-q5_K_M"
  }
}
```

#### Example 2: Deploy a New Network Function

```bash
kubectl exec -n rag-service $RAG_POD -- \
  curl -s -X POST http://localhost:8000/process_intent \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy 2 replicas of SMF with high CPU priority"
  }' | jq .
```

#### Example 3: Configure Network Slice

```bash
kubectl exec -n rag-service $RAG_POD -- \
  curl -s -X POST http://localhost:8000/process_intent \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Create eMBB network slice with 10 Gbps throughput"
  }' | jq .
```

### Step 4: Apply Structured Intent to Kubernetes

Once you receive the structured output, you can create a NetworkIntent CRD:

```bash
# Save structured output to file
cat > my-intent.yaml <<EOF
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: upf-scale-intent
  namespace: default
spec:
  intentType: scale
  targetNF: UPF
  targetNamespace: free5gc
  desiredReplicas: 3
  priority: high
  reason: "High throughput requirement"
EOF

# Apply the intent
kubectl apply -f my-intent.yaml

# Verify intent created
kubectl get networkintents -A
```

### Step 5: Monitor Intent Execution

```bash
# Watch NetworkIntent status
kubectl get networkintent upf-scale-intent -o yaml

# Check operator logs (if running)
kubectl logs -n nephoran-system \
  deployment/nephoran-operator-controller-manager -f

# Verify 5G NF scaled
kubectl get deployment -n free5gc upf2-free5gc-upf-upf2
```

---

## 📊 Access Web Interfaces

### Grafana Dashboard

```bash
# Get Grafana URL
echo "http://$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}'):30300"

# Default credentials
# Username: admin
# Password: prom-operator
```

### Free5GC WebUI

```bash
# Get WebUI URL
echo "http://$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}'):30500"

# Default credentials (check Free5GC docs)
```

### RAG Service Swagger UI

```bash
# Port-forward RAG service
kubectl port-forward -n rag-service svc/rag-service 8000:8000 &

# Open in browser
echo "http://localhost:8000/docs"
```

---

## 🔧 Common Operations

### View All Deployed Components

```bash
# Summary view
kubectl get all -A | grep -E "nephoran|rag|ollama|weaviate|free5gc|ricplt"

# Detailed Helm releases
helm list -A

# Check CRDs
kubectl get crd | grep nephoran
```

### Check RAG Knowledge Base

```bash
kubectl exec -n rag-service $RAG_POD -- \
  curl -s http://localhost:8000/stats | jq '{
    knowledge_objects: .health.knowledge_objects,
    weaviate_ready: .health.weaviate_ready,
    model: .health.model,
    cache_size: .health.cache_size
  }'
```

### Test Ollama LLM Directly

```bash
# List available models
kubectl exec -n ollama deployment/ollama -- ollama list

# Pull a new model
kubectl exec -n ollama deployment/ollama -- \
  ollama pull llama3.1:8b-instruct-q5_K_M

# Test model directly
kubectl exec -n ollama deployment/ollama -- \
  ollama run llama3.1:8b-instruct-q5_K_M "What is a UPF in 5G?"
```

### Query Weaviate Vector Database

```bash
# Run query pod
kubectl run -it --rm weaviate-test \
  --image=curlimages/curl:latest \
  --restart=Never \
  -- curl -s http://weaviate.weaviate.svc.cluster.local:80/v1/meta | jq .

# Check schema
kubectl run -it --rm weaviate-test \
  --image=curlimages/curl:latest \
  --restart=Never \
  -- curl -s http://weaviate.weaviate.svc.cluster.local:80/v1/schema | jq .
```

---

## 🎯 Example Use Cases

### Use Case 1: Auto-Scaling Based on Load

```bash
# Intent: Scale UPF when load exceeds threshold
kubectl exec -n rag-service $RAG_POD -- \
  curl -s -X POST http://localhost:8000/process_intent \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Auto-scale UPF from 1 to 5 replicas when CPU exceeds 80%"
  }' | jq .
```

### Use Case 2: Network Slice Provisioning

```bash
# Intent: Create URLLC slice
kubectl exec -n rag-service $RAG_POD -- \
  curl -s -X POST http://localhost:8000/process_intent \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Provision URLLC network slice with 1ms latency and 99.999% reliability"
  }' | jq .
```

### Use Case 3: Multi-Site Deployment

```bash
# Intent: Deploy across multiple clusters
kubectl exec -n rag-service $RAG_POD -- \
  curl -s -X POST http://localhost:8000/process_intent \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy AMF in edge cluster and SMF in core cluster"
  }' | jq .
```

---

## 🐛 Troubleshooting

### RAG Service Not Responding

```bash
# Check pod status
kubectl get pods -n rag-service

# View logs
kubectl logs -n rag-service deployment/rag-service --tail=100

# Restart RAG service
kubectl rollout restart deployment -n rag-service rag-service
```

### Ollama Model Not Loaded

```bash
# Check available models
kubectl exec -n ollama deployment/ollama -- ollama list

# Pull required model
kubectl exec -n ollama deployment/ollama -- \
  ollama pull llama3.1:8b-instruct-q5_K_M

# Verify model loaded
kubectl exec -n ollama deployment/ollama -- \
  ollama run llama3.1:8b-instruct-q5_K_M "test" --verbose
```

### Weaviate Connection Issues

```bash
# Check Weaviate health
kubectl run -it --rm weaviate-test \
  --image=curlimages/curl:latest \
  --restart=Never \
  -- curl -s http://weaviate.weaviate.svc.cluster.local:80/v1/.well-known/ready

# Check RAG → Weaviate connectivity
kubectl exec -n rag-service $RAG_POD -- \
  curl -s http://weaviate.weaviate.svc.cluster.local:80/v1/meta
```

### NetworkIntent Not Reconciling

```bash
# Check if operator is running
kubectl get deployment -n nephoran-system

# Scale up operator if needed
kubectl scale deployment -n nephoran-system \
  nephoran-operator-controller-manager --replicas=1

# Check operator logs
kubectl logs -n nephoran-system \
  deployment/nephoran-operator-controller-manager -f
```

---

## 📚 Next Steps

1. **Explore O-RAN Interfaces**:
   - Test A1 Policy API: `kubectl get svc -n ricplt service-ricplt-a1mediator-http`
   - Test E2 Manager: `kubectl get svc -n ricplt service-ricplt-e2mgr-http`
   - Test O1 NETCONF: `kubectl get svc -n ricplt service-ricplt-o1mediator-tcp-netconf`

2. **Enhance RAG Knowledge Base**:
   ```bash
   # Upload custom documentation
   kubectl cp my-docs/ rag-service/$RAG_POD:/app/knowledge_base/

   # Trigger re-indexing
   kubectl exec -n rag-service $RAG_POD -- \
     curl -s -X POST http://localhost:8000/knowledge/populate
   ```

3. **Test E2E Workflows**:
   - Run E2E test scripts in `tests/e2e/bash/`
   - Test UERANSIM UE registration: `kubectl logs -n free5gc ueransim-ue`
   - Test PDU session establishment

4. **Monitor System Metrics**:
   - Access Grafana: http://<node-ip>:30300
   - View Prometheus metrics: `kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090`

---

## 🔐 Security Notes

- RAG service runs with cluster-internal networking only
- Sensitive data should NOT be included in natural language intents
- Use Kubernetes RBAC for NetworkIntent CRD access control
- Enable audit logging for production deployments

---

## 📖 Additional Resources

- [System Architecture Validation](./SYSTEM_ARCHITECTURE_VALIDATION.md) - Complete component inventory
- [5G Integration Plan V2](./5G_INTEGRATION_PLAN_V2.md) - Detailed deployment guide
- [CLAUDE.md](../CLAUDE.md) - AI agent instructions
- [Health Check Script](../scripts/health-check.sh) - Automated validation

---

**Last Updated**: 2026-02-21
**Maintainer**: Nephoran Intent Operator Team
