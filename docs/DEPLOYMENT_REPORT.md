# Nephoran Intent Operator -- Deployment Report

**Date**: 2026-02-15
**Environment**: Single-node Kubernetes on Ubuntu 22.04.5 LTS
**Branch**: `feature/phase1-emergency-hotfix`
**Prepared by**: Performance Engineering

---

## Executive Summary

The Nephoran Intent Operator AI/ML infrastructure has been successfully deployed on a single-node Kubernetes 1.35.1 cluster with GPU acceleration. All core components -- Ollama LLM inference engine, Weaviate vector database, Prometheus/Grafana monitoring, and the RAG service -- are operational and have been benchmarked.

**Key Results**:
- 4 LLM models deployed with generation speeds of 89-291 tokens/sec on RTX 5080
- Weaviate vector search latency under 6ms at p50 across all tested dimensions
- GPU utilization reaches 93% SM / 80% memory bandwidth during inference
- Total system memory footprint: 21.6 Gi out of 32 Gi (67% utilization)
- All 28 pods healthy across 8 namespaces
- End-to-end RAG pipeline operational: natural language intent to structured NetworkIntent

---

## System Architecture

```
+-------------------------------------------------------------------+
|                    Ubuntu 22.04.5 LTS (5.15.0-161)                |
|                    CPU: 8 cores | RAM: 32 Gi | Disk: 248 Gi      |
+-------------------------------------------------------------------+
|                                                                   |
|  +------------------+    Kubernetes 1.35.1 (kubeadm)             |
|  | Control Plane    |    Container Runtime: containerd 2.2.1     |
|  | - API Server     |    CNI: Flannel                            |
|  | - etcd           |    Storage: local-path-provisioner         |
|  | - Scheduler      |                                           |
|  | - Controller Mgr |                                           |
|  +------------------+                                            |
|                                                                   |
|  +-------------------+   +-------------------+                   |
|  | gpu-operator NS   |   | nvidia-dra NS     |                  |
|  | - GPU Operator    |   | - DRA Plugin      |                  |
|  | - DCGM Exporter   |   |   (RTX 5080)      |                  |
|  | - Feature Disc.   |   +-------------------+                   |
|  +-------------------+                                           |
|                                                                   |
|  +-------------------+   +-------------------+                   |
|  | Ollama (systemd)  |   | weaviate NS       |                  |
|  | Models:           |   | - Weaviate v1.34  |                  |
|  |  - Llama3.1 8B    |   |   (HNSW index)    |                  |
|  |  - DeepSeek 16B   |   |   10Gi PVC        |                  |
|  |  - Mistral 12B    |   +-------------------+                   |
|  |  - Qwen2.5 14B    |                                          |
|  | GPU: RTX 5080     |   +-------------------+                   |
|  | VRAM: 16,303 MiB  |   | rag-service NS    |                  |
|  +-------------------+   | - FastAPI app     |                   |
|                          | - Ollama client   |                   |
|  +-------------------+   | - Weaviate client |                   |
|  | monitoring NS     |   +-------------------+                   |
|  | - Prometheus      |                                           |
|  | - Grafana         |                                           |
|  | - DCGM metrics    |                                           |
|  | - Node exporter   |                                           |
|  +-------------------+                                           |
|                                                                   |
+-------------------------------------------------------------------+
          |
     +----+----+
     | RTX 5080 |  PCIe Gen4 x16
     | 16 Gi    |  GDDR7 ~960 GB/s
     | 360W TDP |  Blackwell arch
     +----------+
```

---

## Deployed Components

| Component              | Version      | Namespace           | Status  | Pods | CPU Req | Mem Req | Notes                          |
|------------------------|-------------|---------------------|---------|------|---------|---------|--------------------------------|
| Kubernetes             | 1.35.1      | kube-system         | Running | 7    | 850m    | 640Mi   | Control plane + CoreDNS        |
| Flannel CNI            | latest      | kube-flannel        | Running | 1    | 100m    | 50Mi    | Pod networking                 |
| GPU Operator           | v25.10.1    | gpu-operator        | Running | 6    | 315m    | 420Mi   | NVIDIA device management       |
| NVIDIA DRA Driver      | 25.12.0     | nvidia-dra-driver   | Running | 1    | 100m    | 128Mi   | Dynamic Resource Allocation    |
| Ollama                 | v0.16.1     | host (systemd)      | Running | --   | host    | host    | LLM inference engine           |
| Weaviate               | 1.34.0      | weaviate            | Running | 1    | 500m    | 1Gi     | Vector database                |
| RAG Service            | custom      | rag-service         | Running | 1    | 500m    | 512Mi   | Intent processing pipeline     |
| Prometheus             | v0.89.0     | monitoring          | Running | 2    | 550m    | 2.06Gi  | Metrics collection             |
| Grafana                | latest      | monitoring          | Running | 1    | 100m    | 256Mi   | Dashboards                     |
| kube-state-metrics     | latest      | monitoring          | Running | 1    | 50m     | 64Mi    | K8s state metrics              |
| Node Exporter          | latest      | monitoring          | Running | 1    | 50m     | 32Mi    | System metrics                 |
| Alertmanager           | latest      | monitoring          | Running | 1    | 50m     | 64Mi    | Alert routing                  |
| DCGM Exporter          | latest      | gpu-operator        | Running | 1    | --      | --      | GPU metrics                    |
| local-path-provisioner | latest      | local-path-storage  | Running | 1    | --      | --      | Dynamic PV provisioning        |
| Metrics Server         | latest      | kube-system         | Running | 2    | 200m    | 400Mi   | Resource metrics API           |

