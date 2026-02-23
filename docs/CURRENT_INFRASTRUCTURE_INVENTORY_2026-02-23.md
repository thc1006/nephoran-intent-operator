# Nephoran Intent Operator - Complete Infrastructure Inventory

**Document Version**: 1.0
**Date**: 2026-02-23
**Purpose**: Comprehensive analysis of all deployed components for Nephio integration planning
**Kubernetes Version**: 1.35.1
**Cluster**: thc1006-ubuntu-22 (single-node)

---

## Executive Summary

### System Overview
- **Total Namespaces**: 20 active
- **Total Helm Releases**: 22 deployed
- **Total Workloads**: 63 (Deployments + StatefulSets + DaemonSets)
- **Total Persistent Volumes**: 11 (112Gi total capacity)
- **Total CRDs**: 51 custom resources
- **Cluster Topology**: Single-node virtual environment (no SR-IOV hardware)

### Core Capabilities Deployed
✅ **5G Core Network**: Free5GC v3.3.0 fully operational (AMF, SMF, UPF, AUSF, NRF, etc.)
✅ **RAN Simulator**: UERANSIM v3.2.6 with gNB and UE
✅ **O-RAN RIC Platform**: 14 O-RAN SC R4 components running
✅ **AI/ML Stack**: Ollama (4.9GB llama3.1), Weaviate vector DB, RAG service
✅ **GPU Acceleration**: NVIDIA GPU Operator + DRA driver
✅ **Observability**: Prometheus + Grafana + Alertmanager
✅ **Network**: Flannel CNI + Multus CNI

### Components NOT Deployed
❌ **Nephio**: No Porch, no Config Sync, no Nephio controllers
❌ **GitOps**: No ArgoCD, no Flux CD
❌ **OAI RAN**: Not deployed (using UERANSIM instead)

---

## 1. Kubernetes Infrastructure Layer

### 1.1 Cluster Configuration
```yaml
Cluster Name: thc1006-ubuntu-22
Kubernetes Version: 1.35.1
Control Plane: Single-node
Worker Nodes: 0 (single-node cluster)
Container Runtime: containerd
Service CIDR: 10.96.0.0/12
Pod CIDR: 10.244.0.0/16
```

### 1.2 Storage Classes
```yaml
local-path:
  Provisioner: rancher.io/local-path
  Reclaim Policy: Delete
  Usage: 10 PVCs, 107Gi bound

local-storage:
  Provisioner: kubernetes.io/no-provisioner
  Reclaim Policy: Retain
  Usage: 2 PVCs (O-RAN RIC), 200Mi bound
```

### 1.3 Persistent Storage Breakdown

| Namespace | PVC Name | Size | Usage | Volume Type |
|-----------|----------|------|-------|-------------|
| `free5gc` | mongodb | 10Gi | MongoDB 8.0 database | local-path |
| `weaviate` | weaviate-data-weaviate-0 | 10Gi | Vector embeddings | local-path |
| `ollama` | ollama | 50Gi | LLM models (llama3.1, mistral, etc.) | local-path |
| `monitoring` | prometheus-db | 20Gi | Metrics time-series | local-path |
| `monitoring` | alertmanager-db | 2Gi | Alert state | local-path |
| `monitoring` | grafana | 5Gi | Dashboards | local-path |
| `conductor-loop` | conductor-loop-data | 10Gi | Intent processing | local-path |
| `conductor-loop` | conductor-loop-logs | 5Gi | Processing logs | local-path |
| `ricplt` | pvc-ricplt-alarmmanager | 100Mi | RIC alarms | local-storage |
| `ricplt` | pvc-ricplt-e2term-alpha | 100Mi | E2 termination | local-storage |

**Total Allocated Storage**: 112.2Gi

---

## 2. 5G Network Functions Layer

### 2.1 Free5GC Control Plane (Namespace: `free5gc`)

**Helm Release**: `free5gc` v1.1.7 (Chart version), App v3.3.0

