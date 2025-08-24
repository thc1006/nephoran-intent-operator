#!/usr/bin/env python
import json, sys, subprocess, os, shlex

payload = sys.stdin.read()
try:
    data = json.loads(payload)
except Exception:
    data = {}

# Claude 會傳遞這次 Write/Edit 影響的檔案陣列（依官方介面）
files = []
for f in (data.get("files") or []):
    p = f.get("path") or f.get("file") or ""
    if p:
        files.append(p)

go_changed = [f for f in files if f.endswith(".go")]
json_changed = [f for f in files if f.endswith(".json") and f.startswith("docs/contracts/")]

def run(cmd_args):
    """Execute command safely without shell=True"""
    try:
        # Split command into array if it's a string to avoid shell=True
        if isinstance(cmd_args, str):
            # Parse shell command into safe argument list
            import shlex
            cmd_list = shlex.split(cmd_args)
        else:
            cmd_list = cmd_args
        
        out = subprocess.check_output(cmd_list, stderr=subprocess.STDOUT, text=True, shell=False)
        return 0, out
    except subprocess.CalledProcessError as e:
        return e.returncode, e.output

msgs = []

if go_changed:
    # Build gofmt command as array to avoid shell injection
    gofmt_cmd = ["gofmt", "-l", "-w"] + go_changed
    code, out = run(gofmt_cmd)
    msgs.append(f"[gofmt] exit={code}\n{out or ''}")
    code2, out2 = run(["go", "vet", "./..."])
    msgs.append(f"[go vet] exit={code2}\n{out2 or ''}")
    if code2 != 0:
        print("\n".join(msgs), file=sys.stderr)
        sys.exit(3)

if json_changed:
    code3, out3 = run(["ajv", "compile", "-s", "docs/contracts/intent.schema.json"])
    msgs.append(f"[ajv compile intent] exit={code3}\n{out3 or ''}")
    # 也可在這裡針對單檔驗證樣本 JSON（如你專案另有 examples）
    if code3 != 0:
        print("\n".join(msgs), file=sys.stderr)
        sys.exit(4)

print("\n".join(msgs))
sys.exit(0)
