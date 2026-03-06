"""Closed-loop controller v2 — routes all actions through TMF921 API.

Unlike v1 which bypassed the API and called planner/generator/git directly,
v2 POSTs to /tmf-api/intent/v5/intent so that:
  1. All intents (human + automated) share a unified audit trail
  2. The async pipeline handles execution
  3. The state machine tracks lifecycle
"""
from __future__ import annotations

import logging
import os
from typing import Any

import requests

from llm_nephio_oran.closedloop.analyzer import MetricsAnalyzer
from llm_nephio_oran.observability.metrics import check_scaling_thresholds

logger = logging.getLogger(__name__)

COMPONENT_NAMESPACES = {
    "oran": ["oai-odu", "oai-ocu", "oai-cu-cp", "oai-cu-up"],
    "free5gc": ["free5gc-amf", "free5gc-smf", "free5gc-upf"],
    "ricplt": ["ric-kpimon", "ric-ts"],
}

INTENTD_URL = os.getenv("INTENTD_URL", "http://localhost:8080")


class ClosedLoopControllerV2:
    """Observe→Analyze→Act, with Act routed through TMF921 API."""

    def __init__(
        self,
        namespaces: list[str] | None = None,
        max_replicas: int = 5,
        cooldown_seconds: int = 300,
        api_base: str | None = None,
    ):
        self.namespaces = namespaces or ["oran", "free5gc"]
        self.api_base = api_base if api_base is not None else INTENTD_URL
        self.analyzer = MetricsAnalyzer(
            max_replicas=max_replicas,
            cooldown_seconds=cooldown_seconds,
        )

    def analyze(self, metrics: dict[str, Any]) -> dict[str, Any]:
        return self.analyzer.analyze(metrics)

    def act(
        self,
        component: str,
        namespace: str,
        recommendation: dict[str, Any],
    ) -> dict[str, Any] | None:
        """Act: POST scaling intent to TMF921 API (never direct kubectl/git)."""
        if recommendation["action"] == "none":
            return None

        replicas = recommendation["recommended_replicas"]
        reason = recommendation["reason"]

        expression = f"Scale {component} to {replicas} replicas in {namespace} — {reason}"
        logger.info("Closed-loop ACT via TMF921: %s", expression)

        body = {
            "expression": expression,
            "source": "closedloop",
            "use_llm": False,  # deterministic for automated actions
            "dry_run": False,
        }

        result = self._post_intent(body)
        self.analyzer.record_action(component, recommendation["action"])
        return result

    def _post_intent(self, body: dict[str, Any]) -> dict[str, Any]:
        """POST to TMF921 API. Separated for testability."""
        resp = requests.post(
            f"{self.api_base}/tmf-api/intent/v5/intent",
            json=body,
            timeout=10,
        )
        resp.raise_for_status()
        return resp.json()

    def run_cycle(self) -> list[dict[str, Any] | None]:
        results = []
        for namespace in self.namespaces:
            threshold_check = check_scaling_thresholds(namespace)
            if not threshold_check["thresholds_exceeded"]:
                continue

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
