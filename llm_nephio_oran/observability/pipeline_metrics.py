"""Prometheus metrics for the M7 closed-loop pipeline.

Exposes counters, gauges, and histograms that Grafana dashboards consume.
All metrics use the `nephoran_` prefix for namespace isolation.

Metrics:
  nephoran_intent_created_total          — intents created (by source)
  nephoran_intent_completed_total        — intents completed/failed (by result)
  nephoran_pipeline_stage_duration_seconds — time per pipeline stage
  nephoran_e2kpm_prb_usage_percent       — E2 KPM PRB usage per cell
  nephoran_scale_actions_total           — scale actions (by component, action)
  nephoran_porch_lifecycle_total         — Porch lifecycle transitions
  nephoran_active_intents                — currently in-flight intents
"""
from __future__ import annotations

import contextlib
import time
from typing import Any

from prometheus_client import (
    CollectorRegistry,
    Counter,
    Gauge,
    Histogram,
    generate_latest,
)

# Use a dedicated registry so tests can reset without affecting the global one
_registry = CollectorRegistry()

# ── Counters ──────────────────────────────────────────────────────────────

intent_created = Counter(
    "nephoran_intent_created_total",
    "Total intents created",
    labelnames=["source"],
    registry=_registry,
)

intent_completed = Counter(
    "nephoran_intent_completed_total",
    "Total intents reaching terminal state",
    labelnames=["result"],
    registry=_registry,
)

scale_actions = Counter(
    "nephoran_scale_actions_total",
    "Scale actions triggered by closed-loop",
    labelnames=["component", "action"],
    registry=_registry,
)

porch_lifecycle = Counter(
    "nephoran_porch_lifecycle_total",
    "Porch PackageRevision lifecycle transitions",
    labelnames=["lifecycle"],
    registry=_registry,
)

git_commits = Counter(
    "nephoran_git_commits_total",
    "Git commits pushed by the pipeline (excludes dry-run)",
    registry=_registry,
)

# ── Gauges ────────────────────────────────────────────────────────────────

# NOTE: cell_id × gnb_id cardinality is bounded by O-RAN deployment size.
# In production, ensure cell count stays reasonable (<1000) to avoid
# high-cardinality metric explosion.
e2kpm_prb_usage = Gauge(
    "nephoran_e2kpm_prb_usage_percent",
    "E2 KPM PRB usage percentage per cell",
    labelnames=["cell_id", "gnb_id"],
    registry=_registry,
)

active_intents = Gauge(
    "nephoran_active_intents",
    "Number of in-flight intents (not yet terminal)",
    registry=_registry,
)

# ── Histograms ────────────────────────────────────────────────────────────

pipeline_stage_duration = Histogram(
    "nephoran_pipeline_stage_duration_seconds",
    "Duration of each pipeline stage",
    labelnames=["stage"],
    buckets=(0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0),
    registry=_registry,
)


# ── Helpers ───────────────────────────────────────────────────────────────

def get_metric_value(
    metric: Any,
    labels: dict[str, str] | None = None,
    suffix: str = "",
) -> float:
    """Extract a metric's current value (for testing).

    Args:
        metric: A prometheus_client metric object.
        labels: Label dict to select the specific time series.
        suffix: Metric name suffix (e.g., "_count" for histogram count).

    Returns:
        The current float value.
    """
    # Compute target name once outside the loop
    target_name = metric.describe()[0].name + suffix if hasattr(metric, 'describe') else ""
    for metric_family in _registry.collect():
        for sample in metric_family.samples:
            if sample.name != target_name and sample.name != target_name + "_total":
                continue
            if labels:
                if all(sample.labels.get(k) == v for k, v in labels.items()):
                    return sample.value
            elif not sample.labels:
                return sample.value
    return 0.0


@contextlib.contextmanager
def instrument_stage(stage_name: str):
    """Context manager that records pipeline stage duration.

    Usage:
        with instrument_stage("planning"):
            plan = planner.plan(expression)
    """
    start = time.monotonic()
    try:
        yield
    finally:
        elapsed = time.monotonic() - start
        pipeline_stage_duration.labels(stage=stage_name).observe(elapsed)


