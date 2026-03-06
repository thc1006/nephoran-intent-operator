# CLAUDE.md — Nephoran Intent Operator

> Single source of truth for Claude Code. Describes the **actual** system as built and deployed.
> Last verified: 2026-03-05 against live cluster `thc1006-ubuntu-22` (K8s v1.35.1).

---

## 0. Hard Rules

1. **No direct kubectl from LLM** — LLM produces a schema-constrained IntentPlan JSON only. Never raw YAML, never direct apply.
2. **Everything is GitOps** — desired state changes go through Git → Porch/ConfigSync → cluster. No imperative mutations.
3. **Guardrails first** — validate plan (JSON Schema) → validate KRM (OPA + kubeconform + dry-run) → human review for sensitive resources (RBAC, CRDs, privileged, hostNetwork, cluster-scoped).
4. **Upstream first** — prefer O-RAN SC official images for RIC xApps and simulators.
5. **Identity by labels** — scale-out tracking uses resource names + labels, never Pod name suffixes. StatefulSet only when stable pod identity is required.

---

## 1. Project Identity

| Field | Value |
|---|---|
| Module | `github.com/thc1006/nephoran-intent-operator` |
| Go version | 1.26.0 |
| Goal | Intent-driven O-RAN lifecycle management: NL intent → LLM plan → KRM/kpt packages → GitOps → Porch/ConfigSync → workload cluster |
| Target HW | Single workstation, NVIDIA RTX 5080 (16 GB VRAM), 32 GB RAM, Ubuntu 22.04 |
| K8s | kubeadm v1.35.1 single-node, containerd 2.2.1 (no Docker daemon) |
| Standards | 3GPP TS 28.541 · O-RAN WG6/WG10 · TMF 921 (Intent API subset) |

---

## 2. Live Platform Status

### 2.1 Deployed Components

| Component | Namespace | Status | Notes |
|---|---|---|---|
| **Free5GC v3.3.0** | `free5gc` | Running | AMF, SMF, UPF×2, UDM, UDR, AUSF, NRF, NSSF, PCF, WebUI, MongoDB 8.0, UERANSIM gNB+UE |
| **O-RAN SC Near-RT RIC** | `ricplt` | Running | 14 Helm releases: A1, E2, O1, RTmgr, SubMgr, AlarmMgr, VES, Kong, Prometheus, DBAAS(Redis) |
| **RIC xApps** | `ricxapp` | Partial | ricxapp-scaling ✅, e2-test-client ✅, KPIMON scaled-to-0, srsran-gnb scaled-to-0 |
| **Nephoran Operator** | `nephoran-system` | Running | controller-manager (watches `intent.nephoran.com/v1alpha1` NetworkIntent CRD) |
| **Nephoran Web UI** | `nephoran-system` | Running | NodePort 30090 (K8s Dashboard style), old UI at 30081 |
| **Intent Ingest** | `nephoran-intent` | Running | HTTP service accepting NL text, calls Ollama for LLM processing |
| **Conductor Loop** | `conductor-loop` | Running | File-watching orchestration pipeline |
| **Porch v3.0.0** | `porch-system` | Running | porch-server, porch-controllers, function-runner ×2 |
| **ConfigSync** | `config-management-system` | Running | root-reconciler syncing `nephoran-packages` repo |
| **Ollama** | `ollama` + systemd | Running | llama3.1:latest, CPU-only (no GPU passthrough via DRA yet) |
| **RAG Service** | `rag-service` | Running | FastAPI uvicorn on host port 8000, connects to Ollama |
| **Weaviate v1.34.0** | `weaviate` | Running | Vector DB for RAG knowledge base |
| **Gitea** | `gitea` | Running | Local Git server, SSH NodePort 30022 |
| **Prometheus + Grafana** | `monitoring` | Running | kube-prometheus-stack, Grafana NodePort 30300 |
| **GPU Operator** | `gpu-operator` | Running | v25.10.1, DRA mode (not traditional device-plugin) |

### 2.2 CRD: NetworkIntent

```yaml
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
```
- Validated CRD at `config/crd/intent/intent.nephoran.com_networkintents.yaml`
- 42 NetworkIntent resources currently in cluster (default, free5gc, ran-a, ricxapp namespaces)
- 10 Porch PackageRevisions in catalog (5 Free5GC + 4 O-RAN + 1 hello-nephoran)

### 2.3 Known Issues

- **RIC ImagePullSecret missing**: `secret-nexus3-o-ran-sc-org-10002-o-ran-sc` not found; images run from local cache, but rebuild/upgrade will fail
- **GPU DRA mode**: K8s Pods cannot request `nvidia.com/gpu` via traditional resource requests; need DRA ResourceClaim for GPU workloads
- **Ollama dual deployment**: both in-cluster Pod and host systemd service running (~980 Mi total)
- **MongoDB readiness**: occasional 5s probe timeout (non-critical, self-recovers)

