# KICKOFF — E2 KPM Simulator

Read **@CLAUDE_AGENTS_ANALYSIS.md** and `.claude/agents/*` first, then apply the Agent Bootstrap plan.

Goal:
- Emit periodic KPM metrics to a local directory for the planner, supporting `-period` and `-duration` flags.

Scope:
- `cmd/e2-kpm-sim/`, `internal/kpm/`.
- Output directory defaults to `metrics/`.

Deliverables:
- CLI: `go run ./cmd/e2-kpm-sim -out metrics -period 1s -duration 90s`
- File format: one JSON per tick, with timestamp + metric fields.
- Unit tests for sampler and filename format.

Acceptance:
- Smoke test runs for ~20–90s, creating files under `metrics/`.
- Test suite green; minimal diff within module paths only.
