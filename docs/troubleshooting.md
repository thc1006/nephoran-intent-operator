# Troubleshooting Guide

## Overview

This comprehensive troubleshooting guide helps you diagnose and resolve common issues with the Nephoran Intent Operator. The guide is organized by component and symptom to help you quickly identify and fix problems.

## Quick Diagnosis Tools

### Health Check Commands

```bash
# Overall system health
kubectl get pods -n nephoran-system
kubectl get networkintents --all-namespaces
kubectl get e2nodesets --all-namespaces

# Service endpoints health
kubectl port-forward -n nephoran-system svc/llm-processor 8080:8080 &
curl http://localhost:8080/health

kubectl port-forward -n nephoran-system svc/rag-api 8081:8081 &
curl http://localhost:8081/health

# Check component logs
kubectl logs -n nephoran-system deployment/nephio-bridge --tail=50
kubectl logs -n nephoran-system deployment/llm-processor --tail=50
kubectl logs -n nephoran-system deployment/oran-adaptor --tail=50
```

### System Status Overview

```bash
#!/bin/bash
# scripts/system-status.sh

echo "=== Nephoran System Status ==="
echo

echo "Namespace Status:"
kubectl get ns nephoran-system

echo
echo "Pod Status:"
kubectl get pods -n nephoran-system

echo
echo "Service Status:"
kubectl get svc -n nephoran-system

echo
echo "NetworkIntent Resources:"
kubectl get networkintents --all-namespaces

echo
echo "Recent Events:"
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | tail -10
```

## Component-Specific Issues

### NetworkIntent Controller Issues

#### Issue: NetworkIntent Stuck in Pending State

**Symptoms:**
- NetworkIntent resources remain in "Pending" phase
- No processing activity in controller logs
- Status conditions show "Waiting for processing"

**Diagnosis:**
```bash
# Check controller is running
kubectl get deployment nephio-bridge -n nephoran-system

# Check controller logs for errors
kubectl logs deployment/nephio-bridge -n nephoran-system --tail=100

# Check if controller is watching the right namespace
kubectl get networkintent <intent-name> -o yaml | grep -A 10 status

# Verify CRD is properly installed
kubectl get crd networkintents.nephoran.com -o yaml
```

**Common Causes & Solutions:**

1. **Controller Pod Not Running**
   ```bash
   # Check pod status
   kubectl get pods -l app=nephio-bridge -n nephoran-system
   
   # If pod is not running, check events
   kubectl describe pod -l app=nephio-bridge -n nephoran-system
   
   # Solution: Restart deployment
   kubectl rollout restart deployment/nephio-bridge -n nephoran-system
   ```

2. **RBAC Permission Issues**
   ```bash
   # Test permissions
   kubectl auth can-i get networkintents \
     --as=system:serviceaccount:nephoran-system:nephio-bridge
   kubectl auth can-i update networkintents/status \
     --as=system:serviceaccount:nephoran-system:nephio-bridge
   
   # Solution: Apply correct RBAC
   kubectl apply -f deployments/kubernetes/nephio-bridge-rbac.yaml
   ```

3. **LLM Processor Connection Issues**
   ```bash
   # Test service connectivity
   kubectl run debug-pod --image=busybox -it --rm -- \
     wget -O- http://llm-processor.nephoran-system:8080/health
   
   # Check service and endpoints
   kubectl get svc llm-processor -n nephoran-system
   kubectl get endpoints llm-processor -n nephoran-system
   ```

#### Issue: NetworkIntent Processing Fails

**Symptoms:**
- NetworkIntent moves to "Failed" state
- Error messages in status conditions
- Controller logs show processing errors

**Diagnosis:**
```bash
# Check intent status for error details
kubectl get networkintent <intent-name> -o yaml

# Look for specific error conditions
kubectl get networkintent <intent-name> -o jsonpath='{.status.conditions[?(@.type=="ProcessingError")].message}'

# Check controller logs for detailed errors
kubectl logs deployment/nephio-bridge -n nephoran-system | grep ERROR
```

**Common Solutions:**

