# Weaviate API Reference for Nephoran Intent Operator
## Comprehensive API Documentation and Data Models

### Overview

This document provides complete API reference documentation for the Weaviate vector database deployment in the Nephoran Intent Operator system. It covers all endpoints, data models, query patterns, and integration specifications for telecommunications domain operations.

### Base Configuration

```yaml
# API Base Configuration
base_url: "http://weaviate.nephoran-system.svc.cluster.local:8080"
api_version: "v1"
authentication: "API Key (Bearer Token)"
content_type: "application/json"
```

### Authentication

All API requests require authentication using a Bearer token:

```bash
# Authentication Header
Authorization: Bearer <WEAVIATE_API_KEY>

# Example
curl -H "Authorization: Bearer nephoran-rag-key-production" \
  "http://weaviate:8080/v1/meta"
```

## Core API Endpoints

### 1. System Information and Health

#### GET /v1/meta
Returns system metadata and configuration information.

**Request:**
```http
GET /v1/meta HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:**
```json
{
  "hostname": "weaviate-0",
  "version": "1.28.0",
  "modules": {
    "text2vec-openai": {
      "version": "1.28.0",
      "type": "text2vec"
    },
    "generative-openai": {
      "version": "1.28.0", 
      "type": "text2vec"
    }
  },
  "stats": {
    "objectCount": 1250000,
    "classCount": 3,
    "shardCount": 9
  }
}
```

#### GET /v1/.well-known/ready
Health check endpoint for readiness validation.

**Request:**
```http
GET /v1/.well-known/ready HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:**
```json
{
  "status": "ready",
  "checks": {
    "database": "ready",
    "vectorizer": "ready",
    "generative": "ready"
  }
}
```

#### GET /v1/.well-known/live
Liveness check endpoint.

**Request:**
```http
GET /v1/.well-known/live HTTP/1.1
Host: weaviate:8080
```

**Response:**
```json
{
  "status": "live"
}
```

### 2. Schema Management

#### GET /v1/schema
Retrieve complete schema definition.

**Request:**
```http
GET /v1/schema HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:**
```json
{
  "classes": [
    {
      "class": "TelecomKnowledge",
      "description": "Telecommunications domain knowledge base",
      "vectorizer": "text2vec-openai",
      "moduleConfig": {
        "text2vec-openai": {
          "model": "text-embedding-3-large",
          "dimensions": 3072
        }
      },
      "properties": [
        {
          "name": "content",
          "dataType": ["text"],
          "description": "Document content",
          "moduleConfig": {
            "text2vec-openai": {
              "skip": false,
              "vectorizePropertyName": false
            }
          }
        }
      ],
      "vectorIndexConfig": {
        "distance": "cosine",
        "ef": 128,
        "efConstruction": 256
      },
      "shardingConfig": {
        "virtualPerPhysical": 128,
        "desiredCount": 3,
        "actualCount": 3
      }
    }
  ]
}
```

#### GET /v1/schema/{className}
Get schema for specific class.

**Request:**
```http
GET /v1/schema/TelecomKnowledge HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:** Single class schema object (same structure as above)

#### POST /v1/schema
Create new schema class.

**Request:**
```http
POST /v1/schema HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "class": "NetworkProcedures",
  "description": "Network operational procedures",
  "vectorizer": "text2vec-openai",
  "properties": [
    {
      "name": "procedure",
      "dataType": ["text"],
      "description": "Procedure description"
    },
    {
      "name": "category",
      "dataType": ["text"],
      "description": "Procedure category"
    }
  ]
}
```

**Response:**
```json
{
  "class": "NetworkProcedures",
  "properties": [...],
  "status": "created"
}
```

#### DELETE /v1/schema/{className}
Delete schema class and all associated objects.

