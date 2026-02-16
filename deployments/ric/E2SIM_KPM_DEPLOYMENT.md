# O-RAN SC RIC M Release - E2SIM 和 KPM xApp 部署報告

**部署日期**: 2026-02-15
**RIC Release**: M Release (2025-12-20)
**Kubernetes 版本**: 1.35.1
**節點**: thc1006-ubuntu-22 (192.168.10.65)

---

## 📋 執行摘要

成功在 O-RAN SC RIC M Release 環境中部署以下組件：

1. ✅ **E2 測試客戶端** - 用於驗證 E2 Term 連接性
2. ✅ **KPM xApp (KPI Monitor)** - Key Performance Measurement xApp (模擬版本)

**狀態**: 所有組件已部署並正常運行，可進行 E2 介面測試。

---

## 🏗️ 架構概覽

```
┌─────────────────────────────────────────────────────────────────┐
│                         RIC Platform (ricplt)                    │
├─────────────────────────────────────────────────────────────────┤
│  E2 Manager         │  E2 Termination  │  Subscription Manager  │
│  10.100.165.50:3800 │  10.100.232.16   │  4560/4561 (RMR)       │
│                     │  NodePort 32222  │                        │
└─────────────────────────────────────────────────────────────────┘
                              ↓ E2AP (SCTP)
┌─────────────────────────────────────────────────────────────────┐
│                        xApps (ricxapp)                           │
├─────────────────────────────────────────────────────────────────┤
│  E2 Test Client     │  KPM xApp (kpimon)                        │
│  10.244.0.95        │  10.244.0.96                              │
│  - E2 連接測試      │  - RMR: 4560/4561                         │
│  - DNS 驗證         │  - HTTP: 8080                             │
│  - 網絡掃描         │  - 訂閱 KPM 報告 (模擬)                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📦 已部署組件

### 1. E2 測試客戶端 (E2 Test Client)

**用途**: 驗證 RIC 平台的 E2 介面連接性和 API 可訪問性

**部署清單**: `/home/thc1006/dev/nephoran-intent-operator/deployments/ric/e2sim/e2-test-client.yaml`

**功能**:
- DNS 解析測試 (E2 Term, E2 Manager)
- E2 Manager HTTP API 訪問測試
- 網絡連接性測試
- SCTP 工具 (lksctp-tools)

**Pod 狀態**:
```
NAME                              READY   STATUS    RESTARTS   AGE
e2-test-client-59b6668d87-dmkh9   1/1     Running   0          103s
IP: 10.244.0.95
```

**驗證結果**:
```bash
# 執行測試
kubectl exec -n ricxapp deployment/e2-test-client -- bash /scripts/test-e2-connectivity.sh

結果:
✓ E2 Term DNS OK
✓ E2 Manager DNS OK
✓ E2 Manager API OK (返回 [] - 無連接的 E2 節點)
✗ E2 Term TCP not reachable (SCTP 端口不響應 TCP - 這是預期的)
```

### 2. KPM xApp (KPI Monitor)

**用途**: Key Performance Measurement xApp，用於監控和收集 RAN 性能指標

**部署清單**: `/home/thc1006/dev/nephoran-intent-operator/deployments/ric/xapps/kpm/kpm-xapp-deployment.yaml`

**功能**:
- RMR 消息接口 (與 RIC 平台通信)
- E2 訂閱管理 (通過 Subscription Manager)
- HTTP 健康檢查端點
- 配置化測量間隔和粒度

**Pod 狀態**:
```
NAME                              READY   STATUS    RESTARTS   AGE
ricxapp-kpimon-6877c9587b-qjzx4   1/1     Running   0          31s
IP: 10.244.0.96
```

**Service 端點**:
```
service-ricxapp-kpimon-http   ClusterIP   10.109.211.192   8080/TCP
service-ricxapp-kpimon-rmr    ClusterIP   10.102.95.217    4560/TCP,4561/TCP
```

**健康檢查**:
```bash
kubectl exec -n ricxapp deployment/ricxapp-kpimon -- curl -s http://localhost:8080/health

