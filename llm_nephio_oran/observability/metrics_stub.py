from __future__ import annotations

from typing import Any


def snapshot_metrics() -> dict[str, Any]:
    """Return a placeholder metrics snapshot.

    Replace with Prometheus queries + RIC/KPM retrieval.
    """
    return {
        "timestamp": "TODO",
        "k8s": {"cpu_util": 0.0, "mem_util": 0.0},
        "ran": {"prb_util": None},
        "slice": {"throughput_mbps": None, "latency_ms": None},
    }