**Total Pods**: 28 Running + 2 Completed (validators)
**Total CPU Requests**: 3,415m (42% of 8 cores)
**Total CPU Limits**: 7,900m (98% of 8 cores)
**Total Memory Requests**: 5,366 Mi (16% of 32 Gi)
**Total Memory Limits**: 18,162 Mi (56% of 32 Gi)

---

## Performance Metrics Summary

### GPU Performance (RTX 5080)

| Metric                    | Idle        | Under Load  |
|---------------------------|-------------|-------------|
| Power Draw                | 3.45W       | 250W        |
| Temperature               | 38C         | 51C         |
| SM Utilization            | 0%          | 93%         |
| Memory Bandwidth Util.    | 0%          | 80%         |
| Graphics Clock            | 180 MHz     | 2,820 MHz   |
| Memory Clock              | 405 MHz     | 14,801 MHz  |

Thermal headroom: 39C below throttle point. No throttling events observed.

### LLM Inference Performance

| Model                    | Gen tok/s | Prompt tok/s | TTFT (ms) | VRAM (MiB) |
|--------------------------|-----------|--------------|-----------|------------|
| DeepSeek Coder V2 16B   | **291**   | 5,359        | 90-103    | 14,708     |
| Llama 3.1 8B            | 143       | **6,195**    | 118-139   | 6,884      |
| Mistral Nemo 12B        | 95        | 4,216        | 120-154   | 9,962      |
| Qwen 2.5 14B            | 89        | 3,905        | 108-156   | 10,554     |

Concurrent scaling (Llama 3.1 8B):
- 1 request: 143 tok/s
- 2 concurrent: 113 tok/s per request (226 total)
- 4 concurrent: 96 tok/s per request (384 total)

### Weaviate Vector Database

| Dimension | Insert Rate (obj/s) | Query p50 (ms) | Query p95 (ms) | Query p99 (ms) |
|-----------|---------------------|----------------|----------------|----------------|
| 128       | 6,056               | 2.08           | 2.69           | 3.12           |
| 384       | 2,900               | 2.73           | 3.33           | 3.81           |
| 768       | 1,612               | 3.64           | 4.82           | 9.24           |
| 1536      | 870                 | 5.30           | 7.32           | 8.30           |

### Kubernetes Resource Usage (Measured)

| Resource       | Capacity   | Requested  | Actual Used | Utilization |
|----------------|------------|------------|-------------|-------------|
| CPU            | 8 cores    | 3,415m     | 674m        | 8.4%        |
| Memory (K8s)   | 32 Gi      | 5,366 Mi   | 21,624 Mi   | 67%*        |
| Disk           | 248 Gi     | --         | 209 Gi      | 85%         |
| GPU VRAM       | 16,303 MiB | --         | 6,884 MiB   | 42%**       |
| PVC Storage    | 37 Gi      | --         | 55 Mi       | <1%         |

*Memory includes Ollama model cache (host process, not K8s-managed).
**VRAM usage varies by active model (42-90%).

### Pod Startup Times

| Component                | Startup Time |
|--------------------------|-------------|
| kube-proxy               | 1s          |
| Node Exporter            | 5s          |
| local-path-provisioner   | 6s          |
| DCGM Exporter            | 13s         |
| Flannel                  | 11s         |
| Prometheus Operator      | 14s         |
| Prometheus Server        | 16s         |
| kube-scheduler           | 17s         |
| GPU Feature Discovery    | 18s         |
| DRA Plugin               | 20s         |
| RAG Service              | 21s         |
| CoreDNS                  | 28-39s      |
| Alertmanager             | 39s         |
| GPU Operator             | 45s         |
| Grafana                  | 47s         |
| Weaviate                 | **68s**     |

