# O-RAN Integration Phase 1 - Detailed Implementation Tasks

**Phase:** CNF Lifecycle Foundation (Weeks 1-2)
**Goal:** Enable basic CNF deployment via Nephio/Porch
**Total Effort:** 80 hours
**Start Date:** 2026-02-16

---

## Overview

Phase 1 establishes the foundational CNF lifecycle management capabilities, enabling the Nephoran Intent Operator to deploy actual Cloud Native Functions (CNFs) through Nephio/Porch integration. This phase creates the core building blocks for all subsequent O-RAN integration work.

### Success Criteria
- ✅ Deploy AMF CNF from NetworkIntent CR
- ✅ Package visible in Porch repository
- ✅ Status updates in real-time (within 5s)
- ✅ Rollback on deployment failure
- ✅ Test coverage >80%

---

## Task 1.1: Implement CNFLifecycleManager Core

**Priority:** P0 (Critical Path)
**Effort:** 20 hours
**Dependencies:** None
**Owner:** Backend Architect
**Branch:** `feat/phase1-cnf-lifecycle-manager`

### Description

Create the central CNFLifecycleManager component that orchestrates all CNF lifecycle operations. This manager will coordinate between Nephio/Porch, A1 policies, E2 subscriptions, and Kubernetes state.

### Files to Create

```
pkg/cnf/lifecycle_manager.go          # Main lifecycle manager
pkg/cnf/state_machine.go               # State machine logic
pkg/cnf/state_store.go                 # State persistence
pkg/cnf/types.go                       # Common types
pkg/cnf/errors.go                      # Error definitions
pkg/cnf/lifecycle_manager_test.go     # Unit tests
```

### Interface Definition

```go
// pkg/cnf/lifecycle_manager.go
package cnf

import (
    "context"
    "time"

    nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
    "github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

// CNFLifecycleManager orchestrates CNF lifecycle operations
type CNFLifecycleManager struct {
    client          client.Client
    porchClient     *porch.Client
    e2Manager       *e2.E2Manager
    a1Client        *a1.A1Adaptor
    stateStore      *CNFStateStore
    eventRecorder   record.EventRecorder
}

// Core lifecycle operations
func (m *CNFLifecycleManager) Deploy(ctx context.Context, cnf *nephoranv1.CNFDeployment) error
func (m *CNFLifecycleManager) Scale(ctx context.Context, name string, replicas int32) error
func (m *CNFLifecycleManager) Update(ctx context.Context, cnf *nephoranv1.CNFDeployment) error
func (m *CNFLifecycleManager) Delete(ctx context.Context, name string) error
func (m *CNFLifecycleManager) GetStatus(ctx context.Context, name string) (*CNFStatus, error)
func (m *CNFLifecycleManager) Reconcile(ctx context.Context, cnf *nephoranv1.CNFDeployment) error
```

### State Machine

```
States:
- Pending       → Initial state after creation
- Provisioning  → Creating Nephio package, submitting to Porch
- Configuring   → Setting up A1 policies, E2 subscriptions
- Active        → CNF running, healthy
- Scaling       → Replica count changing
- Updating      → Configuration or version update in progress
- Degraded      → Running but unhealthy
- Healing       → Auto-remediation in progress
- Terminating   → Deletion in progress
- Failed        → Unrecoverable error
- Deleted       → Successfully removed

Transitions:
Pending → Provisioning → Configuring → Active
Active → Scaling → Active
Active → Updating → Active
Active → Degraded → Healing → Active
Active → Terminating → Deleted
Any → Failed (on unrecoverable error)
```

### Acceptance Criteria

- [ ] CNFLifecycleManager struct implements all core operations
- [ ] State machine transitions follow defined flow
- [ ] Deploy operation creates package in Porch
- [ ] Scale operation updates replica count
- [ ] Delete operation cleans up all resources
- [ ] Errors properly categorized (Transient, Permanent, Configuration)
- [ ] All operations emit Kubernetes events
- [ ] Unit tests cover success + 2 failure scenarios per operation
- [ ] Test coverage >85%

### Test Requirements

```go
// pkg/cnf/lifecycle_manager_test.go
func TestCNFLifecycleManager_Deploy_Success(t *testing.T)
func TestCNFLifecycleManager_Deploy_PorchFailure(t *testing.T)
func TestCNFLifecycleManager_Deploy_InvalidSpec(t *testing.T)
func TestCNFLifecycleManager_Scale_Success(t *testing.T)
func TestCNFLifecycleManager_Scale_ExceedsMax(t *testing.T)
func TestCNFLifecycleManager_Delete_WithGracefulShutdown(t *testing.T)
func TestCNFLifecycleManager_GetStatus_NotFound(t *testing.T)
func TestStateMachine_Transitions(t *testing.T)
```

