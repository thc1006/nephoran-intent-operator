# NetworkIntent Examples

## Basic Examples

### Simple AMF Deployment

Deploy a basic AMF instance with default settings:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: simple-amf
  namespace: nephoran-system
spec:
  intent: "Deploy an AMF for testing with basic configuration"
  intentType: deployment
  targetComponents:
    - AMF
```

### Scale UPF Deployment

Scale an existing UPF deployment:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: scale-upf
  namespace: nephoran-system
spec:
  intent: "Scale UPF to 5 replicas to handle increased traffic"
  intentType: scaling
  priority: high
  targetComponents:
    - UPF
```

## Production Deployments

### High Availability 5G Core

Deploy a complete 5G core with high availability:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: prod-5g-core-ha
  namespace: nephoran-system
  labels:
    environment: production
    tier: core
  annotations:
    deployment.nephoran.io/team: "network-ops"
    deployment.nephoran.io/cost-center: "telecom-5g"
spec:
  intent: |
    Deploy a production-ready 5G core network with high availability.
    Include AMF with 3 replicas, SMF with 2 replicas, and UPF with auto-scaling.
    Configure for 100,000 subscribers with 99.99% availability target.
    Enable monitoring and alerting for all components.
  intentType: deployment
  priority: high
  targetComponents:
    - AMF
    - SMF
    - UPF
    - NRF
    - AUSF
    - UDM
    - PCF
  resourceConstraints:
    cpu: "16"
    memory: "32Gi"
    storage: "100Gi"
    maxCpu: "32"
    maxMemory: "64Gi"
  targetNamespace: "5g-core-prod"
  targetCluster: "central-prod-cluster"
  region: "us-west-2"
  timeoutSeconds: 1800
  maxRetries: 5
```

### Edge 5G Deployment

Deploy 5G core for edge computing:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: edge-5g-deployment
  namespace: nephoran-system
  labels:
    deployment-type: edge
    site: edge-site-01
spec:
  intent: |
    Deploy lightweight 5G core for edge site with minimal footprint.
    Optimize for low latency with local breakout.
    Include only essential functions: AMF, SMF, and UPF.
    Configure for 1000 IoT devices with URLLC requirements.
  intentType: deployment
  priority: high
  targetComponents:
    - AMF
    - SMF
    - UPF
  resourceConstraints:
    cpu: "4"
    memory: "8Gi"
    storage: "20Gi"
    maxCpu: "8"
    maxMemory: "16Gi"
  targetNamespace: "edge-5g-core"
  targetCluster: "edge-cluster-01"
  region: "edge-west-1"
```

## Network Slicing Examples

### eMBB Slice Creation

Create an enhanced Mobile Broadband slice:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: embb-slice
  namespace: nephoran-system
  labels:
    slice-type: embb
    customer: "enterprise-a"
spec:
  intent: |
    Create eMBB network slice for video streaming service.
    Guarantee 100 Mbps downlink and 50 Mbps uplink.
    Support up to 10,000 concurrent users.
    Enable content caching and CDN integration.
  intentType: deployment
  priority: high
  targetComponents:
    - AMF
    - SMF
    - UPF
    - PCF
  networkSlice: "000001-000100"
  resourceConstraints:
    cpu: "8"
    memory: "16Gi"
    storage: "50Gi"
  targetNamespace: "slice-embb"
  processedParameters:
    performanceRequirements:
      downlinkThroughput: "100Mbps"
      uplinkThroughput: "50Mbps"
      maxLatency: "20ms"
      packetLossRate: "0.001"
```

### URLLC Slice for Industrial IoT

Create Ultra-Reliable Low-Latency slice:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: urllc-industrial-slice
  namespace: nephoran-system
  labels:
    slice-type: urllc
    industry: manufacturing
spec:
  intent: |
    Deploy URLLC network slice for industrial automation.
    Guarantee 1ms end-to-end latency with 99.999% reliability.
    Support 5000 industrial IoT sensors and actuators.
    Enable edge computing with MEC integration.
  intentType: deployment
  priority: critical
  targetComponents:
    - AMF
    - SMF
    - UPF
  networkSlice: "000002-001000"
  resourceConstraints:
    cpu: "16"
    memory: "32Gi"
    storage: "20Gi"
    maxCpu: "32"
    maxMemory: "64Gi"
  targetNamespace: "slice-urllc"
  targetCluster: "edge-cluster"
  region: "factory-floor-1"
```

### mMTC Slice for Smart City

Create massive Machine-Type Communications slice:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: mmtc-smart-city
  namespace: nephoran-system
  labels:
    slice-type: mmtc
    use-case: smart-city
spec:
  intent: |
    Create mMTC slice for smart city IoT devices.
    Support 1 million low-power sensors.
    Optimize for battery life with extended DRX cycles.
    Enable bulk data collection with relaxed latency.
  intentType: deployment
  priority: medium
  targetComponents:
    - AMF
    - SMF
    - UPF
    - NSSF
  networkSlice: "000003-002000"
  resourceConstraints:
    cpu: "4"
    memory: "8Gi"
    storage: "100Gi"
  targetNamespace: "slice-mmtc"