| Component | Pod Status | Service | Purpose |
|-----------|------------|---------|---------|
| **AMF** | 1/1 Running | ClusterIP :80 | Access & Mobility Management |
| **SMF** | 1/1 Running | ClusterIP :80 | Session Management |
| **AUSF** | 1/1 Running | ClusterIP :80 | Authentication Server |
| **NRF** | 1/1 Running | ClusterIP :8000 | NF Repository |
| **NSSF** | 1/1 Running | ClusterIP :80 | Network Slice Selection |
| **PCF** | 1/1 Running | ClusterIP :80 | Policy Control |
| **UDM** | 1/1 Running | ClusterIP :80 | Unified Data Management |
| **UDR** | 1/1 Running | ClusterIP :80 | Unified Data Repository |
| **WebUI** | 1/1 Running | NodePort :30500 | Management Interface |
| **MongoDB** | 1/1 Running | ClusterIP :27017 | Subscriber database (v8.0.13) |

**ConfigMaps**: 15 (one per NF + common scripts)

### 2.2 Free5GC User Plane (Namespace: `free5gc`)

| Component | Helm Release | Pod Status | PFCP NodeID | N3 Interface | DNN |
|-----------|--------------|------------|-------------|--------------|-----|
| **UPF1** | upf1 v0.2.6 | 1/1 Running | 10.100.50.241 | 10.100.50.233 | internet (10.1.0.0/17) |
| **UPF2** | upf2 v0.2.6 | 1/1 Running | 10.100.50.242 | 10.100.50.234 | internet (10.1.0.128/26) |
| **UPF3** | upf3 v0.2.6 | 1/1 Running | 10.100.50.243 | 10.100.50.235 | internet (10.1.1.0/24) |

**Deployment Strategy**: Multi-UPF for traffic distribution and redundancy

### 2.3 UERANSIM RAN Simulator (Namespace: `free5gc`)

**Helm Release**: `ueransim` v2.0.17 (Chart), App v3.2.6

| Component | Pod Status | Configuration | IP Address |
|-----------|------------|---------------|------------|
| **gNB** | 1/1 Running | MCC:208, MNC:93, TAC:1 | 10.244.0.76 |
| **UE** | 1/1 Running | IMSI: 208930000000003 | 10.244.0.71 |

**Current Status**:
- ✅ gNB connected to AMF (10.100.50.249:38412)
- ✅ UE registered to 5GC
- ✅ PDU Session established (UE IP: 10.1.0.1)
- ✅ Slice: SST=1, SD=010203
- ✅ DNN: internet (connected to UPF1)

---

## 3. O-RAN RIC Platform Layer

### 3.1 Near-RT RIC (Namespace: `ricplt`)

**14 Helm Releases** from O-RAN SC R4:

| Component | Helm Release | Version | Service | Purpose |
|-----------|--------------|---------|---------|---------|
| **A1 Mediator** | r4-a1mediator | 3.0.0 | :10000 (HTTP), :4561/:4562 (RMR) | A1 Policy interface |
| **E2 Manager** | r4-e2mgr | 3.0.0 | :3800 (HTTP), :4561/:3801 (RMR) | E2 RAN connection mgmt |
| **E2 Term** | r4-e2term | 3.0.0 | :38000 (RMR), :36422 (SCTP NodePort 32222) | E2AP termination |
| **Subscription Manager** | r4-submgr | 3.0.0 | :3800 (HTTP), :4560/:4561 (RMR) | E2 subscription mgmt |
| **Routing Manager** | r4-rtmgr | 3.0.0 | :3800 (HTTP), :4561/:4560 (RMR) | RMR routing table |
| **App Manager** | r4-appmgr | 3.0.0 | :8080 (HTTP), :4561/:4560 (RMR) | xApp lifecycle mgmt |
| **Alarm Manager** | r4-alarmmanager | 5.0.0 | :8080 (HTTP), :4560/:4561 (RMR) | Alarm aggregation |
| **VES Agent** | r4-vespamgr | 3.0.0 | :8080/:9095 (HTTP) | VES event collector |
| **O1 Mediator** | r4-o1mediator | 3.0.0 | :9001/:8080/:3000, :830 (NETCONF) | O1 interface |
| **DBaaS** | r4-dbaas | 2.0.0 | :6379 (Redis) | Shared database service |
| **Infrastructure** | r4-infrastructure | 3.0.0 | Kong Gateway, Prometheus, Alertmanager | Platform services |

