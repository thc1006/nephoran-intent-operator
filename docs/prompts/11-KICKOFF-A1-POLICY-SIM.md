# KICKOFF — A1 Policy Simulator

Read **@CLAUDE_AGENTS_ANALYSIS.md** and `.claude/agents/*` first, then apply the Agent Bootstrap plan.

Goal:
- Implement a minimal A1 policy simulator that accepts JSON (per `docs/contracts/a1.policy.schema.json`) and can map policies to scaling intents for the planner pipeline.

Scope:
- New/updated code under `cmd/a1-policy-sim/`, `internal/a1/`, tests under `internal/a1/...`.
- Write 2–3 sample policies under `handoff/samples/policy/`.
- No changes to porch publisher or conductor-loop.

Deliverables:
- CLI: `go run ./cmd/a1-policy-sim -in <policy.json> -out handoff/`
- Validation against the schema; on error, exit non-zero with concise codes.
- Unit tests: happy path + invalid thresholds + missing fields.

Verification:
- Show diffs, then run: `go test ./...`
- Provide two example invocations; confirm files written under `handoff/`.

Acceptance:
- Policy → intent mapping logic covered by tests.
- Schema validation errors are human + machine friendly.
