# Test Implementation Guide - P0 Critical Packages

**Target**: Increase critical package coverage from <21% to 80%
**Timeline**: 3 weeks
**Effort**: 136 hours

---

## 1. pkg/controllers (20.8% → 80%)

**Effort**: 40 hours
**Files to Test**: 13 controller files

### Test Strategy

#### Test File Structure
```
pkg/controllers/
├── networkintent_controller_test.go (EXISTS - expand)
├── audittrail_controller_test.go (NEW)
├── backuppolicy_controller_test.go (NEW)
├── cnfdeployment_controller_test.go (NEW)
├── e2nodeset_controller_test.go (NEW)
└── ... (9 more controller tests)
```

#### Critical Test Cases per Controller

**NetworkIntent Controller** (Expand existing tests):
```go
// 1. Happy Path Tests
func TestNetworkIntentController_Reconcile_CreateSuccess(t *testing.T) {
    // Setup: Create fake client with NetworkIntent
    // Execute: Call Reconcile()
    // Assert: Status.Phase = "Processing", Porch package created
}

func TestNetworkIntentController_Reconcile_UpdateSuccess(t *testing.T) {
    // Setup: Existing NetworkIntent + update spec
    // Execute: Reconcile()
    // Assert: Updated package, status reflects changes
}

func TestNetworkIntentController_Reconcile_DeleteWithFinalizer(t *testing.T) {
    // Setup: NetworkIntent with DeletionTimestamp
    // Execute: Reconcile()
    // Assert: Finalizer logic runs, resource cleaned up
}

// 2. Error Handling Tests
func TestNetworkIntentController_Reconcile_InvalidSpec(t *testing.T) {
    // Setup: NetworkIntent with invalid spec
    // Execute: Reconcile()
    // Assert: Error status, condition = "ValidationFailed"
}

func TestNetworkIntentController_Reconcile_PorchAPIFailure(t *testing.T) {
    // Setup: Mock Porch client returns error
    // Execute: Reconcile()
    // Assert: Requeue request, error status, retry count incremented
}

func TestNetworkIntentController_Reconcile_StatusUpdateFailure(t *testing.T) {
    // Setup: Client fails on status update
    // Execute: Reconcile()
    // Assert: Error logged, requeue request
}

// 3. Edge Cases
func TestNetworkIntentController_Reconcile_NotFound(t *testing.T) {
    // Setup: Request for non-existent NetworkIntent
    // Execute: Reconcile()
    // Assert: No error, no requeue (already deleted)
}

func TestNetworkIntentController_Reconcile_SpecUnchanged(t *testing.T) {
    // Setup: NetworkIntent with generation == observedGeneration
    // Execute: Reconcile()
    // Assert: No action, no status update
}
```

**Test Template for Other Controllers**:
```go
package controllers

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client/fake"

    nephoran "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
)

func TestAuditTrailController_Reconcile_Success(t *testing.T) {
    // Arrange
    scheme := runtime.NewScheme()
    _ = nephoran.AddToScheme(scheme)

    auditTrail := &nephoran.AuditTrail{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-audit",
            Namespace: "default",
        },
        Spec: nephoran.AuditTrailSpec{
            Retention: metav1.Duration{Duration: 30 * 24 * time.Hour},
        },
    }

    client := fake.NewClientBuilder().
        WithScheme(scheme).
        WithObjects(auditTrail).
        Build()

    controller := &AuditTrailController{
        Client: client,
        Scheme: scheme,
        Log:    ctrl.Log.WithName("controllers").WithName("AuditTrail"),
    }

    req := ctrl.Request{
        NamespacedName: types.NamespacedName{
            Name:      "test-audit",
            Namespace: "default",
        },
    }

    // Act
    result, err := controller.Reconcile(context.Background(), req)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, ctrl.Result{}, result)

    // Verify status updated
    var updated nephoran.AuditTrail
    err = client.Get(context.Background(), req.NamespacedName, &updated)
    require.NoError(t, err)
    assert.NotEmpty(t, updated.Status.Phase)
}
```

#### Implementation Steps

**Week 1 Days 1-2**: NetworkIntent Controller
- Expand existing test coverage
- Add error handling tests
- Add edge case tests
- Target: 80% coverage

