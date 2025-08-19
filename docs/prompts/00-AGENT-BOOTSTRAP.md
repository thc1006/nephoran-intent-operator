# Agent Bootstrap (Read First)

Read **@CLAUDE_AGENTS_ANALYSIS.md** and all files under `.claude/agents/` first.
Choose the most relevant agents for this task and describe your routing plan in one short paragraph before you edit anything.

Agent routing rules:
- Default: `nephoran-code-analyzer` to understand; `golang-pro` to implement; `code-reviewer` to review.
- O-RAN/Nephio topics: prefer `oran-nephio-dep-doctor`, `oran-network-functions-agent`, `nephio-oran-orchestrator-agent`.
- CI/CD or Docker/K8s: `deployment-engineer`.
- Security checks: `security-compliance-agent` then `security-auditor`.
- Documentation: `nephoran-docs-specialist` and `docs-architect`.
- If multiple agents collaborate, let `context-manager` summarize and hand off.

Output (before coding):
- A 3–5 sentence plan: selected agents + why + exact commands to run next.
