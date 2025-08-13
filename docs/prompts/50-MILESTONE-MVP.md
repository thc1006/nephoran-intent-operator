# Milestone — MVP Integration (Smoke Script)

Read **@CLAUDE_AGENTS_ANALYSIS.md** and `.claude/agents/*` first, then apply the Agent Bootstrap plan.

Goal:
- In `integrate/mvp`, produce `scripts/mvp-smoke.ps1` that:
  1) Starts ingest server and conductor-loop (with flags).
  2) Runs e2-kpm-sim for ~20–90s.
  3) Emits a sample O1 VES event.
  4) Submits an A1 policy and a free-text intent.
  5) Verifies files appear in `handoff/` and `examples/packages/scaling/`.
  6) Stops all background processes cleanly.

Deliverables:
- The script (PowerShell) with clear echo logs and exit codes.
- A short README snippet describing expected outputs and how to run.

Acceptance:
- One command runs end-to-end without manual intervention on Windows.
