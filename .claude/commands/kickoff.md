# /kickoff — Project warm-up & guardrails

You are Claude Code working in a **monorepo** with multiple worktrees and feature branches.
Goals:
- Read repo context (README, docs/contracts/*.json/md).
- Confirm current branch is NOT `main` and not `integrate/mvp` unless we are merging.
- Summarize module status, then propose a minimal plan for this window only.

**Do first (with tool permission requests as needed):**
1. Read("./README.md")
2. Glob("docs/contracts/**/*")
3. Grep("go.mod", "module ")
4. Bash("git status -sb")
5. Bash("git branch --show-current")
6. If `branch in {main, integrate/mvp}` AND task is destructive, ask to create a feature branch first.

**Outputs:**
- 5–10 bullet points: repo layout & what’s relevant to THIS window.
- A minimal step list for the requested task.
- Ask for permission before running tools beyond what /permissions allows.

**Never:**
- Rewrite other modules outside this window’s scope.
- Force-push or delete branches.
