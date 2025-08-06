# Nephoran Intent Operator

**Experimental Proof-of-Concept for Intent-Driven Network Operations**

The Nephoran Intent Operator is an experimental proof-of-concept that explores the integration of Large Language Models (LLMs) with Kubernetes operators for network function orchestration. This project investigates the potential of translating natural language intents into Kubernetes configurations for O-RAN compliant network functions.

## Project Status

**Current Stage: Development/Proof-of-Concept**

- **Experimental**: This is research and development work, not a production system
- **Early Stage**: Basic intent processing capabilities with simple LLM integration
- **O-RAN Exploration**: Early-stage implementations of O-RAN interface concepts
- **GitOps Integration**: Experimental integration with Nephio package management

## What Actually Works

### Core Components
- **NetworkIntent Controller**: Basic Kubernetes controller that processes custom resources
- **E2NodeSet Controller**: Simple replica management for simulated E2 nodes (using ConfigMaps)
- **LLM Processor Service**: Experimental service that calls OpenAI API for intent translation
- **RAG API**: Basic Flask API for document retrieval (proof-of-concept)

### Basic Functionality
- Create NetworkIntent resources with natural language text
- Simple LLM processing to extract basic parameters
- E2NodeSet scaling through Kubernetes controllers
- Basic health checks and monitoring endpoints

## What's Planned vs Reality

### Planned Features (Not Yet Implemented)
- Production-ready enterprise deployment
- Advanced ML optimization engines
- Complete O-RAN interface implementations
- High-availability architecture
- Comprehensive security frameworks

### Limitations
- No production-ready security implementation
- Limited error handling and resilience
- Basic LLM integration without advanced RAG
- Experimental O-RAN interfaces (not fully compliant)
- No performance guarantees or SLA targets

## Quick Start (Development)

### Prerequisites
- Go 1.23+
- Docker and Kubernetes (kind or minikube)
- OpenAI API key
- Python 3.8+ (for RAG components)

### Basic Setup

1. **Clone and setup environment**:
   ```bash
   git clone <repository-url>
   cd nephoran-intent-operator
   
   # Set OpenAI API key
   export OPENAI_API_KEY="sk-your-api-key-here"
   ```

2. **Build components**:
   ```bash
   # Build all services
   make build-all
   
   # Build Docker images
   make docker-build
   ```

3. **Deploy locally**:
   ```bash
   # Deploy to local Kubernetes cluster
   ./deploy.sh local
   ```

4. **Test basic functionality**:
   ```bash
   # Create a simple intent
   kubectl apply -f - <<EOF
   apiVersion: nephoran.com/v1alpha1
   kind: NetworkIntent
   metadata:
     name: test-intent
     namespace: default
   spec:
     intent: "Deploy a simple AMF instance"
   EOF
   
   # Check status
   kubectl get networkintents
   kubectl describe networkintent test-intent
   ```

## Development Environment

### Build System
```bash
# Build all components
make build-all

# Run tests
make test

# Check code quality
make lint

# Clean artifacts
make clean
```

### Component Development
```bash
# Build individual services
make build-llm-processor    # LLM processing service
make build-nephio-bridge    # Main controller
make build-oran-adaptor     # O-RAN interface adapters
make build-rag-api         # Document retrieval API
```

### Testing
```bash
# Run unit tests
go test ./...

# Validate CRDs
kubectl apply --dry-run=client -f config/crd/bases/

# Check pod health
kubectl get pods -l app.kubernetes.io/part-of=nephoran
```

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   NetworkIntent │    │  LLM Processor   │    │   RAG API       │
│   Custom Resource│───▶│  Service         │───▶│   (Flask)       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Nephio Bridge │    │   E2NodeSet      │    │   ConfigMaps    │
│   Controller    │    │   Controller     │───▶│   (Simulation)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

The system processes natural language intents through a simple pipeline:
1. User creates NetworkIntent resource
2. Controller calls LLM Processor service  
3. LLM translates intent to basic parameters
4. Controllers create Kubernetes resources
5. Basic monitoring through health endpoints

## Monitoring and Debugging

```bash
# Check component logs
kubectl logs deployment/llm-processor
kubectl logs deployment/nephio-bridge
kubectl logs deployment/rag-api

# Health checks
kubectl port-forward svc/llm-processor 8080:8080
curl http://localhost:8080/healthz

kubectl port-forward svc/rag-api 5001:5001
curl http://localhost:5001/healthz

# Check resources
kubectl get networkintents
kubectl get e2nodesets
kubectl get configmaps -l e2nodeset
```

## Configuration

### Environment Variables
```bash
# Required
export OPENAI_API_KEY="sk-your-key"

# Optional
export LOG_LEVEL="debug"
export ENABLE_RAG="true"
export RAG_ENDPOINT="http://rag-api:5001"
```

### Kubernetes Configuration
Basic authentication and CORS can be configured through environment variables or ConfigMaps (see deployment manifests).

## Contributing

This is an experimental project exploring new concepts. Contributions are welcome, but please understand this is research-stage work:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## Known Issues

- Limited error handling in LLM processing
- Basic retry logic without circuit breakers  
- Simple health checks without comprehensive monitoring
- Experimental O-RAN interfaces (not production-ready)
- No security hardening for production use

## Roadmap

**Near-term**:
- Improve error handling and resilience
- Add more comprehensive testing
- Better documentation of actual capabilities
- Enhanced LLM prompt engineering

**Medium-term**:
- Production security implementation
- Real O-RAN interface compliance
- Performance optimization
- Multi-cluster deployment support

**Long-term**:
- Complete enterprise feature set
- Advanced ML optimization
- Full O-RAN ecosystem integration
- Production deployment guides

## License

This project is experimental and provided as-is for research and development purposes.

---

**Disclaimer**: This is experimental software not intended for production use. The project explores concepts in intent-driven network operations and LLM integration with Kubernetes.