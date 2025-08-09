# Nephoran Hardening TODO

## P1 — Minimal CRD + Controller (“NetworkIntent”)
**Goal**: Add MVP CRD + controller that reads `spec.intent`, updates `status`, optional LLM call via HTTP.
**Acceptance**:
- `make gen && make build` compiles.
- envtest: NI created → `status.phase` transitions (`Validated`→`Processed` when LLM enabled).
- RBAC markers present for status updates.
**Prompt**:
You are Claude Code with terminal + workspace access. Repo root is CWD.
Constraints: no unnecessary renames; satisfy Acceptance; run `make gen && make build && go test ./...`; branch `feat/crd-controller`.

Task:
1) Create `api/v1/networkintent_types.go`:
- Type NetworkIntent with `Spec{ Intent string }` and `Status{ ObservedGeneration int64, Phase string, LastMessage string, LastUpdateTime metav1.Time }`.
- Kubebuilder markers: `+kubebuilder:object:root=true`, `+kubebuilder:subresource:status`, `+kubebuilder:resource:path=networkintents,scope=Namespaced,shortName=ni`.
2) Create `controllers/networkintent_controller.go`:
- Reconcile: get NI; if `spec.intent==""` → `status.phase="Error"` with message; else set `phase="Validated"`, update `ObservedGeneration`, `LastUpdateTime`.
- If `ENABLE_LLM_INTENT=true`, POST to LLM `/process` JSON `{intent}` with 15s timeout; on 2xx set `phase="Processed"`, else `phase="Error"`.
- RBAC markers for get;list;watch;update status.
3) `cmd/main.go`: register `api/v1` scheme; flags/env: `--enable-network-intent` (default true), `--enable-llm-intent` (default false). Conditionally set up controller.
4) Add `make gen` (controller-gen). Commit generated CRD under `deployments/crds/`.

## P2 — Harden LLM client + validator (timeouts, retries, bounded cache)
**Goal**: Add timeouts, capped retries, strict JSON validation, LRU capacity.
**Acceptance**:
- Timeout via `LLM_TIMEOUT_SECS` (default 15).
- Retries via `LLM_MAX_RETRIES` (default 2).
- Non-JSON rejected.
- Validator reports missing fields.
- Cache capacity via `LLM_CACHE_MAX_ENTRIES` (default 512) with LRU.
- Truncated bodies only at Debug.
**Prompt**:
Task: Improve `pkg/llm` robustness without renaming public types/methods.
1) Wrap `ProcessIntent` with `context.WithTimeout` from env `LLM_TIMEOUT_SECS=15`.
2) Respect env `LLM_MAX_RETRIES=2` in existing exponential backoff.
3) Force `Content-Type: application/json`; reject non-JSON responses.
4) Keep required fields `{type,name,namespace,spec}`; on failure return list of missing fields.
5) Add LRU with capacity `LLM_CACHE_MAX_ENTRIES=512` inside existing cache struct (no API changes).
6) Logging: no bodies at Info; at Debug truncate to 1000 chars.
Add tests: timeout, retry success, fallback success, validator failure, cache eviction.

## P3 — HTTP server hardening (LLM processor)
**Goal**: Add safe defaults without route changes.
**Acceptance**:
- Body limited to `HTTP_MAX_BODY` (default 1048576).
- Security headers set.
- `/metrics` gated by `METRICS_ENABLED`.
- Optional IP allowlist for metrics.
**Prompt**:
Add middleware to limit body by `HTTP_MAX_BODY` (default 1048576). Add headers: HSTS (TLS only), `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, CSP `default-src 'none'`. Only register `/metrics` when `METRICS_ENABLED=true`. If `METRICS_ALLOWED_IPS` set (comma list), enforce allowlist on `/metrics`. Add tests for 413 and metrics gating.

## P4 — Optional RAG via build tag + noop adapter
**Goal**: Build without Weaviate unless `-tags=rag`.
**Acceptance**:
- `go build` (no tags) OK with noop RAG.
- `go build -tags=rag` OK with real client.
**Prompt**:
Define `RAGClient` in `pkg/rag/types.go`: `Retrieve(ctx context.Context, query string) ([]Doc, error)`.
Real Weaviate impl under `//go:build rag` (e.g., `pkg/rag/weaviate_client.go`).
No-op impl (no build tags) in `pkg/rag/noop/client.go` returns empty results.
In LLM `processWithLLMBackend`, for `backendType=="rag"` use `RAGClient`; default build uses noop; `-tags=rag` uses real client.

