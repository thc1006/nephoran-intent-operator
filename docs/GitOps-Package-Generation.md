# GitOps Package Generation with Nephio Porch Integration

## Overview

The Nephoran Intent Operator now supports generating Nephio-compliant KRM (Kubernetes Resource Model) packages from NetworkIntent resources. This feature enables seamless integration with Nephio's Porch package orchestration system for advanced GitOps workflows.

## Architecture

### Package Generation Flow

```
NetworkIntent CR → Controller → Package Generator → KRM Package → Git Repository → Nephio Porch
                                        ↓
                              Template Processing
                                        ↓
                               Resource Generation
                                        ↓
                               Metadata Addition
```

### Generated Package Structure

```
packages/
└── <namespace>/
    └── <intent-name>-package/
        ├── Kptfile                    # Package metadata and pipeline configuration
        ├── README.md                  # Package documentation
        ├── fn-config.yaml            # Function configuration
        ├── setters.yaml              # Configurable parameters
        ├── deployment.yaml           # Kubernetes deployment (if applicable)
        ├── service.yaml              # Kubernetes service (if applicable)
        ├── oran-config.yaml          # O-RAN configuration (if applicable)
        ├── scaling-patch.yaml        # Scaling patches (for scaling intents)
        ├── policy.yaml               # Policy resources (for policy intents)
        └── a1-policy.yaml            # A1 interface policies (if applicable)
```

## Configuration

### Enabling Nephio Porch Integration

Set the environment variable to enable KRM package generation:

```bash
export USE_NEPHIO_PORCH=true
```

### Package Generator Configuration

The package generator is automatically initialized when Nephio Porch integration is enabled. It includes:

- Pre-defined templates for various resource types
- Telecom-specific configurations
- O-RAN interface mappings
- Network slice parameters

## Package Types

### 1. Deployment Packages

Generated for deployment intents, containing:
- Kubernetes Deployment manifests
- Service definitions
- ConfigMaps for O-RAN configuration
- Network slice parameters

Example intent:
```yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: deploy-amf
spec:
  intent: "Deploy AMF with 3 replicas for eMBB slice"
  type: deployment
```

### 2. Scaling Packages

Generated for scaling operations:
- Deployment patches
- HPA (Horizontal Pod Autoscaler) resources
- Scaling setters for customization

Example intent:
```yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: scale-du
spec:
  intent: "Scale O-DU to 5 replicas"
  type: scaling
```

### 3. Policy Packages

Generated for policy management:
- NetworkPolicy resources
- A1 policy configurations
- O1 interface policies

Example intent:
```yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: apply-qos-policy
spec:
  intent: "Apply QoS policy for low latency slice"
  type: policy
```

## Kptfile Structure

Each package includes a Kptfile with:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: <package-name>
  annotations:
    config.kubernetes.io/local-config: "true"
    nephoran.com/intent-id: <intent-id>
info:
  description: |
    Generated from NetworkIntent: <intent-name>
    Original Intent: <natural-language-intent>
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/apply-setters:v0.2.0
  - image: gcr.io/kpt-fn/set-namespace:v0.4.1
  validators:
  - image: gcr.io/kpt-fn/kubeval:v0.3.0
```

## Template System

### Available Templates

1. **Deployment Template**: For network function deployments
2. **Service Template**: For service exposure
3. **ConfigMap Template**: For configuration data
4. **Policy Template**: For network and security policies

### Template Variables

Templates support the following variables:
- `{{.Name}}`: Resource name
- `{{.Namespace}}`: Target namespace
- `{{.Component}}`: Component type (e.g., "5g-core", "o-ran")
- `{{.Image}}`: Container image
- `{{.Replicas}}`: Replica count
- `{{.Ports}}`: Port configurations
- `{{.Resources}}`: Resource requirements

## O-RAN Integration

### O-RAN Specific Features

1. **O1 Interface Configuration**
   - FCAPS management endpoints
   - Performance monitoring settings
   - Fault management parameters

2. **A1 Interface Policies**
   - Policy type definitions
   - SLA parameters
   - QoS configurations

3. **E2 Interface Setup**
   - RAN control parameters
   - Cell configuration
   - Bearer management

### Example O-RAN Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: amf-o1-config
data:
  o1-config.yaml: |
    management_endpoint: https://amf.telecom-core.svc.cluster.local:8081
    fcaps_config:
      pm_enabled: true
      fm_enabled: true
      cm_enabled: true
    metrics:
      interval: 30s
      retention: 7d
```

## Network Slice Support

### Slice Configuration

Each package can include network slice parameters:

```yaml
network_slice:
  slice_id: "embb-001"
  slice_type: "eMBB"
  sla_parameters:
    latency_ms: 20
    throughput_mbps: 1000
    reliability: 0.999
```

### Supported Slice Types

- **eMBB**: Enhanced Mobile Broadband
- **URLLC**: Ultra-Reliable Low-Latency Communication
- **mMTC**: Massive Machine-Type Communication

## Usage Examples

### 1. Deploy 5G Core Function

```bash
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: deploy-smf
spec:
  intent: "Deploy SMF with high availability for enterprise slice"
  type: deployment
EOF
```

### 2. Scale O-RAN Component

```bash
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: scale-cu
spec:
  intent: "Scale O-CU to handle increased traffic"
  type: scaling
EOF
```

### 3. Apply Network Policy

```bash
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: secure-ran
spec:
  intent: "Apply security policy to isolate RAN components"
  type: policy
EOF
```

## Validation and Testing

### Package Validation

Generated packages are validated using:
1. Kpt validators
2. Schema validation
3. Kubernetes dry-run

### Testing Workflow

```bash
# Render the package locally
kpt fn render packages/<namespace>/<package-name>

# Validate the package
kpt fn eval packages/<namespace>/<package-name> --image gcr.io/kpt-fn/kubeval:v0.3.0

# Apply to test cluster
kpt live init packages/<namespace>/<package-name>
kpt live apply packages/<namespace>/<package-name>
```

## Troubleshooting

### Common Issues

1. **Template Parsing Errors**
   - Check template syntax
   - Verify all variables are defined
   - Review logs for specific errors

2. **Git Push Failures**
   - Verify Git credentials
   - Check repository permissions
   - Ensure branch protection rules allow pushes

3. **Package Generation Failures**
   - Verify LLM processed parameters correctly
   - Check intent type is supported
   - Review controller logs

### Debug Mode

Enable debug logging:
```bash
kubectl logs -n nephoran-system deployment/nephio-bridge -f | grep -i package
```

## Best Practices

1. **Intent Clarity**: Write clear, specific intents for better package generation
2. **Parameter Validation**: Ensure LLM outputs valid structured parameters
3. **Version Control**: Tag generated packages for traceability
4. **Testing**: Always test packages in development before production
5. **Documentation**: Keep README files updated in generated packages

## Future Enhancements

- Direct Porch API integration for package lifecycle management
- Package dependency resolution
- Automated rollback capabilities
- Multi-cluster package distribution
- Package versioning and promotion workflows