1. **Invalid Intent Format**
   ```yaml
   # Example of properly formatted intent
   apiVersion: nephoran.com/v1alpha1
   kind: NetworkIntent
   metadata:
     name: valid-intent
   spec:
     intent: "Deploy AMF with high availability"
     target_environment: "development"  # Required field
     priority: "medium"  # Must be: low, medium, high, critical
   ```

2. **Resource Limits Exceeded**
   ```bash
   # Check resource quotas
   kubectl describe resourcequota -n nephoran-system
   
   # Check node resources
   kubectl top nodes
   kubectl describe nodes
   ```

### LLM Processor Issues

#### Issue: LLM API Calls Failing

**Symptoms:**
- HTTP 401/403 errors from OpenAI API
- "API key invalid" or "quota exceeded" errors
- Processing timeouts

**Diagnosis:**
```bash
# Check API key configuration
kubectl get secret llm-secrets -n nephoran-system -o yaml

# Test API key manually
export OPENAI_API_KEY=$(kubectl get secret llm-secrets -n nephoran-system -o jsonpath='{.data.openai-api-key}' | base64 -d)
curl -H "Authorization: Bearer $OPENAI_API_KEY" https://api.openai.com/v1/models

# Check processor logs
kubectl logs deployment/llm-processor -n nephoran-system | grep -E "(ERROR|WARN)"
```

**Solutions:**

1. **Invalid API Key**
   ```bash
   # Update API key secret
   kubectl create secret generic llm-secrets \
     --from-literal=openai-api-key="sk-your-new-key" \
     --namespace nephoran-system \
     --dry-run=client -o yaml | kubectl apply -f -
   
   # Restart processor to pick up new key
   kubectl rollout restart deployment/llm-processor -n nephoran-system
   ```

2. **API Quota Exceeded**
   ```bash
   # Check usage and billing in OpenAI dashboard
   # Temporarily use a different API key or wait for quota reset
   
   # Enable circuit breaker to handle rate limits
   kubectl patch deployment llm-processor -n nephoran-system -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "llm-processor",
               "env": [
                 {"name": "CIRCUIT_BREAKER_ENABLED", "value": "true"},
                 {"name": "RATE_LIMIT_ENABLED", "value": "true"}
               ]
             }
           ]
         }
       }
     }
   }'
   ```

3. **Network Connectivity Issues**
   ```bash
   # Test external connectivity from cluster
   kubectl run debug-pod --image=curlimages/curl -it --rm -- \
     curl -v https://api.openai.com/v1/models
   
   # Check egress network policies
   kubectl get networkpolicy -n nephoran-system
   ```

#### Issue: High Processing Latency

**Symptoms:**
- Long delays between intent creation and processing
- Timeout errors in controller logs
- Poor user experience

**Diagnosis:**
```bash
# Check processing metrics
kubectl port-forward -n nephoran-system svc/llm-processor 8080:8080 &
curl http://localhost:8080/metrics | grep duration

# Monitor resource usage
kubectl top pod -n nephoran-system

# Check for resource throttling
kubectl describe pod -l app=llm-processor -n nephoran-system
```

**Solutions:**

1. **Increase Resource Limits**
   ```yaml
   # Update deployment resources
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: llm-processor
   spec:
     template:
       spec:
         containers:
         - name: llm-processor
           resources:
             requests:
               cpu: "200m"
               memory: "512Mi"
             limits:
               cpu: "500m"
               memory: "1Gi"
   ```

2. **Enable Caching**
   ```bash
   # Enable Redis cache for responses
   kubectl apply -f deployments/kustomize/base/rag-api/redis-cache.yaml
   
   # Update processor configuration
   kubectl patch deployment llm-processor -n nephoran-system -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "llm-processor",
               "env": [
                 {"name": "CACHE_ENABLED", "value": "true"},
                 {"name": "CACHE_TTL", "value": "3600"}
               ]
             }
           ]
         }
       }
     }
   }'
   ```

### RAG System Issues

#### Issue: Vector Database Connection Errors

