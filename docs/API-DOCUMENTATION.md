# Nephoran Intent Operator - API Documentation

## Overview

This document provides comprehensive API documentation for all services in the Nephoran Intent Operator system. The system includes multiple microservices that work together to process natural language intents and deploy O-RAN network functions.

## Service Architecture

```mermaid
graph TD
    A[Network Operators] --> B[LLM Processor Service]
    B --> C[RAG API Service]
    C --> D[Weaviate Vector Database]
    B --> E[NetworkIntent Controller]
    E --> F[Nephio Bridge Service]
    F --> G[O-RAN Adaptor Service]
    G --> H[O-RAN Network Functions]
```

## LLM Processor Service API

**Base URL**: `http://llm-processor:8080`  
**Service Version**: v2.0.0  
**Authentication**: OAuth2 (optional), API Key (optional)

### Core Endpoints

#### Process Intent
Process natural language intents and convert them to structured network operations.

**Endpoint**: `POST /process`  
**Authentication**: Required if `REQUIRE_AUTH=true`  
**Role**: Operator role required

**Request Body**:
```json
{
  "intent": "Deploy AMF with 3 replicas for network slice eMBB",
  "metadata": {
    "namespace": "telecom-core",
    "priority": "high",
    "user_id": "operator-001"
  }
}
```

**Response**:
```json
{
  "result": "Generated network configuration...",
  "status": "success",
  "processing_time": "1.234s",
  "request_id": "1641024000123456789",
  "service_version": "v2.0.0",
  "metadata": {
    "tokens_used": 1456,
    "model_used": "gpt-4o-mini",
    "confidence_score": 0.94
  }
}
```

**Error Response**:
```json
{
  "status": "error",
  "error": "Intent processing failed: OpenAI API timeout",
  "request_id": "1641024000123456789",
  "service_version": "v2.0.0",
  "processing_time": "30.0s"
}
```

**Response Codes**:
- `200 OK`: Intent processed successfully
- `400 Bad Request`: Invalid request body or missing intent
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Insufficient permissions
- `500 Internal Server Error`: Processing failed
- `503 Service Unavailable`: Circuit breaker open

#### Streaming Intent Processing
Real-time streaming processing with Server-Sent Events.

**Endpoint**: `POST /stream`  
**Authentication**: Required if `REQUIRE_AUTH=true`  
**Role**: Operator role required

**Request Body**:
```json
{
  "query": "Configure 5G network slice for enhanced mobile broadband",
  "intent_type": "network_configuration",
  "model_name": "gpt-4o-mini",
  "max_tokens": 2048,
  "enable_rag": true,
  "session_id": "session_123"
}
```

**Response**: Server-Sent Events stream
```
event: start
data: {"session_id":"session_123","status":"started"}

event: context_injection
data: {"type":"context_injection","content":"Context retrieved and injected","metadata":{"context_length":15420,"injection_time":"85ms"}}

event: chunk
data: {"type":"content","delta":"Based on the retrieved 3GPP TS 23.501 specifications...","timestamp":"2025-01-30T10:30:15Z","chunk_index":0}

event: completion
data: {"type":"completion","is_complete":true,"metadata":{"total_chunks":15,"total_bytes":8192,"processing_time":"2.3s"}}
```

### Health and Status Endpoints

#### Health Check (Liveness)
**Endpoint**: `GET /healthz`  
**Authentication**: None  

**Response**:
```json
{
  "status": "healthy",
  "version": "v2.0.0",
  "uptime": "2h34m18s",
  "timestamp": "2025-01-30T10:30:15Z"
}
```

#### Readiness Check
**Endpoint**: `GET /readyz`  
**Authentication**: None  

**Response**:
```json
{
  "status": "ready",
  "version": "v2.0.0",
  "dependencies": {
    "rag_api": "healthy",
    "circuit_breaker": "operational",
    "token_manager": "operational"
  },
  "timestamp": "2025-01-30T10:30:15Z"
}
```

#### Service Status
**Endpoint**: `GET /status`  
**Authentication**: Admin role required if auth enabled

