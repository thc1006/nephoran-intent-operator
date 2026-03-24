"""E2 KPM Simulator — generates E2SM-KPM style metrics (T5, ADR-009, ADR-0006).

Simulates O-RAN E2SM-KPM v3.0 metrics at cell and slice granularity:
  - DL/UL throughput (Mbps)
  - PRB usage (%)
  - Active UE count
  - CQI average (1-15)
  - RSRP average (dBm)

Output:
  - KpmSnapshot dataclass per cell×slice
  - Prometheus exposition text (/metrics endpoint)
  - Configurable load factor for scaling scenarios

Slice profiles:
  - eMBB: high throughput, moderate PRB, many UEs
  - URLLC: low throughput, low PRB, few UEs, high CQI
  - mMTC: very low throughput, low PRB, very many UEs
"""
from __future__ import annotations

import random
from dataclasses import dataclass, field
from typing import Any


@dataclass
class CellConfig:
    """Configuration for a simulated cell."""
    cell_id: str
    gnb_id: str
    max_prb: int = 273  # 100 MHz NR default (273 PRBs for 30 kHz SCS)
    slices: list[str] = field(default_factory=lambda: ["embb", "urllc"])


@dataclass
class KpmSnapshot:
    """Single KPM measurement for a cell×slice pair."""
    cell_id: str
    gnb_id: str
    slice_id: str
    dl_throughput_mbps: float
    ul_throughput_mbps: float
    prb_usage_pct: float
    active_ues: int
    dl_prb_used: int
    ul_prb_used: int
    cqi_average: float
    rsrp_average: float


# Slice-specific metric profiles: (dl_base, ul_base, prb_base, ue_base, cqi_base)
_SLICE_PROFILES: dict[str, dict[str, Any]] = {
    "embb": {
        "dl_range": (100, 500),   # Mbps
        "ul_range": (30, 150),
        "prb_range": (40, 90),    # %
        "ue_range": (20, 200),
        "cqi_range": (8, 14),
        "rsrp_range": (-100, -60),
    },
    "urllc": {
        "dl_range": (5, 50),
        "ul_range": (2, 20),
        "prb_range": (5, 30),
        "ue_range": (1, 20),
        "cqi_range": (11, 15),
        "rsrp_range": (-85, -50),
    },
    "mmtc": {
        "dl_range": (1, 10),
        "ul_range": (0.5, 5),
        "prb_range": (10, 40),
        "ue_range": (100, 10000),
        "cqi_range": (5, 10),
        "rsrp_range": (-120, -80),
    },
}

_DEFAULT_PROFILE = _SLICE_PROFILES["embb"]


class E2KpmSimulator:
    """Generates E2SM-KPM metric snapshots for simulated cells."""

    def __init__(
        self,
        cells: list[CellConfig] | None = None,
        load_factor: float = 0.5,
        seed: int | None = None,
    ):
        self.cells = cells or [
            CellConfig(cell_id="cell-001", gnb_id="gnb-edge01"),
        ]
        self.load_factor = max(0.0, min(1.0, load_factor))
        self._rng = random.Random(seed)

    def generate(self) -> list[KpmSnapshot]:
        """Generate one KPM snapshot per cell×slice pair."""
        snapshots = []
        for cell in self.cells:
            for slice_id in cell.slices:
                snap = self._generate_one(cell, slice_id)
                snapshots.append(snap)
        return snapshots

    def _generate_one(self, cell: CellConfig, slice_id: str) -> KpmSnapshot:
        profile = _SLICE_PROFILES.get(slice_id, _DEFAULT_PROFILE)
        lf = self.load_factor

        dl_lo, dl_hi = profile["dl_range"]
        ul_lo, ul_hi = profile["ul_range"]
        prb_lo, prb_hi = profile["prb_range"]
        ue_lo, ue_hi = profile["ue_range"]
        cqi_lo, cqi_hi = profile["cqi_range"]
        rsrp_lo, rsrp_hi = profile["rsrp_range"]

        # Apply load factor: higher load → values closer to upper range
        dl = self._scaled(dl_lo, dl_hi, lf)
        ul = self._scaled(ul_lo, ul_hi, lf)
        prb_pct = self._scaled(prb_lo, prb_hi, lf)
        ue_count = int(self._scaled(ue_lo, ue_hi, lf))
        cqi = self._scaled(cqi_lo, cqi_hi, lf)
        rsrp = self._scaled(rsrp_lo, rsrp_hi, lf)

        dl_prb = int(prb_pct / 100 * cell.max_prb * 0.6)
        ul_prb = int(prb_pct / 100 * cell.max_prb * 0.4)

        return KpmSnapshot(
            cell_id=cell.cell_id,
            gnb_id=cell.gnb_id,
            slice_id=slice_id,
            dl_throughput_mbps=round(dl, 1),
            ul_throughput_mbps=round(ul, 1),
            prb_usage_pct=round(prb_pct, 1),
            active_ues=ue_count,
            dl_prb_used=dl_prb,
            ul_prb_used=ul_prb,
            cqi_average=round(cqi, 1),
            rsrp_average=round(rsrp, 1),
        )

    def _scaled(self, lo: float, hi: float, load: float) -> float:
        """Generate a value biased by load factor with some randomness."""
        center = lo + (hi - lo) * load
        spread = (hi - lo) * 0.15
        val = self._rng.gauss(center, spread)
        return max(lo, min(hi, val))

    def to_prometheus(self, snapshots: list[KpmSnapshot] | None = None) -> str:
        """Generate Prometheus exposition format text.

        Args:
            snapshots: Pre-generated snapshots to format. If None, generates new ones.
        """
        if snapshots is None:
            snapshots = self.generate()
        lines: list[str] = []

        metrics = [
            ("e2_kpm_dl_throughput_mbps", "Downlink throughput in Mbps", "gauge", "dl_throughput_mbps"),
            ("e2_kpm_ul_throughput_mbps", "Uplink throughput in Mbps", "gauge", "ul_throughput_mbps"),
            ("e2_kpm_prb_usage_percent", "PRB usage percentage", "gauge", "prb_usage_pct"),
            ("e2_kpm_active_ues", "Number of active UEs", "gauge", "active_ues"),
            ("e2_kpm_dl_prb_used", "Downlink PRBs used", "gauge", "dl_prb_used"),
            ("e2_kpm_ul_prb_used", "Uplink PRBs used", "gauge", "ul_prb_used"),
            ("e2_kpm_cqi_average", "Average CQI (1-15)", "gauge", "cqi_average"),
            ("e2_kpm_rsrp_average", "Average RSRP in dBm", "gauge", "rsrp_average"),
        ]

        for metric_name, help_text, metric_type, attr in metrics:
            lines.append(f"# HELP {metric_name} {help_text}")
            lines.append(f"# TYPE {metric_name} {metric_type}")
            for snap in snapshots:
                labels = f'cell_id="{snap.cell_id}",gnb_id="{snap.gnb_id}",slice_id="{snap.slice_id}"'
                value = getattr(snap, attr)
                lines.append(f"{metric_name}{{{labels}}} {value}")
            lines.append("")

        return "\n".join(lines)
