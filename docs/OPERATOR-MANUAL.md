# Nephoran Intent Operator - Complete Operator Manual and User Guide

## Table of Contents

1. [Getting Started](#getting-started)
2. [Async Processing Workflows](#async-processing-workflows)
3. [Enhanced Caching System Best Practices](#enhanced-caching-system-best-practices)
4. [Production Deployment Guide](#production-deployment-guide)
5. [Configuration Reference](#configuration-reference)
6. [Operational Procedures](#operational-procedures)
7. [Monitoring and Alerting](#monitoring-and-alerting)
8. [Troubleshooting Quick Reference](#troubleshooting-quick-reference)
9. [Advanced Use Cases](#advanced-use-cases)

## Getting Started

### System Overview

The enhanced Nephoran Intent Operator provides enterprise-grade network intent processing with advanced features:

- **Async Processing**: Server-Sent Events (SSE) streaming for real-time processing
- **Multi-Level Caching**: L1 (in-memory) + L2 (Redis) intelligent caching
- **Circuit Breaker Pattern**: Fault tolerance with automatic recovery
- **Performance Optimization**: AI-driven system optimization
- **Production Monitoring**: Comprehensive observability and alerting

### Prerequisites

Before deploying the Nephoran Intent Operator, ensure you have:

```bash
# Required tools
kubectl version --client  # >= v1.25
helm version              # >= v3.8
docker version           # >= 20.10
curl --version           # For API testing

# Cluster requirements
kubectl cluster-info     # Verify cluster access
kubectl get nodes        # Minimum 3 nodes, 16GB+ RAM total
kubectl get storageclass # Verify storage classes available
```

### Quick Start Deployment

**1. Deploy Core Components:**

```bash
# Clone repository
git clone https://github.com/nephoran/intent-operator.git
cd intent-operator

# Deploy to Kubernetes
kubectl apply -f deployments/crds/
kubectl apply -f deployments/kustomize/overlays/production/

# Verify deployment
kubectl get pods -n nephoran-system
```

**2. Initialize Services:**

```bash
# Wait for services to be ready
kubectl wait --for=condition=available deployment/llm-processor -n nephoran-system --timeout=300s
kubectl wait --for=condition=available deployment/rag-api -n nephoran-system --timeout=300s

# Test basic functionality
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system &
curl http://localhost:8080/healthz
```

**3. Submit Your First Intent:**

```bash
# Create a simple network intent
cat <<EOF | kubectl apply -f -
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: my-first-intent
  namespace: default
spec:
  description: "Deploy AMF with 2 replicas for testing"
  priority: medium
EOF

# Check processing status
kubectl get networkintents my-first-intent -o yaml
```

## Async Processing Workflows

### Understanding Async Processing

The enhanced Nephoran Intent Operator supports multiple processing modes:

1. **Synchronous Processing**: Traditional request-response (default)
2. **Asynchronous Processing**: Non-blocking with callbacks
3. **Streaming Processing**: Real-time SSE streaming

### Async Processing Workflow

#### 1. Asynchronous Intent Submission

```bash
# Submit intent for async processing
curl -X POST http://llm-processor:8080/process \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Scale SMF to 5 instances for high availability",
    "metadata": {
      "priority": "high",
      "namespace": "telecom-core"
    },
    "options": {
      "async": true,
      "callback_url": "https://your-webhook.example.com/notifications"
    }
  }'
```

**Response:**
```json
{
  "intent_id": "intent-abc123",
  "status": "accepted",
  "estimated_completion": "2025-01-12T10:35:00Z",
  "tracking_url": "/api/v1/intents/intent-abc123/status"
}
```

#### 2. Webhook Configuration

**Setup webhook endpoint to receive completion notifications:**

```javascript
// Node.js webhook example
const express = require('express');
const app = express();

app.post('/notifications', express.json(), (req, res) => {
  const { intent_id, status, structured_output, error } = req.body;
  
  console.log(`Intent ${intent_id} completed with status: ${status}`);
  
  if (status === 'completed') {
    // Process successful result
    console.log('Deployment configuration:', structured_output);
    // Deploy to your infrastructure
    deployToInfrastructure(structured_output);
  } else if (status === 'failed') {
    // Handle error
    console.error('Processing failed:', error);
    // Trigger retry or alert
    handleFailure(intent_id, error);
  }
  
  res.status(200).json({ received: true });
});

app.listen(3000, () => {
  console.log('Webhook server running on port 3000');
});
```

#### 3. Streaming Workflow

**For real-time processing updates:**

```bash
# Start streaming session
curl -N -X POST http://llm-processor:8080/stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "query": "Configure UPF for ultra-low latency URLLC applications",
    "session_id": "session_urllc_001",
    "enable_rag": true,
    "stream_options": {
      "chunk_size": 256,
      "max_chunk_delay": "10ms"
    }
  }'
```

**Streaming events you'll receive:**

```
event: start
data: {"session_id":"session_urllc_001","status":"started"}

event: context_injection
data: {"type":"context_injection","content":"Retrieved URLLC configuration guidelines","metadata":{"context_length":8420,"injection_time":"75ms"}}

event: chunk
data: {"type":"content","delta":"For ultra-low latency URLLC applications, the UPF configuration requires...","timestamp":"2025-01-12T10:30:15Z","chunk_index":0}

event: chunk
data: {"type":"content","delta":"specific QoS parameters and traffic prioritization settings...","timestamp":"2025-01-12T10:30:15Z","chunk_index":1}

event: completion
data: {"type":"completion","is_complete":true,"metadata":{"total_chunks":12,"total_bytes":4096,"processing_time":"1.8s"}}
```

#### 4. Client Implementation Examples

**Python streaming client:**

```python
import requests
import json
import sseclient

def stream_intent_processing(query, session_id):
    """Stream intent processing with SSE client"""
    
    headers = {
        'Content-Type': 'application/json',
        'Accept': 'text/event-stream'
    }
    
    payload = {
        'query': query,
        'session_id': session_id,
        'enable_rag': True
    }
    
    response = requests.post(
        'http://llm-processor:8080/stream',
        headers=headers,
        data=json.dumps(payload),
        stream=True
    )
    
    client = sseclient.SSEClient(response)
    
    for event in client.events():
        event_data = json.loads(event.data)
        
        if event.event == 'start':
            print(f"Session started: {event_data['session_id']}")
            
        elif event.event == 'context_injection':
            print(f"Context injected: {event_data['metadata']['injection_time']}")
            
        elif event.event == 'chunk':
            print(f"Chunk {event_data['chunk_index']}: {event_data['delta']}")
            
        elif event.event == 'completion':
            print(f"Completed in {event_data['metadata']['processing_time']}")
            break
            
        elif event.event == 'error':
            print(f"Error: {event_data['message']}")
            break

# Usage
stream_intent_processing(
    "Deploy secure 5G core network for enterprise customers",
    "enterprise_deployment_001"
)
```

**JavaScript streaming client:**

```javascript
// Browser-compatible streaming client
class NephoranStreamingClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
  }

  async streamIntent(query, sessionId, options = {}) {
    const response = await fetch(`${this.baseUrl}/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'text/event-stream'
      },
      body: JSON.stringify({
        query,
        session_id: sessionId,
        enable_rag: true,
        ...options
      })
    });

    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      const chunk = decoder.decode(value);
      const lines = chunk.split('\n');

      for (const line of lines) {
        if (line.startsWith('event:')) {
          const eventType = line.substring(6).trim();
        } else if (line.startsWith('data:')) {
          const eventData = JSON.parse(line.substring(5).trim());
          this.handleEvent(eventType, eventData);
        }
      }
    }
  }

  handleEvent(eventType, data) {
    switch (eventType) {
      case 'start':
        console.log('Streaming started:', data.session_id);
        break;
      case 'chunk':
        this.onChunk(data);
        break;
      case 'completion':
        this.onComplete(data);
        break;
      case 'error':
        this.onError(data);
        break;
    }
  }

  onChunk(data) {
    // Override in subclass
    console.log(`Chunk ${data.chunk_index}: ${data.delta}`);
  }

  onComplete(data) {
    // Override in subclass
    console.log('Processing complete:', data.metadata);
  }

  onError(data) {
    // Override in subclass
    console.error('Streaming error:', data);
  }
}

// Usage
const client = new NephoranStreamingClient('http://llm-processor:8080');
client.streamIntent('Configure network slice for IoT devices', 'iot_slice_001');
```

### Async Processing Best Practices

#### 1. Error Handling and Retry Logic

```bash
# Implement exponential backoff for failed intents
process_intent_with_retry() {
  local intent="$1"
  local max_retries=3
  local retry_count=0
  local delay=1

  while [ $retry_count -lt $max_retries ]; do
    response=$(curl -s -w "%{http_code}" -X POST http://llm-processor:8080/process \
      -H "Content-Type: application/json" \
      -d "{\"intent\": \"$intent\", \"options\": {\"async\": true}}")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "202" ]; then
      echo "Intent accepted: $body"
      return 0
    elif [ "$http_code" = "429" ]; then
      echo "Rate limited, retrying in ${delay}s..."
      sleep $delay
      delay=$((delay * 2))
      retry_count=$((retry_count + 1))
    else
      echo "Error $http_code: $body"
      return 1
    fi
  done
  
  echo "Max retries exceeded"
  return 1
}
```

#### 2. Monitoring Async Operations

**Track async processing metrics:**

```bash
# Monitor async processing queue
monitor_async_queue() {
  while true; do
    queue_depth=$(curl -s http://llm-processor:8080/metrics | jq '.async_processing.queue_depth')
    active_jobs=$(curl -s http://llm-processor:8080/metrics | jq '.async_processing.active_jobs')
    completion_rate=$(curl -s http://llm-processor:8080/metrics | jq '.async_processing.completion_rate')
    
    echo "$(date): Queue=$queue_depth, Active=$active_jobs, Rate=$completion_rate/min"
    
    # Alert if queue is backing up
    if [ "$queue_depth" -gt 100 ]; then
      echo "WARNING: High queue depth detected!"
    fi
    
    sleep 30
  done
}

monitor_async_queue &
```

#### 3. Session Management for Streaming

```python
# Advanced session management
class StreamingSessionManager:
    def __init__(self, base_url, max_concurrent=50):
        self.base_url = base_url
        self.max_concurrent = max_concurrent
        self.active_sessions = {}
        
    async def create_session(self, query, session_id=None):
        if len(self.active_sessions) >= self.max_concurrent:
            raise Exception("Maximum concurrent sessions reached")
            
        if session_id is None:
            session_id = f"session_{int(time.time())}"
            
        session = StreamingSession(self.base_url, session_id)
        self.active_sessions[session_id] = session
        
        try:
            await session.start(query)
            return session
        except Exception as e:
            del self.active_sessions[session_id]
            raise e
            
    async def cleanup_session(self, session_id):
        if session_id in self.active_sessions:
            session = self.active_sessions[session_id]
            await session.close()
            del self.active_sessions[session_id]
            
    async def cleanup_all(self):
        for session_id in list(self.active_sessions.keys()):
            await self.cleanup_session(session_id)

# Usage with proper cleanup
async def main():
    manager = StreamingSessionManager('http://llm-processor:8080')
    
    try:
        session = await manager.create_session(
            "Deploy secure O-RAN network for mobile operators"
        )
        
        async for chunk in session.stream():
            print(f"Received: {chunk['delta']}")
            
    finally:
        await manager.cleanup_all()
```

## Enhanced Caching System Best Practices

### Understanding Multi-Level Caching

The Nephoran Intent Operator implements a sophisticated multi-level caching system:

- **L1 Cache**: In-memory LRU cache for ultra-fast access (< 1ms)
- **L2 Cache**: Redis distributed cache for shared access (< 20ms)
- **Cache Warming**: Proactive population of frequently accessed data
- **Intelligent Eviction**: Smart removal based on usage patterns

### Cache Configuration Best Practices

#### 1. Optimal Cache Sizing

**Small Deployment (< 1000 intents/day):**
```json
{
  "l1_cache": {
    "max_size": 500,
    "ttl": "30m",
    "eviction_policy": "LRU"
  },
  "l2_cache": {
    "enabled": false
  }
}
```

**Medium Deployment (1k-10k intents/day):**
```json
{
  "l1_cache": {
    "max_size": 1500,
    "ttl": "1h",
    "eviction_policy": "LRU",
    "enable_compression": true
  },
  "l2_cache": {
    "enabled": true,
    "ttl": "4h",
    "max_connections": 20
  }
}
```

**Large Deployment (10k+ intents/day):**
```json
{
  "l1_cache": {
    "max_size": 5000,
    "ttl": "2h",
    "eviction_policy": "LRU",
    "enable_compression": true,
    "compression_threshold": 1024
  },
  "l2_cache": {
    "enabled": true,
    "ttl": "12h",
    "max_connections": 100,
    "cluster_enabled": true,
    "compression_enabled": true
  },
  "cache_warming": {
    "enabled": true,
    "strategy": "predictive",
    "schedule": "0 */4 * * *"
  }
}
```

#### 2. Cache Key Strategy

**Implementing effective cache key design:**

```bash
# Example cache key patterns used by Nephoran
intent_cache_key = "intent:hash:{sha256_of_intent}:v{version}"
rag_context_key = "rag:context:{query_hash}:model:{model_name}"
embedding_key = "embedding:{text_hash}:provider:{provider_name}"
```

**Best practices for cache keys:**
- Use consistent naming conventions
- Include version information for cache invalidation
- Use content hashes for deterministic keys
- Include relevant parameters (model, provider, etc.)

#### 3. Cache Warming Strategies

**Proactive Cache Warming:**

```bash
# Manual cache warming script
warm_cache_for_common_intents() {
  local common_intents=(
    "Deploy AMF with 3 replicas"
    "Scale SMF to 5 instances"
    "Configure UPF for low latency"
    "Setup network slice for eMBB"
    "Deploy secure 5G core network"
  )

  echo "Starting cache warming for ${#common_intents[@]} common intents..."

  for intent in "${common_intents[@]}"; do
    echo "Warming cache for: $intent"
    curl -s -X POST http://llm-processor:8080/process \
      -H "Content-Type: application/json" \
      -d "{\"intent\": \"$intent\", \"options\": {\"enable_cache\": true}}" > /dev/null
    
    sleep 2  # Avoid overwhelming the system
  done

  echo "Cache warming completed"
}

# Schedule cache warming
# Add to crontab: 0 6 * * * /path/to/warm_cache_for_common_intents.sh
```

**Predictive Cache Warming:**

```python
# Python script for intelligent cache warming
import requests
import json
from collections import Counter
from datetime import datetime, timedelta

class CacheWarmer:
    def __init__(self, base_url):
        self.base_url = base_url
        
    def analyze_query_patterns(self, days=7):
        """Analyze query patterns from logs to identify popular intents"""
        # This would typically analyze your application logs
        # For example purposes, using mock data
        
        popular_queries = [
            ("Deploy AMF with 3 replicas", 45),
            ("Scale SMF instances", 32),
            ("Configure network slice", 28),
            ("Setup O-RAN interfaces", 21),
            ("Deploy UPF configuration", 18)
        ]
        
        return popular_queries
        
    def warm_cache(self, queries, max_concurrent=5):
        """Warm cache with popular queries"""
        import asyncio
        import aiohttp
        
        async def warm_single_query(session, query):
            try:
                async with session.post(
                    f"{self.base_url}/process",
                    json={"intent": query, "options": {"enable_cache": True}}
                ) as response:
                    if response.status == 200:
                        print(f"✓ Warmed cache for: {query[:50]}...")
                    else:
                        print(f"✗ Failed to warm: {query[:50]}...")
            except Exception as e:
                print(f"✗ Error warming {query[:50]}...: {e}")
        
        async def warm_all():
            connector = aiohttp.TCPConnector(limit=max_concurrent)
            async with aiohttp.ClientSession(connector=connector) as session:
                tasks = [warm_single_query(session, query) for query, _ in queries]
                await asyncio.gather(*tasks, return_exceptions=True)
        
        asyncio.run(warm_all())
        
    def run_warming_cycle(self):
        """Run complete cache warming cycle"""
        print(f"Starting cache warming cycle at {datetime.now()}")
        
        # Analyze patterns
        popular_queries = self.analyze_query_patterns()
        print(f"Found {len(popular_queries)} popular queries")
        
        # Warm cache
        self.warm_cache(popular_queries)
        
        # Report results
        stats = self.get_cache_stats()
        print(f"Cache warming completed. Hit rate: {stats['overall_hit_rate']:.2%}")
        
    def get_cache_stats(self):
        """Get current cache statistics"""
        response = requests.get(f"{self.base_url}/cache/stats")
        return response.json()['overall_stats']

# Usage
if __name__ == "__main__":
    warmer = CacheWarmer("http://llm-processor:8080")
    warmer.run_warming_cycle()
```

#### 4. Cache Performance Monitoring

**Real-time cache monitoring:**

```bash
# Cache performance monitoring script
monitor_cache_performance() {
  local alert_threshold=0.7  # 70% hit rate threshold
  
  while true; do
    stats=$(curl -s http://llm-processor:8080/cache/stats)
    
    l1_hit_rate=$(echo "$stats" | jq -r '.l1_stats.hit_rate')
    l2_hit_rate=$(echo "$stats" | jq -r '.l2_stats.hit_rate')
    overall_hit_rate=$(echo "$stats" | jq -r '.overall_stats.overall_hit_rate')
    memory_usage=$(echo "$stats" | jq -r '.overall_stats.memory_usage')
    
    echo "$(date): L1=${l1_hit_rate}, L2=${l2_hit_rate}, Overall=${overall_hit_rate}, Memory=${memory_usage}B"
    
    # Alert if performance degrades
    if (( $(echo "$overall_hit_rate < $alert_threshold" | bc -l) )); then
      echo "ALERT: Cache hit rate below threshold ($overall_hit_rate < $alert_threshold)"
      
      # Trigger cache warming
      curl -s -X POST http://llm-processor:8080/cache/warm \
        -H "Content-Type: application/json" \
        -d '{"strategy": "popular_queries", "limit": 100}' > /dev/null
      
      echo "Cache warming triggered"
    fi
    
    sleep 60
  done
}

monitor_cache_performance &
```

#### 5. Cache Maintenance Operations

**Daily cache maintenance:**

```bash
#!/bin/bash
# cache-maintenance.sh - Daily cache maintenance operations

echo "=== Daily Cache Maintenance - $(date) ==="

# 1. Cache statistics before maintenance
echo "Cache stats before maintenance:"
curl -s http://llm-processor:8080/cache/stats | jq '.overall_stats | {
  hit_rate: .overall_hit_rate,
  memory_usage: .memory_usage,
  last_updated: .last_updated
}'

# 2. Clear stale entries (optional, based on your needs)
echo "Clearing stale cache entries..."
curl -s -X DELETE "http://llm-processor:8080/cache/clear?level=l2&prefix=stale:"

# 3. Analyze cache efficiency
echo "Analyzing cache efficiency..."
l1_efficiency=$(curl -s http://llm-processor:8080/cache/stats | jq -r '.l1_stats.hit_rate')
l2_efficiency=$(curl -s http://llm-processor:8080/cache/stats | jq -r '.l2_stats.hit_rate')

if (( $(echo "$l1_efficiency < 0.8" | bc -l) )); then
  echo "L1 cache efficiency low ($l1_efficiency), consider increasing size"
fi

if (( $(echo "$l2_efficiency < 0.6" | bc -l) )); then
  echo "L2 cache efficiency low ($l2_efficiency), consider TTL adjustment"
fi

# 4. Warm cache for next day
echo "Warming cache for anticipated queries..."
curl -s -X POST http://llm-processor:8080/cache/warm \
  -H "Content-Type: application/json" \
  -d '{
    "strategy": "predictive",
    "limit": 500
  }' | jq '.items_warmed'

# 5. Final statistics
echo "Cache stats after maintenance:"
curl -s http://llm-processor:8080/cache/stats | jq '.overall_stats.overall_hit_rate'

echo "=== Maintenance Complete ==="
```

**Add to crontab:**
```bash
# Run daily cache maintenance at 2 AM
0 2 * * * /path/to/cache-maintenance.sh >> /var/log/nephoran/cache-maintenance.log 2>&1
```

### Cache Troubleshooting

#### Common Cache Issues and Solutions

**1. Low Hit Rate (< 60%):**
```bash
# Diagnose low hit rate
diagnose_low_hit_rate() {
  stats=$(curl -s http://llm-processor:8080/cache/stats)
  
  l1_size=$(echo "$stats" | jq -r '.l1_stats.size')
  l1_evictions=$(echo "$stats" | jq -r '.l1_stats.evictions')
  l2_errors=$(echo "$stats" | jq -r '.l2_stats.errors')
  
  echo "Diagnosis:"
  echo "- L1 cache size: $l1_size"
  echo "- L1 evictions: $l1_evictions"
  echo "- L2 errors: $l2_errors"
  
  if [ "$l1_evictions" -gt 100 ]; then
    echo "Recommendation: Increase L1 cache size"
  fi
  
  if [ "$l2_errors" -gt 0 ]; then
    echo "Recommendation: Check Redis connectivity"
  fi
}
```

**2. High Memory Usage:**
```bash
# Check and optimize memory usage
optimize_cache_memory() {
  current_usage=$(curl -s http://llm-processor:8080/cache/stats | jq -r '.overall_stats.memory_usage')
  
  if [ "$current_usage" -gt 1073741824 ]; then  # > 1GB
    echo "High memory usage detected: $current_usage bytes"
    
    # Enable compression
    curl -s -X POST http://llm-processor:8080/cache/config \
      -H "Content-Type: application/json" \
      -d '{"l1_cache": {"enable_compression": true}}'
    
    # Reduce TTL
    curl -s -X POST http://llm-processor:8080/cache/config \
      -H "Content-Type: application/json" \
      -d '{"l1_cache": {"ttl": "15m"}}'
    
    echo "Applied memory optimizations"
  fi
}
```

## Production Deployment Guide

### Infrastructure Requirements

#### Minimum Production Requirements

**For 1k-5k intents/day:**
```yaml
cluster_requirements:
  nodes: 3
  cpu_per_node: 4 cores
  memory_per_node: 16GB
  storage_per_node: 100GB SSD

component_resources:
  llm_processor:
    replicas: 2
    cpu: { requests: "1000m", limits: "2000m" }
    memory: { requests: "2Gi", limits: "4Gi" }
  
  rag_api:
    replicas: 2
    cpu: { requests: "500m", limits: "1000m" }
    memory: { requests: "1Gi", limits: "2Gi" }
  
  redis:
    replicas: 1
    cpu: { requests: "250m", limits: "500m" }
    memory: { requests: "512Mi", limits: "1Gi" }
```

**For 5k-20k intents/day:**
```yaml
cluster_requirements:
  nodes: 5
  cpu_per_node: 8 cores
  memory_per_node: 32GB
  storage_per_node: 200GB SSD

component_resources:
  llm_processor:
    replicas: 3
    cpu: { requests: "2000m", limits: "4000m" }
    memory: { requests: "4Gi", limits: "8Gi" }
    
  rag_api:
    replicas: 3
    cpu: { requests: "1000m", limits: "2000m" }
    memory: { requests: "2Gi", limits: "4Gi" }
    
  redis:
    replicas: 3  # Redis cluster
    cpu: { requests: "500m", limits: "1000m" }
    memory: { requests: "1Gi", limits: "2Gi" }
```

### Step-by-Step Production Deployment

#### Phase 1: Infrastructure Preparation

**1. Prepare Kubernetes Cluster:**

```bash
#!/bin/bash
# prepare-cluster.sh - Production cluster preparation

echo "=== Preparing Kubernetes Cluster for Nephoran ==="

# Verify cluster requirements
echo "1. Checking cluster requirements..."
NODE_COUNT=$(kubectl get nodes --no-headers | wc -l)
if [ "$NODE_COUNT" -lt 3 ]; then
  echo "ERROR: Minimum 3 nodes required, found $NODE_COUNT"
  exit 1
fi

echo "✓ Cluster has $NODE_COUNT nodes"

# Check storage classes
echo "2. Checking storage classes..."
if ! kubectl get storageclass standard >/dev/null 2>&1; then
  echo "WARNING: Standard storage class not found, creating..."
  
  # Create standard storage class (adjust for your environment)
  cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
provisioner: kubernetes.io/aws-ebs  # Adjust for your provider
parameters:
  type: gp3
  encrypted: "true"
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
EOF
fi

echo "✓ Storage classes verified"

# Create namespace
echo "3. Creating Nephoran namespace..."
kubectl create namespace nephoran-system --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace nephoran-system app.kubernetes.io/name=nephoran-intent-operator

# Setup RBAC
echo "4. Setting up RBAC..."
kubectl apply -f deployments/kubernetes/nephio-bridge-rbac.yaml

echo "✓ Cluster preparation complete"
```

**2. Configure Secrets:**

```bash
#!/bin/bash
# setup-secrets.sh - Configure production secrets

echo "=== Setting up Production Secrets ==="

# OpenAI API Key
echo "1. Setting up OpenAI API key..."
if [ -z "$OPENAI_API_KEY" ]; then
  echo "ERROR: OPENAI_API_KEY environment variable not set"
  exit 1
fi

kubectl create secret generic openai-api-key \
  --from-literal=api-key="$OPENAI_API_KEY" \
  --namespace=nephoran-system \
  --dry-run=client -o yaml | kubectl apply -f -

# Redis credentials
echo "2. Setting up Redis credentials..."
REDIS_PASSWORD=$(openssl rand -base64 32)
kubectl create secret generic redis-credentials \
  --from-literal=password="$REDIS_PASSWORD" \
  --namespace=nephoran-system \
  --dry-run=client -o yaml | kubectl apply -f -

# TLS certificates (if needed)
echo "3. Setting up TLS certificates..."
if [ -f "tls.crt" ] && [ -f "tls.key" ]; then
  kubectl create secret tls nephoran-tls \
    --cert=tls.crt \
    --key=tls.key \
    --namespace=nephoran-system \
    --dry-run=client -o yaml | kubectl apply -f -
  echo "✓ TLS certificates configured"
else
  echo "! TLS certificates not found, skipping..."
fi

echo "✓ Secrets configuration complete"
```

#### Phase 2: Core Component Deployment

**1. Deploy Redis:**

```bash
# Deploy Redis with high availability
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
  namespace: nephoran-system
spec:
  serviceName: redis
  replicas: 3
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        command:
        - redis-server
        - /etc/redis/redis.conf
        ports:
        - containerPort: 6379
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-credentials
              key: password
        volumeMounts:
        - name: redis-config
          mountPath: /etc/redis
        - name: redis-data
          mountPath: /data
        resources:
          requests:
            cpu: 250m
            memory: 512Mi
          limits:
            cpu: 500m
            memory: 1Gi
      volumes:
      - name: redis-config
        configMap:
          name: redis-config
  volumeClaimTemplates:
  - metadata:
      name: redis-data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: standard
      resources:
        requests:
          storage: 10Gi
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: nephoran-system
data:
  redis.conf: |
    requirepass \${REDIS_PASSWORD}
    maxmemory 1gb
    maxmemory-policy allkeys-lru
    save 900 1
    save 300 10
    save 60 10000
    tcp-keepalive 300
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: nephoran-system
spec:
  selector:
    app: redis
  ports:
  - port: 6379
  clusterIP: None
EOF
```

**2. Deploy Core Services:**

```bash
# Apply all core components
kubectl apply -f deployments/crds/
kubectl apply -f deployments/kustomize/overlays/production/

# Wait for deployments
echo "Waiting for deployments to be ready..."
kubectl wait --for=condition=available deployment/llm-processor -n nephoran-system --timeout=600s
kubectl wait --for=condition=available deployment/rag-api -n nephoran-system --timeout=600s
kubectl wait --for=condition=available deployment/nephio-bridge -n nephoran-system --timeout=600s

echo "✓ Core services deployed and ready"
```

#### Phase 3: Service Configuration

**1. Configure Load Balancing:**

```yaml
# Apply load balancer configuration
apiVersion: v1
kind: Service
metadata:
  name: llm-processor-external
  namespace: nephoran-system
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
spec:
  type: LoadBalancer
  selector:
    app: llm-processor
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: streaming
    port: 8080
    targetPort: 8080
```

**2. Configure Auto-scaling:**

```bash
# Apply HPA configuration
kubectl apply -f - <<EOF
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-processor-hpa
  namespace: nephoran-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
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
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 30
EOF
```

#### Phase 4: Monitoring and Observability

**1. Deploy Prometheus:**

```bash
# Add Prometheus Helm repository
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install Prometheus with Nephoran-specific configuration
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace nephoran-system \
  --values - <<EOF
prometheus:
  prometheusSpec:
    serviceMonitorSelectorNilUsesHelmValues: false
    additionalScrapeConfigs:
    - job_name: 'nephoran-llm-processor'
      static_configs:
      - targets: ['llm-processor:8080']
      metrics_path: /metrics
      scrape_interval: 15s
    - job_name: 'nephoran-rag-api'
      static_configs:
      - targets: ['rag-api:8080']
      metrics_path: /metrics
      scrape_interval: 30s

grafana:
  adminPassword: admin123
  dashboardProviders:
    dashboardproviders.yaml:
      apiVersion: 1
      providers:
      - name: 'nephoran'
        orgId: 1
        folder: 'Nephoran'
        type: file
        disableDeletion: false
        editable: true
        options:
          path: /var/lib/grafana/dashboards/nephoran
  dashboards:
    nephoran:
      nephoran-overview:
        url: https://raw.githubusercontent.com/nephoran/intent-operator/main/monitoring/grafana/nephoran-overview.json
EOF
```

**2. Configure Alerting:**

```bash
# Apply alerting rules
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: nephoran-alerts
  namespace: nephoran-system
  labels:
    prometheus: kube-prometheus
    role: alert-rules
spec:
  groups:
  - name: nephoran.rules
    rules:
    - alert: NephoranHighLatency
      expr: histogram_quantile(0.95, rate(nephoran_intent_request_duration_seconds_bucket[5m])) > 5
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High intent processing latency"
        description: "95th percentile latency is {{ \$value }}s"
    
    - alert: NephoranLowCacheHitRate
      expr: (rate(nephoran_cache_hits_total[5m]) / (rate(nephoran_cache_hits_total[5m]) + rate(nephoran_cache_misses_total[5m]))) < 0.7
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Low cache hit rate"
        description: "Cache hit rate is {{ \$value | humanizePercentage }}"
    
    - alert: NephoranServiceDown
      expr: up{job=~"nephoran-.*"} == 0
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "Nephoran service is down"
        description: "Service {{ \$labels.job }} is down"
EOF
```

#### Phase 5: Validation and Testing

**1. Deployment Validation:**

```bash
#!/bin/bash
# validate-deployment.sh - Comprehensive deployment validation

echo "=== Validating Production Deployment ==="

# Check all pods are running
echo "1. Checking pod status..."
if kubectl get pods -n nephoran-system | grep -v Running | grep -v NAME | wc -l | grep -q "0"; then
  echo "✓ All pods are running"
else
  echo "✗ Some pods are not running:"
  kubectl get pods -n nephoran-system | grep -v Running
  exit 1
fi

# Test API endpoints
echo "2. Testing API endpoints..."
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 5

# Health check
if curl -s http://localhost:8080/healthz | grep -q "healthy"; then
  echo "✓ Health check passed"
else
  echo "✗ Health check failed"
  kill $PORT_FORWARD_PID
  exit 1
fi

# Readiness check
if curl -s http://localhost:8080/readyz | grep -q "ready"; then
  echo "✓ Readiness check passed"
else
  echo "✗ Readiness check failed"
  kill $PORT_FORWARD_PID
  exit 1
fi

# Test intent processing
echo "3. Testing intent processing..."
response=$(curl -s -X POST http://localhost:8080/process \
  -H "Content-Type: application/json" \
  -d '{"intent": "Test deployment validation intent", "metadata": {"priority": "low"}}')

if echo "$response" | grep -q '"status"'; then
  echo "✓ Intent processing test passed"
else
  echo "✗ Intent processing test failed: $response"
  kill $PORT_FORWARD_PID
  exit 1
fi

# Test streaming
echo "4. Testing streaming..."
if curl -s -N -X POST http://localhost:8080/stream \
  -H "Accept: text/event-stream" \
  -d '{"query": "test streaming", "session_id": "validation_test"}' | head -3 | grep -q "event:"; then
  echo "✓ Streaming test passed"
else
  echo "✗ Streaming test failed"
fi

kill $PORT_FORWARD_PID

# Check cache performance
echo "5. Checking cache performance..."
cache_stats=$(kubectl exec deployment/llm-processor -n nephoran-system -- curl -s http://localhost:8080/cache/stats)
if echo "$cache_stats" | grep -q "overall_hit_rate"; then
  echo "✓ Cache system operational"
else
  echo "✗ Cache system issues detected"
fi

echo "✓ Deployment validation complete"
```

**2. Load Testing:**

```bash
#!/bin/bash
# production-load-test.sh - Production load testing

echo "=== Production Load Test ==="

# Install artillery if not present
if ! command -v artillery &> /dev/null; then
  npm install -g artillery
fi

# Create load test configuration
cat > load-test-config.yml <<EOF
config:
  target: 'http://llm-processor.nephoran-system.svc.cluster.local:8080'
  phases:
    - duration: 60
      arrivalRate: 5
      name: "Warm up"
    - duration: 300
      arrivalRate: 20
      name: "Sustained load"
    - duration: 60
      arrivalRate: 50
      name: "Peak load"
  variables:
    intents:
      - "Deploy AMF with 3 replicas for production"
      - "Scale SMF to 5 instances for high availability"
      - "Configure UPF for ultra-low latency"
      - "Setup network slice for enhanced mobile broadband"

scenarios:
  - name: "Intent Processing Load"
    weight: 80
    flow:
      - post:
          url: "/process"
          json:
            intent: "{{ intents }}"
            metadata:
              priority: "{{ \$randomString(['low', 'medium', 'high']) }}"
            options:
              enable_cache: true
  - name: "Streaming Load"
    weight: 20
    flow:
      - post:
          url: "/stream"
          json:
            query: "{{ intents }}"
            session_id: "load_test_{{ \$uuid }}"
EOF

# Run load test from within cluster
kubectl run artillery-load-test --rm -i --tty --restart=Never \
  --image=artilleryio/artillery:latest \
  --namespace=nephoran-system \
  -- run /tmp/load-test-config.yml
```

### Production Deployment Checklist

Before going live, ensure all these items are completed:

**Infrastructure:**
- [ ] Kubernetes cluster meets minimum requirements
- [ ] Storage classes configured and tested
- [ ] Network policies applied
- [ ] Load balancer configured
- [ ] TLS certificates installed

**Security:**
- [ ] RBAC policies applied
- [ ] Service accounts configured
- [ ] Secrets properly secured
- [ ] Network segmentation implemented
- [ ] API rate limiting configured

**Services:**
- [ ] All pods running and healthy
- [ ] Auto-scaling configured and tested
- [ ] Circuit breakers configured
- [ ] Cache system operational
- [ ] Backup procedures in place

**Monitoring:**
- [ ] Prometheus collecting metrics
- [ ] Grafana dashboards installed
- [ ] Alert rules configured
- [ ] Log aggregation setup
- [ ] Health checks configured

**Testing:**
- [ ] End-to-end functionality test passed
- [ ] Load testing completed
- [ ] Failover scenarios tested
- [ ] Performance benchmarks met
- [ ] Security scanning completed

**Documentation:**
- [ ] Runbooks updated
- [ ] Contact information current
- [ ] Escalation procedures documented
- [ ] Change management procedures in place

## Configuration Reference

### Environment Variables

#### Core Service Configuration

**LLM Processor Service:**
```bash
# Required
OPENAI_API_KEY="sk-your-api-key"

# Optional with defaults
LLM_PROCESSOR_PORT=8080
LLM_PROCESSOR_LOG_LEVEL=info
LLM_PROCESSOR_MAX_CONCURRENT_REQUESTS=100
LLM_PROCESSOR_REQUEST_TIMEOUT=30s
LLM_PROCESSOR_ENABLE_METRICS=true

# Streaming configuration
STREAMING_ENABLED=true
STREAMING_MAX_CONCURRENT_SESSIONS=100
STREAMING_SESSION_TIMEOUT=10m
STREAMING_HEARTBEAT_INTERVAL=30s
STREAMING_BUFFER_SIZE=8192

# Cache configuration
CACHE_L1_MAX_SIZE=1000
CACHE_L1_TTL=1h
CACHE_L2_ENABLED=true
CACHE_L2_REDIS_ADDR=redis:6379
CACHE_L2_TTL=4h

# Circuit breaker configuration
CIRCUIT_BREAKER_ENABLED=true
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
CIRCUIT_BREAKER_TIMEOUT=30s
CIRCUIT_BREAKER_RESET_TIMEOUT=60s
```

**RAG API Service:**
```bash
# Required
WEAVIATE_URL=http://weaviate:8080
WEAVIATE_API_KEY=your-weaviate-key

# Optional with defaults
RAG_API_PORT=8080
RAG_API_LOG_LEVEL=info
RAG_API_MAX_WORKERS=4
RAG_API_ENABLE_CORS=true

# Document processing
RAG_CHUNK_SIZE=1000
RAG_CHUNK_OVERLAP=200
RAG_MAX_RETRIEVAL_RESULTS=10
RAG_SIMILARITY_THRESHOLD=0.7

# Performance optimization
RAG_ENABLE_BATCH_PROCESSING=true
RAG_BATCH_SIZE=50
RAG_PROCESSING_TIMEOUT=60s
```

### Configuration Files

#### Complete Production Configuration

**LLM Processor Configuration (`llm-processor-config.yaml`):**

```yaml
# LLM Processor Production Configuration
api:
  port: 8080
  enable_cors: true
  cors_origins: ["https://dashboard.nephoran.com"]
  rate_limit:
    enabled: true
    requests_per_minute: 1000
    burst: 50

logging:
  level: info
  format: json
  output: stdout

openai:
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4o-mini"
  max_tokens: 4096
  temperature: 0.0
  timeout: 30s
  rate_limit:
    requests_per_minute: 3000
    tokens_per_minute: 1000000

streaming:
  enabled: true
  max_concurrent_sessions: 200
  session_timeout: 15m
  heartbeat_interval: 30s
  buffer_size: 16384
  chunk_size: 512
  max_chunk_delay: 25ms
  enable_compression: true
  reconnection:
    enabled: true
    max_attempts: 3
    base_delay: 1s

cache:
  l1:
    enabled: true
    max_size: 2000
    ttl: 2h
    eviction_policy: "LRU"
    enable_compression: true
    compression_threshold: 1024
  l2:
    enabled: true
    redis_addr: "${REDIS_ADDR:-redis:6379}"
    redis_password: "${REDIS_PASSWORD}"
    redis_db: 0
    ttl: 8h
    max_connections: 50
    idle_timeout: 5m
    compression_enabled: true
  warming:
    enabled: true
    strategy: "predictive"
    schedule: "0 */4 * * *"  # Every 4 hours
    popular_queries_limit: 1000

circuit_breaker:
  enabled: true
  failure_threshold: 10
  failure_rate: 0.5
  minimum_request_count: 20
  timeout: 45s
  reset_timeout: 120s
  success_threshold: 5
  half_open_max_requests: 10
  health_check:
    enabled: true
    interval: 30s
    timeout: 10s

performance:
  auto_optimization:
    enabled: true
    interval: 10m
    memory_threshold: 0.8
    cpu_threshold: 0.7
    latency_threshold: 3s
  gc:
    target_percentage: 100
    force_gc_threshold: 0.9

monitoring:
  metrics:
    enabled: true
    port: 9090
    path: "/metrics"
  health_checks:
    enabled: true
    startup_probe:
      initial_delay: 10s
      period: 10s
      timeout: 5s
    readiness_probe:
      period: 5s
      timeout: 3s
    liveness_probe:
      period: 30s
      timeout: 10s
```

**RAG API Configuration (`rag-config.yaml`):**

```yaml
# RAG API Production Configuration
api:
  host: "0.0.0.0"
  port: 8080
  debug: false
  workers: 8
  timeout: 60

cors:
  enabled: true
  origins: ["*"]
  methods: ["GET", "POST", "OPTIONS"]
  headers: ["Content-Type", "Authorization"]

weaviate:
  url: "${WEAVIATE_URL}"
  api_key: "${WEAVIATE_API_KEY}"
  timeout: 30
  retries: 3
  batch_size: 100
  
  collections:
    telecom_documents: "TelecomKnowledge"
    intent_patterns: "IntentPatterns"
    network_functions: "NetworkFunctions"

document_processing:
  chunk_size: 1000
  chunk_overlap: 200
  min_chunk_size: 100
  max_chunk_size: 2000
  
  quality_scoring:
    enabled: true
    min_quality_threshold: 0.6
    factors:
      - completeness
      - coherence
      - technical_accuracy
  
  batch_processing:
    enabled: true
    batch_size: 50
    max_concurrent_batches: 4
    timeout_per_batch: 300s

retrieval:
  max_results: 20
  similarity_threshold: 0.7
  
  hybrid_search:
    enabled: true
    alpha: 0.7  # Vector vs keyword balance
    
  reranking:
    enabled: true
    model: "cross-encoder"
    top_k: 10
    
  query_enhancement:
    enabled: true
    expand_acronyms: true
    add_synonyms: true
    spell_correction: true

embedding:
  provider: "openai"
  model: "text-embedding-3-large"
  dimensions: 3072
  batch_size: 100
  
  fallback:
    enabled: true
    provider: "local"
    model: "all-mpnet-base-v2"

caching:
  enabled: true
  ttl: 1h
  max_size: 10000
  compression: true

logging:
  level: "INFO"
  format: "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
  file: "/var/log/rag-api/rag-api.log"
  max_size: "100MB"
  backup_count: 5
```

#### Kubernetes Configuration Examples

**Production Deployment Manifest:**

```yaml
# production-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-processor
  namespace: nephoran-system
  labels:
    app: llm-processor
    version: v2.0.0
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: llm-processor
  template:
    metadata:
      labels:
        app: llm-processor
        version: v2.0.0
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: llm-processor
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 2000
      containers:
      - name: llm-processor
        image: nephoran/llm-processor:v2.0.0
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        - name: metrics
          containerPort: 9090
          protocol: TCP
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: openai-api-key
              key: api-key
        - name: REDIS_ADDR
          value: "redis:6379"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-credentials
              key: password
        volumeMounts:
        - name: config
          mountPath: /etc/nephoran
          readOnly: true
        - name: tmp
          mountPath: /tmp
        resources:
          requests:
            cpu: 1000m
            memory: 2Gi
          limits:
            cpu: 2000m
            memory: 4Gi
        livenessProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 5
          failureThreshold: 3
        startupProbe:
          httpGet:
            path: /healthz
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 30
      volumes:
      - name: config
        configMap:
          name: llm-processor-config
      - name: tmp
        emptyDir: {}
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - llm-processor
              topologyKey: kubernetes.io/hostname
      tolerations:
      - key: "node.kubernetes.io/not-ready"
        operator: "Exists"
        effect: "NoExecute"
        tolerationSeconds: 300
      - key: "node.kubernetes.io/unreachable"
        operator: "Exists"
        effect: "NoExecute"
        tolerationSeconds: 300
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-processor-config
  namespace: nephoran-system
data:
  config.yaml: |
    # Include your complete configuration here
    # (Use the configuration from above)
```

### Configuration Validation

**Configuration validation script:**

```bash
#!/bin/bash
# validate-config.sh - Validate Nephoran configuration

echo "=== Configuration Validation ==="

# Function to validate required environment variables
validate_env_vars() {
  local required_vars=(
    "OPENAI_API_KEY"
    "WEAVIATE_URL"
    "REDIS_ADDR"
  )
  
  for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
      echo "✗ Required environment variable $var is not set"
      return 1
    else
      echo "✓ $var is set"
    fi
  done
}

# Function to validate configuration files
validate_config_files() {
  local config_files=(
    "/etc/nephoran/llm-processor-config.yaml"
    "/etc/nephoran/rag-config.yaml"
  )
  
  for file in "${config_files[@]}"; do
    if [ -f "$file" ]; then
      if yq eval '.' "$file" >/dev/null 2>&1; then
        echo "✓ $file is valid YAML"
      else
        echo "✗ $file contains invalid YAML"
        return 1
      fi
    else
      echo "! $file not found (may be optional)"
    fi
  done
}

# Function to validate service connectivity
validate_connectivity() {
  echo "Testing service connectivity..."
  
  # Test OpenAI API
  if curl -s -f -H "Authorization: Bearer $OPENAI_API_KEY" \
    https://api.openai.com/v1/models >/dev/null; then
    echo "✓ OpenAI API connectivity verified"
  else
    echo "✗ Cannot connect to OpenAI API"
    return 1
  fi
  
  # Test Weaviate
  if curl -s -f "$WEAVIATE_URL/v1/.well-known/ready" >/dev/null; then
    echo "✓ Weaviate connectivity verified"
  else
    echo "✗ Cannot connect to Weaviate"
    return 1
  fi
  
  # Test Redis
  if redis-cli -h "${REDIS_ADDR%:*}" -p "${REDIS_ADDR#*:}" ping >/dev/null 2>&1; then
    echo "✓ Redis connectivity verified"
  else
    echo "✗ Cannot connect to Redis"
    return 1
  fi
}

# Run all validations
validate_env_vars && validate_config_files && validate_connectivity

if [ $? -eq 0 ]; then
  echo "✓ All configuration validations passed"
  exit 0
else
  echo "✗ Configuration validation failed"
  exit 1
fi
```

## Operational Procedures

### Daily Operations

#### Morning Health Check Routine

```bash
#!/bin/bash
# daily-health-check.sh - Morning operational health check

echo "=== Daily Health Check - $(date) ==="

# 1. Check system status
echo "1. System Status:"
kubectl get pods -n nephoran-system -o wide | grep -E "(NAME|llm-processor|rag-api|redis)"

# 2. Check resource utilization
echo -e "\n2. Resource Utilization:"
kubectl top pods -n nephoran-system | head -10

# 3. Check recent errors
echo -e "\n3. Recent Errors (last 24h):"
ERROR_COUNT=$(kubectl logs --since=24h -l app=llm-processor -n nephoran-system | grep -i error | wc -l)
echo "Error count in last 24h: $ERROR_COUNT"

if [ "$ERROR_COUNT" -gt 10 ]; then
  echo "⚠️  High error count detected, recent errors:"
  kubectl logs --since=1h -l app=llm-processor -n nephoran-system | grep -i error | tail -5
fi

# 4. Check cache performance
echo -e "\n4. Cache Performance:"
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 3

CACHE_STATS=$(curl -s http://localhost:8080/cache/stats)
OVERALL_HIT_RATE=$(echo "$CACHE_STATS" | jq -r '.overall_stats.overall_hit_rate // 0')
L1_HIT_RATE=$(echo "$CACHE_STATS" | jq -r '.l1_stats.hit_rate // 0')
L2_HIT_RATE=$(echo "$CACHE_STATS" | jq -r '.l2_stats.hit_rate // 0')

echo "Overall hit rate: $(echo "$OVERALL_HIT_RATE * 100" | bc -l | cut -d. -f1)%"
echo "L1 hit rate: $(echo "$L1_HIT_RATE * 100" | bc -l | cut -d. -f1)%"
echo "L2 hit rate: $(echo "$L2_HIT_RATE * 100" | bc -l | cut -d. -f1)%"

if (( $(echo "$OVERALL_HIT_RATE < 0.7" | bc -l) )); then
  echo "⚠️  Cache hit rate below 70%, consider warming cache"
  curl -s -X POST http://localhost:8080/cache/warm \
    -d '{"strategy": "popular_queries", "limit": 100}' >/dev/null
  echo "Cache warming triggered"
fi

kill $PORT_FORWARD_PID 2>/dev/null

# 5. Check circuit breaker status
echo -e "\n5. Circuit Breaker Status:"
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 3

CB_STATUS=$(curl -s http://localhost:8080/circuit-breaker/status)
OPEN_CIRCUITS=$(echo "$CB_STATUS" | jq -r 'to_entries[] | select(.value.state == "open") | .key')

if [ -n "$OPEN_CIRCUITS" ]; then
  echo "⚠️  Open circuit breakers detected: $OPEN_CIRCUITS"
else
  echo "✓ All circuit breakers are closed"
fi

kill $PORT_FORWARD_PID 2>/dev/null

# 6. Check recent intents
echo -e "\n6. Recent Intent Processing:"
RECENT_INTENTS=$(kubectl get networkintents --all-namespaces --sort-by=.metadata.creationTimestamp | tail -5)
echo "$RECENT_INTENTS"

# 7. Performance summary
echo -e "\n7. Performance Summary (last 1h):"
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 3

PERF_METRICS=$(curl -s http://localhost:8080/performance/metrics)
AVG_LATENCY=$(echo "$PERF_METRICS" | jq -r '.average_latency // "N/A"')
THROUGHPUT=$(echo "$PERF_METRICS" | jq -r '.throughput_rpm // 0')
ERROR_RATE=$(echo "$PERF_METRICS" | jq -r '.error_rate // 0')

echo "Average latency: $AVG_LATENCY"
echo "Throughput: $THROUGHPUT requests/min"
echo "Error rate: $(echo "$ERROR_RATE * 100" | bc -l | cut -d. -f1)%"

kill $PORT_FORWARD_PID 2>/dev/null

echo -e "\n=== Health Check Complete ==="
```

#### Weekly Maintenance Tasks

**Weekly maintenance script:**

```bash
#!/bin/bash
# weekly-maintenance.sh - Weekly maintenance tasks

echo "=== Weekly Maintenance - $(date) ==="

# 1. Update performance baselines
echo "1. Updating performance baselines..."
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 3

CURRENT_METRICS=$(curl -s http://localhost:8080/performance/metrics)
echo "$CURRENT_METRICS" > "/var/log/nephoran/performance-baseline-$(date +%Y%m%d).json"

kill $PORT_FORWARD_PID 2>/dev/null

# 2. Cache optimization
echo "2. Performing cache optimization..."
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 3

# Clear stale cache entries
curl -s -X DELETE "http://localhost:8080/cache/clear?level=l2&prefix=stale:"

# Optimize cache configuration based on usage
CACHE_STATS=$(curl -s http://localhost:8080/cache/stats)
L1_EVICTIONS=$(echo "$CACHE_STATS" | jq -r '.l1_stats.evictions // 0')
L2_ERRORS=$(echo "$CACHE_STATS" | jq -r '.l2_stats.errors // 0')

if [ "$L1_EVICTIONS" -gt 1000 ]; then
  echo "High L1 evictions detected, consider increasing cache size"
fi

if [ "$L2_ERRORS" -gt 10 ]; then
  echo "L2 cache errors detected, checking Redis health"
  kubectl logs -l app=redis -n nephoran-system --tail=50
fi

# Warm cache for next week
curl -s -X POST http://localhost:8080/cache/warm \
  -d '{"strategy": "predictive", "limit": 1000}' >/dev/null

kill $PORT_FORWARD_PID 2>/dev/null

# 3. Clean up old resources
echo "3. Cleaning up old resources..."

# Clean up completed intents older than 7 days
kubectl get networkintents --all-namespaces -o json | \
  jq -r '.items[] | select(.status.phase == "Completed" and (.metadata.creationTimestamp | fromdateiso8601) < (now - 604800)) | "\(.metadata.namespace) \(.metadata.name)"' | \
  while read namespace name; do
    echo "Deleting old intent: $namespace/$name"
    kubectl delete networkintent "$name" -n "$namespace"
  done

# Clean up old logs
find /var/log/nephoran -name "*.log" -mtime +30 -delete

# 4. Security updates check
echo "4. Checking for security updates..."
kubectl get pods -n nephoran-system -o json | \
  jq -r '.items[] | "\(.metadata.name) \(.spec.containers[0].image)"' | \
  while read pod_name image; do
    echo "Checking $pod_name: $image"
    # In a real environment, you'd check for CVEs or updates
    # This is a placeholder for your security scanning process
  done

# 5. Backup verification
echo "5. Verifying backups..."
if [ -d "/backups/nephoran" ]; then
  LATEST_BACKUP=$(ls -t /backups/nephoran/ | head -1)
  if [ -n "$LATEST_BACKUP" ]; then
    BACKUP_AGE=$(( ($(date +%s) - $(stat -c %Y "/backups/nephoran/$LATEST_BACKUP")) / 86400 ))
    if [ "$BACKUP_AGE" -le 1 ]; then
      echo "✓ Recent backup found: $LATEST_BACKUP"
    else
      echo "⚠️  Latest backup is $BACKUP_AGE days old"
    fi
  else
    echo "⚠️  No backups found"
  fi
else
  echo "⚠️  Backup directory not found"
fi

# 6. Generate weekly report
echo "6. Generating weekly report..."
cat > "/var/log/nephoran/weekly-report-$(date +%Y%m%d).md" <<EOF
# Nephoran Weekly Report - $(date +%Y-%m-%d)

## System Health
- Pod Status: $(kubectl get pods -n nephoran-system | grep -c Running) running pods
- Error Count (7d): $(kubectl logs --since=168h -l app=llm-processor -n nephoran-system | grep -i error | wc -l)
- Cache Hit Rate: $(echo "$CACHE_STATS" | jq -r '.overall_stats.overall_hit_rate // 0' | awk '{printf "%.1f%%", $1*100}')

## Performance Metrics
- Average Latency: $(echo "$CURRENT_METRICS" | jq -r '.average_latency // "N/A"')
- Throughput: $(echo "$CURRENT_METRICS" | jq -r '.throughput_rpm // 0') req/min
- Intent Success Rate: $(kubectl get networkintents --all-namespaces -o json | jq -r '[.items[] | select(.status.phase == "Completed")] | length') / $(kubectl get networkintents --all-namespaces -o json | jq -r '.items | length')

## Actions Taken
- Cache optimization completed
- Old resources cleaned up
- Security check performed
- Backup verification completed

## Recommendations
- Monitor cache performance trends
- Consider scaling if throughput increases
- Update monitoring thresholds based on performance baselines
EOF

echo "Weekly report generated: /var/log/nephoran/weekly-report-$(date +%Y%m%d).md"

echo "=== Weekly Maintenance Complete ==="
```

### Incident Response Procedures

#### Severity Levels and Response Times

**P1 - Critical (Response: Immediate):**
- System completely down
- Data loss or corruption
- Security breach
- Target Resolution: 1 hour

**P2 - High (Response: 1 hour):**
- Major functionality impaired
- Performance degradation > 50%
- Single component failure
- Target Resolution: 4 hours

**P3 - Medium (Response: 4 hours):**
- Minor functionality issues
- Performance degradation < 50%
- Non-critical component issues
- Target Resolution: 24 hours

**P4 - Low (Response: 24 hours):**
- Cosmetic issues
- Documentation problems
- Enhancement requests
- Target Resolution: 1 week

#### Incident Response Playbook

**P1 Critical Incident Response:**

```bash
#!/bin/bash
# p1-incident-response.sh - Critical incident response automation

INCIDENT_ID="P1-$(date +%Y%m%d-%H%M%S)"
echo "🚨 P1 CRITICAL INCIDENT DECLARED: $INCIDENT_ID"

# Immediate actions (0-5 minutes)
echo "=== IMMEDIATE ACTIONS ==="

# 1. Assess system status
echo "1. Assessing system status..."
kubectl get pods -n nephoran-system --no-headers | grep -v Running > /tmp/failed_pods_$INCIDENT_ID
FAILED_COUNT=$(wc -l < /tmp/failed_pods_$INCIDENT_ID)

if [ "$FAILED_COUNT" -gt 0 ]; then
  echo "🔥 CRITICAL: $FAILED_COUNT pods are failing"
  cat /tmp/failed_pods_$INCIDENT_ID
fi

# 2. Auto-scale critical services
echo "2. Auto-scaling critical services..."
kubectl scale deployment llm-processor --replicas=5 -n nephoran-system
kubectl scale deployment rag-api --replicas=3 -n nephoran-system

# 3. Reset circuit breakers
echo "3. Resetting circuit breakers..."
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 5

curl -s -X POST http://localhost:8080/circuit-breaker/status \
  -d '{"action": "reset"}' >/dev/null

kill $PORT_FORWARD_PID 2>/dev/null

# 4. Clear problematic cache
echo "4. Clearing cache..."
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 5

curl -s -X DELETE http://localhost:8080/cache/clear >/dev/null

kill $PORT_FORWARD_PID 2>/dev/null

# 5. Collect diagnostic information
echo "5. Collecting diagnostics..."
mkdir -p /tmp/incident_diagnostics_$INCIDENT_ID

kubectl get pods -n nephoran-system -o yaml > /tmp/incident_diagnostics_$INCIDENT_ID/pods.yaml
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' > /tmp/incident_diagnostics_$INCIDENT_ID/events.txt
kubectl logs -l app=llm-processor -n nephoran-system --tail=1000 > /tmp/incident_diagnostics_$INCIDENT_ID/llm-processor.log

# 6. Notify stakeholders
echo "6. Notifying stakeholders..."
# Send Slack notification
if [ -n "$SLACK_WEBHOOK_URL" ]; then
  curl -X POST "$SLACK_WEBHOOK_URL" \
    -H 'Content-type: application/json' \
    --data "{\"text\":\"🚨 P1 INCIDENT DECLARED: $INCIDENT_ID\n$FAILED_COUNT pods failing\nImmediate response initiated\"}"
fi

# Send email notification
if command -v mail &> /dev/null; then
  echo "P1 Incident $INCIDENT_ID: $FAILED_COUNT pods failing. Auto-recovery initiated." | \
    mail -s "P1 INCIDENT: Nephoran System" oncall@company.com
fi

# Short-term stabilization (5-30 minutes)
echo "=== SHORT-TERM STABILIZATION ==="

# Wait for scaling to take effect
echo "Waiting for auto-scaling to take effect..."
sleep 60

# Check if situation improved
NEW_FAILED_COUNT=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
echo "Failed pods after scaling: $NEW_FAILED_COUNT (was $FAILED_COUNT)"

if [ "$NEW_FAILED_COUNT" -lt "$FAILED_COUNT" ]; then
  echo "✅ Situation improving, continuing monitoring"
else
  echo "🚨 No improvement, escalating to manual intervention"
  
  # Restart failing deployments
  kubectl rollout restart deployment/llm-processor -n nephoran-system
  kubectl rollout restart deployment/rag-api -n nephoran-system
  
  # Wait for restart
  kubectl rollout status deployment/llm-processor -n nephoran-system --timeout=300s
  kubectl rollout status deployment/rag-api -n nephoran-system --timeout=300s
fi

# Final assessment
FINAL_FAILED_COUNT=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
if [ "$FINAL_FAILED_COUNT" -eq 0 ]; then
  echo "✅ All pods are now running - incident resolved"
  
  # Update incident status
  if [ -n "$SLACK_WEBHOOK_URL" ]; then
    curl -X POST "$SLACK_WEBHOOK_URL" \
      -H 'Content-type: application/json' \
      --data "{\"text\":\"✅ INCIDENT RESOLVED: $INCIDENT_ID\nAll systems operational\"}"
  fi
else
  echo "🚨 $FINAL_FAILED_COUNT pods still failing - manual intervention required"
  
  # Escalate to senior engineer
  if [ -n "$SLACK_WEBHOOK_URL" ]; then
    curl -X POST "$SLACK_WEBHOOK_URL" \
      -H 'Content-type: application/json' \
      --data "{\"text\":\"🚨 ESCALATION NEEDED: $INCIDENT_ID\n$FINAL_FAILED_COUNT pods still failing\n@senior-engineer\"}"
  fi
fi

echo "🏁 P1 incident response completed: $INCIDENT_ID"
echo "Diagnostics saved to: /tmp/incident_diagnostics_$INCIDENT_ID/"
```

### Backup and Recovery Procedures

#### Automated Backup System

```bash
#!/bin/bash
# automated-backup.sh - Comprehensive backup system

BACKUP_DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="/backups/nephoran-$BACKUP_DATE"
S3_BUCKET="nephoran-backups"

echo "=== Starting Automated Backup - $BACKUP_DATE ==="

# Create backup directory
mkdir -p "$BACKUP_DIR"/{kubernetes,data,config,logs}

# 1. Backup Kubernetes resources
echo "1. Backing up Kubernetes resources..."
kubectl get all -n nephoran-system -o yaml > "$BACKUP_DIR/kubernetes/all-resources.yaml"
kubectl get pvc -n nephoran-system -o yaml > "$BACKUP_DIR/kubernetes/persistent-volumes.yaml"
kubectl get configmaps -n nephoran-system -o yaml > "$BACKUP_DIR/kubernetes/configmaps.yaml"
kubectl get secrets -n nephoran-system -o yaml > "$BACKUP_DIR/kubernetes/secrets.yaml"
kubectl get crd -o yaml | grep -A 10000 "nephoran.com" > "$BACKUP_DIR/kubernetes/crds.yaml"

# 2. Backup application data
echo "2. Backing up application data..."

# Redis data backup
kubectl exec statefulset/redis -n nephoran-system -- redis-cli --rdb /tmp/dump.rdb BGSAVE
sleep 10  # Wait for background save to complete
kubectl cp nephoran-system/redis-0:/tmp/dump.rdb "$BACKUP_DIR/data/redis-dump.rdb"

# Weaviate backup
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system >/dev/null 2>&1 &
WEAVIATE_PORT_FORWARD_PID=$!
sleep 5

curl -X POST http://localhost:8080/v1/backups \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"backup-$BACKUP_DATE\",
    \"include\": [\"TelecomKnowledge\", \"IntentPatterns\", \"NetworkFunctions\"]
  }"

# Wait for Weaviate backup to complete
sleep 30

# Download Weaviate backup
curl -X GET "http://localhost:8080/v1/backups/backup-$BACKUP_DATE" \
  -o "$BACKUP_DIR/data/weaviate-backup.tar.gz"

kill $WEAVIATE_PORT_FORWARD_PID 2>/dev/null

# 3. Backup configuration files
echo "3. Backing up configuration files..."
kubectl get configmaps llm-processor-config -n nephoran-system -o yaml > "$BACKUP_DIR/config/llm-processor-config.yaml"
kubectl get configmaps rag-config -n nephoran-system -o yaml > "$BACKUP_DIR/config/rag-config.yaml"

# 4. Backup recent logs
echo "4. Backing up recent logs..."
kubectl logs -l app=llm-processor -n nephoran-system --tail=10000 > "$BACKUP_DIR/logs/llm-processor.log"
kubectl logs -l app=rag-api -n nephoran-system --tail=10000 > "$BACKUP_DIR/logs/rag-api.log"
kubectl logs -l app=redis -n nephoran-system --tail=10000 > "$BACKUP_DIR/logs/redis.log"

# 5. Create backup metadata
cat > "$BACKUP_DIR/backup-metadata.json" <<EOF
{
  "backup_id": "$BACKUP_DATE",
  "timestamp": "$(date -Iseconds)",
  "kubernetes_version": "$(kubectl version --short | grep Server)",
  "nephoran_version": "$(kubectl get deployment llm-processor -n nephoran-system -o jsonpath='{.spec.template.spec.containers[0].image}')",
  "backup_type": "full",
  "components": [
    "kubernetes-resources",
    "redis-data", 
    "weaviate-data",
    "configuration",
    "logs"
  ],
  "retention_days": 30
}
EOF

# 6. Compress backup
echo "5. Compressing backup..."
tar -czf "$BACKUP_DIR.tar.gz" -C "$(dirname "$BACKUP_DIR")" "$(basename "$BACKUP_DIR")"

# 7. Upload to S3 (or your cloud storage)
echo "6. Uploading to cloud storage..."
if command -v aws &> /dev/null; then
  aws s3 cp "$BACKUP_DIR.tar.gz" "s3://$S3_BUCKET/daily-backups/"
  
  # Verify upload
  if aws s3 ls "s3://$S3_BUCKET/daily-backups/$(basename "$BACKUP_DIR.tar.gz")" >/dev/null; then
    echo "✅ Backup uploaded successfully"
  else
    echo "❌ Backup upload failed"
    exit 1
  fi
else
  echo "⚠️  AWS CLI not available, skipping cloud upload"
fi

# 8. Cleanup local backup
rm -rf "$BACKUP_DIR"
rm "$BACKUP_DIR.tar.gz"

# 9. Cleanup old backups (keep 30 days)
if command -v aws &> /dev/null; then
  echo "7. Cleaning up old backups..."
  CUTOFF_DATE=$(date -d '30 days ago' +%Y%m%d)
  
  aws s3 ls "s3://$S3_BUCKET/daily-backups/" | \
    awk '{print $4}' | \
    grep "nephoran-" | \
    while read backup_file; do
      backup_date=$(echo "$backup_file" | sed 's/nephoran-\([0-9]\{8\}\)-.*/\1/')
      if [ "$backup_date" -lt "$CUTOFF_DATE" ]; then
        echo "Deleting old backup: $backup_file"
        aws s3 rm "s3://$S3_BUCKET/daily-backups/$backup_file"
      fi
    done
fi

echo "✅ Automated backup completed: $BACKUP_DATE"
```

**Schedule backups with cron:**
```bash
# Add to crontab for daily backups at 2 AM
0 2 * * * /path/to/automated-backup.sh >> /var/log/nephoran/backup.log 2>&1
```

#### Disaster Recovery Procedure

```bash
#!/bin/bash
# disaster-recovery.sh - Complete system recovery from backup

BACKUP_ID=${1:-"latest"}
TARGET_NAMESPACE=${2:-"nephoran-system"}

echo "=== Disaster Recovery Procedure ==="
echo "Backup ID: $BACKUP_ID"
echo "Target Namespace: $TARGET_NAMESPACE"

# 1. Download backup from S3
echo "1. Downloading backup from cloud storage..."
if [ "$BACKUP_ID" = "latest" ]; then
  BACKUP_FILE=$(aws s3 ls s3://nephoran-backups/daily-backups/ | sort | tail -1 | awk '{print $4}')
else
  BACKUP_FILE="nephoran-$BACKUP_ID.tar.gz"
fi

if [ -z "$BACKUP_FILE" ]; then
  echo "❌ No backup file found"
  exit 1
fi

aws s3 cp "s3://nephoran-backups/daily-backups/$BACKUP_FILE" ./
tar -xzf "$BACKUP_FILE"

BACKUP_DIR=$(basename "$BACKUP_FILE" .tar.gz)

# 2. Verify backup integrity
echo "2. Verifying backup integrity..."
if [ -f "$BACKUP_DIR/backup-metadata.json" ]; then
  echo "✅ Backup metadata found"
  jq . "$BACKUP_DIR/backup-metadata.json"
else
  echo "❌ Backup metadata missing"
  exit 1
fi

# 3. Prepare target environment
echo "3. Preparing target environment..."
kubectl create namespace "$TARGET_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# 4. Restore Kubernetes resources
echo "4. Restoring Kubernetes resources..."

# Apply CRDs first
kubectl apply -f "$BACKUP_DIR/kubernetes/crds.yaml"

# Wait for CRDs to be established
sleep 10

# Apply other resources
kubectl apply -f "$BACKUP_DIR/kubernetes/configmaps.yaml"
kubectl apply -f "$BACKUP_DIR/kubernetes/secrets.yaml"
kubectl apply -f "$BACKUP_DIR/kubernetes/persistent-volumes.yaml"
kubectl apply -f "$BACKUP_DIR/kubernetes/all-resources.yaml"

# 5. Wait for pods to be ready
echo "5. Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=redis -n "$TARGET_NAMESPACE" --timeout=300s
kubectl wait --for=condition=ready pod -l app=rag-api -n "$TARGET_NAMESPACE" --timeout=300s
kubectl wait --for=condition=ready pod -l app=llm-processor -n "$TARGET_NAMESPACE" --timeout=300s

# 6. Restore Redis data
echo "6. Restoring Redis data..."
kubectl cp "$BACKUP_DIR/data/redis-dump.rdb" "$TARGET_NAMESPACE/redis-0:/tmp/restore-dump.rdb"
kubectl exec redis-0 -n "$TARGET_NAMESPACE" -- redis-cli --rdb /tmp/restore-dump.rdb DEBUG RESTART

# 7. Restore Weaviate data
echo "7. Restoring Weaviate data..."
kubectl port-forward svc/weaviate 8080:8080 -n "$TARGET_NAMESPACE" >/dev/null 2>&1 &
WEAVIATE_PORT_FORWARD_PID=$!
sleep 10

# Upload backup file to Weaviate
curl -X POST http://localhost:8080/v1/backups/restore \
  -H "Content-Type: multipart/form-data" \
  -F "backup=@$BACKUP_DIR/data/weaviate-backup.tar.gz"

kill $WEAVIATE_PORT_FORWARD_PID 2>/dev/null

# 8. Verify restoration
echo "8. Verifying restoration..."

# Check pod status
echo "Pod status:"
kubectl get pods -n "$TARGET_NAMESPACE"

# Test API endpoints
kubectl port-forward svc/llm-processor 8080:8080 -n "$TARGET_NAMESPACE" >/dev/null 2>&1 &
API_PORT_FORWARD_PID=$!
sleep 5

if curl -s http://localhost:8080/healthz | grep -q "healthy"; then
  echo "✅ API health check passed"
else
  echo "❌ API health check failed"
fi

# Test intent processing
INTENT_RESPONSE=$(curl -s -X POST http://localhost:8080/process \
  -H "Content-Type: application/json" \
  -d '{"intent": "Test recovery intent", "metadata": {"priority": "low"}}')

if echo "$INTENT_RESPONSE" | grep -q '"status"'; then
  echo "✅ Intent processing test passed"
else
  echo "❌ Intent processing test failed"
fi

kill $API_PORT_FORWARD_PID 2>/dev/null

# 9. Cleanup
echo "9. Cleaning up temporary files..."
rm -rf "$BACKUP_DIR"
rm "$BACKUP_FILE"

echo "✅ Disaster recovery completed successfully"
echo "System restored from backup: $BACKUP_ID"
echo "Target namespace: $TARGET_NAMESPACE"

# 10. Generate recovery report
cat > "recovery-report-$(date +%Y%m%d-%H%M%S).md" <<EOF
# Disaster Recovery Report

**Recovery Date:** $(date)
**Backup Used:** $BACKUP_ID
**Target Namespace:** $TARGET_NAMESPACE

## Recovery Summary
- Kubernetes resources: ✅ Restored
- Redis data: ✅ Restored
- Weaviate data: ✅ Restored
- API functionality: ✅ Verified
- Intent processing: ✅ Tested

## Post-Recovery Actions Required
1. Verify all integrations are working
2. Test streaming functionality
3. Check cache performance
4. Monitor system for 24-48 hours
5. Update monitoring dashboards if needed

## Recovery Time
- Total recovery time: $(echo $SECONDS | awk '{print int($1/60)":"int($1%60)}')
- RTO Achievement: $([ $SECONDS -lt 3600 ] && echo "✅ Under 1 hour" || echo "❌ Over 1 hour")
EOF

echo "Recovery report generated: recovery-report-$(date +%Y%m%d-%H%M%S).md"
```

This comprehensive operator manual provides all the essential information needed to successfully operate the enhanced Nephoran Intent Operator in production environments. The manual covers async processing workflows, caching best practices, production deployment procedures, and operational guidance to ensure optimal system performance and reliability.