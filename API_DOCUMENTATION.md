# Nephoran Intent Operator - API Documentation

## Overview

This document provides comprehensive API documentation for the Nephoran Intent Operator system. The system consists of four main services that work together to process natural language intents and deploy O-RAN network functions through GitOps workflows.

## Service Architecture

```mermaid
graph TD
    A[Natural Language Intent] --> B[LLM Processor Service :8080]
    B --> C[RAG API Service :5001]
    C --> D[Weaviate Vector Database]
    B --> E[NetworkIntent Controller]
    E --> F[Nephio Bridge Service :8081]
    F --> G[O-RAN Adaptor Service :8082]
    G --> H[O-RAN Network Functions]
```

## LLM Processor Service API

**Base URL**: `http://llm-processor:8080`
**Service**: Enterprise-grade LLM processing with circuit breakers and rate limiting
**Authentication**: OAuth2, API Key (configurable)

### Core Endpoints

#### Process Natural Language Intent
Convert natural language intents into structured network operations.

**Endpoint**: `POST /process`
**Authentication**: Required if `REQUIRE_AUTH=true`

**Request Body**:
```json
{
  "intent": "Deploy AMF with 3 replicas for network slice eMBB with high throughput requirements",
  "metadata": {
    "namespace": "telecom-core",
    "priority": "high",
    "user_id": "operator-001"
  },
  "config": {
    "model": "gpt-4o-mini",
    "max_tokens": 2048,
    "enable_rag": true
  }
}
```

**Response**:
```json
{
  "result": "Generated Kubernetes YAML configuration for AMF deployment with 3 replicas, including Service, Deployment, and ConfigMap resources with optimized settings for eMBB network slice requirements.",
  "status": "success",
  "processing_time": "2.345s",
  "request_id": "req-1641024000123456789",
  "metadata": {
    "tokens_used": 1456,
    "model_used": "gpt-4o-mini",
    "confidence_score": 0.94,
    "rag_context_used": true,
    "context_length": 4096
  },
  "structured_output": {
    "action": "deploy",
    "resources": [
      {
        "type": "Deployment",
        "name": "amf-deployment",
        "replicas": 3,
        "namespace": "telecom-core"
      },
      {
        "type": "Service",
        "name": "amf-service", 
        "namespace": "telecom-core"
      }
    ]
  }
}
```

**Error Response**:
```json
{
  "status": "error",
  "error": "Intent processing failed: OpenAI API timeout after 30s",
  "request_id": "req-1641024000123456789",
  "processing_time": "30.0s",
  "error_code": "LLM_TIMEOUT",
  "retryable": true,
  "retry_after": 30
}
```

**Response Codes**:
- `200 OK`: Intent processed successfully
- `400 Bad Request`: Invalid request format or missing required fields
- `401 Unauthorized`: Authentication required or invalid credentials
- `403 Forbidden`: Insufficient permissions for this operation
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Internal processing error
- `503 Service Unavailable`: Circuit breaker open or service overloaded

#### Streaming Intent Processing
Real-time streaming processing with Server-Sent Events for large intents.

**Endpoint**: `POST /stream`
**Authentication**: Required if `REQUIRE_AUTH=true`

**Request Body**:
```json
{
  "query": "Configure complete 5G network slice for enhanced mobile broadband with AMF, SMF, UPF, and network optimization",
  "intent_type": "network_configuration",
  "model_name": "gpt-4o-mini",
  "max_tokens": 4096,
  "enable_rag": true,
  "session_id": "session_demo_001"
}
```

**Response**: Server-Sent Events stream
```
event: start
data: {"session_id":"session_demo_001","status":"started","timestamp":"2025-01-30T10:30:15Z"}

event: context_injection
data: {"type":"context_injection","content":"Retrieved 3GPP TS 23.501 and O-RAN specifications","metadata":{"context_length":15420,"injection_time":"85ms"}}

event: chunk
data: {"type":"content","delta":"Based on 3GPP specifications, configuring 5G network slice for eMBB requires the following components:\n\n1. Access and Mobility Management Function (AMF)","timestamp":"2025-01-30T10:30:16Z","chunk_index":0}

event: chunk  
data: {"type":"content","delta":"2. Session Management Function (SMF) with policy control","timestamp":"2025-01-30T10:30:16Z","chunk_index":1}

event: completion
data: {"type":"completion","is_complete":true,"metadata":{"total_chunks":25,"total_bytes":16384,"processing_time":"4.7s","tokens_used":3456}}
```

