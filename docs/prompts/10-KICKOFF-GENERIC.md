# KICKOFF (Generic Module)

Read **@CLAUDE_AGENTS_ANALYSIS.md** and `.claude/agents/*` first, then apply the Agent Bootstrap plan.

Context:
- Base branch: `integrate/mvp`
- Do not modify other modules or delete unrelated files.
- Contracts live in `docs/contracts/*`.
- Concurrency protection exists in CI; keep PRs small and isolated.

Your task:
1) State the **module**, **goal**, **explicit in-scope files/dirs**, **out-of-scope**.
2) Propose the smallest working change (≤100 lines diff) with tests.
3) List exact commands to build/test/run on Linux.
4) Apply changes and show **unified diffs** for each file you touch.
5) Run tests and paste terminal outputs (trim noise).
6) Append one line to `docs/PROGRESS.md` (format: `| <ISO-8601> | <branch> | <module> | <summary> |`).
7) Prepare a PR description (title + bullet points + verification steps).

Guardrails:
- If a contract change is required, stop and produce a separate PR plan.
- If a dependency upgrade is needed, propose the minimal version bump and explain compatibility.

Acceptance:
- `go test ./...` passes (or relevant tests if not Go).
- Smoke test commands run successfully and produce expected artifacts.
- Diff is limited to the declared module paths.
