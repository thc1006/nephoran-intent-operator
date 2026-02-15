#!/usr/bin/env bash
# =============================================================================
# test-monitoring.sh
# Monitoring Stack Verification E2E Test
#
# Validates the observability infrastructure:
#   1. Prometheus server is running and scraping targets
#   2. PromQL queries return valid data
#   3. Grafana is accessible and healthy
#   4. DCGM GPU metrics are available
#   5. ServiceMonitor CRDs exist
#   6. Alertmanager connectivity (optional)
#
# Prerequisites:
#   - Prometheus deployed (kube-prometheus-stack or standalone)
#   - Grafana deployed
#   - DCGM Exporter running (if GPU nodes exist)
#   - kubectl configured
#
# Usage:
#   ./test-monitoring.sh [--prometheus-url <url>] [--grafana-url <url>]
#                        [--namespace <ns>] [--timeout <seconds>]
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
ALERTMANAGER_URL="${ALERTMANAGER_URL:-http://localhost:9093}"
MONITORING_NS="${MONITORING_NS:-monitoring}"
TIMEOUT="${E2E_TIMEOUT:-60}"
PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0
TEST_START_TIME=$(date +%s)

# ---------------------------------------------------------------------------
# Color helpers
# ---------------------------------------------------------------------------
if [[ -t 1 ]]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
    BLUE='\033[0;34m'; NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; NC=''
fi

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log_info()    { echo -e "${BLUE}[INFO]${NC}  $(date '+%H:%M:%S') $*"; }
log_pass()    { echo -e "${GREEN}[PASS]${NC}  $(date '+%H:%M:%S') $*"; PASS_COUNT=$((PASS_COUNT + 1)); }
log_fail()    { echo -e "${RED}[FAIL]${NC}  $(date '+%H:%M:%S') $*"; FAIL_COUNT=$((FAIL_COUNT + 1)); }
log_skip()    { echo -e "${YELLOW}[SKIP]${NC}  $(date '+%H:%M:%S') $*"; SKIP_COUNT=$((SKIP_COUNT + 1)); }
log_section() { echo -e "\n${BLUE}=== $* ===${NC}"; }

assert_http_status() {
    local description="$1" expected_code="$2" url="$3"
    local actual_code
    actual_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${url}" 2>/dev/null || echo "000")
    if [[ "${actual_code}" == "${expected_code}" ]]; then
        log_pass "${description}: HTTP ${actual_code}"
    else
        log_fail "${description}: expected HTTP ${expected_code}, got HTTP ${actual_code}"
    fi
}

promql_query() {
    local query="$1"
    local encoded_query
    encoded_query=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${query}'))" 2>/dev/null || echo "${query}")
    curl -s --max-time 10 "${PROMETHEUS_URL}/api/v1/query?query=${encoded_query}" 2>/dev/null || echo '{"status":"error"}'
}

promql_has_results() {
    local description="$1" query="$2"
    local response
    response=$(promql_query "${query}")

    local status
    status=$(echo "${response}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','error'))" 2>/dev/null || echo "error")

    if [[ "${status}" != "success" ]]; then
        log_fail "${description}: PromQL query failed (status: ${status})"
        return 1
    fi

    local result_count
    result_count=$(echo "${response}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    results = data.get('data', {}).get('result', [])
    print(len(results))
except:
    print(0)
" 2>/dev/null || echo "0")

    if [[ "${result_count}" -gt 0 ]]; then
        log_pass "${description}: ${result_count} result(s)"
        return 0
    else
        log_fail "${description}: no results returned"
        return 1
    fi
}

# ---------------------------------------------------------------------------
# Parse CLI arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --prometheus-url) PROMETHEUS_URL="$2"; shift 2 ;;
        --grafana-url)    GRAFANA_URL="$2";    shift 2 ;;
        --alertmanager-url) ALERTMANAGER_URL="$2"; shift 2 ;;
        --namespace)      MONITORING_NS="$2";  shift 2 ;;
        --timeout)        TIMEOUT="$2";        shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--prometheus-url <url>] [--grafana-url <url>] [--namespace <ns>] [--timeout <sec>]"
            exit 0
            ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

