# Nephoran Certificate Automation System

## Overview

The Nephoran Certificate Automation System provides enterprise-grade automatic certificate provisioning and rotation with zero-downtime capabilities for the Nephoran Intent Operator. This system implements a complete certificate lifecycle management solution designed for cloud-native telecommunications environments.

## Key Features

### 🚀 **Zero-Downtime Certificate Rotation**
- Coordinated rotation across clustered deployments
- Rolling updates with health verification
- Automatic rollback on failure
- Load balancer-aware rotation strategies

### 🔍 **Intelligent Service Discovery**
- Automatic detection of services requiring certificates
- Template-based certificate provisioning
- Service mesh integration (Istio, Linkerd, Consul)
- Pre-provisioning for faster deployment

### ⚡ **High-Performance Caching**
- Multi-level cache (L1: hot cache, L2: warm cache)
- Pre-provisioned certificate pool
- Batch operations for high throughput
- Intelligent cache warming and cleanup

### 💪 **Enterprise-Grade Reliability**
- Comprehensive health checking (HTTP, HTTPS, gRPC, TCP)
- Circuit breaker patterns for fault tolerance
- Comprehensive monitoring and alerting
- Disaster recovery capabilities

### 🔒 **Security-First Design**
- mTLS communication between components
- Encrypted private key storage
- Comprehensive audit logging
- RBAC and network policies

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Service        │    │  Rotation       │    │  Health         │
│  Discovery      │────│  Coordinator    │────│  Checker        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌─────────────────┐              │
         └──────────────│  Automation     │──────────────┘
                        │  Engine         │
         ┌──────────────│  (Core)         │──────────────┐
         │              └─────────────────┘              │
┌─────────────────┐                            ┌─────────────────┐
│  Performance    │                            │  CA Manager     │
│  Cache          │                            │  Integration    │
└─────────────────┘                            └─────────────────┘
```

## Components

### 1. Automation Engine (Core)
The central component that orchestrates all certificate operations:

- **Certificate Provisioning**: Multi-worker queue processing
- **Renewal Management**: Automatic renewal based on configurable thresholds
- **Validation Framework**: Real-time certificate validation with caching
- **Revocation Checking**: CRL and OCSP validation
- **Kubernetes Integration**: Native integration with admission controllers

### 2. Service Discovery
Automatically discovers and provisions certificates for services:

- **Service Watchers**: Monitor Kubernetes services, pods, and ingresses
- **Template Matching**: Rule-based template selection
- **Auto-Provisioning**: Immediate certificate provisioning for new services
- **Pre-Provisioning**: Prepare certificates before service startup

### 3. Rotation Coordinator
Manages zero-downtime certificate rotation:

- **Session Management**: Track rotation sessions across clustered deployments
- **Batch Processing**: Coordinated rolling updates
- **Health Validation**: Continuous health monitoring during rotation
- **Rollback Capability**: Automatic rollback on failure detection

### 4. Performance Cache
Optimizes certificate operations through intelligent caching:

- **L1 Cache**: Hot in-memory cache for frequently accessed certificates
- **L2 Cache**: Larger warm cache for broader certificate storage
- **Pre-Provisioning Cache**: Pool of pre-generated certificates
- **Batch Operations**: Efficient bulk certificate processing

### 5. Health Checker
Comprehensive health monitoring for certificate rotation:

- **Multi-Protocol Support**: HTTP, HTTPS, gRPC, TCP health checks
- **TLS Validation**: Certificate chain and expiry validation
- **Continuous Monitoring**: Real-time health tracking
- **Performance Metrics**: Detailed health check analytics

## Configuration

### Basic Configuration

```yaml
automation_config:
  provisioning_enabled: true
  auto_inject_certificates: true
  service_discovery_enabled: true
  provisioning_workers: 5
  
  renewal_enabled: true
  renewal_threshold: 720h  # 30 days
  renewal_workers: 3
  graceful_renewal_enabled: true
  
  kubernetes_integration:
    enabled: true
    namespaces: ["nephoran-system", "default"]
    annotation_prefix: "nephoran.com"
```

### Service Discovery Configuration

```yaml
service_discovery:
  enabled: true
  auto_provision_enabled: true
  template_matching:
    enabled: true
    default_template: "default"
    matching_rules:
      - name: "microservice"
        label_selectors:
          app.kubernetes.io/component: "microservice"
        template: "microservice"
