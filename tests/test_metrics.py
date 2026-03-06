"""Tests for llm_nephio_oran.observability.metrics (ADR-010 Observe phase)."""
from __future__ import annotations

from unittest.mock import patch, MagicMock

import pytest

from llm_nephio_oran.observability.metrics import (
    snapshot_metrics,
    snapshot_all_components,
    check_scaling_thresholds,
    _prom_query,
    _scalar,
)


class TestScalar:
    def test_extract_value(self):
        results = [{"value": [1234567890, "0.75"]}]
        assert _scalar(results) == 0.75

    def test_empty_results(self):
        assert _scalar([]) == 0.0

    def test_default_value(self):
        assert _scalar([], default=42.0) == 42.0


class TestPromQuery:
    def test_successful_query(self, mock_prometheus):
        mock_resp = mock_prometheus([{"value": [0, "10"]}])
        with patch("llm_nephio_oran.observability.metrics.requests.get", return_value=mock_resp):
            result = _prom_query("up")
        assert len(result) == 1
        assert result[0]["value"][1] == "10"

    def test_failed_query_returns_empty(self):
        import requests as req_lib
        with patch("llm_nephio_oran.observability.metrics.requests.get",
                   side_effect=req_lib.ConnectionError("timeout")):
            result = _prom_query("up")
        assert result == []

    def test_error_status_returns_empty(self):
        mock_resp = MagicMock()
        mock_resp.raise_for_status = MagicMock()
        mock_resp.json.return_value = {"status": "error", "error": "bad query"}
        with patch("llm_nephio_oran.observability.metrics.requests.get", return_value=mock_resp):
            result = _prom_query("bad{query")
        assert result == []


class TestSnapshotMetrics:
    def _mock_get(self, responses: dict[str, list]):
        """Mock requests.get based on query parameter content."""
        def side_effect(*args, **kwargs):
            query = kwargs.get("params", {}).get("query", "")
            mock = MagicMock()
            mock.raise_for_status = MagicMock()
            for key, results in responses.items():
                if key in query:
                    mock.json.return_value = {"status": "success", "data": {"result": results}}
                    return mock
            mock.json.return_value = {"status": "success", "data": {"result": []}}
            return mock
        return side_effect

    def test_snapshot_returns_structure(self):
        with patch("llm_nephio_oran.observability.metrics.requests.get",
                   side_effect=self._mock_get({
                       "cpu_usage": [{"value": [0, "2.5"]}],
                       "memory_working_set": [{"value": [0, str(1024*1024*512)]}],
                       "kube_pod_info": [{"value": [0, "5"]}],
                       "kube_pod_status_ready": [{"value": [0, "4"]}],
                   })):
            result = snapshot_metrics(namespace="ricplt")
        assert "timestamp" in result
        assert result["k8s"]["cpu_usage_cores"] == 2.5
        assert result["k8s"]["memory_usage_mb"] == 512.0
        assert result["k8s"]["pod_count"] == 5
        assert result["k8s"]["ready_count"] == 4

    def test_snapshot_handles_empty_results(self):
        with patch("llm_nephio_oran.observability.metrics.requests.get",
                   side_effect=self._mock_get({})):
            result = snapshot_metrics()
        assert result["k8s"]["cpu_usage_cores"] == 0.0
        assert result["k8s"]["pod_count"] == 0


class TestCheckScalingThresholds:
    def _mock_get_with_utilization(self, cpu_util: float, mem_util: float):
        def side_effect(*args, **kwargs):
            query = kwargs.get("params", {}).get("query", "")
            mock = MagicMock()
            mock.raise_for_status = MagicMock()
            if "/" in query and "cpu_usage" in query:
                mock.json.return_value = {"status": "success", "data": {"result": [{"value": [0, str(cpu_util)]}]}}
            elif "/" in query and "memory" in query:
                mock.json.return_value = {"status": "success", "data": {"result": [{"value": [0, str(mem_util)]}]}}
            else:
                mock.json.return_value = {"status": "success", "data": {"result": []}}
            return mock
        return side_effect

    def test_no_threshold_exceeded(self):
        with patch("llm_nephio_oran.observability.metrics.requests.get",
                   side_effect=self._mock_get_with_utilization(0.5, 0.4)):
            result = check_scaling_thresholds("ricplt")
        assert result["thresholds_exceeded"] is False
        assert len(result["recommendations"]) == 0

    def test_cpu_threshold_exceeded(self):
        with patch("llm_nephio_oran.observability.metrics.requests.get",
                   side_effect=self._mock_get_with_utilization(0.95, 0.4)):
            result = check_scaling_thresholds("ricplt", cpu_threshold=0.8)
        assert result["thresholds_exceeded"] is True
        assert any("CPU" in r["reason"] for r in result["recommendations"])

    def test_memory_threshold_exceeded(self):
        with patch("llm_nephio_oran.observability.metrics.requests.get",
                   side_effect=self._mock_get_with_utilization(0.3, 0.92)):
            result = check_scaling_thresholds("ricplt", memory_threshold=0.8)
        assert result["thresholds_exceeded"] is True
        assert any("Memory" in r["reason"] for r in result["recommendations"])


class TestSnapshotAllComponents:
    def test_returns_all_namespaces(self):
        mock_resp = MagicMock()
        mock_resp.raise_for_status = MagicMock()
        mock_resp.json.return_value = {"status": "success", "data": {"result": []}}
        with patch("llm_nephio_oran.observability.metrics.requests.get", return_value=mock_resp):
            result = snapshot_all_components()
        assert "ric" in result["components"]
        assert "core" in result["components"]
        assert "monitoring" in result["components"]
