# VES Event Sender (vessend) Examples

The VES Event Sender tool (`vessend`) generates and transmits realistic VES (Virtual Event Streaming) events to O-RAN VES collectors following the FCAPS format defined in `docs/contracts/fcaps.ves.examples.json`. This tool enables comprehensive testing of O1 interface monitoring and fault management systems.

## Quick Start

### Building the Tool

```bash
cd tools/vessend/cmd/vessend
go build -o vessend.exe .
```

### Basic Usage

```bash
# Send 10 heartbeat events every 10 seconds to local VES collector
./vessend.exe

# Send fault events with custom parameters
./vessend.exe -event-type fault -count 5 -interval 30s

# Send continuous measurement events
./vessend.exe -event-type measurement -count 0 -interval 15s
```

## Event Types and Examples

### 1. Heartbeat Events

Heartbeat events provide periodic liveness indicators for network functions.

**Usage:**
```bash
./vessend.exe -event-type heartbeat -count 20 -interval 30s -source nf-du-001
```

**Generated Event Structure:**
```json
{
  "event": {
    "commonEventHeader": {
      "version": "4.1",
      "domain": "heartbeat",
      "eventName": "Heartbeat_NFSim",
      "eventId": "heartbeat-nf-du-001-1723456789",
      "sequence": 0,
      "priority": "Low",
      "reportingEntityName": "nf-du-001",
      "sourceName": "nf-du-001",
      "nfVendorName": "nephoran",
      "startEpochMicrosec": 1723456789000000,
      "lastEpochMicrosec": 1723456789000000,
      "timeZoneOffset": "+00:00"
    },
    "heartbeatFields": {
      "heartbeatFieldsVersion": "3.0",
      "heartbeatInterval": 30
    }
  }
}
```

**Use Cases:**
- Network function health monitoring
- Service availability validation
- Basic connectivity testing
- O1 interface verification

### 2. Fault Events

Fault events represent critical system alarms and operational issues.

**Usage:**
```bash
./vessend.exe -event-type fault -count 3 -interval 5s -source edge-du-002
```

**Generated Event Structure:**
```json
{
  "event": {
    "commonEventHeader": {
      "version": "4.1",
      "domain": "fault",
      "eventName": "Fault_NFSim_LinkDown",
      "eventId": "fault-edge-du-002-1723456850",
      "sequence": 1,
      "priority": "High",
      "reportingEntityName": "edge-du-002",
      "sourceName": "edge-du-002",
      "nfVendorName": "nephoran",
      "startEpochMicrosec": 1723456850000000,
      "lastEpochMicrosec": 1723456850000000,
      "timeZoneOffset": "+00:00"
    },
    "faultFields": {
      "faultFieldsVersion": "4.0",
      "alarmCondition": "LINK_DOWN",
      "eventSeverity": "CRITICAL",
      "specificProblem": "eth0 loss of signal",
      "eventSourceType": "other",
      "vfStatus": "Active",
      "alarmInterfaceA": "eth0"
    }
  }
}
```

**Use Cases:**
- Fault management system testing
- Alarm correlation verification
- Critical incident simulation
- Emergency response procedure testing

### 3. Measurement Events

Measurement events contain performance metrics including KPM data for network optimization.

**Usage:**
```bash
./vessend.exe -event-type measurement -count 50 -interval 15s -source ran-cu-001
```