**Request:**
```http
DELETE /v1/schema/NetworkProcedures HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:**
```json
{
  "status": "deleted",
  "objectsDeleted": 1523
}
```

### 3. Object Management

#### POST /v1/objects
Create new object.

**Request:**
```http
POST /v1/objects HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "class": "TelecomKnowledge",
  "properties": {
    "content": "The Access and Mobility Management Function (AMF) provides registration management, connection management, reachability management, mobility management, and access authentication and authorization.",
    "title": "AMF Functionality Overview",
    "source": "3GPP",
    "specification": "TS 23.501", 
    "version": "17.9.0",
    "release": "Rel-17",
    "category": "Architecture",
    "domain": "Core",
    "keywords": ["AMF", "mobility", "authentication", "registration"],
    "networkFunctions": ["AMF"],
    "interfaces": ["N1", "N2", "N8", "N11", "N12", "N14", "N15"],
    "procedures": ["Registration", "Authentication", "Mobility"],
    "useCase": "eMBB",
    "priority": 9,
    "confidence": 0.95,
    "technicalLevel": "Intermediate"
  }
}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "class": "TelecomKnowledge",
  "creationTimeUnix": 1704067200123,
  "lastUpdateTimeUnix": 1704067200123,
  "properties": {
    "content": "The Access and Mobility Management Function...",
    "title": "AMF Functionality Overview"
  },
  "vector": [0.1234, -0.5678, 0.9012, ...],
  "additional": {
    "certainty": 0.95,
    "distance": 0.05
  }
}
```

#### GET /v1/objects/{id}
Retrieve specific object by ID.

**Request:**
```http
GET /v1/objects/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:** Single object (same structure as POST response)

#### GET /v1/objects
List objects with filtering and pagination.

