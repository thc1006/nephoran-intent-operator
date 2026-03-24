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


class TestD3CorruptionRecovery:
    """D3: Store must survive corrupt WAL lines and support snapshots."""

    def test_corrupt_wal_line_skipped(self, tmp_path):
        """A corrupt line in the WAL should be skipped, not crash the store."""
        state_dir = tmp_path / ".state"
        store = IntentStore(state_dir=state_dir)

        # Create two intents
        r1 = store.create(expression="first", source="cli")
        r2 = store.create(expression="second", source="cli")

        # Manually inject a corrupt line into the WAL
        wal_path = state_dir / "intents.jsonl"
        with wal_path.open("a") as f:
            f.write("THIS IS NOT VALID JSON\n")

        # Create a third intent after corruption
        r3 = store.create(expression="third", source="cli")

        # Reload store from WAL — should recover all 3, skip corrupt line
        store2 = IntentStore(state_dir=state_dir)
        assert store2.get(r1["id"]) is not None
        assert store2.get(r2["id"]) is not None
        assert store2.get(r3["id"]) is not None

    def test_corrupt_wal_only_corrupt_lines_lost(self, tmp_path):
        """Only truly corrupt lines are lost; entries before and after survive."""
        state_dir = tmp_path / ".state"
        wal_path = state_dir / "intents.jsonl"
        state_dir.mkdir(parents=True)

        import json
        # Write valid → corrupt → valid manually
        with wal_path.open("w") as f:
            f.write(json.dumps({"op": "create", "record": {
                "id": "intent-20260306-0001", "state": "acknowledged",
                "expression": "before", "source": "cli",
                "plan": None, "report": None, "porch": None, "git": None,
                "configsync": None, "errors": None,
                "created_at": "2026-03-06T00:00:00Z", "updated_at": "2026-03-06T00:00:00Z",
                "history": [{"to": "acknowledged", "at": "2026-03-06T00:00:00Z", "data": None}],
            }}) + "\n")
            f.write("CORRUPT LINE HERE\n")
            f.write(json.dumps({"op": "create", "record": {
                "id": "intent-20260306-0002", "state": "acknowledged",
                "expression": "after", "source": "web",
                "plan": None, "report": None, "porch": None, "git": None,
                "configsync": None, "errors": None,
                "created_at": "2026-03-06T00:01:00Z", "updated_at": "2026-03-06T00:01:00Z",
                "history": [{"to": "acknowledged", "at": "2026-03-06T00:01:00Z", "data": None}],
            }}) + "\n")

        store = IntentStore(state_dir=state_dir)
        assert store.get("intent-20260306-0001")["expression"] == "before"
        assert store.get("intent-20260306-0002")["expression"] == "after"

    def test_snapshot_written_and_loaded(self, tmp_path):
        """Snapshot should be written periodically and speed up reload."""
        from llm_nephio_oran.intentd.store import SNAPSHOT_INTERVAL
        state_dir = tmp_path / ".state"
        store = IntentStore(state_dir=state_dir)

        # Create an intent and push it through enough transitions to trigger snapshot
        rec = store.create(expression="snapshot test", source="cli")
        iid = rec["id"]

        # Drive transitions: we need SNAPSHOT_INTERVAL transitions total
        # Pipeline: planning→validating→generating→executing→proposed→applied→completed = 7
        transitions_done = 0
        for _ in range(SNAPSHOT_INTERVAL // 7 + 1):
            r = store.create(expression="filler", source="cli")
            fid = r["id"]
            for state in [IntentState.PLANNING, IntentState.VALIDATING,
                          IntentState.GENERATING, IntentState.EXECUTING,
                          IntentState.PROPOSED, IntentState.APPLIED,
                          IntentState.COMPLETED]:
                store.transition(fid, state)
                transitions_done += 1
                if transitions_done >= SNAPSHOT_INTERVAL:
                    break
            if transitions_done >= SNAPSHOT_INTERVAL:
                break

        # Snapshot file should exist
        snapshot_path = state_dir / "intents.snapshot.json"
        assert snapshot_path.exists(), "Snapshot was not written"

        # Load from snapshot — should recover all intents
        store2 = IntentStore(state_dir=state_dir)
        assert store2.get(iid) is not None
        assert store2.get(iid)["expression"] == "snapshot test"

    def test_post_snapshot_transitions_not_lost(self, tmp_path):
        """Critical: transitions AFTER a snapshot must survive reload."""
        from llm_nephio_oran.intentd.store import SNAPSHOT_INTERVAL
        state_dir = tmp_path / ".state"
        store = IntentStore(state_dir=state_dir)

        # Drive enough transitions to trigger a snapshot
        for _ in range(SNAPSHOT_INTERVAL // 7 + 1):
            r = store.create(expression="filler", source="cli")
            for state in [IntentState.PLANNING, IntentState.VALIDATING,
                          IntentState.GENERATING, IntentState.EXECUTING,
                          IntentState.PROPOSED, IntentState.APPLIED,
                          IntentState.COMPLETED]:
                store.transition(r["id"], state)

        # Snapshot should exist now
        assert (state_dir / "intents.snapshot.json").exists()

        # Create a NEW intent and transition it AFTER the snapshot
        post = store.create(expression="post-snapshot", source="web")
        post_id = post["id"]
        store.transition(post_id, IntentState.PLANNING)
        store.transition(post_id, IntentState.VALIDATING)

        # Also transition an EXISTING intent (one that's in the snapshot as completed)
        # — we can't transition completed, so create another and transition it past snapshot
        extra = store.create(expression="extra-post", source="cli")
        store.transition(extra["id"], IntentState.PLANNING)

        # Reload — post-snapshot changes must survive
        store2 = IntentStore(state_dir=state_dir)
        post_rec = store2.get(post_id)
        assert post_rec is not None, "Post-snapshot intent lost!"
        assert post_rec["state"] == "validating", f"Expected validating, got {post_rec['state']}"
        assert post_rec["expression"] == "post-snapshot"

        extra_rec = store2.get(extra["id"])
        assert extra_rec is not None, "Post-snapshot extra intent lost!"
        assert extra_rec["state"] == "planning"

    def test_empty_wal_and_no_snapshot(self, tmp_path):
        """Store should work fine with no WAL and no snapshot."""
        state_dir = tmp_path / ".state"
        store = IntentStore(state_dir=state_dir)
        assert store.list_all() == []

    def test_corrupt_snapshot_falls_back_to_wal(self, tmp_path):
        """If snapshot is corrupt, fall back to WAL replay."""
        state_dir = tmp_path / ".state"
        store = IntentStore(state_dir=state_dir)
        rec = store.create(expression="wal only", source="cli")

        # Write a corrupt snapshot
        snapshot_path = state_dir / "intents.snapshot.json"
        snapshot_path.write_text("NOT VALID JSON")

        # Reload — should fall back to WAL
        store2 = IntentStore(state_dir=state_dir)
        assert store2.get(rec["id"]) is not None
        assert store2.get(rec["id"])["expression"] == "wal only"


class TestD3bWalCompaction:
    """D3b: WAL should be compacted after snapshot."""

    def test_wal_truncated_after_snapshot(self, tmp_path):
        """After snapshot, WAL should be significantly smaller (compacted)."""
        from llm_nephio_oran.intentd.store import SNAPSHOT_INTERVAL
        state_dir = tmp_path / ".state"
        store = IntentStore(state_dir=state_dir)

        total_ops = 0
        # Drive enough transitions to trigger snapshot + compaction
        for _ in range(SNAPSHOT_INTERVAL // 7 + 1):
            r = store.create(expression="filler", source="cli")
            total_ops += 1
            for state in [IntentState.PLANNING, IntentState.VALIDATING,
                          IntentState.GENERATING, IntentState.EXECUTING,
                          IntentState.PROPOSED, IntentState.APPLIED,
                          IntentState.COMPLETED]:
                store.transition(r["id"], state)
                total_ops += 1

        snapshot_path = state_dir / "intents.snapshot.json"
        wal_path = state_dir / "intents.jsonl"
        assert snapshot_path.exists()
        # WAL should have far fewer lines than total ops (only post-compaction entries)
        wal_lines = len([l for l in wal_path.read_text().splitlines() if l.strip()])
        assert wal_lines < total_ops, f"WAL not compacted: {wal_lines} lines vs {total_ops} total ops"
        # Should be under SNAPSHOT_INTERVAL (only entries after the last snapshot)
        assert wal_lines < SNAPSHOT_INTERVAL

    def test_new_entries_after_compaction_survive_reload(self, tmp_path):
        """Entries written after compaction should survive reload."""
        from llm_nephio_oran.intentd.store import SNAPSHOT_INTERVAL
        state_dir = tmp_path / ".state"
        store = IntentStore(state_dir=state_dir)

        # Trigger snapshot + compaction
        for _ in range(SNAPSHOT_INTERVAL // 7 + 1):
            r = store.create(expression="filler", source="cli")
            for state in [IntentState.PLANNING, IntentState.VALIDATING,
                          IntentState.GENERATING, IntentState.EXECUTING,
                          IntentState.PROPOSED, IntentState.APPLIED,
                          IntentState.COMPLETED]:
                store.transition(r["id"], state)

        # Now create a new intent AFTER compaction
        post = store.create(expression="after-compaction", source="web")
        store.transition(post["id"], IntentState.PLANNING)

        # Reload — should have both snapshot intents and post-compaction intent
        store2 = IntentStore(state_dir=state_dir)
        assert store2.get(post["id"]) is not None
        assert store2.get(post["id"])["state"] == "planning"
        assert store2.get(post["id"])["expression"] == "after-compaction"

    def test_compaction_preserves_all_intents(self, tmp_path):
        """Total intent count should be preserved across compaction + reload."""
        from llm_nephio_oran.intentd.store import SNAPSHOT_INTERVAL
        state_dir = tmp_path / ".state"
        store = IntentStore(state_dir=state_dir)

        created_ids = set()
        for _ in range(SNAPSHOT_INTERVAL // 7 + 1):
            r = store.create(expression="filler", source="cli")
            created_ids.add(r["id"])
            for state in [IntentState.PLANNING, IntentState.VALIDATING,
                          IntentState.GENERATING, IntentState.EXECUTING,
                          IntentState.PROPOSED, IntentState.APPLIED,
                          IntentState.COMPLETED]:
                store.transition(r["id"], state)

        # Add a few more after compaction
        for i in range(3):
            r = store.create(expression=f"post-{i}", source="cli")
            created_ids.add(r["id"])

        store2 = IntentStore(state_dir=state_dir)
        assert len(store2.list_all()) == len(created_ids)


class TestD2bComplianceState:
    """D2b: TMF921 ComplianceState in IntentReport."""

    @pytest.fixture
    def store(self, tmp_path):
        return IntentStore(state_dir=tmp_path)

    def test_acknowledged_is_pending(self, store):
        rec = store.create(expression="test", source="cli")
        report = store.build_report(rec["id"])
        assert report["lifecycleStatus"] == "Acknowledged"
        assert report["complianceState"] == "pending"

    def test_planning_is_active_pending(self, store):
        rec = store.create(expression="test", source="cli")
        store.transition(rec["id"], IntentState.PLANNING)
        report = store.build_report(rec["id"])
        assert report["lifecycleStatus"] == "Active"
        assert report["complianceState"] == "pending"

    def test_completed_is_compliant(self, store):
        rec = store.create(expression="test", source="cli")
        iid = rec["id"]
        for state in [IntentState.PLANNING, IntentState.VALIDATING,
                      IntentState.GENERATING, IntentState.EXECUTING,
                      IntentState.PROPOSED, IntentState.APPLIED,
                      IntentState.COMPLETED]:
            store.transition(iid, state)
        report = store.build_report(iid)
        assert report["lifecycleStatus"] == "Active"
        assert report["complianceState"] == "compliant"

    def test_failed_is_non_compliant(self, store):
        rec = store.create(expression="test", source="cli")
        store.transition(rec["id"], IntentState.PLANNING)
        store.transition(rec["id"], IntentState.FAILED, data={"errors": ["boom"]})
        report = store.build_report(rec["id"])
        assert report["complianceState"] == "nonCompliant"

    def test_cancelled_is_non_compliant(self, store):
        rec = store.create(expression="test", source="cli")
        store.cancel(rec["id"])
        report = store.build_report(rec["id"])
        assert report["lifecycleStatus"] == "Cancelled"
        assert report["complianceState"] == "nonCompliant"

    def test_porch_error_degrades_compliance(self, store):
        """If completed but porch had error, compliance is degraded."""
        rec = store.create(expression="test", source="cli")
        iid = rec["id"]
        for state in [IntentState.PLANNING, IntentState.VALIDATING,
                      IntentState.GENERATING, IntentState.EXECUTING]:
            store.transition(iid, state)
        store.transition(iid, IntentState.PROPOSED, data={
            "porch": {"error": "Unauthorized"},
        })
        store.transition(iid, IntentState.APPLIED)
        store.transition(iid, IntentState.COMPLETED)
        report = store.build_report(iid)
        assert report["complianceState"] == "degraded"

    def test_configsync_not_delivered_degrades(self, store):
        """If completed but configsync not delivered, compliance is degraded."""
        rec = store.create(expression="test", source="cli")
        iid = rec["id"]
        for state in [IntentState.PLANNING, IntentState.VALIDATING,
                      IntentState.GENERATING, IntentState.EXECUTING,
                      IntentState.PROPOSED]:
            store.transition(iid, state)
        store.transition(iid, IntentState.APPLIED, data={
            "configsync": {"delivered": False, "commit_match": False},
        })
        store.transition(iid, IntentState.COMPLETED)
        report = store.build_report(iid)
        assert report["complianceState"] == "degraded"

    def test_clean_completion_is_fully_compliant(self, store):
        """Clean completion with no porch/configsync errors is compliant."""
        rec = store.create(expression="test", source="cli")
        iid = rec["id"]
        for state in [IntentState.PLANNING, IntentState.VALIDATING,
                      IntentState.GENERATING, IntentState.EXECUTING]:
            store.transition(iid, state)
        store.transition(iid, IntentState.PROPOSED, data={
            "porch": {"name": "draft-001", "lifecycle": "Proposed"},
        })
        store.transition(iid, IntentState.APPLIED, data={
            "configsync": {"delivered": True, "commit_match": True},
        })
        store.transition(iid, IntentState.COMPLETED)
        report = store.build_report(iid)
        assert report["complianceState"] == "compliant"
