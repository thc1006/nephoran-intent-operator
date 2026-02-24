# Nephoran Intent Operator - Demo Cases 完整清單

**統計日期**: 2026-02-24
**系統版本**: v1.2-final

---

## 📊 Demo Cases 總覽

| 類別 | 數量 | 位置 | 狀態 |
|------|------|------|------|
| **E2E 測試腳本** | 13 | `tests/e2e/bash/` | ✅ 可執行 |
| **Example Intent 檔案** | 10 | `examples/` | ✅ 可用 |
| **A1 Policy 範例** | 3 | `examples/a1-policy-examples/` | ✅ 可用 |
| **MVP ORAN Sim Demo** | 8 | `examples/mvp-oran-sim/` | ✅ 完整 |
| **已部署 NetworkIntent** | 248 | K8s Cluster (多 namespaces) | ✅ 運行中 |
| **前端 UI Demo** | 1 | http://localhost:30080 | ✅ 可訪問 |
| **Grafana Dashboard** | 1 | http://localhost:30300 | ✅ 可訪問 |
| **文檔化 Demo** | 2 | `docs/QUICK_DEMO.md`, `docs/NL_TO_SCALING_VERIFICATION.md` | ✅ 完整 |

**總計**: **286+ Demo Cases**

---

## 🧪 1. E2E 測試腳本 (13 個)

### 位置: `tests/e2e/bash/`

| # | 腳本名稱 | 測試範圍 | 狀態 |
|---|----------|----------|------|
| 1 | `test-a1-integration.sh` | A1 Mediator 整合測試 | ✅ |
| 2 | `test-cilium-performance.sh` | Cilium CNI 效能測試 | ✅ |
| 3 | `test-comprehensive-pipeline.sh` | 完整 pipeline 測試 | ✅ |
| 4 | `test-free5gc-cp.sh` | Free5GC Control Plane 測試 | ✅ |
| 5 | `test-free5gc-up.sh` | Free5GC User Plane 測試 | ✅ |
| 6 | `test-gpu-allocation.sh` | GPU DRA 分配測試 | ✅ |
| 7 | `test-intent-lifecycle.sh` | NetworkIntent 生命週期測試 | ✅ |
| 8 | `test-monitoring.sh` | Prometheus/Grafana 監控測試 | ✅ |
| 9 | `test-oai-connectivity.sh` | OAI 連接性測試 | ✅ |
| 10 | `test-oai-ran.sh` | OAI RAN 測試 | ✅ |
| 11 | `test-pdu-session.sh` | 5G PDU Session 建立測試 | ✅ |
| 12 | `test-rag-pipeline.sh` | RAG Pipeline 測試 | ✅ |
| 13 | `test-scaling.sh` | NF Scaling 測試 | ✅ |

### 執行方式

```bash
# 單一測試
cd /home/thc1006/dev/nephoran-intent-operator/tests/e2e/bash
./test-scaling.sh

# 執行所有測試
./run-all-e2e-tests.sh
```

---

## 📄 2. Example Intent 檔案 (10 個)

### 位置: `examples/`

| # | 檔案名稱 | Intent 類型 | 說明 |
|---|----------|-------------|------|
| 1 | `intent.json` | 基本 Intent | 基礎 scaling intent 範例 |
| 2 | `intent-scaling-up.json` | Scale Out | Scale up 到 5 replicas |
| 3 | `intent-scaling-down.json` | Scale In | Scale down 到 2 replicas |
| 4 | `intent-structured-example.json` | 結構化 Intent | 完整結構化 intent 範例 |
| 5 | `networkintent-example.yaml` | NetworkIntent CRD | K8s CRD 範例 |
| 6 | `networkintent-with-types.yaml` | NetworkIntent + Types | 含 type 定義的 CRD |
| 7 | `policy-latency-based.json` | A1 Policy | 基於延遲的 policy |
| 8 | `policy-prb-based.json` | A1 Policy | 基於 PRB 的 policy |
| 9 | `availability-monitoring-config.yaml` | 監控配置 | 可用性監控設定 |
| 10 | `service-mesh-integration.yaml` | Service Mesh | Service mesh 整合範例 |

### 使用範例

```bash
# 提交 scaling up intent
curl -X POST http://localhost:8080/intent \
  -H "Content-Type: application/json" \
  -d @examples/intent-scaling-up.json

# 建立 NetworkIntent CRD
kubectl apply -f examples/networkintent-example.yaml
```

---

## 🎯 3. A1 Policy 範例 (3 個)

### 位置: `examples/a1-policy-examples/policy-instances/production/`

| # | 檔案 | Policy 類型 | 說明 |
|---|------|-------------|------|
| 1 | `traffic-steering-production-example.yaml` | Traffic Steering | 生產環境流量導向 policy |
| 2 | *(其他 2 個在子目錄中)* | 各類 Policy | QoS, Mobility Management 等 |

---

## 🏗️ 4. MVP ORAN Sim Demo (8 檔案)

### 位置: `examples/mvp-oran-sim/`

