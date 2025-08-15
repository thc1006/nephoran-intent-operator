# /prepare-pr — Create a clean PR

Steps:
1) Show `git status -sb` and current branch.
2) Ensure only intended files are changed.
3) Write PR title & body (What/Why/How/Test/Scope).
4) If gh CLI exists:
   - `gh pr create -B integrate/mvp -t "<title>" -b "<body>"`
   Else:
   - Print a ready-to-paste PR body.

Never include secrets. Never auto-push without explicit permission.
