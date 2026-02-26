# PROJECT AUDIT 2026-02-26 - Single Source of Truth

**Audit Date**: 2026-02-26
**Cluster**: thc1006-ubuntu-22 (K8s 1.35.1)
**Purpose**: Create definitive system state documentation and cleanup plan

---

## PART 1: ACTUAL SYSTEM STATE (Evidence-Based)

### 1.1 Deployed Components (Verified via kubectl/helm)

#### Infrastructure Layer
| Component | Namespace | Status | Version | Evidence |
|-----------|-----------|--------|---------|----------|
| Kubernetes | - | ✅ Running | 1.35.1 | `kubectl version --short` |
| GPU Operator | gpu-operator | ✅ Running | v25.10.1 | `helm list -n gpu-operator` |
| NVIDIA DRA Driver | nvidia-dra-driver-gpu | ✅ Running | 25.12.0 | `helm list -A` |
| Flannel CNI | kube-flannel | ✅ Running | - | `kubectl get pods -n kube-flannel` |

#### AI/ML Stack
| Component | Namespace | Status | Version/Image | Pods |
|-----------|-----------|--------|---------------|------|
| Ollama | ollama | ✅ Running | ollama/ollama:latest (0.16.1) | 1/1 |
| Weaviate | weaviate | ✅ Running | 1.34.0 | StatefulSet 1/1 |
| MongoDB | free5gc | ✅ Running | 8.0.13 | 1/1 |

#### 5G Network Functions
| Component | Namespace | Status | Version | Pods |
|-----------|-----------|--------|---------|------|
| Free5GC Core | free5gc | ✅ Running | v3.3.0 | 9/9 (AMF, SMF, UPF x2, AUSF, NRF, NSSF, PCF, UDM, UDR, WebUI) |
| UERANSIM | free5gc | ✅ Running | v3.2.6 | 2/2 (gNB + UE) |

**Key Finding**: UERANSIM is ACTUALLY deployed and running, contradicting multiple docs saying "NOT DEPLOYED"

#### O-RAN RIC Platform
| Component | Namespace | Status | Image Version | Pods |
|-----------|-----------|--------|---------------|------|
| A1 Mediator | ricplt | ✅ Running | o-ran-sc/ric-plt-a1:3.2.3 | 1/1 |
| E2 Manager | ricplt | ✅ Running | - | 1/1 |
| E2 Term | ricplt | ✅ Running | - | 1/1 |
| O1 Mediator | ricplt | ✅ Running | - | 1/1 |
| RIC DBaaS (Redis) | ricplt | ✅ Running | - | StatefulSet 1/1 |
| KPM xApp | ricxapp | ✅ Running | - | 2/2 |
| Scaling xApp | ricxapp | ✅ Running | - | 1/1 |
| E2 Test Client | ricxapp | ✅ Running | - | 1/1 |

**Total RIC Components**: 11 Helm releases, 13 pods running

#### Observability
| Component | Namespace | Status | Version | Pods |
|-----------|-----------|--------|---------|------|
| Prometheus Stack | monitoring | ✅ Running | v0.89.0 | 6/6 |
| Grafana | monitoring | ✅ Running | - | 1/1 (NodePort 30300) |
| Alertmanager | monitoring | ✅ Running | - | 1/1 |

#### GitOps/Config Management
| Component | Namespace | Status | Version | Pods | Evidence |
|-----------|-----------|--------|---------|------|----------|
| Porch Server | porch-system | ✅ Running | nephio/porch-server:latest | 1/1 | `kubectl get all -n porch-system` |
| Porch Controllers | porch-system | ✅ Running | - | 1/1 | Verified running |
| Function Runner | porch-fn-system | ✅ Running | - | 2/2 | Verified running |
| Gitea | gitea | ✅ Running | 1.25.4 | Multiple pods | `helm list -n gitea` |

**CRITICAL FINDING**: Porch IS DEPLOYED and RUNNING, contradicting CLAUDE.md and multiple docs!

