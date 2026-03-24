"""Multi-layer manifest validation per ADR-008.

Validation layers:
1. YAML syntax
2. K8s resource structure (apiVersion, kind, metadata.name)
3. Guardrail policies (no cluster-scoped, no privileged, no hostNetwork)
4. Naming convention: {domain}-{component}-{site}-{slice}-{instance}
5. Required labels check
"""
from __future__ import annotations

import re
from pathlib import Path
from typing import Any

import yaml

NAMING_RE = re.compile(
    r"^(ran|core|ric|sim|obs)-"
    r"(oai-odu|oai-ocu|oai-cu-cp|oai-cu-up|free5gc-upf|free5gc-smf|free5gc-amf|"
    r"ric-kpimon|ric-ts|sim-e2|trafficgen)-"
    r"(edge01|edge02|regional|central|lab)-"
    r"(embb|urllc|mmtc|shared)-"
    r"(i\d{3})$"
)

REQUIRED_LABELS = [
    "oran.ai/intent-id",
    "oran.ai/component",
    "app.kubernetes.io/managed-by",
    "app.kubernetes.io/part-of",
]

CLUSTER_SCOPED_KINDS = {
    "ClusterRole", "ClusterRoleBinding", "Namespace",
    "PersistentVolume", "StorageClass", "CustomResourceDefinition",
    "PriorityClass", "ValidatingWebhookConfiguration", "MutatingWebhookConfiguration",
}

# Namespace-scoped RBAC resources also denied (matches security_guardrails.rego denyRBACChanges)
DENIED_RBAC_KINDS = {"Role", "RoleBinding"}

WORKLOAD_KINDS = {"Deployment", "StatefulSet", "DaemonSet", "Pod", "Job", "CronJob"}


def _load_yamls(path: Path) -> list[tuple[Path, dict[str, Any]]]:
    """Load all YAML files from a directory tree."""
    docs = []
    for f in path.rglob("*.yaml"):
        try:
            with open(f) as fh:
                for doc in yaml.safe_load_all(fh):
                    if doc and isinstance(doc, dict):
                        docs.append((f, doc))
        except yaml.YAMLError as e:
            docs.append((f, {"_parse_error": str(e)}))
    return docs


def validate_package(pkg_dir: Path) -> list[str]:
    """Validate all YAML manifests in a kpt package directory.

    Returns a list of error strings. Empty list means all checks passed.
    """
    errors: list[str] = []
    docs = _load_yamls(pkg_dir)

    if not docs:
        errors.append(f"No YAML files found in {pkg_dir}")
        return errors

    for fpath, doc in docs:
        rel = fpath.relative_to(pkg_dir)

        # Layer 1: YAML syntax
        if "_parse_error" in doc:
            errors.append(f"[L1-YAML] {rel}: {doc['_parse_error']}")
            continue

        # Skip Kptfile (local config, not a K8s resource)
        if doc.get("kind") == "Kptfile":
            continue

        # Layer 2: K8s structure
        if "apiVersion" not in doc:
            errors.append(f"[L2-K8s] {rel}: missing apiVersion")
        if "kind" not in doc:
            errors.append(f"[L2-K8s] {rel}: missing kind")
        meta = doc.get("metadata", {})
        if not meta.get("name"):
            errors.append(f"[L2-K8s] {rel}: missing metadata.name")

        kind = doc.get("kind", "")

        # Layer 3: Guardrail policies
        if kind in CLUSTER_SCOPED_KINDS:
            errors.append(f"[L3-Policy] {rel}: cluster-scoped resource '{kind}' not allowed (denyClusterScoped)")
        if kind in DENIED_RBAC_KINDS:
            errors.append(f"[L3-Policy] {rel}: RBAC resource '{kind}' not allowed (denyRBACChanges)")

        if kind in WORKLOAD_KINDS:
            spec = doc.get("spec", {})
            if kind == "Pod":
                pod_spec = spec
            elif kind == "CronJob":
                pod_spec = spec.get("jobTemplate", {}).get("spec", {}).get("template", {}).get("spec", {})
            else:
                pod_spec = spec.get("template", {}).get("spec", {})

            # Check privileged containers
            for c in pod_spec.get("containers", []):
                sec = c.get("securityContext", {})
                if sec.get("privileged"):
                    errors.append(f"[L3-Policy] {rel}: container '{c.get('name')}' is privileged (denyPrivileged)")

            # Check hostNetwork
            if pod_spec.get("hostNetwork"):
                errors.append(f"[L3-Policy] {rel}: hostNetwork=true not allowed (denyHostNetwork)")

        # Layer 4: Naming convention (for workloads and services)
        name = meta.get("name", "")
        if kind in WORKLOAD_KINDS or kind == "Service":
            if name and not NAMING_RE.match(name):
                # Only warn if it looks like a generated name (has dashes)
                if name.count("-") >= 3:
                    errors.append(f"[L4-Naming] {rel}: name '{name}' doesn't match naming convention")

        # Layer 5: Required labels (for workloads and services)
        if kind in WORKLOAD_KINDS or kind == "Service":
            labels = meta.get("labels", {})
            for lbl in REQUIRED_LABELS:
                if lbl not in labels:
                    errors.append(f"[L5-Labels] {rel}: missing required label '{lbl}'")

    return errors
