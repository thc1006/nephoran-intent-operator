"""Tests for intentd v2 — unified TMF921 API with async pipeline (R2+R3)."""
from __future__ import annotations

import time
from unittest.mock import patch, MagicMock

import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def client(tmp_path):
    """TestClient with isolated store."""
    with patch("llm_nephio_oran.intentd.app_v2.STORE_DIR", tmp_path):
        from llm_nephio_oran.intentd.app_v2 import create_app
        app = create_app(store_dir=tmp_path)
        yield TestClient(app)


class TestTMF921Create:
    def test_create_returns_acknowledged(self, client):
        """POST /intent returns 201 with acknowledged state immediately."""
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Deploy eMBB slice",
            "source": "web",
        })
        assert resp.status_code == 201
        data = resp.json()
        assert data["state"] == "acknowledged"
        assert data["id"].startswith("intent-")
        assert data["expression"] == "Deploy eMBB slice"

    def test_create_without_expression_fails(self, client):
        resp = client.post("/tmf-api/intent/v5/intent", json={})
        assert resp.status_code == 422  # pydantic validation

    def test_create_with_use_llm(self, client):
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Scale AMF to 5",
            "source": "cli",
            "use_llm": True,
        })
        assert resp.status_code == 201
        assert resp.json()["state"] == "acknowledged"


class TestTMF921Get:
    def test_get_existing(self, client):
        create = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "test", "source": "cli",
        }).json()
        resp = client.get(f"/tmf-api/intent/v5/intent/{create['id']}")
        assert resp.status_code == 200
        assert resp.json()["id"] == create["id"]

    def test_get_nonexistent(self, client):
        resp = client.get("/tmf-api/intent/v5/intent/nonexistent")
        assert resp.status_code == 404

    def test_get_reflects_state_changes(self, client):
        """After pipeline starts, GET should show updated state."""
        create = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "test", "source": "web", "use_llm": False,
        }).json()
        # Pipeline runs in background — wait briefly
        time.sleep(0.5)
        resp = client.get(f"/tmf-api/intent/v5/intent/{create['id']}")
        data = resp.json()
        # Should have progressed beyond acknowledged
        assert data["state"] != "acknowledged"


class TestTMF921List:
    def test_list_empty(self, client):
        resp = client.get("/tmf-api/intent/v5/intent")
        assert resp.status_code == 200
        assert resp.json() == []

    def test_list_returns_all(self, client):
        client.post("/tmf-api/intent/v5/intent", json={"expression": "a", "source": "cli"})
        client.post("/tmf-api/intent/v5/intent", json={"expression": "b", "source": "web"})
        resp = client.get("/tmf-api/intent/v5/intent")
        assert len(resp.json()) == 2

    def test_list_newest_first(self, client):
        r1 = client.post("/tmf-api/intent/v5/intent", json={"expression": "first", "source": "cli"}).json()
        r2 = client.post("/tmf-api/intent/v5/intent", json={"expression": "second", "source": "web"}).json()
        items = client.get("/tmf-api/intent/v5/intent").json()
        assert items[0]["id"] == r2["id"]


class TestAsyncPipeline:
    def test_pipeline_progresses_to_completion(self, client):
        """With stub planner + dry_run, pipeline should reach packages_generated quickly."""
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Deploy eMBB slice",
            "source": "web",
            "use_llm": False,
            "dry_run": True,
        })
        intent_id = resp.json()["id"]
        # Wait for async pipeline to finish
        for _ in range(20):
            time.sleep(0.2)
            state = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            if state["state"] in ("generating", "completed", "failed"):
                break
        final = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
        # Should have a plan populated by the planner
        assert final["plan"] is not None
        assert final["plan"]["intentType"] in ("slice.deploy", "slice.scale")
        # History should show multiple transitions
        assert len(final["history"]) > 1

    def test_pipeline_records_failure(self, client):
        """If planner throws, state should be 'failed' with error."""
        with patch("llm_nephio_oran.planner.llm_planner.plan_from_text", side_effect=ValueError("boom")):
            resp = client.post("/tmf-api/intent/v5/intent", json={
                "expression": "bad intent",
                "source": "cli",
                "use_llm": False,
            })
            intent_id = resp.json()["id"]
            time.sleep(0.5)
            state = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            assert state["state"] == "failed"
            assert state["errors"] is not None

    def test_pipeline_stores_plan_in_intent(self, client):
        """After planning, the LLM-generated plan should be in the intent record."""
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Scale AMF to 3 replicas",
            "source": "web",
            "use_llm": False,
            "dry_run": True,
        })
        intent_id = resp.json()["id"]
        time.sleep(1)
        state = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
        assert state["plan"] is not None
        assert "actions" in state["plan"]


