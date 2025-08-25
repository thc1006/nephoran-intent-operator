# Micro-iteration (Single Small Change)

Read **@CLAUDE_AGENTS_ANALYSIS.md** and `.claude/agents/*` first, then apply the Agent Bootstrap plan.

Do exactly one small change (≤50 lines). Steps:
1) Restate the micro-goal in 1–2 sentences.
2) List files you will touch (2–4 max).
3) Apply the change and show unified diffs.
4) Run the smallest relevant tests/commands; paste outputs.
5) Append one line to `docs/PROGRESS.md` via `scripts/add-progress.ps1`.
6) Stop and await confirmation before the next change.

Guardrails:
- No refactors beyond the scope.
- If touching contracts or cross-module files is required, stop and propose a separate PR plan.
