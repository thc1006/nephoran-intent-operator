---
description: Pre-push self-review gate
allowed-tools: Read, Glob, Grep, Bash(git status:*), Bash(git diff:*), Bash(go test:*), Bash(go vet:*)

---

Run:
- !`git status`
- !`git diff --stat origin/${BRANCH:-HEAD}`
- !`go vet ./...`

Checklist:
- Scope creep? Only files planned in /kickoff touched.
- Tests added/updated? Contracts respected (docs/contracts/*).
- No accidental deletions of other modules’ code.
- Commit messages atomic and conventional?

If anything fails, propose concrete fix or open a TODO with file+line.
End with: “Safe to push” or “Need fix: <1-line>”.
