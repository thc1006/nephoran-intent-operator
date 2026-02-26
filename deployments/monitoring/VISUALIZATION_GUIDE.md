# NF Scale Out/In 視覺化監控完整指南

**版本**: 1.0
**更新日期**: 2026-02-26
**適用對象**: Nephoran Intent Operator 用戶

---

## 🎯 概述

本指南提供**4 種視覺化監控方案**，讓您可以實時觀察 Network Function (NF) 的 Scale Out/In 過程。

---

## 📊 方案比較

| 方案 | 視覺效果 | 技術難度 | 推薦用途 | 啟動時間 |
|------|---------|---------|---------|---------|
| **Grafana Dashboard** | ⭐⭐⭐⭐⭐ | 中 | 生產環境長期監控 | 5 分鐘 |
| **實時終端監控** | ⭐⭐⭐⭐ | 低 | 開發測試即時觀察 | 10 秒 |
| **簡易 Watch** | ⭐⭐⭐ | 極低 | 快速檢查 | 5 秒 |
| **Web 前端集成** | ⭐⭐⭐⭐ | 中 | 用戶友好界面 | 待開發 |

---

## 🚀 方案 1：Grafana Dashboard（推薦）

### 🎨 特色

- ✅ 專業級時間序列圖表
- ✅ 多維度數據展示（Replicas、Pods、NetworkIntent）
- ✅ 歷史數據回溯（可查看過去任意時間）
- ✅ 自動刷新（5 秒 - 1 分鐘可調）
- ✅ 告警功能（可設置閾值告警）
- ✅ 數據導出（PNG、CSV、JSON）

### 📍 訪問方式

```
URL: http://192.168.10.65:30300
帳號: admin
密碼: NephoranMonitor2026!
```

### 🎯 快速開始

**步驟 1: 訪問 Grafana**
```bash
# 在瀏覽器打開
open http://192.168.10.65:30300  # Mac
xdg-open http://192.168.10.65:30300  # Linux
```

**步驟 2: 導入 Dashboard**
```bash
# 方法 A: 透過 UI 上傳 /tmp/grafana-nf-dashboard.json

# 方法 B: 使用 API 自動導入
curl -X POST http://admin:NephoranMonitor2026!@192.168.10.65:30300/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @/tmp/grafana-nf-dashboard.json
```

**步驟 3: 設置自動刷新**
- 點擊右上角時鐘圖標
- 選擇刷新間隔：**10 秒**（推薦）
- 時間範圍：**最近 15 分鐘**

**步驟 4: 測試觀察**

打開兩個窗口：
- **窗口 1**: Grafana Dashboard
- **窗口 2**: 終端機

在終端執行：
```bash
# Scale Out
curl -X POST http://192.168.10.65:30081/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 8 in ns ran-a"
```

您將在 Grafana 看到：
1. 🔵 **期望 Replicas** 線條立即跳到 8
2. 🟢 **就緒 Replicas** 線條逐步爬升到 8（約 30-60 秒）
3. 📊 **Pods 狀態** 條形圖中 Running pods 增加
4. 📈 **NetworkIntent 數量** 增加

### 📸 視覺化效果

```
時間序列圖範例：

Replicas
   10 ┤                    ╭────────
    9 ┤                  ╭─╯
    8 ┤                ╭─╯
    7 ┤              ╭─╯
    6 ┤            ╭─╯
    5 ┤          ╭─╯
    4 ┤        ╭─╯
    3 ┤      ╭─╯
    2 ┤────╮─╯
    1 ┤
    0 └──────────────────────────────
      0s    20s   40s   60s   80s  100s

藍線（期望）：立即變化
綠線（就緒）：逐步跟進
```

### 📋 Dashboard 包含的 Panel

1. **NF-SIM Deployment Replicas 趨勢** (時間序列)
   - 期望 Replicas
   - 就緒 Replicas
   - 可用 Replicas

2. **期望 Replicas** (儀表盤)
   - 當前目標值

3. **就緒 Replicas** (儀表盤)
   - 當前就緒數量
   - 顏色編碼（紅/黃/綠）

4. **Pods 狀態分佈** (堆疊條形圖)
   - Running / Pending / Failed

5. **NetworkIntent CRD 數量** (條形圖)
   - Intent 資源總數

### 🔗 完整指南

詳見：`deployments/monitoring/GRAFANA_SETUP_GUIDE.md`

---

## 🖥️ 方案 2：實時終端監控（開發推薦）

### 🎨 特色

- ✅ 彩色終端輸出
- ✅ ASCII 藝術趨勢圖
- ✅ 實時刷新（2 秒間隔）
- ✅ 歷史記錄（最近 20 次）
- ✅ 無需瀏覽器

### 📍 使用方式

```bash
/tmp/realtime-nf-monitor.sh
```

### 🎯 效果預覽

