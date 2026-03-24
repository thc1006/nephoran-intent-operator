"""Tests for Traffic Steering closed-loop integration."""
from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest

from llm_nephio_oran.closedloop.analyzer import MetricsAnalyzer, CQI_STEER_THRESHOLD
from llm_nephio_oran.e2sim.kpm_simulator import E2KpmSimulator, CellConfig
from llm_nephio_oran.kpimon.bridge import KpimonBridge, KpimonQuery
from llm_nephio_oran.closedloop.e2_closed_loop import ClosedLoopE2EController


# ── CQI Analysis ──────────────────────────────────────────────────────


class TestAnalyzerCQI:
    def setup_method(self):
        self.analyzer = MetricsAnalyzer(cqi_steer_threshold=8.0)

    def test_low_cqi_triggers_steer(self):
        metrics = {"cell_id": "cell-001", "gnb_id": "gnb-edge01", "avg_cqi": 5.0}
        result = self.analyzer.analyze_cqi(metrics)
        assert result["action"] == "traffic_steer"
        assert result["threshold"] > 0
        assert "cell-001" in result["reason"]

    def test_normal_cqi_no_action(self):
        metrics = {"cell_id": "cell-001", "gnb_id": "gnb-edge01", "avg_cqi": 12.0}
        result = self.analyzer.analyze_cqi(metrics)
        assert result["action"] == "none"

    def test_cqi_at_threshold_no_action(self):
        metrics = {"cell_id": "cell-001", "gnb_id": "gnb-edge01", "avg_cqi": 8.0}
        result = self.analyzer.analyze_cqi(metrics)
        assert result["action"] == "none"

    def test_cqi_just_below_threshold_steers(self):
        metrics = {"cell_id": "cell-001", "gnb_id": "gnb-edge01", "avg_cqi": 7.9}
        result = self.analyzer.analyze_cqi(metrics)
        assert result["action"] == "traffic_steer"

    def test_threshold_proportional_to_cqi_gap(self):
        low = self.analyzer.analyze_cqi(
            {"cell_id": "c1", "gnb_id": "g1", "avg_cqi": 3.0})
        medium = self.analyzer.analyze_cqi(
            {"cell_id": "c2", "gnb_id": "g2", "avg_cqi": 6.0})
        assert low["threshold"] > medium["threshold"]

    def test_cooldown_prevents_steer(self):
        self.analyzer.record_action("ts-cell-001", "traffic_steer")
        metrics = {"cell_id": "cell-001", "gnb_id": "gnb-edge01", "avg_cqi": 3.0}
        result = self.analyzer.analyze_cqi(metrics)
        assert result["action"] == "none"
        assert "Cooldown" in result["reason"]


# ── KPIMON Bridge low_cqi_cells ───────────────────────────────────────


class TestBridgeLowCQI:
    def setup_method(self):
        cells = [
            CellConfig(cell_id="cell-good", gnb_id="gnb-edge01", slices=["embb"]),
            CellConfig(cell_id="cell-bad", gnb_id="gnb-edge01", slices=["embb"]),
        ]
        self.sim = E2KpmSimulator(cells=cells, seed=42, load_factor=0.5)
        self.bridge = KpimonBridge(simulator=self.sim)
        self.query = KpimonQuery(bridge=self.bridge)

    def test_low_cqi_cells_returns_list(self):
        result = self.query.low_cqi_cells(threshold=20.0)  # very high threshold
        assert isinstance(result, list)
        for cell in result:
            assert "cell_id" in cell
            assert "avg_cqi" in cell
            assert cell["avg_cqi"] < 20.0

    def test_low_cqi_cells_empty_with_low_threshold(self):
        result = self.query.low_cqi_cells(threshold=1.0)
        assert result == []  # CQI is always > 1

    def test_low_cqi_cells_has_gnb_id(self):
        result = self.query.low_cqi_cells(threshold=20.0)
        if result:
            assert result[0]["gnb_id"] == "gnb-edge01"


# ── KPT Generator A1 Policy ConfigMap ─────────────────────────────────


