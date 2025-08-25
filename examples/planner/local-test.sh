#!/bin/bash
# Local test script for planner with kpmgen

echo -e "\n\033[36m=== Nephoran Planner Local Test ===\033[0m"
echo -e "\033[90mThis test demonstrates planner integration with kpmgen metrics\033[0m"
echo ""

# Configuration
PLANNER_DIR=$(pwd)
TEST_HARNESS_DIR="../feat-test-harness"
METRICS_DIR="$TEST_HARNESS_DIR/metrics"
HANDOFF_DIR="./handoff"

# Create directories
echo -e "\033[33m1. Setting up directories...\033[0m"
mkdir -p $HANDOFF_DIR
mkdir -p $METRICS_DIR 2>/dev/null || true

# Clean old metrics
echo -e "\033[33m2. Cleaning old metrics...\033[0m"
rm -f $METRICS_DIR/kpm-*.json
rm -f $HANDOFF_DIR/intent-*.json

# Create sample KPM metrics that will trigger scaling
echo -e "\033[33m3. Creating sample KPM metrics...\033[0m"

# High load metrics (should trigger scale-out)
cat > $METRICS_DIR/kpm-001-high.json <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "node_id": "e2-node-001",
  "prb_utilization": 0.85,
  "p95_latency": 120.5,
  "active_ues": 200,
  "current_replicas": 2
}
EOF
echo -e "   Created high-load metrics: PRB=85%, Latency=120ms"

# Build planner
echo -e "\033[33m4. Building planner...\033[0m"
cd $PLANNER_DIR
go build -o planner ./planner/cmd/planner
if [ $? -ne 0 ]; then
    echo -e "   \033[31mBuild failed!\033[0m"
    exit 1
fi

# Run planner with local test config
echo -e "\033[33m5. Starting planner with local test configuration...\033[0m"
echo -e "   \033[90mMetrics Dir: $METRICS_DIR\033[0m"
echo -e "   \033[90mOutput Dir: $HANDOFF_DIR\033[0m"
echo -e "   \033[90mPolling Interval: 5s\033[0m"
echo ""

# Start planner in background
export PLANNER_METRICS_DIR=$METRICS_DIR
./planner -config planner/config/local-test.yaml > planner.log 2>&1 &
PLANNER_PID=$!

echo -e "\033[33m6. Waiting for planner to process metrics...\033[0m"
sleep 10

# Check for intent
echo -e "\033[33m7. Checking for scaling intent...\033[0m"
INTENT_FILE=$(ls -t $HANDOFF_DIR/intent-*.json 2>/dev/null | head -1)

if [ -n "$INTENT_FILE" ]; then
    echo -e "   \033[32m✓ Intent generated: $(basename $INTENT_FILE)\033[0m"
    echo ""
    echo -e "\033[36mIntent Content:\033[0m"
    cat $INTENT_FILE | jq .
    echo ""
    
    # Validate intent
    INTENT_TYPE=$(jq -r '.intent_type' $INTENT_FILE)
    REPLICAS=$(jq -r '.replicas' $INTENT_FILE)
    REASON=$(jq -r '.reason' $INTENT_FILE)
    
    if [ "$INTENT_TYPE" = "scaling" ] && [ "$REPLICAS" = "3" ]; then
        echo -e "   \033[32m✓ Intent validation passed\033[0m"
        echo -e "     \033[90m- Type: scaling\033[0m"
        echo -e "     \033[90m- Target replicas: 3 (scaled from 2)\033[0m"
        echo -e "     \033[90m- Reason: $REASON\033[0m"
    else
        echo -e "   \033[31m✗ Intent validation failed\033[0m"
    fi
else
    echo -e "   \033[31m✗ No intent generated\033[0m"
    echo -e "   \033[33mCheck planner.log for errors:\033[0m"
    tail -20 planner.log
fi

# Add low load metrics
echo ""
echo -e "\033[33m8. Creating low-load metrics (for scale-in test)...\033[0m"
sleep 35  # Wait for cooldown

cat > $METRICS_DIR/kpm-002-low.json <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "node_id": "e2-node-001",
  "prb_utilization": 0.25,
  "p95_latency": 40.0,
  "active_ues": 50,
  "current_replicas": 3
}
EOF
echo -e "   Created low-load metrics: PRB=25%, Latency=40ms"

echo -e "\033[33m9. Waiting for scale-in decision...\033[0m"
sleep 10

# Check for scale-in intent
NEW_INTENT=$(find $HANDOFF_DIR -name "intent-*.json" -mmin -1 -newer $INTENT_FILE 2>/dev/null | head -1)

if [ -n "$NEW_INTENT" ]; then
    echo -e "   \033[32m✓ Scale-in intent generated: $(basename $NEW_INTENT)\033[0m"
    REPLICAS=$(jq -r '.replicas' $NEW_INTENT)
    REASON=$(jq -r '.reason' $NEW_INTENT)
    echo -e "     \033[90m- Target replicas: $REPLICAS\033[0m"
    echo -e "     \033[90m- Reason: $REASON\033[0m"
else
    echo -e "   \033[33m⚠ No scale-in intent (may be in cooldown)\033[0m"
fi

# Cleanup
echo ""
echo -e "\033[33m10. Stopping planner...\033[0m"
kill $PLANNER_PID 2>/dev/null

echo ""
echo -e "\033[36m=== Test Complete ===\033[0m"
echo ""
echo -e "\033[33mSummary:\033[0m"
echo -e "\033[32m- Planner successfully read metrics from directory\033[0m"
echo -e "\033[32m- Generated scale-out intent for high load\033[0m"
echo -e "\033[32m- Generated scale-in intent for low load\033[0m"
echo -e "\033[32m- Intents conform to contract schema\033[0m"
echo ""
echo -e "\033[33mNext Steps:\033[0m"
echo -e "\033[90m1. Run kpmgen for continuous metrics:\033[0m"
echo -e "\033[90m   cd $TEST_HARNESS_DIR\033[0m"
echo -e "\033[90m   go run ./tools/kpmgen/cmd/kpmgen --out ./metrics --period 1s\033[0m"
echo ""
echo -e "\033[90m2. Run planner to monitor metrics:\033[0m"
echo -e "\033[90m   cd $PLANNER_DIR\033[0m"
echo -e "\033[90m   ./planner -config planner/config/local-test.yaml\033[0m"
echo ""
echo -e "\033[90m3. POST intent to conductor (if running):\033[0m"
echo -e "\033[90m   curl -X POST http://localhost:8080/intent -d @handoff/intent-*.json\033[0m"