### Health and Status Endpoints

#### Health Check (Liveness Probe)
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
    "circuit_breaker": "closed",
    "token_manager": "operational",
    "openai_api": "available"
  },
  "timestamp": "2025-01-30T10:30:15Z"
}
```

#### Service Metrics
**Endpoint**: `GET /metrics`
**Authentication**: None (Prometheus format)

**Response**: Prometheus metrics format including:
- `nephoran_llm_requests_total`: Total LLM requests processed
- `nephoran_llm_request_duration_seconds`: Request processing duration
- `nephoran_llm_tokens_used_total`: Total tokens consumed
- `nephoran_llm_cache_hits_total`: Cache hit counter
- `nephoran_circuit_breaker_state`: Circuit breaker state (0=closed, 1=open, 2=half-open)

### Circuit Breaker Management

#### Circuit Breaker Status
**Endpoint**: `GET /circuit-breaker/status`
**Authentication**: Admin role required

**Response**:
```json
{
  "llm-processor": {
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

#### Circuit Breaker Control
**Endpoint**: `POST /circuit-breaker/control`
**Authentication**: Admin role required

**Request Body**:
```json
{
  "action": "reset",  // "reset" or "force_open"
  "circuit_name": "llm-processor"
}
```

## RAG API Service

**Base URL**: `http://rag-api:5001`
**Service**: Vector database integration with Weaviate for telecom knowledge
**Authentication**: API Key (optional)

### Core Endpoints

#### Process Intent with RAG Enhancement
**Endpoint**: `POST /process_intent`
**Content-Type**: `application/json`

**Request Body**:
```json
{
  "intent": "Scale E2 nodes to 5 replicas for increased capacity",
  "intent_id": "intent-12345",
  "context": {
    "namespace": "telecom-ran",
    "user": "operator-001",
    "environment": "production"
  }
}
```

**Response**:
```json
{
  "intent_id": "intent-12345",
  "original_intent": "Scale E2 nodes to 5 replicas for increased capacity",
  "enhanced_context": "Based on O-RAN E2 interface specifications and scaling best practices...",
  "structured_output": {
    "action": "scale",
    "resource_type": "E2NodeSet",
    "name": "e2-nodes",
    "namespace": "telecom-ran",
    "spec": {
      "replicas": 5,
      "scaling_policy": "horizontal",
      "resources": {
        "cpu": "500m",
        "memory": "1Gi"
      },
      "e2_configuration": {
        "ric_endpoint": "http://near-rt-ric:8080",
        "service_models": ["KPM", "RC"]
      }
    }
  },
  "status": "completed",
  "metrics": {
    "processing_time_ms": 1847.3,
    "context_retrieval_time_ms": 156.2,
    "llm_processing_time_ms": 1691.1,
    "confidence_score": 0.92,
    "retrieval_score": 0.89,
    "documents_retrieved": 12,
    "cache_hit": false
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
  "query": "AMF registration procedures 5G core network",
  "limit": 10,
  "filters": {
    "category": "5G Core",
    "source": "3GPP",
    "standard": "TS 23.501"
  },
  "include_metadata": true,
  "similarity_threshold": 0.8
}
```

**Response**:
```json
{
  "results": [
    {
      "id": "doc-3gpp-ts23501-amf-001",
      "content": "The AMF receives the registration request from the UE via the RAN. The AMF shall authenticate and authorize the UE according to the subscription information retrieved from UDM...",
      "metadata": {
        "title": "3GPP TS 23.501 - System architecture for the 5G System",
        "section": "4.2.2.2.2 Registration procedure",
        "category": "5G Core",
        "source": "3GPP",
        "standard": "TS 23.501",
        "version": "Release 17",
        "page": 45,
        "confidence": 0.94
      },
      "score": 0.89
    },
    {
      "id": "doc-3gpp-ts23501-amf-002", 
      "content": "During the registration procedure, the AMF performs the following functions: UE authentication, authorization based on subscription...",
      "metadata": {
        "title": "3GPP TS 23.501 - AMF functionality",
        "section": "6.2.1 AMF services",
        "category": "5G Core",
        "confidence": 0.91
      },
      "score": 0.87
    }
  ],
  "query_metadata": {
    "total_results": 25,
    "search_time_ms": 156,
    "query_id": "query-67890",
    "embedding_time_ms": 45,
    "vector_search_time_ms": 111
  }
}
```

### Knowledge Management

#### Upload Document
**Endpoint**: `POST /knowledge/upload`
**Content-Type**: `multipart/form-data`

**cURL Example**:
```bash
curl -X POST http://rag-api:5001/knowledge/upload \
  -F "file=@3gpp-ts-23501.pdf" \
  -F "metadata={\"category\":\"5G Core\",\"source\":\"3GPP\",\"standard\":\"TS 23.501\",\"version\":\"Release 17\"}"
```

**Response**:
```json
{
  "document_id": "doc-98765",
  "filename": "3gpp-ts-23501.pdf",
  "size_bytes": 2048576,
  "chunks_created": 245,
  "processing_time_ms": 12456,
  "status": "processed",
  "metadata": {
    "category": "5G Core",
    "source": "3GPP",
    "standard": "TS 23.501",
    "version": "Release 17"
  }
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
  "standards": {
    "TS 23.501": 123,
    "TS 38.401": 89,
    "O-RAN.WG1.Use-Cases": 67
  },
  "index_size_gb": 4.2,
  "last_updated": "2025-01-30T08:15:22Z"
}
```

### Health Endpoints

#### Health Check
**Endpoint**: `GET /healthz`

**Response**:
```json
{
  "status": "healthy",
  "version": "v1.5.0",
  "uptime": "1d 5h 23m",
  "timestamp": "2025-01-30T10:30:15Z"
}
```

#### Readiness Check
**Endpoint**: `GET /readyz`

**Response**:
```json
{
  "status": "ready",
  "dependencies": {
    "weaviate": "connected",
    "openai": "available",
    "redis": "connected"
  },
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

## Kubernetes CRD APIs

### NetworkIntent Resource

**API Version**: `nephoran.com/v1`
**Kind**: `NetworkIntent`
**Scope**: Namespaced

#### Example NetworkIntent

```yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: deploy-amf-embb
  namespace: telecom-core
spec:
  # Natural language intent
  intent: "Deploy AMF with 3 replicas for network slice eMBB with high throughput requirements"
  
  # Processing priority
  priority: "high"  # low, medium, high, critical
  
  # Optional pre-structured parameters (will be populated by LLM)
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
    enableRag: true

status:
  # Current processing phase
  phase: "Processed"  # Pending, Processing, Processed, Deployed, Failed
  
  # Status conditions
  conditions:
    - type: "LLMProcessed"
      status: "True"
      reason: "IntentProcessedSuccessfully"
      message: "Intent successfully processed with RAG enhancement"
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
    ragEnhanced: true
  
  # Generated Kubernetes resources
  generatedResources:
    - apiVersion: "apps/v1"
      kind: "Deployment"
      name: "amf-deployment"
      namespace: "telecom-core"
    - apiVersion: "v1"
      kind: "Service"
      name: "amf-service"
      namespace: "telecom-core"
    - apiVersion: "v1"
      kind: "ConfigMap"
      name: "amf-config"
      namespace: "telecom-core"
```

#### NetworkIntent Operations

```bash
# Create NetworkIntent
kubectl apply -f networkintent.yaml

# List all NetworkIntents
kubectl get networkintents

# Get detailed status
kubectl describe networkintent deploy-amf-embb

# Watch for changes
kubectl get networkintents -w

# Update priority
kubectl patch networkintent deploy-amf-embb --type merge -p '{"spec":{"priority":"critical"}}'

# Delete NetworkIntent
kubectl delete networkintent deploy-amf-embb
```

### E2NodeSet Resource

**API Version**: `nephoran.com/v1alpha1`
**Kind**: `E2NodeSet`
**Scope**: Namespaced

#### Example E2NodeSet

```yaml
apiVersion: nephoran.com/v1alpha1
kind: E2NodeSet
metadata:
  name: simulated-gnbs
  namespace: telecom-ran
spec:
  # Desired number of E2 node simulators
  replicas: 5
  
  # E2 node configuration template
  nodeTemplate:
    metadata:
      labels:
        type: "gNB-simulator"
        version: "v1.0"
        environment: "production"
    spec:
      # E2 interface connection parameters
      e2Connection:
        ricEndpoint: "http://near-rt-ric:8080"
        nodeId: "auto-generated"  # Will be auto-generated per replica
        plmnId: "12345"
        ricId: "gnb-ric-001"
      
      # Resource requirements per node
      resources:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "1Gi"
      
      # gNB configuration parameters
      config:
        cellCount: 3
        bandwidth: "100MHz"
        txPower: "20dBm"
        frequency: "3500MHz"
        serviceModels: ["KPM", "RC"]

status:
  # Replica status
  replicas: 5
  readyReplicas: 5
  availableReplicas: 5
  unavailableReplicas: 0
  
  # Status conditions
  conditions:
    - type: "ScalingComplete"
      status: "True"
      reason: "ReplicasCreated"
      message: "All E2 node replicas are running and connected"
      lastTransitionTime: "2025-01-30T10:30:25Z"
    
    - type: "E2ConnectionHealthy"
      status: "True"
      reason: "AllNodesConnected"
      message: "All E2 nodes successfully connected to Near-RT RIC"
      lastTransitionTime: "2025-01-30T10:31:00Z"
  
  # Individual node status
  nodeStatus:
    - nodeId: "gnb-001"
      status: "Running"
      configMapName: "e2node-gnb-001"
      e2ConnectionStatus: "Connected"
      lastSeen: "2025-01-30T10:35:15Z"
      metrics:
        cpu: "150m"
        memory: "312Mi"
    - nodeId: "gnb-002"
      status: "Running" 
      configMapName: "e2node-gnb-002"
      e2ConnectionStatus: "Connected"
      lastSeen: "2025-01-30T10:35:14Z"
      metrics:
        cpu: "145m"
        memory: "298Mi"
```

#### E2NodeSet Operations

```bash
# Create E2NodeSet
kubectl apply -f e2nodeset.yaml

# List E2NodeSets
kubectl get e2nodesets

# Scale E2NodeSet
kubectl patch e2nodeset simulated-gnbs --type merge -p '{"spec":{"replicas":8}}'

# Monitor scaling progress
kubectl get e2nodesets -w

# Check generated ConfigMaps
kubectl get configmaps -l e2nodeset=simulated-gnbs

# Get detailed status
kubectl describe e2nodeset simulated-gnbs

# Check individual node configurations
kubectl get configmap e2node-gnb-001 -o yaml
```

## O-RAN Adaptor Service API

**Base URL**: `http://oran-adaptor:8082`
**Service**: O-RAN interface implementations (A1, O1, O2)
**Authentication**: API Key

### A1 Interface (Policy Management)

#### Create Policy Type
**Endpoint**: `POST /a1/policy-types`
**Authentication**: Required

**Request Body**:
```json
{
  "policyTypeId": "1001",
  "name": "QoS Management Policy",
  "description": "Quality of Service policy for network slice management",
  "version": "1.0.0",
  "schema": {
    "type": "object",
    "properties": {
      "sliceId": {
        "type": "string",
        "description": "Network slice identifier"
      },
      "qosParameters": {
        "type": "object",
        "properties": {
          "latency": {
            "type": "number",
            "minimum": 1,
            "maximum": 1000,
            "description": "Maximum latency in milliseconds"
          },
          "throughput": {
            "type": "number",
            "minimum": 1,
            "description": "Minimum throughput in Mbps"
          },
          "reliability": {
            "type": "number",
            "minimum": 0.9,
            "maximum": 1.0,
            "description": "Service reliability (0.9-1.0)"
          }
        },
        "required": ["latency", "throughput"]
      }
    },
    "required": ["sliceId", "qosParameters"]
  }
}
```

**Response**:
```json
{
  "policyTypeId": "1001",
  "status": "created",
  "message": "Policy type created successfully",
  "timestamp": "2025-01-30T10:30:15Z"
}
```

#### Create Policy Instance
**Endpoint**: `POST /a1/policy-types/{policyTypeId}/policies`
**Authentication**: Required

**Request Body**:
```json
{
  "policyInstanceId": "qos-policy-embb-001",
  "policyData": {
    "sliceId": "embb-slice-001",
    "qosParameters": {
      "latency": 20,
      "throughput": 1000,
      "reliability": 0.99
    }
  },
  "metadata": {
    "description": "QoS policy for eMBB slice in urban area",
    "priority": "high",
    "environment": "production"
  }
}
```

#### List Policy Instances
**Endpoint**: `GET /a1/policy-types/{policyTypeId}/policies`

**Response**:
```json
{
  "policies": [
    {
      "policyInstanceId": "qos-policy-embb-001",
      "status": "active",
      "created": "2025-01-30T10:30:15Z",
      "lastModified": "2025-01-30T10:30:15Z"
    }
  ],
  "total": 1
}
```

### O1 Interface (Fault/Configuration Management)

#### Get Fault Management Alarms
**Endpoint**: `GET /o1/fault-management/alarms`
**Authentication**: Required

**Query Parameters**:
- `severity`: Filter by severity (critical, major, minor, warning)
- `status`: Filter by status (active, cleared, acknowledged)
- `source`: Filter by source component
- `limit`: Maximum number of results (default: 100)

**Response**:
```json
{
  "alarms": [
    {
      "alarmId": "alarm-001",
      "severity": "major",
      "source": "amf-deployment-001",
      "managedElement": "amf-001",
      "alarmType": "processingError",
      "description": "High CPU utilization detected",
      "additionalText": "CPU usage exceeded 85% threshold for 5 minutes",
      "timestamp": "2025-01-30T10:30:15Z",
      "status": "active",
      "acknowledgedBy": null,
      "clearedTime": null,
      "correlationId": "corr-123"
    },
    {
      "alarmId": "alarm-002", 
      "severity": "minor",
      "source": "smf-deployment-002",
      "managedElement": "smf-002",
      "alarmType": "communicationAlarm",
      "description": "Intermittent connectivity to UDM",
      "timestamp": "2025-01-30T10:25:30Z",
      "status": "acknowledged",
      "acknowledgedBy": "operator-001",
      "acknowledgedTime": "2025-01-30T10:26:00Z"
    }
  ],
  "totalCount": 2,
  "activeCount": 1,
  "acknowledgedCount": 1
}
```

#### Acknowledge Alarm
**Endpoint**: `POST /o1/fault-management/alarms/{alarmId}/acknowledge`

**Request Body**:
```json
{
  "acknowledgedBy": "operator-001",
  "comment": "Investigating high CPU usage issue"
}
```

#### Update Configuration
**Endpoint**: `POST /o1/configuration-management/config`
**Authentication**: Required

**Request Body**:
```json
{
  "managedElement": "amf-001",
  "configData": {
    "maxConnections": 1000,
    "logLevel": "INFO",
    "healthCheckInterval": "30s",
    "resourceLimits": {
      "cpu": "2000m",
      "memory": "4Gi"
    },
    "networkSettings": {
      "httpTimeout": "60s",
      "maxRetries": 3
    }
  },
  "metadata": {
    "modifiedBy": "operator-001",
    "reason": "Performance optimization"
  }
}
```

**Response**:
```json
{
  "configurationId": "config-12345",
  "status": "applied",
  "appliedAt": "2025-01-30T10:30:15Z",
  "changes": [
    {
      "parameter": "maxConnections",
      "oldValue": 500,
      "newValue": 1000
    },
    {
      "parameter": "resourceLimits.cpu",
      "oldValue": "1000m", 
      "newValue": "2000m"
    }
  ]
}
```

### O2 Interface (Cloud Infrastructure Management)

#### Deploy Infrastructure
**Endpoint**: `POST /o2/infrastructure/deploy`
**Authentication**: Required

**Request Body**:
```json
{
  "deploymentName": "5g-core-infrastructure-prod",
  "resources": [
    {
      "type": "VirtualMachine",
      "name": "amf-vm-pool",
      "specs": {
        "cpu": 8,
        "memory": "32Gi",
        "storage": "500Gi",
        "networkInterfaces": 2
      },
      "count": 3,
      "availability": {
        "zone": "us-central1-a",
        "antiAffinity": true
      }
    },
    {
      "type": "LoadBalancer",
      "name": "amf-lb",
      "config": {
        "algorithm": "round_robin",
        "healthCheck": {
          "path": "/healthz",
          "interval": "30s"
        }
      }
    }
  ],
  "networkConfig": {
    "vpc": "telecom-vpc",
    "subnet": "10.0.0.0/24",
    "securityGroups": ["sg-telecom-core", "sg-5g-amf"],
    "routing": {
      "enableNAT": true,
      "internetAccess": false
    }
  },
  "tags": {
    "environment": "production",
    "project": "5g-core",
    "owner": "telecom-team"
  }
}
```

**Response**:
```json
{
  "deploymentId": "deploy-67890",
  "status": "deploying",
  "estimatedCompletionTime": "15m",
  "resources": [
    {
      "resourceId": "vm-001",
      "type": "VirtualMachine",
      "status": "creating",
      "publicIP": null,
      "privateIP": "10.0.0.10"
    }
  ],
  "networkInfo": {
    "vpcId": "vpc-12345",
    "subnetId": "subnet-67890"
  }
}
```

#### Get Infrastructure Status
**Endpoint**: `GET /o2/infrastructure/deployments/{deploymentId}`

**Response**:
```json
{
  "deploymentId": "deploy-67890",
  "status": "deployed",
  "createdAt": "2025-01-30T10:30:15Z",
  "completedAt": "2025-01-30T10:45:30Z",
  "resources": [
    {
      "resourceId": "vm-001",
      "type": "VirtualMachine",
      "status": "running",
      "publicIP": "203.0.113.10",
      "privateIP": "10.0.0.10",
      "healthStatus": "healthy",
      "metrics": {
        "cpu": 45.2,
        "memory": 68.7,
        "disk": 23.1
      }
    }
  ],
  "cost": {
    "daily": 24.50,
    "monthly": 735.00,
    "currency": "USD"
  }
}
```

## Authentication and Authorization

### OAuth2 Authentication

#### Supported Providers
- Azure Active Directory
- Google Workspace
- Okta
- Keycloak
- Custom OIDC providers

#### OAuth2 Flow
```bash
# 1. Initiate login
curl -X GET "http://llm-processor:8080/auth/login/azure"
# Returns redirect URL

# 2. After OAuth2 flow completion, get token
curl -X GET "http://llm-processor:8080/auth/callback/azure?code=AUTH_CODE&state=STATE"
```

**Token Response**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "user_info": {
    "id": "user-123",
    "email": "operator@company.com",
    "name": "Network Operator",
    "roles": ["operator"]
  }
}
```

#### Using Access Token
```bash
curl -X POST http://llm-processor:8080/process \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"intent":"Deploy AMF with 3 replicas"}'
```

### API Key Authentication

For services that support API key authentication:

```bash
curl -X POST http://rag-api:5001/process_intent \
  -H "X-API-Key: your-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{"intent":"Scale E2 nodes to 5 replicas"}'
```

## Error Handling

### Standard Error Response Format

All services return errors in a consistent format:

```json
{
  "error": {
    "code": "INTENT_PROCESSING_FAILED",
    "type": "processing_error",
    "message": "Failed to process network intent due to invalid parameters",
    "details": {
      "validation_errors": [
        "replicas must be between 1 and 100",
        "namespace is required"
      ],
      "input": "Deploy AMF with 200 replicas",
      "suggestion": "Specify a valid replica count (1-100) and target namespace"
    },
    "timestamp": "2025-01-30T10:30:15Z",
    "request_id": "req-12345",
    "correlation_id": "corr-67890"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description | Retryable |
|------|-------------|-------------|-----------|
| `INTENT_PROCESSING_FAILED` | 400/500 | LLM processing failed | Yes |
| `INVALID_INTENT_FORMAT` | 400 | Invalid intent structure | No |
| `AUTHENTICATION_FAILED` | 401 | Authentication failed | No |
| `AUTHORIZATION_FAILED` | 403 | Insufficient permissions | No |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests | Yes |
| `RAG_SERVICE_UNAVAILABLE` | 503 | RAG API unavailable | Yes |
| `CIRCUIT_BREAKER_OPEN` | 503 | Circuit breaker active | Yes |
| `VECTOR_DB_ERROR` | 500 | Vector database error | Yes |
| `CRD_VALIDATION_FAILED` | 400 | Invalid CRD specification | No |
| `SCALING_FAILED` | 500 | E2NodeSet scaling failed | Yes |

### Rate Limiting

Services implement rate limiting with these headers:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45  
X-RateLimit-Reset: 1641024600
Retry-After: 30
```

Default limits:
- LLM Processor `/process`: 60 req/min
- LLM Processor `/stream`: 30 req/min  
- RAG API `/process_intent`: 100 req/min
- RAG API `/v1/query`: 200 req/min
- O-RAN Adaptor: 120 req/min

## SDK Examples

### Go Client Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/client"
)

func main() {
    // Initialize client with configuration
    config := client.Config{
        LLMProcessorURL: "http://llm-processor:8080",
        RAGAPIUrl:      "http://rag-api:5001",
        ORANAdaptorURL: "http://oran-adaptor:8082",
        APIKey:         "your-api-key",
        Timeout:        30 * time.Second,
    }
    
    client := client.NewNephoranClient(config)
    
    // Process intent
    ctx := context.Background()
    result, err := client.ProcessIntent(ctx, &client.IntentRequest{
        Intent: "Deploy AMF with 3 replicas for network slice eMBB",
        Metadata: map[string]string{
            "namespace": "telecom-core",
            "priority":  "high",
        },
        Config: &client.ProcessingConfig{
            Model:     "gpt-4o-mini",
            EnableRAG: true,
            MaxTokens: 2048,
        },
    })
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Result: %s\n", result.Result)
    fmt.Printf("Confidence: %.2f\n", result.Metadata.ConfidenceScore)
    fmt.Printf("Tokens used: %d\n", result.Metadata.TokensUsed)
}
```

### Python Client Example

```python
import requests
import json
from typing import Dict, Any, Optional

class NephoranClient:
    def __init__(self, base_url: str, api_key: Optional[str] = None):
        self.base_url = base_url
        self.session = requests.Session()
        if api_key:
            self.session.headers.update({'X-API-Key': api_key})
    
    def process_intent(self, intent: str, metadata: Optional[Dict[str, Any]] = None,
                      config: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Process a natural language intent."""
        payload = {
            'intent': intent,
            'metadata': metadata or {},
            'config': config or {}
        }
        
        response = self.session.post(
            f'{self.base_url}/process',
            json=payload,
            timeout=60
        )
        response.raise_for_status()
        return response.json()
    
    def query_knowledge_base(self, query: str, limit: int = 10,
                           filters: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Query the RAG knowledge base."""
        payload = {
            'query': query,
            'limit': limit,
            'filters': filters or {},
            'include_metadata': True
        }
        
        response = self.session.post(
            f'{self.base_url.replace(":8080", ":5001")}/v1/query',
            json=payload,
            timeout=30
        )
        response.raise_for_status()
        return response.json()

# Usage example
if __name__ == "__main__":
    client = NephoranClient('http://llm-processor:8080', 'your-api-key')
    
    # Process intent
    result = client.process_intent(
        'Deploy AMF with 3 replicas for network slice eMBB',
        metadata={'namespace': 'telecom-core', 'priority': 'high'},
        config={'model': 'gpt-4o-mini', 'enable_rag': True}
    )
    
    print(f"Result: {result['result']}")
    print(f"Processing time: {result['processing_time']}")
    
    # Query knowledge base
    kb_result = client.query_knowledge_base(
        'AMF registration procedures',
        filters={'category': '5G Core', 'source': '3GPP'}
    )
    
    print(f"Found {len(kb_result['results'])} knowledge base results")
```

## Monitoring and Observability

### Prometheus Metrics

All services expose metrics at `/metrics` endpoint in Prometheus format:

**LLM Processor Metrics**:
- `nephoran_llm_requests_total{status="success|error"}`: Total requests
- `nephoran_llm_request_duration_seconds`: Request duration histogram
- `nephoran_llm_tokens_used_total`: Total tokens consumed
- `nephoran_llm_cache_hits_total`: Cache hit counter
- `nephoran_streaming_active_sessions`: Active streaming sessions

**RAG API Metrics**:
- `nephoran_rag_queries_total{status="success|error"}`: Total queries
- `nephoran_rag_query_duration_seconds`: Query duration histogram  
- `nephoran_rag_documents_indexed_total`: Total documents indexed
- `nephoran_rag_vector_search_latency_seconds`: Vector search latency

**Controller Metrics**:
- `nephoran_networkintent_processing_duration_seconds`: Intent processing time
- `nephoran_e2nodeset_scaling_duration_seconds`: E2NodeSet scaling time
- `nephoran_controller_reconcile_errors_total`: Reconciliation errors

### Health Check Endpoints

Standard health endpoints across all services:
- `/healthz`: Liveness probe (basic health)
- `/readyz`: Readiness probe (ready for traffic)
- `/metrics`: Prometheus metrics

### Distributed Tracing

All services support OpenTelemetry tracing with automatic span creation for:
- Intent processing requests
- RAG API calls  
- Vector database queries
- Controller reconciliation loops
- O-RAN interface communications

Traces can be viewed in Jaeger UI at `http://jaeger:16686`

This API documentation provides comprehensive coverage of all Nephoran Intent Operator service endpoints, authentication methods, error handling, and usage examples. For additional details, refer to the service-specific documentation and troubleshooting guides.