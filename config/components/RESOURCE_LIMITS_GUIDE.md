# Nephoran Intent Operator Resource Limits Configuration Guide

This guide explains the comprehensive resource limits configuration for the Nephoran Intent Operator, designed to prevent memory issues and ensure stable operation in telecom environments.

## Overview

The Nephoran Intent Operator processes telecom knowledge bases and AI contexts that can consume significant memory. Proper resource limits are essential to prevent Out-Of-Memory (OOM) issues and ensure stable operation.

## Resource Configuration Summary

### Controller Manager
- **CPU Limit**: 200m (0.2 cores)
- **CPU Request**: 100m (0.1 cores)  
- **Memory Limit**: 512Mi
- **Memory Request**: 256Mi
- **Replicas**: 2 (for high availability)

### LLM Processor (AI Component)
- **CPU Limit**: 1000m (1 core)
- **CPU Request**: 200m (0.2 cores)
- **Memory Limit**: 1Gi (higher for AI processing)
- **Memory Request**: 256Mi
- **Replicas**: 2-10 (auto-scaling)

### Webhook Component
- **CPU Limit**: 200m (0.2 cores)
- **CPU Request**: 50m (0.05 cores)
- **Memory Limit**: 256Mi (lower resources)
- **Memory Request**: 64Mi
- **Replicas**: 2-5 (conservative scaling)

### RAG API (Vector Database Interface)
- **CPU Limit**: 500m (0.5 cores)
- **CPU Request**: 100m (0.1 cores)
- **Memory Limit**: 512Mi
- **Memory Request**: 128Mi
- **Replicas**: 2-8 (moderate scaling)

## Files Structure

```
config/components/
├── manager.yaml                    # Updated with proper resource limits
├── llm-processor-resources.yaml   # AI workload configuration
├── webhook-resources.yaml         # Admission controller configuration
├── rag-api-resources.yaml         # Vector database interface
├── controller-hpa.yaml            # HPA for main controller
├── hpa-configurations.yaml        # Comprehensive HPA configs
├── resource-management.yaml       # Quotas and limit ranges
├── deploy-resource-limits.sh      # Deployment script
└── kustomization.yaml             # Updated orchestration
```

## Key Features Implemented

### 1. Resource Quotas and Limits
- **Namespace-level quotas** to prevent resource exhaustion
- **Container-level limits** for each component
- **Pod-level limits** for safety
- **Storage limits** for persistent volumes

### 2. Horizontal Pod Autoscaler (HPA)
- **Multi-metric scaling** (CPU, memory, custom metrics)
- **Component-specific scaling policies**
- **Conservative scaling** for critical components
- **Aggressive scaling** for AI workloads

### 3. Pod Disruption Budgets (PDB)
- **Minimum availability** during updates
- **High availability** for critical components
- **Graceful disruption** handling

### 4. Enhanced Health Probes
- **Startup probes** for slow-starting components
- **Liveness probes** with appropriate timeouts
- **Readiness probes** for traffic management
- **Extended timeouts** for telecom workloads

### 5. Security Enhancements
- **Non-root containers** for security
- **Read-only root filesystems**
- **Capability dropping**
- **Security contexts** with best practices

## Deployment Instructions

### Quick Deployment
```bash
# Deploy all resource configurations
./config/components/deploy-resource-limits.sh deploy

# Verify deployment
./config/components/deploy-resource-limits.sh verify

# Check status
./config/components/deploy-resource-limits.sh status
```

### Manual Deployment
```bash
# Create namespace
kubectl create namespace nephoran-system

# Apply resource management
kubectl apply -f config/components/resource-management.yaml

# Apply controller with limits
kubectl apply -f config/manager/manager.yaml

# Apply component resources
kubectl apply -f config/components/llm-processor-resources.yaml
kubectl apply -f config/components/webhook-resources.yaml
kubectl apply -f config/components/rag-api-resources.yaml

# Apply HPA configurations
kubectl apply -f config/components/hpa-configurations.yaml
kubectl apply -f config/components/controller-hpa.yaml
```

### Using Kustomize
```bash
# Deploy everything with kustomize
kubectl apply -k config/components/
```

## Resource Requirements

### Minimum Cluster Resources
- **CPU**: 2 cores minimum
- **Memory**: 8GB minimum
- **Storage**: 20GB for persistent volumes
- **Nodes**: 3 nodes for high availability

### Recommended Cluster Resources
- **CPU**: 4-8 cores
- **Memory**: 16-32GB
- **Storage**: 100GB SSD storage
- **Nodes**: 5+ nodes across multiple AZs

## Monitoring and Alerting

### Key Metrics to Monitor
1. **Memory Usage**
   - `container_memory_usage_bytes`
   - `container_memory_working_set_bytes`
   - Memory utilization percentages

2. **CPU Usage**
   - `container_cpu_usage_seconds_total`
   - CPU throttling incidents
   - CPU utilization trends

3. **Resource Limits**
   - `kube_pod_container_resource_limits`
   - `kube_pod_container_resource_requests`
   - Resource quota utilization

4. **HPA Metrics**
   - `kube_horizontalpodautoscaler_status_current_replicas`
   - Scaling events
   - Scaling decision metrics

### Recommended Alerts

