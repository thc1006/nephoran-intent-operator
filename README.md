# Nephoran Intent Operator

**Project Status: Development/Proof of Concept**

The Nephoran Intent Operator is a research and development project exploring how Large Language Models (LLMs) can be integrated with Kubernetes controllers to translate natural language network operation intents into infrastructure deployments. This is experimental software designed to investigate the feasibility of AI-driven network orchestration concepts.

## Current Reality

**This is NOT production-ready software.** It's an educational and research project with basic proof-of-concept implementations.

### What Actually Exists and Works
- **Basic Kubernetes Controllers**: Simple controllers that manage custom resources (NetworkIntent, E2NodeSet)
- **LLM API Integration**: Experimental OpenAI API calls to process natural language text
- **Simple RAG API**: Basic Python Flask service for document retrieval (very basic implementation)
- **Kubernetes CRDs**: Custom Resource Definitions for NetworkIntent and E2NodeSet resources
- **Docker Containerization**: Basic containerized deployment setup

### Current Capabilities (Limited)
- Accept natural language intents through Kubernetes resources
- Call OpenAI API to process intent text
- Create and manage ConfigMaps to simulate E2 node deployments
- Basic HTTP health check endpoints
- Simple logging and basic metrics

### Major Limitations
- **No Real Network Function Deployment**: Despite extensive documentation, no actual 5G/O-RAN network functions are deployed
- **No Production Security**: Basic authentication at best, not enterprise-ready
- **No Real O-RAN Compliance**: Simulated interfaces only, not standards-compliant
- **No Advanced AI/ML**: Simple API calls to OpenAI, no sophisticated RAG or optimization
- **No High Availability**: Single-instance deployments with basic error handling
- **No Performance Optimization**: No benchmarking, scaling, or performance tuning
- **Extensive Documentation for Minimal Code**: Documentation far exceeds actual implementation

## Quick Start for Developers

**Warning**: This is experimental software. Only attempt if you want to explore concepts or contribute to development.

### Prerequisites
- Go 1.23+
- Docker and Kubernetes cluster (kind, minikube, or similar)
- OpenAI API key (required for LLM integration)
- Python 3.8+ (for basic RAG API)

### Simple Local Setup

1. **Clone and basic setup**:
   ```bash
   git clone https://github.com/your-repo/nephoran-intent-operator
   cd nephoran-intent-operator
   
   # Required: Set OpenAI API key
   export OPENAI_API_KEY="sk-your-actual-openai-key"
   ```

2. **Build core components** (what actually exists):
   ```bash
   # Build the Go services
   make build
   
   # Build basic Docker images
   make docker-build
   ```

3. **Deploy to local cluster**:
   ```bash
   # Install CRDs and basic controllers
   kubectl apply -f deployments/crds/
   kubectl apply -f deployments/kubernetes/
   ```

4. **Test basic functionality** (limited):
   ```bash
   # Create a test NetworkIntent resource
   kubectl apply -f - <<EOF
   apiVersion: nephoran.com/v1alpha1
   kind: NetworkIntent
   metadata:
     name: test-intent
     namespace: default
   spec:
     intent: "Create a test deployment"
   EOF
   
   # Check if controller processes it
   kubectl get networkintents
   kubectl logs deployment/nephio-bridge
   ```

## What You Can Actually Do

### Experiment With
- **Basic Intent Processing**: Submit natural language text and see OpenAI API responses
- **Kubernetes Controller Patterns**: Study how custom controllers work
- **CRD Management**: Create and modify custom resources
- **Simple RAG Concepts**: Basic document retrieval experiments

### Development Tasks
```bash
# Run unit tests (basic coverage)
go test ./pkg/controllers/...

# Build individual components
go build -o bin/llm-processor ./cmd/llm-processor
go build -o bin/nephio-bridge ./cmd/nephio-bridge

# Check basic functionality
kubectl port-forward svc/llm-processor 8080:8080
curl http://localhost:8080/healthz
```

### Realistic Debugging
```bash
# Check if pods are running
kubectl get pods

# View controller logs
kubectl logs deployment/nephio-bridge
kubectl logs deployment/llm-processor

# Check resource status
kubectl get networkintents -o yaml
kubectl get e2nodesets -o yaml
```

## Actual Architecture (Simplified)

