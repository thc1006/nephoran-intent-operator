"""Tests for IntentStore + State Machine (R1 — TMF921 async refactor)."""
from __future__ import annotations

import time
from pathlib import Path

import pytest

from llm_nephio_oran.intentd.store import IntentState, IntentStore, InvalidTransition


class TestIntentState:
    def test_valid_transitions_from_acknowledged(self):
        assert IntentState.PLANNING in IntentState.ACKNOWLEDGED.valid_next()
        assert IntentState.CANCELLED in IntentState.ACKNOWLEDGED.valid_next()
        assert IntentState.FAILED in IntentState.ACKNOWLEDGED.valid_next()

    def test_valid_transitions_from_planning(self):
        assert IntentState.VALIDATING in IntentState.PLANNING.valid_next()
        assert IntentState.FAILED in IntentState.PLANNING.valid_next()

    def test_no_backward_transitions(self):
        assert IntentState.ACKNOWLEDGED not in IntentState.PLANNING.valid_next()
        assert IntentState.PLANNING not in IntentState.GENERATING.valid_next()

    def test_completed_is_terminal(self):
        assert IntentState.COMPLETED.valid_next() == set()

    def test_failed_is_terminal(self):
        assert IntentState.FAILED.valid_next() == set()

    def test_cancelled_is_terminal(self):
        assert IntentState.CANCELLED.valid_next() == set()

    def test_all_states_have_string_values(self):
        for state in IntentState:
            assert isinstance(state.value, str)


