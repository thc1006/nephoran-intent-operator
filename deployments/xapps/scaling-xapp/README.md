# Scaling xApp for O-RAN RIC

這是一個用於執行 Kubernetes Deployment 擴展的 O-RAN xApp。

## 功能

- 從 A1 Mediator 輪詢 scaling policies (policy type 100)
- 解析 scaling policy 並執行 Kubernetes deployment 擴展
- 支持跨 namespace 操作
- 每 30 秒自動同步一次

## 架構

```
NetworkIntent CRD → NetworkIntent Controller → A1 Policy → 
A1 Mediator → Scaling xApp → Kubernetes Deployment
```

## Policy 格式

```json
{
  "intentType": "scaling",
  "target": "nf-sim",
  "namespace": "ran-a",
  "replicas": 5,
  "source": "user"
}
```

## 構建和部署

```bash
cd deployments/xapps/scaling-xapp
./build-and-deploy.sh
```

## 環境變數

- `A1_MEDIATOR_URL`: A1 Mediator 端點（預設：http://service-ricplt-a1mediator-http.ricplt:10000）
- `POLL_INTERVAL`: 輪詢間隔（預設：30s）

## 權限

需要以下 Kubernetes RBAC 權限：
- `apps/deployments`: get, list, watch, update, patch
- `pods`: get, list, watch

## 測試

```bash
# 查看日誌
kubectl logs -n ricxapp deployment/ricxapp-scaling -f

# 創建測試 NetworkIntent
kubectl apply -f - <<YAML
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: test-scaling
  namespace: ran-a
spec:
  intentType: scaling
  target: nf-sim
  namespace: ran-a
  replicas: 5
YAML

# 驗證擴展
watch -n 1 kubectl get deployment -n ran-a nf-sim
```

## Prometheus Metrics

Scaling xApp 暴露以下 Prometheus metrics 於端口 2112：

### Counters (累計計數)

- `scaling_xapp_policies_processed_total{namespace, deployment, result}`: 已處理的 policies 總數
  - `result`: `success`, `failed`, `already_scaled`
- `scaling_xapp_a1_requests_total{method, status_code}`: A1 API 請求總數

### Gauges (瞬時值)

- `scaling_xapp_active_policies`: 當前活躍的 policies 數量
- `scaling_xapp_last_poll_timestamp`: 最後一次輪詢的時間戳

### Histograms (分佈統計)

- `scaling_xapp_a1_request_duration_seconds{method}`: A1 API 請求延遲
  - `method`: `GET_POLICIES`, `GET_POLICY`
- `scaling_xapp_scaling_duration_seconds{namespace, deployment}`: Scaling 操作耗時

### 訪問 Metrics

```bash
# Port forward
kubectl port-forward -n ricxapp deployment/ricxapp-scaling 2112:2112

# 查看 metrics
curl http://localhost:2112/metrics

# 健康檢查
curl http://localhost:2112/health
```

### ServiceMonitor (Prometheus Operator)

如果使用 Prometheus Operator，可部署 ServiceMonitor：

```bash
kubectl apply -f servicemonitor.yaml
```

### Grafana 示例查詢

```promql
# Scaling 成功率
rate(scaling_xapp_policies_processed_total{result="success"}[5m])
/
rate(scaling_xapp_policies_processed_total[5m])

# A1 API 請求延遲 (95th percentile)
histogram_quantile(0.95,
  rate(scaling_xapp_a1_request_duration_seconds_bucket[5m]))

# 活躍 policies 數量
scaling_xapp_active_policies
```

## 日誌示例

```
2026-02-24T08:00:00Z INFO Starting Scaling xApp
2026-02-24T08:00:00Z INFO A1 Mediator URL: http://service-ricplt-a1mediator-http.ricplt:10000
2026-02-24T08:00:00Z INFO Poll Interval: 30s
2026-02-24T08:00:30Z INFO Found 3 scaling policies
2026-02-24T08:00:30Z INFO Executing scaling policy: policy-test-scale-to-5 (target=nf-sim, namespace=ran-a, replicas=5)
2026-02-24T08:00:30Z INFO ✅ Successfully scaled ran-a/nf-sim: 2 → 5 replicas
```

## 故障排除

### xApp Pod 無法啟動

檢查 RBAC 權限：
```bash
kubectl get clusterrole scaling-xapp-role
kubectl get clusterrolebinding scaling-xapp-binding
```

### 無法連接 A1 Mediator

檢查 A1 Mediator 服務：
```bash
kubectl get svc -n ricplt service-ricplt-a1mediator-http
```

### Deployment 未被更新

檢查 xApp 日誌：
```bash
kubectl logs -n ricxapp deployment/ricxapp-scaling
```

## 與 KPIMON xApp 的區別

| 特性 | KPIMON xApp | Scaling xApp |
|------|-------------|--------------|
| 功能 | 監控 KPI 指標 | 執行 Deployment 擴展 |
| E2 Interface | ✅ 使用 | ❌ 不使用 |
| A1 Interface | ✅ 接收 policy | ✅ 接收 policy |
| K8s API | ❌ 不直接調用 | ✅ 直接調用 |
| RMR Messaging | ✅ 使用 | ❌ 不使用 |