輸出:
{"status":"healthy","xapp":"kpimon","version":"1.0.0"}
```

---

## 🔧 部署命令記錄

### 步驟 1: 創建 ricxapp Namespace
```bash
kubectl create namespace ricxapp --dry-run=client -o yaml | kubectl apply -f -
```

### 步驟 2: 部署 E2 測試客戶端
```bash
kubectl apply -f /home/thc1006/dev/nephoran-intent-operator/deployments/ric/e2sim/e2-test-client.yaml
```

**資源創建**:
- ConfigMap: `e2-test-scripts` (測試腳本)
- Deployment: `e2-test-client` (1 replica)

### 步驟 3: 部署 KPM xApp
```bash
kubectl apply -f /home/thc1006/dev/nephoran-intent-operator/deployments/ric/xapps/kpm/kpm-xapp-deployment.yaml
```

**資源創建**:
- ConfigMap: `kpm-xapp-config` (配置文件和路由表)
- Service: `service-ricxapp-kpimon-rmr` (RMR 端口)
- Service: `service-ricxapp-kpimon-http` (HTTP 端口)
- Deployment: `ricxapp-kpimon` (1 replica)

---

## ✅ 驗證結果

### 1. Pod 狀態
```bash
kubectl get pods -n ricxapp -o wide
```

| Pod Name | Status | IP | Node | Age |
|----------|--------|-------|------|-----|
| e2-test-client-59b6668d87-dmkh9 | Running | 10.244.0.95 | thc1006-ubuntu-22 | 103s |
| ricxapp-kpimon-6877c9587b-qjzx4 | Running | 10.244.0.96 | thc1006-ubuntu-22 | 31s |

### 2. Service 狀態
```bash
kubectl get svc -n ricxapp
```

| Service Name | Type | Cluster-IP | Port(s) |
|--------------|------|------------|---------|
| service-ricxapp-kpimon-http | ClusterIP | 10.109.211.192 | 8080/TCP |
| service-ricxapp-kpimon-rmr | ClusterIP | 10.102.95.217 | 4560/TCP, 4561/TCP |

### 3. E2 Manager API 測試
```bash
kubectl exec -n ricxapp deployment/e2-test-client -- \
  curl -s http://service-ricplt-e2mgr-http.ricplt.svc.cluster.local:3800/v1/nodeb/states
```

**結果**: `[]` (無連接的 E2 節點 - 預期結果，因為尚未部署真實的 RAN 節點)

### 4. KPM xApp 健康檢查
```bash
kubectl exec -n ricxapp deployment/ricxapp-kpimon -- curl -s http://localhost:8080/health
```

**結果**: `{"status":"healthy","xapp":"kpimon","version":"1.0.0"}`

### 5. RIC 平台關鍵服務
```bash
kubectl get svc -n ricplt | grep -E 'e2|submgr|rtmgr'
```

| Service | Type | IP | Port(s) |
|---------|------|-------|---------|
| service-ricplt-e2mgr-http | ClusterIP | 10.100.165.50 | 3800/TCP |
| service-ricplt-e2mgr-rmr | ClusterIP | 10.107.251.91 | 4561/TCP, 3801/TCP |
| service-ricplt-e2term-sctp-alpha | NodePort | 10.100.232.16 | 36422:32222/SCTP |
| service-ricplt-submgr-rmr | ClusterIP | None | 4560/TCP, 4561/TCP |
| service-ricplt-rtmgr-rmr | ClusterIP | 10.111.72.6 | 4561/TCP, 4560/TCP |

---

## 🧪 測試場景

### 場景 1: E2 連接性測試
```bash
# 進入測試客戶端
kubectl exec -it -n ricxapp deployment/e2-test-client -- bash

# 執行完整測試
bash /scripts/test-e2-connectivity.sh

# 手動測試 E2 Manager API
curl http://service-ricplt-e2mgr-http.ricplt.svc.cluster.local:3800/v1/nodeb/states

# 測試 DNS
nslookup service-ricplt-e2term-sctp-alpha.ricplt.svc.cluster.local

# 掃描 E2 Term 端口
nmap -p 36422 service-ricplt-e2term-sctp-alpha.ricplt.svc.cluster.local
```

### 場景 2: KPM xApp 測試
```bash
# 進入 KPM xApp
kubectl exec -it -n ricxapp deployment/ricxapp-kpimon -- bash

# 檢查健康狀態
curl http://localhost:8080/health

