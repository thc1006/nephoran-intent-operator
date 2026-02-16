# Nephoran Intent Operator - Troubleshooting & Diagnostics Guide

## Overview

This comprehensive troubleshooting guide provides systematic diagnostic procedures, root cause analysis methods, and resolution steps for common and complex issues in the Nephoran Intent Operator production environment.

## Diagnostic Framework

### System Health Assessment Matrix

| Component | Health Check | Diagnostic Command | Expected Response |
|-----------|--------------|-------------------|-------------------|
| Controllers | Pod Status | `kubectl get pods -n nephoran-system` | All pods Running |
| CRDs | Resource Registration | `kubectl get crd \| grep nephoran` | 3 CRDs listed |
| LLM Processor | Health Endpoint | `curl http://llm-processor:8080/healthz` | HTTP 200 OK |
| RAG API | Health Endpoint | `curl http://rag-api:8080/health` | HTTP 200 OK |
| Weaviate | Database Status | `curl http://weaviate:8080/v1/.well-known/ready` | HTTP 200 OK |
| Prometheus | Metrics Collection | `curl http://prometheus:9090/-/healthy` | HTTP 200 OK |
| Grafana | Dashboard Access | `curl http://grafana:3000/api/health` | HTTP 200 OK |

### Diagnostic Data Collection

**System Diagnostic Script:**
```bash
#!/bin/bash
# comprehensive-diagnostics.sh - Complete system diagnostics

DIAGNOSTIC_ID="DIAG-$(date +%Y%m%d-%H%M%S)"
REPORT_DIR="/tmp/nephoran-diagnostics-$DIAGNOSTIC_ID"
mkdir -p "$REPORT_DIR"/{logs,configs,metrics,events,resources}

echo "🔍 Starting comprehensive diagnostics: $DIAGNOSTIC_ID"

# 1. Cluster and node information
kubectl cluster-info > "$REPORT_DIR/cluster-info.txt"
kubectl get nodes -o wide > "$REPORT_DIR/nodes.txt"
kubectl top nodes > "$REPORT_DIR/node-resources.txt"

# 2. Nephoran system resources
kubectl get all -n nephoran-system -o wide > "$REPORT_DIR/resources/all-resources.txt"
kubectl get pvc -n nephoran-system -o wide > "$REPORT_DIR/resources/persistent-volumes.txt"
kubectl get secrets -n nephoran-system > "$REPORT_DIR/resources/secrets.txt"
kubectl get configmaps -n nephoran-system > "$REPORT_DIR/resources/configmaps.txt"

# 3. Custom resources
kubectl get networkintents -A -o wide > "$REPORT_DIR/resources/networkintents.txt"
kubectl get e2nodesets -A -o wide > "$REPORT_DIR/resources/e2nodesets.txt"
kubectl get managedelements -A -o wide > "$REPORT_DIR/resources/managedelements.txt"

# 4. Pod logs
for pod in $(kubectl get pods -n nephoran-system -o jsonpath='{.items[*].metadata.name}'); do
  kubectl logs "$pod" -n nephoran-system --previous > "$REPORT_DIR/logs/$pod-previous.log" 2>/dev/null || true
  kubectl logs "$pod" -n nephoran-system > "$REPORT_DIR/logs/$pod-current.log" 2>/dev/null || true
done

# 5. Events
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' > "$REPORT_DIR/events/namespace-events.txt"
kubectl get events -A --sort-by='.lastTimestamp' | tail -100 > "$REPORT_DIR/events/cluster-events.txt"

# 6. Configuration dumps
for deployment in llm-processor rag-api nephio-bridge oran-adaptor weaviate; do
  kubectl get deployment "$deployment" -n nephoran-system -o yaml > "$REPORT_DIR/configs/$deployment-deployment.yaml" 2>/dev/null || true
done

# 7. Service and networking
kubectl get svc -n nephoran-system -o wide > "$REPORT_DIR/resources/services.txt"
kubectl get ingress -n nephoran-system -o wide > "$REPORT_DIR/resources/ingress.txt"
kubectl get networkpolicies -n nephoran-system -o wide > "$REPORT_DIR/resources/network-policies.txt"

# 8. HPA and scaling
kubectl get hpa -n nephoran-system -o wide > "$REPORT_DIR/resources/hpa.txt"
kubectl get scaledobjects -n nephoran-system -o wide > "$REPORT_DIR/resources/keda-scaledobjects.txt" 2>/dev/null || true

# 9. Health checks
echo "=== Health Check Results ===" > "$REPORT_DIR/health-checks.txt"
for service in llm-processor rag-api weaviate prometheus grafana; do
  echo "Checking $service..." >> "$REPORT_DIR/health-checks.txt"
  kubectl exec deployment/"$service" -n nephoran-system -- curl -f http://localhost:8080/health 2>&1 >> "$REPORT_DIR/health-checks.txt" || \
  kubectl exec deployment/"$service" -n nephoran-system -- curl -f http://localhost:8080/healthz 2>&1 >> "$REPORT_DIR/health-checks.txt" || \
  echo "Health check failed for $service" >> "$REPORT_DIR/health-checks.txt"
done

# 10. Performance metrics snapshot
curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=up" > "$REPORT_DIR/metrics/service-availability.json"
curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=rate(nephoran_networkintent_total[5m])" > "$REPORT_DIR/metrics/intent-processing-rate.json"
curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=rate(nephoran_errors_total[5m])" > "$REPORT_DIR/metrics/error-rates.json"

# 11. Generate summary report
cat > "$REPORT_DIR/diagnostic-summary.json" <<EOF
{
  "diagnostic_id": "$DIAGNOSTIC_ID",
  "timestamp": "$(date -Iseconds)",
  "cluster_info": {
    "nodes": $(kubectl get nodes --no-headers | wc -l),
    "nephoran_pods": $(kubectl get pods -n nephoran-system --no-headers | wc -l),
    "running_pods": $(kubectl get pods -n nephoran-system --no-headers | grep Running | wc -l),
    "failed_pods": $(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
  },
  "resource_counts": {
    "networkintents": $(kubectl get networkintents -A --no-headers | wc -l),
    "e2nodesets": $(kubectl get e2nodesets -A --no-headers | wc -l),
    "managedelements": $(kubectl get managedelements -A --no-headers | wc -l)
  }
}
EOF

# Create tarball
tar -czf "/tmp/nephoran-diagnostics-$DIAGNOSTIC_ID.tar.gz" -C /tmp "nephoran-diagnostics-$DIAGNOSTIC_ID"
rm -rf "$REPORT_DIR"

echo "✅ Diagnostics complete: /tmp/nephoran-diagnostics-$DIAGNOSTIC_ID.tar.gz"
```

