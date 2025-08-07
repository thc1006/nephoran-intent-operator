# O2 Infrastructure Management Service (IMS)

## Overview

The O2 Infrastructure Management Service is a production-ready implementation of the O-RAN Alliance O2 interface specification (O-RAN.WG6.O2ims-Interface-v01.01). It provides comprehensive cloud infrastructure management capabilities for telecommunications network functions, enabling standardized resource discovery, provisioning, monitoring, and lifecycle management across multi-cloud environments.

## Features

### Core Capabilities

- **O-RAN Compliant API**: Full implementation of O-RAN O2 interface specification
- **Multi-Cloud Support**: Native support for Kubernetes, AWS, Azure, and Google Cloud
- **CNF Lifecycle Management**: Complete container network function deployment and management
- **Resource Discovery**: Automated infrastructure resource discovery and inventory management
- **Performance Monitoring**: Real-time metrics collection and alerting
- **Standards Compliance**: Full O-RAN Alliance specification compliance with validation testing

### Enterprise-Grade Features

- **High Availability**: 99.95% uptime with automatic failover and recovery
- **Scalability**: Support for 200+ concurrent operations with sub-2-second latency
- **Security**: OAuth2 authentication, RBAC, and comprehensive audit logging
- **Multi-Tenancy**: Isolated resource pools with quotas and policy enforcement
- **Observability**: Prometheus metrics, Grafana dashboards, and distributed tracing

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          O2 IMS Architecture                        │
├─────────────────────────────────────────────────────────────────────┤
│  REST API Layer (O-RAN Compliant)                                  │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Resource Pools │ Resource Types │ Instances │ Monitoring   │    │
│  └─────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────┤
│  Service Layer                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Inventory Mgmt │ Lifecycle Mgmt │ Policy Engine │ Metrics   │    │
│  └─────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────┤
│  Provider Abstraction Layer                                        │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Kubernetes │ AWS │ Azure │ GCP │ OpenStack │ VMware        │    │
│  └─────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────┤
│  Infrastructure Layer                                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Compute │ Storage │ Network │ Accelerators │ Edge Devices   │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.24+)
- Go 1.24+ for development
- Docker for containerized deployment
- Helm 3.0+ for production deployment

### Installation

#### Development Environment

```bash
# Clone the repository
git clone https://github.com/nephoran/nephoran-intent-operator.git
cd nephoran-intent-operator

# Install dependencies
go mod download

# Run tests
make test-o2

# Start development server
make run-o2-dev
```

#### Production Deployment

```bash
# Install via Helm
helm repo add nephoran https://charts.nephoran.com
helm repo update
helm install o2-ims nephoran/o2-ims --namespace o2-system --create-namespace

# Verify installation
kubectl get pods -n o2-system
kubectl get svc -n o2-system
```

### Basic Usage

#### Check Service Status

```bash
curl -X GET http://localhost:8080/o2ims/v1/
```

#### Create Resource Pool

```bash
curl -X POST http://localhost:8080/o2ims/v1/resourcePools \
  -H "Content-Type: application/json" \
  -d '{
    "resourcePoolId": "my-k8s-pool",
    "name": "Kubernetes Resource Pool",
    "description": "Primary Kubernetes cluster pool",
    "location": "us-east-1",
    "oCloudId": "my-ocloud",
    "provider": "kubernetes",
    "region": "us-east-1",
    "zone": "us-east-1a"
  }'
```

#### List Resource Pools

```bash
curl -X GET http://localhost:8080/o2ims/v1/resourcePools
```

## API Reference

### Service Information

- `GET /o2ims/v1/` - Get service information and capabilities

### Resource Pools

- `GET /o2ims/v1/resourcePools` - List resource pools
- `POST /o2ims/v1/resourcePools` - Create resource pool
- `GET /o2ims/v1/resourcePools/{poolId}` - Get specific resource pool
- `PUT /o2ims/v1/resourcePools/{poolId}` - Update resource pool
- `DELETE /o2ims/v1/resourcePools/{poolId}` - Delete resource pool

### Resource Types

- `GET /o2ims/v1/resourceTypes` - List resource types
- `POST /o2ims/v1/resourceTypes` - Create resource type
- `GET /o2ims/v1/resourceTypes/{typeId}` - Get specific resource type
- `PUT /o2ims/v1/resourceTypes/{typeId}` - Update resource type
- `DELETE /o2ims/v1/resourceTypes/{typeId}` - Delete resource type

### Resource Instances

- `GET /o2ims/v1/resources` - List resource instances
- `POST /o2ims/v1/resources` - Create resource instance
- `GET /o2ims/v1/resources/{resourceId}` - Get specific resource instance
- `PUT /o2ims/v1/resources/{resourceId}` - Update resource instance
- `DELETE /o2ims/v1/resources/{resourceId}` - Delete resource instance

### Monitoring & Alarms

- `GET /o2ims/v1/alarms` - List alarms
- `POST /o2ims/v1/alarms` - Create alarm
- `GET /o2ims/v1/alarms/{alarmId}` - Get specific alarm
- `PATCH /o2ims/v1/alarms/{alarmId}` - Update alarm (e.g., acknowledge)
- `DELETE /o2ims/v1/alarms/{alarmId}` - Delete alarm

### Subscriptions

