# Nephoran Intent Operator - Project File Map

## Project Overview
The Nephoran Intent Operator is a Kubernetes operator that translates natural language intents into network function deployments using LLM processing, RAG, and O-RAN compliant orchestration.

## Core Architecture Components

### 🎯 Entry Points
```
cmd/
├── llm-processor/          # LLM service for intent processing
├── nephio-bridge/          # Bridge to Nephio orchestration
├── oran-adaptor/           # O-RAN interface adapter
├── run-envtest/            # Testing environment runner
└── security-demo/          # Security demonstration
```

### 📦 API Definitions (CRDs)
```
api/v1/
├── networkintent_types.go          # Main NetworkIntent CRD
├── networkintent_cnf_types.go      # CNF-specific extensions
├── e2nodeset_types.go               # E2 node management
├── cnfdeployment_types.go           # CNF deployment specs
├── gitopsdeployment_types.go        # GitOps workflow
├── managedelement_types.go          # O-RAN managed elements
├── o1interface_types.go             # O1 interface types
├── resourceplan_types.go            # Resource planning
├── manifestgeneration_types.go      # Manifest generation
├── intentprocessing_types.go        # Intent processing
├── disaster_recovery_types.go       # DR configurations
├── audittrail_types.go              # Audit logging
└── common/types.go                  # Shared types
```

### 🧠 Core Business Logic
```
pkg/
├── llm/                    # LLM Integration Layer
│   ├── client.go          # OpenAI/GPT client
│   ├── processor.go       # Intent processing logic
│   ├── validator.go       # Response validation
│   ├── cache.go           # Token/response caching
│   └── prompt.go          # Prompt engineering
│
├── rag/                    # RAG System (optional build tag)
│   ├── client.go          # Weaviate vector DB client
│   ├── embeddings.go      # Document embeddings
│   ├── retriever.go       # Context retrieval
│   └── noop.go            # No-op implementation for MVP
│
├── controllers/            # Kubernetes Controllers
│   ├── networkintent/     # NetworkIntent controller
│   ├── e2nodeset/         # E2NodeSet controller
│   ├── orchestration/     # Orchestration controllers
│   └── optimization/      # Performance optimization
│
├── oran/                   # O-RAN Integration
│   ├── a1/                # A1 interface (policies)
│   ├── e2/                # E2 interface (RAN control)
│   ├── o1/                # O1 interface (FCAPS)
│   └── o2/                # O2 interface (cloud infra)
│
├── nephio/                 # Nephio Integration
│   ├── porch/             # Package orchestration
│   ├── blueprint/         # Blueprint management
│   ├── krm/               # KRM functions
│   └── multicluster/      # Multi-cluster support
│
└── cnf/                    # CNF Management
    ├── templates/         # CNF templates
    └── lifecycle.go       # Lifecycle management
```

### 🚀 Deployment Configurations
```
deployments/
├── crds/                   # CRD YAML definitions
│   ├── networkintent.yaml
│   ├── e2nodeset.yaml
│   └── ...
│
├── kustomize/              # Kustomize deployments
│   ├── base/              # Base configurations
│   │   ├── llm-processor/
│   │   ├── network-intent-controller/
│   │   ├── oran-adaptor/
│   │   ├── nephio-bridge/
│   │   ├── rag-api/
│   │   ├── weaviate/
│   │   └── rbac/
│   │
│   └── overlays/          # Environment overlays
│       ├── dev/           # Development
│       ├── staging/       # Staging
│       ├── production/    # Production
│       └── local/         # Local testing
│
├── helm/                   # Helm charts
│   ├── nephoran-operator/
│   └── nephoran-performance-monitoring/
│
└── examples/               # Example deployments
    ├── a1-policy-examples/
    ├── cnf-deployments/
    └── xapps/
```

### 🧪 Testing & Quality
```
tests/
├── unit/                   # Unit tests
│   └── controllers/
├── integration/            # Integration tests
│   └── controllers/
├── e2e/                    # End-to-end tests
├── performance/            # Performance tests
│   ├── scenarios/
│   └── validation/
├── security/               # Security tests
└── chaos/                  # Chaos engineering
```

### 📚 Documentation
```
docs/
├── api/                    # API documentation
│   ├── openapi/           # OpenAPI specs
│   └── authentication/    # Auth guides
├── architecture/           # Architecture diagrams
├── crd-reference/          # CRD reference docs
├── getting-started/        # Quick start guides
├── operations/             # Operations manual
├── runbooks/               # Operational runbooks
└── production-readiness/   # Production checklist
```

### 🔧 Configuration & Scripts
```
config/                     # Kubernetes configurations
├── crd/                   # CRD definitions
├── rbac/                  # RBAC policies
├── webhook/               # Webhooks
└── samples/               # Sample resources

scripts/                    # Automation scripts
├── deployment/            # Deployment scripts
├── monitoring/            # Monitoring setup
└── security/              # Security scanning
```

### 🏗️ Build & CI/CD
```
Makefile                    # Primary build file
Makefile.deps               # Dependency management
go.mod                      # Go module definition
go.sum                      # Go module checksums
Dockerfile                  # Container image
.golangci.yml              # Linting configuration
```

## Key Integration Points

### LLM Processing Pipeline
1. **Intent Reception**: NetworkIntent CRD → Controller
2. **LLM Processing**: Controller → LLM Processor Service
3. **Context Retrieval**: LLM Processor → RAG API (optional)
4. **Response Generation**: LLM → Structured Output
5. **Deployment**: Controller → Nephio/O-RAN

### O-RAN Compliance
- **A1 Interface**: Policy management for Near-RT RIC
- **E2 Interface**: Real-time RAN control
- **O1 Interface**: FCAPS management
- **O2 Interface**: Cloud infrastructure orchestration

### Nephio Integration
- **Porch**: Package orchestration
- **ConfigSync**: GitOps deployment
- **KRM Functions**: Package manipulation
- **Multi-cluster**: Cross-cluster deployment

## MVP vs Production Features

### MVP Core (Phase 1) ✅
- Basic NetworkIntent CRD
- LLM processor with GPT-4o-mini
- Simple intent to deployment translation
- Basic validation and error handling
- Single cluster deployment
- Essential monitoring

### Production Extensions (Phase 2+) 🚧
- Full RAG system with Weaviate
- Multi-cluster orchestration
- Complete O-RAN interface suite
- Advanced security (OAuth2, mTLS)
- Comprehensive observability
- Disaster recovery
- Auto-scaling and optimization
- Service mesh integration

## Development Workflow

### Local Development
```bash
# Install dependencies
make deps

# Run tests
make test

# Build binaries
make build

# Deploy locally
make deploy-local
```

### Testing Strategy
- Unit tests: 70% (fast feedback)
- Integration tests: 20% (component validation)
- E2E tests: 10% (critical paths)

### Code Organization
- Domain-driven design with clear boundaries
- Interface-based contracts
- Minimal external dependencies
- Comprehensive error handling

## Next Steps for Contributors

1. **Quick Start**: See `docs/getting-started/`
2. **API Reference**: Check `docs/api/`
3. **Contributing**: Review `CONTRIBUTING.md`
4. **Architecture**: Study `docs/architecture/`

## Important Notes

- The project is transitioning from aggressive production claims to MVP focus
- RAG system is behind a build tag (optional for MVP)
- Keep single Makefile for simplicity
- Documentation should align with actual implementation status