# O-RAN SC RIC Platform Deployment Guide
## Kubernetes 1.35.1 Single-Node Cluster

This directory contains manifests and configurations for deploying the O-RAN Software Community (O-RAN SC) Near-RT RIC Platform on Kubernetes 1.35.1.

## Quick Start

### Prerequisites
- Kubernetes 1.35.1 cluster (single or multi-node)
- Helm 4.1.0 or compatible
- kubectl configured to access cluster
- 31GB RAM, 8 cores, 44GB storage available (minimum)
- local-path StorageClass available

### Environment Verification
```bash
kubectl version
helm version
kubectl get storageclass
kubectl get nodes
```

Expected output:
- Client & Server: v1.35.1
- Helm: v4.1.0
- StorageClass: local-path (rancher.io/local-path)

## Deployment Steps

### Step 1: Prepare Namespaces
```bash
kubectl create namespace ricplt
kubectl create namespace ricxapp
kubectl create namespace ricinfra
```

### Step 2: Review Configuration
Edit `recipe-k135-minimal.yaml` with your environment specifics:
- `extsvcplt.ricip`: Node IP or external service IP (default: 192.168.10.65)
- Resource limits for single-node optimization
- Image pull policy (IfNotPresent for offline deployments)

### Step 3: Deploy Using Official Script (Recommended)
```bash
cd repo
./bin/install -f ../recipe-k135-minimal.yaml
```

**Expected Duration:** 3-5 minutes for all pods to reach Running state

### Step 4: Verify Deployment
```bash
# Check all pods are running
kubectl get pods -n ricplt -o wide

# Check services
kubectl get svc -n ricplt

# Check logs for any errors
kubectl logs -n ricplt deployment/deployment-ricplt-e2mgr --tail=50

# Test RIC API endpoints
kubectl port-forward -n ricplt svc/service-ricplt-e2mgr-http 3800:3800 &
curl http://localhost:3800/v1/health
```

## Files in This Directory

### Configuration Files
- `recipe-k135-minimal.yaml` - RIC deployment recipe for K Release, customized for K8s 1.35.1
- `values-single-node.yaml` - Helm values for single-node optimization (alternative)

### Documentation
- `README.md` - This file
- `../docs/RIC_DEPLOYMENT_REPORT.md` - Detailed compatibility analysis

### Supporting Directories
- `repo/` - O-RAN SC ric-plt/ric-dep repository (cloned from gerrit.o-ran-sc.org)
- `k8s-135-patches/` - K8s 1.35-specific patches (if needed)
- `../monitoring/servicemonitors/` - Prometheus monitoring setup

## Deployment Architecture

### Namespaces
```
ricinfra  - Infrastructure layer
  └─ Kong ingress, storage provisioning, docker registry

ricplt    - RIC Platform (core components)
  ├─ e2mgr (E2 Manager)
  ├─ e2term-alpha (E2 Termination)
  ├─ dbaas (Database/Redis)
  ├─ submgr (Subscription Manager)
  ├─ appmgr (Application Manager)
  ├─ rtmgr (Route Manager)
  ├─ a1mediator (A1 Policy Interface)
  ├─ o1mediator (O1 Management Interface)
  └─ [optional: influxdb, vespamgr, alarmmanager]

ricxapp   - XApp container namespace (empty initially)
```

### Core Components (Mandatory)
| Component | Purpose | Port | Image |
|-----------|---------|------|-------|
| e2mgr | E2 interface management | 3800 | ric-plt-e2mgr:6.0.6 |
| e2term | E2 protocol termination (SCTP) | 36422 | ric-plt-e2:6.0.6 |
| dbaas | Key-value database (Redis) | 6379 | ric-plt-dbaas:0.6.4 |
| submgr | RAN subscription management | 8088 | ric-plt-submgr:0.10.2 |
| appmgr | XApp lifecycle management | 8080 | ric-plt-appmgr:0.5.8 |
| rtmgr | Message routing | 3800 | ric-plt-rtmgr:0.9.6 |

### Optional Components
- **A1 Mediator** - Policy interface to near-RT RIC
- **O1 Mediator** - Management interface (YANG models)
- **InfluxDB** - Time-series database for KPI metrics
- **VES Manager** - FCAPS event collection
- **Alarm Manager** - Event/alert management

## Troubleshooting

### Pod Stuck in Pending
```bash
# Check resource constraints
kubectl describe nodes | grep -A5 "Allocated resources"

# Check PVC binding
kubectl get pvc -n ricplt

# Check events
kubectl get events -n ricplt --sort-by='.lastTimestamp' | tail -20
```

