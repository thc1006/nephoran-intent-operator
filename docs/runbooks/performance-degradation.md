# Nephoran Intent Operator - Performance Degradation Response

## Table of Contents
1. [Overview](#overview)
2. [Performance Metrics to Monitor](#performance-metrics-to-monitor)
3. [Bottleneck Identification](#bottleneck-identification)
4. [Scaling Procedures](#scaling-procedures)
5. [Cache Optimization](#cache-optimization)
6. [Database Tuning](#database-tuning)
7. [Resource Optimization](#resource-optimization)
8. [Alert Thresholds](#alert-thresholds)
9. [Performance Testing](#performance-testing)

## Overview

This runbook provides comprehensive procedures for identifying, diagnosing, and resolving performance degradation in the Nephoran Intent Operator. Performance issues can range from increased latency to complete service slowdowns affecting user experience.

**Performance SLAs:**
- NetworkIntent Processing Time: P95 < 5 seconds
- API Response Time: P99 < 2 seconds
- Knowledge Retrieval: P95 < 500ms
- System Availability: > 99.9%
- Error Rate: < 0.1%

**Performance Degradation Severities:**
- **P1**: Processing time > 10x baseline, error rate > 5%
- **P2**: Processing time > 5x baseline, error rate > 2%
- **P3**: Processing time > 2x baseline, error rate > 1%
- **P4**: Processing time > 1.5x baseline, minor performance impact

## Performance Metrics to Monitor

### Key Performance Indicators (KPIs)

```bash
# Create performance monitoring script
cat > monitor-performance.sh << 'EOF'
#!/bin/bash
set -e

MONITORING_DURATION=${1:-"300"} # 5 minutes default
NAMESPACE="nephoran-system"
OUTPUT_DIR="/tmp/performance-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUTPUT_DIR"

echo "=== PERFORMANCE MONITORING STARTED ==="
echo "Duration: ${MONITORING_DURATION}s"
echo "Output: $OUTPUT_DIR"

# 1. NetworkIntent Processing Metrics
echo "=== NetworkIntent Processing Metrics ==="
kubectl port-forward -n "$NAMESPACE" svc/nephoran-controller 8081:8081 >/dev/null 2>&1 &
CONTROLLER_PF=$!
sleep 3

# Collect baseline metrics
curl -s http://localhost:8081/metrics | grep -E "(networkintent_processing_duration|networkintent_processing_errors|networkintent_active_count)" > "$OUTPUT_DIR/controller-metrics-start.txt"

echo "Monitoring NetworkIntent processing..."
cat > "$OUTPUT_DIR/intent-processing-analysis.txt" << HEADER
NetworkIntent Processing Performance Analysis
Start Time: $(date -u)
Duration: ${MONITORING_DURATION}s

Metrics Timeline:
HEADER

# Monitor for specified duration
for i in $(seq 0 30 $MONITORING_DURATION); do
  timestamp=$(date -u)
  processing_duration=$(curl -s http://localhost:8081/metrics | grep "networkintent_processing_duration_seconds" | awk '{print $2}' | tail -1)
  active_count=$(curl -s http://localhost:8081/metrics | grep "networkintent_active_count" | awk '{print $2}' | tail -1)
  error_count=$(curl -s http://localhost:8081/metrics | grep "networkintent_processing_errors_total" | awk '{print $2}' | tail -1)
  
  echo "$timestamp - Duration: ${processing_duration}s, Active: $active_count, Errors: $error_count" >> "$OUTPUT_DIR/intent-processing-analysis.txt"
  sleep 30
done

kill $CONTROLLER_PF 2>/dev/null

# 2. LLM Processing Metrics
echo "=== LLM Processing Metrics ==="
kubectl port-forward -n "$NAMESPACE" svc/llm-processor 8080:8080 >/dev/null 2>&1 &
LLM_PF=$!
sleep 3

# Collect LLM metrics
cat > "$OUTPUT_DIR/llm-performance-analysis.txt" << HEADER
LLM Processing Performance Analysis
Start Time: $(date -u)

Key Metrics:
HEADER

curl -s http://localhost:8080/metrics | grep -E "(llm_request_duration|llm_cache_hit_rate|llm_requests_total|circuit_breaker)" >> "$OUTPUT_DIR/llm-performance-analysis.txt"
curl -s http://localhost:8080/admin/circuit-breaker/status > "$OUTPUT_DIR/circuit-breaker-status.json" 2>/dev/null || echo "Circuit breaker status unavailable"

kill $LLM_PF 2>/dev/null

# 3. RAG System Metrics  
echo "=== RAG System Metrics ==="
kubectl port-forward -n "$NAMESPACE" svc/rag-api 8082:8080 >/dev/null 2>&1 &
RAG_PF=$!
sleep 3

cat > "$OUTPUT_DIR/rag-performance-analysis.txt" << HEADER
RAG System Performance Analysis
Start Time: $(date -u)

Query Performance:
HEADER

curl -s http://localhost:8082/metrics | grep -E "(rag_query_duration|rag_retrieval_latency|weaviate_connection_pool|cache_hit_rate)" >> "$OUTPUT_DIR/rag-performance-analysis.txt"

# Test RAG query performance
echo "Testing RAG query performance..." >> "$OUTPUT_DIR/rag-performance-analysis.txt"
for i in {1..5}; do
  start_time=$(date +%s%N)
  curl -s -X POST http://localhost:8082/v1/query \
    -H "Content-Type: application/json" \
    -d '{"query":"test performance query","top_k":5}' >/dev/null 2>&1
  end_time=$(date +%s%N)
  duration=$(( (end_time - start_time) / 1000000 ))
  echo "Query $i: ${duration}ms" >> "$OUTPUT_DIR/rag-performance-analysis.txt"
done

kill $RAG_PF 2>/dev/null

# 4. Weaviate Database Metrics
echo "=== Weaviate Database Metrics ==="
kubectl port-forward -n "$NAMESPACE" svc/weaviate 8080:8080 >/dev/null 2>&1 &
WEAVIATE_PF=$!
sleep 3

cat > "$OUTPUT_DIR/weaviate-performance-analysis.txt" << HEADER
Weaviate Database Performance Analysis
Start Time: $(date -u)

Database Status:
HEADER

# Get Weaviate status and performance metrics
curl -s http://localhost:8080/v1/meta >> "$OUTPUT_DIR/weaviate-performance-analysis.txt"
echo -e "\nSchema Information:" >> "$OUTPUT_DIR/weaviate-performance-analysis.txt"
curl -s http://localhost:8080/v1/schema | jq '.classes[] | {class: .class, objects: (.properties | length)}' >> "$OUTPUT_DIR/weaviate-performance-analysis.txt" 2>/dev/null

# Test query performance
echo -e "\nQuery Performance Tests:" >> "$OUTPUT_DIR/weaviate-performance-analysis.txt"
for i in {1..5}; do
  start_time=$(date +%s%N)
  curl -s "http://localhost:8080/v1/objects?limit=10" >/dev/null 2>&1
  end_time=$(date +%s%N)
  duration=$(( (end_time - start_time) / 1000000 ))
  echo "Object query $i: ${duration}ms" >> "$OUTPUT_DIR/weaviate-performance-analysis.txt"
done

kill $WEAVIATE_PF 2>/dev/null

# 5. Resource Usage Metrics
echo "=== Resource Usage Metrics ==="
kubectl top nodes > "$OUTPUT_DIR/node-resource-usage.txt" 2>/dev/null || echo "Node metrics unavailable"
kubectl top pods -n "$NAMESPACE" > "$OUTPUT_DIR/pod-resource-usage.txt" 2>/dev/null || echo "Pod metrics unavailable"

# Get detailed resource information
cat > "$OUTPUT_DIR/resource-analysis.txt" << HEADER
Resource Usage Analysis
Start Time: $(date -u)

Pod Resource Limits and Requests:
HEADER

for pod in $(kubectl get pods -n "$NAMESPACE" -o name); do
  echo "=== $pod ===" >> "$OUTPUT_DIR/resource-analysis.txt"
  kubectl get "$pod" -n "$NAMESPACE" -o json | jq '.spec.containers[] | {name: .name, resources: .resources}' >> "$OUTPUT_DIR/resource-analysis.txt" 2>/dev/null
done

# 6. Generate Performance Summary
echo "=== Generating Performance Summary ==="
cat > "$OUTPUT_DIR/performance-summary.txt" << SUMMARY
Performance Monitoring Summary
Generated: $(date -u)
Duration: ${MONITORING_DURATION}s
Namespace: $NAMESPACE

Files Generated:
- controller-metrics-start.txt: Controller baseline metrics
- intent-processing-analysis.txt: NetworkIntent processing timeline
- llm-performance-analysis.txt: LLM processing metrics
- rag-performance-analysis.txt: RAG system performance
- weaviate-performance-analysis.txt: Database performance
- node-resource-usage.txt: Node resource utilization
- pod-resource-usage.txt: Pod resource utilization
- resource-analysis.txt: Resource limits analysis

Key Findings:
$(grep -h "Duration:" "$OUTPUT_DIR/intent-processing-analysis.txt" 2>/dev/null | tail -1 || echo "Intent processing data unavailable")

Recommendations:
- Review metrics for performance degradation patterns
- Check resource utilization against limits
- Analyze error rates and circuit breaker status
- Monitor cache hit rates for optimization opportunities

Next Steps:
1. Analyze collected metrics for bottlenecks
2. Apply appropriate optimization strategies
3. Monitor improvements after changes
4. Schedule follow-up performance assessment
SUMMARY

echo "✅ Performance monitoring completed"
echo "Results saved in: $OUTPUT_DIR"
echo "Review performance-summary.txt for overview"
EOF

chmod +x monitor-performance.sh
```

### Automated Performance Alerts

```bash
# Performance alerting thresholds
cat > setup-performance-alerts.sh << 'EOF'
#!/bin/bash
set -e

NAMESPACE="nephoran-system"

echo "=== SETTING UP PERFORMANCE ALERTS ==="

# Create Prometheus alerting rules
cat > performance-alerts.yaml << YAML
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: nephoran-performance-alerts
  namespace: $NAMESPACE
spec:
  groups:
  - name: nephoran.performance
    rules:
    # NetworkIntent Processing Performance
    - alert: NetworkIntentProcessingSlow
      expr: histogram_quantile(0.95, rate(networkintent_processing_duration_seconds_bucket[5m])) > 5
      for: 2m
      labels:
        severity: warning
        component: controller
      annotations:
        summary: "NetworkIntent processing is slow"
        description: "P95 NetworkIntent processing time is {{ \$value }}s (threshold: 5s)"
        
    - alert: NetworkIntentProcessingCritical  
      expr: histogram_quantile(0.95, rate(networkintent_processing_duration_seconds_bucket[5m])) > 20
      for: 1m
      labels:
        severity: critical
        component: controller
      annotations:
        summary: "NetworkIntent processing critically slow"
        description: "P95 NetworkIntent processing time is {{ \$value }}s (threshold: 20s)"

    # LLM Processing Performance
    - alert: LLMProcessingSlow
      expr: histogram_quantile(0.95, rate(llm_request_duration_seconds_bucket[5m])) > 10
      for: 2m
      labels:
        severity: warning
        component: llm-processor
      annotations:
        summary: "LLM processing is slow"
        description: "P95 LLM request duration is {{ \$value }}s (threshold: 10s)"

    - alert: LLMCircuitBreakerOpen
      expr: circuit_breaker_state{service="llm-processor"} == 1
      for: 1m
      labels:
        severity: critical
        component: llm-processor
      annotations:
        summary: "LLM circuit breaker is open"
        description: "LLM processor circuit breaker has been open for 1 minute"

    # RAG System Performance
    - alert: RAGQuerySlow
      expr: histogram_quantile(0.95, rate(rag_query_duration_seconds_bucket[5m])) > 2
      for: 2m
      labels:
        severity: warning
        component: rag-api
      annotations:
        summary: "RAG queries are slow"
        description: "P95 RAG query duration is {{ \$value }}s (threshold: 2s)"

    - alert: RAGCacheHitRateLow
      expr: rag_cache_hit_rate < 0.5
      for: 5m
      labels:
        severity: warning
        component: rag-api
      annotations:
        summary: "RAG cache hit rate is low"
        description: "RAG cache hit rate is {{ \$value }} (threshold: 0.5)"

    # Weaviate Performance
    - alert: WeaviateQuerySlow
      expr: histogram_quantile(0.95, rate(weaviate_query_duration_seconds_bucket[5m])) > 1
      for: 2m
      labels:
        severity: warning
        component: weaviate
      annotations:
        summary: "Weaviate queries are slow"
        description: "P95 Weaviate query duration is {{ \$value }}s (threshold: 1s)"

    # Resource Usage Alerts
    - alert: HighCPUUsage
      expr: rate(container_cpu_usage_seconds_total{namespace="$NAMESPACE"}[5m]) > 0.8
      for: 2m
      labels:
        severity: warning
        component: "{{ \$labels.pod }}"
      annotations:
        summary: "High CPU usage detected"
        description: "Pod {{ \$labels.pod }} CPU usage is {{ \$value }} (threshold: 0.8)"

    - alert: HighMemoryUsage
      expr: container_memory_working_set_bytes{namespace="$NAMESPACE"} / container_spec_memory_limit_bytes > 0.8
      for: 2m
      labels:
        severity: warning
        component: "{{ \$labels.pod }}"
      annotations:
        summary: "High memory usage detected"
        description: "Pod {{ \$labels.pod }} memory usage is {{ \$value }}% (threshold: 80%)"

    # Error Rate Alerts
    - alert: HighErrorRate
      expr: rate(networkintent_processing_errors_total[5m]) / rate(networkintent_processing_total[5m]) > 0.05
      for: 1m
      labels:
        severity: critical
        component: controller
      annotations:
        summary: "High NetworkIntent error rate"
        description: "NetworkIntent error rate is {{ \$value }}% (threshold: 5%)"
YAML

echo "Applying performance alerting rules..."
kubectl apply -f performance-alerts.yaml -n "$NAMESPACE"

# Create Grafana dashboard for performance monitoring
cat > performance-dashboard.json << DASHBOARD
{
  "dashboard": {
    "id": null,
    "title": "Nephoran Performance Monitoring",
    "tags": ["nephoran", "performance"],
    "timezone": "UTC",
    "panels": [
      {
        "id": 1,
        "title": "NetworkIntent Processing Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(networkintent_processing_duration_seconds_bucket[5m]))",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.95, rate(networkintent_processing_duration_seconds_bucket[5m]))",
            "legendFormat": "P95"
          },
          {
            "expr": "histogram_quantile(0.99, rate(networkintent_processing_duration_seconds_bucket[5m]))",
            "legendFormat": "P99"
          }
        ],
        "yAxes": [
          {
            "label": "Seconds",
            "min": 0
          }
        ],
        "alert": {
          "conditions": [
            {
              "query": {
                "queryType": "",
                "refId": "A"
              },
              "reducer": {
                "type": "last",
                "params": []
              },
              "evaluator": {
                "params": [5],
                "type": "gt"
              }
            }
          ],
          "executionErrorState": "alerting",
          "noDataState": "no_data",
          "frequency": "30s",
          "handler": 1,
          "name": "NetworkIntent Processing Alert",
          "message": "NetworkIntent processing time exceeded threshold"
        }
      },
      {
        "id": 2,
        "title": "LLM Request Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(llm_request_duration_seconds_bucket[5m]))",
            "legendFormat": "Request Duration P95"
          },
          {
            "expr": "rate(llm_requests_total[5m])",
            "legendFormat": "Requests/sec"
          }
        ]
      },
      {
        "id": 3,
        "title": "RAG Query Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(rag_query_duration_seconds_bucket[5m]))",
            "legendFormat": "Query Duration P95"
          },
          {
            "expr": "rag_cache_hit_rate",
            "legendFormat": "Cache Hit Rate"
          }
        ]
      },
      {
        "id": 4,
        "title": "Resource Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(container_cpu_usage_seconds_total{namespace=\"$NAMESPACE\"}[5m])",
            "legendFormat": "CPU - {{ pod }}"
          },
          {
            "expr": "container_memory_working_set_bytes{namespace=\"$NAMESPACE\"} / 1024 / 1024",
            "legendFormat": "Memory MB - {{ pod }}"
          }
        ]
      }
    ],
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "refresh": "30s"
  }
}
DASHBOARD

echo "✅ Performance alerts configured"
echo "Files created:"
echo "- performance-alerts.yaml: Prometheus alerting rules"
echo "- performance-dashboard.json: Grafana dashboard configuration"
echo ""
echo "Next steps:"
echo "1. Import performance-dashboard.json into Grafana"
echo "2. Configure notification channels in Grafana"
echo "3. Test alerts by triggering performance degradation"
EOF

chmod +x setup-performance-alerts.sh
```

## Bottleneck Identification

### Performance Bottleneck Analysis

```bash
# Comprehensive bottleneck analysis
cat > analyze-bottlenecks.sh << 'EOF'
#!/bin/bash
set -e

ANALYSIS_DURATION=${1:-"300"}
NAMESPACE="nephoran-system"
OUTPUT_DIR="/tmp/bottleneck-analysis-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUTPUT_DIR"

echo "=== BOTTLENECK ANALYSIS STARTED ==="
echo "Duration: ${ANALYSIS_DURATION}s"
echo "Output: $OUTPUT_DIR"

# 1. CPU Bottleneck Analysis
echo "=== CPU Bottleneck Analysis ==="
cat > "$OUTPUT_DIR/cpu-analysis.txt" << HEADER
CPU Bottleneck Analysis
Start Time: $(date -u)

Top CPU Consuming Pods:
HEADER

kubectl top pods -n "$NAMESPACE" --sort-by=cpu >> "$OUTPUT_DIR/cpu-analysis.txt" 2>/dev/null || echo "Metrics server unavailable" >> "$OUTPUT_DIR/cpu-analysis.txt"

# Get CPU limits and requests
echo -e "\nCPU Limits and Requests:" >> "$OUTPUT_DIR/cpu-analysis.txt"
kubectl get pods -n "$NAMESPACE" -o json | jq -r '
  .items[] | 
  "\(.metadata.name): Request=\(.spec.containers[0].resources.requests.cpu // "none") Limit=\(.spec.containers[0].resources.limits.cpu // "none")"
' >> "$OUTPUT_DIR/cpu-analysis.txt"

# 2. Memory Bottleneck Analysis  
echo "=== Memory Bottleneck Analysis ==="
cat > "$OUTPUT_DIR/memory-analysis.txt" << HEADER
Memory Bottleneck Analysis
Start Time: $(date -u)

Top Memory Consuming Pods:
HEADER

kubectl top pods -n "$NAMESPACE" --sort-by=memory >> "$OUTPUT_DIR/memory-analysis.txt" 2>/dev/null || echo "Metrics server unavailable" >> "$OUTPUT_DIR/memory-analysis.txt"

# Check for memory pressure
echo -e "\nMemory Limits and Requests:" >> "$OUTPUT_DIR/memory-analysis.txt"
kubectl get pods -n "$NAMESPACE" -o json | jq -r '
  .items[] | 
  "\(.metadata.name): Request=\(.spec.containers[0].resources.requests.memory // "none") Limit=\(.spec.containers[0].resources.limits.memory // "none")"
' >> "$OUTPUT_DIR/memory-analysis.txt"

# Check for OOMKilled events
echo -e "\nOOM Events:" >> "$OUTPUT_DIR/memory-analysis.txt"
kubectl get events -n "$NAMESPACE" --field-selector reason=OOMKilling >> "$OUTPUT_DIR/memory-analysis.txt" 2>/dev/null || echo "No OOM events found" >> "$OUTPUT_DIR/memory-analysis.txt"

# 3. Network Bottleneck Analysis
echo "=== Network Bottleneck Analysis ==="
cat > "$OUTPUT_DIR/network-analysis.txt" << HEADER
Network Bottleneck Analysis
Start Time: $(date -u)

Service Endpoints:
HEADER

kubectl get endpoints -n "$NAMESPACE" >> "$OUTPUT_DIR/network-analysis.txt"

# Test inter-service connectivity and latency
echo -e "\nInter-service Latency Tests:" >> "$OUTPUT_DIR/network-analysis.txt"
kubectl run network-test-$(date +%s) --rm -i --image=nicolaka/netshoot --restart=Never -- sh -c "
  echo 'Controller to LLM Processor:'
  time curl -s http://llm-processor.${NAMESPACE}.svc.cluster.local:8080/health > /dev/null 2>&1 || echo 'Connection failed'
  
  echo 'LLM Processor to RAG API:'
  time curl -s http://rag-api.${NAMESPACE}.svc.cluster.local:8080/health > /dev/null 2>&1 || echo 'Connection failed'
  
  echo 'RAG API to Weaviate:'
  time curl -s http://weaviate.${NAMESPACE}.svc.cluster.local:8080/v1/meta > /dev/null 2>&1 || echo 'Connection failed'
" >> "$OUTPUT_DIR/network-analysis.txt" 2>/dev/null

# 4. Database Bottleneck Analysis (Weaviate)
echo "=== Database Bottleneck Analysis ==="
kubectl port-forward -n "$NAMESPACE" svc/weaviate 8080:8080 >/dev/null 2>&1 &
WEAVIATE_PF=$!
sleep 5

cat > "$OUTPUT_DIR/database-analysis.txt" << HEADER
Database Bottleneck Analysis (Weaviate)
Start Time: $(date -u)

Database Status:
HEADER

# Get database status and configuration
curl -s http://localhost:8080/v1/meta >> "$OUTPUT_DIR/database-analysis.txt" 2>/dev/null || echo "Database unreachable" >> "$OUTPUT_DIR/database-analysis.txt"

# Test query performance with different loads
echo -e "\nQuery Performance Under Load:" >> "$OUTPUT_DIR/database-analysis.txt"
for concurrent in 1 5 10; do
  echo "Testing with $concurrent concurrent queries..." >> "$OUTPUT_DIR/database-analysis.txt"
  
  # Run concurrent queries
  start_time=$(date +%s%N)
  for i in $(seq 1 $concurrent); do
    curl -s "http://localhost:8080/v1/objects?limit=10" >/dev/null 2>&1 &
  done
  wait
  end_time=$(date +%s%N)
  
  duration=$(( (end_time - start_time) / 1000000 ))
  echo "Concurrent $concurrent: ${duration}ms total" >> "$OUTPUT_DIR/database-analysis.txt"
done

kill $WEAVIATE_PF 2>/dev/null

# 5. LLM API Bottleneck Analysis
echo "=== LLM API Bottleneck Analysis ==="
kubectl port-forward -n "$NAMESPACE" svc/llm-processor 8080:8080 >/dev/null 2>&1 &
LLM_PF=$!
sleep 5

cat > "$OUTPUT_DIR/llm-analysis.txt" << HEADER
LLM API Bottleneck Analysis
Start Time: $(date -u)

Circuit Breaker Status:
HEADER

curl -s http://localhost:8080/admin/circuit-breaker/status >> "$OUTPUT_DIR/llm-analysis.txt" 2>/dev/null

echo -e "\nCache Performance:" >> "$OUTPUT_DIR/llm-analysis.txt"
curl -s http://localhost:8080/metrics | grep cache >> "$OUTPUT_DIR/llm-analysis.txt" 2>/dev/null

# Test LLM processing with different request sizes
echo -e "\nRequest Size Impact:" >> "$OUTPUT_DIR/llm-analysis.txt"
for size in "small test" "medium length test request with more detail" "large comprehensive test request with extensive detail and multiple complex requirements for thorough testing"; do
  start_time=$(date +%s%N)
  curl -s -X POST http://localhost:8080/v1/process \
    -H "Content-Type: application/json" \
    -d "{\"intent\":\"$size\",\"priority\":\"low\"}" >/dev/null 2>&1 || echo "Request failed"
  end_time=$(date +%s%N)
  
  duration=$(( (end_time - start_time) / 1000000 ))
  echo "Size ${#size} chars: ${duration}ms" >> "$OUTPUT_DIR/llm-analysis.txt"
done

kill $LLM_PF 2>/dev/null

# 6. Storage I/O Analysis
echo "=== Storage I/O Analysis ==="
cat > "$OUTPUT_DIR/storage-analysis.txt" << HEADER
Storage I/O Bottleneck Analysis
Start Time: $(date -u)

Persistent Volumes:
HEADER

kubectl get pv,pvc -n "$NAMESPACE" >> "$OUTPUT_DIR/storage-analysis.txt" 2>/dev/null

# Check storage class performance characteristics
echo -e "\nStorage Classes:" >> "$OUTPUT_DIR/storage-analysis.txt"
kubectl get storageclass >> "$OUTPUT_DIR/storage-analysis.txt"

# Test storage performance on Weaviate pod
echo -e "\nStorage Performance Test (Weaviate):" >> "$OUTPUT_DIR/storage-analysis.txt"
kubectl exec -n "$NAMESPACE" weaviate-0 -- sh -c "
  echo 'Write test:'
  dd if=/dev/zero of=/tmp/test_write bs=1M count=100 2>&1 | grep -E 'MB/s|bytes'
  echo 'Read test:'
  dd if=/tmp/test_write of=/dev/null bs=1M 2>&1 | grep -E 'MB/s|bytes'
  rm -f /tmp/test_write
" >> "$OUTPUT_DIR/storage-analysis.txt" 2>/dev/null || echo "Storage test failed" >> "$OUTPUT_DIR/storage-analysis.txt"

# 7. Generate Bottleneck Summary and Recommendations
echo "=== Generating Bottleneck Analysis Summary ==="
cat > "$OUTPUT_DIR/bottleneck-summary.txt" << SUMMARY
Bottleneck Analysis Summary
Generated: $(date -u)
Analysis Duration: ${ANALYSIS_DURATION}s
Namespace: $NAMESPACE

ANALYSIS FILES:
- cpu-analysis.txt: CPU utilization and limits
- memory-analysis.txt: Memory usage and OOM events
- network-analysis.txt: Network connectivity and latency
- database-analysis.txt: Weaviate performance analysis
- llm-analysis.txt: LLM processor performance
- storage-analysis.txt: Storage I/O performance

IDENTIFIED BOTTLENECKS:
$(
# Check for CPU bottlenecks
if grep -q "80%" "$OUTPUT_DIR/cpu-analysis.txt" 2>/dev/null; then
  echo "🔴 CPU BOTTLENECK: High CPU usage detected"
fi

# Check for memory bottlenecks  
if grep -q "OOMKilling" "$OUTPUT_DIR/memory-analysis.txt" 2>/dev/null; then
  echo "🔴 MEMORY BOTTLENECK: Out of memory events detected"
fi

# Check for network bottlenecks
if grep -q "Connection failed" "$OUTPUT_DIR/network-analysis.txt" 2>/dev/null; then
  echo "🔴 NETWORK BOTTLENECK: Inter-service connectivity issues"
fi

# Check for database bottlenecks
if grep -q "unreachable" "$OUTPUT_DIR/database-analysis.txt" 2>/dev/null; then
  echo "🔴 DATABASE BOTTLENECK: Weaviate connectivity issues"
fi

# Default message if no major bottlenecks found
echo "ℹ️  Review individual analysis files for detailed findings"
)

RECOMMENDATIONS:
$(
echo "📊 IMMEDIATE ACTIONS:"
echo "1. Review resource utilization against limits"
echo "2. Check for error patterns in component logs"
echo "3. Analyze inter-service communication latency"
echo "4. Verify cache hit rates and optimize if needed"
echo ""
echo "🔧 OPTIMIZATION STRATEGIES:"
echo "1. Scale up bottlenecked components"
echo "2. Increase resource limits where needed"
echo "3. Optimize cache configurations"
echo "4. Consider horizontal pod autoscaling"
echo ""
echo "📈 MONITORING:"
echo "1. Set up alerts for identified bottleneck patterns"
echo "2. Implement continuous performance monitoring"
echo "3. Schedule regular bottleneck analysis"
)

NEXT STEPS:
1. Apply immediate optimizations for critical bottlenecks
2. Implement scaling strategies for performance improvement
3. Monitor performance improvements after changes
4. Schedule follow-up analysis in 1 week

Generated by: $(whoami)@$(hostname)
SUMMARY

echo "✅ Bottleneck analysis completed"
echo "Results saved in: $OUTPUT_DIR"
echo "Review bottleneck-summary.txt for overview and recommendations"
EOF

chmod +x analyze-bottlenecks.sh
```

## Scaling Procedures

### Horizontal Pod Autoscaling (HPA)

```bash
# Setup and manage HPA for performance scaling
cat > setup-autoscaling.sh << 'EOF'
#!/bin/bash
set -e

NAMESPACE="nephoran-system"
ACTION=${1:-"create"} # create, update, status, delete

echo "=== AUTOSCALING MANAGEMENT ==="
echo "Action: $ACTION"
echo "Namespace: $NAMESPACE"

case "$ACTION" in
  "create")
    echo "Creating HPA configurations..."
    
    # Controller HPA
    cat > controller-hpa.yaml << YAML
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: nephoran-controller-hpa
  namespace: $NAMESPACE
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nephoran-controller
  minReplicas: 2
  maxReplicas: 10
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
  - type: Pods
    pods:
      metric:
        name: networkintent_active_count
      target:
        type: AverageValue
        averageValue: "5"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
YAML

    # LLM Processor HPA
    cat > llm-processor-hpa.yaml << YAML
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-processor-hpa
  namespace: $NAMESPACE
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
  minReplicas: 3
  maxReplicas: 15
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 60
  - type: Pods
    pods:
      metric:
        name: llm_request_queue_size
      target:
        type: AverageValue
        averageValue: "10"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 30
      policies:
      - type: Percent
        value: 200
        periodSeconds: 30
    scaleDown:
      stabilizationWindowSeconds: 600
      policies:
      - type: Percent
        value: 5
        periodSeconds: 60
YAML

    # RAG API HPA
    cat > rag-api-hpa.yaml << YAML
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: rag-api-hpa
  namespace: $NAMESPACE
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: rag-api
  minReplicas: 2
  maxReplicas: 8
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Pods
    pods:
      metric:
        name: rag_query_queue_size
      target:
        type: AverageValue
        averageValue: "5"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 45
      policies:
      - type: Pods
        value: 2
        periodSeconds: 45
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Pods
        value: 1
        periodSeconds: 60
YAML

    # Apply HPA configurations
    kubectl apply -f controller-hpa.yaml
    kubectl apply -f llm-processor-hpa.yaml
    kubectl apply -f rag-api-hpa.yaml
    
    echo "✅ HPA configurations created"
    ;;
    
  "update")
    echo "Updating HPA configurations..."
    
    # Update CPU thresholds based on current load
    current_cpu_controller=$(kubectl get hpa nephoran-controller-hpa -n "$NAMESPACE" -o jsonpath='{.spec.metrics[0].resource.target.averageUtilization}')
    new_cpu_controller=$((current_cpu_controller - 10))
    
    kubectl patch hpa nephoran-controller-hpa -n "$NAMESPACE" --type='merge' -p="{
      \"spec\": {
        \"metrics\": [{
          \"type\": \"Resource\",
          \"resource\": {
            \"name\": \"cpu\",
            \"target\": {
              \"type\": \"Utilization\",
              \"averageUtilization\": $new_cpu_controller
            }
          }
        }]
      }
    }"
    
    echo "✅ HPA configurations updated"
    ;;
    
  "status")
    echo "Current HPA status:"
    kubectl get hpa -n "$NAMESPACE" -o wide
    
    echo -e "\nDetailed HPA information:"
    for hpa in $(kubectl get hpa -n "$NAMESPACE" -o name); do
      echo "=== $hpa ==="
      kubectl describe "$hpa" -n "$NAMESPACE" | grep -A 10 -B 5 "Metrics\|Events"
    done
    ;;
    
  "delete")
    echo "Deleting HPA configurations..."
    kubectl delete hpa --all -n "$NAMESPACE"
    echo "✅ HPA configurations deleted"
    ;;
    
  *)
    echo "❌ Unknown action: $ACTION"
    echo "Supported actions: create, update, status, delete"
    exit 1
    ;;
esac

echo "Current scaling status:"
kubectl get deployments -n "$NAMESPACE" -o custom-columns=NAME:.metadata.name,DESIRED:.spec.replicas,CURRENT:.status.replicas,READY:.status.readyReplicas
EOF

chmod +x setup-autoscaling.sh
```

### Manual Scaling Procedures

```bash
# Manual scaling for immediate performance improvement
cat > manual-scaling.sh << 'EOF'
#!/bin/bash
set -e

COMPONENT=$1
ACTION=${2:-"scale-up"} # scale-up, scale-down, emergency-scale
REPLICAS=${3:-""}
NAMESPACE="nephoran-system"

echo "=== MANUAL SCALING PROCEDURE ==="
echo "Component: $COMPONENT"
echo "Action: $ACTION"
echo "Namespace: $NAMESPACE"

if [ -z "$COMPONENT" ]; then
  echo "❌ Component name required"
  echo "Usage: $0 <component> [scale-up|scale-down|emergency-scale] [replicas]"
  echo "Components: controller, llm-processor, rag-api, weaviate, nephio-bridge"
  exit 1
fi

# Get current replica count
current_replicas=$(kubectl get deployment "$COMPONENT" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || \
                  kubectl get statefulset "$COMPONENT" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || \
                  echo "0")

if [ "$current_replicas" = "0" ]; then
  echo "❌ Component $COMPONENT not found or not scalable"
  exit 1
fi

echo "Current replicas: $current_replicas"

case "$ACTION" in
  "scale-up")
    if [ -n "$REPLICAS" ]; then
      new_replicas="$REPLICAS"
    else
      # Default scale-up strategy: double the current replicas (max 10)
      new_replicas=$((current_replicas * 2))
      if [ $new_replicas -gt 10 ]; then
        new_replicas=10
      fi
    fi
    
    echo "Scaling up $COMPONENT from $current_replicas to $new_replicas replicas"
    
    # Apply scaling
    if kubectl get deployment "$COMPONENT" -n "$NAMESPACE" >/dev/null 2>&1; then
      kubectl scale deployment "$COMPONENT" --replicas="$new_replicas" -n "$NAMESPACE"
      kubectl wait --for=condition=available --timeout=300s deployment/"$COMPONENT" -n "$NAMESPACE"
    elif kubectl get statefulset "$COMPONENT" -n "$NAMESPACE" >/dev/null 2>&1; then
      kubectl scale statefulset "$COMPONENT" --replicas="$new_replicas" -n "$NAMESPACE"
      kubectl wait --for=jsonpath='{.status.readyReplicas}'="$new_replicas" --timeout=300s statefulset/"$COMPONENT" -n "$NAMESPACE"
    fi
    
    echo "✅ Scaled up $COMPONENT to $new_replicas replicas"
    ;;
    
  "scale-down")
    if [ -n "$REPLICAS" ]; then
      new_replicas="$REPLICAS"
    else
      # Default scale-down strategy: reduce by half (min 1)
      new_replicas=$((current_replicas / 2))
      if [ $new_replicas -lt 1 ]; then
        new_replicas=1
      fi
    fi
    
    echo "Scaling down $COMPONENT from $current_replicas to $new_replicas replicas"
    
    # Apply scaling
    if kubectl get deployment "$COMPONENT" -n "$NAMESPACE" >/dev/null 2>&1; then
      kubectl scale deployment "$COMPONENT" --replicas="$new_replicas" -n "$NAMESPACE"
    elif kubectl get statefulset "$COMPONENT" -n "$NAMESPACE" >/dev/null 2>&1; then
      kubectl scale statefulset "$COMPONENT" --replicas="$new_replicas" -n "$NAMESPACE"
    fi
    
    echo "✅ Scaled down $COMPONENT to $new_replicas replicas"
    ;;
    
  "emergency-scale")
    echo "🚨 EMERGENCY SCALING for $COMPONENT"
    
    # Emergency scaling strategy based on component
    case "$COMPONENT" in
      "llm-processor")
        emergency_replicas=10
        ;;
      "rag-api")
        emergency_replicas=6
        ;;
      "nephoran-controller")
        emergency_replicas=5
        ;;
      *)
        emergency_replicas=3
        ;;
    esac
    
    echo "Emergency scaling $COMPONENT to $emergency_replicas replicas"
    
    # Apply emergency scaling
    if kubectl get deployment "$COMPONENT" -n "$NAMESPACE" >/dev/null 2>&1; then
      kubectl scale deployment "$COMPONENT" --replicas="$emergency_replicas" -n "$NAMESPACE"
      
      # Wait for at least half the replicas to be ready
      min_ready=$((emergency_replicas / 2))
      timeout=300
      elapsed=0
      
      while [ $elapsed -lt $timeout ]; do
        ready_replicas=$(kubectl get deployment "$COMPONENT" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        if [ "$ready_replicas" -ge "$min_ready" ]; then
          break
        fi
        sleep 5
        elapsed=$((elapsed + 5))
      done
      
      echo "✅ Emergency scaling completed: $ready_replicas/$emergency_replicas replicas ready"
      
    elif kubectl get statefulset "$COMPONENT" -n "$NAMESPACE" >/dev/null 2>&1; then
      kubectl scale statefulset "$COMPONENT" --replicas="$emergency_replicas" -n "$NAMESPACE"
      echo "✅ Emergency scaling applied to StatefulSet $COMPONENT"
    fi
    
    # Create emergency scaling alert
    cat > "emergency-scaling-alert-$(date +%s).md" << ALERT
# Emergency Scaling Alert

**Component**: $COMPONENT
**Time**: $(date -u)
**Action**: Emergency scaled to $emergency_replicas replicas
**Previous**: $current_replicas replicas

## Reason
Performance degradation requiring immediate scaling intervention.

## Next Steps
1. Monitor system performance improvement
2. Investigate root cause of performance degradation
3. Adjust autoscaling parameters if needed
4. Consider permanent capacity increase

## Monitoring
- Watch component performance metrics
- Monitor resource utilization
- Check error rates and latency
ALERT

    echo "Emergency scaling alert created: emergency-scaling-alert-$(date +%s).md"
    ;;
    
  *)
    echo "❌ Unknown action: $ACTION"
    echo "Supported actions: scale-up, scale-down, emergency-scale"
    exit 1
    ;;
esac

# Show current status
echo -e "\nCurrent deployment status:"
kubectl get deployments,statefulsets -n "$NAMESPACE" -o custom-columns=NAME:.metadata.name,DESIRED:.spec.replicas,CURRENT:.status.replicas,READY:.status.readyReplicas | grep "$COMPONENT"

# Show pod status
echo -e "\nPod status:"
kubectl get pods -n "$NAMESPACE" -l "app=$COMPONENT" -o wide

# Performance validation
echo -e "\n=== Performance Validation ==="
case "$COMPONENT" in
  "llm-processor")
    echo "Testing LLM Processor performance..."
    kubectl port-forward -n "$NAMESPACE" svc/llm-processor 8080:8080 >/dev/null 2>&1 &
    PF_PID=$!
    sleep 3
    
    if curl -s http://localhost:8080/health | jq -e '.status == "healthy"' >/dev/null; then
      echo "✅ LLM Processor health check passed"
    else
      echo "⚠️  LLM Processor health check failed"
    fi
    
    kill $PF_PID 2>/dev/null
    ;;
    
  "rag-api")
    echo "Testing RAG API performance..."
    kubectl port-forward -n "$NAMESPACE" svc/rag-api 8082:8080 >/dev/null 2>&1 &
    PF_PID=$!
    sleep 3
    
    if curl -s http://localhost:8082/health | jq -e '.status == "healthy"' >/dev/null; then
      echo "✅ RAG API health check passed"
    else
      echo "⚠️  RAG API health check failed"
    fi
    
    kill $PF_PID 2>/dev/null
    ;;
    
  *)
    echo "Basic connectivity test..."
    if kubectl get pods -n "$NAMESPACE" -l "app=$COMPONENT" | grep -q Running; then
      echo "✅ $COMPONENT pods are running"
    else
      echo "⚠️  $COMPONENT pods may not be ready"
    fi
    ;;
esac

echo "✅ Manual scaling procedure completed for $COMPONENT"
EOF

chmod +x manual-scaling.sh
```

## Cache Optimization

### Cache Performance Optimization

```bash
# Comprehensive cache optimization script
cat > optimize-caches.sh << 'EOF'
#!/bin/bash
set -e

NAMESPACE="nephoran-system"
OPTIMIZATION_TYPE=${1:-"all"} # all, llm, rag, redis

echo "=== CACHE OPTIMIZATION ==="
echo "Type: $OPTIMIZATION_TYPE"
echo "Namespace: $NAMESPACE"

# 1. LLM Cache Optimization
if [ "$OPTIMIZATION_TYPE" = "all" ] || [ "$OPTIMIZATION_TYPE" = "llm" ]; then
  echo "=== LLM Cache Optimization ==="
  
  # Get current LLM cache metrics
  kubectl port-forward -n "$NAMESPACE" svc/llm-processor 8080:8080 >/dev/null 2>&1 &
  LLM_PF=$!
  sleep 3
  
  echo "Current LLM cache metrics:"
  cache_hit_rate=$(curl -s http://localhost:8080/metrics | grep "llm_cache_hit_rate" | awk '{print $2}' || echo "0")
  cache_size=$(curl -s http://localhost:8080/metrics | grep "llm_cache_size" | awk '{print $2}' || echo "0")
  cache_evictions=$(curl -s http://localhost:8080/metrics | grep "llm_cache_evictions_total" | awk '{print $2}' || echo "0")
  
  echo "- Hit rate: $cache_hit_rate"
  echo "- Cache size: $cache_size"
  echo "- Evictions: $cache_evictions"
  
  kill $LLM_PF 2>/dev/null
  
  # Optimize LLM cache settings
  current_cache_size=$(kubectl get configmap llm-processor-config -n "$NAMESPACE" -o jsonpath='{.data.cache_size}' 2>/dev/null || echo "1000")
  current_cache_ttl=$(kubectl get configmap llm-processor-config -n "$NAMESPACE" -o jsonpath='{.data.cache_ttl}' 2>/dev/null || echo "3600")
  
  echo "Current cache configuration:"
  echo "- Size: $current_cache_size"
  echo "- TTL: $current_cache_ttl seconds"
  
  # Calculate optimal cache settings
  if (( $(echo "$cache_hit_rate < 0.7" | bc -l 2>/dev/null || echo "0") )); then
    # Low hit rate - increase cache size and TTL
    new_cache_size=$((current_cache_size * 2))
    new_cache_ttl=$((current_cache_ttl + 1800))
    echo "Optimizing for low hit rate..."
  elif (( $(echo "$cache_evictions > 100" | bc -l 2>/dev/null || echo "0") )); then
    # High evictions - increase cache size
    new_cache_size=$((current_cache_size + 500))
    new_cache_ttl=$current_cache_ttl
    echo "Optimizing for high evictions..."
  else
    # Good performance - fine-tune
    new_cache_size=$current_cache_size
    new_cache_ttl=$current_cache_ttl
    echo "Cache performance acceptable - no changes needed"
  fi
  
  if [ "$new_cache_size" -ne "$current_cache_size" ] || [ "$new_cache_ttl" -ne "$current_cache_ttl" ]; then
    echo "Applying cache optimizations..."
    kubectl patch configmap llm-processor-config -n "$NAMESPACE" --type='merge' -p="{\"data\":{
      \"cache_size\": \"$new_cache_size\",
      \"cache_ttl\": \"$new_cache_ttl\",
      \"cache_cleanup_interval\": \"300\"
    }}"
    
    # Restart to apply changes
    kubectl rollout restart deployment/llm-processor -n "$NAMESPACE"
    kubectl wait --for=condition=available --timeout=300s deployment/llm-processor -n "$NAMESPACE"
    
    echo "✅ LLM cache optimized: Size=$new_cache_size, TTL=$new_cache_ttl"
  fi
fi

# 2. RAG Cache Optimization
if [ "$OPTIMIZATION_TYPE" = "all" ] || [ "$OPTIMIZATION_TYPE" = "rag" ]; then
  echo -e "\n=== RAG Cache Optimization ==="
  
  # Get current RAG cache metrics
  kubectl port-forward -n "$NAMESPACE" svc/rag-api 8082:8080 >/dev/null 2>&1 &
  RAG_PF=$!
  sleep 3
  
  echo "Current RAG cache metrics:"
  rag_hit_rate=$(curl -s http://localhost:8082/metrics | grep "rag_cache_hit_rate" | awk '{print $2}' || echo "0")
  rag_cache_size=$(curl -s http://localhost:8082/metrics | grep "rag_cache_entries" | awk '{print $2}' || echo "0")
  
  echo "- Hit rate: $rag_hit_rate"
  echo "- Cache entries: $rag_cache_size"
  
  kill $RAG_PF 2>/dev/null
  
  # Optimize RAG cache based on usage patterns
  current_rag_cache_size=$(kubectl get configmap rag-api-config -n "$NAMESPACE" -o jsonpath='{.data.cache_max_size}' 2>/dev/null || echo "5000")
  current_rag_ttl=$(kubectl get configmap rag-api-config -n "$NAMESPACE" -o jsonpath='{.data.cache_ttl}' 2>/dev/null || echo "1800")
  
  # Analyze query patterns to optimize cache
  echo "Analyzing query patterns..."
  if (( $(echo "$rag_hit_rate < 0.6" | bc -l 2>/dev/null || echo "0") )); then
    new_rag_cache_size=$((current_rag_cache_size * 2))
    new_rag_ttl=$((current_rag_ttl + 900))
    
    kubectl patch configmap rag-api-config -n "$NAMESPACE" --type='merge' -p="{\"data\":{
      \"cache_max_size\": \"$new_rag_cache_size\",
      \"cache_ttl\": \"$new_rag_ttl\",
      \"cache_strategy\": \"lru_with_frequency\"
    }}"
    
    kubectl rollout restart deployment/rag-api -n "$NAMESPACE"
    kubectl wait --for=condition=available --timeout=300s deployment/rag-api -n "$NAMESPACE"
    
    echo "✅ RAG cache optimized for better hit rate"
  else
    echo "RAG cache performance acceptable"
  fi
fi

# 3. Redis Cache Optimization (if using Redis)
if [ "$OPTIMIZATION_TYPE" = "all" ] || [ "$OPTIMIZATION_TYPE" = "redis" ]; then
  echo -e "\n=== Redis Cache Optimization ==="
  
  if kubectl get deployment redis -n "$NAMESPACE" >/dev/null 2>&1; then
    echo "Redis deployment found - optimizing..."
    
    # Get Redis info
    kubectl exec -n "$NAMESPACE" deployment/redis -- redis-cli INFO memory > /tmp/redis-info.txt 2>/dev/null || echo "Cannot get Redis info"
    
    if [ -f /tmp/redis-info.txt ]; then
      used_memory=$(grep "used_memory:" /tmp/redis-info.txt | cut -d: -f2 | tr -d '\r')
      max_memory=$(grep "maxmemory:" /tmp/redis-info.txt | cut -d: -f2 | tr -d '\r')
      
      echo "Redis memory usage: $used_memory"
      echo "Redis max memory: $max_memory"
      
      # Optimize Redis configuration
      kubectl exec -n "$NAMESPACE" deployment/redis -- redis-cli CONFIG SET maxmemory-policy "allkeys-lru"
      kubectl exec -n "$NAMESPACE" deployment/redis -- redis-cli CONFIG SET timeout 300
      kubectl exec -n "$NAMESPACE" deployment/redis -- redis-cli CONFIG SET tcp-keepalive 60
      
      echo "✅ Redis configuration optimized"
    fi
    
    rm -f /tmp/redis-info.txt
  else
    echo "Redis not found - skipping Redis optimization"
  fi
fi

# 4. Generate cache optimization report
echo -e "\n=== Cache Optimization Report ==="
cat > "cache-optimization-report-$(date +%Y%m%d-%H%M%S).txt" << REPORT
Cache Optimization Report
Generated: $(date -u)
Namespace: $NAMESPACE
Optimization Type: $OPTIMIZATION_TYPE

BEFORE OPTIMIZATION:
$(if [ -n "$cache_hit_rate" ]; then echo "- LLM Cache Hit Rate: $cache_hit_rate"; fi)
$(if [ -n "$cache_size" ]; then echo "- LLM Cache Size: $cache_size"; fi)
$(if [ -n "$rag_hit_rate" ]; then echo "- RAG Cache Hit Rate: $rag_hit_rate"; fi)

OPTIMIZATIONS APPLIED:
$(if [ "$new_cache_size" -ne "$current_cache_size" ] 2>/dev/null; then echo "- LLM Cache Size: $current_cache_size → $new_cache_size"; fi)
$(if [ "$new_cache_ttl" -ne "$current_cache_ttl" ] 2>/dev/null; then echo "- LLM Cache TTL: $current_cache_ttl → $new_cache_ttl seconds"; fi)
$(if [ "$new_rag_cache_size" -ne "$current_rag_cache_size" ] 2>/dev/null; then echo "- RAG Cache Size: $current_rag_cache_size → $new_rag_cache_size"; fi)

RECOMMENDATIONS:
- Monitor cache hit rates over next 24 hours
- Adjust cache sizes based on usage patterns
- Consider implementing cache warming for frequently accessed data
- Review cache eviction policies for optimal performance

NEXT REVIEW: $(date -d "+1 week" -u)
REPORT

echo "Cache optimization completed"
echo "Report saved: cache-optimization-report-$(date +%Y%m%d-%H%M%S).txt"

# 5. Validate cache optimizations
echo -e "\n=== Validation ==="
sleep 30 # Wait for changes to take effect

# Test LLM cache
if [ "$OPTIMIZATION_TYPE" = "all" ] || [ "$OPTIMIZATION_TYPE" = "llm" ]; then
  kubectl port-forward -n "$NAMESPACE" svc/llm-processor 8080:8080 >/dev/null 2>&1 &
  LLM_PF=$!
  sleep 3
  
  if curl -s http://localhost:8080/health | jq -e '.status == "healthy"' >/dev/null; then
    echo "✅ LLM Processor healthy after cache optimization"
  else
    echo "⚠️  LLM Processor may need attention after cache changes"
  fi
  
  kill $LLM_PF 2>/dev/null
fi

# Test RAG cache
if [ "$OPTIMIZATION_TYPE" = "all" ] || [ "$OPTIMIZATION_TYPE" = "rag" ]; then
  kubectl port-forward -n "$NAMESPACE" svc/rag-api 8082:8080 >/dev/null 2>&1 &
  RAG_PF=$!
  sleep 3
  
  if curl -s http://localhost:8082/health | jq -e '.status == "healthy"' >/dev/null; then
    echo "✅ RAG API healthy after cache optimization"
  else
    echo "⚠️  RAG API may need attention after cache changes"
  fi
  
  kill $RAG_PF 2>/dev/null
fi

echo "✅ Cache optimization and validation completed"
EOF

chmod +x optimize-caches.sh
```

## Database Tuning

### Weaviate Performance Tuning

```bash
# Weaviate database performance tuning
cat > tune-weaviate.sh << 'EOF'
#!/bin/bash
set -e

NAMESPACE="nephoran-system"
TUNING_TYPE=${1:-"performance"} # performance, memory, storage

echo "=== WEAVIATE DATABASE TUNING ==="
echo "Type: $TUNING_TYPE"
echo "Namespace: $NAMESPACE"

# Get current Weaviate status
kubectl port-forward -n "$NAMESPACE" svc/weaviate 8080:8080 >/dev/null 2>&1 &
WEAVIATE_PF=$!
sleep 5

echo "Current Weaviate status:"
curl -s http://localhost:8080/v1/meta | jq '{version: .version, hostname: .hostname}' 2>/dev/null || echo "Weaviate not responding"

case "$TUNING_TYPE" in
  "performance")
    echo "=== Performance Tuning ==="
    
    # Analyze current performance
    echo "Analyzing current performance..."
    
    # Test query performance baseline
    echo "Baseline query performance:"
    for i in {1..5}; do
      start_time=$(date +%s%N)
      curl -s "http://localhost:8080/v1/objects?limit=10" >/dev/null 2>&1
      end_time=$(date +%s%N)
      duration=$(( (end_time - start_time) / 1000000 ))
      echo "Query $i: ${duration}ms"
    done
    
    # Get current configuration
    current_config=$(kubectl get statefulset weaviate -n "$NAMESPACE" -o yaml)
    
    # Performance optimizations
    echo "Applying performance optimizations..."
    
    # 1. Increase memory allocation
    kubectl patch statefulset weaviate -n "$NAMESPACE" --type='merge' -p='{
      "spec": {
        "template": {
          "spec": {
            "containers": [{
              "name": "weaviate",
              "resources": {
                "limits": {"memory": "4Gi", "cpu": "2000m"},
                "requests": {"memory": "2Gi", "cpu": "1000m"}
              },
              "env": [
                {"name": "QUERY_DEFAULTS_LIMIT", "value": "100"},
                {"name": "QUERY_MAXIMUM_RESULTS", "value": "10000"},
                {"name": "ENABLE_MODULES", "value": "text2vec-openai,generative-openai"},
                {"name": "PERSISTENCE_DATA_PATH", "value": "/var/lib/weaviate"},
                {"name": "DEFAULT_VECTORIZER_MODULE", "value": "text2vec-openai"},
                {"name": "CLUSTER_HOSTNAME", "value": "weaviate"},
                {"name": "AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED", "value": "true"},
                {"name": "AUTOSCHEMA_ENABLED", "value": "true"},
                {"name": "TRACK_VECTOR_DIMENSIONS", "value": "true"},
                {"name": "REINDEX_VECTOR_DIMENSIONS_AT_STARTUP", "value": "false"}
              ]
            }]
          }
        }
      }
    }'
    
    # 2. Optimize storage performance
    echo "Optimizing storage settings..."
    storage_class=$(kubectl get pvc -n "$NAMESPACE" -l app=weaviate -o jsonpath='{.items[0].spec.storageClassName}')
    echo "Current storage class: $storage_class"
    
    # If using standard storage, recommend SSD
    if [ "$storage_class" = "standard" ] || [ "$storage_class" = "gp2" ]; then
      echo "⚠️  Consider upgrading to SSD storage class for better performance"
      echo "Current storage may be a bottleneck"
    fi
    
    # 3. Wait for rollout
    kubectl rollout status statefulset/weaviate -n "$NAMESPACE" --timeout=600s
    
    echo "✅ Performance optimizations applied"
    ;;
    
  "memory")
    echo "=== Memory Optimization ==="
    
    # Check memory usage
    kubectl exec -n "$NAMESPACE" weaviate-0 -- sh -c "cat /proc/meminfo | grep -E 'MemTotal|MemFree|MemAvailable'" 2>/dev/null || echo "Cannot check memory"
    
    # Optimize for memory efficiency
    kubectl patch statefulset weaviate -n "$NAMESPACE" --type='merge' -p='{
      "spec": {
        "template": {
          "spec": {
            "containers": [{
              "name": "weaviate",
              "env": [
                {"name": "GOMEMLIMIT", "value": "1500MiB"},
                {"name": "GOGC", "value": "50"},
                {"name": "PERSISTENCE_LSM_BLOOM_FILTER_FALSEPOSITIVE_RATE", "value": "0.01"},
                {"name": "PERSISTENCE_LSM_MEMTABLE_SIZE_MB", "value": "128"},
                {"name": "PERSISTENCE_LSM_MAX_SEGMENT_SIZE", "value": "500mb"}
              ]
            }]
          }
        }
      }
    }'
    
    echo "✅ Memory optimizations applied"
    ;;
    
  "storage")
    echo "=== Storage Optimization ==="
    
    # Check current storage usage
    kubectl exec -n "$NAMESPACE" weaviate-0 -- df -h /var/lib/weaviate 2>/dev/null || echo "Cannot check storage"
    
    # Storage optimization settings
    kubectl patch statefulset weaviate -n "$NAMESPACE" --type='merge' -p='{
      "spec": {
        "template": {
          "spec": {
            "containers": [{
              "name": "weaviate",
              "env": [
                {"name": "PERSISTENCE_LSM_STRATEGY", "value": "replace"},
                {"name": "PERSISTENCE_LSM_COMPACTION_INITIAL_DELAY", "value": "60s"},
                {"name": "PERSISTENCE_LSM_COMPACTION_INTERVAL", "value": "600s"},
                {"name": "PERSISTENCE_DATA_PATH", "value": "/var/lib/weaviate"}
              ]
            }]
          }
        }
      }
    }'
    
    echo "✅ Storage optimizations applied"
    ;;
    
  *)
    echo "❌ Unknown tuning type: $TUNING_TYPE"
    echo "Supported types: performance, memory, storage"
    kill $WEAVIATE_PF 2>/dev/null
    exit 1
    ;;
esac

kill $WEAVIATE_PF 2>/dev/null

# Wait for changes to take effect
echo "Waiting for Weaviate to restart with new configuration..."
kubectl wait --for=condition=ready pod -l app=weaviate -n "$NAMESPACE" --timeout=600s

# Validate tuning
echo "=== Validation ==="
kubectl port-forward -n "$NAMESPACE" svc/weaviate 8080:8080 >/dev/null 2>&1 &
WEAVIATE_PF=$!
sleep 10

echo "Post-tuning validation:"

# Health check
if curl -s http://localhost:8080/v1/meta >/dev/null 2>&1; then
  echo "✅ Weaviate responding after tuning"
  
  # Performance test
  echo "Performance validation:"
  total_time=0
  for i in {1..5}; do
    start_time=$(date +%s%N)
    curl -s "http://localhost:8080/v1/objects?limit=10" >/dev/null 2>&1
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))
    echo "Query $i: ${duration}ms"
    total_time=$((total_time + duration))
  done
  
  avg_time=$((total_time / 5))
  echo "Average query time: ${avg_time}ms"
  
  # Data integrity check
  object_count=$(curl -s "http://localhost:8080/v1/objects?limit=1" | jq -r '.totalResults // 0')
  schema_count=$(curl -s "http://localhost:8080/v1/schema" | jq -r '.classes | length')
  
  echo "✅ Data integrity: $object_count objects, $schema_count schemas"
  
else
  echo "❌ Weaviate not responding after tuning"
  echo "Check pod status and logs:"
  kubectl get pods -n "$NAMESPACE" -l app=weaviate
  kubectl logs -n "$NAMESPACE" -l app=weaviate --tail=20
fi

kill $WEAVIATE_PF 2>/dev/null

# Generate tuning report
cat > "weaviate-tuning-report-$(date +%Y%m%d-%H%M%S).txt" << REPORT
Weaviate Database Tuning Report
Generated: $(date -u)
Namespace: $NAMESPACE
Tuning Type: $TUNING_TYPE

OPTIMIZATIONS APPLIED:
$(case "$TUNING_TYPE" in
  "performance")
    echo "- Increased memory limits to 4Gi"
    echo "- Increased CPU limits to 2000m"
    echo "- Optimized query defaults and limits"
    echo "- Enabled performance-oriented modules"
    ;;
  "memory")
    echo "- Set GOMEMLIMIT to 1500MiB"
    echo "- Optimized garbage collection (GOGC=50)"
    echo "- Tuned LSM tree memory settings"
    ;;
  "storage")
    echo "- Optimized LSM strategy for storage efficiency"
    echo "- Tuned compaction intervals"
    echo "- Optimized data path configuration"
    ;;
esac)

POST-TUNING METRICS:
- Average Query Time: ${avg_time:-"N/A"}ms
- Object Count: ${object_count:-"N/A"}
- Schema Count: ${schema_count:-"N/A"}

RECOMMENDATIONS:
1. Monitor performance for 24 hours after tuning
2. Adjust memory limits if needed based on usage
3. Consider storage class upgrade for better I/O
4. Schedule regular performance reviews

MONITORING:
- Watch memory usage patterns
- Monitor query latency trends
- Check for any data consistency issues
- Validate backup/restore procedures

Next Tuning Review: $(date -d "+1 month" -u)
REPORT

echo "✅ Weaviate tuning completed"
echo "Report saved: weaviate-tuning-report-$(date +%Y%m%d-%H%M%S).txt"
EOF

chmod +x tune-weaviate.sh
```

## Resource Optimization

### System-Wide Resource Optimization

```bash
# Comprehensive resource optimization script
cat > optimize-resources.sh << 'EOF'
#!/bin/bash
set -e

NAMESPACE="nephoran-system"
OPTIMIZATION_SCOPE=${1:-"all"} # all, cpu, memory, network, storage

echo "=== SYSTEM RESOURCE OPTIMIZATION ==="
echo "Scope: $OPTIMIZATION_SCOPE"
echo "Namespace: $NAMESPACE"

OUTPUT_DIR="/tmp/resource-optimization-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUTPUT_DIR"

# Collect current resource utilization
echo "Collecting current resource utilization..."
kubectl top nodes > "$OUTPUT_DIR/nodes-before.txt" 2>/dev/null || echo "Node metrics unavailable"
kubectl top pods -n "$NAMESPACE" > "$OUTPUT_DIR/pods-before.txt" 2>/dev/null || echo "Pod metrics unavailable"

# Get resource requests and limits
kubectl get pods -n "$NAMESPACE" -o json | jq -r '
  .items[] | 
  "\(.metadata.name): CPU_REQ=\(.spec.containers[0].resources.requests.cpu // "none") CPU_LIM=\(.spec.containers[0].resources.limits.cpu // "none") MEM_REQ=\(.spec.containers[0].resources.requests.memory // "none") MEM_LIM=\(.spec.containers[0].resources.limits.memory // "none")"
' > "$OUTPUT_DIR/resource-config-before.txt"

if [ "$OPTIMIZATION_SCOPE" = "all" ] || [ "$OPTIMIZATION_SCOPE" = "cpu" ]; then
  echo "=== CPU Optimization ==="
  
  # Analyze CPU usage patterns
  echo "Analyzing CPU usage patterns..."
  
  # Get CPU utilization for each component
  components=("nephoran-controller" "llm-processor" "rag-api" "weaviate" "nephio-bridge")
  for component in "${components[@]}"; do
    echo "Optimizing CPU for $component..."
    
    # Get current CPU usage
    current_cpu=$(kubectl top pods -n "$NAMESPACE" -l "app=$component" --no-headers 2>/dev/null | awk '{print $2}' | sed 's/m//' | head -1)
    
    if [ -n "$current_cpu" ] && [ "$current_cpu" -gt 0 ]; then
      # Calculate optimal CPU request/limit based on usage
      optimal_request=$(( (current_cpu * 120) / 100 )) # 20% headroom
      optimal_limit=$(( optimal_request * 2 )) # 2x request for bursting
      
      echo "Component: $component"
      echo "  Current usage: ${current_cpu}m"
      echo "  Recommended request: ${optimal_request}m"
      echo "  Recommended limit: ${optimal_limit}m"
      
      # Apply CPU optimization
      if kubectl get deployment "$component" -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl patch deployment "$component" -n "$NAMESPACE" --type='merge' -p="{
          \"spec\": {
            \"template\": {
              \"spec\": {
                \"containers\": [{
                  \"name\": \"$component\",
                  \"resources\": {
                    \"requests\": {\"cpu\": \"${optimal_request}m\"},
                    \"limits\": {\"cpu\": \"${optimal_limit}m\"}
                  }
                }]
              }
            }
          }
        }"
      elif kubectl get statefulset "$component" -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl patch statefulset "$component" -n "$NAMESPACE" --type='merge' -p="{
          \"spec\": {
            \"template\": {
              \"spec\": {
                \"containers\": [{
                  \"name\": \"$component\",
                  \"resources\": {
                    \"requests\": {\"cpu\": \"${optimal_request}m\"},
                    \"limits\": {\"cpu\": \"${optimal_limit}m\"}
                  }
                }]
              }
            }
          }
        }"
      fi
    fi
  done
  
  echo "✅ CPU optimization applied"
fi

if [ "$OPTIMIZATION_SCOPE" = "all" ] || [ "$OPTIMIZATION_SCOPE" = "memory" ]; then
  echo -e "\n=== Memory Optimization ==="
  
  # Analyze memory usage patterns
  echo "Analyzing memory usage patterns..."
  
  for component in "${components[@]}"; do
    echo "Optimizing memory for $component..."
    
    # Get current memory usage
    current_memory=$(kubectl top pods -n "$NAMESPACE" -l "app=$component" --no-headers 2>/dev/null | awk '{print $3}' | sed 's/Mi//' | head -1)
    
    if [ -n "$current_memory" ] && [ "$current_memory" -gt 0 ]; then
      # Calculate optimal memory based on component type
      case "$component" in
        "weaviate")
          optimal_request="${current_memory}Mi"
          optimal_limit="$((current_memory + 1024))Mi"
          ;;
        "llm-processor")
          optimal_request="${current_memory}Mi"
          optimal_limit="$((current_memory + 512))Mi"
          ;;
        *)
          optimal_request="${current_memory}Mi"
          optimal_limit="$((current_memory + 256))Mi"
          ;;
      esac
      
      echo "Component: $component"
      echo "  Current usage: ${current_memory}Mi"
      echo "  Recommended request: $optimal_request"
      echo "  Recommended limit: $optimal_limit"
      
      # Apply memory optimization
      if kubectl get deployment "$component" -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl patch deployment "$component" -n "$NAMESPACE" --type='merge' -p="{
          \"spec\": {
            \"template\": {
              \"spec\": {
                \"containers\": [{
                  \"name\": \"$component\",
                  \"resources\": {
                    \"requests\": {\"memory\": \"$optimal_request\"},
                    \"limits\": {\"memory\": \"$optimal_limit\"}
                  }
                }]
              }
            }
          }
        }"
      elif kubectl get statefulset "$component" -n "$NAMESPACE" >/dev/null 2>&1; then
        kubectl patch statefulset "$component" -n "$NAMESPACE" --type='merge' -p="{
          \"spec\": {
            \"template\": {
              \"spec\": {
                \"containers\": [{
                  \"name\": \"$component\",
                  \"resources\": {
                    \"requests\": {\"memory\": \"$optimal_request\"},
                    \"limits\": {\"memory\": \"$optimal_limit\"}
                  }
                }]
              }
            }
          }
        }"
      fi
    fi
  done
  
  echo "✅ Memory optimization applied"
fi

if [ "$OPTIMIZATION_SCOPE" = "all" ] || [ "$OPTIMIZATION_SCOPE" = "network" ]; then
  echo -e "\n=== Network Optimization ==="
  
  # Optimize service configurations
  echo "Optimizing service configurations..."
  
  # Check current service types and optimize
  services=$(kubectl get services -n "$NAMESPACE" -o name)
  for service in $services; do
    service_name=$(echo "$service" | cut -d'/' -f2)
    service_type=$(kubectl get "$service" -n "$NAMESPACE" -o jsonpath='{.spec.type}')
    
    echo "Service: $service_name, Type: $service_type"
    
    # Optimize service configuration for performance
    if [ "$service_type" = "ClusterIP" ]; then
      kubectl patch service "$service_name" -n "$NAMESPACE" --type='merge' -p='{
        "spec": {
          "sessionAffinity": "None",
          "ipFamilyPolicy": "SingleStack"
        }
      }'
    fi
  done
  
  # Optimize network policies for performance
  echo "Reviewing network policies..."
  netpol_count=$(kubectl get networkpolicies -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
  if [ "$netpol_count" -gt 5 ]; then
    echo "⚠️  High number of network policies ($netpol_count) may impact performance"
    echo "Consider consolidating network policies for better performance"
  fi
  
  echo "✅ Network optimization completed"
fi

if [ "$OPTIMIZATION_SCOPE" = "all" ] || [ "$OPTIMIZATION_SCOPE" = "storage" ]; then
  echo -e "\n=== Storage Optimization ==="
  
  # Analyze storage usage
  echo "Analyzing storage usage..."
  
  # Check PVC usage
  pvcs=$(kubectl get pvc -n "$NAMESPACE" -o name)
  for pvc in $pvcs; do
    pvc_name=$(echo "$pvc" | cut -d'/' -f2)
    storage_class=$(kubectl get "$pvc" -n "$NAMESPACE" -o jsonpath='{.spec.storageClassName}')
    size=$(kubectl get "$pvc" -n "$NAMESPACE" -o jsonpath='{.spec.resources.requests.storage}')
    
    echo "PVC: $pvc_name, Class: $storage_class, Size: $size"
    
    # Recommend storage class optimization
    if [ "$storage_class" = "standard" ] || [ "$storage_class" = "gp2" ]; then
      echo "  ⚠️  Consider upgrading to SSD storage class for better performance"
    fi
  done
  
  # Check storage utilization in pods
  echo "Checking storage utilization..."
  for component in "${components[@]}"; do
    if kubectl get pods -n "$NAMESPACE" -l "app=$component" >/dev/null 2>&1; then
      pod_name=$(kubectl get pods -n "$NAMESPACE" -l "app=$component" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
      if [ -n "$pod_name" ]; then
        echo "Storage usage for $component:"
        kubectl exec -n "$NAMESPACE" "$pod_name" -- df -h 2>/dev/null | grep -E "(Filesystem|/var|/data|/tmp)" || echo "  Cannot check storage usage"
      fi
    fi
  done
  
  echo "✅ Storage analysis completed"
fi

# Wait for all changes to be applied
echo -e "\n=== Applying Changes ==="
echo "Waiting for deployments to roll out..."
kubectl rollout status deployment --all -n "$NAMESPACE" --timeout=600s
kubectl rollout status statefulset --all -n "$NAMESPACE" --timeout=600s

# Collect post-optimization metrics
echo -e "\n=== Post-Optimization Metrics ==="
sleep 60 # Wait for metrics to stabilize

kubectl top nodes > "$OUTPUT_DIR/nodes-after.txt" 2>/dev/null || echo "Node metrics unavailable"
kubectl top pods -n "$NAMESPACE" > "$OUTPUT_DIR/pods-after.txt" 2>/dev/null || echo "Pod metrics unavailable"

kubectl get pods -n "$NAMESPACE" -o json | jq -r '
  .items[] | 
  "\(.metadata.name): CPU_REQ=\(.spec.containers[0].resources.requests.cpu // "none") CPU_LIM=\(.spec.containers[0].resources.limits.cpu // "none") MEM_REQ=\(.spec.containers[0].resources.requests.memory // "none") MEM_LIM=\(.spec.containers[0].resources.limits.memory // "none")"
' > "$OUTPUT_DIR/resource-config-after.txt"

# Generate optimization report
cat > "$OUTPUT_DIR/optimization-report.txt" << REPORT
Resource Optimization Report
Generated: $(date -u)
Namespace: $NAMESPACE
Scope: $OPTIMIZATION_SCOPE

OPTIMIZATION SUMMARY:
$(if [ "$OPTIMIZATION_SCOPE" = "all" ] || [ "$OPTIMIZATION_SCOPE" = "cpu" ]; then echo "✅ CPU optimization applied"; fi)
$(if [ "$OPTIMIZATION_SCOPE" = "all" ] || [ "$OPTIMIZATION_SCOPE" = "memory" ]; then echo "✅ Memory optimization applied"; fi)
$(if [ "$OPTIMIZATION_SCOPE" = "all" ] || [ "$OPTIMIZATION_SCOPE" = "network" ]; then echo "✅ Network optimization applied"; fi)
$(if [ "$OPTIMIZATION_SCOPE" = "all" ] || [ "$OPTIMIZATION_SCOPE" = "storage" ]; then echo "✅ Storage analysis completed"; fi)

FILES GENERATED:
- nodes-before.txt / nodes-after.txt: Node resource usage
- pods-before.txt / pods-after.txt: Pod resource usage
- resource-config-before.txt / resource-config-after.txt: Resource configurations

RECOMMENDATIONS:
1. Monitor resource utilization over next 24 hours
2. Adjust resource limits based on actual usage patterns
3. Consider implementing resource quotas for better governance
4. Schedule regular resource optimization reviews

NEXT STEPS:
1. Validate application performance after optimization
2. Monitor for any resource contention issues
3. Update monitoring thresholds based on new resource allocation
4. Document changes in operational procedures

Review Date: $(date -d "+2 weeks" -u)
REPORT

echo "✅ Resource optimization completed"
echo "Results saved in: $OUTPUT_DIR"
echo "Review optimization-report.txt for summary and recommendations"

# Validation
echo -e "\n=== Validation ==="
echo "Checking component health after optimization..."
failed_components=0
for component in "${components[@]}"; do
  if kubectl get pods -n "$NAMESPACE" -l "app=$component" | grep -q Running; then
    echo "✅ $component: Running"
  else
    echo "❌ $component: Not running properly"
    failed_components=$((failed_components + 1))
  fi
done

if [ $failed_components -eq 0 ]; then
  echo "✅ All components healthy after resource optimization"
else
  echo "⚠️  $failed_components components may need attention"
  echo "Check individual component status and logs"
fi
EOF

chmod +x optimize-resources.sh
```

---

**Document Version**: 1.0  
**Last Updated**: 2025-01-07  
**Next Review**: 2025-02-07  
**Owner**: Nephoran Performance Engineering Team  
**Approvers**: Operations Manager, Engineering Manager, Performance Architect