def generate_metrics_text() -> str:
    """Generate Prometheus text exposition format."""
    return generate_latest(_registry).decode("utf-8")


def sum_metric(metric: Any) -> float:
    """Sum all label-series values for a metric.

    Unlike get_metric_value (which requires exact label match),
    this sums across ALL label combinations. Useful for counters
    with labels (e.g. scale_actions{component=..., action=...}).
    """
    total = 0.0
    target_name = metric.describe()[0].name if hasattr(metric, "describe") else ""
    for family in _registry.collect():
        for sample in family.samples:
            if sample.name == target_name or sample.name == target_name + "_total":
                total += sample.value
    return total


def collect_json_snapshot() -> dict[str, Any]:
    """Collect all metrics into a structured JSON dict for the frontend dashboard.

    Iterates the registry once, categorizes samples by metric name, and returns
    a structured response suitable for the React Closed-Loop dashboard.
    """
    from datetime import datetime, timezone

    overview = {
        "active_intents": 0.0,
        "total_created": 0.0,
        "completed": 0.0,
        "failed": 0.0,
        "git_commits": 0.0,
    }
    cells: dict[tuple[str, str], float] = {}
    sa_map: dict[tuple[str, str], float] = {}
    porch_map: dict[str, float] = {}
    stage_counts: dict[str, float] = {}
    stage_sums: dict[str, float] = {}
    stage_buckets: dict[str, list[tuple[float, float]]] = {}

    for family in _registry.collect():
        for sample in family.samples:
            name = sample.name

            # ── Counters ──
            if name in ("nephoran_intent_created_total", "nephoran_intent_created"):
                overview["total_created"] += sample.value

            elif name in ("nephoran_intent_completed_total", "nephoran_intent_completed"):
                result = sample.labels.get("result", "")
                if result == "completed":
                    overview["completed"] += sample.value
                elif result == "failed":
                    overview["failed"] += sample.value

            elif name in ("nephoran_git_commits_total", "nephoran_git_commits"):
                overview["git_commits"] += sample.value

            elif name in ("nephoran_scale_actions_total", "nephoran_scale_actions"):
                comp = sample.labels.get("component", "")
                action = sample.labels.get("action", "")
                if comp or action:
                    sa_map[(comp, action)] = sa_map.get((comp, action), 0.0) + sample.value

            elif name in ("nephoran_porch_lifecycle_total", "nephoran_porch_lifecycle"):
                lc = sample.labels.get("lifecycle", "")
                if lc:
                    porch_map[lc] = porch_map.get(lc, 0.0) + sample.value

            # ── Gauges ──
            elif name == "nephoran_active_intents":
                overview["active_intents"] = sample.value

            elif name == "nephoran_e2kpm_prb_usage_percent":
                cell_id = sample.labels.get("cell_id", "")
                gnb_id = sample.labels.get("gnb_id", "")
                if cell_id:
                    cells[(cell_id, gnb_id)] = sample.value

            # ── Histogram ──
            elif name == "nephoran_pipeline_stage_duration_seconds_bucket":
                stage = sample.labels.get("stage", "")
                le = sample.labels.get("le", "")
                if stage and le:
                    stage_buckets.setdefault(stage, []).append(
                        (float(le) if le != "+Inf" else float("inf"), sample.value)
                    )
            elif name == "nephoran_pipeline_stage_duration_seconds_count":
                stage = sample.labels.get("stage", "")
                if stage:
                    stage_counts[stage] = sample.value
            elif name == "nephoran_pipeline_stage_duration_seconds_sum":
                stage = sample.labels.get("stage", "")
                if stage:
                    stage_sums[stage] = sample.value

    # Build pipeline stages with p95 from histogram buckets
    pipeline_stages = []
    all_stages = set(stage_counts.keys()) | set(stage_sums.keys()) | set(stage_buckets.keys())
    for stage in sorted(all_stages):
        count = stage_counts.get(stage, 0.0)
        total_sum = stage_sums.get(stage, 0.0)
        p95 = _histogram_quantile(0.95, stage_buckets.get(stage, []), count)
        pipeline_stages.append({
            "stage": stage,
            "count": int(count),
            "sum_seconds": round(total_sum, 4),
            "p95_seconds": round(p95, 4) if p95 is not None else None,
        })

    return {
        "overview": {k: int(v) if v == int(v) else v for k, v in overview.items()},
        "e2kpm": {
            "cells": [
                {"cell_id": cid, "gnb_id": gid, "prb_usage_percent": round(val, 2)}
                for (cid, gid), val in sorted(cells.items())
            ],
        },
        "scale_actions": [
            {"component": comp, "action": action, "count": int(cnt)}
            for (comp, action), cnt in sorted(sa_map.items())
        ],
        "porch_lifecycle": [
            {"lifecycle": lc, "count": int(cnt)}
            for lc, cnt in sorted(porch_map.items())
        ],
        "pipeline_stages": pipeline_stages,
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


def _histogram_quantile(
    q: float,
    buckets: list[tuple[float, float]],
    total_count: float,
) -> float | None:
    """Compute quantile from histogram bucket counts (same algorithm as Prometheus).

    Args:
        q: Quantile (e.g. 0.95).
        buckets: List of (upper_bound, cumulative_count) pairs.
        total_count: Total observation count.

    Returns:
        Estimated quantile value, or None if no observations.
    """
    if total_count == 0 or not buckets:
        return None

    sorted_buckets = sorted(buckets, key=lambda b: b[0])
    target = q * total_count
    prev_bound = 0.0
    prev_count = 0.0

    for upper_bound, cumulative_count in sorted_buckets:
        if upper_bound == float("inf"):
            # Cannot interpolate into +Inf bucket; use previous bound
            if prev_count < target and prev_bound > 0:
                return prev_bound
            continue
        if cumulative_count >= target:
            # Linear interpolation within this bucket
            bucket_count = cumulative_count - prev_count
            if bucket_count > 0:
                fraction = (target - prev_count) / bucket_count
                return prev_bound + fraction * (upper_bound - prev_bound)
            return upper_bound
        prev_bound = upper_bound
        prev_count = cumulative_count

    # All observations in +Inf bucket: return last finite bound
    return prev_bound if prev_bound > 0 else None


def reset_metrics() -> None:
    """Reset all metrics (for testing only)."""
    global _registry, intent_created, intent_completed, scale_actions
    global porch_lifecycle, git_commits, e2kpm_prb_usage, active_intents, pipeline_stage_duration

    _registry = CollectorRegistry()

    intent_created = Counter(
        "nephoran_intent_created_total",
        "Total intents created",
        labelnames=["source"],
        registry=_registry,
    )
    intent_completed = Counter(
        "nephoran_intent_completed_total",
        "Total intents reaching terminal state",
        labelnames=["result"],
        registry=_registry,
    )
    scale_actions = Counter(
        "nephoran_scale_actions_total",
        "Scale actions triggered by closed-loop",
        labelnames=["component", "action"],
        registry=_registry,
    )
    porch_lifecycle = Counter(
        "nephoran_porch_lifecycle_total",
        "Porch PackageRevision lifecycle transitions",
        labelnames=["lifecycle"],
        registry=_registry,
    )
    git_commits = Counter(
        "nephoran_git_commits_total",
        "Git commits pushed by the pipeline (excludes dry-run)",
        registry=_registry,
    )
    e2kpm_prb_usage = Gauge(
        "nephoran_e2kpm_prb_usage_percent",
        "E2 KPM PRB usage percentage per cell",
        labelnames=["cell_id", "gnb_id"],
        registry=_registry,
    )
    active_intents = Gauge(
        "nephoran_active_intents",
        "Number of in-flight intents (not yet terminal)",
        registry=_registry,
    )
    pipeline_stage_duration = Histogram(
        "nephoran_pipeline_stage_duration_seconds",
        "Duration of each pipeline stage",
        labelnames=["stage"],
        buckets=(0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0),
        registry=_registry,
    )