**Response**:
```json
{
  "service": "llm-processor",
  "version": "v2.0.0",
  "uptime": "2h34m18s",
  "healthy": true,
  "ready": true,
  "backend_type": "rag",
  "model": "gpt-4o-mini",
  "rag_enabled": true,
  "timestamp": "2025-01-30T10:30:15Z"
}
```

### Metrics and Monitoring Endpoints

#### Comprehensive Metrics
**Endpoint**: `GET /metrics`  
**Authentication**: None  
**Format**: JSON (Prometheus format available)

**Response**:
```json
{
  "service": "llm-processor",
  "version": "v2.0.0",
  "uptime": "2h34m18s",
  "supported_models": ["gpt-4o", "gpt-4o-mini", "claude-3-haiku"],
  "circuit_breakers": {
    "llm-processor": {
      "state": "closed",
      "failure_count": 2,
      "success_count": 847,
      "failure_rate": 0.0023,
      "total_requests": 849,
      "last_failure_time": "2025-01-30T09:15:32Z"
    }
  },
  "streaming": {
    "active_streams": 3,
    "total_streams": 127,
    "completed_streams": 124,
    "average_stream_time": "2.1s",
    "total_bytes_streamed": 2048576
  },
  "context_builder": {
    "total_requests": 451,
    "successful_builds": 449,
    "average_build_time": "245ms",
    "average_context_size": 4096,
    "truncation_rate": 0.12
  }
}
```

#### Circuit Breaker Management
**Endpoint**: `GET /circuit-breaker/status`  
**Authentication**: Admin role required

**Response**:
```json
{
  "llm-processor": {
    "name": "llm-processor",
    "state": "closed",
    "failure_count": 2,
    "success_count": 847,
    "failure_rate": 0.0023,
    "total_requests": 849,
    "last_failure_time": "2025-01-30T09:15:32Z",
    "uptime": "2h34m18s"
  }
}
```

**Circuit Breaker Control**:
**Endpoint**: `POST /circuit-breaker/status`  
**Authentication**: Admin role required

**Request Body**:
```json
{
  "action": "reset",
  "name": "llm-processor"
}
```

**Actions**: `reset`, `force_open`

### OAuth2 Authentication Endpoints

#### Provider Login
**Endpoint**: `GET /auth/login/{provider}`  
**Parameters**: 
- `provider`: `azure`, `okta`, `keycloak`, `google`

**Response**: Redirect to OAuth2 provider

#### OAuth2 Callback
**Endpoint**: `GET /auth/callback/{provider}`  
**Parameters**: OAuth2 callback parameters

**Response**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "user_info": {
    "id": "user-123",
    "email": "operator@company.com",
    "roles": ["operator"]
  }
}
```

#### Token Refresh
**Endpoint**: `POST /auth/refresh`  
**Authentication**: Valid refresh token

**Request Body**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### User Information
**Endpoint**: `GET /auth/userinfo`  
**Authentication**: Valid access token

**Response**:
```json
{
  "id": "user-123",
  "email": "operator@company.com",
  "name": "Network Operator",
  "roles": ["operator"],
  "permissions": ["process_intents", "view_status"]
}
```

#### Logout
**Endpoint**: `POST /auth/logout`  
**Authentication**: Valid access token

**Response**:
```json
{
  "status": "success",
  "message": "Logged out successfully"
}
```

## RAG API Service

**Base URL**: `http://rag-api:5001`  
**Service Version**: v1.5.0  
**Authentication**: API Key (optional)

### Core Endpoints

#### Process Intent with RAG
**Endpoint**: `POST /process_intent`  
**Content-Type**: `application/json`

**Request Body**:
```json
{
  "intent": "Scale E2 nodes to 5 replicas",
  "intent_id": "intent-12345",
  "context": {
    "namespace": "telecom-ran",
    "user": "operator-001"
  }
}
```

