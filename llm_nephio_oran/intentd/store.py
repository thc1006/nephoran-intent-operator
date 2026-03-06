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

class IntentStore:
    """Thread-safe intent store with state machine enforcement."""

    def __init__(self, state_dir: Path):
        self._dir = state_dir
        self._dir.mkdir(parents=True, exist_ok=True)
        self._log_path = self._dir / "intents.jsonl"
        self._lock = threading.Lock()
        self._intents: dict[str, dict[str, Any]] = {}
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
                for key in ("plan", "report", "porch", "git", "errors", "package_dir"):
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

            return copy.deepcopy(rec)

    def cancel(self, intent_id: str) -> dict[str, Any]:
        """Cancel an intent. Only valid from ACKNOWLEDGED state."""
        return self.transition(intent_id, IntentState.CANCELLED,
                               data={"reason": "cancelled_by_user"})

    def build_report(self, intent_id: str) -> dict[str, Any] | None:
        """Build a TMF921 IntentReport from the intent record."""
        rec = self.get(intent_id)
        if rec is None:
            return None
        terminal = rec["state"] in ("completed", "failed", "cancelled")
        return {
            "intentId": rec["id"],
            "state": rec["state"],
            "expression": rec["expression"],
            "source": rec["source"],
            "compliant": rec["state"] == "completed",
            "terminal": terminal,
            "stageCount": len(rec["history"]),
            "createdAt": rec["created_at"],
            "updatedAt": rec["updated_at"],
            "plan": rec["plan"],
            "porch": rec["porch"],
            "git": rec["git"],
            "errors": rec["errors"],
            "report": rec["report"],
            "history": rec["history"],
        }

    # ── Persistence ────────────────────────────────────────────────────

    def _append_log(self, entry: dict[str, Any]) -> None:
        with self._log_path.open("a", encoding="utf-8") as f:
            f.write(json.dumps(entry, ensure_ascii=False, default=str) + "\n")

    def _load(self) -> None:
        """Replay jsonl log to rebuild in-memory state."""
        if not self._log_path.exists():
            return
        with self._log_path.open("r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                entry = json.loads(line)
                if entry["op"] == "create":
                    rec = entry["record"]
                    self._intents[rec["id"]] = rec
                elif entry["op"] == "transition":
                    iid = entry["id"]
                    if iid in self._intents:
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
                            for key in ("plan", "report", "porch", "git", "errors", "package_dir"):
                                if key in data:
                                    rec[key] = data[key]
        # Ensure seq counter starts above any existing IDs to prevent collision
        if self._intents:
            _init_seq_from_existing(set(self._intents.keys()))
