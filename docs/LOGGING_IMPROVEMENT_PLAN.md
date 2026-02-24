# Logging Improvement Plan - Kubernetes-Style Structured Logging

**目標**: 建立符合 Kubernetes 標準的結構化日誌系統，提高 logging 覆蓋率

**日期**: 2026-02-24
**版本**: 1.0

---

## 🎯 目標

1. **建立統一的 logging package** ✅ 完成
2. **提供 Kubernetes-style structured logging API** ✅ 完成
3. **建立 logging best practices 文檔** ✅ 完成
4. **分析並提高 logging 覆蓋率** ⏳ 進行中
5. **建立 log aggregation 整合** 📋 待完成

---

## ✅ 已完成的工作

### 1. Unified Logging Package

**位置**: `pkg/logging/logger.go`

**功能**:
- ✅ 基於 logr 介面 + zap 實作
- ✅ 支援 4 個 log levels (Debug, Info, Warn, Error)
- ✅ 結構化 JSON 輸出
- ✅ Context-aware logging (request ID, namespace, resource)
- ✅ 專用的 event methods (ReconcileStart, A1PolicyCreated, 等)
- ✅ Component-based logging
- ✅ 環境變數配置 (LOG_LEVEL, ENVIRONMENT)

### 2. Logger API

**基本使用**:
```go
import "github.com/thc1006/nephoran-intent-operator/pkg/logging"

// 建立 logger
logger := logging.NewLogger(logging.ComponentController)

// Info logging
logger.InfoEvent("NetworkIntent created",
    "namespace", "default",
    "name", "test-intent",
)

// Error logging
logger.ErrorEvent(err, "Failed to create A1 policy",
    "policyID", policyID,
)

// Debug logging
logger.DebugEvent("Processing intent file",
    "filename", filename,
)
```

**Context-aware logging**:
```go
// 添加 request ID
logger = logger.WithRequestID("req-123")

// 添加 namespace
logger = logger.WithNamespace("default")

// 添加 resource context
logger = logger.WithResource("NetworkIntent", namespace, name)

// 添加 intent context
logger = logger.WithIntent(intentType, target, namespace)
```

**專用 event methods**:
```go
// Reconciliation events
logger.ReconcileStart(namespace, name)
logger.ReconcileSuccess(namespace, name, duration)
logger.ReconcileError(namespace, name, err, duration)

// HTTP events
logger.HTTPRequest(method, path, statusCode, duration)
logger.HTTPError(method, path, statusCode, err, duration)

// A1 events
logger.A1PolicyCreated(policyID, intentType)
logger.A1PolicyDeleted(policyID)

// Scaling events
logger.ScalingExecuted(deployment, namespace, fromReplicas, toReplicas)

// File processing events
logger.IntentFileProcessed(filename, success, duration)

// Porch events
logger.PorchPackageCreated(packageName, namespace)
```

### 3. 測試

**位置**: `pkg/logging/logger_test.go`

**覆蓋率**: 15 個測試案例
- Logger creation
- Context methods (WithValues, WithNamespace, WithResource, WithIntent)
- Reconcile logging
- HTTP logging
- A1 policy logging
- Scaling logging
- Log level configuration
- Logger chaining

### 4. 文檔

**位置**: `docs/LOGGING_BEST_PRACTICES.md`

**內容**:
- Quick Start 指南
- Log Levels 說明
- Structured Logging 概念
- Component-Specific Logging
- Best Practices (7 大原則)
- Log Aggregation 整合
- 4 個完整範例 (Controller, Ingest, Scaling xApp, File Watcher)
- Migration Guide
- Troubleshooting

### 5. Coverage Analysis Tool

**位置**: `scripts/analyze-logging-coverage.sh`

**功能**:
- 掃描所有 Go 檔案
- 計算 logging 覆蓋率
- 識別 critical files 未加 logging
- 識別使用 plain log 的檔案
- 生成詳細報告

**執行方式**:
```bash
./scripts/analyze-logging-coverage.sh
```

---

## 📋 待完成工作

### Phase 1: 遷移 Critical Components (優先度: P0)

**目標**: 所有 critical components 使用 structured logging

**檔案清單**:

1. **controllers/networkintent_controller.go** (P0)
   - 當前: 使用 ctrl.Log
   - 目標: 遷移到 pkg/logging
   - 估計工作量: 2 小時

2. **cmd/intent-ingest/main.go** (P0)
   - 當前: 使用 log.Printf
   - 目標: 遷移到 pkg/logging
   - 估計工作量: 1 小時

3. **internal/loop/watcher.go** (P0)
   - 當前: 使用 log.Printf
   - 目標: 遷移到 pkg/logging
   - 估計工作量: 2 小時

4. **pkg/porch/client.go** (P0)
   - 當前: 未確認
   - 目標: 添加 pkg/logging
   - 估計工作量: 1 小時

5. **pkg/oran/a1/** (P0)
   - 當前: 未確認
   - 目標: 添加 pkg/logging
   - 估計工作量: 2 小時

**總估計**: 8 小時

### Phase 2: 遷移 Secondary Components (優先度: P1)

**目標**: 60%+ logging 覆蓋率

**檔案類別**:
- pkg/rag/
- pkg/llm/
- pkg/handlers/
- internal/patch/
- internal/conductor/

**估計工作量**: 10 小時

### Phase 3: Log Aggregation 整合 (優先度: P2)

**目標**: 建立生產級 log aggregation 系統

**步驟**:

1. **部署 Loki**
   ```bash
   helm repo add grafana https://grafana.github.io/helm-charts
   helm install loki grafana/loki-stack \
     --namespace monitoring \
     --set grafana.enabled=false
   ```

2. **部署 Promtail** (log shipper)
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: promtail-config
   data:
     promtail.yaml: |
       clients:
         - url: http://loki:3100/loki/api/v1/push
       scrape_configs:
         - job_name: kubernetes-pods
           kubernetes_sd_configs:
             - role: pod
   ```

3. **Grafana Loki Data Source**
   - 添加 Loki data source
   - 建立 log dashboards

4. **Grafana Dashboards**
   - NetworkIntent Controller logs
   - Intent Ingest Service logs
   - Scaling xApp logs
   - A1 Integration logs
   - Error logs dashboard

**估計工作量**: 4 小時

### Phase 4: Log 最佳化 (優先度: P3)

**目標**: 效能優化和進階功能

**項目**:
1. **Log Sampling** - 高頻 logs 採樣 (例如 debug logs)
2. **Log Rotation** - 檔案日誌輪替 (如果需要)
3. **Log Metrics** - 從 logs 提取 metrics
4. **Alert Rules** - 基於 logs 的告警規則
5. **Log Retention Policies** - 日誌保留策略

**估計工作量**: 6 小時

---

## 🚀 Implementation Roadmap

### Week 1: Core Migration

**Day 1-2**: P0 Controllers
- [ ] Migrate networkintent_controller.go
- [ ] Add comprehensive logging to reconciliation loop
- [ ] Add error path logging

**Day 3-4**: P0 Services
- [ ] Migrate intent-ingest/main.go
- [ ] Add HTTP request logging
- [ ] Add LLM integration logging

**Day 5**: P0 Core Packages
- [ ] Migrate internal/loop/watcher.go
- [ ] Migrate pkg/porch/client.go
- [ ] Migrate pkg/oran/a1/

### Week 2: Extended Coverage

**Day 1-3**: P1 Packages
- [ ] Migrate pkg/rag/
- [ ] Migrate pkg/llm/
- [ ] Migrate pkg/handlers/

**Day 4-5**: Log Aggregation
- [ ] Deploy Loki
- [ ] Configure Promtail
- [ ] Create Grafana dashboards

### Week 3: Optimization

**Day 1-2**: Log Sampling & Performance
- [ ] Implement log sampling for debug logs
- [ ] Performance testing

**Day 3-5**: Alerts & Monitoring
- [ ] Create alert rules
- [ ] Set up log-based metrics
- [ ] Documentation finalization

---

## 📊 Success Criteria

### Minimum Viable Product (MVP)

- [x] ✅ Unified logging package created
- [x] ✅ Best practices documented
- [ ] ⏳ 80%+ logging coverage for critical components
- [ ] ⏳ All controllers using structured logging
- [ ] ⏳ All HTTP handlers logging requests

### Full Implementation

- [ ] 80%+ overall logging coverage
- [ ] 100% critical components coverage
- [ ] Loki integration deployed
- [ ] Grafana dashboards created
- [ ] Alert rules configured
- [ ] Log retention policies set

---

## 🔍 Migration Checklist

### For Each File Migration

**Before Migration**:
- [ ] Read current logging implementation
- [ ] Identify all log points
- [ ] Identify error paths

**During Migration**:
- [ ] Replace import statements
- [ ] Create component logger
- [ ] Migrate all log.Printf → logger.InfoEvent
- [ ] Add context fields
- [ ] Add duration logging where appropriate
- [ ] Use specialized event methods where applicable

**After Migration**:
- [ ] Test logging output
- [ ] Verify JSON format
- [ ] Verify log levels
- [ ] Update tests if needed
- [ ] Update documentation

### Example Migration

**Before**:
```go
package controllers

import (
    "log"
    ctrl "sigs.k8s.io/controller-runtime"
)

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log.Printf("Reconciling NetworkIntent: %s/%s", req.Namespace, req.Name)

    // ... logic ...

    if err != nil {
        log.Printf("ERROR: Failed to create A1 policy: %v", err)
        return ctrl.Result{}, err
    }

    log.Printf("Successfully created A1 policy: %s", policyID)
    return ctrl.Result{}, nil
}
```

**After**:
```go
package controllers

import (
    "time"
    "github.com/thc1006/nephoran-intent-operator/pkg/logging"
    ctrl "sigs.k8s.io/controller-runtime"
)

type NetworkIntentReconciler struct {
    logger logging.Logger
}

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    start := time.Now()
    logger := r.logger.ReconcileStart(req.Namespace, req.Name)

    // ... logic ...

    duration := time.Since(start).Seconds()

    if err != nil {
        logger.ReconcileError(req.Namespace, req.Name, err, duration)
        return ctrl.Result{}, err
    }

    logger.A1PolicyCreated(policyID, intentType)
    logger.ReconcileSuccess(req.Namespace, req.Name, duration)
    return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
    r.logger = logging.NewLogger(logging.ComponentController)
    return ctrl.NewControllerManagedBy(mgr).
        For(&intentv1alpha1.NetworkIntent{}).
        Complete(r)
}
```

---

## 🎓 Training Materials

### Quick Reference Card

```go
// 1. 建立 logger
logger := logging.NewLogger(logging.ComponentController)

// 2. 基本 logging
logger.InfoEvent("event", "key", "value")
logger.ErrorEvent(err, "event", "key", "value")
logger.DebugEvent("event", "key", "value")

// 3. 添加 context
logger = logger.WithNamespace(namespace)
logger = logger.WithRequestID(requestID)
logger = logger.WithResource("NetworkIntent", namespace, name)

// 4. 專用 events
logger.ReconcileStart(namespace, name)
logger.A1PolicyCreated(policyID, intentType)
logger.ScalingExecuted(deployment, namespace, from, to)

// 5. Duration tracking
start := time.Now()
// ... operation ...
logger.InfoEvent("completed", "durationSeconds", time.Since(start).Seconds())
```

### Video Tutorial Topics

1. "Why Structured Logging?" (5 min)
2. "Migrating from log to pkg/logging" (10 min)
3. "Adding Context to Logs" (8 min)
4. "Setting up Loki + Grafana" (15 min)
5. "Querying Logs in Grafana" (10 min)

---

## 📚 References

- [Logging Best Practices](./LOGGING_BEST_PRACTICES.md)
- [pkg/logging Package](../pkg/logging/)
- [Kubernetes Logging Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md)
- [go-logr Documentation](https://github.com/go-logr/logr)
- [Grafana Loki Documentation](https://grafana.com/docs/loki/)

---

## 💡 Tips & Tricks

### Tip 1: Use logger chaining
```go
logger.WithNamespace("default").
       WithRequestID("req-123").
       WithValues("operation", "reconcile").
       InfoEvent("processing started")
```

### Tip 2: Create scoped loggers
```go
// Create a logger for the current reconciliation
reconcileLogger := r.logger.WithResource("NetworkIntent", namespace, name)

// Use throughout the reconciliation
reconcileLogger.InfoEvent("fetching resource")
reconcileLogger.InfoEvent("creating A1 policy")
reconcileLogger.InfoEvent("updating status")
```

### Tip 3: Log at entry and exit
```go
func (r *Reconciler) reconcile(...) error {
    logger := r.logger.ReconcileStart(namespace, name)
    defer func() {
        if err != nil {
            logger.ReconcileError(namespace, name, err, duration)
        } else {
            logger.ReconcileSuccess(namespace, name, duration)
        }
    }()

    // ... reconciliation logic ...
}
```

---

**計畫狀態**: 📋 Phase 1 (Core Implementation) ✅ 完成
**下一步**: Phase 2 (Migration) ⏳ 開始執行
**預計完成**: 3 weeks
**負責人**: Development Team