**Response**:
```json
{
  "intent_id": "intent-12345",
  "original_intent": "Scale E2 nodes to 5 replicas",
  "structured_output": {
    "type": "NetworkFunctionScale",
    "name": "e2-nodes",
    "namespace": "telecom-ran",
    "spec": {
      "replicas": 5,
      "scaling_policy": "horizontal",
      "resources": {
        "cpu": "500m",
        "memory": "1Gi"
      }
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 1847.3,
    "tokens_used": 1245,
    "retrieval_score": 0.89,
    "confidence_score": 0.92,
    "cache_hit": false,
    "model_version": "gpt-4o-mini"
  },
  "timestamp": 1641024000.123
}
```

#### Query Vector Database
**Endpoint**: `POST /v1/query`  
**Content-Type**: `application/json`

**Request Body**:
```json
{
  "query": "AMF registration procedures",
  "limit": 10,
  "filters": {
    "category": "5G Core",
    "source": "3GPP"
  },
  "include_metadata": true
}
```

**Response**:
```json
{
  "results": [
    {
      "id": "doc-12345",
      "content": "AMF registration procedure involves...",
      "metadata": {
        "title": "3GPP TS 23.501",
        "category": "5G Core",
        "source": "3GPP",
        "version": "Release 17",
        "confidence": 0.94
      },
      "score": 0.89
    }
  ],
  "query_metadata": {
    "total_results": 25,
    "search_time_ms": 156,
    "query_id": "query-67890"
  }
}
```

### Knowledge Management Endpoints

#### Upload Document
**Endpoint**: `POST /knowledge/upload`  
**Content-Type**: `multipart/form-data`

**Request**: File upload with metadata
```bash
curl -X POST http://rag-api:5001/knowledge/upload \
  -F "file=@3gpp-ts-23501.pdf" \
  -F "metadata={\"category\":\"5G Core\",\"source\":\"3GPP\"}"
```

**Response**:
```json
{
  "document_id": "doc-98765",
  "filename": "3gpp-ts-23501.pdf",
  "size_bytes": 2048576,
  "chunks_created": 245,
  "processing_time_ms": 12456,
  "status": "processed"
}
```

#### Knowledge Base Statistics
**Endpoint**: `GET /knowledge/stats`

**Response**:
```json
{
  "total_documents": 1247,
  "total_chunks": 145789,
  "categories": {
    "5G Core": 456,
    "RAN": 389,
    "Transport": 234,
    "Management": 168
  },
  "sources": {
    "3GPP": 678,
    "O-RAN": 345,
    "ETSI": 156,
    "ITU": 68
  },
  "index_size_gb": 4.2,
  "last_updated": "2025-01-30T08:15:22Z"
}
```

### Health Endpoints

#### Health Check
**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy",
  "version": "v1.5.0",
  "dependencies": {
    "weaviate": "connected",
    "openai": "available",
    "redis": "connected"
  },
  "uptime": "1d 5h 23m",
  "timestamp": "2025-01-30T10:30:15Z"
}
```

#### Readiness Check
**Endpoint**: `GET /ready`

**Response**:
```json
{
  "status": "ready",
  "checks": {
    "vector_db": "operational",
    "embedding_service": "operational",
    "knowledge_base": "loaded"
  },
  "knowledge_base_stats": {
    "total_documents": 1247,
    "last_indexed": "2025-01-30T08:15:22Z"
  }
}
```

## NetworkIntent Controller API

**Base URL**: Kubernetes API Server  
**API Version**: `nephoran.com/v1`  
**Resource**: `networkintents`

### Custom Resource Definition

```yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: example-intent
  namespace: default
spec:
  # Natural language intent description
  intent: "Deploy AMF with 3 replicas for network slice eMBB"
  
  # Processing parameters
  priority: "high"  # low, medium, high, critical
  
  # Optional structured parameters (populated by LLM processing)
  parameters:
    networkFunction: "AMF"
    replicas: 3
    namespace: "telecom-core"
    resources:
      cpu: "2000m"
      memory: "4Gi"
    networkSlice:
      type: "eMBB"
      sla:
        latency: "20ms"
        throughput: "1Gbps"
  
  # Processing configuration
  config:
    llmModel: "gpt-4o-mini"
    timeout: "60s"
    retries: 3

