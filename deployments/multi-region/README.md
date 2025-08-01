# Nephoran Intent Operator - Multi-Region Deployment

## Phase 4 Enterprise Architecture - Global Deployment Capabilities

This directory contains the complete multi-region deployment configuration for the Nephoran Intent Operator, enabling worldwide deployment with intelligent traffic routing, failover capabilities, and global state synchronization.

## 🌍 Architecture Overview

The multi-region architecture consists of:

- **Global Control Plane**: Centralized coordination and state management
- **Regional Data Planes**: Distributed processing and execution
- **Intelligent Traffic Routing**: AI-driven traffic distribution
- **Cross-Region Synchronization**: Real-time state consistency
- **Global Monitoring**: Unified observability across all regions

## 📁 Directory Structure

```
deployments/multi-region/
├── multi-region-architecture.yaml    # Global control plane configuration
├── region-deployment-template.yaml   # Regional deployment template
├── global-state-sync.yaml           # Cross-region synchronization
├── monitoring-dashboard.yaml        # Multi-region observability
├── README.md                        # This file
└── scripts/
    └── deploy-multi-region.sh       # Deployment automation
```

## 🚀 Quick Start

### Prerequisites

1. **Multiple Kubernetes Clusters**: At least 2 clusters in different regions
2. **kubectl Contexts**: Named contexts for each region
3. **Istio Service Mesh**: Deployed in all clusters
4. **Cross-Cluster Networking**: Network connectivity between regions
5. **Shared Storage**: Global state store (CockroachDB)

### Deployment Commands

```bash
# Deploy global control plane
./scripts/deploy-multi-region.sh global

# Deploy all regions
./scripts/deploy-multi-region.sh regions

# Deploy specific region
./scripts/deploy-multi-region.sh us-east-1

# Run complete deployment
./scripts/deploy-multi-region.sh all

# Test multi-region functionality
./scripts/deploy-multi-region.sh test
```

## 🌐 Supported Regions

| Region | Name | Priority | Features |
|--------|------|----------|----------|
| **us-east-1** | US East (Primary) | 1 | Primary control plane, ML training |
| **eu-west-1** | EU West | 2 | GDPR compliant, regional ML inference |
| **ap-southeast-1** | Asia Pacific | 3 | Low-latency O-RAN, edge computing |
| **us-west-2** | US West (DR) | 4 | Disaster recovery, backup control plane |

## 🔧 Configuration

### Regional Template Variables

The deployment uses environment variable substitution:

```bash
# Regional settings
REGION=us-east-1
REGION_NAME="US East (Primary)"
VERSION=latest
REGISTRY=docker.io

# Capacity settings
MAX_CONCURRENT_INTENTS=100
MAX_DEPLOYMENTS_PER_MINUTE=50
MAX_LLM_RPS=10

# Regional features
EDGE_ENABLED=false
LOCAL_ML_ENABLED=true
DATA_RESIDENCY=false
COMPLIANCE_MODE=standard

# Resource allocation
LLM_REPLICAS=3
WEAVIATE_REPLICAS=3
WEAVIATE_STORAGE_SIZE=100Gi
```

### Traffic Routing Strategies

**Available Strategies:**
- `latency`: Route to lowest latency regions
- `capacity`: Route based on available capacity
- `cost`: Route to most cost-effective regions
- `hybrid`: Balanced approach (default)

## 🛡️ Security & Compliance

### Data Residency

- **EU Compliance**: Data processed in `eu-west-1` stays within EU
- **Regional Isolation**: Tenant data can be constrained to specific regions
- **Cross-Border Controls**: Configurable data transfer restrictions

### Security Features

- **mTLS Everywhere**: Cross-region communication secured with mutual TLS
- **Regional Authentication**: Independent OAuth2 providers per region
- **Network Segmentation**: Isolated network policies per region
- **Secret Management**: Regional secret isolation with global sync

## 📊 Monitoring & Observability

### Global Dashboards

Access the multi-region dashboard:
```bash
kubectl port-forward svc/grafana 3000:3000 -n nephoran-global
# Open http://localhost:3000
```

### Key Metrics

- **Global Request Rate**: Total requests across all regions
- **Regional Distribution**: Traffic percentage per region
- **Cross-Region Latency**: Inter-region communication metrics
- **Failover Events**: Automatic failover tracking
- **Cost Analysis**: Per-region cost breakdown

### Alert Rules

- **Global SLA Violations**: Success rate < 95%
- **Regional Outages**: Region health degradation
- **Traffic Imbalance**: Uneven traffic distribution
- **High Failover Rate**: Excessive cross-region failovers

## 🔄 Traffic Management

### Intelligent Routing

The traffic controller makes routing decisions based on:

1. **Region Health**: Success rate, latency, error rate
2. **Capacity Utilization**: CPU, memory, network usage
3. **Geographic Proximity**: User location to region distance
4. **Cost Optimization**: Regional pricing differences

### Failover Scenarios