class TestIntentStore:
    @pytest.fixture
    def store(self, tmp_path):
        return IntentStore(state_dir=tmp_path)

    def test_create_intent(self, store):
        rec = store.create(expression="Deploy eMBB slice", source="web")
        assert rec["id"].startswith("intent-")
        assert rec["state"] == "acknowledged"
        assert rec["expression"] == "Deploy eMBB slice"
        assert rec["source"] == "web"
        assert len(rec["history"]) == 1
        assert rec["history"][0]["to"] == "acknowledged"

    def test_get_intent(self, store):
        rec = store.create(expression="test", source="cli")
        got = store.get(rec["id"])
        assert got["id"] == rec["id"]
        assert got["state"] == "acknowledged"

    def test_get_nonexistent_returns_none(self, store):
        assert store.get("nonexistent") is None

    def test_list_empty(self, store):
        assert store.list_all() == []

    def test_list_returns_all(self, store):
        store.create(expression="a", source="cli")
        store.create(expression="b", source="web")
        assert len(store.list_all()) == 2

    def test_list_ordered_newest_first(self, store):
        r1 = store.create(expression="first", source="cli")
        r2 = store.create(expression="second", source="web")
        items = store.list_all()
        assert items[0]["id"] == r2["id"]
        assert items[1]["id"] == r1["id"]

    def test_valid_transition(self, store):
        rec = store.create(expression="test", source="cli")
        updated = store.transition(rec["id"], IntentState.PLANNING, data={"detail": "calling LLM"})
        assert updated["state"] == "planning"
        assert len(updated["history"]) == 2
        assert updated["history"][1]["from"] == "acknowledged"
        assert updated["history"][1]["to"] == "planning"

    def test_invalid_transition_raises(self, store):
        rec = store.create(expression="test", source="cli")
        with pytest.raises(InvalidTransition, match="acknowledged.*completed"):
            store.transition(rec["id"], IntentState.COMPLETED)

    def test_full_happy_path(self, store):
        """Walk through the entire state machine: acknowledged → completed."""
        rec = store.create(expression="Scale AMF to 5", source="web")
        iid = rec["id"]

        store.transition(iid, IntentState.PLANNING)
        store.transition(iid, IntentState.VALIDATING, data={"plan": {"intentType": "slice.scale"}})
        store.transition(iid, IntentState.GENERATING)
        store.transition(iid, IntentState.EXECUTING, data={"package_dir": "/tmp/test"})
        store.transition(iid, IntentState.PROPOSED, data={"porch": {"name": "draft-001", "lifecycle": "Proposed"}})
        store.transition(iid, IntentState.APPLIED)
        store.transition(iid, IntentState.COMPLETED)

        final = store.get(iid)
        assert final["state"] == "completed"
        assert len(final["history"]) == 8  # 1 create + 7 transitions

    def test_failure_from_any_active_state(self, store):
        """Any non-terminal state can transition to failed."""
        for start_state in [IntentState.PLANNING, IntentState.VALIDATING,
                            IntentState.GENERATING, IntentState.EXECUTING, IntentState.PROPOSED]:
            rec = store.create(expression="test", source="cli")
            store.transition(rec["id"], IntentState.PLANNING)
            if start_state != IntentState.PLANNING:
                # Walk to the target state
                path = {
                    IntentState.VALIDATING: [IntentState.VALIDATING],
                    IntentState.GENERATING: [IntentState.VALIDATING, IntentState.GENERATING],
                    IntentState.EXECUTING: [IntentState.VALIDATING, IntentState.GENERATING, IntentState.EXECUTING],
                    IntentState.PROPOSED: [IntentState.VALIDATING, IntentState.GENERATING, IntentState.EXECUTING, IntentState.PROPOSED],
                }
                for step in path[start_state]:
                    store.transition(rec["id"], step)
            store.transition(rec["id"], IntentState.FAILED, data={"error": "test failure"})
            assert store.get(rec["id"])["state"] == "failed"

    def test_transition_stores_data(self, store):
        rec = store.create(expression="test", source="cli")
        store.transition(rec["id"], IntentState.PLANNING, data={"plan": {"intentId": "x"}})
        got = store.get(rec["id"])
        assert got["plan"] == {"intentId": "x"}

    def test_transition_stores_porch(self, store):
        rec = store.create(expression="test", source="cli")
        store.transition(rec["id"], IntentState.PLANNING)
        store.transition(rec["id"], IntentState.VALIDATING)
        store.transition(rec["id"], IntentState.GENERATING)
        store.transition(rec["id"], IntentState.EXECUTING)
        store.transition(rec["id"], IntentState.PROPOSED, data={
            "porch": {"name": "draft-001", "lifecycle": "Proposed", "files": 3}
        })
        got = store.get(rec["id"])
        assert got["porch"]["name"] == "draft-001"

    def test_transition_nonexistent_raises(self, store):
        with pytest.raises(KeyError):
            store.transition("nonexistent", IntentState.PLANNING)

    def test_persistence_across_reload(self, tmp_path):
        """Store should recover state from jsonl on reload."""
        store1 = IntentStore(state_dir=tmp_path)
        rec = store1.create(expression="persist test", source="cli")
        store1.transition(rec["id"], IntentState.PLANNING)

        store2 = IntentStore(state_dir=tmp_path)
        got = store2.get(rec["id"])
        assert got is not None
        assert got["state"] == "planning"
        assert got["expression"] == "persist test"

    def test_no_id_collision_after_reload(self, tmp_path):
        """C1 fix: new IDs after reload must not collide with existing ones."""
        store1 = IntentStore(state_dir=tmp_path)
        ids_before = set()
        for i in range(5):
            rec = store1.create(expression=f"pre-{i}", source="cli")
            ids_before.add(rec["id"])

        # Simulate restart
        store2 = IntentStore(state_dir=tmp_path)
        ids_after = set()
        for i in range(5):
            rec = store2.create(expression=f"post-{i}", source="cli")
            ids_after.add(rec["id"])

        # No overlap
        assert ids_before.isdisjoint(ids_after), f"Collision: {ids_before & ids_after}"
        # All 10 intents exist
        assert len(store2.list_all()) == 10

    def test_deep_copy_isolation(self, store):
        """M2 fix: returned dicts must not share references with store internals."""
        rec = store.create(expression="test", source="cli")
        got = store.get(rec["id"])
        got["history"].append({"injected": True})
        got["expression"] = "MUTATED"

        original = store.get(rec["id"])
        assert len(original["history"]) == 1  # not 2
        assert original["expression"] == "test"  # not MUTATED

    def test_cancel_from_acknowledged(self, store):
        """T11: cancel() works from acknowledged state."""
        rec = store.create(expression="test cancel", source="web")
        cancelled = store.cancel(rec["id"])
        assert cancelled["state"] == "cancelled"
        assert len(cancelled["history"]) == 2
        assert cancelled["history"][1]["to"] == "cancelled"

    def test_cancel_from_non_acknowledged_raises(self, store):
        """T11: cancel() raises InvalidTransition from non-acknowledged state."""
        rec = store.create(expression="test", source="cli")
        store.transition(rec["id"], IntentState.PLANNING)
        with pytest.raises(InvalidTransition):
            store.cancel(rec["id"])

    def test_build_report_completed(self, store):
        """T11: build_report() for a completed intent."""
        rec = store.create(expression="deploy slice", source="web")
        iid = rec["id"]
        store.transition(iid, IntentState.PLANNING)
        store.transition(iid, IntentState.VALIDATING, data={"plan": {"intentType": "slice.deploy"}})
        store.transition(iid, IntentState.GENERATING)
        store.transition(iid, IntentState.EXECUTING)
        store.transition(iid, IntentState.PROPOSED)
        store.transition(iid, IntentState.APPLIED)
        store.transition(iid, IntentState.COMPLETED)

        report = store.build_report(iid)
        assert report["intentId"] == iid
        assert report["state"] == "completed"
        assert report["compliant"] is True
        assert report["terminal"] is True
        assert report["stageCount"] == 8
        assert report["expression"] == "deploy slice"

    def test_build_report_failed_not_compliant(self, store):
        """T11: failed intent is not compliant."""
        rec = store.create(expression="test", source="cli")
        store.transition(rec["id"], IntentState.PLANNING)
        store.transition(rec["id"], IntentState.FAILED, data={"errors": ["boom"]})

        report = store.build_report(rec["id"])
        assert report["compliant"] is False
        assert report["terminal"] is True
        assert report["errors"] == ["boom"]

    def test_build_report_nonexistent_returns_none(self, store):
        """T11: build_report for nonexistent ID returns None."""
        assert store.build_report("nonexistent") is None

    def test_build_report_active_not_terminal(self, store):
        """T11: active intent has terminal=False."""
        rec = store.create(expression="test", source="cli")
        store.transition(rec["id"], IntentState.PLANNING)

        report = store.build_report(rec["id"])
        assert report["terminal"] is False
        assert report["compliant"] is False
