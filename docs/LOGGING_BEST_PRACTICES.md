# Logging Best Practices - Nephoran Intent Operator

**基於 Kubernetes 標準的結構化日誌指南**

---

## 📖 目錄

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Log Levels](#log-levels)
- [Structured Logging](#structured-logging)
- [Component-Specific Logging](#component-specific-logging)
- [Best Practices](#best-practices)
- [Log Aggregation](#log-aggregation)
- [Examples](#examples)

---

## Overview

Nephoran Intent Operator 使用 **structured logging** (結構化日誌)，遵循 Kubernetes 社群的最佳實踐：

- **基於 logr 介面**: 與 controller-runtime 完美整合
- **底層使用 zap**: 高效能的結構化日誌庫
- **JSON 格式輸出**: 易於被 log aggregation tools 解析
- **Context-aware**: 自動攜帶 request ID、namespace、resource 等上下文
- **Kubernetes-native**: 支援 `kubectl logs` 和 log aggregation

---

## Quick Start

### 1. 匯入 logging package

```go
import (
    "github.com/thc1006/nephoran-intent-operator/pkg/logging"
)
```

### 2. 建立 logger

```go
// 在 main() 或 init() 中初始化全域 logger
logging.InitGlobalLogger(logging.GetLogLevel())

// 在各個 component 中建立 logger
logger := logging.NewLogger(logging.ComponentController)
```

### 3. 使用 logger

```go
// Info level logging
logger.InfoEvent("NetworkIntent created",
    "namespace", "default",
    "name", "test-intent",
)

// Error logging
logger.ErrorEvent(err, "Failed to create A1 policy",
    "policyID", policyID,
)

// Debug logging (only shown when LOG_LEVEL=debug)
logger.DebugEvent("Processing intent file",
    "filename", filename,
)
```

---

## Log Levels

### 支援的 Log Levels

| Level | 說明 | 使用時機 |
|-------|------|----------|
| **Debug** | 詳細的除錯資訊 | 開發環境、問題追蹤 |
| **Info** | 一般資訊性訊息 | 正常操作事件 (default) |
| **Warn** | 警告訊息 | 潛在問題，但不影響運行 |
| **Error** | 錯誤訊息 | 操作失敗，需要注意 |

### 設定 Log Level

**方法 1: 環境變數**
```bash
export LOG_LEVEL=debug
```

**方法 2: Kubernetes Deployment**
```yaml
env:
- name: LOG_LEVEL
  value: "info"
```

**方法 3: 程式碼**
```go
logger := logging.NewLoggerWithLevel(logging.ComponentController, logging.DebugLevel)
```

---

## Structured Logging

### 為什麼使用 Structured Logging?

**傳統 logging** (❌ 不推薦):
```go
log.Printf("Created policy %s for intent %s in namespace %s", policyID, intentName, namespace)
```

**Structured logging** (✅ 推薦):
```go
logger.InfoEvent("Policy created",
    "policyID", policyID,
    "intentName", intentName,
    "namespace", namespace,
)
```

### 優點

1. **易於解析**: JSON 格式可被 Elasticsearch、Loki 等工具自動解析
2. **易於查詢**: 可以用 field 查詢，如 `{namespace="default"}`
3. **易於過濾**: 可以精確過濾特定欄位
4. **型別安全**: 避免字串格式化錯誤

### 輸出範例

```json
{
  "ts": "2026-02-24T10:30:00.123Z",
  "level": "info",
  "msg": "Policy created",
  "component": "controller",
  "policyID": "policy-abc123",
  "intentName": "scale-nf-sim",
  "namespace": "ran-a"
}
```

---

## Component-Specific Logging

### 預定義的 Components

```go
const (
    ComponentController   = "controller"       // NetworkIntent Controller
    ComponentIngest       = "intent-ingest"    // Intent Ingest Service
    ComponentRAG          = "rag-pipeline"     // RAG Pipeline
    ComponentPorch        = "porch-client"     // Porch Client
    ComponentA1           = "a1-client"        // A1 Interface Client
    ComponentScalingXApp  = "scaling-xapp"     // Scaling xApp
    ComponentWatcher      = "file-watcher"     // File Watcher
    ComponentValidator    = "validator"        // Intent Validator
    ComponentLLM          = "llm-client"       // LLM Client
    ComponentWebhook      = "webhook"          // Admission Webhook
    ComponentMetrics      = "metrics"          // Metrics Collector
)
```

### 使用方式

```go
// Controller
logger := logging.NewLogger(logging.ComponentController)

// Intent Ingest
logger := logging.NewLogger(logging.ComponentIngest)

// Scaling xApp
logger := logging.NewLogger(logging.ComponentScalingXApp)
```

---

## Best Practices

### 1. 使用 Context-Aware Logging

**為 reconciliation 添加上下文**:
```go
logger := logger.ReconcileStart(namespace, name)
defer func() {
    duration := time.Since(start).Seconds()
    if err != nil {
        logger.ReconcileError(namespace, name, err, duration)
    } else {
        logger.ReconcileSuccess(namespace, name, duration)
    }
}()
```

**為 HTTP requests 添加上下文**:
```go
start := time.Now()
// ... handle request ...
duration := time.Since(start).Seconds()
logger.HTTPRequest(method, path, statusCode, duration)
```

### 2. 使用專用的 Event Methods

**不要** (❌):
```go
logger.Info("A1 policy created", "policyID", policyID, "intentType", intentType)
```

**應該** (✅):
```go
logger.A1PolicyCreated(policyID, intentType)
```

**可用的專用 methods**:
- `A1PolicyCreated(policyID, intentType)`
- `A1PolicyDeleted(policyID)`
- `IntentFileProcessed(filename, success, duration)`
- `PorchPackageCreated(packageName, namespace)`
- `ScalingExecuted(deployment, namespace, fromReplicas, toReplicas)`
- `ReconcileStart(namespace, name)`
- `ReconcileSuccess(namespace, name, duration)`
- `ReconcileError(namespace, name, err, duration)`
- `HTTPRequest(method, path, statusCode, duration)`
- `HTTPError(method, path, statusCode, err, duration)`

### 3. 攜帶 Request ID

```go
// 從 HTTP request 提取 request ID
requestID := r.Header.Get("X-Request-ID")
if requestID == "" {
    requestID = uuid.New().String()
}

// 建立帶有 request ID 的 logger
logger := logger.WithRequestID(requestID)

// 所有後續的 log 都會自動包含 request ID
logger.InfoEvent("Processing intent")
```

### 4. 為 Resource 添加上下文

```go
// 方法 1: 使用 WithResource
logger := logger.WithResource("NetworkIntent", namespace, name)
logger.InfoEvent("Processing resource")

// 方法 2: 使用 WithIntent (for intents)
logger := logger.WithIntent(intentType, target, namespace)
logger.InfoEvent("Creating intent")
```

### 5. 記錄 Duration

```go
start := time.Now()
// ... operation ...
duration := time.Since(start).Seconds()

logger.InfoEvent("Operation completed",
    "operation", "reconcile",
    "durationSeconds", duration,
)
```

### 6. 適當的 Log Level

**Debug** - 詳細的內部狀態:
```go
logger.DebugEvent("Checking file stability",
    "filename", filename,
    "size", size,
    "modTime", modTime,
)
```

**Info** - 正常操作事件:
```go
logger.InfoEvent("Intent file processed successfully",
    "filename", filename,
)
```

**Warn** - 非致命問題:
```go
logger.WarnEvent("Policy cleanup failed, will retry",
    "policyID", policyID,
    "attempt", attempt,
)
```

**Error** - 操作失敗:
```go
logger.ErrorEvent(err, "Failed to create NetworkIntent",
    "namespace", namespace,
    "name", name,
)
```

### 7. 避免 Sensitive Information

**不要記錄** (❌):
- API keys, tokens, passwords
- User credentials
- Private data (PII)

**可以記錄** (✅):
- Resource names, namespaces
- Operation types
- Durations, counts
- Non-sensitive error messages

```go
// ❌ BAD
logger.InfoEvent("User authenticated", "password", password)

// ✅ GOOD
logger.InfoEvent("User authenticated", "username", username)
```

---

## Log Aggregation

### Kubernetes Integration

所有 logs 自動輸出到 **stdout/stderr**，可被 Kubernetes 收集：

```bash
# 查看即時 logs
kubectl logs -n nephoran-system deployment/controller-manager -f

# 查看過去 1 小時的 logs
kubectl logs -n nephoran-system deployment/controller-manager --since=1h

# 查看特定 pod 的 logs
kubectl logs -n nephoran-system controller-manager-xxxxx-yyy
```

### Log Aggregation Stack

**推薦配置**:
```
Pods (stdout) → Fluentd/Fluent Bit → Loki → Grafana
```

**Loki Query 範例**:
```logql
# 查詢特定 component 的 logs
{namespace="nephoran-system", component="controller"}

# 查詢特定 namespace 的 intent logs
{namespace="nephoran-system"} |= "namespace" |= "ran-a"

# 查詢 errors
{namespace="nephoran-system"} | level="error"

# 查詢特定 policyID
{namespace="nephoran-system"} | policyID="policy-abc123"
```

### Prometheus Integration

可以從 logs 中提取 metrics:

```yaml
# prometheus-operator ServiceMonitor
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: controller-logs
spec:
  selector:
    matchLabels:
      app: controller-manager
  endpoints:
  - port: metrics
```

---

## Examples

### Example 1: NetworkIntent Controller

```go
package controllers

import (
    "context"
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

    // Get NetworkIntent
    logger.DebugEvent("Fetching NetworkIntent from API server")

    // ... reconciliation logic ...

    // Success
    duration := time.Since(start).Seconds()
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

### Example 2: Intent Ingest Service

```go
package main

import (
    "net/http"
    "time"

    "github.com/thc1006/nephoran-intent-operator/pkg/logging"
    "github.com/google/uuid"
)

func handleIntent(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    requestID := uuid.New().String()
    logger := logging.NewLogger(logging.ComponentIngest).WithRequestID(requestID)

    logger.InfoEvent("Intent request received",
        "method", r.Method,
        "path", r.URL.Path,
        "remoteAddr", r.RemoteAddr,
    )

    // Process intent
    filename, err := processIntent(r.Body)
    duration := time.Since(start).Seconds()

    if err != nil {
        logger.HTTPError(r.Method, r.URL.Path, 500, err, duration)
        http.Error(w, "Internal Server Error", 500)
        return
    }

    logger.IntentFileProcessed(filename, true, duration)
    logger.HTTPRequest(r.Method, r.URL.Path, 200, duration)

    w.WriteHeader(http.StatusOK)
}

func main() {
    // Initialize global logger
    logging.InitGlobalLogger(logging.GetLogLevel())

    logger := logging.NewLogger(logging.ComponentIngest)
    logger.InfoEvent("Starting Intent Ingest Service",
        "addr", ":8080",
        "logLevel", logging.GetLogLevel(),
    )

    http.HandleFunc("/intent", handleIntent)
    http.ListenAndServe(":8080", nil)
}
```

### Example 3: Scaling xApp

```go
package main

import (
    "context"
    "time"

    "github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

func (x *ScalingXApp) scaleDeployment(ctx context.Context, spec ScalingSpec) error {
    start := time.Now()
    logger := logging.NewLogger(logging.ComponentScalingXApp).
        WithIntent(spec.IntentType, spec.Target, spec.Namespace)

    logger.DebugEvent("Getting deployment from K8s API")

    deployment, err := x.k8sClient.AppsV1().Deployments(spec.Namespace).Get(
        ctx, spec.Target, metav1.GetOptions{})
    if err != nil {
        logger.ErrorEvent(err, "Failed to get deployment",
            "deployment", spec.Target,
            "namespace", spec.Namespace,
        )
        return err
    }

    currentReplicas := *deployment.Spec.Replicas
    if currentReplicas == spec.Replicas {
        logger.InfoEvent("Deployment already at desired replicas",
            "deployment", spec.Target,
            "replicas", spec.Replicas,
        )
        return nil
    }

    // Update replicas
    deployment.Spec.Replicas = &spec.Replicas
    _, err = x.k8sClient.AppsV1().Deployments(spec.Namespace).Update(
        ctx, deployment, metav1.UpdateOptions{})
    if err != nil {
        logger.ErrorEvent(err, "Failed to update deployment")
        return err
    }

    duration := time.Since(start).Seconds()
    logger.ScalingExecuted(spec.Target, spec.Namespace, currentReplicas, spec.Replicas)
    logger.InfoEvent("Scaling operation completed",
        "durationSeconds", duration,
    )

    return nil
}
```

### Example 4: File Watcher

```go
package loop

import (
    "time"

    "github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

type Watcher struct {
    logger logging.Logger
}

func (w *Watcher) Start() error {
    w.logger = logging.NewLogger(logging.ComponentWatcher)

    w.logger.InfoEvent("Starting file watcher",
        "watchDir", w.watchDir,
        "pollInterval", w.pollInterval,
    )

    for {
        files, err := w.scanDirectory()
        if err != nil {
            w.logger.ErrorEvent(err, "Failed to scan directory")
            continue
        }

        w.logger.DebugEvent("Directory scan completed",
            "filesFound", len(files),
        )

        for _, file := range files {
            if err := w.processFile(file); err != nil {
                w.logger.ErrorEvent(err, "Failed to process file",
                    "filename", file.Name(),
                )
                continue
            }

            w.logger.IntentFileProcessed(file.Name(), true, 0.0)
        }

        time.Sleep(w.pollInterval)
    }
}
```

---

## Migration Guide

### 從標準 log 遷移

**Before** (標準 log):
```go
import "log"

log.Printf("Processing intent: %s/%s", namespace, name)
log.Printf("ERROR: Failed to create policy: %v", err)
```

**After** (structured logging):
```go
import "github.com/thc1006/nephoran-intent-operator/pkg/logging"

logger := logging.NewLogger(logging.ComponentController)
logger.InfoEvent("Processing intent",
    "namespace", namespace,
    "name", name,
)
logger.ErrorEvent(err, "Failed to create policy")
```

### 從 controller-runtime logger 遷移

**Before**:
```go
logger := ctrl.Log.WithName("controller").WithName("NetworkIntent")
logger.Info("reconciling", "namespace", req.Namespace, "name", req.Name)
```

**After**:
```go
logger := logging.NewLogger(logging.ComponentController)
logger.ReconcileStart(req.Namespace, req.Name)
```

---

## Troubleshooting

### 問題 1: Logs 未輸出

**檢查 log level**:
```bash
kubectl set env deployment/controller-manager LOG_LEVEL=debug -n nephoran-system
```

### 問題 2: 無法在 Grafana 查詢 logs

**確認 JSON 格式輸出**:
```bash
kubectl logs -n nephoran-system deployment/controller-manager | head -1
# 應該看到 JSON 格式: {"ts":"2026-02-24T10:00:00Z",...}
```

### 問題 3: Logs 太多

**調整 log level 到 info 或 warn**:
```yaml
env:
- name: LOG_LEVEL
  value: "warn"
```

---

## Summary

### Key Takeaways

1. ✅ **Always use structured logging** with key-value pairs
2. ✅ **Use appropriate log levels** (debug/info/warn/error)
3. ✅ **Add context** (request ID, namespace, resource)
4. ✅ **Record durations** for performance tracking
5. ✅ **Use component-specific loggers**
6. ✅ **Avoid logging sensitive information**
7. ✅ **Integrate with log aggregation tools** (Loki, Elasticsearch)

### Quick Reference

```go
// Basic usage
logger := logging.NewLogger(logging.ComponentController)
logger.InfoEvent("event", "key", "value")

// With context
logger.WithNamespace("default").WithRequestID("req-123").InfoEvent("event")

// Specialized events
logger.A1PolicyCreated(policyID, intentType)
logger.ScalingExecuted(deployment, namespace, fromReplicas, toReplicas)
logger.ReconcileSuccess(namespace, name, duration)
```

---

**文檔版本**: 1.0
**最後更新**: 2026-02-24
**適用版本**: Nephoran Intent Operator v1.2+
