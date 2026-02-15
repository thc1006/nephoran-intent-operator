# Monitoring Stack Deployment Guide

## Overview

Prometheus + Grafana monitoring stack for the Nephoran Intent Operator platform,
deployed via the `kube-prometheus-stack` Helm chart. Provides full observability
for GPU (DCGM), Kubernetes cluster, Weaviate vector DB, and future RAG/Operator services.

## Architecture

```
+------------------+     +------------------+     +------------------+
| DCGM Exporter    |     | Weaviate         |     | Node Exporter    |
| (GPU metrics)    |     | (Vector DB)      |     | (Host metrics)   |
| port: 9400       |     | port: 2112       |     | port: 9100       |
+--------+---------+     +--------+---------+     +--------+---------+
         |                         |                         |
         +----------+--------------+-------------+-----------+
                    |                             |
         +----------v-----------+     +-----------v----------+
         | Prometheus           |     | kube-state-metrics   |
         | (Scrape & Store)     |<----| (K8s object metrics) |
         | port: 9090           |     +----------------------+
         +----------+-----------+
                    |
         +----------v-----------+
         | Grafana              |
         | (Visualization)      |
         | NodePort: 30300      |
         +----------------------+
```

## Access Information

| Service       | URL                             | Credentials           |
|---------------|----------------------------------|-----------------------|
| Grafana       | http://192.168.10.65:30300       | admin / NephoranMonitor2026! |
| Prometheus    | ClusterIP (port-forward: 9090)  | N/A                   |
| Alertmanager  | ClusterIP (port-forward: 9093)  | N/A                   |

### Port-Forward Commands

```bash
# Prometheus UI
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

# Alertmanager UI
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-alertmanager 9093:9093
```

## Deployed Components

| Component            | Pods | Storage | Notes                          |
|----------------------|------|---------|--------------------------------|
| Prometheus           | 1    | 20Gi PVC | 7d retention, 10GB max        |
| Grafana              | 1    | 5Gi PVC  | NodePort 30300                |
| Alertmanager         | 1    | 2Gi PVC  | Persistent alert state        |
| Node Exporter        | 1    | None     | DaemonSet on all nodes        |
| kube-state-metrics   | 1    | None     | K8s object state              |
| Prometheus Operator  | 1    | None     | Manages CRDs                  |

## Scrape Targets (16/16 UP)

| Target                    | Health | Namespace    | Port  |
|---------------------------|--------|-------------|-------|
| apiserver                 | UP     | default     | 6443  |
| coredns (x2)              | UP     | kube-system | 9153  |
| kube-state-metrics        | UP     | monitoring  | 8080  |
| kubelet (metrics)         | UP     | default     | 10250 |
| kubelet (cadvisor)        | UP     | default     | 10250 |
| kubelet (probes)          | UP     | default     | 10250 |
| node-exporter             | UP     | monitoring  | 9100  |
| nvidia-dcgm-exporter      | UP     | gpu-operator| 9400  |
| weaviate-metrics          | UP     | weaviate    | 2112  |
| prometheus-grafana        | UP     | monitoring  | 3000  |
| alertmanager (x2)         | UP     | monitoring  | 9093  |
| prometheus-operator       | UP     | monitoring  | 10250 |
| prometheus (x2)           | UP     | monitoring  | 9090  |

## Grafana Dashboards (26 total)

### Custom Nephoran Dashboards
- **NVIDIA GPU Monitoring (DCGM)** - GPU utilization, memory, temp, power, PCIe bandwidth
- **Nephoran Platform Overview** - Cluster health, GPU, Weaviate, node resources

### Built-in Kubernetes Dashboards
- Kubernetes / Compute Resources (Cluster, Namespace, Node, Pod, Workload)
- Kubernetes / Networking (Cluster, Namespace, Pod, Workload)
- Kubernetes / Persistent Volumes
- Kubernetes / Kubelet
- Node Exporter / Nodes
- CoreDNS
- Prometheus / Overview
- Alertmanager / Overview

## Available Metrics

