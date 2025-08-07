# Nephoran Intent Operator - Complete Troubleshooting Guide

## Table of Contents
1. [Overview](#overview)
2. [Component Health Checks](#component-health-checks)
3. [Common Issues and Resolutions](#common-issues-and-resolutions)
4. [Log Analysis Procedures](#log-analysis-procedures)
5. [Performance Troubleshooting](#performance-troubleshooting)
6. [Network Connectivity Issues](#network-connectivity-issues)
7. [Debugging Tools and Commands](#debugging-tools-and-commands)
8. [Known Issues and Workarounds](#known-issues-and-workarounds)

## Overview

This comprehensive troubleshooting guide provides systematic approaches to diagnose and resolve issues in the Nephoran Intent Operator. The guide follows a structured methodology: Identify, Isolate, Analyze, and Resolve.

**General Troubleshooting Philosophy:**
- Start with high-level health checks
- Work from symptoms to root cause
- Use structured diagnostic commands
- Document findings for knowledge sharing
- Implement fix with minimal disruption

## Component Health Checks

### Core Components Overview

The Nephoran Intent Operator consists of several interconnected components:

```bash
# Get comprehensive component overview
kubectl get pods,services,ingress -n nephoran-system -o wide --show-labels
```

### 1. Nephoran Controller Health Check

**Primary Functions**: NetworkIntent processing, controller reconciliation, status management

```bash
# Basic health check
kubectl get pods -n nephoran-system -l app=nephoran-controller -o wide

# Detailed status with resource usage
kubectl top pods -n nephoran-system -l app=nephoran-controller

# Check controller readiness and liveness probes
kubectl describe pods -n nephoran-system -l app=nephoran-controller | grep -A 5 -B 5 "Liveness\|Readiness"

# Verify controller is processing events
kubectl logs -n nephoran-system -l app=nephoran-controller --tail=20 | grep -E "(Reconciling|Successfully processed)"
```

**Expected Healthy Output:**
```
NAME                                 READY   STATUS    RESTARTS   AGE     IP            NODE
nephoran-controller-7d8b9c5f4-xyz   1/1     Running   0          2d      10.244.1.5   node-1

Readiness:  HTTP GET /healthz on port 8081
Liveness:   HTTP GET /healthz on port 8081
```

**Health Check API Endpoint:**
```bash
kubectl port-forward -n nephoran-system svc/nephoran-controller 8081:8081 &
curl -s http://localhost:8081/healthz | jq .

# Expected response
{
  "status": "healthy",
  "timestamp": "2025-01-07T10:30:00Z",
  "components": {
    "controller": "healthy",
    "webhook": "healthy",
    "metrics": "healthy"
  }
}
```

### 2. LLM Processor Health Check

**Primary Functions**: Natural language processing, intent translation, LLM API integration

```bash
# Check LLM Processor pods
kubectl get pods -n nephoran-system -l app=llm-processor -o wide

# Check service endpoints
kubectl get endpoints -n nephoran-system llm-processor

# Verify configuration
kubectl get configmap llm-processor-config -n nephoran-system -o yaml

# Check secrets (without revealing values)
kubectl get secret llm-secrets -n nephoran-system -o jsonpath='{.data}' | jq 'keys'
```

**LLM Processor API Health:**
```bash
kubectl port-forward -n nephoran-system svc/llm-processor 8080:8080 &

# Health endpoint
curl -s http://localhost:8080/health | jq .

# Metrics endpoint
curl -s http://localhost:8080/metrics | grep -E "(llm_requests_total|llm_request_duration)"

# Circuit breaker status
curl -s http://localhost:8080/admin/circuit-breaker/status | jq .
```

**Expected Healthy Response:**
```json
{
  "status": "healthy",
  "llm_provider": "openai",
  "model": "gpt-4o-mini",
  "circuit_breaker": "closed",
  "cache_hit_rate": 0.78,
  "last_successful_request": "2025-01-07T10:29:45Z"
}
```

### 3. RAG API Health Check

**Primary Functions**: Knowledge retrieval, document search, context enhancement

```bash
# Check RAG API pods
kubectl get pods -n nephoran-system -l app=rag-api -o wide

# Check Weaviate dependency
kubectl get pods -n nephoran-system -l app=weaviate -o wide

# RAG API health
kubectl port-forward -n nephoran-system svc/rag-api 8082:8080 &
curl -s http://localhost:8082/health | jq .

# Vector database connectivity
curl -s http://localhost:8082/admin/weaviate/status | jq .
```

**RAG System Health Test:**
```bash
# Test knowledge retrieval
curl -X POST http://localhost:8082/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "How to deploy AMF in 5G core?",
    "top_k": 3,
    "threshold": 0.7
  }' | jq '.results | length'

# Expected: Should return 1-3 results
```

### 4. Weaviate Vector Database Health

**Primary Functions**: Vector storage, semantic search, knowledge base

```bash
# Check Weaviate StatefulSet
kubectl get statefulsets -n nephoran-system weaviate -o wide

# Check persistent volume claims
kubectl get pvc -n nephoran-system -l app=weaviate

# Direct Weaviate health check
kubectl exec -n nephoran-system -it weaviate-0 -- \
  curl -s http://localhost:8080/v1/meta | jq '.version'

# Check schema and classes
kubectl exec -n nephoran-system -it weaviate-0 -- \
  curl -s http://localhost:8080/v1/schema | jq '.classes[] | .class'
```

**Database Content Verification:**
```bash
# Check document count
kubectl exec -n nephoran-system -it weaviate-0 -- \
  curl -s "http://localhost:8080/v1/objects?limit=1" | jq '.totalResults'

# Check specific classes
for class in TelecomDocument NetworkFunction O_RAN_Spec; do
  echo "Class $class:"
  kubectl exec -n nephoran-system -it weaviate-0 -- \
    curl -s "http://localhost:8080/v1/objects?class=$class&limit=1" | jq '.totalResults'
done
```

### 5. Nephio Bridge Health Check

**Primary Functions**: Nephio integration, package generation, GitOps coordination

```bash
# Check Nephio Bridge deployment
kubectl get deployment -n nephoran-system nephio-bridge -o wide

# Check Nephio Porch connectivity
kubectl get packagerepositories -A
kubectl get packagerevisions -A | head -10

# Verify GitOps repository access
kubectl logs -n nephoran-system -l app=nephio-bridge --tail=50 | grep -E "(git|porch|package)"
```

## Common Issues and Resolutions

### Issue 1: NetworkIntent Stuck in Processing State

**Symptoms:**
- NetworkIntent remains in "Processing" phase for > 5 minutes
- No progress logs from controller
- No errors in controller logs

**Root Cause Analysis:**
```bash
# Check specific NetworkIntent status
kubectl describe networkintent <INTENT_NAME> -n <NAMESPACE>

# Look for events
kubectl get events -n <NAMESPACE> --field-selector involvedObject.name=<INTENT_NAME>

# Check controller reconciliation logs
kubectl logs -n nephoran-system -l app=nephoran-controller | grep <INTENT_NAME>
```

**Common Causes & Solutions:**

**a) LLM API Timeout:**
```bash
# Check LLM processor logs
kubectl logs -n nephoran-system -l app=llm-processor --tail=100 | grep -i timeout

# Solution: Increase timeout or restart LLM processor
kubectl patch configmap llm-processor-config -n nephoran-system --type='merge' -p='{"data":{"timeout":"60s"}}'
kubectl rollout restart deployment/llm-processor -n nephoran-system
```

**b) RAG Service Unavailable:**
```bash
# Check RAG API connectivity
kubectl exec -n nephoran-system -l app=nephoran-controller -- curl -f http://rag-api:8080/health

# Solution: Restart RAG services
kubectl rollout restart deployment/rag-api -n nephoran-system
kubectl rollout restart statefulset/weaviate -n nephoran-system
```

**c) Controller Resource Exhaustion:**
```bash
# Check resource usage
kubectl top pods -n nephoran-system -l app=nephoran-controller

# Check resource limits
kubectl get deployment nephoran-controller -n nephoran-system -o yaml | grep -A 10 resources

# Solution: Increase resource limits
kubectl patch deployment nephoran-controller -n nephoran-system --type='merge' -p='{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "controller",
          "resources": {
            "limits": {"memory": "512Mi", "cpu": "500m"},
            "requests": {"memory": "256Mi", "cpu": "250m"}
          }
        }]
      }
    }
  }
}'
```

### Issue 2: High Error Rate in LLM Processing

**Symptoms:**
- Frequent LLM API failures
- Circuit breaker in "open" state
- High latency in intent processing

**Diagnostic Commands:**
```bash
# Check LLM processor metrics
kubectl port-forward -n nephoran-system svc/llm-processor 8080:8080 &
curl -s http://localhost:8080/metrics | grep -E "(llm_errors_total|llm_request_duration|circuit_breaker_state)"

# Check recent errors
kubectl logs -n nephoran-system -l app=llm-processor --since=1h | grep -i error | tail -20

# Check API key validity
kubectl get secret llm-secrets -n nephoran-system -o jsonpath='{.data.openai-api-key}' | base64 -d | wc -c
```

**Solutions:**

**a) API Rate Limiting:**
```bash
# Check rate limiting configuration
kubectl get configmap llm-processor-config -n nephoran-system -o yaml | grep -A 5 rate

# Implement backoff strategy
kubectl patch configmap llm-processor-config -n nephoran-system --type='merge' -p='{"data":{
  "rate_limit_requests_per_minute": "100",
  "rate_limit_burst": "10",
  "backoff_initial_delay": "1s",
  "backoff_max_delay": "30s"
}}'
```

**b) Circuit Breaker Configuration:**
```bash
# Reset circuit breaker
curl -X POST http://localhost:8080/admin/circuit-breaker/reset

# Adjust circuit breaker settings
kubectl patch configmap llm-processor-config -n nephoran-system --type='merge' -p='{"data":{
  "circuit_breaker_failure_threshold": "10",
  "circuit_breaker_recovery_timeout": "30s",
  "circuit_breaker_half_open_max_requests": "3"
}}'

# Restart to apply changes
kubectl rollout restart deployment/llm-processor -n nephoran-system
```

### Issue 3: Weaviate Connection Failures

**Symptoms:**
- RAG API unable to connect to Weaviate
- Knowledge retrieval failures
- Vector search errors

**Diagnostic Steps:**
```bash
# Check Weaviate pod status
kubectl get pods -n nephoran-system -l app=weaviate -o wide

# Check Weaviate logs
kubectl logs -n nephoran-system -l app=weaviate --tail=100

# Test direct connectivity
kubectl exec -n nephoran-system -l app=rag-api -- curl -v http://weaviate:8080/v1/meta
```

**Common Solutions:**

**a) Weaviate Not Ready:**
```bash
# Check readiness probe
kubectl describe pod -n nephoran-system -l app=weaviate | grep -A 10 Readiness

# Wait for Weaviate to be ready
kubectl wait --for=condition=ready pod -l app=weaviate -n nephoran-system --timeout=300s

# Check persistent volume
kubectl get pvc -n nephoran-system -l app=weaviate
kubectl describe pvc -n nephoran-system -l app=weaviate
```

**b) Schema Issues:**
```bash
# Check current schema
kubectl exec -n nephoran-system -it weaviate-0 -- \
  curl -s http://localhost:8080/v1/schema | jq '.classes[] | {class: .class, properties: (.properties | length)}'

# Restore schema if missing
kubectl apply -f deployments/weaviate/telecom-schema.yaml
```

**c) Data Corruption:**
```bash
# Check data integrity
kubectl exec -n nephoran-system -it weaviate-0 -- \
  curl -s "http://localhost:8080/v1/objects?limit=10" | jq '.objects | length'

# If data is corrupted, restore from backup
kubectl exec -n nephoran-system -it weaviate-0 -- /backup-restore.sh restore
```