---

## 3. Repository Structure (Actual)

```
nephoran-intent-operator/
├── CLAUDE.md                     # THIS FILE
├── go.mod / go.sum               # Go 1.26, 120+ direct dependencies
├── Dockerfile                    # Multi-target: manager, conductor-loop, intent-ingest, llm-processor, etc.
│
├── llm_nephio_oran/              # Python — M1 MVP (intentctl CLI + intentd API + planner + validator)
│   ├── intentctl.py              # CLI: typer + rich, create/push commands
│   ├── intentd/app.py            # FastAPI TMF921 subset API server
│   ├── models.py                 # Pydantic IntentPlan, Action, SliceSpec, Policy, Guardrails
│   ├── planner/stub_planner.py   # Deterministic stub planner (to be replaced by local LLM)
│   ├── generator/kpt_generator.py # Stub kpt package generator
│   ├── validators/schema_validate.py # JSON Schema validator (Draft 2020-12)
│   ├── gitops/pr_stub.py         # PR automation stub
│   └── observability/metrics_stub.py # Prometheus metrics stub
│
├── src/                          # Python/TS microservices (for containerized deployment)
│   ├── llm-gateway/              # FastAPI: /generate, /hydrate, /analyze (wraps vLLM/Ollama)
│   ├── intent-engine/            # FastAPI: TMF921 CRUD API (in-memory store)
│   ├── config-validator/         # FastAPI: YAML validation + OPA policies
│   └── web-ui/                   # React 18 + TypeScript + Vite + Tailwind (skeleton)
│
├── schemas/                      # JSON Schema
│   ├── intent-plan.schema.json   # Canonical IntentPlan schema (Draft 2020-12)
│   ├── intent-plan.example.json  # Example intent (slice.deploy)
│   └── policy.schema.json        # Policy sub-schema
│
├── pkg/                          # Go packages (761 non-test files, 607K lines)
│   ├── controllers/              # K8s controller-runtime reconcilers (NetworkIntent + orchestration)
│   ├── handlers/                 # LLM processor, auth integration
│   ├── nephio/                   # Porch client, lifecycle manager, blueprint validator, KRM pipeline
│   ├── oran/                     # O-RAN adapters: A1 (a1_adaptor), E2 (asn1_codec), O1 (NETCONF), O2 (IMS API)
│   ├── auth/                     # JWT, LDAP, AzureAD, OIDC providers
│   ├── security/                 # CA, mTLS, certificate automation, revocation
│   ├── monitoring/               # Health aggregator, latency tracking, SLA alerting
│   ├── telecom/                  # Knowledge base for telecom domain
│   ├── templates/                # Template engine for KRM generation
│   ├── git/                      # Git client interface (go-git)
│   ├── config/                   # Configuration constants
│   ├── llm/                      # LLM client, circuit breaker, caching
│   ├── rag/                      # RAG pipeline types
│   ├── porch/                    # Porch integration helpers
│   ├── blueprints/               # O-RAN blueprint manager
│   ├── optimization/             # Performance analysis engine
│   ├── disaster/                 # Backup manager
│   ├── resilience/               # Circuit breaker, retry logic
│   ├── validation/               # YANG validation
│   ├── webhooks/                 # Admission webhooks
│   ├── webui/                    # Go-based web UI API server
│   └── ... (56 top-level packages total)
│
├── config/                       # Kustomize overlays + CRDs + RBAC + webhook config
│   ├── crd/intent/               # VALID CRD (intent.nephoran.com_networkintents.yaml)
│   ├── crd/bases/                # 30+ CRD files (many have K8s 1.35 schema issues — use with caution)
│   ├── rbac/                     # Roles, bindings, service accounts
│   ├── webhook/                  # Validating + mutating webhook config
│   ├── manager/                  # Controller manager deployment
│   └── security/                 # Network policies, pod security
│
├── manifests/                    # Minimal: nephio/repository.yaml, oran/namespace.yaml
├── kpt-packages/                 # Nephio kpt packages (coredns-caching, ipam, nf-injector, 5gc, porch, webui)
├── gitops/                       # ArgoCD application.yaml, FluxCD git-repository + kustomization
├── monitoring/                   # Grafana dashboards (5 JSON), Prometheus rules, ServiceMonitors
├── metrics/                      # Sample E2 KPM metrics JSON files
├── testdata/                     # Test intent JSONs (valid, invalid, security edge cases)
│
├── scripts/                      # Deployment scripts
│   ├── setup-kind-cluster.sh
│   ├── install-nephio.sh
│   ├── install-ric.sh
│   ├── deploy-oai-cnfs.sh
│   └── seed-e2sim-traffic.sh
│
├── tools/                        # Dev tools
│   ├── kmpgen/                   # KPM metrics generator
│   ├── vessend/                  # VES event sender
│   ├── flaky-detector/           # Test flaky detector
│   └── verify-scale/             # Scale verification
│
├── bin/                          # Pre-built binaries (conductor, intent-ingest, manager, rag-pipeline, webhook)
├── ric-platform/                 # RIC deployment scripts + submodule refs (appmgr, e2, e2mgr, rtmgr, submgr)
├── docs/                         # ADRs (10 + 7), API OpenAPI spec, SDD, runbooks
├── hack/                         # bootstrap-kind.sh, e2e.sh
└── .claude/                      # Claude Code config (agents, skills, hooks, prompts)
```

