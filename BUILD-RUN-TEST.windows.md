# FCAPS Simulator - Build, Run & Test Guide (Windows)

## Quick Start

```powershell
# Build
go build .\cmd\fcaps-sim

# Run with local reducer (burst detection)
.\fcaps-sim.exe --delay 1 --burst 3 --nf-name nf-sim --out-handoff .\handoff

# Check for generated intents
Get-ChildItem .\handoff\intent-*.json
```

## Overview

The FCAPS simulator (`fcaps-sim`) emits VES (VNF Event Stream) JSON events to a configurable HTTP collector and triggers scaling intents based on event patterns. The reducer (`fcaps-reducer`) detects event bursts and generates scaling intents.

## Prerequisites

- Go 1.21+ installed
- Git Bash or PowerShell
- Network access for HTTP communication

## Build Instructions

### 1. Build the FCAPS Simulator

```powershell
# From repository root
go build -o fcaps-sim.exe ./cmd/fcaps-sim
```

### 2. Build the FCAPS Reducer

```powershell
# From repository root
go build -o fcaps-reducer.exe ./cmd/fcaps-reducer
```

### 3. Build the Intent Ingest Service (Optional)

```powershell
# From repository root
go build -o intent-ingest.exe ./cmd/intent-ingest
```

## Run Instructions

### Component 1: FCAPS Reducer (VES Collector)

Start the reducer first to collect VES events and detect bursts:

```powershell
# Basic startup
./fcaps-reducer.exe

# With custom configuration
./fcaps-reducer.exe --listen :9999 --handoff ./handoff --burst 3 --window 60 --verbose
```

**Flags:**
- `--listen`: HTTP listen address (default: `:9999`)
- `--handoff`: Directory for intent files (default: `./handoff`)
- `--burst`: Number of critical events to trigger scaling (default: 3)
- `--window`: Time window in seconds for burst detection (default: 60)
- `--verbose`: Enable detailed logging

### Component 2: Intent Ingest Service (Optional)

If you want to process intents directly:

```powershell
# Start intent ingest service
./intent-ingest.exe --out ./handoff
```

### Component 3: FCAPS Simulator with Local Reducer

Run the simulator with integrated reducer:

```powershell
# Local reducer mode (no external dependencies)
./fcaps-sim.exe --delay 1 --burst 3 --nf-name nf-sim --out-handoff ./handoff

# With verbose logging
./fcaps-sim.exe --delay 1 --burst 3 --nf-name nf-sim --out-handoff ./handoff --verbose

# Custom VES events
./fcaps-sim.exe `
  --input docs/contracts/fcaps.ves.examples.json `
  --delay 2 `
  --burst 2 `
  --nf-name nf-sim `
  --out-handoff ./handoff `
  --verbose
```

**Flags:**
- `--input`: Path to VES events JSON file (default: `docs/contracts/fcaps.ves.examples.json`)
- `--delay`: Seconds between events (default: 5)
- `--burst`: Critical event threshold for burst detection (default: 1)
- `--target`: Target deployment name (default: `nf-sim`)
- `--namespace`: Target namespace (default: `ran-a`)
- `--nf-name`: Network Function name (default: `nf-sim`)
- `--out-handoff`: Local handoff directory for reducer mode (enables local reducer)
- `--collector-url`: VES collector URL (optional, default: `http://localhost:9999/eventListener/v7`)
- `--intent-url`: Intent service URL (optional, default: `http://localhost:8080/intent`)
- `--verbose`: Enable verbose logging

## Test Scenarios

### Scenario 1: Normal Operation

Test basic event processing without triggering scaling:

```powershell
# Terminal 1: Start reducer
./fcaps-reducer.exe --verbose

# Terminal 2: Send normal load events
./fcaps-sim.exe --delay 10 --verbose
```

Expected: Events processed, no scaling intent generated.

### Scenario 2: High Load Burst

Simulate a burst of critical events to trigger scaling:

1. Create a high-load test file:

```powershell
# Create test-high-load.json with multiple critical events
@'
{
  "critical_1": {
    "event": {
      "commonEventHeader": {
        "version": "4.1",
        "domain": "fault",
        "eventName": "Fault_Critical_1",
        "eventId": "fault-001",
        "sequence": 1,
        "priority": "High",
        "reportingEntityName": "nf-sim",
        "sourceName": "nf-sim",
        "nfVendorName": "nephoran",
        "startEpochMicrosec": 1731000000000,
        "lastEpochMicrosec": 1731000000000,
        "timeZoneOffset": "+00:00"
      },
      "faultFields": {
        "faultFieldsVersion": "4.0",
        "alarmCondition": "OVERLOAD",
        "eventSeverity": "CRITICAL",
        "specificProblem": "System overload detected",
        "eventSourceType": "other",
        "vfStatus": "Degraded",
        "alarmInterfaceA": "system"
      }
    }
  },
  "high_load_1": {
    "event": {
      "commonEventHeader": {
        "version": "4.1",
        "domain": "measurementsForVfScaling",
        "eventName": "Perf_HighLoad",
        "eventId": "perf-002",
        "sequence": 2,
        "priority": "High",
        "reportingEntityName": "nf-sim",
        "sourceName": "nf-sim",
        "nfVendorName": "nephoran",
        "startEpochMicrosec": 1731000010000,
        "lastEpochMicrosec": 1731000010000,
        "timeZoneOffset": "+00:00"
      },
      "measurementsForVfScalingFields": {
        "measurementsForVfScalingVersion": "1.1",
        "additionalFields": {
          "kpm.p95_latency_ms": 250.0,
          "kpm.prb_utilization": 0.92,
          "kpm.cpu_utilization": 0.88
        }
      }
    }
  },
  "critical_2": {
    "event": {
      "commonEventHeader": {
        "version": "4.1",
        "domain": "fault",
        "eventName": "Fault_Critical_2",
        "eventId": "fault-003",
        "sequence": 3,
        "priority": "Critical",
        "reportingEntityName": "nf-sim",
        "sourceName": "nf-sim",
        "nfVendorName": "nephoran",
        "startEpochMicrosec": 1731000020000,
        "lastEpochMicrosec": 1731000020000,
        "timeZoneOffset": "+00:00"
      },
      "faultFields": {
        "faultFieldsVersion": "4.0",
        "alarmCondition": "CONGESTION",
        "eventSeverity": "CRITICAL",
        "specificProblem": "Severe congestion",
        "eventSourceType": "other",
        "vfStatus": "Degraded",
        "alarmInterfaceA": "eth0"
      }
    }
  }
}
'@ > test-high-load.json
```

