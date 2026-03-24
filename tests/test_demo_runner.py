"""Tests for M7 Phase C demo runner (TDD RED→GREEN→Refactor).

Verifies that DemoRunner:
  1. Initializes E2 KPM simulator + closed-loop controller
  2. Observe phase collects cell metrics
  3. Analyze phase produces scaling recommendations
  4. Act phase creates TMF921 intents
  5. Full cycle runs all phases end-to-end
  6. Metrics are emitted during pipeline execution
  7. Dry-run mode avoids Git/Porch side effects
  8. Summary report captures all phase results
"""
from __future__ import annotations

from unittest.mock import patch

import pytest


class TestDemoRunnerInit:
    """DemoRunner initializes E2 KPM simulator and closed-loop controller."""

    def test_default_construction(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner()
        assert demo is not None
        assert demo.dry_run is True  # safe default

    def test_custom_load_factor(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42)
        assert demo._load_factor == 0.95

    def test_custom_cells(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        from llm_nephio_oran.e2sim.kpm_simulator import CellConfig
        cells = [
            CellConfig(cell_id="c1", gnb_id="gnb-edge01"),
            CellConfig(cell_id="c2", gnb_id="gnb-edge02"),
        ]
        demo = DemoRunner(cells=cells, seed=42)
        assert len(demo._cells) == 2

    def test_live_mode_construction(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(dry_run=False)
        assert demo.dry_run is False


class TestDemoObserve:
    """Observe phase collects E2 KPM metrics from simulator."""

    @pytest.fixture(autouse=True)
    def _reset(self):
        from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
        reset_metrics()
        yield

    def test_observe_returns_cell_summaries(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.5, seed=42)
        result = demo.observe()
        assert "cell_summaries" in result
        assert len(result["cell_summaries"]) > 0

    def test_observe_cell_has_required_fields(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.5, seed=42)
        result = demo.observe()
        for cell_id, summary in result["cell_summaries"].items():
            assert "gnb_id" in summary
            assert "avg_prb_usage" in summary
            assert "total_ues" in summary

    def test_observe_high_load_produces_high_prb(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42)
        result = demo.observe()
        for cell_id, summary in result["cell_summaries"].items():
            # At 0.95 load factor, avg PRB across slices should be >40%
            # (eMBB ~86%, URLLC ~28%, avg ~57%)
            assert summary["avg_prb_usage"] > 40.0

    def test_observe_updates_prometheus_gauge(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.5, seed=42)
        demo.observe()
        # PRB gauge should be set for each cell (uses module-level ref, safe after reset)
        val = pm.get_metric_value(
            pm.e2kpm_prb_usage,
            {"cell_id": "cell-001", "gnb_id": "gnb-edge01"},
        )
        assert val > 0.0


class TestDemoAnalyze:
    """Analyze phase produces scaling/steering recommendations."""

    @pytest.fixture(autouse=True)
    def _reset(self):
        from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
        reset_metrics()
        yield

    def test_analyze_after_observe(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42)
        demo.observe()
        result = demo.analyze()
        assert "recommendations" in result
        assert len(result["recommendations"]) > 0

    def test_analyze_high_prb_recommends_scale_out(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42, prb_threshold=80.0)
        demo.observe()
        result = demo.analyze()
        actions = [r["action"] for r in result["recommendations"]]
        assert "scale_out" in actions

    def test_analyze_low_load_recommends_none(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.2, seed=42, prb_threshold=80.0)
        demo.observe()
        result = demo.analyze()
        actions = [r["action"] for r in result["recommendations"]]
        # Low load should not trigger scaling
        assert "scale_out" not in actions

    def test_analyze_without_observe_raises(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner()
        with pytest.raises(RuntimeError, match="observe"):
            demo.analyze()


class TestDemoAct:
    """Act phase creates TMF921 intents from recommendations."""

    @pytest.fixture(autouse=True)
    def _reset(self):
        from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
        reset_metrics()
        yield

    def test_act_dry_run_returns_intents(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42, dry_run=True)
        demo.observe()
        demo.analyze()
        result = demo.act()
        assert "intents" in result
        # At high load, at least one intent should be created
        assert len(result["intents"]) > 0

    def test_act_dry_run_does_not_call_api(self):
        """Dry-run acts locally without posting to TMF921 API."""
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42, dry_run=True)
        demo.observe()
        demo.analyze()
        result = demo.act()
        for intent in result["intents"]:
            assert intent.get("dry_run") is True

    def test_act_without_analyze_raises(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42)
        demo.observe()
        with pytest.raises(RuntimeError, match="analyze"):
            demo.act()

    def test_act_live_mode_posts_to_api(self):
        """Live mode POSTs to TMF921 API at api_base."""
        from unittest.mock import MagicMock
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(
            load_factor=0.95, seed=42, dry_run=False,
            api_base="http://test-api:8080",
        )
        demo.observe()
        demo.analyze()
        mock_resp = MagicMock()
        mock_resp.status_code = 201
        mock_resp.json.return_value = {"id": "test-001", "state": "acknowledged"}
        mock_resp.raise_for_status = MagicMock()
        with patch("llm_nephio_oran.demo.runner.requests.post", return_value=mock_resp) as mock_post:
            result = demo.act()
            assert mock_post.called
            call_url = mock_post.call_args[0][0]
            assert call_url.startswith("http://test-api:8080")
            assert len(result["intents"]) > 0

    def test_act_live_mode_handles_api_error(self):
        """Live mode captures API errors gracefully."""
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42, dry_run=False)
        demo.observe()
        demo.analyze()
        with patch("llm_nephio_oran.demo.runner.requests.post",
                    side_effect=ConnectionError("refused")):
            result = demo.act()
            assert len(result["intents"]) > 0
            assert "error" in result["intents"][0]


class TestDemoFullCycle:
    """Full cycle runs Observe→Analyze→Act and collects summary."""

    @pytest.fixture(autouse=True)
    def _reset(self):
        from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
        reset_metrics()
        yield

    def test_full_cycle_returns_summary(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42, dry_run=True)
        summary = demo.run_full_cycle()
        assert "observe" in summary
        assert "analyze" in summary
        assert "act" in summary
        assert "metrics" in summary
        assert "duration_seconds" in summary

    def test_full_cycle_high_load_produces_actions(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42, dry_run=True)
        summary = demo.run_full_cycle()
        assert len(summary["act"]["intents"]) > 0

    def test_full_cycle_low_load_no_actions(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.1, seed=42, dry_run=True, prb_threshold=80.0)
        summary = demo.run_full_cycle()
        assert len(summary["act"]["intents"]) == 0

    def test_full_cycle_records_duration(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.5, seed=42, dry_run=True)
        summary = demo.run_full_cycle()
        assert summary["duration_seconds"] >= 0


class TestDemoMetrics:
    """Prometheus metrics are correctly emitted during demo."""

    @pytest.fixture(autouse=True)
    def _reset(self):
        from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
        reset_metrics()
        yield

    def test_metrics_snapshot_contains_all_keys(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42, dry_run=True)
        demo.run_full_cycle()
        snapshot = demo.get_metrics_snapshot()
        assert "active_intents" in snapshot
        assert "intents_created" in snapshot
        assert "intents_completed" in snapshot
        assert "scale_actions" in snapshot
        assert "prometheus_text" in snapshot

    def test_metrics_text_contains_nephoran_prefix(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(load_factor=0.95, seed=42, dry_run=True)
        demo.run_full_cycle()
        snapshot = demo.get_metrics_snapshot()
        assert "nephoran_" in snapshot["prometheus_text"]

    def test_scale_actions_incremented_after_act(self):
        from llm_nephio_oran.demo.runner import DemoRunner
        from llm_nephio_oran.observability import pipeline_metrics as pm
        demo = DemoRunner(load_factor=0.95, seed=42, dry_run=True)
        demo.run_full_cycle()
        snapshot = demo.get_metrics_snapshot()
        assert snapshot["scale_actions"] > 0


class TestDemoWithPipeline:
    """Demo runner can optionally trigger the full TMF921 pipeline."""

    @pytest.fixture(autouse=True)
    def _reset(self, tmp_path):
        from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
        reset_metrics()
        self._tmp = tmp_path
        yield

    def test_pipeline_mode_dry_run(self):
        """In pipeline mode, DemoRunner feeds intents through intentd v2."""
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(
            load_factor=0.95, seed=42, dry_run=True,
            pipeline_mode=True, store_dir=self._tmp,
        )
        summary = demo.run_full_cycle()
        # Pipeline mode should have pipeline_results in summary
        assert "pipeline_results" in summary
        for pr in summary["pipeline_results"]:
            assert pr["state"] in ("completed", "failed")

    def test_pipeline_mode_creates_intent_records(self):
        """Pipeline mode creates intent records in the store."""
        from llm_nephio_oran.demo.runner import DemoRunner
        demo = DemoRunner(
            load_factor=0.95, seed=42, dry_run=True,
            pipeline_mode=True, store_dir=self._tmp,
        )
        summary = demo.run_full_cycle()
        # Should have created at least one intent in the store
        assert len(summary["pipeline_results"]) > 0


class TestDemoCLI:
    """Tests for CLI entry point."""

    @pytest.fixture(autouse=True)
    def _reset(self):
        from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
        reset_metrics()
        yield

    def test_cli_json_output(self, capsys):
        from llm_nephio_oran.demo.cli import main
        ret = main(["--json", "--load", "0.95", "--seed", "42"])
        assert ret == 0
        import json
        output = capsys.readouterr().out
        data = json.loads(output)
        assert "observe" in data
        assert "act" in data

    def test_cli_default_returns_zero(self, capsys):
        from llm_nephio_oran.demo.cli import main
        ret = main(["--load", "0.5", "--seed", "42"])
        assert ret == 0

    def test_cli_pipeline_mode(self, capsys):
        from llm_nephio_oran.demo.cli import main
        ret = main(["--pipeline", "--load", "0.95", "--seed", "42"])
        assert ret == 0