```

## O-RAN Deployments

### Near-RT RIC with xApps

Deploy Near-RT RIC with xApp support:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: near-rt-ric-deployment
  namespace: nephoran-system
  labels:
    component: ric
    ric-type: near-rt
spec:
  intent: |
    Deploy Near-RT RIC with traffic steering and admission control xApps.
    Configure E2 interface for O-DU and O-CU connections.
    Enable A1 interface for policy management from Non-RT RIC.
    Set up monitoring and KPI collection.
  intentType: deployment
  priority: high
  targetComponents:
    - Near-RT-RIC
    - xApp
  resourceConstraints:
    cpu: "8"
    memory: "16Gi"
    storage: "50Gi"
  targetNamespace: "oran-ric"
  targetCluster: "ran-cluster"
```

### O-RAN Distributed Unit

Deploy O-DU with specific configuration:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: o-du-deployment
  namespace: nephoran-system
  labels:
    component: ran
    ran-type: o-du
spec:
  intent: |
    Deploy O-DU for cell site with 100 MHz bandwidth.
    Configure for 256 QAM downlink and 64 QAM uplink.
    Enable carrier aggregation with 3 component carriers.
    Set up F1 interface to O-CU and E2 interface to Near-RT RIC.
  intentType: deployment
  priority: high
  targetComponents:
    - O-DU
  resourceConstraints:
    cpu: "16"
    memory: "32Gi"
    storage: "20Gi"
  targetNamespace: "oran-du"
  targetCluster: "cell-site-cluster"
  region: "cell-site-001"
```

## Scaling Operations

### Auto-scaling Configuration

Configure auto-scaling for network function:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: configure-autoscaling
  namespace: nephoran-system
spec:
  intent: |
    Configure horizontal auto-scaling for SMF.
    Scale between 2 and 10 replicas based on CPU utilization.
    Target 70% CPU utilization with 30 second stabilization.
    Enable predictive scaling based on historical patterns.
  intentType: scaling
  priority: medium
  targetComponents:
    - SMF
  processedParameters:
    scalingPolicy:
      minReplicas: 2
      maxReplicas: 10
      targetCPUUtilization: 70
      scaleUpStabilization: "30s"
      scaleDownStabilization: "300s"
      predictiveScaling: true
```

### Vertical Scaling

Increase resources for existing deployment:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: vertical-scale-upf
  namespace: nephoran-system
spec:
  intent: |
    Vertically scale UPF to handle increased throughput.
    Increase CPU to 16 cores and memory to 64GB.
    Maintain current number of replicas.
  intentType: scaling
  priority: high
  targetComponents:
    - UPF
  resourceConstraints:
    cpu: "16"
    memory: "64Gi"
    maxCpu: "32"
    maxMemory: "128Gi"
```

## Optimization Examples

### Latency Optimization

Optimize network for low latency:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: optimize-latency
  namespace: nephoran-system
spec:
  intent: |
    Optimize network slice for ultra-low latency.
    Target sub-1ms processing latency for user plane.
    Enable DPDK acceleration and SR-IOV.
    Configure CPU pinning and NUMA awareness.
  intentType: optimization
  priority: high
  targetComponents:
    - UPF
    - O-DU
  networkSlice: "000002-000001"
  processedParameters:
    performanceRequirements:
      maxLatency: "1ms"
      dpdk: true
      sriov: true
      cpuPinning: true
      numaAware: true
```

### Resource Optimization

Optimize resource utilization:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: optimize-resources
  namespace: nephoran-system
spec:
  intent: |
    Optimize resource utilization across 5G core.
    Reduce overall resource consumption by 30%.
    Consolidate underutilized pods.
    Enable resource sharing where possible.
  intentType: optimization
  priority: medium
  targetComponents:
    - AMF
    - SMF
    - UPF
    - PCF
  processedParameters:
    performanceRequirements:
      resourceReduction: "30%"
      consolidation: true
      resourceSharing: true
```

## Maintenance Operations

### Rolling Update

Perform rolling update of network functions:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: rolling-update-5g-core
  namespace: nephoran-system
spec:
  intent: |
    Perform rolling update of 5G core to version 2.5.0.
    Maintain service availability during update.
    Update one replica at a time with 60 second delay.
    Perform health checks before proceeding.
  intentType: maintenance
  priority: high
  targetComponents:
    - AMF
    - SMF
    - UPF
  processedParameters:
    deploymentConfig:
      version: "2.5.0"
      strategy: "RollingUpdate"
      maxUnavailable: 1
      maxSurge: 1
      minReadySeconds: 60
      progressDeadlineSeconds: 600
```

### Security Patching

Apply critical security patches:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: security-patch-critical
  namespace: nephoran-system
  labels:
    maintenance-type: security
    severity: critical
