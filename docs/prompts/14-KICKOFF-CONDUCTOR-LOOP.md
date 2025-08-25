# KICKOFF — Conductor Loop

Read **@CLAUDE_AGENTS_ANALYSIS.md** and `.claude/agents/*` first, then apply the Agent Bootstrap plan.

Goal:
- Watch a directory (`-handoff`) and invoke the porch publisher to render KRM patches to `-out`, with debounce on Windows.

Scope:
- `cmd/conductor-loop/`, `internal/loop/`.
- Accept flags: `-handoff` (required), `-out` (default `examples/packages/scaling`).
- Do not rely on junctions; use absolute paths.

Deliverables:
- `go run ./cmd/conductor-loop -handoff <absPath> -out <absPath>`
- File-type filter for `intent-*.json` only.
- Unit tests for debounce and path handling.

Acceptance:
- Manual test: POST an intent → detect → run porch → patch created.
- Tests pass; diff limited to loop module.
