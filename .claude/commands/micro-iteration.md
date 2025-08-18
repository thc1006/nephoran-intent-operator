---
description: Single, minimal change with one test or check
allowed-tools: Read, Write, Edit, Grep, Glob, Bash(go test:*), Bash(go build:*), Bash(git status:*), Bash(git add:*), Bash(git commit:*)

argument-hint: implement [short-title]
---

Context:
- Work only on the files listed in the kickoff plan for this branch.
- After edits, run `go build` or the exact test command from the plan.
- If build/test fails, show me the failing snippet and propose ONE fix.

Your task:
- Make the smallest possible change to achieve the stated goal: $ARGUMENTS
- Then run the designated check.
- If success, stage and create ONE atomic commit with a descriptive message.
- Stop and say: “Ready for /self-review.”