**Request:**
```http
GET /v1/objects?class=TelecomKnowledge&limit=10&offset=0 HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Query Parameters:**
- `class` (string): Filter by class name
- `limit` (integer): Maximum number of results (default: 25, max: 10000)
- `offset` (integer): Pagination offset (default: 0)
- `sort` (string): Sort order (asc/desc)
- `after` (string): Cursor for pagination
- `include` (string): Additional properties to include
- `where` (object): Filter conditions

**Response:**
```json
{
  "objects": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "class": "TelecomKnowledge",
      "properties": {...},
      "additional": {...}
    }
  ],
  "totalResults": 1250000,
  "limit": 10,
  "offset": 0
}
```

#### PATCH /v1/objects/{id}
Update existing object.

**Request:**
```http
PATCH /v1/objects/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "properties": {
    "confidence": 0.98,
    "lastUpdated": "2024-07-28T10:30:00Z"
  }
}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "updated",
  "properties": {
    "confidence": 0.98,
    "lastUpdated": "2024-07-28T10:30:00Z"
  }
}
```

#### DELETE /v1/objects/{id}
Delete specific object.

**Request:**
```http
DELETE /v1/objects/550e8400-e29b-41d4-a716-446655440000 HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "deleted"
}
```

### 4. Batch Operations

#### POST /v1/batch/objects
Batch create/update objects.

**Request:**
```http
POST /v1/batch/objects HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "objects": [
    {
      "class": "TelecomKnowledge",
      "properties": {
        "content": "SMF manages PDU sessions...",
        "title": "SMF Session Management"
      }
    },
    {
      "class": "TelecomKnowledge", 
      "properties": {
        "content": "UPF processes user plane traffic...",
        "title": "UPF User Plane Processing"
      }
    }
  ]
}
```

**Response:**
```json
{
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "status": "SUCCESS",
      "creationTimeUnix": 1704067200456
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "status": "SUCCESS", 
      "creationTimeUnix": 1704067200457
    }
  ],
  "summary": {
    "successful": 2,
    "failed": 0,
    "total": 2
  }
}
```

#### DELETE /v1/batch/objects
Batch delete objects.

**Request:**
```http
DELETE /v1/batch/objects HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "match": {
    "class": "TelecomKnowledge",
    "where": {
      "path": ["confidence"],
      "operator": "LessThan",
      "valueNumber": 0.5
    }
  }
}
```

**Response:**
```json
{
  "results": {
    "matches": 245,
    "successful": 245,
    "failed": 0,
    "objects": null
  }
}
```

### 5. GraphQL Query Interface

#### POST /v1/graphql
Execute GraphQL queries for complex data retrieval.

**Basic Query Request:**
```http
POST /v1/graphql HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "query": "{
    Get {
      TelecomKnowledge(limit: 5) {
        content
        title
        source
        specification
        networkFunctions
        confidence
        _additional {
          id
          certainty
          distance
        }
      }
    }
  }"
}
```

**Vector Similarity Query:**
```json
{
  "query": "{
    Get {
      TelecomKnowledge(
        nearText: {
          concepts: [\"AMF registration procedure\"]
          certainty: 0.7
        }
        limit: 10
      ) {
        content
        title
        source
        specification
        _additional {
          certainty
          distance
        }
      }
    }
  }"
}
```

**Hybrid Search Query:**
```json
{
  "query": "{
    Get {
      TelecomKnowledge(
        hybrid: {
          query: \"5G network slicing configuration\"
          alpha: 0.7
        }
        limit: 10
      ) {
        content
        title
        source
        useCase
        _additional {
          score
        }
      }
    }
  }"
}
```

**Filtered Query:**
```json
{
  "query": "{
    Get {
      TelecomKnowledge(
        where: {
          operator: And
          operands: [
            {
              path: [\"source\"]
              operator: Equal
              valueText: \"3GPP\"
            },
            {
              path: [\"release\"]
              operator: Equal
              valueText: \"Rel-17\"
            }
          ]
        }
        limit: 20
      ) {
        content
        title
        specification
        version
        networkFunctions
      }
    }
  }"
}
```

**Cross-Reference Query:**
```json
{
  "query": "{
    Get {
      TelecomKnowledge(
        nearText: {
          concepts: [\"network function deployment\"]
        }
        limit: 5
      ) {
        content
        title
        networkFunctions
        relatedIntents {
          ... on IntentPatterns {
            pattern
            intentType
            confidence
          }
        }
        applicableNFs {
          ... on NetworkFunctions {
            name
            description
            category
          }
        }
      }
    }
  }"
}
```

**Aggregation Query:**
```json
{
  "query": "{
    Aggregate {
      TelecomKnowledge {
        meta {
          count
        }
        source {
          count
          topOccurrences(limit: 10) {
            value
            occurs
          }
        }
        confidence {
          mean
          maximum
          minimum
          sum
        }
      }
    }
  }"
}
```

### 6. Classification and Clustering

#### POST /v1/classifications
Trigger automated classification.

**Request:**
```http
POST /v1/classifications HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "class": "TelecomKnowledge",
  "classifyProperties": ["category"],
  "basedOnProperties": ["content", "title"],
  "trainingSetWhere": {
    "path": ["category"],
    "operator": "NotEqual",
    "valueText": "unknown"
  }
}
```

**Response:**
```json
{
  "id": "classification-task-123",
  "status": "running",
  "meta": {
    "completed": 0,
    "count": 1250000,
    "started": "2024-07-28T10:30:00Z"
  }
}
```

#### GET /v1/classifications/{id}
Check classification status.

**Request:**
```http
GET /v1/classifications/classification-task-123 HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "classification-task-123",
  "status": "completed",
  "meta": {
    "completed": 1250000,
    "count": 1250000,
    "started": "2024-07-28T10:30:00Z",
    "completed_at": "2024-07-28T10:45:00Z"
  },
  "results": {
    "successful": 1248750,
    "failed": 1250
  }
}
```

### 7. Backup and Restore Operations

#### POST /v1/backups/{backend}
Create backup.

**Request:**
```http
POST /v1/backups/filesystem HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "id": "telecom-backup-20240728-103000",
  "include": ["TelecomKnowledge", "IntentPatterns", "NetworkFunctions"],
  "exclude": [],
  "compression": "gzip"
}
```

**Response:**
```json
{
  "id": "telecom-backup-20240728-103000",
  "status": "STARTED",
  "path": "/var/lib/weaviate/backups/telecom-backup-20240728-103000"
}
```

#### GET /v1/backups/{backend}/{backup-id}
Check backup status.

**Request:**
```http
GET /v1/backups/filesystem/telecom-backup-20240728-103000 HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "telecom-backup-20240728-103000",
  "status": "SUCCESS",
  "path": "/var/lib/weaviate/backups/telecom-backup-20240728-103000",
  "meta": {
    "startedAt": "2024-07-28T10:30:00Z",
    "completedAt": "2024-07-28T10:35:00Z",
    "objectsCount": 1250000,
    "size": "2.5GB"
  }
}
```

#### POST /v1/backups/{backend}/{backup-id}/restore
Restore from backup.

**Request:**
```http
POST /v1/backups/filesystem/telecom-backup-20240728-103000/restore HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
Content-Type: application/json

{
  "include": ["TelecomKnowledge"],
  "exclude": [],
  "strategy": "merge"
}
```

**Response:**
```json
{
  "id": "restore-task-456",
  "status": "STARTED",
  "backup": "telecom-backup-20240728-103000"
}
```

### 8. Modules and Vectorizers

#### GET /v1/modules
List available modules.

**Request:**
```http
GET /v1/modules HTTP/1.1
Host: weaviate:8080
Authorization: Bearer <token>
```

**Response:**
```json
{
  "modules": [
    {
      "name": "text2vec-openai",
      "type": "text2vec",
      "version": "1.28.0",
      "config": {
        "model": "text-embedding-3-large",
        "dimensions": 3072
      }
    },
    {
      "name": "generative-openai",
      "type": "text2vec",
      "version": "1.28.0",
      "config": {
        "model": "gpt-4o-mini"
      }
    }
  ]
}
```

## Data Models

### TelecomKnowledge Data Model

```typescript
interface TelecomKnowledge {
  // Core Content
  content: string;              // Main document content (vectorized)
  title: string;                // Document title (vectorized)
  