**Generated Event Structure:**
```json
{
  "event": {
    "commonEventHeader": {
      "version": "4.1",
      "domain": "measurementsForVfScaling",
      "eventName": "Perf_NFSim_Metrics",
      "eventId": "measurement-ran-cu-001-1723456920",
      "sequence": 2,
      "priority": "Normal",
      "reportingEntityName": "ran-cu-001",
      "sourceName": "ran-cu-001",
      "nfVendorName": "nephoran",
      "startEpochMicrosec": 1723456920000000,
      "lastEpochMicrosec": 1723456920000000,
      "timeZoneOffset": "+00:00"
    },
    "measurementsForVfScalingFields": {
      "measurementsForVfScalingVersion": "1.1",
      "vNicUsageArray": [
        {
          "vnfNetworkInterface": "eth0",
          "receivedOctetsDelta": 1456789,
          "transmittedOctetsDelta": 2198765
        }
      ],
      "additionalFields": {
        "kpm.p95_latency_ms": 92.7,
        "kmp.prb_utilization": 0.67,
        "kmp.ue_count": 45
      }
    }
  }
}
```

**Use Cases:**
- Performance monitoring validation
- KPM data collection testing
- Scaling decision verification
- Network optimization algorithm testing

## VES Collector Setup Instructions

### Local Development Setup

**1. Using Docker (Recommended):**
```bash
# Start local VES collector container
docker run -d \
  --name ves-collector \
  -p 9990:9990 \
  -e DMAAPHOST=localhost \
  onap/org.onap.dcaegen2.collectors.ves.vescollector:latest

# Verify collector is running
curl http://localhost:9990/eventListener/v7/health
```

**2. Using ONAP VES Collector:**
```bash
# Clone ONAP VES collector
git clone https://gerrit.onap.org/r/dcaegen2/collectors/ves

# Build and run locally
cd ves
mvn clean install
java -jar target/vescollector-*.jar
```

### Production VES Collector Configuration

**Configuration with Authentication:**
```bash
./vessend.exe \
  -collector "https://ves.oran.production.com/eventListener/v7" \
  -username ves_user \
  -password "$VES_PASSWORD" \
  -event-type measurement \
  -count 0 \
  -interval 30s \
  -source prod-cu-001
```

**TLS Configuration:**
```bash
# For testing with self-signed certificates
./vessend.exe \
  -collector "https://ves.test.local/eventListener/v7" \
  -insecure-tls \
  -event-type heartbeat \
  -count 10
```

### Collector Health Verification

```bash
# Check collector availability
curl -X GET http://localhost:9990/eventListener/v7/health

# Expected response:
# {"status": "UP", "version": "7.2.1"}

# Verify event reception
curl -X POST http://localhost:9990/eventListener/v7 \
  -H "Content-Type: application/json" \
  -d '{"test": "ping"}'
```

## Integration with O1 VES Collector

### O-RAN Compliant Setup

**1. SMO Integration:**
```bash
# Send events to O-RAN SMO VES endpoint
./vessend.exe \
  -collector "https://smo.oran.local:8443/eventListener/v7" \
  -event-type measurement \
  -source o-du-cell-001 \
  -username o1_client \
  -password "$O1_PASSWORD" \
  -count 0 \
  -interval 60s
```

**2. Near-RT RIC Integration:**
```bash
# Send KPM measurement events for RIC processing
./vessend.exe \
  -collector "https://ric.oran.local:9999/eventListener/v7" \
  -event-type measurement \
  -source e2-node-001 \
  -interval 30s \
  -count 0
```

**3. O-Cloud Integration:**
```bash
# Send infrastructure events to O-Cloud VES collector
./vessend.exe \
  -collector "https://ocloud.infra.local/eventListener/v7" \
  -event-type fault \
  -source k8s-worker-001 \
  -interval 10s \
  -count 100
```

### Event Flow Verification

**1. End-to-End Testing:**
```bash
# Generate measurement events and verify planner receives them
./vessend.exe -event-type measurement -count 5 -interval 15s

# Check if planner processes the events
kubectl logs -n nephoran-system deployment/planner -f
```

**2. FCAPS Integration:**
```bash
# Send fault events and verify alert manager integration
./vessend.exe -event-type fault -count 3 -source test-node

# Check Prometheus alerts
curl http://localhost:9093/api/v1/alerts | jq '.data[] | select(.labels.source=="test-node")'
```

## Command-Line Reference