2. Run the test:

```powershell
# Terminal 1: Start reducer with low threshold
./fcaps-reducer.exe --burst 2 --window 30 --verbose

# Terminal 2: Send high load events
./fcaps-sim.exe --input test-high-load.json --delay 1 --verbose
```

Expected output:
- Reducer detects burst
- Intent file created in `./handoff/intent-*.json`
- Scaling intent shows increased replicas

### Scenario 3: End-to-End with Intent Processing

```powershell
# Terminal 1: Start reducer
./fcaps-reducer.exe --verbose

# Terminal 2: Start intent ingest
./intent-ingest.exe --out ./handoff

# Terminal 3: Simulate events
./fcaps-sim.exe --delay 2 --verbose
```

## Verification with curl

### Test VES Collector Endpoint

```bash
# Send a test VES event
curl -X POST http://localhost:9999/eventListener/v7 \
  -H "Content-Type: application/json" \
  -H "X-MinorVersion: 1" \
  -H "X-PatchVersion: 0" \
  -H "X-LatestVersion: 7.3" \
  -d @docs/contracts/fcaps.ves.examples.json

# Check health
curl http://localhost:9999/health
```

### Test Intent Endpoint

```bash
# Send a scaling intent
curl -X POST http://localhost:8080/intent \
  -H "Content-Type: application/json" \
  -d '{
    "intent_type": "scaling",
    "target": "nf-sim",
    "namespace": "ran-a",
    "replicas": 5,
    "reason": "Manual scaling test",
    "source": "curl"
  }'
```

## Expected Log Output

### Local Reducer Mode

```
2025/08/17 10:00:00 FCAPS Simulator starting with config: {...}
2025/08/17 10:00:00 Loaded 6 FCAPS events from docs\contracts\fcaps.ves.examples.json
2025/08/17 10:00:00 Local reducer enabled: burst=3, handoff=./handoff
2025/08/17 10:00:01 Processing event: Fault_NFSim_LinkDown (domain: fault, source: nf-sim)
2025/08/17 10:00:02 Processing event: Perf_NFSim_Metrics (domain: measurementsForVfScaling, source: nf-sim)
2025/08/17 10:00:03 Processing event: Heartbeat_NFSim (domain: heartbeat, source: nf-sim)
2025/08/17 10:00:04 Processing event: Perf_NFSim_HighLoad (domain: measurementsForVfScaling, source: nf-sim)
2025/08/17 10:00:05 *** BURST DETECTED *** Intent written: handoff\intent-20250817T020005Z.json
2025/08/17 10:00:05     Scaling nf-sim to 3 replicas (reason: Burst detected: 3 critical events in 1m0s window)
```

## Intent File Structure

Generated intent files in `./handoff/` directory:

```json
{
  "intent_type": "scaling",
  "target": "nf-sim",
  "namespace": "ran-a",
  "replicas": 4,
  "reason": "Burst detected: 3 critical events in 60s window",
  "source": "fcaps-reducer",
  "correlation_id": "burst-1731074400"
}
```

## Troubleshooting

### Port Already in Use

```powershell
# Find process using port 9999
netstat -ano | findstr :9999

# Kill the process (replace PID with actual process ID)
taskkill /PID <PID> /F
```

### Permission Issues

```powershell
# Run as Administrator or ensure write permissions for handoff directory
mkdir handoff
icacls handoff /grant Everyone:F
```

### Build Errors

```powershell
# Clean module cache
go clean -modcache

# Download dependencies
go mod download

# Rebuild
go build -v ./cmd/fcaps-sim
```

## Performance Testing

### Load Test with Multiple Events

```powershell
# Generate continuous load
while ($true) {
    ./fcaps-sim.exe --delay 1 --verbose
    Start-Sleep -Seconds 5
}
```

### Monitor Intent Generation

```powershell
# Watch for new intent files
Get-ChildItem ./handoff -Filter "intent-*.json" | 
    Sort-Object LastWriteTime -Descending | 
    Select-Object -First 5
```

## Integration with Conductor

The generated intent files in `./handoff/` are designed to be picked up by the conductor-watch loop:

```powershell
# Start conductor watch (in separate terminal)
./conductor-watch.exe --input ./handoff --output ./processed
```

## Clean Up

```powershell
# Remove generated files
Remove-Item ./handoff/intent-*.json -Force
Remove-Item *.exe -Force
Remove-Item test-*.json -Force
```