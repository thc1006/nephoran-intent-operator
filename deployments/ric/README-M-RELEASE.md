# O-RAN SC RIC M Release Deployment Guide

## 📋 概述

本目錄包含 O-RAN SC RIC M Release 的部署文件和增強型部署腳本，專為 Kubernetes 1.35.1 單節點環境設計。

## 🎯 版本信息

- **RIC Release**: M Release (2025-12-20)
- **Kubernetes**: 1.35.1 (⚠️ 未經官方測試)
- **Target Environment**: 單節點測試環境
- **容器運行時**: containerd 2.2.1

## 📦 組件版本 (M Release)

| 組件 | 版本 | 映像 |
|------|------|------|
| e2mgr | 6.0.7 | nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-e2mgr |
| rtmgr | 0.9.7 | nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-rtmgr |
| appmgr | 0.5.9 | nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-appmgr |
| submgr | 0.10.3 | nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-submgr |
| e2term | 6.0.7 | nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-e2 |
| dbaas | 0.6.5 | nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-dbaas |
| a1mediator | 3.2.3 | nexus3.o-ran-sc.org:10002/o-ran-sc/ric-plt-a1 |

## 🚀 快速開始

### 1. 執行部署腳本

增強型部署腳本包含詳細的日誌記錄和錯誤處理：

```bash
cd /home/thc1006/dev/nephoran-intent-operator/deployments/ric
./deploy-m-release.sh
```

### 2. 監控部署進度

腳本會自動：
- ✅ 執行 pre-flight 檢查
- ✅ 創建必要的 namespaces (ricplt, ricxapp, ricinfra, ricaux)
- ✅ 安裝 ric-common Helm package
- ✅ 準備並自定義 recipe 文件
- ✅ 部署 RIC 平台組件
- ✅ 驗證所有 pods 狀態
- ✅ 生成部署摘要報告

### 3. 查看日誌

所有操作都有詳細日誌記錄：

```bash
# 查看最新部署日誌
tail -f logs/ric-deploy-*.log

# 查看部署摘要
cat logs/deployment-summary.txt
```

## 📂 目錄結構

```
deployments/ric/
├── deploy-m-release.sh          # 增強型部署腳本（帶 logger）
├── recipe-m-release-k135.yaml   # M Release recipe（自動生成）
├── ric-dep/                     # O-RAN SC ric-dep 倉庫 (m-release)
│   ├── bin/install              # 原始安裝腳本
│   ├── RECIPE_EXAMPLE/          # 官方 recipe 範例
│   ├── helm/                    # Helm charts
│   └── ric-common/              # ric-common Helm package
└── logs/                        # 部署日誌目錄
    ├── ric-deploy-*.log         # 詳細部署日誌
    └── deployment-summary.txt   # 部署摘要報告
```

## 🔍 Logger 功能

部署腳本包含以下 logger 功能：

### 日誌級別
- **INFO**: 一般信息（藍色）
- **SUCCESS**: 成功操作（綠色）
- **WARN**: 警告信息（黃色）
- **ERROR**: 錯誤信息（紅色）
- **STEP**: 主要步驟（洋紅色）
- **DEBUG**: 調試信息（青色）

### 日誌輸出
- **終端輸出**: 彩色實時輸出
- **日誌文件**: 完整的時間戳記錄
- **命令追蹤**: 所有執行的命令及其輸出

### 錯誤處理
- 自動捕獲錯誤
- 收集診斷信息
- 保存 pod logs
- 記錄 Kubernetes events

## 🔧 手動操作（如需要）

### 檢查部署狀態

```bash
# 查看所有 RIC namespaces
kubectl get ns | grep ric

# 查看 ricplt pods
kubectl get pods -n ricplt

# 查看 ricinfra pods
kubectl get pods -n ricinfra

# 查看服務
kubectl get svc -n ricplt
```

### 查看 Pod 日誌

```bash
# 查看特定 pod 日誌
kubectl logs -n ricplt <pod-name>

# 查看所有 e2mgr pods
kubectl logs -n ricplt -l app=ricplt-e2mgr --tail=100

# 跟蹤日誌
kubectl logs -n ricplt -f <pod-name>
```

