# Nephoran Operator Helm Chart

This Helm chart deploys the Nephoran Intent Operator, an intelligent telecom network intent management system.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installation

### Basic Installation

```bash
helm install nephoran-operator ./deployments/helm/nephoran-operator
```

### With RAG Enabled

```bash
helm install nephoran-operator ./deployments/helm/nephoran-operator \
  --set rag.enabled=true
```

### With ML Enabled

```bash
helm install nephoran-operator ./deployments/helm/nephoran-operator \
  --set ml.enabled=true
```

### With Git Token Secret

```bash
# First create the secret
kubectl create secret generic git-credentials --from-literal=token=your-git-token

# Then install with the secret reference
helm install nephoran-operator ./deployments/helm/nephoran-operator \
  --set git.tokenSecret=git-credentials
```

## Configuration

The following table lists the configurable parameters and their default values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rag.enabled` | Enable RAG functionality | `false` |
| `ml.enabled` | Enable ML functionality | `false` |
| `git.tokenSecret` | Name of secret containing git token | `""` |
| `llmProcessor.enabled` | Enable LLM processor deployment | `true` |
| `llmProcessor.replicaCount` | Number of LLM processor replicas | `1` |
| `llmProcessor.image.repository` | LLM processor image repository | `thc1006/nephoran-llm-processor` |
| `llmProcessor.image.tag` | LLM processor image tag | `latest` |
| `llmProcessor.service.port` | LLM processor service port | `8080` |
| `llmProcessor.resources.limits.cpu` | CPU limit | `500m` |
| `llmProcessor.resources.limits.memory` | Memory limit | `512Mi` |

## Health Checks

The chart includes comprehensive health checks:

- **Liveness Probe**: HTTP GET `/health` on port 8080
  - Failure threshold: 3
  - Period: 10 seconds
  - Initial delay: 30 seconds

- **Readiness Probe**: HTTP GET `/health` on port 8080
  - Failure threshold: 3
  - Period: 10 seconds
  - Initial delay: 5 seconds

## Security

The chart implements security best practices:

- Non-root container execution
- Read-only root filesystem
- Security context with dropped capabilities
- Resource limits and requests

## Uninstallation

```bash
helm uninstall nephoran-operator
```