## Common Issues and Resolutions

### 1. Intent Processing Issues

#### 1.1 Intents Stuck in Processing State

**Symptoms:**
- NetworkIntent resources remain in "Processing" state for extended periods
- No status updates or error messages
- Intent processing queue backing up

**Diagnostic Steps:**
```bash
# Check intent status
kubectl get networkintents -A -o wide
kubectl describe networkintent <intent-name> -n <namespace>

# Check LLM processor logs
kubectl logs deployment/llm-processor -n nephoran-system --tail=100

# Check RAG API connectivity
kubectl exec deployment/llm-processor -n nephoran-system -- \
  curl -v http://rag-api:8080/health

# Check OpenAI API connectivity
kubectl exec deployment/llm-processor -n nephoran-system -- \
  curl -v https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

**Root Cause Analysis:**
1. **OpenAI API Key Issues**: Invalid, expired, or rate-limited API key
2. **Network Connectivity**: RAG API or OpenAI API unreachable
3. **Resource Constraints**: LLM processor pods resource-starved
4. **Circuit Breaker Activation**: Too many failures causing circuit breaker to open

**Resolution Steps:**
```bash
# 1. Verify and update OpenAI API key
kubectl get secret openai-api-key -n nephoran-system -o yaml
kubectl create secret generic openai-api-key \
  --from-literal=api-key="$NEW_OPENAI_API_KEY" \
  --namespace=nephoran-system \
  --dry-run=client -o yaml | kubectl apply -f -

# 2. Restart LLM processor to pick up new credentials
kubectl rollout restart deployment/llm-processor -n nephoran-system

