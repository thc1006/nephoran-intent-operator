#!/bin/bash

###############################################################################
# 清理測試資源腳本
# 版本: 1.0.0
###############################################################################

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}[INFO]${NC} 開始清理測試資源..."

# 刪除測試 NetworkIntents
echo -e "${BLUE}[INFO]${NC} 刪除測試 NetworkIntents..."
kubectl delete networkintents -n default -l test-suite=intent-ric-e2e 2>/dev/null || \
    echo -e "${BLUE}[INFO]${NC} 沒有找到測試 NetworkIntents"

kubectl delete networkintent e2e-test-invalid-intent -n default 2>/dev/null || \
    echo -e "${BLUE}[INFO]${NC} 沒有找到 invalid intent"

# 刪除測試 Pod
echo -e "${BLUE}[INFO]${NC} 刪除測試 Pod..."
kubectl delete pod test-network-tools 2>/dev/null || \
    echo -e "${BLUE}[INFO]${NC} 沒有找到測試 Pod"

# 檢查是否還有殘留資源
remaining_intents=$(kubectl get networkintents -n default -l test-suite=intent-ric-e2e --no-headers 2>/dev/null | wc -l)
if [ "$remaining_intents" -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} 所有測試資源已清理完成"
else
    echo -e "${RED}[WARNING]${NC} 仍有 $remaining_intents 個 NetworkIntent 未清理"
fi

echo -e "${GREEN}[DONE]${NC} 清理完成"
