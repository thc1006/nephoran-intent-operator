#!/usr/bin/env bash
# =============================================================================
# test-rag-pipeline.sh
# RAG Service Integration E2E Test
#
# Validates the RAG (Retrieval-Augmented Generation) pipeline:
#   1. Health endpoint (/healthz, /readyz, /health)
#   2. Weaviate connectivity (vector store)
#   3. Ollama LLM inference
#   4. Complete RAG flow via /process endpoint
#   5. Stats endpoint for observability
#   6. Response format and timing measurements
#
# Prerequisites:
#   - RAG service deployed (FastAPI on port 8000)
#   - Weaviate deployed (port 8080)
#   - Ollama deployed with at least one model (port 11434)
#   - kubectl port-forward or direct network access
#
# Usage:
#   ./test-rag-pipeline.sh [--rag-url <url>] [--weaviate-url <url>]
#                          [--ollama-url <url>] [--timeout <seconds>]
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
RAG_URL="${RAG_URL:-http://localhost:8000}"
WEAVIATE_URL="${WEAVIATE_URL:-http://localhost:8080}"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
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

assert_json_field() {
    local description="$1" json="$2" field="$3"
    local value
    value=$(echo "${json}" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('${field}',''))" 2>/dev/null || echo "")
    if [[ -n "${value}" ]]; then
        log_pass "${description}: ${field}='${value}'"
    else
        log_fail "${description}: field '${field}' missing or empty"
    fi
}

timed_request() {
    local url="$1" method="${2:-GET}" data="${3:-}"
    local start_ms end_ms duration_ms response
    start_ms=$(date +%s%N)

    if [[ "${method}" == "POST" ]]; then
        response=$(curl -s --max-time "${TIMEOUT}" -X POST \
            -H "Content-Type: application/json" \
            -d "${data}" \
            "${url}" 2>/dev/null || echo '{"error":"request_failed"}')
    else
        response=$(curl -s --max-time "${TIMEOUT}" "${url}" 2>/dev/null || echo '{"error":"request_failed"}')
    fi

    end_ms=$(date +%s%N)
    duration_ms=$(( (end_ms - start_ms) / 1000000 ))

    echo "${response}"
    log_info "Request to ${url} completed in ${duration_ms}ms"
}

# ---------------------------------------------------------------------------
# Parse CLI arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --rag-url)      RAG_URL="$2";      shift 2 ;;
        --weaviate-url) WEAVIATE_URL="$2"; shift 2 ;;
        --ollama-url)   OLLAMA_URL="$2";   shift 2 ;;
        --timeout)      TIMEOUT="$2";      shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--rag-url <url>] [--weaviate-url <url>] [--ollama-url <url>] [--timeout <sec>]"
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
echo "  RAG Pipeline Integration E2E Test"
echo "  RAG URL:      ${RAG_URL}"
echo "  Weaviate URL: ${WEAVIATE_URL}"
echo "  Ollama URL:   ${OLLAMA_URL}"
echo "  Timeout:      ${TIMEOUT}s"
echo "  Timestamp:    $(date -Iseconds)"
echo "============================================================"
echo ""

# ---------------------------------------------------------------------------
# Phase 1: Service Reachability
# ---------------------------------------------------------------------------
log_section "PHASE 1: Service Reachability"

# RAG service health endpoints
log_info "Testing RAG /healthz endpoint..."
assert_http_status "RAG /healthz" "200" "${RAG_URL}/healthz"

log_info "Testing RAG /health endpoint..."
assert_http_status "RAG /health" "200" "${RAG_URL}/health"

log_info "Testing RAG /readyz endpoint..."
READYZ_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${RAG_URL}/readyz" 2>/dev/null || echo "000")
if [[ "${READYZ_CODE}" == "200" ]]; then
    log_pass "RAG /readyz: HTTP 200 (pipeline initialized)"
elif [[ "${READYZ_CODE}" == "503" ]]; then
    log_skip "RAG /readyz: HTTP 503 (pipeline not yet initialized -- downstream deps may be missing)"
else
    log_fail "RAG /readyz: unexpected HTTP ${READYZ_CODE}"
fi

# Weaviate health
log_info "Testing Weaviate readiness..."
WEAVIATE_READY_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${WEAVIATE_URL}/v1/.well-known/ready" 2>/dev/null || echo "000")
if [[ "${WEAVIATE_READY_CODE}" == "200" ]]; then
    log_pass "Weaviate ready: HTTP 200"
else
    log_fail "Weaviate not ready: HTTP ${WEAVIATE_READY_CODE}"
fi

# Ollama health
log_info "Testing Ollama API..."
OLLAMA_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${OLLAMA_URL}/api/tags" 2>/dev/null || echo "000")
if [[ "${OLLAMA_CODE}" == "200" ]]; then
    log_pass "Ollama API reachable: HTTP 200"