class TestTMF921Delete:
    def test_delete_from_acknowledged(self, client):
        """DELETE cancels an acknowledged intent (pipeline suppressed)."""
        with patch("llm_nephio_oran.intentd.app_v2._run_pipeline"):
            create = client.post("/tmf-api/intent/v5/intent", json={
                "expression": "test cancel", "source": "web",
            }).json()
        resp = client.delete(f"/tmf-api/intent/v5/intent/{create['id']}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["state"] == "cancelled"
        assert data["id"] == create["id"]

    def test_delete_nonexistent_404(self, client):
        resp = client.delete("/tmf-api/intent/v5/intent/nonexistent")
        assert resp.status_code == 404

    def test_delete_from_completed_409(self, client):
        """Cannot cancel a completed intent."""
        create = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "test", "source": "cli",
            "use_llm": False, "dry_run": True,
        }).json()
        # Wait for pipeline to complete
        for _ in range(20):
            time.sleep(0.2)
            state = client.get(f"/tmf-api/intent/v5/intent/{create['id']}").json()
            if state["state"] in ("completed", "failed"):
                break
        resp = client.delete(f"/tmf-api/intent/v5/intent/{create['id']}")
        assert resp.status_code == 409

    def test_delete_from_planning_409(self, client):
        """Cannot cancel an intent that has progressed past acknowledged."""
        with patch("llm_nephio_oran.planner.llm_planner.plan_from_text", side_effect=lambda *a, **kw: time.sleep(5)):
            create = client.post("/tmf-api/intent/v5/intent", json={
                "expression": "slow intent", "source": "web",
            }).json()
            time.sleep(0.3)
            # Should be in planning state now
            resp = client.delete(f"/tmf-api/intent/v5/intent/{create['id']}")
            assert resp.status_code == 409


class TestTMF921Report:
    def test_report_completed(self, client):
        """IntentReport for a completed intent is compliant + terminal."""
        create = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Deploy eMBB", "source": "web",
            "use_llm": False, "dry_run": True,
        }).json()
        for _ in range(20):
            time.sleep(0.2)
            state = client.get(f"/tmf-api/intent/v5/intent/{create['id']}").json()
            if state["state"] in ("completed", "failed"):
                break
        resp = client.get(f"/tmf-api/intent/v5/intent/{create['id']}/report")
        assert resp.status_code == 200
        report = resp.json()
        assert report["intentId"] == create["id"]
        assert report["compliant"] is True
        assert report["terminal"] is True
        assert report["stageCount"] > 1

    def test_report_nonexistent_404(self, client):
        resp = client.get("/tmf-api/intent/v5/intent/nonexistent/report")
        assert resp.status_code == 404

    def test_report_active_not_compliant(self, client):
        """Active intent report has compliant=false, terminal=false."""
        with patch("llm_nephio_oran.intentd.app_v2._run_pipeline"):
            create = client.post("/tmf-api/intent/v5/intent", json={
                "expression": "test", "source": "cli",
            }).json()
        resp = client.get(f"/tmf-api/intent/v5/intent/{create['id']}/report")
        assert resp.status_code == 200
        report = resp.json()
        assert report["compliant"] is False
        assert report["terminal"] is False

    def test_report_cancelled(self, client):
        """Cancelled intent report is terminal but not compliant."""
        with patch("llm_nephio_oran.intentd.app_v2._run_pipeline"):
            create = client.post("/tmf-api/intent/v5/intent", json={
                "expression": "cancel me", "source": "web",
            }).json()
        client.delete(f"/tmf-api/intent/v5/intent/{create['id']}")
        resp = client.get(f"/tmf-api/intent/v5/intent/{create['id']}/report")
        report = resp.json()
        assert report["terminal"] is True
        assert report["compliant"] is False
        assert report["state"] == "cancelled"


class TestHealthz:
    def test_health(self, client):
        assert client.get("/healthz").json()["status"] == "ok"


class TestFrontendEndpoints:
    def test_scale_status(self, client):
        resp = client.get("/api/scale/status")
        assert resp.status_code == 200
        assert "components" in resp.json()

    def test_porch_packages(self, client):
        resp = client.get("/api/porch/packages")
        assert resp.status_code == 200
