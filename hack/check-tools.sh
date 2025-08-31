#!/usr/bin/env bash
set -euo pipefail

need() {
  local n="$1"
  if command -v "$n" >/dev/null 2>&1; then
    echo "✅ $n = $(command -v "$n")"
  elif command -v "$n.exe" >/dev/null 2>&1; then
    echo "✅ $n.exe = $(command -v "$n.exe")"
  else
    echo "❌ missing: $n (or $n.exe) in this shell's PATH" >&2
    return 1
  fi
}

need kind
need kubectl

# Check for kustomize - either standalone or built into kubectl
echo -n "Checking kustomize: "
if command -v kustomize >/dev/null 2>&1; then
  echo "✅ standalone kustomize = $(command -v kustomize)"
elif command -v kustomize.exe >/dev/null 2>&1; then
  echo "✅ standalone kustomize.exe = $(command -v kustomize.exe)"
elif kubectl version --client 2>/dev/null | grep -q "Kustomize Version"; then
  KUSTOMIZE_VERSION=$(kubectl version --client 2>/dev/null | grep "Kustomize Version" | awk '{print $3}')
  echo "✅ kubectl has built-in kustomize ${KUSTOMIZE_VERSION} (use 'kubectl apply -k')"
else
  echo "❌ missing: kustomize (install standalone or use kubectl with built-in support)" >&2
fi