spec:
  intent: |
    Apply critical security patch CVE-2024-12345 to all components.
    Prioritize control plane functions.
    Minimize downtime with blue-green deployment.
    Verify patch application with security scan.
  intentType: maintenance
  priority: critical
  targetComponents:
    - AMF
    - SMF
    - AUSF
    - UDM
    - PCF
  processedParameters:
    securityPolicy:
      patchId: "CVE-2024-12345"
      strategy: "BlueGreen"
      verifyPatch: true
      securityScan: true
```

## Multi-Region Deployments

### Geo-Distributed 5G Core

Deploy across multiple regions:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: multi-region-5g
  namespace: nephoran-system
  labels:
    deployment: multi-region
spec:
  intent: |
    Deploy geo-distributed 5G core across three regions.
    Configure active-active setup with regional failover.
    Ensure data consistency with multi-master replication.
    Set up global load balancing with GeoDNS.
  intentType: deployment
  priority: high
  targetComponents:
    - AMF
    - SMF
    - UPF
    - UDR
  resourceConstraints:
    cpu: "32"
    memory: "64Gi"
    storage: "200Gi"
  parametersMap:
    regions: "us-west-2,eu-central-1,ap-southeast-1"
    replication: "multi-master"
    consistency: "eventual"
    loadBalancing: "geodns"
```

## Complex Scenarios

### Complete MEC Platform

Deploy Multi-access Edge Computing platform:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: mec-platform
  namespace: nephoran-system
  labels:
    platform: mec
    edge-site: "site-001"
spec:
  intent: |
    Deploy complete MEC platform at edge site.
    Include local 5G core (AMF, SMF, UPF) with MEC host.
    Configure local breakout for edge applications.
    Enable application mobility and state synchronization.
    Set up MEC service registry and discovery.
    Configure northbound APIs for third-party applications.
  intentType: deployment
  priority: high
  targetComponents:
    - AMF
    - SMF
    - UPF
    - NEF
  resourceConstraints:
    cpu: "32"
    memory: "64Gi"
    storage: "500Gi"
  targetNamespace: "mec-platform"
  targetCluster: "edge-mec-cluster"
  region: "edge-site-001"
  processedParameters:
    deploymentConfig:
      mecEnabled: true
      localBreakout: true
      appMobility: true
      serviceRegistry: true
      northboundAPI: true
```

### Network Function Testing

Deploy test environment with traffic generation:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: nf-test-environment
  namespace: nephoran-system
  labels:
    environment: test
    purpose: validation
spec:
  intent: |
    Deploy test environment for 5G core validation.
    Include traffic generators and test UEs.
    Configure for functional and performance testing.
    Enable packet capture and analysis.
    Set up test automation framework.
  intentType: deployment
  priority: low
  targetComponents:
    - AMF
    - SMF
    - UPF
  resourceConstraints:
    cpu: "8"
    memory: "16Gi"
    storage: "50Gi"
  targetNamespace: "test-environment"
  parametersMap:
    testMode: "true"
    trafficGenerator: "enabled"
    packetCapture: "enabled"
    testAutomation: "enabled"
    testUEs: "1000"
```

## Best Practices

### Using Labels and Annotations

```yaml
metadata:
  labels:
    app.kubernetes.io/name: "5g-core"
    app.kubernetes.io/instance: "production"
    app.kubernetes.io/version: "2.5.0"
    app.kubernetes.io/component: "network-function"
    app.kubernetes.io/part-of: "nephoran"
    app.kubernetes.io/managed-by: "nephoran-operator"
  annotations:
    nephoran.io/contact: "network-ops@example.com"
    nephoran.io/documentation: "https://docs.example.com/5g-core"
    nephoran.io/runbook: "https://runbooks.example.com/5g-core"
    nephoran.io/cost-center: "telecom-infrastructure"
```

### Resource Calculation Guidelines

```yaml
# Development/Test Environment
resourceConstraints:
  cpu: "1"        # 1 core
  memory: "2Gi"   # 2 GiB
  storage: "10Gi" # 10 GiB

# Staging Environment
resourceConstraints:
  cpu: "4"        # 4 cores
  memory: "8Gi"   # 8 GiB
  storage: "50Gi" # 50 GiB

# Production Environment
resourceConstraints:
  cpu: "16"        # 16 cores
  memory: "32Gi"   # 32 GiB
  storage: "100Gi" # 100 GiB
  maxCpu: "32"     # Allow burst to 32 cores
  maxMemory: "64Gi" # Allow burst to 64 GiB
```

### Intent Writing Tips

1. **Be Specific**: Include exact requirements
2. **Use Numbers**: Specify quantities, percentages, thresholds
3. **Include Context**: Mention environment, purpose, constraints
4. **Define Success**: Specify expected outcomes
5. **Consider Dependencies**: Mention related components

## Troubleshooting Common Issues

For detailed troubleshooting, see [Troubleshooting Guide](overview.md).

## Next Steps

- [Status Reference](spec.md) - Understanding status fields
- [Troubleshooting](overview.md) - Debugging failed intents
- [E2NodeSet Examples](../e2nodeset/overview.md) - O-RAN specific examples
- [Disaster Recovery](../../../../runbooks/disaster-recovery.md) - Backup and failover
