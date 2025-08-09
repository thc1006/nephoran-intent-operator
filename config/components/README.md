# Nephoran Intent Operator - Resource Management Configuration

This directory contains comprehensive resource management configurations for the Nephoran Intent Operator, designed to ensure optimal performance, cost efficiency, and high availability across all components.

## Overview

The resource management system provides:

- **Resource Limits & Requests**: Proper CPU, memory, and storage allocation for each component
- **Auto-scaling**: Horizontal Pod Autoscaler (HPA) and KEDA-based scaling
- **High Availability**: Pod Disruption Budgets and anti-affinity rules
- **Cost Optimization**: Spot instance utilization and resource right-sizing
- **Performance Monitoring**: Metrics collection and alerting for resource usage

## Files Structure

```
config/components/
├── README.md                           # This documentation
├── manager.yaml                        # Main operator deployment
├── llm-processor-resources.yaml       # AI processor with high memory allocation
├── rag-api-resources.yaml            # Data retrieval service resources
├── webhook-resources.yaml            # Admission webhook with minimal resources
├── nephio-bridge-resources.yaml      # GitOps bridge component
├── oran-adaptor-resources.yaml       # O-RAN interface adapter
├── values-resource-management.yaml   # Helm values for resource management
└── hpa-configurations.yaml           # Auto-scaling configurations
```

## Resource Allocation Strategy

### Component Classification

Components are classified based on their resource requirements and criticality:

| Component | Classification | CPU Limit | Memory Limit | Scaling Strategy |
|-----------|---------------|-----------|--------------|------------------|
| Main Operator | Control Plane | 500m | 512Mi | Static (1 replica) |
| LLM Processor | AI Workload | 1000m | 1Gi | Dynamic (2-10 replicas) |
| RAG API | Data Service | 500m | 512Mi | Dynamic (2-8 replicas) |
| Webhook | Critical Service | 200m | 256Mi | Conservative (2-5 replicas) |
| Nephio Bridge | GitOps | 300m | 384Mi | Static (1 replica) |
| O-RAN Adaptor | Telecom Interface | 400m | 512Mi | Dynamic (2-6 replicas) |

### Resource Management Principles

1. **Prevent Resource Exhaustion**: All containers have resource limits to prevent resource starvation
2. **Enable Proper Scheduling**: Resource requests ensure proper node placement
3. **Support Auto-scaling**: HPA configurations with multiple metrics for intelligent scaling
4. **Follow Best Practices**: Security contexts, health checks, and disruption budgets

## Deployment Guide

### 1. Basic Deployment

Deploy all components with default resource limits:

```bash
# Apply main operator
kubectl apply -f config/manager/manager.yaml

# Apply component resources
kubectl apply -f config/components/llm-processor-resources.yaml
kubectl apply -f config/components/rag-api-resources.yaml
kubectl apply -f config/components/webhook-resources.yaml
kubectl apply -f config/components/nephio-bridge-resources.yaml
kubectl apply -f config/components/oran-adaptor-resources.yaml

# Apply auto-scaling configurations
kubectl apply -f config/components/hpa-configurations.yaml
```

### 2. Helm Deployment with Resource Management

Use the enhanced Helm values for comprehensive resource management:

```bash
# Install with resource management configurations
helm upgrade --install nephoran-intent-operator \
  deployments/helm/nephoran-operator \
  -f config/components/values-resource-management.yaml \
  --namespace nephoran-system \
  --create-namespace
```

### 3. Environment-Specific Deployments

#### Development Environment
```bash
helm upgrade --install nephoran-intent-operator \
  deployments/helm/nephoran-operator \
  -f config/components/values-resource-management.yaml \
  --set global.resourceManagement.strategy=minimal \
  --set llmProcessor.resources.limits.memory=512Mi \
  --set llmProcessor.autoscaling.maxReplicas=3
```

#### Production Environment
```bash
helm upgrade --install nephoran-intent-operator \
  deployments/helm/nephoran-operator \
  -f config/components/values-resource-management.yaml \
  --set global.resourceManagement.strategy=performance \
  --set llmProcessor.resources.limits.memory=2Gi \
  --set llmProcessor.autoscaling.maxReplicas=15 \
  --set resourceManagement.verticalPodAutoscaler.enabled=true
```

## Auto-scaling Configuration

### HPA Metrics

The system uses multiple metrics for intelligent scaling:

1. **Resource Metrics**: CPU and memory utilization
2. **Custom Metrics**: Component-specific metrics (queue depth, connection count)
3. **External Metrics**: Prometheus metrics and external service indicators

### Scaling Policies

- **Scale Up**: Aggressive scaling for AI workloads, conservative for critical services
- **Scale Down**: Conservative with longer stabilization windows
- **Minimum Replicas**: Ensures high availability with at least 2 replicas for most services

### KEDA Integration