  // Source Information
  source: "3GPP" | "O-RAN" | "ETSI" | "ITU-T" | "Custom";
  specification: string;        // e.g., "TS 23.501"
  version: string;              // Specification version
  release: string;              // 3GPP Release or O-RAN version
  
  // Classification
  category: "Architecture" | "Procedures" | "Interfaces" | "Management" | "Security";
  domain: "Core" | "RAN" | "Transport" | "Edge" | "Management";
  technicalLevel: "Basic" | "Intermediate" | "Advanced" | "Expert";
  
  // Telecom Specific
  keywords: string[];           // Technical keywords
  networkFunctions: string[];   // Referenced NFs
  interfaces: string[];         // Referenced interfaces
  procedures: string[];         // Referenced procedures
  useCase: "eMBB" | "URLLC" | "mMTC" | "Private" | "General";
  
  // Quality Metrics
  priority: number;             // 1-10 priority
  confidence: number;           // 0.0-1.0 confidence
  
  // Metadata
  documentUrl?: string;         // Source URL
  chunkIndex?: number;          // Chunk sequence
  totalChunks?: number;         // Total chunks in document
  language: string;             // ISO 639-1 code
  lastUpdated: string;          // ISO 8601 timestamp
  
  // Relationships
  relatedIntents?: IntentPattern[];
  applicableNFs?: NetworkFunction[];
}
```

### IntentPattern Data Model

```typescript
interface IntentPattern {
  // Pattern Definition
  pattern: string;              // Template pattern (vectorized)
  intentType: "NetworkFunctionDeployment" | "NetworkFunctionScale" | 
             "NetworkSliceConfiguration" | "PolicyConfiguration";
  
  // Parameters
  parameters: string[];         // Expected parameters
  examples: string[];           // Example phrases
  
  // Quality Metrics
  confidence: number;           // Pattern confidence threshold
  accuracy?: number;            // Historical accuracy
  frequency?: number;           // Usage frequency
  
  // Learning Data
  successRate?: number;         // Success rate
  lastUsed?: string;           // Last usage timestamp
  userFeedback?: number;       // User feedback score
  
  // Relationships
  relatedKnowledge?: TelecomKnowledge[];
}
```

### NetworkFunction Data Model

```typescript
interface NetworkFunction {
  // Basic Information
  name: string;                 // NF name (AMF, SMF, etc.)
  description: string;          // Detailed description (vectorized)
  category: "Core" | "RAN" | "Management" | "Edge";
  version?: string;             // NF version
  vendor?: string;              // Vendor name
  
  // Deployment Configuration
  deploymentOptions: string[];  // Deployment patterns
  resourceRequirements: string; // Resource specifications
  scalingOptions: string;       // Scaling capabilities
  configurationTemplates?: string[]; // Config templates
  healthChecks?: string[];      // Health check endpoints
  
  // Standards Compliance
  standardsCompliance: string[]; // Applicable standards
  interfaces: string[];         // Supported interfaces
  supportedReleases: string[];  // Compatible releases
  certificationLevel?: string;  // Certification status
  
  // Operational Information
  operationalStatus?: "Active" | "Deprecated" | "Development";
  
