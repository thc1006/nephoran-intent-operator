# Nephoran Demo場景與 E2E 測試

## 真實測試：從前端提交 Intent

### 測試場景 1: 擴展 nf-sim

```bash
# 當前狀態
kubectl get deployment -n ran-a nf-sim
# 輸出: nf-sim   2/2     2            2           14h

# 在前端輸入
scale nf-sim to 5 in ns ran-a

# 驗證
kubectl get networkintents -n ran-a
kubectl get deployment -n ran-a nf-sim
# 預期: 5/5 replicas
```

### 測試場景 2: Free5GC AMF 部署

```bash
deploy free5gc-amf with 2 replicas in namespace free5gc
```

## User Stories

1. **RAN 工程師擴展容量**: 用自然語言快速擴展 NF 數量
2. **運維緊急回滾**: 快速縮減有問題的服務
3. **5G 核心網部署**: 部署新的 Network Function