# 檢查 E2 Manager
curl http://service-ricplt-e2mgr-http.ricplt.svc.cluster.local:3800/v1/nodeb/states

# 測試與 Subscription Manager 的連接
nc -zv service-ricplt-submgr-rmr.ricplt.svc.cluster.local 4560
```

### 場景 3: 查看日誌
```bash
# E2 測試客戶端日誌
kubectl logs -n ricxapp -l app=e2-test-client --tail=50

# KPM xApp 日誌
kubectl logs -n ricxapp -l app=ricxapp-kpimon --tail=50

# E2 Manager 日誌
kubectl logs -n ricplt -l app=ricplt-e2mgr --tail=50

# E2 Term 日誌
kubectl logs -n ricplt -l app=ricplt-e2term --tail=50
```

---

## 📝 配置文件

### KPM xApp 配置 (`config.json`)
```json
{
  "name": "kpimon",
  "version": "1.0.0",
  "messaging": {
    "ports": [
      {
        "name": "rmr-data",
        "port": 4560,
        "rxMessages": ["RIC_SUB_RESP", "RIC_INDICATION", "RIC_SUB_FAILURE"],
        "txMessages": ["RIC_SUB_REQ", "RIC_SUB_DEL_REQ"]
      }
    ]
  },
  "controls": {
    "measurement_interval": 10000,
    "granularity_period": 1000
  }
}
```

### RMR 路由表 (`local.rt`)
```
newrt|start
# RIC Subscription Request
rte|12010|service-ricplt-submgr-rmr.ricplt:4560
# RIC Subscription Response
rte|12011|-1|service-ricxapp-kpimon-rmr.ricxapp:4560
# RIC Subscription Failure
rte|12012|-1|service-ricxapp-kpimon-rmr.ricxapp:4560
# RIC Indication
rte|12050|-1|service-ricxapp-kpimon-rmr.ricxapp:4560
newrt|end
```

---

## ⚠️ 已知限制

### 1. E2SIM 映像不可用
**問題**: O-RAN SC 官方 E2SIM 映像 (`nexus3.o-ran-sc.org:10004/o-ran-sc/xapp-onboarder:1.0.0`) 不存在

**解決方案**: 使用基於 Ubuntu 的測試客戶端進行連接性驗證

**替代方案**:
- 從源碼構建 E2SIM (需要編譯依賴)
- 使用 srsRAN gNB (需要 `softwareradiosystems/srsran-project:release_avx2-latest` 映像)
- 等待 O-RAN SC 發布 M Release 兼容的映像

### 2. KPM xApp 為模擬版本
**問題**: 官方 KPM xApp 映像 (`oranscdoc/ric-app-kpimon-go`) 需要真實的 E2 節點連接

**當前狀態**: 部署了模擬版本，提供基本的健康檢查和 E2 Manager API 訪問

**生產就緒所需**:
- 真實的 RAN 節點 (gNB) 連接到 E2 Term
- RMR 庫集成
- E2AP 消息編碼/解碼
- E2 訂閱管理實現

### 3. xApp Onboarder 不可用
**問題**: xApp Onboarder 的 Ingress API 版本不兼容 Kubernetes 1.35 (已修復)，但映像仍不可用

**影響**: 無法使用標準的 xApp onboarding 流程

**解決方案**: 直接使用 Kubernetes Deployment manifest 部署 xApp

---

## 🚀 下一步驟

### 短期 (開發/測試)
1. **部署真實的 E2 節點**:
   - 選項 A: 使用 srsRAN gNB (需要構建或獲取正確的映像)
   - 選項 B: 從源碼編譯 O-RAN SC E2SIM
   - 選項 C: 使用 FlexRIC 或其他開源 E2 模擬器

2. **增強 KPM xApp**:
   - 集成 RMR 庫進行 RIC 消息傳遞
   - 實現 E2 訂閱請求 (E2AP 編碼)
   - 處理 RIC Indication 消息
   - 存儲和展示 KPM 指標

3. **監控和可觀測性**:
   - 配置 Prometheus 抓取 KPM 指標
   - 創建 Grafana 儀表板
   - 設置告警規則

### 中期 (集成)
1. **與 Nephoran Intent Operator 集成**:
   - 從 NetworkIntent CRD 觸發 KPM 訂閱
   - 基於 KPM 數據的自動擴縮容
   - 閉環控制實現

2. **部署額外的 xApps**:
   - Traffic Steering xApp
   - QoE Prediction xApp
   - Admission Control xApp

3. **多 E2 節點場景**:
   - 部署多個 gNB 模擬器
   - 測試切換和負載均衡

### 長期 (生產)
1. **使用官方映像**:
   - 等待 O-RAN SC M Release 官方映像
   - 遷移到生產級 xApp

2. **安全加固**:
   - TLS/mTLS 啟用
   - RBAC 精細化
   - 秘密管理 (Vault)

3. **高可用性**:
   - xApp 多副本部署
   - 節點親和性/反親和性
   - Pod Disruption Budgets

---

## 📚 參考資料

### O-RAN SC 文檔
- [E2 Interface Wiki](https://wiki.o-ran-sc.org/display/RICP/E2+Interface)
- [xApp Developer Guide](https://wiki.o-ran-sc.org/display/RICP/xApp+Developer+Guide)
- [RIC Platform Documentation](https://wiki.o-ran-sc.org/display/RICP)

### 源碼倉庫
- [E2 Simulator](https://gerrit.o-ran-sc.org/r/admin/repos/sim/e2-interface)
- [KPM Monitor xApp](https://github.com/o-ran-sc/ric-app-kpimon-go)
- [srsRAN Project](https://github.com/srsran/srsRAN_Project)

### 部署文件位置
- E2 測試客戶端: `/home/thc1006/dev/nephoran-intent-operator/deployments/ric/e2sim/e2-test-client.yaml`
- KPM xApp: `/home/thc1006/dev/nephoran-intent-operator/deployments/ric/xapps/kpm/kpm-xapp-deployment.yaml`
- 本報告: `/home/thc1006/dev/nephoran-intent-operator/deployments/ric/E2SIM_KPM_DEPLOYMENT.md`

---

## 🔍 故障排除

### 問題 1: Pod 無法啟動 (ImagePullBackOff)
```bash
# 檢查映像拉取錯誤
kubectl describe pod -n ricxapp <pod-name>