```
User Input
    │
    ▼
┌─────────────────────┐    HTTP Call    ┌─────────────────────┐
│  NetworkIntent      │──────────────▶ │  LLM Processor      │
│  Kubernetes CRD     │                │  (OpenAI API calls) │
└─────────────────────┘                └─────────────────────┘
    │                                           │
    │ Controller                               │ Response
    │ Watches                                   │
    ▼                                          ▼
┌─────────────────────┐                ┌─────────────────────┐
│  Nephio Bridge      │                │  Basic RAG API      │
│  Controller         │                │  (Flask/Python)     │
└─────────────────────┘                └─────────────────────┘
    │
    │ Creates
    ▼
┌─────────────────────┐
│  ConfigMaps         │
│  (Simulates nodes)  │
└─────────────────────┘
```

**What Actually Happens**:
1. User creates NetworkIntent with natural language text
2. Kubernetes controller detects the resource
3. Controller makes HTTP call to LLM Processor service
4. LLM Processor calls OpenAI API to process text
5. Simple response is returned (basic parameter extraction)
6. Controller creates ConfigMaps to simulate deployments
7. Status is updated in the NetworkIntent resource

**That's it.** No actual network functions are deployed.

## Basic Configuration

### Required Environment Variables
```bash
# Only this is actually required
export OPENAI_API_KEY="sk-your-openai-key"

# Optional debugging
export LOG_LEVEL="debug"
```

### Simple Kubernetes Setup
Most of the complex configurations in the `/deployments` directory are aspirational. For basic functionality:

```bash
# Apply CRDs
kubectl apply -f deployments/crds/nephoran.com_networkintents.yaml
kubectl apply -f deployments/crds/nephoran.com_e2nodesets.yaml

# Deploy basic controllers (if they work)
kubectl apply -f deployments/kubernetes/nephio-bridge-deployment.yaml
```

## Contributing to Development

This is an experimental research project. Contributions are welcome if you're interested in:
- Kubernetes controller development patterns
- LLM integration with infrastructure systems
- O-RAN/5G concepts and simulations
- Educational projects in cloud-native networking

### How to Contribute
1. **Understand the scope**: This is proof-of-concept work, not production software
2. **Check existing issues**: Look for areas that need actual implementation
3. **Start small**: Focus on basic functionality improvements
4. **Test your changes**: Run `go test ./...` and validate with local Kubernetes
5. **Document honestly**: Don't inflate capabilities in documentation

### Realistic Development Areas
- Improve error handling in controllers
- Add more comprehensive unit tests
- Better LLM prompt engineering for intent parsing
- Enhanced documentation of actual vs. planned features
- Bug fixes in existing basic functionality

## Current Issues (Honest Assessment)

### Technical Debt
- **Over-engineered documentation** with minimal implementation
- **Complex deployment configurations** for simple functionality
- **Many placeholder components** that don't work
- **Inconsistent error handling** across services
- **No production readiness** despite extensive production documentation

### What Needs Work
- The RAG API is very basic and may not work reliably
- LLM integration has minimal error handling
- O-RAN interfaces are mostly stubs and simulations
- No real security implementation
- Most "enterprise features" are documentation-only

## Development Roadmap (Realistic)

### Phase 1: Make Basic Features Reliable
- Fix any broken controller logic
- Improve error handling and logging
- Add proper unit tests for core functionality
- Document what actually works vs. what's planned

### Phase 2: Expand Core Functionality  
- Implement more sophisticated intent parsing
- Add basic security hardening
- Create working examples of common use cases
- Improve documentation accuracy

### Phase 3: Add Production Features (If Needed)
- Real authentication and authorization
- Actual network function integration
- Performance optimization
- Standards compliance

## Important Disclaimers

- **Not Production Ready**: This is experimental research code
- **Educational Purpose**: Best used for learning about Kubernetes operators and LLM integration
- **Limited Scope**: Despite extensive documentation, actual capabilities are basic
- **No Support**: This is a development project without production support
- **No Warranty**: Use at your own risk for educational/research purposes only

## License

This project is experimental and provided as-is for educational and research purposes.

---

**Final Note**: This project demonstrates concepts in AI-driven infrastructure orchestration but should not be used in production environments. The extensive documentation serves as a design exploration rather than current implementation status.