# 3. Check and adjust resource limits
kubectl get pods -n nephoran-system -o jsonpath='{.items[*].spec.containers[*].resources}'
kubectl set resources deployment/llm-processor \
  --requests=cpu=1000m,memory=2Gi \
  --limits=cpu=4000m,memory=8Gi \
  -n nephoran-system

# 4. Reset circuit breaker if activated
kubectl exec deployment/llm-processor -n nephoran-system -- \
  curl -X POST http://localhost:8080/circuit-breaker/reset

# 5. Clear and retry stuck intents
kubectl patch networkintent <intent-name> -n <namespace> \
  --type=merge -p='{"spec":{"forceRetry":true}}'
```

#### 1.2 High Intent Processing Latency

**Symptoms:**
- P95 latency > 5 seconds
- User complaints about slow processing
- Alert: `IntentProcessingLatencyHigh`

**Diagnostic Steps:**
```bash
# Check current latency metrics
curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=histogram_quantile(0.95,rate(nephoran_llm_request_duration_seconds_bucket[5m]))"

# Check processing queue depth
curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=nephoran_llm_processing_queue_depth"

# Check cache hit rates
curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=nephoran_rag_cache_hit_rate"

# Check resource utilization
kubectl top pods -n nephoran-system
```

**Root Cause Analysis:**
1. **Cache Performance**: Low cache hit rates causing repeated expensive operations
2. **Resource Bottlenecks**: CPU/memory constraints on processing pods
3. **Network Latency**: Slow external API calls (OpenAI, Weaviate)
4. **Queue Backup**: Too many concurrent requests overwhelming system

**Resolution Steps:**
```bash
# 1. Optimize cache performance
kubectl exec deployment/rag-api -n nephoran-system -- \
  curl -X POST http://localhost:8080/cache/warm

# 2. Scale up processing capacity
kubectl scale deployment llm-processor --replicas=5 -n nephoran-system
kubectl scale deployment rag-api --replicas=3 -n nephoran-system

# 3. Adjust HPA thresholds for faster scaling
kubectl patch hpa llm-processor -n nephoran-system --type merge \
  -p='{"spec":{"metrics":[{"type":"Resource","resource":{"name":"cpu","target":{"type":"Utilization","averageUtilization":60}}}]}}'

# 4. Optimize request routing
kubectl annotate service llm-processor -n nephoran-system \
  service.alpha.kubernetes.io/tolerate-unready-endpoints=true

# 5. Enable request prioritization
kubectl patch configmap llm-processor-config -n nephoran-system --type merge \
  -p='{"data":{"enable_priority_queue":"true","high_priority_ratio":"0.3"}}'
```

### 2. Vector Database (Weaviate) Issues

#### 2.1 Weaviate Pod Crash Loops

**Symptoms:**
- Weaviate pods constantly restarting
- CrashLoopBackOff status
- Vector search operations failing

**Diagnostic Steps:**
```bash
# Check pod status and events
kubectl get pods -l app=weaviate -n nephoran-system
kubectl describe pod -l app=weaviate -n nephoran-system

# Check logs for crash reasons
kubectl logs -l app=weaviate -n nephoran-system --previous

# Check resource usage and limits
kubectl top pods -l app=weaviate -n nephoran-system
kubectl get pods -l app=weaviate -n nephoran-system -o jsonpath='{.items[*].spec.containers[*].resources}'

# Check persistent volume status
kubectl get pvc -l app=weaviate -n nephoran-system
kubectl describe pvc -l app=weaviate -n nephoran-system
```

**Common Root Causes:**
1. **Memory Limits**: Weaviate exceeding memory limits during vector operations
2. **Storage Issues**: PV corruption or insufficient space
3. **Configuration Errors**: Invalid Weaviate configuration
4. **Index Corruption**: Corrupted vector indices causing startup failures

**Resolution Steps:**
```bash
# 1. Increase memory limits
kubectl patch deployment weaviate -n nephoran-system --type merge \
  -p='{"spec":{"template":{"spec":{"containers":[{"name":"weaviate","resources":{"limits":{"memory":"16Gi"},"requests":{"memory":"8Gi"}}}]}}}}'

