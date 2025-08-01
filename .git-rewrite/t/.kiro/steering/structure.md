# Project Structure

## Directory Organization

### Core Application Code
- **`cmd/`** - Application entry points (main.go files)
  - `llm-processor/` - LLM intent processing service
  - `nephio-bridge/` - Core Nephio bridge controller
  - `oran-adaptor/` - O-RAN interface adaptors

- **`pkg/`** - Reusable Go packages
  - `controllers/` - Kubernetes controllers (reconciliation logic)
  - `rag/` - RAG pipeline implementation (Python)
  - `oran/` - O-RAN interface bridges
  - `git/` - Git operations for GitOps workflow
  - `llm/` - LLM integration utilities
  - `config/` - Configuration management

- **`api/v1/`** - Kubernetes API definitions
  - Custom Resource Definitions (CRDs)
  - Type definitions for NetworkIntent, E2NodeSet, ManagedElement

### Deployment and Configuration
- **`deployments/`** - All deployment configurations
  - `crds/` - Custom Resource Definition YAML files
  - `kubernetes/` - Raw Kubernetes manifests
  - `kustomize/` - Kustomize base and overlays (preferred)

- **`kpt-packages/`** - Nephio-compatible KPT packages
  - `nephio/` - Core Nephio packages
  - `ric-plt/` - RIC platform packages
  - `ric-sim/` - E2 node simulator packages

- **`config/samples/`** - Example configurations and intents

### Development and Documentation
- **`scripts/`** - Automation and utility scripts
- **`docs/`** - Technical documentation
- **`examples/`** - Usage examples
- **`knowledge_base/`** - RAG knowledge base content (3GPP specs, O-RAN docs)

## Naming Conventions
- **Go packages**: lowercase, descriptive names
- **CRD types**: PascalCase (NetworkIntent, E2NodeSet)
- **YAML files**: kebab-case with descriptive suffixes
- **Container images**: kebab-case matching service names

## Architecture Patterns
- **Controller Pattern**: Standard Kubernetes controller reconciliation loops
- **GitOps**: Separate repositories for control intents vs deployment manifests  
- **Layered Architecture**: Interface → Processing → Control → Orchestration
- **Event-Driven**: Controllers react to CR changes and Git repository updates