### Issue 4: Nephio Integration Failures

**Symptoms:**
- NetworkIntents processed but no packages created
- GitOps repository not updated
- Porch API errors

**Diagnostic Commands:**
```bash
# Check Nephio Bridge logs
kubectl logs -n nephoran-system -l app=nephio-bridge --tail=100

# Check Porch connectivity
kubectl get packagerepositories -A
kubectl get packagerevisions -A

# Test GitOps repository access
kubectl get secrets -n nephoran-system git-credentials -o yaml
kubectl logs -n nephoran-system -l app=nephio-bridge | grep -E "(git|auth|push|pull)"
```

**Solutions:**

**a) Git Credentials Issue:**
```bash
# Verify git credentials
kubectl get secret git-credentials -n nephoran-system -o jsonpath='{.data}' | jq 'keys'

# Test git access
kubectl run git-test --rm -i --tty --image=alpine/git -- git ls-remote https://<token>@github.com/your-org/nephio-packages.git

# Update credentials if needed
kubectl create secret generic git-credentials \
  --from-literal=token=<new-token> \
  --from-literal=username=<username> \
  -n nephoran-system --dry-run=client -o yaml | kubectl apply -f -
```

**b) Porch API Issues:**
```bash
# Check Porch server status
kubectl get pods -n porch-system -l app=porch-server

# Test Porch API directly
kubectl port-forward -n porch-system svc/porch-server 9443:9443 &
curl -k https://localhost:9443/api/porch/v1alpha1/packagerevisions | jq '.items | length'

# Check RBAC permissions
kubectl auth can-i create packagerevisions --as=system:serviceaccount:nephoran-system:nephio-bridge
```