status:
  # Processing phase
  phase: "Processed"  # Pending, Processing, Processed, Deployed, Failed
  
  # Processing results
  conditions:
    - type: "LLMProcessed"
      status: "True"
      reason: "IntentProcessedSuccessfully"
      message: "Intent successfully processed by LLM"
      lastTransitionTime: "2025-01-30T10:30:15Z"
    
    - type: "ParametersExtracted"
      status: "True"
      reason: "ParametersValid"
      message: "Structured parameters extracted and validated"
      lastTransitionTime: "2025-01-30T10:30:20Z"
  
  # Processing metadata
  processingInfo:
    llmModel: "gpt-4o-mini"
    processingTime: "2.3s"
    tokensUsed: 1456
    confidenceScore: 0.94
    requestID: "req-12345"
  
  # Generated resources
  generatedResources:
    - apiVersion: "apps/v1"
      kind: "Deployment"
      name: "amf-deployment"
      namespace: "telecom-core"
    - apiVersion: "v1"
      kind: "Service"
      name: "amf-service"
      namespace: "telecom-core"
```

### API Operations

#### Create NetworkIntent
```bash
kubectl apply -f networkintent.yaml
```

#### List NetworkIntents
```bash
kubectl get networkintents
```

#### Get NetworkIntent Details
```bash
kubectl describe networkintent example-intent
```

#### Update NetworkIntent
```bash
kubectl patch networkintent example-intent --type merge -p '{"spec":{"priority":"critical"}}'
```

#### Delete NetworkIntent
```bash
kubectl delete networkintent example-intent
```

## E2NodeSet Controller API

**Base URL**: Kubernetes API Server  
**API Version**: `nephoran.com/v1`  
**Resource**: `e2nodesets`

### Custom Resource Definition

```yaml
apiVersion: nephoran.com/v1
kind: E2NodeSet
metadata:
  name: simulated-gnbs
  namespace: default
spec:
  # Desired number of E2 node simulators
  replicas: 5
  
  # Node configuration template
  nodeTemplate:
    metadata:
      labels:
        type: "gNB-simulator"
        version: "v1.0"
    spec:
      # E2 connection parameters
      e2Connection:
        ricEndpoint: "http://near-rt-ric:8080"
        nodeId: "auto-generated"
        plmnId: "12345"
      
      # Resource requirements
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "1Gi"
      
      # Configuration parameters
      config:
        cellCount: 3
        bandwidth: "100MHz"
        txPower: "20dBm"

status:
  # Replica status
  replicas: 5
  readyReplicas: 5
  availableReplicas: 5
  
  # Scaling conditions
  conditions:
    - type: "ScalingComplete"
      status: "True"
      reason: "ReplicasCreated"
      message: "All E2 node replicas are running"
      lastTransitionTime: "2025-01-30T10:30:25Z"
  
  # Individual node status
  nodeStatus:
    - nodeId: "gnb-001"
      status: "Running"
      configMapName: "e2node-gnb-001"
      lastSeen: "2025-01-30T10:35:15Z"
    - nodeId: "gnb-002"
      status: "Running"
      configMapName: "e2node-gnb-002"
      lastSeen: "2025-01-30T10:35:14Z"
```

### API Operations

#### Create E2NodeSet
```bash
kubectl apply -f e2nodeset.yaml
```

#### Scale E2NodeSet
```bash
kubectl patch e2nodeset simulated-gnbs --type merge -p '{"spec":{"replicas":8}}'
```

#### Monitor Scaling
```bash
kubectl get e2nodesets -w
kubectl get configmaps -l e2nodeset=simulated-gnbs
```

## O-RAN Adaptor Service API

**Base URL**: `http://oran-adaptor:8082`  
**Service Version**: v1.0.0  
**Authentication**: API Key

### A1 Interface (Policy Management)

#### Create Policy Type
**Endpoint**: `POST /a1/policy-types`

