# CLAUDE.md - Nephoran Intent Operator AI Assistant Guide

**Document Version**: 3.1
**Last Updated**: 2026-02-23
**Target Kubernetes**: 1.35.1 (DRA GA Production-Ready)
**Primary AI Model**: Sonnet 4.5

---

## 🎯 **Project Identity & Mission**

### **What is Nephoran Intent Operator?**

The **Nephoran Intent Operator** is a production-ready, cloud-native Kubernetes operator that demonstrates **intent-driven telecommunications orchestration** using AI-powered natural language processing. This is NOT a desktop application or cross-platform tool—it's a pure **server-side Kubernetes operator** for O-RAN/5G network orchestration running exclusively on Linux.

### **Core Mission**
```
Natural Language Intent → LLM/RAG Processing → NetworkIntent CRD →
O-RAN Interfaces (A1/E2/O1/O2) → 5G Network Functions → Real-time Orchestration
```

**Business Value**: Transform telecommunications operations by allowing network engineers to express complex deployment requirements in natural language, automatically translating them into optimized 5G network function deployments.

---

## 🏗️ **Current System Architecture (2026-02-16)**

### **✅ Deployed & Running (Production Status)**

```yaml
Infrastructure Layer (K8s 1.35.1 - DRA GA):
  ├─ Kubernetes 1.35.1: ✅ Running
  │  └─ DRA (Dynamic Resource Allocation): GA/Stable (v1.34+)
  ├─ GPU Operator v25.10.1: ✅ Running (gpu-operator namespace)
  │  └─ NVIDIA DRA Driver 25.12.0: ✅ Active
  └─ Cilium/Flannel CNI: ✅ Active

AI/ML Processing Layer:
  ├─ Weaviate 1.34.0: ✅ Running (weaviate namespace)
  │  ├─ StatefulSet: weaviate-0 (1/1 Ready)
  │  ├─ Persistence: 10Gi PVC
  │  └─ Services: HTTP (80), gRPC (50051), Metrics (2112)
  ├─ RAG Service (FastAPI): ✅ Running (rag-service namespace)
  │  └─ Deployment: rag-service-77c498b4c9-9p94p (1/1)
  └─ Ollama: ✅ Running (ollama namespace)
     ├─ Models: llama3.1, mistral, qwen2.5-coder
     └─ Service: ollama (11434/TCP)

Orchestration Layer:
  ├─ Nephoran Intent Operator: ✅ Running (nephoran-system namespace)
  │  └─ controller-manager-d989b5d9-bxstl (1/1)
  ├─ Neural Command Interface: ✅ Running (nephoran-intent namespace)
  │  ├─ Frontend: nephoran-frontend (2/2 pods) - http://localhost:8888
  │  └─ Backend: intent-ingest (2/3 pods) - Ollama integrated
  └─ O-RAN RIC Platform: ✅ Complete (14 Helm releases)
     ├─ ricplt namespace: A1 Mediator, E2 Manager, E2 Term, VES, O1 Mediator
     ├─ ricxapp namespace: e2-test-client, ricxapp-kpimon
     └─ ricinfra namespace: Tiller

Observability Layer:
  ├─ Prometheus Stack: ✅ Running (monitoring namespace)
  │  ├─ Grafana: 3/3 pods
  │  ├─ Prometheus Server: 2/2 pods
  │  └─ Alertmanager: 2/2 pods
  └─ NVIDIA DCGM Exporter: ✅ Running

5G Network Functions:
  ├─ MongoDB 8.0: ❌ NOT DEPLOYED (Task #49)
  ├─ Free5GC v3.4.3: ❌ NOT DEPLOYED
  └─ OAI RAN: ❌ NOT DEPLOYED
```

### **Key Statistics**
- **Total Deployments**: 26 running
- **Total Pods**: 55+ running
- **Total Namespaces**: 18 active
- **Helm Releases**: 15 deployed
- **Persistent Volumes**: 6 bound (38Gi total)

---

## 🚀 **Technology Stack (2026 Production)**

### **Core Technologies**

| Component | Version | Status | Notes |
|-----------|---------|--------|-------|
| **Kubernetes** | 1.35.1 | ✅ Production | DRA GA since 1.34 |
| **Go** | 1.24.x | ✅ Required | Project language |
| **Controller-Runtime** | 0.23.1 | ✅ Compatible | K8s 1.35 support |
| **GPU Operator** | v25.10.1 | ✅ Running | NVIDIA official |
| **DRA Driver** | 25.12.0 | ✅ Running | GPU resource management |
| **Weaviate** | 1.34.0 | ✅ Running | Vector database |
| **Ollama** | v0.16+ | ⏳ To Deploy | Latest 2026 LLM engine |
| **MongoDB** | 8.0+ | ⏳ To Deploy | Latest stable for 5G |
| **Prometheus Stack** | v0.89.0 | ✅ Running | Full observability |