**Week 1 Days 3-5**: Top 5 Controllers
- AuditTrail, BackupPolicy, CNFDeployment, E2NodeSet, FailoverPolicy
- Use template above
- Focus on Reconcile() method + status updates
- Target: 70% coverage each

**Week 2**: Remaining 8 Controllers
- Apply same pattern
- Copy/adapt tests from NetworkIntent
- Target: 60% coverage minimum

---

## 2. pkg/oran/o1 (3.0% → 80%)

**Effort**: 16 hours
**Files**: `o1_adaptor.go`, `accounting_manager.go`, `o1_helpers.go`

### Test Strategy

#### Key Functions to Test (0% coverage):
1. `ComprehensiveAccountingManager` lifecycle
2. Usage data collection
3. Billing generation
4. Fraud detection
5. Payment processing

#### Critical Test Cases

```go
package o1

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestComprehensiveAccountingManager_Lifecycle(t *testing.T) {
    // Arrange
    config := &AccountingConfig{
        UsageCollectionInterval: 10 * time.Second,
        BillingCycleDay:        1,
        FraudDetectionEnabled:  true,
    }

    manager := NewComprehensiveAccountingManager(config, nil)

    // Act - Start
    err := manager.Start(context.Background())
    require.NoError(t, err)

    // Assert - Running
    stats := manager.GetAccountingStatistics()
    assert.NotNil(t, stats)

    // Act - Stop
    err = manager.Stop()
    require.NoError(t, err)
}

func TestAccountingManager_RecordUsage(t *testing.T) {
    // Arrange
    manager := NewComprehensiveAccountingManager(defaultConfig(), nil)
    manager.Start(context.Background())
    defer manager.Stop()

    usageEvent := &UsageEvent{
        SubscriberID: "sub-123",
        ResourceType: "data",
        Quantity:     1024, // 1GB
        Timestamp:    time.Now(),
    }

    // Act
    err := manager.RecordUsage(context.Background(), usageEvent)

    // Assert
    require.NoError(t, err)

    records := manager.GetUsageRecords("sub-123", time.Now().Add(-1*time.Hour), time.Now())
    assert.Len(t, records, 1)
    assert.Equal(t, int64(1024), records[0].Quantity)
}

func TestAccountingManager_GenerateBill(t *testing.T) {
    // Arrange
    manager := setupManagerWithUsage(t)

    // Act
    bill, err := manager.GenerateBill(context.Background(), "sub-123", time.Now().AddDate(0, 0, -30), time.Now())

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, bill)
    assert.Greater(t, bill.TotalAmount, 0.0)
    assert.NotEmpty(t, bill.LineItems)
}

func TestAccountingManager_ProcessPayment(t *testing.T) {
    // Arrange
    manager := setupManagerWithBill(t)
    payment := &Payment{
        BillID:        "bill-123",
        Amount:        100.50,
        PaymentMethod: "credit_card",
    }

    // Act
    result, err := manager.ProcessPayment(context.Background(), payment)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "success", result.Status)

    // Verify bill marked as paid
    bill := manager.GetBill("bill-123")
    assert.Equal(t, "paid", bill.Status)
}

func TestAccountingManager_FraudDetection(t *testing.T) {
    // Arrange
    manager := NewComprehensiveAccountingManager(&AccountingConfig{
        FraudDetectionEnabled: true,
        FraudThreshold:       1000.0,
    }, nil)
    manager.Start(context.Background())
    defer manager.Stop()

    // Act - Record suspicious usage
    for i := 0; i < 100; i++ {
        _ = manager.RecordUsage(context.Background(), &UsageEvent{
            SubscriberID: "sub-suspicious",
            ResourceType: "data",
            Quantity:     100000, // 100GB each
            Timestamp:    time.Now(),
        })
    }

    // Assert - Fraud alert triggered
    stats := manager.GetAccountingStatistics()
    assert.Greater(t, stats.ActiveFraudAlerts, int64(0))
}

// Helper functions
func defaultConfig() *AccountingConfig {
    return &AccountingConfig{
        UsageCollectionInterval: 10 * time.Second,
        BillingCycleDay:        1,
        FraudDetectionEnabled:  true,
        FraudThreshold:         1000.0,
    }
}

func setupManagerWithUsage(t *testing.T) *ComprehensiveAccountingManager {
    manager := NewComprehensiveAccountingManager(defaultConfig(), nil)
    manager.Start(context.Background())

    // Seed with usage data
    for i := 0; i < 10; i++ {
        _ = manager.RecordUsage(context.Background(), &UsageEvent{
            SubscriberID: "sub-123",
            ResourceType: "data",
            Quantity:     1024,
            Timestamp:    time.Now().Add(-time.Duration(i) * time.Hour),
        })
    }

    return manager
}

func setupManagerWithBill(t *testing.T) *ComprehensiveAccountingManager {
    manager := setupManagerWithUsage(t)
    _, _ = manager.GenerateBill(context.Background(), "sub-123", time.Now().AddDate(0, 0, -30), time.Now())
    return manager
}
```

