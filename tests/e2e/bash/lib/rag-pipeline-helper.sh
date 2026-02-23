#!/usr/bin/env bash
# =============================================================================
# rag-pipeline-helper.sh
# Helper functions for creating NetworkIntents via rag-pipeline CLI
# =============================================================================

# Get the path to rag-pipeline binary
RAG_PIPELINE_BIN="${RAG_PIPELINE_BIN:-$(dirname "$0")/../../../bin/rag-pipeline}"

# Check if rag-pipeline binary exists and is executable
check_rag_pipeline() {
    if [[ ! -x "${RAG_PIPELINE_BIN}" ]]; then
        echo "ERROR: rag-pipeline binary not found or not executable at ${RAG_PIPELINE_BIN}"
        echo "Build it with: go build -o bin/rag-pipeline cmd/rag-pipeline/main.go"
        return 1
    fi
    return 0
}

# Create NetworkIntent using rag-pipeline CLI
# Usage: create_intent_with_pipeline "natural language intent" "namespace" [timeout_seconds]
create_intent_with_pipeline() {
    local intent_text="$1"
    local target_namespace="${2:-default}"
    local timeout_sec="${3:-60}"
    local start_time end_time duration_ms

    # Check rag-pipeline binary exists
    if ! check_rag_pipeline; then
        return 1
    fi

    log_info "Creating NetworkIntent via rag-pipeline CLI"
    log_info "Intent: ${intent_text}"
    log_info "Target namespace: ${target_namespace}"

    start_time=$(date +%s%N)

    # Call rag-pipeline CLI
    local output exit_code
    output=$("${RAG_PIPELINE_BIN}" \
        --rag-url "${RAG_URL}" \
        --namespace "${target_namespace}" \
        --watch=false \
        --timeout "${timeout_sec}s" \
        "${intent_text}" 2>&1)
    exit_code=$?

    end_time=$(date +%s%N)
    duration_ms=$(( (end_time - start_time) / 1000000 ))

    log_info "rag-pipeline execution time: ${duration_ms}ms"

    # Check for success
    if [[ ${exit_code} -ne 0 ]]; then
        log_fail "rag-pipeline failed with exit code ${exit_code}"
        echo "${output}" | grep -E "ERROR|error|failed" | head -5 >&2
        return 1
    fi

    # Extract NetworkIntent name from output
    local intent_name
    intent_name=$(echo "${output}" | grep -E "^\s+Name\s+:" | awk '{print $3}')

    if [[ -z "${intent_name}" ]]; then
        log_fail "Could not extract NetworkIntent name from rag-pipeline output"
        echo "${output}" >&2
        return 1
    fi

    log_pass "NetworkIntent created: ${intent_name}"

    # Extract additional metrics from output
    local rag_time model_name retrieval_score
    rag_time=$(echo "${output}" | grep -E "^\s+Time\s+:" | awk '{print $3}' | sed 's/ms//')
    model_name=$(echo "${output}" | grep -E "^\s+Model\s+:" | awk '{print $3}')
    retrieval_score=$(echo "${output}" | grep -E "^\s+RAG Score\s+:" | awk '{print $4}')

    if [[ -n "${rag_time}" ]]; then
        log_info "RAG processing time: ${rag_time}ms"

        # Performance check
        if [[ ${rag_time%.*} -gt 10000 ]]; then
            log_fail "RAG processing time exceeded 10s threshold (${rag_time}ms)"
        else
            log_pass "RAG processing time within threshold (${rag_time}ms < 10s)"
        fi
    fi

    # Return intent name and metrics as JSON
    cat <<EOF
{
  "intent_name": "${intent_name}",
  "namespace": "${target_namespace}",
  "total_duration_ms": ${duration_ms},
  "rag_duration_ms": ${rag_time:-0},
  "model": "${model_name:-unknown}",
  "retrieval_score": ${retrieval_score:-0}
}
EOF

    return 0
}

# Create NetworkIntent and wait for phase
# Usage: create_and_wait_intent "intent text" "namespace" "expected_phase" [timeout_sec]
create_and_wait_intent() {
    local intent_text="$1"
    local target_namespace="$2"
    local expected_phase="${3:-Deployed}"
    local timeout_sec="${4:-120}"

    # Create intent
    local result
    result=$(create_intent_with_pipeline "${intent_text}" "${target_namespace}" ${timeout_sec})

    if [[ $? -ne 0 ]]; then
        log_fail "Failed to create NetworkIntent"
        return 1
    fi

    # Extract intent name from JSON result
    local intent_name
    intent_name=$(echo "${result}" | python3 -c "import sys,json; print(json.load(sys.stdin)['intent_name'])" 2>/dev/null)

    if [[ -z "${intent_name}" ]]; then
        log_fail "Could not determine NetworkIntent name"
        return 1
    fi

    # Wait for expected phase
    log_info "Waiting for NetworkIntent '${intent_name}' to reach phase '${expected_phase}'..."

    local retries=0
    local max_retries=$((timeout_sec / CHECK_INTERVAL))
    local last_phase=""

    while [[ ${retries} -lt ${max_retries} ]]; do
        local current_phase
        current_phase=$(kubectl get networkintent.nephoran.com "${intent_name}" -n "${target_namespace}" \
            -o jsonpath='{.status.phase}' 2>/dev/null || echo "Pending")

        if [[ "${current_phase}" != "${last_phase}" ]]; then
            log_info "Phase transition: ${last_phase:-None} -> ${current_phase}"
            last_phase="${current_phase}"
        fi

        # Success conditions
        if [[ "${current_phase}" == "${expected_phase}" ]]; then
            log_pass "NetworkIntent reached phase '${expected_phase}'"
            echo "${intent_name}"
            return 0
        fi

        # Error conditions
        if [[ "${current_phase}" == "Error" ]] || [[ "${current_phase}" == "Failed" ]]; then
            log_fail "NetworkIntent entered error state: ${current_phase}"
            kubectl get networkintent.nephoran.com "${intent_name}" -n "${target_namespace}" -o yaml | head -50 >&2
            return 1
        fi

        retries=$((retries + 1))
        sleep "${CHECK_INTERVAL}"
    done

    log_fail "Timeout waiting for phase '${expected_phase}' (last seen: ${current_phase})"
    kubectl get networkintent.nephoran.com "${intent_name}" -n "${target_namespace}" -o yaml | head -50 >&2
    return 1
}

# Verify NetworkIntent has expected replicas
# Usage: verify_intent_replicas "intent_name" "namespace" "expected_replicas"
verify_intent_replicas() {
    local intent_name="$1"
    local namespace="$2"
    local expected_replicas="$3"

    local actual_replicas
    actual_replicas=$(kubectl get networkintent.nephoran.com "${intent_name}" -n "${namespace}" \
        -o jsonpath='{.spec.desiredReplicas}' 2>/dev/null || echo "0")

    if [[ "${actual_replicas}" == "${expected_replicas}" ]]; then
        log_pass "NetworkIntent has correct replicas: ${actual_replicas}"
        return 0
    else
        log_fail "NetworkIntent replicas mismatch: expected ${expected_replicas}, got ${actual_replicas}"
        return 1
    fi
}

# Cleanup NetworkIntent
# Usage: cleanup_intent "intent_name" "namespace"
cleanup_intent() {
    local intent_name="$1"
    local namespace="$2"

    log_info "Cleaning up NetworkIntent: ${intent_name}"
    kubectl delete networkintent.nephoran.com "${intent_name}" -n "${namespace}" \
        --ignore-not-found=true --wait=true --timeout=30s &>/dev/null || true
}
