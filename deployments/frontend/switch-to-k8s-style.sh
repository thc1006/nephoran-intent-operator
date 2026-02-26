#!/bin/bash
# 切換到 K8s Dashboard 風格前端

set -euo pipefail

echo "=== 切換到 K8s Dashboard 風格前端 ==="
echo ""

# 1. 從 git 歷史恢復 K8s Dashboard 版本
echo "1. 恢復 K8s Dashboard 版本..."
git show 3422ec4b5:deployments/frontend/index.html > /tmp/k8s-style-ui.html

# 2. 更新 ConfigMap
echo "2. 更新 ConfigMap..."
kubectl create configmap nephoran-ui-html \
  --from-file=index.html=/tmp/k8s-style-ui.html \
  --namespace=nephoran-system \
  --dry-run=client -o yaml | kubectl apply -f -

# 3. 重啟 pods
echo "3. 重啟前端 pods..."
kubectl rollout restart deployment/nephoran-ui -n nephoran-system

# 4. 等待部署完成
echo "4. 等待部署完成..."
kubectl rollout status deployment/nephoran-ui -n nephoran-system --timeout=60s

echo ""
echo "✅ 成功切換到 K8s Dashboard 風格！"
echo ""
echo "訪問地址:"
echo "  - Ngrok: https://lennie-unfatherly-profusely.ngrok-free.dev"
echo "  - NodePort: http://192.168.10.65:30081"
echo ""