#### Implementation Steps

**Day 1**: Accounting Manager Lifecycle
- Start/Stop tests
- Configuration tests
- Health check tests

**Day 2**: Usage Recording
- RecordUsage tests
- GetUsageRecords tests
- Usage validation tests

**Day 3**: Billing
- GenerateBill tests
- Bill calculation tests
- Billing cycle tests

**Day 4**: Payment & Fraud
- ProcessPayment tests
- Fraud detection tests
- Statistics tests

---

## 3. pkg/oran/o2 (2.0% → 80%)

**Effort**: 16 hours
**Files**: `adaptor.go`, `api_handlers.go`, `additional_types.go`

### Test Strategy

#### Mock O2 IMS Server
```go
package o2

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func newMockO2Server(t *testing.T) *httptest.Server {
    handler := http.NewServeMux()

    // GET /o2ims/v1/resourcePools
    handler.HandleFunc("/o2ims/v1/resourcePools", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodGet {
            pools := []ResourcePool{
                {ID: "pool-1", Name: "compute-pool", ResourceType: "compute"},
                {ID: "pool-2", Name: "storage-pool", ResourceType: "storage"},
            }
            json.NewEncoder(w).Encode(pools)
        }
    })

    // GET /o2ims/v1/resourcePools/{id}
    handler.HandleFunc("/o2ims/v1/resourcePools/", func(w http.ResponseWriter, r *http.Request) {
        // Extract ID from path
        id := r.URL.Path[len("/o2ims/v1/resourcePools/"):]

        if r.Method == http.MethodGet {
            pool := ResourcePool{
                ID:           id,
                Name:         "test-pool",
                ResourceType: "compute",
                Capacity:     100,
            }
            json.NewEncoder(w).Encode(pool)
        }
    })

    return httptest.NewServer(handler)
}
```

#### Critical Test Cases

```go
func TestO2Adaptor_GetResourcePools(t *testing.T) {
    // Arrange
    server := newMockO2Server(t)
    defer server.Close()

    adaptor, err := NewO2Adaptor(&O2Config{
        BaseURL: server.URL,
        Timeout: 10 * time.Second,
    })
    require.NoError(t, err)

    // Act
    pools, err := adaptor.GetResourcePools(context.Background())

    // Assert
    require.NoError(t, err)
    assert.Len(t, pools, 2)
    assert.Equal(t, "compute-pool", pools[0].Name)
}

func TestO2Adaptor_CreateResourcePool(t *testing.T) {
    // Arrange
    server := newMockO2Server(t)
    defer server.Close()

    adaptor, _ := NewO2Adaptor(&O2Config{BaseURL: server.URL})

    pool := &ResourcePool{
        Name:         "new-pool",
        ResourceType: "compute",
        Capacity:     50,
    }

    // Act
    created, err := adaptor.CreateResourcePool(context.Background(), pool)

    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, created.ID)
    assert.Equal(t, "new-pool", created.Name)
}

func TestO2Adaptor_GetSystemInfo(t *testing.T) {
    // Arrange
    adaptor, _ := NewO2Adaptor(DefaultO2Config())

    // Act
    info, err := adaptor.GetSystemInfo(context.Background())

    // Assert
    require.NoError(t, err)
    assert.NotEmpty(t, info.Version)
    assert.NotEmpty(t, info.SupportedResourceTypes)
}

func TestResourceEventBus_PublishSubscribe(t *testing.T) {
    // Arrange
    bus := NewResourceEventBus()

    received := make(chan ResourceEvent, 1)
    subID := bus.Subscribe("pool-created", func(event ResourceEvent) {
        received <- event
    })
    defer bus.Unsubscribe(subID)

    event := ResourceEvent{
        Type:       "pool-created",
        ResourceID: "pool-123",
        Timestamp:  time.Now(),
    }

    // Act
    bus.Publish(event)

    // Assert
    select {
    case evt := <-received:
        assert.Equal(t, "pool-created", evt.Type)
        assert.Equal(t, "pool-123", evt.ResourceID)
    case <-time.After(1 * time.Second):
        t.Fatal("event not received")
    }
}
```

