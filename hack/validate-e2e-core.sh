#!/usr/bin/env bash
set -euo pipefail

# Core E2E Validation - 驗收腳本
# 專注於驗證核心E2E組件和預期輸出格式

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-nephoran-e2e}"
NAMESPACE="${NAMESPACE:-nephoran-system}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}" >&2; }

# Validation functions
validate_core_files() {
    log_info "驗證核心E2E檔案存在..."
    
    local files=(
        "hack/run-e2e.sh"
        "hack/run-e2e-simple.sh"
        "tools/verify-scale.go"
        "E2E.md"
        "config/crd/bases/intent.nephoran.com_networkintents.yaml"
    )
    
    for file in "${files[@]}"; do
        if [ -f "$PROJECT_ROOT/$file" ]; then
            log_success "✓ $file 存在"
        else
            log_error "✗ $file 不存在"
            return 1
        fi
    done
}

validate_script_format() {
    log_info "驗證腳本輸出格式..."
    
    # 檢查是否包含所需的輸出格式
    if grep -q "Replicas.*desired.*ready.*OK" "$PROJECT_ROOT/hack/run-e2e.sh"; then
        log_success "✓ E2E腳本包含正確的結果格式"
    else
        log_error "✗ E2E腳本缺少預期的結果格式"
        return 1
    fi
    
    # 檢查是否有失敗時的除錯功能
    if grep -q "debug_failed_pods\|Pod.*debug\|POD DEBUG" "$PROJECT_ROOT/hack/run-e2e.sh"; then
        log_success "✓ E2E腳本包含失敗時的除錯功能"
    else
        log_error "✗ E2E腳本缺少失敗時的除錯功能"
        return 1
    fi
}

validate_go_verifier() {
    log_info "驗證Go verifier工具..."
    
    cd "$PROJECT_ROOT"
    if go build -o /tmp/test-verify-scale ./tools/verify-scale.go 2>/dev/null; then
        log_success "✓ verify-scale.go 編譯成功"
        
        # 測試help輸出
        if /tmp/test-verify-scale --help | grep -q "namespace.*name.*target-replicas"; then
            log_success "✓ verify-scale.go 參數格式正確"
        else
            log_warning "⚠ verify-scale.go help輸出可能有問題"
        fi
        
        rm -f /tmp/test-verify-scale
    else
        log_error "✗ verify-scale.go 編譯失敗"
        return 1
    fi
}

demonstrate_expected_output() {
    log_info "展示預期的E2E測試輸出格式..."
    
    echo -e "\n${BLUE}預期的成功輸出應該包含：${NC}"
    echo -e "${GREEN}Replicas (nf-sim, ran-a): desired=3, ready=3 (OK)${NC}"
    
    echo -e "\n${BLUE}預期的失敗輸出應該包含：${NC}"
    echo -e "${RED}Replicas (nf-sim, ran-a): desired=3, ready=1 (FAILED)${NC}"
    echo -e "${YELLOW}Pod Status for debugging:${NC}"
    echo -e "${YELLOW}Pod Logs (last 100 lines each):${NC}"
    echo -e "${YELLOW}Recent Cluster Events:${NC}"
}

validate_crd_structure() {
    log_info "驗證NetworkIntent CRD結構..."
    
    if [ -f "$PROJECT_ROOT/config/crd/bases/intent.nephoran.com_networkintents.yaml" ]; then
        if grep -q "intentType.*scaling" "$PROJECT_ROOT/config/crd/bases/intent.nephoran.com_networkintents.yaml"; then
            log_success "✓ CRD包含scaling intentType"
        else
            log_warning "⚠ CRD可能缺少scaling intentType"
        fi
        
        if grep -q "v1alpha1" "$PROJECT_ROOT/config/crd/bases/intent.nephoran.com_networkintents.yaml"; then
            log_success "✓ CRD使用v1alpha1版本"
        else
            log_error "✗ CRD版本不正確"
            return 1
        fi
    fi
}

run_lightweight_cluster_test() {
    log_info "執行輕量級集群測試..."
    
    # 創建一個測試集群來驗證基本功能
    if command -v kind >/dev/null 2>&1; then
        log_info "檢查kind是否可用..."
        
        # 創建簡單的集群配置測試
        if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
            log_warning "測試集群已存在，跳過創建"
        else
            log_info "創建輕量級測試集群..."
            if kind create cluster --name "$CLUSTER_NAME" --wait 60s 2>/dev/null; then
                log_success "✓ 測試集群創建成功"
                
                # 測試CRD安裝
                if kubectl apply -f "$PROJECT_ROOT/config/crd/bases/intent.nephoran.com_networkintents.yaml" 2>/dev/null; then
                    log_success "✓ NetworkIntent CRD安裝成功"
                    
                    # 創建測試NetworkIntent
                    cat > /tmp/test-intent.yaml << EOF
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: test-intent
  namespace: default
spec:
  intentType: scaling
  namespace: default
  target: "test deployment"
  replicas: 3
EOF
                    
                    if kubectl apply -f /tmp/test-intent.yaml 2>/dev/null; then
                        log_success "✓ NetworkIntent創建成功"
                        
                        # 驗證資源存在
                        if kubectl get networkintent test-intent -o jsonpath='{.spec.replicas}' 2>/dev/null | grep -q "3"; then
                            log_success "✓ NetworkIntent規格驗證成功"
                        fi
                    fi
                fi
                
                # 清理測試集群
                log_info "清理測試集群..."
                kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
                log_success "✓ 測試集群已清理"
            else
                log_warning "⚠ 無法創建測試集群（可能是資源限制）"
            fi
        fi
    else
        log_warning "⚠ kind未安裝，跳過集群測試"
    fi
}

# 主要驗收流程
main() {
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}        Nephoran E2E Core Validation       ${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}         E2E核心功能驗收測試                ${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}\n"
    
    local validation_passed=0
    
    # 執行各項驗證
    if validate_core_files; then
        ((validation_passed++))
    fi
    
    if validate_script_format; then
        ((validation_passed++))
    fi
    
    if validate_go_verifier; then
        ((validation_passed++))
    fi
    
    if validate_crd_structure; then
        ((validation_passed++))
    fi
    
    # 展示預期輸出格式
    demonstrate_expected_output
    
    # 可選的集群測試
    echo -e "\n${BLUE}是否執行輕量級集群測試？ (可能需要2-3分鐘)${NC}"
    run_lightweight_cluster_test
    
    # 總結報告
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}           驗收結果總結                     ${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    
    if [ $validation_passed -eq 4 ]; then
        echo -e "${GREEN}🎉 E2E核心驗收 - 全部通過！${NC}"
        echo -e "${GREEN}   ✅ 所有核心檔案存在${NC}"
        echo -e "${GREEN}   ✅ 腳本輸出格式正確${NC}"
        echo -e "${GREEN}   ✅ Go驗證工具可用${NC}" 
        echo -e "${GREEN}   ✅ CRD結構正確${NC}"
        echo -e "\n${GREEN}準備執行完整E2E測試：${NC}"
        echo -e "   bash hack/run-e2e.sh"
        echo -e "\n${GREEN}驗收標準：${NC}"
        echo -e "   結尾顯示: ${GREEN}Replicas (nf-sim, ran-a): desired=3, ready=3 (OK)${NC}"
        echo -e "   失敗時自動輸出Pod日誌進行除錯"
        return 0
    else
        echo -e "${RED}❌ E2E核心驗收 - 部分失敗${NC}"
        echo -e "   通過: $validation_passed/4 項檢查"
        return 1
    fi
}

# Execute validation
main