### Pod CrashLoopBackOff
```bash
# Check logs
kubectl logs -n ricplt <pod-name> --previous

# Check resource limits
kubectl describe pod -n ricplt <pod-name> | grep -A5 "Limits\|Requests"

# Check image pull
kubectl describe pod -n ricplt <pod-name> | grep -A5 "Image"
```

### Service Unreachable
```bash
# Verify service endpoints
kubectl get endpoints -n ricplt

# Test DNS resolution
kubectl run -it --rm debug --image=alpine --restart=Never -- \
  sh -c 'nslookup service-ricplt-e2mgr-http.ricplt.svc.cluster.local'

# Test port connectivity
kubectl run -it --rm debug --image=alpine --restart=Never -- \
  sh -c 'nc -zv service-ricplt-e2mgr-http.ricplt.svc.cluster.local 3800'
```

### E2 Connection Failed
```bash
# Verify e2term service is NodePort-exposed
kubectl get svc -n ricplt service-ricplt-e2term-sctp-alpha

# Check SCTP support
grep CONFIG_IP_SCTP /boot/config-$(uname -r) || echo "SCTP support needed"

# View e2term logs
kubectl logs -n ricplt -l app=ricplt-e2term -f
```

## Uninstallation

```bash
# Using the official uninstall script
cd repo
./bin/uninstall

# Or manual cleanup
helm uninstall -n ricplt -l release=r4
kubectl delete namespace ricplt ricxapp ricinfra
```

## Network Connectivity

### Port Mappings
| Service | Protocol | Port | Access |
|---------|----------|------|--------|
| e2mgr-http | TCP | 3800 | Internal (ClusterIP) |
| e2mgr-rmr | TCP | 4561/3801 | Internal (RMR messaging) |
| e2term-sctp | SCTP | 36422 | External (NodePort) |
| submgr-http | TCP | 8088 | Internal |
| appmgr-http | TCP | 8080 | Internal |

### Testing Connectivity
```bash
# From within cluster
kubectl exec -it -n ricplt <pod-name> -- \
  curl http://service-ricplt-e2mgr-http:3800/v1/health

# From outside cluster (port-forward)
kubectl port-forward -n ricplt svc/service-ricplt-e2mgr-http 3800:3800
curl http://localhost:3800/v1/health

# SCTP E2 connection (requires simulator)
# See: https://gerrit.o-ran-sc.org/r/sim/e2-interface
```

## Performance Optimization

### Single-Node Tuning
The deployment is pre-configured for single-node with:
- Resource limits: 256Mi requests / 512Mi limits per component
- No pod anti-affinity (enables co-location)
- Local-path storage (no network overhead)
- Reduced probe intervals for faster startup

### Scaling for Multi-Node
If expanding to 3+ nodes:
1. Set `dbaas.enableHighAvailability: true`
2. Set `dbaas.enablePodAntiAffinity: true` (requires 3+ nodes)
3. Increase resource requests for production workloads
4. Configure external storage instead of local-path

## Monitoring Integration

### Prometheus ServiceMonitors
ServiceMonitors are created for automatic Prometheus scraping:
```bash
# View created monitors
kubectl get servicemonitor -n monitoring

# Check Prometheus targets
kubectl port-forward -n monitoring svc/kube-prometheus-prometheus 9090:9090
# Visit: http://localhost:9090/targets
```

### Grafana Dashboards
RIC platform metrics available in Grafana:
- Pod metrics (CPU, memory, network)
- RIC-specific KPIs (E2 connections, subscriptions, message rates)
- Database metrics (Redis capacity, connection count)

Access: http://192.168.10.65:30300

## Documentation

- **Official Docs:** https://docs.o-ran-sc.org/
- **RIC Installation Guide:** https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-ric-dep/en/latest/
- **K Release:** https://docs.o-ran-sc.org/en/k-release/
- **This Analysis:** `../../docs/RIC_DEPLOYMENT_REPORT.md`

## Support

For issues:
1. Check troubleshooting section above
2. Review `../docs/RIC_DEPLOYMENT_REPORT.md` for K8s 1.35 compatibility notes
3. Consult O-RAN SC documentation: https://wiki.o-ran-sc.org/
4. GitHub issues: https://github.com/o-ran-sc/
5. Gerrit reviews: https://gerrit.o-ran-sc.org/

## License

O-RAN SC RIC Platform is licensed under Apache 2.0.
See `repo/LICENSE` for details.
