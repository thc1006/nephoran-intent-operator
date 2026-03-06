"""Shared pytest fixtures for llm_nephio_oran test suite."""
from __future__ import annotations

import json
import os
import tempfile
from pathlib import Path
from typing import Any, Generator
from unittest.mock import MagicMock, patch

import pytest

# Ensure we use the project root for schema resolution
PROJECT_ROOT = Path(__file__).parent.parent
SCHEMA_PATH = str(PROJECT_ROOT / "schemas" / "intent-plan.schema.json")


def _make_valid_plan(**overrides: Any) -> dict[str, Any]:
    """Create a minimal valid IntentPlan dict."""
    plan: dict[str, Any] = {
        "intentId": "intent-20260305-0001",
        "intentType": "slice.deploy",
        "description": "Test intent",
        "slice": {"sliceType": "eMBB", "name": "embb-lab-001", "site": "lab"},
        "constraints": {"minReplicas": 1, "maxReplicas": 5, "allowedNamespaces": ["oran"]},
        "actions": [
            {
                "kind": "deploy",
                "component": "ric-kpimon",
                "replicas": 1,
                "naming": {"domain": "ric", "site": "lab", "slice": "embb", "instance": "i001"},
            }
        ],
        "policy": {
            "requireHumanReview": True,
            "guardrails": {
                "denyClusterScoped": True,
                "denyPrivileged": True,
                "denyHostNetwork": True,
                "denyCRDChanges": True,
                "denyRBACChanges": True,
            },
        },
        "metadata": {
            "createdAt": "2026-03-05T00:00:00Z",
            "createdBy": "test",
            "source": "cli",
        },
    }
    plan.update(overrides)
    return plan


@pytest.fixture
def valid_plan() -> dict[str, Any]:
    return _make_valid_plan()


@pytest.fixture
def valid_plan_scale() -> dict[str, Any]:
    return _make_valid_plan(
        intentType="slice.scale",
        actions=[
            {
                "kind": "scale",
                "component": "free5gc-amf",
                "replicas": 3,
                "naming": {"domain": "core", "site": "lab", "slice": "embb", "instance": "i001"},
            }
        ],
    )


@pytest.fixture
def schema_path() -> str:
    return SCHEMA_PATH


@pytest.fixture
def tmp_pkg_dir(tmp_path: Path) -> Path:
    """Temporary directory for generated kpt packages."""
    d = tmp_path / "packages" / "instances"
    d.mkdir(parents=True)
    return d


@pytest.fixture
def mock_ollama_response():
    """Mock a successful Ollama /v1/chat/completions response."""
    def _make(plan_json: dict[str, Any]) -> MagicMock:
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.raise_for_status = MagicMock()
        mock_resp.json.return_value = {
            "choices": [
                {"message": {"content": json.dumps(plan_json)}}
            ]
        }
        return mock_resp
    return _make


@pytest.fixture
def mock_prometheus():
    """Mock Prometheus API responses."""
    def _make(results: list[dict] | None = None) -> MagicMock:
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_resp.raise_for_status = MagicMock()
        mock_resp.json.return_value = {
            "status": "success",
            "data": {"result": results or []},
        }
        return mock_resp
    return _make
