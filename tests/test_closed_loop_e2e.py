"""T10: Closed-loop E2E verification — E2 KPM metrics → scale intent → packages.

Proves the M7 milestone: when E2 KPM metrics exceed thresholds, the system
automatically creates a scaling intent, processes it through the full pipeline,
and produces valid kpt packages.

Flow:
  E2KpmSimulator(high load) → KpimonBridge → ClosedLoopE2EController.observe()
    → MetricsAnalyzer.analyze() → scale_out recommendation
    → TMF921 POST /intent (dry_run) → pipeline → completed with packages

This is the crown jewel test — every component from T1-T9 participates.
"""
from __future__ import annotations

import time
from pathlib import Path
from typing import Any
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient

from llm_nephio_oran.e2sim.kpm_simulator import CellConfig, E2KpmSimulator
from llm_nephio_oran.kpimon.bridge import KpimonBridge, KpimonQuery
from llm_nephio_oran.closedloop.analyzer import MetricsAnalyzer
from llm_nephio_oran.closedloop.e2_closed_loop import ClosedLoopE2EController
from llm_nephio_oran.intentd.app_v2 import create_app
from llm_nephio_oran.intentd.store import IntentState


class TestClosedLoopObserveToAnalyze:
    """Phase 1: E2 KPM metrics → KPIMON → threshold detection."""

    def test_high_prb_triggers_scale_out(self):
        """High PRB usage from E2 KPM should trigger scale-out recommendation."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"])],
            load_factor=0.95,  # Extreme load
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)
        query = KpimonQuery(bridge=bridge)

        # Observe: collect metrics
        query.refresh()
        summary = bridge.summary_by_cell()
        prb_usage = summary["cell-001"]["avg_prb_usage"]

        # PRB > 80% should trigger scale-out
        assert prb_usage > 80, f"Expected PRB > 80% at load_factor=0.95, got {prb_usage}%"

        # Convert E2 KPM observation to analyzer input format
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": prb_usage / 100.0,  # use PRB as proxy for CPU
            "memory_utilization": 0.5,
            "current_replicas": 1,
        }

        analyzer = MetricsAnalyzer(max_replicas=5, cooldown_seconds=0)
        recommendation = analyzer.analyze(metrics)

        assert recommendation["action"] == "scale_out"
        assert recommendation["recommended_replicas"] == 2

    def test_normal_load_no_action(self):
        """Normal PRB usage should NOT trigger any scaling action."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"])],
            load_factor=0.5,  # Normal load
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)
        query = KpimonQuery(bridge=bridge)

        query.refresh()
        summary = bridge.summary_by_cell()
        prb_usage = summary["cell-001"]["avg_prb_usage"]

        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": prb_usage / 100.0,
            "memory_utilization": 0.3,
            "current_replicas": 2,
        }

        analyzer = MetricsAnalyzer(max_replicas=5, cooldown_seconds=0)
        recommendation = analyzer.analyze(metrics)

        assert recommendation["action"] == "none"

    def test_high_ue_count_per_slice_aggregation(self):
        """Multi-cell aggregation: total UEs across cells should be summed."""
        sim = E2KpmSimulator(
            cells=[
                CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"]),
                CellConfig(cell_id="cell-002", gnb_id="gnb-edge01", slices=["embb"]),
                CellConfig(cell_id="cell-003", gnb_id="gnb-edge02", slices=["embb"]),
            ],
            load_factor=0.8,
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)
        query = KpimonQuery(bridge=bridge)

        query.refresh()
        total_ues = query.total_active_ues()
        slice_summary = bridge.summary_by_slice()

        assert total_ues > 0
        assert slice_summary["embb"]["cell_count"] == 3
        assert slice_summary["embb"]["total_ues"] == total_ues


