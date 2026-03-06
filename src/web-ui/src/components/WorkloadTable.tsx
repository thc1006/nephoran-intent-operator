import { useCallback } from "react";
import { getScaleStatus, type ScaleComponent } from "../api/client";
import { usePolling } from "../hooks/usePolling";

const DOMAIN_STYLE: Record<string, string> = {
  ran: "bg-orange-900/30 text-orange-300",
  core: "bg-blue-900/30 text-blue-300",
  ric: "bg-purple-900/30 text-purple-300",
  sim: "bg-gray-700/40 text-gray-400",
  obs: "bg-cyan-900/30 text-cyan-300",
};

export default function WorkloadTable() {
  const fetcher = useCallback(() => getScaleStatus(), []);
  const { data, error, loading } = usePolling(fetcher, 5000);

  const components = data?.components ?? [];

  const total = components.length;
  const healthy = components.filter((c) => c.ready === c.replicas && c.replicas > 0).length;
  const totalReplicas = components.reduce((s, c) => s + c.replicas, 0);
  const readyReplicas = components.reduce((s, c) => s + c.ready, 0);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-lg font-semibold text-gray-100">Workloads</h1>
        <p className="text-sm text-gray-500 mt-0.5">
          Live deployment status across all managed namespaces. Intent tracking via <code className="text-gray-400">oran.ai/intent-id</code> labels.
        </p>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <SummaryCard label="Deployments" value={String(total)} />
        <SummaryCard label="Healthy" value={`${healthy}/${total}`} variant={healthy === total ? "ok" : "warn"} />
        <SummaryCard label="Total Replicas" value={String(totalReplicas)} />
        <SummaryCard label="Ready" value={`${readyReplicas}/${totalReplicas}`} variant={readyReplicas === totalReplicas ? "ok" : "warn"} />
      </div>

      {/* Table */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 overflow-hidden">
        {loading && !data && (
          <div className="p-8 text-center text-sm text-gray-500">Loading...</div>
        )}
        {error && <div className="p-4 text-sm text-red-400">{error}</div>}

        {components.length === 0 && !loading && (
          <div className="p-8 text-center text-sm text-gray-500">
            No deployments found. {data?.error && `(${data.error})`}
          </div>
        )}

        {components.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-gray-800 text-gray-500 uppercase tracking-wider">
                  <th className="text-left px-4 py-3 font-medium">Name</th>
                  <th className="text-left px-4 py-3 font-medium">Namespace</th>
                  <th className="text-left px-4 py-3 font-medium">Domain</th>
                  <th className="text-center px-4 py-3 font-medium">Ready</th>
                  <th className="text-left px-4 py-3 font-medium">Health</th>
                  <th className="text-left px-4 py-3 font-medium">Intent</th>
                  <th className="text-left px-4 py-3 font-medium">Slice / Site</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800/50">
                {components.map((c) => (
                  <DeploymentRow key={`${c.namespace}/${c.name}`} c={c} />
                ))}
              </tbody>
            </table>
          </div>
        )}

        {data?.timestamp && (
          <div className="border-t border-gray-800 px-4 py-2 text-[10px] text-gray-600">
            Last updated: {new Date(data.timestamp).toLocaleTimeString()}
          </div>
        )}
      </div>
    </div>
  );
}

function DeploymentRow({ c }: { c: ScaleComponent }) {
  const healthy = c.ready === c.replicas && c.replicas > 0;
  const pct = c.replicas > 0 ? Math.round((c.ready / c.replicas) * 100) : 0;

  return (
    <tr className="hover:bg-gray-800/30 transition-colors">
      <td className="px-4 py-3 font-mono text-gray-300 whitespace-nowrap">{c.name}</td>
      <td className="px-4 py-3 text-gray-500 whitespace-nowrap">{c.namespace}</td>
      <td className="px-4 py-3 whitespace-nowrap">
        <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${
          DOMAIN_STYLE[c.domain] ?? "bg-gray-700/40 text-gray-400"
        }`}>
          {c.domain}
        </span>
      </td>
      <td className="px-4 py-3 text-center whitespace-nowrap">
        <span className={healthy ? "text-green-400" : "text-yellow-400"}>
          {c.ready}/{c.replicas}
        </span>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <div className="flex items-center gap-2">
          <div className="w-20 bg-gray-700 rounded-full h-1.5">
            <div
              className={`h-1.5 rounded-full transition-all ${healthy ? "bg-green-500" : pct > 0 ? "bg-yellow-500" : "bg-red-500"}`}
              style={{ width: `${pct}%` }}
            />
          </div>
          <span className="text-gray-500 text-[10px] w-8">{pct}%</span>
        </div>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        {c.intentId ? (
          <span className="px-1.5 py-0.5 rounded bg-blue-900/30 text-blue-300 text-[10px] font-mono"
                title={`Created by intent ${c.intentId}`}>
            {c.intentId}
          </span>
        ) : (
          <span className="text-gray-600">-</span>
        )}
      </td>
      <td className="px-4 py-3 whitespace-nowrap text-gray-500">
        {c.slice || c.site ? (
          <span className="text-[10px]">
            {c.slice && <span className="text-gray-400">{c.slice}</span>}
            {c.slice && c.site && <span className="text-gray-600"> / </span>}
            {c.site && <span className="text-gray-400">{c.site}</span>}
          </span>
        ) : "-"}
      </td>
    </tr>
  );
}

function SummaryCard({ label, value, variant }: { label: string; value: string; variant?: "ok" | "warn" }) {
  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 px-4 py-3">
      <div className="text-[10px] text-gray-500 uppercase tracking-wider">{label}</div>
      <div className={`text-xl font-semibold mt-1 ${
        variant === "ok" ? "text-green-400" : variant === "warn" ? "text-yellow-400" : "text-gray-200"
      }`}>
        {value}
      </div>
    </div>
  );
}
