# Grafana NF Scale 監控 Dashboard 設置指南

## 📊 訪問 Grafana

- **URL**: http://192.168.10.65:30300
- **帳號**: `admin`
- **密碼**: `NephoranMonitor2026!`

## 🎯 導入 NF Scale Dashboard

### 方法 1：透過 Grafana UI 導入

1. 登入 Grafana
2. 點擊左側 **「+」** 圖標 → 選擇 **「Import」**
3. 點擊 **「Upload JSON file」**
4. 選擇檔案：`/tmp/grafana-nf-dashboard.json`
5. 選擇 Prometheus 數據源
6. 點擊 **「Import」**

### 方法 2：使用 API 自動導入

```bash
# 使用 Grafana API 導入 dashboard
curl -X POST http://admin:NephoranMonitor2026!@192.168.10.65:30300/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @/tmp/grafana-nf-dashboard.json
```

## 📈 Dashboard 功能

導入後，您將看到：

### Panel 1: NF-SIM Deployment Replicas 趨勢
- **類型**: 時間序列圖
- **顯示內容**:
  - 🔵 期望 Replicas（spec.replicas）
  - 🟢 就緒 Replicas（status.readyReplicas）
  - 🟡 可用 Replicas（status.availableReplicas）
- **功能**: 實時顯示 replicas 變化趨勢，可清楚看到 Scale Out/In

### Panel 2: 期望 Replicas (Gauge)
- **類型**: 儀表盤
- **顯示內容**: 當前期望的 replica 數量
- **用途**: 一目了然地看到目標值

### Panel 3: 就緒 Replicas (Gauge)
- **類型**: 儀表盤
- **顯示內容**: 當前就緒的 replica 數量
- **顏色編碼**:
  - 🔴 紅色: 0 個就緒
  - 🟡 黃色: 1 個就緒
  - 🟢 綠色: 2+ 個就緒

### Panel 4: Pods 狀態分佈
- **類型**: 堆疊條形圖
- **顯示內容**:
  - 🟢 Running pods
  - 🟡 Pending pods
  - 🔴 Failed pods
- **用途**: 視覺化 pods 的狀態變化

### Panel 5: NetworkIntent CRD 數量
- **類型**: 條形圖
- **顯示內容**: NetworkIntent 資源總數
- **用途**: 監控 intent 創建情況

## 🧪 測試監控

### 1. 開啟 Grafana Dashboard

訪問 http://192.168.10.65:30300 並登入，開啟 「Nephoran NF Scale Out/In 監控」dashboard。

### 2. 在另一個終端執行 Scale Out

```bash
curl -X POST http://192.168.10.65:30081/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 10 in ns ran-a"
```

### 3. 觀察 Grafana 變化

您將看到：
1. **期望 Replicas** 從當前值上升到 10
2. **就緒 Replicas** 逐步增加（可能需要 20-60 秒）
3. **Pods 狀態分佈** 圖中 Running pods 增加
4. **NetworkIntent 數量** 增加

### 4. 執行 Scale In

```bash
curl -X POST http://192.168.10.65:30081/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 3 in ns ran-a"
```

### 5. 再次觀察變化

您將看到：
1. **期望 Replicas** 從 10 降到 3
2. **就緒 Replicas** 逐步減少
3. **Pods 狀態分佈** 圖中 Running pods 減少

## ⏱️ 時間範圍調整

- 預設顯示最近 **15 分鐘**
- 可在右上角調整時間範圍：
  - 最近 5 分鐘（觀察即時變化）
  - 最近 1 小時（觀察長期趨勢）
  - 最近 24 小時（歷史分析）

## 🔄 自動刷新

- 點擊右上角的 **刷新圖標**
- 選擇自動刷新間隔：
  - 5 秒（即時監控）
  - 10 秒（推薦）
  - 30 秒（省資源）

## 📸 視覺化示例

當您提交 "scale nf-sim to 10" 後，您將看到：

```
時間序列圖：
Replicas
   10 ┤                 ╭─────────
    9 ┤               ╭─╯
    8 ┤             ╭─╯
    7 ┤           ╭─╯
    6 ┤         ╭─╯
    5 ┤       ╭─╯
    4 ┤     ╭─╯
    3 ┤   ╭─╯
    2 ┤─╮─╯
    1 ┤ ╰
    0 └─────────────────────────────
      0s   20s   40s   60s   80s  100s

      藍線：期望 Replicas（立即到達 10）
      綠線：就緒 Replicas（逐步爬升到 10）
```

## 🎨 自定義 Dashboard

### 添加更多 Panel

您可以添加：

1. **CPU 使用率**
   ```promql
   sum(rate(container_cpu_usage_seconds_total{namespace="ran-a",pod=~"nf-sim.*"}[5m]))
   ```

2. **記憶體使用**
   ```promql
   sum(container_memory_usage_bytes{namespace="ran-a",pod=~"nf-sim.*"})
   ```

3. **Pod 重啟次數**
   ```promql
   sum(kube_pod_container_status_restarts_total{namespace="ran-a",pod=~"nf-sim.*"})
   ```

### 設置告警

1. 點擊 Panel → Edit
2. 切換到 **Alert** 標籤
3. 設置條件（例如：當 ready replicas < spec replicas 超過 5 分鐘時告警）
4. 配置通知渠道（Email、Slack、Webhook 等）

## 🔗 其他監控選項

除了 Grafana，您還可以使用：

### 1. 終端實時監控
```bash
/tmp/realtime-nf-monitor.sh
```

### 2. 簡單 Watch 命令
```bash
watch -n 2 'kubectl get deployment -n ran-a nf-sim && echo "" && kubectl get pods -n ran-a -l app=nf-sim'
```

### 3. K8s Dashboard（如果已部署）
```bash
kubectl proxy --port=8001 &
# 訪問 http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/
```

## 🆘 故障排除

### Dashboard 沒有數據

1. 檢查 Prometheus 是否正常：
   ```bash
   kubectl get pods -n monitoring | grep prometheus
   ```

2. 檢查 kube-state-metrics：
   ```bash
   kubectl get pods -n monitoring | grep kube-state-metrics
   ```

3. 測試 Prometheus 查詢：
   訪問 http://192.168.10.65:30090（Prometheus UI，如果有 NodePort）
   執行查詢：`kube_deployment_spec_replicas{namespace="ran-a"}`

### Grafana 無法訪問

```bash
# 檢查 Grafana pod
kubectl get pods -n monitoring | grep grafana

# 檢查 Service
kubectl get svc -n monitoring prometheus-grafana

# 查看日誌
kubectl logs -n monitoring deployment/prometheus-grafana -c grafana
```

## 📚 參考資料

- [Grafana 官方文檔](https://grafana.com/docs/)
- [Prometheus 查詢語法](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Kubernetes Metrics](https://kubernetes.io/docs/tasks/debug-application-cluster/resource-metrics-pipeline/)

---

**更新日期**: 2026-02-26
**維護者**: Nephoran Team
