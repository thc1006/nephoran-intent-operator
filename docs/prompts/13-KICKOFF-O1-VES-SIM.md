# KICKOFF — O1 VES/FCAPS Simulator

Read **@CLAUDE_AGENTS_ANALYSIS.md** and `.claude/agents/*` first, then apply the Agent Bootstrap plan.

Goal:
- Generate minimal VES/FCAPS events aligned with examples in `docs/contracts/fcaps.ves.examples.json` and O1-style management signals (simulated).

Scope:
- `cmd/o1-ves-sim/`, `internal/ves/`, `handoff/samples/ves/`.
- No network calls; write events to disk for downstream ingestion.

Deliverables:
- CLI: `go run ./cmd/o1-ves-sim -out handoff/ves -count 3 -interval 2s`
- Event JSON validated against local examples; fail fast with clear messages.
- Unit tests for payload shape.

Acceptance:
- Three sample events written; tests pass; diff is scoped.
