# Nephoran Intent Operator 與 O-RAN SC RIC 端到端整合測試

此目錄包含 Nephoran Intent Operator 與 O-RAN SC RIC 平台的端到端整合測試套件。

## 目錄結構

```
tests/e2e/
├── README.md                           # 本文件
├── INTENT_RIC_INTEGRATION_TEST.md      # 詳細測試報告
├── test-intent-ric-integration.sh      # 主測試腳本
├── cleanup.sh                          # 清理腳本
└── sample-intents/                     # 測試 Intent 範例
    ├── 01-simple-scale.yaml            # 簡單擴容測試
    ├── 02-qos-optimization.yaml        # QoS 優化測試
    └── 03-complex-intent.yaml          # 複雜多參數測試
```

## 快速開始

### 前置條件

1. **Kubernetes 集群**: 已部署並運行
2. **kubectl**: 已配置並可訪問集群
3. **Nephoran Intent Operator**: 已部署到 `nephoran-system` namespace
4. **O-RAN SC RIC**: 已部署到 `ricplt` namespace
5. **工具**: `curl`, `jq`, `kubectl`

### 運行測試

```bash
# 進入測試目錄
cd /path/to/nephoran-intent-operator/deployments/ric/tests/e2e

# 運行完整測試套件
./test-intent-ric-integration.sh

# 運行測試但保留資源（用於調試）
./test-intent-ric-integration.sh --no-cleanup
```

### 清理資源

```bash
# 使用清理腳本
./cleanup.sh

# 或手動清理
kubectl delete networkintents -n default -l test-suite=intent-ric-e2e
kubectl delete pod test-network-tools
```

## 測試階段

測試套件分為 6 個主要階段：

### 階段 1: 環境驗證 ✅
- Intent Operator Pod 運行狀態
- NetworkIntent CRD 註冊
- A1 Mediator 服務可用性
- RIC 平台組件健康狀態

### 階段 2: 網絡連接性測試 ✅
- DNS 解析測試
- A1 Mediator 健康檢查
- A1 Policy Types API 訪問
- Policy Type Schema 驗證

### 階段 3: NetworkIntent 創建與驗證 ⚠️
- 簡單擴容 Intent ✅
- QoS 優化 Intent ✅
- 複雜多參數 Intent ✅

### 階段 4: Intent Operator 日誌分析 ⚠️
- 檢查 Operator 日誌
- 錯誤計數驗證
- Reconciliation 追蹤

### 階段 5: A1 Policy 驗證 ❌
- 查詢 A1 Policies
- 驗證 Policy 格式
- 檢查 Policy 狀態
- **狀態**: A1 集成尚未實現

### 階段 6: 錯誤處理測試 ✅
- 無效 Intent 拒絕測試
- 錯誤訊息驗證

## 測試結果

最近一次測試結果 (2026-02-15):

```
總測試數: 15
通過: 13
失敗: 2
成功率: 86.67%
```

### 失敗的測試

1. **Operator Error-Free Logs**: 腳本 bug（實際無錯誤）
2. **A1 Policy Integration**: A1 集成未實現（預期失敗）

詳細結果請查看: [INTENT_RIC_INTEGRATION_TEST.md](./INTENT_RIC_INTEGRATION_TEST.md)

## 手動測試命令

### 創建測試 Intent

```bash
# 簡單擴容
kubectl apply -f sample-intents/01-simple-scale.yaml

# QoS 優化
kubectl apply -f sample-intents/02-qos-optimization.yaml

# 複雜 Intent
kubectl apply -f sample-intents/03-complex-intent.yaml
```

### 查看 Intent 狀態

```bash
# 列出所有 Intent
kubectl get networkintents -A

# 查看特定 Intent 詳情
kubectl describe networkintent e2e-test-simple-scale -n default

# 查看 Intent 狀態
kubectl get networkintent e2e-test-simple-scale -n default -o jsonpath='{.status}'
```

### 測試 A1 Mediator 連接

```bash
# 創建測試 Pod
kubectl run test-curl --rm -i --tty --image=curlimages/curl -- sh

# 在 Pod 內執行
curl -s http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000/A1-P/v2/healthcheck
curl -s http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000/A1-P/v2/policytypes
curl -s http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000/A1-P/v2/policytypes/100
```