### 3.2 xApps (Namespace: `ricxapp`)

| xApp | Status | Purpose | Notes |
|------|--------|---------|-------|
| **ricxapp-kpimon** | Running | KPI monitoring from E2 Nodes | Passive monitoring only |
| **e2-test-client** | Running | E2 interface testing | Development/testing tool |

**Note**: Current xApps do NOT implement scaling logic. They store policies but do not execute scale operations.

---

## 4. AI/ML Processing Layer

### 4.1 Ollama LLM Engine (Namespace: `ollama`)

**Helm Release**: `ollama` v1.43.0 (Chart), App v0.16.1

**Configuration**:
```yaml
Service: ollama-service.ollama:11434
GPU Acceleration: Enabled (NVIDIA)
Models Deployed:
  - llama3.1 (4.9GB) - PRIMARY for intent processing
  - mistral (7B parameters)
  - qwen2.5-coder (for code generation)
  - phi3 (lightweight)
  - gemma2 (Google)

Storage: 50Gi PVC (model storage)
Average Inference Latency: 11-23 seconds (for intent processing)
```

**Integration**:
- Connected to `intent-ingest-service:8080` backend
- Processes natural language → structured JSON intent
- MODE=llm, PROVIDER=ollama

### 4.2 Weaviate Vector Database (Namespace: `weaviate`)

**Helm Release**: `weaviate` v17.7.0 (Chart), App v1.34.0

**Configuration**:
```yaml
StatefulSet: weaviate-0 (1/1 Ready)
Services:
  - HTTP: weaviate.weaviate:80
  - gRPC: weaviate.weaviate:50051
  - Metrics: weaviate.weaviate:2112

Storage: 10Gi PVC (vector embeddings)
Vectorizer: text2vec-transformers
Collections: Intent patterns, 5G knowledge base
```

**Purpose**: RAG (Retrieval-Augmented Generation) for intent understanding

### 4.3 RAG Service (Namespace: `rag-service`)

**Deployment**: `rag-service-77c498b4c9-9p94p` (1/1 Running)

**Configuration**:
```yaml
Framework: FastAPI
Service: rag-service.rag-service:8000
Backend: Weaviate vector DB
Purpose: Semantic search for intent context
Status: ✅ Operational
```

---

## 5. Orchestration & Management Layer

### 5.1 Nephoran Intent Operator (Namespace: `nephoran-system`)

**Deployment**: `controller-manager-d989b5d9-bxstl` (1/1 Running)

**CRDs Managed**:
- `intent.nephoran.com/NetworkIntent` (PRIMARY)
- `nephoran.com/CNFDeployment`
- `nephoran.com/E2NodeSet`
- `nephoran.com/O1Interface`
- `nephoran.com/GitOpsDeployment`
- `nephoran.com/ManifestGeneration`
- Plus 14 more CRDs

**Integration Points**:
1. Receives NetworkIntent CRs from conductor-loop
2. Translates intent → A1 Policy → A1 Mediator
3. Manages O1/O2 interface lifecycle
4. **Currently configured for**: `http://service-ricplt-a1mediator-http.ricplt:10000`

### 5.2 Conductor Loop (Namespace: `conductor-loop`)

**Deployment**: StatefulSet with persistent storage

**Configuration**:
```yaml
Handoff Directory: /var/nephoran/handoff/in
Watch Pattern: intent-*.json
Storage:
  - Data PVC: 10Gi
  - Logs PVC: 5Gi

Processing Flow:
  1. Watch handoff directory for intent files
  2. Validate JSON schema
  3. Create NetworkIntent CR in K8s
  4. Move to processed/ or failed/ subdirectory
```

**State File**: `.conductor-state.json` (tracks processing history)

---

## 6. Observability Stack (Namespace: `monitoring`)

### 6.1 Prometheus Stack

