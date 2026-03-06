import { useCallback } from "react";
import { listPorchPackages, type PorchPackage } from "../api/client";
import { usePolling } from "../hooks/usePolling";

const LIFECYCLE_STYLE: Record<string, string> = {
  Draft: "bg-yellow-900/40 text-yellow-300",
  Proposed: "bg-purple-900/40 text-purple-300",
  Published: "bg-green-900/40 text-green-300",
};

export default function PackageTable() {
  const fetcher = useCallback(() => listPorchPackages(), []);
  const { data: pkgs, error, loading } = usePolling(fetcher, 10000);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold text-gray-100">Porch Packages</h1>
          <p className="text-sm text-gray-500 mt-0.5">
            Nephio Porch PackageRevisions in the <code className="text-gray-400">nephoran-packages</code> repository.
          </p>
        </div>
        {pkgs && pkgs.length > 0 && (
          <span className="text-xs text-gray-500 bg-gray-800 px-2.5 py-1 rounded-md">
            {pkgs.length} packages
          </span>
        )}
      </div>

      <div className="rounded-lg border border-gray-800 bg-gray-900 overflow-hidden">
        {loading && !pkgs && (
          <div className="p-8 text-center text-sm text-gray-500">Loading...</div>
        )}
        {error && <div className="p-4 text-sm text-red-400">{error}</div>}

        {pkgs && pkgs.length === 0 && (
          <div className="p-8 text-center text-sm text-gray-500">No packages found.</div>
        )}

        {pkgs && pkgs.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-gray-800 text-gray-500 uppercase tracking-wider">
                  <th className="text-left px-4 py-3 font-medium">Package</th>
                  <th className="text-left px-4 py-3 font-medium">Revision</th>
                  <th className="text-left px-4 py-3 font-medium">Workspace</th>
                  <th className="text-left px-4 py-3 font-medium">Repository</th>
                  <th className="text-left px-4 py-3 font-medium">Lifecycle</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800/50">
                {pkgs.map((p) => (
                  <tr key={p.name} className="hover:bg-gray-800/30 transition-colors">
                    <td className="px-4 py-3 font-mono text-gray-300">{p.package}</td>
                    <td className="px-4 py-3 font-mono text-gray-500">{p.name}</td>
                    <td className="px-4 py-3 text-gray-400">{p.workspace}</td>
                    <td className="px-4 py-3 text-gray-500">{p.repository}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-medium ${
                        LIFECYCLE_STYLE[p.lifecycle] ?? "bg-gray-700/40 text-gray-400"
                      }`}>
                        {p.lifecycle}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
