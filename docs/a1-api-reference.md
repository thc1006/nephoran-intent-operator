# A1 Policy Management Service - API Reference

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [A1-P Policy Interface (v2)](#a1-p-policy-interface-v2)
4. [A1-C Consumer Interface (v1)](#a1-c-consumer-interface-v1)
5. [A1-EI Enrichment Information Interface (v1)](#a1-ei-enrichment-information-interface-v1)
6. [Health and Monitoring Endpoints](#health-and-monitoring-endpoints)
7. [Error Handling](#error-handling)
8. [Rate Limiting](#rate-limiting)
9. [OpenAPI Specification](#openapi-specification)

## Overview

The Nephoran A1 Policy Management Service provides RESTful APIs compliant with O-RAN Alliance specifications. The service implements three main interfaces:

- **A1-P (Policy Interface v2)**: Policy type and instance management
- **A1-C (Consumer Interface v1)**: Consumer registration and notification management
- **A1-EI (Enrichment Information Interface v1)**: Enrichment information type and job management

### Base URLs

- **Production**: `https://a1-api.nephoran.io`
- **Development**: `http://localhost:8080`

### Content Types

All APIs accept and return `application/json` content type unless otherwise specified.

```http
Content-Type: application/json
Accept: application/json
```

## Authentication

### OAuth2 Authentication

The A1 Policy Service supports OAuth2 with multiple providers:

```http
Authorization: Bearer <access_token>
```

#### Supported OAuth2 Providers

| Provider | Issuer URL | Audience |
|----------|------------|----------|
| Azure AD | `https://login.microsoftonline.com/TENANT_ID/v2.0` | `api://a1-policy-service` |
| Google | `https://accounts.google.com` | `a1-policy-service` |
| AWS Cognito | `https://cognito-idp.REGION.amazonaws.com/USER_POOL_ID` | Client ID |

#### Required Scopes

- `a1:policy:read` - Read policy types and instances
- `a1:policy:write` - Create and update policies
- `a1:consumer:read` - Read consumer information
- `a1:consumer:write` - Register and manage consumers
- `a1:ei:read` - Read enrichment information
- `a1:ei:write` - Create and manage EI jobs

### API Key Authentication (Alternative)

For service-to-service communication:

```http
X-API-Key: <api_key>
```

## A1-P Policy Interface (v2)

The A1-P interface provides comprehensive policy management capabilities between Non-RT RIC and Near-RT RIC components.

### Policy Types

#### Get All Policy Types

```http
GET /A1-P/v2/policytypes
```

**Description**: Retrieve all available policy types.

**Response**:
```json
[20008, 20009, 20010, 20011]
```

**Example**:
```bash
curl -X GET "https://a1-api.nephoran.io/A1-P/v2/policytypes" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json"
```

#### Get Policy Type Details

```http
GET /A1-P/v2/policytypes/{policy_type_id}
```

**Parameters**:
- `policy_type_id` (integer, path) - Policy type identifier

**Response**:
```json
{
  "policy_type_id": 20008,
  "policy_type_name": "Traffic Steering Policy",
  "description": "Policy for steering traffic based on load balancing requirements",
  "created_at": "2024-01-15T10:30:00Z",
  "modified_at": "2024-01-15T10:30:00Z",
  "schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "scope": {
        "type": "object",
        "properties": {
          "ue_id": {
            "type": "string",
            "description": "UE identifier or * for all UEs"
          },
          "cell_list": {
            "type": "array",
            "items": {"type": "string"},
            "description": "List of cell identifiers"
          }
        },
        "required": ["ue_id"]
      },
      "qos_preference": {
        "type": "object",
        "properties": {
          "priority_level": {
            "type": "integer",
            "minimum": 1,
            "maximum": 15
          },
          "load_balancing": {
            "type": "object",
            "properties": {
              "site_a_weight": {
                "type": "integer",
                "minimum": 0,
                "maximum": 100
              },
              "site_b_weight": {
                "type": "integer",
                "minimum": 0,
                "maximum": 100
              }
            },
            "required": ["site_a_weight", "site_b_weight"]
          }
        },
        "required": ["priority_level"]
      }
    },
    "required": ["scope", "qos_preference"]
  }
}
```

#### Create Policy Type

```http
PUT /A1-P/v2/policytypes/{policy_type_id}
```

**Parameters**:
- `policy_type_id` (integer, path) - Policy type identifier (1-2147483647)

**Request Body**:
```json
{
  "policy_type_name": "Custom QoS Policy",
  "description": "Custom policy for QoS management",
  "schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "qos_requirements": {
        "type": "object",
        "properties": {
          "bandwidth": {"type": "integer"},
          "latency": {"type": "integer"}
        },
        "required": ["bandwidth", "latency"]
      }
    },
    "required": ["qos_requirements"]
  }
}
```

**Response**: `201 Created`

#### Delete Policy Type

```http
DELETE /A1-P/v2/policytypes/{policy_type_id}
```

**Parameters**:
- `policy_type_id` (integer, path) - Policy type identifier

**Response**: `204 No Content` (if no policy instances exist)
**Error**: `409 Conflict` (if policy instances exist)

### Policy Instances

#### Get Policy Instances for a Type

```http
GET /A1-P/v2/policytypes/{policy_type_id}/policies
```

**Parameters**:
- `policy_type_id` (integer, path) - Policy type identifier

**Response**:
```json
["traffic-steering-001", "traffic-steering-002", "load-balancing-001"]
```

#### Get Policy Instance Details

```http
GET /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}
```

**Parameters**:
- `policy_type_id` (integer, path) - Policy type identifier
- `policy_id` (string, path) - Policy instance identifier

**Response**:
```json
{
  "policy_id": "traffic-steering-001",
  "policy_type_id": 20008,
  "policy_data": {
    "scope": {
      "ue_id": "*",
      "cell_list": ["cell_001", "cell_002"]
    },
    "qos_preference": {
      "priority_level": 1,
      "load_balancing": {
        "site_a_weight": 70,
        "site_b_weight": 30
      }
    }
  },
  "policy_info": {
    "notification_destination": "http://consumer-callback.example.com/notifications",
    "request_id": "req_12345"
  },
  "created_at": "2024-01-15T10:30:00Z",
  "modified_at": "2024-01-15T11:00:00Z"
}
```

#### Create Policy Instance

```http
PUT /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}
```

**Parameters**:
- `policy_type_id` (integer, path) - Policy type identifier
- `policy_id` (string, path) - Policy instance identifier (max 256 characters)

**Request Body**:
```json
{
  "policy_data": {
    "scope": {
      "ue_id": "*",
      "cell_list": ["cell_001", "cell_002"]
    },
    "qos_preference": {
      "priority_level": 1,
      "load_balancing": {
        "site_a_weight": 80,
        "site_b_weight": 20
      }
    }
  },
  "policy_info": {
    "notification_destination": "http://consumer-callback.example.com/notifications",
    "request_id": "req_67890",
    "additional_params": {
      "owner": "network-operator-1",
      "environment": "production"
    }
  }
}
```

**Response**: `201 Created`

#### Update Policy Instance

```http
PUT /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}
```

Same as create - the API is idempotent.

**Response**: `200 OK` (if updated) or `201 Created` (if created)

#### Delete Policy Instance

```http
DELETE /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}
```

**Parameters**:
- `policy_type_id` (integer, path) - Policy type identifier
- `policy_id` (string, path) - Policy instance identifier

**Response**: `202 Accepted` (deletion initiated)

### Policy Status

#### Get Policy Status

```http
GET /A1-P/v2/policytypes/{policy_type_id}/policies/{policy_id}/status
```

**Parameters**:
- `policy_type_id` (integer, path) - Policy type identifier
- `policy_id` (string, path) - Policy instance identifier

**Response**:
```json
{
  "enforcement_status": "ENFORCED",
  "enforcement_reason": "Policy successfully applied to all target cells",
  "has_been_deleted": false,
  "deleted": false,
  "created_at": "2024-01-15T10:30:00Z",
  "modified_at": "2024-01-15T10:35:00Z",
  "additional_info": {
    "applied_cells": ["cell_001", "cell_002"],
    "xapp_instances": ["traffic-steering-xapp-1", "traffic-steering-xapp-2"],
    "last_update_source": "near-rt-ric-001"
  }
}
```

**Enforcement Status Values**:
- `NOT_ENFORCED` - Policy is not currently enforced
- `ENFORCED` - Policy is successfully enforced
- `UNKNOWN` - Enforcement status is unknown

## A1-C Consumer Interface (v1)

The A1-C interface manages consumer registration and notification delivery.

### Consumer Management

#### List All Consumers

```http
GET /A1-C/v1/consumers
```

**Response**:
```json
[
  {
    "consumer_id": "traffic-steering-xapp",
    "consumer_name": "Traffic Steering xApp",
    "callback_url": "http://traffic-steering-xapp.oran.svc.cluster.local:8080/a1-callbacks",
    "capabilities": ["policy_notifications", "status_updates"],
    "metadata": {
      "version": "1.2.0",
      "description": "Intelligent traffic steering application",
      "supported_types": [20008, 20009]
    },
    "created_at": "2024-01-15T09:00:00Z",
    "modified_at": "2024-01-15T09:00:00Z"
  }
]
```

#### Get Consumer Details

```http
GET /A1-C/v1/consumers/{consumer_id}
```

**Parameters**:
- `consumer_id` (string, path) - Consumer identifier

**Response**: Same as individual consumer object above.

#### Register Consumer

```http
POST /A1-C/v1/consumers/{consumer_id}
```

**Parameters**:
- `consumer_id` (string, path) - Consumer identifier (max 256 characters)

**Request Body**:
```json
{
  "consumer_name": "QoS Management xApp",
  "callback_url": "http://qos-mgmt-xapp.oran.svc.cluster.local:8080/a1-callbacks",
  "capabilities": ["policy_notifications", "status_updates", "heartbeat"],
  "metadata": {
    "version": "2.1.0",
    "description": "Advanced QoS management application",
    "supported_types": [20009, 20010],
    "additional_info": {
      "deployment_id": "qos-mgmt-prod-001",
      "cluster_id": "oran-cluster-east"
    }
  }
}
```

**Response**: `201 Created`

#### Unregister Consumer

```http
DELETE /A1-C/v1/consumers/{consumer_id}
```

**Parameters**:
- `consumer_id` (string, path) - Consumer identifier

**Response**: `204 No Content`

### Consumer Notifications

Consumer notifications are sent asynchronously via HTTP POST to the registered callback URL.

#### Policy Notification Format

```json
{
  "notification_type": "CREATE",
  "policy_id": "traffic-steering-001",
  "policy_type_id": 20008,
  "timestamp": "2024-01-15T10:30:00Z",
  "status": "CREATED",
  "message": "Policy instance created successfully",
  "additional_details": {
    "created_by": "network-operator-1",
    "deployment_target": "cell_001,cell_002"
  }
}
```

**Notification Types**:
- `CREATE` - Policy instance created
- `UPDATE` - Policy instance updated  
- `DELETE` - Policy instance deleted

## A1-EI Enrichment Information Interface (v1)

The A1-EI interface manages enrichment information types and jobs.

### Enrichment Information Types

#### Get All EI Types

```http
GET /A1-EI/v1/eitypes
```

**Response**:
```json
["traffic-stats", "performance-metrics", "network-topology"]
```

#### Get EI Type Details

```http
GET /A1-EI/v1/eitypes/{ei_type_id}
```

**Parameters**:
- `ei_type_id` (string, path) - EI type identifier

**Response**:
```json
{
  "ei_type_id": "traffic-stats",
  "ei_type_name": "Traffic Statistics Collection",
  "description": "Collects and provides traffic statistics for network optimization",
  "ei_job_data_schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "collection_scope": {
        "type": "object",
        "properties": {
          "cell_list": {
            "type": "array",
            "items": {"type": "string"}
          },
          "time_window": {
            "type": "integer",
            "minimum": 60,
            "description": "Collection window in seconds"
          }
        },
        "required": ["cell_list", "time_window"]
      },
      "metrics_to_collect": {
        "type": "array",
        "items": {
          "type": "string",
          "enum": ["throughput", "latency", "packet_loss", "ue_count"]
        }
      }
    },
    "required": ["collection_scope", "metrics_to_collect"]
  },
  "ei_job_result_schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "timestamp": {"type": "string", "format": "date-time"},
      "cell_id": {"type": "string"},
      "metrics": {
        "type": "object",
        "properties": {
          "throughput_mbps": {"type": "number"},
          "latency_ms": {"type": "number"},
          "packet_loss_percent": {"type": "number"},
          "active_ue_count": {"type": "integer"}
        }
      }
    }
  },
  "created_at": "2024-01-15T08:00:00Z",
  "modified_at": "2024-01-15T08:00:00Z"
}
```

#### Create EI Type

```http
PUT /A1-EI/v1/eitypes/{ei_type_id}
```

**Parameters**:
- `ei_type_id` (string, path) - EI type identifier (max 256 characters)

**Request Body**:
```json
{
  "ei_type_name": "Custom Network Metrics",
  "description": "Custom metrics collection for specific network analysis",
  "ei_job_data_schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "target_nodes": {
        "type": "array",
        "items": {"type": "string"}
      }
    },
    "required": ["target_nodes"]
  },
  "ei_job_result_schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "node_id": {"type": "string"},
      "custom_metric": {"type": "number"}
    }
  }
}
```

**Response**: `201 Created`

#### Delete EI Type

```http
DELETE /A1-EI/v1/eitypes/{ei_type_id}
```

**Parameters**:
- `ei_type_id` (string, path) - EI type identifier

**Response**: `204 No Content` (if no EI jobs exist)
**Error**: `409 Conflict` (if EI jobs exist)

### Enrichment Information Jobs

#### Get All EI Jobs

```http
GET /A1-EI/v1/eijobs
```

**Query Parameters**:
- `ei_type_id` (string, optional) - Filter by EI type
- `job_owner` (string, optional) - Filter by job owner

**Response**:
```json
["traffic-stats-job-001", "performance-metrics-job-001"]
```

#### Get EI Job Details

```http
GET /A1-EI/v1/eijobs/{ei_job_id}
```

**Parameters**:
- `ei_job_id` (string, path) - EI job identifier

**Response**:
```json
{
  "ei_job_id": "traffic-stats-job-001",
  "ei_type_id": "traffic-stats",
  "ei_job_data": {
    "collection_scope": {
      "cell_list": ["cell_001", "cell_002"],
      "time_window": 300
    },
    "metrics_to_collect": ["throughput", "latency", "ue_count"]
  },
  "target_uri": "http://traffic-analyzer.analytics.svc.cluster.local:8080/data",
  "job_owner": "traffic-optimization-service",
  "job_status_url": "http://job-status-service.oran.svc.cluster.local:8080/status",
  "job_definition": {
    "delivery_info": [
      {
        "topic_name": "traffic-stats",
        "boot_strap_server": "kafka.messaging.svc.cluster.local:9092"
      }
    ],
    "job_parameters": {
      "aggregation_level": "cell",
      "reporting_frequency": "1min"
    },
    "status_notification_uri": "http://job-status-callback.oran.svc.cluster.local:8080/status"
  },
  "created_at": "2024-01-15T10:00:00Z",
  "modified_at": "2024-01-15T10:00:00Z",
  "last_executed_at": "2024-01-15T10:05:00Z"
}
```

#### Create EI Job

```http
PUT /A1-EI/v1/eijobs/{ei_job_id}
```

**Parameters**:
- `ei_job_id` (string, path) - EI job identifier (max 256 characters)

**Request Body**:
```json
{
  "ei_type_id": "traffic-stats",
  "ei_job_data": {
    "collection_scope": {
      "cell_list": ["cell_003", "cell_004"],
      "time_window": 600
    },
    "metrics_to_collect": ["throughput", "latency"]
  },
  "target_uri": "http://new-analyzer.analytics.svc.cluster.local:8080/data",
  "job_owner": "capacity-planning-service",
  "job_definition": {
    "delivery_info": [
      {
        "topic_name": "capacity-stats",
        "boot_strap_server": "kafka.messaging.svc.cluster.local:9092"
      }
    ],
    "job_parameters": {
      "aggregation_level": "site",
      "reporting_frequency": "5min"
    }
  }
}
```

**Response**: `201 Created`

#### Update EI Job

```http
PUT /A1-EI/v1/eijobs/{ei_job_id}
```

Same as create - the API is idempotent.

**Response**: `200 OK` (if updated) or `201 Created` (if created)

#### Delete EI Job

```http
DELETE /A1-EI/v1/eijobs/{ei_job_id}
```

**Parameters**:
- `ei_job_id` (string, path) - EI job identifier

**Response**: `202 Accepted` (deletion initiated)

#### Get EI Job Status

```http
GET /A1-EI/v1/eijobs/{ei_job_id}/status
```

**Parameters**:
- `ei_job_id` (string, path) - EI job identifier

**Response**:
```json
{
  "status": "ENABLED",
  "producers": ["traffic-stats-producer-001", "traffic-stats-producer-002"],
  "last_updated": "2024-01-15T10:05:00Z",
  "status_info": {
    "execution_count": 24,
    "last_execution_duration_ms": 1250,
    "average_execution_duration_ms": 1180,
    "success_rate": 0.958,
    "error_count": 1,
    "last_error": "Temporary connection timeout to cell_004"
  }
}
```

**Status Values**:
- `ENABLED` - Job is active and executing
- `DISABLED` - Job is disabled/paused

## Health and Monitoring Endpoints

### Health Check

```http
GET /health
```

**Response**:
```json
{
  "status": "UP",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h15m30s",
  "components": {
    "database": "UP",
    "redis": "UP",
    "near_rt_ric": "UP",
    "oauth2_provider": "UP"
  },
  "checks": [
    {
      "name": "database_connection",
      "status": "UP",
      "message": "Database connection healthy",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": "2ms"
    },
    {
      "name": "redis_connection",
      "status": "UP", 
      "message": "Redis cache operational",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": "1ms"
    }
  ]
}
```

### Readiness Check

```http
GET /ready
```

**Response**: Same format as health check, but indicates readiness to serve requests.

### Metrics

```http
GET /metrics
```

**Response**: Prometheus format metrics

**Key Metrics**:
- `nephoran_a1_requests_total` - Total API requests
- `nephoran_a1_request_duration_seconds` - Request duration histogram
- `nephoran_a1_policy_instances_total` - Active policy instances
- `nephoran_a1_consumers_total` - Registered consumers
- `nephoran_a1_circuit_breaker_state` - Circuit breaker state

## Error Handling

The API uses standard HTTP status codes and returns error details in JSON format.

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_POLICY_DATA",
    "message": "Policy data validation failed",
    "details": "Field 'priority_level' must be between 1 and 15",
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "req_1705315800123",
    "path": "/A1-P/v2/policytypes/20008/policies/invalid-policy"
  }
}
```

### HTTP Status Codes

| Status Code | Description | Common Causes |
|-------------|-------------|---------------|
| 200 | OK | Successful GET request |
| 201 | Created | Resource created successfully |
| 202 | Accepted | Async operation initiated |
| 204 | No Content | Resource deleted successfully |
| 400 | Bad Request | Invalid request data, validation errors |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource does not exist |
| 405 | Method Not Allowed | HTTP method not supported |
| 409 | Conflict | Resource conflict (e.g., dependencies exist) |
| 422 | Unprocessable Entity | Semantic validation errors |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server-side error |
| 502 | Bad Gateway | Downstream service error |
| 503 | Service Unavailable | Service temporarily unavailable |
| 504 | Gateway Timeout | Downstream service timeout |

### Common Error Codes

| Error Code | Description | Resolution |
|------------|-------------|------------|
| `INVALID_POLICY_TYPE_ID` | Policy type ID is invalid or out of range | Use valid integer between 1 and 2147483647 |
| `POLICY_TYPE_NOT_FOUND` | Policy type does not exist | Create the policy type first |
| `POLICY_INSTANCE_NOT_FOUND` | Policy instance does not exist | Check policy ID and type ID |
| `INVALID_POLICY_DATA` | Policy data validation failed | Check against policy type schema |
| `CONSUMER_NOT_FOUND` | Consumer is not registered | Register consumer first |
| `INVALID_CALLBACK_URL` | Consumer callback URL is invalid | Provide valid HTTP/HTTPS URL |
| `EI_TYPE_NOT_FOUND` | EI type does not exist | Create the EI type first |
| `EI_JOB_NOT_FOUND` | EI job does not exist | Check EI job ID |
| `AUTHENTICATION_REQUIRED` | Authentication token missing | Provide valid Bearer token |
| `INSUFFICIENT_PERMISSIONS` | Token lacks required scope | Check token scopes |
| `RATE_LIMIT_EXCEEDED` | Too many requests | Reduce request rate |
| `CIRCUIT_BREAKER_OPEN` | Service circuit breaker is open | Wait for circuit breaker to close |

## Rate Limiting

The API implements rate limiting to ensure fair usage and service stability.

### Rate Limit Headers

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1705315860
X-RateLimit-Window: 60
```

### Default Limits

| Endpoint Pattern | Requests per Minute | Burst Limit |
|------------------|--------------------|--------------| 
| `/A1-P/v2/*` | 1000 | 100 |
| `/A1-C/v1/*` | 500 | 50 |
| `/A1-EI/v1/*` | 800 | 80 |
| `/health` | 2000 | 200 |
| `/metrics` | 1000 | 100 |

### Rate Limit Exceeded Response

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1705315860

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded for endpoint",
    "details": "You have exceeded the rate limit of 1000 requests per minute",
    "timestamp": "2024-01-15T10:30:00Z",
    "retry_after": 30
  }
}
```

## OpenAPI Specification

The complete OpenAPI 3.0 specification is available at:

```
GET /swagger.json
GET /swagger.yaml
```

### Interactive API Documentation

Swagger UI is available at:
- **Production**: `https://a1-api.nephoran.io/swagger/`
- **Development**: `http://localhost:8080/swagger/`

### Code Generation

The OpenAPI specification can be used to generate client libraries:

```bash
# Generate Go client
openapi-generator-cli generate \
  -i https://a1-api.nephoran.io/swagger.yaml \
  -g go \
  -o ./a1-client-go

# Generate Python client  
openapi-generator-cli generate \
  -i https://a1-api.nephoran.io/swagger.yaml \
  -g python \
  -o ./a1-client-python

# Generate JavaScript client
openapi-generator-cli generate \
  -i https://a1-api.nephoran.io/swagger.yaml \
  -g javascript \
  -o ./a1-client-js
```

### Example Client Usage (Go)

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    a1client "github.com/nephoran/a1-client-go"
    "golang.org/x/oauth2"
)

func main() {
    // OAuth2 configuration
    config := &oauth2.Config{
        ClientID: "your-client-id",
        ClientSecret: "your-client-secret",
        Endpoint: oauth2.Endpoint{
            AuthURL:  "https://login.microsoftonline.com/tenant-id/oauth2/v2.0/authorize",
            TokenURL: "https://login.microsoftonline.com/tenant-id/oauth2/v2.0/token",
        },
        Scopes: []string{"api://a1-policy-service/a1:policy:read"},
    }
    
    // Get token (implementation depends on OAuth2 flow)
    token, err := getAccessToken(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create authenticated HTTP client
    client := config.Client(context.Background(), token)
    
    // Initialize A1 API client
    cfg := a1client.NewConfiguration()
    cfg.BasePath = "https://a1-api.nephoran.io"
    cfg.HTTPClient = client
    
    apiClient := a1client.NewAPIClient(cfg)
    
    // Get all policy types
    policyTypes, _, err := apiClient.PolicyInterfaceApi.GetPolicyTypes(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Policy types: %v\n", policyTypes)
    
    // Create a policy instance
    policyInstance := a1client.PolicyInstance{
        PolicyData: map[string]interface{}{
            "scope": map[string]interface{}{
                "ue_id": "*",
                "cell_list": []string{"cell_001", "cell_002"},
            },
            "qos_preference": map[string]interface{}{
                "priority_level": 1,
                "load_balancing": map[string]interface{}{
                    "site_a_weight": 70,
                    "site_b_weight": 30,
                },
            },
        },
    }
    
    _, err = apiClient.PolicyInterfaceApi.CreatePolicyInstance(
        context.Background(),
        20008,
        "example-policy-001",
        policyInstance,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Policy instance created successfully")
}
```

This comprehensive API reference provides all the necessary information for integrating with the A1 Policy Management Service, including detailed examples, error handling, and best practices for production use.