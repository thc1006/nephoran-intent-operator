# KPM Generator (kpmgen) Examples

The KPM Generator tool (`kpmgen`) creates realistic KPM (Key Performance Measurement) window JSON files according to the E2SM-KPM profile defined in `docs/contracts/e2.kmp.profile.md`. These files simulate E2 node measurements that can be consumed by the planner component for network intent scaling decisions.

## Quick Start

### Building the Tool

```bash
cd tools/kmpgen/cmd/kmpgen
go build -o kpmgen.exe .
```

### Basic Usage

```bash
# Generate 10 random profile KPM windows every second
./kpmgen.exe -profile random -burst 10 -period 1s -out ./test-metrics

# Generate high-load profile windows for stress testing
./kpmgen.exe -profile high -burst 100 -period 500ms -out ./stress-test

# Generate low-load profile windows for baseline testing  
./kpmgen.exe -profile low -burst 50 -period 2s -out ./baseline-test
```

## Load Profiles

### High Load Profile
Simulates high-utilization network conditions that should trigger scale-up operations.

**Characteristics:**
- P95 Latency: 200-500ms (poor performance)
- PRB Utilization: 0.80-0.95 (very high resource usage)
- UE Count: 800-1000 (high user load)

**Example Usage:**
```bash
./kpmgen.exe -profile high -burst 60 -period 30s -out ./high-load-metrics
```

**Sample Output JSON:**
```json
{
  "kmp.p95_latency_ms": 347.2,
  "kmp.prb_utilization": 0.89,
  "kmp.ue_count": 923,
  "timestamp": "2025-08-14T10:30:45Z",
  "window_id": "window-15"
}
```

### Low Load Profile
Simulates light-utilization network conditions that should trigger scale-down operations.

**Characteristics:**
- P95 Latency: 10-50ms (excellent performance)
- PRB Utilization: 0.10-0.30 (low resource usage)
- UE Count: 50-200 (light user load)

**Example Usage:**
```bash
./kpmgen.exe -profile low -burst 30 -period 1m -out ./low-load-metrics
```

**Sample Output JSON:**
```json
{
  "kmp.p95_latency_ms": 23.8,
  "kmp.prb_utilization": 0.18,
  "kmp.ue_count": 127,
  "timestamp": "2025-08-14T10:30:45Z", 
  "window_id": "window-8"
}
```

### Random Profile
Generates varied measurements across the full spectrum for comprehensive testing.

**Characteristics:**
- P95 Latency: 10-500ms (full range)
- PRB Utilization: 0.10-0.95 (full range)
- UE Count: 50-1000 (full range)

**Example Usage:**
```bash
./kpmgen.exe -profile random -burst 0 -period 15s -out ./random-metrics
```

**Sample Output JSON:**
```json
{
  "kmp.p95_latency_ms": 156.7,
  "kmp.prb_utilization": 0.43,
  "kmp.ue_count": 367,
  "timestamp": "2025-08-14T10:30:45Z",
  "window_id": "window-42"
}
```

## Command-Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-profile` | `random` | Load profile: `high`, `low`, or `random` |
| `-burst` | `60` | Number of windows to generate (0 for unlimited) |
| `-period` | `1s` | Time between emissions (e.g., `500ms`, `30s`, `1m`) |
| `-out` | `./kmp-windows` | Output directory for JSON files |

## File Naming Convention

Generated files follow the pattern: `YYYYMMDDTHHMMSSZ_kmp-window.json`

Examples:
- `20250814T103045Z_kmp-window.json`
- `20250814T103115Z_kmp-window.json`
- `20250814T103145Z_kmp-window.json`

## Integration with Planner Workflow

### 1. Generate High-Load Scenario
```bash
# Simulate network congestion that should trigger scale-up
./kpmgen.exe -profile high -burst 10 -period 30s -out ./planner-input
```

### 2. Feed to Planner
The planner component monitors KPM windows and generates NetworkIntent scaling decisions based on the metrics:

```bash
# The planner watches the output directory and processes new files
# When it detects high PRB utilization (>0.8) and high latency (>200ms),
# it generates a scale-up NetworkIntent
```

### 3. Expected Planner Behavior
- **High Load Profile**: Should generate scale-up NetworkIntents
- **Low Load Profile**: Should generate scale-down NetworkIntents  
- **Random Profile**: Should generate mixed scaling decisions

