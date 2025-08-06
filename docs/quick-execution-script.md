# 快速執行腳本：Nephoran Multi-Agent 協調

## 準備工作

### 1. 環境設置
```bash
# 確保 Claude Code 已安裝並配置
claude --version

# 進入專案目錄
cd /path/to/nephoran-intent-operator

# 確保 git 狀態乾淨
git status
git add .
git commit -m "開始 multi-agent 協調之前的狀態"
```

### 2. Agent 配置準備
```bash
# 建立 .claude/agents 目錄（如果不存在）
mkdir -p .claude/agents

# 複製您的 agent 檔案到正確位置
cp nephoran-code-analyzer.md .claude/agents/
cp nephoran-troubleshooter.md .claude/agents/
cp nephoran-docs-specialist.md .claude/agents/
```

### 3. 建立共享狀態文件
```bash
# 建立多 Agent 協調計劃文件
cat > MULTI_AGENT_PLAN.md << 'EOF'
# Nephoran Multi-Agent 執行狀態

## 當前階段：等待開始

### 階段執行狀態
- [ ] 階段 1: 架構分析 (nephoran-code-analyzer)
  - 分析時間：
  - 狀態：未開始
  - 輸出檔案：ARCHITECTURE_ANALYSIS.md

- [ ] 階段 2: 問題修復 (nephoran-troubleshooter)  
  - 修復時間：
  - 狀態：等待階段 1 完成
  - 輸出檔案：TROUBLESHOOTING_LOG.md

- [ ] 階段 3: 功能驗證 (並行執行)
  - 驗證時間：
  - 狀態：等待階段 2 完成
  - 輸出檔案：VALIDATION_REPORT.md

- [ ] 階段 4: 文檔生成 (nephoran-docs-specialist)
  - 生成時間：
  - 狀態：等待前序階段完成
  - 輸出檔案：完整部署指南

### 關鍵發現
<!-- 各階段在此記錄重要發現 -->

### 待解決問題
<!-- 需要跨階段協調的問題 -->

### 最後更新：開始執行
EOF
```

## 執行階段

### 階段 1：架構分析
```bash
# Terminal 1 - 開啟第一個 Claude Code 實例
# 確保在專案根目錄
pwd  # 應該是 nephoran-intent-operator 目錄

# 啟動 Claude Code
claude

# 在 Claude Code 中執行以下命令：
```

**在 Claude Code 中輸入：**
```
/agents

# 選擇 nephoran-code-analyzer，然後執行：

請使用 nephoran-code-analyzer agent 執行完整的專案架構分析：

1. **依賴相容性檢查**：
   - 分析 go.mod 中的 Go 版本和依賴相容性
   - 檢查是否有版本衝突或過時的依賴
   - 驗證 Kubernetes 相關依賴的版本一致性

2. **CRD 實現分析**：
   - 檢查 api/v1/ 中的 NetworkIntent 和 E2NodeSet 類型定義
   - 驗證 CRD 的 kubebuilder 標記和驗證規則
   - 分析 controller-runtime 整合的正確性

3. **控制器邏輯評估**：
   - 分析 pkg/controllers/ 中的 reconcile 邏輯實現
   - 檢查錯誤處理和重試機制
   - 驗證 Kubernetes 事件和狀態管理

4. **LLM 整合架構檢查**：
   - 分析 cmd/llm-processor/ 的服務實現
   - 檢查 API 端點定義和錯誤處理
   - 驗證與 OpenAI API 的整合模式

5. **RAG 管道驗證**：
   - 檢查 pkg/rag/ 中的向量資料庫整合
   - 分析 Weaviate 連接池和查詢邏輯
   - 評估文檔處理和檢索性能

6. **建構系統分析**：
   - 檢查 Makefile 中的所有目標定義
   - 驗證 Docker 建構和部署配置
   - 分析跨平台相容性實現

請將完整分析結果寫入 ARCHITECTURE_ANALYSIS.md 檔案，並更新 MULTI_AGENT_PLAN.md 中的階段 1 狀態。
```

