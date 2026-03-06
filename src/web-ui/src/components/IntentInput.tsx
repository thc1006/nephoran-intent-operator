import { useCallback, useEffect, useRef, useState } from "react";
import {
  createIntent,
  getIntent,
  isTerminal,
  isActive,
  stateIndex,
  PIPELINE_STATES,
  type IntentRecord,
} from "../api/client";

interface Props {
  onComplete?: (intent: IntentRecord) => void;
}

export default function IntentInput({ onComplete }: Props) {
  const [text, setText] = useState("");
  const [useLlm, setUseLlm] = useState(true);
  const [dryRun, setDryRun] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [intent, setIntent] = useState<IntentRecord | null>(null);
  const [error, setError] = useState<string | null>(null);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Poll intent state while pipeline is active
  const startPolling = useCallback((intentId: string) => {
    if (pollingRef.current) clearInterval(pollingRef.current);
    pollingRef.current = setInterval(async () => {
      try {
        const updated = await getIntent(intentId);
        setIntent(updated);
        if (isTerminal(updated.state)) {
          if (pollingRef.current) clearInterval(pollingRef.current);
          pollingRef.current = null;
          onComplete?.(updated);
        }
      } catch {
        // keep polling
      }
    }, 800);
  }, [onComplete]);

  useEffect(() => {
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, []);

  async function handleSubmit() {
    if (!text.trim() || submitting) return;
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

  const currentIdx = intent ? stateIndex(intent.state) : -1;
  const active = intent ? isActive(intent.state) : false;
  const failed = intent?.state === "failed";

  return (
    <section className="rounded-lg border border-gray-800 bg-gray-900 p-5">
      <h2 className="text-lg font-semibold mb-3">Intent Pipeline</h2>

      <textarea
        className="w-full rounded bg-gray-800 border border-gray-700 p-3 text-sm
                   placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500
                   resize-none"
        rows={3}
        placeholder="e.g. Deploy eMBB slice with 2 O-DU instances"
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => { if (e.key === "Enter" && e.metaKey) handleSubmit(); }}
        disabled={submitting || active}
      />

      <div className="flex items-center gap-4 mt-3">
        <button
          className="px-5 py-2 rounded font-medium text-sm
                     bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 disabled:text-gray-500
                     transition-colors"
          onClick={handleSubmit}
          disabled={submitting || active || !text.trim()}
        >
          {submitting ? "Submitting..." : active ? "Processing..." : "Submit Intent"}
        </button>

        <label className="flex items-center gap-1.5 text-xs text-gray-400">
          <input type="checkbox" checked={useLlm} onChange={(e) => setUseLlm(e.target.checked)} className="rounded" />
          Use LLM
        </label>
        <label className="flex items-center gap-1.5 text-xs text-gray-400">
          <input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} className="rounded" />
          Dry run
        </label>
      </div>

      {/* Live pipeline progress — each stage lights up in real-time */}
      {intent && (
        <div className="flex gap-1 mt-4">
          {PIPELINE_STATES.map((stage, i) => {
            const done = currentIdx >= i;
            const isCurrent = currentIdx === i && active;
            return (
              <div
                key={stage}
                className={`flex-1 text-center text-xs py-1.5 rounded transition-colors ${
                  failed && currentIdx < i
                    ? "bg-gray-800 text-gray-600"
                    : failed
                      ? "bg-red-900/60 text-red-300"
                      : done
                        ? "bg-green-900/60 text-green-300"
                        : isCurrent
                          ? "bg-blue-900/60 text-blue-300 animate-pulse"
                          : "bg-gray-800 text-gray-600"
                }`}
              >
                {stage}
              </div>
            );
          })}
        </div>
      )}

      {/* Intent state summary */}
      {intent && (
        <div className="mt-3 rounded bg-gray-800 p-3 text-xs font-mono space-y-1">
          <div className="flex justify-between">
            <span><span className="text-gray-400">id:</span> {intent.id}</span>
            <span className={
              failed ? "text-red-400"
                : intent.state === "completed" ? "text-green-400"
                  : "text-blue-400"
            }>
              {intent.state}
            </span>
          </div>
          <div className="text-gray-500">
            source: {intent.source} | transitions: {intent.history.length}
          </div>
          {intent.git?.pr && (
            <div><span className="text-gray-400">PR:</span> #{intent.git.pr.number} ({intent.git.branch})</div>
          )}
          {intent.porch && !intent.porch.error && (
            <div><span className="text-gray-400">Porch:</span> {intent.porch.lifecycle} ({intent.porch.files} files)</div>
          )}
          {intent.errors && (
            <div className="text-red-400">Errors: {intent.errors.join(", ")}</div>
          )}
        </div>
      )}

      {/* LLM plan detail (collapsible) */}
      {intent?.plan && <PlanDetail plan={intent.plan} />}

      {error && (
        <div className="mt-3 rounded bg-red-900/30 border border-red-800 p-3 text-xs text-red-300">
          {error}
        </div>
      )}
    </section>
  );
}

function PlanDetail({ plan }: { plan: Record<string, unknown> }) {
  const intentType = String(plan.intentType ?? "");
  const actions = Array.isArray(plan.actions) ? (plan.actions as Array<Record<string, unknown>>) : [];

  return (
    <details className="mt-2">
      <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-300">
        IntentPlan (LLM output)
      </summary>
      <div className="mt-1 rounded bg-gray-800 p-2.5 text-xs font-mono">
        <div className="text-gray-400 mb-1">intentType: {intentType}</div>
        {actions.map((a, i) => {
          const naming = a.naming as Record<string, string> | undefined;
          return (
            <div key={i} className="flex gap-3 text-gray-300">
              <span className="text-blue-400">{String(a.kind)}</span>
              <span>{String(a.component)}</span>
              {a.replicas != null && <span className="text-yellow-400">x{Number(a.replicas)}</span>}
              {naming && <span className="text-gray-500">{naming.domain}/{naming.site}</span>}
            </div>
          );
        })}
      </div>
      <details className="mt-1 text-xs">
        <summary className="text-gray-600 cursor-pointer hover:text-gray-400">raw JSON</summary>
        <pre className="mt-1 rounded bg-gray-800 p-2 text-gray-400 overflow-x-auto max-h-48 overflow-y-auto">
          {JSON.stringify(plan, null, 2)}
        </pre>
      </details>
    </details>
  );
}