Advanced scaling with KEDA for external metrics:

```yaml
# Example KEDA ScaledObject
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: llm-processor-keda-scaler
spec:
  triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus:9090
      metricName: llm_queue_depth
      threshold: '5'
```

## Resource Optimization

### Cost Optimization Features

1. **Spot Instance Utilization**: Configurable spot instance preferences
2. **Right-sizing Recommendations**: Analysis-based resource adjustments
3. **Idle Resource Detection**: Identification of underutilized resources
4. **Scheduling Windows**: Time-based replica adjustments

### Performance Optimization

1. **Node Affinity**: Preferred node types for optimal performance
2. **Pod Anti-Affinity**: Distribution across nodes for high availability
3. **Resource Requests**: Proper scheduling based on actual needs
4. **Health Checks**: Proactive detection of resource-related issues

## Monitoring and Alerting

### Key Metrics

- **Resource Utilization**: CPU, memory, storage usage per component
- **Scaling Events**: HPA scaling decisions and frequency
- **Performance**: Response times and throughput per component
- **Cost**: Resource costs and optimization opportunities

### Alerts

- **HighCPUUsage**: >80% CPU utilization for 5 minutes
- **HighMemoryUsage**: >90% memory utilization for 5 minutes
- **PodRestartingFrequently**: Restart rate >0.1 per 15 minutes
- **ScalingFailure**: HPA unable to scale for 10 minutes

## Troubleshooting

### Common Issues

#### 1. Pod Not Scheduled
```bash
# Check resource requests vs available capacity
kubectl describe node

# Check pending pods
kubectl get pods -o wide | grep Pending
kubectl describe pod <pod-name>
```

#### 2. HPA Not Scaling
```bash
# Check HPA status
kubectl get hpa
kubectl describe hpa <hpa-name>

# Verify metrics availability
kubectl top pods
kubectl get --raw "/apis/metrics.k8s.io/v1beta1/pods"
```

#### 3. Resource Exhaustion
```bash
# Check resource usage
kubectl top pods --sort-by=cpu
kubectl top pods --sort-by=memory

# Check resource quotas
kubectl get resourcequota
kubectl describe resourcequota
```

### Performance Tuning

#### CPU-Intensive Workloads (LLM Processor)
- Increase CPU limits and requests
- Use CPU-optimized node types
- Enable aggressive HPA scaling
- Consider vertical scaling with VPA

#### Memory-Intensive Workloads (Weaviate)
- Increase memory limits
- Use memory-optimized instances
- Enable persistent storage
- Monitor for memory leaks

#### I/O-Intensive Workloads (RAG API)
- Use SSD storage classes
- Optimize connection pooling
- Enable caching strategies
- Monitor disk I/O metrics

## Security Considerations

### Pod Security Context
All pods run with:
- Non-root user (65532)
- Read-only root filesystem
- No privilege escalation
- Dropped capabilities

### Network Policies
- Ingress and egress traffic control
- Component-to-component communication rules
- External service access restrictions

### Resource Quotas
- Namespace-level resource limits
- Prevents resource exhaustion attacks
- Cost control and governance

## Best Practices

1. **Start Conservative**: Begin with lower resource allocations and scale up based on monitoring
2. **Monitor Continuously**: Use comprehensive monitoring to understand actual resource usage
3. **Test Scaling**: Regularly test auto-scaling behavior under load
4. **Review and Adjust**: Periodically review and optimize resource allocations
5. **Document Changes**: Keep track of resource configuration changes and their impact

## Advanced Configuration

### Custom Metrics
Define custom metrics for application-specific scaling:

```yaml
- type: Pods
  pods:
    metric:
      name: llm_processing_queue_depth
    target:
      type: AverageValue
      averageValue: "5"
```

### Multi-Dimensional Scaling
Combine multiple metrics for intelligent scaling decisions:

```yaml
behavior:
  scaleUp:
    policies:
    - type: Percent
      value: 100
      periodSeconds: 15
    - type: Pods
      value: 2
      periodSeconds: 60
    selectPolicy: Max
```

### Cost-Performance Balance
Configure cost optimization while maintaining performance:

```yaml
costOptimization:
  enabled: true
  spotInstances:
    enabled: true
    allowedComponents: ["llm-processor", "rag-api"]
  rightsizing:
    enabled: true
    analysisWindow: "7d"
```

## Support and Troubleshooting

For issues with resource management:

1. Check the monitoring dashboards for resource utilization trends
2. Review HPA events and scaling decisions
3. Analyze pod resource consumption patterns
4. Consult the troubleshooting section above
5. Check Kubernetes events for resource-related issues

For more information, refer to:
- [Kubernetes Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- [HPA Documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [KEDA Scaling](https://keda.sh/docs/)
- [Nephoran Intent Operator Documentation](../docs/README.md)