#### Critical Alerts
- **OOM Kills**: Any pod restart due to memory issues
- **Resource Exhaustion**: Namespace quota exceeded
- **Component Unavailable**: All replicas of critical component down

#### Warning Alerts
- **High Memory Usage**: >80% of limit for 5 minutes
- **High CPU Usage**: >80% of limit for 5 minutes
- **HPA at Maximum**: HPA cannot scale further

#### Info Alerts
- **Scaling Events**: HPA scaling up/down
- **Resource Requests**: New resource requests rejected

### Grafana Dashboard Queries

```promql
# Memory usage by component
sum(container_memory_usage_bytes{namespace="nephoran-system"}) by (pod)

# CPU usage by component
sum(rate(container_cpu_usage_seconds_total{namespace="nephoran-system"}[5m])) by (pod)

# Resource limit utilization
(
  sum(container_memory_usage_bytes{namespace="nephoran-system"}) by (pod) /
  sum(container_spec_memory_limit_bytes{namespace="nephoran-system"}) by (pod)
) * 100

# HPA current vs desired replicas
kube_horizontalpodautoscaler_status_current_replicas{namespace="nephoran-system"}
```

## Troubleshooting

### Common Issues

#### 1. OOM Kills
**Symptoms**: Pods restarting frequently, memory alerts
```bash
# Check for OOM kills
kubectl get events -n nephoran-system | grep OOMKilled

# Check resource usage
kubectl top pods -n nephoran-system

# Increase memory limits
kubectl patch deployment llm-processor -n nephoran-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-processor","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
```

#### 2. CPU Throttling
**Symptoms**: High CPU usage, slow response times
```bash
# Check CPU metrics
kubectl top pods -n nephoran-system

# Check throttling events
kubectl get events -n nephoran-system | grep Throttled

# Increase CPU limits
kubectl patch deployment controller-manager -n nephoran-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"manager","resources":{"limits":{"cpu":"400m"}}}]}}}}'
```

#### 3. HPA Not Scaling
**Symptoms**: High resource usage but no scaling
```bash
# Check HPA status
kubectl get hpa -n nephoran-system

# Check metrics availability
kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes

# Ensure metrics server is running
kubectl get deployment metrics-server -n kube-system
```

#### 4. Resource Quota Exceeded
**Symptoms**: New pods cannot be created
```bash
# Check quota status
kubectl describe resourcequota -n nephoran-system

# Check current usage
kubectl get resourcequota -n nephoran-system -o yaml

# Increase quota if needed
kubectl patch resourcequota nephoran-system-quota -n nephoran-system -p '{"spec":{"hard":{"limits.memory":"32Gi"}}}'
```

### Performance Tuning

#### Memory Optimization
1. **Enable memory caching** in LLM processor
2. **Tune garbage collection** for Go applications
3. **Use memory-mapped files** for large datasets
4. **Implement memory pooling** for frequent allocations

#### CPU Optimization
1. **Enable CPU affinity** for compute-intensive pods
2. **Use GOMAXPROCS** environment variable
3. **Profile CPU usage** during peak loads
4. **Optimize goroutine usage** in controllers

#### Storage Optimization
1. **Use SSD storage classes** for databases
2. **Implement storage quotas** per component
3. **Enable storage monitoring** and alerts
4. **Regular storage cleanup** policies

## Advanced Configuration

### KEDA Integration
For advanced scaling based on external metrics:
```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: llm-processor-advanced-scaler
  namespace: nephoran-system
spec:
  scaleTargetRef:
    name: llm-processor
  minReplicaCount: 2
  maxReplicaCount: 20
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus:9090
      metricName: telecom_intent_queue_depth
      threshold: '10'
```

### Vertical Pod Autoscaler (VPA)
For automatic resource recommendation:
```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: llm-processor-vpa
  namespace: nephoran-system
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: llm-processor
      maxAllowed:
        memory: 4Gi
        cpu: 2
```

### Node Affinity for Telecom Workloads
```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: node.kubernetes.io/instance-type
          operator: In
          values:
          - c5.xlarge
          - c5.2xlarge
        - key: failure-domain.beta.kubernetes.io/zone
          operator: In
          values:
          - us-west-2a
          - us-west-2b
```

## Best Practices

### Resource Allocation
1. **Always set both requests and limits**
2. **Use appropriate QoS classes** (Guaranteed for critical components)
3. **Monitor resource utilization** regularly
4. **Adjust limits based on actual usage** patterns

### High Availability
1. **Deploy multiple replicas** for critical components
2. **Use Pod Disruption Budgets** appropriately
3. **Spread replicas across nodes** with anti-affinity
4. **Plan for node failures** and maintenance

### Security
1. **Run containers as non-root** users
2. **Use read-only root filesystems** where possible
3. **Drop all capabilities** by default
4. **Implement network policies** for traffic control

### Monitoring
1. **Set up comprehensive alerting** for all components
2. **Monitor both resource usage and business metrics**
3. **Implement SLA monitoring** for telecom requirements
4. **Regular performance reviews** and optimization

## Conclusion

This comprehensive resource limits configuration ensures the Nephoran Intent Operator runs reliably in production telecom environments. The configuration includes proper resource limits, auto-scaling, high availability, and monitoring to prevent memory issues and maintain optimal performance.

Regular monitoring and tuning based on actual usage patterns will help optimize the configuration for your specific deployment requirements.