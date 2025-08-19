---
description: Append concise progress notes to docs/PROGRESS.md
allowed-tools: Read, Write, Edit, Bash(git add:*), Bash(git commit:*)

argument-hint: [branch] [tag] ["note"]
---

Append a one-line entry to docs/PROGRESS.md with timestamp, branch, tag, and note:
[YYYY-MM-DD HH:mm] [$ARGUMENTS]

Stage and commit "docs: progress update [$ARGUMENTS]".
