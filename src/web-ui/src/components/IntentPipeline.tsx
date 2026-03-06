import { useCallback, useEffect, useRef, useState } from "react";
import { Send, Cpu, TestTube } from "lucide-react";
import {
  createIntent,
  getIntent,
  isTerminal,
  isActive,
  stateIndex,
  PIPELINE_STATES,
  type IntentRecord,
} from "../api/client";

export default function IntentPipeline() {
  const [text, setText] = useState("");
  const [useLlm, setUseLlm] = useState(true);
  const [dryRun, setDryRun] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [intent, setIntent] = useState<IntentRecord | null>(null);
  const [error, setError] = useState<string | null>(null);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const startPolling = useCallback((intentId: string) => {
    if (pollingRef.current) clearInterval(pollingRef.current);
    pollingRef.current = setInterval(async () => {
      try {
        const updated = await getIntent(intentId);
        setIntent(updated);
        if (isTerminal(updated.state)) {
          if (pollingRef.current) clearInterval(pollingRef.current);
          pollingRef.current = null;
        }
      } catch {
        /* keep polling */
      }
    }, 800);
  }, []);

  useEffect(() => {
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, []);

  async function handleSubmit() {
    if (!text.trim() || submitting || (intent && isActive(intent.state))) return;
    setSubmitting(true);
    setIntent(null);
    setError(null);
    try {
      const rec = await createIntent(text.trim(), "web", useLlm, dryRun);
      setIntent(rec);
      startPolling(rec.id);
    } catch (e) {
      setError(String(e));
    } finally {
      setSubmitting(false);
    }
  }

  const busy = submitting || (intent !== null && isActive(intent.state));
  const currentIdx = intent ? stateIndex(intent.state) : -1;
  const failed = intent?.state === "failed";

  return (
    <div className="space-y-6">
      {/* Page title */}
      <div>
        <h1 className="text-lg font-semibold text-gray-100">Intent Pipeline</h1>
        <p className="text-sm text-gray-500 mt-0.5">
          Submit a natural-language intent. The async pipeline runs: Plan → Validate → Generate → Git → Porch.
        </p>
      </div>

      {/* Submit card */}
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-5">
        <textarea
          className="w-full rounded-md bg-gray-800 border border-gray-700 p-3 text-sm
                     placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500
                     resize-none font-mono"
          rows={3}
          placeholder="e.g. Deploy eMBB slice with 2 O-DU instances at edge01"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter" && e.metaKey) handleSubmit(); }}
          disabled={busy}
        />

        <div className="flex items-center gap-4 mt-3">
          <button
            className="inline-flex items-center gap-2 px-4 py-2 rounded-md font-medium text-sm
                       bg-blue-600 hover:bg-blue-500 disabled:bg-gray-800 disabled:text-gray-500
                       transition-colors"
            onClick={handleSubmit}
            disabled={busy || !text.trim()}
          >
            <Send size={14} />
            {submitting ? "Submitting..." : busy ? "Processing..." : "Submit Intent"}
          </button>

          <label className="flex items-center gap-1.5 text-xs text-gray-400 select-none">
            <input type="checkbox" checked={useLlm} onChange={(e) => setUseLlm(e.target.checked)}
              className="rounded border-gray-600 bg-gray-800 text-blue-600 focus:ring-blue-500/30" />
            <Cpu size={12} /> LLM
          </label>
          <label className="flex items-center gap-1.5 text-xs text-gray-400 select-none">
            <input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)}
              className="rounded border-gray-600 bg-gray-800 text-blue-600 focus:ring-blue-500/30" />
            <TestTube size={12} /> Dry run
          </label>
        </div>
      </div>

      {/* Pipeline stage progress */}
      {intent && (
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-gray-200">Pipeline Progress</h2>
            <span className={`text-xs font-medium px-2 py-0.5 rounded ${
              failed ? "bg-red-900/50 text-red-300"
                : intent.state === "completed" ? "bg-green-900/50 text-green-300"
                  : "bg-blue-900/50 text-blue-300"
            }`}>
              {intent.state}
            </span>
          </div>

          {/* Stage bar */}
          <div className="flex gap-1">
            {PIPELINE_STATES.map((stage, i) => {
              const done = currentIdx >= i;
              const isCurrent = currentIdx === i && isActive(intent.state);
              return (
                <div
                  key={stage}
                  className={`flex-1 text-center text-[11px] py-2 rounded-md font-medium transition-colors ${
                    failed && currentIdx < i
                      ? "bg-gray-800/50 text-gray-600"
                      : failed
                        ? "bg-red-900/40 text-red-400"
                        : done
                          ? "bg-green-900/40 text-green-400"
                          : isCurrent
                            ? "bg-blue-900/40 text-blue-300 animate-pulse"
                            : "bg-gray-800/50 text-gray-600"
                  }`}
                >
                  {stage}
                </div>
              );
            })}
          </div>

          {/* Summary grid */}
          <div className="mt-4 grid grid-cols-2 lg:grid-cols-4 gap-3">
            <InfoCell label="Intent ID" value={intent.id} />
            <InfoCell label="Source" value={intent.source} />
            <InfoCell label="Transitions" value={String(intent.history.length)} />
            <InfoCell label="Updated" value={intent.updated_at ? new Date(intent.updated_at).toLocaleTimeString() : "-"} />
          </div>

          {/* Git & Porch details */}
          {(intent.git || intent.porch) && (
            <div className="mt-3 grid grid-cols-2 gap-3">
              {intent.git?.pr && (
                <InfoCell label="Git PR" value={`#${intent.git.pr.number} (${intent.git.branch})`} />
              )}
              {intent.git && !intent.git.pr && intent.git.branch && (
                <InfoCell label="Git Branch" value={intent.git.branch} />
              )}
              {intent.porch && !intent.porch.error && (
                <InfoCell label="Porch" value={`${intent.porch.lifecycle} (${intent.porch.files ?? 0} files)`} />
              )}
              {intent.porch?.error && (
                <InfoCell label="Porch" value={`Error: ${intent.porch.error}`} variant="error" />
              )}
            </div>
          )}

          {intent.errors && (
            <div className="mt-3 rounded-md bg-red-900/20 border border-red-800/50 p-3 text-xs text-red-300 font-mono">
              {intent.errors.join("\n")}
            </div>
          )}

          {/* Plan detail */}
          {intent.plan && <PlanDetail plan={intent.plan} />}
        </div>
      )}

      {error && (
        <div className="rounded-md bg-red-900/20 border border-red-800/50 p-4 text-sm text-red-300">
          {error}
        </div>
      )}
    </div>
  );
}