# =============================================================================
# TEST EXECUTION
# =============================================================================

echo ""
echo "============================================================"
echo "  Monitoring Stack Verification E2E Test"
echo "  Prometheus:    ${PROMETHEUS_URL}"
echo "  Grafana:       ${GRAFANA_URL}"
echo "  Alertmanager:  ${ALERTMANAGER_URL}"
echo "  Namespace:     ${MONITORING_NS}"
echo "  Timestamp:     $(date -Iseconds)"
echo "============================================================"
echo ""

# ---------------------------------------------------------------------------
# Phase 1: Kubernetes Monitoring Resources
# ---------------------------------------------------------------------------
log_section "PHASE 1: Kubernetes Monitoring Resources"

# Check monitoring namespace
log_info "Checking monitoring namespace '${MONITORING_NS}'..."
if kubectl get namespace "${MONITORING_NS}" >/dev/null 2>&1; then
    log_pass "Monitoring namespace '${MONITORING_NS}' exists"
else
    log_skip "Monitoring namespace '${MONITORING_NS}' not found"
fi

# Check monitoring pods
log_info "Listing monitoring pods..."
MON_PODS=$(kubectl get pods -n "${MONITORING_NS}" --no-headers 2>/dev/null || echo "")
if [[ -n "${MON_PODS}" ]]; then
    MON_POD_COUNT=$(echo "${MON_PODS}" | wc -l)
    log_pass "Found ${MON_POD_COUNT} monitoring pod(s)"

    RUNNING_COUNT=$(echo "${MON_PODS}" | grep -c "Running" || echo "0")
    log_info "Running pods: ${RUNNING_COUNT}/${MON_POD_COUNT}"

    # Check Prometheus pods specifically
    PROM_PODS=$(echo "${MON_PODS}" | grep -i "prometheus" || echo "")
    if [[ -n "${PROM_PODS}" ]]; then
        log_pass "Prometheus pod(s) found"
    else
        log_skip "No Prometheus pods found in ${MONITORING_NS}"
    fi

    # Check Grafana pods
    GRAFANA_PODS=$(echo "${MON_PODS}" | grep -i "grafana" || echo "")
    if [[ -n "${GRAFANA_PODS}" ]]; then
        log_pass "Grafana pod(s) found"
    else
        log_skip "No Grafana pods found in ${MONITORING_NS}"
    fi
else
    log_skip "No pods in monitoring namespace"
fi

# Check ServiceMonitor CRD
log_info "Checking ServiceMonitor CRD..."
SM_CRD=$(kubectl get crd servicemonitors.monitoring.coreos.com -o name 2>/dev/null || echo "")
if [[ -n "${SM_CRD}" ]]; then
    log_pass "ServiceMonitor CRD registered"

    # Count ServiceMonitors
    SM_COUNT=$(kubectl get servicemonitor --all-namespaces --no-headers 2>/dev/null | wc -l || echo "0")
    log_info "Total ServiceMonitors across cluster: ${SM_COUNT}"
else
    log_skip "ServiceMonitor CRD not found (Prometheus Operator may not be installed)"
fi

# Check PrometheusRule CRD
log_info "Checking PrometheusRule CRD..."
PR_CRD=$(kubectl get crd prometheusrules.monitoring.coreos.com -o name 2>/dev/null || echo "")
if [[ -n "${PR_CRD}" ]]; then
    log_pass "PrometheusRule CRD registered"
else
    log_skip "PrometheusRule CRD not found"
fi

# ---------------------------------------------------------------------------
# Phase 2: Prometheus Connectivity
# ---------------------------------------------------------------------------
log_section "PHASE 2: Prometheus Connectivity"

# Basic connectivity
log_info "Testing Prometheus API..."
PROM_READY_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${PROMETHEUS_URL}/-/ready" 2>/dev/null || echo "000")
if [[ "${PROM_READY_CODE}" == "200" ]]; then
    log_pass "Prometheus ready: HTTP 200"
else
    log_fail "Prometheus not ready: HTTP ${PROM_READY_CODE}"