else
    log_fail "Ollama API unreachable: HTTP ${OLLAMA_CODE}"
fi

# ---------------------------------------------------------------------------
# Phase 2: Weaviate Validation
# ---------------------------------------------------------------------------
log_section "PHASE 2: Weaviate Vector Store"

log_info "Querying Weaviate schema..."
WEAVIATE_SCHEMA=$(curl -s --max-time 10 "${WEAVIATE_URL}/v1/schema" 2>/dev/null || echo '{}')

# Check if schema endpoint returns valid JSON
if echo "${WEAVIATE_SCHEMA}" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
    log_pass "Weaviate schema endpoint returns valid JSON"
else
    log_fail "Weaviate schema endpoint returned invalid JSON"
fi

# Count classes
CLASS_COUNT=$(echo "${WEAVIATE_SCHEMA}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    classes = data.get('classes', [])
    print(len(classes))
except:
    print(0)
" 2>/dev/null || echo "0")
log_info "Weaviate has ${CLASS_COUNT} classes defined"

# Test Weaviate meta endpoint
log_info "Querying Weaviate metadata..."
WEAVIATE_META=$(curl -s --max-time 10 "${WEAVIATE_URL}/v1/meta" 2>/dev/null || echo '{}')
WEAVIATE_VERSION=$(echo "${WEAVIATE_META}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('version','unknown'))" 2>/dev/null || echo "unknown")
if [[ "${WEAVIATE_VERSION}" != "unknown" ]]; then
    log_pass "Weaviate version: ${WEAVIATE_VERSION}"
else
    log_fail "Unable to determine Weaviate version"
fi

# ---------------------------------------------------------------------------
# Phase 3: Ollama LLM Validation
# ---------------------------------------------------------------------------
log_section "PHASE 3: Ollama LLM Inference"

log_info "Listing available Ollama models..."
OLLAMA_MODELS=$(curl -s --max-time 10 "${OLLAMA_URL}/api/tags" 2>/dev/null || echo '{"models":[]}')
MODEL_COUNT=$(echo "${OLLAMA_MODELS}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    models = data.get('models', [])
    for m in models:
        print(f\"  - {m.get('name', 'unknown')}\")
    print(f'COUNT:{len(models)}')
except:
    print('COUNT:0')
" 2>/dev/null || echo "COUNT:0")
echo "${MODEL_COUNT}" | grep -v "^COUNT:" || true
MODEL_NUM=$(echo "${MODEL_COUNT}" | grep "^COUNT:" | cut -d: -f2)
if [[ "${MODEL_NUM}" -gt 0 ]]; then
    log_pass "Ollama has ${MODEL_NUM} model(s) loaded"
else
    log_fail "No Ollama models found"
fi

# Test a basic generation request
log_info "Testing Ollama inference with a simple prompt..."
FIRST_MODEL=$(echo "${OLLAMA_MODELS}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    models = data.get('models', [])
    if models:
        print(models[0].get('name', ''))
except:
    pass
" 2>/dev/null || echo "")

if [[ -n "${FIRST_MODEL}" ]]; then
    log_info "Using model: ${FIRST_MODEL}"
    INFERENCE_START=$(date +%s%N)

    OLLAMA_RESPONSE=$(curl -s --max-time "${TIMEOUT}" -X POST \
        "${OLLAMA_URL}/api/generate" \
        -H "Content-Type: application/json" \
        -d "{\"model\":\"${FIRST_MODEL}\",\"prompt\":\"Reply with only: OK\",\"stream\":false}" \
        2>/dev/null || echo '{"error":"timeout"}')

    INFERENCE_END=$(date +%s%N)
    INFERENCE_MS=$(( (INFERENCE_END - INFERENCE_START) / 1000000 ))

    HAS_RESPONSE=$(echo "${OLLAMA_RESPONSE}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    resp = data.get('response', '')
    print('yes' if resp else 'no')
except:
    print('no')
" 2>/dev/null || echo "no")

    if [[ "${HAS_RESPONSE}" == "yes" ]]; then
        log_pass "Ollama inference successful (${INFERENCE_MS}ms)"
    else
        log_fail "Ollama inference returned empty response (${INFERENCE_MS}ms)"
    fi
else
    log_skip "No model available for inference test"
fi

# ---------------------------------------------------------------------------
# Phase 4: RAG Service Stats
# ---------------------------------------------------------------------------
log_section "PHASE 4: RAG Service Stats"

log_info "Fetching RAG service stats..."
STATS_RESPONSE=$(timed_request "${RAG_URL}/stats")

if echo "${STATS_RESPONSE}" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
    log_pass "RAG /stats returns valid JSON"
    assert_json_field "Stats has timestamp" "${STATS_RESPONSE}" "timestamp"
    assert_json_field "Stats has config" "${STATS_RESPONSE}" "config"
else
    log_fail "RAG /stats returned invalid response"
fi

# ---------------------------------------------------------------------------
# Phase 5: Complete RAG Flow
# ---------------------------------------------------------------------------
log_section "PHASE 5: Complete RAG Pipeline Flow"

INTENT_PAYLOAD='{"intent":"Deploy a high-availability AMF with 3 replicas in namespace 5g-core","intent_id":"e2e-test-001"}'

log_info "Sending intent processing request to ${RAG_URL}/process..."
RAG_START=$(date +%s%N)

RAG_RESPONSE=$(curl -s --max-time "${TIMEOUT}" -X POST \
    "${RAG_URL}/process" \
    -H "Content-Type: application/json" \
    -d "${INTENT_PAYLOAD}" 2>/dev/null || echo '{"error":"request_failed"}')

RAG_END=$(date +%s%N)
RAG_DURATION_MS=$(( (RAG_END - RAG_START) / 1000000 ))

log_info "RAG pipeline response time: ${RAG_DURATION_MS}ms"

# Validate response
RAG_HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time "${TIMEOUT}" -X POST \
    "${RAG_URL}/process" \
    -H "Content-Type: application/json" \
    -d "${INTENT_PAYLOAD}" 2>/dev/null || echo "000")

if [[ "${RAG_HTTP_CODE}" == "200" ]]; then
    log_pass "RAG /process returned HTTP 200"

    if echo "${RAG_RESPONSE}" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
        log_pass "RAG response is valid JSON"
    else
        log_fail "RAG response is not valid JSON"
    fi

    # Performance check: RAG pipeline should respond within a reasonable time
    if [[ ${RAG_DURATION_MS} -lt 30000 ]]; then
        log_pass "RAG response within 30s threshold (${RAG_DURATION_MS}ms)"
    else
        log_fail "RAG response exceeded 30s threshold (${RAG_DURATION_MS}ms)"
    fi
elif [[ "${RAG_HTTP_CODE}" == "503" ]]; then
    log_skip "RAG /process returned 503 -- pipeline not initialized (check LLM provider config)"
else
    log_fail "RAG /process returned unexpected HTTP ${RAG_HTTP_CODE}"
fi

# ---------------------------------------------------------------------------
# Phase 6: Legacy Endpoint Compatibility
# ---------------------------------------------------------------------------
log_section "PHASE 6: Legacy Endpoint Compatibility"

log_info "Testing legacy /process_intent endpoint..."
LEGACY_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time "${TIMEOUT}" -X POST \
    "${RAG_URL}/process_intent" \
    -H "Content-Type: application/json" \
    -d '{"intent":"Scale UPF to 2 replicas"}' 2>/dev/null || echo "000")

if [[ "${LEGACY_CODE}" == "200" ]]; then
    log_pass "Legacy /process_intent returns HTTP 200"
elif [[ "${LEGACY_CODE}" == "503" ]]; then
    log_skip "Legacy /process_intent returns 503 (pipeline not initialized)"
else
    log_fail "Legacy /process_intent returns unexpected HTTP ${LEGACY_CODE}"
fi

# ---------------------------------------------------------------------------
# Phase 7: OpenAPI Documentation
# ---------------------------------------------------------------------------
log_section "PHASE 7: OpenAPI Documentation"

log_info "Checking FastAPI /docs (Swagger UI)..."
assert_http_status "Swagger UI /docs" "200" "${RAG_URL}/docs"

log_info "Checking /openapi.json schema..."
OPENAPI_RESPONSE=$(curl -s --max-time 10 "${RAG_URL}/openapi.json" 2>/dev/null || echo "{}")
if echo "${OPENAPI_RESPONSE}" | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'paths' in d" 2>/dev/null; then
    log_pass "OpenAPI schema has paths defined"
else
    log_fail "OpenAPI schema missing or incomplete"
fi

# ---------------------------------------------------------------------------
# Phase 8: Timing Summary
# ---------------------------------------------------------------------------
log_section "PHASE 8: Timing Summary"

TEST_END_TIME=$(date +%s)
TOTAL_DURATION=$((TEST_END_TIME - TEST_START_TIME))

echo ""
echo "  Timing Measurements:"
echo "  - RAG /process:     ${RAG_DURATION_MS}ms"
if [[ -n "${INFERENCE_MS:-}" ]]; then
    echo "  - Ollama inference: ${INFERENCE_MS}ms"
fi
echo "  - Total test time:  ${TOTAL_DURATION}s"
echo ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
log_section "TEST SUMMARY"

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
