# Nephoran Intent Operator - Troubleshooting Runbooks

## Overview

This document provides comprehensive troubleshooting runbooks for the enhanced Nephoran Intent Operator, covering async processing failures, multi-level caching issues, circuit breaker problems, and performance optimization challenges. Each runbook includes diagnostic procedures, resolution steps, and prevention strategies.

## Table of Contents

1. [Async Processing Failures](#async-processing-failures)
2. [Streaming Session Issues](#streaming-session-issues)
3. [Multi-Level Caching Problems](#multi-level-caching-problems)
4. [Circuit Breaker Failures](#circuit-breaker-failures)
5. [Performance Optimization Issues](#performance-optimization-issues)
6. [Intent Processing Errors](#intent-processing-errors)
7. [API Gateway and Rate Limiting](#api-gateway-and-rate-limiting)
8. [System Resource Issues](#system-resource-issues)

---

## Async Processing Failures

### 🚨 **Symptom: High Rate of Async Processing Timeouts**

**Error Code**: `ASYNC_PROCESSING_TIMEOUT`

**Symptoms**:
- Async intents stuck in "processing" state
- Webhook callbacks not being triggered
- Increasing queue depth metrics
- Client timeout errors

#### **Diagnostic Steps**

1. **Check Processing Queue Status**:
```bash
# Check current queue depth
curl -s http://llm-processor:8080/metrics | grep -E "queue_depth|processing_time"

# Monitor queue growth over time
watch -n 5 "curl -s http://llm-processor:8080/metrics | grep async_queue_depth"
```

2. **Analyze Processing Times**:
```bash
# Get processing time breakdown
curl -X GET http://llm-processor:8080/performance/metrics | jq '{
  "document_processing": .document_processing_time,
  "embedding_generation": .embedding_generation_time,
  "retrieval": .retrieval_time,
  "context_assembly": .context_assembly_time
}'
```

3. **Check Worker Health**:
```bash
# Verify async workers are running
kubectl logs -l app=llm-processor -n nephoran-system | grep -i "async worker"

# Check worker resource utilization
kubectl top pods -l app=llm-processor -n nephoran-system
```

#### **Resolution Steps**

**Immediate Actions (0-15 minutes)**:

1. **Scale Up Processing Capacity**:
```bash
# Increase replica count
kubectl scale deployment llm-processor --replicas=5 -n nephoran-system

# Monitor scaling progress
kubectl rollout status deployment/llm-processor -n nephoran-system
```

2. **Clear Processing Queue Backlog**:
```bash
# Priority process high-priority intents
curl -X POST http://llm-processor:8080/queue/priority-process \
  -H "Content-Type: application/json" \
  -d '{"priority": "high", "max_items": 50}'
```

3. **Temporary Timeout Extension**:
```bash
# Increase processing timeout temporarily
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{"ASYNC_PROCESSING_TIMEOUT":"600s"}}'

# Restart pods to pick up new config
kubectl rollout restart deployment/llm-processor -n nephoran-system
```

**Short-term Fixes (15-60 minutes)**:

4. **Optimize Resource Allocation**:
```bash
# Increase memory limits for better performance
kubectl patch deployment llm-processor -n nephoran-system \
  --type json -p='[{
    "op": "replace",
    "path": "/spec/template/spec/containers/0/resources/limits/memory",
    "value": "4Gi"
  }]'
```

5. **Enable Circuit Breaker Fast-Fail Mode**:
```bash
# Reduce circuit breaker thresholds for faster failure detection
curl -X POST http://llm-processor:8080/circuit-breaker/configure \
  -H "Content-Type: application/json" \
  -d '{
    "name": "llm-processor",
    "failure_threshold": 3,
    "timeout": "15s"
  }'
```

**Long-term Solutions**:

6. **Implement Auto-Scaling Based on Queue Depth**:
```yaml
# Add HPA based on custom queue depth metric
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-processor-queue-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Pods
    pods:
      metric:
        name: async_queue_depth
      target:
        type: AverageValue
        averageValue: "10"
```

7. **Optimize Processing Pipeline**:
```bash
# Enable batch processing for better throughput
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_BATCH_PROCESSING": "true",
    "BATCH_SIZE": "25",
    "BATCH_TIMEOUT": "5s"
  }}'
```

#### **Prevention Strategies**

- **Implement Predictive Scaling**: Use queue depth trends to pre-scale resources
- **Optimize Cache Hit Rates**: Improve caching to reduce processing times
- **Queue Management**: Implement priority queues and load balancing
- **Regular Load Testing**: Perform monthly capacity planning exercises

#### **Monitoring and Alerting**

```yaml
# Prometheus Alert Rules
- alert: AsyncProcessingTimeout
  expr: increase(async_processing_timeout_total[5m]) > 5
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "High rate of async processing timeouts"
    runbook_url: "https://docs.nephoran.com/runbooks/async-processing-failures"

- alert: AsyncQueueDepthHigh  
  expr: async_queue_depth > 100
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Async processing queue depth is critically high"
```

---

## Streaming Session Issues

### 🚨 **Symptom: Streaming Sessions Disconnecting Frequently**

**Error Code**: `STREAMING_SESSION_DISCONNECTED`

**Symptoms**:
- Clients frequently losing SSE connections
- High rate of session reconnection attempts
- Incomplete streaming responses
- Network timeout errors

#### **Diagnostic Steps**

1. **Check Active Session Status**:
```bash
# List all active streaming sessions
curl -X GET http://llm-processor:8080/stream/sessions | jq '.active_sessions[] | {
  id: .id,
  status: .status,
  duration: .duration,
  last_activity: .last_activity
}'
```

2. **Analyze Connection Patterns**:
```bash
# Check for connection issues in logs  
kubectl logs -l app=llm-processor -n nephoran-system | \
  grep -E "stream.*disconnect|session.*timeout" | tail -20
```

3. **Monitor Network Connectivity**:
```bash
# Test streaming endpoint connectivity
curl -N -X POST http://llm-processor:8080/stream \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"query":"test connection","session_id":"diagnostic_test"}' \
  | head -10
```

#### **Resolution Steps**

**Immediate Actions**:

1. **Increase Session Timeouts**:
```bash
# Extend session timeout values
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "STREAM_TIMEOUT": "600s",
    "HEARTBEAT_INTERVAL": "10s",
    "MAX_IDLE_TIMEOUT": "300s"
  }}'
```

2. **Clear Stuck Sessions**:
```bash
# Force cleanup of stuck sessions
curl -X DELETE http://llm-processor:8080/stream/sessions/cleanup \
  -H "Content-Type: application/json" \
  -d '{"max_idle_time": "300s", "force": true}'
```

3. **Enable Connection Persistence**:
```bash
# Configure keep-alive settings for better connection stability
kubectl patch service llm-processor -n nephoran-system \
  --type merge -p='{"spec":{"sessionAffinity":"ClientIP"}}'
```

**Network-Level Fixes**:

4. **Configure Load Balancer for Streaming**:
```yaml
# Ingress configuration for streaming support
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: llm-processor-streaming
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "600"
    nginx.ingress.kubernetes.io/proxy-buffering: "off"
    nginx.ingress.kubernetes.io/proxy-request-buffering: "off"
spec:
  rules:
  - host: api.nephoran.com
    http:
      paths:
      - path: /stream
        pathType: Prefix
        backend:
          service:
            name: llm-processor
            port:
              number: 8080
```

5. **Implement Connection Pooling**:
```bash
# Enable HTTP/2 and connection reuse
kubectl patch deployment llm-processor -n nephoran-system \
  --type json -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {"name": "ENABLE_HTTP2", "value": "true"}
  }]'
```

**Application-Level Optimizations**:

6. **Optimize Buffer Sizes**:
```bash
# Adjust streaming buffer configuration
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "STREAM_BUFFER_SIZE": "16384",
    "CHUNK_SIZE": "512",
    "MAX_CHUNK_DELAY": "10ms"
  }}'
```

7. **Enable Compression for Large Responses**:
```bash
# Enable streaming compression for bandwidth optimization
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{"ENABLE_STREAM_COMPRESSION": "true"}}'
```

#### **Client-Side Recommendations**

**JavaScript Client Example**:
```javascript
// Robust SSE client with automatic reconnection
class RobustEventSource {
  constructor(url, options = {}) {
    this.url = url;
    this.options = options;
    this.maxRetries = options.maxRetries || 5;
    this.retryDelay = options.retryDelay || 1000;
    this.retryCount = 0;
    
    this.connect();
  }
  
  connect() {
    this.eventSource = new EventSource(this.url);
    
    this.eventSource.onopen = () => {
      console.log('SSE connection established');
      this.retryCount = 0;
    };
    
    this.eventSource.onerror = (error) => {
      console.error('SSE connection error:', error);
      this.handleReconnection();
    };
    
    this.eventSource.onmessage = (event) => {
      if (this.options.onMessage) {
        this.options.onMessage(event);
      }
    };
  }
  
  handleReconnection() {
    if (this.retryCount < this.maxRetries) {
      this.retryCount++;
      console.log(`Attempting reconnection ${this.retryCount}/${this.maxRetries}`);
      
      setTimeout(() => {
        this.eventSource.close();
        this.connect();
      }, this.retryDelay * Math.pow(2, this.retryCount - 1)); // Exponential backoff
    } else {
      console.error('Max reconnection attempts reached');
      if (this.options.onError) {
        this.options.onError('Max reconnection attempts exceeded');
      }
    }
  }
  
  close() {
    if (this.eventSource) {
      this.eventSource.close();
    }
  }
}

// Usage
const stream = new RobustEventSource('/stream', {
  maxRetries: 5,
  retryDelay: 1000,
  onMessage: (event) => {
    console.log('Received:', event.data);
  },
  onError: (error) => {
    console.error('Stream failed:', error);
  }
});
```

#### **Prevention Strategies**

- **Connection Health Monitoring**: Implement client-side connection health checks
- **Graceful Degradation**: Provide fallback to polling for unreliable connections
- **Network Optimization**: Use CDN and edge caching for better connectivity
- **Load Testing**: Regular streaming load tests under various network conditions

---

## Multi-Level Caching Problems

### 🚨 **Symptom: Cache Hit Rate Below 60%**

**Error Code**: `CACHE_PERFORMANCE_DEGRADATION`

**Symptoms**:
- Overall cache hit rate below expected threshold (60-70%)
- High latency on cached requests
- Frequent cache evictions
- Memory pressure warnings

#### **Diagnostic Steps**

1. **Analyze Cache Performance**:
```bash
# Get detailed cache statistics
curl -X GET http://llm-processor:8080/cache/stats | jq '{
  "l1_hit_rate": .l1_stats.hit_rate,
  "l2_hit_rate": .l2_stats.hit_rate,
  "overall_hit_rate": .overall_stats.overall_hit_rate,
  "eviction_rate": (.l1_stats.evictions / .l1_stats.sets),
  "error_rate": (.l2_stats.errors / (.l2_stats.hits + .l2_stats.misses))
}'
```

2. **Check Cache Key Distribution**:
```bash
# Analyze cache key patterns (requires debug mode)
curl -X GET http://llm-processor:8080/cache/debug/keys | \
  jq '.keys | group_by(.prefix) | map({prefix: .[0].prefix, count: length})'
```

3. **Monitor Memory Usage**:
```bash
# Check memory utilization
kubectl top pods -l app=llm-processor -n nephoran-system
kubectl top pods -l app=redis -n nephoran-system
```

4. **Redis Health Check**:
```bash
# Test Redis connectivity and performance
kubectl exec -it redis-0 -n nephoran-system -- redis-cli info | grep -E "used_memory|keyspace|expires"
kubectl exec -it redis-0 -n nephoran-system -- redis-cli ping
```

#### **Resolution Steps**

**Immediate Cache Optimization**:

1. **Clear and Warm Cache Strategically**:
```bash
# Clear low-value cache entries
curl -X DELETE http://llm-processor:8080/cache/clear?prefix=temporary:

# Warm cache with popular queries
curl -X POST http://llm-processor:8080/cache/warm \
  -H "Content-Type: application/json" \
  -d '{
    "strategy": "popular_queries",
    "limit": 1000,
    "ttl": "3600s"
  }'
```

2. **Optimize Cache Configuration**:
```bash
# Increase cache sizes if memory allows
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "L1_CACHE_MAX_SIZE": "2000",
    "L1_TTL": "30m",
    "L2_TTL": "2h",
    "COMPRESSION_ENABLED": "true"
  }}'
```

3. **Fix Redis Configuration Issues**:
```bash
# Optimize Redis memory policy
kubectl exec -it redis-0 -n nephoran-system -- redis-cli config set maxmemory-policy allkeys-lru

# Increase Redis memory limit if needed
kubectl patch statefulset redis -n nephoran-system \
  --type json -p='[{
    "op": "replace",
    "path": "/spec/template/spec/containers/0/resources/limits/memory",
    "value": "2Gi"
  }]'
```

**Cache Strategy Improvements**:

4. **Implement Intelligent Cache Warming**:
```yaml
# CronJob for proactive cache warming
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cache-warmer
  namespace: nephoran-system
spec:
  schedule: "0 */4 * * *"  # Every 4 hours
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cache-warmer
            image: nephoran/cache-warmer:latest
            command:
            - /bin/sh
            - -c
            - |
              # Warm cache with frequently accessed patterns
              curl -X POST http://llm-processor:8080/cache/warm \
                -H "Content-Type: application/json" \
                -d '{
                  "strategy": "predictive",
                  "analysis_window": "24h",
                  "limit": 500
                }'
          restartPolicy: OnFailure
```

5. **Optimize Cache Key Strategy**:
```bash
# Enable semantic cache key generation for better hit rates
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_SEMANTIC_KEYS": "true",
    "KEY_NORMALIZATION": "true",
    "ENABLE_QUERY_SIMILARITY": "true"
  }}'
```

6. **Implement Cache Hierarchy Optimization**:
```bash
# Configure cache promotion strategies
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "CACHE_PROMOTION_THRESHOLD": "3",
    "AUTO_PROMOTE_HOT_KEYS": "true",
    "L1_PROMOTION_STRATEGY": "access_frequency"
  }}'
```

**Redis Performance Tuning**:

7. **Optimize Redis Persistence**:
```bash
# Reduce Redis persistence overhead for better performance
kubectl exec -it redis-0 -n nephoran-system -- redis-cli config set save "900 1 300 10"

# Enable Redis pipelining for better throughput
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "REDIS_PIPELINE_SIZE": "100",
    "REDIS_POOL_SIZE": "20"
  }}'
```

8. **Implement Redis Clustering (for high-scale deployments)**:
```yaml
# Redis Cluster for horizontal scaling
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-cluster-config
data:
  redis.conf: |
    cluster-enabled yes
    cluster-config-file nodes.conf
    cluster-node-timeout 5000
    appendonly yes
    maxmemory 1gb
    maxmemory-policy allkeys-lru
```

#### **Cache Monitoring and Alerting**

```yaml
# Prometheus Alert Rules for Cache Performance
- alert: CacheHitRateLow
  expr: cache_hit_rate < 0.6
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Cache hit rate is below 60%"
    description: "Overall cache hit rate is {{ $value | humanizePercentage }}"
    runbook_url: "https://docs.nephoran.com/runbooks/cache-performance"

- alert: CacheMemoryPressure
  expr: (cache_memory_used / cache_memory_limit) > 0.9
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Cache memory usage is critically high"

- alert: RedisConnectionErrors
  expr: increase(redis_connection_errors_total[5m]) > 10
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "High rate of Redis connection errors"
```

#### **Prevention Strategies**

- **Regular Cache Analysis**: Weekly analysis of cache patterns and optimization opportunities
- **Capacity Planning**: Monitor cache growth trends and plan capacity increases
- **Key Design Guidelines**: Establish standards for cache key design and TTL values
- **Performance Testing**: Include cache performance in regular load testing

---

## Circuit Breaker Failures

### 🚨 **Symptom: Circuit Breaker Stuck in Open State**

**Error Code**: `CIRCUIT_BREAKER_OPEN`

**Symptoms**:
- Circuit breaker not transitioning to half-open state
- All requests being rejected immediately
- Downstream services appear healthy but circuit remains open
- High rate of circuit breaker rejection errors

#### **Diagnostic Steps**

1. **Check Circuit Breaker Status**:
```bash
# Get detailed circuit breaker status
curl -X GET http://llm-processor:8080/circuit-breaker/status | jq 'to_entries[] | {
  name: .key,
  state: .value.state,
  failure_rate: .value.failure_rate,
  last_failure: .value.last_failure_time,
  uptime: .value.uptime
}'
```

2. **Analyze Failure Patterns**:
```bash
# Check recent failure logs
kubectl logs -l app=llm-processor -n nephoran-system --since=1h | \
  grep -E "circuit.*open|failure.*threshold" | tail -20
```

3. **Test Downstream Service Health**:
```bash
# Test downstream services directly
curl -f http://rag-api:8080/health
curl -f http://weaviate:8080/v1/.well-known/ready

# Check service response times
time curl -s http://rag-api:8080/health > /dev/null
```

4. **Review Circuit Configuration**:
```bash
# Check current circuit breaker configuration
curl -X GET http://llm-processor:8080/circuit-breaker/config | jq '.'
```

#### **Resolution Steps**

**Immediate Recovery Actions**:

1. **Manual Circuit Breaker Reset**:
```bash
# Force reset all circuit breakers
curl -X POST http://llm-processor:8080/circuit-breaker/status \
  -H "Content-Type: application/json" \
  -d '{"action": "reset"}'

# Reset specific circuit breaker
curl -X POST http://llm-processor:8080/circuit-breaker/status \
  -H "Content-Type: application/json" \
  -d '{"action": "reset", "name": "rag-api"}'
```

2. **Verify Service Recovery**:
```bash
# Test circuit breaker functionality after reset
for i in {1..10}; do
  echo "Test $i:"
  curl -s -w "%{http_code} - %{time_total}s\n" \
    http://llm-processor:8080/process \
    -H "Content-Type: application/json" \
    -d '{"intent": "test circuit breaker recovery"}' \
    -o /dev/null
  sleep 2
done
```

3. **Adjust Circuit Breaker Sensitivity**:
```bash
# Temporarily reduce circuit breaker sensitivity
curl -X POST http://llm-processor:8080/circuit-breaker/configure \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rag-api",
    "failure_threshold": 10,
    "failure_rate": 0.7,
    "reset_timeout": "30s"
  }'
```

**Root Cause Analysis and Fixes**:

4. **Identify Underlying Service Issues**:
```bash
# Check downstream service resource utilization
kubectl top pods -l app=rag-api -n nephoran-system
kubectl describe pod -l app=rag-api -n nephoran-system | grep -A 10 Events

# Check for service mesh issues (if using Istio)
kubectl get virtualservices,destinationrules -n nephoran-system
```

5. **Fix Network Connectivity Issues**:
```bash
# Test pod-to-pod connectivity
kubectl exec -it deployment/llm-processor -n nephoran-system -- \
  nc -zv rag-api 8080

# Check DNS resolution
kubectl exec -it deployment/llm-processor -n nephoran-system -- \
  nslookup rag-api.nephoran-system.svc.cluster.local
```

6. **Optimize Service Performance**:
```bash
# Scale up struggling downstream services
kubectl scale deployment rag-api --replicas=3 -n nephoran-system

# Increase resource limits for downstream services
kubectl patch deployment rag-api -n nephoran-system \
  --type json -p='[{
    "op": "replace",
    "path": "/spec/template/spec/containers/0/resources/limits/memory",
    "value": "2Gi"
  }]'
```

**Configuration Optimization**:

7. **Implement Adaptive Circuit Breaker Configuration**:
```yaml
# ConfigMap with environment-specific circuit breaker settings
apiVersion: v1
kind: ConfigMap
metadata:
  name: circuit-breaker-config
  namespace: nephoran-system
data:
  development.json: |
    {
      "failure_threshold": 3,
      "failure_rate": 0.3,
      "reset_timeout": "15s",
      "success_threshold": 2
    }
  production.json: |
    {
      "failure_threshold": 5,
      "failure_rate": 0.5,
      "reset_timeout": "60s",
      "success_threshold": 3,
      "enable_health_check": true,
      "health_check_interval": "30s"
    }
```

8. **Enable Circuit Breaker Health Checks**:
```bash
# Configure automatic health checking for circuit recovery
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_CIRCUIT_HEALTH_CHECK": "true",
    "HEALTH_CHECK_INTERVAL": "30s",
    "HEALTH_CHECK_TIMEOUT": "10s",
    "AUTO_RECOVERY_ENABLED": "true"
  }}'
```

**Advanced Recovery Mechanisms**:

9. **Implement Fallback Mechanisms**:
```bash
# Enable fallback processing for critical services
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_FALLBACK_PROCESSING": "true",
    "FALLBACK_CACHE_ENABLED": "true",
    "FALLBACK_MODEL": "local",
    "DEGRADED_MODE_ENABLED": "true"
  }}'
```

10. **Circuit Breaker State Persistence**:
```bash
# Enable circuit breaker state persistence across restarts
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "CIRCUIT_STATE_PERSISTENCE": "true",
    "STATE_STORAGE_BACKEND": "redis",
    "STATE_SYNC_INTERVAL": "10s"
  }}'
```

#### **Advanced Troubleshooting**

**Circuit Breaker Debugging Mode**:
```bash
# Enable detailed circuit breaker logging
kubectl patch deployment llm-processor -n nephoran-system \
  --type json -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {"name": "CIRCUIT_BREAKER_DEBUG", "value": "true"}
  }]'

# Monitor circuit breaker decision making
kubectl logs -f deployment/llm-processor -n nephoran-system | \
  grep -E "circuit.*decision|breaker.*state"
```

**Performance Impact Analysis**:
```bash
# Measure circuit breaker overhead
curl -X POST http://llm-processor:8080/performance/profile \
  -H "Content-Type: application/json" \
  -d '{
    "component": "circuit_breaker",
    "duration": "60s",
    "include_metrics": ["latency", "cpu", "memory"]
  }'
```

#### **Prevention Strategies**

- **Proactive Health Monitoring**: Implement comprehensive health checks for all downstream services
- **Graceful Degradation**: Design fallback mechanisms for when circuit breakers open
- **Capacity Planning**: Ensure downstream services can handle expected load
- **Chaos Engineering**: Regularly test circuit breaker behavior under failure conditions
- **Configuration Management**: Use infrastructure-as-code for circuit breaker configurations

---

## Performance Optimization Issues

### 🚨 **Symptom: Performance Optimizer Not Triggering**

**Error Code**: `PERFORMANCE_OPTIMIZATION_DISABLED`

**Symptoms**:
- High resource utilization but no automatic optimization
- Performance metrics degrading over time
- Manual optimization requests timing out
- System performance below baseline thresholds

#### **Diagnostic Steps**

1. **Check Performance Optimizer Status**:
```bash
# Verify optimizer is enabled and running
curl -X GET http://llm-processor:8080/performance/optimizer/status | jq '{
  enabled: .enabled,
  last_run: .last_optimization,
  active_tasks: .active_optimizations,
  health_score: .health_score
}'
```

2. **Analyze Current Performance Metrics**:
```bash
# Get comprehensive performance metrics
curl -X GET http://llm-processor:8080/performance/metrics | jq '{
  cpu_usage: .cpu_usage,
  memory_usage: .memory_usage,
  latency: .average_latency,
  throughput: .throughput_rpm,
  cache_hit_rate: .cache_hit_rate
}'
```

3. **Check Optimization Thresholds**:
```bash
# Review current optimization configuration
curl -X GET http://llm-processor:8080/performance/optimizer/config | jq '{
  thresholds: .thresholds,
  enabled_optimizations: .enabled_optimizations,
  interval: .optimization_interval
}'
```

4. **Review Optimization History**:
```bash
# Check recent optimization attempts
kubectl logs -l app=llm-processor -n nephoran-system | \
  grep -E "optimization.*start|optimization.*complete|optimization.*failed" | tail -10
```

#### **Resolution Steps**

**Immediate Optimizer Activation**:

1. **Enable Performance Optimizer**:
```bash
# Ensure optimizer is enabled
curl -X POST http://llm-processor:8080/performance/optimizer/enable \
  -H "Content-Type: application/json" \
  -d '{"enable": true, "auto_optimization": true}'
```

2. **Trigger Manual Optimization**:
```bash
# Force immediate optimization for critical metrics
curl -X POST http://llm-processor:8080/performance/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "target_metrics": ["memory_usage", "cpu_usage", "latency"],
    "force_optimization": true
  }'
```

3. **Adjust Optimization Thresholds**:
```bash
# Lower thresholds to trigger optimization sooner
curl -X POST http://llm-processor:8080/performance/optimizer/configure \
  -H "Content-Type: application/json" \
  -d '{
    "memory_threshold": 0.7,
    "cpu_threshold": 0.6,
    "latency_threshold": "3s",
    "optimization_interval": "5m"
  }'
```

**Configuration Fixes**:

4. **Update Optimizer Configuration**:
```bash
# Enable all optimization types
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_AUTO_OPTIMIZATION": "true",
    "ENABLE_MEMORY_OPTIMIZATION": "true",
    "ENABLE_CACHE_OPTIMIZATION": "true",
    "ENABLE_BATCH_OPTIMIZATION": "true",
    "OPTIMIZATION_INTERVAL": "10m"
  }}'
```

5. **Fix Optimizer Service Dependencies**:
```bash
# Ensure optimizer can access required services
kubectl exec -it deployment/llm-processor -n nephoran-system -- \
  curl -f http://rag-api:8080/health

# Check Redis connectivity for optimization state storage
kubectl exec -it deployment/llm-processor -n nephoran-system -- \
  ping redis.nephoran-system.svc.cluster.local
```

6. **Restart Optimizer Service**:
```bash
# Restart pods to reinitialize optimizer
kubectl rollout restart deployment/llm-processor -n nephoran-system

# Wait for optimizer to initialize
sleep 60
curl -X GET http://llm-processor:8080/performance/optimizer/status
```

**Advanced Optimization Configuration**:

7. **Implement Custom Optimization Rules**:
```yaml
# ConfigMap with custom optimization rules
apiVersion: v1
kind: ConfigMap
metadata:
  name: performance-optimization-rules
  namespace: nephoran-system
data:
  rules.json: |
    {
      "rules": [
        {
          "name": "high_memory_usage",
          "condition": "memory_usage > 0.8",
          "actions": ["garbage_collection", "cache_cleanup", "reduce_batch_size"],
          "priority": "high"
        },
        {
          "name": "high_latency",
          "condition": "average_latency > 5s",
          "actions": ["connection_pooling", "cache_warming", "timeout_reduction"],
          "priority": "critical"
        },
        {
          "name": "low_cache_efficiency",
          "condition": "cache_hit_rate < 0.7",
          "actions": ["cache_warming", "ttl_optimization", "key_optimization"],
          "priority": "medium"
        }
      ]
    }
```

8. **Enable ML-Based Optimization**:
```bash
# Enable machine learning-based optimization predictions
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_ML_OPTIMIZATION": "true",
    "ML_MODEL_PATH": "/models/performance-optimizer.pkl",
    "PREDICTION_WINDOW": "1h",
    "LEARNING_RATE": "0.01"
  }}'
```

**Resource Allocation Optimization**:

9. **Dynamic Resource Adjustment**:
```bash
# Enable dynamic resource adjustment based on performance
kubectl patch deployment llm-processor -n nephoran-system \
  --type json -p='[{
    "op": "add",
    "path": "/spec/template/metadata/annotations",
    "value": {
      "prometheus.io/scrape": "true",
      "prometheus.io/port": "8080",
      "prometheus.io/path": "/metrics"
    }
  }]'

# Add VPA for automatic resource recommendations
kubectl apply -f - <<EOF
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: llm-processor-vpa
  namespace: nephoran-system
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: llm-processor
      maxAllowed:
        cpu: 4
        memory: 8Gi
      minAllowed:
        cpu: 500m
        memory: 1Gi
EOF
```

10. **Implement Performance-Based Scaling**:
```yaml
# HPA based on custom performance metrics
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-processor-performance-hpa
  namespace: nephoran-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Pods
    pods:
      metric:
        name: performance_score
      target:
        type: AverageValue
        averageValue: "0.7"  # Scale up when performance score drops below 0.7
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 120
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
```

#### **Monitoring and Validation**

```bash
# Continuous monitoring of optimization effectiveness
watch -n 30 "curl -s http://llm-processor:8080/performance/report | jq '{
  health_score: .health_score,
  active_optimizations: (.active_optimizations | length),
  recommendations: (.recommendations | length),
  last_optimization: .current_metrics.last_optimization
}'"

# Performance trend analysis
curl -X GET http://llm-processor:8080/performance/trends?window=24h | \
  jq '.trends | {
    latency_trend: .latency,
    memory_trend: .memory_usage,
    optimization_impact: .performance_gains
  }'
```

#### **Prevention Strategies**

- **Proactive Monitoring**: Set up alerts for performance degradation before optimization thresholds are reached
- **Regular Performance Audits**: Weekly reviews of performance metrics and optimization effectiveness
- **Capacity Planning**: Use performance trends to predict and prevent resource bottlenecks
- **Configuration Validation**: Implement automated testing of optimization configurations
- **Performance Baselines**: Maintain and update performance baselines as system evolves

---

## Intent Processing Errors

### 🚨 **Symptom: High Rate of Intent Processing Failures**

**Error Code**: `INTENT_PROCESSING_FAILED`

**Symptoms**:
- Intents failing with validation errors
- Structured output generation failures
- LLM API timeouts or rate limiting
- Inconsistent response quality

#### **Diagnostic Steps**

1. **Analyze Error Patterns**:
```bash
# Get breakdown of processing errors
curl -X GET http://llm-processor:8080/metrics | \
  grep -E "intent.*error|processing.*failed" | \
  sort | uniq -c | sort -nr
```

2. **Check LLM API Health**:
```bash
# Test LLM API connectivity and performance
curl -X POST http://llm-processor:8080/llm/test \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o-mini", "prompt": "test", "max_tokens": 10}'
```

3. **Validate RAG Pipeline**:
```bash
# Test RAG API functionality
curl -X POST http://rag-api:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "test network deployment", "limit": 5}'
```

4. **Review Recent Failed Intents**:
```bash
# Get details of recent failures
kubectl logs -l app=llm-processor -n nephoran-system | \
  grep -E "INTENT_PROCESSING_FAILED|validation.*error" | tail -20
```

#### **Resolution Steps**

**Immediate Error Mitigation**:

1. **Enable Fallback Processing**:
```bash
# Enable degraded mode processing for critical intents
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_FALLBACK_MODE": "true",
    "FALLBACK_MODEL": "local",
    "SIMPLIFIED_OUTPUT_MODE": "true"
  }}'
```

2. **Adjust Processing Timeouts**:
```bash
# Increase timeouts to handle high-latency periods
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "LLM_REQUEST_TIMEOUT": "60s",
    "RAG_QUERY_TIMEOUT": "30s",
    "TOTAL_PROCESSING_TIMEOUT": "120s"
  }}'
```

3. **Implement Retry Logic with Backoff**:
```bash
# Configure intelligent retry behavior
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "MAX_RETRIES": "3",
    "RETRY_BACKOFF_BASE": "2s",
    "RETRY_BACKOFF_MAX": "30s",
    "ENABLE_EXPONENTIAL_BACKOFF": "true"
  }}'
```

**Input Validation and Preprocessing**:

4. **Enhance Input Validation**:
```bash
# Enable comprehensive input validation
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_STRICT_VALIDATION": "true",
    "MIN_INTENT_LENGTH": "10",
    "MAX_INTENT_LENGTH": "2000",
    "SANITIZE_INPUT": "true",
    "VALIDATE_INTENT_STRUCTURE": "true"
  }}'
```

5. **Improve Intent Preprocessing**:
```bash
# Enable intent preprocessing and normalization
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_INTENT_NORMALIZATION": "true",
    "EXPAND_ABBREVIATIONS": "true",
    "CONTEXT_ENRICHMENT": "true",
    "SPELL_CHECK_ENABLED": "true"
  }}'
```

**LLM API Optimization**:

6. **Implement Rate Limiting Protection**:
```bash
# Configure rate limiting and quota management
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_RATE_LIMITING": "true",
    "REQUESTS_PER_MINUTE": "100",
    "TOKENS_PER_MINUTE": "10000",
    "ENABLE_QUOTA_MANAGEMENT": "true",
    "DAILY_TOKEN_LIMIT": "100000"
  }}'
```

7. **Optimize Prompt Engineering**:
```bash
# Enable advanced prompt optimization
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_PROMPT_OPTIMIZATION": "true",
    "USE_CONTEXT_AWARE_PROMPTS": "true",
    "DYNAMIC_PROMPT_SELECTION": "true",
    "PROMPT_TEMPLATE_VERSION": "v2.1"
  }}'
```

**Response Quality Improvement**:

8. **Implement Response Validation**:
```bash
# Enable comprehensive response validation
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_RESPONSE_VALIDATION": "true",
    "VALIDATE_JSON_STRUCTURE": "true",
    "CONFIDENCE_THRESHOLD": "0.7",
    "ENABLE_RESPONSE_SANITIZATION": "true"
  }}'
```

9. **Quality Scoring and Filtering**:
```yaml
# ConfigMap with quality scoring rules
apiVersion: v1
kind: ConfigMap
metadata:
  name: response-quality-config
  namespace: nephoran-system
data:
  quality_rules.json: |
    {
      "scoring_rules": [
        {
          "name": "structure_completeness",
          "weight": 0.3,
          "criteria": ["required_fields_present", "valid_json_format"]
        },
        {
          "name": "content_relevance",
          "weight": 0.4,
          "criteria": ["intent_alignment", "technical_accuracy"]
        },
        {
          "name": "response_clarity",
          "weight": 0.3,
          "criteria": ["parameter_specificity", "deployment_feasibility"]
        }
      ],
      "minimum_score": 0.7,
      "enable_auto_retry": true,
      "max_quality_retries": 2
    }
```

**Error Recovery and Learning**:

10. **Implement Error Pattern Learning**:
```bash
# Enable machine learning from processing errors
kubectl patch configmap llm-processor-config -n nephoran-system \
  --type merge -p='{"data":{
    "ENABLE_ERROR_LEARNING": "true",
    "ERROR_PATTERN_ANALYSIS": "true",
    "AUTO_PROMPT_ADJUSTMENT": "true",
    "FEEDBACK_LOOP_ENABLED": "true"
  }}'
```

#### **Advanced Troubleshooting Techniques**

**Intent Processing Pipeline Debug Mode**:
```bash
# Enable detailed processing pipeline logging
kubectl patch deployment llm-processor -n nephoran-system \
  --type json -p='[{
    "op": "add",
    "path": "/spec/template/spec/containers/0/env/-",
    "value": {"name": "DEBUG_PIPELINE", "value": "true"}
  }]'

# Monitor detailed processing flow
kubectl logs -f deployment/llm-processor -n nephoran-system | \
  grep -E "pipeline.*step|processing.*stage|validation.*check"
```

**Response Quality Analysis**:
```bash
# Analyze response quality trends
curl -X GET http://llm-processor:8080/analytics/quality-trends?window=24h | \
  jq '{
    average_confidence: .average_confidence,
    validation_failures: .validation_failure_rate,
    retry_rate: .retry_rate,
    quality_distribution: .quality_score_distribution
  }'
```

#### **Prevention Strategies**

- **Input Quality Gates**: Implement comprehensive input validation and preprocessing
- **Response Quality Monitoring**: Continuous monitoring of output quality with automated feedback
- **Prompt Engineering**: Regular review and optimization of prompt templates
- **Model Performance Tracking**: Monitor LLM model performance and consider model updates
- **User Training**: Provide guidelines for effective intent formulation

---

This comprehensive troubleshooting guide provides detailed procedures for diagnosing and resolving the most common issues with the enhanced Nephoran Intent Operator. Each runbook includes immediate actions, root cause analysis, and prevention strategies to maintain optimal system performance.

For additional support or complex issues not covered in these runbooks, please refer to the system logs, performance metrics, and contact the Nephoran support team with relevant diagnostic information.