# Test Harness Complete Workflow

This guide demonstrates how to use the Nephoran test harness tools (`kpmgen` and `vessend`) together to create comprehensive end-to-end testing scenarios for the O-RAN Intent Operator. The workflow covers the complete cycle from KPM metric generation through VES event transmission to intent-based scaling decisions.

## Architecture Overview

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   kpmgen    │────│   Planner   │────│ NetworkIntent│────│   Nephio    │
│ (E2 Metrics)│    │ (Scaling    │    │   (CRD)     │    │  (Package   │
│             │    │  Logic)     │    │             │    │   Deploy)   │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                           │
                           ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   vessend   │────│ VES Collector│────│  FCAPS      │
│ (VES Events)│    │ (O1 Interface│    │ (Monitoring)│
│             │    │  Endpoint)   │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
```

## Prerequisites

### 1. Build Test Tools

```bash
# Build kpmgen
cd tools/kmpgen/cmd/kmpgen
go build -o kpmgen.exe .

# Build vessend  
cd ../../../vessend/cmd/vessend
go build -o vessend.exe .

# Verify builds
./kmpgen.exe -h
./vessend.exe -h
```

### 2. Environment Setup

```bash
# Create test directories
mkdir -p test-harness/{kpm-data,ves-events,results}
cd test-harness

# Set up VES collector (local testing)
docker run -d \
  --name ves-collector \
  -p 9990:9990 \
  onap/org.onap.dcaegen2.collectors.ves.vescollector:latest

# Verify VES collector
curl http://localhost:9990/eventListener/v7/health
```

### 3. Kubernetes Environment

```bash
# Ensure Nephoran is deployed
kubectl get pods -n nephoran-system

# Expected components:
# - conductor-loop
# - llm-processor
# - nephio-bridge
# - porch-publisher
# - planner

# Verify CRDs are installed
kubectl get crd | grep nephoran
```

## Complete End-to-End Workflow

### Phase 1: Baseline Measurement Generation

Generate baseline KPM measurements representing normal network operation.

```bash
# Generate low-load baseline metrics
../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
  -profile low \
  -burst 20 \
  -period 30s \
  -out ./kmp-data/baseline

# Monitor generation
ls -la ./kmp-data/baseline/
# Expected: 20 JSON files with timestamps
```

**Sample Baseline Output:**
```json
{
  "kmp.p95_latency_ms": 28.4,
  "kmp.prb_utilization": 0.22,
  "kmp.ue_count": 143,
  "timestamp": "2025-08-14T14:30:00Z",
  "window_id": "window-1"
}
```

### Phase 2: VES Event Generation

Send corresponding VES measurement events to simulate O1 interface reporting.

```bash
# Send baseline measurement events
../../tools/vessend/cmd/vessend/vessend.exe \
  -collector "http://localhost:9990/eventListener/v7" \
  -event-type measurement \
  -source "e2-node-baseline" \
  -count 20 \
  -interval 30s \
  > ./ves-events/baseline.log 2>&1 &

# Monitor VES event transmission
tail -f ./ves-events/baseline.log
```

### Phase 3: Load Scenario Testing

#### Scenario A: High Load (Scale-Up Trigger)

**Generate High Load KPM Metrics:**
```bash
# Simulate network congestion
../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
  -profile high \
  -burst 15 \
  -period 30s \
  -out ./kmp-data/high-load

