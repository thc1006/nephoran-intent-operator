import { useCallback } from "react";
import { listIntents, type IntentRecord } from "../api/client";
import { usePolling } from "../hooks/usePolling";

const STATE_COLOR: Record<string, string> = {
  acknowledged: "text-gray-400",
  planning: "text-blue-400",
  validating: "text-blue-400",
  generating: "text-blue-400",
  executing: "text-cyan-400",
  proposed: "text-purple-400",
  applied: "text-indigo-400",
  completed: "text-green-400",
  failed: "text-red-400",
  cancelled: "text-gray-500",
};

const SOURCE_LABEL: Record<string, string> = {
  web: "Web",
  cli: "CLI",
  tmf921: "API",
  closedloop: "Auto",
};

export default function IntentHistory() {
  const fetcher = useCallback(() => listIntents(), []);
  const { data: intents, error, loading } = usePolling(fetcher, 3000);

  return (
    <section className="rounded-lg border border-gray-800 bg-gray-900 p-5">
      <h2 className="text-lg font-semibold mb-3">
        Intent History
        {intents && intents.length > 0 && (
          <span className="ml-2 text-xs text-gray-500">({intents.length})</span>
        )}
      </h2>

      {loading && !intents && <p className="text-xs text-gray-500">Loading...</p>}
      {error && <p className="text-xs text-red-400">{error}</p>}
      {intents && intents.length === 0 && <p className="text-xs text-gray-500">No intents yet.</p>}

      <div className="space-y-2 max-h-96 overflow-y-auto">
        {intents?.map((intent) => (
          <IntentRow key={intent.id} intent={intent} />
        ))}
      </div>
    </section>
  );
}

function IntentRow({ intent }: { intent: IntentRecord }) {
  const ts = intent.updated_at
    ? new Date(intent.updated_at).toLocaleTimeString()
    : "";
  const actions = Array.isArray(intent.plan?.actions)
    ? (intent.plan!.actions as Array<Record<string, unknown>>)
    : [];

  return (
    <div className="rounded bg-gray-800 px-3 py-2 text-xs font-mono">
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-2">
          <span className="text-gray-300">{intent.id}</span>
          <span className="px-1.5 py-0.5 rounded bg-gray-700/60 text-gray-400">
            {SOURCE_LABEL[intent.source] ?? intent.source}
          </span>
        </div>
        <span className={STATE_COLOR[intent.state] ?? "text-gray-400"}>
          {intent.state}
        </span>
      </div>
      <div className="text-gray-500 mt-0.5 truncate">
        {intent.expression}
        <span className="ml-2 text-gray-600">{ts}</span>
      </div>
      {actions.length > 0 && (
        <div className="mt-1 flex flex-wrap gap-1.5">
          {actions.map((a, i) => (
            <span key={i} className="inline-flex items-center gap-1 rounded bg-gray-700/60 px-1.5 py-0.5">
              <span className="text-blue-400">{String(a.kind)}</span>
              <span className="text-gray-300">{String(a.component)}</span>
              {a.replicas != null && <span className="text-yellow-400">x{Number(a.replicas)}</span>}
            </span>
          ))}
        </div>
      )}
      {intent.porch && !intent.porch.error && (
        <div className="mt-1 flex items-center gap-1.5">
          <span className="px-1.5 py-0.5 rounded bg-purple-900/40 text-purple-300">
            Porch: {intent.porch.lifecycle}
          </span>
          {intent.porch.files != null && (
            <span className="text-gray-500">{intent.porch.files} files</span>
          )}
        </div>
      )}
      {intent.git?.pr && (
        <div className="mt-1">
          <span className="px-1.5 py-0.5 rounded bg-green-900/40 text-green-300">
            PR #{intent.git.pr.number}
          </span>
          <span className="ml-1.5 text-gray-500">{intent.git.branch}</span>
        </div>
      )}
    </div>
  );
}
