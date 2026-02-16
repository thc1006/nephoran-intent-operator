# Nephoran Intent Operator - E2E Testing Guide

This document describes how to run the end-to-end test suite, interpret results,
and troubleshoot common issues.

## Overview

The E2E test suite validates the full Nephoran Intent Operator deployment stack:

| Test | Script | What it validates |
|------|--------|-------------------|
| Intent Lifecycle | `test-intent-lifecycle.sh` | NetworkIntent CR create / read / update / delete |
| RAG Pipeline | `test-rag-pipeline.sh` | RAG service, Weaviate, Ollama, intent processing |
| GPU Allocation | `test-gpu-allocation.sh` | GPU Operator, DRA, nvidia-smi inside pods |
| Monitoring | `test-monitoring.sh` | Prometheus, Grafana, DCGM GPU metrics |

All scripts live under `tests/e2e/bash/`.

## Prerequisites

### Required tools

- `kubectl` configured with cluster access
- `curl` for HTTP requests
- `python3` for JSON parsing (optional but recommended)
- `bash` 4.0+ (for associative arrays in the master runner)

### Cluster requirements

| Component | Required for |
|-----------|-------------|
| Kubernetes 1.35+ | All tests |
| NetworkIntent CRD deployed | intent-lifecycle |
| RAG service (FastAPI, port 8000) | rag-pipeline |
| Weaviate (port 8080) | rag-pipeline |
| Ollama with loaded models (port 11434) | rag-pipeline |
| GPU Operator + DRA driver | gpu-allocation |
| Prometheus (port 9090) | monitoring |
| Grafana (port 3000) | monitoring |

If a service is not available the test will mark affected assertions as **SKIPPED**
rather than **FAILED**, allowing partial validation.

## Quick Start

Run the entire suite:

```bash
cd tests/e2e/bash
./run-all-tests.sh
```

Run a single test:

```bash
./test-intent-lifecycle.sh
./test-rag-pipeline.sh
./test-gpu-allocation.sh
./test-monitoring.sh
```

## Configuration

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `E2E_NAMESPACE` | `nephoran-e2e-test` | Kubernetes namespace for test resources |
| `E2E_TIMEOUT` | `120`/`180` | Timeout in seconds per test |
| `RAG_URL` | `http://localhost:8000` | RAG service base URL |
| `WEAVIATE_URL` | `http://localhost:8080` | Weaviate base URL |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API base URL |
| `PROMETHEUS_URL` | `http://localhost:9090` | Prometheus API base URL |
| `GRAFANA_URL` | `http://localhost:3000` | Grafana base URL |
| `GPU_DEVICE_CLASS` | `gpu.nvidia.com` | DRA DeviceClass name |

### CLI flags for individual tests

Each script supports `--help` for full usage. Common flags:

```
--namespace <ns>    Override Kubernetes namespace
--timeout <sec>     Override timeout
--help / -h         Show usage
```

### Master runner flags

```
--skip <tests>       Comma-separated tests to skip
--only <tests>       Only run specified tests
--report-dir <dir>   Output directory for reports (default: ./reports)
--timeout <sec>      Per-test timeout
```

Examples:

```bash
# Skip GPU and monitoring tests
./run-all-tests.sh --skip gpu-allocation,monitoring

# Only run intent lifecycle
./run-all-tests.sh --only intent-lifecycle

# Custom timeout and report directory
./run-all-tests.sh --timeout 300 --report-dir /tmp/e2e-reports
```

## Setting Up Port Forwarding

If services are running inside the cluster, set up port forwarding before running tests:

```bash
# RAG service
kubectl port-forward svc/rag-service -n nephoran-system 8000:8000 &

# Weaviate
kubectl port-forward svc/weaviate -n weaviate 8080:8080 &

# Ollama
kubectl port-forward svc/ollama -n ollama 11434:11434 &

# Prometheus
kubectl port-forward svc/prometheus-server -n monitoring 9090:9090 &

# Grafana
kubectl port-forward svc/grafana -n monitoring 3000:3000 &
```

Alternatively, use NodePort or LoadBalancer URLs directly:

```bash
export RAG_URL=http://<node-ip>:30800
export PROMETHEUS_URL=http://<node-ip>:30090
./run-all-tests.sh
```

## Expected Outputs

### Successful run

Each test prints a summary block:

```
=== TEST SUMMARY ===

  Total tests:  12
  Passed:       10
  Failed:       0
  Skipped:      2
  Duration:     45s

RESULT: PASSED
```

The master runner produces a combined table:

```
  TEST                      RESULT     DURATION
  ----                      ------     --------
  intent-lifecycle          PASSED     32s
  rag-pipeline              PASSED     48s
  gpu-allocation            PASSED     67s
  monitoring                PASSED     21s
```

### Reports

After running the master suite, the `reports/` directory contains:

| File | Format | Purpose |
|------|--------|---------|
| `e2e-summary.txt` | Plain text | Human-readable results table |
| `e2e-summary.json` | JSON | Machine-parseable results for CI integration |
| `<test-name>.log` | Plain text | Full console output for each test |