fi

PROM_HEALTHY_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${PROMETHEUS_URL}/-/healthy" 2>/dev/null || echo "000")
if [[ "${PROM_HEALTHY_CODE}" == "200" ]]; then
    log_pass "Prometheus healthy: HTTP 200"
else
    log_fail "Prometheus not healthy: HTTP ${PROM_HEALTHY_CODE}"
fi

# ---------------------------------------------------------------------------
# Phase 3: Prometheus Targets
# ---------------------------------------------------------------------------
log_section "PHASE 3: Prometheus Scrape Targets"

log_info "Querying active scrape targets..."
TARGETS_RESPONSE=$(curl -s --max-time 10 "${PROMETHEUS_URL}/api/v1/targets" 2>/dev/null || echo '{"status":"error"}')
TARGETS_STATUS=$(echo "${TARGETS_RESPONSE}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status','error'))" 2>/dev/null || echo "error")

if [[ "${TARGETS_STATUS}" == "success" ]]; then
    # Parse target health distribution
    TARGET_STATS=$(echo "${TARGETS_RESPONSE}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    active = data.get('data', {}).get('activeTargets', [])
    up = sum(1 for t in active if t.get('health') == 'up')
    down = sum(1 for t in active if t.get('health') == 'down')
    unknown = sum(1 for t in active if t.get('health') not in ('up', 'down'))
    print(f'total={len(active)} up={up} down={down} unknown={unknown}')
except:
    print('total=0 up=0 down=0 unknown=0')
" 2>/dev/null || echo "total=0 up=0 down=0 unknown=0")

    log_info "Target health: ${TARGET_STATS}"

    TOTAL_TARGETS=$(echo "${TARGET_STATS}" | grep -oP 'total=\K[0-9]+')
    UP_TARGETS=$(echo "${TARGET_STATS}" | grep -oP 'up=\K[0-9]+')
    DOWN_TARGETS=$(echo "${TARGET_STATS}" | grep -oP 'down=\K[0-9]+')

    if [[ "${TOTAL_TARGETS}" -gt 0 ]]; then
        log_pass "Prometheus has ${TOTAL_TARGETS} active target(s)"
    else
        log_fail "No active Prometheus targets"
    fi

    if [[ "${UP_TARGETS}" -gt 0 ]]; then
        log_pass "${UP_TARGETS} target(s) are UP"
    else
        log_fail "No targets reporting UP"
    fi

    if [[ "${DOWN_TARGETS}" -gt 0 ]]; then
        log_info "WARNING: ${DOWN_TARGETS} target(s) are DOWN"
    fi

    # List first few target jobs
    echo "${TARGETS_RESPONSE}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    active = data.get('data', {}).get('activeTargets', [])
    jobs = set(t.get('labels', {}).get('job', 'unknown') for t in active[:20])
    for j in sorted(jobs):
        print(f'  - {j}')
except:
    pass
" 2>/dev/null || true
else
    log_fail "Failed to query Prometheus targets"
fi

# ---------------------------------------------------------------------------
# Phase 4: PromQL Metric Queries
# ---------------------------------------------------------------------------
log_section "PHASE 4: PromQL Metric Queries"

# Basic Kubernetes metrics
promql_has_results "kube_pod_info metric" "kube_pod_info" || true
promql_has_results "up metric" "up" || true
promql_has_results "process_cpu_seconds_total" "process_cpu_seconds_total" || true
promql_has_results "node_memory_MemTotal_bytes" "node_memory_MemTotal_bytes" || true

# Container metrics
promql_has_results "container_cpu_usage_seconds_total" "container_cpu_usage_seconds_total" || true

# Kubernetes state metrics
promql_has_results "kube_node_info" "kube_node_info" || true

# ---------------------------------------------------------------------------
# Phase 5: DCGM GPU Metrics
# ---------------------------------------------------------------------------
log_section "PHASE 5: DCGM GPU Metrics"

# Check DCGM exporter pods
log_info "Checking DCGM Exporter pods..."
DCGM_PODS=$(kubectl get pods --all-namespaces -l app=nvidia-dcgm-exporter --no-headers 2>/dev/null || \
            kubectl get pods --all-namespaces --no-headers 2>/dev/null | grep -i "dcgm" || echo "")
if [[ -n "${DCGM_PODS}" ]]; then
    log_pass "DCGM Exporter pod(s) found"
else
    log_skip "No DCGM Exporter pods found"
fi

# Query GPU metrics via Prometheus
log_info "Querying GPU metrics via Prometheus..."

GPU_METRICS=(
    "DCGM_FI_DEV_GPU_UTIL:GPU Utilization"
    "DCGM_FI_DEV_MEM_COPY_UTIL:Memory Utilization"
    "DCGM_FI_DEV_GPU_TEMP:GPU Temperature"
    "DCGM_FI_DEV_POWER_USAGE:Power Usage"
    "DCGM_FI_DEV_FB_FREE:Framebuffer Free"
    "DCGM_FI_DEV_FB_USED:Framebuffer Used"
)

GPU_METRICS_FOUND=0
for metric_info in "${GPU_METRICS[@]}"; do
    IFS=':' read -r metric_name metric_desc <<< "${metric_info}"
    METRIC_RESPONSE=$(promql_query "${metric_name}")
    METRIC_STATUS=$(echo "${METRIC_RESPONSE}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    results = data.get('data', {}).get('result', [])
    if results:
        val = results[0].get('value', [None, 'N/A'])[1]
        print(f'found:{val}')
    else:
        print('empty')
except:
    print('error')
" 2>/dev/null || echo "error")

    if [[ "${METRIC_STATUS}" == empty ]] || [[ "${METRIC_STATUS}" == error ]]; then
        log_skip "${metric_desc} (${metric_name}): not available"
    else
        VALUE=$(echo "${METRIC_STATUS}" | cut -d: -f2)
        log_pass "${metric_desc} (${metric_name}): ${VALUE}"
        GPU_METRICS_FOUND=$((GPU_METRICS_FOUND + 1))
    fi
done

if [[ ${GPU_METRICS_FOUND} -gt 0 ]]; then
    log_pass "Found ${GPU_METRICS_FOUND} GPU metric(s) in Prometheus"
else
    log_skip "No DCGM GPU metrics found in Prometheus"
fi

# ---------------------------------------------------------------------------
# Phase 6: Grafana Connectivity
# ---------------------------------------------------------------------------
log_section "PHASE 6: Grafana"

# Basic health
log_info "Testing Grafana health..."
GRAFANA_HEALTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${GRAFANA_URL}/api/health" 2>/dev/null || echo "000")
if [[ "${GRAFANA_HEALTH_CODE}" == "200" ]]; then
    log_pass "Grafana healthy: HTTP 200"

    GRAFANA_HEALTH=$(curl -s --max-time 10 "${GRAFANA_URL}/api/health" 2>/dev/null || echo '{}')
    GRAFANA_DB_STATUS=$(echo "${GRAFANA_HEALTH}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('database','unknown'))" 2>/dev/null || echo "unknown")
    log_info "Grafana database status: ${GRAFANA_DB_STATUS}"
else
    log_fail "Grafana not accessible: HTTP ${GRAFANA_HEALTH_CODE}"
fi

# List dashboards (requires auth -- try anonymous or admin:admin)
log_info "Listing Grafana dashboards..."
DASHBOARD_RESPONSE=$(curl -s --max-time 10 "${GRAFANA_URL}/api/search?type=dash-db" 2>/dev/null || \
                     curl -s --max-time 10 -u admin:admin "${GRAFANA_URL}/api/search?type=dash-db" 2>/dev/null || echo '[]')

DASHBOARD_COUNT=$(echo "${DASHBOARD_RESPONSE}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if isinstance(data, list):
        for d in data[:10]:
            print(f\"  - {d.get('title', 'untitled')} (uid: {d.get('uid', 'N/A')})\")
        print(f'COUNT:{len(data)}')
    else:
        print('COUNT:0')
except:
    print('COUNT:0')
" 2>/dev/null || echo "COUNT:0")

echo "${DASHBOARD_COUNT}" | grep -v "^COUNT:" || true
DB_NUM=$(echo "${DASHBOARD_COUNT}" | grep "^COUNT:" | cut -d: -f2)
if [[ "${DB_NUM}" -gt 0 ]]; then
    log_pass "Found ${DB_NUM} Grafana dashboard(s)"
else
    log_skip "No dashboards found (may require authentication)"
fi

# Check datasources
log_info "Listing Grafana datasources..."
DS_RESPONSE=$(curl -s --max-time 10 "${GRAFANA_URL}/api/datasources" 2>/dev/null || \
              curl -s --max-time 10 -u admin:admin "${GRAFANA_URL}/api/datasources" 2>/dev/null || echo '[]')
DS_COUNT=$(echo "${DS_RESPONSE}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if isinstance(data, list):
        for d in data:
            print(f\"  - {d.get('name', 'unnamed')} (type: {d.get('type', 'unknown')})\")
        print(f'COUNT:{len(data)}')
    else:
        print('COUNT:0')
except:
    print('COUNT:0')
" 2>/dev/null || echo "COUNT:0")

echo "${DS_COUNT}" | grep -v "^COUNT:" || true
DS_NUM=$(echo "${DS_COUNT}" | grep "^COUNT:" | cut -d: -f2)
if [[ "${DS_NUM}" -gt 0 ]]; then
    log_pass "Found ${DS_NUM} Grafana datasource(s)"
else
    log_skip "No datasources found (may require authentication)"
fi

# ---------------------------------------------------------------------------
# Phase 7: Alertmanager (optional)
# ---------------------------------------------------------------------------
log_section "PHASE 7: Alertmanager (Optional)"

log_info "Testing Alertmanager connectivity..."
AM_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${ALERTMANAGER_URL}/-/ready" 2>/dev/null || echo "000")
if [[ "${AM_CODE}" == "200" ]]; then
    log_pass "Alertmanager ready: HTTP 200"

    # Check active alerts
    ALERTS=$(curl -s --max-time 10 "${ALERTMANAGER_URL}/api/v2/alerts" 2>/dev/null || echo '[]')
    ALERT_COUNT=$(echo "${ALERTS}" | python3 -c "import sys,json; data=json.load(sys.stdin); print(len(data) if isinstance(data,list) else 0)" 2>/dev/null || echo "0")
    log_info "Active alerts: ${ALERT_COUNT}"
else
    log_skip "Alertmanager not accessible: HTTP ${AM_CODE}"
fi

# ---------------------------------------------------------------------------
# Phase 8: Nephoran-Specific Metrics
# ---------------------------------------------------------------------------
log_section "PHASE 8: Nephoran-Specific Metrics"

# Check for operator-specific metrics (if controller is running)
NEPHORAN_METRICS=(
    "controller_runtime_reconcile_total:Reconcile Total"
    "controller_runtime_reconcile_errors_total:Reconcile Errors"
    "workqueue_adds_total:Workqueue Adds"
    "workqueue_depth:Workqueue Depth"
)

for metric_info in "${NEPHORAN_METRICS[@]}"; do
    IFS=':' read -r metric_name metric_desc <<< "${metric_info}"
    promql_has_results "${metric_desc} (${metric_name})" "${metric_name}" || true
done

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
log_section "TEST SUMMARY"

TEST_END_TIME=$(date +%s)
TOTAL_DURATION=$((TEST_END_TIME - TEST_START_TIME))
TOTAL_TESTS=$((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))

echo ""
echo "  Total tests:  ${TOTAL_TESTS}"
echo -e "  Passed:       ${GREEN}${PASS_COUNT}${NC}"
echo -e "  Failed:       ${RED}${FAIL_COUNT}${NC}"
echo -e "  Skipped:      ${YELLOW}${SKIP_COUNT}${NC}"
echo "  Duration:     ${TOTAL_DURATION}s"
echo ""

if [[ ${FAIL_COUNT} -gt 0 ]]; then
    echo -e "${RED}RESULT: FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}RESULT: PASSED${NC}"
    exit 0
fi
