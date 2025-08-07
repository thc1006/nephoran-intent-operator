# O2 Infrastructure Management Service (IMS) API Specification

## Overview

The O2 Infrastructure Management Service (IMS) API provides comprehensive cloud infrastructure orchestration capabilities following O-RAN.WG6.O2ims-Interface-v01.01 specification. This RESTful API enables management of infrastructure resources, monitoring, and CNF lifecycle across multiple cloud providers.

### Key Features

- **Multi-Cloud Support**: Kubernetes, OpenStack, AWS, Azure, GCP, VMware, and Edge
- **O-RAN Compliance**: Full adherence to O-RAN Alliance specifications
- **CNF Lifecycle Management**: Complete container network function orchestration
- **Infrastructure Monitoring**: Real-time metrics, alarms, and health monitoring
- **Resource Lifecycle**: Creation, modification, scaling, and deletion operations
- **High Performance**: Sub-2s response times, 200+ concurrent operations

### API Versions

- **Current Version**: v1.0
- **Base URL**: `https://api.nephoran.io/o2ims/v1`
- **Content Type**: `application/json`
- **Authentication**: OAuth 2.0, JWT Bearer tokens

## Authentication

### OAuth 2.0 Flow

```http
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET
```

### Using Bearer Tokens

```http
Authorization: Bearer <your-jwt-token>
```

### Supported Scopes

- `o2ims:read` - Read access to all resources
- `o2ims:write` - Write access to all resources
- `o2ims:admin` - Administrative access
- `o2ims:monitoring` - Access to monitoring and metrics

## Service Information

### Get Service Information

Retrieve comprehensive service information including capabilities and supported providers.

```http
GET /o2ims/v1/
```

#### Response

```json
{
  "name": "Nephoran O2 IMS",
  "version": "1.0.0",
  "description": "O-RAN Infrastructure Management Service",
  "apiVersion": "v1",
  "specification": "O-RAN.WG6.O2ims-Interface-v01.01",
  "capabilities": [
    "InfrastructureInventory",
    "InfrastructureMonitoring",
    "InfrastructureProvisioning"
  ],
  "supported_providers": [
    "kubernetes",
    "openstack",
    "aws",
    "azure",
    "gcp",
    "vmware",
    "edge"
  ],
  "endpoints": {
    "resourcePools": "/o2ims/v1/resourcePools",
    "resourceTypes": "/o2ims/v1/resourceTypes",
    "resources": "/o2ims/v1/resources",
    "deployments": "/o2ims/v1/deployments",
    "subscriptions": "/o2ims/v1/subscriptions",
    "alarms": "/o2ims/v1/alarms"
  },
  "timestamp": "2025-01-07T10:30:00Z"
}
```

### Health Check

Monitor service health and component status.

```http
GET /health
```

#### Response

```json
{
  "status": "UP",
  "timestamp": "2025-01-07T10:30:00Z",
  "components": {
    "database": {
      "status": "UP",
      "details": {
        "connectionPool": "healthy",
        "responseTime": "5ms"
      }
    },
    "providers": {
      "status": "UP",
      "details": {
        "kubernetes": "UP",
        "aws": "UP",
        "azure": "DEGRADED"
      }
    },
    "monitoring": {
      "status": "UP",
      "details": {
        "metricsCollection": "active",
        "alerting": "active"
      }
    }
  }
}
```

## Infrastructure Inventory Management

### Resource Pools

Resource pools represent collections of infrastructure resources from cloud providers.

#### List Resource Pools

```http
GET /o2ims/v1/resourcePools
```

##### Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `filter` | string | Filter expression | `provider,eq,kubernetes` |
| `fields` | string | Comma-separated field list | `resourcePoolId,name,provider` |
| `limit` | integer | Maximum results (1-1000) | `50` |
| `offset` | integer | Pagination offset | `0` |
| `sort` | string | Sort specification | `name,asc` |

##### Filter Syntax

- Equality: `field,eq,value`
- Not equal: `field,neq,value`
- Greater than: `field,gt,value`
- Less than: `field,lt,value`
- In list: `field,in,value1;value2`
- Multiple filters: `filter1;filter2`

##### Response