# Verify high metrics
head -1 ./kmp-data/high-load/*.json | grep "prb_utilization"
# Expected: values > 0.8
```

**Send Corresponding VES Events:**
```bash
# Send high-load measurement events  
../../tools/vessend/cmd/vessend/vessend.exe \
  -collector "http://localhost:9990/eventListener/v7" \
  -event-type measurement \
  -source "e2-node-high-load" \
  -count 15 \
  -interval 30s \
  > ./ves-events/high-load.log 2>&1 &
```

**Expected Planner Behavior:**
```bash
# Monitor planner logs for scale-up decisions
kubectl logs -n nephoran-system deployment/planner -f | \
  grep -E "(scaling|high.*utilization|scale.*up)"

# Check for generated NetworkIntents
kubectl get networkintents -n nephoran-system
kubectl describe networkintent <intent-name> -n nephoran-system
```

#### Scenario B: Fault Injection

**Inject Critical Faults:**
```bash
# Generate fault events during high load
../../tools/vessend/cmd/vessend/vessend.exe \
  -collector "http://localhost:9990/eventListener/v7" \
  -event-type fault \
  -source "e2-node-fault-test" \
  -count 5 \
  -interval 60s \
  > ./ves-events/fault-injection.log 2>&1 &
```

**Expected FCAPS Response:**
```bash
# Monitor for alarm processing
kubectl logs -n nephoran-system deployment/fcaps-processor -f

# Check Prometheus alerts
curl http://localhost:9093/api/v1/alerts | \
  jq '.data[] | select(.labels.source=="e2-node-fault-test")'
```

### Phase 4: Scale-Down Scenario

**Generate Low Load Metrics:**
```bash
# Simulate low network utilization
../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
  -profile low \
  -burst 10 \
  -period 45s \
  -out ./kmp-data/low-load
```

**Send Low Load VES Events:**
```bash
../../tools/vessend/cmd/vessend/vessend.exe \
  -collector "http://localhost:9990/eventListener/v7" \
  -event-type measurement \
  -source "e2-node-low-load" \
  -count 10 \
  -interval 45s \
  > ./ves-events/low-load.log 2>&1 &
```

**Expected Scale-Down Behavior:**
```bash
# Monitor for scale-down decisions
kubectl logs -n nephoran-system deployment/planner -f | \
  grep -E "(scaling|low.*utilization|scale.*down)"

# Verify NetworkIntent with scale-down action
kubectl get networkintents -n nephoran-system -o yaml | \
  grep -A 5 -B 5 "scale.*down"
```

### Phase 5: Verification and Results Collection

#### A. Metrics Verification

**Validate Generated KPM Data:**
```bash
# Check all generated KPM files
find ./kmp-data -name "*.json" -exec echo "=== {} ===" \; -exec cat {} \; | \
  jq '.["kmp.prb_utilization"], .["kmp.p95_latency_ms"], .["kmp.ue_count"]'

# Verify profile compliance
echo "High load metrics (PRB utilization should be > 0.8):"
cat ./kmp-data/high-load/*.json | jq '.["kmp.prb_utilization"]'

echo "Low load metrics (PRB utilization should be < 0.3):"
cat ./kmp-data/low-load/*.json | jq '.["kmp.prb_utilization"]'
```

**Validate VES Event Delivery:**
```bash
# Check VES event delivery success rates
echo "Baseline events:"
grep "Event sent successfully" ./ves-events/baseline.log | wc -l

echo "High load events:"
grep "Event sent successfully" ./ves-events/high-load.log | wc -l

echo "Low load events:"
grep "Event sent successfully" ./ves-events/low-load.log | wc -l

echo "Fault events:"
grep "Event sent successfully" ./ves-events/fault-injection.log | wc -l
```

#### B. System Response Verification

**Check NetworkIntent Generation:**
```bash
# List all generated NetworkIntents with timestamps
kubectl get networkintents -n nephoran-system \
  -o custom-columns=NAME:.metadata.name,CREATED:.metadata.creationTimestamp,STATUS:.status.phase

# Verify scaling decisions match load patterns
kubectl get networkintents -n nephoran-system -o yaml | \
  grep -A 10 -B 5 "spec:" | grep -E "(replicas|scale)"
```

**Verify Nephio Package Generation:**
```bash
# Check if packages were generated and published
kubectl get packagerevisions -A | grep nephoran

# Verify package contents
kubectl describe packagerevision <package-name> -n <namespace>
```

#### C. End-to-End Latency Measurement

**Measure Intent Processing Latency:**
```bash
#!/bin/bash
# measure-e2e-latency.sh

echo "Starting end-to-end latency measurement..."

# Record start time
START_TIME=$(date +%s)

# Generate high-load trigger
../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
  -profile high -burst 1 -period 1s -out ./kmp-data/latency-test

# Wait for NetworkIntent generation
while true; do
  INTENT_COUNT=$(kubectl get networkintents -n nephoran-system --no-headers | wc -l)
  if [ $INTENT_COUNT -gt 0 ]; then
    END_TIME=$(date +%s)
    LATENCY=$((END_TIME - START_TIME))
    echo "End-to-end latency: ${LATENCY} seconds"
    break
  fi
  sleep 1
done
```

## Advanced Testing Scenarios

### Multi-Node Load Testing

**Simulate Multiple E2 Nodes:**
```bash
#!/bin/bash
# multi-node-test.sh

# Array of node identifiers
NODES=("e2-node-001" "e2-node-002" "e2-node-003" "e2-node-004")

for node in "${NODES[@]}"; do
  echo "Starting load generation for $node"
  
  # Generate KPM metrics
  ../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
    -profile random -burst 30 -period 15s \
    -out "./kmp-data/$node" &
  
  # Send VES events
  ../../tools/vessend/cmd/vessend/vessend.exe \
    -source "$node" \
    -event-type measurement \
    -count 30 -interval 15s \
    > "./ves-events/$node.log" 2>&1 &
done

wait
echo "Multi-node testing completed"
```

### Load Pattern Simulation

**Daily Load Cycle:**
```bash
#!/bin/bash
# daily-load-cycle.sh

echo "Simulating daily load cycle..."

# Morning peak (high load)
echo "Morning peak simulation..."
../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
  -profile high -burst 20 -period 30s -out ./kmp-data/morning-peak

../../tools/vessend/cmd/vessend/vessend.exe \
  -source "daily-cycle" -event-type measurement \
  -count 20 -interval 30s > ./ves-events/morning-peak.log 2>&1

sleep 60

# Daytime variable (random load)
echo "Daytime variable simulation..."
../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
  -profile random -burst 40 -period 45s -out ./kmp-data/daytime

../../tools/vessend/cmd/vessend/vessend.exe \
  -source "daily-cycle" -event-type measurement \
  -count 40 -interval 45s > ./ves-events/daytime.log 2>&1 &

sleep 120

# Overnight (low load)
echo "Overnight simulation..."
../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
  -profile low -burst 30 -period 60s -out ./kmp-data/overnight

../../tools/vessend/cmd/vessend/vessend.exe \
  -source "daily-cycle" -event-type measurement \
  -count 30 -interval 60s > ./ves-events/overnight.log 2>&1

wait
echo "Daily cycle simulation completed"
```

### Chaos Testing

**Network Partition Simulation:**
```bash
#!/bin/bash
# chaos-test.sh

echo "Starting chaos testing..."

# Generate normal traffic
../../tools/vessend/cmd/vessend/vessend.exe \
  -source "chaos-test" -event-type heartbeat \
  -count 0 -interval 10s > ./ves-events/chaos-heartbeat.log 2>&1 &

# Simulate network partition (events will fail)
echo "Simulating network partition..."
../../tools/vessend/cmd/vessend/vessend.exe \
  -collector "http://unreachable:9990/eventListener/v7" \
  -source "chaos-test" -event-type measurement \
  -count 10 -interval 5s > ./ves-events/chaos-partition.log 2>&1

# Restore connectivity and verify recovery
echo "Restoring connectivity..."
../../tools/vessend/cmd/vessend/vessend.exe \
  -source "chaos-test" -event-type measurement \
  -count 10 -interval 5s > ./ves-events/chaos-recovery.log 2>&1

# Cleanup
killall vessend.exe
echo "Chaos testing completed"
```

## Results Analysis and Reporting

### Automated Report Generation

```bash
#!/bin/bash
# generate-test-report.sh

REPORT_FILE="./results/test-report-$(date +%Y%m%d-%H%M%S).md"

cat > $REPORT_FILE << EOF
# Test Harness Execution Report
Generated: $(date)

## Test Summary
EOF

# KPM Generation Statistics
echo "## KPM Generation Statistics" >> $REPORT_FILE
echo "| Profile | Files Generated | Avg PRB Utilization | Avg Latency (ms) |" >> $REPORT_FILE
echo "|---------|-----------------|-------------------|------------------|" >> $REPORT_FILE

for profile in baseline high-load low-load; do
  if [ -d "./kmp-data/$profile" ]; then
    COUNT=$(ls ./kmp-data/$profile/*.json | wc -l)
    AVG_PRB=$(cat ./kmp-data/$profile/*.json | jq '.["kmp.prb_utilization"]' | awk '{sum+=$1} END {print sum/NR}')
    AVG_LAT=$(cat ./kmp-data/$profile/*.json | jq '.["kmp.p95_latency_ms"]' | awk '{sum+=$1} END {print sum/NR}')
    echo "| $profile | $COUNT | $AVG_PRB | $AVG_LAT |" >> $REPORT_FILE
  fi
done

# VES Event Statistics
echo "" >> $REPORT_FILE
echo "## VES Event Delivery Statistics" >> $REPORT_FILE
echo "| Event Type | Total Sent | Successful | Failed | Success Rate |" >> $REPORT_FILE
echo "|------------|------------|------------|--------|--------------|" >> $REPORT_FILE

for log in ./ves-events/*.log; do
  if [ -f "$log" ]; then
    TOTAL=$(grep -c "Sent event" "$log" 2>/dev/null || echo 0)
    SUCCESS=$(grep -c "Event sent successfully" "$log" 2>/dev/null || echo 0)
    FAILED=$((TOTAL - SUCCESS))
    if [ $TOTAL -gt 0 ]; then
      SUCCESS_RATE=$(echo "scale=2; $SUCCESS * 100 / $TOTAL" | bc)
    else
      SUCCESS_RATE="0"
    fi
    BASENAME=$(basename "$log" .log)
    echo "| $BASENAME | $TOTAL | $SUCCESS | $FAILED | ${SUCCESS_RATE}% |" >> $REPORT_FILE
  fi
done

# NetworkIntent Analysis
echo "" >> $REPORT_FILE
echo "## Generated NetworkIntents" >> $REPORT_FILE
kubectl get networkintents -n nephoran-system -o wide >> $REPORT_FILE 2>/dev/null || echo "No NetworkIntents found" >> $REPORT_FILE

echo "Report generated: $REPORT_FILE"
```

### Performance Analysis

```bash
#!/bin/bash
# performance-analysis.sh

echo "Performance Analysis Report"
echo "=========================="

# KPM Generation Performance
echo "KPM Generation Performance:"
for profile in high low random; do
  echo "Testing $profile profile..."
  time ../../tools/kmpgen/cmd/kmpgen/kmpgen.exe \
    -profile $profile -burst 100 -period 100ms -out ./perf-test/$profile
done

# VES Event Throughput
echo "VES Event Throughput:"
time ../../tools/vessend/cmd/vessend/vessend.exe \
  -event-type measurement -count 1000 -interval 10ms \
  > ./results/throughput-test.log 2>&1

# Analyze results
SUCCESS_COUNT=$(grep -c "Event sent successfully" ./results/throughput-test.log)
echo "Successfully sent $SUCCESS_COUNT/1000 events"

# Memory usage analysis
echo "Memory Usage Analysis:"
ps aux | grep -E "(kmpgen|vessend)" | grep -v grep
```

## Expected Outputs and Verification Steps

### 1. KPM Generation Verification

**Expected File Structure:**
```
kmp-data/
├── baseline/
│   ├── 20250814T143000Z_kmp-window.json
│   ├── 20250814T143030Z_kmp-window.json
│   └── ...
├── high-load/
│   ├── 20250814T144500Z_kmp-window.json
│   └── ...
└── low-load/
    ├── 20250814T150000Z_kmp-window.json
    └── ...
```

**Verification Commands:**
```bash
# Verify JSON structure
jq keys ./kmp-data/baseline/*.json | head -1
# Expected: ["kmp.p95_latency_ms", "kmp.prb_utilization", "kmp.ue_count", "timestamp", "window_id"]

# Verify profile compliance
echo "High load PRB utilization range:"
cat ./kmp-data/high-load/*.json | jq '.["kmp.prb_utilization"]' | sort -n | head -1
cat ./kmp-data/high-load/*.json | jq '.["kmp.prb_utilization"]' | sort -n | tail -1
# Expected: values between 0.80 and 0.95
```

### 2. VES Event Delivery Verification

**Expected Log Entries:**
```
2025/08/14 14:30:30 Starting VES event sender...
2025/08/14 14:30:30 Collector URL: http://localhost:9990/eventListener/v7
2025/08/14 14:30:30 Event Type: measurement
2025/08/14 14:30:30 Interval: 30s
2025/08/14 14:30:30 Count: 20
2025/08/14 14:30:30 Source: e2-node-baseline
2025/08/14 14:30:31 Event sent successfully (status: 202)
2025/08/14 14:30:31 Sent event 1/20 (sequence: 0)
...
```

**Verification Commands:**
```bash
# Check delivery success rate
TOTAL_EVENTS=$(grep -c "Sent event" ./ves-events/*.log)
SUCCESS_EVENTS=$(grep -c "Event sent successfully" ./ves-events/*.log)
echo "Success rate: $SUCCESS_EVENTS/$TOTAL_EVENTS"

# Verify VES collector received events
curl http://localhost:9990/eventListener/v7/statistics
```

### 3. System Integration Verification

**NetworkIntent Generation:**
```bash
# Expected NetworkIntent structure
kubectl get networkintents -n nephoran-system -o yaml
```

**Expected NetworkIntent YAML:**
```yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: scale-up-intent-20250814143045
  namespace: nephoran-system
spec:
  targetCNF: "o-du"
  action: "scale-up"
  replicas: 3
  reason: "High PRB utilization detected: 0.87"
  kmpTrigger:
    p95LatencyMs: 347.2
    prbUtilization: 0.87
    ueCount: 923
status:
  phase: "Processing"
  conditions:
    - type: "Validated"
      status: "True"
    - type: "Published"
      status: "True"
```

### 4. Comprehensive System Health Check

```bash
#!/bin/bash
# system-health-check.sh

echo "Comprehensive System Health Check"
echo "================================"

# Check Kubernetes components
echo "1. Kubernetes Component Status:"
kubectl get pods -n nephoran-system
kubectl get networkintents -n nephoran-system
kubectl get packagerevisions -A | grep nephoran

# Check VES collector
echo "2. VES Collector Status:"
curl -s http://localhost:9990/eventListener/v7/health | jq .

# Check generated files
echo "3. Generated Test Data:"
echo "KPM files: $(find ./kmp-data -name "*.json" | wc -l)"
echo "VES logs: $(ls ./ves-events/*.log | wc -l)"

# Check log integrity
echo "4. Event Delivery Success Rate:"
for log in ./ves-events/*.log; do
  if [ -f "$log" ]; then
    SUCCESS=$(grep -c "Event sent successfully" "$log" 2>/dev/null || echo 0)
    TOTAL=$(grep -c "Sent event" "$log" 2>/dev/null || echo 0)
    echo "$(basename "$log"): $SUCCESS/$TOTAL events delivered"
  fi
done

echo "System health check completed"
```

## Cleanup and Maintenance

### Cleanup Test Environment

```bash
#!/bin/bash
# cleanup-test-environment.sh

echo "Cleaning up test environment..."

# Stop running processes
killall kmpgen.exe vessend.exe 2>/dev/null || true

# Stop VES collector
docker stop ves-collector 2>/dev/null || true
docker rm ves-collector 2>/dev/null || true

# Archive test data
if [ -d "./kmp-data" ] || [ -d "./ves-events" ]; then
  ARCHIVE_NAME="test-data-$(date +%Y%m%d-%H%M%S).tar.gz"
  tar -czf "./results/$ARCHIVE_NAME" ./kmp-data ./ves-events
  echo "Test data archived: ./results/$ARCHIVE_NAME"
fi

# Clean up temporary files
rm -rf ./kmp-data ./ves-events ./perf-test
mkdir -p test-harness/{kmp-data,ves-events,results}

echo "Environment cleaned up"
```

### Maintenance Tasks

```bash
# Regular maintenance script
#!/bin/bash
# maintenance.sh

# Update tool binaries
echo "Updating test tools..."
cd tools/kmpgen/cmd/kmpgen && go build -o kmpgen.exe .
cd ../../../vessend/cmd/vessend && go build -o vessend.exe .

# Clean old test results (keep last 7 days)
find ./results -name "*.tar.gz" -mtime +7 -delete
find ./results -name "test-report-*.md" -mtime +7 -delete

# Verify tools
echo "Tool verification:"
../../tools/kmpgen/cmd/kmpgen/kmpgen.exe -h > /dev/null && echo "kmpgen: OK"
../../tools/vessend/cmd/vessend/vessend.exe -h > /dev/null && echo "vessend: OK"

echo "Maintenance completed"
```

This comprehensive workflow provides a complete testing framework that validates the entire Nephoran Intent Operator pipeline from metric generation to package deployment, ensuring reliable O-RAN compliant network automation.