### All Available Flags

| Flag | Default | Description | Example |
|------|---------|-------------|---------|
| `-collector` | `http://localhost:9990/eventListener/v7` | VES collector URL | `-collector "https://ves.prod.com/eventListener/v7"` |
| `-event-type` | `heartbeat` | Event type to send | `-event-type fault` |
| `-interval` | `10s` | Time between events | `-interval 30s` |
| `-count` | `100` | Number of events (0=unlimited) | `-count 50` |
| `-source` | `vessend-tool` | Event source name | `-source "oran-du-001"` |
| `-username` | `""` | Basic auth username | `-username admin` |
| `-password` | `""` | Basic auth password | `-password secret123` |
| `-insecure-tls` | `false` | Skip TLS verification | `-insecure-tls` |

### Duration Format Examples

```bash
# Seconds
-interval 30s

# Minutes  
-interval 5m

# Hours
-interval 2h

# Milliseconds
-interval 500ms

# Combined
-interval 1m30s
```

## Testing Scenarios

### 1. Load Testing

**High-Frequency Event Generation:**
```bash
# Stress test VES collector with rapid events
./vessend.exe -event-type measurement -interval 100ms -count 1000
```

**Sustained Load Testing:**
```bash
# Generate continuous load for extended periods
./vessend.exe -event-type heartbeat -interval 5s -count 0 &
./vessend.exe -event-type measurement -interval 15s -count 0 &
./vessend.exe -event-type fault -interval 120s -count 0 &
```

### 2. Failure Scenario Testing

**Network Partition Simulation:**
```bash
# Generate events during simulated network issues
./vessend.exe -collector "http://unreachable-collector:9990/eventListener/v7" \
  -event-type fault -count 10 -interval 1s
```

**Authentication Failure Testing:**
```bash
# Test with invalid credentials
./vessend.exe -username invalid -password wrong \
  -event-type heartbeat -count 5
```

### 3. Integration Testing

**Multi-Source Event Generation:**
```bash
#!/bin/bash
# Simulate multiple network functions

./vessend.exe -source "o-du-001" -event-type measurement -count 20 &
./vessend.exe -source "o-du-002" -event-type measurement -count 20 &
./vessend.exe -source "o-cu-001" -event-type heartbeat -count 50 &
./vessend.exe -source "edge-node-001" -event-type fault -count 5 &

wait
echo "Multi-source event generation completed"
```

**Event Type Validation:**
```bash
# Send one of each event type for validation
./vessend.exe -event-type heartbeat -count 1 -source test-heartbeat
./vessend.exe -event-type fault -count 1 -source test-fault
./vessend.exe -event-type measurement -count 1 -source test-measurement
```

### 4. Performance Benchmarking

**Throughput Testing:**
```bash
# Measure maximum sustainable event rate
time ./vessend.exe -event-type measurement -interval 10ms -count 1000
```

**Latency Testing:**
```bash
# Test event delivery latency with timestamps
./vessend.exe -event-type heartbeat -count 10 -interval 1s | \
  while read line; do
    echo "$(date): $line"
  done
```

## Monitoring and Observability

### Event Delivery Verification

**Log Analysis:**
```bash
# Monitor vessend output for delivery status
./vessend.exe -event-type measurement -count 10 2>&1 | \
  grep "Event sent successfully"
```

**Collector Side Verification:**
```bash
# Check VES collector logs for received events
docker logs ves-collector | grep "Event received"

# For ONAP VES collector
kubectl logs -n onap deployment/ves-collector | grep "POST /eventListener"
```

### Metrics Collection

**Event Statistics:**
```bash
# Generate events with detailed logging
./vessend.exe -event-type measurement -count 100 -interval 1s 2>&1 | \
  tee event-delivery.log

# Analyze success rate
grep "Event sent successfully" event-delivery.log | wc -l
grep "Failed to send" event-delivery.log | wc -l
```