### Test Commands

```bash
# Unit tests
go test ./pkg/cnf/... -v -run TestCNFLifecycleManager

# With coverage
go test ./pkg/cnf/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Race detection
go test ./pkg/cnf/... -race -v
```

---

## Task 1.2: Create Nephio PorchClient

**Priority:** P0 (Critical Path)
**Effort:** 16 hours
**Dependencies:** Task 1.1 (interfaces)
**Owner:** Backend Architect
**Branch:** `feat/phase1-porch-client`

### Description

Enhance the existing Porch client (`pkg/nephio/porch/client.go`) to support CNF package lifecycle operations. Add package template generation, CRUD operations, and status polling.

### Files to Modify/Create

```
pkg/nephio/porch/client.go              # Enhance existing client
pkg/nephio/porch/package_builder.go     # NEW: Package template builder
pkg/nephio/porch/status_poller.go       # NEW: Status polling logic
pkg/nephio/porch/templates/             # NEW: Package templates
pkg/nephio/porch/client_cnf_test.go     # NEW: CNF-specific tests
```

### Enhanced Interface

```go
// pkg/nephio/porch/package_builder.go
package porch

import (
    nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// PackageSpec defines CNF package specification
type PackageSpec struct {
    Name            string
    Namespace       string
    CNFType         nephoranv1.CNFType
    Function        nephoranv1.CNFFunction
    Replicas        int32
    Resources       nephoranv1.ResourceRequirements
    Configuration   map[string]interface{}
    Dependencies    []string
    NetworkConfig   *NetworkConfig
}

// NetworkConfig defines CNF networking
type NetworkConfig struct {
    Interfaces      []NetworkInterface
    ServiceType     string
    ExposePorts     []int32
}

// PackageBuilder creates Nephio packages from CNF specs
type PackageBuilder struct {
    templateDir string
}

func (b *PackageBuilder) BuildPackage(spec *PackageSpec) (*Package, error)
func (b *PackageBuilder) GenerateKptfile(spec *PackageSpec) ([]byte, error)
func (b *PackageBuilder) GenerateDeploymentManifest(spec *PackageSpec) ([]byte, error)
func (b *PackageBuilder) GenerateServiceManifest(spec *PackageSpec) ([]byte, error)
func (b *PackageBuilder) GenerateConfigMap(spec *PackageSpec) ([]byte, error)
```

### Package Template Structure

```
package/{name}/
├── Kptfile                          # Package metadata
├── deployment.yaml                  # Deployment manifest
├── service.yaml                     # Service definition
├── configmap.yaml                   # Configuration
├── network-attachment-def.yaml      # Multus CNI (if needed)
└── function-config.yaml             # CNF-specific config
```

### Acceptance Criteria

- [ ] CreatePackage creates package in Porch repository
- [ ] Package contains all required files (Kptfile, manifests)
- [ ] UpdatePackage modifies existing package
- [ ] DeletePackage removes package and cleans up
- [ ] GetPackageStatus polls until completion or timeout
- [ ] Authentication works (mTLS/token)
- [ ] Retry logic handles transient failures (3 retries with backoff)
- [ ] Connection pooling implemented
- [ ] Integration tests with mock Porch server
- [ ] Test coverage >80%

### Test Requirements

```go
// pkg/nephio/porch/client_cnf_test.go
func TestPackageBuilder_BuildPackage_AMF(t *testing.T)
func TestPackageBuilder_BuildPackage_UPF(t *testing.T)
func TestClient_CreatePackage_Success(t *testing.T)
func TestClient_CreatePackage_AlreadyExists(t *testing.T)
func TestClient_UpdatePackage_NotFound(t *testing.T)
func TestClient_GetPackageStatus_Polling(t *testing.T)
func TestClient_DeletePackage_GracefulCleanup(t *testing.T)
```

### Test Commands

```bash
# Unit tests
go test ./pkg/nephio/porch/... -v -run TestPackageBuilder
go test ./pkg/nephio/porch/... -v -run TestClient

# Integration test with mock Porch
go test ./pkg/nephio/porch/... -v -run TestPorchIntegration -tags=integration

# End-to-end with real Porch (requires cluster)
PORCH_URL=http://localhost:7007 go test ./pkg/nephio/porch/... -v -run TestE2E -tags=e2e
```

---

## Task 1.3: Enhance CNFDeploymentController

**Priority:** P0 (Critical Path)
**Effort:** 20 hours
**Dependencies:** Task 1.1, Task 1.2
**Owner:** Backend Architect
**Branch:** `feat/phase1-cnfdeployment-controller`