```

### Performance Cache Configuration

```yaml
performance_cache:
  l1_cache_size: 1000
  l1_cache_ttl: 5m
  l2_cache_size: 5000
  l2_cache_ttl: 30m
  pre_provisioning_enabled: true
  batch_operations_enabled: true
```

## Deployment

### Prerequisites

1. **Kubernetes Cluster**: Version 1.20+
2. **cert-manager**: For certificate provisioning backend
3. **Prometheus**: For monitoring (optional)
4. **Istio/Service Mesh**: For advanced features (optional)

### Quick Start

1. **Deploy the system**:
```bash
kubectl apply -f deployments/cert-automation/certificate-automation-deployment.yaml
```

2. **Verify deployment**:
```bash
kubectl get pods -n nephoran-system -l app.kubernetes.io/component=certificate-automation
```

3. **Check logs**:
```bash
kubectl logs -n nephoran-system -l app.kubernetes.io/component=certificate-automation
```

### Configuration Customization

1. **Edit ConfigMap**:
```bash
kubectl edit configmap nephoran-cert-automation-config -n nephoran-system
```

2. **Restart deployment**:
```bash
kubectl rollout restart deployment nephoran-cert-automation -n nephoran-system
```

## Usage

### Automatic Certificate Provisioning

Services can request automatic certificate provisioning through annotations:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    nephoran.com/auto-certificate: "true"
    nephoran.com/certificate-template: "microservice"
    nephoran.com/external-dns: "api.example.com,app.example.com"
spec:
  ports:
  - port: 443
    name: https
  selector:
    app: my-app
```

### Pod Certificate Injection

Pods can have certificates automatically injected:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      annotations:
        nephoran.com/inject-certificate: "true"
        nephoran.com/certificate-template: "microservice"
    spec:
      containers:
      - name: app
        image: my-app:latest
        env:
        - name: TLS_CERT_PATH
          value: "/etc/ssl/certs/service/tls.crt"
        - name: TLS_KEY_PATH
          value: "/etc/ssl/certs/service/tls.key"
```

### Zero-Downtime Rotation

Enable coordinated rotation for critical services:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: critical-service
  annotations:
    nephoran.com/auto-certificate: "true"
    nephoran.com/zero-downtime-rotation: "true"
    nephoran.com/health-check-endpoint: "/health"
    nephoran.com/rotation-batch-size: "2"
```

## Certificate Templates

### Default Templates

The system includes several built-in templates:

- **default**: RSA 2048, 90-day validity, basic server auth
- **microservice**: ECDSA P-256, 30-day validity, client+server auth
- **public-facing**: RSA 4096, 1-year validity, server auth only
- **gateway**: ECDSA P-384, 6-month validity, gateway-specific DNS

### Custom Templates

Create custom certificate templates:

```yaml
certificate_templates:
  custom:
    name: "custom"
    description: "Custom certificate template"
    validity_duration: "2160h"  # 90 days
    key_type: "ECDSA"
    key_size: 256
    dns_name_patterns:
      - "{service}.{namespace}.custom.local"
    extended_key_usages:
      - "ServerAuth"
      - "ClientAuth"
    auto_renew: true
    renewal_threshold: "720h"  # 30 days
```

## Monitoring

### Metrics

The system exposes comprehensive Prometheus metrics:

- `certificate_provisioning_duration`: Time taken to provision certificates
- `certificate_renewal_success_rate`: Success rate of certificate renewals
- `active_certificates_count`: Number of active certificates
- `cache_hit_rate`: Performance cache efficiency
- `rotation_session_duration`: Time taken for coordinated rotations

### Health Checks

Health endpoints are available:

- `/health`: General system health
- `/ready`: Readiness for traffic
- `/metrics`: Prometheus metrics

### Alerting

Pre-configured alerts include:

- Certificate provisioning failures
- Certificate renewal failures
- Certificates expiring soon
- Service health check failures
- Cache performance degradation

## Security

### mTLS Communication

All internal communication uses mTLS:

```yaml
security:
  mtls:
    enabled: true
    client_cert_required: true
    verify_client_cert: true
```

### RBAC

Comprehensive RBAC permissions:

- Certificate and secret management
- Service discovery across namespaces
- Admission webhook configuration
- Event creation for audit logging

### Network Policies