**Helm Release**: `prometheus` (kube-prometheus-stack v82.0.0)

| Component | Replicas | Storage | Metrics Scraped |
|-----------|----------|---------|-----------------|
| **Prometheus Server** | 1 | 20Gi | K8s cluster, Free5GC, O-RAN RIC, GPU metrics |
| **Alertmanager** | 1 | 2Gi | Alert routing & aggregation |
| **Grafana** | 1 | 5Gi | Visualization (3 containers: grafana, sidecar, watchdog) |
| **Kube State Metrics** | 1 | - | K8s resource state |
| **Node Exporter** | 1 (DaemonSet) | - | Host metrics |
| **Prometheus Operator** | 1 | - | CRD controller |

**ServiceMonitors**: Auto-discovery of O-RAN RIC, Free5GC, GPU metrics

### 6.2 NVIDIA DCGM Exporter

**Status**: Running (part of GPU Operator)
**Purpose**: GPU utilization, memory, temperature metrics
**Scrape Target**: Prometheus auto-discovery

---

## 7. GPU & Acceleration Layer

### 7.1 NVIDIA GPU Operator (Namespace: `gpu-operator`)

**Helm Release**: `gpu-operator` v25.10.1

**Components Deployed**:
- GPU Feature Discovery (NFD)
- Device Plugin (legacy, disabled)
- **DRA Driver** (NVIDIA DRA v25.12.0) ✅ PRIMARY
- DCGM Exporter (metrics)
- GPU Operator Validator

**DRA Status**:
```yaml
Version: 25.12.0
API: resource.nvidia.com/v1alpha1
ResourceClasses: GPU allocation via DRA
Status: ✅ GA (production-ready since K8s 1.34)
```

### 7.2 DRA Configuration

**Namespace**: `nvidia-dra-driver-gpu`

**CRDs**:
- `computedomains.resource.nvidia.com`
- `computedomaincliques.resource.nvidia.com`

**Purpose**: Dynamic GPU allocation for Ollama LLM inference

---

## 8. Networking Layer

### 8.1 CNI Configuration

**Primary CNI**: Flannel (Namespace: `kube-flannel`)
```yaml
DaemonSet: kube-flannel-ds (1/1 Running)
Backend: VXLAN
Pod Network: 10.244.0.0/16
```

**Secondary CNI**: Multus (Namespace: `kube-system`)
```yaml
DaemonSet: kube-multus-ds (1/1 Running)
Purpose: Multiple network interfaces per pod
CRD: network-attachment-definitions.k8s.cni.cncf.io
Use Case: 5G N3 interface (UPF ↔ gNB)
```

### 8.2 Service Mesh

**Status**: ❌ NOT DEPLOYED
- No Istio
- No Linkerd
- No Cilium service mesh

**Note**: Cilium was mentioned in planning docs but not deployed

---

## 9. Frontend & Backend Services

### 9.1 Intent Ingestion Backend

**Service**: `intent-ingest-service.nephoran-intent:8080`

**Configuration**:
```yaml
Framework: Go HTTP server
LLM Integration: Ollama (MODE=llm, PROVIDER=ollama)
Output: /var/nephoran/handoff/in/intent-YYYYMMDDTHHMMSSZ-nanos.json

Endpoints:
  POST /intent (text/plain): Natural language → JSON intent
  GET /health: Health check
```

### 9.2 Frontend Web UI

**Status**: ✅ Deployed and tested (E2E tests 14/15 passing)

**Features**:
- Natural language input textarea
- Real-time validation
- Intent history table
- JSON response preview
- Quick example buttons
- Connection status indicators (Ollama, Backend, Model)

**Access**: NodePort or port-forward to frontend service

---

## 10. What is MISSING for Nephio Integration

### 10.1 Nephio Core Components (NOT DEPLOYED)