# 2. Check and repair storage
kubectl exec -it deployment/weaviate -n nephoran-system -- df -h /var/lib/weaviate
kubectl exec -it deployment/weaviate -n nephoran-system -- \
  /bin/sh -c "cd /var/lib/weaviate && find . -type f -name '*.corrupt' | wc -l"

# 3. If corruption detected, restore from backup
kubectl scale deployment weaviate --replicas=0 -n nephoran-system
./scripts/ops/disaster-recovery-system.sh restore-weaviate

# 4. Restart with clean configuration
kubectl delete pod -l app=weaviate -n nephoran-system
kubectl wait --for=condition=ready pod -l app=weaviate -n nephoran-system --timeout=600s

# 5. Verify vector search functionality
kubectl exec deployment/weaviate -n nephoran-system -- \
  curl -X POST http://localhost:8080/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{Get{TelecomKnowledge(limit:1){title}}}"}'
```

#### 2.2 Vector Search Performance Issues

**Symptoms:**
- Slow vector search responses (>1 second)
- High memory usage during searches
- RAG API timeouts

**Diagnostic Steps:**
```bash
# Check vector search performance
kubectl exec deployment/weaviate -n nephoran-system -- \
  curl -X POST http://localhost:8080/v1/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{Get{TelecomKnowledge(nearText:{concepts:[\"AMF\"]},limit:10){title,_additional{certainty,distance}}}}"}'

# Check index statistics
kubectl exec deployment/weaviate -n nephoran-system -- \
  curl http://localhost:8080/v1/meta

# Monitor resource usage during search
kubectl top pods -l app=weaviate -n nephoran-system --watch
```

**Optimization Steps:**
```bash
# 1. Optimize HNSW parameters
kubectl exec deployment/weaviate -n nephoran-system -- \
  curl -X PUT http://localhost:8080/v1/schema/TelecomKnowledge \
  -H "Content-Type: application/json" \
  -d '{
    "vectorIndexConfig": {
      "distance": "cosine",
      "ef": 64,
      "efConstruction": 128,
      "maxConnections": 16
    }
  }'

# 2. Enable compression
kubectl exec deployment/weaviate -n nephoran-system -- \
  curl -X PUT http://localhost:8080/v1/schema/TelecomKnowledge \
  -H "Content-Type: application/json" \
  -d '{
    "vectorIndexConfig": {
      "pq": {
        "enabled": true,
        "segments": 256,
        "centroids": 256
      }
    }
  }'

# 3. Increase memory allocation
kubectl patch deployment weaviate -n nephoran-system --type merge \
  -p='{"spec":{"template":{"spec":{"containers":[{"name":"weaviate","env":[{"name":"LIMIT_RESOURCES","value":"false"},{"name":"GOMEMLIMIT","value":"12GiB"}]}]}}}}'

# 4. Optimize query patterns
kubectl patch configmap rag-api-config -n nephoran-system --type merge \
  -p='{"data":{"vector_search_limit":"5","enable_hybrid_search":"true","hybrid_alpha":"0.7"}}'
```

### 3. Auto-Scaling Issues

#### 3.1 HPA Not Scaling

**Symptoms:**
- High resource usage but no scaling
- HPA showing "unknown" metrics
- Pods remain at minimum replica count

**Diagnostic Steps:**
```bash
# Check HPA status
kubectl get hpa -n nephoran-system -o wide
kubectl describe hpa llm-processor -n nephoran-system

# Check metrics server
kubectl get pods -n kube-system | grep metrics-server
kubectl logs -l k8s-app=metrics-server -n kube-system

# Check custom metrics
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1" | jq .

# Check KEDA status
kubectl get scaledobjects -n nephoran-system
kubectl describe scaledobject -n nephoran-system
```

**Resolution Steps:**
```bash
# 1. Restart metrics server if issues detected
kubectl rollout restart deployment/metrics-server -n kube-system

# 2. Fix HPA configuration
kubectl patch hpa llm-processor -n nephoran-system --type merge \
  -p='{"spec":{"metrics":[{"type":"Resource","resource":{"name":"cpu","target":{"type":"Utilization","averageUtilization":70}}}]}}'

# 3. Verify service monitor for custom metrics
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: llm-processor-metrics
  namespace: nephoran-system
