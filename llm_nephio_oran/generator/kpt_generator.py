"""KRM/kpt package generator from IntentPlan (ADR-003, ADR-006, ADR-0003).

Generates deterministic Kubernetes manifests from IntentPlan actions.
Each action produces a subdirectory with Kptfile + deployment.yaml + service.yaml.
Naming convention: {domain}-{component}-{site}-{slice}-{instance}
Required labels: oran.ai/intent-id, oran.ai/slice, oran.ai/site, oran.ai/component,
                 app.kubernetes.io/managed-by=nephio, app.kubernetes.io/part-of=llm-nephio-oran
"""
from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any

import yaml

logger = logging.getLogger(__name__)

COMPONENT_DEFAULTS: dict[str, dict[str, Any]] = {
    "oai-odu": {"image": "oaisoftwarealliance/oai-gnb:develop", "port": 38472, "protocol": "SCTP", "domain": "ran", "cpu": "2", "memory": "2Gi"},
    "oai-ocu": {"image": "oaisoftwarealliance/oai-gnb:develop", "port": 38472, "protocol": "SCTP", "domain": "ran", "cpu": "2", "memory": "2Gi"},
    "oai-cu-cp": {"image": "oaisoftwarealliance/oai-gnb:develop", "port": 38472, "protocol": "SCTP", "domain": "ran", "cpu": "2", "memory": "2Gi"},
    "oai-cu-up": {"image": "oaisoftwarealliance/oai-gnb:develop", "port": 2152, "protocol": "UDP", "domain": "ran", "cpu": "2", "memory": "2Gi"},
    "free5gc-upf": {"image": "free5gc/upf:v3.3.0", "port": 2152, "protocol": "UDP", "domain": "core", "cpu": "1", "memory": "1Gi"},
    "free5gc-smf": {"image": "free5gc/smf:v3.3.0", "port": 8000, "protocol": "TCP", "domain": "core", "cpu": "500m", "memory": "512Mi"},
    "free5gc-amf": {"image": "free5gc/amf:v3.3.0", "port": 8000, "protocol": "TCP", "domain": "core", "cpu": "500m", "memory": "512Mi"},
    "ric-kpimon": {"image": "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-app-kpimon-go:1.0.1", "port": 8080, "protocol": "TCP", "domain": "ric", "cpu": "250m", "memory": "256Mi"},
    "ric-ts": {"image": "nexus3.o-ran-sc.org:10002/o-ran-sc/ric-app-ts:1.2.1", "port": 8080, "protocol": "TCP", "domain": "ric", "cpu": "250m", "memory": "256Mi"},
    "sim-e2": {"image": "nexus3.o-ran-sc.org:10002/o-ran-sc/sim-e2-interface:1.0.0", "port": 36421, "protocol": "SCTP", "domain": "sim", "cpu": "500m", "memory": "512Mi"},
    "trafficgen": {"image": "networkstatic/iperf3:latest", "port": 5201, "protocol": "TCP", "domain": "sim", "cpu": "250m", "memory": "256Mi"},
}

NAMESPACE_MAP = {
    "ran": "oran",
    "core": "free5gc",
    "ric": "ricplt",
    "sim": "sim",
    "obs": "monitoring",
}


def _resource_name(action: dict[str, Any]) -> str:
    """Generate resource name: {domain}-{component}-{site}-{slice}-{instance}."""
    naming = action.get("naming", {})
    component = action["component"]
    defaults = COMPONENT_DEFAULTS.get(component, {})
    domain = naming.get("domain", defaults.get("domain", "sim"))
    site = naming.get("site", "lab")
    slice_name = naming.get("slice", "embb")
    instance = naming.get("instance", "i001")
    return f"{domain}-{component}-{site}-{slice_name}-{instance}"


def _labels(intent_id: str, action: dict[str, Any]) -> dict[str, str]:
    """Generate required labels per CLAUDE.md §5."""
    naming = action.get("naming", {})
    component = action["component"]
    defaults = COMPONENT_DEFAULTS.get(component, {})
    return {
        "oran.ai/intent-id": intent_id,
        "oran.ai/slice": naming.get("slice", "embb"),
        "oran.ai/site": naming.get("site", "lab"),
        "oran.ai/component": component,
        "app.kubernetes.io/managed-by": "nephio",
        "app.kubernetes.io/part-of": "llm-nephio-oran",
        "app.kubernetes.io/name": component,
    }


def _generate_kptfile(pkg_dir: Path, name: str, intent_id: str) -> None:
    """Generate Kptfile for the package."""
    kptfile = {
        "apiVersion": "kpt.dev/v1",
        "kind": "Kptfile",
        "metadata": {
            "name": name,
            "annotations": {
                "config.kubernetes.io/local-config": "true",
                "nephio.org/intent-id": intent_id,
            },
        },
        "info": {
            "description": f"Generated package for intent {intent_id}",
        },
        "pipeline": {
            "mutators": [
                {
                    "image": "gcr.io/kpt-fn/set-labels:v0.2.0",
                    "configMap": {
                        "oran.ai/intent-id": intent_id,
                        "app.kubernetes.io/managed-by": "nephio",
                    },
                },
            ],
            "validators": [
                {
                    "image": "gcr.io/kpt-fn/kubeval:v0.3.0",
                },
            ],
        },
    }
    (pkg_dir / "Kptfile").write_text(yaml.dump(kptfile, default_flow_style=False, sort_keys=False))


def _generate_deployment(pkg_dir: Path, name: str, intent_id: str, action: dict[str, Any]) -> None:
    """Generate Deployment YAML for a component action."""
    component = action["component"]
    defaults = COMPONENT_DEFAULTS.get(component, {})
    replicas = action.get("replicas", 1)
    labels = _labels(intent_id, action)
    namespace = NAMESPACE_MAP.get(defaults.get("domain", "sim"), "default")

    deployment = {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "metadata": {
            "name": name,
            "namespace": namespace,
            "labels": labels,
        },
        "spec": {
            "replicas": replicas,
            "selector": {
                "matchLabels": {
                    "app.kubernetes.io/name": component,
                    "oran.ai/intent-id": intent_id,
                },
            },
            "template": {
                "metadata": {
                    "labels": labels,
                },
                "spec": {
                    "securityContext": {
                        "runAsNonRoot": True,
                        "seccompProfile": {"type": "RuntimeDefault"},
                    },
                    "containers": [
                        {
                            "name": component,
                            "image": defaults.get("image", "busybox:latest"),
                            "ports": [
                                {
                                    "containerPort": defaults.get("port", 8080),
                                    "protocol": defaults.get("protocol", "TCP"),
                                },
                            ],
                            "resources": {
                                "requests": {
                                    "cpu": defaults.get("cpu", "100m"),
                                    "memory": defaults.get("memory", "128Mi"),
                                },
                                "limits": {
                                    "cpu": defaults.get("cpu", "100m"),
                                    "memory": defaults.get("memory", "128Mi"),
                                },
                            },
                            "securityContext": {
                                "allowPrivilegeEscalation": False,
                                "readOnlyRootFilesystem": True,
                                "capabilities": {"drop": ["ALL"]},
                            },
                        },
                    ],
                },
            },
        },
    }

    # Merge custom params into container env
    params = action.get("params", {})
    if params:
        env_vars = [{"name": k.upper(), "value": str(v)} for k, v in params.items()]
        deployment["spec"]["template"]["spec"]["containers"][0]["env"] = env_vars

    (pkg_dir / "deployment.yaml").write_text(yaml.dump(deployment, default_flow_style=False, sort_keys=False))


def _generate_service(pkg_dir: Path, name: str, intent_id: str, action: dict[str, Any]) -> None:
    """Generate Service YAML for a component."""
    component = action["component"]
    defaults = COMPONENT_DEFAULTS.get(component, {})
    labels = _labels(intent_id, action)
    namespace = NAMESPACE_MAP.get(defaults.get("domain", "sim"), "default")

    service = {
        "apiVersion": "v1",
        "kind": "Service",
        "metadata": {
            "name": name,
            "namespace": namespace,
            "labels": labels,
        },
        "spec": {
            "selector": {
                "app.kubernetes.io/name": component,
                "oran.ai/intent-id": intent_id,
            },
            "ports": [
                {
                    "port": defaults.get("port", 8080),
                    "targetPort": defaults.get("port", 8080),
                    "protocol": defaults.get("protocol", "TCP"),
                },
            ],
        },
    }
    (pkg_dir / "service.yaml").write_text(yaml.dump(service, default_flow_style=False, sort_keys=False))


def generate_kpt_packages(plan: dict[str, Any], out_dir: Path) -> Path:
    """Generate kpt packages from an IntentPlan.

    Creates a package directory under out_dir/{intentId}/ with one
    subdirectory per action, each containing Kptfile + deployment.yaml + service.yaml.

    Args:
        plan: Validated IntentPlan dict.
        out_dir: Root output directory (e.g., packages/instances/).

    Returns:
        Path to the generated instance package directory.
    """
    intent_id = plan["intentId"]
    pkg_root = out_dir / intent_id
    pkg_root.mkdir(parents=True, exist_ok=True)

    # Write the plan itself for traceability
    (pkg_root / "intent-plan.json").write_text(
        json.dumps(plan, indent=2, ensure_ascii=False) + "\n"
    )

    # Generate top-level Kptfile
    _generate_kptfile(pkg_root, intent_id, intent_id)

    actions = plan.get("actions", [])
    manifest_files = []

    for i, action in enumerate(actions):
        name = _resource_name(action)
        action_dir = pkg_root / name
        action_dir.mkdir(parents=True, exist_ok=True)

        kind = action.get("kind", "deploy")

        if kind in ("deploy", "scale", "configure"):
            _generate_kptfile(action_dir, name, intent_id)
            _generate_deployment(action_dir, name, intent_id, action)
            _generate_service(action_dir, name, intent_id, action)
            manifest_files.extend([
                str(action_dir / "Kptfile"),
                str(action_dir / "deployment.yaml"),
                str(action_dir / "service.yaml"),
            ])
            logger.info("Generated package for %s (%s): %s", name, kind, action_dir)
        elif kind in ("promote", "rollback"):
            # For promote/rollback, generate a metadata-only marker
            marker = {
                "apiVersion": "v1",
                "kind": "ConfigMap",
                "metadata": {
                    "name": f"{name}-{kind}-marker",
                    "namespace": NAMESPACE_MAP.get(COMPONENT_DEFAULTS.get(action["component"], {}).get("domain", "sim"), "default"),
                    "labels": _labels(intent_id, action),
                    "annotations": {"nephio.org/action": kind},
                },
                "data": {"intent-plan": json.dumps(plan, ensure_ascii=False)},
            }
            (action_dir / f"{kind}-marker.yaml").write_text(
                yaml.dump(marker, default_flow_style=False, sort_keys=False)
            )
            manifest_files.append(str(action_dir / f"{kind}-marker.yaml"))
            logger.info("Generated %s marker for %s: %s", kind, name, action_dir)

    logger.info("Generated %d manifests for intent %s under %s", len(manifest_files), intent_id, pkg_root)
    return pkg_root
