#!/bin/bash
# LLM Inference Benchmark Script
# Benchmarks all Ollama models with varying prompt lengths
# Measures: tokens/sec, time to first token, total time, VRAM usage

set -euo pipefail

RESULTS_FILE="/home/thc1006/dev/nephoran-intent-operator/benchmarks/llm-raw-results.json"
OLLAMA_URL="http://localhost:11434"

# Models to test
MODELS=(
  "llama3.1:8b-instruct-q5_K_M"
  "deepseek-coder-v2:16b-lite-instruct-q4_K_M"
  "mistral-nemo:12b-instruct-2407-q5_K_M"
  "qwen2.5:14b-instruct-q4_K_M"
)

# Prompts of varying lengths
SHORT_PROMPT="What is 2+2?"
MEDIUM_PROMPT="Explain the concept of distributed systems in cloud computing. Cover the key principles including fault tolerance, scalability, consistency models (CAP theorem), and how modern microservices architectures leverage these principles. Include examples of real-world distributed systems."
LONG_PROMPT="You are a telecommunications network architect. Provide a comprehensive analysis of O-RAN (Open Radio Access Network) architecture, covering the following aspects in detail: 1) The O-RAN Alliance specifications and their impact on the telecommunications industry, including the disaggregation of traditional RAN components. 2) The role of the RAN Intelligent Controller (RIC) in both near-real-time and non-real-time configurations, and how it enables AI/ML-driven network optimization. 3) The interfaces defined by O-RAN including E2, A1, O1, and O2, and their specific functions in the architecture. 4) How network functions like CU (Central Unit), DU (Distributed Unit), and RU (Radio Unit) interact in the disaggregated model. 5) The deployment challenges including interoperability, security considerations, and performance requirements. 6) The integration with 5G Core network and how O-RAN supports network slicing for different use cases such as eMBB, URLLC, and mMTC. Provide specific technical details and real-world deployment considerations."

echo "{ \"benchmarks\": [" > "$RESULTS_FILE"
FIRST=true

benchmark_model() {
  local model="$1"
  local prompt="$2"
  local prompt_type="$3"

  echo "  Testing $model with $prompt_type prompt..."

  # Get VRAM before
  local vram_before=$(nvidia-smi --query-gpu=memory.used --format=csv,noheader,nounits 2>/dev/null || echo "N/A")

  # Time the request and capture streaming response
  local start_time=$(date +%s%N)

  local response=$(curl -s -w "\n%{time_total}" "$OLLAMA_URL/api/generate" \
    -d "{\"model\": \"$model\", \"prompt\": \"$prompt\", \"stream\": false, \"options\": {\"num_predict\": 256}}" 2>&1)

  local end_time=$(date +%s%N)

  # Get VRAM after
  local vram_after=$(nvidia-smi --query-gpu=memory.used --format=csv,noheader,nounits 2>/dev/null || echo "N/A")

  # Parse response - extract from the JSON line (not the curl timing line)
  local json_line=$(echo "$response" | head -1)
  local curl_time=$(echo "$response" | tail -1)

  local total_duration=$(echo "$json_line" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('total_duration',0))" 2>/dev/null || echo "0")
  local load_duration=$(echo "$json_line" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('load_duration',0))" 2>/dev/null || echo "0")
  local prompt_eval_duration=$(echo "$json_line" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('prompt_eval_duration',0))" 2>/dev/null || echo "0")
  local eval_duration=$(echo "$json_line" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('eval_duration',0))" 2>/dev/null || echo "0")
  local eval_count=$(echo "$json_line" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('eval_count',0))" 2>/dev/null || echo "0")
  local prompt_eval_count=$(echo "$json_line" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('prompt_eval_count',0))" 2>/dev/null || echo "0")

  # Calculate tokens/sec
  local tokens_per_sec="0"
  if [ "$eval_duration" != "0" ] && [ "$eval_count" != "0" ]; then
    tokens_per_sec=$(python3 -c "print(round($eval_count / ($eval_duration / 1e9), 2))" 2>/dev/null || echo "0")
  fi

  # Calculate prompt processing speed
  local prompt_tokens_per_sec="0"
  if [ "$prompt_eval_duration" != "0" ] && [ "$prompt_eval_count" != "0" ]; then
    prompt_tokens_per_sec=$(python3 -c "print(round($prompt_eval_count / ($prompt_eval_duration / 1e9), 2))" 2>/dev/null || echo "0")
  fi

  # Time to first token (load + prompt eval)
  local ttft="0"
  if [ "$load_duration" != "0" ] || [ "$prompt_eval_duration" != "0" ]; then
    ttft=$(python3 -c "print(round(($load_duration + $prompt_eval_duration) / 1e9, 4))" 2>/dev/null || echo "0")
  fi

  local total_sec=$(python3 -c "print(round($total_duration / 1e9, 4))" 2>/dev/null || echo "0")

  if [ "$FIRST" = true ]; then
    FIRST=false
  else
    echo "," >> "$RESULTS_FILE"
  fi

  cat >> "$RESULTS_FILE" << ENTRY
  {
    "model": "$model",
    "prompt_type": "$prompt_type",
    "prompt_eval_count": $prompt_eval_count,
    "eval_count": $eval_count,
    "total_duration_sec": $total_sec,
    "time_to_first_token_sec": $ttft,
    "generation_tokens_per_sec": $tokens_per_sec,
    "prompt_tokens_per_sec": $prompt_tokens_per_sec,
    "vram_before_mib": "$vram_before",
    "vram_after_mib": "$vram_after",
    "timestamp": "$(date -Iseconds)"
  }
ENTRY

  echo "    -> ${tokens_per_sec} tok/s generation, ${prompt_tokens_per_sec} tok/s prompt, TTFT: ${ttft}s, Total: ${total_sec}s"
}

echo "=== LLM Inference Benchmark ==="
echo "Timestamp: $(date -Iseconds)"
echo "GPU: $(nvidia-smi --query-gpu=name --format=csv,noheader)"
echo ""

for model in "${MODELS[@]}"; do
  echo "Loading model: $model"
  # Warm up / ensure model is loaded
  curl -s "$OLLAMA_URL/api/generate" -d "{\"model\": \"$model\", \"prompt\": \"hello\", \"stream\": false, \"options\": {\"num_predict\": 1}}" > /dev/null 2>&1
  sleep 2

  benchmark_model "$model" "$SHORT_PROMPT" "short"
  benchmark_model "$model" "$MEDIUM_PROMPT" "medium"
  benchmark_model "$model" "$LONG_PROMPT" "long"
  echo ""
done

echo "" >> "$RESULTS_FILE"
echo "]}" >> "$RESULTS_FILE"

echo "=== Single-model benchmarks complete ==="
echo "Results saved to $RESULTS_FILE"