- `GET /o2ims/v1/subscriptions` - List subscriptions
- `POST /o2ims/v1/subscriptions` - Create subscription
- `GET /o2ims/v1/subscriptions/{subscriptionId}` - Get subscription
- `DELETE /o2ims/v1/subscriptions/{subscriptionId}` - Delete subscription

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `O2_IMS_PORT` | Server port | `8080` | No |
| `O2_IMS_HOST` | Server host | `0.0.0.0` | No |
| `O2_IMS_TLS_ENABLED` | Enable TLS | `false` | No |
| `O2_IMS_TLS_CERT_PATH` | TLS certificate path | - | If TLS enabled |
| `O2_IMS_TLS_KEY_PATH` | TLS private key path | - | If TLS enabled |
| `O2_IMS_LOG_LEVEL` | Log level | `info` | No |
| `O2_IMS_DATABASE_URL` | Database connection URL | - | Yes |
| `O2_IMS_METRICS_ENABLED` | Enable metrics | `true` | No |
| `O2_IMS_TRACING_ENABLED` | Enable tracing | `false` | No |

### Configuration File

```yaml
# config/o2-ims.yaml
server:
  address: "0.0.0.0"
  port: 8080
  tls:
    enabled: false
    certFile: ""
    keyFile: ""

database:
  type: "postgresql"
  host: "localhost"
  port: 5432
  database: "o2ims"
  username: "o2user"
  password: "password"
  sslMode: "require"

providers:
  kubernetes:
    enabled: true
    kubeconfig: "/etc/kubeconfig"
  aws:
    enabled: false
    region: "us-east-1"
    accessKeyId: ""
    secretAccessKey: ""
  azure:
    enabled: false
    subscriptionId: ""
    tenantId: ""
    clientId: ""
    clientSecret: ""
  gcp:
    enabled: false
    projectId: ""
    credentialsFile: ""

monitoring:
  metrics:
    enabled: true
    path: "/metrics"
  tracing:
    enabled: false
    endpoint: "http://jaeger:14268/api/traces"

logging:
  level: "info"
  format: "json"
  output: "stdout"
```

## Testing

### Unit Tests

```bash
# Run all O2 unit tests
make test-o2-unit

# Run specific test package
go test ./tests/o2/unit/...

# Run with coverage
go test -coverprofile=coverage.out ./tests/o2/unit/...
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Run integration tests (requires test environment)
make test-o2-integration

# Run specific integration test
go test ./tests/o2/integration/api_endpoints_test.go

# Run multi-cloud integration tests
go test ./tests/o2/integration/multi_cloud_test.go
```

### Performance Tests

```bash
# Run load tests
make test-o2-load

# Run benchmarks
go test -bench=. ./tests/o2/performance/...

# Generate performance profile
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./tests/o2/performance/...
go tool pprof cpu.prof
```

### Compliance Tests

```bash
# Run O-RAN compliance validation
make test-o2-compliance

# Validate specific O-RAN requirements
go test ./tests/o2/compliance/oran_compliance_test.go
```

## Monitoring & Observability

### Metrics

The O2 IMS service exposes Prometheus metrics at `/metrics`:

- **Request Metrics**: HTTP request rate, latency, and error rate
- **Resource Metrics**: Resource pool utilization, instance counts
- **Performance Metrics**: Database query times, provider API latency
- **Business Metrics**: CNF deployment success rate, SLA compliance

### Health Checks

- `GET /health` - Health status of all components
- `GET /ready` - Readiness status for load balancer
- `GET /metrics` - Prometheus metrics endpoint

### Logging

Structured logging with configurable levels and formats:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "service": "o2-ims",
  "component": "resource-manager",
  "message": "Resource pool created successfully",
  "resourcePoolId": "k8s-pool-001",
  "provider": "kubernetes",
  "duration_ms": 150,
  "correlation_id": "req-abc123"
}
```

## Security

### Authentication

- **OAuth2**: Support for multiple OAuth2 providers
- **API Keys**: Service-to-service authentication
- **mTLS**: Mutual TLS for secure service communication

### Authorization

- **RBAC**: Role-based access control with fine-grained permissions
- **Resource-level**: Per-resource access control
- **Multi-tenancy**: Tenant isolation and quota enforcement

### Security Headers

- Content Security Policy (CSP)
- HTTP Strict Transport Security (HSTS)
- X-Frame-Options
- X-Content-Type-Options

## Troubleshooting

### Common Issues

#### Service Won't Start

1. Check configuration file syntax
2. Verify database connectivity
3. Check port availability
4. Review logs for specific errors

#### High Latency

1. Check database performance
2. Monitor provider API response times
3. Review resource allocation
4. Check for network issues

#### Resource Discovery Issues

1. Verify provider credentials
2. Check provider API quotas
3. Review RBAC permissions
4. Validate network connectivity

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
export O2_IMS_LOG_LEVEL=debug
./o2-ims
```

### Support

- **Documentation**: [Full API documentation](./api/)
- **Examples**: [Usage examples](./examples/)
- **Issues**: [GitHub Issues](https://github.com/nephoran/nephoran-intent-operator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/nephoran/nephoran-intent-operator/discussions)

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Run tests: `make test-o2`
4. Commit changes: `git commit -am 'Add my feature'`
5. Push branch: `git push origin feature/my-feature`
6. Create Pull Request

### Development Setup

```bash
# Install development tools
make install-tools

# Run pre-commit hooks
make pre-commit

# Generate API documentation
make docs-generate
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../../LICENSE) file for details.

## Acknowledgments

- O-RAN Alliance for the O2 interface specifications
- Kubernetes SIG-Network for CNI standards
- Cloud Native Computing Foundation for cloud-native best practices