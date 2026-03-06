import { useCallback, useState } from "react";
import { listIntents, deleteIntent, type IntentRecord } from "../api/client";
import { usePolling } from "../hooks/usePolling";

const STATE_STYLE: Record<string, string> = {
  acknowledged: "bg-gray-700/50 text-gray-300",
  planning: "bg-blue-900/40 text-blue-300",
  validating: "bg-blue-900/40 text-blue-300",
  generating: "bg-blue-900/40 text-blue-300",
  executing: "bg-cyan-900/40 text-cyan-300",
  proposed: "bg-purple-900/40 text-purple-300",
  applied: "bg-indigo-900/40 text-indigo-300",
  completed: "bg-green-900/40 text-green-300",
  failed: "bg-red-900/40 text-red-300",
  cancelled: "bg-gray-700/50 text-gray-400",
};

const SOURCE_LABEL: Record<string, string> = {
  web: "Web",
  cli: "CLI",
  tmf921: "API",
  closedloop: "Auto",
};

export default function IntentTable() {
  const fetcher = useCallback(() => listIntents(), []);
  const { data: intents, error, loading } = usePolling(fetcher, 3000);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold text-gray-100">Intent History</h1>
          <p className="text-sm text-gray-500 mt-0.5">
            All intents from Web, CLI, API, and closed-loop automation.
          </p>
        </div>
        {intents && intents.length > 0 && (
          <span className="text-xs text-gray-500 bg-gray-800 px-2.5 py-1 rounded-md">
            {intents.length} intents
          </span>
        )}
      </div>

      <div className="rounded-lg border border-gray-800 bg-gray-900 overflow-hidden">
        {loading && !intents && (
          <div className="p-8 text-center text-sm text-gray-500">Loading...</div>
        )}
        {error && (
          <div className="p-4 text-sm text-red-400">{error}</div>
        )}

        {intents && intents.length === 0 && (
          <div className="p-8 text-center text-sm text-gray-500">No intents yet.</div>
        )}

        {intents && intents.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-gray-800 text-gray-500 uppercase tracking-wider">
                  <th className="text-left px-4 py-3 font-medium">Intent ID</th>
                  <th className="text-left px-4 py-3 font-medium">Expression</th>
                  <th className="text-left px-4 py-3 font-medium">Source</th>
                  <th className="text-left px-4 py-3 font-medium">State</th>
                  <th className="text-left px-4 py-3 font-medium">Porch</th>
                  <th className="text-left px-4 py-3 font-medium">Git</th>
                  <th className="text-right px-4 py-3 font-medium">Updated</th>
                  <th className="text-center px-4 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800/50">
                {intents.map((intent) => (
                  <IntentRow key={intent.id} intent={intent} onRefresh={fetcher} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

function IntentRow({ intent, onRefresh }: { intent: IntentRecord; onRefresh: () => void }) {
  const [cancelling, setCancelling] = useState(false);

  const ts = intent.updated_at
    ? new Date(intent.updated_at).toLocaleTimeString()
    : "";

  const actions = Array.isArray(intent.plan?.actions)
    ? (intent.plan!.actions as Array<Record<string, unknown>>)
    : [];

  const handleCancel = async () => {
    setCancelling(true);
    try {
      await deleteIntent(intent.id);
      onRefresh();
    } catch {
      // Error will be visible on next poll
    } finally {
      setCancelling(false);
    }
  };

  return (
    <tr className="hover:bg-gray-800/30 transition-colors">
      <td className="px-4 py-3 font-mono text-gray-300 whitespace-nowrap">{intent.id}</td>
      <td className="px-4 py-3 text-gray-400 max-w-xs truncate">
        {intent.expression}
        {actions.length > 0 && (
          <div className="flex gap-1 mt-1">
            {actions.slice(0, 3).map((a, i) => (
              <span key={i} className="inline-flex items-center gap-0.5 rounded bg-gray-800 px-1.5 py-0.5 text-[10px]">
                <span className="text-blue-400">{String(a.kind)}</span>
                <span className="text-gray-400">{String(a.component)}</span>
                {a.replicas != null && <span className="text-yellow-400">x{Number(a.replicas)}</span>}
              </span>
            ))}
            {actions.length > 3 && (
              <span className="text-gray-600 text-[10px]">+{actions.length - 3}</span>
            )}
          </div>
        )}
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <span className="px-1.5 py-0.5 rounded bg-gray-800 text-gray-400 text-[10px] font-medium">
          {SOURCE_LABEL[intent.source] ?? intent.source}
        </span>
      </td>
      <td className="px-4 py-3 whitespace-nowrap">
        <span className={`px-2 py-0.5 rounded text-[10px] font-medium ${
          STATE_STYLE[intent.state] ?? "bg-gray-700/50 text-gray-400"
        }`}>
          {intent.state}
        </span>
      </td>
      <td className="px-4 py-3 text-gray-500 whitespace-nowrap">
        {intent.porch && !intent.porch.error
          ? <span className="text-purple-400">{intent.porch.lifecycle}</span>
          : intent.porch?.error
            ? <span className="text-red-400">Error</span>
            : "-"}
      </td>
      <td className="px-4 py-3 text-gray-500 whitespace-nowrap">
        {intent.git?.pr
          ? <span className="text-green-400">PR #{intent.git.pr.number}</span>
          : intent.git?.branch
            ? <span className="text-gray-400">{intent.git.branch}</span>
            : "-"}
      </td>
      <td className="px-4 py-3 text-gray-500 text-right whitespace-nowrap">{ts}</td>
      <td className="px-4 py-3 text-center whitespace-nowrap">
        {intent.state === "acknowledged" ? (
          <button
            onClick={handleCancel}
            disabled={cancelling}
            className="px-2 py-1 rounded text-[10px] font-medium bg-red-900/30 text-red-400 hover:bg-red-900/50 disabled:opacity-50 transition-colors"
          >
            {cancelling ? "Cancelling..." : "Cancel"}
          </button>
        ) : (
          <span className="text-gray-700">-</span>
        )}
      </td>
    </tr>
  );
}