### 階段 2：問題修復
```bash
# Terminal 2 - 建立新的工作樹用於修復
git worktree add ../nephoran-troubleshooting troubleshooting-branch
cd ../nephoran-troubleshooting

# 啟動第二個 Claude Code 實例  
claude
```

**在第二個 Claude Code 中輸入：**
```
/agents

# 選擇 nephoran-troubleshooter，然後執行：

請讀取主專案目錄中的 ARCHITECTURE_ANALYSIS.md 檔案，並基於分析結果執行系統性問題修復：

1. **Go 建構問題修復**：
   - 修復所有依賴衝突和版本不相容問題
   - 解決 import cycle 錯誤
   - 確保所有 Go 模組正確編譯

2. **CRD 註冊修復**：
   - 修正任何 CRD 定義或驗證錯誤
   - 確保 API 伺服器正確識別自訂資源
   - 修復 kubebuilder 生成的程式碼問題

3. **控制器實現修復**：
   - 修正 reconcile 邏輯中的錯誤
   - 完善事件處理和狀態更新
   - 確保正確的資源生命週期管理

4. **LLM 處理器修復**：
   - 解決 API 連接和超時問題
   - 修復 JSON 序列化/反序列化問題
   - 完善錯誤處理和重試邏輯

5. **RAG 系統最佳化**：
   - 修復 Weaviate 連接和查詢問題
   - 最佳化文檔處理效能
   - 解決向量檢索的準確性問題

6. **建構和部署修復**：
   - 修復 Makefile 中的錯誤目標
   - 解決 Docker 建構失敗問題
   - 確保跨平台部署相容性

對於每個修復：
- 在修復前先執行測試以確認問題
- 進行最小變更修復根本原因
- 修復後驗證功能正常
- 記錄修復步驟和解決方案

請將所有修復活動記錄到 TROUBLESHOOTING_LOG.md，並更新 MULTI_AGENT_PLAN.md 中的進度。

修復完成後請執行：
git add .
git commit -m "階段 2 完成：問題診斷和修復"
```

### 階段 3：功能驗證
```bash
# Terminal 3 - 建立驗證分支
git worktree add ../nephoran-validation validation-branch
cd ../nephoran-validation

# 啟動第三個 Claude Code 實例
claude
```

**在第三個 Claude Code 中輸入：**
```
請執行完整的功能驗證測試，確保所有 README.md 中宣告的功能都能正常工作：

1. **建構系統驗證**：
   - 執行 `make build-all` 並確保所有元件成功建構
   - 測試 `make docker-build` 的容器建構
   - 驗證 `make test-integration` 的測試套件

2. **CRD 生命週期測試**：
   - 測試 NetworkIntent 資源的建立、更新、刪除
   - 驗證 E2NodeSet 的副本管理功能
   - 確認控制器正確回應資源變更

3. **LLM 處理器測試**：
   - 測試自然語言意圖的解析功能
   - 驗證 API 端點的回應和錯誤處理
   - 確認與 OpenAI API 的正確整合

4. **RAG 系統驗證**：
   - 測試知識庫的查詢和檢索功能
   - 驗證文檔處理和向量化
   - 確認 Weaviate 整合的效能

5. **端到端工作流程測試**：
   - 測試完整的意圖處理到部署流程
   - 驗證 GitOps 工作流程的正確性
   - 確認所有微服務間的整合

6. **部署測試**：
   - 在本地 Kubernetes 環境測試部署
   - 驗證所有 Pod 能正常啟動和執行
   - 測試服務間的網路連通性

對於每個測試：
- 記錄測試步驟和期望結果
- 記錄實際結果和任何偏差
- 對失敗的測試提供詳細錯誤資訊
- 建議修復或改進方案

請將所有驗證結果記錄到 VALIDATION_REPORT.md，並更新 MULTI_AGENT_PLAN.md。

驗證完成後請執行：
git add .
git commit -m "階段 3 完成：功能驗證測試"
```

