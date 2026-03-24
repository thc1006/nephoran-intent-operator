#!/usr/bin/env python3
"""M7 Demo CLI — interactive demonstration of the Nephoran closed-loop pipeline.

Usage:
    python -m llm_nephio_oran.demo.cli                     # default: dry-run, load=0.95
    python -m llm_nephio_oran.demo.cli --load 0.5          # moderate load
    python -m llm_nephio_oran.demo.cli --live              # live mode (requires running intentd)
    python -m llm_nephio_oran.demo.cli --pipeline          # dry-run through intentd v2 pipeline
    python -m llm_nephio_oran.demo.cli --cells 3           # 3 cells
    python -m llm_nephio_oran.demo.cli --step              # step-by-step (pause between phases)
"""
from __future__ import annotations

import argparse
import json
import sys
import time

from llm_nephio_oran.demo.runner import DemoRunner
from llm_nephio_oran.e2sim.kpm_simulator import CellConfig

# ── ANSI helpers (no external deps) ──────────────────────────────────────

BOLD = "\033[1m"
DIM = "\033[2m"
GREEN = "\033[32m"
YELLOW = "\033[33m"
RED = "\033[31m"
CYAN = "\033[36m"
MAGENTA = "\033[35m"
RESET = "\033[0m"


def _header(text: str) -> None:
    bar = "=" * 60
    print(f"\n{BOLD}{CYAN}{bar}{RESET}")
    print(f"{BOLD}{CYAN}  {text}{RESET}")
    print(f"{BOLD}{CYAN}{bar}{RESET}\n")


def _phase(name: str) -> None:
    print(f"\n{BOLD}{MAGENTA}>>> Phase: {name}{RESET}")
    print(f"{DIM}{'─' * 50}{RESET}")


def _kv(key: str, value: object, indent: int = 2) -> None:
    pad = " " * indent
    print(f"{pad}{DIM}{key}:{RESET} {value}")


def _ok(msg: str) -> None:
    print(f"  {GREEN}✓{RESET} {msg}")


def _warn(msg: str) -> None:
    print(f"  {YELLOW}!{RESET} {msg}")


def _err(msg: str) -> None:
    print(f"  {RED}✗{RESET} {msg}")


def _wait_enter() -> None:
    input(f"\n{DIM}  Press Enter to continue...{RESET}")


# ── Cell factory ─────────────────────────────────────────────────────────

def _make_cells(count: int) -> list[CellConfig]:
    cells = []
    for i in range(count):
        idx = i + 1
        site = "edge" if idx <= 2 else "central" if idx <= 4 else "lab"
        cells.append(CellConfig(
            cell_id=f"cell-{idx:03d}",
            gnb_id=f"gnb-{site}{idx:02d}",
            slices=["embb", "urllc"],
        ))
    return cells


# ── Main ─────────────────────────────────────────────────────────────────