### Description

Wire the CNFLifecycleManager into the existing CNFDeployment controller reconcile loop. Implement proper status updates, event recording, and error handling.

### Files to Modify/Create

```
pkg/controllers/cnfdeployment_controller.go       # Enhance existing controller
pkg/controllers/cnfdeployment_status.go           # NEW: Status update logic
pkg/controllers/cnfdeployment_finalizer.go        # NEW: Finalizer handling
pkg/controllers/cnfdeployment_controller_test.go  # Enhance tests
```

### Controller Enhancement

```go
// pkg/controllers/cnfdeployment_controller.go
type CNFDeploymentReconciler struct {
    client.Client
    Scheme            *runtime.Scheme
    LifecycleManager  *cnf.CNFLifecycleManager
    Recorder          record.EventRecorder
}

func (r *CNFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // 1. Fetch CNFDeployment CR
    var cnfDep nephoranv1.CNFDeployment
    if err := r.Get(ctx, req.NamespacedName, &cnfDep); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Handle deletion (finalizer)
    if !cnfDep.DeletionTimestamp.IsZero() {
        return r.handleDeletion(ctx, &cnfDep)
    }

    // 3. Add finalizer if not present
    if !controllerutil.ContainsFinalizer(&cnfDep, cnf.CNFOrchestratorFinalizer) {
        controllerutil.AddFinalizer(&cnfDep, cnf.CNFOrchestratorFinalizer)
        return ctrl.Result{}, r.Update(ctx, &cnfDep)
    }

    // 4. Delegate to lifecycle manager
    if err := r.LifecycleManager.Reconcile(ctx, &cnfDep); err != nil {
        r.Recorder.Event(&cnfDep, "Warning", "ReconcileFailed", err.Error())
        return ctrl.Result{RequeueAfter: 30 * time.Second}, err
    }

    // 5. Update status
    if err := r.updateStatus(ctx, &cnfDep); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

### Status Update Logic

```go
// pkg/controllers/cnfdeployment_status.go
func (r *CNFDeploymentReconciler) updateStatus(ctx context.Context, cnf *nephoranv1.CNFDeployment) error {
    // Get current status from lifecycle manager
    status, err := r.LifecycleManager.GetStatus(ctx, cnf.Name)
    if err != nil {
        return err
    }

    // Update status subresource
    cnf.Status.Phase = status.Phase
    cnf.Status.Conditions = status.Conditions
    cnf.Status.Replicas = status.Replicas
    cnf.Status.ReadyReplicas = status.ReadyReplicas
    cnf.Status.PackageRevision = status.PackageRevision
    cnf.Status.LastUpdated = metav1.Now()

    return r.Status().Update(ctx, cnf)
}
```

### Acceptance Criteria

- [ ] Controller uses CNFLifecycleManager for all operations
- [ ] Status subresource updates reflect actual state
- [ ] Events emitted for major lifecycle changes
- [ ] Finalizer properly handles deletion
- [ ] Reconciliation handles errors with exponential backoff
- [ ] No reconcile loops (stable state doesn't re-queue)
- [ ] Controller tests use envtest
- [ ] Test coverage >80%

### Test Requirements

```go
// pkg/controllers/cnfdeployment_controller_test.go
func TestReconcile_NewCNFDeployment(t *testing.T)
func TestReconcile_UpdateReplicas(t *testing.T)
func TestReconcile_Deletion(t *testing.T)
func TestReconcile_StatusUpdate(t *testing.T)
func TestReconcile_RecoveryAfterError(t *testing.T)
func TestFinalizer_AddedAutomatically(t *testing.T)
func TestFinalizer_BlocksDeletion(t *testing.T)
```

### Test Commands

```bash
# Controller tests with envtest
make test-controllers

# Or directly
go test ./pkg/controllers/... -v -run TestReconcile

# E2E test script
./hack/test-cnf-deployment.sh
```

---

## Task 1.4: CNF State Persistence

**Priority:** P1 (High)
**Effort:** 12 hours
**Dependencies:** Task 1.1
**Owner:** Backend Architect
**Branch:** `feat/phase1-state-persistence`

### Description

Implement persistent state storage for CNF lifecycle state to survive controller restarts. Use ConfigMaps for state storage with optional etcd backend for production.

### Files to Create

```
pkg/cnf/state_store.go              # State storage interface
pkg/cnf/state_store_configmap.go    # ConfigMap implementation
pkg/cnf/state_store_etcd.go         # Etcd implementation (optional)
pkg/cnf/state_store_test.go         # Unit tests
```

### Interface Definition

```go
// pkg/cnf/state_store.go
package cnf

