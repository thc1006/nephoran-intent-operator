"""Tests for llm_nephio_oran.intentd.app (ADR-007, ADR-0001)."""
from __future__ import annotations

import json
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient

from llm_nephio_oran.intentd.app import app


@pytest.fixture
def client(tmp_path):
    """TestClient with isolated state directory."""
    with patch("llm_nephio_oran.intentd.app.APP_STATE_DIR", tmp_path), \
         patch("llm_nephio_oran.intentd.app.INTENTS_FILE", tmp_path / "intents.jsonl"):
        yield TestClient(app)


class TestHealthz:
    def test_health_ok(self, client):
        resp = client.get("/healthz")
        assert resp.status_code == 200
        assert resp.json()["status"] == "ok"


class TestCreateIntent:
    def test_create_with_text_stub(self, client):
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "text": "Deploy eMBB slice",
            "use_llm": False,
        })
        assert resp.status_code == 201
        data = resp.json()
        assert data["status"] == "planned"
        assert data["id"].startswith("intent-")
        assert data["plan"]["intentType"] == "slice.deploy"

    def test_create_with_payload(self, client, valid_plan):
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "payload": valid_plan,
        })
        assert resp.status_code == 201
        assert resp.json()["id"] == "intent-20260305-0001"

    def test_create_without_text_or_payload_fails(self, client):
        resp = client.post("/tmf-api/intent/v5/intent", json={})
        assert resp.status_code == 400

    def test_create_with_scale_text(self, client):
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "text": "Scale AMF to 5 replicas",
            "use_llm": False,
        })
        assert resp.status_code == 201
        assert resp.json()["plan"]["intentType"] == "slice.scale"


class TestGetIntent:
    def test_get_existing_intent(self, client):
        # Create first
        create_resp = client.post("/tmf-api/intent/v5/intent", json={
            "text": "Deploy test", "use_llm": False,
        })
        intent_id = create_resp.json()["id"]
        # Get
        resp = client.get(f"/tmf-api/intent/v5/intent/{intent_id}")
        assert resp.status_code == 200
        assert resp.json()["id"] == intent_id

    def test_get_nonexistent_returns_404(self, client):
        resp = client.get("/tmf-api/intent/v5/intent/nonexistent")
        assert resp.status_code == 404


class TestValidateIntentPlan:
    def test_valid_plan(self, client, valid_plan):
        resp = client.post("/internal/validate/intent-plan", json=valid_plan)
        assert resp.status_code == 200
        assert resp.json()["valid"] is True

    def test_invalid_plan(self, client):
        with pytest.raises(ValueError, match="Schema validation failed"):
            client.post("/internal/validate/intent-plan", json={"bad": "data"})


class TestPipeline:
    def test_pipeline_dry_run(self, client):
        resp = client.post("/internal/pipeline", json={
            "text": "Deploy eMBB slice",
            "use_llm": False,
            "dry_run": True,
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "packages_generated"
        assert data["id"].startswith("intent-")


class TestListIntents:
    def test_empty_list(self, client):
        resp = client.get("/api/intents")
        assert resp.status_code == 200
        assert resp.json() == []

    def test_list_after_create(self, client):
        client.post("/tmf-api/intent/v5/intent", json={"text": "Deploy test", "use_llm": False})
        resp = client.get("/api/intents")
        assert resp.status_code == 200
        intents = resp.json()
        assert len(intents) >= 1
        assert intents[0]["id"].startswith("intent-")

    def test_deduplicates_by_id(self, client):
        """Multiple entries for same id should show only latest."""
        client.post("/tmf-api/intent/v5/intent", json={"text": "Deploy A", "use_llm": False})
        client.post("/tmf-api/intent/v5/intent", json={"text": "Deploy B", "use_llm": False})
        resp = client.get("/api/intents")
        ids = [i["id"] for i in resp.json()]
        assert len(ids) == len(set(ids))  # no duplicates


class TestPorchPackages:
    def test_returns_list(self, client):
        with patch("llm_nephio_oran.intentd.app.PorchClient", create=True) as MockPC:
            instance = MockPC.return_value
            instance.list_packages.return_value = [
                {
                    "metadata": {"name": "pkg1"},
                    "spec": {"packageName": "catalog/test", "lifecycle": "Published",
                             "workspaceName": "main", "repository": "nephoran-packages"},
                },
            ]
            # Need to re-import to pick up mock — just call directly
            from llm_nephio_oran.intentd.app import list_porch_packages
            with patch("llm_nephio_oran.intentd.app.PorchClient", MockPC):
                result = list_porch_packages()
        assert len(result) >= 1
        assert result[0]["lifecycle"] == "Published"

    def test_handles_error_gracefully(self, client):
        resp = client.get("/api/porch/packages")
        # Even if Porch is unreachable in test, should return empty list not 500
        assert resp.status_code == 200


class TestScaleStatus:
    def test_returns_structure(self, client):
        resp = client.get("/api/scale/status")
        assert resp.status_code == 200
        data = resp.json()
        assert "components" in data