**Symptoms:**
- RAG API returns 500 errors
- "Cannot connect to Weaviate" errors
- Empty or irrelevant context in responses

**Diagnosis:**
```bash
# Check Weaviate pod status
kubectl get pods -l app=weaviate -n nephoran-system

# Test Weaviate connectivity
kubectl port-forward -n nephoran-system svc/weaviate 8080:8080 &
curl http://localhost:8080/v1/meta

# Check RAG API logs
kubectl logs deployment/rag-api -n nephoran-system
```

**Solutions:**

1. **Weaviate Pod Issues**
   ```bash
   # Check Weaviate logs
   kubectl logs -l app=weaviate -n nephoran-system
   
   # Check persistent storage
   kubectl get pvc -n nephoran-system
   kubectl describe pvc weaviate-data -n nephoran-system
   
   # Restart Weaviate if needed
   kubectl rollout restart statefulset/weaviate -n nephoran-system
   ```

2. **Schema Issues**
   ```bash
   # Check Weaviate schema
   kubectl port-forward -n nephoran-system svc/weaviate 8080:8080 &
   curl http://localhost:8080/v1/schema
   
   # Reinitialize schema if corrupted
   kubectl exec -it -n nephoran-system statefulset/weaviate -- /bin/sh
   # Inside pod: delete and recreate schema
   ```

#### Issue: Poor RAG Context Quality

**Symptoms:**
- Irrelevant information in LLM responses
- Low confidence scores
- Processing results don't match intent

**Diagnosis:**
```bash
# Check knowledge base content
kubectl port-forward -n nephoran-system svc/weaviate 8080:8080 &
curl "http://localhost:8080/v1/objects?class=TelecomKnowledge&limit=10"

# Test search queries
curl -X POST http://localhost:8080/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ Get { TelecomKnowledge(nearText: {concepts: [\"AMF deployment\"]}) { content confidence } } }"}'
```

**Solutions:**

1. **Update Knowledge Base**
   ```bash
   # Repopulate vector store with updated content
   kubectl create job populate-kb --from=cronjob/knowledge-base-update -n nephoran-system
   
   # Check job completion
   kubectl get job populate-kb -n nephoran-system
   kubectl logs job/populate-kb -n nephoran-system
   ```

2. **Improve Embeddings**
   ```bash
   # Update embedding model configuration
   kubectl patch deployment rag-api -n nephoran-system -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "rag-api",
               "env": [
                 {"name": "EMBEDDING_MODEL", "value": "text-embedding-3-large"},
                 {"name": "SIMILARITY_THRESHOLD", "value": "0.8"}
               ]
             }
           ]
         }
       }
     }
   }'
   ```

### E2NodeSet Controller Issues

#### Issue: E2 Nodes Not Creating

**Symptoms:**
- E2NodeSet resource exists but no nodes are created
- ConfigMaps for nodes are missing
- E2 Manager connection errors

**Diagnosis:**
```bash
# Check E2NodeSet status
kubectl get e2nodeset -o yaml

# Check controller logs
kubectl logs deployment/oran-adaptor -n nephoran-system

# Verify node configurations
kubectl get configmaps -l nephoran.com/component=e2node
```

**Solutions:**

1. **Controller Configuration**
   ```yaml
   # Check E2NodeSet specification
   apiVersion: nephoran.com/v1alpha1
   kind: E2NodeSet
   metadata:
     name: test-nodes
   spec:
     replica: 3
     nodeType: "gNodeB"  # Must be valid type
     configuration:
       cellId: "001"
       trackingAreaCode: "123"
   ```

2. **Resource Creation Issues**
   ```bash
   # Check RBAC permissions for ConfigMap creation
   kubectl auth can-i create configmaps \
     --as=system:serviceaccount:nephoran-system:oran-adaptor
   
   # Apply correct RBAC if needed
   kubectl apply -f deployments/kubernetes/oran-adaptor-rbac.yaml
   ```

## Performance Issues

### High Memory Usage

**Symptoms:**
- Pods being OOMKilled
- Slow response times
- Memory usage growing over time

