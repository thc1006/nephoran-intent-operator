# Archive Directory

## Purpose

This directory contains a collection of example and test YAML configurations for the Nephoran Intent Operator. These files serve as practical examples, testing resources, and learning materials for users getting started with the platform. They demonstrate the core functionality of the system and provide templates for common deployment scenarios.

## File Descriptions

### my-first-intent.yaml
**Purpose**: Simple UPF (User Plane Function) deployment example  
**Use Case**: Basic NetworkIntent demonstration for new users  
**Description**: Contains a minimal NetworkIntent custom resource that requests deployment of a UPF in the central region for enterprise customers. This example showcases how natural language intents are translated into network function deployments.

**Content**:
- NetworkIntent CRD with natural language specification
- Demonstrates intent-driven deployment approach
- Perfect for first-time users and quickstart scenarios

### test-deployment.yaml
**Purpose**: Complete deployment configuration with core services  
**Use Case**: Full system deployment for testing and development environments  
**Description**: Comprehensive Kubernetes deployment manifest that includes the LLM Processor service, Nephio Bridge controller, and all necessary RBAC configurations. This file demonstrates the complete microservices architecture of the platform.

**Content**:
- LLM Processor deployment with health checks and environment configuration
- Nephio Bridge controller with proper RBAC permissions
- Service definitions for inter-component communication
- ServiceAccount and ClusterRole configurations
- Production-ready container configurations with liveness/readiness probes

### test-networkintent.yaml
**Purpose**: E2 node deployment example  
**Use Case**: Testing O-RAN E2 interface functionality and multi-node scenarios  
**Description**: Demonstrates deployment of multiple E2 nodes for testing network connectivity. This example showcases the platform's O-RAN compliance and ability to handle complex network topologies.

**Content**:
- NetworkIntent for E2 node cluster deployment
- Multi-replica configuration example
- O-RAN interface testing scenario

## Usage Instructions

### For Learning and Exploration
1. **Start with the basics**: Use `my-first-intent.yaml` to understand NetworkIntent structure
2. **Understand the architecture**: Review `test-deployment.yaml` to see complete system deployment
3. **Explore advanced features**: Use `test-networkintent.yaml` for O-RAN functionality testing

### For Development and Testing
```bash
# Deploy core services
kubectl apply -f test-deployment.yaml

# Test basic intent processing
kubectl apply -f my-first-intent.yaml

# Test E2 node functionality
kubectl apply -f test-networkintent.yaml
```

### For Quickstart Scenarios
These files are referenced by automated setup scripts:
- Quickstart scripts (`scripts/quickstart.sh`, `scripts/quickstart.ps1`) create `my-first-intent.yaml` dynamically
- Deployment scripts (`scripts/local-deploy.ps1`, `scripts/deploy-windows.ps1`) reference these examples
- Setup guides (`docs/COMPLETE-SETUP-GUIDE.md`) use these files for testing

### Command Examples
```bash
# Apply a NetworkIntent
kubectl apply -f archive/my-first-intent.yaml

# Check intent status
kubectl get networkintents

# View intent details
kubectl describe networkintent deploy-upf-central

# Monitor deployment progress
kubectl get pods -w
```

## Important Notes

### Image Tags and Registry
⚠️ **Important**: The container images in `test-deployment.yaml` use specific tags and registry paths:
- `us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran/llm-processor:f87350a-dirty`
- `us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran/nephio-bridge:f87350a-dirty`

**Before deployment**, you may need to:
1. Update image tags to match your available versions
2. Change registry paths to your container registry
3. Modify `imagePullPolicy` based on your deployment scenario
4. Ensure proper image pull secrets are configured if using private registries

### Environment Configuration
- The deployments use default environment variables suitable for testing
- Production deployments should review and customize environment settings
- Pay attention to service URLs and networking configuration

### RBAC Permissions
- `test-deployment.yaml` includes comprehensive RBAC configurations
- The permissions are designed for testing and may need adjustment for production
- Review ClusterRole permissions before applying in production environments

## References

### Scripts That Use These Files
- `scripts/quickstart.sh` - Creates and applies `my-first-intent.yaml`
- `scripts/quickstart.ps1` - PowerShell version of quickstart
- `scripts/quick-deploy.bat` - Batch deployment script
- `scripts/local-k8s-setup.ps1` - Local Kubernetes setup
- `scripts/local-deploy.ps1` - Local deployment automation
- `scripts/deploy-windows.ps1` - Windows deployment script

### Documentation References
- `README.md` - Main project documentation references these examples
- `QUICKSTART.md` - Uses `my-first-intent.yaml` for getting started
- `docs/COMPLETE-SETUP-GUIDE.md` - References examples for testing

### Development and Testing
These files are essential for:
- Developer onboarding and learning
- Continuous integration testing
- System validation and verification
- Demonstration and training scenarios

## Troubleshooting

### Common Issues
1. **Image Pull Errors**: Update image tags and registry paths
2. **RBAC Failures**: Ensure proper cluster permissions
3. **Service Discovery**: Verify service names match deployment labels
4. **Resource Conflicts**: Use unique namespaces for parallel testing

### Debugging Commands
```bash
# Check pod status
kubectl get pods -o wide

# View logs
kubectl logs deployment/llm-processor
kubectl logs deployment/nephio-bridge

# Check service endpoints
kubectl get endpoints

# Verify NetworkIntent processing
kubectl get networkintents -o yaml
```

For additional support and advanced usage scenarios, refer to the main project documentation and the comprehensive setup guide.