"""M7 Demo Runner — orchestrates the full closed-loop pipeline for demonstration.

Provides a step-by-step demo of the M7 milestone:
  1. Observe: E2 KPM Simulator → KPIMON Bridge → per-cell PRB/CQI metrics
  2. Analyze: MetricsAnalyzer → scaling/steering recommendations
  3. Act: TMF921 intent creation (dry-run or live)
  4. Pipeline: plan → validate → generate → git → Porch → ConfigSync
  5. Metrics: Prometheus counters/gauges/histograms for Grafana

Usage:
    demo = DemoRunner(load_factor=0.95, seed=42, dry_run=True)
    summary = demo.run_full_cycle()

    # Or step-by-step:
    demo.observe()
    demo.analyze()
    demo.act()
    metrics = demo.get_metrics_snapshot()
"""
from __future__ import annotations

import time
from pathlib import Path
from typing import Any

import requests

from llm_nephio_oran.closedloop.analyzer import MetricsAnalyzer
from llm_nephio_oran.closedloop.e2_closed_loop import _GNB_COMPONENT_MAP
from llm_nephio_oran.e2sim.kpm_simulator import CellConfig, E2KpmSimulator
from llm_nephio_oran.kpimon.bridge import KpimonBridge
from llm_nephio_oran.observability import pipeline_metrics as pm


_DEFAULT_CELLS = [
    CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb", "urllc"]),
]


