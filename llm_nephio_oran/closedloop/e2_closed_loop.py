"""E2 KPM closed-loop controller — bridges KPIMON metrics to TMF921 scaling intents.

Wires the full M7 closed loop:
  E2KpmSimulator → KpimonBridge → observe (PRB aggregation)
    → MetricsAnalyzer → scale recommendation
    → TMF921 POST /intent → async pipeline → Git PR → ConfigSync

This replaces Prometheus-based observation (controller_v2.py) with direct
E2 KPM metrics from KPIMON, which is more accurate for RAN scaling decisions
because PRB usage is the primary capacity indicator.
"""
from __future__ import annotations

import logging
from typing import Any

import requests

from llm_nephio_oran.closedloop.analyzer import MetricsAnalyzer
from llm_nephio_oran.kpimon.bridge import KpimonBridge, KpimonQuery
from llm_nephio_oran.a1.client import A1Client

logger = logging.getLogger(__name__)

# Map cell gnb_id prefix → namespace/component for scaling
_GNB_COMPONENT_MAP: dict[str, tuple[str, str]] = {
    "gnb-edge": ("oran", "oai-odu"),
    "gnb-central": ("core", "free5gc-amf"),
    "gnb-lab": ("oran", "oai-odu"),
}


class ClosedLoopE2EController:
    """Observe→Analyze→Act using E2 KPM metrics from KPIMON.

    Observe: KpimonBridge.collect() → per-cell PRB/UE aggregation
    Analyze: MetricsAnalyzer with PRB usage as CPU proxy
    Act: POST scaling intent to TMF921 API (dry_run configurable)
    """

    def __init__(
        self,
        bridge: KpimonBridge,
        api_base: str = "http://localhost:8080",
        max_replicas: int = 5,
        cooldown_seconds: int = 300,
        prb_threshold: float = 80.0,
        dry_run: bool = False,
        a1_client: A1Client | None = None,
    ):
        self._bridge = bridge
        self._query = KpimonQuery(bridge=bridge)
        self._analyzer = MetricsAnalyzer(
            max_replicas=max_replicas,
            cooldown_seconds=cooldown_seconds,
            scale_out_threshold=prb_threshold / 100.0,
        )
        self._api_base = api_base
        self._prb_threshold = prb_threshold
        self._dry_run = dry_run
        self._client: Any = None  # injectable TestClient for testing
        self._replica_source: Any = None  # injectable (ns, comp) -> int for testing
        self._a1 = a1_client  # injectable A1Client for TS policies

    def observe(self) -> dict[str, dict[str, Any]]:
        """Observe: collect E2 KPM metrics and aggregate per-cell.

        Returns:
            Dict of cell_id → {gnb_id, avg_prb_usage, total_ues, total_dl_throughput, slices}.
        """
        self._query.refresh()
        return self._bridge.summary_by_cell()

    def analyze_cell(
        self,
        cell_id: str,
        cell_summary: dict[str, Any],
        current_replicas: int = 1,
    ) -> dict[str, Any]:
        """Analyze a single cell's metrics and produce a scaling recommendation.

        Uses PRB usage as the primary capacity indicator (replaces CPU-based
        scaling from controller_v2.py).
        """
        gnb_id = cell_summary["gnb_id"]
        prb_usage = cell_summary["avg_prb_usage"] / 100.0  # normalize to 0-1

        # Resolve which component to scale based on gNB ID
        namespace, component = self._resolve_component(gnb_id)

        metrics = {
            "namespace": namespace,
            "component": component,
            "cpu_utilization": prb_usage,  # PRB as capacity proxy
            "memory_utilization": 0.0,     # not applicable for E2 KPM
            "current_replicas": current_replicas,
        }

        recommendation = self._analyzer.analyze(metrics)
        recommendation["cell_id"] = cell_id
        recommendation["gnb_id"] = gnb_id
        recommendation["namespace"] = namespace
        recommendation["component"] = component
        recommendation["prb_usage_pct"] = cell_summary["avg_prb_usage"]
        recommendation["total_ues"] = cell_summary["total_ues"]
        return recommendation

    def act(self, recommendation: dict[str, Any]) -> dict[str, Any] | None:
        """Act: POST scaling intent to TMF921 API."""
        if recommendation["action"] == "none":
            return None

        component = recommendation.get("component",
                                       recommendation.get("gnb_id", "unknown"))
        replicas = recommendation["recommended_replicas"]
        prb = recommendation.get("prb_usage_pct", 0)
        ues = recommendation.get("total_ues", 0)
        cell = recommendation.get("cell_id", "unknown")

        expression = (
            f"Scale {component} to {replicas} replicas — "
            f"cell {cell} PRB usage {prb:.1f}%, {ues} active UEs"
        )
        logger.info("E2 closed-loop ACT: %s", expression)

        if self._dry_run and self._client is None:
            # No API server available — return local dry-run result
            self._analyzer.record_action(component, recommendation["action"])
            return {"action": recommendation["action"], "expression": expression,
                    "component": component, "replicas": replicas, "dry_run": True}

        body = {
            "expression": expression,
            "source": "closedloop",
            "use_llm": False,
            "dry_run": self._dry_run,
        }

        result = self._post_intent(body)
        self._analyzer.record_action(component, recommendation["action"])
        return result

    def run_cycle(self) -> list[dict[str, Any]]:
        """Run one full Observe→Analyze→Act cycle.

        Two analysis paths:
        1. PRB-based scaling: high PRB → scale_out intent (existing)
        2. CQI-based steering: low CQI → A1 TS policy (new)

        Returns list of created intent/policy records.
        """
        cell_summaries = self.observe()
        results = []

        # ── Path 1: PRB-based scaling ────────────────────────────────
        # Use per-slice max PRB (not cross-slice average) for capacity decisions.
        # A cell with eMBB at 87% and URLLC at 28% is capacity-constrained
        # on eMBB, even though the average is only 57%.
        snapshots = self._bridge._ensure_cache()
        component_cells: dict[tuple[str, str], list[tuple[str, dict[str, Any]]]] = {}
        for cell_id, summary in cell_summaries.items():
            gnb_id = summary["gnb_id"]
            key = self._resolve_component(gnb_id)
            # Compute max PRB across slices for this cell (capacity indicator)
            cell_snaps = [s for s in snapshots if s.cell_id == cell_id]
            max_prb = max((s.prb_usage_pct for s in cell_snaps), default=0.0)
            summary_with_max = dict(summary, avg_prb_usage=round(max_prb, 1))
            component_cells.setdefault(key, []).append((cell_id, summary_with_max))

        for (namespace, component), cells in component_cells.items():
            prb_values = [s["avg_prb_usage"] for _, s in cells]
            max_prb = max(prb_values) if prb_values else 0.0
            total_ues = sum(s["total_ues"] for _, s in cells)
            cell_ids = [cid for cid, _ in cells]

            current_replicas = self._get_current_replicas(namespace, component)

            aggregated_summary: dict[str, Any] = {
                "gnb_id": cells[0][1]["gnb_id"],
                "avg_prb_usage": round(max_prb, 1),
                "total_ues": total_ues,
            }
            recommendation = self.analyze_cell(
                cell_ids[0], aggregated_summary, current_replicas=current_replicas,
            )
            recommendation["cell_ids"] = cell_ids
            recommendation["cell_count"] = len(cells)

            if recommendation["action"] != "none":
                logger.info(
                    "Component %s/%s: %s across %d cells (PRB=%.1f%%, UEs=%d)",
                    namespace, component, recommendation["action"],
                    len(cells), max_prb, total_ues,
                )

            result = self.act(recommendation)
            if result is not None:
                results.append(result)

        # ── Path 2: CQI-based traffic steering ──────────────────────
        low_cqi_cells = self._query.low_cqi_cells()
        for cell_info in low_cqi_cells:
            recommendation = self._analyzer.analyze_cqi(cell_info)
            if recommendation["action"] == "traffic_steer":
                result = self._act_traffic_steer(recommendation)
                if result is not None:
                    results.append(result)

        return results

    def _act_traffic_steer(self, recommendation: dict[str, Any]) -> dict[str, Any] | None:
        """Act on a traffic steering recommendation by creating an A1 policy.

        If A1 client is available, creates the policy directly.
        Otherwise, creates a configure intent via TMF921 API.
        """
        cell_id = recommendation["cell_id"]
        threshold = recommendation["threshold"]
        reason = recommendation["reason"]
        policy_id = f"ts-{cell_id}"

        logger.info("Traffic steer ACT: %s (threshold=%d%%)", reason, threshold)

        if self._dry_run:
            return {"action": "traffic_steer", "cell_id": cell_id,
                    "threshold": threshold, "dry_run": True}

        # Try direct A1 policy creation
        if self._a1 is not None:
            try:
                self._a1.create_ts_policy(policy_id, threshold=threshold)
                self._analyzer.record_action(f"ts-{cell_id}", "traffic_steer")
                return {"action": "traffic_steer", "cell_id": cell_id,
                        "policy_id": policy_id, "threshold": threshold,
                        "method": "a1_direct"}
            except Exception:
                logger.warning("A1 direct policy creation failed, falling back to intent",
                             exc_info=True)

        # Fallback: create intent via TMF921 API
        expression = (
            f"Configure ric-ts with threshold {threshold}% for cell {cell_id} — {reason}"
        )
        body = {
            "expression": expression,
            "source": "closedloop",
            "use_llm": False,
            "dry_run": False,
        }
        result = self._post_intent(body)
        self._analyzer.record_action(f"ts-{cell_id}", "traffic_steer")
        result["method"] = "tmf921_intent"
        return result

    def _get_current_replicas(self, namespace: str, component: str) -> int:
        """Query current replica count. Returns 1 as fallback."""
        if self._replica_source is not None:
            return self._replica_source(namespace, component)
        try:
            from kubernetes import client as k8s_client, config as k8s_config
            try:
                k8s_config.load_incluster_config()
            except k8s_config.ConfigException:
                k8s_config.load_kube_config()
            apps_v1 = k8s_client.AppsV1Api()
            dep = apps_v1.read_namespaced_deployment(name=component, namespace=namespace)
            return dep.spec.replicas or 1
        except Exception:
            logger.debug("Cannot query replicas for %s/%s, using 1", namespace, component)
            return 1

    def _post_intent(self, body: dict[str, Any]) -> dict[str, Any]:
        """POST to TMF921 API. Uses injectable client for testing."""
        if self._client is not None:
            # TestClient / injectable client — use path only
            resp = self._client.post(
                "/tmf-api/intent/v5/intent",
                json=body,
            )
            resp.raise_for_status()
            return resp.json()

        resp = requests.post(
            f"{self._api_base}/tmf-api/intent/v5/intent",
            json=body,
            timeout=10,
        )
        resp.raise_for_status()
        return resp.json()

    @staticmethod
    def _resolve_component(gnb_id: str) -> tuple[str, str]:
        """Resolve gNB ID to (namespace, component) for scaling."""
        for prefix, (ns, comp) in _GNB_COMPONENT_MAP.items():
            if gnb_id.startswith(prefix):
                return ns, comp
        return "oran", "oai-odu"  # default for unknown gNBs
