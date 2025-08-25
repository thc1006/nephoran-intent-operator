# CLAUDE.md

## ⚠️ CRITICAL PROJECT CONTEXT - READ FIRST ⚠️

### 🎯 Project Type: Kubernetes Operator for O-RAN/5G Network Orchestration
This is the **Nephoran Intent Operator** - a cloud-native Kubernetes operator that:
- **Runs ONLY on Linux-based Kubernetes clusters** (production target: Ubuntu)
- **Manages telecommunications network functions** (5G Core, RAN components)
- **Deploys to cloud providers** (AWS EKS, Azure AKS, Google GKE)
- **NOT a desktop application** - pure server-side Kubernetes operator

### 🚫 Platform Requirements - NO CROSS-PLATFORM SUPPORT
- **ONLY Ubuntu Linux testing required** - no Windows/macOS support needed
- **All GitHub Actions workflows MUST run on `ubuntu-latest` only**
- **Remove any cross-platform conditionals** (e.g., `if: runner.os != 'Windows'`)
- **Production deployment target: Linux containers on Kubernetes**
- **DO NOT add Windows or macOS to any CI matrix strategies**

## Always import these prompts
@docs/prompts/00-AGENTS-AND-CLAUDE-BOOT.md
@docs/prompts/01-PROJECT-GUARDRAILS.md
@docs/prompts/10-KICKOFF-GENERIC.md

## Agents
@CLAUDE_AGENTS_ANALYSIS.md

## Project Conductor: Policies & Contracts
Role: Project Conductor (coordination & contracts)

Goal:
- Keep the repository structure stable, maintain interface contracts in docs/contracts/,
- ensure branches are isolated, and create integration PRs without rewriting others' modules.

Scope:
- docs/contracts/, .github/, CODEOWNERS, CI pipelines, Makefile targets, examples/.

Inputs:
- Existing repo, current module layout, and these contracts to create:
  - docs/contracts/intent.schema.json (scaling intent)
  - docs/contracts/a1.policy.schema.json
  - docs/contracts/fcaps.ves.examples.json
  - docs/contracts/e2.kpm.profile.md

Tasks:
1) Add CODEOWNERS mapping: api/, controllers/, pkg/nephio/ → "nephio-team"; sim/, planner/ → "ran-sim-team".
2) Add branch protections documentation and CI concurrency groups:
   - One concurrency group per branch: `${{ github.ref }}`.
3) Provide Make targets:
   - make mvp-scale-up / mvp-scale-down (call kpt/porch or kubectl patch).
4) Write a short README in docs/contracts/ that explains the JSON fields and versioning.

Deliverables:
- PR against branch `integrate/mvp` with the new contracts and CI config.
- .github/workflows/ci.yml using `concurrency` to avoid overlapping runs.

Tests:
- CI passes, JSON schemas validate using ajv-cli.
- Contracts reviewed by module owners via CODEOWNERS.

Guardrails:
- Do not touch module code (api/, controllers/, pkg/nephio/) other than adding read-only references to schemas.
- If a schema change is needed, open a separate PR that updates examples and notifies maintainers.


## Project Mission
Natural language → LLM → structured NetworkIntent → Nephio/Porch → scale out/in CNFs in an O-RAN-like simulated stack (O1/O2/E2/A1 semantics), with FCAPS (VES) events.

## Repository Map & Ownership (read-only for Claude)
- api/, controllers/ → CRDs, webhooks, controller logic
- pkg/nephio/ → kpt/Porch package generation & publishing
- sim/ → NF simulators, E2SIM, VES event sender
- planner/ → rule-based closed-loop planner (KPM/FCAPS → NetworkIntent)
- ops/ → install scripts (Porch, RIC, A1 mediator, VES collector, optional O2)
- docs/contracts/ → JSON schemas, A1 policy types, VES examples (single source of truth)

## Ground Truth & Contracts
- `docs/contracts/intent.schema.json` defines the only accepted JSON for scaling.
- `docs/contracts/a1.policy.schema.json`, `docs/contracts/fcaps.ves.examples.json`.
- Do not change contracts without a dedicated PR that updates examples and version notes.

## Branch & PR Rules (must follow)
- One module per branch (e.g., feat/porch-publisher).
- Small, atomic PRs with tests and examples; never edit modules outside your scope.
- Follow CODEOWNERS and required reviews (protected branches).

## Tooling Policies
- Go 1.24.x; run `go test ./...`, `golangci-lint run`.
- Use Porch/kpt for package lifecycle; do not kubectl-apply raw YAML directly in controllers.
- Never commit secrets.

## Self-Review Checklist (run before every PR)
- Scope-only diff; contracts respected; tests cover success + two failure cases.
- No unrelated deletions or refactors; run `go mod tidy` without breaking other modules.
- Provide a 60s demo script or README for the change.

## Communication to Claude
- Prefer precise tasks: input → output → constraints → tests.
- If a change affects other modules, open an issue + contract PR first.

## Progress updates (append-only)
- After each micro-iteration, append **one line** to `docs/PROGRESS.md`:
  `| <ISO-8601> | <branch> | <module> | <<= 15 words summary> |`
- Use `Get-Date -Format o` for ISO timestamps and `git rev-parse --abbrev-ref HEAD` for the branch.
- Do not rewrite history; append only.
- If a merge conflict happens, keep both lines (duplicate timestamps are acceptable).

- 在各自工作樹
git status
go test ./...
git add -A
git commit -m "feat(<area>): <short message>"
git push -u origin HEAD

# 選擇其一開 PR
# 1) 網頁 Compare & pull request（Base: integrate/mvp, Compare: <your-branch>）
# 2) gh CLI（如果安裝了）
# gh pr create --base integrate/mvp --head <your-branch> --title "<title>" --body "<body>"