## Log Analysis Procedures

### Structured Log Analysis

The Nephoran Intent Operator uses structured logging with JSON format. Key fields to analyze:

```bash
# Extract structured logs with key fields
kubectl logs -n nephoran-system -l app=nephoran-controller --since=1h | \
  jq -r 'select(.level == "ERROR" or .level == "WARN") | "\(.timestamp) [\(.level)] \(.component): \(.message)"'

# Analyze error patterns
kubectl logs -n nephoran-system -l app=nephoran-controller --since=1h | \
  jq -r 'select(.level == "ERROR") | .error' | sort | uniq -c | sort -nr

# Track specific NetworkIntent processing
kubectl logs -n nephoran-system -l app=nephoran-controller --since=1h | \
  jq -r 'select(.networkintent_name == "my-intent") | "\(.timestamp): \(.message)"'
```

### Log Correlation Across Components

```bash
# Create correlation script
cat > correlate-logs.sh << 'EOF'
#!/bin/bash
INTENT_NAME=$1
NAMESPACE=${2:-default}
TIME_RANGE=${3:-"1h"}

echo "=== Controller Logs ==="
kubectl logs -n nephoran-system -l app=nephoran-controller --since=$TIME_RANGE | \
  grep -i "$INTENT_NAME" | head -20

echo -e "\n=== LLM Processor Logs ==="
kubectl logs -n nephoran-system -l app=llm-processor --since=$TIME_RANGE | \
  grep -i "$INTENT_NAME" | head -10

echo -e "\n=== RAG API Logs ==="
kubectl logs -n nephoran-system -l app=rag-api --since=$TIME_RANGE | \
  grep -i "$INTENT_NAME" | head -10

echo -e "\n=== Nephio Bridge Logs ==="
kubectl logs -n nephoran-system -l app=nephio-bridge --since=$TIME_RANGE | \
  grep -i "$INTENT_NAME" | head -10
EOF

chmod +x correlate-logs.sh

# Usage
./correlate-logs.sh "my-failing-intent" "default" "30m"
```

