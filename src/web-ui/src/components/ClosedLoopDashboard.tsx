import { useCallback } from "react";
import {
  getClosedLoopMetrics,
  type ClosedLoopMetrics,
  type CellPrbUsage,
  type ScaleActionMetric,
  type PipelineStageMetric,
  type PorchLifecycleMetric,
} from "../api/client";
import { usePolling } from "../hooks/usePolling";

/** PRB threshold matching Grafana panel 6 (80% triggers scale-out). */
const PRB_THRESHOLD = 80;

/** Pipeline stage latency thresholds (seconds) — green/orange/red. */
const LATENCY_OK = 5;
const LATENCY_WARN = 15;

export default function ClosedLoopDashboard() {
  const fetcher = useCallback(() => getClosedLoopMetrics(), []);
  const { data, error, loading } = usePolling<ClosedLoopMetrics>(fetcher, 5000);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-lg font-semibold text-gray-100">Closed Loop</h1>
        <p className="text-sm text-gray-500 mt-0.5">
          M7 closed-loop metrics: E2 KPM observe, analyze, scale — real-time Prometheus data.
        </p>
      </div>

      {loading && !data && (
        <div className="p-8 text-center text-sm text-gray-500">Loading metrics...</div>
      )}
      {error && <div className="p-4 text-sm text-red-400 rounded-lg border border-red-900 bg-red-950/30">{error}</div>}

      {data && (
        <>
          {/* Row 1: Overview */}
          <OverviewRow overview={data.overview} />

          {/* Row 2: E2 KPM Metrics */}
          <E2KpmRow cells={data.e2kpm.cells} />

          {/* Row 3: Pipeline Performance */}
          <PipelineRow stages={data.pipeline_stages} />

          {/* Row 4: GitOps & Porch */}
          <GitOpsRow
            scaleActions={data.scale_actions}
            porchLifecycle={data.porch_lifecycle}
            gitCommits={data.overview.git_commits}
          />

          {/* Footer timestamp */}
          <div className="text-[10px] text-gray-600">
            Last updated: {new Date(data.timestamp).toLocaleTimeString()}
          </div>
        </>
      )}
    </div>
  );
}

// ── Row 1: Overview ──────────────────────────────────────────────────

function OverviewRow({ overview }: { overview: ClosedLoopMetrics["overview"] }) {
  const successRate =
    overview.completed + overview.failed > 0
      ? Math.round((overview.completed / (overview.completed + overview.failed)) * 100)
      : 0;

  return (
    <section>
      <h2 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Overview</h2>
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-3">
        <StatCard label="Active Intents" value={overview.active_intents} variant={overview.active_intents > 0 ? "active" : undefined} />
        <StatCard label="Total Created" value={overview.total_created} />
        <StatCard label="Completed" value={overview.completed} variant="ok" />
        <StatCard label="Failed" value={overview.failed} variant={overview.failed > 0 ? "error" : undefined} />
        <StatCard label="Success Rate" value={`${successRate}%`} variant={successRate >= 90 ? "ok" : successRate > 0 ? "warn" : undefined} />
      </div>
    </section>
  );
}

// ── Row 2: E2 KPM ───────────────────────────────────────────────────

