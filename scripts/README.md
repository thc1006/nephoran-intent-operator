# Scripts Directory

This directory contains essential scripts for building, deploying, and managing the Nephoran Intent Operator.

## Build Scripts

### `docker-build.sh`
Production Docker build script with security scanning and multi-architecture support.

**Usage:**
```bash
./docker-build.sh <service> [options]
```

**Services:**
- `llm-processor` - Build LLM Processor service
- `nephio-bridge` - Build Nephio Bridge service  
- `oran-adaptor` - Build ORAN Adaptor service
- `all` - Build all services

**Options:**
- `--push` - Push images to registry after build
- `--scan` - Run security scan using Trivy
- `--multi-arch` - Build for multiple architectures
- `--no-cache` - Build without using Docker cache
- `--registry` - Set custom registry
- `--version` - Set version tag
- `--platforms` - Set target platforms

**Examples:**
```bash
./docker-build.sh llm-processor --push --scan
./docker-build.sh all --multi-arch --push
```

### `docker-security-scan.sh`
Comprehensive security validation for container images.

**Usage:**
```bash
./docker-security-scan.sh [options] <target>
```

**Targets:**
- `image:<name>` - Scan specific container image
- `dockerfile:<path>` - Scan Dockerfile for security issues
- `all` - Run comprehensive security scan

**Options:**
- `--severity` - Set Trivy severity levels (default: HIGH,CRITICAL)
- `--format` - Output format: table|json|sarif
- `--docker-bench` - Run Docker Bench Security test
- `--no-hadolint` - Skip Hadolint Dockerfile scanning

## Deployment Scripts

### `deploy.sh`
Basic deployment script for local and remote environments.

**Usage:**
```bash
./deploy.sh [local|remote]
```

- `local` - Build images and deploy to current Kubernetes context using 'Never' imagePullPolicy
- `remote` - Build, push images to Google Artifact Registry, then deploy

### `deploy-optimized.sh`
Advanced GitOps-friendly deployment script with comprehensive options.

**Usage:**
```bash
./deploy-optimized.sh [OPTIONS] COMMAND
```

**Commands:**
- `build` - Build container images
- `deploy` - Deploy to Kubernetes cluster
- `all` - Build and deploy (default)

**Options:**
- `-e, --env ENV` - Environment: local, dev, staging, production
- `-r, --registry REG` - Container registry
- `-t, --tag TAG` - Image tag
- `-n, --namespace NS` - Kubernetes namespace
- `-d, --dry-run` - Show what would be done without executing
- `-s, --skip-build` - Skip image building phase
- `-p, --push` - Push images to registry

**Examples:**
```bash
./deploy-optimized.sh --env local build
./deploy-optimized.sh --env staging --tag v1.2.3 deploy
./deploy-optimized.sh --env production --registry gcr.io/myproject/nephoran all
```

## Windows Scripts

### `setup-windows.ps1`
Windows development environment setup script.

**Usage:**
```powershell
.\setup-windows.ps1 [options]
```

**Options:**
- `-Force` - Force reinstall even if components exist
- `-SkipKubernetes` - Skip Kubernetes cluster setup
- `-Help` - Show help message

**Features:**
- Installs Chocolatey package manager
- Installs Go, Python, Docker Desktop, kubectl, kind, Git
- Sets up Python virtual environment
- Creates Windows-compatible environment script
- Sets up local Kubernetes cluster with kind

### `local-deploy.ps1`
PowerShell deployment script for Windows local development.

### `test-comprehensive.ps1`
Comprehensive test suite runner for Windows.

### `validate-environment.ps1`
Environment validation script for Windows.

## Utility Scripts

### `mcp-helm-server.py`
MCP (Model Context Protocol) Helm server for AI-assisted operations.

## RAG and AI Scripts

### `populate_vector_store.py`
Script to populate the vector database with telecom knowledge base.

### `populate_vector_store_enhanced.py`
Enhanced version with additional features for vector store population.

### `prompt_ab_testing.py`
A/B testing framework for prompt optimization.

### `prompt_evaluation_metrics.py`
Evaluation metrics for prompt performance analysis.

### `prompt_optimization_demo.py`
Demonstration script for prompt optimization techniques.

## Infrastructure Scripts

### `deploy-istio-mesh.sh`
Deploy and configure Istio service mesh for the Nephoran system.

### `deploy-multi-region.sh`
Multi-region deployment automation script.

### `deploy-rag-system.sh`
Deploy the RAG (Retrieval-Augmented Generation) system components.

### `disaster-recovery-system.sh`
Disaster recovery automation and orchestration.

### `disaster-recovery.sh`
Basic disaster recovery operations.

### `dr-automation-scheduler.sh`
Automated disaster recovery scheduler.

## Validation and Testing Scripts

### `edge-deployment-validation.sh`
Validate edge computing deployment configurations.

### `execute-production-load-test.sh`
Execute production-grade load testing scenarios.

### `execute-security-audit.sh`
Comprehensive security audit execution.

### `oran-compliance-validator.sh`
Validate ORAN (Open RAN) compliance requirements.

### `performance-benchmark-suite.sh`
Performance benchmarking and analysis suite.

### `validate-build.sh`
Build validation and verification.

### `validate-load-test-capability.sh`
Validate load testing infrastructure and capabilities.

### `validate-security-hardening.sh`
Security hardening validation script.

## Maintenance Scripts

### `build-fix-summary.sh`
Generate build fix summaries and reports.

### `fix-critical-errors.sh`
Automated critical error resolution.

### `monitoring-operations.sh`
Monitoring system operations and maintenance.

### `setup-enhanced-cicd.sh`
Enhanced CI/CD pipeline setup and configuration.

### `update-dependencies.sh`
Automated dependency updates with validation.

## Security Scripts

### `security-config-validator.sh`
Security configuration validation.

### `security-penetration-test.sh`
Automated penetration testing framework.

### `security-scan.sh`
General security scanning operations.

### `vulnerability-scanner.sh`
Vulnerability assessment and scanning.

## Usage Guidelines

1. **Prerequisites**: Ensure all required tools are installed (Docker, kubectl, Go, Python)
2. **Permissions**: Make scripts executable: `chmod +x script-name.sh`
3. **Environment**: Set appropriate environment variables before running deployment scripts
4. **Testing**: Use `--dry-run` options where available to test operations before execution
5. **Security**: Review security scan results and address findings before production deployment

## Environment Variables

Common environment variables used across scripts:

- `NEPHORAN_REGISTRY` - Default container registry
- `NEPHORAN_TAG` - Default image tag
- `NEPHORAN_NAMESPACE` - Default Kubernetes namespace
- `KUBECONFIG` - Kubernetes configuration file
- `DOCKER_BUILDKIT` - Enable Docker BuildKit
- `TRIVY_SEVERITY` - Trivy scan severity levels

## Support

For issues with any script, check the individual script's help output using `--help` flag, or refer to the main project documentation.