"""Tests for /api/metrics/json endpoint and collect_json_snapshot().

TDD tests covering:
  1. collect_json_snapshot() pure function (unit tests)
  2. /api/metrics/json HTTP endpoint (integration via TestClient)
  3. Histogram p95 calculation edge cases
"""
from __future__ import annotations

import pytest
from starlette.testclient import TestClient


@pytest.fixture(autouse=True)
def _reset():
    from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
    reset_metrics()
    yield


# ── collect_json_snapshot unit tests ─────────────────────────────────


class TestCollectJsonSnapshot:
    """Pure function tests for collect_json_snapshot()."""

    def test_empty_state_returns_all_sections(self):
        from llm_nephio_oran.observability.pipeline_metrics import collect_json_snapshot
        snap = collect_json_snapshot()
        assert "overview" in snap
        assert "e2kpm" in snap
        assert "scale_actions" in snap
        assert "porch_lifecycle" in snap
        assert "pipeline_stages" in snap
        assert "timestamp" in snap

    def test_empty_state_overview_all_zeros(self):
        from llm_nephio_oran.observability.pipeline_metrics import collect_json_snapshot
        snap = collect_json_snapshot()
        ov = snap["overview"]
        assert ov["active_intents"] == 0
        assert ov["total_created"] == 0
        assert ov["completed"] == 0
        assert ov["failed"] == 0
        assert ov["git_commits"] == 0

    def test_empty_state_arrays_empty(self):
        from llm_nephio_oran.observability.pipeline_metrics import collect_json_snapshot
        snap = collect_json_snapshot()
        assert snap["e2kpm"]["cells"] == []
        assert snap["scale_actions"] == []
        assert snap["porch_lifecycle"] == []
        assert snap["pipeline_stages"] == []

    def test_intent_created_reflected(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.intent_created.labels(source="web").inc()
        pm.intent_created.labels(source="web").inc()
        pm.intent_created.labels(source="closedloop").inc()
        snap = pm.collect_json_snapshot()
        assert snap["overview"]["total_created"] == 3

    def test_intent_completed_reflected(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.intent_completed.labels(result="completed").inc()
        pm.intent_completed.labels(result="completed").inc()
        pm.intent_completed.labels(result="failed").inc()
        snap = pm.collect_json_snapshot()
        assert snap["overview"]["completed"] == 2
        assert snap["overview"]["failed"] == 1

    def test_active_intents_gauge(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.active_intents.inc()
        pm.active_intents.inc()
        snap = pm.collect_json_snapshot()
        assert snap["overview"]["active_intents"] == 2

    def test_git_commits_reflected(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.git_commits.inc()
        pm.git_commits.inc()
        pm.git_commits.inc()
        snap = pm.collect_json_snapshot()
        assert snap["overview"]["git_commits"] == 3

    def test_e2kpm_cells_reflected(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.e2kpm_prb_usage.labels(cell_id="cell-001", gnb_id="gnb-edge01").set(87.3)
        pm.e2kpm_prb_usage.labels(cell_id="cell-002", gnb_id="gnb-edge01").set(45.1)
        snap = pm.collect_json_snapshot()
        cells = snap["e2kpm"]["cells"]
        assert len(cells) == 2
        c1 = next(c for c in cells if c["cell_id"] == "cell-001")
        assert c1["gnb_id"] == "gnb-edge01"
        assert c1["prb_usage_percent"] == 87.3
        c2 = next(c for c in cells if c["cell_id"] == "cell-002")
        assert c2["prb_usage_percent"] == 45.1

    def test_scale_actions_reflected(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.scale_actions.labels(component="oai-odu", action="scale_out").inc()
        pm.scale_actions.labels(component="oai-odu", action="scale_out").inc()
        pm.scale_actions.labels(component="free5gc-upf", action="scale_in").inc()
        snap = pm.collect_json_snapshot()
        sa = snap["scale_actions"]
        assert len(sa) == 2
        odu = next(s for s in sa if s["component"] == "oai-odu")
        assert odu["action"] == "scale_out"
        assert odu["count"] == 2
        upf = next(s for s in sa if s["component"] == "free5gc-upf")
        assert upf["count"] == 1

    def test_porch_lifecycle_reflected(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.porch_lifecycle.labels(lifecycle="Proposed").inc()
        pm.porch_lifecycle.labels(lifecycle="Proposed").inc()
        pm.porch_lifecycle.labels(lifecycle="Published").inc()
        snap = pm.collect_json_snapshot()
        pl = snap["porch_lifecycle"]
        assert len(pl) == 2
        proposed = next(p for p in pl if p["lifecycle"] == "Proposed")
        assert proposed["count"] == 2

    def test_pipeline_stages_reflected(self):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.pipeline_stage_duration.labels(stage="planning").observe(1.5)
        pm.pipeline_stage_duration.labels(stage="planning").observe(2.5)
        pm.pipeline_stage_duration.labels(stage="validating").observe(0.1)
        snap = pm.collect_json_snapshot()
        stages = snap["pipeline_stages"]
        assert len(stages) == 2
        planning = next(s for s in stages if s["stage"] == "planning")
        assert planning["count"] == 2
        assert planning["sum_seconds"] == pytest.approx(4.0, abs=0.01)
        assert planning["p95_seconds"] is not None

    def test_timestamp_is_iso_format(self):
        from llm_nephio_oran.observability.pipeline_metrics import collect_json_snapshot
        from datetime import datetime
        snap = collect_json_snapshot()
        # Should be parseable as ISO 8601
        dt = datetime.fromisoformat(snap["timestamp"])
        assert dt is not None

    def test_overview_values_are_integers_when_whole(self):
        """Ensure counters return ints, not floats like 3.0."""
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.intent_created.labels(source="web").inc()
        snap = pm.collect_json_snapshot()
        assert isinstance(snap["overview"]["total_created"], int)
        assert isinstance(snap["overview"]["active_intents"], int)


# ── Histogram p95 edge cases ─────────────────────────────────────────


class TestHistogramQuantile:
    """Edge cases for _histogram_quantile()."""

    def test_no_observations_returns_none(self):
        from llm_nephio_oran.observability.pipeline_metrics import _histogram_quantile
        assert _histogram_quantile(0.95, [], 0) is None

    def test_single_observation(self):
        from llm_nephio_oran.observability.pipeline_metrics import _histogram_quantile
        # One observation of 0.3s falls in bucket (0.25, 0.5]
        buckets = [
            (0.05, 0), (0.1, 0), (0.25, 0), (0.5, 1),
            (1.0, 1), (2.5, 1), (5.0, 1), (10.0, 1), (30.0, 1), (60.0, 1),
            (float("inf"), 1),
        ]
        p95 = _histogram_quantile(0.95, buckets, 1)
        assert p95 is not None
        # p95 of a single observation in (0.25, 0.5] bucket should be ~0.4875
        assert 0.25 <= p95 <= 0.5

    def test_all_in_inf_bucket(self):
        from llm_nephio_oran.observability.pipeline_metrics import _histogram_quantile
        # All observations exceed all finite bounds
        buckets = [
            (0.05, 0), (0.1, 0), (0.25, 0), (0.5, 0),
            (1.0, 0), (2.5, 0), (5.0, 0), (10.0, 0), (30.0, 0), (60.0, 0),
            (float("inf"), 5),
        ]
        p95 = _histogram_quantile(0.95, buckets, 5)
        # Should return last finite bound (60.0)
        assert p95 == 60.0

    def test_multiple_observations_interpolation(self):
        from llm_nephio_oran.observability.pipeline_metrics import _histogram_quantile
        # 100 observations: 90 in (0, 1.0], 10 in (1.0, 2.5]
        buckets = [
            (0.05, 5), (0.1, 10), (0.25, 25), (0.5, 50),
            (1.0, 90), (2.5, 100), (5.0, 100), (10.0, 100),
            (30.0, 100), (60.0, 100), (float("inf"), 100),
        ]
        p95 = _histogram_quantile(0.95, buckets, 100)
        assert p95 is not None
        # p95 target = 95th obs, falls in (1.0, 2.5] bucket
        assert 1.0 <= p95 <= 2.5


# ── HTTP endpoint integration tests ──────────────────────────────────


class TestMetricsJsonEndpoint:
    """Test /api/metrics/json via TestClient."""

    @pytest.fixture()
    def client(self, tmp_path):
        from llm_nephio_oran.intentd.app_v2 import create_app
        app = create_app(store_dir=tmp_path)
        return TestClient(app)

    def test_returns_200(self, client):
        resp = client.get("/api/metrics/json")
        assert resp.status_code == 200

    def test_response_has_all_sections(self, client):
        resp = client.get("/api/metrics/json")
        data = resp.json()
        assert "overview" in data
        assert "e2kpm" in data
        assert "scale_actions" in data
        assert "porch_lifecycle" in data
        assert "pipeline_stages" in data
        assert "timestamp" in data

    def test_reflects_metrics_after_increment(self, client):
        from llm_nephio_oran.observability import pipeline_metrics as pm
        pm.intent_created.labels(source="web").inc()
        pm.scale_actions.labels(component="oai-odu", action="scale_out").inc()
        pm.e2kpm_prb_usage.labels(cell_id="cell-001", gnb_id="gnb-edge01").set(85.5)

        resp = client.get("/api/metrics/json")
        data = resp.json()
        assert data["overview"]["total_created"] == 1
        assert len(data["scale_actions"]) == 1
        assert data["scale_actions"][0]["component"] == "oai-odu"
        assert len(data["e2kpm"]["cells"]) == 1
        assert data["e2kpm"]["cells"][0]["prb_usage_percent"] == 85.5

    def test_empty_state_returns_zeros(self, client):
        resp = client.get("/api/metrics/json")
        data = resp.json()
        ov = data["overview"]
        assert ov["active_intents"] == 0
        assert ov["total_created"] == 0
        assert ov["completed"] == 0
        assert ov["failed"] == 0
        assert ov["git_commits"] == 0