---

## 4. Data Flow

```
User (NL text)
     │
     ▼
intentctl (CLI)  ──or──  Web UI (port 30090)  ──or──  Intent Ingest Service
     │                                                        │
     ▼                                                        ▼
intentd (TMF921 API)  ◄──────────────────────────────  Ollama (llama3.1)
     │
     ▼
Planner (stub / LLM)  →  IntentPlan.json (schema-validated)
     │
     ▼
Generator  →  kpt packages / KRM YAML
     │
     ▼
Validator (OPA + kubeconform + naming check)
     │
     ▼
Git commit → PR → CI checks → merge
     │
     ▼
Porch (draft → proposed → published) + ConfigSync
     │
     ▼
Workload cluster (Free5GC, RIC, xApps, CNFs)
     │
     ▼
Observe (Prometheus + KPIMON) → Analyze (LLM) → Act (new PR)  ← Closed Loop
```

---

## 5. IntentPlan Schema

File: `schemas/intent-plan.schema.json` (Draft 2020-12, strict `additionalProperties: false`)

| Field | Type | Required | Description |
|---|---|---|---|
| `intentId` | string | ✅ | Pattern: `^intent-[0-9]{8}-[0-9]{4}$` |
| `intentType` | enum | ✅ | `slice.deploy`, `slice.scale`, `closedloop.act`, `config.update` |
| `actions[]` | array | ✅ | min 1 item |
| `actions[].kind` | enum | ✅ | `deploy`, `scale`, `configure`, `promote`, `rollback` |
| `actions[].component` | enum | ✅ | 11 allowlisted: `oai-odu`, `oai-ocu`, `oai-cu-cp`, `oai-cu-up`, `free5gc-upf`, `free5gc-smf`, `free5gc-amf`, `ric-kpimon`, `ric-ts`, `sim-e2`, `trafficgen` |
| `actions[].replicas` | int | if kind=scale | 0–200 |
| `policy` | object | ✅ | `requireHumanReview` + `guardrails` (deny cluster-scoped, privileged, hostNetwork, CRD changes, RBAC changes) |
| `slice` | object | — | `sliceType` (eMBB/URLLC/mMTC/shared), `name`, `site`, `targets` |
| `constraints` | object | — | `maxReplicas`, `minReplicas`, `allowedNamespaces`, `forbiddenKinds` |
| `metadata` | object | — | `createdAt`, `createdBy`, `source` (cli/web/tmf921) |

---

## 6. Naming & Labeling Standard

Resource name: `<domain>-<component>-<site>-<slice>-<instance>`
- domain: `ran|core|ric|sim|obs`
- component: `odu|ocu|cu-cp|cu-up|upf|amf|smf|kpimon|ts|e2sim|trafficgen`
- site: `edge01|edge02|regional|central|lab`
- slice: `embb|urllc|mmtc|shared`
- instance: `i001|i002|...`

Required labels on all managed resources:
```yaml
oran.ai/intent-id: "<intent-id>"
oran.ai/slice: "<slice-type>"
oran.ai/site: "<site>"
oran.ai/component: "<component>"
app.kubernetes.io/managed-by: nephio
app.kubernetes.io/part-of: llm-nephio-oran
```

---

## 7. Development

### 7.1 Key Binaries (from Dockerfile)

| Binary | Source | Description |
|---|---|---|
| `manager` | `cmd/main.go` | K8s operator controller-manager (NetworkIntent reconciler) |
| `conductor-loop` | `cmd/conductor-loop/main.go` | File-watching orchestration pipeline |
| `intent-ingest` | `cmd/intent-ingest/main.go` | HTTP service: NL text → intent JSON → file handoff |
| `llm-processor` | `cmd/llm-processor/main.go` | LLM processing service |
| `porch-direct` | `cmd/porch-direct/main.go` | Direct Porch API integration |
| `conductor` | `cmd/conductor/main.go` | Conductor orchestrator |
| `a1-sim` | `cmd/a1-sim/main.go` | A1 interface simulator |
| `e2-kpm-sim` | `cmd/e2-kpm-sim/main.go` | E2 KPM metrics simulator |
| `fcaps-sim` | `cmd/fcaps-sim/main.go` | FCAPS/VES event simulator |

