# 瀏覽器前端功能測試指南

**版本**: K8s Dashboard 增強版
**更新日期**: 2026-02-26
**訪問地址**:
- NodePort: http://192.168.10.65:30081
- Ngrok HTTPS: https://lennie-unfatherly-profusely.ngrok-free.dev

---

## ⚠️ 如果頁面看起來不像 K8s Dashboard

### 問題：瀏覽器快取

可能您的瀏覽器快取了舊版本（Cyber-Terminal）。請按照以下步驟清除：

#### Chrome / Edge
1. 按 `Ctrl+Shift+Delete` 打開清除瀏覽資料視窗
2. 選擇「快取的圖片和檔案」
3. 時間範圍選「過去 1 小時」
4. 點擊「清除資料」
5. 或者直接按 `Ctrl+F5` 強制重新載入

#### Firefox
1. 按 `Ctrl+Shift+Delete`
2. 勾選「快取」
3. 點擊「立即清除」
4. 或者直接按 `Ctrl+Shift+R` 強制重新載入

#### Safari
1. 按 `Cmd+Option+E` 清空快取
2. 或在「開發」選單選擇「清空快取」
3. 然後按 `Cmd+R` 重新載入

---

## ✅ 確認您看到的是正確版本

### 視覺特徵檢查表

**K8s Dashboard 版本（正確）應該有：**
- ✅ **淺色主題**（白色背景）
- ✅ **專業藍色** (#326ce5) 作為主色調
- ✅ **左側邊欄**：Nephoran 標誌 + 導航選單
- ✅ **卡片式佈局**：白色卡片帶陰影
- ✅ **標準字體**：Segoe UI / Roboto（非等寬字體）

**Cyber-Terminal 版本（錯誤）特徵：**
- ❌ 深色主題（黑色/深藍背景）
- ❌ 漸層動畫背景
- ❌ 等寬字體（JetBrains Mono / Azeret Mono）
- ❌ 霓虹色彩（紫色、青色）

### 功能特徵檢查

開啟瀏覽器的開發者工具（F12）→ Console，輸入：
```javascript
document.querySelector('.logo-icon').textContent
```
應該顯示：`"N"`（Nephoran 標誌）

---

## 🧪 功能測試步驟

### 1. 即時驗證測試

1. 在「Natural Language Input」文字框中輸入：`scale`
2. **預期結果**：輸入框下方應出現黃色警告框：
   ```
   ⚠ Intent format not recognized - will attempt to process
   ```
3. 繼續輸入：`scale nf-sim to 5 in ns ran-a`
4. **預期結果**：警告框變為綠色成功框：
   ```
   ✓ Valid scale intent detected
   ```

### 2. 範例模板測試

1. 點擊「Quick Examples」下的第一個範例
2. **預期結果**：文字框自動填入範例內容
3. **預期結果**：同時觸發即時驗證顯示綠色成功框

### 3. 鍵盤快捷鍵測試

1. 確認文字框中有內容
2. 按下 `Ctrl+Enter`（Mac 用戶：`Cmd+Enter`）
3. **預期結果**：
   - 出現「Processing with Ollama LLM...」載入動畫
   - 約 30-120 秒後顯示響應結果
   - JSON 響應應有**彩色語法高亮**

### 4. 語法高亮測試

1. 提交任何 intent 並等待響應
2. **預期結果**：JSON 響應應顯示以下顏色：
   - 鍵名（Key）：藍色粗體
   - 字串（String）：綠色
   - 數字（Number）：藍色
   - 布林值（Boolean）：紫色粗體
   - null：灰色斜體

### 5. 歷史記錄測試

1. 提交 2-3 個不同的 intent
2. 滾動到頁面底部「Recent Intents」表格
3. **預期結果**：
   - 顯示最近提交的 intent（最多 10 個）
   - 每條記錄有時間戳、命令、狀態（SUCCESS/ERROR）
   - 點擊「View」按鈕可查看詳細響應

### 6. Clear 功能測試

1. 在文字框輸入一些內容
2. 點擊「Clear」按鈕
3. **預期結果**：
   - 文字框清空
   - 驗證反饋消失

---

## 🐛 常見問題排查

### Q1: 提交 intent 後一直顯示「Processing...」超過 2 分鐘

**原因**：Ollama LLM 服務可能正在處理其他請求

**解決方法**：
```bash
# 檢查 Ollama 狀態
kubectl get pods -n ollama
kubectl logs -n ollama deployment/ollama --tail=50

# 檢查 CPU 使用率
ps aux | grep ollama
```

如果 Ollama CPU 使用率 > 80%，表示正在處理，請耐心等待。

### Q2: 提交 intent 後顯示「502 Bad Gateway」

**原因**：後端服務暫時不可用或正在重啟

**解決方法**：
```bash
# 檢查後端服務
kubectl get pods -n nephoran-intent -l app=intent-ingest
kubectl logs -n nephoran-intent deployment/intent-ingest --tail=50

# 如果 pod 不是 Running，重啟
kubectl rollout restart deployment/intent-ingest -n nephoran-intent
```

### Q3: 即時驗證沒有顯示

**原因**：瀏覽器 JavaScript 被禁用或版本過舊

**解決方法**：
1. 確認瀏覽器版本（推薦 Chrome 100+, Firefox 100+, Safari 15+）
2. 開啟開發者工具（F12）→ Console 查看是否有錯誤
3. 確認 JavaScript 已啟用

### Q4: 語法高亮沒有顯示（JSON 是純黑色）

**原因**：CSS 樣式未正確載入

**解決方法**：
1. 按 `Ctrl+Shift+R` 強制重新載入頁面
2. 清除瀏覽器快取
3. 檢查瀏覽器開發者工具 Console 是否有 CSS 錯誤

---

## 📊 系統狀態檢查命令

從終端執行以下命令檢查系統狀態：

```bash
# 檢查前端 pods
kubectl get pods -n nephoran-system -l app=nephoran-ui

# 檢查後端服務
kubectl get pods -n nephoran-intent -l app=intent-ingest
kubectl get pods -n ollama
kubectl get pods -n rag-service

# 檢查最近的 intent 處理日誌
kubectl logs -n nephoran-intent deployment/intent-ingest --tail=20

# 檢查 Ollama 狀態
kubectl exec -n ollama deployment/ollama -- ollama list

# 檢查 NetworkIntent 資源
kubectl get networkintents -A | head -20
```

---

## 📸 正確版本截圖參考

### 主頁面（應該看起來像 K8s Dashboard）
- 淺色主題，白色背景
- 左側藍色邊欄
- 標題「Network Intent Management」
- 卡片式佈局

### 即時驗證（綠色成功框）
```
Natural Language Input
[輸入框：scale nf-sim to 5 in ns ran-a]

✓ Valid scale intent detected    ← 綠色框
```

### JSON 響應（彩色語法高亮）
```json
{
  "apiVersion": "intent.nephoran.com/v1alpha1",  ← 藍色鍵名，綠色字串值
  "kind": "NetworkIntent",                       ← 藍色鍵名，綠色字串值
  "spec": {                                      ← 藍色鍵名
    "replicas": 5                                ← 藍色鍵名，藍色數字值
  }
}
```

---

## ✅ 測試完成檢查表

請確認以下所有項目都已測試通過：

- [ ] 頁面是淺色主題（K8s Dashboard 風格）
- [ ] 左側邊欄顯示正確
- [ ] 即時驗證正常工作（顯示綠色/黃色反饋）
- [ ] 範例模板可以點擊填入
- [ ] Ctrl+Enter 快捷鍵可以提交
- [ ] JSON 響應有彩色語法高亮
- [ ] 歷史記錄可以查看
- [ ] Clear 按鈕可以清空輸入

---

## 🚀 下一步

如果所有測試都通過，您可以：

1. **提交真實 intent**：
   ```
   scale nf-sim to 10 in ns ran-a
   deploy nginx with 3 replicas in namespace production
   ```

2. **檢查 NetworkIntent CRD 資源**：
   ```bash
   kubectl get networkintents -n ran-a
   ```

3. **監控 NF pods 變化**：
   ```bash
   watch kubectl get pods -n ran-a -l app=nf-sim
   ```

---

**技術支援**：
- 系統狀態報告腳本：`/tmp/system-status-report.sh`
- 前端部署文件：`deployments/frontend/deployment.yaml`
- 升級摘要：`deployments/frontend/UPGRADE_SUMMARY.md`