class TestKPTGeneratorTSPolicy:
    def test_configure_ric_ts_generates_a1_policy(self, tmp_path):
        from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages
        import yaml

        plan = {
            "intentId": "intent-20260306-0001",
            "intentType": "closedloop.act",
            "actions": [{
                "kind": "configure",
                "component": "ric-ts",
                "params": {"threshold": 15},
                "naming": {"domain": "ric", "site": "lab", "slice": "embb", "instance": "i001"},
            }],
            "policy": {
                "requireHumanReview": True,
                "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True},
            },
        }
        pkg_dir = generate_kpt_packages(plan, tmp_path)

        # Find the a1-policy.yaml
        a1_files = list(pkg_dir.rglob("a1-policy.yaml"))
        assert len(a1_files) == 1

        with open(a1_files[0]) as f:
            cm = yaml.safe_load(f)

        assert cm["kind"] == "ConfigMap"
        assert cm["metadata"]["namespace"] == "ricplt"
        assert cm["metadata"]["annotations"]["nephio.org/a1-policy-type"] == "20008"
        assert cm["data"]["policy-type-id"] == "20008"

        import json
        policy_data = json.loads(cm["data"]["policy-data"])
        assert policy_data["threshold"] == 15

    def test_configure_ric_ts_no_deployment(self, tmp_path):
        from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages

        plan = {
            "intentId": "intent-20260306-0002",
            "intentType": "closedloop.act",
            "actions": [{
                "kind": "configure",
                "component": "ric-ts",
                "params": {"threshold": 5},
                "naming": {"domain": "ric", "site": "lab", "slice": "embb", "instance": "i001"},
            }],
            "policy": {
                "requireHumanReview": True,
                "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True},
            },
        }
        pkg_dir = generate_kpt_packages(plan, tmp_path)

        # Should NOT have deployment.yaml
        deployment_files = list(pkg_dir.rglob("deployment.yaml"))
        assert len(deployment_files) == 0

    def test_configure_non_ts_still_generates_deployment(self, tmp_path):
        from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages

        plan = {
            "intentId": "intent-20260306-0003",
            "intentType": "config.update",
            "actions": [{
                "kind": "configure",
                "component": "free5gc-amf",
                "naming": {"domain": "core", "site": "lab", "slice": "embb", "instance": "i001"},
            }],
            "policy": {
                "requireHumanReview": True,
                "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True},
            },
        }
        pkg_dir = generate_kpt_packages(plan, tmp_path)

        # Non-TS configure should still produce deployment
        deployment_files = list(pkg_dir.rglob("deployment.yaml"))
        assert len(deployment_files) == 1


# ── E2 Closed-Loop TS Integration ─────────────────────────────────────


class TestE2ClosedLoopTS:
    def test_act_traffic_steer_dry_run(self):
        sim = E2KpmSimulator(seed=42)
        bridge = KpimonBridge(simulator=sim)
        ctrl = ClosedLoopE2EController(
            bridge=bridge, dry_run=True,
        )
        recommendation = {
            "action": "traffic_steer",
            "cell_id": "cell-001",
            "threshold": 20,
            "reason": "CQI low",
        }
        result = ctrl._act_traffic_steer(recommendation)
        assert result is not None
        assert result["dry_run"] is True
        assert result["threshold"] == 20

    def test_act_traffic_steer_with_a1_client(self):
        sim = E2KpmSimulator(seed=42)
        bridge = KpimonBridge(simulator=sim)
        mock_a1 = MagicMock()
        ctrl = ClosedLoopE2EController(
            bridge=bridge, a1_client=mock_a1, dry_run=False,
        )
        recommendation = {
            "action": "traffic_steer",
            "cell_id": "cell-001",
            "threshold": 15,
            "reason": "CQI degraded",
        }
        result = ctrl._act_traffic_steer(recommendation)
        mock_a1.create_ts_policy.assert_called_once_with("ts-cell-001", threshold=15)
        assert result["method"] == "a1_direct"
        assert result["policy_id"] == "ts-cell-001"
