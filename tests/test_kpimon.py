"""Tests for KPIMON bridge — E2 KPM metrics → Prometheus (T6, ADR-005).

The KPIMON bridge:
  1. Collects KPM snapshots from E2KpmSimulator (or real E2 interface)
  2. Exposes them as Prometheus metrics via /metrics HTTP endpoint
  3. Provides query helpers for the closed-loop Observe phase
"""
from __future__ import annotations

from unittest.mock import patch

import pytest

from llm_nephio_oran.e2sim.kpm_simulator import CellConfig, E2KpmSimulator
from llm_nephio_oran.kpimon.bridge import (
    KpimonBridge,
    KpimonQuery,
)


class TestKpimonBridge:
    @pytest.fixture
    def bridge(self):
        sim = E2KpmSimulator(
            cells=[
                CellConfig(cell_id="cell-001", gnb_id="gnb-edge01"),
                CellConfig(cell_id="cell-002", gnb_id="gnb-edge01"),
            ],
            seed=42,
        )
        return KpimonBridge(simulator=sim)

    def test_collect_returns_snapshots(self, bridge):
        snapshots = bridge.collect()
        assert len(snapshots) == 4  # 2 cells × 2 slices

    def test_metrics_endpoint_text(self, bridge):
        text = bridge.metrics_text()
        assert "e2_kpm_dl_throughput_mbps" in text
        assert "# HELP" in text

    def test_summary_by_cell(self, bridge):
        summary = bridge.summary_by_cell()
        assert "cell-001" in summary
        assert "cell-002" in summary
        assert summary["cell-001"]["gnb_id"] == "gnb-edge01"
        assert "total_ues" in summary["cell-001"]
        assert "avg_prb_usage" in summary["cell-001"]

    def test_summary_by_slice(self, bridge):
        summary = bridge.summary_by_slice()
        assert "embb" in summary
        assert "urllc" in summary
        assert "total_dl_throughput" in summary["embb"]


class TestKpimonQuery:
    @pytest.fixture
    def query(self):
        sim = E2KpmSimulator(
            cells=[
                CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb", "urllc"]),
            ],
            load_factor=0.7,
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)
        return KpimonQuery(bridge=bridge)

    def test_get_cell_metrics(self, query):
        query.refresh()
        m = query.cell_metrics("cell-001")
        assert m is not None
        assert len(m) == 2  # embb + urllc slices

    def test_get_cell_metrics_nonexistent(self, query):
        query.refresh()
        m = query.cell_metrics("nonexistent")
        assert m == []

    def test_get_slice_metrics(self, query):
        query.refresh()
        m = query.slice_metrics("embb")
        assert len(m) >= 1
        assert all(s.slice_id == "embb" for s in m)

    def test_high_prb_cells(self, query):
        """Find cells with PRB usage above threshold."""
        # Refresh so cache is populated before querying
        query.refresh()
        cells = query.high_prb_cells(threshold=20.0)
        assert isinstance(cells, list)
        # With load_factor=0.7 and seed=42, eMBB PRB range is 40-90% → should have at least 1 hit
        assert len(cells) >= 1, f"Expected high PRB cells at threshold=20% with load_factor=0.7, got {cells}"

    def test_total_active_ues(self, query):
        query.refresh()
        total = query.total_active_ues()
        assert total > 0
