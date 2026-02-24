# Package Consolidation Plan - Task #69

**Date**: 2026-02-24
**Objective**: Consolidate duplicate packages and reorganize pkg/ structure from 57 packages to ~30 well-organized packages

## Current State Analysis

### Package Count
- Total pkg/ packages: 57
- Non-test Go files: 372
- Largest packages:
  1. pkg/oran (119 files) - ✅ Already well-organized with subdirs a1/, e2/, o1/, o2/, health/
  2. pkg/nephio (96 files) - ✅ Already consolidated
  3. pkg/security (78 files)
  4. pkg/monitoring (67 files)
  5. pkg/llm (63 files)
  6. pkg/controllers (62 files)

### Identified Duplicates

#### 1. Porch Packages (HIGH PRIORITY)
- **internal/porch/** (9 files):
  - executor.go - StatefulExecutor for porch CLI execution
  - graceful_executor*.go - OS-specific graceful shutdown
  - writer.go - WriteIntent() for JSON output
  - client.go - CreateDraftPackageRevision() K8s dynamic client
  - testutil.go - Test helpers
  - cmd_*.go - OS-specific command execution

- **pkg/porch/** (5 files):
  - client.go - HTTP Client for Porch API
  - direct_client.go - DirectClient (stub implementation)
  - intent.go - Intent-to-package conversion
  - krm_builder.go - KRM package builder
  - types.go - Package types

**Usage Analysis**:
- internal/porch: Used by internal/loop/watcher.go (main pipeline)
- pkg/porch: Used by cmd/intent-ingest, cmd/porch-direct, internal/ingest

**Consolidation Strategy**:
- Move ALL to pkg/porch/ (public API)
- Structure:
  ```
  pkg/porch/
  ├── client.go           # HTTP API client (from pkg)
  ├── direct_client.go    # Direct client (from pkg)
  ├── executor.go         # CLI executor (from internal)
  ├── graceful_executor.go # Graceful shutdown (from internal)
  ├── graceful_executor_unix.go
  ├── graceful_executor_windows.go
  ├── writer.go           # JSON writer (from internal)
  ├── k8s_client.go       # K8s dynamic client (from internal/client.go)
  ├── intent.go           # Intent conversion (from pkg)
  ├── krm_builder.go      # KRM builder (from pkg)
  ├── types.go            # Shared types
  ├── cmd_unix.go         # OS-specific (from internal)
  ├── cmd_windows.go      # OS-specific (from internal)
  ├── testutil.go         # Test utilities (from internal)
  └── internal/           # True private utilities if needed
  ```

#### 2. LLM Packages (MEDIUM PRIORITY)
- **internal/llm/providers/** (8 files):
  - interface.go - Provider interface
  - factory.go - Provider factory
  - offline.go, openai.go, anthropic.go, mock.go

- **pkg/llm/** (63 files - BLOATED):
  - Many overlapping/duplicate files
  - client*.go, interface*.go (multiple versions)
  - Lots of disabled/stub files

**Usage Analysis**:
- internal/llm/providers: NOT USED (0 imports found)
- pkg/llm: Used by 34 files

**Consolidation Strategy**:
- Keep pkg/llm/ as main package
- Move internal/llm/providers → pkg/llm/providers/
- Clean up duplicates in pkg/llm:
  ```
  pkg/llm/
  ├── client.go               # Main client (consolidated)
  ├── interface.go            # Main interface
  ├── types.go                # Shared types
  ├── cache.go                # Caching logic
  ├── processing.go           # Request processing
  ├── metrics.go              # Prometheus metrics
  ├── security_validator.go   # Security validation
  ├── providers/              # Provider implementations
  │   ├── interface.go
  │   ├── factory.go
  │   ├── offline.go
  │   ├── openai.go
  │   ├── anthropic.go
  │   └── mock.go
  ├── examples/               # Usage examples
  └── mocks/                  # Test mocks
  ```

  **Files to DELETE** (duplicates/obsolete):
  - client_consolidated.go, client_disabled.go, client_test_consolidated.go
  - interface_consolidated.go, interface_disabled.go
  - types_consolidated.go, common_types.go, disabled_types.go, missing_types.go
  - integration_test_consolidated.go, integration_test_stub.go
  - clean_stubs.go, rag_disabled_stubs.go
  - All *_example.go files (merge into examples/)

## Package Reorganization Plan

### Categories to Create/Maintain

1. **pkg/oran/** - ✅ Already well-organized (a1/, e2/, o1/, o2/, health/)
2. **pkg/nephio/** - ✅ Already well-organized
3. **pkg/security/** - Consolidate auth, crypto, validation
4. **pkg/monitoring/** - Consolidate metrics, alerting, observability
5. **pkg/testutil/** - Consolidate testing, testutils, testcontainers

### Proposed Final Structure (~30 packages)

```
pkg/
├── audit/                  # Audit trail and compliance
├── auth/                   # Authentication (merge into security if small)
├── automation/             # Automation workflows
├── blueprints/             # Blueprint management
├── chaos/                  # Chaos engineering
├── clients/                # External client libraries
├── cnf/                    # Cloud-native network functions
├── config/                 # Configuration management
├── context/                # Request context handling
├── contracts/              # API contracts
├── controllers/            # Kubernetes controllers
├── disaster/               # Disaster recovery
├── edge/                   # Edge computing
├── errors/                 # Error types and handling
├── git/                    # Git integration
├── handlers/               # HTTP/API handlers
├── health/                 # Health checks (merge into monitoring?)
├── injection/              # Dependency injection
├── interfaces/             # Shared interfaces
├── knowledge/              # Knowledge base
├── llm/                    # ✅ LLM integration (consolidated)
├── logging/                # Structured logging
├── metrics/                # Metrics (merge into monitoring?)
├── middleware/             # HTTP middleware
├── ml/                     # Machine learning utilities
├── models/                 # Data models
├── monitoring/             # ✅ Observability suite (metrics, alerts, traces)
├── multicluster/           # Multi-cluster management
├── mvp/                    # MVP utilities
├── nephio/                 # ✅ Nephio/Porch/KPT integration
├── optimization/           # Performance optimization
├── oran/                   # ✅ O-RAN interfaces (a1, e2, o1, o2)
├── packagerevision/        # Package revision management
├── performance/            # Performance testing
├── porch/                  # ✅ Porch API client (consolidated)
├── rag/                    # RAG (Retrieval-Augmented Generation)
├── recovery/               # Recovery mechanisms (merge into disaster?)
├── resilience/             # Resilience patterns
├── runtime/                # Runtime utilities
├── security/               # ✅ Security suite (auth, crypto, RBAC, validation)
├── servicemesh/            # Service mesh integration
├── services/               # Business logic services
├── shared/                 # Shared utilities
├── telecom/                # Telecom-specific logic
├── templates/              # Template engine
├── testcontainers/         # Testcontainers support (merge into testutil?)
├── testutil/               # ✅ Testing utilities (consolidated)
├── testing/                # Testing framework (merge into testutil?)
├── testutils/              # Test utilities (merge into testutil?)
├── types/                  # Common types
├── validation/             # Validation logic (merge into security?)
├── webhooks/               # Kubernetes webhooks
└── webui/                  # Web UI backend
```

**Consolidation Opportunities**:
- testing + testutils + testcontainers → testutil/
- metrics + health → monitoring/
- auth + validation → security/
- recovery → disaster/

**Target**: 30-35 packages (down from 57)

## Execution Plan

### Phase 1: Consolidate Duplicates (HIGH PRIORITY)
1. ✅ Consolidate internal/porch + pkg/porch → pkg/porch/
2. ✅ Consolidate internal/llm + pkg/llm → pkg/llm/

### Phase 2: Merge Related Packages (MEDIUM PRIORITY)
3. Merge testing/ + testutils/ + testcontainers/ → testutil/
4. Merge metrics/ + health/ → monitoring/
5. Merge auth/ + validation/ → security/
6. Merge recovery/ → disaster/

### Phase 3: Clean Up (LOW PRIORITY)
7. Remove obsolete/duplicate files in pkg/llm/
8. Update all import statements
9. Run full test suite
10. Update documentation

## Success Criteria

- [ ] No duplicate packages (internal/porch, internal/llm removed)
- [ ] pkg/ has 30-35 packages (down from 57)
- [ ] All tests passing: `go test ./...`
- [ ] All imports updated
- [ ] No circular dependencies
- [ ] Documentation updated (README.md, PROGRESS.md)
- [ ] Task #69 marked complete

## Rollback Plan

If consolidation breaks critical functionality:
1. Git revert to pre-consolidation state
2. Identify breaking changes
3. Create incremental consolidation plan
4. Re-test each step

## Notes

- Use `git grep` to find all import statements before moving
- Use `gofmt -w` to fix imports after moving
- Use `go mod tidy` to clean dependencies
- Test after each major move, not all at once
- Maintain backward compatibility with type aliases if needed
