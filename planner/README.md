# Closed-Loop Planner

The Closed-Loop Planner is a rule-based decision engine that monitors network metrics (KPM/FCAPS) and generates scaling intents to maintain optimal network performance.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  KPM Data   │────▶│ Rule Engine  │────▶│   Intent    │
│  (Metrics)  │     │              │     │  Generator  │
└─────────────┘     │  - Evaluate  │     └─────────────┘
                    │  - Cooldown   │            │
┌─────────────┐     │  - State     │            ▼
│ FCAPS Events│────▶│              │     ┌─────────────┐
│    (VES)    │     └──────────────┘     │  handoff/   │
└─────────────┘                          │  directory  │
                                         └─────────────┘
```

## Features

- **Rule-Based Scaling**: Configurable thresholds for PRB utilization and P95 latency
- **Cooldown Management**: Prevents rapid scaling oscillations
- **State Persistence**: Maintains decision history across restarts
- **Multi-Source Input**: Supports both KPM metrics and FCAPS events
- **Simulation Mode**: Test scaling logic with sample data

## Configuration

```yaml
planner:
  metrics_url: "http://localhost:9090/metrics/kpm"
  output_dir: "./handoff"
  polling_interval: 30s
  
scaling_rules:
  cooldown_duration: 60s
  min_replicas: 1
  max_replicas: 10
  
  thresholds:
    latency:
      scale_out: 100.0  # ms
      scale_in: 50.0     # ms
    prb_utilization:
      scale_out: 0.8     # 80%
      scale_in: 0.3      # 30%
```

## Scaling Rules

### Scale-Out Conditions
- PRB utilization > 80% OR
- P95 latency > 100ms
- Current replicas < max_replicas
- Cooldown period has elapsed

### Scale-In Conditions
- PRB utilization < 30% AND
- P95 latency < 50ms
- Current replicas > min_replicas
- Cooldown period has elapsed

## Building

```bash
go build -o planner ./planner/cmd/planner
```

## Running

### Production Mode
```bash
./planner -config planner/config/config.yaml
```

### Simulation Mode
```bash
export PLANNER_SIM_MODE=true
export PLANNER_OUTPUT_DIR=./handoff
./planner -config planner/config/config.yaml
```

## Testing

### Unit Tests
```bash
go test ./planner/internal/rules/...
```

### Demo Script
```bash
# Linux/Mac
./examples/planner/demo.sh

# Windows
powershell -ExecutionPolicy Bypass -File examples/planner/demo.ps1
```

## Environment Variables

- `PLANNER_METRICS_URL`: Override metrics endpoint
- `PLANNER_OUTPUT_DIR`: Override output directory
- `PLANNER_SIM_MODE`: Enable simulation mode
- `PLANNER_SIM_DATA_FILE`: Path to simulation data

## Output Format

The planner generates intent JSON files conforming to `docs/contracts/intent.schema.json`:

```json
{
  "intent_type": "scale",
  "target": "e2-node-001",
  "namespace": "default",
  "replicas": 3,
  "reason": "High PRB utilization",
  "source": "planner",
  "correlation_id": "planner-1234567890"
}
```

## Integration

The planner integrates with the conductor-loop via file handoff:
1. Planner writes intent JSON to `handoff/` directory
2. Conductor-loop monitors directory for `intent-*.json` files
3. Conductor processes intent through Nephio/Porch pipeline

## Metrics Window

- **Evaluation Window**: 90 seconds of historical data
- **Minimum Data Points**: 3 samples required for decision
- **History Retention**: 24 hours for debugging

## State Management

State is persisted to `/tmp/planner-state.json` including:
- Last decision timestamp
- Current replica count
- Metrics history (24h)
- Decision history (last 100)