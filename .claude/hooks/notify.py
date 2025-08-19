#!/usr/bin/env python
import json, sys
payload = sys.stdin.read()
try:
    data = json.loads(payload)
except Exception:
    data = {}
tool = (data.get("tool") or {}).get("name","?")
args = (data.get("tool") or {}).get("args","")
print(f"[notify] {tool}: {args}")
sys.exit(0)
