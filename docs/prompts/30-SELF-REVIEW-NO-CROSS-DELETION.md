# Self-review (No Cross-deletion / Safe PR)

Read **@CLAUDE_AGENTS_ANALYSIS.md** and `.claude/agents/*` first, then apply the Agent Bootstrap plan.

Checklist (produce a short report):
- Scope: confirm all changed paths are within this module only.
- Contracts: confirm `docs/contracts/*` still satisfied.
- Tests: run `go test ./...` (or module tests); paste summary.
- Lint: run linter if configured; paste summary.
- CI: assert our workflow uses `concurrency: { group: ${{ github.ref }}, cancel-in-progress: true }`.
- Risks: list any risk + mitigation in 3 bullets max.
- PR text: title + bullet summary + verification steps.

Output:
- “PASS/FAIL” per checklist item, with one-line evidence.
- If any FAIL, propose the minimal patch to fix it.