### 查看日誌

```bash
# Intent Operator 日誌
kubectl logs -n nephoran-system -l control-plane=controller-manager --tail=100 -f

# A1 Mediator 日誌
kubectl logs -n ricplt deployment/deployment-ricplt-a1mediator --tail=100 -f

# 所有 RIC 組件日誌
kubectl logs -n ricplt --all-containers=true --tail=50
```

## NetworkIntent CRD Schema

### 支持的欄位

```yaml
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: example-intent
  namespace: default
spec:
  source: "user"                    # 必填
  intentType: "scaling"             # 可選
  target: "odu-high"                # 可選
  namespace: "ricxapp"              # 可選
  replicas: 3                       # 可選

  scalingParameters:
    replicas: 3
    autoscalingPolicy:
      minReplicas: 1
      maxReplicas: 5
      metricThresholds:
        - type: "cpu"
          value: 80
        - type: "memory"
          value: 75

  networkParameters:
    networkSliceId: "slice-001"
    qosProfile:
      priority: 1
      maximumDataRate: "1Gbps"
```

### 不支持的欄位

以下欄位在當前版本中**不支持**（將在未來版本中添加）:

- `networkParameters.qosProfile.latency`
- `networkParameters.qosProfile.jitter`
- `networkParameters.qosProfile.minimumDataRate`
- `networkParameters.qosProfile.packetLoss`
- `networkParameters.sla.*`
- `networkParameters.topology.*`

## 已知問題

### 1. A1 Mediator 集成未實現 (嚴重)

**描述**: Intent Operator 當前只驗證 NetworkIntent，但不會創建 A1 Policy。

**影響**: 無法實現端到端的 Intent → RIC 流程。

**解決方案**: 正在開發中，預計 2 週內完成。

### 2. 日誌錯誤計數 Bug (輕微)

**描述**: 測試腳本的錯誤計數邏輯有時會誤報。

**影響**: 測試報告可能顯示 "Found 00 errors"。

**解決方案**: 已修正，但可能需要重新測試驗證。

## 故障排查

### Intent 一直處於 Validated 狀態

**原因**: 這是預期行為，A1 集成尚未實現。

**驗證**:
```bash
kubectl get networkintent <name> -o jsonpath='{.status.phase}'
# 應返回: Validated
```

### A1 Mediator 連接失敗

**檢查服務**:
```bash
kubectl get svc -n ricplt service-ricplt-a1mediator-http
kubectl get pods -n ricplt -l app=ricplt-a1mediator
```

**檢查日誌**:
```bash
kubectl logs -n ricplt deployment/deployment-ricplt-a1mediator --tail=50
```

### 測試 Pod 無法啟動

**檢查**:
```bash
kubectl get pod test-network-tools
kubectl describe pod test-network-tools
```

**重新創建**:
```bash
kubectl delete pod test-network-tools
kubectl run test-network-tools --image=nicolaka/netshoot --restart=Never -- sleep 3600
```

## 下一步

1. **實現 A1 集成** (優先級: P0)
   - 添加 NetworkIntent → A1 Policy 轉換邏輯
   - 實現 A1 API 調用
   - 處理 Policy 生命週期

2. **擴展 CRD Schema** (優先級: P1)
   - 添加更多 QoS 參數
   - 支持 SLA 定義
   - 添加 Topology 信息

3. **增強測試套件** (優先級: P1)
   - 添加 A1 Policy 驗證測試
   - 實現並發測試
   - 添加性能測試

詳細計劃請參閱: [INTENT_RIC_INTEGRATION_TEST.md](./INTENT_RIC_INTEGRATION_TEST.md#下一步建議)

## 參考資料

- [詳細測試報告](./INTENT_RIC_INTEGRATION_TEST.md)
- [O-RAN A1 Interface Specification](https://orandownloadsweb.azurewebsites.net/specifications)
- [O-RAN SC RIC Platform Documentation](https://docs.o-ran-sc.org/)
- [Nephoran Intent Operator Architecture](../../../docs/architecture/)

## 聯繫方式

如有問題或建議，請聯繫:
- **問題追蹤**: GitHub Issues
- **郵件**: nephoran-dev@example.com
- **Slack**: #nephoran-testing

---

**最後更新**: 2026-02-15
**維護者**: Nephoran Test Team