**Diagnosis:**
```bash
# Check memory usage
kubectl top pods -n nephoran-system

# Check for memory leaks
kubectl get events -n nephoran-system | grep OOMKilled

# Monitor memory usage over time
kubectl exec -n nephoran-system deployment/llm-processor -- ps aux
```

**Solutions:**

1. **Increase Memory Limits**
   ```yaml
   # Update resource limits
   spec:
     containers:
     - name: llm-processor
       resources:
         limits:
           memory: "2Gi"
         requests:
           memory: "1Gi"
   ```

2. **Enable Memory Optimization**
   ```bash
   # Enable garbage collection tuning
   kubectl patch deployment llm-processor -n nephoran-system -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "llm-processor",
               "env": [
                 {"name": "GOGC", "value": "100"},
                 {"name": "GOMEMLIMIT", "value": "1800MiB"}
               ]
             }
           ]
         }
       }
     }
   }'
   ```

### High CPU Usage

**Symptoms:**
- CPU throttling
- Slow request processing
- High load averages

**Solutions:**

1. **Scale Horizontally**
   ```bash
   # Increase replica count
   kubectl scale deployment llm-processor --replicas=3 -n nephoran-system
   
   # Or enable HPA
   kubectl apply -f deployments/kustomize/base/llm-processor/hpa.yaml
   ```

2. **Optimize Processing**
   ```bash
   # Enable batch processing
   kubectl patch deployment llm-processor -n nephoran-system -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "llm-processor",
               "env": [
                 {"name": "BATCH_SIZE", "value": "5"},
                 {"name": "BATCH_TIMEOUT", "value": "30s"}
               ]
             }
           ]
         }
       }
     }
   }'
   ```

## Network and Connectivity Issues

### Service Discovery Problems

**Symptoms:**
- Services cannot connect to each other
- DNS resolution failures
- Connection timeouts

**Diagnosis:**
```bash
# Test DNS resolution
kubectl run debug-pod --image=busybox -it --rm -- \
  nslookup llm-processor.nephoran-system.svc.cluster.local

# Check service endpoints
kubectl get endpoints -n nephoran-system

# Test service connectivity
kubectl run debug-pod --image=busybox -it --rm -- \
  telnet llm-processor.nephoran-system 8080
```

**Solutions:**

1. **Service Configuration Issues**
   ```bash
   # Check service selectors match pod labels
   kubectl get svc llm-processor -n nephoran-system -o yaml
   kubectl get pods -l app=llm-processor -n nephoran-system --show-labels
   ```

2. **Network Policy Restrictions**
   ```bash
   # Check network policies
   kubectl get networkpolicy -n nephoran-system
   
   # Temporarily disable network policies for testing
   kubectl delete networkpolicy --all -n nephoran-system
   ```

### External API Connectivity

**Diagnosis:**
```bash
# Test external connectivity
kubectl run debug-pod --image=curlimages/curl -it --rm -- \
  curl -v https://api.openai.com/v1/models

# Check egress rules
kubectl get networkpolicy -n nephoran-system -o yaml
```

**Solutions:**

1. **Update Network Policies**
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: allow-external-apis
   spec:
     podSelector:
       matchLabels:
         app: llm-processor
     policyTypes:
     - Egress
     egress:
     - to: []
       ports:
       - protocol: TCP
         port: 443
       - protocol: TCP
         port: 80
   ```

## Data and Storage Issues

### Persistent Volume Problems

**Symptoms:**
- Weaviate pod stuck in Pending state
- Data loss after pod restarts
- Storage capacity errors

**Diagnosis:**
```bash
# Check PVC status
kubectl get pvc -n nephoran-system

# Check storage class
kubectl get storageclass

# Check node disk space
kubectl get nodes -o wide
kubectl describe node <node-name>
```

**Solutions:**

1. **Storage Class Issues**
   ```bash
   # Create or update storage class
   kubectl apply -f - <<EOF
   apiVersion: storage.k8s.io/v1
   kind: StorageClass
   metadata:
     name: fast-ssd
   provisioner: kubernetes.io/gce-pd
   parameters:
     type: pd-ssd
   EOF
   ```

2. **Increase Storage Size**
   ```bash
   # Expand PVC (if storage class supports it)
   kubectl patch pvc weaviate-data -n nephoran-system -p '{"spec":{"resources":{"requests":{"storage":"100Gi"}}}}'
   ```

## Debugging Tools and Scripts

### Log Aggregation Script

```bash
#!/bin/bash
# scripts/collect-logs.sh