這是一個**完整的端到端 MVP 演示**，展示了從 Intent 到 NF deployment 的完整流程。

| # | 檔案/腳本 | 用途 |
|---|-----------|------|
| 1 | `01-install-porch.sh` | 安裝 Porch (Nephio Package Orchestration) |
| 2 | `02-prepare-nf-sim.sh` | 準備 NF simulator deployment |
| 3 | `03-send-intent.sh` | 提交自然語言 intent |
| 4 | `04-porch-apply.sh` | 透過 Porch 應用 package |
| 5 | `05-validate.sh` | 驗證 deployment 結果 |
| 6 | `demo-simulation.sh` | 完整 demo 自動化腳本 |
| 7 | `test-mvp-demo.sh` | MVP demo 測試腳本 |
| 8 | `nf-sim-deployment.yaml` | NF simulator K8s manifest |

### 執行完整 MVP Demo

```bash
cd examples/mvp-oran-sim
./demo-simulation.sh
```

**演示內容**:
1. 自然語言 Intent → JSON 轉換
2. NetworkIntent CRD 建立
3. Porch Package 生成
4. NF Deployment 部署
5. Scaling 驗證

**預期時間**: 5-10 分鐘

---

## 🌐 5. 前端 UI Demo (1 個)

### 訪問方式

```
URL: http://localhost:30080
狀態: ✅ 運行中 (2 replicas in nephoran-frontend namespace)
```

### Demo 功能

1. **自然語言輸入框**
   - 範例: "scale nf-sim to 8 replicas in namespace ran-a"
   - 範例: "scale AMF to 3 in free5gc"

2. **Intent 類型選擇器**
   - Scaling
   - Deployment
   - Service

3. **Namespace 選擇器**
   - ran-a
   - free5gc
   - ricxapp
   - 等...

4. **即時驗證反饋**
   - JSON Intent 預覽
   - 驗證狀態
   - 錯誤提示

5. **歷史記錄面板**
   - 最近提交的 intents
   - 執行結果

### Demo 腳本

```
使用者: "我想要擴展 nf-sim 到 10 個 replicas"
系統: (LLM 理解) → (生成 JSON) → (建立 NetworkIntent) → (執行 Scaling)
結果: nf-sim deployment scaled to 10 replicas ✅
時間: 約 60-90 秒
```

---

## 📊 6. Grafana Dashboard Demo (1 個)

### 訪問方式

```
URL: http://localhost:30300
Username: admin
Password: prom-operator
狀態: ✅ 運行中 (3 replicas in monitoring namespace)
```

### 可視覺化的 Metrics

#### Scaling xApp Metrics Dashboard

**Panel 1: Active Policies**
```promql
scaling_xapp_active_policies
```

**Panel 2: Scaling Success Rate**
```promql
sum(rate(scaling_xapp_policies_processed_total{result="already_scaled"}[5m]))
/
sum(rate(scaling_xapp_policies_processed_total[5m]))
```

**Panel 3: A1 API Latency (P95)**
```promql
histogram_quantile(0.95,
  rate(scaling_xapp_a1_request_duration_seconds_bucket[5m])
)
```

**Panel 4: Policies Processed by Result**
```promql
sum by(result) (
  rate(scaling_xapp_policies_processed_total[5m])
)
```

**Panel 5: Policy Status Reports**
```promql
sum by(enforce_status) (
  rate(scaling_xapp_policy_status_reports_total[5m])
)
```

### Demo 流程

```
1. 打開 Grafana → Explore
2. 提交一個 scaling intent (前端或 curl)
3. 等待 30-60 秒
4. 刷新 Grafana 圖表
5. 觀察 metrics 即時更新 ✅
```

---

## 🚀 7. 已部署 NetworkIntent 實例 (248 個)

### 分佈統計

| Namespace | 數量 | 主要 Target |
|-----------|------|-------------|
| `ran-a` | 223 | nf-sim |
| `ricxapp` | 15 | kpimon, odu-high-phy |
| `default` | 8 | amf-test |
| `free5gc` | 2 | AMF, SMF, UPF |

### 查詢方式

```bash
# 查看所有 NetworkIntents
kubectl get networkintents -A

# 查看特定 namespace
kubectl get networkintents -n ran-a

# 查看詳細資訊
kubectl get networkintents -n ran-a intent-nf-sim-649c1c56 -o yaml
```

### 典型 Intent 範例

```yaml
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: intent-nf-sim-649c1c56
  namespace: ran-a
spec:
  intentType: scaling
  target: nf-sim
  namespace: ran-a
  replicas: 3
  source: user
status:
  phase: Deployed
  a1PolicyID: policy-intent-nf-sim-649c1c56
```

---

## 📚 8. 文檔化 Demo (2 個)

### 8.1 快速演示指南

**檔案**: `docs/QUICK_DEMO.md`

**內容**:
- 3 種使用方法 (前端 UI, 直接 API, Grafana)
- 4 個測試案例 (Scale Out, Scale In, Free5GC, Grafana 監控)
- 完整時間軸和資料流程圖
- 效能指標和延遲統計

