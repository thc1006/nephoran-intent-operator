# Ollama Integration Guide for Nephoran Intent Operator

本文檔介紹如何將 Ollama 作為本地 LLM provider 整合到 Nephoran Intent Operator 的 RAG 管線中。

## 📋 目錄

- [為什麼使用 Ollama？](#為什麼使用-ollama)
- [系統需求](#系統需求)
- [安裝 Ollama](#安裝-ollama)
- [配置 RAG 服務](#配置-rag-服務)
- [測試整合](#測試整合)
- [生產部署](#生產部署)
- [故障排除](#故障排除)

---

## 為什麼使用 Ollama？

### 優勢
- ✅ **完全本地部署** - 無需依賴外部 API
- ✅ **數據隱私** - 所有數據保留在本地
- ✅ **無 API 成本** - 免費使用
- ✅ **離線工作** - 不需要網路連接
- ✅ **低延遲** - 本地推理速度快
- ✅ **多模型支援** - 支援 Llama 2, Mistral, CodeLlama 等

### 適用場景
- 開發和測試環境
- 數據敏感的生產環境
- 需要離線運行的邊緣部署
- 成本優化的場景

---

## 系統需求

### 硬體需求

| 組件 | 最低配置 | 推薦配置 |
|------|---------|---------|
| CPU | 4 核心 | 8+ 核心 |
| RAM | 8 GB | 16+ GB |
| 磁碟 | 10 GB | 20+ GB (多模型) |
| GPU | 無 (可選) | NVIDIA GPU (加速推理) |

### 軟體需求
- **作業系統**: Linux (Ubuntu 20.04+), macOS, Windows (WSL2)
- **Docker**: 20.10+ (如果使用容器化部署)
- **Python**: 3.11+ (已在 requirements.txt 中)

### 模型大小參考

| 模型 | 參數量 | 記憶體需求 | 推理速度 | 適用場景 |
|------|--------|-----------|---------|---------|
| llama2:7b | 7B | ~4 GB | 快 | 開發/測試 |
| llama2:13b | 13B | ~8 GB | 中等 | 一般生產 |
| mistral:7b | 7B | ~4 GB | 快 | 高品質輸出 |
| codellama:7b | 7B | ~4 GB | 快 | 程式碼生成 |

---

## 安裝 Ollama

### 方法 1: Linux/macOS 一鍵安裝（推薦）

```bash
# 下載並安裝 Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 驗證安裝
ollama --version
```

### 方法 2: Docker 安裝

```bash
# 拉取 Ollama Docker 映像
docker pull ollama/ollama

# 運行 Ollama 容器
docker run -d \
  --name ollama \
  -p 11434:11434 \
  -v ollama-data:/root/.ollama \
  ollama/ollama

# 驗證運行
docker logs ollama
```

### 方法 3: Windows (WSL2)

```powershell
# 在 WSL2 Ubuntu 中執行
wsl

# 然後運行 Linux 安裝腳本
curl -fsSL https://ollama.com/install.sh | sh
```

---

## 下載和配置模型

### 下載推薦模型

```bash
# Llama 2 7B (推薦入門使用)
ollama pull llama2:7b

# Mistral 7B (更好的品質)
ollama pull mistral:7b

# CodeLlama (程式碼專用)
ollama pull codellama:7b

# 查看已下載的模型
ollama list
```

### 測試模型

```bash
# 測試 Llama 2
ollama run llama2:7b "Deploy AMF with 3 replicas in namespace 5g-core. Output as JSON."

# 測試 Mistral
ollama run mistral:7b "Generate a NetworkIntent JSON for scaling UPF to 5 replicas"
```

### 自定義模型配置（可選）

創建 `Modelfile` 優化模型輸出：

```dockerfile
# custom-telecom-model.Modelfile
FROM llama2:7b

# 設定系統提示詞
SYSTEM """
You are an expert telecommunications network engineer.
You translate natural language commands into structured JSON for O-RAN network functions.
Always output valid JSON without explanations.
"""

# 參數優化
PARAMETER temperature 0
PARAMETER top_p 0.9
PARAMETER num_predict 2048

# 範例對話
MESSAGE user Deploy AMF with 3 replicas
MESSAGE assistant {"type": "NetworkFunctionDeployment", "name": "amf", "namespace": "5g-core", "spec": {"replicas": 3}}
```

創建自定義模型：

```bash
ollama create telecom-assistant -f custom-telecom-model.Modelfile
ollama run telecom-assistant "Scale UPF to 5 replicas"
```

---

## 配置 RAG 服務

### 選項 A: 環境變數配置（推薦）

創建 `.env` 檔案：

```bash
# .env
# LLM Provider 配置
LLM_PROVIDER=ollama
LLM_MODEL=llama2:7b  # 或 mistral:7b, codellama:7b, telecom-assistant
OLLAMA_BASE_URL=http://localhost:11434

# Weaviate 配置
WEAVIATE_URL=http://localhost:8080

# 快取配置
CACHE_MAX_SIZE=1000
CACHE_TTL_SECONDS=3600

# 知識庫路徑
KNOWLEDGE_BASE_PATH=/app/knowledge_base
```

### 選項 B: Docker Compose 配置

創建 `docker-compose.ollama.yml`：

```yaml
version: '3.8'

services:
  # Ollama 服務
  ollama:
    image: ollama/ollama:latest
    container_name: nephoran-ollama
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    restart: unless-stopped
    # 如果有 GPU
    # deploy:
    #   resources:
    #     reservations:
    #       devices:
    #         - driver: nvidia
    #           count: 1
    #           capabilities: [gpu]

  # Weaviate 向量資料庫
  weaviate:
    image: semitechnologies/weaviate:1.24.5
    container_name: nephoran-weaviate
    ports:
      - "8080:8080"
      - "50051:50051"
    environment:
      QUERY_DEFAULTS_LIMIT: 25
      AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: 'true'
      PERSISTENCE_DATA_PATH: /var/lib/weaviate
      DEFAULT_VECTORIZER_MODULE: none
      ENABLE_MODULES: text2vec-openai
      CLUSTER_HOSTNAME: weaviate
    volumes:
      - weaviate-data:/var/lib/weaviate
    restart: unless-stopped

  # RAG 服務
  rag-service:
    build:
      context: .
      dockerfile: rag-python/Dockerfile
    container_name: nephoran-rag-service
    ports:
      - "8000:8000"
    environment:
      LLM_PROVIDER: ollama
      LLM_MODEL: llama2:7b
      OLLAMA_BASE_URL: http://ollama:11434
      WEAVIATE_URL: http://weaviate:8080
      LOG_LEVEL: INFO
    volumes:
      - ./knowledge_base:/app/knowledge_base:ro
    depends_on:
      - ollama
      - weaviate
    restart: unless-stopped

volumes:
  ollama-data:
  weaviate-data:
```

啟動服務：

```bash
# 啟動所有服務
docker-compose -f docker-compose.ollama.yml up -d

# 在 Ollama 容器中下載模型
docker exec -it nephoran-ollama ollama pull llama2:7b

# 查看日誌
docker-compose -f docker-compose.ollama.yml logs -f rag-service
```

### 選項 C: Kubernetes 部署

更新 `deployments/rag-service.yaml` 的 ConfigMap：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-config
  namespace: nephoran-rag
data:
  llm_provider: "ollama"
  llm_model: "llama2:7b"
  ollama_base_url: "http://ollama-service:11434"
  chunk_size: "1000"
  chunk_overlap: "200"
  embedding_model: "text-embedding-3-large"
  log_level: "INFO"
---
# Ollama Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama
  namespace: nephoran-rag
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ollama
  template:
    metadata:
      labels:
        app: ollama
    spec:
      containers:
      - name: ollama
        image: ollama/ollama:latest
        ports:
        - containerPort: 11434
          name: http
        resources:
          requests:
            memory: "4Gi"
            cpu: "2000m"
          limits:
            memory: "8Gi"
            cpu: "4000m"
        volumeMounts:
        - name: ollama-data
          mountPath: /root/.ollama
      volumes:
      - name: ollama-data
        persistentVolumeClaim:
          claimName: ollama-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: ollama-service
  namespace: nephoran-rag
spec:
  selector:
    app: ollama
  ports:
  - protocol: TCP
    port: 11434
    targetPort: 11434
  type: ClusterIP
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ollama-pvc
  namespace: nephoran-rag
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
```

部署到 Kubernetes：

```bash
# 應用配置
kubectl apply -f deployments/rag-service.yaml

# 在 Ollama pod 中下載模型
kubectl exec -it deployment/ollama -n nephoran-rag -- ollama pull llama2:7b

# 驗證服務
kubectl get pods -n nephoran-rag
kubectl logs -f deployment/rag-service -n nephoran-rag
```

---

## 測試整合

### 本地測試

```bash
# 1. 啟動 Ollama (如果尚未運行)
ollama serve &

# 2. 下載模型
ollama pull llama2:7b

# 3. 設定環境變數
export LLM_PROVIDER=ollama
export LLM_MODEL=llama2:7b
export OLLAMA_BASE_URL=http://localhost:11434
export WEAVIATE_URL=http://localhost:8080

# 4. 啟動 RAG 服務
cd rag-python
uvicorn api:app --reload --port 8000

# 5. 測試健康檢查
curl http://localhost:8000/health

# 6. 查看配置
curl http://localhost:8000/stats | jq '.config'

# 預期輸出:
# {
#   "llm_provider": "ollama",
#   "llm_model": "llama2:7b",
#   "ollama_base_url": "http://localhost:11434",
#   ...
# }
```

### 測試意圖處理

```bash
# 測試部署意圖
curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy AMF with 3 replicas in namespace 5g-core"
  }' | jq

# 測試擴展意圖
curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Scale UPF to 5 replicas"
  }' | jq

# 測試複雜意圖
curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy SMF with high availability configuration and 4 replicas"
  }' | jq
```

### 效能測試

```bash
# 測試回應時間
time curl -X POST http://localhost:8000/process \
  -H "Content-Type: application/json" \
  -d '{"intent": "Deploy AMF with 3 replicas"}'

# 負載測試 (需要安裝 hey)
hey -n 100 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"intent": "Deploy AMF with 3 replicas"}' \
  http://localhost:8000/process
```

---

## 生產部署最佳實踐

### 1. 模型選擇建議

| 環境 | 推薦模型 | 理由 |
|------|---------|------|
| 開發 | llama2:7b | 快速迭代 |
| 測試 | mistral:7b | 更好的輸出品質 |
| 生產 | llama2:13b 或自定義微調 | 平衡效能和品質 |

### 2. 資源配置

```yaml
# Kubernetes 資源配置建議
resources:
  requests:
    memory: "4Gi"   # 7B 模型
    cpu: "2000m"
  limits:
    memory: "8Gi"   # 為推理預留空間
    cpu: "4000m"
```

### 3. 監控指標

監控以下指標：
- **推理延遲**: P50, P95, P99
- **記憶體使用**: Ollama 容器記憶體
- **CPU 使用率**: 推理期間的 CPU 負載
- **模型加載時間**: 容器啟動到模型就緒
- **錯誤率**: 失敗的推理請求

### 4. 高可用性配置

```yaml
# 多副本部署
spec:
  replicas: 3  # Ollama 服務
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
```

### 5. 模型快取策略

```bash
# 預加載模型到快取
kubectl exec -it deployment/ollama -- sh -c "
  ollama pull llama2:7b
  ollama run llama2:7b 'Test prompt' --verbose
"
```

---

## 故障排除

### 問題 1: Ollama 連接失敗

**症狀**:
```
ConnectionError: Failed to connect to Ollama at http://localhost:11434
```

**解決方案**:
```bash
# 1. 檢查 Ollama 是否運行
ps aux | grep ollama
curl http://localhost:11434/api/tags

# 2. 重啟 Ollama
pkill ollama
ollama serve &

# 3. 檢查防火牆
sudo ufw allow 11434/tcp

# 4. 檢查 Docker 網路 (如果使用 Docker)
docker network inspect bridge
```

### 問題 2: 模型未找到

**症狀**:
```
Error: model 'llama2:7b' not found
```

**解決方案**:
```bash
# 查看已下載的模型
ollama list

# 下載缺少的模型
ollama pull llama2:7b

# 驗證模型
ollama show llama2:7b
```

### 問題 3: JSON 輸出格式錯誤

**症狀**: LLM 輸出不是有效的 JSON

**解決方案**:

1. 使用自定義 Modelfile 加強 JSON 輸出：
   ```dockerfile
   FROM llama2:7b

   SYSTEM """
   You MUST output valid JSON only. No explanations.
   """

   PARAMETER temperature 0
   ```

2. 在 Python 代碼中添加重試邏輯（已在代碼中實現）

3. 使用 `mistral:7b` - 它在結構化輸出方面通常更可靠

### 問題 4: 記憶體不足

**症狀**:
```
OOM killed: ollama process
```

**解決方案**:
```bash
# 1. 使用更小的模型
ollama pull llama2:7b  # 而非 13b

# 2. 增加 swap 空間 (Linux)
sudo fallocate -l 8G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# 3. 限制並發請求
# 在 api.py 中設定 workers=1
```

### 問題 5: 推理速度慢

**優化策略**:
```bash
# 1. 使用 GPU 加速 (如果可用)
OLLAMA_NUM_GPU=1 ollama serve

# 2. 調整模型參數
# 在 Modelfile 中:
PARAMETER num_thread 4

# 3. 預加載模型到記憶體
ollama run llama2:7b "warmup"
```

---

## 與 OpenAI 的效能對比

| 指標 | OpenAI (gpt-4o-mini) | Ollama (llama2:7b) | Ollama (mistral:7b) |
|------|---------------------|-------------------|-------------------|
| **延遲** | ~1-2 秒 | ~3-5 秒 | ~3-5 秒 |
| **成本** | $0.15/1M tokens | 免費 | 免費 |
| **品質** | 優秀 | 良好 | 良好-優秀 |
| **隱私** | 雲端處理 | 完全本地 | 完全本地 |
| **離線** | ❌ | ✅ | ✅ |
| **自定義** | 有限 | 完全可控 | 完全可控 |

---

## 進階功能

### 多模型負載均衡

部署多個模型實例：

```yaml
# 使用不同模型的多個副本
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama-llama2
spec:
  replicas: 2
  # ... llama2:7b 配置
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama-mistral
spec:
  replicas: 1
  # ... mistral:7b 配置
```

### 模型微調 (Fine-tuning)

使用 Ollama 的 Modelfile 進行領域特定微調：

```dockerfile
FROM llama2:7b

# 添加大量電信領域範例
MESSAGE user Deploy AMF with 5 replicas
MESSAGE assistant {"type": "NetworkFunctionDeployment", "name": "amf", "namespace": "5g-core", "spec": {"replicas": 5}}

MESSAGE user Scale UPF to 3 instances
MESSAGE assistant {"type": "NetworkFunctionScale", "name": "upf", "namespace": "5g-core", "replicas": 3}

# ... 添加更多範例
```

---

## 參考資源

- **Ollama 官方文檔**: https://ollama.com/docs
- **Ollama GitHub**: https://github.com/ollama/ollama
- **LangChain Ollama 整合**: https://python.langchain.com/docs/integrations/llms/ollama
- **模型庫**: https://ollama.com/library
- **Llama 2 論文**: https://arxiv.org/abs/2307.09288
- **Mistral 文檔**: https://docs.mistral.ai/

---

## 附錄: 快速配置檢查清單

- [ ] Ollama 已安裝並運行
- [ ] 模型已下載 (`ollama list` 檢查)
- [ ] `LLM_PROVIDER=ollama` 環境變數已設定
- [ ] `LLM_MODEL` 設定為有效的模型名稱
- [ ] `OLLAMA_BASE_URL` 正確指向 Ollama 服務
- [ ] Weaviate 已運行並可訪問
- [ ] RAG 服務可以連接到 Ollama (檢查 `/stats` 端點)
- [ ] 測試意圖處理成功返回 JSON

---

**最後更新**: 2026-02-14
**版本**: 1.0.0
**維護者**: Nephoran Intent Operator Team
