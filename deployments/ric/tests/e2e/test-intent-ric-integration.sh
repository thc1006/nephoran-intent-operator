#!/bin/bash

###############################################################################
# Nephoran Intent Operator 與 O-RAN SC RIC 端到端整合測試
# 版本: 1.0.0
# 日期: 2026-02-15
###############################################################################

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 測試結果追蹤
PASSED_TESTS=0
FAILED_TESTS=0
TOTAL_TESTS=0

# 日誌函數
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# 測試結果記錄
record_test_result() {
    local test_name=$1
    local result=$2
    local message=$3

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [ "$result" == "PASS" ]; then
        log_success "✅ TEST $TOTAL_TESTS PASSED: $test_name - $message"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        log_error "❌ TEST $TOTAL_TESTS FAILED: $test_name - $message"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# 檢查必需的工具
check_prerequisites() {
    log_info "==================== 階段 0: 前置條件檢查 ===================="

    local missing_tools=()

    for tool in kubectl curl jq; do
        if ! command -v $tool &> /dev/null; then
            missing_tools+=($tool)
        fi
    done

    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "缺少必需工具: ${missing_tools[*]}"
        exit 1
    fi

    log_success "所有必需工具已安裝"
}

# 階段 1: 環境驗證
test_environment_validation() {
    log_info "==================== 階段 1: 環境驗證 ===================="

    # 1.1 檢查 Intent Operator
    log_info "1.1 檢查 Intent Operator 運行狀態"
    if kubectl get pods -n nephoran-system -l control-plane=controller-manager 2>/dev/null | grep -q Running; then
        local pod_name=$(kubectl get pods -n nephoran-system -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}')
        record_test_result "Intent Operator Pod Running" "PASS" "Pod: $pod_name"
    else
        record_test_result "Intent Operator Pod Running" "FAIL" "Pod not found or not running"
        return 1
    fi

    # 1.2 檢查 NetworkIntent CRD
    log_info "1.2 檢查 NetworkIntent CRD"
    if kubectl get crd | grep -q "networkintents.intent.nephoran.com"; then
        record_test_result "NetworkIntent CRD Exists" "PASS" "CRD registered"
    else
        record_test_result "NetworkIntent CRD Exists" "FAIL" "CRD not found"
        return 1
    fi

    # 1.3 檢查 RIC A1 Mediator
    log_info "1.3 檢查 RIC A1 Mediator 服務"
    if kubectl get svc -n ricplt service-ricplt-a1mediator-http 2>/dev/null | grep -q ClusterIP; then
        local svc_ip=$(kubectl get svc -n ricplt service-ricplt-a1mediator-http -o jsonpath='{.spec.clusterIP}')
        record_test_result "A1 Mediator Service" "PASS" "Service IP: $svc_ip"
    else
        record_test_result "A1 Mediator Service" "FAIL" "Service not found"
        return 1
    fi

    # 1.4 檢查所有 RIC 組件
    log_info "1.4 檢查 RIC 平台組件狀態"
    local ric_pods=$(kubectl get pods -n ricplt --no-headers 2>/dev/null | wc -l)
    local ric_running=$(kubectl get pods -n ricplt --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)

    if [ "$ric_pods" -eq "$ric_running" ] && [ "$ric_pods" -gt 0 ]; then
        record_test_result "RIC Platform Components" "PASS" "$ric_running/$ric_pods pods running"
    else
        record_test_result "RIC Platform Components" "FAIL" "Only $ric_running/$ric_pods pods running"
    fi

    log_success "環境驗證完成"
}

# 階段 2: 網絡連接性測試
test_network_connectivity() {
    log_info "==================== 階段 2: 網絡連接性測試 ===================="

    # 確保測試 pod 存在
    log_info "2.0 部署網絡測試 Pod"
    if ! kubectl get pod test-network-tools 2>/dev/null | grep -q Running; then
        kubectl run test-network-tools --image=nicolaka/netshoot --restart=Never -- sleep 3600 2>/dev/null || true
        sleep 5
    fi

    # 等待 pod 就緒
    kubectl wait --for=condition=Ready pod/test-network-tools --timeout=60s 2>/dev/null || true

    # 2.1 DNS 解析測試
    log_info "2.1 測試 DNS 解析"
    if kubectl exec test-network-tools -- nslookup service-ricplt-a1mediator-http.ricplt.svc.cluster.local 2>&1 | grep -q "Address:"; then
        record_test_result "DNS Resolution" "PASS" "A1 Mediator DNS resolves"
    else
        record_test_result "DNS Resolution" "FAIL" "DNS resolution failed"
    fi

    # 2.2 A1 Mediator 健康檢查
    log_info "2.2 測試 A1 Mediator 健康檢查"
    local health_check=$(kubectl exec test-network-tools -- curl -s -m 5 http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000/A1-P/v2/healthcheck 2>&1 || echo "FAILED")

    if [ "$health_check" != "FAILED" ]; then
        record_test_result "A1 Health Check" "PASS" "Health check successful"
    else
        record_test_result "A1 Health Check" "FAIL" "Health check failed"
    fi

    # 2.3 A1 Policy Types API
    log_info "2.3 測試 A1 Policy Types API"
    local policy_types=$(kubectl exec test-network-tools -- curl -s http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000/A1-P/v2/policytypes 2>&1)

    if echo "$policy_types" | grep -q "100"; then
        record_test_result "A1 Policy Types API" "PASS" "Policy type 100 found"

        # 2.4 獲取 Policy Type Schema
        log_info "2.4 驗證 Policy Type 100 Schema"
        local schema=$(kubectl exec test-network-tools -- curl -s http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000/A1-P/v2/policytypes/100 2>&1)

        if echo "$schema" | jq -e '.policy_type_id == 100' &>/dev/null; then
            record_test_result "Policy Type Schema" "PASS" "Schema validation successful"
        else
            record_test_result "Policy Type Schema" "FAIL" "Invalid schema"
        fi
    else
        record_test_result "A1 Policy Types API" "FAIL" "No policy types found"
    fi

    log_success "網絡連接性測試完成"
}

# 階段 3: NetworkIntent 創建與驗證
test_intent_creation() {
    log_info "==================== 階段 3: NetworkIntent 創建與驗證 ===================="

    local test_dir="$(dirname "$0")/sample-intents"

    # 3.1 創建簡單擴容 Intent
    log_info "3.1 測試簡單擴容 Intent"
    if [ -f "$test_dir/01-simple-scale.yaml" ]; then
        kubectl apply -f "$test_dir/01-simple-scale.yaml" 2>&1 | head -1
        sleep 3

        if kubectl get networkintent e2e-test-simple-scale -n default 2>/dev/null | grep -q "e2e-test-simple-scale"; then
            record_test_result "Simple Scale Intent Creation" "PASS" "Intent created successfully"

            # 檢查 Intent 狀態
            local phase=$(kubectl get networkintent e2e-test-simple-scale -n default -o jsonpath='{.status.phase}' 2>/dev/null)
            log_info "Intent Phase: $phase"

            if [ "$phase" == "Validated" ] || [ "$phase" == "Processed" ]; then
                record_test_result "Simple Scale Intent Validation" "PASS" "Phase: $phase"
            else
                record_test_result "Simple Scale Intent Validation" "FAIL" "Unexpected phase: $phase"
            fi
        else
            record_test_result "Simple Scale Intent Creation" "FAIL" "Intent not found after creation"
        fi
    else
        log_warning "測試文件不存在: $test_dir/01-simple-scale.yaml"
    fi

    # 3.2 創建 QoS 優化 Intent
    log_info "3.2 測試 QoS 優化 Intent"
    if [ -f "$test_dir/02-qos-optimization.yaml" ]; then
        kubectl apply -f "$test_dir/02-qos-optimization.yaml" 2>&1 | head -1
        sleep 3

        if kubectl get networkintent e2e-test-qos-optimization -n default 2>/dev/null | grep -q "e2e-test-qos-optimization"; then
            record_test_result "QoS Optimization Intent Creation" "PASS" "Intent created successfully"
        else
            record_test_result "QoS Optimization Intent Creation" "FAIL" "Intent not found after creation"
        fi
    fi

    # 3.3 創建複雜 Intent
    log_info "3.3 測試複雜多參數 Intent"
    if [ -f "$test_dir/03-complex-intent.yaml" ]; then
        kubectl apply -f "$test_dir/03-complex-intent.yaml" 2>&1 | head -1
        sleep 3

        if kubectl get networkintent e2e-test-complex-intent -n default 2>/dev/null | grep -q "e2e-test-complex-intent"; then
            record_test_result "Complex Intent Creation" "PASS" "Intent created successfully"
        else
            record_test_result "Complex Intent Creation" "FAIL" "Intent not found after creation"
        fi
    fi

    # 3.4 查看所有測試 Intent
    log_info "3.4 列出所有測試 Intent"
    kubectl get networkintents -n default -l test-suite=intent-ric-e2e

    log_success "NetworkIntent 創建測試完成"
}

# 階段 4: Intent Operator 日誌分析
test_operator_logs() {
    log_info "==================== 階段 4: Intent Operator 日誌分析 ===================="

    log_info "4.1 檢查 Intent Operator 最近日誌"
    local pod_name=$(kubectl get pods -n nephoran-system -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}')

    if [ -n "$pod_name" ]; then
        log_info "從 Pod $pod_name 獲取日誌..."
        kubectl logs -n nephoran-system "$pod_name" --tail=50 | tail -20

        # 檢查是否有錯誤
        local error_count=$(kubectl logs -n nephoran-system "$pod_name" --tail=100 2>/dev/null | grep -c "ERROR" 2>/dev/null || echo "0")
        error_count=$(echo "$error_count" | tr -d '\n' | tr -d ' ')

        if [ "$error_count" == "0" ]; then
            record_test_result "Operator Error-Free Logs" "PASS" "No errors in recent logs"
        else
            record_test_result "Operator Error-Free Logs" "FAIL" "Found $error_count errors in logs"
        fi
    else
        record_test_result "Operator Logs Retrieval" "FAIL" "Pod not found"
    fi

    log_success "日誌分析完成"
}

# 階段 5: A1 Policy 驗證
test_a1_policy_verification() {
    log_info "==================== 階段 5: A1 Policy 驗證 ===================="

    log_info "5.1 查詢現有 Policies"
    local policies=$(kubectl exec test-network-tools -- curl -s http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000/A1-P/v2/policytypes/100/policies 2>&1)

    echo "Current policies: $policies"

    if [ "$policies" == "[]" ]; then
        log_warning "目前沒有任何 A1 Policy - Intent Operator 可能尚未實現 A1 集成"
        record_test_result "A1 Policy Integration" "FAIL" "No policies found - A1 integration not implemented"
    else
        record_test_result "A1 Policy Integration" "PASS" "Found policies: $policies"

        # 檢查每個 policy 的狀態
        for policy_id in $(echo "$policies" | jq -r '.[]' 2>/dev/null); do
            log_info "檢查 Policy: $policy_id"
            local policy_detail=$(kubectl exec test-network-tools -- curl -s "http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000/A1-P/v2/policytypes/100/policies/$policy_id" 2>&1)
            echo "$policy_detail" | jq '.' 2>/dev/null || echo "$policy_detail"
        done
    fi

    log_success "A1 Policy 驗證完成"
}

# 階段 6: 錯誤處理測試
test_error_handling() {
    log_info "==================== 階段 6: 錯誤處理測試 ===================="

    # 6.1 測試無效的 Intent (空 source)
    log_info "6.1 測試無效 Intent (空 source 字段)"

    cat <<EOF | kubectl apply -f - 2>&1 | head -1
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: e2e-test-invalid-intent
  namespace: default
spec:
  source: ""
  intentType: "scaling"
  target: "test"
  namespace: "default"
  replicas: 1
EOF

    sleep 3

    local invalid_phase=$(kubectl get networkintent e2e-test-invalid-intent -n default -o jsonpath='{.status.phase}' 2>/dev/null || echo "NOT_FOUND")

    if [ "$invalid_phase" == "Error" ]; then
        record_test_result "Invalid Intent Rejection" "PASS" "Invalid intent properly rejected"
    else
        record_test_result "Invalid Intent Rejection" "FAIL" "Invalid intent not rejected (phase: $invalid_phase)"
    fi

    log_success "錯誤處理測試完成"
}

# 清理函數
cleanup() {
    log_info "==================== 清理測試資源 ===================="

    log_info "刪除測試 NetworkIntents"
    kubectl delete networkintents -n default -l test-suite=intent-ric-e2e 2>/dev/null || true
    kubectl delete networkintent e2e-test-invalid-intent -n default 2>/dev/null || true

    log_info "刪除測試 Pod"
    kubectl delete pod test-network-tools 2>/dev/null || true

    log_success "清理完成"
}

# 生成測試報告
generate_report() {
    log_info "==================== 測試報告 ===================="

    echo ""
    echo "======================================================================"
    echo "  Nephoran Intent Operator 與 O-RAN SC RIC 端到端整合測試報告"
    echo "======================================================================"
    echo ""
    echo "測試時間: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    echo "總測試數: $TOTAL_TESTS"
    echo -e "${GREEN}通過: $PASSED_TESTS${NC}"
    echo -e "${RED}失敗: $FAILED_TESTS${NC}"
    echo ""

    local success_rate=0
    if [ $TOTAL_TESTS -gt 0 ]; then
        success_rate=$(awk "BEGIN {printf \"%.2f\", ($PASSED_TESTS/$TOTAL_TESTS)*100}")
    fi

    echo "成功率: ${success_rate}%"
    echo ""

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 所有測試通過！${NC}"
        echo ""
        return 0
    else
        echo -e "${RED}⚠️  有測試失敗，請檢查上面的日誌${NC}"
        echo ""
        return 1
    fi
}

# 主函數
main() {
    echo "======================================================================"
    echo "  Nephoran Intent Operator 與 O-RAN SC RIC 端到端整合測試"
    echo "  版本: 1.0.0"
    echo "  日期: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "======================================================================"
    echo ""

    check_prerequisites

    # 執行測試階段
    test_environment_validation || true
    test_network_connectivity || true
    test_intent_creation || true
    test_operator_logs || true
    test_a1_policy_verification || true
    test_error_handling || true

    # 清理（如果指定了 --no-cleanup 則跳過）
    if [ "$1" != "--no-cleanup" ]; then
        cleanup
    else
        log_info "跳過清理（保留測試資源用於調試）"
    fi

    # 生成報告
    generate_report
}

# 執行主函數
main "$@"
