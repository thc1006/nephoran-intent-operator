# /fix-one-comment — Micro-iteration template

Task: Fix exactly **ONE** review comment or test failure.

Checklist:
1) Quote the exact comment or test failure you’re fixing.
2) Show the smallest diff to resolve it.
3) Run only the minimal checks (e.g., `go build ./...`, targeted tests).
4) Don’t refactor unrelated code.
5) Write a crisp commit message: `fix(<area>): <what> (refs <issue/PR link>)`.
6) Stop and summarize.

If additional issues appear, STOP and ask to run /fix-one-comment again for the next item.