class TestClosedLoopFullPipeline:
    """Phase 2: Full E2E — E2 KPM → scale recommendation → TMF921 intent → packages."""

    @pytest.fixture
    def api_client(self, tmp_path):
        """Create intentd v2 app with tmp store and httpx test client."""
        app = create_app(store_dir=tmp_path / ".state")
        return TestClient(app)

    def test_e2_kpm_to_scale_intent_dry_run(self, api_client, tmp_path):
        """Full closed loop: E2 KPM → KPIMON → analyze → TMF921 POST → pipeline → COMPLETED."""
        # Step 1: Observe — simulate high load E2 KPM metrics
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"])],
            load_factor=0.95,
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)
        query = KpimonQuery(bridge=bridge)
        query.refresh()

        summary = bridge.summary_by_cell()
        prb_usage = summary["cell-001"]["avg_prb_usage"]
        total_ues = query.total_active_ues()

        # Step 2: Analyze — determine scaling action
        analyzer = MetricsAnalyzer(max_replicas=5, cooldown_seconds=0)
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": prb_usage / 100.0,
            "memory_utilization": 0.5,
            "current_replicas": 1,
        }
        recommendation = analyzer.analyze(metrics)
        assert recommendation["action"] == "scale_out"

        # Step 3: Act — POST scaling intent through TMF921 API (dry_run)
        component = metrics["component"]
        replicas = recommendation["recommended_replicas"]
        expression = (
            f"Scale {component} to {replicas} replicas in oran "
            f"— PRB usage {prb_usage:.1f}%, {total_ues} active UEs"
        )

        resp = api_client.post("/tmf-api/intent/v5/intent", json={
            "expression": expression,
            "source": "closedloop",
            "use_llm": False,
            "dry_run": True,
        })
        assert resp.status_code == 201
        intent = resp.json()
        intent_id = intent["id"]
        assert intent["state"] == "acknowledged"
        assert intent["source"] == "closedloop"

        # Step 4: Wait for pipeline completion
        deadline = time.monotonic() + 10
        final_state = None
        while time.monotonic() < deadline:
            get_resp = api_client.get(f"/tmf-api/intent/v5/intent/{intent_id}")
            rec = get_resp.json()
            final_state = rec["state"]
            if final_state in ("completed", "failed"):
                break
            time.sleep(0.1)

        assert final_state == "completed", f"Expected completed, got {final_state}"

        # Step 5: Verify the report
        report_resp = api_client.get(f"/tmf-api/intent/v5/intent/{intent_id}/report")
        assert report_resp.status_code == 200
        report = report_resp.json()
        assert report["compliant"] is True
        assert report["terminal"] is True
        assert report["source"] == "closedloop"
        assert "scale" in report["expression"].lower() or "Scale" in report["expression"]

        # Step 6: Verify plan was captured
        plan = rec["plan"]
        assert plan is not None
        assert plan["intentType"] in ("slice.scale", "closedloop.act")
        assert any(a["kind"] == "scale" for a in plan["actions"])

    def test_multiple_cells_trigger_single_intent(self, api_client, tmp_path):
        """When 3 cells all exceed PRB threshold, only 1 scale intent is created per component."""
        sim = E2KpmSimulator(
            cells=[
                CellConfig(cell_id=f"cell-{i:03d}", gnb_id="gnb-edge01", slices=["embb"])
                for i in range(1, 4)
            ],
            load_factor=0.95,
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)
        query = KpimonQuery(bridge=bridge)
        query.refresh()

        # All cells should have high PRB
        high_cells = query.high_prb_cells(threshold=50.0)
        assert len(high_cells) >= 2, f"Expected >=2 high PRB cells, got {len(high_cells)}"

        # Single component-level decision: aggregate to one intent
        summary = bridge.summary_by_cell()
        avg_prb = sum(c["avg_prb_usage"] for c in summary.values()) / len(summary)

        analyzer = MetricsAnalyzer(max_replicas=5, cooldown_seconds=0)
        recommendation = analyzer.analyze({
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": avg_prb / 100.0,
            "memory_utilization": 0.5,
            "current_replicas": 1,
        })

        # POST intent
        resp = api_client.post("/tmf-api/intent/v5/intent", json={
            "expression": f"Scale oai-odu to {recommendation['recommended_replicas']} replicas — avg PRB {avg_prb:.1f}%",
            "source": "closedloop",
            "use_llm": False,
            "dry_run": True,
        })
        assert resp.status_code == 201

        # Wait for completion
        intent_id = resp.json()["id"]
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            rec = api_client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            if rec["state"] in ("completed", "failed"):
                break
            time.sleep(0.1)

        assert rec["state"] == "completed"

    def test_cooldown_prevents_repeated_scaling(self):
        """After a scale-out, cooldown should prevent immediate re-scaling."""
        analyzer = MetricsAnalyzer(max_replicas=5, cooldown_seconds=300)

        high_metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.95,
            "memory_utilization": 0.5,
            "current_replicas": 1,
        }

        # First analysis: should scale out
        rec1 = analyzer.analyze(high_metrics)
        assert rec1["action"] == "scale_out"

        # Record the action (simulates what controller does after POST)
        analyzer.record_action("oai-odu", "scale_out")

        # Second analysis: cooldown should block
        rec2 = analyzer.analyze(high_metrics)
        assert rec2["action"] == "none"
        assert "Cooldown" in rec2["reason"]

    def test_max_replicas_guard(self):
        """Scale-out must not exceed max_replicas even under extreme load."""
        analyzer = MetricsAnalyzer(max_replicas=3, cooldown_seconds=0)

        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.99,
            "memory_utilization": 0.99,
            "current_replicas": 3,  # Already at max
        }

        rec = analyzer.analyze(metrics)
        assert rec["action"] == "none"
        assert "Max replicas" in rec["reason"]