**Request Body**:
```json
{
  "policyTypeId": "1000",
  "name": "QoS Policy",
  "description": "Quality of Service policy for network slices",
  "schema": {
    "type": "object",
    "properties": {
      "sliceId": {"type": "string"},
      "qosParameters": {
        "type": "object",
        "properties": {
          "latency": {"type": "number"},
          "throughput": {"type": "number"}
        }
      }
    }
  }
}
```

#### Create Policy Instance
**Endpoint**: `POST /a1/policy-types/{policyTypeId}/policies`

**Request Body**:
```json
{
  "policyInstanceId": "policy-001",
  "policyData": {
    "sliceId": "embb-001",
    "qosParameters": {
      "latency": 20,
      "throughput": 1000
    }
  }
}
```

### O1 Interface (Fault/Configuration Management)

#### Get Fault Management Status
**Endpoint**: `GET /o1/fault-management/alarms`

**Response**:
```json
{
  "alarms": [
    {
      "alarmId": "alarm-001",
      "severity": "major",
      "source": "amf-001",
      "description": "High CPU utilization",
      "timestamp": "2025-01-30T10:30:15Z",
      "status": "active"
    }
  ],
  "totalCount": 1
}
```

#### Update Configuration
**Endpoint**: `POST /o1/configuration-management/config`

**Request Body**:
```json
{
  "managedElement": "amf-001",
  "configData": {
    "maxConnections": 1000,
    "logLevel": "INFO",
    "healthCheckInterval": "30s"
  }
}
```

### O2 Interface (Cloud Infrastructure)

#### Deploy Infrastructure
**Endpoint**: `POST /o2/infrastructure/deploy`

**Request Body**:
```json
{
  "deploymentName": "5g-core-infrastructure",
  "resources": [
    {
      "type": "VirtualMachine",
      "specs": {
        "cpu": 8,
        "memory": "32Gi",
        "storage": "500Gi"
      },
      "count": 3
    }
  ],
  "networkConfig": {
    "subnet": "10.0.0.0/24",
    "securityGroups": ["sg-telecom"]
  }
}
```

## Error Handling and Status Codes

### Common HTTP Status Codes

- **200 OK**: Request successful
- **201 Created**: Resource created successfully
- **202 Accepted**: Request accepted for async processing
- **400 Bad Request**: Invalid request parameters
- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Resource not found
- **409 Conflict**: Resource conflict (e.g., duplicate name)
- **422 Unprocessable Entity**: Valid request but semantic errors
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Internal server error
- **502 Bad Gateway**: Upstream service error
- **503 Service Unavailable**: Service temporarily unavailable
- **504 Gateway Timeout**: Upstream service timeout

### Error Response Format

All services use a consistent error response format:

```json
{
  "error": {
    "code": "INTENT_PROCESSING_FAILED",
    "message": "Failed to process network intent",
    "details": {
      "reason": "OpenAI API timeout",
      "intent": "Deploy AMF with 3 replicas",
      "retryable": true,
      "retry_after": 30
    },
    "timestamp": "2025-01-30T10:30:15Z",
    "request_id": "req-12345"
  }
}
```

### Error Codes Reference

| Code | Description | Service | Retryable |
|------|-------------|---------|-----------|
| `INTENT_PROCESSING_FAILED` | LLM processing failed | LLM Processor | Yes |
| `RAG_SERVICE_UNAVAILABLE` | RAG API unavailable | LLM Processor | Yes |
| `INVALID_INTENT_FORMAT` | Invalid intent structure | LLM Processor | No |
| `AUTHENTICATION_FAILED` | Authentication failed | All Services | No |
| `RATE_LIMIT_EXCEEDED` | Too many requests | All Services | Yes |
| `CIRCUIT_BREAKER_OPEN` | Circuit breaker active | LLM Processor | Yes |
| `VECTOR_DB_ERROR` | Vector database error | RAG API | Yes |
| `KNOWLEDGE_BASE_EMPTY` | No documents indexed | RAG API | No |
| `CRD_VALIDATION_FAILED` | Invalid CRD specification | Controllers | No |
| `SCALING_FAILED` | E2NodeSet scaling failed | E2NodeSet Controller | Yes |
| `ORAN_INTERFACE_ERROR` | O-RAN interface error | O-RAN Adaptor | Yes |