---

## Persistent Volume Claims

| PVC                          | Namespace   | Size  | Actual Usage | Purpose            |
|------------------------------|-------------|-------|-------------|---------------------|
| weaviate-data-weaviate-0     | weaviate    | 10 Gi | 360 KB      | Vector data storage |
| prometheus-grafana           | monitoring  | 5 Gi  | 50 MB       | Dashboard storage   |
| prometheus-db                | monitoring  | 20 Gi | 5.1 MB      | Time-series data    |
| alertmanager-db              | monitoring  | 2 Gi  | 8 KB        | Alert state         |
| **Total**                    |             | **37 Gi** | **55 Mi** | --                |

---

## GPU Utilization Patterns

```
GPU State Machine:

  [IDLE]                    [INFERENCE]                 [COOL-DOWN]
  P8 state                  P0 state                    P0 -> P8
  3W, 38C                   250W, 51C                   87W -> 3W
  180/405 MHz               2820/14801 MHz              Clocks ramp down
       |                         |                           |
       +--- Request arrives ---->+                           |
                                 +--- Generation done ------>+
                                                             |
                                 +<--- Next request ---------+
                                 |
                                 +--- (or idle timeout) ---> [IDLE]
```

**Typical Inference Cycle** (512 tokens, Llama 3.1 8B):
- 0-0.5s: Prompt evaluation (GPU ramps to full power)
- 0.5-4s: Token generation (sustained 93% SM, 80% memory BW)
- 4-5s: Cool-down (GPU returns to idle within ~2s)

**VRAM Allocation Strategy**:
- Ollama loads one model at a time (largest first if concurrent)
- Model swap overhead: ~2 seconds for cold load
- Recommendation: Pin Llama 3.1 8B as default (6.9 Gi), leaving 9.4 Gi headroom

---

## Network Architecture

| Service            | Type      | Cluster IP      | Ports           |
|--------------------|-----------|-----------------|-----------------|
| weaviate           | ClusterIP | 10.108.49.161   | 80/TCP          |
| weaviate-grpc      | ClusterIP | 10.99.240.28    | 50051/TCP       |
| weaviate-metrics   | ClusterIP | 10.105.10.142   | 2112/TCP        |
| rag-service        | ClusterIP | 10.110.166.224  | 8000/TCP        |
| prometheus-grafana | ClusterIP | (monitoring NS) | 80/TCP          |
| prometheus-server  | ClusterIP | (monitoring NS) | 9090/TCP        |
| ollama             | Host      | localhost       | 11434/TCP       |

---

## Known Limitations

### Infrastructure
1. **Single-node cluster**: No high availability; single point of failure for all components
2. **Disk usage at 85%**: Only 40 Gi free on the root partition; model downloads and data growth could cause disk pressure
3. **Ollama runs outside Kubernetes**: Running as a systemd service, not managed by K8s scheduler or subject to resource limits
4. **PCIe Gen 1 at idle**: PCIe link drops to Gen 1 when idle; initial model load from disk may be slower than Gen 4 speeds

### Performance
5. **Model swap latency**: Switching between LLM models costs ~2 seconds for VRAM load
6. **Concurrent request degradation**: 33% per-request throughput loss at 4 concurrent requests
7. **Weaviate p99 spike**: Occasional latency spikes at 768 dimensions (9.24ms p99 vs 4.82ms p95), likely GC pauses
8. **No GPU time-slicing**: Only one model can use the GPU at a time; no MPS or MIG support on consumer GPU

### Security
9. **Weaviate API key in plaintext**: API key stored in ConfigMap, not encrypted at rest
10. **No TLS between services**: All intra-cluster communication is plaintext HTTP
11. **Single admin user**: Weaviate has one admin user with full access

### Data
12. **Minimal seed data**: Only 2 objects in TelecomKnowledge collection; RAG quality depends on knowledge base size
13. **No backup strategy**: No automated backups for Weaviate data or Prometheus metrics

---

## Recommendations for Optimization

### Immediate (Week 1)

| Priority | Action                               | Impact | Effort |
|----------|--------------------------------------|--------|--------|
| HIGH     | Pin Llama 3.1 8B as default model    | Eliminates 2s model swap latency | Low |
| HIGH     | Set Ollama OLLAMA_KEEP_ALIVE=24h     | Prevents model unloading | Low |
| HIGH     | Free disk space (remove unused data) | Prevent disk pressure at 85% | Low |
| MEDIUM   | Add Weaviate readiness probe tuning  | Reduce 68s startup time | Low |
| MEDIUM   | Enable Prometheus remote write       | Prevent metrics loss on restart | Medium |

### Short-term (Month 1)