```json
[
  {
    "resourcePoolId": "pool-k8s-us-west-2-001",
    "name": "Kubernetes US West 2 Pool",
    "description": "Primary Kubernetes cluster in US West 2",
    "location": "us-west-2",
    "oCloudId": "nephoran-cloud-001",
    "globalLocationId": "geo-us-west-2",
    "provider": "kubernetes",
    "region": "us-west-2",
    "zone": "us-west-2a",
    "capacity": {
      "cpu": {
        "total": "1000",
        "available": "750",
        "used": "250",
        "unit": "cores",
        "utilization": 25.0
      },
      "memory": {
        "total": "4000Gi",
        "available": "3000Gi",
        "used": "1000Gi",
        "unit": "bytes",
        "utilization": 25.0
      },
      "storage": {
        "total": "50Ti",
        "available": "35Ti",
        "used": "15Ti",
        "unit": "bytes",
        "utilization": 30.0
      }
    },
    "status": {
      "state": "AVAILABLE",
      "health": "HEALTHY",
      "utilization": 25.0,
      "lastHealthCheck": "2025-01-07T10:25:00Z"
    },
    "extensions": {
      "kubernetesVersion": "1.28.0",
      "nodeCount": 25,
      "supportedWorkloads": ["deployment", "statefulset", "daemonset"]
    },
    "createdAt": "2025-01-01T00:00:00Z",
    "updatedAt": "2025-01-07T10:00:00Z"
  }
]
```

#### Get Resource Pool

```http
GET /o2ims/v1/resourcePools/{resourcePoolId}
```

#### Create Resource Pool

```http
POST /o2ims/v1/resourcePools
Content-Type: application/json

{
  "resourcePoolId": "pool-aws-us-east-1-001",
  "name": "AWS US East 1 Pool",
  "description": "AWS EC2 resources in US East 1",
  "location": "us-east-1",
  "oCloudId": "nephoran-cloud-001",
  "provider": "aws",
  "region": "us-east-1",
  "capacity": {
    "cpu": {
      "total": "500",
      "available": "450",
      "used": "50",
      "unit": "vCPUs",
      "utilization": 10.0
    }
  },
  "extensions": {
    "awsConfig": {
      "vpcId": "vpc-12345678",
      "subnetIds": ["subnet-12345678", "subnet-87654321"],
      "instanceTypes": ["t3.medium", "t3.large", "c5.xlarge"]
    }
  }
}
```

#### Update Resource Pool

```http
PATCH /o2ims/v1/resourcePools/{resourcePoolId}
Content-Type: application/json

{
  "description": "Updated resource pool description",
  "capacity": {
    "cpu": {
      "total": "600",
      "available": "550",
      "used": "50"
    }
  }
}
```

#### Delete Resource Pool

```http
DELETE /o2ims/v1/resourcePools/{resourcePoolId}
```

##### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `cascade` | boolean | Delete dependent resources |
| `force` | boolean | Force delete ignoring dependencies |

### Resource Types

#### List Resource Types

```http
GET /o2ims/v1/resourceTypes
```

#### Response

```json
[
  {
    "resourceTypeId": "compute-standard-v1",
    "name": "Standard Compute Node",
    "description": "Standard compute resource for general workloads",
    "vendor": "Nephoran",
    "model": "Standard-Compute-v1",
    "version": "1.0.0",
    "specifications": {
      "category": "COMPUTE",
      "minResources": {
        "cpu": "1",
        "memory": "1Gi",
        "storage": "10Gi"
      },
      "maxResources": {
        "cpu": "64",
        "memory": "256Gi",
        "storage": "1Ti"
      },
      "defaultResources": {
        "cpu": "4",
        "memory": "8Gi",
        "storage": "100Gi"
      },
      "properties": {
        "architecture": "x86_64",
        "virtualization": "kvm",
        "networking": "sriov-capable"
      },
      "constraints": [
        "cpu must be multiple of 2",
        "memory must be multiple of 1Gi"
      ]
    },
    "supportedActions": [
      "CREATE", "DELETE", "UPDATE", "SCALE", "BACKUP", "RESTORE"
    ],
    "alarmDictionary": {
      "id": "compute-alarms-v1",
      "name": "Compute Node Alarms",
      "alarmDefinitions": [
        {
          "alarmDefinitionId": "cpu-high-utilization",
          "alarmName": "High CPU Utilization",
          "alarmDescription": "CPU utilization exceeds threshold",
          "proposedRepairActions": [
            "Scale out the workload",
            "Migrate workload to less utilized nodes"
          ]
        }
      ]
    },
    "createdAt": "2025-01-01T00:00:00Z",
    "updatedAt": "2025-01-07T09:00:00Z"
  }
]
```