class DemoRunner:
    """Orchestrates the M7 closed-loop demo: Observe → Analyze → Act."""

    def __init__(
        self,
        cells: list[CellConfig] | None = None,
        load_factor: float = 0.5,
        seed: int | None = None,
        dry_run: bool = True,
        prb_threshold: float = 80.0,
        max_replicas: int = 5,
        cooldown_seconds: int = 0,
        pipeline_mode: bool = False,
        store_dir: Path | None = None,
        api_base: str = "http://localhost:8080",
    ):
        self._cells = cells or list(_DEFAULT_CELLS)
        self._load_factor = load_factor
        self.dry_run = dry_run
        self._prb_threshold = prb_threshold
        self._pipeline_mode = pipeline_mode
        self._store_dir = store_dir
        self._api_base = api_base

        self._sim = E2KpmSimulator(
            cells=self._cells, load_factor=load_factor, seed=seed,
        )
        self._bridge = KpimonBridge(simulator=self._sim)
        self._analyzer = MetricsAnalyzer(
            max_replicas=max_replicas,
            cooldown_seconds=cooldown_seconds,
            scale_out_threshold=prb_threshold / 100.0,
        )

        # Phase results (set after each phase runs)
        self._observe_result: dict[str, Any] | None = None
        self._analyze_result: dict[str, Any] | None = None
        self._act_result: dict[str, Any] | None = None

    # ── Phase 1: Observe ──────────────────────────────────────────────────

    def observe(self) -> dict[str, Any]:
        """Collect E2 KPM metrics from simulator and aggregate per-cell.

        Also updates Prometheus gauges for PRB usage.
        """
        # Collect fresh snapshots via bridge (wraps E2KpmSimulator.generate)
        snapshots = self._bridge.collect()
        cell_summaries = self._bridge.summary_by_cell()

        # Update Prometheus gauges
        for cell_id, summary in cell_summaries.items():
            pm.e2kpm_prb_usage.labels(
                cell_id=cell_id,
                gnb_id=summary["gnb_id"],
            ).set(summary["avg_prb_usage"])

        self._snapshots = snapshots  # cached for analyze phase
        self._observe_result = {
            "cell_summaries": cell_summaries,
            "snapshot_count": len(snapshots),
        }
        return self._observe_result

    # ── Phase 2: Analyze ──────────────────────────────────────────────────

    def analyze(self) -> dict[str, Any]:
        """Analyze observed metrics and produce scaling recommendations.

        Requires observe() to have been called first.
        """
        if self._observe_result is None:
            raise RuntimeError("Must call observe() before analyze()")

        cell_summaries = self._observe_result["cell_summaries"]
        recommendations = []

        # Group cells by component (same logic as ClosedLoopE2EController.run_cycle)
        component_cells: dict[tuple[str, str], list[tuple[str, dict[str, Any]]]] = {}
        for cell_id, summary in cell_summaries.items():
            gnb_id = summary["gnb_id"]
            key = self._resolve_component(gnb_id)
            # Use per-slice max PRB for capacity decisions
            snapshots = [s for s in self._snapshots if s.cell_id == cell_id]
            max_prb = max((s.prb_usage_pct for s in snapshots), default=0.0)
            summary_with_max = dict(summary, avg_prb_usage=round(max_prb, 1))
            component_cells.setdefault(key, []).append((cell_id, summary_with_max))

        for (namespace, component), cells in component_cells.items():
            prb_values = [s["avg_prb_usage"] for _, s in cells]
            max_prb = max(prb_values) if prb_values else 0.0
            total_ues = sum(s["total_ues"] for _, s in cells)
            cell_ids = [cid for cid, _ in cells]

            # Use PRB as capacity proxy (same as e2_closed_loop.analyze_cell)
            prb_normalized = max_prb / 100.0
            metrics = {
                "namespace": namespace,
                "component": component,
                "cpu_utilization": prb_normalized,
                "memory_utilization": 0.0,
                "current_replicas": 1,
            }
            rec = self._analyzer.analyze(metrics)
            rec["cell_ids"] = cell_ids
            rec["namespace"] = namespace
            rec["component"] = component
            rec["prb_usage_pct"] = round(max_prb, 1)
            rec["total_ues"] = total_ues
            recommendations.append(rec)

        self._analyze_result = {"recommendations": recommendations}
        return self._analyze_result

    # ── Phase 3: Act ──────────────────────────────────────────────────────

    def act(self) -> dict[str, Any]:
        """Create intents based on recommendations.

        In dry-run mode, creates local intent records without API calls.
        In live mode, POST to TMF921 API.

        Requires analyze() to have been called first.
        """
        if self._analyze_result is None:
            raise RuntimeError("Must call analyze() before act()")

        recommendations = self._analyze_result["recommendations"]
        intents: list[dict[str, Any]] = []

        for rec in recommendations:
            if rec["action"] == "none":
                continue

            component = rec["component"]
            replicas = rec["recommended_replicas"]
            prb = rec.get("prb_usage_pct", 0)
            ues = rec.get("total_ues", 0)
            cell_ids = rec.get("cell_ids", [])

            expression = (
                f"Scale {component} to {replicas} replicas — "
                f"PRB usage {prb:.1f}%, {ues} active UEs"
            )

            # Record scale action in Prometheus
            pm.scale_actions.labels(
                component=component,
                action=rec["action"],
            ).inc()

            if self.dry_run:
                intents.append({
                    "action": rec["action"],
                    "expression": expression,
                    "component": component,
                    "replicas": replicas,
                    "prb_usage_pct": prb,
                    "cell_ids": cell_ids,
                    "dry_run": True,
                })
            else:
                # Live mode: POST to TMF921 API
                body = {
                    "expression": expression,
                    "source": "closedloop",
                    "use_llm": False,
                    "dry_run": False,
                }
                try:
                    resp = requests.post(
                        f"{self._api_base}/tmf-api/intent/v5/intent",
                        json=body,
                        timeout=10,
                    )
                    resp.raise_for_status()
                    intents.append(resp.json())
                except Exception as e:
                    intents.append({
                        "action": rec["action"],
                        "expression": expression,
                        "error": str(e),
                    })

            self._analyzer.record_action(component, rec["action"])

        self._act_result = {"intents": intents}
        return self._act_result

    # ── Full Cycle ────────────────────────────────────────────────────────

    def run_full_cycle(self) -> dict[str, Any]:
        """Run the complete Observe→Analyze→Act cycle.

        Returns a summary dict with results from each phase,
        metrics snapshot, and total duration.
        """
        start = time.monotonic()

        observe_result = self.observe()
        analyze_result = self.analyze()
        act_result = self.act()

        # Optional: feed through intentd v2 pipeline
        pipeline_results: list[dict[str, Any]] = []
        if self._pipeline_mode and act_result["intents"]:
            pipeline_results = self.run_pipeline(act_result["intents"])

        elapsed = time.monotonic() - start
        metrics = self.get_metrics_snapshot()

        summary: dict[str, Any] = {
            "observe": observe_result,
            "analyze": analyze_result,
            "act": act_result,
            "metrics": metrics,
            "duration_seconds": round(elapsed, 3),
        }
        if self._pipeline_mode:
            summary["pipeline_results"] = pipeline_results
        return summary

    # ── Metrics ───────────────────────────────────────────────────────────

    def get_metrics_snapshot(self) -> dict[str, Any]:
        """Capture current Prometheus metrics as a dict (for display/testing)."""
        return {
            "active_intents": pm.get_metric_value(pm.active_intents),
            "intents_created": pm.sum_metric(pm.intent_created),
            "intents_completed": pm.sum_metric(pm.intent_completed),
            "scale_actions": pm.sum_metric(pm.scale_actions),
            "prometheus_text": pm.generate_metrics_text(),
        }

    # ── Pipeline Mode ─────────────────────────────────────────────────────

    def run_pipeline(
        self,
        intents: list[dict[str, Any]],
        poll_timeout: float = 30.0,
    ) -> list[dict[str, Any]]:
        """Feed intents through the intentd v2 pipeline (dry-run).

        Creates a temporary IntentStore + FastAPI app and runs each
        intent expression through the full plan→validate→generate pipeline.
        """
        from starlette.testclient import TestClient

        from llm_nephio_oran.intentd.app_v2 import create_app

        app = create_app(store_dir=self._store_dir)

        results = []
        with TestClient(app) as client:
            for intent in intents:
                expression = intent.get("expression", "")
                if not expression:
                    continue

                resp = client.post(
                    "/tmf-api/intent/v5/intent",
                    json={
                        "expression": expression,
                        "source": "closedloop",
                        "use_llm": False,
                        "dry_run": True,
                    },
                )
                if resp.status_code != 201:
                    results.append({"expression": expression, "error": resp.text})
                    continue

                rec = resp.json()
                intent_id = rec["id"]

                # Poll until terminal state
                terminal = False
                max_polls = int(poll_timeout / 0.1)
                for _ in range(max_polls):
                    time.sleep(0.1)
                    poll = client.get(f"/tmf-api/intent/v5/intent/{intent_id}")
                    if poll.status_code == 200:
                        state = poll.json().get("state", "")
                        if state in ("completed", "failed"):
                            results.append(poll.json())
                            terminal = True
                            break
                if not terminal:
                    results.append({
                        "id": intent_id,
                        "state": "timeout",
                        "expression": expression,
                    })

        return results

    # ── Helpers ────────────────────────────────────────────────────────────

    @staticmethod
    def _resolve_component(gnb_id: str) -> tuple[str, str]:
        """Resolve gNB ID to (namespace, component) for scaling."""
        for prefix, (ns, comp) in _GNB_COMPONENT_MAP.items():
            if gnb_id.startswith(prefix):
                return ns, comp
        return "oran", "oai-odu"
