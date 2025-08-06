# Nephoran Intent Operator 優化版執行指南

## 📋 執行概述

基於前期執行中遇到的 context 管理問題，本指南採用**小任務分解 + 序列執行**策略，避免複雜的多終端並行操作，確保穩定完成專案重建。

### 核心優化策略
- ✅ 單終端序列執行（取代四終端並行）
- ✅ 小任務分解（每個任務<10分鐘）
- ✅ 主動 context 管理（禁用 auto-compact）
- ✅ 增量 Git 提交（頻繁檢查點）
- ✅ 簡化提示語（減少 token 消耗）

## 🔧 環境準備

### 步驟1：確認專案狀態
```bash
# 導航到專案目錄
cd C:\Users\tingy\Desktop\nephoran-intent-operator\nephoran-intent-operator

# 檢查當前狀態
pwd
git status
git log --oneline -5
```

### 步驟2：清理之前的工作樹（如有）
```bash
# 檢查現有工作樹
git worktree list

# 清理殘留工作樹
git worktree prune
git branch -D troubleshooting-branch
git branch -D validation-branch  
git branch -D documentation-branch

# 確認清理完成
git worktree list
git branch -a
```

### 步驟3：建立 Agent 環境
```bash
# 建立 .claude/agents 目錄
mkdir -p .claude/agents

# 複製 Agent 配置檔案（確保這些檔案在專案根目錄）
copy nephoran-code-analyzer.md .claude\agents\
copy nephoran-troubleshooter.md .claude\agents\
copy nephoran-docs-specialist.md .claude\agents\

# 驗證檔案存在
dir .claude\agents\
```

### 步驟4：建立簡化的 CLAUDE.md
```bash
# 建立或更新 CLAUDE.md（簡化版本）
cat > CLAUDE.md << 'EOF'
# Nephoran Intent Operator

## 專案概述
Kubernetes 控制器專案，整合自然語言處理與 O-RAN 網路管理

## 關鍵指令
- `make build-all`: 建構所有元件
- `make test`: 執行測試套件
- `make docker-build`: 建構容器映像
- `git add . && git commit -m "檢查點"`: 儲存進度

## 工作原則
- 小任務分解，避免 context 過載
- 每完成一個子任務立即 commit
- 主動管理 context，預防警告
- 使用增量式開發方法

## 專案結構
- `api/v1/`: CRD 定義
- `pkg/controllers/`: 控制器邏輯
- `cmd/llm-processor/`: LLM 處理器服務
- `pkg/rag/`: RAG 管道實現
EOF
```

### 步驟5：建立執行狀態追蹤檔案
```bash
# 建立執行狀態追蹤
cat > EXECUTION_STATUS.md << 'EOF'
# Nephoran 優化執行狀態

## 執行策略
- 單終端序列執行
- 小任務分解（每個任務<10分鐘）
- 主動 context 管理
- 增量 Git 提交

## 階段進度
### 第一階段：基礎診斷
- [ ] 1A. 基礎結構掃描
- [ ] 1B. Go 模組檢查
- [ ] 1C. 建構系統測試

### 第二階段：問題修復
- [ ] 2A. 依賴問題修復
- [ ] 2B. CRD 實現修復
- [ ] 2C. 控制器邏輯修復

### 第三階段：功能驗證
- [ ] 3A. 建構驗證
- [ ] 3B. 基礎功能測試
- [ ] 3C. 整合測試

### 第四階段：文檔完善
- [ ] 4A. 更新 README
- [ ] 4B. 建立部署指南
- [ ] 4C. API 文檔生成

## 執行日誌
- 開始時間：[待填入]
- 當前階段：準備中
- 已完成任務：0/12
- 遇到的問題：無

## 重要發現
<!-- 執行過程中的重要發現記錄在此 -->

EOF
```

### 步驟6：初始提交
```bash
# 提交準備階段的變更
git add .
git commit -m "準備階段：建立優化版執行環境"
```

## 🚀 執行階段

### 第一階段：基礎診斷

#### 任務 1A：基礎結構掃描
```bash
# 啟動 Claude Code
claude

# 在 Claude Code 中首先禁用 auto-compact
/settings
# 確保 auto-compact 設為 false

# 執行基礎掃描任務
```