### Resources

#### List Resources

```http
GET /o2ims/v1/resources
```

#### Get Resource

```http
GET /o2ims/v1/resources/{resourceId}
```

#### Response

```json
{
  "resourceId": "res-compute-001",
  "resourceName": "Production Compute Node 001",
  "resourceTypeId": "compute-standard-v1",
  "resourcePoolId": "pool-k8s-us-west-2-001",
  "description": "Production compute node for web services",
  "resourceState": "ACTIVE",
  "administrativeState": "UNLOCKED",
  "operationalState": "ENABLED",
  "usageState": "BUSY",
  "resourceSpec": {
    "cpu": "8",
    "memory": "16Gi",
    "storage": "200Gi",
    "network": "10Gbps"
  },
  "parentResourceId": null,
  "extensions": {
    "kubernetesNode": {
      "nodeName": "worker-node-001",
      "labels": {
        "node-type": "compute",
        "availability-zone": "us-west-2a"
      },
      "taints": []
    },
    "monitoring": {
      "metricsEndpoint": "https://metrics.example.com/node/worker-node-001",
      "healthCheckUrl": "https://health.example.com/node/worker-node-001"
    }
  },
  "createdAt": "2025-01-03T08:00:00Z",
  "updatedAt": "2025-01-07T09:30:00Z"
}
```

## Infrastructure Monitoring

### Alarms

#### List Alarms

```http
GET /o2ims/v1/alarms
```

##### Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `filter` | string | Filter alarms | `perceivedSeverity,eq,CRITICAL` |
| `fields` | string | Field selection | `alarmEventRecordId,perceivedSeverity` |
| `sort` | string | Sort order | `alarmRaisedTime,desc` |

#### Response

```json
[
  {
    "alarmEventRecordId": "alarm-001-20250107",
    "resourceTypeID": "compute-standard-v1",
    "resourceID": "res-compute-001",
    "alarmDefinitionID": "cpu-high-utilization",
    "probableCause": "CPU utilization exceeded 90% for 5 minutes",
    "specificProblem": "Node worker-node-001 CPU usage at 95%",
    "perceivedSeverity": "MAJOR",
    "alarmRaisedTime": "2025-01-07T10:15:00Z",
    "alarmChangedTime": "2025-01-07T10:15:00Z",
    "alarmAcknowledged": false,
    "alarmAcknowledgedTime": null,
    "extensions": {
      "currentCpuUsage": 95.2,
      "threshold": 90.0,
      "duration": "5m",
      "affectedPods": ["web-app-1", "web-app-2"]
    }
  }
]
```

#### Acknowledge Alarm

```http
PATCH /o2ims/v1/alarms/{alarmEventRecordId}
Content-Type: application/json

{
  "alarmAcknowledged": true,
  "alarmAcknowledgedTime": "2025-01-07T10:20:00Z"
}
```

### Subscriptions

#### Create Subscription

```http
POST /o2ims/v1/subscriptions
Content-Type: application/json

{
  "subscriptionId": "sub-critical-alarms-001",
  "consumerSubscriptionId": "external-monitoring-system-001",
  "filter": "(eq,perceivedSeverity,CRITICAL);(eq,resourceTypeId,compute-standard-v1)",
  "callback": "https://monitoring.company.com/webhooks/o2ims-alarms",
  "authentication": {
    "type": "bearer",
    "token": "your-webhook-token"
  },
  "extensions": {
    "retryPolicy": {
      "maxRetries": 3,
      "backoffFactor": 2,
      "initialDelay": "5s"
    },
    "batchSize": 10,
    "batchTimeout": "30s"
  }
}
```

#### List Subscriptions

```http
GET /o2ims/v1/subscriptions
```

## CNF Management

### CNF Packages

#### Register CNF Package

