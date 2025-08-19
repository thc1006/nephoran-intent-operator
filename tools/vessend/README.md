# VES Event Sender Tool (vessend)

A command-line tool for sending VES (Virtual Event Streaming) events to O-RAN VES collectors. This tool generates and sends FCAPS (Fault, Configuration, Accounting, Performance, Security) events following the VES format defined in `docs/contracts/fcaps.ves.examples.json`.

## Features

- **Multiple Event Types**: Supports fault, measurement (performance), and heartbeat events
- **Configurable Timing**: Adjustable interval between events and total event count  
- **Authentication**: Basic HTTP authentication support
- **Retry Logic**: Exponential backoff retry mechanism with configurable attempts
- **Signal Handling**: Graceful shutdown on SIGINT/SIGTERM
- **TLS Support**: Optional TLS certificate verification skip for testing
- **Realistic Data**: Generates randomized performance metrics and timestamps

## Installation

```bash
cd tools/vessend/cmd/vessend
go build -o vessend.exe .
```

## Usage

### Basic Usage

```bash
# Send 10 heartbeat events every 5 seconds to local VES collector
./vessend.exe -count 10 -interval 5s

# Send fault events to specific collector with authentication
./vessend.exe -collector "https://ves.example.com/eventListener/v7" \
              -event-type fault \
              -username admin \
              -password secret \
              -count 5

# Send continuous measurement events (no limit)
./vessend.exe -event-type measurement -count 0 -interval 30s
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-collector` | `http://localhost:9990/eventListener/v7` | VES collector URL |
| `-event-type` | `heartbeat` | Event type: `fault`, `measurement`, or `heartbeat` |
| `-interval` | `10s` | Time between events (e.g., `5s`, `1m30s`) |
| `-count` | `100` | Number of events to send (0 for unlimited) |
| `-source` | `vessend-tool` | Event source name |
| `-username` | `""` | Basic auth username (optional) |
| `-password` | `""` | Basic auth password (optional) |
| `-insecure-tls` | `false` | Skip TLS certificate verification |

## Event Types

### Heartbeat Events
Periodic liveness indicators sent at regular intervals.

```json
{
  "event": {
    "commonEventHeader": {
      "domain": "heartbeat",
      "eventName": "Heartbeat_NFSim",
      "priority": "Low"
    },
    "heartbeatFields": {
      "heartbeatFieldsVersion": "3.0",
      "heartbeatInterval": 30
    }
  }
}
```

### Fault Events
Critical system alerts with alarm information.

```json
{
  "event": {
    "commonEventHeader": {
      "domain": "fault",
      "eventName": "Fault_NFSim_LinkDown",
      "priority": "High"
    },
    "faultFields": {
      "alarmCondition": "LINK_DOWN",
      "eventSeverity": "CRITICAL",
      "specificProblem": "eth0 loss of signal"
    }
  }
}
```

### Measurement Events
Performance metrics with KPM (Key Performance Metrics) data.

```json
{
  "event": {
    "commonEventHeader": {
      "domain": "measurementsForVfScaling",
      "eventName": "Perf_NFSim_Metrics",
      "priority": "Normal"
    },
    "measurementsForVfScalingFields": {
      "vNicUsageArray": [...],
      "additionalFields": {
        "kpm.p95_latency_ms": 85.3,
        "kpm.prb_utilization": 0.62,
        "kpm.ue_count": 37
      }
    }
  }
}
```

## Examples

### Load Testing
```bash
# High-frequency fault events for stress testing
./vessend.exe -event-type fault -interval 100ms -count 1000

# Continuous measurement events with realistic intervals
./vessend.exe -event-type measurement -interval 15s -count 0
```

### Integration Testing
```bash
# Send one of each event type for validation
./vessend.exe -event-type heartbeat -count 1 -source test-heartbeat
./vessend.exe -event-type fault -count 1 -source test-fault  
./vessend.exe -event-type measurement -count 1 -source test-measurement
```

### Production Simulation
```bash
# Simulate realistic O-RAN VES traffic
./vessend.exe -collector "https://ves.oran.local/eventListener/v7" \
              -event-type measurement \
              -interval 30s \
              -source oran-cu-001 \
              -username ves_user \
              -password $VES_PASSWORD \
              -count 0
```

## Output

The tool provides structured logging with timestamps:

```
2025/08/14 09:30:29 Starting VES event sender...
2025/08/14 09:30:29 Collector URL: http://localhost:9990/eventListener/v7
2025/08/14 09:30:29 Event Type: heartbeat
2025/08/14 09:30:29 Interval: 10s
2025/08/14 09:30:29 Count: 100
2025/08/14 09:30:29 Source: vessend-tool
2025/08/14 09:30:30 Event sent successfully (status: 202)
2025/08/14 09:30:30 Sent event 1/100 (sequence: 0)
```

## Error Handling

The tool includes robust error handling:
- **Retry Logic**: 3 attempts with exponential backoff (1s, 2s, 4s delays)
- **HTTP Errors**: Detailed HTTP status code reporting
- **Network Errors**: Connection timeout and DNS resolution errors
- **Graceful Shutdown**: Clean termination on interrupt signals

## Testing

Run the included tests to verify functionality:

```bash
cd tools/vessend/cmd/vessend
go test -v
```

The tests validate:
- Event structure generation for all types
- JSON marshaling compatibility  
- Event ID uniqueness
- Configuration validation

## Integration with O-RAN Systems

This tool is designed to work with:
- **VES Collectors**: ONAP VES Collector, Nokia VES Collector
- **O-RAN Components**: O-RAN SMO, Near-RT RIC, O-DU/O-CU
- **FCAPS Systems**: Performance monitoring, fault management
- **Testing Frameworks**: Load testing, integration validation

## Compliance

Events generated follow:
- **VES Event Listener API Specification v7.2**  
- **O-RAN WG4 Management Plane Specification**
- **ETSI NFV-SOL 002/003 VNF Lifecycle Management**
- **3GPP TS 28.532 Management and orchestration**

## Related Files

- `docs/contracts/fcaps.ves.examples.json` - VES event format examples
- `docs/contracts/intent.schema.json` - NetworkIntent schema  
- `cmd/o1-ves-sim/` - O1 interface VES simulator
- `cmd/fcaps-sim/` - FCAPS event simulator