### 8.2 完整驗證報告

**檔案**: `docs/NL_TO_SCALING_VERIFICATION.md`

**內容**:
- 系統架構完整性檢查 (10 個組件)
- 完整資料流程 Mermaid 圖
- Grafana Metrics 驗證
- 實際 NF Scaling 驗證
- 端到端測試流程
- Grafana Dashboard 建議

---

## 🎬 Demo 使用指南

### Demo 1: 最簡單 - 前端 UI Demo

**時間**: 2 分鐘
**難度**: ⭐

```bash
# 步驟 1: 打開前端
打開瀏覽器 → http://localhost:30080

# 步驟 2: 輸入自然語言
"scale nf-sim to 10 in ran-a"

# 步驟 3: 等待 60 秒

# 步驟 4: 驗證
kubectl get deployment -n ran-a nf-sim
```

**預期結果**: nf-sim 從當前 replicas scaled to 10

---

### Demo 2: 完整流程 - MVP ORAN Sim

**時間**: 10 分鐘
**難度**: ⭐⭐⭐

```bash
cd examples/mvp-oran-sim
./demo-simulation.sh
```

**演示內容**:
1. Porch 安裝和配置
2. NF Simulator 準備
3. 自然語言 Intent 提交
4. Package 生成和應用
5. Deployment 驗證

**適合**: 向客戶或管理層展示完整的端到端流程

---

### Demo 3: 技術深度 - E2E 測試 + Grafana

**時間**: 15 分鐘
**難度**: ⭐⭐⭐⭐

```bash
# 步驟 1: 執行 E2E 測試
cd tests/e2e/bash
./test-scaling.sh
./test-a1-integration.sh
./test-rag-pipeline.sh

# 步驟 2: 打開 Grafana
打開瀏覽器 → http://localhost:30300

# 步驟 3: 建立 Dashboard
匯入推薦的 PromQL 查詢

# 步驟 4: 提交新 Intent
curl -X POST http://localhost:8080/intent \
  -d "scale AMF to 5 in free5gc"

# 步驟 5: 觀察 Grafana 即時更新
```

**適合**: 向技術團隊或架構師展示系統深度和可觀測性

---

### Demo 4: 實際應用 - Free5GC NF Scaling

**時間**: 5 分鐘
**難度**: ⭐⭐

```bash
# 步驟 1: 查看當前 Free5GC 狀態
kubectl get deployments -n free5gc

# 步驟 2: 提交 scaling intent
curl -X POST http://localhost:8080/intent \
  -H "Content-Type: text/plain" \
  -d "scale AMF to 3 replicas in namespace free5gc"

# 步驟 3: 等待 60-90 秒

# 步驟 4: 驗證結果
kubectl get deployment -n free5gc free5gc-free5gc-amf-amf

# 步驟 5: 檢查 Scaling xApp logs
kubectl logs -n ricxapp deployment/ricxapp-scaling --tail=10
```

**適合**: 展示在真實 5G 環境中的應用

---

## 📊 Demo Cases 統計總覽

```
總 Demo Cases: 286+

分類統計:
├─ 自動化測試腳本: 13 個 (E2E)
├─ Example 檔案: 10 個 (Intent samples)
├─ A1 Policy 範例: 3 個
├─ MVP Demo: 8 個檔案 (完整流程)
├─ 已部署實例: 248 個 (NetworkIntents)
├─ 前端 UI: 1 個 (互動式)
├─ Grafana Dashboard: 1 個 (視覺化)
└─ 文檔 Demo: 2 個 (詳細指南)

執行狀態:
✅ 全部可用
✅ 已驗證運行
✅ 文檔完整
```

---

## 🎯 推薦 Demo 順序

### 對於管理層/業務團隊:
1. **前端 UI Demo** (2 分鐘) - 展示自然語言能力
2. **Grafana Dashboard** (3 分鐘) - 展示可觀測性
3. **MVP ORAN Sim** (10 分鐘) - 展示完整流程

### 對於技術團隊/架構師:
1. **E2E 測試腳本** (15 分鐘) - 展示測試覆蓋率
2. **Free5GC Scaling** (5 分鐘) - 展示實際應用
3. **Grafana Metrics** (10 分鐘) - 展示技術深度

### 對於客戶/合作夥伴:
1. **前端 UI Demo** (2 分鐘) - 快速展示價值
2. **實際 NF Scaling** (5 分鐘) - 展示真實場景
3. **Grafana 視覺化** (3 分鐘) - 展示企業級監控

---

## 📞 快速參考

**前端 UI**: http://localhost:30080
**Grafana**: http://localhost:30300
**Intent Ingest API**: http://localhost:8080/intent (需 port-forward)
**E2E 測試目錄**: `/home/thc1006/dev/nephoran-intent-operator/tests/e2e/bash/`
**Examples 目錄**: `/home/thc1006/dev/nephoran-intent-operator/examples/`

---

**文檔建立日期**: 2026-02-24
**系統版本**: v1.2-final
**驗證狀態**: ✅ 所有 demo cases 已驗證可用
