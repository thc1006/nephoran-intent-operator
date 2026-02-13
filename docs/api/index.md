# API Reference

Complete API documentation for the Nephoran Intent Operator, including REST APIs, Custom Resource Definitions, and integration specifications.

## Overview

The Nephoran Intent Operator provides comprehensive APIs for managing intent-driven network automation:

- **REST APIs**: HTTP-based interfaces for external systems integration
- **Custom Resource Definitions (CRDs)**: Kubernetes-native resource management
- **gRPC APIs**: High-performance internal service communication
- **OpenAPI Specifications**: Machine-readable API documentation
- **Webhook APIs**: Event-driven integration endpoints

## API Categories

<div class="grid cards" markdown>

-   :material-api: **[REST APIs](API_REFERENCE.md)**

    ---

    HTTP REST endpoints for intent processing, status queries, and system management.

-   :material-kubernetes: **[Custom Resource Definitions](../crd-reference/docs/index.md)**

    ---

    Kubernetes CRDs for NetworkIntent, E2NodeSet, and ManagedElement resources.

-   :material-file-document: **[OpenAPI Specification](openapi-spec.yaml)**

    ---

    Complete OpenAPI 3.0 specifications for all HTTP APIs with examples.

-   :material-code-json: **[Examples](VECTOR-OPERATIONS-EXAMPLES.md)**

    ---

    Practical API usage examples, request/response samples, and integration patterns.

</div>

## Core APIs

### NetworkIntent API

The primary API for creating and managing network automation intents.

**Endpoint**: `/api/v1alpha1/networkintents`

=== "Create Intent"
    ```bash
    POST /api/v1alpha1/namespaces/default/networkintents
    Content-Type: application/json

    {
      "apiVersion": "nephoran.com/v1alpha1",
      "kind": "NetworkIntent",
      "metadata": {
        "name": "5g-core-deployment",
        "namespace": "default"
      },
      "spec": {
        "intent": "Deploy a complete 5G standalone core network with AMF, SMF, UPF, and NSSF",
        "priority": "high",
        "context": {
          "region": "us-west-2",
          "environment": "production"
        }
      }
    }
    ```

=== "Get Intent Status"
    ```bash
    GET /api/v1alpha1/namespaces/default/networkintents/5g-core-deployment
    ```

=== "List Intents"
    ```bash
    GET /api/v1alpha1/namespaces/default/networkintents
    ```

### LLM Processor API

API for direct language model processing and RAG retrieval.

**Endpoint**: `/api/v1alpha1/llm`

=== "Process Intent"
    ```bash
    POST /api/v1alpha1/llm/process
    Content-Type: application/json

    {
      "intent": "Deploy O-DU with massive MIMO support for dense urban areas",
      "context": {
        "deployment_type": "production",
        "region": "europe-west1"
      }
    }
    ```

=== "Streaming Process"
    ```bash
    POST /api/v1alpha1/llm/stream
    Content-Type: application/json
    Accept: text/event-stream

    {
      "intent": "Create URLLC network slice for industrial IoT applications",
      "streaming": true
    }
    ```

### RAG API

Semantic search and knowledge retrieval interface.

**Endpoint**: `/api/v1alpha1/rag`

=== "Search Knowledge"
    ```bash
    POST /api/v1alpha1/rag/search
    Content-Type: application/json

    {
      "query": "AMF configuration parameters",
      "limit": 10,
      "filters": {
        "document_type": "3gpp_spec"
      }
    }
    ```

=== "Get Context"
    ```bash
    POST /api/v1alpha1/rag/context
    Content-Type: application/json

    {
      "intent": "Deploy high-availability AMF instance",
      "max_chunks": 5
    }
    ```

## Authentication

All APIs support multiple authentication methods:

### OAuth2 / OIDC
```bash
# Bearer token authentication
Authorization: Bearer <access_token>
```

### Kubernetes Service Account
```bash
# Service account token
Authorization: Bearer <service_account_token>
```

### API Keys (Development)
```bash
# API key header
X-API-Key: <api_key>
```

## Rate Limiting

API rate limits are applied per client:

| API Category | Rate Limit | Burst |
|-------------|------------|-------|
| NetworkIntent CRUD | 100/min | 20 |
| LLM Processing | 20/min | 5 |
| RAG Search | 200/min | 50 |
| Status Queries | 500/min | 100 |

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```

## Error Handling

All APIs use consistent error response format:

```json
{
  "error": {
    "code": "INVALID_INTENT",
    "message": "Intent processing failed: insufficient context",
    "details": {
      "field": "spec.intent",
      "reason": "Intent text is too vague for processing"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "req_abc123"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request body or parameters |
| `UNAUTHORIZED` | 401 | Authentication required or invalid |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `INVALID_INTENT` | 422 | Intent cannot be processed |
| `PROCESSING_FAILED` | 500 | Internal processing error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |

## API Versioning

APIs follow semantic versioning with backward compatibility:

- **v1alpha1**: Current development version
- **v1beta1**: Stable beta (planned)
- **v1**: General availability (planned)

Version is specified in the URL path and API objects:
```
/api/v1alpha1/networkintents
```

## SDK and Client Libraries

Official SDKs are available for popular languages:

=== "Go"
    ```go
    import "github.com/nephoran/nephoran-intent-operator/pkg/client"

    client := client.NewClient("https://api.nephoran.example.com")
    
    intent := &v1alpha1.NetworkIntent{
        Spec: v1alpha1.NetworkIntentSpec{
            Intent: "Deploy 5G core for testing",
        },
    }
    
    result, err := client.NetworkIntents().Create(context.Background(), intent)
    ```

=== "Python"
    ```python
    from nephoran_client import NephoranClient

    client = NephoranClient("https://api.nephoran.example.com")
    
    intent = {
        "intent": "Deploy 5G core for testing",
        "priority": "medium"
    }
    
    result = client.network_intents.create(intent)
    ```

=== "JavaScript"
    ```javascript
    import { NephoranClient } from '@nephoran/client';

    const client = new NephoranClient('https://api.nephoran.example.com');
    
    const intent = await client.networkIntents.create({
        intent: 'Deploy 5G core for testing',
        priority: 'medium'
    });
    ```

## WebSocket APIs

Real-time streaming APIs for intent processing updates:

```javascript
const ws = new WebSocket('wss://api.nephoran.example.com/api/v1alpha1/stream');

ws.onmessage = function(event) {
    const update = JSON.parse(event.data);
    console.log('Intent update:', update);
};
```

## Monitoring and Observability

All APIs provide comprehensive monitoring:

- **Metrics**: Prometheus metrics for all endpoints
- **Tracing**: Distributed tracing with correlation IDs  
- **Logging**: Structured request/response logging
- **Health Checks**: `/healthz` and `/readyz` endpoints

Continue exploring the API documentation:
- [REST API Details →](API_REFERENCE.md)
- [CRD Specifications →](../crd-reference/docs/index.md)  
- [Integration Examples →](VECTOR-OPERATIONS-EXAMPLES.md)
