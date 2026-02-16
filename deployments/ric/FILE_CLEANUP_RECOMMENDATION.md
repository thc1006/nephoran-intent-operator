# RIC 部署文件清理建議

## 📅 分析日期
**日期**: 2026-02-15
**目的**: 清理過時文件，保留有用文檔

---

## 🗑️ 建議刪除的文件

### 臨時日誌文件
這些是部署過程中的臨時日誌，已經可以刪除：

```bash
# 刪除命令
cd /home/thc1006/dev/nephoran-intent-operator/deployments/ric

# 臨時日誌文件 (保留在 logs/ 目錄的即可)
rm -f deployment.log                    # 560 bytes, 過時
rm -f deployment-direct.log             # 3.7K, 測試日誌
rm -f deployment-live.log               # 6.9K, 測試日誌
```

**原因**:
- 這些是早期測試的日誌
- `logs/` 目錄中有更完整的日誌記錄
- 不影響當前運行狀態

---

## 🗂️ 建議保留但重新組織的文件

### 過時的 Recipe 文件
```bash
# 過時的 K Release recipes (非 M Release)
deployments/ric/recipe-k-release.yaml       # 4.7K, K Release (未使用)
deployments/ric/recipe-k135-minimal.yaml    # 2.5K, 最小化配置 (未使用)
deployments/ric/values-single-node.yaml     # 4.6K, 舊配置 (未使用)
```

**建議**: 移動到 archive/ 子目錄而不是刪除

```bash
mkdir -p archive/unused-recipes
mv recipe-k-release.yaml archive/unused-recipes/
mv recipe-k135-minimal.yaml archive/unused-recipes/
mv values-single-node.yaml archive/unused-recipes/
```

**原因**: 可能作為參考，但不應出現在主目錄

---

### 過時的部署腳本
```bash
deployments/ric/deploy-direct.sh           # 4.7K, 直接部署腳本 (未使用)
```

**建議**: 移動到 archive/ 或刪除

```bash
mv deploy-direct.sh archive/
```

**原因**: 實際使用的是 `ric-dep/bin/install` 和 `deploy-m-release.sh`

---

## ✅ 應該保留的重要文件

### 當前使用的配置
```bash
✅ recipe-m-release-k135.yaml              # 4.7K, 當前使用的 M Release 配置
✅ deploy-m-release.sh                     # 15K, 增強型部署腳本（帶 logger）
✅ ric-dep/                                # 整個目錄, O-RAN SC 官方倉庫
```

### 文檔文件
```bash
✅ README.md                               # 7.6K, 原始 README
✅ README-M-RELEASE.md                     # 6.7K, M Release 部署指南
✅ DEPLOYMENT_SUCCESS.md                   # 6.7K, 部署成功報告
✅ RIC_FUNCTIONAL_TEST.md                  # 6.7K, 功能測試報告
```

### 日誌目錄
```bash
✅ logs/                                   # 保留所有結構化日誌
   ├── ric-deploy-20260215-131204.log
   ├── ric-deploy-20260215-131233.log
   ├── ric-deploy-20260215-131256.log
   └── ric-deploy-20260215-131332.log
```

---

## 📦 其他目錄評估

### dep/ 目錄
```bash
deployments/ric/dep/                       # 16 個子目錄
```

**狀態**: ❓ 需要檢查
**建議**:
- 如果是舊的部署嘗試：移動到 archive/old-dep/
- 如果包含有用配置：保留

### repo/ 目錄
```bash
deployments/ric/repo/                      # 包含舊的安裝器
```

**狀態**: ❓ 可能過時
**建議**:
- 檢查是否與 `ric-dep/` 重複
- 如果重複則移動到 archive/

### k8s-135-patches/ 目錄
```bash
deployments/ric/k8s-135-patches/
```

**狀態**: ✅ 保留
**原因**: 包含 K8s 1.35 兼容性補丁，可能有用

---

## 🎯 推薦的最終目錄結構