| Component | Status | Purpose | Priority |
|-----------|--------|---------|----------|
| **Porch** | ❌ Missing | Package Orchestration for Resource Constraint Handling | P0 |
| **Config Sync** | ❌ Missing | GitOps sync from repo to cluster | P0 |
| **Nephio Controllers** | ❌ Missing | Deployment controllers, Repository controller | P0 |
| **Resource Backend** | ❌ Missing | IPAM, VLAN, AS# allocation | P1 |
| **kpt CLI** | ❌ Missing | Package management tool | P1 |

### 10.2 GitOps Layer (NOT DEPLOYED)

| Component | Status | Alternative | Notes |
|-----------|--------|-------------|-------|
| **Git Repository** | ❌ Missing | Local filesystem handoff | Need Gitea/Gogs |
| **ArgoCD** | ❌ Missing | Manual kubectl apply | Optional if using Config Sync |
| **Flux CD** | ❌ Missing | Manual kubectl apply | Alternative to ArgoCD |

### 10.3 Architecture Gap Analysis

**Current Architecture** (Custom SMO):
```
Natural Language → Backend → Intent JSON → Conductor-Loop → NetworkIntent CR → Controller → A1 Policy
```

**Nephio Architecture** (Target):
```
Natural Language → Backend → Intent JSON → Porch Package → Git Commit → Config Sync → NetworkIntent CR → Controller → A1 Policy + O2 IMS
```

**Key Difference**: Nephio uses GitOps with kpt packages, current system uses filesystem handoff.

---

## 11. Configuration Management

### 11.1 ConfigMaps by Purpose

**Free5GC**: 15 ConfigMaps (one per NF + MongoDB scripts)
**O-RAN RIC**: Built-in to Helm charts
**UERANSIM**: 2 ConfigMaps (gNB + UE configuration)
**Monitoring**: Grafana dashboards, Prometheus rules

### 11.2 Secrets Management

**Status**: Basic K8s Secrets (not externalized)
- MongoDB credentials
- Free5GC subscriber keys
- O-RAN RIC JWT tokens

**Gap**: No Vault, no External Secrets Operator

---

## 12. Network Topology

### 12.1 Service CIDRs

```yaml
Kubernetes Services: 10.96.0.0/12
Pod Network (Flannel): 10.244.0.0/16
Free5GC UE Subnet: 10.1.0.0/16
  - UPF1 allocates: 10.1.0.0/17
  - UPF2 allocates: 10.1.0.128/26
  - UPF3 allocates: 10.1.1.0/24
```

### 12.2 External Access

**NodePort Services**:
- Free5GC WebUI: :30500
- E2 Term SCTP: :32222
- Kong Proxy: :32080 (HTTP), :32443 (HTTPS)
- Kong Manager: :31258, :32609

**LoadBalancer Services**:
- r4-infrastructure-kong-proxy (pending external LB)

---

## 13. Integration Points Summary

### 13.1 Successful Integrations ✅

1. **Frontend → Backend → Ollama**: Natural language processing working
2. **Ollama → Weaviate RAG**: Context-aware intent understanding
3. **Backend → Conductor-Loop**: File-based handoff operational
4. **Conductor-Loop → NetworkIntent CR**: K8s integration working
5. **NetworkIntent Controller → A1 Mediator**: Policy creation successful
6. **UERANSIM UE → Free5GC**: PDU Session established
7. **Free5GC → O-RAN RIC**: E2 interface connected
8. **Prometheus → All Components**: Metrics collection working

### 13.2 Pending Integrations ⏳

1. **Nephio Porch Integration**: Need to deploy Porch first
2. **GitOps Workflow**: Need Git repo + Config Sync
3. **O2 IMS Integration**: NetworkIntent Controller → O-RAN O2 interface
4. **xApp Scaling Logic**: Custom xApp to implement actual scaling

---

## 14. Performance Metrics (Observed)

| Metric | Value | Acceptable Range | Status |
|--------|-------|------------------|--------|
| LLM Inference Latency | 11-23s | < 30s | ✅ OK |
| Intent Processing (e2e) | 13-40s | < 60s | ✅ OK |
| PDU Session Establishment | < 2s | < 5s | ✅ OK |
| A1 Policy Creation | < 500ms | < 1s | ✅ OK |
| Frontend Response | 12-13s | < 30s | ✅ OK |