### 階段 4：文檔生成
```bash  
# Terminal 4 - 建立文檔分支
git worktree add ../nephoran-documentation documentation-branch
cd ../nephoran-documentation

# 啟動第四個 Claude Code 實例
claude
```

**在第四個 Claude Code 中輸入：**
```
/agents

# 選擇 nephoran-docs-specialist，然後執行：

請基於前三個階段的所有結果，生成完整且準確的專案文檔：

1. **更新 CLAUDE.md**：
   - 反映實際的專案實現狀態
   - 包含所有可工作的命令和工作流程
   - 添加針對電信和 O-RAN 領域的專業指導

2. **使用現有部署指南**：
   - 參考 docs/NetworkIntent-Controller-Guide.md 獲取詳細的逐步部署說明
   - 參考 docs/GitOps-Package-Generation.md 獲取 GitOps 相關部署資訊
   - 包含所有先決條件和環境設置
   - 提供故障排除指南和常見問題解決方案
   - 包含本地和雲端部署選項

3. **生成 API_DOCUMENTATION.md**：
   - 記錄所有 REST API 端點
   - 包含請求/回應範例
   - 提供認證和錯誤處理指導
   - 包含效能特性和限制

4. **建立 DEVELOPER_GUIDE.md**：
   - 新開發者快速上手指南
   - 程式碼結構和架構說明
   - 開發工作流程和最佳實踐
   - 測試和除錯指導

5. **更新 README.md**：
   - 確保所有功能描述準確無誤
   - 更新安裝和使用說明
   - 添加實際的使用範例
   - 包含效能指標和系統需求

6. **建立 TROUBLESHOOTING_REFERENCE.md**：
   - 整合所有階段發現的問題和解決方案
   - 提供診斷工具和方法
   - 包含效能調優建議
   - 添加監控和日誌分析指導

要求：
- 所有文檔必須反映實際可工作的實現
- 包含可驗證的步驟和範例
- 使用清晰的 Markdown 格式
- 確保文檔間的一致性

請將所有生成的文檔放在適當位置，並更新 MULTI_AGENT_PLAN.md 標記專案完成。

文檔完成後請執行：
git add .
git commit -m "階段 4 完成：完整文檔生成"
```

## 結果整合

### 最終整合步驟
```bash
# 回到主專案目錄
cd /path/to/nephoran-intent-operator

# 整合所有分支的變更
git checkout main
git merge troubleshooting-branch
git merge validation-branch  
git merge documentation-branch

# 最終驗證
make build-all
make test-integration

# 如果一切正常，標記完成
git tag -a v1.0-complete -m "Multi-agent coordination complete - fully functional"
git push origin v1.0-complete

# 清理工作樹
git worktree remove ../nephoran-troubleshooting
git worktree remove ../nephoran-validation
git worktree remove ../nephoran-documentation
```

### 成功標準檢查
- [ ] 所有 make 目標成功執行
- [ ] 完整的端到端測試通過
- [ ] 所有 README.md 功能已實現並可驗證
- [ ] 完整的部署文檔可讓新用戶成功搭建
- [ ] 所有生成的文檔保持一致性和準確性

## 故障排除

### 如果 Agent 失去上下文：
```bash
# 提示重新讀取狀態
"請重新讀取 MULTI_AGENT_PLAN.md 和相關的輸出檔案，然後繼續您的任務"
```

### 如果遇到衝突：
```bash
# 讓架構分析 Agent 做最終決定
"請 nephoran-code-analyzer agent 檢查衝突並提供最終架構決策"
```

### 如果需要重新開始某個階段：
```bash
# 重置特定分支
git worktree remove ../nephoran-[stage-name]
git branch -D [stage-name]-branch
git worktree add ../nephoran-[stage-name] [stage-name]-branch
```

這個快速執行指南提供了具體的命令和步驟，讓您能夠有效協調多個 Claude Code Agent 來完成複雜的 Nephoran Intent Operator 專案重建任務。