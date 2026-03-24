"""Tests for E2 KPM Simulator — E2SM-KPM metrics generation (T5, ADR-009, ADR-0006).

Verifies:
  - Metric generation with realistic O-RAN KPM values
  - Prometheus exposition format output
  - Cell-level and slice-level metric breakdown
  - Configurable cell/slice parameters
  - Metric value ranges (non-negative, bounded)
"""
from __future__ import annotations

import pytest

from llm_nephio_oran.e2sim.kpm_simulator import (
    CellConfig,
    E2KpmSimulator,
    KpmSnapshot,
)


class TestCellConfig:
    def test_defaults(self):
        cell = CellConfig(cell_id="cell-001", gnb_id="gnb-edge01")
        assert cell.cell_id == "cell-001"
        assert cell.max_prb == 273  # 100 MHz NR default
        assert cell.slices == ["embb", "urllc"]

    def test_custom_slices(self):
        cell = CellConfig(cell_id="c1", gnb_id="g1", slices=["embb", "urllc", "mmtc"])
        assert len(cell.slices) == 3


class TestKpmSnapshot:
    def test_has_required_fields(self):
        snap = KpmSnapshot(
            cell_id="cell-001",
            gnb_id="gnb-001",
            slice_id="embb",
            dl_throughput_mbps=150.5,
            ul_throughput_mbps=45.2,
            prb_usage_pct=62.3,
            active_ues=42,
            dl_prb_used=170,
            ul_prb_used=85,
            cqi_average=11.5,
            rsrp_average=-85.0,
        )
        assert snap.dl_throughput_mbps == 150.5
        assert snap.active_ues == 42
        assert snap.cqi_average == 11.5


class TestE2KpmSimulator:
    @pytest.fixture
    def sim(self):
        cells = [
            CellConfig(cell_id="cell-001", gnb_id="gnb-edge01"),
            CellConfig(cell_id="cell-002", gnb_id="gnb-edge01"),
        ]
        return E2KpmSimulator(cells=cells, seed=42)

    def test_generate_snapshots(self, sim):
        snapshots = sim.generate()
        # 2 cells × 2 slices each = 4 snapshots
        assert len(snapshots) == 4

    def test_snapshot_values_bounded(self, sim):
        for snap in sim.generate():
            assert snap.dl_throughput_mbps >= 0
            assert snap.ul_throughput_mbps >= 0
            assert 0 <= snap.prb_usage_pct <= 100
            assert snap.active_ues >= 0
            assert 1 <= snap.cqi_average <= 15
            assert -140 <= snap.rsrp_average <= -44

    def test_prometheus_exposition(self, sim):
        text = sim.to_prometheus()
        assert "e2_kpm_dl_throughput_mbps" in text
        assert "e2_kpm_ul_throughput_mbps" in text
        assert "e2_kpm_prb_usage_percent" in text
        assert "e2_kpm_active_ues" in text
        assert "e2_kpm_cqi_average" in text
        assert "e2_kpm_rsrp_average" in text
        # Labels present
        assert 'cell_id="cell-001"' in text
        assert 'gnb_id="gnb-edge01"' in text
        assert 'slice_id="embb"' in text

    def test_prometheus_help_and_type(self, sim):
        text = sim.to_prometheus()
        assert "# HELP e2_kpm_dl_throughput_mbps" in text
        assert "# TYPE e2_kpm_dl_throughput_mbps gauge" in text

    def test_single_cell_single_slice(self):
        sim = E2KpmSimulator(cells=[
            CellConfig(cell_id="c1", gnb_id="g1", slices=["urllc"]),
        ])
        snapshots = sim.generate()
        assert len(snapshots) == 1
        assert snapshots[0].slice_id == "urllc"

    def test_urllc_lower_latency_profile(self):
        """URLLC slices should have lower throughput but higher CQI than eMBB."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="c1", gnb_id="g1", slices=["embb", "urllc"])],
            seed=42,
        )
        snapshots = sim.generate()
        embb = [s for s in snapshots if s.slice_id == "embb"][0]
        urllc = [s for s in snapshots if s.slice_id == "urllc"][0]
        # eMBB range: dl 100-500, URLLC range: dl 5-50 → eMBB always higher
        assert embb.dl_throughput_mbps > urllc.dl_throughput_mbps, (
            f"eMBB DL {embb.dl_throughput_mbps} should exceed URLLC DL {urllc.dl_throughput_mbps}"
        )
        # URLLC CQI range 11-15, eMBB CQI range 8-14 → URLLC >= eMBB on average
        assert urllc.cqi_average >= embb.cqi_average - 2, (
            f"URLLC CQI {urllc.cqi_average} should be close to or above eMBB CQI {embb.cqi_average}"
        )

    def test_high_load_scenario(self):
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="c1", gnb_id="g1", slices=["embb"])],
            load_factor=0.9,
        )
        snapshots = sim.generate()
        # High load eMBB → PRB usage should be above 40% (range 40-90%)
        assert all(s.prb_usage_pct > 35 for s in snapshots)

    def test_low_load_scenario(self):
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="c1", gnb_id="g1", slices=["embb"])],
            load_factor=0.1,
        )
        snapshots = sim.generate()
        # Low load eMBB → PRB usage should be below 75%
        assert all(s.prb_usage_pct < 75 for s in snapshots)