### 1.2 Total System Resources

```
Total Namespaces: 35
Total Helm Releases: 23
Total Pods: 70+ running
Total Persistent Volumes: 11 (112Gi allocated)
Total Deployments: 50+
```

### 1.3 Components NOT Deployed (Verified Absence)

| Component | Status | Evidence |
|-----------|--------|----------|
| OAI RAN | ❌ NOT DEPLOYED | No OAI namespace, no OAI pods, using UERANSIM |
| srsRAN | ❌ NOT DEPLOYED | No srsRAN deployment found |
| Nephio Controllers | ❌ NOT DEPLOYED | Config Sync not running, package-deployment-controller not deployed |
| ArgoCD | ❌ NOT DEPLOYED | No argocd namespace |
| Flux CD | ❌ NOT DEPLOYED | No flux-system namespace |

---

## PART 2: DOCUMENTATION REALITY CHECK

### 2.1 Critical Contradictions Found

#### Contradiction 1: Porch Deployment Status
**CLAUDE.md Line 46** (2026-02-23):
```
- **Porch**: NOT DEPLOYED - `http://porch-server:8080` will fail DNS
```

**REALITY**:
```bash
$ kubectl get all -n porch-system
NAME                                     READY   STATUS    RESTARTS   AGE
pod/porch-server-54f4459f9-pg4r8         1/1     Running   0          3d
pod/porch-controllers-569b4c9c67-sjbhc   1/1     Running   0          3d
pod/function-runner-56578f844b-dfk6k     1/1     Running   0          3d