```
╔═══════════════════════════════════════════════════════════════════╗
║                  NF 實時監控 - Scale Out/In 視覺化                ║
╚═══════════════════════════════════════════════════════════════════╝

⏰ 更新時間: 2026-02-26 19:30:45   刷新間隔: 2s

📦 Deployment: nf-sim
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  狀態: ✓ 穩定

  期望 Replicas:    7
  就緒 Replicas:    7 / 7
  可用 Replicas:    7
  已更新 Replicas:  7

  Progress:
  [██████████████████████████████████████████████████] 7/7

🔵 Pods 狀態
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ● Running:     7

最新 Pods (前 5):
  ▸ nf-sim-7757c6bd66-abc12   1/1   Running   0   2m
  ▸ nf-sim-7757c6bd66-def34   1/1   Running   0   2m
  ▸ nf-sim-7757c6bd66-ghi56   1/1   Running   0   2m
  ▸ nf-sim-7757c6bd66-jkl78   1/1   Running   0   2m
  ▸ nf-sim-7757c6bd66-mno90   1/1   Running   0   2m

📋 NetworkIntent CRD
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  總數: 2 個資源

📈 歷史趨勢
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Replicas 趨勢 (最近 20 次):
 7 │ ████████████████████
 6 │ ████████
 5 │ ████
 4 │ ██
 3 │ █
 2 │ ██████
 1 │
 0 │
   └────────────────────

時間軸 (最近 5 次):
  19:30:35 → spec: 2, ready: 2
  19:30:37 → spec: 7, ready: 2
  19:30:39 → spec: 7, ready: 4
  19:30:41 → spec: 7, ready: 6
  19:30:43 → spec: 7, ready: 7

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 提示: 在另一個終端執行 intent 提交觀察變化
🛑 停止: 按 Ctrl+C 退出監控
```

### 🎬 使用流程

**終端 1: 啟動監控**
```bash
/tmp/realtime-nf-monitor.sh
```

**終端 2: 提交 Intent**
```bash
# Scale Out
curl -X POST http://192.168.10.65:30081/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 10 in ns ran-a"

# 等待 60 秒觀察

# Scale In
curl -X POST http://192.168.10.65:30081/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 3 in ns ran-a"
```

**在終端 1 觀察**：
- 期望 Replicas 立即變化
- 就緒 Replicas 逐步變化
- Pods 列表實時更新
- 趨勢圖動態繪製

---

## 🔍 方案 3：簡易 Watch 監控

### 🎨 特色

- ✅ 極簡單（單一命令）
- ✅ 標準 kubectl 輸出
- ✅ 無需安裝額外工具

### 📍 使用方式

**選項 A: 預製腳本**
```bash
/tmp/simple-watch-monitor.sh
```

**選項 B: 直接 watch**
```bash
watch -n 2 'kubectl get deployment -n ran-a nf-sim && echo "" && kubectl get pods -n ran-a -l app=nf-sim'
```

**選項 C: 只看 Deployment**
```bash
watch -n 2 kubectl get deployment -n ran-a nf-sim -o wide
```

### 🎯 效果

```
Every 2.0s: kubectl get deployment...

NAME     READY   UP-TO-DATE   AVAILABLE   AGE
nf-sim   7/7     7            7           3d

NAME                      READY   STATUS    RESTARTS   AGE
nf-sim-7757c6bd66-abc12   1/1     Running   0          2m
nf-sim-7757c6bd66-def34   1/1     Running   0          2m
nf-sim-7757c6bd66-ghi56   1/1     Running   0          2m
nf-sim-7757c6bd66-jkl78   1/1     Running   0          2m
nf-sim-7757c6bd66-mno90   1/1     Running   0          2m
nf-sim-7757c6bd66-pqr01   1/1     Running   0          2m
nf-sim-7757c6bd66-stu23   1/1     Running   0          2m
```

---

## 🌐 方案 4：Web 前端集成（未來開發）

### 🎨 規劃功能

- ✅ 在前端 UI 嵌入實時監控面板
- ✅ WebSocket 實時推送
- ✅ 提交 Intent 後立即顯示進度條
- ✅ 動畫效果（Pods 創建/刪除動畫）

### 📍 實現方式

在 `deployments/frontend/index.html` 中添加監控面板：

```html
<!-- 監控面板 -->
<div class="monitoring-panel">
    <h3>實時監控</h3>
    <div class="replicas-chart" id="replicasChart"></div>
    <div class="pods-list" id="podsList"></div>
</div>

<script>
// WebSocket 連接
const ws = new WebSocket('ws://192.168.10.65:30081/ws/monitor');

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    updateChart(data.replicas);
    updatePodsList(data.pods);
};
</script>
```

**狀態**: 待實現（需要後端 WebSocket 支持）

---

## 🧪 完整測試流程

### 準備工作

1. **啟動 Grafana Dashboard**
   ```bash
   # 瀏覽器打開
   open http://192.168.10.65:30300
   # 登入並導入 dashboard
   ```

2. **啟動終端監控**
   ```bash
   # 終端 1
   /tmp/realtime-nf-monitor.sh
   ```

3. **準備提交 Intent**
   ```bash
   # 終端 2（待命）
   ```

### 測試 Scale Out

**終端 2 執行**:
```bash
curl -X POST http://192.168.10.65:30081/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 12 in ns ran-a"
```