| Category          | Count | Examples                                          |
|-------------------|-------|---------------------------------------------------|
| DCGM (GPU)        | 20    | DCGM_FI_DEV_GPU_UTIL, DCGM_FI_DEV_GPU_TEMP      |
| Weaviate          | 165   | batch_objects_processed_total, concurrent_queries  |
| Node Exporter     | 256   | node_cpu_seconds_total, node_memory_MemAvailable  |
| Kubernetes        | 157   | kube_pod_status_phase, kube_node_status_condition |
| Total             | 2104  | All registered metric names                       |

## Example PromQL Queries

### GPU Queries
```promql
# GPU utilization percentage
DCGM_FI_DEV_GPU_UTIL

# GPU temperature
DCGM_FI_DEV_GPU_TEMP

# GPU memory usage percentage
DCGM_FI_DEV_FB_USED / (DCGM_FI_DEV_FB_USED + DCGM_FI_DEV_FB_FREE) * 100

# GPU power consumption
DCGM_FI_DEV_POWER_USAGE

# Tensor core utilization (inference workloads)
DCGM_FI_PROF_PIPE_TENSOR_ACTIVE

# PCIe bandwidth (TX)
rate(DCGM_FI_PROF_PCIE_TX_BYTES[5m])
```

### Weaviate Queries
```promql
# Active concurrent queries
sum(concurrent_queries_count)

# Batch processing rate (objects/sec)
rate(batch_objects_processed_total[5m])

# Batch bytes processed rate
rate(batch_objects_processed_bytes[5m])
```

### Cluster Queries
```promql
# CPU usage (all cores, percentage)
1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m]))

# Memory usage
node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes

# Pod count by namespace
sum by (namespace) (kube_pod_status_phase{phase="Running"})

# Disk I/O
rate(node_disk_read_bytes_total[5m])
rate(node_disk_written_bytes_total[5m])
```

## Alert Rules

### GPU Alerts
| Alert                  | Condition                              | Severity |
|------------------------|----------------------------------------|----------|
| GPUHighUtilization     | GPU util > 95% for 10m                | warning  |
| GPUHighTemperature     | GPU temp > 85C for 5m                 | critical |
| GPUMemoryAlmostFull    | GPU mem > 90% for 5m                  | warning  |
| GPUHighPowerUsage      | GPU power > 300W for 10m              | warning  |

### Weaviate Alerts
| Alert                        | Condition                        | Severity |
|------------------------------|----------------------------------|----------|
| WeaviateHighConcurrentQueries| Concurrent queries > 50 for 5m  | warning  |

## Adding New ServiceMonitors

To monitor a new service, create a ServiceMonitor YAML:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-service
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: my-service          # must match the Service labels
  namespaceSelector:
    matchNames:
      - my-namespace           # namespace where Service lives
  endpoints:
    - port: metrics            # must match Service port name
      interval: 30s
      path: /metrics
```

Then apply:
```bash
kubectl apply -f my-servicemonitor.yaml
```

Prometheus will auto-discover the new target within 30 seconds.

## Files in This Directory

```
monitoring/
  kube-prometheus-stack-values.yaml     # Helm values for kube-prometheus-stack
  DEPLOYMENT.md                         # This file
  servicemonitors/
    dcgm-exporter-servicemonitor.yaml   # NVIDIA GPU metrics
    weaviate-servicemonitor.yaml        # Weaviate vector DB metrics
    weaviate-metrics-service.yaml       # K8s Service exposing Weaviate port 2112
    rag-service-servicemonitor.yaml     # RAG FastAPI (apply when deployed)
    intent-operator-servicemonitor.yaml # Intent Operator (apply when deployed)
  grafana-dashboards/
    gpu-monitoring-configmap.yaml       # NVIDIA GPU dashboard (auto-loaded)
    nephoran-platform-overview-configmap.yaml  # Platform overview (auto-loaded)
  prometheus-rules/
    gpu-alerts.yaml                     # GPU and Weaviate alert rules
```

## Helm Management

```bash
# Check release status
helm list -n monitoring

# Upgrade with new values
helm upgrade prometheus prometheus-community/kube-prometheus-stack \
  -n monitoring \
  -f monitoring/kube-prometheus-stack-values.yaml

# Uninstall (WARNING: destroys all data)
helm uninstall prometheus -n monitoring
```
