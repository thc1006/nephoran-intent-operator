#!/bin/bash
# LLM Concurrent Request Benchmark
# Tests how models handle simultaneous requests

set -euo pipefail

OLLAMA_URL="http://localhost:11434"
RESULTS_DIR="/home/thc1006/dev/nephoran-intent-operator/benchmarks"
PROMPT="Explain the key differences between TCP and UDP protocols in networking."

# Use the fastest model for concurrent testing
MODEL="llama3.1:8b-instruct-q5_K_M"

echo "=== Concurrent Request Benchmark ==="
echo "Model: $MODEL"
echo "Timestamp: $(date -Iseconds)"
echo ""

# Warm up
curl -s "$OLLAMA_URL/api/generate" -d "{\"model\": \"$MODEL\", \"prompt\": \"hello\", \"stream\": false, \"options\": {\"num_predict\": 1}}" > /dev/null 2>&1
sleep 2

# Function to run a single request and capture timing
run_request() {
  local id=$1
  local start=$(date +%s%N)
  local result=$(curl -s "$OLLAMA_URL/api/generate" \
    -d "{\"model\": \"$MODEL\", \"prompt\": \"$PROMPT\", \"stream\": false, \"options\": {\"num_predict\": 128}}" 2>&1)
  local end=$(date +%s%N)

  local total_duration=$(echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('total_duration',0))" 2>/dev/null || echo "0")
  local eval_count=$(echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('eval_count',0))" 2>/dev/null || echo "0")
  local eval_duration=$(echo "$result" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('eval_duration',0))" 2>/dev/null || echo "0")
  local wall_time=$(python3 -c "print(round(($end - $start) / 1e9, 4))")

  local tokens_per_sec="0"
  if [ "$eval_duration" != "0" ] && [ "$eval_count" != "0" ]; then
    tokens_per_sec=$(python3 -c "print(round($eval_count / ($eval_duration / 1e9), 2))" 2>/dev/null || echo "0")
  fi

  echo "req_${id}:wall=${wall_time}s:tokens=${eval_count}:tps=${tokens_per_sec}"
}

# Baseline: 1 sequential request
echo "--- Baseline (1 request) ---"
run_request 1
echo ""

# 2 concurrent requests
echo "--- 2 Concurrent Requests ---"
nvidia-smi --query-gpu=memory.used,utilization.gpu --format=csv,noheader > "${RESULTS_DIR}/gpu_during_concurrent_2.txt" &
GPU_MON_PID=$!
for i in 1 2; do
  run_request $i &
done
wait
kill $GPU_MON_PID 2>/dev/null || true
echo "GPU during test: $(cat "${RESULTS_DIR}/gpu_during_concurrent_2.txt")"
echo ""

sleep 3

# 4 concurrent requests
echo "--- 4 Concurrent Requests ---"
nvidia-smi --query-gpu=memory.used,utilization.gpu --format=csv,noheader > "${RESULTS_DIR}/gpu_during_concurrent_4.txt" &
GPU_MON_PID=$!
for i in 1 2 3 4; do
  run_request $i &
done
wait
kill $GPU_MON_PID 2>/dev/null || true
echo "GPU during test: $(cat "${RESULTS_DIR}/gpu_during_concurrent_4.txt")"
echo ""

echo "=== Concurrent benchmark complete ==="