**同時觀察**:
- **Grafana**: 藍色線條立即跳到 12，綠色線條逐步爬升
- **終端監控**: 趨勢圖實時更新，Pods 列表增加
- **預期時間**: 40-80 秒完成（取決於 node 資源）

### 測試 Scale In

**等待 60 秒後，終端 2 執行**:
```bash
curl -X POST http://192.168.10.65:30081/intent \
  -H "Content-Type: text/plain" \
  -d "scale nf-sim to 2 in ns ran-a"
```

**同時觀察**:
- **Grafana**: 藍色線條立即降到 2，綠色線條逐步下降
- **終端監控**: Pods 列表減少，出現 Terminating 狀態
- **預期時間**: 10-30 秒完成（刪除比創建快）

---

## 📊 數據指標說明

### Deployment Metrics

| 指標 | 說明 | Prometheus Query |
|-----|------|------------------|
| **spec.replicas** | 期望的 replica 數量 | `kube_deployment_spec_replicas` |
| **status.readyReplicas** | 就緒的 replica 數量 | `kube_deployment_status_replicas_ready` |
| **status.availableReplicas** | 可用的 replica 數量 | `kube_deployment_status_replicas_available` |
| **status.updatedReplicas** | 已更新的 replica 數量 | `kube_deployment_status_replicas_updated` |

### Pod Metrics

| 指標 | 說明 | Prometheus Query |
|-----|------|------------------|
| **Running** | 運行中的 pods | `sum(kube_pod_status_phase{phase="Running"})` |
| **Pending** | 等待中的 pods | `sum(kube_pod_status_phase{phase="Pending"})` |
| **Failed** | 失敗的 pods | `sum(kube_pod_status_phase{phase="Failed"})` |

### NetworkIntent Metrics

| 指標 | 說明 | 查詢方式 |
|-----|------|---------|
| **總數** | NetworkIntent 資源數量 | `kubectl get networkintents -n ran-a \| wc -l` |

---

## ⚙️ 進階配置

### 自定義刷新間隔

**Grafana**:
```
右上角 → 刷新圖標 → 自定義間隔
最小：1 秒（高負載）
推薦：10 秒
最大：5 分鐘
```

**終端監控**:
編輯 `/tmp/realtime-nf-monitor.sh`，修改：
```bash
REFRESH_INTERVAL=2  # 改為你想要的秒數
```

### 添加告警

**Grafana 告警設置**:
1. 編輯 Panel → Alert 標籤
2. 設置條件：
   ```
   WHEN avg() OF query(A, 5m, now) IS BELOW 2
   ```
3. 通知渠道：Email / Slack / Webhook

### 導出監控數據

**Grafana 導出**:
- PNG: Panel 右上角 → Share → Export → PNG
- CSV: Panel 右上角 → Inspect → Data → Download CSV
- JSON: Settings → JSON Model → Copy

**Prometheus 原始數據**:
```bash
# 查詢最近 1 小時的數據
curl 'http://192.168.10.65:30090/api/v1/query_range?query=kube_deployment_spec_replicas{namespace="ran-a"}&start=2026-02-26T18:00:00Z&end=2026-02-26T19:00:00Z&step=60s'
```

---

## 🆘 故障排除

### Grafana 沒有數據

**檢查 Prometheus**:
```bash
kubectl get pods -n monitoring | grep prometheus
kubectl logs -n monitoring prometheus-prometheus-kube-prometheus-prometheus-0
```

**檢查 kube-state-metrics**:
```bash
kubectl get pods -n monitoring | grep kube-state-metrics
kubectl logs -n monitoring deployment/prometheus-kube-state-metrics
```

**測試查詢**:
```bash
curl http://192.168.10.65:30090/api/v1/query?query=kube_deployment_spec_replicas
```

### 終端監控無輸出

**檢查 kubectl 訪問**:
```bash
kubectl get deployment -n ran-a nf-sim
```

**檢查腳本權限**:
```bash
chmod +x /tmp/realtime-nf-monitor.sh
```

### Watch 命令失敗

**安裝 watch**:
```bash
# Ubuntu/Debian
sudo apt-get install procps

# macOS
brew install watch
```

---

## 📚 相關資源

- **Grafana Dashboard JSON**: `/tmp/grafana-nf-dashboard.json`
- **實時監控腳本**: `/tmp/realtime-nf-monitor.sh`
- **簡易監控腳本**: `/tmp/simple-watch-monitor.sh`
- **Grafana 設置指南**: `deployments/monitoring/GRAFANA_SETUP_GUIDE.md`
- **E2E 測試腳本**: `/tmp/auto-clean-test.sh`

---

## 🎓 最佳實踐

1. **日常開發**: 使用終端實時監控（快速啟動，即時反饋）
2. **生產環境**: 使用 Grafana Dashboard（專業、穩定、歷史回溯）
3. **快速檢查**: 使用 simple watch（極簡，無需準備）
4. **演示展示**: Grafana + 實時監控雙屏展示（視覺衝擊力強）

---

**版本**: 1.0
**維護者**: Nephoran Team
**最後更新**: 2026-02-26
