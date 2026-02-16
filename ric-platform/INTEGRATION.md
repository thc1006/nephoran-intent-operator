# RIC Platform Integration Guide

## Overview

This guide explains how the Nephoran Intent Operator integrates with O-RAN Near-RT RIC Platform components using Git submodules, following O-RAN Software Community (O-RAN SC) best practices.

## Why Git Submodules?

Based on O-RAN SC's architecture ([ric-dep documentation](https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-ric-dep/en/latest/overview.html)), we use Git submodules because:

1. **Independent Development**: Each RIC component can be developed, tested, and released independently
2. **Version Control**: Pin specific versions of each component
3. **Modularity**: Deploy only needed components
4. **Upstream Tracking**: Easily sync with upstream O-RAN SC repositories
5. **Clear Boundaries**: Each component has its own repository and ownership

## Architecture Alignment

Our implementation follows O-RAN SC's three-tier architecture:

### Tier 1: Infrastructure (ricinfra)
- Database services
- Message routing (RMR)
- Service discovery

### Tier 2: Platform (ricplt)
- **submgr**: Manages E2 subscriptions from xApps
- **rtmgr**: Provides dynamic routing tables
- **e2mgr**: Manages E2 connections with gNB/eNB
- **e2**: Terminates E2AP protocol
- **appmgr**: Manages xApp lifecycle

### Tier 3: Applications (ricxapp)
- xApps deployed and managed by appmgr

## Integration with Nephoran Intent Operator

### Intent Flow

```
User Intent (NL)
    ↓
Nephoran LLM Processing
    ↓
NetworkIntent CRD
    ↓
Intent Controller
    ↓
RIC Platform API (via appmgr)
    ↓
xApp Deployment/Configuration
```

### Component Integration Points

#### 1. E2 Manager Integration

**Purpose**: Manage E2 connections to RAN nodes

**Integration**:
```go
// pkg/oran/e2/client.go
type E2Client struct {
    e2mgrURL string
    httpClient *http.Client
}

func (c *E2Client) GetNodeBInfo() (*NodeInfo, error) {
    // Call e2mgr API to get E2 node information
}
```

**Usage in Intent Controller**:
- Query available RAN nodes
- Verify E2 connection status before deploying policies

#### 2. Subscription Manager Integration

**Purpose**: Manage E2 subscriptions for monitoring/control

**Integration**:
```go
// pkg/oran/subscriptions/client.go
type SubscriptionClient struct {
    submgrURL string
}

func (c *SubscriptionClient) CreateSubscription(req SubscriptionRequest) error {
    // Create E2 subscription via submgr
}
```

**Usage in Intent Controller**:
- Create subscriptions for network monitoring
- Update subscription parameters based on intents

#### 3. Application Manager Integration

**Purpose**: Deploy and manage xApps

**Integration**:
```go
// pkg/oran/appmgr/client.go
type AppmgrClient struct {
    appmgrURL string
}

func (c *AppmgrClient) DeployXApp(name, helmChart string) error {
    // Deploy xApp via appmgr
}
```

**Usage in Intent Controller**:
- Deploy xApps based on network intents
- Scale xApp instances
- Configure xApp parameters

## Deployment Integration

### Helm Integration

The Nephoran Helm chart includes RIC Platform as a dependency:

```yaml
# charts/nephoran/Chart.yaml
dependencies:
  - name: ric-platform
    version: "1.0.0"
    repository: "file://../ric-platform/helm/ric-platform"
    condition: ric.enabled
```

### Kubernetes Operators

Both systems work together:

```yaml
# Nephoran manages high-level intents
apiVersion: nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: scale-amf
spec:
  action: scale
  target: amf
  replicas: 5

# RIC Platform manages low-level RAN policies
apiVersion: ric.o-ran-sc.org/v1
kind: XApp
metadata:
  name: kpm-monitor
spec:
  namespace: ricxapp
  helm:
    chart: kpm-monitor
    version: 1.0.0
```

