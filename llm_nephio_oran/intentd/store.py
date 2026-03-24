"""Intent Store with TMF921-aligned state machine.

Single source of truth for all intents. Backed by in-memory dict + append-only
jsonl for persistence. Every state transition is recorded in history[] for audit.

State machine:
  acknowledged → planning → validating → generating → executing
    → proposed → applied → completed
  Any non-terminal state → failed
  acknowledged → cancelled
"""
from __future__ import annotations

import copy
import json
import logging
import threading
from datetime import datetime, timezone
from enum import Enum
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

# ── State Machine ──────────────────────────────────────────────────────────

_FORWARD_TRANSITIONS: dict[str, set[str]] = {
    "acknowledged": {"planning", "cancelled", "failed"},
    "planning":     {"validating", "failed"},
    "validating":   {"generating", "failed"},
    "generating":   {"executing", "failed"},
    "executing":    {"proposed", "failed"},
    "proposed":     {"applied", "failed"},
    "applied":      {"completed", "failed"},
    # terminal states
    "completed":    set(),
    "failed":       set(),
    "cancelled":    set(),
}


class IntentState(Enum):
    ACKNOWLEDGED = "acknowledged"
    PLANNING     = "planning"
    VALIDATING   = "validating"
    GENERATING   = "generating"
    EXECUTING    = "executing"
    PROPOSED     = "proposed"
    APPLIED      = "applied"
    COMPLETED    = "completed"
    FAILED       = "failed"
    CANCELLED    = "cancelled"

    def valid_next(self) -> set[IntentState]:
        names = _FORWARD_TRANSITIONS.get(self.value, set())
        return {IntentState(n) for n in names}


class ComplianceState(Enum):
    """TMF921 TR292B compliance states for IntentReport.

    Maps internal pipeline states to TMF921 compliance semantics:
      PENDING       — Intent is in-flight (acknowledged through executing)
      COMPLIANT     — ConfigSync synced, workload healthy (completed)
      DEGRADED      — Porch/ConfigSync partial failure but intent applied
      NON_COMPLIANT — Pipeline failed or cancelled
    """
    PENDING = "pending"
    COMPLIANT = "compliant"
    DEGRADED = "degraded"
    NON_COMPLIANT = "nonCompliant"


# TMF921 lifecycle mapping: internal state → (lifecycleStatus, complianceState)
_TMF921_MAP: dict[str, tuple[str, str]] = {
    "acknowledged": ("Acknowledged", ComplianceState.PENDING.value),
    "planning":     ("Active", ComplianceState.PENDING.value),
    "validating":   ("Active", ComplianceState.PENDING.value),
    "generating":   ("Active", ComplianceState.PENDING.value),
    "executing":    ("Active", ComplianceState.PENDING.value),
    "proposed":     ("Active", ComplianceState.PENDING.value),
    "applied":      ("Active", ComplianceState.COMPLIANT.value),
    "completed":    ("Active", ComplianceState.COMPLIANT.value),
    "failed":       ("Active", ComplianceState.NON_COMPLIANT.value),
    "cancelled":    ("Cancelled", ComplianceState.NON_COMPLIANT.value),
}


class InvalidTransition(Exception):
    pass


# ── Sequence Counter ───────────────────────────────────────────────────────

_seq_lock = threading.Lock()
_seq_counter = 0


def _init_seq_from_existing(intent_ids: set[str]) -> None:
    """Set _seq_counter above the max existing sequence to prevent ID collision."""
    global _seq_counter
    max_seq = 0
    for iid in intent_ids:
        # intent-YYYYMMDD-NNNN
        parts = iid.rsplit("-", 1)
        if len(parts) == 2:
            try:
                max_seq = max(max_seq, int(parts[1]))
            except ValueError:
                pass
    with _seq_lock:
        _seq_counter = max(max_seq, _seq_counter)


def _next_intent_id() -> str:
    global _seq_counter
    with _seq_lock:
        _seq_counter += 1
        today = datetime.now(timezone.utc).strftime("%Y%m%d")
        seq = _seq_counter
    return f"intent-{today}-{seq:04d}"


