import { useCallback } from "react";
import { getScaleStatus, type ScaleComponent } from "../api/client";
import { usePolling } from "../hooks/usePolling";

const DOMAIN_COLOR: Record<string, string> = {
  ran: "border-orange-600",
  core: "border-blue-600",
  ric: "border-purple-600",
  sim: "border-gray-600",
};

export default function ScaleDashboard() {
  const fetcher = useCallback(() => getScaleStatus(), []);
  const { data, error, loading } = usePolling(fetcher, 5000);

  const components = data?.components ?? [];

  // Group by domain
  const byDomain = components.reduce<Record<string, ScaleComponent[]>>((acc, c) => {
    (acc[c.domain] ??= []).push(c);
    return acc;
  }, {});

  return (
    <section className="rounded-lg border border-gray-800 bg-gray-900 p-5">
      <h2 className="text-lg font-semibold mb-3">
        CNF Scale Status
        {data?.timestamp && (
          <span className="ml-2 text-xs text-gray-500 font-normal">
            {new Date(data.timestamp).toLocaleTimeString()}
          </span>
        )}
      </h2>

      {loading && !data && <p className="text-xs text-gray-500">Loading...</p>}
      {error && <p className="text-xs text-red-400">{error}</p>}

      {components.length === 0 && !loading && (
        <p className="text-xs text-gray-500">
          No deployments found. {data?.error && `(${data.error})`}
        </p>
      )}

      <div className="space-y-4">
        {Object.entries(byDomain).map(([domain, comps]) => (
          <div key={domain}>
            <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-1.5">
              {domain}
            </h3>
            <div className="grid gap-1.5">
              {comps.map((c) => (
                <ComponentRow key={c.name} component={c} domain={domain} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

function ComponentRow({ component: c, domain }: { component: ScaleComponent; domain: string }) {
  const pct = c.replicas > 0 ? Math.round((c.ready / c.replicas) * 100) : 0;
  const healthy = c.ready === c.replicas && c.replicas > 0;

  return (
    <div className={`rounded bg-gray-800 px-3 py-2 text-xs font-mono border-l-2 ${DOMAIN_COLOR[domain] ?? "border-gray-700"}`}>
      <div className="flex items-center justify-between">
        <div>
          <span className="text-gray-300">{c.name}</span>
          <span className="ml-2 text-gray-600">{c.namespace}</span>
        </div>
        <div className="flex items-center gap-3">
          <div className="w-16 bg-gray-700 rounded-full h-1.5">
            <div
              className={`h-1.5 rounded-full ${healthy ? "bg-green-500" : "bg-yellow-500"}`}
              style={{ width: `${pct}%` }}
            />
          </div>
          <span className={healthy ? "text-green-400" : "text-yellow-400"}>
            {c.ready}/{c.replicas}
          </span>
        </div>
      </div>
      {c.intentId && (
        <div className="mt-1 flex items-center gap-1.5">
          <span className="px-1.5 py-0.5 rounded bg-blue-900/40 text-blue-300 text-[10px]"
                title={`Created by intent ${c.intentId}`}>
            {c.intentId}
          </span>
          {c.slice && (
            <span className="px-1 py-0.5 rounded bg-gray-700/60 text-gray-400 text-[10px]">
              {c.slice}
            </span>
          )}
          {c.site && (
            <span className="px-1 py-0.5 rounded bg-gray-700/60 text-gray-400 text-[10px]">
              {c.site}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