## Test Details

### test-intent-lifecycle.sh

**Phases:**

1. **Prerequisites** -- checks CRD registration and cluster access
2. **Create** -- applies a sample `NetworkIntent` CR and verifies spec fields
3. **Reconciliation** -- waits for `status.phase` to be populated by the controller
4. **List/Describe** -- lists resources and validates YAML structure
5. **Update** -- patches the intent and checks generation increment
6. **Negative test** -- submits an invalid CR and expects rejection
7. **Delete** -- removes the CR and confirms it no longer exists

### test-rag-pipeline.sh

**Phases:**

1. **Reachability** -- `/healthz`, `/readyz`, `/health` endpoints
2. **Weaviate** -- schema, metadata, version
3. **Ollama inference** -- list models, run simple generation, measure latency
4. **Stats** -- `/stats` endpoint returns valid config
5. **RAG flow** -- POST to `/process` with a sample intent, validate JSON response
6. **Legacy compat** -- `/process_intent` endpoint works
7. **OpenAPI** -- `/docs` and `/openapi.json` available
8. **Timing** -- reports latency measurements

### test-gpu-allocation.sh

**Phases:**

1. **Prerequisites** -- cluster access, namespace creation
2. **GPU Operator** -- pods running in `gpu-operator` namespace
3. **DRA API** -- `ResourceClaim` and `DeviceClass` API availability
4. **ResourceClaim** -- creates a DRA claim (falls back to device plugin)
5. **Pod creation** -- creates pod referencing GPU
6. **Wait** -- polls until pod is Running
7. **nvidia-smi** -- executes inside pod, extracts GPU info
8. **Cleanup** -- deletes pod and claim

### test-monitoring.sh

**Phases:**

1. **Kubernetes resources** -- monitoring namespace, pods, CRDs
2. **Prometheus connectivity** -- `/-/ready`, `/-/healthy`
3. **Scrape targets** -- active targets and health distribution
4. **PromQL queries** -- `kube_pod_info`, `up`, CPU, memory metrics
5. **DCGM GPU metrics** -- utilization, temperature, power, framebuffer
6. **Grafana** -- health, dashboards, datasources
7. **Alertmanager** -- connectivity and active alerts (optional)
8. **Nephoran metrics** -- controller reconcile counters, workqueue depth

## Troubleshooting

### "Cannot reach Kubernetes cluster"

Verify your kubeconfig is set:

```bash
kubectl cluster-info
kubectl config current-context
```

### "NetworkIntent CRD not found"

Apply the CRD manually:

```bash
kubectl apply -f deployments/crds/nephoran.com_networkintents.yaml
```

### "RAG /readyz returns 503"

The RAG pipeline requires either an OpenAI API key or a running Ollama instance.
Check environment variables:

```bash
kubectl exec -n nephoran-system deployment/rag-service -- env | grep -E 'LLM_PROVIDER|OLLAMA'
```

Ensure Ollama has models loaded:

```bash
curl http://localhost:11434/api/tags
```

### "GPU test pod stuck in Pending"

Check if GPU resources are available:

```bash
kubectl describe node | grep -A5 "nvidia.com/gpu"
kubectl get deviceclass
kubectl get resourceclaim -n nephoran-gpu-test
```

Look at pod events:

```bash
kubectl describe pod <pod-name> -n nephoran-gpu-test
```

### "No DCGM GPU metrics in Prometheus"

Verify DCGM exporter is running:

```bash
kubectl get pods --all-namespaces | grep dcgm
```

Check if a ServiceMonitor exists:

```bash
kubectl get servicemonitor --all-namespaces | grep dcgm
```

### "Prometheus targets are DOWN"

Check Prometheus target status page:

```bash
curl http://localhost:9090/api/v1/targets | python3 -m json.tool | grep -A5 '"health": "down"'
```

### "Grafana returns 401"

Default credentials for kube-prometheus-stack are typically `admin:prom-operator`.
Override with environment:

```bash
export GRAFANA_URL="http://admin:prom-operator@localhost:3000"
```

## CI Integration

### GitHub Actions

```yaml
- name: Run E2E Tests
  run: |
    cd tests/e2e/bash
    ./run-all-tests.sh --report-dir ${{ runner.temp }}/e2e-reports

- name: Upload E2E Reports
  if: always()
  uses: actions/upload-artifact@v4
  with:
    name: e2e-reports
    path: ${{ runner.temp }}/e2e-reports/
```

### Reading JSON results in CI

```bash
jq '.summary' reports/e2e-summary.json
# { "total": 4, "passed": 4, "failed": 0, "timeout": 0, "skipped": 0 }

# Fail the CI step if any test failed
jq -e '.overall_result == "PASSED"' reports/e2e-summary.json
```

## Idempotency

All test scripts are designed to be idempotent:

- Resources are created with unique names (timestamp-based suffixes)
- Cleanup handlers run via `trap EXIT INT TERM`
- Namespaces are only deleted if they match the test pattern
- Tests tolerate missing optional services (SKIP instead of FAIL)