```http
POST /o2ims/v1/cnf/packages
Content-Type: application/json

{
  "packageId": "5g-amf-package-v1.2.0",
  "name": "5G AMF Network Function",
  "description": "5G Access and Mobility Management Function",
  "version": "1.2.0",
  "vendor": "Nephoran",
  "vnfdInfo": {
    "vnfdId": "amf-vnfd-v1.2",
    "vnfdVersion": "1.2",
    "vnfProvider": "Nephoran",
    "vnfProductName": "5G-AMF",
    "vnfSoftwareVersion": "1.2.0",
    "vnfdModel": "etsi-nfv-vnfd-2.8.1"
  },
  "packageType": "helm",
  "packageSource": {
    "type": "helm-repo",
    "repository": "nephoran-5g-charts",
    "chart": "5g-amf",
    "version": "1.2.0"
  },
  "operationalState": "ENABLED",
  "usageState": "NOT_IN_USE",
  "onboardingState": "ONBOARDED",
  "deploymentFlavors": [
    {
      "id": "small",
      "description": "Small deployment for development/testing",
      "requirements": {
        "cpu": "2",
        "memory": "4Gi",
        "storage": "20Gi",
        "replicas": 1
      }
    },
    {
      "id": "production",
      "description": "Production deployment with high availability",
      "requirements": {
        "cpu": "8",
        "memory": "16Gi",
        "storage": "100Gi",
        "replicas": 3
      }
    }
  ],
  "softwareImages": [
    {
      "id": "amf-main-image",
      "name": "5g-amf-main",
      "provider": "Nephoran",
      "version": "1.2.0",
      "containerFormat": "docker",
      "diskFormat": "raw",
      "minDisk": "1",
      "minRam": "2048",
      "imagePath": "registry.nephoran.io/5g/amf:1.2.0",
      "checksum": "sha256:a1b2c3d4e5f6..."
    }
  ],
  "dependencies": [
    {
      "name": "mongodb",
      "version": ">=6.0.0",
      "repository": "bitnami",
      "required": true
    },
    {
      "name": "redis",
      "version": ">=7.0.0",
      "repository": "bitnami",
      "required": true
    }
  ]
}
```

#### List CNF Packages

```http
GET /o2ims/v1/cnf/packages
```

### CNF Instances

#### Create CNF Instance

```http
POST /o2ims/v1/cnf/instances
Content-Type: application/json

{
  "vnfInstanceName": "AMF Production Instance 001",
  "vnfInstanceDescription": "Production AMF instance for region US West 2",
  "vnfdId": "amf-vnfd-v1.2",
  "flavourId": "production",
  "instantiationLevelId": "production-level-1",
  "additionalParams": {
    "namespace": "5g-core-production",
    "resourcePoolId": "pool-k8s-us-west-2-001",
    "values": {
      "image": {
        "repository": "registry.nephoran.io/5g/amf",
        "tag": "1.2.0",
        "pullPolicy": "IfNotPresent"
      },
      "replicaCount": 3,
      "resources": {
        "requests": {
          "cpu": "2",
          "memory": "4Gi"
        },
        "limits": {
          "cpu": "4",
          "memory": "8Gi"
        }
      },
      "service": {
        "type": "ClusterIP",
        "ports": [
          {"name": "sbi", "port": 80, "targetPort": 8080},
          {"name": "metrics", "port": 9090, "targetPort": 9090}
        ]
      },
      "config": {
        "amf": {
          "plmnList": [
            {"mcc": "001", "mnc": "01"}
          ],
          "servedGuamiList": [
            {
              "plmnId": {"mcc": "001", "mnc": "01"},
              "amfId": "cafe00"
            }
          ]
        }
      }
    }
  }
}
```

#### Response

```json
{
  "vnfInstanceId": "amf-instance-prod-001",
  "vnfInstanceName": "AMF Production Instance 001",
  "vnfInstanceDescription": "Production AMF instance for region US West 2",
  "vnfdId": "amf-vnfd-v1.2",
  "vnfProvider": "Nephoran",
  "vnfProductName": "5G-AMF",
  "vnfSoftwareVersion": "1.2.0",
  "vnfdVersion": "1.2",
  "flavourId": "production",
  "instantiationState": "INSTANTIATED",
  "metadata": {
    "createdAt": "2025-01-07T11:00:00Z",
    "lastModified": "2025-01-07T11:05:00Z"
  },
  "extensions": {
    "deploymentInfo": {
      "namespace": "5g-core-production",
      "helmReleaseName": "amf-prod-001",
      "kubernetesResources": [
        "deployment/amf-prod-001",
        "service/amf-prod-001",
        "configmap/amf-prod-001-config"
      ]
    }
  }
}
```