```
deployments/ric/
├── README-M-RELEASE.md               # 主要文檔
├── DEPLOYMENT_SUCCESS.md             # 部署報告
├── RIC_FUNCTIONAL_TEST.md            # 測試報告
├── FILE_CLEANUP_RECOMMENDATION.md    # 本文件
├── deploy-m-release.sh               # 增強型腳本
├── recipe-m-release-k135.yaml        # 當前配置
├── ric-dep/                          # 官方倉庫
├── logs/                             # 結構化日誌
├── k8s-135-patches/                  # K8s 補丁
└── archive/                          # 歸檔
    ├── unused-recipes/
    │   ├── recipe-k-release.yaml
    │   ├── recipe-k135-minimal.yaml
    │   └── values-single-node.yaml
    ├── old-scripts/
    │   └── deploy-direct.sh
    └── old-logs/
        ├── deployment.log
        ├── deployment-direct.log
        └── deployment-live.log
```

---

## 🔧 自動化清理腳本

創建一個清理腳本：

```bash
#!/bin/bash
# cleanup-ric-deployment.sh

cd /home/thc1006/dev/nephoran-intent-operator/deployments/ric

# 創建 archive 目錄
mkdir -p archive/{unused-recipes,old-scripts,old-logs}

# 移動過時的 recipes
mv recipe-k-release.yaml archive/unused-recipes/ 2>/dev/null
mv recipe-k135-minimal.yaml archive/unused-recipes/ 2>/dev/null
mv values-single-node.yaml archive/unused-recipes/ 2>/dev/null

# 移動過時的腳本
mv deploy-direct.sh archive/old-scripts/ 2>/dev/null

# 移動臨時日誌
mv deployment.log archive/old-logs/ 2>/dev/null
mv deployment-direct.log archive/old-logs/ 2>/dev/null
mv deployment-live.log archive/old-logs/ 2>/dev/null

# 顯示清理結果
echo "✅ 清理完成！"
echo ""
echo "📁 歸檔的文件："
find archive -type f -exec ls -lh {} \;
```

---

## 📊 清理前後對比

### 清理前
- **文件數**: ~25+ 文件（主目錄）
- **冗餘配置**: 3 個過時 recipes
- **臨時日誌**: 3 個散落的日誌文件
- **清晰度**: ⭐⭐ (混亂)

### 清理後
- **文件數**: ~10 文件（主目錄）
- **冗餘配置**: 0 (全部歸檔)
- **臨時日誌**: 0 (移動到 archive/)
- **清晰度**: ⭐⭐⭐⭐⭐ (清晰)

---

## ⚠️ 不要刪除的重要目錄

### ❌ 絕對不要刪除
```bash
❌ ric-dep/                              # 官方倉庫，正在使用
❌ ric-dep/ric-common/                   # ric-common Helm package
❌ logs/                                 # 結構化日誌
❌ recipe-m-release-k135.yaml            # 當前配置
```

### ⚠️ 謹慎處理
```bash
⚠️ dep/                                  # 需要先檢查內容
⚠️ repo/                                 # 需要先檢查是否重複
```

---

## 🎯 執行建議

### 選項 1: 保守清理（推薦）
只刪除明確無用的臨時日誌：
```bash
cd /home/thc1006/dev/nephoran-intent-operator/deployments/ric
rm -f deployment.log deployment-direct.log deployment-live.log
```

### 選項 2: 完整清理
移動所有過時文件到 archive/：
```bash
cd /home/thc1006/dev/nephoran-intent-operator/deployments/ric
mkdir -p archive/{unused-recipes,old-scripts,old-logs}
# 執行上述移動命令
```

### 選項 3: 不清理
如果不確定，可以暫時保留所有文件，等確認無誤後再清理。

---

## 📝 清理檢查清單

完成清理後，驗證：
- [ ] RIC 平台仍正常運行（`kubectl get pods -n ricplt`）
- [ ] 可以重新部署（保留了 recipe 和腳本）
- [ ] 日誌可追溯（`logs/` 目錄完整）
- [ ] 文檔齊全（README 和報告都在）

---

**建議執行**: 選項 1（保守清理）
**清理後**: 記得提交到 Git（如果這些文件在版本控制中）
**最後更新**: 2026-02-15