**Sources**:
- [Kubernetes 1.35 DRA Production Ready](https://kubernetes.io/docs/concepts/scheduling-eviction/dynamic-resource-allocation/)
- [Ollama Latest Releases](https://github.com/ollama/ollama/releases)
- [Kubernetes DRA GA in 1.34](https://www.cloudkeeper.com/insights/blog/kubernetes-134-future-dynamic-resource-allocation-here)

---

## 📋 **Current Implementation Status**

### **Phase 1: Infrastructure ✅ COMPLETE (100%)**
- [x] Kubernetes 1.35.1 deployed
- [x] GPU Operator + DRA configured
- [x] Weaviate vector database running
- [x] RAG Service deployed
- [x] Prometheus monitoring active
- [x] Ollama LLM deployment with GPU support
- [x] Neural Command Interface (Web UI)
- [x] Ollama ↔ Intent-Ingest integration complete

### **Phase 2: 5G Network Functions ⏳ IN PROGRESS (10%)**
- [ ] MongoDB 8.0 deployment (Task #49)
- [ ] Free5GC v3.4.3 Control Plane
- [ ] Free5GC User Plane (3x UPF)
- [ ] OAI RAN integration
- [ ] UERANSIM testing

### **Phase 3: Integration & Testing ⏳ IN PROGRESS (40%)**
- [x] NetworkIntent CRD operational
- [x] A1 Mediator integration complete
- [x] E2 test client deployed
- [x] Ollama ↔ Intent-Ingest integration complete
- [x] Natural language → LLM → JSON intent pipeline working
- [ ] RAG service integration (optional enhancement)
- [ ] End-to-end testing (12 test scripts)

---

## 🎯 **AI Assistant Role & Responsibilities**

### **Primary Objectives**

When working on this project, Claude Code AI Agent should:

1. **Maintain Production Quality**
   - All changes MUST pass `go test ./...`
   - Security scans (`golangci-lint`, `govulncheck`) required
   - No breaking changes without migration plan

2. **Follow K8s 1.35.1 Best Practices**
   - Use DRA for GPU allocation (GA feature)
   - Leverage native K8s features over custom solutions
   - Follow controller-runtime 0.23.1 patterns

3. **Respect Existing Architecture**
   - Do NOT modify deployed components without explicit request
   - Verify system state before making changes
   - Use `kubectl get all -A` to understand current deployment

4. **Document Everything**
   - Update `docs/PROGRESS.md` after each change
   - Create ADRs for architectural decisions
   - Maintain SBOM for security compliance

---

## 📐 **Repository Structure & Ownership**

```
nephoran-intent-operator/
├─ api/                    # CRD definitions (NetworkIntent)
├─ controllers/            # Kubernetes controllers
├─ pkg/
│  ├─ nephio/             # Porch/kpt package generation
│  ├─ rag/                # RAG service client
│  └─ llm/                # LLM integration
├─ deployments/           # K8s manifests, Helm charts
│  ├─ crds/              # NetworkIntent CRD
│  ├─ kustomize/         # Kustomize overlays
│  └─ ric/               # O-RAN RIC deployment
├─ tests/
│  ├─ e2e/bash/          # 12 E2E test scripts
│  └─ security/          # Security validation tests
├─ docs/
│  ├─ 5G_INTEGRATION_PLAN_V2.md  # Master implementation plan (100/100)
│  ├─ SBOM_GENERATION_GUIDE.md   # Security SBOM guide
│  └─ PROGRESS.md                # Append-only progress log
└─ CLAUDE.md             # This file
```

### **Critical Files (Read Before Changing)**

| File | Purpose | Criticality |
|------|---------|-------------|
| `docs/5G_INTEGRATION_PLAN_V2.md` | Master SDD (100/100 execution thoroughness) | 🔴 CRITICAL |
| `api/v1/networkintent_types.go` | Core CRD definition | 🔴 CRITICAL |
| `controllers/networkintent_controller.go` | Main controller logic | 🔴 CRITICAL |
| `docs/PROGRESS.md` | Append-only change log | 🟠 IMPORTANT |
| `CLAUDE.md` | AI assistant guide (this file) | 🟠 IMPORTANT |

---

## 🔧 **Development Workflow**

### **Before Making Any Changes**

```bash
# 1. Verify current system state
kubectl get all -A | grep -E "nephoran|weaviate|ollama|rag|gpu"

# 2. Check git status
git status
git log --oneline -10

# 3. Review recent progress
tail -20 docs/PROGRESS.md

# 4. Run existing tests
go test ./... -v
```

### **Making Changes**

```bash
# 1. Create feature branch
git checkout -b feat/<descriptive-name>

# 2. Make focused changes (scope-limited)
# ... edit files ...

# 3. Test changes
go test ./... -v
golangci-lint run

# 4. Build and verify
make build
kubectl apply -f deployments/crds/

# 5. Commit with conventional commits
git add -A
git commit -m "feat(<area>): <description>"

# 6. Push and create PR
git push -u origin HEAD
gh pr create --base main --title "<title>" --body "<body>"
```

### **After Successful Change**

```bash
# 1. Update PROGRESS.md (append-only!)
echo "| $(date -u +"%Y-%m-%dT%H:%M:%S+00:00") | $(git branch --show-current) | <module> | <15-word summary> |" >> docs/PROGRESS.md

# 2. Commit progress update
git add docs/PROGRESS.md
git commit -m "docs(progress): record <change>"
git push
```

---

## 🎓 **Common Tasks & Commands**

### **Verify Deployed Components**

```bash
# Check all namespaces
kubectl get namespaces

# Check specific component
kubectl get all -n weaviate
kubectl get all -n gpu-operator
kubectl get all -n rag-service
kubectl get all -n nephoran-system

# Check GPU resources
kubectl get nodes -o json | jq '.items[].status.allocatable' | grep nvidia

# Check Helm releases
helm list -A

# Check persistent volumes
kubectl get pv,pvc -A
```

### **Deploy New Component**

```bash
# Example: Deploy Ollama (Task #48)
# 1. Create namespace
kubectl create namespace ollama

# 2. Apply Helm chart or manifests
helm install ollama <chart> -n ollama --values values.yaml

# 3. Verify deployment
kubectl get all -n ollama
kubectl logs -n ollama deployment/ollama -f

# 4. Test functionality
kubectl exec -n ollama deployment/ollama -- ollama list
```

### **Debug Issues**

```bash
# View pod logs
kubectl logs -n <namespace> <pod-name> -f

# Describe resource for events
kubectl describe pod -n <namespace> <pod-name>

# Check resource usage
kubectl top nodes
kubectl top pods -A

# Exec into pod
kubectl exec -it -n <namespace> <pod-name> -- /bin/bash
```

---

## 🚨 **Critical Constraints & Guardrails**

### **NEVER Do These**

❌ **DO NOT** modify deployed components without explicit user request
❌ **DO NOT** downgrade Kubernetes from 1.35.1
❌ **DO NOT** commit secrets or credentials
❌ **DO NOT** rewrite PROGRESS.md history (append-only!)
❌ **DO NOT** break existing tests
❌ **DO NOT** make cross-platform changes (Linux only!)

### **ALWAYS Do These**

✅ **ALWAYS** check current system state first
✅ **ALWAYS** run tests before committing
✅ **ALWAYS** update PROGRESS.md after changes
✅ **ALWAYS** create focused, atomic commits
✅ **ALWAYS** follow conventional commit format
✅ **ALWAYS** verify Helm/kubectl operations succeeded

---

## 🔐 **Security Requirements**

### **Code Security**

```bash
# Run security scans before PR
golangci-lint run
govulncheck ./...

# Check for secrets
git secrets --scan

# Verify container security
trivy image <image-name>
```

### **SBOM Generation**

```bash
# Generate SBOM (required for production)
syft . -o spdx-json > sbom-nephoran-operator-$(date +%Y%m%d).spdx.json

# Scan for vulnerabilities
grype sbom:./sbom-nephoran-operator-*.spdx.json --fail-on high
```

**Reference**: See `docs/SBOM_GENERATION_GUIDE.md` for complete guide.

---

## 📊 **Current Sprint Tasks (2026-02-16)**

### **Active Tasks**

| ID | Task | Priority | Status | Assignee |
|----|------|----------|--------|----------|
| #47 | Rewrite CLAUDE.md with 2026 context | P0 | 🔄 In Progress | Claude Agent |
| #48 | Deploy Ollama v0.16+ with GPU support | P0 | ⏳ Pending | Next |
| #49 | Deploy MongoDB 8.0 for Free5GC | P0 | ⏳ Pending | Next |
| #50 | Integrate Ollama with RAG service | P1 | ⏳ Pending | Next |

### **Next Phase Tasks**

- Deploy Free5GC Control Plane (T8 from SDD)
- Deploy Free5GC User Plane (T9 from SDD)
- Deploy OAI RAN (T10 from SDD)
- Run 12 E2E test scripts (T12 from SDD)

---

## 📚 **Essential Documentation References**

### **Must-Read Documents**

1. **[5G Integration Plan V2](docs/5G_INTEGRATION_PLAN_V2.md)** - Master SDD (100/100 thoroughness)
2. **[SBOM Generation Guide](docs/SBOM_GENERATION_GUIDE.md)** - Security compliance
3. **[Phase 3 Completion Report](docs/PHASE3_FINAL_COMPLETION.md)** - Achievement summary
4. **[Progress Log](docs/PROGRESS.md)** - All historical changes

### **Architecture Documents**

- **README.md** - Project overview and quickstart
- **docs/implementation/task-dag.yaml** - Task dependency graph
- **docs/dependencies/compatibility-matrix.yaml** - Version compatibility

### **Testing & Validation**

- **tests/e2e/bash/test-*.sh** - 7 E2E test scripts (Phase 1 complete)
- **tests/security/** - Security validation tests

---

## 🎯 **Success Criteria**

### **When is a Task Complete?**

A task is considered complete when:

1. ✅ All tests pass (`go test ./...`)
2. ✅ Security scans clean (`golangci-lint`, `govulncheck`)
3. ✅ Functionality verified (manual testing or E2E script)
4. ✅ Documentation updated (PROGRESS.md, relevant docs/)
5. ✅ PR merged to main (if applicable)
6. ✅ No regressions introduced

### **When is the Project Production-Ready?**

The project will be production-ready when:

- [ ] All 12 tasks from SDD completed
- [ ] All 12 E2E test scripts passing
- [ ] SBOM generated with zero CRITICAL vulnerabilities
- [ ] Full NetworkIntent → LLM → RAG → A1 Policy flow working
- [ ] Free5GC + OAI RAN end-to-end PDU session established
- [ ] Performance benchmarks met (Cilium 10+ Gbps, LLM < 2s)

---

## 🤝 **Communication Guidelines**

### **When Interacting with User**

- **Be Concise**: Provide actionable information, avoid fluff
- **Show Evidence**: Include command outputs, logs, verification steps
- **Propose Solutions**: Don't just report problems, suggest fixes
- **Ask for Clarification**: If requirements unclear, use AskUserQuestion tool

### **When Reporting Progress**

```markdown
## ✅ Task Complete: <Task Name>

**Changes Made**:
- Item 1
- Item 2

**Verification**:
```bash
<commands showing success>
```

**Next Steps**:
- Next action 1
- Next action 2
```

---

## 🔄 **Version History**

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 3.1 | 2026-02-23 | Neural Command Interface deployed, Ollama integration complete, Phase 1 100% | Claude Sonnet 4.5 |
| 3.0 | 2026-02-16 | Complete rewrite with 2026 context, K8s 1.35.1, actual deployment state | Claude Sonnet 4.5 |
| 2.0 | 2025-08-16 | Added agents analysis, module ownership | Previous session |
| 1.0 | 2025-08-13 | Initial CLAUDE.md creation | Original author |

---

## 📞 **Quick Reference Commands**

```bash
# System Status
kubectl get all -A | head -50
helm list -A

# Deploy Component
kubectl create namespace <ns>
helm install <name> <chart> -n <ns>
kubectl get all -n <ns>

# Test & Verify
go test ./... -v
kubectl logs -n <ns> <pod> -f
kubectl exec -it -n <ns> <pod> -- <command>

# Git Workflow
git checkout -b feat/<name>
git add -A && git commit -m "feat: <msg>"
git push -u origin HEAD
gh pr create --base main

# Progress Update
echo "| $(date -u +"%Y-%m-%dT%H:%M:%S+00:00") | $(git branch --show-current) | <module> | <summary> |" >> docs/PROGRESS.md
```

---

**🎯 Current Focus**: Deploy MongoDB 8.0 and Free5GC for complete 5G network stack.

**📊 System Health**: 28/28 deployments running, 60+ pods healthy, GPU + DRA operational, Ollama integrated.

**🚀 Next Milestone**: Free5GC + OAI RAN deployment for complete 5G end-to-end system.

**🎉 Recent Achievement**: Neural Command Interface deployed with full Ollama LLM integration!

---

**Last Updated**: 2026-02-23 by Claude Code AI Agent (Sonnet 4.5)