spec:
  selector:
    matchLabels:
      app: llm-processor
  endpoints:
  - port: http-metrics
    interval: 15s
    path: /metrics
EOF

# 4. Test manual scaling to verify functionality
kubectl scale deployment llm-processor --replicas=3 -n nephoran-system
kubectl wait --for=condition=available deployment/llm-processor -n nephoran-system

# 5. Reset HPA if needed
kubectl delete hpa llm-processor -n nephoran-system
kubectl apply -f deployments/kustomize/base/llm-processor/hpa.yaml
```

#### 3.2 KEDA ScaledObject Issues

**Symptoms:**
- KEDA not responding to external metrics
- ScaledObject in "Unknown" state
- No scaling despite high queue depths

**Diagnostic Steps:**
```bash
# Check KEDA operator status
kubectl get pods -n keda-system
kubectl logs -l app=keda-operator -n keda-system

# Check ScaledObject configuration
kubectl get scaledobjects -n nephoran-system -o yaml

# Test Prometheus connectivity from KEDA
kubectl exec -n keda-system deployment/keda-operator -- \
  curl -v "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=up"

# Check scaling trigger metrics
kubectl logs -l app=keda-metrics-apiserver -n keda-system
```

**Resolution Steps:**
```bash
# 1. Verify Prometheus query works
curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=nephoran_llm_processing_queue_depth"

# 2. Fix ScaledObject configuration
kubectl apply -f - <<EOF
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: llm-processor-scaler
  namespace: nephoran-system
spec:
  scaleTargetRef:
    name: llm-processor
  minReplicaCount: 2
  maxReplicaCount: 10
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus.nephoran-monitoring.svc.cluster.local:9090
      metricName: llm_queue_depth
      threshold: '10'
      query: sum(nephoran_llm_processing_queue_depth)
EOF

# 3. Restart KEDA if needed
kubectl rollout restart deployment/keda-operator -n keda-system
kubectl rollout restart deployment/keda-metrics-apiserver -n keda-system

# 4. Verify scaling behavior
kubectl patch deployment llm-processor -n nephoran-system --type merge \
  -p='{"metadata":{"annotations":{"test-load":"true"}}}'
```

### 4. Network and Connectivity Issues

#### 4.1 Service Discovery Problems

**Symptoms:**
- Services unable to reach each other
- DNS resolution failures
- Connection timeouts between components

**Diagnostic Steps:**
```bash
# Test DNS resolution
kubectl exec deployment/llm-processor -n nephoran-system -- \
  nslookup rag-api.nephoran-system.svc.cluster.local

# Check service endpoints
kubectl get svc -n nephoran-system -o wide
kubectl get endpoints -n nephoran-system

# Test connectivity between services
kubectl exec deployment/llm-processor -n nephoran-system -- \
  curl -v http://rag-api.nephoran-system.svc.cluster.local:8080/health

# Check network policies
kubectl get networkpolicies -n nephoran-system
kubectl describe networkpolicy -n nephoran-system
```

**Resolution Steps:**
```bash
# 1. Verify CoreDNS is working
kubectl get pods -n kube-system | grep coredns
kubectl logs -l k8s-app=kube-dns -n kube-system

# 2. Check and fix service configurations
kubectl get svc rag-api -n nephoran-system -o yaml
kubectl patch svc rag-api -n nephoran-system --type merge \
  -p='{"spec":{"ports":[{"port":8080,"targetPort":"http","protocol":"TCP"}]}}'

# 3. Update network policies if blocking traffic
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-nephoran-internal
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/part-of: nephoran-intent-operator
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
  - to: []
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
EOF

# 4. Restart affected pods
kubectl rollout restart deployment/llm-processor -n nephoran-system
kubectl rollout restart deployment/rag-api -n nephoran-system
```

#### 4.2 Istio Service Mesh Issues

**Symptoms:**
- Sidecar injection not working
- mTLS communication failures
- High latency through service mesh

**Diagnostic Steps:**
```bash
# Check Istio installation
kubectl get pods -n istio-system
istioctl version

# Verify sidecar injection
kubectl get pods -n nephoran-system -o jsonpath='{.items[*].spec.containers[*].name}' | grep istio-proxy

# Check mTLS configuration
istioctl authn tls-check llm-processor.nephoran-system.svc.cluster.local