def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Nephoran M7 Closed-Loop Demo",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--load", type=float, default=0.95,
                        help="E2 KPM load factor 0.0-1.0 (default: 0.95)")
    parser.add_argument("--cells", type=int, default=1,
                        help="Number of simulated cells (default: 1)")
    parser.add_argument("--seed", type=int, default=42,
                        help="Random seed for reproducible output (default: 42)")
    parser.add_argument("--threshold", type=float, default=80.0,
                        help="PRB threshold for scale-out (default: 80.0)")
    parser.add_argument("--live", action="store_true",
                        help="Live mode: POST to running intentd API")
    parser.add_argument("--pipeline", action="store_true",
                        help="Pipeline mode: run through intentd v2 (dry-run)")
    parser.add_argument("--step", action="store_true",
                        help="Step-by-step: pause between phases")
    parser.add_argument("--json", action="store_true",
                        help="Output summary as JSON (machine-readable)")
    args = parser.parse_args(argv)

    cells = _make_cells(args.cells)
    dry_run = not args.live

    demo = DemoRunner(
        cells=cells,
        load_factor=args.load,
        seed=args.seed,
        dry_run=dry_run,
        prb_threshold=args.threshold,
        pipeline_mode=args.pipeline,
    )

    if args.json:
        summary = demo.run_full_cycle()
        # Remove prometheus_text from JSON output (too verbose)
        if "metrics" in summary:
            summary["metrics"].pop("prometheus_text", None)
        json.dump(summary, sys.stdout, indent=2, default=str)
        print()
        return 0

    # ── Interactive demo ──────────────────────────────────────────────
    _header("Nephoran M7 Closed-Loop Demo")
    _kv("Mode", "DRY-RUN" if dry_run else "LIVE")
    _kv("Load factor", f"{args.load:.2f}")
    _kv("Cells", f"{args.cells} ({', '.join(c.cell_id for c in cells)})")
    _kv("PRB threshold", f"{args.threshold:.0f}%")
    _kv("Pipeline mode", "ON" if args.pipeline else "OFF")

    if args.step:
        _wait_enter()

    # Phase 1: Observe
    _phase("1. OBSERVE — E2 KPM Metrics Collection")
    print(f"  Simulating E2SM-KPM v3.0 metrics (load_factor={args.load})...")
    t0 = time.monotonic()
    obs = demo.observe()
    t_obs = time.monotonic() - t0

    for cell_id, summary in obs["cell_summaries"].items():
        print(f"\n  {BOLD}Cell {cell_id}{RESET} (gNB: {summary['gnb_id']})")
        prb = summary["avg_prb_usage"]
        color = RED if prb > args.threshold else YELLOW if prb > 60 else GREEN
        _kv("PRB usage (avg)", f"{color}{prb:.1f}%{RESET}", indent=4)
        _kv("Active UEs", summary["total_ues"], indent=4)
        _kv("DL throughput", f"{summary.get('total_dl_throughput', 0):.1f} Mbps", indent=4)

    _ok(f"Collected {obs['snapshot_count']} KPM snapshots in {t_obs:.3f}s")

    if args.step:
        _wait_enter()

    # Phase 2: Analyze
    _phase("2. ANALYZE — Scaling Recommendations")
    t0 = time.monotonic()
    analysis = demo.analyze()
    t_ana = time.monotonic() - t0

    for rec in analysis["recommendations"]:
        action = rec["action"]
        comp = rec["component"]
        prb = rec.get("prb_usage_pct", 0)
        if action == "none":
            _kv(f"{comp}", f"no action needed (PRB {prb:.1f}%)")
        elif action == "scale_out":
            _warn(f"{comp}: {YELLOW}SCALE OUT{RESET} to {rec['recommended_replicas']} replicas "
                  f"(PRB {prb:.1f}% > {args.threshold}%)")
        elif action == "scale_in":
            _kv(f"{comp}", f"scale in to {rec['recommended_replicas']} replicas")

    _ok(f"Analysis complete in {t_ana:.3f}s")

    if args.step:
        _wait_enter()

    # Phase 3: Act
    _phase("3. ACT — Intent Creation")
    t0 = time.monotonic()
    act = demo.act()
    t_act = time.monotonic() - t0

    if not act["intents"]:
        _kv("Result", "No actions needed — system is within thresholds")
    else:
        for intent in act["intents"]:
            mode = f"{DIM}[dry-run]{RESET}" if intent.get("dry_run") else "[LIVE]"
            print(f"\n  {mode} {BOLD}{intent.get('action', 'unknown')}{RESET}")
            _kv("Expression", intent.get("expression", ""), indent=4)
            _kv("Component", intent.get("component", ""), indent=4)
            _kv("Replicas", intent.get("replicas", ""), indent=4)

    _ok(f"Created {len(act['intents'])} intent(s) in {t_act:.3f}s")

    # Phase 4: Pipeline (optional)
    if args.pipeline and act["intents"]:
        if args.step:
            _wait_enter()
        _phase("4. PIPELINE — TMF921 Async Processing")
        print("  Running intents through intentd v2 pipeline (dry-run)...")
        # Pipeline was already run in run_full_cycle if we used that,
        # but in step mode we ran phases individually
        t0 = time.monotonic()
        pipeline_results = demo.run_pipeline(act["intents"])
        t_pipe = time.monotonic() - t0

        for pr in pipeline_results:
            state = pr.get("state", "unknown")
            color = GREEN if state == "completed" else RED
            _kv(f"Intent {pr.get('id', '?')[:8]}...",
                f"{color}{state}{RESET}", indent=4)

        _ok(f"Pipeline processed {len(pipeline_results)} intent(s) in {t_pipe:.3f}s")

    if args.step:
        _wait_enter()

    # Metrics summary
    _phase("5. METRICS — Prometheus Counters")
    snapshot = demo.get_metrics_snapshot()
    _kv("Active intents", int(snapshot["active_intents"]))
    _kv("Intents created (total)", int(snapshot["intents_created"]))
    _kv("Intents completed (total)", int(snapshot["intents_completed"]))
    _kv("Scale actions (total)", int(snapshot["scale_actions"]))

    # Prometheus text preview
    text = snapshot["prometheus_text"]
    lines = [l for l in text.splitlines() if l and not l.startswith("#")]
    if lines:
        print(f"\n  {DIM}Prometheus metrics sample:{RESET}")
        for line in lines[:10]:
            print(f"    {DIM}{line}{RESET}")
        if len(lines) > 10:
            print(f"    {DIM}... ({len(lines) - 10} more lines){RESET}")

    # Final summary
    _header("Demo Complete")
    total_time = t_obs + t_ana + t_act
    _ok(f"Total duration: {total_time:.3f}s")
    _ok(f"Cells observed: {len(obs['cell_summaries'])}")
    _ok(f"Recommendations: {len(analysis['recommendations'])}")
    _ok(f"Intents created: {len(act['intents'])}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
