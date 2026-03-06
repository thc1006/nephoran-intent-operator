"""R7: End-to-end integration test — full pipeline from NL text to completed intent.

Tests the entire chain: create → plan → validate → generate → execute → complete,
verifying each state transition, data propagation, and TMF921 compliance.
"""
from __future__ import annotations

import time
from pathlib import Path
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def client(tmp_path):
    """TestClient with isolated store."""
    from llm_nephio_oran.intentd.app_v2 import create_app
    app = create_app(store_dir=tmp_path)
    yield TestClient(app)


class TestE2EPipelineDryRun:
    """Full pipeline with dry_run=True (skips git+porch, but runs plan→validate→generate)."""

    def test_full_lifecycle_dry_run(self, client):
        """Create → plan → validate → generate → complete (dry_run)."""
        # 1. Create
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Deploy eMBB slice at edge01 with 3 ODU replicas",
            "source": "web",
            "use_llm": False,
            "dry_run": True,
        })
        assert resp.status_code == 201
        intent_id = resp.json()["id"]

        # 2. Poll until terminal
        final = None
        for _ in range(30):
            time.sleep(0.2)
            final = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            if final["state"] in ("completed", "failed"):
                break

        assert final is not None
        assert final["state"] == "completed", f"Expected completed, got {final['state']}: {final.get('errors')}"

        # 3. Verify plan was generated
        assert final["plan"] is not None
        assert final["plan"]["intentId"] is not None
        assert final["plan"]["intentType"] in ("slice.deploy", "slice.scale")
        assert len(final["plan"]["actions"]) >= 1

        # 4. Verify state machine history
        states = [h["to"] for h in final["history"]]
        assert states[0] == "acknowledged"
        assert "planning" in states
        assert "validating" in states
        assert "generating" in states
        assert "completed" in states

        # 5. Verify via IntentReport
        report = client.get(f"/tmf-api/intent/v5/intent/{intent_id}/report").json()
        assert report["compliant"] is True
        assert report["terminal"] is True
        assert report["stageCount"] == len(final["history"])
        assert report["intentId"] == intent_id

        # 6. Verify in list
        all_intents = client.get("/tmf-api/intent/v5/intent").json()
        found = [i for i in all_intents if i["id"] == intent_id]
        assert len(found) == 1
        assert found[0]["state"] == "completed"

    def test_scale_intent_dry_run(self, client):
        """Scale intent produces valid plan with replicas."""
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Scale AMF to 5 replicas",
            "source": "cli",
            "use_llm": False,
            "dry_run": True,
        })
        intent_id = resp.json()["id"]

        for _ in range(30):
            time.sleep(0.2)
            final = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            if final["state"] in ("completed", "failed"):
                break

        assert final["state"] == "completed"
        plan = final["plan"]
        assert plan["intentType"] in ("slice.deploy", "slice.scale")
        # Stub planner produces actions with replicas for scale intents
        assert any(a.get("replicas") is not None for a in plan["actions"])


class TestE2EFailure:
    """Pipeline failure scenarios."""

    def test_planner_failure_records_error(self, client):
        """If planner throws, intent moves to failed with error details."""
        with patch(
            "llm_nephio_oran.planner.llm_planner.plan_from_text",
            side_effect=RuntimeError("LLM connection refused"),
        ):
            resp = client.post("/tmf-api/intent/v5/intent", json={
                "expression": "fail me", "source": "web",
            })
            intent_id = resp.json()["id"]
            time.sleep(0.5)

            final = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            assert final["state"] == "failed"
            assert final["errors"] is not None
            assert any("LLM connection refused" in e for e in final["errors"])

            # Report should reflect failure
            report = client.get(f"/tmf-api/intent/v5/intent/{intent_id}/report").json()
            assert report["compliant"] is False
            assert report["terminal"] is True


class TestE2ECancelFlow:
    """Cancel flow: create → cancel (before pipeline runs)."""

    def test_create_then_cancel(self, client):
        """Create acknowledged intent, cancel before pipeline progresses."""
        with patch("llm_nephio_oran.intentd.app_v2._run_pipeline"):
            resp = client.post("/tmf-api/intent/v5/intent", json={
                "expression": "I will be cancelled", "source": "web",
            })
            intent_id = resp.json()["id"]

        # Still acknowledged (pipeline was suppressed)
        assert client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()["state"] == "acknowledged"

        # Cancel
        del_resp = client.delete(f"/tmf-api/intent/v5/intent/{intent_id}")
        assert del_resp.status_code == 200
        assert del_resp.json()["state"] == "cancelled"

        # Verify report
        report = client.get(f"/tmf-api/intent/v5/intent/{intent_id}/report").json()
        assert report["terminal"] is True
        assert report["compliant"] is False
        assert report["state"] == "cancelled"

        # Cannot cancel again
        resp2 = client.delete(f"/tmf-api/intent/v5/intent/{intent_id}")
        assert resp2.status_code == 409


class TestE2EMultipleIntents:
    """Multiple intents running concurrently."""

    def test_multiple_concurrent_intents(self, client):
        """Create multiple intents; all should complete independently."""
        intent_ids = []
        for i in range(3):
            resp = client.post("/tmf-api/intent/v5/intent", json={
                "expression": f"Deploy component {i}",
                "source": "web",
                "use_llm": False,
                "dry_run": True,
            })
            assert resp.status_code == 201
            intent_ids.append(resp.json()["id"])

        # All IDs should be unique
        assert len(set(intent_ids)) == 3

        # Wait for all to complete
        for _ in range(40):
            time.sleep(0.2)
            states = [
                client.get(f"/tmf-api/intent/v5/intent/{iid}").json()["state"]
                for iid in intent_ids
            ]
            if all(s in ("completed", "failed") for s in states):
                break

        # Verify all completed
        for iid in intent_ids:
            final = client.get(f"/tmf-api/intent/v5/intent/{iid}").json()
            assert final["state"] == "completed", f"{iid} ended in {final['state']}"

        # List should have all 3
        all_intents = client.get("/tmf-api/intent/v5/intent").json()
        assert len(all_intents) == 3


class TestE2EPersistence:
    """Store persistence across restarts."""

    def test_store_survives_restart(self, tmp_path):
        """Intents persist across app restarts."""
        from llm_nephio_oran.intentd.app_v2 import create_app

        # First app instance
        app1 = create_app(store_dir=tmp_path)
        c1 = TestClient(app1)
        with patch("llm_nephio_oran.intentd.app_v2._run_pipeline"):
            resp = c1.post("/tmf-api/intent/v5/intent", json={
                "expression": "persist me", "source": "cli",
            })
        intent_id = resp.json()["id"]

        # Simulate restart — new app instance, same directory
        app2 = create_app(store_dir=tmp_path)
        c2 = TestClient(app2)

        got = c2.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
        assert got["id"] == intent_id
        assert got["expression"] == "persist me"

        # New IDs should not collide
        with patch("llm_nephio_oran.intentd.app_v2._run_pipeline"):
            resp2 = c2.post("/tmf-api/intent/v5/intent", json={
                "expression": "after restart", "source": "cli",
            })
        assert resp2.json()["id"] != intent_id
