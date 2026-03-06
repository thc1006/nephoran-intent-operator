"""Tests for llm_nephio_oran.closedloop.analyzer (ADR-010 Analyze phase).

Guard rails per ADR-010:
- Max 5 replicas per component
- 5-minute cooldown between same-component actions
- Human approval gate for scale-out > 3
- Auto rollback if health check fails within 2 minutes
"""
from __future__ import annotations

import time
from unittest.mock import patch, MagicMock

import pytest


class TestAnalyzeMetrics:
    @pytest.fixture
    def analyzer(self):
        from llm_nephio_oran.closedloop.analyzer import MetricsAnalyzer
        return MetricsAnalyzer(max_replicas=5, cooldown_seconds=300)

    def test_high_cpu_recommends_scale_out(self, analyzer):
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.92,
            "memory_utilization": 0.5,
            "current_replicas": 1,
        }
        result = analyzer.analyze(metrics)
        assert result["action"] == "scale_out"
        assert result["recommended_replicas"] > 1

    def test_low_utilization_recommends_scale_in(self, analyzer):
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.15,
            "memory_utilization": 0.10,
            "current_replicas": 3,
        }
        result = analyzer.analyze(metrics)
        assert result["action"] == "scale_in"
        assert result["recommended_replicas"] < 3

    def test_within_bounds_no_action(self, analyzer):
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.55,
            "memory_utilization": 0.50,
            "current_replicas": 2,
        }
        result = analyzer.analyze(metrics)
        assert result["action"] == "none"

    def test_max_replicas_guard(self, analyzer):
        """Never exceed max_replicas (default 5) per ADR-010."""
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.99,
            "memory_utilization": 0.95,
            "current_replicas": 5,
        }
        result = analyzer.analyze(metrics)
        assert result["action"] == "none"
        assert "max replicas" in result.get("reason", "").lower()

    def test_cooldown_respected(self, analyzer):
        """No action if last action was < cooldown_seconds ago."""
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.95,
            "memory_utilization": 0.5,
            "current_replicas": 2,
        }
        # First call should recommend action
        result1 = analyzer.analyze(metrics)
        assert result1["action"] == "scale_out"

        # Record the action
        analyzer.record_action("oai-odu", "scale_out")

        # Second call within cooldown should be blocked
        result2 = analyzer.analyze(metrics)
        assert result2["action"] == "none"
        assert "cooldown" in result2.get("reason", "").lower()

    def test_human_approval_gate(self, analyzer):
        """Require human approval for scale-out > 3 replicas per ADR-010."""
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.95,
            "memory_utilization": 0.5,
            "current_replicas": 3,
        }
        result = analyzer.analyze(metrics)
        assert result["action"] == "scale_out"
        assert result["requires_human_approval"] is True

    def test_scale_in_never_below_1(self, analyzer):
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.05,
            "memory_utilization": 0.05,
            "current_replicas": 1,
        }
        result = analyzer.analyze(metrics)
        # Already at 1 replica, can't scale in further
        assert result["action"] == "none"

    def test_memory_threshold_triggers_scale_out(self, analyzer):
        metrics = {
            "namespace": "oran",
            "component": "free5gc-upf",
            "cpu_utilization": 0.4,
            "memory_utilization": 0.88,
            "current_replicas": 2,
        }
        result = analyzer.analyze(metrics)
        assert result["action"] == "scale_out"
