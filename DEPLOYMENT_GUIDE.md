# Nephoran Intent Operator Deployment Guide

## 概述

本指南基於專案架構分析結果，提供 Nephoran Intent Operator 的實用部署步驟。系統包含4個核心服務：LLM Processor、RAG API、Nephio Bridge、O-RAN Adaptor。

## 環境準備

### 先決條件

基於 `Makefile` 和專案配置，需要以下工具：

- **Go**: 1.24+ (參考 go.mod: go 1.24.0)
- **Docker**: 用於容器化建構
- **kubectl**: Kubernetes 命令行工具
- **Kind/Minikube**: 本地 Kubernetes 集群
- **Python**: 3.8+ (用於 RAG API 服務)

### 環境變數設置

```bash
# 必要的環境變數
export OPENAI_API_KEY="your-openai-api-key"
export WEAVIATE_URL="http://weaviate:8080"
export RAG_API_URL="http://rag-api:5001"
```

## 本地開發環境設置

### 1. 專案克隆和依賴安裝

```bash
# 克隆專案
git clone <repository-url>
cd nephoran-intent-operator

# 安裝 Go 依賴 (基於 make setup-dev)
make setup-dev

# 驗證環境
go version
docker version
kubectl version --client
```

### 2. 代碼生成

```bash
# 生成 Kubernetes 代碼 (基於 Makefile)
make generate
```

## Docker 建構流程

基於 `TEST_3C_INTEGRATION.md` 分析的4個服務建構：

### 1. 建構所有服務

```bash
# 建構所有 Docker 映像
make docker-build

# 或分別建構各服務
make build-llm-processor
make build-nephio-bridge
make build-oran-adaptor
# RAG API 使用 Python Dockerfile
```

### 2. 映像標籤和推送

```bash
# 查看建構的映像
docker images | grep nephoran

# 如需推送到遠程倉庫
make docker-push
```

## 服務部署順序

基於依賴關係和服務架構：

### 1. 基礎設施服務

```bash
# 部署 CRDs
kubectl apply -f deployments/crds/

# 部署 RBAC
kubectl apply -f deployments/kubernetes/nephio-bridge-rbac.yaml
```

### 2. 存儲和數據服務

```bash
# 部署 Weaviate (向量數據庫)
kubectl apply -f deployments/weaviate/weaviate-deployment.yaml

# 等待 Weaviate 就緒
kubectl wait --for=condition=ready pod -l app=weaviate --timeout=300s
```

### 3. 核心服務部署

```bash
# 1. RAG API 服務 (Python Flask)
kubectl apply -f deployments/kustomize/base/rag-api/

# 2. LLM Processor 服務
kubectl apply -f deployments/kustomize/base/llm-processor/

# 3. Nephio Bridge 控制器
kubectl apply -f deployments/kustomize/base/nephio-bridge/

# 4. O-RAN Adaptor 服務
kubectl apply -f deployments/kustomize/base/oran-adaptor/
```

### 4. 環境特定部署

```bash
# 本地環境
make deploy-dev

# 或使用 deploy.sh 腳本
./deploy.sh local
```

## 知識庫初始化

```bash
# 初始化 Weaviate 知識庫
./populate-knowledge-base.ps1

# 或使用 Make 目標
make populate-kb
```

## 服務驗證

### 1. 檢查部署狀態

```bash
# 查看所有 Pod
kubectl get pods -A

# 查看服務狀態
kubectl get svc

# 查看 CRD 註冊
kubectl get crd | grep nephoran
```

### 2. 健康檢查

```bash
# RAG API 健康檢查
kubectl port-forward svc/rag-api 5001:5001
curl http://localhost:5001/health

# LLM Processor 健康檢查
kubectl port-forward svc/llm-processor 8080:8080
curl http://localhost:8080/healthz
```

### 3. 功能測試

```bash
# 運行集成測試
make test-integration

# 創建測試 NetworkIntent
kubectl apply -f examples/networkintent-example.yaml

# 查看處理結果
kubectl get networkintents
kubectl describe networkintent <name>
```

## 故障排除

### 常見問題

1. **Pod 無法啟動**
   ```bash
   kubectl describe pod <pod-name>
   kubectl logs <pod-name>
   ```

2. **CRD 註冊問題**
   ```bash
   kubectl get crd | grep nephoran
   kubectl describe crd networkintents.nephoran.com
   ```

3. **服務連通性問題**
   ```bash
   kubectl get svc
   kubectl get endpoints
   ```

### 日誌查看

```bash
# 查看控制器日誌
kubectl logs deployment/nephio-bridge -f

# 查看 RAG API 日誌
kubectl logs deployment/rag-api -f

# 查看 LLM Processor 日誌
kubectl logs deployment/llm-processor -f
```

## 開發工作流程

1. **代碼修改後重新部署**
   ```bash
   make docker-build
   ./deploy.sh local
   ```

2. **清理環境**
   ```bash
   kubectl delete -f deployments/kustomize/base/
   kubectl delete -f deployments/crds/
   ```

3. **完整重置**
   ```bash
   kind delete cluster
   kind create cluster
   # 重新執行部署流程
   ```

## 生產環境部署注意事項

- 確保所有環境變數正確設置
- 使用適當的資源限制和請求
- 配置持久化存儲
- 設置監控和日誌收集
- 實施適當的安全策略

## 下一步

部署完成後，可以：
- 測試自然語言意圖處理
- 探索 O-RAN 接口功能
- 集成 GitOps 工作流程
- 配置監控和警報