### Error Pattern Recognition

```bash
# Common error patterns and their meanings
cat > analyze-errors.sh << 'EOF'
#!/bin/bash
COMPONENT=${1:-"all"}
TIME_RANGE=${2:-"1h"}

if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "controller" ]; then
  echo "=== Controller Error Analysis ==="
  kubectl logs -n nephoran-system -l app=nephoran-controller --since=$TIME_RANGE | \
    grep -E "(ERROR|FATAL)" | \
    sed -E 's/.*"error":"([^"]*)",?.*/\1/' | \
    sort | uniq -c | sort -nr | head -10
fi

if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "llm" ]; then
  echo -e "\n=== LLM Processor Error Analysis ==="
  kubectl logs -n nephoran-system -l app=llm-processor --since=$TIME_RANGE | \
    grep -E "(ERROR|timeout|failed)" | \
    awk '{print $NF}' | sort | uniq -c | sort -nr | head -10
fi

if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "rag" ]; then
  echo -e "\n=== RAG API Error Analysis ==="
  kubectl logs -n nephoran-system -l app=rag-api --since=$TIME_RANGE | \
    grep -E "(ERROR|exception)" | \
    cut -d' ' -f5- | sort | uniq -c | sort -nr | head -10
fi
EOF

chmod +x analyze-errors.sh

# Usage
./analyze-errors.sh "controller" "2h"
```

## Performance Troubleshooting

### Performance Metrics Collection

