# 🚀 Nephoran Intent Operator

<div align="center">

![Nephoran Intent Operator](https://img.shields.io/badge/Nephoran-Intent%20Operator-blue?style=for-the-badge&logo=kubernetes)

**MVP: Demonstrating intent-driven network orchestration with AI-powered translation**

[![Build Status](https://img.shields.io/github/actions/workflow/status/thc1006/nephoran-intent-operator/ci.yml?branch=main&style=flat-square&logo=github)](https://github.com/thc1006/nephoran-intent-operator/actions/workflows/ci.yml)
[![Documentation](https://img.shields.io/github/actions/workflow/status/thc1006/nephoran-intent-operator/docs-unified.yml?branch=main&style=flat-square&logo=gitbook&label=docs)](https://thc1006.github.io/nephoran-intent-operator)

<!-- CI Trigger: Force workflow execution for graceful shutdown exit codes PR - Updated for merge readiness -->
[![Go Version](https://img.shields.io/github/go-mod/go-version/thc1006/nephoran-intent-operator?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/github/license/thc1006/nephoran-intent-operator?style=flat-square)](LICENSE)
[![Code Coverage](https://img.shields.io/codecov/c/github/thc1006/nephoran-intent-operator?style=flat-square&logo=codecov)](https://codecov.io/gh/thc1006/nephoran-intent-operator)
[![Docker Pulls](https://img.shields.io/docker/pulls/nephoran/intent-operator?style=flat-square&logo=docker)](https://hub.docker.com/r/nephoran/intent-operator)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.30+-blue?style=flat-square&logo=kubernetes)](https://kubernetes.io/)
[![O-RAN Ready](https://img.shields.io/badge/O--RAN-Ready-blue?style=flat-square&logo=verified)](https://www.o-ran.org/)
[![Security Scan](https://img.shields.io/github/actions/workflow/status/thc1006/nephoran-intent-operator/security-consolidated.yml?branch=main&style=flat-square&logo=security&label=security)](https://github.com/thc1006/nephoran-intent-operator/actions/workflows/security-consolidated.yml)
[![Release](https://img.shields.io/github/v/release/thc1006/nephoran-intent-operator?style=flat-square&logo=github)](https://github.com/thc1006/nephoran-intent-operator/releases)
[![DOI](https://zenodo.org/badge/1026653090.svg)](https://doi.org/10.5281/zenodo.16813086)

</div>

---

## 🎯 Project Overview

The **Nephoran Intent Operator** is a proof-of-concept cloud-native platform that demonstrates the potential of intent-driven telecommunications orchestration. This MVP showcases how natural language intents can be translated into structured network function configurations through AI-powered processing, providing a foundation for future production-ready telecommunications automation.

**🌟 Key Value Proposition (MVP Scope):**
- **Natural Language Interface**: Translate network intents into structured configurations using AI
- **Kubernetes-Native**: CRD-based controller for NetworkIntent resource management
- **LLM Integration**: GPT-4o-mini processing with optional RAG enhancement (behind build tag)
- **Extensible Architecture**: Foundation for O-RAN interface implementations and production features
- **Proof-of-Concept**: Demonstrates intent-driven orchestration potential with simulated network functions

### 🔬 MVP Status - Proof of Concept

Currently an **MVP/proof-of-concept** demonstrating intent-driven network orchestration capabilities. The system includes core functionality for NetworkIntent processing, LLM integration, and basic controller operations with comprehensive testing coverage.

## ✨ Core Features & Capabilities

### 🧠 AI-Powered Intent Processing (MVP Features)
- **LLM Integration**: GPT-4o-mini for natural language processing
- **Optional RAG System**: Weaviate vector database integration available (when enabled via build tags)
- **Context Enhancement**: Framework for semantic retrieval and knowledge augmentation
- **Extensible Architecture**: Support for multiple LLM providers

### 🔧 Core MVP Components
- **NetworkIntent CRD**: Kubernetes custom resource for capturing network intents
- **Intent Controller**: Processes NetworkIntent resources and manages lifecycle
- **LLM Processor**: Translates natural language to structured network configurations
- **RAG System**: Optional context enhancement (enabled via build tags)
- **FCAPS Simulator**: Automated scaling decisions based on telecom events ([docs](docs/FCAPS_SIMULATOR.md))

### 🏗️ Cloud-Native Architecture
- **Kubernetes-Native**: Custom resources, operators, and webhooks following K8s best practices
- **GitOps Foundation**: Basic package generation capabilities for future Nephio integration
- **Cloud-Native Patterns**: Service-oriented architecture ready for production scaling
- **Observability**: Prometheus metrics and health endpoints for monitoring

### 🔒 Enhanced Security Features (v0.2.0)
- **Critical Security Fixes**: Comprehensive input validation, path traversal prevention, and command injection protection
- **Secure Patch Generation**: Migration from `internal/patch` to `internal/patchgen` with enhanced security validation
- **Timestamp Security**: RFC3339 format with collision prevention and replay attack mitigation
- **JSON Schema Validation**: Strict input validation with JSON Schema 2020-12 compliance
- **HTTP Security**: Basic authentication and configurable endpoint access
- **Kubernetes RBAC**: Standard service account and role-based permissions
- **Container Security**: Base image scanning in CI pipeline
- **Configuration Security**: Environment-based secrets management

### 📊 MVP Observability
- **Basic Metrics**: Prometheus metrics for LLM requests, controller operations, and system health
- **Health Endpoints**: Kubernetes liveness and readiness probes
- **Structured Logging**: JSON-formatted logs with request tracing
- **Debugging Support**: Comprehensive error handling and status reporting

### 🚀 Simulated Network Functions
- **Intent Translation**: Convert natural language to network function parameters
- **Configuration Generation**: Create structured YAML/JSON for network deployments
- **Status Management**: Track intent processing lifecycle and deployment state
- **Extensibility**: Framework for integrating real network function deployments

## ⚡ 15-Minute Quickstart

Get from zero to your first deployed network function in exactly 15 minutes! 

### 🔧 Prerequisites (2 minutes)

Ensure you have these tools installed:
```bash
# Check required tools
docker --version      # Docker 20.10+
kubectl version --client  # Kubernetes v1.30+
git --version         # Git 2.30+
go version            # Go 1.24+
```

Quick install if needed:
```bash
# Linux/WSL
curl -fsSL https://get.docker.com | sh
curl -LO "https://dl.k8s.io/release/stable.txt" && curl -LO "https://dl.k8s.io/release/$(cat stable.txt)/bin/linux/amd64/kubectl"

# macOS
brew install docker kubectl kind

# Windows (PowerShell as Administrator)
winget install Docker.DockerDesktop Kubernetes.kubectl
```

### 🚀 Environment Setup (5 minutes)

```bash
# Clone the repository
git clone https://github.com/thc1006/nephoran-intent-operator.git
cd nephoran-intent-operator

# Create Kind cluster with optimal configuration
cat <<EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: nephoran-quickstart
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

kind create cluster --config=kind-config.yaml

# Install CRDs and deploy core services
kubectl create namespace nephoran-system
kubectl apply -f deployments/crds/
kubectl apply -f deployments/kustomize/base/llm-processor/
kubectl apply -f deployments/kustomize/base/nephio-bridge/
```

### 🎯 Deploy Your First Intent (5 minutes)

```bash
# Create a production-ready AMF network function using natural language
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: deploy-amf-production
  namespace: default
spec:
  intent: |
    Deploy a production-ready AMF (Access and Mobility Management Function) 
    for a 5G core network with:
    - High availability with 3 replicas
    - Auto-scaling (min: 3, max: 10 pods) 
    - Resource limits: 2 CPU cores, 4GB memory per pod
    - Prometheus monitoring on port 9090
    - Standard 3GPP interfaces (N1, N2, N11)
    - Support for 100k concurrent UE connections
EOF

# Watch the magic happen! 🪄
kubectl get networkintent deploy-amf-production -w

# View generated resources
kubectl get all -l generated-from=deploy-amf-production
```

### ✅ Success Validation (2 minutes)

Run our automated validation:
```bash
# Use the included quickstart script for full automation
./scripts/quickstart.sh

# Or run just the validation portion
./scripts/quickstart.sh --skip-prereq

# Expected output: 🎉 All checks passed!
```

**Time-Saving Alternative**: Run the entire quickstart with a single command:
```bash
# Automated 15-minute setup (includes validation)
./scripts/quickstart.sh --demo
```

### 🆘 Need Help?

If you encounter issues:
- Check our comprehensive [QUICKSTART.md](QUICKSTART.md) for detailed steps
- View [Documentation](docs/README.md) for organized guides and references
- Open [GitHub Issues](https://github.com/thc1006/nephoran-intent-operator/issues) for support

## 🏗️ System Architecture

The Nephoran Intent Operator implements a sophisticated five-layer cloud-native architecture:

```mermaid
graph TB
    A[Natural Language Intent] --> B[LLM/RAG Processing Layer]
    B --> C[Nephio R5 Control Plane]
    C --> D[O-RAN Interface Bridge]
    D --> E[Network Function Orchestration]
    
    B1[GPT-4o-mini + RAG] --> B
    B2[Weaviate Vector DB] --> B
    C1[Porch Package Orchestration] --> C
    C2[GitOps Workflows] --> C
    D1[A1/O1/O2/E2 Interfaces] --> D
    E1[5G Core + RAN Functions] --> E
```

### 🔄 Processing Pipeline

1. **Intent Capture**: Natural language requirements captured via NetworkIntent CRD
2. **AI Processing**: LLM analyzes intent with RAG-enhanced telecommunications knowledge
3. **Package Generation**: Structured parameters create Nephio-compliant packages
4. **GitOps Deployment**: Multi-cluster orchestration via ConfigSync and ArgoCD
5. **O-RAN Integration**: Standards-compliant network function deployment
6. **Monitoring & Feedback**: Comprehensive observability with status propagation

### 🔄 GitOps + RAG + Controllers Flow

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│  Natural Lang   │    │   LLM/RAG        │    │   NetworkIntent     │
│  Intent Input   │───▶│   Processor      │───▶│   Controller        │
│                 │    │                  │    │                     │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                                ▲                         │
                                │                         ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Weaviate      │    │   Knowledge      │    │   KRM Package       │
│   Vector DB     │◀───│   Base + RAG     │    │   Generation        │
│                 │    │                  │    │                     │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                                                         │
                                                         ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Monitoring    │    │   O-RAN Network  │    │   GitOps Repository │
│   & Feedback    │◀───│   Functions      │◀───│   (ConfigSync)      │
│                 │    │                  │    │                     │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
```

### 📈 Performance Characteristics

| Metric | MVP Status | Notes |
| Intent Processing Latency | Not optimized | Depends on LLM API response time |
| Concurrent Intents | Limited testing | Bounded by Kubernetes resources |
| Throughput | Development mode | Not tested for production load |  
| Availability | Basic K8s patterns | No SLA targets |
| Knowledge Base | Basic telco docs | Expandable via RAG system |
| LLM Integration | GPT-4o-mini | Multi-provider ready |

## 🔄 Recent Updates & Migration Guide

### Module Migration: internal/patch → internal/patchgen

The latest release includes a significant security-focused migration from `internal/patch` to `internal/patchgen` module with enhanced features:

#### Key Improvements:
- **Enhanced Security**: Comprehensive input validation and path traversal prevention
- **Timestamp Security**: RFC3339 format with collision prevention
- **JSON Schema Validation**: Strict validation using JSON Schema 2020-12
- **Secure File Operations**: Proper permissions and error handling

#### Migration Steps:
```go
// Before (internal/patch)
import "github.com/thc1006/nephoran-intent-operator/internal/patch"

// After (internal/patchgen)  
import "github.com/thc1006/nephoran-intent-operator/internal/patchgen"

// New validation requirement
validator, err := patchgen.NewValidator(logger)
intent, err := validator.ValidateIntent(intentData)
```

#### Security Enhancements:
- Path traversal attack prevention
- Command injection protection
- Secure timestamp generation
- Enhanced input validation framework

For detailed migration information, see [CHANGELOG.md](CHANGELOG.md#breaking-changes) and [SECURITY.md](SECURITY.md#recent-security-enhancements-v020).

## 🧪 MVP Use Cases & Demonstrations

### Intent Translation Example
```yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
spec:
  intent: |
    Deploy a production-ready AMF (Access and Mobility Management Function)
    for a 5G core network with:
    - High availability with 3 replicas
    - Auto-scaling configuration (min: 3, max: 10 pods)
    - Resource limits: 2 CPU cores, 4GB memory per pod
    - Prometheus monitoring on port 9090
    - Standard 3GPP interfaces (N1, N2, N11)
```

### RAN Function Configuration  
```yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
spec:
  intent: |
    Configure basic O-RAN components for testing:
    - E2 node simulation with configurable parameters
    - Basic RIC integration testing framework
    - Container-based network function templates
    - Development and testing resource specifications
```

### Basic Network Configuration
```yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
spec:
  intent: |
    Generate network function configurations:
    - Basic QoS parameter specification
    - Resource allocation templates
    - Container deployment manifests
    - Development environment setup
```

### FCAPS-Driven Automated Scaling
```bash
# Start the intent ingest service
go run ./cmd/intent-ingest &

# Run FCAPS simulator with telecom events
./fcaps-sim --verbose
# Automatically detects:
# - Critical faults → Scale up by 2 replicas
# - High PRB utilization (>0.8) → Scale up by 1
# - High latency (>100ms) → Scale up by 1

# Generated intent (automatic):
{
  "intent_type": "scaling",
  "target": "nf-sim",
  "replicas": 3,
  "reason": "Critical fault detected: LINK_DOWN",
  "source": "planner"
}
```

## 📚 Documentation & Learning

### 🎓 Getting Started
- **[15-Minute Quickstart](QUICKSTART.md)**: Complete tutorial from zero to deployed network function
- **[Developer Guide](docs/DEVELOPER_GUIDE.md)**: Architecture deep-dive and contribution guidelines  
- **[Operator Manual](docs/OPERATOR-MANUAL.md)**: Production deployment and operations
- **[API Reference](docs/API_REFERENCE.md)**: Complete REST and gRPC API documentation
- **[Deployment Fixes Guide](docs/DEPLOYMENT_FIXES_GUIDE.md)**: Latest infrastructure improvements and fixes
- **[CI/CD Infrastructure](docs/CI_CD_INFRASTRUCTURE.md)**: Comprehensive build pipeline documentation
- **[Enhanced Troubleshooting](docs/troubleshooting.md)**: Updated with recent fixes and solutions

### 🔍 Technical Reference

#### Recent Infrastructure Fixes (August 2025)
The latest release includes critical CI/CD and infrastructure improvements:
- **GitHub Actions Registry Cache**: Fixed GHCR authentication and Docker buildx configurations
- **Multi-platform Builds**: Enhanced AMD64/ARM64 support with improved caching (85% hit rate)
- **Build Pipeline**: Resolved Makefile syntax errors and Go 1.24+ compatibility issues
- **Quality Gates**: Updated golangci-lint to v1.62.0, fixed gocyclo installation issues
- **Performance**: Reduced average build time from 5.4 to 3.2 minutes (-41% improvement)

For complete details, see [Deployment Fixes Guide](docs/DEPLOYMENT_FIXES_GUIDE.md) and [CI/CD Infrastructure Documentation](docs/CI_CD_INFRASTRUCTURE.md).

#### Health and Probes
The system provides standardized health endpoints for Kubernetes liveness and readiness probes:
- **Liveness Endpoint**: `/healthz` - Basic service availability check
- **Readiness Endpoint**: `/readyz` - Ready to accept traffic indicator

#### RAG System Endpoints
The RAG (Retrieval-Augmented Generation) system supports multiple API endpoints:
- **Preferred Endpoints**: 
  - `POST /process` - Primary intent processing endpoint
  - `POST /stream` - Streaming intent processing with Server-Sent Events
- **Legacy Support**: 
  - `POST /process_intent` - Legacy endpoint (supported when enabled via configuration)

#### Security Configuration
Enhanced security features include:
- **Metrics Exposure Control**: Configure metrics endpoint exposure via `METRICS_ENABLED` flag
- **IP Allowlist**: Restrict metrics endpoint access using `METRICS_ALLOWED_IPS` configuration
- **HTTP Security Headers**: Automatically applied security headers including:
  - `Strict-Transport-Security` (HSTS) for HTTPS enforcement
  - `Content-Security-Policy` (CSP) for XSS protection
  - `X-Frame-Options` for clickjacking prevention
  - `X-Content-Type-Options` for MIME type sniffing protection

### 📁 Archive Directory
The **[archive/](archive/)** directory contains essential example YAML configurations and reference templates actively used throughout the project. Despite its name, these are not deprecated files but rather canonical examples that serve critical purposes:
- **Reference Templates**: Canonical YAML configurations used by deployment scripts and quickstart guides
- **Active Examples**: Referenced by 12+ scripts and documentation files for demonstrations
- **Testing Resources**: Used in continuous integration and system validation workflows
- **Learning Materials**: Comprehensive examples for understanding NetworkIntent specifications and system deployment

Key files include:
- `my-first-intent.yaml`: Basic NetworkIntent example used by quickstart scripts
- `test-deployment.yaml`: Complete system deployment manifest with LLM Processor and Nephio Bridge
- `test-networkintent.yaml`: Advanced E2 node deployment example for O-RAN testing

For detailed information about each file and usage instructions, see the comprehensive [archive/README.md](archive/README.md)

## ⚙️ Configuration

The Nephoran Intent Operator provides comprehensive configuration options through environment variables, enabling flexible deployment across different environments without code changes.

### Core Environment Variables

The operator supports 8 key environment variables for controlling system behavior:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ENABLE_NETWORK_INTENT` | Boolean | `true` | Enable/disable NetworkIntent controller |
| `ENABLE_LLM_INTENT` | Boolean | `false` | Enable/disable LLM Intent processing |
| `LLM_TIMEOUT_SECS` | Integer | `15` | Timeout for individual LLM requests (seconds) |
| `LLM_MAX_RETRIES` | Integer | `2` | Maximum retry attempts for LLM requests |
| `LLM_CACHE_MAX_ENTRIES` | Integer | `512` | Maximum entries in LLM cache |
| `HTTP_MAX_BODY` | Integer | `1048576` | Maximum HTTP request body size (bytes) |
| `METRICS_ENABLED` | Boolean | `false` | Enable/disable metrics endpoint |
| `METRICS_ALLOWED_IPS` | String | `""` | Comma-separated IPs allowed to access metrics |

### Quick Configuration Examples

#### Development Environment
```bash
export ENABLE_NETWORK_INTENT=true
export ENABLE_LLM_INTENT=true
export LLM_TIMEOUT_SECS=5
export METRICS_ENABLED=true
export METRICS_ALLOWED_IPS="*"  # Open access for development
```

#### Production Environment
```bash
export ENABLE_NETWORK_INTENT=true
export ENABLE_LLM_INTENT=true
export LLM_TIMEOUT_SECS=30
export LLM_MAX_RETRIES=3
export METRICS_ENABLED=true
export METRICS_ALLOWED_IPS="10.0.0.50,10.0.0.51"  # Monitoring systems only
```

### Comprehensive Configuration Documentation

For detailed information about all environment variables, including:
- Complete variable reference with examples
- Security considerations and best practices
- Troubleshooting guide and common issues
- Migration guide for version upgrades

See: **[Environment Variables Reference Guide](docs/ENVIRONMENT_VARIABLES.md)**

## 📊 Monitoring & Observability

The Nephoran Intent Operator provides comprehensive observability through Prometheus metrics, distributed tracing, and structured logging, enabling complete visibility into system performance and operational health.

### 🎯 Metrics Overview

The system exposes **11 specialized Prometheus metrics** across two main categories:

#### LLM Processing Metrics
- **`nephoran_llm_requests_total`**: Total LLM requests by model and status
- **`nephoran_llm_errors_total`**: LLM errors categorized by type and model
- **`nephoran_llm_processing_duration_seconds`**: Processing latency histograms with P95/P99 analysis
- **`nephoran_llm_cache_hits_total`** / **`nephoran_llm_cache_misses_total`**: Cache efficiency tracking
- **`nephoran_llm_fallback_attempts_total`**: Model fallback frequency monitoring
- **`nephoran_llm_retry_attempts_total`**: Request retry pattern analysis

#### Controller Operations Metrics
- **`networkintent_reconciles_total`**: Controller reconciliation success/failure rates
- **`networkintent_reconcile_errors_total`**: Error categorization for troubleshooting
- **`networkintent_processing_duration_seconds`**: Phase-by-phase processing timing
- **`networkintent_status`**: Real-time resource status (Failed=0, Processing=1, Ready=2)

### ⚡ Quick Metrics Setup

```bash
# Enable metrics collection
export METRICS_ENABLED=true

# Optional: Restrict metrics access (production recommended)
export METRICS_ALLOWED_IPS="10.0.0.50,10.0.0.51"

# Verify metrics endpoint
curl http://localhost:8080/metrics | grep nephoran_
```

### 📈 Key Performance Indicators

Monitor these essential metrics for production health:

| Metric | Ideal Range | Alert Threshold | Business Impact |
|--------|-------------|-----------------|-----------------|
| **LLM Success Rate** | > 95% | < 90% | Intent processing failures |
| **Cache Hit Rate** | > 70% | < 50% | Increased costs and latency |
| **P95 Processing Time** | < 2s | > 5s | User experience degradation |
| **Controller Error Rate** | < 5% | > 10% | Deployment failures |
| **Fallback Frequency** | < 2% | > 5% | Primary model reliability issues |

### 🔍 Common Monitoring Queries

**System Health Overview:**
```promql
# Overall system success rate
(rate(nephoran_llm_requests_total{status="success"}[5m]) + 
 rate(networkintent_reconciles_total{result="success"}[5m])) /
(rate(nephoran_llm_requests_total[5m]) + 
 rate(networkintent_reconciles_total[5m])) * 100
```

**Performance Monitoring:**
```promql
# 95th percentile end-to-end processing time
histogram_quantile(0.95, 
  rate(networkintent_processing_duration_seconds_bucket{phase="total"}[5m]))
```

**Cost Optimization:**
```promql
# Cache efficiency by model
rate(nephoran_llm_cache_hits_total[5m]) / 
(rate(nephoran_llm_cache_hits_total[5m]) + rate(nephoran_llm_cache_misses_total[5m]))
```

### 🎨 Grafana Dashboard Features

Our pre-configured dashboard provides:
- **Executive Summary**: High-level KPIs and system health status
- **LLM Performance**: Model-specific latency, error rates, and cost tracking
- **Controller Operations**: NetworkIntent lifecycle and processing efficiency
- **Troubleshooting Views**: Error categorization and debugging assistance
- **Capacity Planning**: Resource utilization trends and scaling recommendations

### 🚨 Production Alerts

Essential alerting rules for operational teams:

```yaml
# High-priority alerts for immediate attention
- alert: LLMProcessingFailures
  expr: rate(nephoran_llm_errors_total[5m]) / rate(nephoran_llm_requests_total[5m]) > 0.1
  severity: critical
  
- alert: SlowIntentProcessing  
  expr: histogram_quantile(0.95, rate(networkintent_processing_duration_seconds_bucket[5m])) > 10
  severity: warning
```

### 📋 Comprehensive Metrics Documentation

For complete metrics reference including:
- Detailed metric descriptions with example values
- Label specifications and cardinality considerations
- Performance tuning and troubleshooting guides
- Advanced Prometheus queries and alerting rules
- Grafana dashboard configuration and best practices

See: **[Complete Prometheus Metrics Documentation](PROMETHEUS_METRICS.md)**

### 📖 Advanced Topics
- **[O-RAN Compliance Certification](docs/ORAN-COMPLIANCE-CERTIFICATION.md)**: Standards compliance details
- **[Security Documentation](docs/security/README.md)**: Complete security implementation guide
  - **[OAuth2 Security Guide](docs/security/OAuth2-Security-Guide.md)**: Comprehensive OAuth2 implementation
  - **[CORS Configuration](docs/security/CORS-Security-Configuration-Guide.md)**: CORS security setup
- **[Operational Runbooks](docs/runbooks/README.md)**: Production operations and incident response
- **[Performance Optimization](docs/reports/performance-optimization.md)**: Tuning and scaling guides
- **[Multi-Region Deployment](deployments/multi-region/README.md)**: Global architecture patterns

### 🎯 Tutorials & Examples
- **[Network Slicing Guide](docs/NetworkIntent-Controller-Guide.md)**: End-to-end slice deployment with NetworkIntent
- **[xApp Development](docs/xApp-Development-SDK-Guide.md)**: Custom application integration
- **[GitOps Workflows](docs/GitOps-Package-Generation.md)**: CI/CD pipeline integration
- **[Production Examples](examples/production/)**: Real-world deployment configurations

## 🤝 Community & Contribution

### 🌟 Get Involved

We welcome contributions from telecommunications engineers, cloud-native developers, AI/ML researchers, and network operators!

[![GitHub Issues](https://img.shields.io/badge/GitHub-Issues-24292e?style=flat-square&logo=github)](https://github.com/thc1006/nephoran-intent-operator/issues)
[![GitHub Discussions](https://img.shields.io/badge/GitHub-Discussions-24292e?style=flat-square&logo=github)](https://github.com/thc1006/nephoran-intent-operator/discussions)

### 🛠️ Development Workflow

```bash
# Fork and clone
git clone https://github.com/yourusername/nephoran-intent-operator.git
cd nephoran-intent-operator

# Run comprehensive test suite
make test-all  # Unit, integration, E2E, security, and performance tests

# Build and validate
make build docker-build validate-all

# Submit PR with required checks
# ✅ All tests passing (90%+ coverage)
# ✅ Security scans clean  
# ✅ Documentation updated
# ✅ Performance benchmarks maintained
```

### 🎯 Contribution Areas

| Area | Difficulty | Impact | Examples |
|------|------------|--------|----------|
| **LLM/RAG Enhancement** | 🔴 Advanced | 🔥 High | Prompt optimization, model fine-tuning |
| **O-RAN Interface Development** | 🔴 Advanced | 🔥 High | E2AP codec implementation, xApp SDK |
| **Security Hardening** | 🟡 Intermediate | 🔥 High | mTLS automation, vulnerability scanning |
| **Performance Optimization** | 🟡 Intermediate | 🟠 Medium | Caching layers, connection pooling |
| **Documentation & Tutorials** | 🟢 Beginner | 🟠 Medium | Use cases, troubleshooting guides |
| **Testing & Quality** | 🟡 Intermediate | 🟠 Medium | Chaos engineering, load testing |

### 🏆 Recognition Program

Contributors receive recognition through:
- 🥇 **Hall of Fame**: Top contributors featured in documentation
- 🎖️ **Expert Status**: Technical advisor program for significant contributions
- 📢 **Conference Speaking**: Present at telecommunications and cloud-native events
- 💼 **Professional Network**: Connect with industry leaders and potential employers

## 🚀 Deployment Options

### Cloud Providers

#### ☁️ Public Cloud (Development/Testing)
```bash
# AWS EKS with Terraform
cd deployments/multi-region/terraform
terraform init && terraform apply

# Azure AKS with ARM templates  
az deployment group create --template-file deployments/azure/aks-cluster.json

# Google GKE with Helm
helm install nephoran deployments/helm/nephoran-operator \
  --set cloudProvider=gcp \
  --set monitoring.enabled=true
```

#### 🏢 On-Premises (Development)
```bash
# Red Hat OpenShift
oc apply -k deployments/kustomize/overlays/production/

# VMware Tanzu
kubectl apply -f deployments/kubernetes/ --recursive

# Bare Metal with kubeadm  
./scripts/deploy-production.sh --target bare-metal
```

#### 🌐 Edge/Multi-Cloud (Future)
```bash
# Edge computing deployment
./scripts/deploy-edge.sh --regions us-west,eu-central,asia-southeast

# Hybrid cloud with GitOps
kubectl apply -k deployments/kustomize/overlays/gitops/
```

### GitOps Configuration

The operator includes optimized GitOps settings for concurrent operations:

- **`GIT_CONCURRENT_PUSH_LIMIT`** (Environment Variable)
  - **Default**: 4 concurrent operations per process
  - **Behavior**: Limits the number of simultaneous `CommitAndPush` operations to prevent git repository lock contention and improve overall system stability
  - **Tuning**: Increase for high-throughput environments with robust git infrastructure; decrease for environments with limited git server resources
  
Example configuration:
```bash
# Set via environment variable
export GIT_CONCURRENT_PUSH_LIMIT=8

# Or in Kubernetes deployment
env:
  - name: GIT_CONCURRENT_PUSH_LIMIT
    value: "8"

# Or in Helm values
git:
  concurrentPushLimit: 8
```

This setting helps prevent git operation bottlenecks in high-load scenarios while maintaining data consistency.

## 📈 Roadmap & Innovation

### 🎯 Current MVP (v0.x)
- ✅ NetworkIntent CRD and controller implementation
- ✅ Basic LLM integration with GPT-4o-mini
- ✅ Optional RAG system (behind build tags)
- ✅ Kubernetes-native deployment patterns
- ✅ Prometheus metrics and observability foundation

## 🗺️ Roadmap - From MVP to Production

### 🚧 Phase 1: Core Platform (v1.0)
- 🔄 **Full O-RAN Interface Implementation**: Complete A1, O1, O2, E2 interface specifications
- 🤖 **Production GitOps**: Nephio R5 integration with Porch package orchestration
- 🌍 **Real Network Functions**: Integration with actual 5G Core and RAN components
- 🔒 **Enterprise Security**: OAuth2, mTLS, RBAC, and comprehensive audit trails

### 🏢 Phase 2: Enterprise Features (v2.0)
- 📊 **Production Observability**: 99.95% availability targets, comprehensive SLI/SLO tracking
- 🚀 **High-Scale Performance**: 200+ concurrent intents, sub-2-second processing
- 🌐 **Multi-Cloud Orchestration**: AWS, Azure, GCP, and edge deployment
- 🔧 **Service Mesh Integration**: Native Istio/Linkerd with advanced traffic management

### 🔮 Phase 3: Advanced Automation (v3.0+)
- 🧠 **ML-Driven Optimization**: Autonomous network optimization and self-healing
- 🔗 **6G Readiness**: Next-generation wireless standards integration
- 🎨 **Visual Intent Designer**: Low-code interface for network operators
- 🏭 **Industry Verticals**: Specialized templates for automotive, manufacturing, healthcare

### 🎯 Production Readiness Goals
- **Availability**: 99.95% uptime SLA
- **Performance**: Sub-2-second P95 intent processing
- **Scale**: 200+ concurrent intent operations
- **Standards**: Full O-RAN Alliance compliance certification
- **Security**: SOC 2 Type II, GDPR/CCPA compliance

## ⭐ Support & Enterprise Services

### 🆘 Community Support (Free)
- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and community help  
- **Documentation**: Comprehensive guides and tutorials
- **Stack Overflow**: Tagged questions with `nephoran-operator`

### 🏢 Enterprise Support
- **Priority Support**: 24/7 technical assistance with SLA guarantees
- **Professional Services**: Custom deployment, training, and consulting
- **Dedicated Success Manager**: Ongoing optimization and best practices
- **Custom Development**: Feature development for specific requirements

[**Contact Enterprise Sales →**](mailto:enterprise@nephoran.com)

### 🔒 Security & Compliance
- **SOC 2 Type II Certified**: Annual security audits and compliance reporting
- **GDPR/CCPA Compliant**: Data privacy and protection standards
- **NIST Framework**: Security controls aligned with cybersecurity framework
- **Supply Chain Security**: SLSA Level 3 compliant with attestation signatures

## 📜 License

Licensed under the [Apache License, Version 2.0](LICENSE). 

**Enterprise licenses** with additional features, support, and compliance certifications are available. [Contact us](mailto:licensing@nephoran.com) for details.

---

<div align="center">

**🌟 Star us on GitHub** • **🐛 Report Issues** • **💬 Discuss on GitHub** • **📖 Read Docs** • **🤝 Contribute**

*Transforming telecommunications through intelligent automation*

**[Documentation](https://thc1006.github.io/nephoran-intent-operator)** • **[Getting Started](QUICKSTART.md)** • **[API Reference](docs/API_REFERENCE.md)** • **[GitHub Issues](https://github.com/thc1006/nephoran-intent-operator/issues)**

</div>
</div>