### 檢查 ConfigMaps

```bash
kubectl get cm -n ricplt
kubectl describe cm -n ricplt <configmap-name>
```

## ⚠️ 已知風險與注意事項

### Kubernetes 1.35.1 兼容性

- ⚠️ **M Release 官方測試環境是 K8s 1.32.8**
- ⚠️ **K8s 1.35.1 沒有官方測試記錄**
- ⚠️ **您將是此版本組合的先行者**

### 可能遇到的問題

1. **API 版本不兼容**
   - 症狀: CRD 創建失敗或 pod 無法啟動
   - 解決: 檢查 K8s 1.35 API deprecations

2. **網絡策略問題**
   - 症狀: Pod 之間無法通訊
   - 解決: 檢查 CNI 插件版本，確認 NetworkPolicy 支持

3. **資源限制**
   - 症狀: Pod 處於 Pending 狀態
   - 解決: 檢查 node resources，調整 recipe 中的資源請求

4. **映像拉取失敗**
   - 症狀: ImagePullBackOff 錯誤
   - 解決: 確認 nexus3.o-ran-sc.org 可訪問

## 🐛 故障排除

### 1. Pods 不是 Running 狀態

```bash
# 檢查 pod 狀態詳情
kubectl describe pod -n ricplt <pod-name>

# 查看 events
kubectl get events -n ricplt --sort-by='.lastTimestamp'

# 查看 pod 日誌
kubectl logs -n ricplt <pod-name> --previous
```

### 2. ric-common 安裝失敗

```bash
# 檢查 Helm repo
helm repo list

# 手動安裝 ric-common
cd ric-dep/ric-common
helm package .
helm repo add local file://$HOME/.helm/repository/local
cp ric-common-*.tgz ~/.helm/repository/local/
helm repo update
```

### 3. 部署腳本失敗

```bash
# 查看詳細日誌
cat logs/ric-deploy-*.log

# 檢查最後幾行錯誤
tail -50 logs/ric-deploy-*.log

# 手動清理並重試
kubectl delete ns ricplt ricinfra --ignore-not-found
./deploy-m-release.sh
```

## 📊 驗證部署成功

部署成功的標誌：

```bash
# 所有 ricplt pods 應該是 Running
kubectl get pods -n ricplt
NAME                                    READY   STATUS    RESTARTS   AGE
r4-e2mgr-...                           1/1     Running   0          5m
r4-rtmgr-...                           1/1     Running   0          5m
r4-appmgr-...                          1/1     Running   0          5m
r4-submgr-...                          1/1     Running   0          5m
r4-e2term-alpha-...                    1/1     Running   0          5m
r4-dbaas-server-...                    1/1     Running   0          5m

# 檢查服務
kubectl get svc -n ricplt
```

## 📝 下一步

部署完成後：

1. **驗證 E2 termination**
   ```bash
   kubectl logs -n ricplt -l app=ricplt-e2term
   ```

2. **測試 A1 mediator**
   ```bash
   kubectl port-forward -n ricplt svc/r4-a1mediator 10000:10000
   curl http://localhost:10000/a1-p/healthcheck
   ```

3. **部署 xApps** (可選)
   - 使用 appmgr 部署自定義 xApps

4. **集成 Intent Operator**
   - 配置 NetworkIntent 與 RIC 平台通訊

## 🔗 參考資料

- [O-RAN SC M Release Documentation](https://docs.o-ran-sc.org/)
- [RIC Platform Installation Guide](https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-ric-dep/en/latest/installation-guides.html)
- [O-RAN SC Gerrit Repository](https://gerrit.o-ran-sc.org/r/ric-plt/ric-dep)

## 📞 支持

如遇到問題：
1. 檢查 `logs/ric-deploy-*.log` 詳細日誌
2. 參考故障排除章節
3. 如果是 K8s 1.35 兼容性問題，考慮回報給 O-RAN SC 社群

---

**最後更新**: 2026-02-15
**維護者**: Nephoran Intent Operator Team