#### Scale CNF Instance

```http
POST /o2ims/v1/cnf/instances/{vnfInstanceId}/scale
Content-Type: application/json

{
  "type": "SCALE_OUT",
  "aspectId": "amf-aspect",
  "numberOfSteps": 2,
  "additionalParams": {
    "targetReplicas": 5
  }
}
```

#### Terminate CNF Instance

```http
POST /o2ims/v1/cnf/instances/{vnfInstanceId}/terminate
Content-Type: application/json

{
  "terminationType": "GRACEFUL",
  "gracefulTerminationTimeout": 300,
  "additionalParams": {
    "preserveData": true,
    "backupConfig": true
  }
}
```

## Multi-Cloud Provider Management

### List Providers

```http
GET /o2ims/v1/providers
```

#### Response

```json
[
  {
    "name": "kubernetes",
    "type": "container-orchestration",
    "status": "AVAILABLE",
    "region": "global",
    "capabilities": [
      "container-deployment",
      "auto-scaling",
      "load-balancing",
      "service-mesh",
      "cnf-management"
    ],
    "configuration": {
      "version": "1.28.0",
      "nodeCount": 25,
      "maxPods": 110,
      "networkPlugin": "calico"
    }
  },
  {
    "name": "aws",
    "type": "public-cloud",
    "status": "AVAILABLE",
    "region": "us-west-2",
    "capabilities": [
      "vm-deployment",
      "container-services",
      "managed-databases",
      "load-balancers",
      "auto-scaling"
    ],
    "configuration": {
      "availabilityZones": ["us-west-2a", "us-west-2b", "us-west-2c"],
      "vpcId": "vpc-12345678",
      "supportedInstanceTypes": ["t3.micro", "t3.small", "c5.large", "m5.xlarge"]
    }
  }
]
```

### Get Provider Details

```http
GET /o2ims/v1/providers/{providerName}
```

### Get Provider Health

```http
GET /o2ims/v1/providers/health
```

## Error Handling

### Standard Error Response

All API errors follow RFC 7807 Problem Details format:

```json
{
  "type": "https://api.nephoran.io/problems/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "The resource pool ID must be unique within the O-Cloud",
  "instance": "/o2ims/v1/resourcePools",
  "extra": {
    "field": "resourcePoolId",
    "value": "duplicate-pool-id",
    "constraint": "unique"
  }
}
```

### Common Error Types

| Status | Type | Description |
|--------|------|-------------|
| 400 | validation-error | Invalid input data |
| 401 | authentication-required | Missing or invalid authentication |
| 403 | authorization-failed | Insufficient permissions |
| 404 | resource-not-found | Requested resource does not exist |
| 409 | conflict | Resource conflict or dependency issue |
| 422 | unprocessable-entity | Valid JSON but business logic error |
| 429 | rate-limit-exceeded | Request rate limit exceeded |
| 500 | internal-error | Server internal error |
| 502 | provider-unavailable | Cloud provider service unavailable |
| 503 | service-unavailable | Service temporarily unavailable |

## Rate Limiting

### Headers

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1673097600
X-RateLimit-Retry-After: 3600
```

### Limits by Scope

| Scope | Requests per Hour | Burst Limit |
|-------|-------------------|-------------|
| o2ims:read | 10,000 | 100 |
| o2ims:write | 1,000 | 50 |
| o2ims:admin | 500 | 25 |

## Pagination

### Request Parameters

```http
GET /o2ims/v1/resourcePools?limit=50&offset=100
```

### Response Headers

```http
X-Total-Count: 1250
X-Pagination-Limit: 50
X-Pagination-Offset: 100
Link: </o2ims/v1/resourcePools?limit=50&offset=150>; rel="next",
      </o2ims/v1/resourcePools?limit=50&offset=50>; rel="prev"