**在 Claude Code 中輸入以下提示語：**
```
執行基礎專案結構掃描：

1. 檢查專案根目錄的主要檔案（go.mod, Makefile, README.md）
2. 識別關鍵目錄結構（api/, pkg/, cmd/）
3. 確認 Git 狀態和分支情況

請將結果寫入 SCAN_1A_STRUCTURE.md 檔案，並更新 EXECUTION_STATUS.md 中的對應項目。

注意：保持任務簡潔，避免過度分析。
```

**完成後執行：**
```bash
# 檢查產生的檔案
type SCAN_1A_STRUCTURE.md
type EXECUTION_STATUS.md

# 提交進度
git add .
git commit -m "完成 1A：基礎結構掃描"

# 檢查 context 狀態，如接近限制則執行 /compact
```

#### 任務 1B：Go 模組檢查
```bash
# 繼續在 Claude Code 中執行
```

**提示語：**
```
基於 SCAN_1A_STRUCTURE.md 的結果，檢查 Go 模組狀態：

1. 分析 go.mod 中的 Go 版本和主要依賴
2. 檢查是否有明顯的版本衝突
3. 執行 `go mod tidy` 並記錄結果

將結果寫入 SCAN_1B_GOMOD.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 1B：Go 模組檢查"
```

#### 任務 1C：建構系統測試
```bash
# 繼續在 Claude Code 中執行
```

**提示語：**
```
測試專案建構系統：

1. 嘗試執行 `make build-all`
2. 記錄任何錯誤訊息
3. 檢查 Makefile 的主要目標

將結果寫入 SCAN_1C_BUILD.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 1C：建構系統測試"

# 檢查第一階段完成狀態
type EXECUTION_STATUS.md
```

### 第二階段：問題修復

#### 任務 2A：依賴問題修復
```bash
# 在 Claude Code 中切換 Agent
/agents
# 選擇 nephoran-troubleshooter
```

**提示語：**
```
使用 nephoran-troubleshooter agent，基於前面三個掃描檔案的結果修復依賴問題：

1. 讀取 SCAN_1B_GOMOD.md 和 SCAN_1C_BUILD.md
2. 修復最關鍵的依賴衝突
3. 確保 `go mod tidy` 成功執行

將修復過程記錄到 FIX_2A_DEPS.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 2A：依賴問題修復"
```

#### 任務 2B：CRD 實現修復
```bash
# 繼續使用 nephoran-troubleshooter agent
```

**提示語：**
```
修復 CRD 相關問題：

1. 檢查 api/v1/ 目錄中的 CRD 定義
2. 修復任何語法或結構問題
3. 確保 kubebuilder 標記正確

將結果記錄到 FIX_2B_CRD.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 2B：CRD 實現修復"
```

#### 任務 2C：控制器邏輯修復
```bash
# 繼續使用 nephoran-troubleshooter agent
```

**提示語：**
```
修復控制器基礎問題：

1. 檢查 pkg/controllers/ 中的主要錯誤
2. 修復編譯錯誤和明顯的邏輯問題
3. 確保基本的 reconcile 結構完整

將結果記錄到 FIX_2C_CONTROLLER.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 2C：控制器邏輯修復"
```

### 第三階段：功能驗證

#### 任務 3A：建構驗證
```bash
# 使用預設 agent（不需要特定 agent）
```

**提示語：**
```
驗證修復後的建構狀態：

1. 執行 `make build-all`
2. 檢查所有元件是否成功建構
3. 記錄任何剩餘的問題

將結果寫入 TEST_3A_BUILD.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 3A：建構驗證"
```

#### 任務 3B：基礎功能測試
```bash
# 繼續在 Claude Code 中執行
```

**提示語：**
```
執行基礎功能測試：

1. 運行 `make test`（如果存在）
2. 檢查關鍵模組的單元測試
3. 測試 CRD 的基本驗證

將結果寫入 TEST_3B_BASIC.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 3B：基礎功能測試"
```

#### 任務 3C：整合測試
```bash
# 繼續在 Claude Code 中執行
```