# Analyze traffic flow
istioctl proxy-config cluster llm-processor-<pod-id> -n nephoran-system
```

**Resolution Steps:**
```bash
# 1. Enable sidecar injection
kubectl label namespace nephoran-system istio-injection=enabled

# 2. Restart deployments to inject sidecars
kubectl rollout restart deployment -n nephoran-system

# 3. Configure DestinationRules for mTLS
kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: nephoran-services
  namespace: nephoran-system
spec:
  host: "*.nephoran-system.svc.cluster.local"
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
EOF

# 4. Configure PeerAuthentication
kubectl apply -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: nephoran-system
spec:
  mtls:
    mode: STRICT
EOF
```

## Performance Troubleshooting

### 5.1 CPU and Memory Analysis

**Memory Leak Detection:**
```bash
#!/bin/bash
# memory-leak-analysis.sh
POD_NAME=$1
NAMESPACE=${2:-nephoran-system}

if [ -z "$POD_NAME" ]; then
  echo "Usage: $0 <pod-name> [namespace]"
  exit 1
fi

echo "Analyzing memory usage for $POD_NAME in $NAMESPACE"

# Collect memory metrics over time
for i in {1..10}; do
  MEMORY_USAGE=$(kubectl top pod "$POD_NAME" -n "$NAMESPACE" --no-headers | awk '{print $3}')
  echo "$(date): Memory usage: $MEMORY_USAGE"
  sleep 30
done

# Check for memory pressure events
kubectl describe pod "$POD_NAME" -n "$NAMESPACE" | grep -A 10 "Events:"

# Get heap dump if available
kubectl exec "$POD_NAME" -n "$NAMESPACE" -- \
  curl -s http://localhost:8080/debug/pprof/heap > "/tmp/$POD_NAME-heap.pprof"
```

**CPU Profiling:**
```bash
#!/bin/bash
# cpu-profiling.sh
POD_NAME=$1
NAMESPACE=${2:-nephoran-system}
DURATION=${3:-30}

echo "Profiling CPU usage for $POD_NAME for ${DURATION}s"

# Get CPU profile
kubectl exec "$POD_NAME" -n "$NAMESPACE" -- \
  curl -s "http://localhost:8080/debug/pprof/profile?seconds=$DURATION" > "/tmp/$POD_NAME-cpu.pprof"

# Analyze top CPU consumers
kubectl exec "$POD_NAME" -n "$NAMESPACE" -- \
  curl -s http://localhost:8080/debug/pprof/top
```

### 5.2 Database Performance Analysis

**Weaviate Query Optimization:**
```bash
#!/bin/bash
# weaviate-performance-analysis.sh

echo "Analyzing Weaviate performance..."

# Check index statistics
kubectl exec deployment/weaviate -n nephoran-system -- \
  curl -s http://localhost:8080/v1/meta | jq '.classes[] | {class: .class, objects: .objectsCount, shards: .shardsStatus}'

# Test query performance with different parameters
for limit in 5 10 20; do
  echo "Testing with limit=$limit"
  time kubectl exec deployment/weaviate -n nephoran-system -- \
    curl -s -X POST http://localhost:8080/v1/graphql \
    -H "Content-Type: application/json" \
    -d "{\"query\": \"{Get{TelecomKnowledge(nearText:{concepts:[\\\"AMF\\\"]},limit:$limit){title}}}\"}"
done

# Check for slow queries
kubectl logs deployment/weaviate -n nephoran-system | grep -i "slow\|timeout\|error" | tail -20
```

## Log Analysis and Debugging

### 6.1 Centralized Log Analysis

**Log Aggregation Script:**
```bash
#!/bin/bash
# log-analysis.sh - Comprehensive log analysis

COMPONENT=${1:-all}
TIME_RANGE=${2:-1h}
OUTPUT_DIR="/tmp/nephoran-logs-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUTPUT_DIR"

echo "Collecting logs for component: $COMPONENT, time range: $TIME_RANGE"

if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "llm-processor" ]; then
  kubectl logs deployment/llm-processor -n nephoran-system --since="$TIME_RANGE" > "$OUTPUT_DIR/llm-processor.log"
fi