## Troubleshooting

### Common Issues and Solutions

**1. Connection Refused:**
```bash
# Error: connection refused
# Solution: Verify VES collector is running and accessible
curl http://localhost:9990/eventListener/v7/health

# If using Docker, check container status
docker ps | grep ves-collector
```

**2. Authentication Failures:**
```bash
# Error: HTTP 401 Unauthorized
# Solution: Verify credentials and authentication method
./vessend.exe -username correct_user -password correct_pass -event-type heartbeat -count 1
```

**3. TLS Certificate Issues:**
```bash
# Error: x509: certificate signed by unknown authority
# Solution: Use -insecure-tls for testing or provide proper certificates
./vessend.exe -insecure-tls -collector "https://ves.test.local/eventListener/v7"
```

**4. High Memory Usage:**
```bash
# Issue: Memory usage grows with unlimited count
# Solution: Use bounded count or monitor memory usage
./vessend.exe -count 1000 -interval 100ms  # Bounded
# vs
./vessend.exe -count 0 -interval 100ms     # Monitor memory
```

### Debugging Tips

**1. JSON Validation:**
```bash
# Capture and validate generated JSON
./vessend.exe -count 1 -event-type measurement 2>&1 | \
  grep -A 50 "Event sent successfully" | jq .
```

**2. Network Debugging:**
```bash
# Use tcpdump to monitor network traffic
sudo tcpdump -i any port 9990 -A

# Use curl for manual testing
curl -X POST http://localhost:9990/eventListener/v7 \
  -H "Content-Type: application/json" \
  -d @sample-event.json
```

**3. Performance Profiling:**
```bash
# Profile CPU usage during high-frequency sending
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Monitor memory allocation
go tool pprof http://localhost:6060/debug/pprof/heap
```

## Advanced Usage

### Custom Event Templates

**Creating Event Variants:**
```bash
# Modify source code to create custom event templates
# See tools/vessend/cmd/vessend/main.go, createEvent() function

# Example: Custom measurement fields
"additionalFields": {
  "kmp.p95_latency_ms": 92.7,
  "kmp.prb_utilization": 0.67,
  "kmp.ue_count": 45,
  "custom.cell_id": "cell-001",
  "custom.bandwidth_mbps": 100.5
}
```

### Batch Event Generation

**Script for Multiple Scenarios:**
```bash
#!/bin/bash
# comprehensive-ves-test.sh

echo "Starting comprehensive VES event testing..."

# Normal operation baseline
./vessend.exe -source "baseline-test" -event-type heartbeat -count 30 -interval 60s &
./vessend.exe -source "baseline-test" -event-type measurement -count 20 -interval 90s &

# Fault injection
sleep 300
./vessend.exe -source "fault-test" -event-type fault -count 5 -interval 30s &

# High load simulation  
sleep 180
./vessend.exe -source "load-test" -event-type measurement -count 50 -interval 10s &

wait
echo "Comprehensive VES testing completed"
```

### Integration with CI/CD

**Automated Testing Pipeline:**
```bash
# test-ves-integration.sh
#!/bin/bash

# Start VES collector
docker run -d --name test-ves-collector -p 9990:9990 onap/vescollector

# Wait for startup
sleep 10

# Run tests
./vessend.exe -count 10 -event-type heartbeat
TEST_RESULT=$?

# Cleanup
docker stop test-ves-collector
docker rm test-ves-collector

exit $TEST_RESULT
```

## Related Documentation

- `docs/contracts/fcaps.ves.examples.json` - VES event format specification
- `tools/vessend/README.md` - Detailed tool documentation
- `cmd/o1-ves-sim/` - O1 interface VES simulator
- `cmd/fcaps-sim/` - FCAPS event simulator
- `examples/tools/test-harness-workflow.md` - Complete testing workflow
- O-RAN WG4 Management Plane Specification
- VES Event Listener API Specification v7.2