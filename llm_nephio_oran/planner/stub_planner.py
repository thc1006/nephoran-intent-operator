from __future__ import annotations

import re
import threading
from datetime import datetime, timezone


# Allowlisted components and their domain mapping
_COMPONENTS = {
    "oai-odu": "ran", "oai-ocu": "ran", "oai-cu-cp": "ran", "oai-cu-up": "ran",
    "free5gc-upf": "core", "free5gc-smf": "core", "free5gc-amf": "core",
    "ric-kpimon": "ric", "ric-ts": "ric", "sim-e2": "sim", "trafficgen": "sim",
}

# Thread-safe sequence counter for unique intentId generation
_seq_lock = threading.Lock()
_seq_counter = 0


def _next_intent_id() -> str:
    """Generate a unique intentId: intent-YYYYMMDD-NNNN."""
    global _seq_counter
    with _seq_lock:
        _seq_counter += 1
        seq = _seq_counter
    today = datetime.now(timezone.utc).strftime("%Y%m%d")
    return f"intent-{today}-{seq:04d}"


def _extract_component(text: str) -> str:
    """Extract a known component name from text, or default to trafficgen."""
    text_l = text.lower()
    for comp in _COMPONENTS:
        if comp in text_l:
            return comp
    return "trafficgen"


def _extract_replicas(text: str) -> int:
    """Extract replica count from text like 'to 3 replicas', or default to 2."""
    m = re.search(r"(?:to\s+)?(\d+)\s*replica", text, re.IGNORECASE)
    if m:
        return max(1, min(200, int(m.group(1))))
    return 2


def _extract_threshold(text: str) -> int:
    """Extract threshold percentage from text like 'threshold 30%', or default to 5."""
    m = re.search(r"threshold\s+(\d+)\s*%?", text, re.IGNORECASE)
    if m:
        return max(0, min(100, int(m.group(1))))
    return 5


def plan_from_text(text: str) -> dict:
    """MVP deterministic mapping. Parses component and replicas from text."""
    intent_id = _next_intent_id()

    text_l = text.lower()
    component = _extract_component(text)
    domain = _COMPONENTS.get(component, "sim")

    if "scale" in text_l:
        intent_type = "slice.scale"
        replicas = _extract_replicas(text)
        actions = [
            {
                "kind": "scale",
                "component": component,
                "replicas": replicas,
                "naming": {"domain": domain, "site": "lab", "slice": "embb", "instance": "i001"},
            }
        ]
    elif "configure" in text_l or "traffic steer" in text_l or "threshold" in text_l:
        intent_type = "closedloop.act"
        threshold = _extract_threshold(text)
        actions = [
            {
                "kind": "configure",
                "component": "ric-ts",
                "params": {"threshold": threshold},
                "naming": {"domain": "ric", "site": "lab", "slice": "embb", "instance": "i001"},
            }
        ]
    else:
        intent_type = "slice.deploy"
        actions = [
            {
                "kind": "deploy",
                "component": component,
                "replicas": 1,
                "naming": {"domain": domain, "site": "lab", "slice": "embb", "instance": "i001"},
            },
        ]

    plan = {
        "intentId": intent_id,
        "intentType": intent_type,
        "description": text,
        "slice": {
            "sliceType": "eMBB",
            "name": "embb-slice-1",
            "site": "lab",
            "targets": {"clusters": ["ran-lab-1"], "namespaces": ["oran", "ric", "core", "sim"]},
        },
        "constraints": {"minReplicas": 1, "maxReplicas": 5, "allowedNamespaces": ["oran", "ric", "core", "sim"]},
        "actions": actions,
        "policy": {
            "requireHumanReview": True,
            "guardrails": {
                "denyClusterScoped": True,
                "denyPrivileged": True,
                "denyHostNetwork": True,
                "denyCRDChanges": True,
                "denyRBACChanges": True,
            },
        },
        "metadata": {
            "createdAt": datetime.now(timezone.utc).replace(microsecond=0).isoformat() + "Z",
            "createdBy": "intentctl",
            "source": "cli",
        },
    }
    return plan
