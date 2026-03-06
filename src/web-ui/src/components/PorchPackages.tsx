import { useCallback } from "react";
import { listPorchPackages, type PorchPackage } from "../api/client";
import { usePolling } from "../hooks/usePolling";

const LIFECYCLE_COLOR: Record<string, string> = {
  Draft: "bg-yellow-900/50 text-yellow-300",
  Proposed: "bg-purple-900/50 text-purple-300",
  Published: "bg-green-900/50 text-green-300",
};

export default function PorchPackages() {
  const fetcher = useCallback(() => listPorchPackages(), []);
  const { data: pkgs, error, loading } = usePolling(fetcher, 10000);

  return (
    <section className="rounded-lg border border-gray-800 bg-gray-900 p-5">
      <h2 className="text-lg font-semibold mb-3">
        Porch Packages
        {pkgs && <span className="ml-2 text-xs text-gray-500">({pkgs.length})</span>}
      </h2>

      {loading && !pkgs && <p className="text-xs text-gray-500">Loading...</p>}
      {error && <p className="text-xs text-red-400">{error}</p>}

      <div className="space-y-1.5 max-h-72 overflow-y-auto">
        {pkgs?.map((p) => (
          <div key={p.name} className="flex items-center justify-between rounded bg-gray-800 px-3 py-2 text-xs font-mono">
            <div>
              <span className="text-gray-300">{p.package}</span>
              <span className="ml-2 text-gray-600">{p.workspace}</span>
            </div>
            <span className={`px-2 py-0.5 rounded text-xs ${LIFECYCLE_COLOR[p.lifecycle] ?? "bg-gray-700 text-gray-400"}`}>
              {p.lifecycle}
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