function E2KpmRow({ cells }: { cells: CellPrbUsage[] }) {
  return (
    <section>
      <h2 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">E2 KPM Metrics</h2>
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
        {cells.length === 0 ? (
          <div className="text-sm text-gray-500 text-center py-4">No cell metrics available</div>
        ) : (
          <div className="space-y-3">
            {cells.map((cell) => (
              <CellPrbBar key={`${cell.cell_id}-${cell.gnb_id}`} cell={cell} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}

function CellPrbBar({ cell }: { cell: CellPrbUsage }) {
  const pct = Math.min(cell.prb_usage_percent, 100);
  const color =
    pct >= 85 ? "bg-red-500" : pct >= 70 ? "bg-orange-500" : "bg-green-500";
  const textColor =
    pct >= 85 ? "text-red-400" : pct >= 70 ? "text-orange-400" : "text-green-400";

  return (
    <div className="flex items-center gap-3">
      <div className="w-28 flex-shrink-0">
        <div className="text-xs text-gray-300 font-mono">{cell.cell_id}</div>
        <div className="text-[10px] text-gray-600">{cell.gnb_id}</div>
      </div>
      <div className="flex-1 relative">
        <div className="w-full bg-gray-700 rounded-full h-2.5">
          <div
            className={`h-2.5 rounded-full transition-all ${color}`}
            style={{ width: `${pct}%` }}
          />
        </div>
        {/* 80% threshold marker */}
        <div
          className="absolute top-0 h-2.5 border-r-2 border-dashed border-yellow-500/50"
          style={{ left: `${PRB_THRESHOLD}%` }}
          title={`Threshold: ${PRB_THRESHOLD}%`}
        />
      </div>
      <span className={`text-xs font-mono w-14 text-right ${textColor}`}>
        {pct.toFixed(1)}%
      </span>
    </div>
  );
}

// ── Row 3: Pipeline Performance ─────────────────────────────────────

function PipelineRow({ stages }: { stages: PipelineStageMetric[] }) {
  const maxP95 = Math.max(...stages.map((s) => s.p95_seconds ?? 0), 1);

  return (
    <section>
      <h2 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Pipeline Performance</h2>
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
        {stages.length === 0 ? (
          <div className="text-sm text-gray-500 text-center py-4">No pipeline data yet</div>
        ) : (
          <div className="space-y-3">
            {stages.map((stage) => (
              <StageBar key={stage.stage} stage={stage} maxP95={maxP95} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}

function StageBar({ stage, maxP95 }: { stage: PipelineStageMetric; maxP95: number }) {
  const p95 = stage.p95_seconds ?? 0;
  const mean = stage.count > 0 ? stage.sum_seconds / stage.count : 0;
  const barPct = maxP95 > 0 ? Math.min((p95 / maxP95) * 100, 100) : 0;
  const color =
    p95 >= LATENCY_WARN ? "bg-red-500" : p95 >= LATENCY_OK ? "bg-orange-500" : "bg-blue-500";

  return (
    <div className="flex items-center gap-3">
      <div className="w-24 flex-shrink-0 text-xs text-gray-300 font-mono">{stage.stage}</div>
      <div className="flex-1">
        <div className="w-full bg-gray-700 rounded-full h-2">
          <div
            className={`h-2 rounded-full transition-all ${color}`}
            style={{ width: `${barPct}%` }}
          />
        </div>
      </div>
      <div className="w-32 flex-shrink-0 text-right text-[10px] text-gray-500 font-mono">
        <span className="text-gray-300">p95 {p95.toFixed(2)}s</span>
        {" / "}
        <span>avg {mean.toFixed(2)}s</span>
        {" / "}
        <span>n={stage.count}</span>
      </div>
    </div>
  );
}

// ── Row 4: GitOps & Porch ───────────────────────────────────────────

function GitOpsRow({
  scaleActions,
  porchLifecycle,
  gitCommits,
}: {
  scaleActions: ScaleActionMetric[];
  porchLifecycle: PorchLifecycleMetric[];
  gitCommits: number;
}) {
  return (
    <section>
      <h2 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">GitOps & Porch</h2>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-3">
        {/* Scale Actions */}
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-2">Scale Actions</div>
          {scaleActions.length === 0 ? (
            <div className="text-sm text-gray-500">None</div>
          ) : (
            <div className="space-y-1.5">
              {scaleActions.map((sa) => (
                <div key={`${sa.component}-${sa.action}`} className="flex items-center justify-between text-xs">
                  <span className="font-mono text-gray-300">{sa.component}</span>
                  <div className="flex items-center gap-2">
                    <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${
                      sa.action === "scale_out" ? "bg-green-900/30 text-green-300" : "bg-orange-900/30 text-orange-300"
                    }`}>
                      {sa.action}
                    </span>
                    <span className="text-gray-400 font-mono w-6 text-right">{sa.count}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Porch Lifecycle */}
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-2">Porch Lifecycle</div>
          {porchLifecycle.length === 0 ? (
            <div className="text-sm text-gray-500">None</div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {porchLifecycle.map((pl) => (
                <span
                  key={pl.lifecycle}
                  className={`px-2 py-1 rounded text-xs font-medium ${
                    pl.lifecycle === "Published"
                      ? "bg-green-900/30 text-green-300"
                      : pl.lifecycle === "Proposed"
                        ? "bg-purple-900/30 text-purple-300"
                        : "bg-yellow-900/30 text-yellow-300"
                  }`}
                >
                  {pl.lifecycle}: {pl.count}
                </span>
              ))}
            </div>
          )}
        </div>

        {/* Git Commits */}
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-2">Git Commits</div>
          <div className="text-2xl font-semibold text-gray-200">{gitCommits}</div>
          <div className="text-[10px] text-gray-600 mt-1">Commits pushed by pipeline</div>
        </div>
      </div>
    </section>
  );
}

// ── Shared stat card ────────────────────────────────────────────────

function StatCard({
  label,
  value,
  variant,
}: {
  label: string;
  value: number | string;
  variant?: "ok" | "warn" | "error" | "active";
}) {
  const colorMap: Record<string, string> = {
    ok: "text-green-400",
    warn: "text-yellow-400",
    error: "text-red-400",
    active: "text-blue-400",
  };

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 px-4 py-3">
      <div className="text-[10px] text-gray-500 uppercase tracking-wider">{label}</div>
      <div className={`text-xl font-semibold mt-1 ${variant ? colorMap[variant] : "text-gray-200"}`}>
        {value}
      </div>
    </div>
  );
}