## CI/CD Integration

### Submodule Update Workflow

```yaml
# .github/workflows/update-ric-submodules.yml
name: Update RIC Submodules
on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive

      - name: Update submodules
        run: |
          git submodule update --remote --merge
          git add ric-platform/submodules
          git commit -m "chore: update RIC submodules"
          git push
```

### Testing with Submodules

```bash
# Integration tests that span both systems
make test-integration-ric
```

## Development Workflow

### Scenario: Update Subscription Manager

1. **Update submodule**:
```bash
cd ric-platform/submodules/submgr
git checkout -b feature/new-api
# Make changes...
git commit -m "feat: add new subscription API"
git push origin feature/new-api
```

2. **Update Nephoran integration**:
```bash
cd pkg/oran/subscriptions
# Update client.go to use new API
```

3. **Update parent repo**:
```bash
cd /home/thc1006/dev/nephoran-intent-operator
git add ric-platform/submodules/submgr
git add pkg/oran/subscriptions/client.go
git commit -m "feat: integrate new submgr subscription API"
```

### Scenario: Debug RIC Component

```bash
# Enter submodule
cd ric-platform/submodules/e2mgr

# Build and run locally
go build ./cmd/e2mgr
./e2mgr --config config/e2mgr-config.yaml

# Or use Docker
docker build -t e2mgr-debug .
docker run -p 3800:3800 e2mgr-debug
```

## Monitoring and Observability

### Metrics Collection

Both systems export Prometheus metrics:

```yaml
# ServiceMonitor for Nephoran
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nephoran-metrics
spec:
  selector:
    matchLabels:
      app: nephoran-controller

---
# ServiceMonitor for RIC components
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: ric-platform-metrics
  namespace: ricplt
spec:
  selector:
    matchLabels:
      ric-component: "true"
```

### Distributed Tracing

Both systems use OpenTelemetry:

```go
// Trace context propagated across Nephoran → RIC
ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
span := tracer.Start(ctx, "DeployXApp")
defer span.End()
```

## Security Considerations

### mTLS Between Systems

```yaml
# Certificate management
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: nephoran-ric-client
spec:
  secretName: nephoran-ric-tls
  issuerRef:
    name: ca-issuer
  dnsNames:
    - nephoran-controller.nephoran-system.svc
```

### RBAC

```yaml
# Nephoran ServiceAccount needs access to RIC resources
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-ric-access
rules:
  - apiGroups: ["ric.o-ran-sc.org"]
    resources: ["xapps", "subscriptions"]
    verbs: ["get", "list", "create", "update", "delete"]
```

## Troubleshooting

### Submodule Issues

```bash
# Submodule not initialized
git submodule update --init --recursive

# Submodule detached HEAD
cd ric-platform/submodules/submgr
git checkout master
git pull

# Reset submodule to committed version
git submodule update --init --force
```

### Integration Issues

```bash
# Check RIC API connectivity
kubectl port-forward -n ricplt svc/e2mgr 3800:3800
curl http://localhost:3800/v1/nodeb/states

# Check logs
kubectl logs -n ricplt -l app=submgr -f
kubectl logs -n nephoran-system -l app=nephoran-controller -f
```

## Best Practices

1. **Version Pinning**: Always pin submodule versions in production
2. **Testing**: Test integration with specific submodule versions
3. **Documentation**: Document which submodule versions are compatible
4. **Updates**: Review submodule updates carefully before merging
5. **Rollback**: Keep previous working submodule commits for rollback

## References

- [O-RAN SC RIC Platform](https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-ric-dep/en/latest/)
- [Git Submodules Documentation](https://git-scm.com/book/en/v2/Git-Tools-Submodules)
- [O-RAN Architecture](https://www.o-ran.org/specifications)
- [Near-RT RIC Deployment Guide](https://lf-o-ran-sc.atlassian.net/wiki/spaces/IAT/pages/218464281/Near+RealTime+RIC+Deployment+Guideline)