import (
    "context"
    "time"
)

// CNFState represents persisted CNF state
type CNFState struct {
    Name              string
    Namespace         string
    Phase             string
    PackageRevision   string
    LastScaleTime     time.Time
    ScaleCount        int
    CooldownUntil     time.Time
    Metadata          map[string]string
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

// StateStore interface for state persistence
type StateStore interface {
    Save(ctx context.Context, state *CNFState) error
    Get(ctx context.Context, name, namespace string) (*CNFState, error)
    Delete(ctx context.Context, name, namespace string) error
    List(ctx context.Context, namespace string) ([]*CNFState, error)
    Watch(ctx context.Context, namespace string) (<-chan StateEvent, error)
}

type StateEvent struct {
    Type  StateEventType  // Added, Updated, Deleted
    State *CNFState
}
```

### ConfigMap Implementation

```go
// pkg/cnf/state_store_configmap.go
type ConfigMapStateStore struct {
    client    client.Client
    namespace string  // Operator namespace for ConfigMaps
}

// State stored as: cnf-state-{name} ConfigMap with JSON data
func (s *ConfigMapStateStore) Save(ctx context.Context, state *CNFState) error
func (s *ConfigMapStateStore) Get(ctx context.Context, name, namespace string) (*CNFState, error)
```

### Acceptance Criteria

- [ ] State survives controller pod restart
- [ ] No state loss during crashes
- [ ] State recovery completes within 10s
- [ ] ConfigMap backend fully functional
- [ ] List operation supports namespace filtering
- [ ] Watch provides real-time updates
- [ ] Unit tests with mock client
- [ ] Test coverage >85%

### Test Requirements

```go
// pkg/cnf/state_store_test.go
func TestStateStore_SaveAndGet(t *testing.T)
func TestStateStore_Update(t *testing.T)
func TestStateStore_Delete(t *testing.T)
func TestStateStore_List(t *testing.T)
func TestStateStore_Watch(t *testing.T)
func TestStateStore_ConcurrentAccess(t *testing.T)
```

### Test Commands

```bash
# Unit tests
go test ./pkg/cnf/... -v -run TestStateStore

# Race detection
go test ./pkg/cnf/... -race -v -run TestStateStore
```

---

## Task 1.5: Deployment Status Monitoring

**Priority:** P1 (High)
**Effort:** 12 hours
**Dependencies:** Task 1.2, Task 1.3
**Owner:** Backend Architect
**Branch:** `feat/phase1-status-monitoring`

### Description

Implement active monitoring of CNF deployment status by polling Porch/Kubernetes and aggregating health checks. Export metrics to Prometheus.

### Files to Create

```
pkg/cnf/status_monitor.go           # Main monitoring logic
pkg/cnf/health_checker.go           # Health check aggregation
pkg/cnf/metrics.go                  # Prometheus metrics
pkg/cnf/status_monitor_test.go      # Unit tests
```

### Interface Definition

```go
// pkg/cnf/status_monitor.go
package cnf

import (
    "context"
    "time"

    "github.com/prometheus/client_golang/prometheus"
)

// StatusMonitor monitors CNF deployment status
type StatusMonitor struct {
    porchClient   *porch.Client
    k8sClient     client.Client
    stateStore    *CNFStateStore
    metrics       *CNFMetrics
    pollInterval  time.Duration
}

func (m *StatusMonitor) Start(ctx context.Context) error
func (m *StatusMonitor) Stop() error
func (m *StatusMonitor) MonitorCNF(ctx context.Context, name, namespace string) error
func (m *StatusMonitor) GetHealth(ctx context.Context, name, namespace string) (*HealthStatus, error)
```

### Health Check Logic

```go
// pkg/cnf/health_checker.go
type HealthStatus struct {
    Overall         HealthState  // Healthy, Degraded, Unhealthy
    Checks          []HealthCheck
    LastCheck       time.Time
    ConsecutiveFails int
}

type HealthCheck struct {
    Name      string
    Status    HealthState
    Message   string
    Timestamp time.Time
}

// Aggregated health checks:
// 1. Porch package status (Approved, Published)
// 2. Kubernetes Deployment ready replicas
// 3. Pod readiness probes
// 4. Service endpoint availability
```

### Prometheus Metrics

```go
// pkg/cnf/metrics.go
var (
    cnfDeploymentsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cnf_deployments_total",
            Help: "Total CNF deployments",
        },
        []string{"status", "cnf_type", "function"},
    )

    cnfDeploymentDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "cnf_deployment_duration_seconds",
            Help: "CNF deployment duration",
            Buckets: []float64{10, 30, 60, 120, 300, 600},
        },
        []string{"cnf_type", "function"},
    )

    cnfActiveCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "cnf_active_count",
            Help: "Number of active CNFs",
        },
        []string{"cnf_type", "function", "phase"},
    )

    cnfHealthStatus = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "cnf_health_status",
            Help: "CNF health status (1=healthy, 0=unhealthy)",
        },
        []string{"name", "namespace"},
    )
)
```

### Acceptance Criteria

- [ ] Status updates within 5s of actual change
- [ ] Health checks accurately reflect readiness
- [ ] Failed deployments detected immediately
- [ ] Metrics exported to Prometheus
- [ ] Poll interval configurable (default 10s)
- [ ] No memory leaks during long-running monitoring
- [ ] Integration tests with mock Porch
- [ ] Test coverage >80%

### Test Requirements

```go
// pkg/cnf/status_monitor_test.go
func TestStatusMonitor_Start(t *testing.T)
func TestStatusMonitor_MonitorCNF_Healthy(t *testing.T)
func TestStatusMonitor_MonitorCNF_Degraded(t *testing.T)
func TestStatusMonitor_GetHealth_NotFound(t *testing.T)
func TestHealthChecker_Aggregation(t *testing.T)
func TestMetrics_Exported(t *testing.T)
```

### Test Commands

```bash
# Unit tests
go test ./pkg/cnf/... -v -run TestStatusMonitor

