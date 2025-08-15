#!/usr/bin/env python
import json, sys, re, subprocess

payload = sys.stdin.read()
try:
    data = json.loads(payload)
except Exception:
    data = {}

cmd = (data.get("tool", {}) or {}).get("args", "")
# 粗略阻擋危險操作與誤分支
danger = [
    r"git\s+push\s+--force",
    r"rm\s+-rf\s+\/",
    r"kubectl\s+delete\s+.*\s+--all(\s|$)"
]
for pat in danger:
    if re.search(pat, cmd, re.IGNORECASE):
        print(f"[guard] blocked dangerous command: {cmd}", file=sys.stderr)
        sys.exit(2)  # 非 0 代表阻擋

# 檢查目前分支（避免在 main 或 integrate/mvp 上做破壞性動作）
try:
    branch = subprocess.check_output(["git","branch","--show-current"], text=True).strip()
except Exception:
    branch = ""
if branch in ("main", "integrate/mvp"):
    if re.search(r"\bgit\s+(commit|merge|rebase|push)\b", cmd):
        print(f"[guard] on protected branch '{branch}', refusing: {cmd}", file=sys.stderr)
        sys.exit(2)

# 放行
sys.exit(0)