## Testing Scenarios

### Stress Testing
Generate rapid high-load conditions to test planner responsiveness:
```bash
./kpmgen.exe -profile high -burst 100 -period 100ms -out ./stress-test
```

### Stability Testing
Generate continuous random load for long-term testing:
```bash
./kpmgen.exe -profile random -burst 0 -period 15s -out ./stability-test
```

### Scaling Threshold Testing
Test specific threshold conditions:
```bash
# Test scale-up threshold (high utilization)
./kpmgen.exe -profile high -burst 5 -period 30s -out ./scale-up-test

# Test scale-down threshold (low utilization)  
./kpmgen.exe -profile low -burst 5 -period 30s -out ./scale-down-test
```

### Load Pattern Simulation
Simulate realistic daily load patterns:
```bash
# Morning peak (high load)
./kpmgen.exe -profile high -burst 20 -period 30s -out ./morning-peak

# Overnight (low load)
./kpmgen.exe -profile low -burst 30 -period 1m -out ./overnight

# Variable daytime (random)
./kpmgen.exe -profile random -burst 50 -period 45s -out ./daytime-variable
```

## Planner Integration Points

### KPM Metrics Mapping
The generated KPM windows map to planner scaling decisions as follows:

| KPM Metric | Threshold | Action |
|------------|-----------|--------|
| `kmp.prb_utilization` | > 0.8 | Scale up CNFs |
| `kmp.prb_utilization` | < 0.2 | Scale down CNFs |
| `kmp.p95_latency_ms` | > 200 | Scale up CNFs |
| `kmp.p95_latency_ms` | < 50 | Consider scale down |
| `kmp.ue_count` | > 800 | Scale up CNFs |
| `kmp.ue_count` | < 100 | Consider scale down |

### File Processing
The planner component:
1. Watches the output directory for new KPM window files
2. Parses the JSON to extract metrics
3. Applies scaling rules based on thresholds
4. Generates NetworkIntent CRDs for Kubernetes

### Verification
After generating KPM windows, verify planner processing:
```bash
# Check generated NetworkIntents
kubectl get networkintents -n nephoran-system

# Verify scaling decisions match expected outcomes
kubectl describe networkintent <intent-name> -n nephoran-system
```

## Troubleshooting

### Common Issues

**Permission Denied:**
```bash
# Ensure output directory is writable
chmod 755 ./output-directory
```

**Invalid Profile:**
```bash
# Verify profile name is correct
./kpmgen.exe -profile high  # Valid: high, low, random
```

**High Memory Usage:**
```bash
# For unlimited burst (-burst 0), monitor memory usage
# Consider using smaller periods or limiting burst size
./kpmgen.exe -profile random -burst 1000 -period 1s  # Bounded
```

### Validation
Verify generated JSON structure matches E2SM-KPM profile:
```bash
# Check JSON format
cat output-directory/20250814T103045Z_kmp-window.json | jq .

# Verify required fields
jq '.["kmp.p95_latency_ms"], .["kmp.prb_utilization"], .["kmp.ue_count"]' output-file.json
```

## Advanced Usage

### Batch Processing
Generate multiple scenarios in sequence:
```bash
#!/bin/bash
# Generate test scenarios for comprehensive planner testing

# High load morning peak
./kpmgen.exe -profile high -burst 20 -period 30s -out ./scenarios/morning-peak

# Low load overnight
./kmpgen.exe -profile low -burst 30 -period 1m -out ./scenarios/overnight

# Variable daytime load
./kmpgen.exe -profile random -burst 40 -period 45s -out ./scenarios/daytime
```

### Integration with CI/CD
Use in automated testing pipelines:
```bash
# Generate test data for planner integration tests
./kmpgen.exe -profile high -burst 5 -period 1s -out ./test-data
./kmpgen.exe -profile low -burst 5 -period 1s -out ./test-data

# Run planner tests with generated data
go test ./internal/planner/... -kmp-data-dir=./test-data
```

## Related Documentation

- `docs/contracts/e2.kmp.profile.md` - E2SM-KPM specification
- `docs/contracts/intent.schema.json` - NetworkIntent schema
- `internal/planner/planner.go` - Planner implementation
- `examples/tools/test-harness-workflow.md` - Complete testing workflow