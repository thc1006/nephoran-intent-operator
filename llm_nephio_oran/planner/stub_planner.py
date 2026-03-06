from __future__ import annotations

from datetime import datetime


def plan_from_text(text: str) -> dict:
    # MVP deterministic mapping. Replace with local LLM later.
    # The intentId pattern required by schema: intent-YYYYMMDD-NNNN
    today = datetime.utcnow().strftime("%Y%m%d")
    intent_id = f"intent-{today}-0001"

    text_l = text.lower()
    if "scale" in text_l:
        intent_type = "slice.scale"
        actions = [
            {
                "kind": "scale",
                "component": "trafficgen",
                "replicas": 2,
                "naming": {"domain": "sim", "site": "lab", "slice": "embb", "instance": "i001"},
            }
        ]
    else:
        intent_type = "slice.deploy"
        actions = [
            {
                "kind": "deploy",
                "component": "ric-kpimon",
                "replicas": 1,
                "naming": {"domain": "ric", "site": "lab", "slice": "shared", "instance": "i001"},
            },
            {
                "kind": "deploy",
                "component": "trafficgen",
                "replicas": 1,
                "params": {"mode": "iperf3", "durationSeconds": 600},
                "naming": {"domain": "sim", "site": "lab", "slice": "embb", "instance": "i001"},
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
            "createdAt": datetime.utcnow().replace(microsecond=0).isoformat() + "Z",
            "createdBy": "intentctl",
            "source": "cli",
        },
    }
    return plan
