# Kubernetes Version Selection Note

**Document Type**: Technical Decision Record
**Date**: 2026-02-16
**Status**: Active

---

## Target Version: Kubernetes 1.32.3

### Rationale

The Nephoran Intent Operator SDD targets **Kubernetes 1.32.3** as the baseline deployment version for the following reasons:

1. **Proven Stability**: v1.32.3 is a stable patch release with production-grade reliability
2. **Availability**: Released and available as of Q4 2024 (December 2024)
3. **Compatibility**: Fully compatible with controller-runtime 0.23.1
4. **CNI Support**: Full Cilium eBPF compatibility (v1.16.3)

### DRA (Dynamic Resource Allocation) Considerations

**Important**: DRA reached **General Availability (GA)** in **Kubernetes 1.34** (May 2025).

#### DRA Timeline
- **v1.26** (December 2022): DRA KEP introduced (Alpha)
- **v1.30** (April 2024): DRA promoted to Beta
- **v1.34** (May 2025): DRA reached GA
- **v1.35+**: Full DRA ecosystem maturity

#### Impact on This Deployment

**Current Status with K8s 1.32.3**:
- ✅ GPU Operator works (uses DevicePlugin mode as fallback)
- ⚠️ DRA API available but **not GA** (Beta status in 1.32)
- ✅ All core features functional without DRA dependency

**For DRA Native Support** (Future):
- **Minimum**: Upgrade to K8s 1.34.0+ (DRA GA)
- **Recommended**: K8s 1.35.0+ (DRA ecosystem maturity)
- **Migration Path**: GPU Operator can be reconfigured from DevicePlugin → DRA mode

### Upgrade Path

When DRA becomes critical for production workloads:

```bash
# Step 1: Verify cluster health
kubectl get nodes
kubectl get pods --all-namespaces

# Step 2: Drain nodes (for multi-node)
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data

# Step 3: Upgrade control plane
sudo apt-get update
sudo apt-get install -y kubeadm=1.34.3-1.1
sudo kubeadm upgrade plan
sudo kubeadm upgrade apply v1.34.3

# Step 4: Upgrade kubelet and kubectl
sudo apt-get install -y kubelet=1.34.3-1.1 kubectl=1.34.3-1.1
sudo systemctl daemon-reload
sudo systemctl restart kubelet

# Step 5: Uncordon nodes
kubectl uncordon <node-name>

# Step 6: Reconfigure GPU Operator for DRA
helm upgrade gpu-operator nvidia/gpu-operator \
  --set devicePlugin.enabled=false \
  --set dra.enabled=true \
  --reuse-values
```

### References

- [Kubernetes Release Notes 1.32](https://kubernetes.io/blog/2024/12/11/kubernetes-v1-32-release/)
- [DRA KEP-3063](https://github.com/kubernetes/enhancements/issues/3063)
- [NVIDIA GPU Operator DRA Support](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/gpu-operator-dra.html)
- [Nephoran SDD: docs/5G_INTEGRATION_PLAN_V2.md](./5G_INTEGRATION_PLAN_V2.md)

### Decision

**Use K8s 1.32.3 for Phase 1 deployment** to ensure stability and compatibility, with documented upgrade path to 1.34+ when DRA GA support becomes a requirement.

---

**Approved By**: Claude Code AI Agent (Sonnet 4.5)
**Next Review**: 2026-06-16 (when evaluating K8s 1.34+ upgrade)