# Integration tests
go test ./pkg/cnf/... -v -run TestStatusMonitor -tags=integration

# Check metrics
curl localhost:8080/metrics | grep cnf_
```

---

## Implementation Sequence

**Week 1:**
- Day 1-2: Task 1.1 (CNFLifecycleManager) - 20h
- Day 3-4: Task 1.2 (PorchClient) - 16h

**Week 2:**
- Day 1-2.5: Task 1.3 (Controller) - 20h
- Day 2.5-3.5: Task 1.4 (State Persistence) - 12h
- Day 3.5-5: Task 1.5 (Status Monitoring) - 12h

**Total:** 80 hours over 2 weeks

---

## Integration Test

### End-to-End Scenario

```bash
#!/bin/bash
# hack/test-cnf-deployment.sh

set -e

echo "==> Phase 1 E2E Test: CNF Deployment"

# 1. Deploy operator
kubectl apply -f config/crd/bases/
kubectl apply -f config/manager/

# 2. Wait for operator ready
kubectl wait --for=condition=available deployment/nephoran-intent-operator -n nephoran-system --timeout=60s

# 3. Create CNFDeployment CR
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: CNFDeployment
metadata:
  name: test-amf
  namespace: default
spec:
  type: 5G-Core
  function: AMF
  replicas: 2
  resources:
    requests:
      cpu: "1"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "4Gi"
EOF

# 4. Wait for provisioning
kubectl wait --for=condition=Provisioning cnfdeployment/test-amf --timeout=60s

# 5. Wait for active
kubectl wait --for=condition=Active cnfdeployment/test-amf --timeout=300s

# 6. Verify Porch package created
# (requires porch CLI or kubectl porch plugin)

# 7. Verify status
STATUS=$(kubectl get cnfdeployment test-amf -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Active" ]; then
    echo "ERROR: Expected phase Active, got $STATUS"
    exit 1
fi

# 8. Verify replicas
READY=$(kubectl get cnfdeployment test-amf -o jsonpath='{.status.readyReplicas}')
if [ "$READY" != "2" ]; then
    echo "ERROR: Expected 2 ready replicas, got $READY"
    exit 1
fi

# 9. Check metrics
curl -s localhost:8080/metrics | grep -q 'cnf_deployments_total{status="success"}'

echo "==> Phase 1 E2E Test: PASSED"
```

---

## Definition of Done

### Code Quality
- [ ] All code follows Go best practices
- [ ] golangci-lint passes with no errors
- [ ] go vet passes
- [ ] No race conditions detected
- [ ] Test coverage >80%

### Functionality
- [ ] All 5 tasks completed
- [ ] E2E test passes
- [ ] CNF deployment works end-to-end
- [ ] Status updates in real-time
- [ ] Rollback works on failure

### Documentation
- [ ] Code comments added
- [ ] API documentation generated
- [ ] E2E test documented
- [ ] Troubleshooting guide created

### Integration
- [ ] PR created against `integrate/mvp`
- [ ] CI/CD pipeline passes
- [ ] Code review completed
- [ ] Merged to integration branch

---

**Document Version:** 1.0
**Created:** 2026-02-16
**Owner:** Backend Architect
