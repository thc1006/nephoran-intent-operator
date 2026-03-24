"""KPIMON bridge — E2 KPM metrics collection and query (T6, ADR-005).

Bridges E2KpmSimulator output to:
  1. Prometheus exposition text (/metrics)
  2. Structured query API for closed-loop Observe phase
  3. Cell-level and slice-level aggregation summaries

In production, this would read from the RIC KPIMON xApp's SDL/Redis store.
For MVP, it wraps E2KpmSimulator to provide the same interface.
"""
from __future__ import annotations

from typing import Any

from llm_nephio_oran.e2sim.kpm_simulator import E2KpmSimulator, KpmSnapshot


class KpimonBridge:
    """Collects KPM snapshots and exposes them for Prometheus and queries."""

    def __init__(self, simulator: E2KpmSimulator):
        self._sim = simulator
        self._cache: list[KpmSnapshot] | None = None

    def collect(self) -> list[KpmSnapshot]:
        """Collect fresh KPM snapshots from all cells."""
        self._cache = self._sim.generate()
        return self._cache

    def _ensure_cache(self) -> list[KpmSnapshot]:
        if self._cache is None:
            self.collect()
        return self._cache  # type: ignore[return-value]

    def metrics_text(self) -> str:
        """Return Prometheus exposition format text.

        Uses the same snapshots as the cache to ensure consistency
        between metrics_text() output and summary_by_*() queries.
        """
        snapshots = self.collect()  # always fresh
        return self._sim.to_prometheus(snapshots=snapshots)

    def summary_by_cell(self) -> dict[str, Any]:
        """Aggregate metrics per cell (across slices)."""
        snapshots = self._ensure_cache()
        cells: dict[str, dict[str, Any]] = {}

        for snap in snapshots:
            if snap.cell_id not in cells:
                cells[snap.cell_id] = {
                    "gnb_id": snap.gnb_id,
                    "total_ues": 0,
                    "total_dl_throughput": 0.0,
                    "total_ul_throughput": 0.0,
                    "prb_values": [],
                    "slices": [],
                }
            cell = cells[snap.cell_id]
            cell["total_ues"] += snap.active_ues
            cell["total_dl_throughput"] += snap.dl_throughput_mbps
            cell["total_ul_throughput"] += snap.ul_throughput_mbps
            cell["prb_values"].append(snap.prb_usage_pct)
            cell["slices"].append(snap.slice_id)

        # Compute averages
        for cell in cells.values():
            prb_vals = cell.pop("prb_values")
            cell["avg_prb_usage"] = round(sum(prb_vals) / len(prb_vals), 1) if prb_vals else 0.0
            cell["total_dl_throughput"] = round(cell["total_dl_throughput"], 1)
            cell["total_ul_throughput"] = round(cell["total_ul_throughput"], 1)

        return cells

    def summary_by_slice(self) -> dict[str, Any]:
        """Aggregate metrics per slice type (across cells)."""
        snapshots = self._ensure_cache()
        slices: dict[str, dict[str, Any]] = {}

        for snap in snapshots:
            if snap.slice_id not in slices:
                slices[snap.slice_id] = {
                    "total_dl_throughput": 0.0,
                    "total_ul_throughput": 0.0,
                    "total_ues": 0,
                    "cell_count": 0,
                    "avg_cqi": [],
                }
            sl = slices[snap.slice_id]
            sl["total_dl_throughput"] += snap.dl_throughput_mbps
            sl["total_ul_throughput"] += snap.ul_throughput_mbps
            sl["total_ues"] += snap.active_ues
            sl["cell_count"] += 1
            sl["avg_cqi"].append(snap.cqi_average)

        for sl in slices.values():
            cqi_vals = sl.pop("avg_cqi")
            sl["avg_cqi"] = round(sum(cqi_vals) / len(cqi_vals), 1) if cqi_vals else 0.0
            sl["total_dl_throughput"] = round(sl["total_dl_throughput"], 1)
            sl["total_ul_throughput"] = round(sl["total_ul_throughput"], 1)

        return slices


class KpimonQuery:
    """Query helper for closed-loop Observe phase."""

    def __init__(self, bridge: KpimonBridge):
        self._bridge = bridge

    def refresh(self) -> list[KpmSnapshot]:
        """Force-refresh the cache and return fresh snapshots."""
        return self._bridge.collect()

    def cell_metrics(self, cell_id: str) -> list[KpmSnapshot]:
        """Get all KPM snapshots for a specific cell."""
        snapshots = self._bridge._ensure_cache()
        return [s for s in snapshots if s.cell_id == cell_id]

    def slice_metrics(self, slice_id: str) -> list[KpmSnapshot]:
        """Get all KPM snapshots for a specific slice type."""
        snapshots = self._bridge._ensure_cache()
        return [s for s in snapshots if s.slice_id == slice_id]

    def high_prb_cells(self, threshold: float = 80.0) -> list[dict[str, Any]]:
        """Find cells with PRB usage above threshold (candidates for scaling)."""
        snapshots = self._bridge._ensure_cache()
        results = []
        for snap in snapshots:
            if snap.prb_usage_pct > threshold:
                results.append({
                    "cell_id": snap.cell_id,
                    "gnb_id": snap.gnb_id,
                    "slice_id": snap.slice_id,
                    "prb_usage_pct": snap.prb_usage_pct,
                    "active_ues": snap.active_ues,
                })
        return results

    def low_cqi_cells(self, threshold: float = 8.0) -> list[dict[str, Any]]:
        """Find cells with average CQI below threshold (TS candidates).

        Low CQI indicates poor channel quality, suggesting traffic should be
        steered to cells with better conditions.
        """
        by_cell = self._bridge.summary_by_cell()
        # summary_by_cell doesn't include per-cell CQI — compute it here
        snapshots = self._bridge._ensure_cache()
        cell_cqi: dict[str, list[float]] = {}
        for snap in snapshots:
            cell_cqi.setdefault(snap.cell_id, []).append(snap.cqi_average)

        results = []
        for cell_id, cqi_vals in cell_cqi.items():
            avg_cqi = sum(cqi_vals) / len(cqi_vals)
            if avg_cqi < threshold:
                cell_info = by_cell.get(cell_id, {})
                results.append({
                    "cell_id": cell_id,
                    "gnb_id": cell_info.get("gnb_id", "unknown"),
                    "avg_cqi": round(avg_cqi, 1),
                    "avg_prb_usage": cell_info.get("avg_prb_usage", 0.0),
                    "total_ues": cell_info.get("total_ues", 0),
                })
        return results

    def total_active_ues(self) -> int:
        """Total active UEs across all cells and slices."""
        snapshots = self._bridge._ensure_cache()
        return sum(s.active_ues for s in snapshots)