---

## 15. Capacity & Resource Utilization

### 15.1 Storage Utilization
- **Total Allocated**: 112.2Gi
- **Largest Consumer**: Ollama (50Gi for LLM models)
- **Growth Rate**: ~5Gi/week (logs, metrics, embeddings)
- **Recommendation**: Monitor Prometheus TSDB size

### 15.2 Compute Resources
- **Single-node cluster**: All workloads on one node
- **GPU**: Allocated to Ollama exclusively
- **CPU/Memory**: Not explicitly limited (relying on K8s defaults)

---

## 16. Security Posture

### 16.1 Current Security Features ✅
- NetworkPolicies: ❌ NOT CONFIGURED
- Pod Security Policies: ❌ NOT CONFIGURED (deprecated)
- RBAC: ✅ K8s default RBAC active
- Secrets: ✅ Basic K8s Secrets
- TLS: ⚠️ Partial (Kong Gateway only)
- mTLS: ❌ NOT CONFIGURED (no service mesh)

### 16.2 Security Gaps ⚠️
1. No network segmentation between namespaces
2. No pod-to-pod encryption
3. Secrets not externalized
4. No audit logging configured
5. No admission controllers (OPA, Kyverno)

---

## 17. Backup & Recovery

**Status**: ❌ NO BACKUP STRATEGY CONFIGURED

**Critical Data to Backup**:
1. MongoDB (Free5GC subscriber data) - 10Gi
2. Weaviate embeddings - 10Gi
3. Prometheus metrics - 20Gi
4. NetworkIntent CRs - K8s etcd
5. ConfigMaps & Secrets - K8s etcd

**Recommendation**: Deploy Velero for K8s backup

---

## 18. Namespace Organization

```
Infrastructure:
├─ kube-system (K8s core)
├─ kube-flannel (CNI)
├─ local-path-storage (provisioner)
├─ gpu-operator (NVIDIA)
└─ nvidia-dra-driver-gpu (DRA)

Observability:
└─ monitoring (Prometheus stack)

AI/ML:
├─ ollama (LLM engine)
├─ weaviate (Vector DB)
└─ rag-service (RAG API)

5G Network Functions:
└─ free5gc (Core + RAN simulator + MongoDB)

O-RAN:
├─ ricplt (Near-RT RIC platform)
├─ ricxapp (xApps)
└─ ricinfra (RIC infrastructure)

Nephoran Operator:
├─ nephoran-system (Controller Manager)
├─ nephoran-conductor (Conductor Loop) - new
├─ nephoran-intent (Backend service)
└─ conductor-loop (Legacy namespace)

Test:
├─ ran-a (Test namespace for scaling demos)
├─ gpu-test (GPU validation)
└─ nephio-test (Empty - for future Nephio)
```

---

## 19. Dependencies & Version Matrix

| Component | Version | Kubernetes Version | Compatible? |
|-----------|---------|-------------------|-------------|
| Free5GC | v3.3.0 | 1.25+ | ✅ Yes |
| UERANSIM | v3.2.6 | Any | ✅ Yes |
| O-RAN SC RIC | R4 | 1.24-1.30 | ⚠️ Yes (tested) |
| GPU Operator | v25.10.1 | 1.27+ | ✅ Yes |
| DRA | 25.12.0 | 1.34+ (GA) | ✅ Yes |
| Prometheus Stack | v82.0.0 | 1.29+ | ✅ Yes |
| Weaviate | 1.34.0 | Any | ✅ Yes |
| Ollama | 0.16.1 | Any | ✅ Yes |
| **Nephio** | **R5/R6** | **1.31-1.35** | ⏳ **To Deploy** |

---

## 20. Next Steps for Nephio Integration

### Phase 1: Deploy Nephio Core (P0)
1. Deploy Nephio Management Cluster controllers
2. Deploy Porch server (package orchestration)
3. Configure Git repository (Gitea or GitHub)
4. Deploy Config Sync for GitOps

