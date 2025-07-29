# Technology Stack

## Core Technologies
- **Language**: Go 1.24+ (primary), Python 3.x (RAG components)
- **Framework**: Kubernetes controller-runtime, Kubebuilder patterns
- **Container Runtime**: Docker with multi-stage builds
- **Orchestration**: Kubernetes with Custom Resource Definitions (CRDs)

## Key Dependencies
- **Kubernetes**: v0.33.3 (client-go, apimachinery, api)
- **Controller Runtime**: sigs.k8s.io/controller-runtime v0.21.0
- **Git Operations**: github.com/go-git/go-git/v5 v5.16.2
- **Testing**: Ginkgo v2.23.4, Gomega v1.38.0
- **Python RAG**: Haystack framework, Weaviate vector database

## Build System
Uses Make-based build system with cross-platform support (Windows/Linux).

### Common Commands
```bash
# Development setup
make setup-dev          # Install dependencies and setup environment

# Building
make build-all          # Build all service binaries
make build-llm-processor    # Build individual components
make build-nephio-bridge
make build-oran-adaptor

# Code generation and quality
make generate           # Generate Kubernetes code (after API changes)
make lint              # Run linters (golangci-lint, flake8)

# Testing
make test-integration   # Run integration tests with envtest

# Docker operations
make docker-build       # Build all container images
make docker-push        # Push images to registry

# Deployment
make deploy-dev         # Deploy to development environment
./deploy.sh local       # Local deployment script
./deploy.sh remote      # Remote (GKE) deployment script

# Knowledge base
make populate-kb        # Populate Weaviate vector store
```

## Deployment Patterns
- **Kustomize**: Primary deployment tool with base/overlay pattern
- **Local**: Uses imagePullPolicy=Never, loads images directly to cluster
- **Remote**: Google Artifact Registry with proper RBAC and image pull secrets
- **GitOps**: Nephio-based GitOps workflow with separate control/deployment repos