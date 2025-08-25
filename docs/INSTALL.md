# Installation Guide

This guide provides step-by-step instructions for installing the Nephoran Intent Operator.

## Prerequisites

- Kubernetes cluster (1.28+ recommended)
- kubectl configured to access your cluster
- Helm 3.x installed (optional, for Helm deployment)
- Go 1.24+ (for building from source)

## Quick Install

### Using Helm (Recommended)

```bash
# Add the Nephoran Helm repository
helm repo add nephoran https://charts.nephoran.io
helm repo update

# Install the operator
helm install nephoran-operator nephoran/nephoran-operator \
  --namespace nephoran-system \
  --create-namespace \
  --values deployments/helm/nephoran-operator/values.yaml
```

### Using kubectl

```bash
# Apply CRDs
kubectl apply -f deployments/crds/

# Create namespace
kubectl create namespace nephoran-system

# Deploy the operator
kubectl apply -f deployments/kubernetes/ -n nephoran-system
```

## Production Installation

For production deployments, follow the comprehensive guide:

```bash
# 1. Configure TLS certificates
kubectl apply -f deployments/cert-manager/

# 2. Set up monitoring
kubectl apply -f deployments/monitoring/

# 3. Configure network policies
kubectl apply -f deployments/security/network-policies.yaml

# 4. Deploy with production values
helm install nephoran-operator nephoran/nephoran-operator \
  --namespace nephoran-system \
  --create-namespace \
  --values deployments/helm/nephoran-operator/values-production.yaml
```

## Verification

Verify the installation:

```bash
# Check operator pods
kubectl get pods -n nephoran-system

# Check CRDs
kubectl get crds | grep nephoran

# View operator logs
kubectl logs -n nephoran-system deployment/nephoran-operator
```

## Configuration

### Environment Variables

Configure the operator using environment variables:

```yaml
env:
  - name: LLM_ENDPOINT_URL
    value: "https://api.openai.com/v1/chat/completions"
  - name: GIT_REPO_URL
    value: "https://github.com/your-org/kpt-packages"
  - name: ENABLE_RAG
    value: "true"
```

### Custom Values

Create a custom values file for your environment:

```yaml
# my-values.yaml
nephoranOperator:
  replicaCount: 3
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
```

## Upgrading

To upgrade an existing installation:

```bash
# Using Helm
helm upgrade nephoran-operator nephoran/nephoran-operator \
  --namespace nephoran-system \
  --values deployments/helm/nephoran-operator/values.yaml

# Using kubectl
kubectl apply -f deployments/kubernetes/ -n nephoran-system
```

## Uninstallation

To remove the operator:

```bash
# Using Helm
helm uninstall nephoran-operator -n nephoran-system

# Using kubectl
kubectl delete -f deployments/kubernetes/ -n nephoran-system
kubectl delete -f deployments/crds/
kubectl delete namespace nephoran-system
```

## Troubleshooting

### Common Issues

1. **CRDs not found**: Ensure CRDs are applied before deploying the operator
2. **Image pull errors**: Check image registry credentials
3. **Permission denied**: Verify RBAC configuration

### Getting Help

- Check logs: `kubectl logs -n nephoran-system deployment/nephoran-operator`
- Review events: `kubectl get events -n nephoran-system`
- See troubleshooting guide: [docs/troubleshooting.md](troubleshooting.md)

## Next Steps

- [Configure GitOps integration](GitOps-Package-Generation.md)
- [Set up monitoring](monitoring/README.md)
- [Deploy sample workloads](examples/README.md)