## P5 — Prune go.mod; pin essentials
**Goal**: Remove unused deps; pin only required; set real Go version.
**Acceptance**:
- `go mod verify` clean.
- `go list all` OK.
- `govulncheck` no high severity.
**Prompt**:
Remove unused imports from code, run `go mod tidy`. Pin `sigs.k8s.io/controller-runtime` (K8s 1.29/1.30 compatible) and align `k8s.io/*` versions accordingly. Keep `golang.org/x/oauth2` only if imported. Update `go` version in `go.mod` to actual toolchain (e.g., `1.22`). Run `go mod verify && go list all && make build && go test ./...`.

## P6 — Unify Makefile; remove Makefile.go124
**Goal**: Single `Makefile` used in CI/dev.
**Acceptance**:
- `make build`, `make test`, `make gen`, `make lint`, `make security-scan` work.
- No hardcoded non-existent Go 1.24 toolchain.
**Prompt**:
Delete `Makefile.go124`. Keep one `Makefile` with targets: `build`, `test`, `vet`, `lint` (golangci-lint), `security-scan` (govulncheck), `gen` (controller-gen), `docker-build`, `docker-push`. Remove hardcoded toolchain; rely on local `go`. Run `make help && make build && make test`.

## P7 — Minimal CI reflecting reality
**Goal**: CI compiles, generates CRDs, tests, lints, vuln-scans.
**Acceptance**:
- Workflow passes without skipping controller setup.
- Fails on linter/vuln findings.
**Prompt**:
Update `.github/workflows/ci.yaml`:
- Steps: checkout; setup-go (from `go.mod`); cache; `make gen`; `make build`; `make test`; `golangci-lint`; `govulncheck`.
- Fail on linter/vuln errors.
- Exclude e2e for now.

## P8 — Env/flag plumbing
**Goal**: Centralize env defaults; keep existing helpers.
**Acceptance**:
- Tests confirm env flips behavior as intended.
**Prompt**:
Add env vars with defaults and wire up:
- `ENABLE_NETWORK_INTENT=true`
- `ENABLE_LLM_INTENT=false`
- `LLM_TIMEOUT_SECS=15`
- `LLM_MAX_RETRIES=2`
- `LLM_CACHE_MAX_ENTRIES=512`
- `HTTP_MAX_BODY=1048576`
- `METRICS_ENABLED=false`
- `METRICS_ALLOWED_IPS=""`
Integrate via the existing env helper file; add unit tests for each.

## P9 — Metrics matching current code
**Goal**: Minimal Prometheus metrics; gated by env.
**Acceptance**:
- LLM client: requests/errors/cache hits-misses/fallbacks/retries + duration histogram/summary.
- Controller: reconcile counts/errors via controller-runtime.
- `/metrics` only when `METRICS_ENABLED=true`.
**Prompt**:
Add LLM client counters: `requests_total`, `errors_total`, `cache_hits_total`, `cache_misses_total`, `fallback_attempts_total`, `retry_attempts_total`; add `processing_duration_seconds` histogram/summary. Expose controller reconcile metrics via controller-runtime. Register metrics only when `METRICS_ENABLED=true`. Add a basic scrape test.

## P10 — README alignment with MVP
**Goal**: Truthful first screen; simulations clearly marked; roadmap separated.
**Acceptance**:
- README describes CRD+controller+LLM (+optional RAG); NF simulation explicit.
- Quickstart matches code.
**Prompt**:
Rewrite README top section: replace TRL/SLA/full O-RAN claims with MVP scope (CRD+controller, LLM processor, optional RAG, NF simulated). Keep Quickstart; ensure commands reflect current project. Add “Roadmap” for future O-RAN interfaces, GitOps flows, xApps.

## P11 — Kustomize bases (controller + LLM)
**Goal**: Manifests carry new flags/envs.
**Acceptance**:
- `kubectl apply -k .../network-intent-controller/` deploys healthy.
- LLM processor Deployment includes hardened envs.
**Prompt**:
Update `deployments/kustomize/base/llm-processor` to include `HTTP_MAX_BODY`, `METRICS_ENABLED`, `LLM_TIMEOUT_SECS` envs and optional ServiceMonitor annotation. Create `deployments/kustomize/base/network-intent-controller` with Deployment, RBAC, SA, and flags `ENABLE_NETWORK_INTENT`, `ENABLE_LLM_INTENT`; default namespace `nephoran-system`. Verify with `kubectl apply -k`.

## P12 — Dead asset cleanup / quarantine
**Goal**: Remove obsolete files; quarantine unused scripts.
**Acceptance**:
- `rg -n "Makefile.go124"` shows no active references.
- Build/tests pass.
**Prompt**:
Delete `Makefile.go124`. Move unused scripts/endpoints to `archive/` if unreferenced. Update or remove references. Verify no dangling refs with ripgrep; then `make build && make test`.