```bash
# Comprehensive performance check
cat > perf-check.sh << 'EOF'
#!/bin/bash

echo "=== CPU and Memory Usage ==="
kubectl top pods -n nephoran-system --sort-by=cpu

echo -e "\n=== NetworkIntent Processing Metrics ==="
kubectl port-forward -n nephoran-system svc/nephoran-controller 8081:8081 &
PF_PID=$!
sleep 2
curl -s http://localhost:8081/metrics | grep -E "(networkintent_processing_duration|networkintent_processing_errors|networkintent_active_count)"
kill $PF_PID

echo -e "\n=== LLM Performance Metrics ==="
kubectl port-forward -n nephoran-system svc/llm-processor 8080:8080 &
PF_PID=$!
sleep 2
curl -s http://localhost:8080/metrics | grep -E "(llm_request_duration|llm_cache_hit_rate|llm_requests_total)"
kill $PF_PID

echo -e "\n=== RAG Performance Metrics ==="
kubectl port-forward -n nephoran-system svc/rag-api 8082:8080 &
PF_PID=$!
sleep 2
curl -s http://localhost:8080/metrics | grep -E "(rag_query_duration|rag_retrieval_latency|weaviate_connection_pool)"
kill $PF_PID

echo -e "\n=== Resource Quotas ==="
kubectl describe resourcequota -n nephoran-system 2>/dev/null || echo "No resource quotas configured"

echo -e "\n=== Network Policies ==="
kubectl get networkpolicies -n nephoran-system -o wide
EOF

chmod +x perf-check.sh
./perf-check.sh
```

### Performance Bottleneck Identification

```bash
# Memory usage analysis
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=nephoran-controller -o jsonpath='{.items[0].metadata.name}') -- \
  sh -c 'cat /proc/meminfo | grep -E "(MemTotal|MemFree|MemAvailable)"'

# CPU usage analysis
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=nephoran-controller -o jsonpath='{.items[0].metadata.name}') -- \
  sh -c 'cat /proc/loadavg'

# Network connectivity test
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=rag-api -o jsonpath='{.items[0].metadata.name}') -- \
  sh -c 'time curl -s http://weaviate:8080/v1/meta > /dev/null'

# Database query performance
kubectl exec -n nephoran-system -it weaviate-0 -- \
  curl -s -w "Total time: %{time_total}s\n" "http://localhost:8080/v1/objects?limit=100" -o /dev/null
```

### Performance Optimization Steps

```bash
# 1. Scale up resource-intensive components
kubectl scale deployment nephoran-controller -n nephoran-system --replicas=2
kubectl scale deployment llm-processor -n nephoran-system --replicas=3

# 2. Increase resource limits
kubectl patch deployment nephoran-controller -n nephoran-system --type='merge' -p='{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "controller",
          "resources": {
            "limits": {"memory": "1Gi", "cpu": "1000m"},
            "requests": {"memory": "512Mi", "cpu": "500m"}
          }
        }]
      }
    }
  }
}'

# 3. Enable cache optimizations
kubectl patch configmap llm-processor-config -n nephoran-system --type='merge' -p='{"data":{
  "cache_enabled": "true",
  "cache_ttl": "3600",
  "cache_size": "1000"
}}'

# 4. Optimize garbage collection
kubectl patch configmap rag-api-config -n nephoran-system --type='merge' -p='{"data":{
  "GOGC": "100",
  "GOMEMLIMIT": "512MiB"
}}'
```

## Network Connectivity Issues

### Network Diagnostics

```bash
# Check all network policies
kubectl get networkpolicies -A -o wide

# Test pod-to-pod connectivity
kubectl run net-test --rm -i --tty --image=nicolaka/netshoot -- /bin/bash
# Inside the pod:
# nslookup nephoran-controller.nephoran-system.svc.cluster.local
# curl -v http://llm-processor.nephoran-system.svc.cluster.local:8080/health
# traceroute rag-api.nephoran-system.svc.cluster.local

# Check DNS resolution
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=nephoran-controller -o jsonpath='{.items[0].metadata.name}') -- \
  nslookup llm-processor.nephoran-system.svc.cluster.local

# Check service endpoints
kubectl get endpoints -n nephoran-system -o wide
```

### Service Discovery Issues

