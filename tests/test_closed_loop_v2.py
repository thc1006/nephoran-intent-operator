"""Tests for closed-loop → TMF921 integration (R4 — unified audit trail)."""
from __future__ import annotations

import time
from unittest.mock import patch, MagicMock

import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def client(tmp_path):
    from llm_nephio_oran.intentd.app_v2 import create_app
    app = create_app(store_dir=tmp_path)
    yield TestClient(app)


class TestClosedLoopThroughTMF921:
    def test_closedloop_creates_intent_via_api(self, client):
        """Closed-loop controller should POST to TMF921 API, not bypass it."""
        from llm_nephio_oran.closedloop.controller_v2 import ClosedLoopControllerV2

        ctrl = ClosedLoopControllerV2(api_base="")  # will use TestClient
        recommendation = {
            "action": "scale_out",
            "recommended_replicas": 3,
            "reason": "CPU 92% > 80%",
            "requires_human_approval": False,
        }

        with patch.object(ctrl, "_post_intent") as mock_post:
            mock_post.return_value = {"id": "intent-20260306-0001", "state": "acknowledged"}
            result = ctrl.act("oai-odu", "oran", recommendation)

        mock_post.assert_called_once()
        call_args = mock_post.call_args
        body = call_args[0][0]
        assert "Scale oai-odu to 3 replicas" in body["expression"]
        assert body["source"] == "closedloop"
        assert body["use_llm"] is False

    def test_closedloop_skips_no_action(self):
        from llm_nephio_oran.closedloop.controller_v2 import ClosedLoopControllerV2
        ctrl = ClosedLoopControllerV2(api_base="")
        result = ctrl.act("oai-odu", "oran", {
            "action": "none",
            "recommended_replicas": 2,
            "reason": "Within bounds",
            "requires_human_approval": False,
        })
        assert result is None

    def test_closedloop_intent_appears_in_list(self, client):
        """Intent created by closed-loop should be visible in TMF921 list."""
        resp = client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Scale oai-odu to 3 replicas in oran — CPU 92%",
            "source": "closedloop",
            "use_llm": False,
            "dry_run": True,
        })
        assert resp.status_code == 201
        intent_id = resp.json()["id"]

        time.sleep(1)
        items = client.get("/tmf-api/intent/v5/intent").json()
        ids = [i["id"] for i in items]
        assert intent_id in ids

        # Verify source is tracked
        intent = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
        assert intent["source"] == "closedloop"

    def test_mixed_sources_visible(self, client):
        """Web, CLI, and closedloop intents all show in unified list."""
        client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Deploy slice", "source": "web", "use_llm": False, "dry_run": True,
        })
        client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Scale AMF", "source": "cli", "use_llm": False, "dry_run": True,
        })
        client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Scale ODU", "source": "closedloop", "use_llm": False, "dry_run": True,
        })

        items = client.get("/tmf-api/intent/v5/intent").json()
        sources = {i["source"] for i in items}
        assert sources == {"web", "cli", "closedloop"}