```

## Webhooks and Notifications

### Webhook Payload

```json
{
  "eventType": "ALARM_RAISED",
  "timestamp": "2025-01-07T10:15:00Z",
  "subscriptionId": "sub-critical-alarms-001",
  "data": {
    "alarmEventRecordId": "alarm-001-20250107",
    "perceivedSeverity": "CRITICAL",
    "resourceId": "res-compute-001",
    "probableCause": "Node failure detected"
  },
  "signature": "sha256:a1b2c3d4e5f6..."
}
```

### Webhook Security

- HTTPS required for all webhook URLs
- Request signing using HMAC-SHA256
- Retry policy with exponential backoff
- Timeout after 30 seconds

## SDK and Client Libraries

### cURL Examples

#### Basic Authentication
```bash
# Get access token
curl -X POST https://api.nephoran.io/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET"

# Use API with token
curl -X GET https://api.nephoran.io/o2ims/v1/resourcePools \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Accept: application/json"
```

#### Create Resource Pool
```bash
curl -X POST https://api.nephoran.io/o2ims/v1/resourcePools \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d @resource-pool.json
```

### Python SDK Example

```python
from nephoran_o2ims import O2IMSClient

# Initialize client
client = O2IMSClient(
    base_url="https://api.nephoran.io/o2ims/v1",
    client_id="your-client-id",
    client_secret="your-client-secret"
)

# List resource pools
pools = client.resource_pools.list(
    filter="provider,eq,kubernetes",
    limit=50
)

# Create new resource pool
new_pool = client.resource_pools.create({
    "resourcePoolId": "my-new-pool",
    "name": "My Kubernetes Pool",
    "provider": "kubernetes",
    "oCloudId": "my-ocloud"
})

# Deploy CNF
cnf_instance = client.cnf.instances.create({
    "vnfInstanceName": "My AMF Instance",
    "vnfdId": "amf-vnfd-v1.2",
    "flavourId": "production"
})
```

## Performance Characteristics

### Service Level Agreements (SLAs)

| Metric | Target | Description |
|--------|--------|-------------|
| API Response Time P50 | < 100ms | 50th percentile response time |
| API Response Time P95 | < 500ms | 95th percentile response time |
| API Response Time P99 | < 1000ms | 99th percentile response time |
| Availability | 99.95% | Service uptime percentage |
| Throughput | > 1000 RPS | Requests per second capacity |
| Concurrent Operations | 200+ | Simultaneous intent processing |

### Optimization Tips

1. **Use Field Selection**: Include `fields` parameter to reduce payload size
2. **Implement Pagination**: Use `limit` and `offset` for large result sets
3. **Cache Frequently Accessed Data**: Leverage HTTP caching headers
4. **Use Webhooks**: Prefer event-driven patterns over polling
5. **Batch Operations**: Group multiple operations when possible

## Monitoring and Observability

### Metrics Endpoint

```http
GET /metrics
```

#### Key Metrics

- `o2ims_api_requests_total` - Total API requests
- `o2ims_api_request_duration_seconds` - Request duration histogram
- `o2ims_resource_pools_total` - Total resource pools
- `o2ims_cnf_instances_total` - Total CNF instances
- `o2ims_provider_health` - Provider health status

### Distributed Tracing

- **Trace ID**: `X-Trace-Id` header in requests/responses
- **Jaeger Integration**: Trace data exported to Jaeger
- **Span Correlation**: End-to-end request tracing

## Compliance and Security

### O-RAN Compliance

- Full adherence to O-RAN.WG6.O2ims-Interface-v01.01
- Standardized data models and API patterns
- Interoperability with O-RAN ecosystem components

### Security Features

- OAuth 2.0 / JWT authentication
- Role-based access control (RBAC)
- API rate limiting and throttling
- Request/response encryption (TLS 1.3)
- Audit logging for all operations
- Input validation and sanitization

## Support and Resources

### Documentation

- [API Reference](https://docs.nephoran.io/o2ims/api)
- [Developer Guide](https://docs.nephoran.io/o2ims/developer-guide)
- [Integration Examples](https://github.com/nephoran/o2ims-examples)

### Support Channels

- Email: support@nephoran.io
- Slack: [#nephoran-support](https://nephoran.slack.com)
- Issues: [GitHub Issues](https://github.com/nephoran/nephoran-intent-operator/issues)

### Status Page

- Service Status: https://status.nephoran.io
- Incident Reports: Available on status page
- Maintenance Windows: Announced via status page