# ── Store ──────────────────────────────────────────────────────────────────

SNAPSHOT_INTERVAL = 50  # Take a snapshot every N transitions


class IntentStore:
    """Thread-safe intent store with state machine enforcement.

    Persistence: jsonl WAL + periodic snapshots.
    - WAL (intents.jsonl): append-only, replayed on startup
    - Snapshot (intents.snapshot.json): full state, written every SNAPSHOT_INTERVAL transitions
    - On load: read snapshot (if newer) → replay WAL entries after snapshot
    - Corrupt WAL lines are skipped with a warning (crash-safe)
    """

    def __init__(self, state_dir: Path):
        self._dir = state_dir
        self._dir.mkdir(parents=True, exist_ok=True)
        self._log_path = self._dir / "intents.jsonl"
        self._snapshot_path = self._dir / "intents.snapshot.json"
        self._lock = threading.Lock()
        self._intents: dict[str, dict[str, Any]] = {}
        self._transition_count = 0
        self._snapshot_seq = 0  # sequence at last snapshot
        self._wal_lines_at_snapshot = 0  # WAL line count when snapshot was taken
        self._wal_line_count = 0  # current WAL line count
        self._load()

    # ── Public API ─────────────────────────────────────────────────────

    def create(self, expression: str, source: str) -> dict[str, Any]:
        """Create a new intent in ACKNOWLEDGED state."""
        now = datetime.now(timezone.utc).replace(microsecond=0).isoformat() + "Z"
        intent_id = _next_intent_id()

        rec: dict[str, Any] = {
            "id": intent_id,
            "state": IntentState.ACKNOWLEDGED.value,
            "expression": expression,
            "source": source,
            "plan": None,
            "report": None,
            "porch": None,
            "git": None,
            "configsync": None,
            "reposync": None,
            "errors": None,
            "created_at": now,
            "updated_at": now,
            "history": [
                {"to": IntentState.ACKNOWLEDGED.value, "at": now, "data": None},
            ],
        }
        with self._lock:
            self._intents[intent_id] = rec
            self._append_log({"op": "create", "record": rec})
        return copy.deepcopy(rec)

    def get(self, intent_id: str) -> dict[str, Any] | None:
        """Get intent by ID. Returns deep copy (safe to mutate)."""
        with self._lock:
            rec = self._intents.get(intent_id)
            return copy.deepcopy(rec) if rec else None

    def list_all(self) -> list[dict[str, Any]]:
        """List all intents, newest first (by insertion order)."""
        with self._lock:
            items = list(self._intents.values())
        return [copy.deepcopy(r) for r in reversed(items)]

    def transition(
        self,
        intent_id: str,
        new_state: IntentState,
        data: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """Transition intent to a new state. Raises InvalidTransition or KeyError."""
        with self._lock:
            rec = self._intents.get(intent_id)
            if rec is None:
                raise KeyError(f"Intent {intent_id} not found")

            current = IntentState(rec["state"])
            if new_state not in current.valid_next():
                raise InvalidTransition(
                    f"Cannot transition from {current.value} to {new_state.value} "
                    f"(valid: {', '.join(s.value for s in current.valid_next())})"
                )

            now = datetime.now(timezone.utc).replace(microsecond=0).isoformat() + "Z"
            rec["state"] = new_state.value
            rec["updated_at"] = now
            rec["history"].append({
                "from": current.value,
                "to": new_state.value,
                "at": now,
                "data": data,
            })

            # Merge well-known data fields
            if data:
                for key in ("plan", "report", "porch", "git", "configsync", "reposync", "errors", "package_dir"):
                    if key in data:
                        rec[key] = data[key]

            self._append_log({
                "op": "transition",
                "id": intent_id,
                "from": current.value,
                "to": new_state.value,
                "data": data,
                "at": now,
            })

            self._transition_count += 1
            if self._transition_count % SNAPSHOT_INTERVAL == 0:
                self._write_snapshot()

            return copy.deepcopy(rec)

    def cancel(self, intent_id: str) -> dict[str, Any]:
        """Cancel an intent. Only valid from ACKNOWLEDGED state."""
        return self.transition(intent_id, IntentState.CANCELLED,
                               data={"reason": "cancelled_by_user"})

    def build_report(self, intent_id: str) -> dict[str, Any] | None:
        """Build a TMF921 IntentReport from the intent record.

        Includes TR292B compliance state and lifecycle mapping:
          - lifecycleStatus: Acknowledged | Active | Cancelled
          - complianceState: pending | compliant | degraded | nonCompliant
        """
        rec = self.get(intent_id)
        if rec is None:
            return None
        terminal = rec["state"] in ("completed", "failed", "cancelled")
        lifecycle, compliance = _TMF921_MAP.get(
            rec["state"], ("Active", ComplianceState.PENDING.value)
        )

        # Detect degraded: applied/completed but porch or configsync had errors
        if compliance == ComplianceState.COMPLIANT.value:
            porch = rec.get("porch") or {}
            configsync = rec.get("configsync") or {}
            if porch.get("error") or configsync.get("error") or not configsync.get("delivered", True):
                compliance = ComplianceState.DEGRADED.value

        return {
            "intentId": rec["id"],
            "state": rec["state"],
            "expression": rec["expression"],
            "source": rec["source"],
            "compliant": rec["state"] == "completed",
            "terminal": terminal,
            "lifecycleStatus": lifecycle,
            "complianceState": compliance,
            "stageCount": len(rec["history"]),
            "createdAt": rec["created_at"],
            "updatedAt": rec["updated_at"],
            "plan": rec["plan"],
            "porch": rec["porch"],
            "git": rec["git"],
            "configsync": rec.get("configsync"),
            "reposync": rec.get("reposync"),
            "errors": rec["errors"],
            "report": rec["report"],
            "history": rec["history"],
        }

    # ── Persistence ────────────────────────────────────────────────────

    def _append_log(self, entry: dict[str, Any]) -> None:
        with self._log_path.open("a", encoding="utf-8") as f:
            f.write(json.dumps(entry, ensure_ascii=False, default=str) + "\n")
        self._wal_line_count += 1

    def _write_snapshot(self) -> None:
        """Write full state to snapshot file (atomic via tmp+rename on POSIX)."""
        with _seq_lock:
            seq = _seq_counter
        snapshot = {
            "version": 1,
            "seq": seq,
            "transition_count": self._transition_count,
            "wal_line_count": self._wal_line_count,
            "intents": self._intents,
        }
        tmp = self._snapshot_path.with_suffix(".tmp")
        try:
            with tmp.open("w", encoding="utf-8") as f:
                json.dump(snapshot, f, ensure_ascii=False, default=str)
            tmp.rename(self._snapshot_path)  # atomic on POSIX
            self._snapshot_seq = seq
            self._wal_lines_at_snapshot = self._wal_line_count
            logger.info("Snapshot written: %d intents, seq=%d", len(self._intents), seq)
            self._compact_wal()
        except Exception:
            logger.exception("Failed to write snapshot")
            if tmp.exists():
                tmp.unlink()

    def _compact_wal(self) -> None:
        """Truncate WAL after snapshot — keep only empty file for new appends.

        Safe because _write_snapshot() already persisted full state atomically.
        After truncating, we update the snapshot's wal_line_count to 0 so that
        on reload, _replay_wal() correctly replays any new WAL entries written
        after compaction instead of skipping them.
        """
        try:
            self._log_path.write_text("")
            self._wal_line_count = 0
            self._wal_lines_at_snapshot = 0
            # Update snapshot to reflect compacted state
            self._update_snapshot_wal_offset()
            logger.info("WAL compacted (truncated after snapshot)")
        except Exception:
            logger.exception("WAL compaction failed (non-fatal)")

    def _update_snapshot_wal_offset(self) -> None:
        """Update snapshot's wal_line_count to 0 after compaction."""
        if not self._snapshot_path.exists():
            return
        try:
            with self._snapshot_path.open("r", encoding="utf-8") as f:
                snapshot = json.load(f)
            snapshot["wal_line_count"] = 0
            tmp = self._snapshot_path.with_suffix(".tmp")
            with tmp.open("w", encoding="utf-8") as f:
                json.dump(snapshot, f, ensure_ascii=False, default=str)
            tmp.rename(self._snapshot_path)
        except Exception:
            logger.exception("Failed to update snapshot WAL offset (non-fatal)")

    def _load(self) -> None:
        """Load state: snapshot (if available) → replay WAL entries after snapshot."""
        snapshot_loaded = self._load_snapshot()
        self._replay_wal(skip_before_snapshot=snapshot_loaded)

        # Ensure seq counter starts above any existing IDs to prevent collision
        if self._intents:
            _init_seq_from_existing(set(self._intents.keys()))

    def _load_snapshot(self) -> bool:
        """Load snapshot if it exists. Returns True if loaded."""
        if not self._snapshot_path.exists():
            return False
        try:
            with self._snapshot_path.open("r", encoding="utf-8") as f:
                snapshot = json.load(f)
            self._intents = snapshot.get("intents", {})
            self._snapshot_seq = snapshot.get("seq", 0)
            self._transition_count = snapshot.get("transition_count", 0)
            self._wal_lines_at_snapshot = snapshot.get("wal_line_count", 0)

            global _seq_counter
            with _seq_lock:
                _seq_counter = max(_seq_counter, self._snapshot_seq)

            logger.info(
                "Snapshot loaded: %d intents, seq=%d",
                len(self._intents), self._snapshot_seq,
            )
            return True
        except (json.JSONDecodeError, KeyError) as e:
            logger.warning("Corrupt snapshot, ignoring: %s", e)
            return False

    def _replay_wal(self, skip_before_snapshot: bool = False) -> None:
        """Replay WAL entries. Skip corrupt lines. Optionally skip entries covered by snapshot.

        When a snapshot exists, we skip the first `_wal_lines_at_snapshot` valid+empty
        lines (they're already captured in the snapshot) and only replay entries after that.
        """
        if not self._log_path.exists():
            return

        corrupt_count = 0
        replayed = 0
        skipped = 0
        skip_until = self._wal_lines_at_snapshot if skip_before_snapshot else 0
        line_index = 0  # counts all lines (valid entries + empty lines, NOT corrupt)

        with self._log_path.open("r", encoding="utf-8") as f:
            for line_no, raw_line in enumerate(f, 1):
                raw_line = raw_line.strip()
                if not raw_line:
                    line_index += 1
                    continue

                try:
                    entry = json.loads(raw_line)
                except json.JSONDecodeError as e:
                    corrupt_count += 1
                    logger.warning("Corrupt WAL line %d, skipping: %s", line_no, e)
                    continue

                line_index += 1
                if line_index <= skip_until:
                    skipped += 1
                    continue

                op = entry.get("op")
                if op == "create":
                    rec = entry["record"]
                    iid = rec["id"]
                    self._intents[iid] = rec
                    replayed += 1
                elif op == "transition":
                    iid = entry["id"]
                    if iid not in self._intents:
                        skipped += 1
                        continue
                    rec = self._intents[iid]
                    rec["state"] = entry["to"]
                    rec["updated_at"] = entry.get("at", rec["updated_at"])
                    rec["history"].append({
                        "from": entry["from"],
                        "to": entry["to"],
                        "at": entry.get("at"),
                        "data": entry.get("data"),
                    })
                    data = entry.get("data")
                    if data:
                        for key in ("plan", "report", "porch", "git", "configsync", "reposync", "errors", "package_dir"):
                            if key in data:
                                rec[key] = data[key]
                    replayed += 1

        # Update WAL line count so future snapshots record correct offset
        self._wal_line_count = line_index

        if corrupt_count:
            logger.warning("WAL replay: skipped %d corrupt lines", corrupt_count)
        if replayed or skipped:
            logger.info("WAL replay: %d applied, %d skipped (snapshot)", replayed, skipped)
