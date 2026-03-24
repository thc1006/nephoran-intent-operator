"""Tests for M7 pipeline Prometheus metrics (Phase C).

Verifies that intentd v2 pipeline exposes correct Prometheus counters,
gauges, and histograms for Grafana visualization.
"""
from __future__ import annotations

from unittest.mock import patch

import pytest


class TestPipelineMetrics:
    """Test Prometheus metrics exposed by the M7 pipeline."""

    @pytest.fixture(autouse=True)
    def _reset_registry(self):
        """Reset prometheus_client collectors between tests."""
        from llm_nephio_oran.observability.pipeline_metrics import reset_metrics
        reset_metrics()
        yield

    def test_intent_created_counter(self):
        from llm_nephio_oran.observability.pipeline_metrics import (
            intent_created, get_metric_value,
        )
        assert get_metric_value(intent_created) == 0
        intent_created.labels(source="closedloop").inc()
        assert get_metric_value(intent_created, {"source": "closedloop"}) == 1.0

    def test_intent_completed_counter(self):
        from llm_nephio_oran.observability.pipeline_metrics import (
            intent_completed, get_metric_value,
        )
        intent_completed.labels(result="completed").inc()
        intent_completed.labels(result="failed").inc()
        intent_completed.labels(result="completed").inc()
        assert get_metric_value(intent_completed, {"result": "completed"}) == 2.0
        assert get_metric_value(intent_completed, {"result": "failed"}) == 1.0

    def test_pipeline_stage_duration(self):
        from llm_nephio_oran.observability.pipeline_metrics import (
            pipeline_stage_duration, get_metric_value,
        )
        pipeline_stage_duration.labels(stage="planning").observe(0.5)
        pipeline_stage_duration.labels(stage="validating").observe(0.1)
        pipeline_stage_duration.labels(stage="executing").observe(2.0)
        # Histogram _count should increment
        assert get_metric_value(
            pipeline_stage_duration, {"stage": "planning"}, suffix="_count"
        ) == 1.0

    def test_e2kpm_prb_gauge(self):
        from llm_nephio_oran.observability.pipeline_metrics import (
            e2kpm_prb_usage, get_metric_value,
        )
        e2kpm_prb_usage.labels(cell_id="cell-001", gnb_id="gnb-edge01").set(87.3)
        assert get_metric_value(
            e2kpm_prb_usage, {"cell_id": "cell-001", "gnb_id": "gnb-edge01"}
        ) == 87.3

    def test_scale_action_counter(self):
        from llm_nephio_oran.observability.pipeline_metrics import (
            scale_actions, get_metric_value,
        )
        scale_actions.labels(component="oai-odu", action="scale_out").inc()
        scale_actions.labels(component="oai-odu", action="scale_out").inc()
        scale_actions.labels(component="oai-odu", action="scale_in").inc()
        assert get_metric_value(
            scale_actions, {"component": "oai-odu", "action": "scale_out"}
        ) == 2.0

    def test_porch_lifecycle_counter(self):
        from llm_nephio_oran.observability.pipeline_metrics import (
            porch_lifecycle, get_metric_value,
        )
        porch_lifecycle.labels(lifecycle="Proposed").inc()
        assert get_metric_value(
            porch_lifecycle, {"lifecycle": "Proposed"}
        ) == 1.0

    def test_active_intents_gauge(self):
        from llm_nephio_oran.observability.pipeline_metrics import (
            active_intents, get_metric_value,
        )
        active_intents.inc()
        active_intents.inc()
        assert get_metric_value(active_intents) == 2.0
        active_intents.dec()
        assert get_metric_value(active_intents) == 1.0

    def test_prometheus_text_output(self):
        """Verify /metrics endpoint produces valid Prometheus text."""
        from llm_nephio_oran.observability.pipeline_metrics import (
            intent_created, generate_metrics_text,
        )
        intent_created.labels(source="cli").inc()
        text = generate_metrics_text()
        assert "nephoran_intent_created_total" in text
        assert 'source="cli"' in text

    def test_instrument_pipeline_context_manager(self):
        """The instrument_stage context manager records duration."""
        from llm_nephio_oran.observability.pipeline_metrics import (
            instrument_stage, pipeline_stage_duration, get_metric_value,
        )
        with instrument_stage("planning"):
            pass  # zero-duration is fine; we only verify the count increments
        count = get_metric_value(
            pipeline_stage_duration, {"stage": "planning"}, suffix="_count"
        )
        assert count == 1.0
