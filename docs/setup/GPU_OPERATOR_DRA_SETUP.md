# NVIDIA GPU Operator with DRA (Dynamic Resource Allocation) Setup

## Environment

| Component | Version |
|-----------|---------|
| Kubernetes | v1.35.1 |
| containerd | 2.2.1 |
| GPU | NVIDIA GeForce RTX 5080 (16GB VRAM, Blackwell) |
| NVIDIA Driver | 580.126.09 |
| CUDA | 13.0 (runtime 13.1.80) |
| NVIDIA Container Toolkit | 1.18.2-1 |
| GPU Operator | v25.10.1 |
| DRA Driver for GPUs | 25.12.0 |

## Prerequisites

1. **Kubernetes 1.34+** with DRA feature gate enabled (enabled by default in 1.34+).
2. **NVIDIA GPU driver** installed on the host.
3. **NVIDIA Container Toolkit** installed with CDI spec generated at `/etc/cdi/nvidia.yaml`.
4. **containerd** configured with `enable_cdi = true` and `cdi_spec_dirs`.
5. **Kubelet** feature gate `DynamicResourceAllocation=true` (default in 1.34+).

## Architecture

```
DRA API (resource.k8s.io/v1)
    |
    +-- DeviceClass (gpu.nvidia.com)  -- Selects NVIDIA GPU devices
    +-- ResourceSlice                 -- Published by DRA kubelet plugin (GPU inventory)
    +-- ResourceClaim                 -- User request for GPU devices
    +-- ResourceClaimTemplate         -- Template for ephemeral GPU claims
    |
DRA Kubelet Plugin (nvidia-dra-driver-gpu)
    |
    +-- Discovers GPUs via NVML
    +-- Publishes ResourceSlices
    +-- Prepares/unprepares GPU devices for pods via CDI
    |
GPU Operator (gpu-operator)
    |
    +-- NFD (Node Feature Discovery) -- Labels nodes with GPU attributes
    +-- GPU Feature Discovery         -- Exposes GPU properties
    +-- DCGM Exporter                -- GPU metrics for Prometheus
    +-- Operator Validator           -- Validates GPU stack health
```

## Installation Steps

### Step 1: Add NVIDIA Helm Repository

```bash
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update nvidia
```

### Step 2: Label Nodes

```bash
# Required for DRA kubelet plugin scheduling
kubectl label node <NODE_NAME> nvidia.com/dra-kubelet-plugin=true

# Required for GPU Feature Discovery affinity (if NFD has not yet run)
kubectl label node <NODE_NAME> nvidia.com/gpu.present=true
kubectl label node <NODE_NAME> feature.node.kubernetes.io/pci-10de.present=true
```

### Step 3: Install GPU Operator

Since the driver and container toolkit are pre-installed, disable those components.
Also disable the device plugin since DRA replaces it.

```bash
helm install gpu-operator nvidia/gpu-operator \
    --version=v25.10.1 \
    --create-namespace \
    --namespace gpu-operator \
    --set driver.enabled=false \
    --set toolkit.enabled=false \
    --set devicePlugin.enabled=false \
    --set operator.runtimeClass=nvidia \
    --set cdi.enabled=true \
    --set nfd.enabled=true \
    --wait \
    --timeout 5m
```

### Step 4: Install DRA Driver for GPUs

Create values file (`dra-values.yaml`):

```yaml
nvidiaDriverRoot: /
gpuResourcesEnabledOverride: true

image:
  pullPolicy: IfNotPresent

resources:
  gpus:
    enabled: true
  computeDomains:
    enabled: false

logVerbosity: "4"

controller:
  priorityClassName: "system-node-critical"
  tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
  containers:
    computeDomain:
      resources:
        limits:
          cpu: 200m
          memory: 256Mi
        requests:
          cpu: 50m
          memory: 64Mi

kubeletPlugin:
  priorityClassName: "system-node-critical"
  nodeSelector:
    nvidia.com/dra-kubelet-plugin: "true"
  containers:
    gpus:
      resources:
        limits:
          cpu: 300m
          memory: 512Mi
        requests:
          cpu: 100m
          memory: 128Mi
    computeDomains:
      resources:
        limits:
          cpu: 200m
          memory: 256Mi
        requests:
          cpu: 50m
          memory: 64Mi
    init:
      resources:
        limits:
          cpu: 100m
          memory: 128Mi
        requests:
          cpu: 50m
          memory: 64Mi
```

Install:

```bash
helm install nvidia-dra-driver-gpu nvidia/nvidia-dra-driver-gpu \
    --version="25.12.0" \
    --create-namespace \
    --namespace nvidia-dra-driver-gpu \
    -f dra-values.yaml \
    --wait \
    --timeout 5m
```

## Verification

### Check DRA Resources

```bash
# DeviceClasses should include gpu.nvidia.com
kubectl get deviceclasses

# ResourceSlice should show GPU with attributes
kubectl get resourceslices -o wide

# Inspect GPU details in the ResourceSlice
kubectl get resourceslice -o yaml
```

Expected DeviceClasses:
- `gpu.nvidia.com` - Full GPU allocation
- `mig.nvidia.com` - MIG (Multi-Instance GPU) partitions
- `vfio.gpu.nvidia.com` - VFIO passthrough

### Check Pods

```bash
kubectl get pods -n gpu-operator
kubectl get pods -n nvidia-dra-driver-gpu
```

### Check Node Labels

```bash
kubectl get node <NODE_NAME> -o json | jq '.metadata.labels | to_entries[] | select(.key | contains("nvidia"))'
```

## Usage Examples

### Ephemeral GPU Claim (via ResourceClaimTemplate)

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: gpu-claim-template
spec:
  spec:
    devices:
      requests:
      - name: gpu
        exactly:
          deviceClassName: gpu.nvidia.com
          count: 1
---
apiVersion: v1
kind: Pod
metadata:
  name: my-gpu-workload
spec:
  restartPolicy: Never
  containers:
  - name: cuda-app
    image: nvcr.io/nvidia/cuda:13.0.1-base-ubuntu24.04
    command: ["nvidia-smi"]
    resources:
      claims:
      - name: gpu
  resourceClaims:
  - name: gpu
    resourceClaimTemplateName: gpu-claim-template
```

### Persistent GPU Claim (via ResourceClaim)

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: my-gpu
spec:
  devices:
    requests:
    - name: gpu
      exactly:
        deviceClassName: gpu.nvidia.com
        count: 1
---
apiVersion: v1
kind: Pod
metadata:
  name: my-gpu-workload
spec:
  restartPolicy: Never
  containers:
  - name: cuda-app
    image: nvcr.io/nvidia/cuda:13.0.1-base-ubuntu24.04
    command: ["nvidia-smi"]
    resources:
      claims:
      - name: gpu
  resourceClaims:
  - name: gpu
    resourceClaimName: my-gpu
```

## Key Configuration Notes

### nvidiaDriverRoot

- Set to `/` when the GPU driver is installed directly on the host.
- Set to `/run/nvidia/driver` when the driver is managed by the GPU Operator driver container.

### gpuResourcesEnabledOverride

- Set to `true` to force GPU resource publishing even when using a host-provided driver.
- Without this, the DRA driver may not publish ResourceSlices.

### computeDomains.enabled

- Set to `false` for single-node or non-NVLink setups.
- Enable for multi-GPU NVLink domains (e.g., DGX systems).

### devicePlugin.enabled (GPU Operator)

- Set to `false` when using DRA. The traditional device plugin (`nvidia.com/gpu` resource)
  conflicts with DRA-based allocation.

## Troubleshooting

### No ResourceSlice created

1. Check the DRA kubelet plugin pod logs:
   ```bash
   kubectl logs -n nvidia-dra-driver-gpu -l app.kubernetes.io/component=kubelet-plugin
   ```
2. Verify node labels include `nvidia.com/dra-kubelet-plugin=true`.
3. Check `gpuResourcesEnabledOverride=true` is set.

### Pod stuck in Pending

1. Check if the ResourceClaim shows `allocated`:
   ```bash
   kubectl get resourceclaims
   ```
2. Verify the DeviceClass exists:
   ```bash
   kubectl get deviceclass gpu.nvidia.com
   ```
3. Check scheduler logs for DRA allocation failures.

### nvidia-smi not found inside container

Ensure the CDI spec at `/etc/cdi/nvidia.yaml` includes the correct device nodes and hooks.
Verify containerd has `enable_cdi = true` in its configuration.

## Uninstall

```bash
helm uninstall nvidia-dra-driver-gpu -n nvidia-dra-driver-gpu
helm uninstall gpu-operator -n gpu-operator
kubectl delete namespace nvidia-dra-driver-gpu gpu-operator
kubectl delete deviceclass gpu.nvidia.com mig.nvidia.com vfio.gpu.nvidia.com
```