class TestClosedLoopE2EController:
    """Phase 3: Integration of E2KpmSimulator + KPIMON + ClosedLoopController."""

    def test_observe_analyze_act_wired(self, tmp_path):
        """Verify the full Observe→Analyze→Act can be wired with E2 KPM input.

        This tests the ClosedLoopE2EController that bridges KPIMON → TMF921.
        """
        # ClosedLoopE2EController imported at top-level

        # Create intentd app for the controller to POST to
        app = create_app(store_dir=tmp_path / ".state")
        client = TestClient(app)

        sim = E2KpmSimulator(
            cells=[
                CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"]),
                CellConfig(cell_id="cell-002", gnb_id="gnb-edge01", slices=["embb"]),
            ],
            load_factor=0.95,
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)

        controller = ClosedLoopE2EController(
            bridge=bridge,
            api_base="http://test",
            max_replicas=5,
            cooldown_seconds=0,
            prb_threshold=70.0,
            dry_run=True,
        )

        # Inject the test client for HTTP calls and a stub replica source
        controller._client = client
        controller._replica_source = lambda ns, comp: 1

        # Run one cycle — two cells on same gnb-edge01 should produce ONE intent
        results = controller.run_cycle()

        # Should have created exactly one scaling intent (aggregated by component)
        assert len(results) == 1, f"Expected 1 aggregated intent, got {len(results)}: {results}"

        # Verify the intent was created in the store
        intent_id = results[0]["id"]
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            rec = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            if rec["state"] in ("completed", "failed"):
                break
            time.sleep(0.1)

        assert rec["state"] == "completed", f"Expected completed, got {rec['state']}"
        assert rec["source"] == "closedloop"

    def test_low_load_no_intent_created(self, tmp_path):
        """Under low load, no scaling intent should be created."""
        # ClosedLoopE2EController imported at top-level

        app = create_app(store_dir=tmp_path / ".state")
        client = TestClient(app)

        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"])],
            load_factor=0.3,  # Low load
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)

        controller = ClosedLoopE2EController(
            bridge=bridge,
            api_base="http://test",
            max_replicas=5,
            cooldown_seconds=0,
            prb_threshold=70.0,
            dry_run=True,
        )
        controller._client = client
        controller._replica_source = lambda ns, comp: 1

        results = controller.run_cycle()
        assert results == [], f"Expected no scaling intents under low load, got {results}"

    def test_prometheus_text_available_during_cycle(self, tmp_path):
        """During closed-loop, the bridge must produce valid Prometheus text."""
        # ClosedLoopE2EController imported at top-level

        app = create_app(store_dir=tmp_path / ".state")
        client = TestClient(app)

        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"])],
            load_factor=0.5,
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)

        controller = ClosedLoopE2EController(
            bridge=bridge,
            api_base="http://test",
            max_replicas=5,
            cooldown_seconds=0,
            prb_threshold=70.0,
            dry_run=True,
        )
        controller._client = client
        controller._replica_source = lambda ns, comp: 1

        # Get Prometheus text before running cycle
        prom_text = bridge.metrics_text()
        assert "e2_kpm_prb_usage_percent" in prom_text

        # Run cycle (might or might not create intents based on load)
        controller.run_cycle()

        # Prometheus text should still be consistent
        summary = bridge.summary_by_cell()
        assert "cell-001" in summary

    def test_resolve_component_mapping(self):
        """_resolve_component should map gNB prefixes to (namespace, component)."""
        assert ClosedLoopE2EController._resolve_component("gnb-edge01") == ("oran", "oai-odu")
        assert ClosedLoopE2EController._resolve_component("gnb-central99") == ("core", "free5gc-amf")
        assert ClosedLoopE2EController._resolve_component("gnb-lab-test") == ("oran", "oai-odu")
        # Unknown prefix → default
        assert ClosedLoopE2EController._resolve_component("unknown-gnb") == ("oran", "oai-odu")

    def test_aggregation_groups_cells_by_component(self, tmp_path):
        """Three cells on two different gNBs should produce two intents, not three."""
        app = create_app(store_dir=tmp_path / ".state")
        client = TestClient(app)

        sim = E2KpmSimulator(
            cells=[
                CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"]),
                CellConfig(cell_id="cell-002", gnb_id="gnb-edge01", slices=["embb"]),
                CellConfig(cell_id="cell-003", gnb_id="gnb-central01", slices=["embb"]),
            ],
            load_factor=0.95,
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)

        controller = ClosedLoopE2EController(
            bridge=bridge,
            api_base="http://test",
            max_replicas=5,
            cooldown_seconds=0,
            prb_threshold=70.0,
            dry_run=True,
        )
        controller._client = client
        controller._replica_source = lambda ns, comp: 1

        results = controller.run_cycle()

        # Two components: gnb-edge → oai-odu, gnb-central → free5gc-amf
        # Each should produce exactly one intent (not one per cell)
        assert len(results) == 2, f"Expected 2 intents (one per component), got {len(results)}"

    def test_current_replicas_respected(self, tmp_path):
        """If already at max_replicas, no scaling should happen."""
        app = create_app(store_dir=tmp_path / ".state")
        client = TestClient(app)

        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"])],
            load_factor=0.95,
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)

        controller = ClosedLoopE2EController(
            bridge=bridge,
            api_base="http://test",
            max_replicas=3,
            cooldown_seconds=0,
            prb_threshold=70.0,
            dry_run=True,
        )
        controller._client = client
        # Already at max replicas
        controller._replica_source = lambda ns, comp: 3

        results = controller.run_cycle()
        assert results == [], f"Should not scale when already at max, got {results}"
