# 🚀 Ollama 快速啟動指南

本地 LLM 部署 - 5 分鐘快速上手

## 方法 1: 自動化設定（最簡單）⭐

```bash
# 運行自動化設定腳本
./scripts/setup-ollama.sh

# 按照提示選擇：
# 1. 安裝 Ollama (如果未安裝)
# 2. 選擇模型 (推薦: llama2:7b)
# 3. 測試模型
# 4. 生成配置檔案
```

腳本會自動：
- ✅ 安裝 Ollama
- ✅ 下載您選擇的模型
- ✅ 測試模型功能
- ✅ 創建 `.env` 配置檔案

---

## 方法 2: Docker Compose（推薦生產環境）

```bash
# 1. 啟動所有服務（Ollama + Weaviate + RAG）
docker-compose -f docker-compose.ollama.yml up -d

# 2. 下載模型
docker exec -it nephoran-ollama ollama pull llama2:7b

# 3. 驗證服務
docker-compose -f docker-compose.ollama.yml ps
curl http://localhost:8000/health
curl http://localhost:8000/stats | jq '.config'

# 4. 測試意圖處理
curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{"intent": "Deploy AMF with 3 replicas in namespace 5g-core"}'
```

### 查看日誌
```bash
# RAG 服務日誌
docker-compose -f docker-compose.ollama.yml logs -f rag-service

# Ollama 日誌
docker-compose -f docker-compose.ollama.yml logs -f ollama
```

### 停止服務
```bash
docker-compose -f docker-compose.ollama.yml down
```

---

## 方法 3: 手動本地運行

### Step 1: 安裝 Ollama

```bash
# Linux / macOS
curl -fsSL https://ollama.com/install.sh | sh

# 驗證安裝
ollama --version
```

### Step 2: 下載模型

```bash
# 推薦模型（選一個）
ollama pull llama2:7b    # 快速，4GB
ollama pull mistral:7b   # 高品質，4GB
ollama pull llama2:13b   # 生產級，8GB

# 驗證模型
ollama list
```

### Step 3: 啟動 Ollama 服務

```bash
# 在後台運行
ollama serve &

# 驗證服務
curl http://localhost:11434/api/tags
```

### Step 4: 配置 RAG 服務

```bash
# 複製範例配置
cp .env.ollama.example .env

# 編輯 .env（或直接設定環境變數）
export LLM_PROVIDER=ollama
export LLM_MODEL=llama2:7b
export OLLAMA_BASE_URL=http://localhost:11434
export WEAVIATE_URL=http://localhost:8080  # 需要先啟動 Weaviate
```

### Step 5: 啟動 Weaviate（如果尚未運行）

```bash
docker run -d \
  --name weaviate \
  -p 8080:8080 \
  -p 50051:50051 \
  -e AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  -e PERSISTENCE_DATA_PATH=/var/lib/weaviate \
  -e DEFAULT_VECTORIZER_MODULE=none \
  semitechnologies/weaviate:1.24.5
```

### Step 6: 啟動 RAG 服務

```bash
cd rag-python

# 安裝依賴（如果尚未安裝）
pip install -r requirements.txt

# 啟動服務
uvicorn api:app --reload --port 8000
```

### Step 7: 測試

```bash
# 健康檢查
curl http://localhost:8000/health

# 查看配置
curl http://localhost:8000/stats | jq '.config'
# 應該看到: "llm_provider": "ollama"

# Swagger UI
open http://localhost:8000/docs

# 測試意圖處理
curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy AMF with 3 replicas in namespace 5g-core"
  }' | jq
```

---

## 🎯 快速測試命令

### 測試不同的意圖

```bash
# 1. 部署意圖
curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{"intent": "Deploy AMF with 3 replicas"}' | jq

# 2. 擴展意圖
curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{"intent": "Scale UPF to 5 replicas"}' | jq

# 3. 複雜意圖
curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{"intent": "Deploy SMF with high availability and 4 replicas in namespace 5g-core"}' | jq
```

---

## 🔧 切換模型

### 動態切換（不需重啟）

```bash
# 方法 1: 環境變數（推薦）
export LLM_MODEL=mistral:7b
# 重啟 RAG 服務

# 方法 2: Docker
docker-compose -f docker-compose.ollama.yml down
# 編輯 docker-compose.ollama.yml 中的 OLLAMA_MODEL
docker-compose -f docker-compose.ollama.yml up -d
```

### 模型對比

| 模型 | 速度 | 品質 | 記憶體 | 推薦場景 |
|------|------|------|--------|---------|
| llama2:7b | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | 4GB | 開發/測試 |
| mistral:7b | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 4GB | 生產環境 |
| llama2:13b | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 8GB | 高品質需求 |

---

## 📊 驗證整合

### 1. 檢查 Ollama 狀態

```bash
# 查看運行的模型
ollama ps

# 查看已下載的模型
ollama list

# 測試模型回應
ollama run llama2:7b "Deploy AMF with 3 replicas. Output JSON only."
```

### 2. 檢查 RAG 服務配置

```bash
# 查看配置（應該顯示 ollama provider）
curl http://localhost:8000/stats | jq '.config'

# 預期輸出:
# {
#   "llm_provider": "ollama",
#   "llm_model": "llama2:7b",
#   "ollama_base_url": "http://localhost:11434",
#   ...
# }
```

### 3. 端到端測試

```bash
# 測試完整流程
time curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{"intent": "Deploy AMF with 3 replicas in namespace 5g-core"}' | jq

# 檢查回應時間（應該在 3-5 秒內）
# 檢查 JSON 輸出格式是否正確
```

---

## ⚠️ 常見問題

### Q1: Ollama 連接失敗
```bash
# 檢查 Ollama 是否運行
ps aux | grep ollama
curl http://localhost:11434/api/tags

# 如果沒運行，啟動它
ollama serve &
```

### Q2: 模型未找到
```bash
# 下載缺少的模型
ollama pull llama2:7b

# 驗證
ollama list
```

### Q3: JSON 輸出格式錯誤
```bash
# 切換到更可靠的模型
export LLM_MODEL=mistral:7b
ollama pull mistral:7b
# 重啟 RAG 服務
```

### Q4: 記憶體不足
```bash
# 使用更小的模型
export LLM_MODEL=llama2:7b  # 而非 13b
```

### Q5: 回應速度慢
```bash
# 檢查是否使用 GPU
OLLAMA_NUM_GPU=1 ollama serve

# 或減少並發請求
# 在 api.py 中設定 workers=1
```

---

## 📚 進階配置

詳細文檔請參考：
- **完整指南**: `docs/OLLAMA_INTEGRATION.md`
- **Docker Compose**: `docker-compose.ollama.yml`
- **環境變數**: `.env.ollama.example`
- **設定腳本**: `scripts/setup-ollama.sh`

---

## 🎯 下一步

1. **自定義模型**: 創建電信領域優化的 Modelfile
2. **生產部署**: 使用 Kubernetes 部署（見文檔）
3. **效能調優**: GPU 加速和並發配置
4. **監控**: 添加 Prometheus metrics

---

**需要幫助？**
- 查看完整文檔: `docs/OLLAMA_INTEGRATION.md`
- GitHub Issues: https://github.com/thc1006/nephoran-intent-operator/issues
- PR #344: https://github.com/thc1006/nephoran-intent-operator/pull/344

**最後更新**: 2026-02-14
