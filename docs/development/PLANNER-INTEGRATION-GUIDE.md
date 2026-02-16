# Planner Integration Guide

## Overview
The Nephoran Closed-Loop Planner integrates with the MVP stack to enable autonomous network scaling based on KPM metrics.

## Integration Flow

```
┌──────────┐     ┌─────────┐     ┌───────────┐     ┌───────────┐     ┌─────────┐
│ KPM Gen  │────▶│ Planner │────▶│  Intent   │────▶│ Conductor │────▶│   CRD   │
│ (metrics)│     │ (rules) │     │  (JSON)   │     │  (HTTP)   │     │ (K8s)   │
└──────────┘     └─────────┘     └───────────┘     └───────────┘     └─────────┘
                                        │                                   │
                                        ▼                                   ▼
                                  handoff/*.json                     NetworkIntent
                                                                           │
                                                                           ▼
                                                                    ┌─────────────┐
                                                                    │   Porch     │
                                                                    │ (packages)  │
                                                                    └─────────────┘
                                                                           │
                                                                           ▼
                                                                    ┌─────────────┐
                                                                    │ Deployment  │
                                                                    │  (scaled)   │
                                                                    └─────────────┘
```

## Quick Start

### 1. Generate Metrics (Terminal 1)
```powershell
cd C:\Users\tingy\dev\_worktrees\nephoran\feat-test-harness
go run .\tools\kpmgen\cmd\kpmgen --out .\metrics --period 1s
```

### 2. Run Planner (Terminal 2)
```powershell
cd C:\Users\tingy\dev\_worktrees\nephoran\feat-planner
.\planner.exe -config planner/config/local-test.yaml
```

### 3. Verify Intent Generation
```powershell
# Check for generated intents
dir handoff\intent-*.json

# View latest intent
Get-Content (Get-ChildItem handoff\intent-*.json | Sort-Object LastWriteTime -Descending | Select-Object -First 1).FullName | ConvertFrom-Json
```

### 4. POST to Conductor
```bash
# Using curl
curl -X POST http://localhost:8080/intent \
  -H "Content-Type: application/json" \
  -d @handoff/intent-*.json

# Using PowerShell
Invoke-RestMethod -Uri "http://localhost:8080/intent" `
  -Method Post `
  -ContentType "application/json" `
  -InFile (Get-ChildItem handoff\intent-*.json | Select -First 1).FullName
```

## Configuration

### Planner Config (`planner/config/local-test.yaml`)
```yaml
planner:
  metrics_dir: "../feat-test-harness/metrics"  # Read from kpmgen
  output_dir: "./handoff"                      # Write intents here
  polling_interval: 5s                         # Check every 5 seconds
  
scaling_rules:
  cooldown_duration: 30s                       # Wait between decisions
  min_replicas: 1
  max_replicas: 10
  thresholds:
    latency:
      scale_out: 100.0                        # Scale out if P95 > 100ms
      scale_in: 50.0                           # Scale in if P95 < 50ms
    prb_utilization:
      scale_out: 0.8                           # Scale out if PRB > 80%
      scale_in: 0.3                            # Scale in if PRB < 30%
```

## Testing Scripts

### Quick Test (Simulation Mode)
```powershell
.\examples\planner\quick-test.ps1
```

### Local Integration Test
```powershell
.\examples\planner\local-test.ps1
```

### Full Integration Test
```powershell
.\examples\planner\full-integration-test.ps1
```

## Scaling Rules

| Condition | PRB Utilization | P95 Latency | Action | Replicas Change |
|-----------|----------------|-------------|---------|-----------------|
| High Load | > 80% | OR > 100ms | Scale Out | +1 |
| Low Load | < 30% | AND < 50ms | Scale In | -1 |
| Cooldown | - | - | No Action | 0 |

## Intent Schema

Generated intents conform to `docs/contracts/intent.schema.json`:

```json
{
  "intent_type": "scaling",
  "target": "e2-node-001",
  "namespace": "default",
  "replicas": 3,
  "reason": "High PRB utilization",
  "source": "planner",
  "correlation_id": "planner-1234567890"
}
```

## Verification

### Check Intent Files
```powershell
Get-ChildItem handoff\intent-*.json | Select Name, LastWriteTime
```

### Check Conductor Response
```json
{
  "status": "accepted",
  "saved": "C:\\path\\to\\handoff\\intent-timestamp.json",
  "preview": { ... }
}
```

### Check NetworkIntent CRDs (if K8s connected)
```bash
kubectl get networkintents
kubectl describe networkintent <name>
```

## Troubleshooting

### No Intent Generated
- Check if metrics files exist in metrics directory
- Verify PRB/latency exceed thresholds
- Check cooldown period (default 30s)
- Review planner logs

### Conductor Not Accepting
- Verify conductor is running: `curl http://localhost:8080`
- Check intent JSON matches schema
- Review conductor logs

### State Persistence Issues
- Clear state: `rm planner-state.json`
- Check file permissions
- Use absolute path for state file

## Environment Variables

```powershell
$env:PLANNER_METRICS_DIR = "../feat-test-harness/metrics"
$env:PLANNER_OUTPUT_DIR = "./handoff"
$env:PLANNER_SIM_MODE = "true"  # For simulation mode
```

## Makefile Targets

```bash
make planner-build    # Build planner
make planner-test     # Run tests
make planner-demo     # Run demo
make planner-run      # Run planner
```

## Next Steps

1. **Connect to Kubernetes**: Ensure conductor creates NetworkIntent CRDs
2. **Porch Integration**: Verify package generation from intents
3. **End-to-End Testing**: Validate deployment scaling
4. **Production Config**: Tune thresholds based on real metrics
5. **Monitoring**: Add Prometheus metrics for planner decisions