function InfoCell({ label, value, variant }: { label: string; value: string; variant?: "error" }) {
  return (
    <div className="rounded-md bg-gray-800/60 px-3 py-2">
      <div className="text-[10px] text-gray-500 uppercase tracking-wider">{label}</div>
      <div className={`text-xs font-mono mt-0.5 truncate ${variant === "error" ? "text-red-400" : "text-gray-200"}`}>
        {value}
      </div>
    </div>
  );
}

function PlanDetail({ plan }: { plan: Record<string, unknown> }) {
  const intentType = String(plan.intentType ?? "");
  const actions = Array.isArray(plan.actions)
    ? (plan.actions as Array<Record<string, unknown>>)
    : [];

  return (
    <details className="mt-4">
      <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-300 select-none">
        IntentPlan (LLM output)
      </summary>
      <div className="mt-2 rounded-md bg-gray-800/60 p-3 text-xs font-mono space-y-1">
        <div className="text-gray-500">intentType: <span className="text-gray-300">{intentType}</span></div>
        {actions.map((a, i) => (
          <div key={i} className="flex gap-3 text-gray-300">
            <span className="text-blue-400">{String(a.kind)}</span>
            <span>{String(a.component)}</span>
            {a.replicas != null && <span className="text-yellow-400">x{Number(a.replicas)}</span>}
          </div>
        ))}
      </div>
      <details className="mt-1 text-xs">
        <summary className="text-gray-600 cursor-pointer hover:text-gray-400">raw JSON</summary>
        <pre className="mt-1 rounded-md bg-gray-800/60 p-3 text-gray-400 overflow-x-auto max-h-48 overflow-y-auto">
          {JSON.stringify(plan, null, 2)}
        </pre>
      </details>
    </details>
  );
}