## Rate Limiting

All services implement rate limiting to ensure fair usage and system stability.

### Default Rate Limits

| Service | Endpoint | Rate Limit | Burst |
|---------|----------|------------|--------|
| LLM Processor | `/process` | 60 req/min | 10 |
| LLM Processor | `/stream` | 30 req/min | 5 |
| RAG API | `/process_intent` | 100 req/min | 20 |
| RAG API | `/v1/query` | 200 req/min | 50 |
| O-RAN Adaptor | All endpoints | 120 req/min | 30 |

### Rate Limit Headers

All services include rate limiting information in response headers:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1641024600
Retry-After: 30
```

## SDK and Client Libraries

### Go Client Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/thc1006/nephoran-intent-operator/pkg/client"
)

func main() {
    // Initialize client
    config := client.Config{
        LLMProcessorURL: "http://llm-processor:8080",
        RAGAPIUrl:      "http://rag-api:5001",
        APIKey:         "your-api-key",
    }
    
    client := client.NewNephoranClient(config)
    
    // Process intent
    result, err := client.ProcessIntent(context.Background(), &client.IntentRequest{
        Intent: "Deploy AMF with 3 replicas for network slice eMBB",
        Metadata: map[string]string{
            "namespace": "telecom-core",
            "priority":  "high",
        },
    })
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Result: %s\n", result.Result)
}
```

### Python Client Example

```python
import requests
import json

class NephoranClient:
    def __init__(self, base_url, api_key=None):
        self.base_url = base_url
        self.api_key = api_key
        self.session = requests.Session()
        if api_key:
            self.session.headers.update({'Authorization': f'Bearer {api_key}'})
    
    def process_intent(self, intent, metadata=None):
        payload = {
            'intent': intent,
            'metadata': metadata or {}
        }
        
        response = self.session.post(
            f'{self.base_url}/process',
            json=payload,
            timeout=60
        )
        response.raise_for_status()
        return response.json()

# Usage
client = NephoranClient('http://llm-processor:8080', 'your-api-key')
result = client.process_intent(
    'Deploy AMF with 3 replicas for network slice eMBB',
    {'namespace': 'telecom-core', 'priority': 'high'}
)
print(f"Result: {result['result']}")
```

## Monitoring and Observability

### Prometheus Metrics

All services expose Prometheus metrics at `/metrics` endpoint:

#### LLM Processor Metrics
- `nephoran_llm_requests_total`: Total LLM requests
- `nephoran_llm_request_duration_seconds`: Request duration histogram
- `nephoran_llm_tokens_used_total`: Total tokens consumed
- `nephoran_llm_cache_hits_total`: Cache hit counter
- `nephoran_streaming_active_sessions`: Active streaming sessions

#### RAG API Metrics
- `nephoran_rag_queries_total`: Total RAG queries
- `nephoran_rag_query_duration_seconds`: Query duration histogram
- `nephoran_rag_documents_indexed_total`: Total documents indexed
- `nephoran_rag_vector_search_latency_seconds`: Vector search latency

#### Controller Metrics
- `nephoran_networkintent_processing_duration_seconds`: Intent processing time
- `nephoran_e2nodeset_scaling_duration_seconds`: E2NodeSet scaling time
- `nephoran_controller_reconcile_errors_total`: Controller reconciliation errors

### Health Check Endpoints

All services provide standardized health check endpoints:
- `/healthz`: Liveness probe (basic service health)
- `/readyz`: Readiness probe (service ready to handle requests)
- `/metrics`: Prometheus metrics endpoint

### Distributed Tracing

Services support distributed tracing with Jaeger integration. Trace spans are automatically created for:
- Intent processing requests
- RAG API calls
- Vector database queries
- Controller reconciliation loops
- O-RAN interface communications

## Conclusion

This comprehensive API documentation provides detailed information for integrating with all Nephoran Intent Operator services. For additional examples, troubleshooting, and advanced configuration, refer to the service-specific documentation and the main project README.

For support and questions, please refer to the project's GitHub repository or contact the development team.