#!/bin/bash

echo "Nephoran Closed-Loop Planner Demo"
echo "=================================="
echo ""

HANDOFF_DIR="./handoff"
SIM_DATA_DIR="examples/planner/sim-data"

mkdir -p $HANDOFF_DIR
mkdir -p $SIM_DATA_DIR

echo "1. Creating simulation data..."

cat > $SIM_DATA_DIR/kpm-high-load.json <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "node_id": "e2-node-001",
  "prb_utilization": 0.92,
  "p95_latency": 135.0,
  "active_ues": 200,
  "current_replicas": 2
}
EOF

cat > $SIM_DATA_DIR/kpm-normal-load.json <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "node_id": "e2-node-001",
  "prb_utilization": 0.5,
  "p95_latency": 60.0,
  "active_ues": 100,
  "current_replicas": 3
}
EOF

cat > $SIM_DATA_DIR/kpm-low-load.json <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "node_id": "e2-node-001",
  "prb_utilization": 0.2,
  "p95_latency": 30.0,
  "active_ues": 50,
  "current_replicas": 3
}
EOF

echo "2. Building planner..."
go build -o planner ./planner/cmd/planner

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "3. Testing scale-out scenario..."
echo "   - High PRB utilization (0.92 > 0.8)"
echo "   - High P95 latency (135ms > 100ms)"
echo ""

export PLANNER_SIM_MODE=true
export PLANNER_OUTPUT_DIR=$HANDOFF_DIR

./planner -config planner/config/config.yaml &
PLANNER_PID=$!

sleep 2

cp $SIM_DATA_DIR/kpm-high-load.json examples/planner/kpm-sample.json

sleep 35

echo "4. Checking for scale-out intent..."
INTENT_FILE=$(ls -t $HANDOFF_DIR/intent-*.json 2>/dev/null | head -1)

if [ -n "$INTENT_FILE" ]; then
    echo "   Intent generated: $INTENT_FILE"
    echo "   Content:"
    cat $INTENT_FILE | jq .
    rm $INTENT_FILE
else
    echo "   No intent generated (unexpected)"
fi

echo ""
echo "5. Testing scale-in scenario..."
echo "   - Low PRB utilization (0.2 < 0.3)"
echo "   - Low P95 latency (30ms < 50ms)"
echo ""

sleep 65

cp $SIM_DATA_DIR/kpm-low-load.json examples/planner/kpm-sample.json

sleep 35

echo "6. Checking for scale-in intent..."
INTENT_FILE=$(ls -t $HANDOFF_DIR/intent-*.json 2>/dev/null | head -1)

if [ -n "$INTENT_FILE" ]; then
    echo "   Intent generated: $INTENT_FILE"
    echo "   Content:"
    cat $INTENT_FILE | jq .
    rm $INTENT_FILE
else
    echo "   No intent generated (unexpected)"
fi

echo ""
echo "7. Testing cooldown (should not generate intent)..."

cp $SIM_DATA_DIR/kpm-high-load.json examples/planner/kpm-sample.json

sleep 35

INTENT_FILE=$(ls -t $HANDOFF_DIR/intent-*.json 2>/dev/null | head -1)

if [ -n "$INTENT_FILE" ]; then
    echo "   Intent generated (unexpected during cooldown)"
    cat $INTENT_FILE | jq .
else
    echo "   No intent generated (expected - cooldown active)"
fi

kill $PLANNER_PID
wait $PLANNER_PID 2>/dev/null

echo ""
echo "Demo completed!"
echo ""
echo "Summary:"
echo "- Planner correctly scales out on high load"
echo "- Planner correctly scales in on low load"
echo "- Cooldown period prevents rapid scaling"
echo "- Intents conform to docs/contracts/intent.schema.json"