```bash
# Verify service configuration
kubectl get services -n nephoran-system -o yaml | grep -A 5 -B 5 "selector\|port"

# Check if pods match service selectors
for svc in nephoran-controller llm-processor rag-api weaviate; do
  echo "=== Service: $svc ==="
  kubectl get service $svc -n nephoran-system -o jsonpath='{.spec.selector}' | jq .
  echo "Matching pods:"
  kubectl get pods -n nephoran-system -l $(kubectl get service $svc -n nephoran-system -o jsonpath='{.spec.selector}' | jq -r 'to_entries[] | "\(.key)=\(.value)"' | paste -sd, -) -o wide
  echo
done

# Test service reachability
kubectl run service-test --rm -i --tty --image=curlimages/curl -- sh
# Inside the pod:
# curl -v http://nephoran-controller.nephoran-system.svc.cluster.local:8081/healthz
# curl -v http://llm-processor.nephoran-system.svc.cluster.local:8080/health
# curl -v http://rag-api.nephoran-system.svc.cluster.local:8080/health
```

### External Connectivity Issues

```bash
# Test external API connectivity (OpenAI)
kubectl run external-test --rm -i --tty --image=curlimages/curl -- \
  curl -v -H "Authorization: Bearer sk-test" https://api.openai.com/v1/models

# Check proxy/firewall settings
kubectl get configmap kube-proxy-config -n kube-system -o yaml | grep -A 10 proxy

# Test egress connectivity
kubectl run egress-test --rm -i --tty --image=busybox -- nslookup api.openai.com

# Check network policies for egress rules
kubectl get networkpolicies -n nephoran-system -o yaml | grep -A 10 -B 5 egress
```

## Debugging Tools and Commands

### Debug Pod Deployment

```bash
# Deploy debug pod with comprehensive tools
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: nephoran-debug
  namespace: nephoran-system
spec:
  containers:
  - name: debug
    image: nicolaka/netshoot:latest
    command: ['sleep', '3600']
    volumeMounts:
    - name: podlogs
      mountPath: /var/log/pods
      readOnly: true
  volumes:
  - name: podlogs
    hostPath:
      path: /var/log/pods
  hostNetwork: true
  restartPolicy: Never
EOF

# Use debug pod
kubectl exec -n nephoran-system -it nephoran-debug -- bash
```

### Advanced Debugging Commands

```bash
# 1. Memory dump analysis
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=nephoran-controller -o jsonpath='{.items[0].metadata.name}') -- \
  kill -USR1 1  # Trigger memory dump for Go processes

# 2. Network traffic analysis
kubectl exec -n nephoran-system -it nephoran-debug -- \
  tcpdump -i any -w /tmp/traffic.pcap host 10.244.0.0/16

# 3. Process analysis
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=nephoran-controller -o jsonpath='{.items[0].metadata.name}') -- \
  ps aux | head -20

# 4. File descriptor analysis
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=nephoran-controller -o jsonpath='{.items[0].metadata.name}') -- \
  ls -la /proc/1/fd | wc -l

# 5. Kubernetes API debugging
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | tail -20
kubectl api-resources | grep nephoran
```

### Custom Debug Scripts

```bash
# Create comprehensive debug script
cat > debug-nephoran.sh << 'EOF'
#!/bin/bash
set -e

NAMESPACE="nephoran-system"
OUTPUT_DIR="debug-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUTPUT_DIR"

echo "Collecting debug information..."

# Basic cluster info
kubectl cluster-info > "$OUTPUT_DIR/cluster-info.txt"
kubectl version > "$OUTPUT_DIR/version.txt"
kubectl get nodes -o wide > "$OUTPUT_DIR/nodes.txt"

# Nephoran components
kubectl get all -n "$NAMESPACE" -o wide > "$OUTPUT_DIR/components.txt"
kubectl get configmaps -n "$NAMESPACE" -o yaml > "$OUTPUT_DIR/configmaps.yaml"
kubectl get secrets -n "$NAMESPACE" -o yaml > "$OUTPUT_DIR/secrets.yaml" 2>/dev/null || echo "Unable to read secrets" > "$OUTPUT_DIR/secrets.yaml"

# Logs
for component in nephoran-controller llm-processor rag-api weaviate nephio-bridge; do
  kubectl logs -n "$NAMESPACE" -l app="$component" --tail=1000 > "$OUTPUT_DIR/${component}-logs.txt" 2>/dev/null || echo "No logs for $component"
done

# Events
kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' > "$OUTPUT_DIR/events.txt"

# NetworkIntents
kubectl get networkintents -A -o yaml > "$OUTPUT_DIR/networkintents.yaml"

# Custom resources
kubectl get e2nodesets -A -o yaml > "$OUTPUT_DIR/e2nodesets.yaml" 2>/dev/null || echo "No E2NodeSets found"

# Resource usage
kubectl top pods -n "$NAMESPACE" > "$OUTPUT_DIR/resource-usage.txt" 2>/dev/null || echo "Metrics not available"

# Network policies
kubectl get networkpolicies -n "$NAMESPACE" -o yaml > "$OUTPUT_DIR/network-policies.yaml"

echo "Debug information collected in: $OUTPUT_DIR"
echo "To share with support, run: tar -czf debug-bundle.tar.gz $OUTPUT_DIR"
EOF

chmod +x debug-nephoran.sh
```