---

## 4. pkg/security (5.8% → 90%)

**Effort**: 24 hours

### Test Strategy

#### Atomic File Operations
```go
func TestAtomicFileWriter_CreateIntentFile_Success(t *testing.T) {
    // Arrange
    dir := t.TempDir()
    writer := NewAtomicFileWriter(dir)

    intent := map[string]interface{}{
        "apiVersion": "intent.nephoran.com/v1alpha1",
        "kind":       "NetworkIntent",
        "metadata":   map[string]string{"name": "test"},
    }

    // Act
    path, err := writer.CreateIntentFile(context.Background(), intent)

    // Assert
    require.NoError(t, err)
    assert.FileExists(t, path)

    // Verify content
    data, _ := os.ReadFile(path)
    var loaded map[string]interface{}
    json.Unmarshal(data, &loaded)
    assert.Equal(t, "NetworkIntent", loaded["kind"])
}

func TestAtomicFileWriter_WriteIntentAtomic_PartialFailure(t *testing.T) {
    // Arrange
    dir := t.TempDir()
    writer := NewAtomicFileWriter(dir)

    // Simulate disk full by setting quota
    intent := strings.Repeat("x", 1024*1024*100) // 100MB

    // Act
    err := writer.WriteIntentAtomic(context.Background(), "test.json", []byte(intent))

    // Assert - Should fail gracefully
    if err != nil {
        // Verify no partial file left behind
        files, _ := os.ReadDir(dir)
        for _, f := range files {
            assert.NotContains(t, f.Name(), ".tmp")
        }
    }
}
```

#### Secret Management
```go
func TestFileBackend_StoreRetrieve(t *testing.T) {
    // Arrange
    dir := t.TempDir()
    backend := NewFileBackend(dir)

    secret := &Secret{
        Name:  "api-key",
        Value: "super-secret-value",
    }

    // Act - Store
    err := backend.Store(context.Background(), secret)
    require.NoError(t, err)

    // Act - Retrieve
    retrieved, err := backend.Retrieve(context.Background(), "api-key")

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "super-secret-value", retrieved.Value)
}

func TestAuditLogger_LogSecretAccess(t *testing.T) {
    // Arrange
    var buf bytes.Buffer
    logger := &AuditLogger{writer: &buf}

    // Act
    logger.LogSecretAccess("user-123", "api-key", "read", true)

    // Assert
    logged := buf.String()
    assert.Contains(t, logged, "user-123")
    assert.Contains(t, logged, "api-key")
    assert.Contains(t, logged, "read")
}
```

---

## 5. pkg/llm (7.2% → 70%)

**Effort**: 16 hours

### Test Strategy

```go
func TestAdvancedCircuitBreaker_OpenClose(t *testing.T) {
    // Arrange
    cb := NewAdvancedCircuitBreaker(&CircuitBreakerConfig{
        Threshold:      3,
        Timeout:        100 * time.Millisecond,
        ResetTimeout:   200 * time.Millisecond,
    })

    // Act - Cause failures to open circuit
    for i := 0; i < 3; i++ {
        cb.Execute(func() error {
            return errors.New("fail")
        })
    }

    // Assert - Circuit open
    assert.Equal(t, StateOpen, cb.GetState())

    // Wait for reset timeout
    time.Sleep(250 * time.Millisecond)

    // Assert - Circuit half-open
    assert.Equal(t, StateHalfOpen, cb.GetState())

    // Success should close circuit
    cb.Execute(func() error { return nil })
    assert.Equal(t, StateClosed, cb.GetState())
}
```

---

## Summary

**Total P0 Effort**: 136 hours (3.4 weeks for 1 engineer)
**Expected Coverage Increase**: 17.7% → 45%+

**Week 1**: Controllers + O1
**Week 2**: O2 + Security
**Week 3**: LLM + Commands + Polish

Next: Run tests and iterate to achieve targets.