# 驗證映像是否存在
nerdctl pull <image-name>

# 檢查 imagePullSecrets
kubectl get secrets -n ricxapp
```

### 問題 2: E2 Manager API 不可訪問
```bash
# 檢查 E2 Manager pod
kubectl get pods -n ricplt -l app=ricplt-e2mgr

# 查看日誌
kubectl logs -n ricplt -l app=ricplt-e2mgr

# 測試服務
kubectl run test --rm -it --image=curlimages/curl -- \
  curl http://service-ricplt-e2mgr-http.ricplt:3800/v1/nodeb/states
```

### 問題 3: RMR 路由問題
```bash
# 檢查 RTMgr (Routing Manager)
kubectl logs -n ricplt -l app=ricplt-rtmgr

# 驗證路由表配置
kubectl exec -n ricxapp deployment/ricxapp-kpimon -- cat /config/local.rt

# 測試 RMR 端口
kubectl exec -n ricxapp deployment/ricxapp-kpimon -- \
  nc -zv service-ricplt-submgr-rmr.ricplt 4560
```

---

## ✅ 總結

### 成功部署
- ✅ E2 測試客戶端 - 用於驗證 RIC 平台連接性
- ✅ KPM xApp (模擬版本) - 提供基本的 xApp 框架
- ✅ 所有 Pod 狀態為 Running
- ✅ E2 Manager API 可訪問
- ✅ 健康檢查端點正常

### 待完成
- ⏳ 真實的 E2 節點連接 (需要 E2SIM 或 srsRAN gNB)
- ⏳ E2 訂閱和 KPM 報告處理
- ⏳ RMR 消息傳遞實現
- ⏳ 與 Nephoran Intent Operator 集成

### 建議
1. 優先解決 E2SIM 映像問題，建議從源碼構建或使用 srsRAN
2. 為 KPM xApp 添加真實的 RMR 和 E2AP 實現
3. 創建端到端測試場景 (E2 Setup → Subscription → Indication)
4. 集成到 CI/CD 流水線

---

**報告作者**: Nephoran DevOps Team
**最後更新**: 2026-02-15 13:35 UTC
**狀態**: ✅ 部署成功 (模擬環境)
