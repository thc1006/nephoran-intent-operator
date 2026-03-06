"""Tests for llm_nephio_oran.closedloop.controller (ADR-010 — full Observe→Analyze→Act cycle)."""
from __future__ import annotations

from unittest.mock import patch, MagicMock, call

import pytest


class TestClosedLoopController:
    @pytest.fixture
    def controller(self):
        from llm_nephio_oran.closedloop.controller import ClosedLoopController
        return ClosedLoopController(
            namespaces=["oran", "free5gc"],
            max_replicas=5,
            cooldown_seconds=300,
        )

    def test_observe_phase(self, controller):
        """Observe: collect Prometheus metrics snapshot."""
        mock_snapshot = {
            "timestamp": "2026-03-05T12:00:00Z",
            "namespace": "oran",
            "k8s": {"cpu_usage_cores": 3.0, "memory_usage_mb": 2048.0, "pod_count": 4, "ready_count": 4},
        }
        with patch("llm_nephio_oran.closedloop.controller.snapshot_metrics", return_value=mock_snapshot):
            result = controller.observe("oran")
        assert result["k8s"]["pod_count"] == 4

    def test_analyze_phase(self, controller):
        """Analyze: evaluate metrics against thresholds."""
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.92,
            "memory_utilization": 0.5,
            "current_replicas": 2,
        }
        result = controller.analyze(metrics)
        assert result["action"] == "scale_out"

    def test_act_creates_intent_plan(self, controller):
        """Act: scaling recommendation → IntentPlan → Git PR (never kubectl)."""
        recommendation = {
            "action": "scale_out",
            "recommended_replicas": 3,
            "reason": "CPU 92% > 80%",
            "requires_human_approval": False,
        }
        with patch("llm_nephio_oran.closedloop.controller.plan_from_text") as mock_plan, \
             patch("llm_nephio_oran.closedloop.controller.generate_kpt_packages") as mock_gen, \
             patch("llm_nephio_oran.closedloop.controller.validate_package", return_value=[]), \
             patch("llm_nephio_oran.closedloop.controller.push_to_gitops") as mock_git:

            mock_plan.return_value = {
                "intentId": "intent-20260305-9001",
                "intentType": "closedloop.act",
                "actions": [{"kind": "scale", "component": "oai-odu", "replicas": 3}],
                "policy": {"requireHumanReview": False,
                           "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True}},
            }
            mock_gen.return_value = MagicMock()
            mock_git.return_value = {"branch": "intent/test", "commit_sha": "abc123", "pr": {"number": 5}}

            result = controller.act("oai-odu", "oran", recommendation)

        assert result is not None
        assert result["pr"]["number"] == 5
        mock_plan.assert_called_once()
        mock_git.assert_called_once()

    def test_act_skipped_for_no_action(self, controller):
        recommendation = {
            "action": "none",
            "recommended_replicas": 2,
            "reason": "Within bounds",
            "requires_human_approval": False,
        }
        result = controller.act("oai-odu", "oran", recommendation)
        assert result is None

    def test_full_cycle(self, controller):
        """Full Observe→Analyze→Act cycle produces a PR."""
        mock_thresholds = {
            "cpu_utilization": 0.90,
            "memory_utilization": 0.50,
            "thresholds_exceeded": True,
            "metrics": {"k8s": {"pod_count": 2}},
        }
        with patch("llm_nephio_oran.closedloop.controller.check_scaling_thresholds", return_value=mock_thresholds), \
             patch("llm_nephio_oran.closedloop.controller.snapshot_metrics"), \
             patch("llm_nephio_oran.closedloop.controller.plan_from_text") as mock_plan, \
             patch("llm_nephio_oran.closedloop.controller.generate_kpt_packages") as mock_gen, \
             patch("llm_nephio_oran.closedloop.controller.validate_package", return_value=[]), \
             patch("llm_nephio_oran.closedloop.controller.push_to_gitops") as mock_git:

            mock_plan.return_value = {
                "intentId": "intent-20260305-9002",
                "intentType": "closedloop.act",
                "actions": [{"kind": "scale", "component": "oai-odu", "replicas": 3}],
                "policy": {"requireHumanReview": False,
                           "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True}},
            }
            mock_gen.return_value = MagicMock()
            mock_git.return_value = {"branch": "intent/test", "commit_sha": "abc", "pr": {"number": 10}}

            results = controller.run_cycle()

        assert len(results) > 0
        # At least one namespace should have been checked
        assert any(r.get("pr") for r in results if r is not None)

    def test_cooldown_blocks_repeat_action(self, controller):
        """After an action, same component is blocked for cooldown period."""
        controller.analyzer.record_action("oai-odu", "scale_out")
        metrics = {
            "namespace": "oran",
            "component": "oai-odu",
            "cpu_utilization": 0.95,
            "memory_utilization": 0.5,
            "current_replicas": 3,
        }
        result = controller.analyze(metrics)
        assert result["action"] == "none"
        assert "cooldown" in result["reason"].lower()