service/api               ClusterIP   10.102.136.206   <none>        443/TCP,8443/TCP
```

**Impact**: HIGH - Porch is actually running and functional for 3 days

#### Contradiction 2: UERANSIM Deployment
**CLAUDE.md** says:
```
- UERANSIM: ❌ NOT DEPLOYED (Task #49)
```

**REALITY**:
```bash
$ kubectl get pods -n free5gc | grep ueransim
ueransim-gnb-66bbb9d95c-5jczm    1/1     Running   0          3d2h
ueransim-ue-6fcdcbb84c-7d2mz     1/1     Running   0          3d2h
```

**Impact**: HIGH - UERANSIM has been running for 3+ days

#### Contradiction 3: Ollama Deployment Status
**CLAUDE.md Line 45** (dated 2026-02-16):
```
└─ Ollama: ❌ NOT DEPLOYED (Task #48)
```

**Later in same file Line 45** (updated section):
```
└─ Ollama: ✅ Running (ollama namespace)
   ├─ Models: llama3.1, mistral, qwen2.5-coder
```

**Impact**: MEDIUM - Document contradicts itself

#### Contradiction 4: Document Dates
**CLAUDE.md header**:
```
Last Updated: 2026-02-23
Current System Architecture (2026-02-16)
```

**File timestamp**: `2026-02-23 15:12:01`

**Impact**: LOW - Minor inconsistency but confusing

### 2.2 Documentation Inventory

**Total Documentation Files**: 316 markdown files

#### Root Level (12 files)
```
CLAUDE.md (16K, 2026-02-23) - PRIMARY GUIDE BUT OUTDATED
README.md (35K, 2026-02-23) - Main project documentation
QUICKSTART.md (17K) - Duplicate of docs/QUICKSTART.md
CONTRIBUTING.md (19K)
SECURITY.md (13K)
CODE_OF_CONDUCT.md (14K)
CONDUCTOR_LOOP_K8S_DEPLOYMENT_REPORT.md (11K)
K8S_SUBMIT_ARCHITECTURE.md (16K)
K8S_SUBMIT_FIXES_SUMMARY.md (10K)
TASK_68_COMPLETE.md (5.6K)
TASK_69_COMPLETE.md (7.5K)
TASK_72_COMPLETE.md (12K)
```

**Issue**: Multiple TASK_XX_COMPLETE.md files in root should be in docs/

#### docs/ Directory (100+ active docs)
Key documents by category:

**Integration Plans** (9 files, 300K+ total):
- `5G_INTEGRATION_PLAN.md` (59K)
- `5G_INTEGRATION_PLAN_V2.md` (77K) - Which is authoritative?
- `NEPHIO_INTEGRATION_PLAN_2026-02-23.md` (50K)
- `NEPHIO_PLAN_VALIDATION_2026-02-23.md` (16K)
- `ORAN_PRODUCTION_MIGRATION_PLAN.md` (7.3K)
- `RAG_KNOWLEDGE_BASE_EXPANSION_PLAN.md` (24K)
- `REALITY_VS_PLAN_2026-02-23.md` (18K)
- `PACKAGE_CONSOLIDATION_PLAN.md` (9.3K)
- `LOGGING_IMPROVEMENT_PLAN.md` (12K)

**Status Reports** (20+ files):
- `CURRENT_INFRASTRUCTURE_INVENTORY_2026-02-23.md` (23K) - MOST ACCURATE
- `SESSION_SUCCESS_2026-02-23.md`
- `E2E_SUCCESS_2026-02-24.md`
- `TESTING_STATUS_2026-02-24.md`
- Multiple dated completion reports

**Issue**: Too many overlapping plans, unclear which is active

---

## PART 3: FILE STRUCTURE GARBAGE ANALYSIS

### 3.1 Generated Test Packages (MAJOR WASTE)

**Location**: `cmd/porch-structured-patch/examples/packages/structured/`

**Evidence**:
```bash
$ find cmd/porch-structured-patch/examples/packages/structured/ -type d | wc -l
214

$ du -sh cmd/porch-structured-patch/examples/packages/structured/
3.4M
```

**Details**:
- 214 auto-generated package directories
- Each has timestamp format: `*-patch-20260217-HHMMSS-NNNN`
- Created during testing on Feb 15-17
- Total waste: 3.4MB disk space
- Examples:
  ```
  batch-processor-scaling-patch-20260217-055026-1750/
  batch-processor-scaling-patch-20260217-060502-8266/
  batch-processor-scaling-patch-20260217-071412-3929/
  ... (211 more)
  ```

**Recommendation**: DELETE all timestamped directories, keep only base templates

### 3.2 Duplicate Documentation

#### Case 1: Integration Plans
- `5G_INTEGRATION_PLAN.md` (59K, older)
- `5G_INTEGRATION_PLAN_V2.md` (77K, newer)

**Issue**: V2 supersedes V1, but V1 still exists

#### Case 2: Quickstart Guides
- `/QUICKSTART.md` (root, 17K)
- `/docs/QUICKSTART.md` (exists)

**Issue**: Duplication between root and docs/

#### Case 3: README Files
Found 20+ README.md files across:
- Root
- docs/
- hack/
- examples/
- kpt-packages/
- Multiple subdirectories

**Issue**: Multiple READMEs with varying levels of accuracy

### 3.3 Deprecated/Obsolete Files

**Task completion reports in root** (should be archived):
```
TASK_68_COMPLETE.md
TASK_69_COMPLETE.md
TASK_72_COMPLETE.md
```

**Old RIC deployment files** (deployments/ric/):
```
dep/ - Old Helm charts
repo/ - Old git checkout
ric-dep/ - Duplicate Helm checkout
logs/ - Build logs from weeks ago
```

**Evidence**:
```bash
$ ls -lh deployments/ric/ | grep "^d"
drwxrwxr-x 7 thc1006 thc1006 4.0K Feb 15 13:16 dep
drwxrwxr-x 3 thc1006 thc1006 4.0K Feb 15 13:16 repo
drwxrwxr-x 7 thc1006 thc1006 4.0K Feb 15 13:16 ric-dep
```

**Issue**: Multiple copies of same Helm chart checkout

### 3.4 .claude Directory Bloat

**Location**: `.claude/agents/`

**Evidence**:
```bash
$ find .claude/agents -name "*.md" | wc -l
40+
```

**Content**: 40+ agent persona files (accessibility-audit.md, api-documenter.md, backend-architect.md, etc.)

**Issue**: These are Claude Code preset agents, not project-specific

---

## PART 4: CLEANUP ACTION PLAN

### 4.1 IMMEDIATE DELETES (Safe to Remove)

#### Priority 1: Test Artifacts (3.4MB)
```bash
# Delete all timestamped test packages
rm -rf cmd/porch-structured-patch/examples/packages/structured/*-20260217-*
rm -rf cmd/porch-structured-patch/examples/packages/structured/*-20260215-*
rm -rf cmd/porch-structured-patch/examples/packages/structured/*-20260216-*

# Keep only:
# - batch-processor-scaling-patch/
# - web-app-scaling-patch/
# - worker-scaling-patch/
# - cnf-simulator-scaling-patch/
```

**Impact**: Removes 210+ redundant directories

#### Priority 2: Old RIC Checkout Copies
```bash
# Keep only the final Helm charts, delete git repo copies
cd deployments/ric/
rm -rf repo/ ric-dep/
# Keep: dep/ (actual Helm charts), deploy scripts, values files
```

**Impact**: Removes ~50MB of duplicate git history

#### Priority 3: Duplicate/Obsolete Docs
```bash
# Move task completion reports to archive
mkdir -p docs/archive/task-completions/
mv TASK_*_COMPLETE.md docs/archive/task-completions/

# Archive superseded plans
mkdir -p docs/archive/old-plans/
mv docs/5G_INTEGRATION_PLAN.md docs/archive/old-plans/
# Keep: 5G_INTEGRATION_PLAN_V2.md as primary
```

### 4.2 CONSOLIDATE (Merge/Update)

#### Action 1: Update CLAUDE.md with Reality
**File**: `/CLAUDE.md`

**Required Changes**:
1. Line 46: Change "Porch: NOT DEPLOYED" → "Porch: ✅ DEPLOYED (porch-system namespace)"
2. Remove "Task #48" Ollama deployment (already done)
3. Remove "Task #49" MongoDB/UERANSIM (already done)
4. Update system architecture section with actual Porch pods
5. Update "Last Updated" to 2026-02-26
6. Remove conflicting date headers

#### Action 2: Consolidate Integration Plans
**Create**: `docs/INTEGRATION_STATUS_2026-02-26.md` (this file)

**Consolidates**:
- CURRENT_INFRASTRUCTURE_INVENTORY_2026-02-23.md (most accurate)
- REALITY_VS_PLAN_2026-02-23.md
- NEPHIO_INTEGRATION_PLAN_2026-02-23.md

**Archive**:
- Multiple overlapping plan documents

#### Action 3: README Hierarchy
**Keep**:
- `/README.md` - Main project overview
- `/docs/README.md` - Documentation index
- Component-specific READMEs in subdirs

**Remove**:
- Duplicate QUICKSTART.md from root (keep in docs/)

### 4.3 ARCHIVE (Move to docs/archive/)

```bash
mkdir -p docs/archive/{2026-02,plans,reports,tasks}

# Archive old plans (pre-Feb 23)
mv docs/*_PLAN.md docs/archive/plans/ # (except current ones)

# Archive old status reports (keep only latest)
mv docs/SESSION_SUCCESS_*.md docs/archive/reports/

# Archive task completions
mv docs/TASK_*.md docs/archive/tasks/
```

---

## PART 5: SINGLE SOURCE OF TRUTH STRUCTURE

### 5.1 Definitive Document Hierarchy

```
/
├── CLAUDE.md                                    [PRIMARY: AI agent guide - NEEDS UPDATE]
├── README.md                                    [PRIMARY: Project overview]
├── SECURITY.md                                  [PRIMARY: Security policy]
├── CONTRIBUTING.md                              [PRIMARY: Contribution guide]
│
├── docs/
│   ├── README.md                                [INDEX: Documentation map]
│   ├── CURRENT_INFRASTRUCTURE_INVENTORY_2026-02-23.md  [AUTHORITATIVE: System state]
│   ├── 5G_INTEGRATION_PLAN_V2.md                [AUTHORITATIVE: Integration SDD]
│   ├── PROGRESS.md                              [APPEND-ONLY: Change log]
│   │
│   ├── architecture/                            [Design decisions]
│   ├── deployment/                              [K8s manifests documentation]
│   ├── contracts/                               [Interface specs]
│   ├── sbom/                                    [Security SBOMs]
│   │
│   └── archive/                                 [OLD: Superseded documents]
│       ├── 2026-02/                             [February 2026 snapshots]
│       ├── plans/                               [Old integration plans]
│       ├── reports/                             [Old session reports]
│       └── tasks/                               [Task completion records]
│
├── deployments/
│   ├── crds/                                    [CRD definitions]
│   ├── kustomize/                               [Kustomize overlays]
│   └── ric/                                     [O-RAN RIC deployment]
│       ├── deploy-m-release.sh                  [PRIMARY: Deployment script]
│       ├── values-single-node.yaml              [PRIMARY: Config]
│       └── dep/                                 [Helm charts ONLY]
│
└── examples/
    └── packages/
        └── structured/
            ├── batch-processor-scaling-patch/   [Template ONLY]
            ├── web-app-scaling-patch/           [Template ONLY]
            └── worker-scaling-patch/            [Template ONLY]
```

### 5.2 Document Truth Priority

When documents conflict, priority order:
1. **kubectl/helm output** (ground truth)
2. **CURRENT_INFRASTRUCTURE_INVENTORY_2026-02-23.md** (most accurate snapshot)
3. **CLAUDE.md** (AI guide, but NEEDS UPDATING)
4. **5G_INTEGRATION_PLAN_V2.md** (design intent)
5. Other docs (reference only)

### 5.3 Update Protocol

**For System Changes**:
1. Make change (deploy/remove component)
2. Verify with kubectl/helm
3. Update PROGRESS.md (append-only)
4. Update CLAUDE.md if affects AI agent workflow
5. Create dated snapshot if major change (e.g., `INFRASTRUCTURE_INVENTORY_2026-03-01.md`)

**For Documentation**:
1. Create new dated document for major updates
2. Archive superseded versions to docs/archive/
3. Update docs/README.md index
4. Never delete PROGRESS.md entries (append-only)

---

## PART 6: CRITICAL ISSUES SUMMARY

### 6.1 High Priority (P0)

1. **CLAUDE.md is 3 days out of date**
   - Claims Porch NOT DEPLOYED (actually running 3+ days)
   - Claims UERANSIM NOT DEPLOYED (actually running 3+ days)
   - Mixed status on Ollama (contradicts self)
   - **Action**: Update immediately before AI makes wrong decisions

2. **214 Garbage Test Packages (3.4MB)**
   - Pollutes git status output
   - Confuses developers
   - **Action**: Delete all timestamped dirs, keep templates

3. **Unclear Which Integration Plan is Active**
   - 5G_INTEGRATION_PLAN.md vs V2?
   - NEPHIO_INTEGRATION_PLAN vs ORAN_PRODUCTION_MIGRATION_PLAN?
   - **Action**: Consolidate into single authoritative plan

### 6.2 Medium Priority (P1)

4. **Duplicate Documentation**
   - Multiple READMEs with varying accuracy
   - Duplicate QUICKSTART in root and docs/
   - **Action**: Establish hierarchy, remove duplicates

5. **Old RIC Deployment Artifacts**
   - Multiple git repo checkouts in deployments/ric/
   - **Action**: Keep Helm charts, delete git repos

6. **Task Completion Files in Root**
   - TASK_XX_COMPLETE.md should be in docs/archive/
   - **Action**: Move to proper location

### 6.3 Low Priority (P2)

7. **Excessive Plan Documents**
   - 9+ overlapping plan documents
   - **Action**: Archive old, keep current

8. **.claude Agent Bloat**
   - 40+ generic agent persona files
   - **Action**: Add to .gitignore if not needed

---

## PART 7: EXECUTION PLAN

### Phase 1: Immediate Cleanup (15 min)

```bash
# 1. Delete test package garbage
cd /home/thc1006/dev/nephoran-intent-operator
rm -rf cmd/porch-structured-patch/examples/packages/structured/*-2026*-*

# 2. Create archive structure
mkdir -p docs/archive/{2026-02,plans,reports,tasks}

# 3. Move task completions
mv TASK_*_COMPLETE.md docs/archive/tasks/

# 4. Clean RIC deployment
cd deployments/ric
rm -rf repo/ ric-dep/

# Verify
git status
```

**Expected Result**: 210+ files removed, cleaner git status

### Phase 2: Documentation Consolidation (30 min)

```bash
# 1. Archive superseded plans
mv docs/5G_INTEGRATION_PLAN.md docs/archive/plans/

# 2. Archive old session reports
mv docs/SESSION_SUCCESS_2026-02-*.md docs/archive/reports/

# 3. Update CLAUDE.md with reality (manual edit)
nano CLAUDE.md
# - Fix Porch status
# - Fix UERANSIM status
# - Fix Ollama status
# - Update "Last Updated" date

# 4. Create this audit as permanent record
# (already done - this file)
```

### Phase 3: Git Commit

```bash
git add -A
git commit -m "chore: cleanup 210+ test artifacts and archive obsolete docs

- Delete 210+ auto-generated test packages (3.4MB)
- Remove duplicate RIC git repo checkouts
- Archive superseded integration plans
- Move task completion reports to docs/archive/
- Update CLAUDE.md with actual deployment status
- Add PROJECT_AUDIT_2026-02-26.md as reference

Refs: Cleanup analysis, CLAUDE.md contradictions"
```

---

## APPENDICES

### Appendix A: Full Namespace List (35 total)
```
conductor-loop, config-management-monitoring, config-management-system,
default, free5gc, gitea, gpu-operator, gpu-test, kube-flannel,
kube-node-lease, kube-public, kube-system, local-path-storage,
monitoring, nephio-test, nephoran-conductor, nephoran-e2e-test,
nephoran-frontend, nephoran-intent, nephoran-system, nvidia-dra-driver-gpu,
ollama, porch-fn-system, porch-system, rag-service, ran-a,
resource-group-system, ricaux, ricinfra, ricplt, ricxapp,
test-resilience-irbacucf, test-resilience-nufpdqxi,
test-resilience-ymgaszbm, weaviate
```

### Appendix B: All Helm Releases (23 total)
```
free5gc (free5gc), gitea (gitea), gpu-operator (gpu-operator),
mongodb (free5gc), nvidia-dra-driver-gpu (nvidia-dra-driver-gpu),
ollama (ollama), prometheus (monitoring),
r4-a1mediator (ricplt), r4-alarmmanager (ricplt), r4-appmgr (ricplt),
r4-dbaas (ricplt), r4-e2mgr (ricplt), r4-e2term (ricplt),
r4-infrastructure (ricplt), r4-o1mediator (ricplt), r4-rtmgr (ricplt),
r4-submgr (ricplt), r4-vespamgr (ricplt),
ueransim (free5gc), upf1 (free5gc), upf2 (free5gc), upf3 (free5gc),
weaviate (weaviate)
```

### Appendix C: Documentation Size Analysis
```
Total markdown files: 316
Total docs/ files: 100+
Average doc size: 10-20K
Largest docs: 5G_INTEGRATION_PLAN_V2.md (77K), NEPHIO_INTEGRATION_PLAN (50K)
Smallest docs: < 1K (various)
```

---

**END OF AUDIT REPORT**

**Next Action**: Await user approval to execute cleanup plan