  // Relationships
  relatedKnowledge?: TelecomKnowledge[];
  compatibleIntents?: IntentPattern[];
}
```

## Query Patterns and Best Practices

### Semantic Search Patterns

```graphql
# Intent-based semantic search
{
  Get {
    TelecomKnowledge(
      nearText: {
        concepts: ["AMF authentication procedure"]
        certainty: 0.75
      }
      where: {
        path: ["source"]
        operator: Equal
        valueText: "3GPP"
      }
      limit: 5
    ) {
      content
      title
      specification
      procedures
      _additional {
        certainty
      }
    }
  }
}
```

### Hybrid Search Optimization

```graphql
# Balanced hybrid search for technical queries
{
  Get {
    TelecomKnowledge(
      hybrid: {
        query: "5G network slice configuration AMF SMF"
        alpha: 0.7  # Favor vector search
        properties: ["content", "title", "keywords"]
      }
      where: {
        path: ["useCase"]
        operator: Equal
        valueText: "eMBB"
      }
      limit: 10
    ) {
      content
      title
      networkFunctions
      useCase
      _additional {
        score
        explainScore
      }
    }
  }
}
```

### Complex Filtering

```graphql
# Multi-condition filtering with aggregation
{
  Get {
    TelecomKnowledge(
      where: {
        operator: And
        operands: [
          {
            path: ["source"]
            operator: Equal
            valueText: "3GPP"
          },
          {
            path: ["release"]
            operator: Like
            valueText: "Rel-1*"
          },
          {
            path: ["confidence"]
            operator: GreaterThan
            valueNumber: 0.8
          },
          {
            path: ["networkFunctions"]
            operator: ContainsAny
            valueTextArray: ["AMF", "SMF", "UPF"]
          }
        ]
      }
      limit: 50
    ) {
      content
      title
      specification
      release
      networkFunctions
      confidence
    }
  }
}
```

## Error Handling

### Common Error Responses

#### 400 Bad Request
```json
{
  "error": [
    {
      "message": "invalid GraphQL field: unknown field 'invalidField' on type 'TelecomKnowledge'"
    }
  ]
}
```

#### 401 Unauthorized
```json
{
  "error": [
    {
      "message": "invalid api key"
    }
  ]
}
```

#### 404 Not Found
```json
{
  "error": [
    {
      "message": "class 'NonExistentClass' not found"
    }
  ]
}
```

#### 422 Unprocessable Entity
```json
{
  "error": [
    {
      "message": "invalid property 'invalidProperty' on class 'TelecomKnowledge': no such property exists"
    }
  ]
}
```

#### 500 Internal Server Error
```json
{
  "error": [
    {
      "message": "vectorizer error: OpenAI API rate limit exceeded"
    }
  ]
}
```

### Error Handling Best Practices

```javascript
// Example error handling in client code
async function queryWeaviate(query) {
  try {
    const response = await fetch('http://weaviate:8080/v1/graphql', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${WEAVIATE_API_KEY}`
      },
      body: JSON.stringify({ query })
    });
    
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(`Weaviate API error: ${errorData.error[0].message}`);
    }
    
    const data = await response.json();
    
    if (data.errors) {
      throw new Error(`GraphQL error: ${data.errors[0].message}`);
    }
    
    return data.data;
    
  } catch (error) {
    console.error('Weaviate query failed:', error.message);
    
    // Implement retry logic for transient errors
    if (error.message.includes('rate limit') || 
        error.message.includes('timeout')) {
      await new Promise(resolve => setTimeout(resolve, 5000));
      return queryWeaviate(query); // Retry once
    }
    
    throw error;
  }
}
```

## Rate Limiting and Performance

### Request Limits

| Operation Type | Default Limit | Burst Limit | Window |
|---------------|---------------|-------------|--------|
| Query Requests | 100/minute | 200/minute | 60 seconds |
| Object Creation | 1000/minute | 2000/minute | 60 seconds |
| Batch Operations | 10/minute | 20/minute | 60 seconds |
| Schema Operations | 10/minute | 15/minute | 60 seconds |

### Performance Optimization Headers

```http
# Request headers for optimization
X-Request-ID: unique-request-id-123
X-Client-Version: nephoran-operator-v1.0
X-Priority: high|medium|low
Cache-Control: max-age=300  # For cacheable queries
```

### Response Headers

```http
# Response headers with performance info
X-Request-ID: unique-request-id-123
X-Response-Time: 245ms
X-Query-Complexity: 3.2
X-Rate-Limit-Remaining: 95
X-Rate-Limit-Reset: 1704067500
```

## Integration Examples

### Python Client Integration

```python
import weaviate
import json
from typing import List, Dict, Any

class NephoranWeaviateClient:
    def __init__(self, url: str, api_key: str, openai_api_key: str):
        self.client = weaviate.Client(
            url=url,
            auth_client_secret=weaviate.AuthApiKey(api_key=api_key),
            additional_headers={
                "X-OpenAI-Api-Key": openai_api_key
            }
        )
    
    def search_telecom_knowledge(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Search telecom knowledge using hybrid search"""
        result = self.client.query.get("TelecomKnowledge", [
            "content", "title", "source", "specification", 
            "networkFunctions", "confidence"
        ]).with_hybrid(
            query=query,
            alpha=0.7
        ).with_limit(limit).with_additional(["score"]).do()
        
        return result["data"]["Get"]["TelecomKnowledge"]
    
    def get_network_functions_by_category(self, category: str) -> List[Dict[str, Any]]:
        """Get network functions by category"""
        result = self.client.query.get("NetworkFunctions", [
            "name", "description", "category", "interfaces", "deploymentOptions"
        ]).with_where({
            "path": ["category"],
            "operator": "Equal",
            "valueText": category
        }).do()
        
        return result["data"]["Get"]["NetworkFunctions"]
    
    def find_related_intents(self, nf_name: str) -> List[Dict[str, Any]]:
        """Find intent patterns related to specific network function"""
        result = self.client.query.get("IntentPatterns", [
            "pattern", "intentType", "parameters", "confidence"
        ]).with_where({
            "path": ["parameters"],
            "operator": "ContainsAny", 
            "valueTextArray": [nf_name.lower()]
        }).do()
        
        return result["data"]["Get"]["IntentPatterns"]
```

### Go Client Integration

```go
package main

import (
    "context"
    "fmt"
    "github.com/weaviate/weaviate-go-client/v4/weaviate"
    "github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
    "github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
)

type NephoranWeaviateClient struct {
    client *weaviate.Client
}

func NewNephoranClient(url, apiKey, openaiKey string) *NephoranWeaviateClient {
    cfg := weaviate.Config{
        Host:   "weaviate:8080",
        Scheme: "http",
        AuthConfig: auth.ApiKey{Value: apiKey},
        Headers: map[string]string{
            "X-OpenAI-Api-Key": openaiKey,
        },
    }
    
    client, err := weaviate.NewClient(cfg)
    if err != nil {
        panic(err)
    }
    
    return &NephoranWeaviateClient{client: client}
}

func (c *NephoranWeaviateClient) SearchTelecomKnowledge(ctx context.Context, query string) (*graphql.Response, error) {
    result, err := c.client.GraphQL().Get().
        WithClassName("TelecomKnowledge").
        WithFields(
            graphql.Field{Name: "content"},
            graphql.Field{Name: "title"}, 
            graphql.Field{Name: "source"},
            graphql.Field{Name: "specification"},
            graphql.Field{Name: "networkFunctions"},
            graphql.Field{Name: "_additional", Fields: []graphql.Field{
                {Name: "score"},
            }},
        ).
        WithHybrid(&graphql.HybridArgument{
            Query: query,
            Alpha: 0.7,
        }).
        WithLimit(10).
        Do(ctx)
    
    return result, err
}
```

### REST API Integration (cURL Examples)

```bash
#!/bin/bash
# Comprehensive cURL examples for Weaviate API

WEAVIATE_URL="http://weaviate:8080"
API_KEY="your-weaviate-api-key"

# 1. Health check
curl -X GET "$WEAVIATE_URL/v1/.well-known/ready" \
  -H "Authorization: Bearer $API_KEY"

# 2. Get schema
curl -X GET "$WEAVIATE_URL/v1/schema" \
  -H "Authorization: Bearer $API_KEY" | jq '.'

# 3. Search with GraphQL
curl -X POST "$WEAVIATE_URL/v1/graphql" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{
      Get {
        TelecomKnowledge(
          hybrid: {
            query: \"AMF registration procedure\"
            alpha: 0.7
          }
          limit: 5
        ) {
          content
          title
          specification
          _additional {
            score
          }
        }
      }
    }"
  }' | jq '.data.Get.TelecomKnowledge'

# 4. Create object
curl -X POST "$WEAVIATE_URL/v1/objects" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "class": "TelecomKnowledge",
    "properties": {
      "content": "Network slicing enables multiple virtual networks...",
      "title": "5G Network Slicing Overview",
      "source": "3GPP",
      "specification": "TS 23.501",
      "category": "Architecture",
      "networkFunctions": ["AMF", "NSSF"],
      "confidence": 0.95
    }
  }'

# 5. Batch operations
curl -X POST "$WEAVIATE_URL/v1/batch/objects" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "objects": [
      {
        "class": "TelecomKnowledge",
        "properties": {
          "content": "QoS flow management in 5G...",
          "title": "QoS Flow Management"
        }
      }
    ]
  }'

# 6. Create backup
curl -X POST "$WEAVIATE_URL/v1/backups/filesystem" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "telecom-backup-'$(date +%Y%m%d)'",
    "include": ["TelecomKnowledge", "IntentPatterns", "NetworkFunctions"]
  }'
```

This comprehensive API reference provides complete documentation for integrating with the Weaviate vector database in the Nephoran Intent Operator system. Use this reference for building robust telecommunications applications with semantic search capabilities.