## Known Issues and Workarounds

### Issue: Controller Memory Leak

**Description**: Controller memory usage continuously increases over time
**Affected Versions**: v1.0.0 - v1.0.2
**Workaround**:
```bash
# Monitor memory usage
kubectl top pods -n nephoran-system -l app=nephoran-controller

# Restart when memory usage > 80%
kubectl rollout restart deployment/nephoran-controller -n nephoran-system

# Apply memory limit
kubectl patch deployment nephoran-controller -n nephoran-system --type='merge' -p='{"spec":{"template":{"spec":{"containers":[{"name":"controller","resources":{"limits":{"memory":"512Mi"}}}]}}}}'
```

### Issue: Weaviate Startup Slow on Large Datasets

**Description**: Weaviate takes >5 minutes to start with large knowledge bases
**Affected Versions**: All versions with >10GB datasets
**Workaround**:
```bash
# Increase readiness probe initial delay
kubectl patch statefulset weaviate -n nephoran-system --type='merge' -p='{"spec":{"template":{"spec":{"containers":[{"name":"weaviate","readinessProbe":{"initialDelaySeconds":300}}]}}}}'

# Use faster storage class
kubectl patch statefulset weaviate -n nephoran-system --type='merge' -p='{"spec":{"volumeClaimTemplates":[{"metadata":{"name":"weaviate-data"},"spec":{"storageClassName":"fast-ssd"}}]}}'
```

### Issue: LLM API Rate Limits

**Description**: OpenAI API rate limiting causes failures during high load
**Affected Versions**: All versions
**Workaround**:
```bash
# Implement exponential backoff
kubectl patch configmap llm-processor-config -n nephoran-system --type='merge' -p='{"data":{
  "retry_max_attempts": "5",
  "retry_base_delay": "2s",
  "retry_max_delay": "30s"
}}'

# Enable request batching
kubectl patch configmap llm-processor-config -n nephoran-system --type='merge' -p='{"data":{
  "batch_enabled": "true",
  "batch_size": "5",
  "batch_timeout": "5s"
}}'
```

### Issue: NetworkPolicy Blocks Internal Communication

**Description**: Overly restrictive network policies prevent component communication
**Affected Versions**: Kubernetes 1.25+ with strict network policies
**Workaround**:
```bash
# Temporary: Disable network policies for troubleshooting
kubectl delete networkpolicy --all -n nephoran-system

# Permanent: Apply corrected network policies
kubectl apply -f deployments/security/corrected-network-policies.yaml
```

### Issue: Certificate Rotation Failures

**Description**: mTLS certificates expire causing service disruption
**Affected Versions**: All versions with mTLS enabled
**Workaround**:
```bash
# Check certificate expiration
kubectl get certificates -n nephoran-system -o custom-columns=NAME:.metadata.name,READY:.status.conditions[0].status,EXPIRY:.status.notAfter

# Manual certificate renewal
kubectl delete certificate nephoran-tls-cert -n nephoran-system
kubectl apply -f deployments/security/tls-certificates.yaml

# Automated renewal setup
kubectl apply -f deployments/security/cert-manager-auto-renewal.yaml
```

---

**Document Version**: 1.0  
**Last Updated**: 2025-01-07  
**Next Review**: 2025-02-07  
**Owner**: Nephoran Operations Team  
**Contributors**: DevOps Team, SRE Team