if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "rag-api" ]; then
  kubectl logs deployment/rag-api -n nephoran-system --since="$TIME_RANGE" > "$OUTPUT_DIR/rag-api.log"
fi

if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "weaviate" ]; then
  kubectl logs deployment/weaviate -n nephoran-system --since="$TIME_RANGE" > "$OUTPUT_DIR/weaviate.log"
fi

if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "controllers" ]; then
  kubectl logs deployment/nephio-bridge -n nephoran-system --since="$TIME_RANGE" > "$OUTPUT_DIR/nephio-bridge.log"
  kubectl logs deployment/oran-adaptor -n nephoran-system --since="$TIME_RANGE" > "$OUTPUT_DIR/oran-adaptor.log"
fi

# Analyze error patterns
echo "=== Error Analysis ===" > "$OUTPUT_DIR/error-summary.txt"
grep -i "error\|exception\|failed\|timeout" "$OUTPUT_DIR"/*.log | \
  sort | uniq -c | sort -nr >> "$OUTPUT_DIR/error-summary.txt"

# Analyze performance patterns
echo "=== Performance Analysis ===" > "$OUTPUT_DIR/performance-summary.txt"
grep -i "duration\|latency\|took\|ms\|seconds" "$OUTPUT_DIR"/*.log | \
  head -50 >> "$OUTPUT_DIR/performance-summary.txt"

echo "Log analysis complete: $OUTPUT_DIR"
```

### 6.2 Structured Log Parsing

**Error Pattern Detection:**
```bash
#!/bin/bash
# error-pattern-detection.sh

echo "Detecting error patterns in Nephoran logs..."

# Common error patterns to look for
PATTERNS=(
  "OpenAI API.*error"
  "context deadline exceeded"
  "connection refused"
  "out of memory"
  "disk space"
  "authentication.*failed"
  "rate limit"
  "circuit breaker"
  "timeout"
)

for pattern in "${PATTERNS[@]}"; do
  echo "Checking pattern: $pattern"
  count=$(kubectl logs deployment/llm-processor -n nephoran-system --since=24h | grep -i "$pattern" | wc -l)
  if [ "$count" -gt 0 ]; then
    echo "  Found $count occurrences"
    kubectl logs deployment/llm-processor -n nephoran-system --since=24h | grep -i "$pattern" | tail -5
  fi
done
```

## Escalation Procedures

### 7.1 Escalation Matrix

| Severity | Response Time | Escalation Level | Contact |
|----------|--------------|------------------|---------|
| P1 - Critical | 0-15 minutes | L1: On-call Engineer | Immediate PagerDuty |
| P2 - High | 15-60 minutes | L2: Senior Engineer | Phone + Slack |
| P3 - Medium | 1-4 hours | L3: Team Lead | Slack + Email |
| P4 - Low | 4-24 hours | L4: Product Owner | Email |

### 7.2 Vendor Support Procedures

**When to Engage Vendor Support:**
1. **OpenAI API Issues**: Persistent rate limiting or API errors
2. **Kubernetes Platform Issues**: Cluster-level problems
3. **Cloud Provider Issues**: Infrastructure or storage problems
4. **Weaviate Issues**: Vector database corruption or performance

**Support Information Collection:**
```bash
#!/bin/bash
# vendor-support-info.sh

echo "Collecting information for vendor support..."

# OpenAI API issues
if [ "$1" = "openai" ]; then
  echo "OpenAI API Key (masked): $(kubectl get secret openai-api-key -n nephoran-system -o jsonpath='{.data.api-key}' | base64 -d | sed 's/./*/g')"
  echo "Recent API calls:"
  kubectl logs deployment/llm-processor -n nephoran-system --since=1h | grep "openai" | tail -10
fi

# Kubernetes issues
if [ "$1" = "kubernetes" ]; then
  kubectl version
  kubectl get nodes -o wide
  kubectl describe node | grep -A 10 "Conditions:"
fi

# Cloud provider issues
if [ "$1" = "cloud" ]; then
  kubectl get pv
  kubectl get storageclass
  kubectl get events -A | grep -i "volume\|storage\|node" | tail -20
fi
```

This comprehensive troubleshooting guide provides systematic approaches to diagnosing and resolving issues in the Nephoran Intent Operator production environment.