NAMESPACE=${1:-nephoran-system}
OUTPUT_DIR="logs-$(date +%Y%m%d-%H%M%S)"

mkdir -p $OUTPUT_DIR

# Collect pod logs
for deployment in nephio-bridge llm-processor oran-adaptor rag-api; do
    echo "Collecting logs for $deployment..."
    kubectl logs deployment/$deployment -n $NAMESPACE > $OUTPUT_DIR/$deployment.log 2>&1
done

# Collect events
kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' > $OUTPUT_DIR/events.log

# Collect resource status
kubectl get all -n $NAMESPACE -o yaml > $OUTPUT_DIR/resources.yaml
kubectl get networkintents --all-namespaces -o yaml > $OUTPUT_DIR/intents.yaml
kubectl get e2nodesets --all-namespaces -o yaml > $OUTPUT_DIR/e2nodesets.yaml

echo "Logs collected in $OUTPUT_DIR/"
```

### Health Check Script

```bash
#!/bin/bash
# scripts/health-check.sh

NAMESPACE=nephoran-system
FAILED=0

echo "=== Nephoran Health Check ==="

# Check namespace
if ! kubectl get namespace $NAMESPACE >/dev/null 2>&1; then
    echo "❌ Namespace $NAMESPACE not found"
    FAILED=1
else
    echo "✅ Namespace $NAMESPACE exists"
fi

# Check deployments
for deployment in nephio-bridge llm-processor oran-adaptor; do
    if kubectl get deployment $deployment -n $NAMESPACE >/dev/null 2>&1; then
        READY=$(kubectl get deployment $deployment -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
        DESIRED=$(kubectl get deployment $deployment -n $NAMESPACE -o jsonpath='{.spec.replicas}')
        if [ "$READY" = "$DESIRED" ]; then
            echo "✅ $deployment is ready ($READY/$DESIRED)"
        else
            echo "❌ $deployment is not ready ($READY/$DESIRED)"
            FAILED=1
        fi
    else
        echo "❌ $deployment not found"
        FAILED=1
    fi
done

# Check services
for service in llm-processor rag-api weaviate; do
    if kubectl get service $service -n $NAMESPACE >/dev/null 2>&1; then
        echo "✅ Service $service exists"
    else
        echo "❌ Service $service not found"
        FAILED=1
    fi
done

# Check CRDs
for crd in networkintents.nephoran.com e2nodesets.nephoran.com; do
    if kubectl get crd $crd >/dev/null 2>&1; then
        echo "✅ CRD $crd is installed"
    else
        echo "❌ CRD $crd not found"
        FAILED=1
    fi
done

if [ $FAILED -eq 0 ]; then
    echo "✅ All health checks passed"
    exit 0
else
    echo "❌ Some health checks failed"
    exit 1
fi
```

## Getting Help

### Community Resources

- **GitHub Issues**: Report bugs and request features
- **Discussions**: Ask questions and share experiences
- **Documentation**: Check the latest documentation for updates

### Support Checklist

When reporting issues, please include:

1. **System Information**
   - Kubernetes version: `kubectl version`
   - Nephoran version: `kubectl get deployment nephio-bridge -o yaml | grep image`
   - Cloud provider/environment

2. **Error Details**
   - Exact error messages
   - Component logs (use log collection script)
   - Resource configurations

3. **Steps to Reproduce**
   - Detailed steps to reproduce the issue
   - Expected vs actual behavior
   - Sample intent configurations

4. **Environment Details**
   - Resource limits and usage
   - Network configuration
   - External dependencies

---

**Remember**: This is experimental software. Many issues may require code-level investigation. Always check the latest documentation and GitHub issues for known problems and solutions.