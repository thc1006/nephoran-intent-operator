# nephoran-operator

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.0.0](https://img.shields.io/badge/AppVersion-1.0.0-informational?style=flat-square)

A Helm chart for Nephoran Intent Operator - an intelligent telecom network intent management system

**Homepage:** <https://github.com/thc1006/nephoran-intent-operator>

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
  - [Add Helm Repository](#add-helm-repository)
  - [Basic Installation](#basic-installation)
  - [Production Installation](#production-installation)
  - [Custom Values](#custom-values)
- [Configuration](#configuration)
  - [Core Components](#core-components)
  - [Feature Flags](#feature-flags)
  - [Security Configuration](#security-configuration)
  - [Monitoring and Observability](#monitoring-and-observability)
- [Upgrading](#upgrading)
- [Uninstallation](#uninstallation)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)

## Overview

The Nephoran Intent Operator is an intelligent telecom network intent management system that transforms natural language intents into deployed network functions. It leverages:

- **LLM/RAG Processing**: Natural language intent interpretation with telecommunications domain knowledge
- **O-RAN Compliance**: Full support for A1, O1, O2, and E2 interfaces
- **Nephio Integration**: GitOps-based package orchestration with R5 compatibility
- **Production-Ready**: Enterprise-grade security, monitoring, and high availability
- **Cloud-Native**: Kubernetes-native with support for multi-cloud deployments

### Key Features

- 🎯 **Intent-Driven Orchestration**: Deploy network functions using natural language
- 🤖 **AI-Powered Processing**: GPT-4o-mini with RAG for domain-specific knowledge
- 📊 **Complete Observability**: Prometheus, Grafana, and Jaeger integration
- 🔒 **Enterprise Security**: OAuth2, mTLS, RBAC, and network policies
- 🚀 **High Performance**: Sub-2s intent processing with 99.95% availability
- 🌐 **Multi-Cloud Support**: AWS, Azure, GCP, and edge deployments

## Prerequisites

- Kubernetes 1.26+ cluster
- Helm 3.8+
- kubectl configured to access your cluster
- (Optional) Prometheus Operator for monitoring
- (Optional) cert-manager for TLS certificate management
- (Optional) External Secrets Operator for secret management

### Required Resources

Minimum cluster resources for basic deployment:
- Nodes: 3 (for HA)
- CPU: 4 cores total
- Memory: 8GB total
- Storage: 20GB (if Weaviate is enabled)

Production deployment recommendations:
- Nodes: 5+ across multiple availability zones
- CPU: 16+ cores total
- Memory: 32GB+ total
- Storage: 100GB+ SSD for vector database

## Installation

### Add Helm Repository

```bash
# Add the Nephoran repository
helm repo add nephoran https://charts.nephoran.io
helm repo update
```

### Basic Installation

For a basic installation with default values:

```bash
# Create namespace
kubectl create namespace nephoran-system

# Install the chart
helm install nephoran-operator nephoran/nephoran-operator \
  --namespace nephoran-system
```

### Production Installation

For production deployments with high availability and full features:

```bash
# Create namespace with labels
kubectl create namespace nephoran-system
kubectl label namespace nephoran-system \
  environment=production \
  nephoran.io/managed=true

# Create secrets (example)
kubectl create secret generic openai-credentials \
  --from-literal=apiKey=<your-openai-api-key> \
  -n nephoran-system

kubectl create secret generic git-credentials \
  --from-literal=token=<your-git-token> \
  -n nephoran-system

# Install with production values
helm install nephoran-operator nephoran/nephoran-operator \
  --namespace nephoran-system \
  --values https://raw.githubusercontent.com/thc1006/nephoran-intent-operator/main/deployments/helm/nephoran-operator/values-production.yaml
```

### Custom Values

Create a custom values file for your specific requirements:

```yaml
# custom-values.yaml
global:
  imageRegistry: "my-registry.example.com/"
  
rag:
  enabled: true
  
ml:
  enabled: true

llmProcessor:
  replicaCount: 3
  providers:
    openai:
      enabled: true
      model: "gpt-4"
      
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    
security:
  mtls:
    enabled: true
  networkPolicies:
    enabled: true
```

Then install with:

```bash
helm install nephoran-operator nephoran/nephoran-operator \
  --namespace nephoran-system \
  --values custom-values.yaml
```

## Configuration

### Core Components

The chart deploys several core components:

1. **Nephoran Operator Controller**: Main Kubernetes controller managing NetworkIntent CRDs
2. **LLM Processor**: Service for natural language processing with LLM integration
3. **RAG API** (optional): Retrieval-Augmented Generation service for domain knowledge
4. **Weaviate** (optional): Vector database for semantic search
5. **Nephio Bridge**: Integration with Nephio package orchestration
6. **O-RAN Adaptor**: Interface implementations for O-RAN compliance

### Feature Flags

Control which features are enabled:

```yaml
# Enable/disable major features
rag:
  enabled: true  # Enable RAG for enhanced intent processing
  
ml:
  enabled: true  # Enable ML optimization features

monitoring:
  enabled: true  # Enable Prometheus monitoring
  
security:
  mtls:
    enabled: true  # Enable mutual TLS
```

### Security Configuration

Configure security features:

```yaml
security:
  # mTLS configuration
  mtls:
    enabled: true
    
  # Pod Security Standards
  podSecurityStandards:
    enforce: "restricted"
    
  # Network Policies
  networkPolicies:
    enabled: true
    allowNamespaces:
      - prometheus-system
      - nephio-system
      
  # Security scanning
  scanning:
    enabled: true
    trivy:
      enabled: true
```

### Monitoring and Observability

Configure monitoring integration:

```yaml
monitoring:
  enabled: true
  
  serviceMonitor:
    enabled: true
    interval: 30s
    scrapeTimeout: 10s
    
  prometheusRule:
    enabled: true
    
  # Grafana dashboards
  dashboards:
    enabled: true
    
  # Distributed tracing
  tracing:
    enabled: true
    jaeger:
      endpoint: "http://jaeger-collector:14268/api/traces"
```

### Environment-Specific Configurations

The chart includes pre-configured values for different environments:

- **Enterprise**: `values-enterprise.yaml` - Optimized for enterprise deployments
- **Telecom Operator**: `values-telecom-operator.yaml` - Carrier-grade configuration
- **Edge Computing**: `values-edge-computing.yaml` - Optimized for edge deployments

Example usage:

```bash
# Deploy for telecom operator environment
helm install nephoran-operator nephoran/nephoran-operator \
  --namespace nephoran-system \
  --values environments/values-telecom-operator.yaml
```

## Upgrading

### Standard Upgrade

```bash
# Update repository
helm repo update nephoran

# Upgrade the release
helm upgrade nephoran-operator nephoran/nephoran-operator \
  --namespace nephoran-system \
  --values custom-values.yaml
```

### Rolling Back

If an upgrade fails:

```bash
# View history
helm history nephoran-operator -n nephoran-system

# Rollback to previous version
helm rollback nephoran-operator -n nephoran-system

# Or rollback to specific revision
helm rollback nephoran-operator 3 -n nephoran-system
```

## Uninstallation

```bash
# Uninstall the release
helm uninstall nephoran-operator -n nephoran-system

# Clean up CRDs (optional - this will delete all NetworkIntent resources)
kubectl delete crd networkintents.nephoran.io
kubectl delete crd e2nodesets.nephoran.io

# Delete namespace
kubectl delete namespace nephoran-system
```

## Architecture

The Nephoran Intent Operator follows a microservices architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                      User Interface Layer                    │
│         (kubectl, REST API, Web UI, Automation Tools)        │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                    LLM/RAG Processing Layer                  │
│      (GPT-4o-mini, Haystack RAG, Weaviate Vector DB)        │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                   Nephio R5 Control Plane                    │
│        (Porch, KRM Functions, ConfigSync, ArgoCD)           │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                  O-RAN Interface Bridge Layer                │
│              (A1, O1-FCAPS, O2-Cloud, E2-RIC)               │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                Network Function Orchestration                │
│        (5G Core NFs, O-RAN NFs, Network Slicing)            │
└─────────────────────────────────────────────────────────────┘
```

## Troubleshooting

### Common Issues

#### 1. LLM Processor Not Starting

Check if the OpenAI API key is configured:

```bash
kubectl get secret openai-credentials -n nephoran-system
kubectl logs -l app.kubernetes.io/component=llm-processor -n nephoran-system
```

#### 2. RAG Service Connection Issues

Verify Weaviate is running:

```bash
kubectl get pods -l app.kubernetes.io/name=weaviate -n nephoran-system
kubectl logs -l app.kubernetes.io/name=weaviate -n nephoran-system
```

#### 3. NetworkIntent Not Processing

Check operator logs:

```bash
kubectl logs -l app.kubernetes.io/component=controller -n nephoran-system
kubectl describe networkintent <intent-name> -n nephoran-system
```

### Debug Mode

Enable debug logging:

```yaml
llmProcessor:
  logLevel: "debug"
  
nephoranOperator:
  logLevel: "debug"
```

### Support

- Documentation: [GitHub Repository](https://github.com/thc1006/nephoran-intent-operator)
- Issues: [GitHub Issues](https://github.com/thc1006/nephoran-intent-operator/issues)
- Slack: [#nephoran-operator](https://kubernetes.slack.com/channels/nephoran-operator)

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| thc1006 | <thc1006@example.com> |  |

## Source Code

* <https://github.com/thc1006/nephoran-intent-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| git.tokenSecret | string | `""` | Secret containing Git access token |
| global.imagePullSecrets | list | `[]` | Global image pull secrets |
| global.imageRegistry | string | `""` | Global Docker image registry |
| ingress.annotations | object | `{}` | Ingress annotations |
| ingress.className | string | `""` | Ingress class name |
| ingress.enabled | bool | `false` | Enable ingress controller resource |
| ingress.hosts[0].host | string | `"nephoran-operator.local"` | Default hostname |
| ingress.hosts[0].paths[0].path | string | `"/"` | Default path |
| ingress.hosts[0].paths[0].pathType | string | `"Prefix"` | Path type |
| ingress.tls | list | `[]` | TLS configuration |
| llmProcessor.affinity | object | `{}` | Affinity rules |
| llmProcessor.database.enabled | bool | `false` | Enable database integration |
| llmProcessor.database.type | string | `"postgres"` | Database type |
| llmProcessor.enabled | bool | `true` | Enable LLM Processor deployment |
| llmProcessor.healthCheck.enabled | bool | `true` | Enable health checks |
| llmProcessor.healthCheck.livenessProbe.failureThreshold | int | `3` | Failure threshold |
| llmProcessor.healthCheck.livenessProbe.httpGet.path | string | `"/healthz"` | Liveness probe path |
| llmProcessor.healthCheck.livenessProbe.httpGet.port | int | `8080` | Liveness probe port |
| llmProcessor.healthCheck.livenessProbe.initialDelaySeconds | int | `30` | Initial delay seconds |
| llmProcessor.healthCheck.livenessProbe.periodSeconds | int | `10` | Period seconds |
| llmProcessor.healthCheck.livenessProbe.timeoutSeconds | int | `5` | Timeout seconds |
| llmProcessor.healthCheck.readinessProbe.failureThreshold | int | `3` | Failure threshold |
| llmProcessor.healthCheck.readinessProbe.httpGet.path | string | `"/readyz"` | Readiness probe path |
| llmProcessor.healthCheck.readinessProbe.httpGet.port | int | `8080` | Readiness probe port |
| llmProcessor.healthCheck.readinessProbe.initialDelaySeconds | int | `5` | Initial delay seconds |
| llmProcessor.healthCheck.readinessProbe.periodSeconds | int | `10` | Period seconds |
| llmProcessor.healthCheck.readinessProbe.timeoutSeconds | int | `5` | Timeout seconds |
| llmProcessor.image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| llmProcessor.image.repository | string | `"thc1006/nephoran-llm-processor"` | Docker image repository |
| llmProcessor.image.tag | string | `"latest"` | Image tag |
| llmProcessor.logLevel | string | `"info"` | Log level (debug, info, warn, error) |
| llmProcessor.metrics.allowedCIDRs | list | `["10.0.0.0/8","172.16.0.0/12","192.168.0.0/16","127.0.0.0/8"]` | CIDR blocks allowed to access metrics |
| llmProcessor.metrics.exposePublicly | bool | `false` | Expose metrics publicly (not recommended for production) |
| llmProcessor.nodeSelector | object | `{}` | Node selector |
| llmProcessor.podSecurityContext.fsGroup | int | `65534` | File system group |
| llmProcessor.providers.anthropic.enabled | bool | `false` | Enable Anthropic Claude |
| llmProcessor.providers.anthropic.model | string | `"claude-3-sonnet"` | Claude model |
| llmProcessor.providers.azure.deployment | string | `"gpt-4"` | Azure OpenAI deployment |
| llmProcessor.providers.azure.enabled | bool | `false` | Enable Azure OpenAI |
| llmProcessor.providers.google.enabled | bool | `false` | Enable Google Gemini |
| llmProcessor.providers.google.model | string | `"gemini-pro"` | Gemini model |
| llmProcessor.providers.openai.enabled | bool | `true` | Enable OpenAI |
| llmProcessor.providers.openai.model | string | `"gpt-4o-mini"` | OpenAI model |
| llmProcessor.ragService.url | string | `"http://rag-api:5001"` | RAG API service URL |
| llmProcessor.replicaCount | int | `1` | Number of replicas |
| llmProcessor.resources.limits.cpu | string | `"500m"` | CPU limit |
| llmProcessor.resources.limits.memory | string | `"512Mi"` | Memory limit |
| llmProcessor.resources.requests.cpu | string | `"100m"` | CPU request |
| llmProcessor.resources.requests.memory | string | `"128Mi"` | Memory request |
| llmProcessor.securityContext.allowPrivilegeEscalation | bool | `false` | Disallow privilege escalation |
| llmProcessor.securityContext.capabilities.drop[0] | string | `"ALL"` | Drop all capabilities |
| llmProcessor.securityContext.readOnlyRootFilesystem | bool | `true` | Read-only root filesystem |
| llmProcessor.securityContext.runAsNonRoot | bool | `true` | Run as non-root user |
| llmProcessor.securityContext.runAsUser | int | `65534` | User ID |
| llmProcessor.service.port | int | `8080` | Service port |
| llmProcessor.service.targetPort | int | `8080` | Target port |
| llmProcessor.service.type | string | `"ClusterIP"` | Service type |
| llmProcessor.tolerations | list | `[]` | Tolerations |
| ml.enabled | bool | `false` | Enable Machine Learning features |
| monitoring.enabled | bool | `false` | Enable monitoring |
| monitoring.serviceMonitor.enabled | bool | `false` | Enable ServiceMonitor |
| monitoring.serviceMonitor.interval | string | `"30s"` | Scrape interval |
| monitoring.serviceMonitor.scrapeTimeout | string | `"10s"` | Scrape timeout |
| rag.compat.enableLegacyProcessIntent | bool | `true` | Use legacy /process_intent endpoint |
| rag.enabled | bool | `false` | Enable Retrieval-Augmented Generation |
| ragApi.enabled | bool | `false` | Enable RAG API deployment (controlled by rag.enabled) |
| ragApi.env[0].name | string | `"WEAVIATE_URL"` | Weaviate URL environment variable |
| ragApi.env[0].value | string | `"http://weaviate.weaviate.svc.cluster.local:8080"` | Weaviate service URL |
| ragApi.env[1].name | string | `"OPENAI_API_KEY"` | OpenAI API key environment variable |
| ragApi.env[1].valueFrom.secretKeyRef.key | string | `"apiKey"` | Secret key |
| ragApi.env[1].valueFrom.secretKeyRef.name | string | `"openai-credentials"` | Secret name |
| ragApi.image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| ragApi.image.repository | string | `"python"` | Docker image repository |
| ragApi.image.tag | string | `"3.11-slim"` | Image tag |
| ragApi.replicaCount | int | `1` | Number of replicas |
| ragApi.resources.limits.cpu | string | `"500m"` | CPU limit |
| ragApi.resources.limits.memory | string | `"1Gi"` | Memory limit |
| ragApi.resources.requests.cpu | string | `"100m"` | CPU request |
| ragApi.resources.requests.memory | string | `"256Mi"` | Memory request |
| ragApi.service.port | int | `5001` | Service port |
| ragApi.service.targetPort | int | `5001` | Target port |
| ragApi.service.type | string | `"ClusterIP"` | Service type |
| rbac.create | bool | `true` | Create RBAC resources |
| secrets.apiKeys.name | string | `"nephoran-api-keys"` | API keys secret name |
| secrets.apiKeys.namespace | string | `"nephoran-system"` | API keys secret namespace |
| secrets.auth.name | string | `"nephoran-auth-secrets"` | Auth secrets name |
| secrets.auth.namespace | string | `"nephoran-system"` | Auth secrets namespace |
| secrets.database.name | string | `"nephoran-database-credentials"` | Database credentials secret name |
| secrets.database.namespace | string | `"nephoran-system"` | Database credentials namespace |
| secrets.external.backend | string | `"vault"` | External secrets backend |
| secrets.external.enabled | bool | `false` | Enable external secrets operator |
| secrets.external.refreshInterval | string | `"1h"` | Refresh interval |
| secrets.mtls.name | string | `"nephoran-mtls-certificates"` | mTLS certificates secret name |
| secrets.mtls.namespace | string | `"nephoran-system"` | mTLS certificates namespace |
| secrets.oran.name | string | `"nephoran-oran-credentials"` | O-RAN credentials secret name |
| secrets.oran.namespace | string | `"nephoran-system"` | O-RAN credentials namespace |
| secrets.tls.name | string | `"nephoran-tls-certificates"` | TLS certificates secret name |
| secrets.tls.namespace | string | `"nephoran-system"` | TLS certificates namespace |
| security.mtls.enabled | bool | `false` | Enable mTLS for service-to-service communication |
| security.networkPolicies.allowNamespaces | list | `[]` | Allowed namespaces |
| security.networkPolicies.enabled | bool | `true` | Enable network policies |
| security.podSecurityStandards.audit | string | `"restricted"` | Audit level |
| security.podSecurityStandards.enforce | string | `"restricted"` | Enforce level |
| security.podSecurityStandards.warn | string | `"restricted"` | Warning level |
| security.scanning.enabled | bool | `true` | Enable security scanning |
| security.scanning.falco.enabled | bool | `false` | Enable Falco runtime security |
| security.scanning.trivy.enabled | bool | `true` | Enable Trivy vulnerability scanning |
| serviceAccount.annotations | object | `{}` | ServiceAccount annotations |
| serviceAccount.create | bool | `true` | Create ServiceAccount |
| serviceAccount.name | string | `""` | ServiceAccount name (auto-generated if empty) |
| weaviate.enabled | bool | `false` | Enable Weaviate vector database |
| weaviate.service.grpcPort | int | `50051` | gRPC port |
| weaviate.service.port | int | `8080` | HTTP port |

----------------------------------------------

## Additional Documentation

### Custom Resource Definitions (CRDs)

The chart installs the following CRDs:

#### NetworkIntent

Defines natural language network intents for automated deployment:

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: high-availability-amf
spec:
  intent: "Deploy a high-availability AMF instance for production with auto-scaling"
  priority: high
  environment: production
  parameters:
    replicas: 3
    resources:
      cpu: "2"
      memory: "4Gi"
```

#### E2NodeSet

Manages E2 node configurations for O-RAN RIC testing:

```yaml
apiVersion: nephoran.io/v1
kind: E2NodeSet
metadata:
  name: test-e2-nodes
spec:
  replicas: 5
  nodeType: "simulated"
  ricEndpoint: "ric.nephoran.local:36422"
```

### Performance Tuning

For optimal performance, consider these configurations:

```yaml
# High-performance configuration
llmProcessor:
  replicaCount: 5
  resources:
    requests:
      cpu: "1"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "4Gi"

weaviate:
  persistence:
    size: 100Gi
    storageClass: fast-ssd
  resources:
    requests:
      cpu: "2"
      memory: "8Gi"
```

### Multi-Cluster Deployment

For multi-cluster deployments:

1. Deploy the controller in the management cluster
2. Configure service accounts for target clusters
3. Use the multi-cluster values:

```yaml
multiCluster:
  enabled: true
  targetClusters:
    - name: edge-1
      endpoint: https://edge1.example.com
      secretRef: edge1-kubeconfig
    - name: edge-2
      endpoint: https://edge2.example.com
      secretRef: edge2-kubeconfig
```

### Integration with CI/CD

Example GitOps integration with ArgoCD:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nephoran-operator
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://charts.nephoran.io
    targetRevision: 0.1.0
    chart: nephoran-operator
    helm:
      values: |
        rag:
          enabled: true
        monitoring:
          enabled: true
  destination:
    server: https://kubernetes.default.svc
    namespace: nephoran-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### License

Apache License 2.0

### Contributing

We welcome contributions! Please see our [Contributing Guide](https://github.com/thc1006/nephoran-intent-operator/blob/main/CONTRIBUTING.md) for details.

---

Generated by [helm-docs](https://github.com/norwoodj/helm-docs)