**Automatic Failover Triggers:**
- Region success rate < 95%
- P95 latency > 5 seconds
- Resource utilization > 90%
- Network connectivity issues

**Failover Process:**
1. Traffic controller detects unhealthy region
2. Gradually shifts traffic to healthy regions
3. Monitors failed region for recovery
4. Automatically restores traffic when healthy

## 🗄️ Global State Management

### CockroachDB Global Store

The global state store maintains:
- **Intent Synchronization**: Cross-region intent state
- **Configuration Sync**: Global configuration consistency
- **Metrics Aggregation**: Regional metrics rollup
- **Deployment Tracking**: Multi-region deployment status

### Synchronization Process

```bash
# State sync runs every 5 minutes
kubectl get cronjob global-state-synchronizer -n nephoran-global

# Check sync status
kubectl logs -l app=state-synchronizer -n nephoran-global
```

## 🧪 Testing Multi-Region Setup

### Health Check Tests

```bash
# Test regional health endpoints
for region in us-east-1 eu-west-1 ap-southeast-1; do
  curl -s https://$region.nephoran.io/health
done

# Test cross-region connectivity
kubectl exec -n nephoran-system deployment/llm-processor -- \
  curl -s https://eu-west-1.nephoran.io/health
```

### Intent Distribution Test

```bash
# Create test intent with multi-region annotation
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: multi-region-test
  namespace: nephoran-system
  annotations:
    nephoran.io/multi-region: "true"
    nephoran.io/regions: "us-east-1,eu-west-1"
spec:
  description: "Deploy global AMF with regional distribution"
  priority: high
EOF

# Monitor processing across regions
kubectl get networkintent multi-region-test -o wide
```

### Failover Testing

```bash
# Simulate region failure
kubectl scale deployment llm-processor-regional --replicas=0 -n nephoran-system

# Monitor traffic redistribution
watch kubectl get virtualservices global-traffic-routing -o yaml
```

## 🔍 Troubleshooting

### Common Issues

**1. Cross-Region Connectivity**
```bash
# Check Istio multi-cluster setup
istioctl proxy-status
istioctl analyze -A

# Verify remote secrets
kubectl get secrets -l istio/multiCluster=remote -A
```

**2. State Synchronization Issues**
```bash
# Check CockroachDB cluster status
kubectl exec cockroachdb-0 -n nephoran-global -- \
  /cockroach/cockroach node status --certs-dir=/cockroach/cockroach-certs

# Check sync job logs
kubectl logs job/global-state-synchronizer-xxx -n nephoran-global
```

**3. Traffic Routing Problems**
```bash
# Check traffic controller logs
kubectl logs deployment/traffic-controller -n nephoran-global

# Verify routing decisions
kubectl get virtualservices -A
istioctl proxy-config routes <pod-name>
```

### Debug Commands

```bash
# Global cluster status
kubectl get pods -n nephoran-global

# Regional status check
for context in us-east-1 eu-west-1 ap-southeast-1; do
  kubectl --context=$context get pods -n nephoran-system
done

# Cross-region network test
kubectl exec -n nephoran-system deployment/llm-processor -- \
  nslookup eu-west-1.nephoran.local

# Metrics verification
curl -s http://prometheus:9090/api/v1/query?query=nephoran:global:request_rate
```

## 📋 Operational Procedures

### Daily Operations

1. **Health Check**: Monitor regional health status
2. **Traffic Review**: Analyze traffic distribution patterns
3. **Cost Analysis**: Review per-region cost metrics
4. **Capacity Planning**: Monitor resource utilization trends

### Weekly Maintenance

1. **Failover Testing**: Test automatic failover mechanisms
2. **Performance Tuning**: Optimize traffic routing parameters
3. **Security Review**: Audit cross-region access patterns
4. **Backup Validation**: Verify global state backups

### Emergency Procedures

**Total Region Failure:**
1. Verify other regions are healthy
2. Update traffic routing to exclude failed region
3. Scale remaining regions to handle additional load
4. Monitor global SLA metrics
5. Coordinate region recovery efforts

**Cross-Region Network Partition:**
1. Identify affected regions
2. Enable autonomous mode for isolated regions
3. Queue state synchronization for later
4. Monitor regional health independently
5. Resume synchronization when connectivity restored

## 📚 Additional Resources

- [Multi-Region Architecture Guide](../../docs/multi-region-architecture.md)
- [Traffic Controller API Reference](../../docs/api/traffic-controller.md)
- [Global State Management Guide](../../docs/global-state.md)
- [Istio Multi-Cluster Setup](https://istio.io/latest/docs/setup/install/multicluster/)
- [CockroachDB Multi-Region](https://www.cockroachlabs.com/docs/stable/multiregion-overview.html)

## 🎯 Performance Targets

- **Global SLA**: 99.9% availability
- **Cross-Region Latency**: < 200ms P95
- **Failover Time**: < 30 seconds
- **State Sync Delay**: < 5 minutes
- **Traffic Rebalancing**: < 60 seconds