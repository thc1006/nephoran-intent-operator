# Nephoran Intent Operator - Developer Portal

## 🚀 Welcome to the Nephoran Developer Portal

The Nephoran Intent Operator transforms natural language network intents into structured O-RAN network function deployments using advanced AI/ML capabilities. This developer portal provides everything you need to get started, from quick setup to advanced integrations.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Development Environment Setup](#development-environment-setup)
- [API Integration Guide](#api-integration-guide)
- [SDK and Client Libraries](#sdk-and-client-libraries)
- [Code Examples](#code-examples)
- [Testing and Validation](#testing-and-validation)
- [Deployment Strategies](#deployment-strategies)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Community and Support](#community-and-support)

## 🎯 Quick Start

### 5-Minute Setup

Get your development environment running in under 5 minutes:

```bash
# 1. Clone the repository
git clone https://github.com/nephoran/intent-operator.git
cd intent-operator

# 2. Set up local environment
make setup-dev

# 3. Start local cluster
kind create cluster --config deployments/kind-config.yaml

# 4. Deploy Nephoran services
./deploy.sh local

# 5. Test the installation
kubectl get pods -n nephoran-system
```

### First Intent Processing

```bash
# Process your first intent
curl -X POST http://localhost:8080/process \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy AMF with 3 replicas for enhanced mobile broadband"
  }'
```

Expected response:
```json
{
  "intent_id": "intent-12345",
  "status": "completed",
  "structured_output": {
    "type": "NetworkFunctionDeployment",
    "name": "amf-deployment",
    "spec": {
      "replicas": 3,
      "image": "registry.nephoran.com/5g-core/amf:v2.1.0"
    }
  },
  "processing_time": "1.234s",
  "confidence_score": 0.94
}
```

## 🛠 Development Environment Setup

### Prerequisites

- **Docker**: v20.10+
- **Kubernetes**: v1.25+ (Kind, Minikube, or cluster)
- **Go**: v1.21+ 
- **Python**: v3.8+
- **Node.js**: v18+ (for UI development)

### Environment Variables

```bash
# Core Configuration
export OPENAI_API_KEY="your-openai-api-key"
export WEAVIATE_URL="http://weaviate:8080"
export REDIS_URL="redis://redis:6379"

# Development Settings
export LOG_LEVEL="debug"
export STREAMING_ENABLED="true"
export CACHE_ENABLED="true"

# Authentication (optional)
export AUTH_ENABLED="false"
export JWT_SECRET_KEY="your-secret-key"
```

### Local Development Setup

#### Option 1: Docker Compose (Recommended for Development)

```bash
# Start all services
docker-compose -f deployments/docker-compose.dev.yml up -d

# Check services status
docker-compose ps

# View logs
docker-compose logs -f llm-processor
```

#### Option 2: Kubernetes (Recommended for Integration Testing)

```bash
# Deploy to Kind cluster
./deploy.sh local

# Port forward for local access
kubectl port-forward svc/llm-processor 8080:8080 -n nephoran-system &
kubectl port-forward svc/rag-api 5001:5001 -n nephoran-system &
kubectl port-forward svc/grafana 3000:3000 -n nephoran-system &
```

## 🔌 API Integration Guide

### Authentication Methods

#### 1. API Key Authentication (Simple)

```bash
export API_KEY="your-api-key"
curl -H "X-API-Key: $API_KEY" http://localhost:8080/process
```

#### 2. JWT Bearer Token (Recommended)

```python
import requests
import jwt

# Generate token (in production, get from OAuth2 flow)
token = jwt.encode({"user": "developer", "roles": ["operator"]}, "secret", algorithm="HS256")

headers = {"Authorization": f"Bearer {token}"}
response = requests.post("http://localhost:8080/process", headers=headers, json={
    "intent": "Deploy SMF with 5 replicas"
})
```

#### 3. OAuth2 Flow (Production)

```javascript
// Frontend OAuth2 integration
const authUrl = `${baseUrl}/auth/login/azure?redirect_uri=${redirectUri}`;
window.location.href = authUrl;

// Handle callback
const urlParams = new URLSearchParams(window.location.search);
const accessToken = urlParams.get('access_token');
```

### Core API Patterns

#### Synchronous Processing

```python
import requests

class NephoranClient:
    def __init__(self, base_url, api_key=None):
        self.base_url = base_url
        self.session = requests.Session()
        if api_key:
            self.session.headers.update({'X-API-Key': api_key})
    
    def process_intent(self, intent, metadata=None):
        response = self.session.post(f'{self.base_url}/process', json={
            'intent': intent,
            'metadata': metadata or {}
        })
        response.raise_for_status()
        return response.json()

# Usage
client = NephoranClient('http://localhost:8080', 'your-api-key')
result = client.process_intent('Deploy AMF with 3 replicas')
```

#### Asynchronous Processing

```python
def process_intent_async(client, intent, callback_url):
    response = client.session.post(f'{client.base_url}/process', json={
        'intent': intent,
        'options': {
            'async': True,
            'callback_url': callback_url
        }
    })
    return response.json()['intent_id']

# Usage
intent_id = process_intent_async(client, 'Scale UPF to 10 instances', 'https://myapp.com/webhook')
```

#### Streaming Integration

```javascript
// JavaScript SSE client
class NephoranStreamingClient {
    constructor(baseUrl, apiKey) {
        this.baseUrl = baseUrl;
        this.apiKey = apiKey;
    }
    
    streamIntent(query, onChunk, onComplete, onError) {
        const eventSource = new EventSource(`${this.baseUrl}/stream`, {
            headers: {
                'X-API-Key': this.apiKey,
                'Content-Type': 'application/json'
            }
        });
        
        eventSource.addEventListener('chunk', (event) => {
            const data = JSON.parse(event.data);
            onChunk(data.delta);
        });
        
        eventSource.addEventListener('completion', (event) => {
            const data = JSON.parse(event.data);
            onComplete(data);
            eventSource.close();
        });
        
        eventSource.addEventListener('error', (event) => {
            onError(event);
            eventSource.close();
        });
        
        return eventSource;
    }
}

// Usage
const client = new NephoranStreamingClient('http://localhost:8080', 'your-api-key');
const stream = client.streamIntent(
    'Configure 5G network slice for IoT devices',
    (chunk) => console.log('Received:', chunk),
    (result) => console.log('Complete:', result),
    (error) => console.error('Error:', error)
);
```

## 📚 SDK and Client Libraries

### Official SDKs

#### Go SDK

```go
package main

import (
    "context"
    "log"
    
    "github.com/nephoran/intent-operator/pkg/client"
)

func main() {
    config := client.Config{
        BaseURL: "http://localhost:8080",
        APIKey:  "your-api-key",
        Timeout: time.Minute,
    }
    
    client := client.New(config)
    
    result, err := client.ProcessIntent(context.Background(), &client.IntentRequest{
        Intent: "Deploy AMF with 3 replicas",
        Metadata: map[string]string{
            "namespace": "telecom-core",
            "priority":  "high",
        },
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Result: %+v", result)
}
```

#### Python SDK

```python
from nephoran_client import NephoranClient, IntentRequest

# Initialize client
client = NephoranClient(
    base_url="http://localhost:8080",
    api_key="your-api-key"
)

# Process intent
result = client.process_intent(
    IntentRequest(
        intent="Deploy SMF with 5 replicas in production",
        metadata={
            "namespace": "telecom-core",
            "environment": "production"
        }
    )
)

print(f"Intent processed: {result.intent_id}")
print(f"Status: {result.status}")
```

#### JavaScript/TypeScript SDK

```typescript
import { NephoranClient, IntentRequest } from '@nephoran/intent-operator-sdk';

const client = new NephoranClient({
    baseUrl: 'http://localhost:8080',
    apiKey: 'your-api-key'
});

async function processIntent() {
    try {
        const result = await client.processIntent({
            intent: 'Scale UPF to 10 instances for high throughput',
            metadata: {
                namespace: 'telecom-core',
                priority: 'medium'
            }
        });
        
        console.log('Result:', result);
    } catch (error) {
        console.error('Error:', error);
    }
}
```

### Community SDKs

- **Rust SDK**: [nephoran-rust-client](https://github.com/nephoran/rust-client)
- **Java SDK**: [nephoran-java-client](https://github.com/nephoran/java-client)
- **C# SDK**: [NephoranClient.NET](https://github.com/nephoran/dotnet-client)

## 💡 Code Examples

### Network Function Deployment

```python
# Deploy 5G Core Network Functions
intents = [
    "Deploy AMF with 3 replicas for authentication",
    "Deploy SMF with 5 replicas for session management",
    "Deploy UPF with 8 replicas for user plane processing",
    "Configure network slice for enhanced mobile broadband"
]

for intent in intents:
    result = client.process_intent(intent, metadata={
        "namespace": "5g-core",
        "environment": "production"
    })
    print(f"Deployed: {result.structured_output['name']}")
```

### Scaling Operations

```python
# Automated scaling based on metrics
def auto_scale_network_functions():
    # Get current metrics
    metrics = client.get_performance_metrics()
    
    if metrics['cpu_usage'] > 0.8:
        result = client.process_intent(
            "Scale all network functions by 2 replicas for increased capacity"
        )
        return result
    
    if metrics['cpu_usage'] < 0.3:
        result = client.process_intent(
            "Scale down network functions by 1 replica to optimize resources"
        )
        return result
```

### Batch Processing

```python
import asyncio

async def process_intents_batch(intents):
    tasks = []
    for intent in intents:
        task = asyncio.create_task(
            client.process_intent_async(intent)
        )
        tasks.append(task)
    
    results = await asyncio.gather(*tasks)
    return results

# Usage
intents = [
    "Deploy AMF in us-east-1 region",
    "Deploy SMF in us-west-1 region", 
    "Deploy UPF in eu-central-1 region"
]

results = asyncio.run(process_intents_batch(intents))
```

### Real-time Monitoring

```python
def monitor_intent_processing():
    # Get system metrics
    metrics = client.get_metrics()
    
    print(f"Active sessions: {metrics['streaming']['active_streams']}")
    print(f"Cache hit rate: {metrics['cache']['hit_rate']:.2%}")
    print(f"Average latency: {metrics['performance']['average_latency']}")
    
    # Check circuit breakers
    cb_status = client.get_circuit_breaker_status()
    for name, status in cb_status.items():
        if status['state'] != 'closed':
            print(f"⚠️ Circuit breaker {name} is {status['state']}")
```

### Error Handling and Retry Logic

```python
import time
from exponential_backoff import exponential_backoff

@exponential_backoff(max_retries=3, base_delay=1)
def robust_intent_processing(intent):
    try:
        result = client.process_intent(intent)
        return result
    except CircuitBreakerOpenError:
        # Wait for circuit breaker to reset
        time.sleep(60)
        raise
    except RateLimitExceededError as e:
        # Respect rate limit
        time.sleep(e.retry_after)
        raise
    except Exception as e:
        print(f"Processing failed: {e}")
        raise

# Usage
try:
    result = robust_intent_processing("Deploy critical AMF service")
except Exception as e:
    print(f"Failed after retries: {e}")
```

## 🧪 Testing and Validation

### Unit Testing

```python
import unittest
from unittest.mock import patch, MagicMock

class TestNephoranClient(unittest.TestCase):
    def setUp(self):
        self.client = NephoranClient('http://localhost:8080', 'test-key')
    
    @patch('requests.Session.post')
    def test_process_intent_success(self, mock_post):
        # Mock successful response
        mock_response = MagicMock()
        mock_response.json.return_value = {
            'intent_id': 'test-123',
            'status': 'completed'
        }
        mock_post.return_value = mock_response
        
        result = self.client.process_intent('Test intent')
        
        self.assertEqual(result['status'], 'completed')
        mock_post.assert_called_once()
    
    def test_invalid_intent_handling(self):
        with self.assertRaises(ValueError):
            self.client.process_intent('')  # Empty intent
```

### Integration Testing

```python
class TestNephoranIntegration(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        # Start test cluster
        subprocess.run(['kind', 'create', 'cluster', '--name', 'test'])
        subprocess.run(['./deploy.sh', 'local'])
        
        cls.client = NephoranClient('http://localhost:8080')
    
    def test_end_to_end_deployment(self):
        # Process deployment intent
        result = self.client.process_intent(
            'Deploy test AMF with 1 replica'
        )
        
        # Verify intent was processed
        self.assertEqual(result['status'], 'completed')
        
        # Verify Kubernetes resources were created
        k8s_client = kubernetes.client.ApiClient()
        apps_v1 = kubernetes.client.AppsV1Api(k8s_client)
        
        deployments = apps_v1.list_namespaced_deployment(
            namespace='default'
        )
        
        amf_deployment = next(
            (d for d in deployments.items if 'amf' in d.metadata.name),
            None
        )
        self.assertIsNotNone(amf_deployment)
```

### Performance Testing

```python
import time
import concurrent.futures
import statistics

def performance_test():
    client = NephoranClient('http://localhost:8080')
    
    def process_test_intent():
        start_time = time.time()
        result = client.process_intent('Deploy test service')
        end_time = time.time()
        return end_time - start_time, result['status'] == 'completed'
    
    # Concurrent load test
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        futures = [executor.submit(process_test_intent) for _ in range(100)]
        results = [future.result() for future in futures]
    
    latencies = [r[0] for r in results]
    success_rate = sum(r[1] for r in results) / len(results)
    
    print(f"Average latency: {statistics.mean(latencies):.2f}s")
    print(f"P95 latency: {statistics.quantiles(latencies, n=20)[18]:.2f}s")
    print(f"Success rate: {success_rate:.2%}")

if __name__ == '__main__':
    performance_test()
```

## 🚀 Deployment Strategies

### Local Development

```bash
# Quick local setup
docker-compose up -d

# Or Kubernetes
kind create cluster
./deploy.sh local
```

### Staging Environment

```yaml
# staging-values.yaml
replicaCount: 2
resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
monitoring:
  enabled: true
```

```bash
helm upgrade --install nephoran-staging ./helm-chart \
  -f staging-values.yaml \
  -n nephoran-staging
```

### Production Deployment

```yaml
# production-values.yaml
replicaCount: 5
resources:
  requests:
    memory: "2Gi"
    cpu: "1000m"
  limits:
    memory: "4Gi"
    cpu: "2000m"
autoscaling:
  enabled: true
  minReplicas: 5
  maxReplicas: 50
security:
  enabled: true
  tls: true
monitoring:
  enabled: true
  alerts: true
backup:
  enabled: true
  schedule: "0 2 * * *"
```

### CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
name: Deploy Nephoran
on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Run tests
      run: |
        make test-integration
        make test-e2e
  
  deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
    - name: Deploy to staging
      run: |
        helm upgrade --install nephoran-staging ./helm-chart \
          -f staging-values.yaml
    
    - name: Run smoke tests
      run: |
        ./scripts/smoke-tests.sh
    
    - name: Deploy to production
      if: github.ref == 'refs/heads/main'
      run: |
        helm upgrade --install nephoran-prod ./helm-chart \
          -f production-values.yaml
```

## 📖 Best Practices

### Intent Design

```python
# ✅ Good: Specific, actionable intents
good_intents = [
    "Deploy AMF with 3 replicas in production namespace",
    "Scale SMF to 5 instances for increased session capacity",
    "Configure UPF with DPDK for high-throughput applications"
]

# ❌ Bad: Vague or incomplete intents
bad_intents = [
    "Deploy something",
    "Make it faster",
    "Fix the network"
]
```

### Error Handling

```python
class IntentProcessingError(Exception):
    def __init__(self, intent_id, error_code, message, retryable=False):
        self.intent_id = intent_id
        self.error_code = error_code
        self.retryable = retryable
        super().__init__(message)

def process_with_robust_error_handling(client, intent):
    try:
        return client.process_intent(intent)
    except CircuitBreakerOpenError:
        # Circuit breaker is open, wait before retry
        raise IntentProcessingError(
            intent_id=None,
            error_code="CIRCUIT_BREAKER_OPEN",
            message="Service temporarily unavailable",
            retryable=True
        )
    except Exception as e:
        # Log unexpected error
        logger.error(f"Unexpected error processing intent: {e}")
        raise
```

### Performance Optimization

```python
# Use caching for repeated intents
@lru_cache(maxsize=100)
def process_cached_intent(intent_hash):
    return client.process_intent(intent_hash)

# Batch processing for multiple intents
def process_intents_efficiently(intents):
    # Group similar intents
    grouped = group_similar_intents(intents)
    
    results = []
    for group in grouped:
        if len(group) > 1:
            # Use batch processing
            result = client.process_intents_batch(group)
        else:
            # Single intent processing
            result = client.process_intent(group[0])
        results.extend(result)
    
    return results
```

### Monitoring Integration

```python
import logging
from prometheus_client import Counter, Histogram, Gauge

# Metrics
INTENT_COUNTER = Counter('nephoran_intents_processed_total', 'Total intents processed')
PROCESSING_TIME = Histogram('nephoran_processing_duration_seconds', 'Intent processing time')
ACTIVE_SESSIONS = Gauge('nephoran_active_sessions', 'Active streaming sessions')

def monitored_intent_processing(client, intent):
    with PROCESSING_TIME.time():
        try:
            result = client.process_intent(intent)
            INTENT_COUNTER.inc()
            return result
        except Exception as e:
            logging.error(f"Intent processing failed: {e}")
            raise
```

## 🔧 Troubleshooting

### Common Issues

#### 1. High Latency

**Symptoms:** Request processing takes >5 seconds

**Diagnosis:**
```bash
# Check cache hit rates
curl http://localhost:8080/cache/stats

# Monitor circuit breaker status
curl http://localhost:8080/circuit-breaker/status

# Check system performance
curl http://localhost:8080/performance/metrics
```

**Solutions:**
- Enable caching: `enable_cache: true`
- Warm cache with popular queries
- Check circuit breaker thresholds
- Scale replicas if needed

#### 2. Authentication Failures

**Symptoms:** 401/403 errors

**Diagnosis:**
```bash
# Test API key
curl -H "X-API-Key: $API_KEY" http://localhost:8080/status

# Verify JWT token
jwt decode $JWT_TOKEN

# Check user permissions
curl -H "Authorization: Bearer $JWT_TOKEN" http://localhost:8080/auth/userinfo
```

**Solutions:**
- Verify API key is correct
- Check token expiration
- Ensure user has proper roles

#### 3. Circuit Breaker Issues

**Symptoms:** 503 Service Unavailable errors

**Diagnosis:**
```bash
# Check circuit breaker status
curl http://localhost:8080/circuit-breaker/status

# Review failure patterns
kubectl logs deployment/llm-processor | grep -i error
```

**Solutions:**
```bash
# Reset circuit breaker
curl -X POST http://localhost:8080/circuit-breaker/control \
  -d '{"action": "reset", "circuit_name": "llm-processor"}'

# Adjust thresholds if needed
export CIRCUIT_BREAKER_THRESHOLD=10
```

#### 4. Cache Problems

**Symptoms:** Low cache hit rates, high latency

**Diagnosis:**
```bash
# Check cache statistics
curl http://localhost:8080/cache/stats

# Verify Redis connectivity
kubectl exec deployment/redis -- redis-cli ping
```

**Solutions:**
```bash
# Clear and warm cache
curl -X DELETE http://localhost:8080/cache/clear
curl -X POST http://localhost:8080/cache/warm \
  -d '{"strategy": "popular_queries"}'
```

### Debug Tools

#### Health Check Script

```bash
#!/bin/bash
# health-check.sh - Comprehensive system health check

echo "=== Nephoran System Health Check ==="

# Check service availability
services=("llm-processor" "rag-api" "weaviate" "redis")
for service in "${services[@]}"; do
    if curl -s "http://$service/health" >/dev/null; then
        echo "✅ $service: Healthy"
    else
        echo "❌ $service: Unhealthy"
    fi
done

# Check Kubernetes resources
echo -e "\n=== Kubernetes Resources ==="
kubectl get pods -n nephoran-system

# Check metrics
echo -e "\n=== Performance Metrics ==="
curl -s http://localhost:8080/performance/metrics | jq '.cpu_usage, .memory_usage, .average_latency'

echo -e "\n=== Health Check Complete ==="
```

#### Log Analysis

```bash
# Analyze error patterns
kubectl logs deployment/llm-processor | grep -E "(ERROR|WARN)" | tail -20

# Monitor real-time logs
kubectl logs -f deployment/llm-processor

# Search for specific errors
kubectl logs deployment/llm-processor | grep "circuit breaker"
```

## 🌟 Community and Support

### Getting Help

- **Documentation**: [https://docs.nephoran.com](https://docs.nephoran.com)
- **GitHub Issues**: [Report bugs or request features](https://github.com/nephoran/intent-operator/issues)
- **Discord Community**: [Join our Discord](https://discord.gg/nephoran)
- **Stack Overflow**: Tag questions with `nephoran-intent-operator`

### Contributing

We welcome contributions! Here's how to get started:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes** and add tests
4. **Run the test suite**: `make test-all`
5. **Submit a pull request**

### Code of Conduct

Please read our [Code of Conduct](../../CODE_OF_CONDUCT.md) before participating in the community.

### Roadmap

- **Q2 2025**: Multi-tenancy support
- **Q3 2025**: Advanced AI/ML optimization
- **Q4 2025**: Edge computing integration
- **Q1 2026**: Global deployment features

---

## 📚 Additional Resources

### Tutorials

- [Building Your First Intent-Driven Application](../getting-started/quickstart.md)
- [Advanced RAG Integration](../knowledge-base/RAG-Pipeline-Implementation.md)
- [Production Deployment Guide](../operations/01-production-deployment-guide.md)
- [Monitoring and Observability](../operations/02-monitoring-alerting-runbooks.md)

### API Reference

- [Complete API Documentation](../api/API_REFERENCE.md)
- [OpenAPI Specification](../api/openapi-spec.yaml)
- [SDK Documentation](../api/sdk/)
- [Webhook Integration Guide](../api/webhook-validation-api.md)

### Examples Repository

- [Python Examples](https://github.com/nephoran/examples-python)
- [Go Examples](https://github.com/nephoran/examples-go)
- [JavaScript Examples](https://github.com/nephoran/examples-js)
- [Integration Examples](https://github.com/nephoran/examples-integration)

---

*Welcome to the future of intent-driven network automation! 🚀*