### 7.2 Environment Variables

| Variable | Default | Description |
|---|---|---|
| `LLM_BACKEND` | `ollama` | `vllm` or `ollama` |
| `LLM_MODEL` | `Qwen/Qwen2.5-Coder-32B-Instruct` | Model ID for vLLM; `llama3.1:latest` for Ollama |
| `LLM_BASE_URL` | `http://localhost:8000/v1` | LLM API base URL |
| `GIT_REPO_URL` | (required) | GitOps target repository |
| `NEPHIO_PORCH_URL` | `http://porch-server.porch-system:7007` | Porch API endpoint |
| `PROMETHEUS_URL` | `http://prometheus:9090` | Prometheus endpoint |
| `ENABLE_LLM_INTENT` | `false` | Enable LLM-based intent processing in operator |

### 7.3 Testing

```bash
go test ./pkg/...                    # Unit tests (91 packages pass)
go test ./test/envtest/...           # envtest with live cluster (UseExistingCluster: true)
go test ./test/integration/...       # Integration tests (need Porch, cluster services)
python -m pytest llm_nephio_oran/    # Python unit tests
```

Known test constraints:
- envtest uses `config/crd/intent/` (not `config/crd/bases/` which has schema issues)
- `test/integration/porch` has build error (`ScalingConfig` should be `ScalingParameters`)
- Several test suites need running infrastructure (Porch, RAG, TLS server, chaos infra)

### 7.4 Cluster Access

```bash
kubectl get nodes                     # thc1006-ubuntu-22 (single node)
kubectl get pods -n nephoran-system   # Operator + UIs
kubectl get networkintents -A         # All NetworkIntent CRDs
kubectl get packagerevisions          # Porch catalog
```

### 7.5 External Access

| Service | URL |
|---|---|
| Nephoran Web UI | `http://192.168.10.65:30090` |
| Grafana | `http://192.168.10.65:30300` |
| Free5GC WebUI | `http://192.168.10.65:30500` |
| Gitea SSH | `ssh://192.168.10.65:30022` |
| RIC Kong | `http://192.168.10.65:32080` |
| O1 NETCONF | `192.168.10.65:30830` |
| Ngrok tunnel | `https://lennie-unfatherly-profusely.ngrok-free.dev` → localhost:8888 |

---

## 8. Milestones

| Milestone | Description | Status |
|---|---|---|
| **M0** | Repo + CI + schema validation + dev tooling | ✅ Done |
| **M1** | intentctl CLI + intentd API + plan validation + stub planner | ✅ Done (`llm_nephio_oran/`) |
| **M2** | Porch/ConfigSync integration | ✅ Done (live, syncing) |
| **M3** | Observability scaffolding + traffic generator | ⚠️ Partial (Prometheus/Grafana deployed; traffic gen stub) |
| **M4** | RIC integration + KPIMON | ⚠️ Partial (RIC deployed; KPIMON scaled-to-0; scaling xApp running) |
| **M5** | Traffic Steering scaffolding | 🔲 Not started |
| **M6** | O-RAN CNF packages + E2SIM | ⚠️ Partial (e2-test-client deployed; OAI gNB crashes) |
| **M7** | Closed-loop demo: observe → analyze → scale PR | 🔲 Not started |

---

## 9. What to Do Next

1. **Wire planner to local LLM**: Replace `stub_planner.py` with Ollama/vLLM call using prompt templates from `src/llm-gateway/prompt_templates/`.
2. **Wire generator to produce real kpt packages**: Replace `kpt_generator.py` stub with actual KRM YAML generation using `pkg/templates/engine.go`.
3. **Wire intentd → generator → git commit → PR**: Connect the pipeline end-to-end through `gitops/pr_stub.py`.
4. **Fix KPIMON**: Scale KPIMON xApp back up, verify E2 metrics collection.
5. **Implement closed-loop**: Prometheus query → LLM `/analyze` → scaling IntentPlan → Git PR.
6. **GPU passthrough for Ollama**: Configure DRA ResourceClaim so in-cluster Ollama can use RTX 5080.
7. **Clean up pkg/**: 55 top-level packages, 607K lines — consolidate duplicates (internal/porch + pkg/porch, internal/llm + pkg/llm), refactor monolithic files (4000+ line files).

---

## 10. Codebase Stats

| Metric | Value |
|---|---|
| Go files | 927 (761 source + 166 test) |
| Go source lines | 607,497 |
| Go test lines | 89,174 |
| Python files | 31 |
| Python lines | 2,130 |
| YAML/config files | 241 |
| pkg/ top-level packages | 55 |
| Helm releases (live) | 23 |
| CRDs (installed) | 62 |
| K8s namespaces | 35 |
| Running pods | ~82 |
