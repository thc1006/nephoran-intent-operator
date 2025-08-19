# FCAPS Simulator Validation Guide

## Validation Results ✅

All validation tasks completed successfully:
1. ✅ Built fcaps-sim executable
2. ✅ Generated VES events (including threshold crossing alerts)
3. ✅ Verified handoff JSON files are created
4. ✅ Confirmed agent definitions exist in `.claude/agents`

## How to Run the FCAPS → Intent Pipeline

### Prerequisites
- Go 1.24+ installed
- Working directory: `C:\Users\tingy\dev\_worktrees\nephoran\feat-sim-fcaps`

### Step 1: Build the Executables
```bash
# Build all required components
go build ./cmd/fcaps-sim
go build ./cmd/intent-ingest
go build ./cmd/porch-publisher
```

### Step 2: Start the Intent Ingest Service
```bash
# Start the intent-ingest service (runs on port 8080)
./intent-ingest.exe
```
Keep this running in a separate terminal.

### Step 3: Run FCAPS Simulator
```bash
# Run the simulator with VES events
./fcaps-sim.exe --verbose --delay 1
```

This will:
- Load VES events from `docs/contracts/fcaps.ves.examples.json`
- Process fault, measurement, and heartbeat events
- Generate scaling intents when critical faults are detected
- Send intents to the intent-ingest service

### Step 4: Verify Intent JSON Creation
```bash
# Check the handoff directory for new intent files
ls -la handoff/intent-*.json
```

New intent files will be created with timestamps (e.g., `intent-20250815T111408Z.json`)

### Step 5: Generate Kubernetes Patches (Optional)
```bash
# Convert intent JSON to Kubernetes scaling patch
./porch-publisher.exe -intent handoff/intent-TIMESTAMP.json -out examples/packages/scaling
```

This creates `scaling-patch.yaml` for Kubernetes deployment scaling.

## Event → Policy → Intent Mapping

The current implementation includes:

### Fault Events
- **Critical Faults** (e.g., LINK_DOWN) → Scale to 3 replicas
- **Major Faults** → Scale to 2 replicas  
- **Minor/Warning** → No scaling action

### Measurement Events
- **PRB Utilization > 80%** → Scale up by 2 replicas
- **PRB Utilization > 70%** → Scale up by 1 replica
- **P95 Latency > 100ms** → Scale up by 1 replica

### Combined Conditions
- High load (>70% PRB) + High latency (>100ms) → Scale up by 2 replicas

## VES Event Structure

Events follow ONAP VES Event Listener 7.2 specification with:
- `commonEventHeader`: Standard VES headers with domain, eventName, severity
- Domain-specific fields:
  - `faultFields`: For fault domain events
  - `measurementsForVfScalingFields`: For performance metrics
  - `heartbeatFields`: For heartbeat events

## Available Agents

The `.claude/agents` directory contains specialized agents for:
- O-RAN/Nephio integration and orchestration
- Network function management
- Performance optimization
- Security compliance
- Configuration management
- Monitoring and analytics

These agents can be orchestrated for complex telecom automation workflows.

## Testing Custom VES Events

To test with custom events:

1. Edit `docs/contracts/fcaps.ves.examples.json` with your VES events
2. Run the simulator: `./fcaps-sim.exe --input your-events.json`
3. Adjust thresholds in `internal/fcaps/processor.go` as needed

## Troubleshooting

- **Port 8080 in use**: Stop any services using port 8080 or modify the port in `cmd/intent-ingest/main.go`
- **No intents generated**: Check event severity/metrics meet thresholds in the processor
- **Build errors**: Ensure Go 1.24+ and run `go mod download`