| Priority | Action                               | Impact | Effort |
|----------|--------------------------------------|--------|--------|
| HIGH     | Populate knowledge base (1000+ docs) | Dramatically improve RAG quality | Medium |
| HIGH     | Move Ollama into K8s DaemonSet       | Unified resource management | Medium |
| MEDIUM   | Enable TLS for Weaviate/RAG service  | Security hardening | Medium |
| MEDIUM   | Add automated Weaviate backups       | Data protection | Low |
| MEDIUM   | Implement connection pooling for RAG  | Better concurrent handling | Medium |

### Long-term (Quarter 1)

| Priority | Action                                | Impact | Effort |
|----------|---------------------------------------|--------|--------|
| HIGH     | Multi-node cluster with HA            | Eliminate single point of failure | High |
| HIGH     | Dedicated GPU node for inference      | Isolate GPU workloads | High |
| MEDIUM   | Add Weaviate replication (2 replicas) | Data redundancy | Medium |
| MEDIUM   | Implement model caching/preloading    | Sub-100ms TTFT for all models | Medium |
| LOW      | Explore vLLM for better batching      | Higher concurrent throughput | High |

---

## Monitoring and Observability

### Active Metrics Collection
- **Prometheus**: Scraping 16 targets at 15s intervals
- **DCGM Exporter**: GPU utilization, temperature, power, memory metrics
- **Node Exporter**: CPU, memory, disk, network system metrics
- **kube-state-metrics**: Pod, deployment, service Kubernetes state
- **Weaviate metrics**: Available at port 2112 (recently deployed)

### Grafana Dashboards
- Accessible via port-forward to monitoring/prometheus-grafana service
- Pre-configured with Prometheus data source
- GPU dashboard available via DCGM metrics

### Alerting
- Alertmanager deployed with 2 Gi PVC
- Default alerts from kube-prometheus-stack active
- Custom alerts recommended for:
  - GPU temperature > 80C
  - VRAM usage > 90%
  - LLM inference latency > 5s
  - Weaviate query p99 > 50ms
  - Disk usage > 90%

---

## Benchmark Data Inventory

All raw benchmark data is stored in `/benchmarks/`:

| File                          | Description                           |
|-------------------------------|---------------------------------------|
| `gpu-performance.md`          | GPU hardware and utilization analysis |
| `llm-inference.md`           | LLM model benchmarks and comparisons |
| `weaviate-performance.md`    | Vector database insert/query benchmarks|
| `llm-raw-results.json`      | Raw JSON data: all LLM measurements  |
| `weaviate-raw-results.json`  | Raw JSON data: Weaviate measurements  |
| `llm-benchmark.sh`           | LLM benchmark automation script       |
| `llm-concurrent-benchmark.sh`| Concurrent request test script        |
| `weaviate-benchmark.py`      | Weaviate benchmark automation script  |

---

## Deployment Verification Checklist

| Check                              | Status | Details                                    |
|------------------------------------|--------|--------------------------------------------|
| Kubernetes cluster healthy         | PASS   | 1 node Ready, v1.35.1                     |
| GPU detected and allocated         | PASS   | RTX 5080, DRA driver operational           |
| Ollama serving models              | PASS   | 4 models loaded, API responding            |
| Weaviate accepting requests        | PASS   | v1.34.0, schema created, queries working   |
| RAG service healthy                | PASS   | Health/ready endpoints responding           |
| Prometheus scraping metrics        | PASS   | 16 targets UP                              |
| Grafana accessible                 | PASS   | Dashboards operational                     |
| PVCs bound                         | PASS   | 4/4 PVCs bound to local-path PVs          |
| LLM inference functional           | PASS   | 89-291 tok/s across models                |
| Vector search functional           | PASS   | Sub-10ms p99 latency                      |
| End-to-end RAG pipeline            | PASS   | Intent processing operational              |
| GPU metrics flowing                | PASS   | DCGM exporter scraping successfully        |

**Overall Status: ALL SYSTEMS OPERATIONAL**

---

## Next Steps

1. **Knowledge Base Population**: Ingest O-RAN, 3GPP, and Nephio documentation into Weaviate
2. **Intent Pipeline Testing**: Run end-to-end intent processing scenarios with realistic telecom intents
3. **Load Testing**: Simulate production traffic patterns with concurrent users
4. **CI/CD Integration**: Connect RAG service deployment to the existing CI pipeline
5. **Production Hardening**: TLS, RBAC, backup automation, multi-node migration plan

---

*Report generated: 2026-02-15T12:30:00+00:00*
*All measurements are real, taken from the live system using the scripts in `benchmarks/`.*
*For methodology details, see individual benchmark documents.*