**提示語：**
```
執行整合測試：

1. 測試 Docker 建構（如果適用）
2. 檢查主要服務的啟動能力
3. 驗證元件間的基本整合

將結果寫入 TEST_3C_INTEGRATION.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 3C：整合測試"
```

### 第四階段：文檔完善

#### 任務 4A：更新 README
```bash
# 切換到 nephoran-docs-specialist agent
/agents
# 選擇 nephoran-docs-specialist
```

**提示語：**
```
使用 nephoran-docs-specialist agent 更新 README.md：

1. 基於實際的建構和測試結果更新 README
2. 確保所有指令都是可執行的
3. 添加必要的先決條件和安裝步驟

將更新記錄到 DOC_4A_README.md，更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 4A：README 更新"
```

#### 任務 4B：建立部署指南
```bash
# 繼續使用 nephoran-docs-specialist agent
```

**提示語：**
```
建立簡潔的部署指南：

1. 建立 DEPLOYMENT_GUIDE.md
2. 包含本地開發環境設置
3. 提供基本的部署步驟

更新 EXECUTION_STATUS.md。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 4B：部署指南建立"
```

#### 任務 4C：API 文檔生成
```bash
# 繼續使用 nephoran-docs-specialist agent
```

**提示語：**
```
生成基礎 API 文檔：

1. 記錄主要的 CRD 資源定義
2. 建立 API_REFERENCE.md
3. 包含基本的使用範例

更新 EXECUTION_STATUS.md，標記所有任務完成。
```

**完成後執行：**
```bash
git add .
git commit -m "完成 4C：API 文檔生成"
```

## 🎯 最終整合

### 步驟1：檢查完成狀態
```bash
# 檢查所有生成的檔案
dir *.md

# 驗證 EXECUTION_STATUS.md 顯示所有任務完成
type EXECUTION_STATUS.md

# 執行最終建構測試
make build-all
```

### 步驟2：建立總結報告
```bash
# 建立專案完成報告
cat > PROJECT_COMPLETION_REPORT.md << 'EOF'
# Nephoran Intent Operator 專案完成報告

## 執行概要
- 開始時間：[填入]
- 完成時間：[填入]
- 總計任務：12
- 成功完成：[填入]

## 主要成就
- [ ] 專案可成功建構
- [ ] 基礎測試通過
- [ ] 文檔完整更新
- [ ] 部署指南可用

## 檔案清單
[列出所有生成的檔案]

## 已知問題
[記錄尚未解決的問題]

## 後續建議
[建議下一步改進方向]
EOF
```

### 步驟3：最終提交和標記
```bash
# 最終提交
git add .
git commit -m "專案重建完成：Nephoran Intent Operator 全功能版本"

# 建立版本標記
git tag -a v1.0-optimized -m "使用優化版執行指南完成的穩定版本"

# 推送到遠端（如果有）
# git push origin main
# git push origin v1.0-optimized
```

## ⚠️ 注意事項與故障排除

### Context 管理
- **監控右下角的 token 計數**
- **90% 規則**：達到 90% 時執行 `/compact [簡要進度說明]`
- **緊急重置**：遇到迴圈時使用 `/clear` 然後提供簡潔的上下文重新開始

### 任務執行原則
- **時間限制**：每個任務不超過 10-15 分鐘
- **單一焦點**：每次只處理一個具體問題
- **頻繁提交**：完成任何有意義的進度立即 commit

### 常見問題處理
```bash
# Agent 切換失效
/agents
# 重新選擇所需的 agent

# Context 過載
/compact "已完成 [任務名稱]，準備執行 [下一任務]"

# 需要重新開始
/clear
# 然後提供：「繼續執行 [任務名稱]，基於之前的 [相關檔案] 結果」
```

### 成功標準
- ✅ 所有 12 個任務順利完成
- ✅ `make build-all` 成功執行
- ✅ 基礎測試通過
- ✅ 文檔完整且準確
- ✅ Git 歷史記錄清晰

這個優化版指南採用**小步快跑**的策略，確保每一步都穩定可控，避免 context 管理問題，讓您能夠順利完成 Nephoran Intent Operator 專案的重建工作。
