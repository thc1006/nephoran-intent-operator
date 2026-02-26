# ACTUAL SYSTEM STATE - 2026-02-26

**Purpose**: Single Source of Truth (verified via kubectl)
**Last Verified**: 2026-02-26 22:00 UTC
**Cluster**: thc1006-ubuntu-22 (K8s 1.35.1)

---

## DEPLOYED COMPONENTS (VERIFIED)

### Infrastructure
- ✅ Kubernetes 1.35.1
- ✅ GPU Operator v25.10.1 + NVIDIA DRA 25.12.0
- ✅ Flannel CNI

### AI/ML Stack
- ✅ Ollama v0.16.1 (ollama namespace)
  - Models: llama3.1, mistral, qwen2.5-coder
- ✅ Weaviate 1.34.0 (weaviate namespace)
- ✅ RAG Service (rag-service namespace)
- ✅ MongoDB 8.0.13 (free5gc namespace)

### 5G Network Functions
- ✅ Free5GC v3.3.0 (free5gc namespace, 9/9 pods)
  - AMF, SMF, UPF x2, AUSF, NRF, NSSF, PCF, UDM, UDR, WebUI
- ✅ UERANSIM v3.2.6 (free5gc namespace, 2/2 pods)
  - gNB + UE running for 3+ days

### O-RAN RIC Platform
- ✅ RIC Platform (ricplt namespace, 11 Helm releases)
  - A1 Mediator (o-ran-sc/ric-plt-a1:3.2.3)
  - E2 Manager
  - E2 Term
  - O1 Mediator
  - DBaaS (Redis StatefulSet)
- ✅ xApps (ricxapp namespace)
  - scaling-xapp (1/1) - REAL, working
  - kpimon (2/2) - MOCK version
  - e2-test-client (1/1) - Test tool

### GitOps/Package Management
- ✅ Porch (porch-system namespace, 3/3 pods)
  - porch-server (running 3+ days)
  - porch-controllers
  - function-runner (porch-fn-system)
- ✅ Gitea 1.25.4 (gitea namespace)

### Observability
- ✅ Prometheus Stack v0.89.0 (monitoring namespace)
- ✅ Grafana (NodePort 30300, password: NephoranMonitor2026!)
- ✅ Alertmanager

### Application Layer
- ✅ Nephoran Intent Operator (nephoran-system namespace)
  - controller-manager (1/1)
  - nephoran-ui (2/2, NodePort 30081)
- ✅ Intent Ingest Service (nephoran-intent namespace)
- ✅ Conductor Loop (nephoran-conductor namespace)

---

## NOT DEPLOYED (CONFIRMED ABSENCE)

- ❌ OAI RAN - Using UERANSIM instead
- ❌ srsRAN - Configuration exists but not deployed
- ❌ Nephio Controllers - Config Sync not running
- ❌ ArgoCD - Not deployed
- ❌ Flux CD - Not deployed

---

## SYSTEM STATISTICS

- Total Namespaces: 35
- Total Helm Releases: 23
- Total Pods Running: 95
- Total PVs: 12 (127.2Gi allocated)
- Total Deployments: 61

---

## CRITICAL ISSUES

### Issue 1: A1 Policy Status Update (HTTP 405)
**Status**: Identified
**Cause**: xApp POSTs to `/status`, O-RAN spec only allows GET
**Solution**: Redis status store (implementation pending)

### Issue 2: Mock KPI Monitor
**Status**: Running but non-functional
**Reality**: 2 pods running, but pure placeholder with no E2 subscription
**Solution**: Deploy o-ran-sc/ric-app-kpimon-go

### Issue 3: No E2-Enabled gNB
**Status**: UERANSIM lacks E2 interface
**Reality**: srsRAN configured but not deployed
**Solution**: Deploy srsRAN with E2 agent

---

## DOCUMENTATION STATUS

**Authoritative Documents**:
- THIS FILE (SYSTEM_STATE.md) - Single source of truth
- PROJECT_AUDIT_2026-02-26.md - Detailed audit report
- 5G_INTEGRATION_PLAN_V2.md - Implementation plan (V2 supersedes V1)

**Obsolete Documents** (archived to docs/archive/):
- TASK_XX_COMPLETE.md files (moved to archive/task-completions/)
- 5G_INTEGRATION_PLAN.md V1 (superseded by V2)
- Multiple overlapping plans

**Known Issues**:
- CLAUDE.md outdated (contains falsehoods about Porch/UERANSIM)
- MEMORY.md partially outdated
- Multiple README files with varying accuracy

---

## VERIFICATION COMMANDS

```bash
# Verify Porch
kubectl get all -n porch-system

# Verify Free5GC
kubectl get pods -n free5gc

# Verify RIC
kubectl get all -n ricplt
kubectl get pods -n ricxapp

# Verify Ollama
kubectl exec -n ollama deployment/ollama -- ollama list

# Verify Monitoring
kubectl get pods -n monitoring
# Grafana: http://192.168.10.65:30300

# Verify Frontend
kubectl get svc -n nephoran-system nephoran-ui
# UI: http://192.168.10.65:30081
```

---

**Last Audit**: 2026-02-26 by Claude Sonnet 4.5 (comprehensive system audit)