Restrictive network policies limit communication:

- Allow Kubernetes API access
- Allow cert-manager integration
- Allow metrics scraping
- Block unnecessary traffic

## Troubleshooting

### Common Issues

1. **Provisioning Failures**:
   ```bash
   # Check cert-manager status
   kubectl get certificates -A
   kubectl describe certificate <name> -n <namespace>
   
   # Check automation engine logs
   kubectl logs -n nephoran-system -l app.kubernetes.io/component=certificate-automation
   ```

2. **Service Discovery Not Working**:
   ```bash
   # Verify service annotations
   kubectl get svc <service> -o yaml
   
   # Check discovery logs
   kubectl logs -n nephoran-system -l app.kubernetes.io/component=certificate-automation -c cert-automation | grep "service discovery"
   ```

3. **Rotation Failures**:
   ```bash
   # Check rotation sessions
   kubectl logs -n nephoran-system -l app.kubernetes.io/component=certificate-automation | grep "rotation"
   
   # Verify health checks
   kubectl logs -n nephoran-system -l app.kubernetes.io/component=certificate-automation | grep "health check"
   ```

### Debug Mode

Enable debug logging:

```yaml
logging:
  level: "debug"
  debug_logging: true
```

### Performance Tuning

Optimize for your environment:

```yaml
automation_config:
  provisioning_workers: 10  # Increase for higher load
  max_concurrent_operations: 100
  batch_size: 20

performance_cache:
  l1_cache_size: 2000  # Increase for better hit rates
  pre_provisioning_size: 200
```

## Best Practices

### Service Configuration

1. **Use appropriate templates**: Choose templates based on service requirements
2. **Configure health checks**: Enable health checks for zero-downtime rotation
3. **Set renewal thresholds**: Configure appropriate renewal windows
4. **Monitor certificate expiry**: Set up alerting for expiring certificates

### Performance Optimization

1. **Enable caching**: Use performance cache for high-throughput scenarios
2. **Batch operations**: Enable batch processing for bulk operations
3. **Pre-provisioning**: Use pre-provisioning for frequently deployed services
4. **Resource tuning**: Adjust worker counts based on load

### Security Hardening

1. **Enable mTLS**: Use mutual TLS for all internal communication
2. **Network policies**: Implement restrictive network policies
3. **RBAC**: Use minimal required permissions
4. **Audit logging**: Enable comprehensive audit logging

## Integration Examples

### Istio Service Mesh

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: my-service-tls
spec:
  host: my-service
  trafficPolicy:
    tls:
      mode: MUTUAL
      clientCertificate: /etc/ssl/certs/service/tls.crt
      privateKey: /etc/ssl/certs/service/tls.key
      caCertificates: /etc/ssl/certs/service/ca.crt
```

### External DNS Integration

```yaml
apiVersion: v1
kind: Service
metadata:
  name: public-api
  annotations:
    nephoran.com/auto-certificate: "true"
    nephoran.com/certificate-template: "public-facing"
    external-dns.alpha.kubernetes.io/hostname: "api.example.com"
```

### GitOps Integration

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  annotations:
    nephoran.com/auto-certificate: "true"
    nephoran.com/pre-provision-certificate: "true"
spec:
  # ... ArgoCD application spec
```

## FAQ

**Q: How does zero-downtime rotation work?**
A: The system coordinates rotation across service instances in batches, performing health checks between batches and providing automatic rollback if issues are detected.

**Q: Can I use external CAs?**
A: Yes, the system supports multiple CA backends including cert-manager, Vault, and custom external CAs.

**Q: How do I backup certificates?**
A: Certificates are automatically backed up to Kubernetes secrets and can be exported to external storage systems.

**Q: What happens if the automation system fails?**
A: Existing certificates continue to work, and the system includes high availability deployment options with automatic failover.

**Q: How do I migrate existing certificates?**
A: The system can import existing certificates and gradually migrate them to the automated lifecycle management.

## Support and Contributing

### Support

- **Documentation**: Complete API reference and configuration guide
- **Community**: GitHub discussions and issue tracking
- **Enterprise**: Commercial support available

### Contributing

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Submit a pull request

### License

Licensed under the Apache License 2.0. See LICENSE file for details.

---

*This documentation covers the complete Certificate Automation System for the Nephoran Intent Operator. For additional information, please refer to the API reference and configuration examples.*