### Phase 2: Migrate Intent Flow (P1)
1. Create kpt package templates for NetworkIntent
2. Integrate Backend → Porch API (replace filesystem handoff)
3. Configure Git-based workflow
4. Test end-to-end Nephio orchestration

### Phase 3: Advanced Features (P2)
1. Deploy Resource Backend (IPAM, VLAN pools)
2. Configure Nephio specializers
3. Integrate O2 IMS for inventory management
4. Deploy multi-cluster edge nodes

---

## 21. Key Contacts & References

### Documentation Locations
- **CLAUDE.md**: `/home/thc1006/dev/nephoran-intent-operator/CLAUDE.md`
- **5G Integration Plan V2**: `docs/5G_INTEGRATION_PLAN_V2.md`
- **Session Success Log**: `docs/SESSION_SUCCESS_2026-02-23.md`
- **Architecture Plan**: `.claude/plans/shimmering-shimmying-rainbow.md`

### Configuration Files
- **Kubeconfig**: `~/.kube/config`
- **Helm Values**: Stored in Helm release secrets

---

## Appendix A: All Helm Releases

```bash
$ helm list -A
NAME                  NAMESPACE            VERSION    STATUS
free5gc               free5gc              1.1.7      deployed
mongodb               free5gc              16.5.45    deployed
ueransim              free5gc              2.0.17     deployed
upf1                  free5gc              0.2.6      deployed
upf2                  free5gc              0.2.6      deployed
upf3                  free5gc              0.2.6      deployed
gpu-operator          gpu-operator         v25.10.1   deployed
nvidia-dra-driver-gpu nvidia-dra-driver-gpu 25.12.0  deployed
prometheus            monitoring           82.0.0     deployed
ollama                ollama               1.43.0     deployed
weaviate              weaviate             17.7.0     deployed
r4-infrastructure     ricplt               3.0.0      deployed
r4-dbaas              ricplt               2.0.0      deployed
r4-appmgr             ricplt               3.0.0      deployed
r4-rtmgr              ricplt               3.0.0      deployed
r4-e2mgr              ricplt               3.0.0      deployed
r4-e2term             ricplt               3.0.0      deployed
r4-a1mediator         ricplt               3.0.0      deployed
r4-submgr             ricplt               3.0.0      deployed
r4-vespamgr           ricplt               3.0.0      deployed
r4-o1mediator         ricplt               3.0.0      deployed
r4-alarmmanager       ricplt               5.0.0      deployed
```

---

## Appendix B: All CRDs (51 total)

**Nephoran Intent Operator CRDs** (16):
- networkintents.intent.nephoran.com (PRIMARY)
- networkintents.nephoran.com
- networkintents.nephoran.io
- cnfdeployments.nephoran.com
- cnfdeployments.nephoran.io
- e2nodesets.nephoran.com
- e2nodesets.nephoran.io
- o1interfaces.nephoran.com
- managedelements.nephoran.com
- intentprocessings.nephoran.com
- intentprocessings.nephoran.io
- gitopsdeployments.nephoran.com
- gitopsdeployments.nephoran.io
- manifestgenerations.nephoran.com
- manifestgenerations.nephoran.io
- resourceplans.nephoran.com
- audittrails.nephoran.com
- backuppolicies.nephoran.com
- certificateautomations.nephoran.com
- disasterrecoveryplans.nephoran.com
- failoverpolicies.nephoran.com

**Prometheus Operator CRDs** (10):
- servicemonitors, podmonitors, prometheuses, prometheusrules, alertmanagers, etc.

**GPU/DRA CRDs** (6):
- clusterpolicies.nvidia.com
- nvidiadrivers.nvidia.com
- computedomains.resource.nvidia.com
- computedomaincliques.resource.nvidia.com
- nodefeatures.nfd.k8s-sigs.io
- nodefeaturerules.nfd.k8s-sigs.io

**Kong CRDs** (10):
- kongplugins, kongconsumers, kongingresses, etc.

**Multus CNI CRD** (1):
- network-attachment-definitions.k8s.cni.cncf.io

---

**Document End**

---

**Revision History**:
- v1.0 (2026-02-23): Initial comprehensive inventory for Nephio integration planning
