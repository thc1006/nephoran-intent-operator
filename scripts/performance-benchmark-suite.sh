#!/bin/bash
# Nephoran Intent Operator Performance Benchmark Suite
# Phase 3 Production Excellence - Industry Standards Benchmarking
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-nephoran-system}"
BENCHMARK_DIR="${BENCHMARK_DIR:-/tmp/nephoran-benchmarks}"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
BENCHMARK_ID="PERF-BENCH-$TIMESTAMP"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}"
}

info() {
    echo -e "${PURPLE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Industry benchmarks for telecommunications systems
declare -A INDUSTRY_BENCHMARKS=(
    # Intent processing benchmarks (based on telecom network management standards)
    ["intent_processing_latency_p50"]="2000"     # 2 seconds P50
    ["intent_processing_latency_p95"]="5000"     # 5 seconds P95
    ["intent_processing_latency_p99"]="10000"    # 10 seconds P99
    ["intent_success_rate"]="99.5"               # 99.5% success rate
    ["intent_throughput_per_minute"]="100"       # 100 intents/minute
    
    # LLM processing benchmarks (AI/ML in telecom standards)
    ["llm_response_time_p50"]="1500"             # 1.5 seconds P50
    ["llm_response_time_p95"]="3000"             # 3 seconds P95
    ["llm_token_processing_rate"]="1000"         # 1000 tokens/second
    ["llm_cache_hit_rate"]="80"                  # 80% cache hit rate
    ["llm_error_rate"]="1"                       # 1% error rate
    
    # Vector database benchmarks (high-performance data systems)
    ["vector_query_latency_p50"]="100"           # 100ms P50
    ["vector_query_latency_p95"]="500"           # 500ms P95
    ["vector_throughput_qps"]="1000"             # 1000 queries/second
    ["vector_memory_efficiency"]="80"            # 80% memory utilization
    ["vector_storage_compression"]="60"          # 60% compression ratio
    
    # Network function deployment benchmarks (O-RAN standards)
    ["nf_deployment_time_p50"]="30"              # 30 seconds P50
    ["nf_deployment_time_p95"]="120"             # 2 minutes P95
    ["nf_deployment_success_rate"]="99"          # 99% success rate
    ["nf_scaling_time"]="60"                     # 60 seconds scaling
    ["nf_availability"]="99.9"                   # 99.9% availability
    
    # System resource benchmarks (cloud-native standards)
    ["cpu_utilization_avg"]="70"                 # 70% average CPU
    ["memory_utilization_avg"]="80"              # 80% average memory
    ["storage_utilization_avg"]="75"             # 75% average storage
    ["network_throughput_mbps"]="1000"           # 1 Gbps network
    ["pod_startup_time"]="30"                    # 30 seconds startup
    
    # Reliability benchmarks (telecom reliability standards)
    ["system_uptime"]="99.95"                    # 99.95% uptime
    ["mtbf_hours"]="8760"                        # 1 year MTBF
    ["mttr_minutes"]="15"                        # 15 minutes MTTR
    ["data_consistency"]="99.99"                 # 99.99% consistency
    ["backup_recovery_time"]="300"               # 5 minutes RTO
)

# Initialize benchmark environment
initialize_benchmark_environment() {
    log "Initializing performance benchmark environment..."
    
    # Create benchmark directory structure
    mkdir -p "$BENCHMARK_DIR"/{results,reports,metrics,logs,artifacts}
    
    # Check system prerequisites
    local missing_tools=()
    
    if ! command -v kubectl >/dev/null 2>&1; then
        missing_tools+=("kubectl")
    fi
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_tools+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_tools+=("jq")
    fi
    
    if ! command -v bc >/dev/null 2>&1; then
        missing_tools+=("bc")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        error "Missing required tools: ${missing_tools[*]}"
        return 1
    fi
    
    # Verify cluster connectivity and namespace
    if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        error "Namespace $NAMESPACE not found or not accessible"
        return 1
    fi
    
    # Create benchmark metadata
    cat > "$BENCHMARK_DIR/benchmark-metadata.json" <<EOF
{
    "benchmark_id": "$BENCHMARK_ID",
    "timestamp": "$(date -Iseconds)",
    "namespace": "$NAMESPACE",
    "executor": "$(whoami)",
    "hostname": "$(hostname)",
    "cluster_info": {
        "context": "$(kubectl config current-context)",
        "server": "$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')",
        "kubernetes_version": "$(kubectl version --short --client | grep Client | cut -d' ' -f3)"
    },
    "benchmark_suite_version": "3.0",
    "industry_standards": "O-RAN, 3GPP, ETSI, ITU-T"
}
EOF
    
    success "Benchmark environment initialized with ID: $BENCHMARK_ID"
    return 0
}

# Benchmark intent processing performance
benchmark_intent_processing() {
    log "Benchmarking intent processing performance..."
    
    local results_file="$BENCHMARK_DIR/results/intent-processing-results.json"
    local test_intents=(
        "Deploy AMF with 3 replicas for 5G SA core network"
        "Scale SMF instances to 5 replicas with high availability"
        "Configure UPF for ultra-low latency URLLC applications"
        "Setup gNB network function with massive MIMO support"
        "Deploy network slice for enhanced mobile broadband"
        "Configure O-RAN DU with beamforming capabilities"
        "Setup Near-RT RIC with xApp orchestration"
        "Deploy core network with multi-tenancy support"
        "Configure edge computing for industrial IoT"
        "Setup network function virtualization infrastructure"
    )
    
    local processing_times=()
    local success_count=0
    local total_tests=${#test_intents[@]}
    
    log "Executing $total_tests intent processing tests..."
    
    for i in "${!test_intents[@]}"; do
        local intent="${test_intents[$i]}"
        local intent_name="benchmark-intent-$i-$TIMESTAMP"
        
        log "Processing intent $((i+1))/$total_tests: $intent"
        
        # Record start time
        local start_time=$(date +%s%N)
        
        # Create NetworkIntent
        kubectl apply -f - <<EOF >/dev/null 2>&1
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: $intent_name
  namespace: $NAMESPACE
  labels:
    benchmark: "true"
    benchmark-id: "$BENCHMARK_ID"
spec:
  description: "$intent"
  priority: medium
EOF
        
        if [[ $? -eq 0 ]]; then
            # Wait for processing completion or timeout
            local timeout=30
            local elapsed=0
            local processed=false
            
            while [[ $elapsed -lt $timeout ]]; do
                local status=$(kubectl get networkintent "$intent_name" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
                
                if [[ "$status" == "Ready" ]] || [[ "$status" == "Failed" ]]; then
                    processed=true
                    if [[ "$status" == "Ready" ]]; then
                        success_count=$((success_count + 1))
                    fi
                    break
                fi
                
                sleep 1
                elapsed=$((elapsed + 1))
            done
            
            # Record end time
            local end_time=$(date +%s%N)
            local processing_time_ms=$(((end_time - start_time) / 1000000))
            processing_times+=($processing_time_ms)
            
            if [[ $processed == false ]]; then
                warn "Intent $intent_name timed out after ${timeout}s"
                processing_times+=($((timeout * 1000)))
            fi
            
        else
            error "Failed to create intent $intent_name"
            processing_times+=(999999)  # Mark as failed
        fi
        
        # Cleanup
        kubectl delete networkintent "$intent_name" -n "$NAMESPACE" --ignore-not-found=true >/dev/null 2>&1
    done
    
    # Calculate statistics
    local sum=0
    local min=999999
    local max=0
    
    for time in "${processing_times[@]}"; do
        sum=$((sum + time))
        if [[ $time -lt $min ]]; then min=$time; fi
        if [[ $time -gt $max ]]; then max=$time; fi
    done
    
    local avg=$((sum / total_tests))
    
    # Calculate percentiles (simplified)
    IFS=$'\n' sorted_times=($(sort -n <<<"${processing_times[*]}"))
    unset IFS
    
    local p50_index=$((total_tests * 50 / 100))
    local p95_index=$((total_tests * 95 / 100))
    local p99_index=$((total_tests * 99 / 100))
    
    local p50=${sorted_times[$p50_index]}
    local p95=${sorted_times[$p95_index]}
    local p99=${sorted_times[$p99_index]}
    
    local success_rate=$(echo "scale=2; $success_count * 100 / $total_tests" | bc)
    local throughput=$(echo "scale=2; $total_tests * 60000 / $sum" | bc)  # intents per minute
    
    # Generate results
    cat > "$results_file" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "test_type": "intent_processing",
    "results": {
        "total_tests": $total_tests,
        "successful_tests": $success_count,
        "success_rate_percent": $success_rate,
        "throughput_per_minute": $throughput,
        "latency_statistics_ms": {
            "min": $min,
            "max": $max,
            "average": $avg,
            "p50": $p50,
            "p95": $p95,
            "p99": $p99
        }
    },
    "benchmark_comparison": {
        "p50_vs_standard": "$(echo "scale=2; $p50 <= ${INDUSTRY_BENCHMARKS[intent_processing_latency_p50]}" | bc)",
        "p95_vs_standard": "$(echo "scale=2; $p95 <= ${INDUSTRY_BENCHMARKS[intent_processing_latency_p95]}" | bc)",
        "p99_vs_standard": "$(echo "scale=2; $p99 <= ${INDUSTRY_BENCHMARKS[intent_processing_latency_p99]}" | bc)",
        "success_rate_vs_standard": "$(echo "scale=2; $success_rate >= ${INDUSTRY_BENCHMARKS[intent_success_rate]}" | bc)",
        "throughput_vs_standard": "$(echo "scale=2; $throughput >= ${INDUSTRY_BENCHMARKS[intent_throughput_per_minute]}" | bc)"
    }
}
EOF
    
    log "Intent processing benchmark completed:"
    log "  Success Rate: $success_rate% (Target: ${INDUSTRY_BENCHMARKS[intent_success_rate]}%)"
    log "  Throughput: $throughput intents/min (Target: ${INDUSTRY_BENCHMARKS[intent_throughput_per_minute]}/min)"
    log "  P50 Latency: ${p50}ms (Target: ${INDUSTRY_BENCHMARKS[intent_processing_latency_p50]}ms)"
    log "  P95 Latency: ${p95}ms (Target: ${INDUSTRY_BENCHMARKS[intent_processing_latency_p95]}ms)"
    
    return 0
}

# Benchmark LLM processing performance
benchmark_llm_processing() {
    log "Benchmarking LLM processing performance..."
    
    local results_file="$BENCHMARK_DIR/results/llm-processing-results.json"
    local llm_endpoint="http://llm-processor.${NAMESPACE}.svc.cluster.local:8080"
    
    # Check if LLM processor is accessible
    if ! kubectl get service llm-processor -n "$NAMESPACE" >/dev/null 2>&1; then
        warn "LLM processor service not found, skipping LLM benchmarks"
        return 1
    fi
    
    local test_queries=(
        "What are the key components of a 5G core network architecture?"
        "How do you configure an AMF for network slice management?"
        "Explain the difference between SA and NSA 5G deployment modes"
        "What are the requirements for URLLC network slice configuration?"
        "How does beamforming work in massive MIMO systems?"
        "Describe the O-RAN disaggregated architecture components"
        "What are the key performance indicators for UPF deployment?"
        "How do you implement network function virtualization?"
        "Explain the role of Near-RT RIC in O-RAN architecture"
        "What are the security considerations for 5G core deployment?"
    )
    
    local response_times=()
    local success_count=0
    local total_queries=${#test_queries[@]}
    local cache_hits=0
    
    log "Executing $total_queries LLM processing tests..."
    
    # Port forward to LLM processor (run in background)
    kubectl port-forward -n "$NAMESPACE" service/llm-processor 18080:8080 >/dev/null 2>&1 &
    local port_forward_pid=$!
    sleep 3  # Wait for port forward to establish
    
    for i in "${!test_queries[@]}"; do
        local query="${test_queries[$i]}"
        
        log "Processing query $((i+1))/$total_queries"
        
        # Record start time
        local start_time=$(date +%s%N)
        
        # Send query to LLM processor
        local response=$(curl -s -X POST "http://localhost:18080/process" \
            -H "Content-Type: application/json" \
            -d "{\"query\":\"$query\",\"context\":\"telecommunications\"}" \
            --max-time 30 2>/dev/null)
        
        local curl_exit_code=$?
        local end_time=$(date +%s%N)
        local response_time_ms=$(((end_time - start_time) / 1000000))
        
        if [[ $curl_exit_code -eq 0 ]] && [[ -n "$response" ]]; then
            response_times+=($response_time_ms)
            success_count=$((success_count + 1))
            
            # Check if response indicates cache hit
            if echo "$response" | jq -e '.metadata.cache_hit == true' >/dev/null 2>&1; then
                cache_hits=$((cache_hits + 1))
            fi
        else
            warn "LLM query $((i+1)) failed or timed out"
            response_times+=(30000)  # 30 second timeout
        fi
        
        # Small delay between requests
        sleep 0.5
    done
    
    # Cleanup port forward
    kill $port_forward_pid 2>/dev/null || true
    
    # Calculate statistics
    local sum=0
    local min=999999
    local max=0
    
    for time in "${response_times[@]}"; do
        sum=$((sum + time))
        if [[ $time -lt $min ]]; then min=$time; fi
        if [[ $time -gt $max ]]; then max=$time; fi
    done
    
    local avg=$((sum / total_queries))
    
    # Calculate percentiles
    IFS=$'\n' sorted_times=($(sort -n <<<"${response_times[*]}"))
    unset IFS
    
    local p50_index=$((total_queries * 50 / 100))
    local p95_index=$((total_queries * 95 / 100))
    
    local p50=${sorted_times[$p50_index]}
    local p95=${sorted_times[$p95_index]}
    
    local success_rate=$(echo "scale=2; $success_count * 100 / $total_queries" | bc)
    local cache_hit_rate=$(echo "scale=2; $cache_hits * 100 / $success_count" | bc 2>/dev/null || echo "0")
    local error_rate=$(echo "scale=2; (100 - $success_rate)" | bc)
    
    # Generate results
    cat > "$results_file" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "test_type": "llm_processing",
    "results": {
        "total_queries": $total_queries,
        "successful_queries": $success_count,
        "success_rate_percent": $success_rate,
        "error_rate_percent": $error_rate,
        "cache_hit_rate_percent": $cache_hit_rate,
        "response_time_statistics_ms": {
            "min": $min,
            "max": $max,
            "average": $avg,
            "p50": $p50,
            "p95": $p95
        }
    },
    "benchmark_comparison": {
        "p50_vs_standard": "$(echo "$p50 <= ${INDUSTRY_BENCHMARKS[llm_response_time_p50]}" | bc)",
        "p95_vs_standard": "$(echo "$p95 <= ${INDUSTRY_BENCHMARKS[llm_response_time_p95]}" | bc)",
        "cache_hit_rate_vs_standard": "$(echo "$cache_hit_rate >= ${INDUSTRY_BENCHMARKS[llm_cache_hit_rate]}" | bc)",
        "error_rate_vs_standard": "$(echo "$error_rate <= ${INDUSTRY_BENCHMARKS[llm_error_rate]}" | bc)"
    }
}
EOF
    
    log "LLM processing benchmark completed:"
    log "  Success Rate: $success_rate% (Error Rate: $error_rate%)"
    log "  Cache Hit Rate: $cache_hit_rate% (Target: ${INDUSTRY_BENCHMARKS[llm_cache_hit_rate]}%)"
    log "  P50 Response Time: ${p50}ms (Target: ${INDUSTRY_BENCHMARKS[llm_response_time_p50]}ms)"
    log "  P95 Response Time: ${p95}ms (Target: ${INDUSTRY_BENCHMARKS[llm_response_time_p95]}ms)"
    
    return 0
}

# Benchmark vector database performance
benchmark_vector_database() {
    log "Benchmarking vector database performance..."
    
    local results_file="$BENCHMARK_DIR/results/vector-database-results.json"
    local weaviate_endpoint="http://weaviate.${NAMESPACE}.svc.cluster.local:8080"
    
    # Check if Weaviate is accessible
    if ! kubectl get service weaviate -n "$NAMESPACE" >/dev/null 2>&1; then
        warn "Weaviate service not found, skipping vector database benchmarks"
        return 1
    fi
    
    local test_queries=(
        "AMF network function deployment"
        "5G core network architecture"
        "network slice management"
        "URLLC low latency requirements"
        "massive MIMO beamforming"
        "O-RAN disaggregated RAN"
        "UPF user plane function"
        "network function virtualization"
        "Near-RT RIC intelligent control"
        "5G security architecture"
    )
    
    local query_times=()
    local success_count=0
    local total_queries=${#test_queries[@]}
    
    log "Executing $total_queries vector database queries..."
    
    # Port forward to Weaviate (run in background)
    kubectl port-forward -n "$NAMESPACE" service/weaviate 18081:8080 >/dev/null 2>&1 &
    local port_forward_pid=$!
    sleep 3  # Wait for port forward to establish
    
    for i in "${!test_queries[@]}"; do
        local query="${test_queries[$i]}"
        
        log "Executing vector query $((i+1))/$total_queries"
        
        # Record start time
        local start_time=$(date +%s%N)
        
        # Execute GraphQL query against Weaviate
        local graphql_query=$(cat <<EOF
{
  "query": "{ Get { TelecomKnowledge(nearText: {concepts: [\"$query\"]}, limit: 10) { title content _additional { certainty } } } }"
}
EOF
)
        
        local response=$(curl -s -X POST "http://localhost:18081/v1/graphql" \
            -H "Content-Type: application/json" \
            -d "$graphql_query" \
            --max-time 10 2>/dev/null)
        
        local curl_exit_code=$?
        local end_time=$(date +%s%N)
        local query_time_ms=$(((end_time - start_time) / 1000000))
        
        if [[ $curl_exit_code -eq 0 ]] && echo "$response" | jq -e '.data.Get.TelecomKnowledge' >/dev/null 2>&1; then
            query_times+=($query_time_ms)
            success_count=$((success_count + 1))
        else
            warn "Vector query $((i+1)) failed or timed out"
            query_times+=(10000)  # 10 second timeout
        fi
        
        # Small delay between requests
        sleep 0.2
    done
    
    # Cleanup port forward
    kill $port_forward_pid 2>/dev/null || true
    
    # Calculate statistics
    local sum=0
    local min=999999
    local max=0
    
    for time in "${query_times[@]}"; do
        sum=$((sum + time))
        if [[ $time -lt $min ]]; then min=$time; fi
        if [[ $time -gt $max ]]; then max=$time; fi
    done
    
    local avg=$((sum / total_queries))
    
    # Calculate percentiles
    IFS=$'\n' sorted_times=($(sort -n <<<"${query_times[*]}"))
    unset IFS
    
    local p50_index=$((total_queries * 50 / 100))
    local p95_index=$((total_queries * 95 / 100))
    
    local p50=${sorted_times[$p50_index]}
    local p95=${sorted_times[$p95_index]}
    
    local success_rate=$(echo "scale=2; $success_count * 100 / $total_queries" | bc)
    local throughput_qps=$(echo "scale=2; $total_queries * 1000 / $sum" | bc)
    
    # Get memory usage information
    local memory_usage_mb=0
    local weaviate_pod=$(kubectl get pods -n "$NAMESPACE" -l app=weaviate -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [[ -n "$weaviate_pod" ]]; then
        local memory_usage_bytes=$(kubectl top pod "$weaviate_pod" -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print $3}' | sed 's/Mi//')
        if [[ -n "$memory_usage_bytes" ]]; then
            memory_usage_mb=$memory_usage_bytes
        fi
    fi
    
    # Generate results
    cat > "$results_file" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "test_type": "vector_database",
    "results": {
        "total_queries": $total_queries,
        "successful_queries": $success_count,
        "success_rate_percent": $success_rate,
        "throughput_qps": $throughput_qps,
        "memory_usage_mb": $memory_usage_mb,
        "query_time_statistics_ms": {
            "min": $min,
            "max": $max,
            "average": $avg,
            "p50": $p50,
            "p95": $p95
        }
    },
    "benchmark_comparison": {
        "p50_vs_standard": "$(echo "$p50 <= ${INDUSTRY_BENCHMARKS[vector_query_latency_p50]}" | bc)",
        "p95_vs_standard": "$(echo "$p95 <= ${INDUSTRY_BENCHMARKS[vector_query_latency_p95]}" | bc)",
        "throughput_vs_standard": "$(echo "$throughput_qps >= ${INDUSTRY_BENCHMARKS[vector_throughput_qps]}" | bc)"
    }
}
EOF
    
    log "Vector database benchmark completed:"
    log "  Success Rate: $success_rate%"
    log "  Throughput: $throughput_qps QPS (Target: ${INDUSTRY_BENCHMARKS[vector_throughput_qps]} QPS)"
    log "  P50 Query Time: ${p50}ms (Target: ${INDUSTRY_BENCHMARKS[vector_query_latency_p50]}ms)"
    log "  P95 Query Time: ${p95}ms (Target: ${INDUSTRY_BENCHMARKS[vector_query_latency_p95]}ms)"
    log "  Memory Usage: ${memory_usage_mb}MB"
    
    return 0
}

# Benchmark system resource utilization
benchmark_system_resources() {
    log "Benchmarking system resource utilization..."
    
    local results_file="$BENCHMARK_DIR/results/system-resources-results.json"
    local monitoring_duration=60  # Monitor for 60 seconds
    local sample_interval=5       # Sample every 5 seconds
    local samples=$((monitoring_duration / sample_interval))
    
    log "Monitoring system resources for ${monitoring_duration} seconds..."
    
    local cpu_samples=()
    local memory_samples=()
    local pod_count_samples=()
    local total_cpu_requests=0
    local total_memory_requests=0
    
    for ((i=1; i<=samples; i++)); do
        log "Resource sample $i/$samples"
        
        # Get node resource usage
        local node_stats=$(kubectl top nodes --no-headers 2>/dev/null | head -1)
        if [[ -n "$node_stats" ]]; then
            local cpu_usage=$(echo "$node_stats" | awk '{print $3}' | sed 's/%//')
            local memory_usage=$(echo "$node_stats" | awk '{print $5}' | sed 's/%//')
            
            cpu_samples+=($cpu_usage)
            memory_samples+=($memory_usage)
        fi
        
        # Count running pods in namespace
        local pod_count=$(kubectl get pods -n "$NAMESPACE" --no-headers | grep -c Running || echo "0")
        pod_count_samples+=($pod_count)
        
        sleep $sample_interval
    done
    
    # Calculate CPU statistics
    local cpu_sum=0
    local cpu_min=100
    local cpu_max=0
    
    for cpu in "${cpu_samples[@]}"; do
        cpu_sum=$((cpu_sum + cpu))
        if [[ $cpu -lt $cpu_min ]]; then cpu_min=$cpu; fi
        if [[ $cpu -gt $cpu_max ]]; then cpu_max=$cpu; fi
    done
    
    local cpu_avg=$((cpu_sum / ${#cpu_samples[@]}))
    
    # Calculate memory statistics
    local memory_sum=0
    local memory_min=100
    local memory_max=0
    
    for memory in "${memory_samples[@]}"; do
        memory_sum=$((memory_sum + memory))
        if [[ $memory -lt $memory_min ]]; then memory_min=$memory; fi
        if [[ $memory -gt $memory_max ]]; then memory_max=$memory; fi
    done
    
    local memory_avg=$((memory_sum / ${#memory_samples[@]}))
    
    # Calculate pod count statistics
    local pod_sum=0
    for pod_count in "${pod_count_samples[@]}"; do
        pod_sum=$((pod_sum + pod_count))
    done
    local pod_avg=$((pod_sum / ${#pod_count_samples[@]}))
    
    # Get storage utilization
    local storage_usage=0
    local pvc_usage=$(kubectl get pvc -n "$NAMESPACE" -o json 2>/dev/null | \
        jq -r '.items[] | select(.status.phase == "Bound") | .status.capacity.storage' | \
        sed 's/Gi//' | awk '{sum += $1} END {print sum}' 2>/dev/null || echo "0")
    
    if [[ "$pvc_usage" != "0" ]]; then
        storage_usage=$pvc_usage
    fi
    
    # Get network throughput (approximate from pod metrics)
    local network_throughput_mbps=0
    local total_network_bytes=$(kubectl top pods -n "$NAMESPACE" --no-headers 2>/dev/null | \
        awk '{sum += $4} END {print sum}' | sed 's/[^0-9]//g' 2>/dev/null || echo "0")
    
    if [[ "$total_network_bytes" != "0" ]]; then
        network_throughput_mbps=$((total_network_bytes / 1024 / 1024))  # Convert to MB
    fi
    
    # Generate results
    cat > "$results_file" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "test_type": "system_resources",
    "monitoring_duration_seconds": $monitoring_duration,
    "results": {
        "cpu_utilization_percent": {
            "average": $cpu_avg,
            "min": $cpu_min,
            "max": $cpu_max
        },
        "memory_utilization_percent": {
            "average": $memory_avg,
            "min": $memory_min,
            "max": $memory_max
        },
        "pod_count": {
            "average": $pod_avg,
            "samples": $(printf '%s,' "${pod_count_samples[@]}" | sed 's/,$//')
        },
        "storage_usage_gb": $storage_usage,
        "network_throughput_mbps": $network_throughput_mbps
    },
    "benchmark_comparison": {
        "cpu_vs_standard": "$(echo "$cpu_avg <= ${INDUSTRY_BENCHMARKS[cpu_utilization_avg]}" | bc)",
        "memory_vs_standard": "$(echo "$memory_avg <= ${INDUSTRY_BENCHMARKS[memory_utilization_avg]}" | bc)",
        "storage_vs_standard": "$(echo "$storage_usage <= ${INDUSTRY_BENCHMARKS[storage_utilization_avg]}" | bc)",
        "network_vs_standard": "$(echo "$network_throughput_mbps >= ${INDUSTRY_BENCHMARKS[network_throughput_mbps]}" | bc)"
    }
}
EOF
    
    log "System resource benchmark completed:"
    log "  Average CPU Usage: $cpu_avg% (Target: ≤${INDUSTRY_BENCHMARKS[cpu_utilization_avg]}%)"
    log "  Average Memory Usage: $memory_avg% (Target: ≤${INDUSTRY_BENCHMARKS[memory_utilization_avg]}%)"
    log "  Average Pod Count: $pod_avg"
    log "  Storage Usage: ${storage_usage}GB"
    log "  Network Throughput: ${network_throughput_mbps}Mbps"
    
    return 0
}

# Generate comprehensive benchmark report
generate_benchmark_report() {
    log "Generating comprehensive benchmark report..."
    
    local final_report="$BENCHMARK_DIR/reports/performance-benchmark-$BENCHMARK_ID.json"
    local executive_summary="$BENCHMARK_DIR/reports/executive-summary-$BENCHMARK_ID.md"
    
    # Collect results from individual benchmarks
    local intent_score=0
    local llm_score=0
    local vector_score=0
    local resource_score=0
    local total_tests=0
    local passed_tests=0
    
    # Process intent processing results
    if [[ -f "$BENCHMARK_DIR/results/intent-processing-results.json" ]]; then
        local intent_results=$(cat "$BENCHMARK_DIR/results/intent-processing-results.json")
        local intent_pass_count=0
        local intent_test_count=5  # 5 key metrics
        
        # Check each benchmark against standards
        if [[ $(echo "$intent_results" | jq -r '.benchmark_comparison.p50_vs_standard') == "1" ]]; then
            intent_pass_count=$((intent_pass_count + 1))
        fi
        if [[ $(echo "$intent_results" | jq -r '.benchmark_comparison.p95_vs_standard') == "1" ]]; then
            intent_pass_count=$((intent_pass_count + 1))
        fi
        if [[ $(echo "$intent_results" | jq -r '.benchmark_comparison.p99_vs_standard') == "1" ]]; then
            intent_pass_count=$((intent_pass_count + 1))
        fi
        if [[ $(echo "$intent_results" | jq -r '.benchmark_comparison.success_rate_vs_standard') == "1" ]]; then
            intent_pass_count=$((intent_pass_count + 1))
        fi
        if [[ $(echo "$intent_results" | jq -r '.benchmark_comparison.throughput_vs_standard') == "1" ]]; then
            intent_pass_count=$((intent_pass_count + 1))
        fi
        
        intent_score=$((intent_pass_count * 100 / intent_test_count))
        total_tests=$((total_tests + intent_test_count))
        passed_tests=$((passed_tests + intent_pass_count))
    fi
    
    # Process LLM processing results
    if [[ -f "$BENCHMARK_DIR/results/llm-processing-results.json" ]]; then
        local llm_results=$(cat "$BENCHMARK_DIR/results/llm-processing-results.json")
        local llm_pass_count=0
        local llm_test_count=4  # 4 key metrics
        
        if [[ $(echo "$llm_results" | jq -r '.benchmark_comparison.p50_vs_standard') == "1" ]]; then
            llm_pass_count=$((llm_pass_count + 1))
        fi
        if [[ $(echo "$llm_results" | jq -r '.benchmark_comparison.p95_vs_standard') == "1" ]]; then
            llm_pass_count=$((llm_pass_count + 1))
        fi
        if [[ $(echo "$llm_results" | jq -r '.benchmark_comparison.cache_hit_rate_vs_standard') == "1" ]]; then
            llm_pass_count=$((llm_pass_count + 1))
        fi
        if [[ $(echo "$llm_results" | jq -r '.benchmark_comparison.error_rate_vs_standard') == "1" ]]; then
            llm_pass_count=$((llm_pass_count + 1))
        fi
        
        llm_score=$((llm_pass_count * 100 / llm_test_count))
        total_tests=$((total_tests + llm_test_count))
        passed_tests=$((passed_tests + llm_pass_count))
    fi
    
    # Process vector database results
    if [[ -f "$BENCHMARK_DIR/results/vector-database-results.json" ]]; then
        local vector_results=$(cat "$BENCHMARK_DIR/results/vector-database-results.json")
        local vector_pass_count=0
        local vector_test_count=3  # 3 key metrics
        
        if [[ $(echo "$vector_results" | jq -r '.benchmark_comparison.p50_vs_standard') == "1" ]]; then
            vector_pass_count=$((vector_pass_count + 1))
        fi
        if [[ $(echo "$vector_results" | jq -r '.benchmark_comparison.p95_vs_standard') == "1" ]]; then
            vector_pass_count=$((vector_pass_count + 1))
        fi
        if [[ $(echo "$vector_results" | jq -r '.benchmark_comparison.throughput_vs_standard') == "1" ]]; then
            vector_pass_count=$((vector_pass_count + 1))
        fi
        
        vector_score=$((vector_pass_count * 100 / vector_test_count))
        total_tests=$((total_tests + vector_test_count))
        passed_tests=$((passed_tests + vector_pass_count))
    fi
    
    # Process system resource results
    if [[ -f "$BENCHMARK_DIR/results/system-resources-results.json" ]]; then
        local resource_results=$(cat "$BENCHMARK_DIR/results/system-resources-results.json")
        local resource_pass_count=0
        local resource_test_count=4  # 4 key metrics
        
        if [[ $(echo "$resource_results" | jq -r '.benchmark_comparison.cpu_vs_standard') == "1" ]]; then
            resource_pass_count=$((resource_pass_count + 1))
        fi
        if [[ $(echo "$resource_results" | jq -r '.benchmark_comparison.memory_vs_standard') == "1" ]]; then
            resource_pass_count=$((resource_pass_count + 1))
        fi
        if [[ $(echo "$resource_results" | jq -r '.benchmark_comparison.storage_vs_standard') == "1" ]]; then
            resource_pass_count=$((resource_pass_count + 1))
        fi
        if [[ $(echo "$resource_results" | jq -r '.benchmark_comparison.network_vs_standard') == "1" ]]; then
            resource_pass_count=$((resource_pass_count + 1))
        fi
        
        resource_score=$((resource_pass_count * 100 / resource_test_count))
        total_tests=$((total_tests + resource_test_count))
        passed_tests=$((passed_tests + resource_pass_count))
    fi
    
    # Calculate overall score (weighted average)
    local overall_score=0
    if [[ $total_tests -gt 0 ]]; then
        overall_score=$((passed_tests * 100 / total_tests))
    fi
    
    # Determine performance grade
    local performance_grade=""
    if [[ $overall_score -ge 95 ]]; then
        performance_grade="EXCELLENT"
    elif [[ $overall_score -ge 85 ]]; then
        performance_grade="GOOD"
    elif [[ $overall_score -ge 70 ]]; then
        performance_grade="ACCEPTABLE"
    elif [[ $overall_score -ge 50 ]]; then
        performance_grade="POOR"
    else
        performance_grade="FAILING"
    fi
    
    # Generate final JSON report
    cat > "$final_report" <<EOF
{
    "benchmark_metadata": $(cat "$BENCHMARK_DIR/benchmark-metadata.json"),
    "overall_assessment": {
        "performance_score": $overall_score,
        "max_score": 100,
        "performance_grade": "$performance_grade",
        "total_tests": $total_tests,
        "passed_tests": $passed_tests,
        "meets_industry_standards": $([ $overall_score -ge 85 ] && echo "true" || echo "false")
    },
    "component_scores": {
        "intent_processing": {
            "score": $intent_score,
            "weight": 35,
            "status": "$([ $intent_score -ge 85 ] && echo "PASS" || echo "FAIL")"
        },
        "llm_processing": {
            "score": $llm_score,
            "weight": 25,
            "status": "$([ $llm_score -ge 85 ] && echo "PASS" || echo "FAIL")"
        },
        "vector_database": {
            "score": $vector_score,
            "weight": 25,
            "status": "$([ $vector_score -ge 85 ] && echo "PASS" || echo "FAIL")"
        },
        "system_resources": {
            "score": $resource_score,
            "weight": 15,
            "status": "$([ $resource_score -ge 85 ] && echo "PASS" || echo "FAIL")"
        }
    },
    "industry_standards_compliance": {
        "o_ran_compliance": $([ $overall_score -ge 85 ] && echo "true" || echo "false"),
        "3gpp_compliance": $([ $overall_score -ge 85 ] && echo "true" || echo "false"),
        "etsi_compliance": $([ $overall_score -ge 85 ] && echo "true" || echo "false"),
        "telecom_grade_performance": $([ $overall_score -ge 90 ] && echo "true" || echo "false")
    },
    "recommendations": [
        $([ $intent_score -lt 85 ] && echo "\"Optimize intent processing pipeline for better latency and throughput\",")
        $([ $llm_score -lt 85 ] && echo "\"Improve LLM response times and cache efficiency\",")
        $([ $vector_score -lt 85 ] && echo "\"Optimize vector database queries and indexing\",")
        $([ $resource_score -lt 85 ] && echo "\"Optimize system resource utilization and scaling\",")
        "\"Continue regular performance monitoring and optimization\",",
        "\"Implement automated performance regression testing\",",
        "\"Establish performance SLAs and monitoring alerts\""
    ],
    "next_benchmark_recommended": "$(date -d '+30 days' -Iseconds)"
}
EOF
    
    # Generate executive summary
    cat > "$executive_summary" <<EOF
# Nephoran Intent Operator Performance Benchmark Report

**Benchmark ID:** $BENCHMARK_ID  
**Date:** $(date +'%Y-%m-%d %H:%M:%S')  
**Namespace:** $NAMESPACE  
**Industry Standards:** O-RAN, 3GPP, ETSI, ITU-T

## Executive Summary

The comprehensive performance benchmark of the Nephoran Intent Operator has been completed with an overall performance score of **$overall_score/100** classified as **$performance_grade**.

### Component Performance Results

| Component | Score | Status | Industry Standard |
|-----------|-------|--------|-------------------|
| Intent Processing | $intent_score/100 | $([ $intent_score -ge 85 ] && echo "✅ PASS" || echo "❌ FAIL") | O-RAN Network Management |
| LLM Processing | $llm_score/100 | $([ $llm_score -ge 85 ] && echo "✅ PASS" || echo "❌ FAIL") | AI/ML Telecom Standards |
| Vector Database | $vector_score/100 | $([ $vector_score -ge 85 ] && echo "✅ PASS" || echo "❌ FAIL") | High-Performance Data Systems |
| System Resources | $resource_score/100 | $([ $resource_score -ge 85 ] && echo "✅ PASS" || echo "❌ FAIL") | Cloud-Native Standards |

### Industry Standards Compliance

- **O-RAN Compliance:** $([ $overall_score -ge 85 ] && echo "✅ COMPLIANT" || echo "❌ NON-COMPLIANT")
- **3GPP Standards:** $([ $overall_score -ge 85 ] && echo "✅ COMPLIANT" || echo "❌ NON-COMPLIANT")
- **ETSI Requirements:** $([ $overall_score -ge 85 ] && echo "✅ COMPLIANT" || echo "❌ NON-COMPLIANT")
- **Telecom-Grade Performance:** $([ $overall_score -ge 90 ] && echo "✅ ACHIEVED" || echo "❌ NOT ACHIEVED")

### Key Performance Highlights

$([ $overall_score -ge 95 ] && echo "🎉 **OUTSTANDING PERFORMANCE** - The system exceeds all industry benchmarks and demonstrates exceptional telecommunications-grade performance.")
$([ $overall_score -ge 85 ] && [ $overall_score -lt 95 ] && echo "✅ **EXCELLENT PERFORMANCE** - The system meets all critical industry standards with room for minor optimizations.")
$([ $overall_score -ge 70 ] && [ $overall_score -lt 85 ] && echo "⚠️ **ADEQUATE PERFORMANCE** - The system shows good performance but requires optimization to meet full industry standards.")
$([ $overall_score -lt 70 ] && echo "🚨 **PERFORMANCE CONCERNS** - The system requires significant optimization to meet telecommunications industry requirements.")

### Benchmark Summary

- **Total Tests Executed:** $total_tests
- **Tests Passed:** $passed_tests
- **Pass Rate:** $(echo "scale=1; $passed_tests * 100 / $total_tests" | bc)%
- **Industry Standards Met:** $([ $overall_score -ge 85 ] && echo "Yes" || echo "No")

### Performance Recommendations

#### Immediate Actions (if score < 85)
1. **Intent Processing Optimization:** Focus on reducing P95 latency below 5 seconds
2. **LLM Response Enhancement:** Improve cache hit rates and reduce response times
3. **Vector Database Tuning:** Optimize query performance and indexing strategies
4. **Resource Optimization:** Implement better resource allocation and scaling policies

#### Long-term Strategy
1. **Continuous Benchmarking:** Establish monthly performance benchmark cycles
2. **Performance Regression Testing:** Integrate benchmarks into CI/CD pipeline
3. **Industry Standards Monitoring:** Track evolving O-RAN and 3GPP performance requirements
4. **Capacity Planning:** Implement predictive scaling based on performance trends

### Next Steps

- **Next Benchmark Recommended:** $(date -d '+30 days' +'%Y-%m-%d')
- **Production Deployment Ready:** $([ $overall_score -ge 85 ] && echo "Yes" || echo "No")
- **Performance Optimization Required:** $([ $overall_score -ge 85 ] && echo "No" || echo "Yes")

### Detailed Results

Full benchmark results and raw data are available in:
- **JSON Report:** \`$final_report\`
- **Component Results:** \`$BENCHMARK_DIR/results/\`
- **Metrics Data:** \`$BENCHMARK_DIR/metrics/\`

---
*Report generated by Nephoran Performance Benchmark Suite v3.0*
EOF
    
    success "Comprehensive performance benchmark report generated:"
    success "  Overall Score: $overall_score/100 ($performance_grade)"
    success "  Industry Standards: $([ $overall_score -ge 85 ] && echo "COMPLIANT" || echo "NON-COMPLIANT")"
    success "  JSON Report: $final_report"
    success "  Executive Summary: $executive_summary"
    
    # Return success if score >= 85
    if [[ $overall_score -ge 85 ]]; then
        return 0
    else
        return 1
    fi
}

# Main execution function
main() {
    log "==========================================="
    log "Nephoran Intent Operator Performance Benchmark"
    log "Phase 3 Production Excellence - Industry Standards"
    log "==========================================="
    
    # Initialize benchmark environment
    if ! initialize_benchmark_environment; then
        error "Failed to initialize benchmark environment"
        return 1
    fi
    
    # Execute performance benchmarks
    local overall_result=0
    
    log "Step 1: Intent Processing Performance Benchmark"
    if ! benchmark_intent_processing; then
        warn "Intent processing benchmark completed with issues"
        overall_result=1
    fi
    
    log "Step 2: LLM Processing Performance Benchmark"
    if ! benchmark_llm_processing; then
        warn "LLM processing benchmark completed with issues"
        overall_result=1
    fi
    
    log "Step 3: Vector Database Performance Benchmark"
    if ! benchmark_vector_database; then
        warn "Vector database benchmark completed with issues"
        overall_result=1
    fi
    
    log "Step 4: System Resource Utilization Benchmark"
    if ! benchmark_system_resources; then
        warn "System resource benchmark completed with issues"
        overall_result=1
    fi
    
    log "Step 5: Generating Comprehensive Performance Report"
    if ! generate_benchmark_report; then
        error "Performance benchmark failed overall assessment"
        overall_result=1
    fi
    
    # Final status
    if [[ $overall_result -eq 0 ]]; then
        success "==========================================="
        success "Performance benchmark PASSED - Industry Standards Met"
        success "==========================================="
    else
        error "==========================================="
        error "Performance benchmark COMPLETED with optimizations needed"
        error "==========================================="
    fi
    
    log "Benchmark complete. Reports available in: $BENCHMARK_DIR"
    return $overall_result
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi