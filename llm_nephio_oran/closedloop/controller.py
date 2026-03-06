"""Closed-loop controller — Observe→Analyze→Act cycle (ADR-010).

Observe: Prometheus metrics via metrics.py
Analyze: MetricsAnalyzer with guard rails
Act: Generate IntentPlan → kpt packages → Git PR (NEVER direct kubectl, per ADR-0004)

The controller runs periodically (default 15s scrape interval) and proposes
scaling actions as Git PRs through the existing pipeline.
"""
from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

from llm_nephio_oran.closedloop.analyzer import MetricsAnalyzer
from llm_nephio_oran.observability.metrics import snapshot_metrics, check_scaling_thresholds
from llm_nephio_oran.planner.llm_planner import plan_from_text
from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages
from llm_nephio_oran.validators.manifest_validator import validate_package
from llm_nephio_oran.gitops.git_ops import push_to_gitops

logger = logging.getLogger(__name__)

# Component → namespace mapping for Observe phase
COMPONENT_NAMESPACES = {
    "oran": ["oai-odu", "oai-ocu", "oai-cu-cp", "oai-cu-up"],
    "free5gc": ["free5gc-amf", "free5gc-smf", "free5gc-upf"],
    "ricplt": ["ric-kpimon", "ric-ts"],
}


class ClosedLoopController:
    """Observe→Analyze→Act controller with guard rails."""

    def __init__(
        self,
        namespaces: list[str] | None = None,
        max_replicas: int = 5,
        cooldown_seconds: int = 300,
        output_dir: Path = Path("packages/instances"),
    ):
        self.namespaces = namespaces or ["oran", "free5gc"]
        self.output_dir = output_dir
        self.analyzer = MetricsAnalyzer(
            max_replicas=max_replicas,
            cooldown_seconds=cooldown_seconds,
        )

    def observe(self, namespace: str) -> dict[str, Any]:
        """Observe phase: collect metrics from Prometheus."""
        return snapshot_metrics(namespace=namespace)

    def analyze(self, metrics: dict[str, Any]) -> dict[str, Any]:
        """Analyze phase: evaluate metrics against thresholds."""
        return self.analyzer.analyze(metrics)

    def act(
        self,
        component: str,
        namespace: str,
        recommendation: dict[str, Any],
    ) -> dict[str, Any] | None:
        """Act phase: convert recommendation to IntentPlan → kpt packages → Git PR.

        NEVER applies directly to cluster (ADR-0004). All changes go through Git PRs.
        """
        if recommendation["action"] == "none":
            return None

        action_type = recommendation["action"]
        replicas = recommendation["recommended_replicas"]
        reason = recommendation["reason"]

        # Generate intent text for the planner
        intent_text = f"Scale {component} to {replicas} replicas in {namespace} — {reason}"
        logger.info("Closed-loop ACT: %s", intent_text)

        # Use stub planner for deterministic closed-loop actions
        plan = plan_from_text(intent_text, use_llm=False)
        plan["intentType"] = "closedloop.act"
        plan["actions"] = [{
            "kind": "scale",
            "component": component,
            "replicas": replicas,
            "naming": {
                "domain": self._domain_for_component(component),
                "site": "lab",
                "slice": "embb",
                "instance": "i001",
            },
        }]

        # Generate packages
        pkg_dir = generate_kpt_packages(plan, self.output_dir)

        # Validate
        errors = validate_package(pkg_dir)
        if errors:
            logger.error("Closed-loop manifest validation failed: %s", errors)
            return {"action": action_type, "status": "validation_failed", "errors": errors}

        # Push to Git as PR (Act = PR, NEVER kubectl)
        git_result = push_to_gitops(pkg_dir, plan)

        # Record action for cooldown tracking
        self.analyzer.record_action(component, action_type)

        return git_result

    def run_cycle(self) -> list[dict[str, Any] | None]:
        """Run one full Observe→Analyze→Act cycle across all namespaces."""
        results = []
        for namespace in self.namespaces:
            threshold_check = check_scaling_thresholds(namespace)
            if not threshold_check["thresholds_exceeded"]:
                continue

            # Find which components are in this namespace
            components = COMPONENT_NAMESPACES.get(namespace, [])
            for component in components:
                metrics = {
                    "namespace": namespace,
                    "component": component,
                    "cpu_utilization": threshold_check["cpu_utilization"],
                    "memory_utilization": threshold_check["memory_utilization"],
                    "current_replicas": threshold_check["metrics"]["k8s"].get("pod_count", 1),
                }
                recommendation = self.analyze(metrics)
                result = self.act(component, namespace, recommendation)
                if result is not None:
                    results.append(result)
        return results

    @staticmethod
    def _domain_for_component(component: str) -> str:
        """Map component to domain for naming."""
        if component.startswith("oai-"):
            return "ran"
        if component.startswith("free5gc-"):
            return "core"
        if component.startswith("ric-"):
            return "ric"
        return "sim"
