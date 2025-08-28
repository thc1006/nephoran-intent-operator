# Nephoran Intent Operator - Istio Service Mesh Integration

## Phase 4 Enterprise Architecture - Service Mesh Implementation

This directory contains the complete Istio service mesh configuration for the Nephoran Intent Operator, implementing enterprise-grade traffic management, security, and observability.

## 📁 Directory Structure

```
deployments/istio/
├── istio-service-mesh-config.yaml  # Istio control plane configuration
├── virtual-services.yaml           # Traffic routing and management
├── destination-rules.yaml          # Load balancing and circuit breaking
├── security-policies.yaml          # mTLS and authorization policies
├── gateway.yaml                    # External access configuration
├── telemetry-config.yaml          # Advanced observability settings
└── README.md                       # This file
```

## 🚀 Quick Start

Deploy the complete Istio service mesh:

```bash
# Run the deployment script
./scripts/deploy-istio-mesh.sh

# Verify deployment
kubectl get pods -n istio-system
kubectl get pods -n nephoran-system
istioctl proxy-status
```

## 🔧 Components

### 1. **Istio Control Plane** (`istio-service-mesh-config.yaml`)
- Production-ready Istio operator configuration
- Multi-cluster support enabled
- Enhanced security settings with mTLS enforcement
- Optimized resource allocation
- Telemetry v2 configuration

### 2. **Virtual Services** (`virtual-services.yaml`)
- Intelligent traffic routing based on headers
- Canary deployment support (20/80 split)
- Priority-based routing for high-priority intents
- Multi-region traffic distribution
- Retry and timeout configurations

### 3. **Destination Rules** (`destination-rules.yaml`)
- Consistent hash load balancing for session affinity
- Circuit breaker configuration
- Outlier detection for automatic failover
- Connection pool management
- Service subsets for A/B testing

### 4. **Security Policies** (`security-policies.yaml`)
- Strict mTLS enforcement
- Zero-trust authorization model
- JWT authentication support
- Multi-tenancy isolation
- Service-to-service access control

### 5. **Gateway Configuration** (`gateway.yaml`)
- TLS 1.3 minimum protocol
- HTTPS redirect enforcement
- CORS policy configuration
- Rate limiting for admin endpoints
- Multi-region gateway support

### 6. **Telemetry Configuration** (`telemetry-config.yaml`)
- Custom business metrics
- Request classification
- Distributed tracing with Jaeger
- Access logging with filtering
- SLI/SLO metrics

## 🛡️ Security Features

### mTLS Configuration
- **Namespace-wide**: Strict mTLS for all services
- **Per-service**: Additional security policies
- **External services**: TLS origination for OpenAI API

### Authorization Policies
- **Default deny**: All traffic blocked by default
- **Explicit allow**: Service-specific permissions
- **JWT validation**: Token-based authentication
- **Tenant isolation**: Multi-tenancy support

## 📊 Observability

### Metrics
- Request rate, error rate, duration (RED metrics)
- Custom business metrics (intent type, priority, tenant)
- Circuit breaker and outlier detection metrics
- SLI/SLO tracking

### Tracing
- 10% sampling for general traffic
- 25% sampling for critical services
- Custom tags for correlation
- End-to-end request tracking

### Access Logs
- Conditional logging (errors, slow requests, debug)
- Structured log format with custom fields
- Tenant and intent type tracking

## 🔄 Traffic Management

### Load Balancing Strategies
- **LLM Processor**: Consistent hash on session ID
- **RAG API**: Least request algorithm
- **Weaviate**: Consistent hash on query class
- **Default**: Round-robin

### Circuit Breaking
- **Consecutive errors**: 5 failures trigger circuit break
- **Ejection time**: 30-120 seconds based on service
- **Max ejection**: 50% of endpoints
- **Health checking**: Automatic recovery detection

### Retry Configuration
- **LLM requests**: 3 attempts, 20s timeout
- **RAG queries**: 2 attempts, 15s timeout
- **O-RAN interfaces**: 3 attempts, 10s timeout

## 🌍 Multi-Region Support

### Region-based Routing
- Header-based region selection (`x-region`)
- Automatic failover between regions
- Weighted traffic distribution:
  - US-East: 40%
  - EU-West: 30%
  - Asia-Pacific: 30%

### East-West Gateway
- Multi-cluster communication
- ISTIO_MUTUAL TLS mode
- Automatic endpoint discovery

## 📋 Deployment Checklist

- [ ] Istio control plane installed
- [ ] Namespace labeled for injection
- [ ] All pods have sidecar proxies
- [ ] mTLS verification passed
- [ ] Virtual services applied
- [ ] Destination rules configured
- [ ] Security policies enforced
- [ ] Gateway accessible
- [ ] Telemetry data flowing
- [ ] Test traffic successful

## 🔍 Troubleshooting

### Common Issues

1. **Sidecar not injected**
   ```bash
   kubectl label namespace nephoran-system istio-injection=enabled
   kubectl rollout restart deployment -n nephoran-system
   ```

2. **mTLS errors**
   ```bash
   istioctl authn tls-check -n nephoran-system
   kubectl get peerauthentication -n nephoran-system
   ```

3. **Authorization denied**
   ```bash
   kubectl logs -n nephoran-system -l app=llm-processor -c istio-proxy
   kubectl get authorizationpolicy -n nephoran-system
   ```

4. **Traffic not routed**
   ```bash
   istioctl proxy-config routes <pod-name> -n nephoran-system
   istioctl analyze -n nephoran-system
   ```

### Useful Commands

```bash
# Check proxy status
istioctl proxy-status

# Analyze configuration
istioctl analyze -n nephoran-system

# View proxy configuration
istioctl proxy-config all <pod-name> -n nephoran-system

# Check mTLS status
istioctl authn tls-check -n nephoran-system

# View metrics
kubectl exec -n istio-system deployment/prometheus -- \
  promtool query instant 'istio_request_total{destination_service_namespace="nephoran-system"}'
```

## 📚 Additional Resources

- [Istio Documentation](https://istio.io/latest/docs/)
- [Istio Best Practices](https://istio.io/latest/docs/ops/best-practices/)
- [Istio Performance Tuning](https://istio.io/latest/docs/ops/performance-and-scalability/)
- [Nephoran Operator Guide](../../docs/operator-guide.md)